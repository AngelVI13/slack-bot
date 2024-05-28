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
	return sectionBlock
}

func (p *Personal) generateTemporaryReleasesList(space *spaces.Space) []slack.Block {
	releaseScheduled := ""
	releaseInfo := p.data.ParkingLot.ToBeReleased.Get(space.Key())
	if releaseInfo == nil {
		return nil
	}

	releaseScheduled = fmt.Sprintf(
		"\t\tScheduled release from %s to %s",
		releaseInfo.StartDate.Format("2006-01-02"),
		releaseInfo.EndDate.Format("2006-01-02"),
	)
	sectionText := slack.NewTextBlockObject("mrkdwn", releaseScheduled, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	return []slack.Block{sectionBlock}
}

func generateFakeTemporaryRelease(release string, id int) *slack.SectionBlock {
	clockN := (id % 12) + 1
	releaseTxt := fmt.Sprintf(":clock%d: %s", clockN, release)
	sectionText := slack.NewTextBlockObject("mrkdwn", releaseTxt, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	return sectionBlock
}

func generateReleaseButton(space *spaces.Space) *slack.ActionBlock {
	cancelBtn := slack.NewButtonBlockElement(
		CancelTempReleaseParkingActionId,
		string(space.Key()),
		slack.NewTextBlockObject("plain_text", "Add Temp Release!", true, false),
	)
	cancelBtn = cancelBtn.WithStyle(slack.StylePrimary)
	actionBlock := slack.NewActionBlock("", cancelBtn)
	return actionBlock
}

func generateCancelReleaseButton(space *spaces.Space, id int) *slack.ActionBlock {
	tempReleaseButton := slack.NewButtonBlockElement(
		TempReleaseParkingActionId,
		fmt.Sprintf("%s_%d", string(space.Key()), id),
		slack.NewTextBlockObject("plain_text", "Cancel", true, false),
	)
	tempReleaseButton = tempReleaseButton.WithStyle(slack.StyleDanger)
	actionBlock := slack.NewActionBlock("", tempReleaseButton)
	return actionBlock
}

const personalModelDescription = `This is your personal parking space page.
Here you can add/delete/cancel temporary releases of your parking space.
`

// generatePersonalInfoBlocks Generates space block objects to be used as elements in modal
func (p *Personal) generatePersonalInfoBlocks(userId, errorTxt string) []slack.Block {
	allBlocks := []slack.Block{}

	allBlocks = append(allBlocks, createTextBlock(personalModelDescription))

	if errorTxt != "" {
		errorBlock := createErrorTextBlock(errorTxt)
		allBlocks = append(allBlocks, errorBlock)
	}

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	space := p.data.ParkingLot.GetOwnedSpaceByUserId(userId)

	// TODO: Add more information to parking block and idicate that its actually the
	// user's space. If the space is temporarily booked by someone else then indicate that
	spaceBlock := p.generateParkingSpaceBlock(space)
	allBlocks = append(allBlocks, spaceBlock)

	tempReleaseBtn := generateReleaseButton(space)
	allBlocks = append(allBlocks, tempReleaseBtn)

	allBlocks = append(allBlocks, div)

	// releaseInfoBlocks := p.generateTemporaryReleasesList(space)
	releases := []string{
		"Scheduled release from 2024-05-29 to 2024-05-29",
		"Scheduled release from 2024-06-01 to 2024-06-10",
		"Scheduled release from 2024-06-20 to 2024-06-21",
	}
	for i, release := range releases {
		releaseBlock := generateFakeTemporaryRelease(release, i)
		allBlocks = append(allBlocks, releaseBlock)
		cancelBtn := generateCancelReleaseButton(space, i)
		allBlocks = append(allBlocks, cancelBtn)
	}

	return allBlocks
}

func createErrorTextBlock(errorTxt string) *slack.SectionBlock {
	txt := fmt.Sprintf(`:warning: %s`, errorTxt)
	return createTextBlock(txt)
}

func createTextBlock(text string) *slack.SectionBlock {
	return slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", text, false, false), nil, nil)
}
