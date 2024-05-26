package views

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces/model"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/slack-go/slack"
)

type Personal struct {
	Title string
	data  *model.Data
}

func NewPersonal(identifier string, managerData *model.Data) *Personal {
	return &Personal{
		Title: identifier + "Personal",
		data:  managerData,
	}
}

func (p *Personal) Generate(userId string, errorTxt string) slack.ModalViewRequest {
	spacesSectionBlocks := p.generatePersonalInfoBlocks(
		userId,
		errorTxt,
	)
	return common.GenerateInfoModalRequest(p.Title, spacesSectionBlocks)
}

// generateParkingInfo Generate sections of text that contain space info such as status (taken/free), taken by etc..
func (p *Personal) generateParkingInfo(spaces spaces.SpacesInfo) []slack.Block {
	var sections []slack.Block
	for _, space := range spaces {
		sectionBlock := p.generateParkingSpaceBlock(space)
		sections = append(sections, *sectionBlock)
	}
	return sections
}

// TODO: make this a general method for all parking spaces (i.e. can be used by booking modal as well)
func (p *Personal) generateParkingSpaceBlock(space *spaces.Space) *slack.SectionBlock {
	status := space.GetStatusDescription()
	emoji := space.GetStatusEmoji()

	releaseScheduled := ""
	releaseInfo := p.data.ParkingLot.ToBeReleased.Get(space.Key())
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
	return sectionBlock
}

func (p *Personal) generateParkingButtons(
	space *spaces.Space,
	userId string,
) []slack.BlockElement {
	var buttons []slack.BlockElement

	isAdminUser := p.data.UserManager.IsAdminId(userId)
	hasPermanentParkingUser := p.data.UserManager.HasParkingById(userId)

	releaseInfo := p.data.ParkingLot.ToBeReleased.Get(space.Key())
	if releaseInfo != nil && (releaseInfo.OwnerId == userId || isAdminUser) &&
		!releaseInfo.Cancelled {
		cancelTempReleaseButton := slack.NewButtonBlockElement(
			CancelTempReleaseParkingActionId,
			string(space.Key()),
			slack.NewTextBlockObject(
				"plain_text",
				"Cancel Scheduled Release!",
				true,
				false,
			),
		)
		cancelTempReleaseButton = cancelTempReleaseButton.WithStyle(slack.StyleDanger)
		buttons = append(buttons, cancelTempReleaseButton)
	}

	if space.Reserved && (space.ReservedById == userId || isAdminUser) {
		// space reserved but hasn't yet been schedule for release
		if (isAdminUser || hasPermanentParkingUser) && releaseInfo == nil {
			permanentSpace := p.data.UserManager.HasParkingById(space.ReservedById)
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
		!p.data.ParkingLot.HasSpace(userId) &&
		!p.data.ParkingLot.HasTempRelease(userId) &&
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

// generatePersonalInfoBlocks Generates space block objects to be used as elements in modal
func (p *Personal) generatePersonalInfoBlocks(userId, errorTxt string) []slack.Block {
	allBlocks := []slack.Block{}

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

	space := p.data.ParkingLot.GetSpaceByUserId(userId)
	sectionBlock := p.generateParkingSpaceBlock(space)
	buttons := p.generateParkingButtons(space, userId)

	allBlocks = append(allBlocks, sectionBlock)
	if len(buttons) > 0 {
		actions := slack.NewActionBlock("", buttons...)
		allBlocks = append(allBlocks, actions)
	}
	allBlocks = append(allBlocks, div)

	return allBlocks
}
