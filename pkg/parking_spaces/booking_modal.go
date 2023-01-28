package parking_spaces

import (
	"fmt"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

const (
	floorActionId                    = "floorActionId"
	floorOptionId                    = "floorOptionId"
	reserveParkingActionId           = "reserveParking"
	releaseParkingActionId           = "releaseParking"
	tempReleaseParkingActionId       = "tempReleaseParking"
	cancelTempReleaseParkingActionId = "cancelTempReleaseParking"
)

var (
	floors             = [3]string{"-2nd floor", "-1st floor", "1st floor"}
	defaultFloorOption = floors[0]
)

var parkingBookingTitle = Identifier + "Booking"

func (m *Manager) generateBookingModalRequest(command event.Event, userId, selectedFloor, errorTxt string) slack.ModalViewRequest {
	spacesSectionBlocks := m.generateParkingInfoBlocks(userId, selectedFloor, errorTxt)
	return common.GenerateInfoModalRequest(parkingBookingTitle, spacesSectionBlocks)
}

// generateParkingInfo Generate sections of text that contain space info such as status (taken/free), taken by etc..
func (m *Manager) generateParkingInfo(spaces SpacesInfo) []slack.Block {
	var sections []slack.Block
	for _, space := range spaces {
		status := space.GetStatusDescription()
		emoji := space.GetStatusEmoji()

		releaseScheduled := ""
		releaseInfo := m.parkingLot.ToBeReleased.Get(space.ParkingKey())
		if releaseInfo != nil {
			releaseScheduled = fmt.Sprintf(
				"\n\t\tScheduled release from %s to %s",
				releaseInfo.StartDate.Format("2006-01-02"),
				releaseInfo.EndDate.Format("2006-01-02"),
			)
		}

		spaceProps := space.GetPropsText()
		text := fmt.Sprintf(
			"%s *%s* \t%s\t %s%s",
			emoji,
			fmt.Sprint(space.Number),
			spaceProps,
			status,
			releaseScheduled,
		)

		sectionText := slack.NewTextBlockObject("mrkdwn", text, false, false)
		sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)

		sections = append(sections, *sectionBlock)

	}
	return sections
}

func (m *Manager) generateParkingButtons(
	space *ParkingSpace,
	userId string,
) []slack.BlockElement {
	var buttons []slack.BlockElement

	isAdminUser := m.userManager.IsAdminId(userId)
	hasPermanentParkingUser := m.userManager.HasParkingById(userId)

	releaseInfo := m.parkingLot.ToBeReleased.Get(space.ParkingKey())
	if releaseInfo != nil && (releaseInfo.OwnerId == userId || isAdminUser) && !releaseInfo.Cancelled {
		cancelTempReleaseButton := slack.NewButtonBlockElement(
			cancelTempReleaseParkingActionId,
			string(space.ParkingKey()),
			slack.NewTextBlockObject("plain_text", "Cancel Scheduled Release!", true, false),
		)
		cancelTempReleaseButton = cancelTempReleaseButton.WithStyle(slack.StyleDanger)
		buttons = append(buttons, cancelTempReleaseButton)
	}

	if space.Reserved && (space.ReservedById == userId || isAdminUser) {
		// space reserved but hasn't yet been schedule for release
		if (isAdminUser || hasPermanentParkingUser) && releaseInfo == nil {
			permanentSpace := m.userManager.HasParkingById(space.ReservedById)
			if permanentSpace {
				// Only allow the temporary parking button if the correct user is using
				// the modal and the space hasn't already been released.
				// For example, an admin can only temporary release a space if either he
				// owns the space & has permanent parking rights or if he is releasing
				// the space on behalf of somebody that has a permanent parking rights
				tempReleaseButton := slack.NewButtonBlockElement(
					tempReleaseParkingActionId,
					string(space.ParkingKey()),
					slack.NewTextBlockObject("plain_text", "Temp Release!", true, false),
				)
				tempReleaseButton = tempReleaseButton.WithStyle(slack.StyleDanger)
				buttons = append(buttons, tempReleaseButton)
			}
		}

		if isAdminUser || !hasPermanentParkingUser {
			releaseButton := slack.NewButtonBlockElement(
				releaseParkingActionId,
				string(space.ParkingKey()),
				slack.NewTextBlockObject("plain_text", "Release!", true, false),
			)
			releaseButton = releaseButton.WithStyle(slack.StyleDanger)
			buttons = append(buttons, releaseButton)
		}
	} else if (!space.Reserved &&
		!m.parkingLot.HasSpace(userId) &&
		!m.parkingLot.HasTempRelease(userId) &&
		!isAdminUser) || (!space.Reserved && isAdminUser) {
		// Only allow user to reserve space if he hasn't already reserved one
		actionButtonText := "Reserve!"
		reserveWithAutoButton := slack.NewButtonBlockElement(
			reserveParkingActionId,
			string(space.ParkingKey()),
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("%s :eject:", actionButtonText), true, false),
		)
		reserveWithAutoButton = reserveWithAutoButton.WithStyle(slack.StylePrimary)
		buttons = append(buttons, reserveWithAutoButton)
	}
	return buttons
}

func generateParkingPlanBlocks() []slack.Block {
	description := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"In the links below you can find the parking plans for each floor so you can locate your parking space.",
			false,
			false,
		),
		nil,
		nil,
	)
	outsideParking := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "<https://ibb.co/PFNyGsn|1st floor plan>", false, false),
		nil,
		nil,
	)
	minusOneParking := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "<https://ibb.co/zHw2T9w|-1st floor plan>", false, false),
		nil,
		nil,
	)
	minusTwoParking := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "<https://ibb.co/mt15xrz|-2nd floor plan>", false, false),
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
			fmt.Sprintf("_Reservation is valid for %d-%d-%d_", todayYear, todayMonth, todayDay),
			false,
			false,
		),
		nil,
		nil,
	)
	return []slack.Block{
		description,
		outsideParking,
		minusOneParking,
		minusTwoParking,
		selectionEffectTime,
	}
}

// generateParkingInfoBlocks Generates space block objects to be used as elements in modal
func (m *Manager) generateParkingInfoBlocks(userId, selectedFloor, errorTxt string) []slack.Block {
	allBlocks := []slack.Block{}

	descriptionBlocks := generateParkingPlanBlocks()
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

	spaces := m.parkingLot.GetSpacesByFloor(userId, selectedFloor)
	parkingSpaceSections := m.generateParkingInfo(spaces)

	for idx, space := range spaces {
		sectionBlock := parkingSpaceSections[idx]
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
	optionLabel := slack.NewTextBlockObject("plain_text", "Choose a parking floor", false, false)
	// Default option shown for option box
	defaultOption := slack.NewTextBlockObject("plain_text", selectedFloor, false, false)

	optionGroupBlockObject := slack.NewOptionGroupBlockElement(optionLabel, optionBlocks...)
	newOptionsGroupSelectBlockElement := slack.NewOptionsGroupSelectBlockElement("static_select", defaultOption, floorOptionId, optionGroupBlockObject)

	actionBlock := slack.NewActionBlock(floorActionId, newOptionsGroupSelectBlockElement)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}
