package parking_spaces

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
	parkingModel "github.com/AngelVI13/slack-bot/pkg/parking_spaces/model"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces/views"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Parking: "
	SlashCmd   = "/parking"
	// SlashCmd = "/test-park"

	ResetParking = "Reset parking status"
	ResetHour    = parkingModel.ResetHour
	ResetMin     = parkingModel.ResetMin
)

type Manager struct {
	eventManager *event.EventManager
	data         *parkingModel.ParkingData
	bookingView  *views.Booking
	releaseView  *views.Release
	personalView *views.Personal
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
) *Manager {
	parkingData := parkingModel.NewParkingData(data)
	return &Manager{
		eventManager: eventManager,
		data:         parkingData,
		bookingView:  views.NewBooking(Identifier, parkingData),
		releaseView:  views.NewRelease(Identifier, parkingData),
		personalView: views.NewPersonal(Identifier, parkingData),
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
		m.data.ParkingLot.ReleaseSpaces(data.Time)
	case event.ViewSubmissionEvent:
		data := e.(*slackApi.ViewSubmission)

		// NOTE: Currently only modal with ViewSubmission for parking pkg
		// is related to parking booking (temporary release of parking space)
		if data.Title != m.releaseView.Title {
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

	var modal slack.ModalViewRequest
	if m.data.ParkingLot.OwnsSpace(data.UserId) != nil {
		modal = m.personalView.Generate(data.UserId, errorTxt)
	} else {
		modal = m.bookingView.Generate(data.UserId, errorTxt)
	}

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(data.UserName, action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	var actions []event.ResponseAction

	m.data.SetDefaultFloorIfMissing(data.UserId, parkingModel.DefaultFloorOption)

	for _, action := range data.Actions {
		switch action.ActionID {
		case views.FloorOptionId:
			selectedFloor := data.IValueSingle(views.FloorActionId, views.FloorOptionId)
			m.data.SelectedFloor[data.UserId] = selectedFloor
			errorTxt := ""
			modal := m.bookingView.Generate(data.UserId, errorTxt)
			actions = append(
				actions,
				common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errorTxt),
			)

		case views.ReserveParkingActionId:
			actionValues := views.ActionValues{}.Decode(action.Value)
			actions = m.handleReserveParking(data, actionValues)

		case views.ReleaseParkingActionId:
			actionValues := views.ActionValues{}.Decode(action.Value)
			actions = m.handleReleaseParking(data, actionValues)

		case views.TempReleaseParkingActionId:
			actionValues := views.ActionValues{}.Decode(action.Value)
			actions = m.handleTempReleaseParking(data, actionValues)

		case views.CancelTempReleaseParkingActionId:
			actionValues := views.ActionValues{}.Decode(action.Value)
			actions = m.handleCancelTempReleaseParking(data, actionValues)

		case views.ReleaseStartDateActionId, views.ReleaseEndDateActionId:
			selectedDate := action.SelectedDate
			isStartDate := action.ActionID == views.ReleaseStartDateActionId

			actions = m.handleReleaseRange(data, selectedDate, isStartDate)

		case views.ShowOptionId:
			selectedShowValue := data.IValueSingle(views.ShowActionId, views.ShowOptionId)
			selectedShowOption := selectedShowValue == parkingModel.ShowTakenOption
			m.data.SelectedShowTaken[data.UserId] = selectedShowOption
			errorTxt := ""
			modal := m.bookingView.Generate(data.UserId, errorTxt)
			actions = append(
				actions,
				common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errorTxt),
			)
		case views.SwitchToAllSpacesOverviewId:
			errorTxt := ""
			modal := m.bookingView.Generate(data.UserId, errorTxt)
			actions = append(
				actions,
				common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errorTxt),
			)
		case views.SwitchToPersonalViewId:
			errorTxt := ""
			modal := m.personalView.Generate(data.UserId, errorTxt)
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

	submittedData, ok := data.Values[views.ReleaseBlockId]
	if !ok {
		return nil
	}

	startDate, endDate, errResp := m.validateTempReleaseDateInputs(data, submittedData)
	if errResp != nil {
		return errResp
	}

	releaseInfo := m.data.ParkingLot.ToBeReleased.GetByViewId(data.ViewId)

	rootViewId := releaseInfo.RootViewId

	releaseInfo.StartDate = startDate
	releaseInfo.EndDate = endDate
	releaseInfo.MarkSubmitted(data.UserName)

	overlaps := m.data.ParkingLot.ToBeReleased.CheckOverlap(releaseInfo)
	if len(overlaps) > 0 {
		spaceKey := releaseInfo.Space.Key()

		errTxt := fmt.Sprintf(
			"Failed to temporary release space %s: "+
				"Selected date range %s overlaps with "+
				"some of the previously scheduled releases: %v",
			spaceKey,
			releaseInfo.DateRange(),
			overlaps,
		)
		slog.Error(errTxt)

		// NOTE: can't use the handleViewSubmissionError because it removes releases
		// based on ViewId and that is reset after a space is marked as submitted
		m.data.ParkingLot.ToBeReleased.Remove(releaseInfo)
		m.data.ParkingLot.SynchronizeToFile()

		actions = []event.ResponseAction{
			common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
		}
		return common.NewResponseEvent(data.UserName, actions...)
	}

	currentTime := time.Now()

	if common.EqualDate(*startDate, currentTime) || (currentTime.Before(*startDate) &&
		startDate.Sub(currentTime).Hours() < 24 &&
		currentTime.Hour() >= ResetHour && currentTime.Minute() >= ResetMin) {
		// Directly release space in two cases:
		// * Release starts from today
		// * Release starts from tomorrow & current time is after Reset time
		m.data.ParkingLot.Release(releaseInfo.Space.Key(), data.UserName, data.UserId)
		releaseInfo.MarkActive()
	}

	m.data.ParkingLot.SynchronizeToFile()

	errTxt := ""

	var modal slack.ModalViewRequest
	// only show personal view if owner is temp releasing their space
	// if another user(admin) is temp releasing the space -> show overview modal
	if m.data.ParkingLot.OwnsSpace(data.UserId) != nil &&
		data.UserId == releaseInfo.OwnerId {
		modal = m.personalView.Generate(data.UserId, errTxt)
	} else {
		modal = m.bookingView.Generate(data.UserId, errTxt)
	}

	updateAction := common.NewUpdateViewAction(data.TriggerId, rootViewId, modal, errTxt)
	actions = append(actions, updateAction)

	return common.NewResponseEvent(data.UserName, actions...)
}

func (m *Manager) validateTempReleaseDateInputs(
	data *slackApi.ViewSubmission,
	submittedData map[string]slack.BlockAction,
) (*time.Time, *time.Time, *common.Response) {
	startDateStr := submittedData[views.ReleaseStartDateActionId].SelectedDate
	if startDateStr == "" {
		return nil, nil, m.handleViewSubmissionError(data, "no start date provided")
	}

	currentLocation := time.Now().Location()

	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, currentLocation)
	if err != nil {
		errTxt := fmt.Sprintf(
			"failure to parse start date format %s: %v",
			startDateStr,
			err,
		)
		return nil, nil, m.handleViewSubmissionError(data, errTxt)
	}

	endDateStr := submittedData[views.ReleaseEndDateActionId].SelectedDate
	if endDateStr == "" {
		return nil, nil, m.handleViewSubmissionError(data, "no end date provided")
	}

	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, currentLocation)
	if err != nil {
		errTxt := fmt.Sprintf(
			"failure to parse end date format %s: %v",
			endDateStr,
			err,
		)
		return nil, nil, m.handleViewSubmissionError(data, errTxt)
	}

	errorTxt := common.CheckDateRange(startDate, endDate)
	if errorTxt != "" {
		return nil, nil, m.handleViewSubmissionError(data, errorTxt)
	}
	return &startDate, &endDate, nil
}

func (m *Manager) handleViewSubmissionError(
	data *slackApi.ViewSubmission,
	errTxt string,
) *common.Response {
	// Remove space from temporary release queue
	spaceKey, _ := m.data.ParkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
	m.data.ParkingLot.SynchronizeToFile()

	errTxt = fmt.Sprintf("Failed to temporary release space %s: %s", spaceKey, errTxt)

	actions := []event.ResponseAction{
		common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false),
	}
	return common.NewResponseEvent(data.UserName, actions...)
}

func (m *Manager) handleViewOpened(data *slackApi.ViewOpened) {
	releaseInfo := m.data.ParkingLot.ToBeReleased.GetByRootViewId(data.RootViewId)
	if releaseInfo == nil {
		return
	}

	// Associates original view with the new pushed view
	releaseInfo.ViewId = data.ViewId
}

func (m *Manager) handleViewClosed(data *slackApi.ViewClosed) {
	space, success := m.data.ParkingLot.ToBeReleased.RemoveByViewId(data.ViewId)
	if success {
		slog.Info("Removed from ToBeReleased queue", "space", space)
		m.data.ParkingLot.SynchronizeToFile()
	}
}

func (m *Manager) handleReserveParking(
	data *slackApi.BlockAction,
	actionValues views.ActionValues,
) []event.ResponseAction {
	// Check if an admin has made the request

	autoRelease := true // by default parking reservation is always with auto release
	if m.data.UserManager.HasParkingById(data.UserId) {
		// unless we have a special user (i.e. user with designated parking space)
		autoRelease = false
	}

	parkingSpace := actionValues.SpaceKey

	errStr := m.data.ParkingLot.Reserve(
		parkingSpace,
		data.UserName,
		data.UserId,
		autoRelease,
	)

	var bookingModal slack.ModalViewRequest
	if actionValues.ModalType == views.PersonalModal {
		bookingModal = m.personalView.Generate(data.UserId, errStr)
	} else {
		bookingModal = m.bookingView.Generate(data.UserId, errStr)
	}

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
	actionValues views.ActionValues,
) []event.ResponseAction {
	parkingSpace := actionValues.SpaceKey

	actions := []event.ResponseAction{}
	// Special User handling
	chosenParkingSpace := m.data.ParkingLot.GetSpace(parkingSpace)
	if chosenParkingSpace == nil {
		return nil
	}

	// NOTE: here we use the original parking space reserve name and id.
	// this allows us to restore the space to the original user after the temporary release is over.
	// NOTE: here the current view Id is used to help us later identify which space the release
	// modal is referring to
	info, err := m.data.ParkingLot.ToBeReleased.Add(
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
		bookingModal := m.bookingView.Generate(data.UserId, errTxt)
		action := common.NewUpdateViewAction(
			data.TriggerId,
			data.ViewId,
			bookingModal,
			errTxt,
		)
		actions = append(actions, action)
		return actions
	}

	releaseModal := m.releaseView.Generate(chosenParkingSpace, info.Check())
	action := common.NewPushViewAction(data.TriggerId, releaseModal)
	actions = append(actions, action)
	return actions
}

func (m *Manager) handleCancelTempReleaseParking(
	data *slackApi.BlockAction,
	actionValues views.ActionValues,
) []event.ResponseAction {
	parkingSpace := actionValues.SpaceKey
	releaseId := actionValues.ReleaseId

	actions := []event.ResponseAction{}
	// Special User handling
	chosenParkingSpace := m.data.ParkingLot.GetSpace(parkingSpace)
	if chosenParkingSpace == nil {
		return nil
	}

	errorTxt := ""

	releaseInfo := m.data.ParkingLot.ToBeReleased.Get(parkingSpace, releaseId)
	if releaseInfo == nil {
		errorTxt = fmt.Sprintf("Couldn't find release info for space %s", parkingSpace)
	} else if !releaseInfo.Active {
		slog.Info("Cancel scheduled (not active) temp. release", "space", parkingSpace, "releaseInfo", releaseInfo)
		err := m.data.ParkingLot.ToBeReleased.Remove(releaseInfo)
		if err != nil {
			slog.Error(
				"failed to remove release", "space", parkingSpace, "id", releaseId,
				"releaseInfo", releaseInfo, "err", err,
			)
		}
	} else if releaseInfo.StartDate.After(time.Now()) {
		slog.Info("Cancel temp. release & return to owner", "space", parkingSpace, "releaseInfo", releaseInfo)
		chosenParkingSpace.Reserved = true
		chosenParkingSpace.AutoRelease = false
		chosenParkingSpace.ReservedBy = releaseInfo.OwnerName
		chosenParkingSpace.ReservedById = releaseInfo.OwnerId

		err := m.data.ParkingLot.ToBeReleased.Remove(releaseInfo)
		if err != nil {
			errorTxt = fmt.Sprintf(
				"Failed to cancel temporary release for space %s. %v. Please contact an administrator",
				parkingSpace, err)
		}
	} else { // release data was before now - i.e. temp release is currently active
		now := time.Now()
		// If user cancelled before the daily reset
		if (now.Hour() < ResetHour) || (now.Hour() == ResetHour && now.Minute() < ResetMin) {
			// Somebody already booked it for the day -> return it at end of day
			if chosenParkingSpace.Reserved {
				slog.Info(
					"Temporary release cancelled (before eod). Space is taken. Return to owner at eod.",
					"space", parkingSpace, "releaseInfo", releaseInfo)
				releaseInfo.EndDate = &now
				errorTxt = fmt.Sprintf(
					"Temporary release cancelled. The space %s will be returned to you today at %d:%02d",
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

				err := m.data.ParkingLot.ToBeReleased.Remove(releaseInfo)
				if err != nil {
					slog.Error("Failed removing release info", "space", parkingSpace, "err", err)
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
                    for tomorrow. The space %s will be returned to you tomorrow at %d:%02d.`,
					parkingSpace,
					ResetHour,
					ResetMin,
				)
				hoursTillMidnight := 24 - now.Hour()
				tomorrow := now.Add(time.Duration(hoursTillMidnight) * time.Hour)
				releaseInfo.EndDate = &tomorrow
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

				err := m.data.ParkingLot.ToBeReleased.Remove(releaseInfo)
				if err != nil {
					slog.Error("Failed removing release info", "space", parkingSpace, "err", err)
				}
			}
		}
	}
	m.data.ParkingLot.SynchronizeToFile()

	var bookingModal slack.ModalViewRequest
	if actionValues.ModalType == views.PersonalModal {
		bookingModal = m.personalView.Generate(data.UserId, errorTxt)
	} else {
		bookingModal = m.bookingView.Generate(data.UserId, errorTxt)
	}

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
	actionValues views.ActionValues,
) []event.ResponseAction {
	parkingSpace := actionValues.SpaceKey
	isReleaserAdmin := m.data.UserManager.IsAdminId(data.UserId)
	space := m.data.ParkingLot.GetSpace(parkingSpace)
	isSpaceTempReserved := space.Reserved && space.AutoRelease

	actions := []event.ResponseAction{}

	// NOTE: there are 2 scenarios that should be handled here
	// 1. User without a space has booked a space and decided to release it
	//     - Release space and inform victim if needed
	// 2. Admin is releasing someone's space
	//     - if space currently has a temp release active and someone has booked the space
	//         - release space and inform victim if needed
	//     - if space does not have a temp release
	//         - release space and inform victim if needed
	//         - and remove any associated releases for that space (full release)

	if isSpaceTempReserved || isReleaserAdmin {
		// Handle general case: normal user releasing a space
		victimId, errStr := m.data.ParkingLot.Release(
			parkingSpace,
			data.UserName,
			data.UserId,
		)
		if victimId != "" {
			slog.Warn(errStr)
			action := common.NewPostEphemeralAction(victimId, victimId, errStr, false)
			actions = append(actions, action)
		}
	}

	if isReleaserAdmin && !isSpaceTempReserved {
		// Only remove release info from a space if an Admin is permanently releasing the space
		m.data.ParkingLot.ToBeReleased.RemoveAllReleases(parkingSpace)
	}

	errorTxt := ""
	var modal slack.ModalViewRequest
	if actionValues.ModalType == views.PersonalModal {
		modal = m.personalView.Generate(data.UserId, errorTxt)
	} else {
		modal = m.bookingView.Generate(data.UserId, errorTxt)
	}

	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errorTxt)
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
		slog.Error("Failed to parse date format", "date", selectedDate, "err", err)
		return nil
	}

	releaseInfo := m.data.ParkingLot.ToBeReleased.GetByViewId(data.ViewId)
	// NOTE: releaseInfo is created when the user clicks "Release" button
	if releaseInfo == nil {
		slog.Error(
			"Expected release info to be not nil",
			"ToBeReleased",
			m.data.ParkingLot.ToBeReleased,
		)
		return nil
	}

	if isStartDate {
		releaseInfo.StartDate = &date
	} else {
		releaseInfo.EndDate = &date
	}

	errTxt := releaseInfo.Check()
	modal := m.releaseView.Generate(releaseInfo.Space, errTxt)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, errTxt)
	return []event.ResponseAction{action}
}
