package workspaces

import (
	"log/slog"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Workspaces: "
	SlashCmd   = "/workspace"
	// SlashCmd    = "/test-workspace"
	ChannelName = "qdev_technologies"

	defaultUserOption = ""

	ResetWorkspaces = "Reset workspaces status"
	ResetHour       = 17
	ResetMin        = 0
)

type Manager struct {
	eventManager  *event.EventManager
	userManager   *user.Manager
	workspacesLot *spaces.SpacesLot
	slackClient   *slack.Client

	selectedFloor     map[string]string
	selectedShowTaken map[string]bool
}

func NewManager(
	eventManager *event.EventManager,
	userManager *user.Manager,
	filename string,
) *Manager {
	worspacesLot := spaces.GetSpacesLot(filename)
	return &Manager{
		eventManager:      eventManager,
		userManager:       userManager,
		workspacesLot:     &worspacesLot,
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

		if data.ChannelName != ChannelName {
			// command is only allowed in a specific channel
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

		slog.Info("ReleaseWorkspaces")
		m.workspacesLot.ReleaseSpaces(data.Time)
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

		case reserveWorkspaceActionId:
			workSpace := spaces.SpaceKey(action.Value)
			actions = m.handleReserveWorkspace(
				data,
				workSpace,
				m.selectedFloor[data.UserId],
				m.selectedShowTaken[data.UserId],
			)

		case releaseWorkspaceActionId:
			workSpace := spaces.SpaceKey(action.Value)
			actions = m.handleReleaseWorkspace(
				data,
				workSpace,
				m.selectedFloor[data.UserId],
				m.selectedShowTaken[data.UserId],
			)
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

func (m *Manager) handleReserveWorkspace(
	data *slackApi.BlockAction,
	workSpace spaces.SpaceKey,
	selectedFloor string,
	selectedShowTaken bool,
) []event.ResponseAction {
	autoRelease := true // by default workspace reservation is always with auto release

	errStr := m.workspacesLot.Reserve(
		workSpace,
		data.UserName,
		data.UserId,
		autoRelease,
	)

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

func (m *Manager) handleReleaseWorkspace(
	data *slackApi.BlockAction,
	workSpace spaces.SpaceKey,
	selectedFloor string,
	selectedShowTaken bool,
) []event.ResponseAction {
	actions := []event.ResponseAction{}

	// Handle general case: normal user releasing a space
	victimId, errStr := m.workspacesLot.Release(workSpace, data.UserName, data.UserId)
	if victimId != "" {
		slog.Info(errStr)
		action := common.NewPostEphemeralAction(victimId, victimId, errStr, false)
		actions = append(actions, action)
	}

	// Only remove release info from a space if an Admin is permanently releasing the space
	if m.userManager.IsAdminId(data.UserId) {
		ok := m.workspacesLot.ToBeReleased.Remove(workSpace)
		if !ok {
			slog.Error("Failed to remove release info", "space", workSpace)
		}
	}

	errTxt := ""
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
