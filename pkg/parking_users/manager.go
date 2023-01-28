package parking_users

import (
	"fmt"
	"log"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Users: "
	// SlashCmd   = "/users"
	SlashCmd = "/test-users"

	defaultUserOption = ""
)

type Manager struct {
	eventManager   *event.EventManager
	userManager    *user.Manager
	parkingManager *parking_spaces.Manager
	selectedUser   map[string]string
}

func NewManager(
	eventManager *event.EventManager,
	userManager *user.Manager,
	parkingManager *parking_spaces.Manager,
) *Manager {
	return &Manager{
		eventManager:   eventManager,
		userManager:    userManager,
		parkingManager: parkingManager,
		selectedUser:   map[string]string{},
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

	case event.ViewSubmissionEvent:
		data := e.(*slackApi.ViewSubmission)

		if data.Title != usersManagementTitle {
			return
		}

		// Changes take place as soon as user clicks checkbox
		// There is nothing to do on view submission
		log.Println("Modal closed")
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	if !m.userManager.IsAdminId(data.UserId) {
		errTxt := fmt.Sprintf("You don't have permission to execute '%s' command", SlashCmd)
		action := common.NewPostEphemeralAction(data.UserId, data.UserId, slack.MsgOptionText(errTxt, false))
		return common.NewResponseEvent(action)
	}

	selectedUser, ok := m.selectedUser[data.UserId]
	if !ok {
		selectedUser = defaultUserOption
	}
	modal := m.generateUsersModalRequest(data, selectedUser)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	var actions []event.ResponseAction

	if _, ok := m.selectedUser[data.UserId]; !ok {
		m.selectedUser[data.UserId] = defaultUserOption
	}

	for _, action := range data.Actions {
		switch action.ActionID {
		case userActionId:
			m.selectedUser[data.UserId] = action.SelectedUser

			modal := m.generateUsersModalRequest(data, action.SelectedUser)
			actions = append(actions, common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal))
		}
	}

	if actions == nil || len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(actions...)
}
