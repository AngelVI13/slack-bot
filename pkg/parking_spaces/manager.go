package parking_spaces

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/AngelVI13/slack-bot/pkg/user"
)

const (
	Identifier = "Parking: "
	SlashCmd   = "/parking"
	// SlashCmd = "/test-park"

	ResetParking = "Reset parking status"
	ResetHour    = 17
	ResetMin     = 0
)

type Manager struct {
	eventManager *event.EventManager
	parkingLot   *spaces.SpacesLot
	userManager  *user.Manager

	releaseInfo *spaces.ReleaseInfo

	selectedFloor     map[string]string
	selectedShowTaken map[string]bool
}

func NewManager(
	eventManager *event.EventManager,
	userManager *user.Manager,
	filename string,
) *Manager {
	parkingLot := spaces.GetSpacesLot(filename)

	return &Manager{
		eventManager:      eventManager,
		parkingLot:        &parkingLot,
		userManager:       userManager,
		releaseInfo:       nil,
		selectedFloor:     map[string]string{},
		selectedShowTaken: map[string]bool{},
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

		slog.Info("ReleaseSpaces")
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

	modal := m.generateBookingModalRequest(
		data,
		data.UserId,
		selectedFloor,
		m.selectedShowTaken[data.UserId],
		errorTxt,
	)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(data.UserName, action)
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
			modal := m.generateBookingModalRequest(
				data,
				data.UserId,
				selectedFloor,
				m.selectedShowTaken[data.UserId],
				errorTxt,
			)
			actions = append(
				actions,
				common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errorTxt),
			)

		case reserveParkingActionId:
			isSpecialUser := m.userManager.HasParkingById(data.UserId)
			parkingSpace := spaces.SpaceKey(action.Value)
			actions = m.handleReserveParking(
				data,
				parkingSpace,
				m.selectedFloor[data.UserId],
				m.selectedShowTaken[data.UserId],
				isSpecialUser,
			)

		case releaseParkingActionId:
			parkingSpace := spaces.SpaceKey(action.Value)
			actions = m.handleReleaseParking(
				data,
				parkingSpace,
				m.selectedFloor[data.UserId],
				m.selectedShowTaken[data.UserId],
			)

		case tempReleaseParkingActionId:
			parkingSpace := spaces.SpaceKey(action.Value)
			actions = m.handleTempReleaseParking(
				data,
				parkingSpace,
				m.selectedFloor[data.UserId],
				m.selectedShowTaken[data.UserId],
			)

		case cancelTempReleaseParkingActionId:
			parkingSpace := spaces.SpaceKey(action.Value)
			actions = m.handleCancelTempReleaseParking(
				data,
				parkingSpace,
				m.selectedFloor[data.UserId],
				m.selectedShowTaken[data.UserId],
			)

		case releaseStartDateActionId, releaseEndDateActionId:
			selectedDate := action.SelectedDate
			isStartDate := action.ActionID == releaseStartDateActionId

			actions = m.handleReleaseRange(data, selectedDate, isStartDate)

		case showOptionId:
			selectedShowValue := data.Values[showActionId][showOptionId].SelectedOption.Value
			selectedShowOption := selectedShowValue == showTakenOption
			m.selectedShowTaken[data.UserId] = selectedShowOption
			errorTxt := ""
			modal := m.generateBookingModalRequest(
				data,
				data.UserId,
				m.selectedFloor[data.UserId],
				selectedShowOption,
				errorTxt,
			)
			actions = append(
				actions,
				common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errorTxt),
			)
		}
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
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
			common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
		}
		return common.NewResponseEvent(data.UserName, actions...)
	}

	currentLocation := time.Now().Location()

	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, currentLocation)
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
			common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
		}
		return common.NewResponseEvent(data.UserName, actions...)
	}

	endDateStr := submittedData[releaseEndDateActionId].SelectedDate
	if endDateStr == "" {
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: no end date provided",
			spaceKey,
		)
		actions = []event.ResponseAction{
			common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
		}
		return common.NewResponseEvent(data.UserName, actions...)
	}
	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, currentLocation)
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
			common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
		}
		return common.NewResponseEvent(data.UserName, actions...)
	}

	errorTxt := common.CheckDateRange(startDate, endDate)
	if errorTxt != "" {
		// Remote space from temporary release queue
		spaceKey, _ := m.parkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
		m.parkingLot.SynchronizeToFile()

		// TODO: maybe this should be a dialog window instead
		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: %s",
			spaceKey,
			errorTxt,
		)
		actions = []event.ResponseAction{
			common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
		}
		return common.NewResponseEvent(data.UserName, actions...)
	}

	releaseInfo := m.parkingLot.ToBeReleased.GetByViewId(data.ViewId)

	rootViewId := releaseInfo.RootViewId
	releaseInfo.MarkSubmitted()
	m.parkingLot.SynchronizeToFile()
	currentTime := time.Now()

	if common.EqualDate(startDate, currentTime) || (currentTime.Before(startDate) &&
		startDate.Sub(currentTime).Hours() < 24 &&
		currentTime.Hour() >= ResetHour && currentTime.Minute() >= ResetMin) {
		// Directly release space in two cases:
		// * Release starts from today
		// * Release starts from tomorrow & current time is after Reset time
		m.parkingLot.Release(releaseInfo.Space.Key(), data.UserName, data.UserId)
	}

	errTxt := ""
	modal := m.generateBookingModalRequest(
		data,
		data.UserId,
		m.selectedFloor[data.UserId],
		m.selectedShowTaken[data.UserId],
		errTxt,
	)

	actions = append(
		actions,
		common.NewUpdateViewAction(data.TriggerId, rootViewId, modal, errTxt),
	)

	if len(actions) == 0 {
		return nil
	}
	return common.NewResponseEvent(data.UserName, actions...)
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
		slog.Info("Removed from ToBeReleased queue", "space", space)
		m.parkingLot.SynchronizeToFile()
	}
}

func (m *Manager) handleReserveParking(
	data *slackApi.BlockAction,
	parkingSpace spaces.SpaceKey,
	selectedFloor string,
	selectedShowTaken bool,
	isSpecialUser bool,
) []event.ResponseAction {
	// Check if an admin has made the request
	autoRelease := true // by default parking reservation is always with auto release
	if isSpecialUser {  // unless we have a special user (i.e. user with designated parking space)
		autoRelease = false
	}

	errStr := m.parkingLot.Reserve(parkingSpace, data.UserName, data.UserId, autoRelease)

	bookingModal := m.generateBookingModalRequest(
		data,
		data.UserId,
		selectedFloor,
		selectedShowTaken,
		errStr,
	)
	action := common.NewUpdateViewAction(
		data.TriggerId,
		data.ViewId,
		bookingModal,
		errStr,
	)
	return []event.ResponseAction{action}
}

func (m *Manager) handleTempReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace spaces.SpaceKey,
	selectedFloor string,
	selectedShowTaken bool,
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
		data.UserName,
		data.UserId,
		chosenParkingSpace.ReservedBy,
		chosenParkingSpace.ReservedById,
		chosenParkingSpace,
	)
	// If we can't add a space for temporary release queue it likely means that someone
	// is already trying to do the same thing -> show error in modal
	if err != nil {
		errTxt := err.Error()
		bookingModal := m.generateBookingModalRequest(
			data,
			data.UserId,
			selectedFloor,
			selectedShowTaken,
			errTxt,
		)
		action := common.NewUpdateViewAction(
			data.TriggerId,
			data.ViewId,
			bookingModal,
			errTxt,
		)
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
	parkingSpace spaces.SpaceKey,
	selectedFloor string,
	selectedShowTaken bool,
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
		slog.Info("Cancel temp. release & return to owner", "space", parkingSpace, "releaseInfo", releaseInfo)
		chosenParkingSpace.Reserved = true
		chosenParkingSpace.AutoRelease = false
		chosenParkingSpace.ReservedBy = releaseInfo.OwnerName
		chosenParkingSpace.ReservedById = releaseInfo.OwnerId

		ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
		if !ok {
			errorTxt = fmt.Sprintf(
				"Failed to cancel temporary release for space %s. Please contact an administrator",
				parkingSpace,
			)
		}
	} else {
		now := time.Now()
		// If user cancelled before the daily reset
		if (now.Hour() < ResetHour) || (now.Hour() == ResetHour && now.Minute() < ResetMin) {
			// Somebody already booked it for the day -> return it at end of day
			if chosenParkingSpace.Reserved {
				slog.Info(
					"Temporary release cancelled (before eod). Space is taken. Return to owner at eod.",
					"space", parkingSpace, "releaseInfo", releaseInfo)
				releaseInfo.EndDate = &now
				releaseInfo.MarkCancelled()
				errorTxt = fmt.Sprintf(
					"Temporary release cancelled. The space %s will be returned to you today at %d:%d",
					parkingSpace,
					ResetHour,
					ResetMin,
				)
			} else {
				// if parking space was not already reserved for the day transfer it to owner
				slog.Info(
					"Temporary release cancelled (before eod). Space is not taken. Return to owner immediately.",
					"space", parkingSpace, "releaseInfo", releaseInfo)
				chosenParkingSpace.Reserved = true
				chosenParkingSpace.AutoRelease = false
				chosenParkingSpace.ReservedBy = releaseInfo.OwnerName
				chosenParkingSpace.ReservedById = releaseInfo.OwnerId

				ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
				if !ok {
					slog.Error("Failed removing release info", "space", parkingSpace)
				}
			}
		} else { // User cancelled space after EOD
			if chosenParkingSpace.Reserved {
				// if parking space was already reserved by someone else -> transfer
				// back to owner on the day after tomorrow
				slog.Info(
					"Temporary release cancelled (after eod). Space is taken. Return to owner tomorrow after eod.",
					"space", parkingSpace, "releaseInfo", releaseInfo)
				errorTxt = fmt.Sprintf(
					`Temporary release cancelled but someone already reserved the space 
                    for tomorrow. The space %s will be returned to you tomorrow at %d:%d.`,
					parkingSpace,
					ResetHour,
					ResetMin,
				)
				hoursTillMidnight := 24 - now.Hour()
				tomorrow := now.Add(time.Duration(hoursTillMidnight) * time.Hour)
				releaseInfo.EndDate = &tomorrow
				releaseInfo.MarkCancelled()
			} else {
				// if parking space was not already reserved for the next day
				// transfer it to owner
				slog.Info(
					"Temporary release cancelled (after eod). Space is not taken. Return to owner immediately.",
					"space", parkingSpace, "releaseInfo", releaseInfo)
				chosenParkingSpace.Reserved = true
				chosenParkingSpace.AutoRelease = false
				chosenParkingSpace.ReservedBy = releaseInfo.OwnerName
				chosenParkingSpace.ReservedById = releaseInfo.OwnerId

				ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
				if !ok {
					slog.Error("Failed removing release info", "space", parkingSpace)
				}
			}
		}
	}
	m.parkingLot.SynchronizeToFile()

	bookingModal := m.generateBookingModalRequest(
		data,
		data.UserId,
		selectedFloor,
		selectedShowTaken,
		errorTxt,
	)
	action := common.NewUpdateViewAction(
		data.TriggerId,
		data.ViewId,
		bookingModal,
		errorTxt,
	)
	actions = append(actions, action)
	return actions
}

func (m *Manager) handleReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace spaces.SpaceKey,
	selectedFloor string,
	selectedShowTaken bool,
) []event.ResponseAction {
	actions := []event.ResponseAction{}

	// Handle general case: normal user releasing a space
	victimId, errStr := m.parkingLot.Release(parkingSpace, data.UserName, data.UserId)
	if victimId != "" {
		slog.Warn(errStr)
		action := common.NewPostEphemeralAction(victimId, victimId, errStr, false)
		actions = append(actions, action)
	}

	// Only remove release info from a space if an Admin is permanently releasing the space
	if m.userManager.IsAdminId(data.UserId) {
		ok := m.parkingLot.ToBeReleased.Remove(parkingSpace)
		if !ok {
			slog.Error("Failed to remove release info", "space", parkingSpace)
		}
	}

	errorTxt := ""
	bookingModal := m.generateBookingModalRequest(
		data,
		data.UserId,
		selectedFloor,
		selectedShowTaken,
		errorTxt,
	)
	action := common.NewUpdateViewAction(
		data.TriggerId,
		data.ViewId,
		bookingModal,
		errorTxt,
	)
	actions = append(actions, action)

	return actions
}

func (m *Manager) handleReleaseRange(
	data *slackApi.BlockAction,
	selectedDate string,
	isStartDate bool,
) []event.ResponseAction {
	currentLocation := time.Now().Location()
	date, err := time.ParseInLocation("2006-01-02", selectedDate, currentLocation)
	if err != nil {
		// TODO: should this return here or just log ?
		slog.Error("Failed to parse date format", "date", selectedDate, "err", err)
	}

	releaseInfo := m.parkingLot.ToBeReleased.GetByViewId(data.ViewId)
	// NOTE: releaseInfo is created when the user clicks "Release" button
	if releaseInfo == nil {
		slog.Error(
			"Expected release info to be not nil",
			"ToBeReleased",
			m.parkingLot.ToBeReleased,
		)
		return nil
	}

	if isStartDate {
		releaseInfo.StartDate = &date
	} else {
		releaseInfo.EndDate = &date
	}

	errTxt := releaseInfo.Check()
	modal := generateReleaseModalRequest(data, releaseInfo.Space, errTxt)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errTxt)
	return []event.ResponseAction{action}
}
