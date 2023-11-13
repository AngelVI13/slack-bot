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
)

var (
	floors             = [3]string{"4th floor", "5th floor", "7th floor"}
	defaultFloorOption = floors[0]
)

var workspaceBookingTitle = Identifier + "Booking"

func (m *Manager) generateBookingModalRequest(
	command event.Event,
	userId, selectedFloor, errorTxt string,
) slack.ModalViewRequest {
	spacesSectionBlocks := m.generateWorkspaceInfoBlocks(userId, selectedFloor, errorTxt)
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

func generateWorkspacePlanBlocks() []slack.Block {
	description := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"In the links below you can find the workspace plans for each floor so you can locate your workspace.",
			false,
			false,
		),
		nil,
		nil,
	)
	// TODO: replace links with actual workspace plans
	fourthFloorWorkspaces := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"<https://ibb.co/PFNyGsn|4th floor plan>",
			false,
			false,
		),
		nil,
		nil,
	)
	fifthFloorWorkspaces := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"<https://ibb.co/zHw2T9w|5th floor plan>",
			false,
			false,
		),
		nil,
		nil,
	)
	seventhFloorWorkspaces := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"<https://ibb.co/mt15xrz|7th floor plan>",
			false,
			false,
		),
		nil,
		nil,
	)

	now := time.Now()

	todayYear, todayMonth, todayDay := now.Date()
	if now.Hour() >= ResetHour && now.Minute() > ResetMin {
		todayDay++
	}

	selectionEffectTime := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf(
				"_Reservation is valid for %d-%d-%d and will be auto released at %d:%02d_",
				todayYear,
				todayMonth,
				todayDay,
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
		description,
		fourthFloorWorkspaces,
		fifthFloorWorkspaces,
		seventhFloorWorkspaces,
		selectionEffectTime,
	}
}

// generateWorkspaceInfoBlocks Generates space block objects to be used as elements in modal
func (m *Manager) generateWorkspaceInfoBlocks(
	userId, selectedFloor, errorTxt string,
) []slack.Block {
	allBlocks := []slack.Block{}

	descriptionBlocks := generateWorkspacePlanBlocks()
	allBlocks = append(allBlocks, descriptionBlocks...)

	floorOptionBlocks := m.generateFloorOptions(userId)
	allBlocks = append(allBlocks, floorOptionBlocks...)

	if errorTxt != "" {
		txt := fmt.Sprintf(`:warning: %s`, errorTxt)
		errorSection := slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", txt, false, false),
			nil,
			nil,
		)
		// TODO: this should be in red color
		allBlocks = append(allBlocks, errorSection)
	}

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	spaces := m.workspacesLot.GetSpacesByFloor(userId, selectedFloor)
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
