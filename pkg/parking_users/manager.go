package parking_users

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
	"github.com/AngelVI13/slack-bot/pkg/model/user"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/slack-go/slack"
)

const (
	Identifier   = "Users: "
	SlashCmd     = "/users-parking"
	TestSlashCmd = "/test-users"

	defaultUserOption = ""
)

type selectedUser struct {
	UserId   string
	UserName string
}

type Manager struct {
	eventManager  *event.EventManager
	data          *model.Data
	slackClient   *slack.Client
	selectedUser  map[string]*selectedUser
	testingActive bool
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
	conf *config.Config,
) *Manager {
	return &Manager{
		eventManager:  eventManager,
		data:          data,
		selectedUser:  map[string]*selectedUser{},
		testingActive: conf.TestingActive,
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.SlashCmdEvent:
		data := e.(*slackApi.Slash)
		if !common.ShouldProcessSlash(
			data.Command,
			SlashCmd,
			TestSlashCmd,
			m.testingActive,
		) {
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

		response := m.handleViewSubmission(data)
		if response == nil {
			return
		}

		m.eventManager.Publish(response)
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	if !m.data.UserManager.IsAdminId(data.UserId) {
		errTxt := fmt.Sprintf(
			"You don't have permission to execute '%s' command",
			data.Command,
		)
		action := common.NewPostAction(data.UserId, errTxt, false)
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
			errTxt := ""
			clearedModal := m.generateUsersModalRequest(data, defaultUserOption)
			actions = append(actions, common.NewUpdateViewAction(
				data.TriggerId, data.ViewId, clearedModal, errTxt,
			))

			modalWithData := m.generateUsersModalRequest(data, selectedUserId)
			actions = append(actions, common.NewUpdateViewAction(
				data.TriggerId, data.ViewId, modalWithData, errTxt,
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
			if !m.data.UserManager.Exists(selectedUser.UserId) {
				m.data.UserManager.
					InsertUser(selectedUser.UserId, selectedUser.UserName)
			}

			m.data.UserManager.SetAccessRights(selectedUser.UserId, isAdmin)
			m.data.UserManager.
				SetParkingPermission(selectedUser.UserId, hasParkingSpace)
			m.data.UserManager.SynchronizeToFile()

			errTxt := ""
			modal := m.generateUsersModalRequest(data, selectedUser.UserId)
			actions = append(actions, common.NewUpdateViewAction(
				data.TriggerId, data.ViewId, modal, errTxt,
			))
		}
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
}

func (m *Manager) handleViewSubmission(data *slackApi.ViewSubmission) *common.Response {
	var actions []event.ResponseAction

	// Reset selected user
	m.selectedUser[data.UserId] = nil

	// NOTE: For parking rights changes take place as soon as user clicks checkbox
	// so we don't need to handle those on view submission

	qdevBss := strings.TrimSpace(data.IValueString(qdevBssBlockId, qdevBssActionId))
	quadBss := strings.TrimSpace(data.IValueString(quadBssBlockId, quadBssActionId))

	// TODO: currently we don't have possibility to remove the BSS ID, do we need this?
	if qdevBss != "" || quadBss != "" {
		if !m.data.UserManager.Exists(data.UserId) {
			m.data.UserManager.
				InsertUser(data.UserId, data.UserName)
		}

		if qdevBss != "" {
			m.data.UserManager.SetBssId(data.UserName, qdevBss, user.Qdev)
		}

		if quadBss != "" {
			m.data.UserManager.SetBssId(data.UserName, quadBss, user.Quad)
		}

		m.data.UserManager.SynchronizeToFile()
	}
	slog.Info(
		"USERS ViewSubmission",
		"name",
		data.UserName,
		"qdev",
		qdevBss,
		"quad",
		quadBss,
	)

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
}
