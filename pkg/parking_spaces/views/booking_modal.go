package views

import (
	"fmt"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces/model"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/slack-go/slack"
)

const (
	FloorActionId                    = "floorActionId"
	FloorOptionId                    = "floorOptionId"
	ReserveParkingActionId           = "reserveParking"
	ReleaseParkingActionId           = "releaseParking"
	TempReleaseParkingActionId       = "tempReleaseParking"
	CancelTempReleaseParkingActionId = "cancelTempReleaseParking"
	ShowActionId                     = "showActionId"
	ShowOptionId                     = "showOptionId"
)

type Booking struct {
	Title string
	data  *model.Data
}

func NewBooking(identifier string, managerData *model.Data) *Booking {
	return &Booking{
		Title: identifier + "Booking",
		data:  managerData,
	}
}

func (b *Booking) Generate(userId string, errorTxt string) slack.ModalViewRequest {
	selectedFloor := model.DefaultFloorOption
	selected, ok := b.data.SelectedFloor[userId]
	if ok {
		selectedFloor = selected
	}
	selectedShowTaken := b.data.SelectedShowTaken[userId]

	spacesSectionBlocks := b.generateParkingInfoBlocks(
		userId,
		selectedFloor,
		selectedShowTaken,
		errorTxt,
	)
	return common.GenerateInfoModalRequest(b.Title, spacesSectionBlocks)
}

// generateParkingInfo Generate sections of text that contain space info such as status (taken/free), taken by etc..
func (b *Booking) generateParkingInfo(spaces spaces.SpacesInfo) []slack.Block {
	var sections []slack.Block
	for _, space := range spaces {
		status := space.GetStatusDescription()
		emoji := space.GetStatusEmoji()

		releaseScheduled := ""
		/*
		   TODO: No need to have the cancel temporary release button in this view
		   because user will only schedule and cancel temporary releases in their
		   personal view. If you're an admin, I dont see a use case for you to cancel
		   temporary releases so only perma release the space button should be there.

		   TODO: Implement ToBeReleased.GetActive(space.Key()) to get currently active
		   temp release (just for visual representation of when a space is under release).

		   releaseInfo := b.data.ParkingLot.ToBeReleased.Get(space.Key())
		   if releaseInfo != nil {
		       releaseScheduled = fmt.Sprintf(
		           "\n\t\tScheduled release from %s to %s",
		           releaseInfo.StartDate.Format("2006-01-02"),
		           releaseInfo.EndDate.Format("2006-01-02"),
		       )
		   }
		*/

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

func (b *Booking) generateParkingButtons(
	space *spaces.Space,
	userId string,
) []slack.BlockElement {
	var buttons []slack.BlockElement

	isAdminUser := b.data.UserManager.IsAdminId(userId)
	hasPermanentParkingUser := b.data.UserManager.HasParkingById(userId)

	if space.Reserved && (space.ReservedById == userId || isAdminUser) {
		// space reserved but hasn't yet been schedule for release
		if isAdminUser {
			permanentSpace := b.data.UserManager.HasParkingById(space.ReservedById)
			if permanentSpace {
				// Only allow the temporary parking button if the correct user is using
				// the modal and the space hasn't already been released.
				// For example, an admin can only temporary release a space if either he
				// owns the space & has permanent parking rights or if he is releasing
				// the space on behalf of somebody that has a permanent parking rights
				tempReleaseButton := slack.NewButtonBlockElement(
					TempReleaseParkingActionId,
					string(space.Key()),
					slack.NewTextBlockObject("plain_text", "Temp Release!", true, false),
				)
				tempReleaseButton = tempReleaseButton.WithStyle(slack.StyleDanger)
				buttons = append(buttons, tempReleaseButton)
			}
		}

		if isAdminUser || !hasPermanentParkingUser {
			releaseButton := slack.NewButtonBlockElement(
				ReleaseParkingActionId,
				string(space.Key()),
				slack.NewTextBlockObject("plain_text", "Release!", true, false),
			)
			releaseButton = releaseButton.WithStyle(slack.StyleDanger)
			buttons = append(buttons, releaseButton)
		}
	} else if (!space.Reserved &&
		!b.data.ParkingLot.HasSpace(userId) &&
		!b.data.ParkingLot.HasTempRelease(userId) &&
		!isAdminUser) || (!space.Reserved && isAdminUser) {
		// Only allow user to reserve space if he hasn't already reserved one
		actionButtonText := "Reserve!"
		reserveWithAutoButton := slack.NewButtonBlockElement(
			ReserveParkingActionId,
			string(space.Key()),
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
		slack.NewTextBlockObject(
			"mrkdwn",
			"<https://ibb.co/PFNyGsn|1st floor plan>",
			false,
			false,
		),
		nil,
		nil,
	)
	minusOneParking := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"<https://ibb.co/zHw2T9w|-1st floor plan>",
			false,
			false,
		),
		nil,
		nil,
	)
	minusTwoParking := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			"<https://ibb.co/mt15xrz|-2nd floor plan>",
			false,
			false,
		),
		nil,
		nil,
	)

	now := time.Now()

	if now.Hour() >= model.ResetHour && now.Minute() > model.ResetMin {
		now = now.Add(24 * time.Hour)
	}

	selectionEffectTime := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf(
				"_Reservation is valid for %d-%d-%d_",
				now.Year(),
				now.Month(),
				now.Day(),
			),
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
func (b *Booking) generateParkingInfoBlocks(
	userId, selectedFloor string, selectedShowTaken bool, errorTxt string,
) []slack.Block {
	allBlocks := []slack.Block{}

	descriptionBlocks := generateParkingPlanBlocks()
	allBlocks = append(allBlocks, descriptionBlocks...)

	floorOptionBlocks := b.generateFloorOptions(userId)
	allBlocks = append(allBlocks, floorOptionBlocks...)

	showOptionBlocks := b.generateFreeTakenOptions(userId)
	allBlocks = append(allBlocks, showOptionBlocks...)

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

	spaces := b.data.ParkingLot.GetSpacesByFloor(
		userId,
		selectedFloor,
		selectedShowTaken,
	)

	parkingSpaceSections := b.generateParkingInfo(spaces)

	for idx, space := range spaces {
		sectionBlock := parkingSpaceSections[idx]
		buttons := b.generateParkingButtons(space, userId)

		allBlocks = append(allBlocks, sectionBlock)
		if len(buttons) > 0 {
			actions := slack.NewActionBlock("", buttons...)
			allBlocks = append(allBlocks, actions)
		}
		allBlocks = append(allBlocks, div)
	}

	return allBlocks
}

func (b *Booking) generateFloorOptions(userId string) []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionBlocks []*slack.OptionBlockObject

	for _, floor := range model.Floors {
		optionBlock := slack.NewOptionBlockObject(
			floor,
			slack.NewTextBlockObject("plain_text", floor, false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	selectedFloor := model.DefaultFloorOption
	selected, ok := b.data.SelectedFloor[userId]
	if ok {
		selectedFloor = selected
	}

	// Text shown as title when option box is opened/expanded
	optionLabel := slack.NewTextBlockObject(
		"plain_text",
		"Choose a parking floor",
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
		FloorOptionId,
		optionGroupBlockObject,
	)

	actionBlock := slack.NewActionBlock(FloorActionId, newOptionsGroupSelectBlockElement)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}

func (b *Booking) generateFreeTakenOptions(userId string) []slack.Block {
	var allBlocks []slack.Block

	// Options
	var optionBlocks []*slack.OptionBlockObject

	for _, showOption := range model.ShowOptions {
		optionBlock := slack.NewOptionBlockObject(
			showOption,
			slack.NewTextBlockObject("plain_text", showOption, false, false),
			slack.NewTextBlockObject("plain_text", " ", false, false),
		)
		optionBlocks = append(optionBlocks, optionBlock)
	}

	selectedOption := model.ShowFreeOption
	showTaken := b.data.SelectedShowTaken[userId]
	if showTaken {
		selectedOption = model.ShowTakenOption
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
		ShowOptionId,
		optionGroupBlockObject,
	)

	actionBlock := slack.NewActionBlock(ShowActionId, newOptionsGroupSelectBlockElement)
	allBlocks = append(allBlocks, actionBlock)

	return allBlocks
}
