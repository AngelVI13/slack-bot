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

func generateTemporaryReleaseBlock(
	release *spaces.ReleaseInfo,
) *slack.SectionBlock {
	releaseId := release.UniqueId
	clockN := (releaseId % 12) + 1
	releaseScheduled := fmt.Sprintf(
		":clock%d: Scheduled release from %s to %s",
		clockN,
		release.StartDate.Format("2006-01-02"),
		release.EndDate.Format("2006-01-02"),
	)
	sectionText := slack.NewTextBlockObject("mrkdwn", releaseScheduled, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	return sectionBlock
}

func generateFakeTemporaryRelease(release string, id int) *slack.SectionBlock {
	clockN := (id % 12) + 1
	releaseTxt := fmt.Sprintf(":clock%d: %s", clockN, release)
	sectionText := slack.NewTextBlockObject("mrkdwn", releaseTxt, false, false)
	sectionBlock := slack.NewSectionBlock(sectionText, nil, nil)
	return sectionBlock
}

func generateReleaseButton(space *spaces.Space) *slack.ActionBlock {
	tempReleaseBtn := slack.NewButtonBlockElement(
		TempReleaseParkingActionId,
		string(space.Key()),
		slack.NewTextBlockObject("plain_text", "Add Temp Release!", true, false),
	)
	tempReleaseBtn = tempReleaseBtn.WithStyle(slack.StylePrimary)
	actionBlock := slack.NewActionBlock("", tempReleaseBtn)
	return actionBlock
}

func generateCancelReleaseButton(space *spaces.Space, releaseId int) *slack.ActionBlock {
	cancelBtn := slack.NewButtonBlockElement(
		CancelTempReleaseParkingActionId,
		fmt.Sprintf("%s__%d", string(space.Key()), releaseId),
		slack.NewTextBlockObject("plain_text", "Cancel", true, false),
	)
	cancelBtn = cancelBtn.WithStyle(slack.StyleDanger)
	actionBlock := slack.NewActionBlock("", cancelBtn)
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

	releases := p.data.ParkingLot.ToBeReleased.GetAll(space.Key())
	if len(releases) == 0 {
		// If not release available -> return early
		return allBlocks
	}

	for _, release := range releases {
		releaseBlock := generateTemporaryReleaseBlock(release)
		cancelBtn := generateCancelReleaseButton(space, release.UniqueId)
		allBlocks = append(allBlocks, releaseBlock, cancelBtn)
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
