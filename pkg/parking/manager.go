package parking

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier   = "Parking: "
	SlashCmd     = "/parking"
	ResetParking = "Reset parking status"
	ResetHour    = 17
	ResetMin     = 0
)

type Manager struct {
	eventManager *event.EventManager
	parkingLot   *ParkingLot
	userManager  *user.Manager

	releaseInfo *ReleaseInfo
}

func NewManager(
	eventManager *event.EventManager,
	config *config.Config,
	userManager *user.Manager,
) *Manager {
	parkingLot := getParkingLot(config)

	return &Manager{
		eventManager: eventManager,
		parkingLot:   &parkingLot,
		userManager:  userManager,
		releaseInfo:  nil,
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.SlashCmdEvent:
		data := e.(*slackApi.Slash)
		if data.Command != SlashCmd {
			return
		}

		response := m.handleSlashCmd(data)
		m.eventManager.Publish(response)
	case event.BlockActionEvent:
		data := e.(*slackApi.BlockAction)

		response := m.handleBlockActions(data)
		if response == nil {
			return
		}

		m.eventManager.Publish(response)

	case event.TimerEvent:
		data := e.(*event.TimerDone)
		if data.Label != ResetParking {
			return
		}

		log.Println("ReleaseSpaces")
		m.parkingLot.ReleaseSpaces(data.Time)
	case event.ViewSubmissionEvent:
		data := e.(*slackApi.ViewSubmission)

		// NOTE: Currently only modal with ViewSubmission for parking pkg
		// is related to parking booking (temporary release of parking space)
		if data.Title != parkingBookingTitle {
			return
		}

		response := m.handleViewSubmission(data)
		if response == nil {
			return
		}
		m.eventManager.Publish(response)
	case event.ViewOpenedEvent:
		data := e.(*slackApi.ViewOpened)

		m.handleViewOpened(data)
	case event.ViewClosedEvent:
		data := e.(*slackApi.ViewClosed)

		m.handleViewClosed(data)
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	errorTxt := ""
	modal := m.generateBookingModalRequest(data, data.UserId, errorTxt)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {

	var actions []event.ResponseAction

	for _, action := range data.Actions {
		switch action.ActionID {
		case reserveParkingActionId:
			isSpecialUser := m.userManager.HasParking(data.UserName)
			parkingSpace := action.Value
			actions = m.handleReserveParking(data, parkingSpace, isSpecialUser)
		case releaseParkingActionId:
			parkingSpace := action.Value
			actions = m.handleReleaseParking(data, parkingSpace)
		case tempReleaseParkingActionId:
			parkingSpace := action.Value
			actions = m.handleTempReleaseParking(data, parkingSpace)
		case cancelTempReleaseParkingActionId:
			parkingSpace := action.Value
			actions = m.handleCancelTempReleaseParking(data, parkingSpace)
		case releaseStartDateActionId, releaseEndDateActionId:
			selectedDate := action.SelectedDate
			isStartDate := action.ActionID == releaseStartDateActionId

			actions = m.handleReleaseRange(data, selectedDate, isStartDate)
		}
	}

	if actions == nil || len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(actions...)
}

func (m *Manager) handleViewSubmission(data *slackApi.ViewSubmission) *common.Response {
	var actions []event.ResponseAction

	submittedData, ok := data.Values[releaseBlockId]
	if !ok {
		return nil
	}

	startDateStr := submittedData[releaseStartDateActionId].SelectedDate
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		// Remote space from temporary release queue
		spaceNum, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		errTxt := fmt.Sprintf(
			"Failed to temporary release space %d: failure to parse start date format %s: %v",
			spaceNum,
			startDateStr,
			err,
		)
		actions = []event.ResponseAction{
			common.NewPostEphemeralAction(
				data.UserId,
				data.UserId,
				slack.MsgOptionText(errTxt, false),
			),
		}
		return common.NewResponseEvent(actions...)
	}

	endDateStr := submittedData[releaseEndDateActionId].SelectedDate
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		// Remote space from temporary release queue
		spaceNum, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		errTxt := fmt.Sprintf(
			"Failed to temporary release space %d: failure to parse end date format %s: %v",
			spaceNum,
			endDateStr,
			err,
		)
		actions = []event.ResponseAction{
			common.NewPostEphemeralAction(
				data.UserId,
				data.UserId,
				slack.MsgOptionText(errTxt, false),
			),
		}
		return common.NewResponseEvent(actions...)
	}

	errorTxt := common.CheckDateRange(startDate, endDate)
	if errorTxt != "" {
		// Remote space from temporary release queue
		spaceNum, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		// TODO: maybe this should be a dialog window instead
		errTxt := fmt.Sprintf("Failed to temporary release space %d: %s", spaceNum, errorTxt)
		actions = []event.ResponseAction{
			common.NewPostEphemeralAction(
				data.UserId,
				data.UserId,
				slack.MsgOptionText(errTxt, false),
			),
		}
		return common.NewResponseEvent(actions...)
	}

	releaseInfo := m.parkingLot.ToBeReleased.GetByViewId(data.ViewId)

	rootViewId := releaseInfo.RootViewId
	releaseInfo.MarkSubmitted()
	m.parkingLot.SynchronizeToFile()

	if common.EqualDate(startDate, time.Now()) {
		m.parkingLot.Release(strconv.Itoa(releaseInfo.Space.Number), data.UserName)
	}

	modal := m.generateBookingModalRequest(data, data.UserId, "")

	actions = append(
		actions,
		common.NewUpdateViewAction(data.TriggerId, rootViewId, modal),
	)

	if actions == nil || len(actions) == 0 {
		return nil
	}
	return common.NewResponseEvent(actions...)
}

func (m *Manager) handleViewOpened(data *slackApi.ViewOpened) {
	releaseInfo := m.parkingLot.ToBeReleased.GetByRootViewId(data.RootViewId)
	if releaseInfo == nil {
		return
	}

	// Associates original view with the new pushed view
	releaseInfo.ViewId = data.ViewId
}

func (m *Manager) handleViewClosed(data *slackApi.ViewClosed) {
	space, success := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
	if success {
		log.Printf("Removed space %d from ToBeReleased queue", space)
		m.parkingLot.SynchronizeToFile()
	}
}

func (m *Manager) handleReserveParking(
	data *slackApi.BlockAction,
	parkingSpace string,
	isSpecialUser bool,
) []event.ResponseAction {
	// Check if an admin has made the request
	autoRelease := true // by default parking reservation is always with auto release
	if isSpecialUser {  // unless we have a special user (i.e. user with designated parking space)
		autoRelease = false
	}

	errStr := m.parkingLot.Reserve(parkingSpace, data.UserName, data.UserId, autoRelease)

	bookingModal := m.generateBookingModalRequest(data, data.UserId, errStr)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	return []event.ResponseAction{action}
}

func (m *Manager) handleTempReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}
	// Special User handling
	chosenParkingSpace := m.parkingLot.GetSpace(parkingSpace)

	// NOTE: here we use the original parking space reserve name and id.
	// this allows us to restore the space to the original user after the temporary release is over.
	// NOTE: here the current view Id is used to help us later identify which space the release
	// modal is referring to
	info, err := m.parkingLot.ToBeReleased.Add(
		data.ViewId,
		data.UserId,
		chosenParkingSpace.ReservedBy,
		chosenParkingSpace.ReservedById,
		chosenParkingSpace,
	)
	// If we can't add a space for temporary release queue it likely means that someone
	// is already trying to do the same thing -> show error in modal
	if err != nil {
		bookingModal := m.generateBookingModalRequest(data, data.UserId, err.Error())
		action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
		actions = append(actions, action)
		return actions
	}

	releaseModal := generateReleaseModalRequest(data, chosenParkingSpace, info.Check())
	action := common.NewPushViewAction(data.TriggerId, releaseModal)
	actions = append(actions, action)
	return actions
}

func (m *Manager) handleCancelTempReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}
	// Special User handling
	chosenParkingSpace := m.parkingLot.GetSpace(parkingSpace)
	errorTxt := ""

	releaseInfo := m.parkingLot.ToBeReleased.Get(chosenParkingSpace.Number)
	if releaseInfo == nil {
		errorTxt = fmt.Sprintf(
			"Couldn't find release info for space %d",
			chosenParkingSpace.Number,
		)
	} else if releaseInfo.StartDate.After(time.Now()) {
		ok := m.parkingLot.ToBeReleased.Remove(chosenParkingSpace.Number)
		if !ok {
			errorTxt = fmt.Sprintf(
				"Failed to cancel temporary release for space %d. Please contact an administrator",
				chosenParkingSpace.Number,
			)
		}
	} else {
		now := time.Now()
		// If user cancelled before the daily reset -> just update end date and
		// auto release of spaces will handle the returning of space to permanent owner
		if now.Hour() <= ResetHour && now.Minute() < ResetMin {
			releaseInfo.EndDate = &now
			releaseInfo.MarkCancelled()
			errorTxt = fmt.Sprintf(
				"Temporary release cancelled. The space %d will be returned tomorrow",
				chosenParkingSpace.Number,
			)
		} else if !chosenParkingSpace.Reserved {
			// if parking space was not already reserved for the next day
			// transfer it to owner
			log.Println("TempReserve chosenParkingSpace ", chosenParkingSpace.Number, releaseInfo)
			chosenParkingSpace.Reserved = true
			chosenParkingSpace.AutoRelease = false
			chosenParkingSpace.ReservedBy = releaseInfo.OwnerName
			chosenParkingSpace.ReservedById = releaseInfo.OwnerId

			ok := m.parkingLot.ToBeReleased.Remove(releaseInfo.Space.Number)
			if !ok {
				log.Printf("Failed removing release info for space %d", releaseInfo.Space.Number)
			}
		} else if chosenParkingSpace.Reserved {
			// if parking space was already reserved by someone else -> transfer back to owner
			// on the day after tomorrow
			errorTxt = fmt.Sprintf(
				`Temporary release cancelled but someone already reserved the space 
for tomorrow. The space %d will be returned to you on the day after tomorrow.`,
				chosenParkingSpace.Number,
			)
			hoursTillMidnight := 24 - now.Hour()
			tomorrow := now.Add(time.Duration(hoursTillMidnight) * time.Hour)
			releaseInfo.EndDate = &tomorrow
			releaseInfo.MarkCancelled()
		}
	}
	m.parkingLot.SynchronizeToFile()

	bookingModal := m.generateBookingModalRequest(data, data.UserId, errorTxt)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	actions = append(actions, action)
	return actions

}

func (m *Manager) handleReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}

	// Handle general case: normal user releasing a space
	victimId, errStr := m.parkingLot.Release(parkingSpace, data.UserName)
	if victimId != "" {
		log.Println(errStr)
		action := common.NewPostEphemeralAction(victimId, victimId, slack.MsgOptionText(errStr, false))
		actions = append(actions, action)
	}

	if m.userManager.IsAdminId(data.UserId) {
		spaceNum, err := strconv.Atoi(parkingSpace)
		if err != nil {
			log.Printf("Failed to convert parking space from str to int: %s", parkingSpace)
		}
		ok := m.parkingLot.ToBeReleased.Remove(spaceNum)
		if !ok {
			log.Printf("Failed to remove release info for space %d", spaceNum)
		}
	}

	errorTxt := ""
	bookingModal := m.generateBookingModalRequest(data, data.UserId, errorTxt)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	actions = append(actions, action)

	return actions
}

func (m *Manager) handleReleaseRange(data *slackApi.BlockAction, selectedDate string, isStartDate bool) []event.ResponseAction {
	date, err := time.Parse("2006-01-02", selectedDate)
	if err != nil {
		log.Printf("Failed to parse date format %s: %v", selectedDate, err)
	}

	releaseInfo := m.parkingLot.ToBeReleased.GetByViewId(data.ViewId)
	// NOTE: releaseInfo is created when the user clicks "Release" button
	if releaseInfo == nil {
		log.Fatalf("Expected release info to be not nil: %v", m.parkingLot.ToBeReleased)
	}

	if isStartDate {
		releaseInfo.StartDate = &date
	} else {
		releaseInfo.EndDate = &date
	}

	modal := generateReleaseModalRequest(data, releaseInfo.Space, releaseInfo.Check())
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal)
	return []event.ResponseAction{action}
}
