package workspaces

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces/views"
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
	showOptions     = [2]string{"Free", "Taken"}
	showFreeOption  = showOptions[0]
	showTakenOption = showOptions[1]
)

var workspaceBookingTitle = Identifier + "Booking"

func (m *Manager) generateBookingModalRequest(
	command event.Event,
	userId string, selectedFloorTaken bool, errorTxt string,
) slack.ModalViewRequest {
	spacesSectionBlocks := m.generateWorkspaceInfoBlocks(
		userId,
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

func (m *Manager) generateWorkspaceButtons(
	space *spaces.Space,
	userId string,
) []slack.BlockElement {
	var buttons []slack.BlockElement

	isAdminUser := m.data.UserManager.IsAdminId(userId)

	if space.Reserved && (space.ReservedById == userId || isAdminUser) {
		releaseButton := slack.NewButtonBlockElement(
			releaseWorkspaceActionId,
			views.ActionValues{SpaceKey: space.Key()}.Encode(),
			slack.NewTextBlockObject("plain_text", "Release!", true, false),
		)
		releaseButton = releaseButton.WithStyle(slack.StyleDanger)
		buttons = append(buttons, releaseButton)
	} else if (!space.Reserved &&
		!m.data.WorkspacesLot.HasSpace(userId) &&
		!isAdminUser) || (!space.Reserved && isAdminUser) {
		// Only allow user to reserve space if he hasn't already reserved one
		actionButtonText := "Reserve!"
		reserveWithAutoButton := slack.NewButtonBlockElement(
			reserveWorkspaceActionId,
			views.ActionValues{SpaceKey: space.Key()}.Encode(),
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

func (m *Manager) generateNoWorkspacesBlocks(userId string) []slack.Block {
	selectedChannel := m.selectedChannel[userId]
	floors := m.floorsForChannel(selectedChannel)

	noWorkspacesBlock := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf(
				"_No existing workspaces for the following floors: %v_",
				floors,
			),
			false,
			false,
		),
		nil,
		nil,
	)
	return []slack.Block{
		noWorkspacesBlock,
	}
}

func generateWorkspacePlanBlocks(selectedFloor string) []slack.Block {
	floorPlanLink := ""
	if strings.HasPrefix(selectedFloor, "4th") {
		floorPlanLink = "https://ibb.co/BHYTXW79"
	} else if strings.HasPrefix(selectedFloor, "6th") {
		floorPlanLink = "https://ibb.co/pBfBKfmK"
	} else if strings.HasPrefix(selectedFloor, "5th") {
		floorPlanLink = "https://ibb.co/wxjTP2w"
	} else if strings.HasPrefix(selectedFloor, "7th") {
		floorPlanLink = "https://ibb.co/vCq0y0cb"
	}

	if floorPlanLink == "" {
		return nil
	}

	floorPlan := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf("<%s|Workspace Plan for %s>", floorPlanLink, selectedFloor),
			false,
			false,
		),
		nil,
		nil,
	)
	return []slack.Block{floorPlan}
}

// generateWorkspaceInfoBlocks Generates space block objects to be used as elements in modal
func (m *Manager) generateWorkspaceInfoBlocks(
	userId string, selectedShowTaken bool, errorTxt string,
) []slack.Block {
	allBlocks := []slack.Block{}
	selectedFloor := m.selectedFloorByChannel(userId)

	workspacePlanBlocks := generateWorkspacePlanBlocks(selectedFloor)
	allBlocks = append(allBlocks, workspacePlanBlocks...)

	descriptionBlocks := generateWorkspaceTimeBlocks()
	allBlocks = append(allBlocks, descriptionBlocks...)

	floorOptionBlocks := m.generateFloorOptions(userId)
	if len(floorOptionBlocks) == 0 {
		return m.generateNoWorkspacesBlocks(userId)
	}
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

	selectedSpaceType := spaces.SpaceFree
	if selectedShowTaken {
		selectedSpaceType = spaces.SpaceTaken
	}
	spaces := m.data.WorkspacesLot.
		GetSpacesByFloor(userId, selectedFloor, selectedSpaceType)
	workspaceSections := m.generateSpacesInfo(spaces)

	for idx, space := range spaces {
		sectionBlock := workspaceSections[idx]
		buttons := m.generateWorkspaceButtons(space, userId)

		allBlocks = append(allBlocks, sectionBlock)
		if len(buttons) > 0 {
			actions := slack.NewActionBlock("", buttons...)
			allBlocks = append(allBlocks, actions)
		}
		allBlocks = append(allBlocks, div)
	}

	return allBlocks
}

// TODO: this has a bad name. It is using userId to find out what channel the
// person selected and based on that to return his selected floor
func (m *Manager) selectedFloorByChannel(userId string) string {
	selectedChannel := m.selectedChannel[userId]
	allowedFloors := m.floorsForChannel(selectedChannel)
	allFloors := m.data.WorkspacesLot.GetExistingFloors(allowedFloors)

	selectedFloor := m.defaultFloorOption(selectedChannel)

	selected, ok := m.selectedFloor[userId]
	if ok && slices.Contains(allFloors, selected) {
		selectedFloor = selected
	}

	return selectedFloor
}

func (m *Manager) generateFloorOptions(userId string) []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionBlocks []*slack.OptionBlockObject

	selectedChannel := m.selectedChannel[userId]
	allowedFloors := m.floorsForChannel(selectedChannel)
	allFloors := m.data.WorkspacesLot.GetExistingFloors(allowedFloors)
	if len(allFloors) == 0 {
		return allBlocks
	}

	for _, floor := range allFloors {
		optionBlock := slack.NewOptionBlockObject(
			floor,
			slack.NewTextBlockObject("plain_text", floor, false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	selectedFloor := m.defaultFloorOption(selectedChannel)

	selected, ok := m.selectedFloor[userId]
	if ok && slices.Contains(allFloors, selected) {
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
