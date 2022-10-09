package parking

import (
	"fmt"
	"log"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

const (
	reserveParkingActionId           = "reserveParking"
	releaseParkingActionId           = "releaseParking"
	tempReleaseParkingActionId       = "tempReleaseParking"
	cancelTempReleaseParkingActionId = "cancelTempReleaseParking"
)

var parkingBookingTitle = Identifier + "Booking"

func (m *Manager) generateBookingModalRequest(command event.Event, userId, errorTxt string) slack.ModalViewRequest {
	// TODO: highlight your parking space?
	spacesSectionBlocks := m.generateParkingInfoBlocks(userId, errorTxt)
	return common.GenerateInfoModalRequest(parkingBookingTitle, spacesSectionBlocks)
}

// generateParkingInfo Generate sections of text that contain space info such as status (taken/free), taken by etc..
func (m *Manager) generateParkingInfo(spaces SpacesInfo) []slack.Block {
	var sections []slack.Block
	for _, space := range spaces {
		status := space.GetStatusDescription()
		emoji := space.GetStatusEmoji()

		releaseScheduled := ""
		releaseInfo := m.parkingLot.ToBeReleased.Get(space.Number)
		if releaseInfo != nil {
			releaseScheduled = fmt.Sprintf(
				"\tScheduled release from %s to %s",
				releaseInfo.StartDate.Format("2006-01-02"),
				releaseInfo.EndDate.Format("2006-01-02"),
			)
		}

		spaceProps := space.GetPropsText()
		text := fmt.Sprintf(
			"%s *%s* \t%s\t %s\n\t\t%s",
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
	alreadyReservedSpace,
	alreadyReleasedSpace bool,
) []slack.BlockElement {
	var buttons []slack.BlockElement

	isAdminUser := m.userManager.IsAdminId(userId)
	hasPermanentParkingUser := m.userManager.HasParkingById(userId)

	if space.Reserved && (space.ReservedById == userId || isAdminUser) {
		releaseInfo := m.parkingLot.ToBeReleased.Get(space.Number)
		// Only show cancel button before the temprary release is in effect
		if releaseInfo != nil && releaseInfo.StartDate.After(time.Now()) {
			cancelTempReleaseButton := slack.NewButtonBlockElement(
				cancelTempReleaseParkingActionId,
				fmt.Sprint(space.Number),
				slack.NewTextBlockObject("plain_text", "Cancel Scheduled Release!", true, false),
			)
			cancelTempReleaseButton = cancelTempReleaseButton.WithStyle(slack.StyleDanger)
			buttons = append(buttons, cancelTempReleaseButton)
		} else { // space reserved but hasn't yet been schedule for release
			if (isAdminUser || hasPermanentParkingUser) && releaseInfo == nil {
				// Only allow the temporary parking button if the correct user is using
				// the modal and the space hasn't already been released
				tempReleaseButton := slack.NewButtonBlockElement(
					tempReleaseParkingActionId,
					fmt.Sprint(space.Number),
					slack.NewTextBlockObject("plain_text", "Temp Release!", true, false),
				)
				tempReleaseButton = tempReleaseButton.WithStyle(slack.StyleDanger)
				buttons = append(buttons, tempReleaseButton)
			}

			if isAdminUser || !hasPermanentParkingUser {
				releaseButton := slack.NewButtonBlockElement(
					releaseParkingActionId,
					fmt.Sprint(space.Number),
					slack.NewTextBlockObject("plain_text", "Release!", true, false),
				)
				releaseButton = releaseButton.WithStyle(slack.StyleDanger)
				buttons = append(buttons, releaseButton)
			}
		}
	} else if (!space.Reserved &&
		!alreadyReservedSpace &&
		!alreadyReleasedSpace &&
		!isAdminUser) || (!space.Reserved && isAdminUser) {
		// Only allow user to reserve space if he hasn't already reserved one
		actionButtonText := "Reserve!"
		reserveWithAutoButton := slack.NewButtonBlockElement(
			reserveParkingActionId,
			fmt.Sprint(space.Number),
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("%s :eject:", actionButtonText), true, false),
		)
		reserveWithAutoButton = reserveWithAutoButton.WithStyle(slack.StylePrimary)
		buttons = append(buttons, reserveWithAutoButton)
	}
	return buttons
}

func generateParkingPlanBlocks() []slack.Block {
	// TODO: should 1 user only be allowed to book 1 parking space ?
	description := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "In the pictures below you can find the parking plan so you can locate your parking space.", false, false), nil, nil)
	imgLink := "https://w7.pngwing.com/pngs/610/377/png-transparent-parking-parking-lot-car-park.png"
	parkingPlanImage := slack.NewImageBlockElement(imgLink, "parking plan")

	plan1 := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Parking Plan (Floor 1)", false, false),
		nil,
		slack.NewAccessory(parkingPlanImage),
	)
	plan2 := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Parking Plan (Floor -1)", false, false),
		nil,
		slack.NewAccessory(parkingPlanImage),
	)

	plan3 := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", "Parking Plan (Floor -2)", false, false),
		nil,
		// TODO: Use image element instead
		slack.NewAccessory(parkingPlanImage),
	)
	// TODO: figure out the difference between image block element and image element
	// img1 := slack.NewImageBlock(imgLink, "parking plan", "img1", slack.NewTextBlockObject("mrkdwn", "Parking Plan (Floor 1)", false, false))
	// img1 := slack.NewImageBlock(imgLink, "parking plan", "", nil)

	// return []slack.Block{description, plan1, plan2, plan3, img1}
	return []slack.Block{description, plan1, plan2, plan3}
}

// generateParkingInfoBlocks Generates space block objects to be used as elements in modal
func (m *Manager) generateParkingInfoBlocks(userId, errorTxt string) []slack.Block {
	allBlocks := []slack.Block{}

	descriptionBlocks := generateParkingPlanBlocks()
	allBlocks = append(allBlocks, descriptionBlocks...)

	if errorTxt != "" {
		txt := fmt.Sprintf(`:warning: *%s*`, errorTxt)
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

	userName := m.userManager.GetNameFromId(userId)
	spaces := m.parkingLot.GetSpacesInfo(userName)
	parkingSpaceSections := m.generateParkingInfo(spaces)

	userAlreadyReservedSpace := false
	for _, space := range spaces {
		if space.Reserved && space.ReservedById == userId {
			userAlreadyReservedSpace = true
			break
		}
	}
	userAlreadyReleasedSpace := false
	for _, releaseInfo := range m.parkingLot.ToBeReleased {
		if releaseInfo.Submitted && releaseInfo.OwnerId == userId {
			userAlreadyReleasedSpace = true
			break
		}
	}

	for idx, space := range spaces {
		sectionBlock := parkingSpaceSections[idx]
		buttons := m.generateParkingButtons(space, userId, userAlreadyReservedSpace, userAlreadyReleasedSpace)

		allBlocks = append(allBlocks, sectionBlock)
		if len(buttons) > 0 {
			actions := slack.NewActionBlock("", buttons...)
			allBlocks = append(allBlocks, actions)
		}
		allBlocks = append(allBlocks, div)
	}

	return allBlocks
}
