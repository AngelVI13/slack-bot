package edit_parking_spaces

import (
	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

const (
	selectEditActionId = "selectEditActionId"
	selectEditOptionId = "selectEditOptionId"
)

type editOption string

const (
	notSelectedOption editOption = "Not Selected"
	addSpaceOption    editOption = "Add"
	removeSpaceOption editOption = "Remove"
)

var editOptions = [3]editOption{
	notSelectedOption,
	addSpaceOption,
	removeSpaceOption,
}

var usersManagementTitle = Identifier

func (m *Manager) generateEditSpacesModalRequest(
	command event.Event,
	userId string,
) slack.ModalViewRequest {
	sectionBlocks := m.generateAddRemoveBlocks(userId)
	return common.GenerateModalRequest(usersManagementTitle, sectionBlocks)
}

func (m *Manager) generateAddRemoveBlocks(userId string) []slack.Block {
	allBlocks := []slack.Block{}

	text := "Select user for which to change settings"
	sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	allBlocks = append(allBlocks, sectionBlock)

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	addRemoveOptionBlocks := m.generateAddRemoveOptions(userId)
	allBlocks = append(allBlocks, addRemoveOptionBlocks...)
	return allBlocks
}

func (m *Manager) generateAddRemoveOptions(userId string) []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionBlocks []*slack.OptionBlockObject

	for _, editOpt := range editOptions {
		optionBlock := slack.NewOptionBlockObject(
			string(editOpt),
			slack.NewTextBlockObject("plain_text", string(editOpt), false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	selectedOption := m.selectedEditOption.Get(userId)

	// Text shown as title when option box is opened/expanded
	optionLabel := slack.NewTextBlockObject(
		"plain_text",
		"Choose what spaces to show",
		false,
		false,
	)
	// Default option shown for option box
	defaultOption := slack.NewTextBlockObject(
		"plain_text",
		string(selectedOption),
		false,
		false,
	)

	optionGroupBlockObject := slack.NewOptionGroupBlockElement(
		optionLabel,
		optionBlocks...)
	newOptionsGroupSelectBlockElement := slack.NewOptionsGroupSelectBlockElement(
		"static_select",
		defaultOption,
		selectEditOptionId,
		optionGroupBlockObject,
	)

	actionBlock := slack.NewActionBlock(
		selectEditActionId,
		newOptionsGroupSelectBlockElement,
	)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}
