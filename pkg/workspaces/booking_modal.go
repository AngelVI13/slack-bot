package workspaces

import (
	"fmt"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/slack-go/slack"
)

const (
	floorActionId            = "workspaceFloorActionId"
	floorOptionId            = "workspaceFloorOptionId"
	reserveWorkspaceActionId = "reserveWorkspace"
	releaseWorkspaceActionId = "releaseWorkspace"
	showActionId             = "showActionId"
	showOptionId             = "showOptionId"
)

var (
	floors             = [...]string{"4th floor"}
	defaultFloorOption = floors[0]
	showOptions        = [2]string{"Free", "Taken"}
	showFreeOption     = showOptions[0]
	showTakenOption    = showOptions[1]
)

var workspaceBookingTitle = Identifier + "Booking"

func (m *Manager) generateBookingModalRequest(
	command event.Event,
	userId, selectedFloor string, selectedFloorTaken bool, errorTxt string,
) slack.ModalViewRequest {
	spacesSectionBlocks := m.generateWorkspaceInfoBlocks(
		userId,
		selectedFloor,
		selectedFloorTaken,
		errorTxt,
	)
	return common.GenerateInfoModalRequest(workspaceBookingTitle, spacesSectionBlocks)
}

// generateSpacesInfo Generate sections of text that contain space info such as status (taken/free), taken by etc..
func (m *Manager) generateSpacesInfo(spaces spaces.SpacesInfo) []slack.Block {
	var sections []slack.Block
	for _, space := range spaces {
		status := space.GetStatusDescription()
		emoji := space.GetStatusEmoji()

		spaceProps := space.GetPropsText()
		text := fmt.Sprintf(
			"%s *%s* \t%s\t %s",
			emoji,
			fmt.Sprint(space.Number),
			spaceProps,
			status,
		)

		sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
		sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)

		sections = append(sections, *sectionBlock)

	}
	return sections
}

func (m *Manager) generateParkingButtons(
	space *spaces.Space,
	userId string,
) []slack.BlockElement {
	var buttons []slack.BlockElement

	isAdminUser := m.userManager.IsAdminId(userId)

	if space.Reserved && (space.ReservedById == userId || isAdminUser) {
		releaseButton := slack.NewButtonBlockElement(
			releaseWorkspaceActionId,
			string(space.Key()),
			slack.NewTextBlockObject("plain_text", "Release!", true, false),
		)
		releaseButton = releaseButton.WithStyle(slack.StyleDanger)
		buttons = append(buttons, releaseButton)
	} else if (!space.Reserved &&
		!m.workspacesLot.HasSpace(userId) &&
		!isAdminUser) || (!space.Reserved && isAdminUser) {
		// Only allow user to reserve space if he hasn't already reserved one
		actionButtonText := "Reserve!"
		reserveWithAutoButton := slack.NewButtonBlockElement(
			reserveWorkspaceActionId,
			string(space.Key()),
			slack.NewTextBlockObject("plain_text", actionButtonText, true, false),
		)
		reserveWithAutoButton = reserveWithAutoButton.WithStyle(slack.StylePrimary)
		buttons = append(buttons, reserveWithAutoButton)
	}
	return buttons
}

func generateWorkspaceTimeBlocks() []slack.Block {
	now := time.Now()

	if now.Hour() >= ResetHour && now.Minute() > ResetMin {
		now = now.Add(24 * time.Hour)
	}

	selectionEffectTime := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf(
				"_Reservation is valid for %d-%d-%d and will be auto released at %d:%02d_",
				now.Year(),
				now.Month(),
				now.Day(),
				ResetHour,
				ResetMin,
			),
			false,
			false,
		),
		nil,
		nil,
	)
	return []slack.Block{
		selectionEffectTime,
	}
}

// generateWorkspaceInfoBlocks Generates space block objects to be used as elements in modal
func (m *Manager) generateWorkspaceInfoBlocks(
	userId, selectedFloor string, selectedShowTaken bool, errorTxt string,
) []slack.Block {
	allBlocks := []slack.Block{}

	descriptionBlocks := generateWorkspaceTimeBlocks()
	allBlocks = append(allBlocks, descriptionBlocks...)

	floorOptionBlocks := m.generateFloorOptions(userId)
	allBlocks = append(allBlocks, floorOptionBlocks...)

	showOptionBlocks := m.generateFreeTakenOptions(userId)
	allBlocks = append(allBlocks, showOptionBlocks...)

	if errorTxt != "" {
		txt := fmt.Sprintf(`:warning: %s`, errorTxt)
		errorSection := slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", txt, false, false),
			nil,
			nil,
		)
		allBlocks = append(allBlocks, errorSection)
	}

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	spaces := m.workspacesLot.GetSpacesByFloor(userId, selectedFloor, selectedShowTaken)
	workspaceSections := m.generateSpacesInfo(spaces)

	for idx, space := range spaces {
		sectionBlock := workspaceSections[idx]
		buttons := m.generateParkingButtons(space, userId)

		allBlocks = append(allBlocks, sectionBlock)
		if len(buttons) > 0 {
			actions := slack.NewActionBlock("", buttons...)
			allBlocks = append(allBlocks, actions)
		}
		allBlocks = append(allBlocks, div)
	}

	return allBlocks
}

func (m *Manager) generateFloorOptions(userId string) []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionBlocks []*slack.OptionBlockObject

	for _, floor := range floors {
		optionBlock := slack.NewOptionBlockObject(
			floor,
			slack.NewTextBlockObject("plain_text", floor, false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	selectedFloor := defaultFloorOption
	selected, ok := m.selectedFloor[userId]
	if ok {
		selectedFloor = selected
	}

	// Text shown as title when option box is opened/expanded
	optionLabel := slack.NewTextBlockObject(
		"plain_text",
		"Choose a workspace floor",
		false,
		false,
	)
	// Default option shown for option box
	defaultOption := slack.NewTextBlockObject("plain_text", selectedFloor, false, false)

	optionGroupBlockObject := slack.NewOptionGroupBlockElement(
		optionLabel,
		optionBlocks...)
	newOptionsGroupSelectBlockElement := slack.NewOptionsGroupSelectBlockElement(
		"static_select",
		defaultOption,
		floorOptionId,
		optionGroupBlockObject,
	)

	actionBlock := slack.NewActionBlock(floorActionId, newOptionsGroupSelectBlockElement)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}

func (m *Manager) generateFreeTakenOptions(userId string) []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionBlocks []*slack.OptionBlockObject

	for _, showOption := range showOptions {
		optionBlock := slack.NewOptionBlockObject(
			showOption,
			slack.NewTextBlockObject("plain_text", showOption, false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	selectedOption := showFreeOption
	showTaken := m.selectedShowTaken[userId]
	if showTaken {
		selectedOption = showTakenOption
	}

	// Text shown as title when option box is opened/expanded
	optionLabel := slack.NewTextBlockObject(
		"plain_text",
		"Choose what spaces to show",
		false,
		false,
	)
	// Default option shown for option box
	defaultOption := slack.NewTextBlockObject("plain_text", selectedOption, false, false)

	optionGroupBlockObject := slack.NewOptionGroupBlockElement(
		optionLabel,
		optionBlocks...)
	newOptionsGroupSelectBlockElement := slack.NewOptionsGroupSelectBlockElement(
		"static_select",
		defaultOption,
		showOptionId,
		optionGroupBlockObject,
	)

	actionBlock := slack.NewActionBlock(showActionId, newOptionsGroupSelectBlockElement)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}
