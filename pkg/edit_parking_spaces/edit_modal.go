package edit_parking_spaces

import (
	"log"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

const (
	selectEditOptionId  = "selectEditOptionId"
	selectSpaceOptionId = "selectSpaceOptionId"
)

type editOption string

const (
	notSelectedOption editOption = "Not Selected"
	addSpaceOption    editOption = "Add Space"
	removeSpaceOption editOption = "Remove Space/s"
)

var editOptions = []editOption{
	addSpaceOption,
	removeSpaceOption,
}

var parkSpaceManagementTitle = Identifier

func (m *Manager) generateEditSpacesModalRequest(
	command event.Event,
	userId string,
) slack.ModalViewRequest {
	sectionBlocks := m.generateAddRemoveBlocks(userId)
	return common.GenerateModalRequest(parkSpaceManagementTitle, sectionBlocks)
}

func (m *Manager) generateAddRemoveBlocks(userId string) []slack.Block {
	allBlocks := []slack.Block{}

	selectedOption := m.selectedEditOption.Get(userId)

	addRemoveOptionBlocks := m.generateAddRemoveOptions(selectedOption)
	allBlocks = append(allBlocks, addRemoveOptionBlocks...)

	switch selectedOption {
	case addSpaceOption:
		allBlocks = append(allBlocks, m.generateAddSpaceBlocks()...)
	case removeSpaceOption:
		allBlocks = append(allBlocks, m.generateRemoveSpaceBlocks()...)
	case notSelectedOption:
		// do nothing
	default:
		log.Fatalf("Unsupported edit parking space option: %q", selectedOption)
	}
	return allBlocks
}

func (m *Manager) generateAddRemoveOptions(selectedOption editOption) []slack.Block {
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

	// Text shown as title when option box is opened/expanded
	optionLabel := slack.NewTextBlockObject(
		"plain_text",
		"Choose an action",
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

	accessory := slack.NewAccessory(newOptionsGroupSelectBlockElement)

	text := "Select operation you want to perform"
	sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, accessory)

	allBlocks = append(allBlocks, sectionBlock)

	return allBlocks
}

func (m *Manager) generateAddSpaceBlocks() []slack.Block {
	var allBlocks []slack.Block

	text := "Add space"
	sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	allBlocks = append(allBlocks, sectionBlock)

	return allBlocks
}

func (m *Manager) generateRemoveSpaceBlocks() []slack.Block {
	var allBlocks []slack.Block

	allBlocks = append(allBlocks, m.generateSelectSpaceOptions()...)

	return allBlocks
}

func (m *Manager) generateSpaceOptionsByFloor(
	floor string,
) *slack.OptionGroupBlockObject {
	// Options
	var optionBlocks []*slack.OptionBlockObject

	onlyTaken := false
	userId := "" // we don't care about spaces that belong to a user

	// NOTE: slack only supports 100 elements in each floor group
	for _, space := range m.data.ParkingLot.GetSpacesByFloor(userId, floor, onlyTaken) {
		spaceKey := space.Key()
		optionBlock := slack.NewOptionBlockObject(
			string(spaceKey),
			slack.NewTextBlockObject("plain_text", string(spaceKey), false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	floorLabel := slack.NewTextBlockObject("plain_text", floor, false, false)
	optionsGroup := slack.NewOptionGroupBlockElement(floorLabel, optionBlocks...)
	return optionsGroup
}

func (m *Manager) generateSelectSpaceOptions() []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionGroups []*slack.OptionGroupBlockObject

	for _, floor := range m.data.ParkingLot.GetAllFloors() {
		optionGroup := m.generateSpaceOptionsByFloor(floor)
		optionGroups = append(optionGroups, optionGroup)
	}

	// Default option shown for option box
	defaultOption := slack.NewTextBlockObject(
		"plain_text",
		"Select space to remove",
		false,
		false,
	)

	newOptionsGroupSelectBlockElement := slack.NewOptionsGroupMultiSelectBlockElement(
		"multi_static_select",
		defaultOption,
		selectSpaceOptionId,
		optionGroups...,
	)

	accessory := slack.NewAccessory(newOptionsGroupSelectBlockElement)

	text := "Select spaces you want to remove"
	sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, accessory)

	allBlocks = append(allBlocks, sectionBlock)

	return allBlocks
}
