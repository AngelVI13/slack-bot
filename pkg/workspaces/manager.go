package workspaces

import (
	"log"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Workspaces: "
	// SlashCmd   = "/workspace"
	SlashCmd = "/test-workspace"

	defaultUserOption = ""

	ResetWorkspaces = "Reset workspaces status"
	ResetHour       = 17
	ResetMin        = 0
)

type Manager struct {
	eventManager *event.EventManager
	userManager  *user.Manager
	parkingLot   *spaces.SpacesLot
	slackClient  *slack.Client

	selectedFloor map[string]string
}

func NewManager(
	eventManager *event.EventManager,
	userManager *user.Manager,
	filename string,
) *Manager {
	parkingLot := spaces.GetSpacesLot(filename)
	return &Manager{
		eventManager:  eventManager,
		userManager:   userManager,
		parkingLot:    &parkingLot,
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
		if data.Label != ResetWorkspaces {
			return
		}

		log.Println("ReleaseWorkspaces")
		// m.parkingLot.ReleaseSpaces(data.Time)
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
			modal := m.generateBookingModalRequest(
				data,
				data.UserId,
				selectedFloor,
				errorTxt,
			)
			actions = append(
				actions,
				common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal),
			)

		case reserveParkingActionId:
			isSpecialUser := m.userManager.HasParkingById(data.UserId)
			parkingSpace := spaces.SpaceKey(action.Value)
			actions = m.handleReserveParking(
				data,
				parkingSpace,
				m.selectedFloor[data.UserId],
				isSpecialUser,
			)

		case releaseParkingActionId:
			parkingSpace := spaces.SpaceKey(action.Value)
			actions = m.handleReleaseParking(
				data,
				parkingSpace,
				m.selectedFloor[data.UserId],
			)
		}
	}

	if actions == nil || len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(actions...)
}

func (m *Manager) handleReserveParking(
	data *slackApi.BlockAction,
	parkingSpace spaces.SpaceKey,
	selectedFloor string,
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
		errStr,
	)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	return []event.ResponseAction{action}
}

func (m *Manager) handleReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace spaces.SpaceKey,
	selectedFloor string,
) []event.ResponseAction {
	actions := []event.ResponseAction{}

	// Handle general case: normal user releasing a space
	victimId, errStr := m.parkingLot.Release(parkingSpace, data.UserName, data.UserId)
	if victimId != "" {
		log.Println(errStr)
		action := common.NewPostEphemeralAction(
			victimId,
			victimId,
			slack.MsgOptionText(errStr, false),
		)
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
	bookingModal := m.generateBookingModalRequest(
		data,
		data.UserId,
		selectedFloor,
		errorTxt,
	)
	action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, bookingModal)
	actions = append(actions, action)

	return actions
}
