package parking

import (
	"fmt"
	"log"
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

	selectedFloor map[string]string
}

func NewManager(
	eventManager *event.EventManager,
	config *config.Config,
	userManager *user.Manager,
) *Manager {
	parkingLot := getParkingLot(config)

	return &Manager{
		eventManager:  eventManager,
		parkingLot:    &parkingLot,
		userManager:   userManager,
		releaseInfo:   nil,
		selectedFloor: map[string]string{},
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
	selectedFloor := defaultFloorOption
	selected, ok := m.selectedFloor[data.UserId]
	if ok {
		selectedFloor = selected
	}
	modal := m.generateBookingModalRequest(data, data.UserId, selectedFloor, errorTxt)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	var actions []event.ResponseAction

	if _, ok := m.selectedFloor[data.UserId]; !ok {
		m.selectedFloor[data.UserId] = defaultFloorOption
	}

	for _, action := range data.Actions {
		switch action.ActionID {
		case floorOptionId:
			selectedFloor := data.Values[floorActionId][floorOptionId].SelectedOption.Value
			m.selectedFloor[data.UserId] = selectedFloor
			errorTxt := ""
			modal := m.generateBookingModalRequest(data, data.UserId, selectedFloor, errorTxt)
			actions = append(actions, common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal))

		case reserveParkingActionId:
			isSpecialUser := m.userManager.HasParking(data.UserName)
			parkingSpace := ParkingKey(action.Value)
			actions = m.handleReserveParking(data, parkingSpace, m.selectedFloor[data.UserId], isSpecialUser)

		case releaseParkingActionId:
			parkingSpace := ParkingKey(action.Value)
			actions = m.handleReleaseParking(data, parkingSpace, m.selectedFloor[data.UserId])

		case tempReleaseParkingActionId:
			parkingSpace := ParkingKey(action.Value)
			actions = m.handleTempReleaseParking(data, parkingSpace, m.selectedFloor[data.UserId])

		case cancelTempReleaseParkingActionId:
			parkingSpace := ParkingKey(action.Value)
			actions = m.handleCancelTempReleaseParking(data, parkingSpace, m.selectedFloor[data.UserId])

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
	if startDateStr == "" {
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: no start date provided",
			spaceKey,
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

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		// Remote space from temporary release queue
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: failure to parse start date format %s: %v",
			spaceKey,
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
	if endDateStr == "" {
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: no end date provided",
			spaceKey,
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
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		// Remote space from temporary release queue
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: failure to parse end date format %s: %v",
			spaceKey,
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
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		// TODO: maybe this should be a dialog window instead
		errTxt := fmt.Sprintf("Failed to temporary release space %s: %s", spaceKey, errorTxt)
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
		m.parkingLot.Release(releaseInfo.Space.ParkingKey(), data.UserName, data.UserId)
	}

	modal := m.generateBookingModalRequest(data, data.UserId, m.selectedFloor[data.UserId], "")

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
		log.Printf("Removed space %s from ToBeReleased queue", space)
		m.parkingLot.SynchronizeToFile()
	}
}

func (m *Manager) handleReserveParking(
	data *slackApi.BlockAction,
	parkingSpace ParkingKey,
	selectedFloor string,
	isSpecialUser bool,
) []event.ResponseAction {
	// Check if an admin has made the request
	autoRelease := true // by default parking reservation is always with auto release
	if isSpecialUser {  // unless we have a special user (i.e. user with designated parking space)
		autoRelease = false
	}

	errStr := m.parkingLot.Reserve(parkingSpace, data.UserName, data.UserId, autoRelease)

	bookingModal := m.generateBookingModalRequest(data, data.UserId, selectedFloor, errStr)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	return []event.ResponseAction{action}
}

func (m *Manager) handleTempReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace ParkingKey,
	selectedFloor string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}
	// Special User handling
	chosenParkingSpace := m.parkingLot.GetSpace(parkingSpace)
	if chosenParkingSpace == nil {
		return nil
	}

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
		bookingModal := m.generateBookingModalRequest(data, data.UserId, selectedFloor, err.Error())
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
	parkingSpace ParkingKey,
	selectedFloor string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}
	// Special User handling
	chosenParkingSpace := m.parkingLot.GetSpace(parkingSpace)
	if chosenParkingSpace == nil {
		return nil
	}

	errorTxt := ""

	releaseInfo := m.parkingLot.ToBeReleased.Get(parkingSpace)
	if releaseInfo == nil {
		errorTxt = fmt.Sprintf("Couldn't find release info for space %s", parkingSpace)
	} else if releaseInfo.StartDate.After(time.Now()) {
		ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
		if !ok {
			errorTxt = fmt.Sprintf(
				"Failed to cancel temporary release for space %s. Please contact an administrator",
				parkingSpace,
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
				"Temporary release cancelled. The space %s will be returned tomorrow",
				parkingSpace,
			)
		} else if !chosenParkingSpace.Reserved {
			// if parking space was not already reserved for the next day
			// transfer it to owner
			log.Println("TempReserve chosenParkingSpace ", parkingSpace, releaseInfo)
			chosenParkingSpace.Reserved = true
			chosenParkingSpace.AutoRelease = false
			chosenParkingSpace.ReservedBy = releaseInfo.OwnerName
			chosenParkingSpace.ReservedById = releaseInfo.OwnerId

			ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
			if !ok {
				log.Printf("Failed removing release info for space %s", parkingSpace)
			}
		} else if chosenParkingSpace.Reserved {
			// if parking space was already reserved by someone else -> transfer
			// back to owner on the day after tomorrow
			errorTxt = fmt.Sprintf(
				`Temporary release cancelled but someone already reserved the space 
for tomorrow. The space %s will be returned to you on the day after tomorrow.`,
				parkingSpace,
			)
			hoursTillMidnight := 24 - now.Hour()
			tomorrow := now.Add(time.Duration(hoursTillMidnight) * time.Hour)
			releaseInfo.EndDate = &tomorrow
			releaseInfo.MarkCancelled()
		}
	}
	m.parkingLot.SynchronizeToFile()

	bookingModal := m.generateBookingModalRequest(data, data.UserId, selectedFloor, errorTxt)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	actions = append(actions, action)
	return actions

}

func (m *Manager) handleReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace ParkingKey,
	selectedFloor string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}

	// Handle general case: normal user releasing a space
	victimId, errStr := m.parkingLot.Release(parkingSpace, data.UserName, data.UserId)
	if victimId != "" {
		log.Println(errStr)
		action := common.NewPostEphemeralAction(victimId, victimId, slack.MsgOptionText(errStr, false))
		actions = append(actions, action)
	}

	// Only remove release info from a space if an Admin is permanently releasing the space
	if m.userManager.IsAdminId(data.UserId) {
		ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
		if !ok {
			log.Printf("Failed to remove release info for space %s", parkingSpace)
		}
	}

	errorTxt := ""
	bookingModal := m.generateBookingModalRequest(data, data.UserId, selectedFloor, errorTxt)
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
		log.Printf("ERROR: Expected release info to be not nil: %v", m.parkingLot.ToBeReleased)
		return nil
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
