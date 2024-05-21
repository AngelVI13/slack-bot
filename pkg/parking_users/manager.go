package parking_users

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Users: "
	SlashCmd   = "/users-parking"
	// SlashCmd = "/test-users"

	defaultUserOption = ""
)

type selectedUser struct {
	UserId   string
	UserName string
}

type Manager struct {
	eventManager   *event.EventManager
	userManager    *user.Manager
	parkingManager *parking_spaces.Manager
	slackClient    *slack.Client
	selectedUser   map[string]*selectedUser
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
		selectedUser:   map[string]*selectedUser{},
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

		// Reset selected user
		m.selectedUser[data.UserId] = nil

		// Changes take place as soon as user clicks checkbox
		// There is nothing to do on view submission
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	if !m.userManager.IsAdminId(data.UserId) {
		errTxt := fmt.Sprintf(
			"You don't have permission to execute '%s' command",
			SlashCmd,
		)
		action := common.NewPostEphemeralAction(
			data.UserId,
			data.UserId,
			slack.MsgOptionText(errTxt, false),
		)
		return common.NewResponseEvent(data.UserName, action)
	}

	selectedUserId := defaultUserOption
	selectedUser, ok := m.selectedUser[data.UserId]
	if ok && selectedUser != nil {
		selectedUserId = selectedUser.UserId
	}
	modal := m.generateUsersModalRequest(data, selectedUserId)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(data.UserName, action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	var actions []event.ResponseAction

	if _, ok := m.selectedUser[data.UserId]; !ok {
		m.selectedUser[data.UserId] = nil
	}

	for _, action := range data.Actions {
		switch action.ActionID {
		case userActionId:
			selectedUserId := action.SelectedUser
			m.selectedUser[data.UserId] = &selectedUser{
				UserId:   selectedUserId,
				UserName: data.SelectedUserName,
			}

			// NOTE: In theory i should not need an empty modal but when we
			// click a checkbox and then select a different user slack fails
			// to update the existing checkboxes with their new values.
			// Thats why we update the view with a clean modal
			// and then just load the modal with actual data afterwards
			clearedModal := m.generateUsersModalRequest(data, defaultUserOption)
			actions = append(actions, common.NewUpdateViewAction(
				data.TriggerId, data.ViewId, clearedModal,
			))

			modalWithData := m.generateUsersModalRequest(data, selectedUserId)
			actions = append(actions, common.NewUpdateViewAction(
				data.TriggerId, data.ViewId, modalWithData,
			))
		case userOptionId:
			isAdmin := user.STANDARD
			hasParkingSpace := false

			for _, option := range action.SelectedOptions {
				if option.Value == userRightsOption {
					isAdmin = user.ADMIN
				}
				if option.Value == userPermanentSpaceOption {
					hasParkingSpace = true
				}
			}

			selectedUser := m.selectedUser[data.UserId]
			if !m.userManager.Exists(selectedUser.UserId) {
				m.userManager.InsertUser(selectedUser.UserId, selectedUser.UserName)
			}

			userHadParking := m.userManager.HasParkingById(selectedUser.UserId)

			m.userManager.SetAccessRights(selectedUser.UserId, isAdmin)
			m.userManager.SetParkingPermission(selectedUser.UserId, hasParkingSpace)
			m.userManager.SynchronizeToFile()

			// TODO: add automatic handling for these cases
			if userHadParking && !hasParkingSpace {
				// TODO: Make his space reservation to auto release
			} else if !userHadParking && hasParkingSpace {
				// TODO: make his space reservation to permanent
			}

			modal := m.generateUsersModalRequest(data, selectedUser.UserId)
			actions = append(actions, common.NewUpdateViewAction(
				data.TriggerId, data.ViewId, modal,
			))
		}
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
}
