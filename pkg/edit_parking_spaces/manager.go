package edit_parking_spaces

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
	Identifier = "Edit Parking Spaces: "
	// SlashCmd   = "/spaces-parking"
	SlashCmd = "/test-spaces"

	defaultUserOption = ""
)

type selectedUser struct {
	UserId   string
	UserName string
}

type SelectedEditOptionMap map[string]editOption

// TODO: do any of the below methods copy or mutate the map????
func (m SelectedEditOptionMap) Get(userId string) editOption {
	val, found := m[userId]
	if !found {
		return notSelectedOption
	}
	return val
}

func (m SelectedEditOptionMap) ResetSelectionForUser(userId string) {
	m[userId] = notSelectedOption
}

type Manager struct {
	eventManager       *event.EventManager
	userManager        *user.Manager
	parkingManager     *parking_spaces.Manager
	slackClient        *slack.Client
	selectedEditOption SelectedEditOptionMap
}

func NewManager(
	eventManager *event.EventManager,
	userManager *user.Manager,
	parkingManager *parking_spaces.Manager,
) *Manager {
	return &Manager{
		eventManager:       eventManager,
		userManager:        userManager,
		parkingManager:     parkingManager,
		selectedEditOption: SelectedEditOptionMap{},
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
		m.selectedEditOption.ResetSelectionForUser(data.UserId)

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
		action := common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false)
		return common.NewResponseEvent(data.UserName, action)
	}

	modal := m.generateEditSpacesModalRequest(data, data.UserId)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(data.UserName, action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	var actions []event.ResponseAction

	for _, action := range data.Actions {
		switch action.ActionID {
		}
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
}
