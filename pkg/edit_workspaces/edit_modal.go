package edit_workspaces

import (
	"log"
	"slices"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
	"github.com/slack-go/slack"
)

const (
	selectEditOptionId  = "selectEditOptionId"
	selectSpaceOptionId = "selectSpaceOptionId"
	addFloorActionId    = "addFloorActionId"
	addFloorBlockId     = "addFloorBlockId"
	addSpaceActionId    = "addSpaceActionId"
	addSpaceBlockId     = "addSpaceBlockId"
	addDescActionId     = "addDescActionId"
	addDescBlockId      = "addDescBlockId"
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

var workSpaceManagementTitle = Identifier

// NOTE: this file is a copy of edit_parking_spaces package
func (m *Manager) generateEditSpacesModalRequest(
	command event.Event,
	userId string,
) slack.ModalViewRequest {
	sectionBlocks := m.generateAddRemoveBlocks(userId)
	return common.GenerateModalRequest(workSpaceManagementTitle, sectionBlocks)
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
		log.Fatalf("Unsupported edit workspace option: %q", selectedOption)
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

	// Floor Input
	floorPlaceholder := slack.NewTextBlockObject("plain_text", "4", false, false)
	isDecimalAllowed := false
	floorInput := slack.NewNumberInputBlockElement(
		floorPlaceholder,
		addFloorActionId,
		isDecimalAllowed,
	)
	floorInput.MinValue = "2"
	floorInput.MaxValue = "8"
	floorLabel := slack.NewTextBlockObject("plain_text", "Floor", false, false)
	floorHint := slack.NewTextBlockObject(
		"plain_text",
		"The floor where the workspace is located",
		false,
		false,
	)

	floorInputBlock := slack.NewInputBlock(
		addFloorBlockId,
		floorLabel,
		floorHint,
		floorInput,
	)
	allBlocks = append(allBlocks, floorInputBlock)

	// Space Number Input
	numberPlaceholder := slack.NewTextBlockObject("plain_text", "7", false, false)
	numberInput := slack.NewNumberInputBlockElement(
		numberPlaceholder,
		addSpaceActionId,
		isDecimalAllowed,
	)
	numberInput.MinValue = "1"
	numberInput.MaxValue = "100"
	numberLabel := slack.NewTextBlockObject(
		"plain_text",
		"Workspace Number",
		false,
		false,
	)
	numberHint := slack.NewTextBlockObject(
		"plain_text",
		"Don't forget to add a sticker or sign to the workspace",
		false,
		false,
	)

	numberInputBlock := slack.NewInputBlock(
		addSpaceBlockId,
		numberLabel,
		numberHint,
		numberInput,
	)
	allBlocks = append(allBlocks, numberInputBlock)

	// Description
	descPlaceholder := slack.NewTextBlockObject(
		"plain_text",
		"Verification Room",
		false,
		false,
	)
	descInput := slack.NewPlainTextInputBlockElement(descPlaceholder, addDescActionId)
	descLabel := slack.NewTextBlockObject(
		"plain_text",
		"Description/Space location",
		false,
		false,
	)
	descHint := slack.NewTextBlockObject(
		"plain_text",
		"For example, 'Verification Room' or 'Development Room'. Keep it short.",
		false,
		false,
	)

	descInputBlock := slack.NewInputBlock(
		addDescBlockId,
		descLabel,
		descHint,
		descInput,
	)
	allBlocks = append(allBlocks, descInputBlock)

	return allBlocks
}

func (m *Manager) generateRemoveSpaceBlocks() []slack.Block {
	return m.generateSelectSpaceOptions()
}

func (m *Manager) generateSpaceOptionsByFloor(
	floor string,
) *slack.OptionGroupBlockObject {
	// Options
	var optionBlocks []*slack.OptionBlockObject

	userId := "" // we don't care about spaces that belong to a specific user

	allSpaces := m.data.WorkspacesLot.GetSpacesByFloor(userId, floor, spaces.SpaceAny)
	slices.SortFunc(allSpaces, // sort spaces based on their number
		func(a, b *spaces.Space) int {
			if a.Number <= b.Number {
				return -1
			}
			return 1
		})

	// NOTE: slack only supports 100 elements in each floor group
	for _, space := range allSpaces {
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

	for _, floor := range m.data.WorkspacesLot.GetAllFloors() {
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
