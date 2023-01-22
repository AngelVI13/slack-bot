package parking

import (
	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

const (
	userActionId             = "userActionId"
	userBlockId              = "userBlockId"
	userOptionId             = "userOptionId"
	userCheckboxActionId     = "userCheckboxActionId"
	userRightsOption         = "userRightsOption"
	userPermanentSpaceOption = "userPermanentSpaceOption"
)

var usersManagementTitle = UsersIdentifier + "Settings"

func (m *Manager) generateUsersModalRequest(command event.Event, userId string) slack.ModalViewRequest {
	// TODO: is userId actually needed here ?
	// maybe it makes sense to disable user to change their own settings ?
	sectionBlocks := m.generateUsersBlocks(userId)
	// return common.GenerateInfoModalRequest(usersManagementTitle, sectionBlocks)
	return common.GenerateModalRequest(usersManagementTitle, sectionBlocks)
}

func (m *Manager) generateUsersBlocks(userId string) []slack.Block {
	allBlocks := []slack.Block{}

	text := "Select user for which to change settings"
	sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	allBlocks = append(allBlocks, sectionBlock)

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	userText := slack.NewTextBlockObject(slack.PlainTextType, "Invitee from static list", false, false)
	userOption := slack.NewOptionsSelectBlockElement(slack.OptTypeUser, userText, userActionId)
	userBlock := slack.NewInputBlock(userBlockId, userText, nil, userOption)

	allBlocks = append(allBlocks, userBlock)

	// NOTE:
	// Input with users select - this input will be included in the view_submission's view.state.values
	// It can be fetched as for example "payload.View.State.Values["user"]["user"].SelectedUser"

	var sectionBlocks []*slack.OptionBlockObject

	adminOptionSectionBlock := slack.NewOptionBlockObject(
		userRightsOption,
		slack.NewTextBlockObject("mrkdwn", "Admin", false, false),
		slack.NewTextBlockObject("mrkdwn", "Select to assign Admin rights.", false, false),
	)
	reviewerOptionSectionBlock := slack.NewOptionBlockObject(
		userPermanentSpaceOption,
		slack.NewTextBlockObject("mrkdwn", "Permanent Space", false, false),
		slack.NewTextBlockObject("mrkdwn", "Select to assign permanent space.", false, false),
	)

	sectionBlocks = append(sectionBlocks, adminOptionSectionBlock, reviewerOptionSectionBlock)

	deviceCheckboxGroup := slack.NewCheckboxGroupsBlockElement(userOptionId, sectionBlocks...)
	actionBlock := slack.NewActionBlock(userCheckboxActionId, deviceCheckboxGroup)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}
