package parking_users

import (
	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

const (
	userPreffix              = "user"
	userActionId             = userPreffix + "ActionId"
	userBlockId              = userPreffix + "BlockId"
	userOptionId             = userPreffix + "OptionId"
	userCheckboxActionId     = userPreffix + "CheckboxActionId"
	userRightsOption         = userPreffix + "RightsOption"
	userPermanentSpaceOption = userPreffix + "PermanentSpaceOption"
)

var usersManagementTitle = UsersIdentifier + "Settings"

func (m *Manager) generateUsersModalRequest(command event.Event, userId string) slack.ModalViewRequest {
	// TODO: is userId actually needed here ?
	// maybe it makes sense to disable user to change their own settings ?
	sectionBlocks := m.generateUsersBlocks(userId)
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

	userText := slack.NewTextBlockObject(slack.PlainTextType, "User", false, false)
	userOption := slack.NewOptionsSelectBlockElement(slack.OptTypeUser, userText, userActionId)
	userSection := slack.NewSectionBlock(userText, nil, slack.NewAccessory(userOption))

	allBlocks = append(allBlocks, userSection)

	// TODO: 1. only show checkboxes after user has been selected
	// TODO: 2. their values should be taken from db/json files and prefilled
	// TODO: 3. update value in db/json as soon as user selects/deselects a checkbox
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
