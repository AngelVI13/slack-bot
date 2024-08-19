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

var usersManagementTitle = Identifier + "Settings"

func (m *Manager) generateUsersModalRequest(
	command event.Event,
	selectedUserId string,
) slack.ModalViewRequest {
	sectionBlocks := m.generateUsersBlocks(selectedUserId)
	return common.GenerateModalRequest(usersManagementTitle, sectionBlocks)
}

func (m *Manager) generateUsersBlocks(selectedUserId string) []slack.Block {
	allBlocks := []slack.Block{}

	text := "Select user for which to change settings"
	sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	allBlocks = append(allBlocks, sectionBlock)

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	userText := slack.NewTextBlockObject(slack.PlainTextType, "User", false, false)
	userOption := slack.NewOptionsSelectBlockElement(
		slack.OptTypeUser,
		userText,
		userActionId,
	)
	if selectedUserId != defaultUserOption {
		userOption.InitialUser = selectedUserId
	}

	userSection := slack.NewSectionBlock(userText, nil, slack.NewAccessory(userOption))

	allBlocks = append(allBlocks, userSection)

	// Do not add checkboxes if user is not selected
	if selectedUserId == defaultUserOption {
		return allBlocks
	}

	warningText := ":warning: *Before changing parking rights " +
		"make sure the user has NOT booked any parking space!*"
	warningSectionText := slack.NewTextBlockObject("mrkdwn", warningText, false, false)
	warningSectionBlock := slack.NewSectionBlock(warningSectionText, nil, nil)
	allBlocks = append(allBlocks, warningSectionBlock)

	var sectionBlocks []*slack.OptionBlockObject

	adminOptionSectionBlock := slack.NewOptionBlockObject(
		userRightsOption,
		slack.NewTextBlockObject("mrkdwn", "Admin", false, false),
		slack.NewTextBlockObject(
			"mrkdwn",
			"Select to assign Admin rights.",
			false,
			false,
		),
	)
	hasParkingOptionSectionBlock := slack.NewOptionBlockObject(
		userPermanentSpaceOption,
		slack.NewTextBlockObject("mrkdwn", "Permanent Space", false, false),
		slack.NewTextBlockObject(
			"mrkdwn",
			"Select to assign permanent space.",
			false,
			false,
		),
	)

	sectionBlocks = append(
		sectionBlocks,
		adminOptionSectionBlock,
		hasParkingOptionSectionBlock,
	)

	deviceCheckboxGroup := slack.NewCheckboxGroupsBlockElement(
		userOptionId,
		sectionBlocks...)

	if selectedUserId != defaultUserOption &&
		m.data.UserManager.IsAdminId(selectedUserId) {
		deviceCheckboxGroup.InitialOptions = append(
			deviceCheckboxGroup.InitialOptions, adminOptionSectionBlock,
		)
	}

	if selectedUserId != defaultUserOption &&
		m.data.UserManager.HasParkingById(selectedUserId) {
		deviceCheckboxGroup.InitialOptions = append(
			deviceCheckboxGroup.InitialOptions, hasParkingOptionSectionBlock,
		)
	}

	actionBlock := slack.NewActionBlock(userCheckboxActionId, deviceCheckboxGroup)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}
