package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
	parkingModel "github.com/AngelVI13/slack-bot/pkg/parking_spaces/model"
	"github.com/slack-go/slack"
)

const (
	CancelActionValueSeparator       = "__"
	CancelTempReleaseParkingActionId = "cancelTempReleaseParking"
	SwitchToAllSpacesOverviewId      = "switchToAllSpacesOverview"
)

type Personal struct {
	Title string
	data  *parkingModel.ParkingData
	Type  ModalType
}

func NewPersonal(identifier string, managerData *parkingModel.ParkingData) *Personal {
	return &Personal{
		Title: identifier + "Personal",
		data:  managerData,
		Type:  PersonalModal,
	}
}

func (p *Personal) Generate(userId string, errorTxt string) slack.ModalViewRequest {
	spacesSectionBlocks := p.generatePersonalInfoBlocks(
		userId,
		errorTxt,
	)
	return common.GenerateInfoModalRequest(p.Title, spacesSectionBlocks)
}

func GetOwnerText(space *spaces.Space, ownerId string) string {
	out := ""
	if space.Reserved && space.ReservedById != ownerId {
		out = fmt.Sprintf("\n\tOwner: <@%s>", ownerId)
	}
	return out
}

// TODO: make this a general method for all parking spaces (i.e. can be used by booking modal as well)
func (p *Personal) generateParkingSpaceBlock(
	space *spaces.Space,
	ownerId string,
) *slack.SectionBlock {
	status := space.GetStatusDescription()
	emoji := space.GetStatusEmoji()
	ownerTxt := GetOwnerText(space, ownerId)

	spaceProps := space.GetPropsText()
	text := fmt.Sprintf(
		"%s *%s* \t%s\t %s%s",
		emoji,
		fmt.Sprint(space.Number),
		spaceProps,
		status,
		ownerTxt,
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

func generateTempReleaseButton(
	space *spaces.Space,
	modalType ModalType,
) *slack.ActionBlock {
	tempReleaseBtn := slack.NewButtonBlockElement(
		TempReleaseParkingActionId,
		ActionValues{
			SpaceKey:  space.Key(),
			ModalType: modalType,
		}.Encode(),
		slack.NewTextBlockObject("plain_text", "Add Temp Release!", true, false),
	)
	tempReleaseBtn = tempReleaseBtn.WithStyle(slack.StylePrimary)
	actionBlock := slack.NewActionBlock("", tempReleaseBtn)
	return actionBlock
}

func generateReleaseButton(space *spaces.Space, modalType ModalType) *slack.ActionBlock {
	releaseButton := slack.NewButtonBlockElement(
		ReleaseParkingActionId,
		ActionValues{
			SpaceKey:  space.Key(),
			ModalType: modalType,
		}.Encode(),
		slack.NewTextBlockObject("plain_text", "Release!", true, false),
	)
	releaseButton = releaseButton.WithStyle(slack.StyleDanger)
	actionBlock := slack.NewActionBlock("", releaseButton)
	return actionBlock
}

func generateSwitchOverviewButton(modalType ModalType) *slack.ActionBlock {
	switchOverviewBtn := slack.NewButtonBlockElement(
		SwitchToAllSpacesOverviewId,
		ActionValues{ModalType: modalType}.Encode(),
		slack.NewTextBlockObject("plain_text", "View All Spaces", true, false),
	)
	actionBlock := slack.NewActionBlock("", switchOverviewBtn)
	return actionBlock
}

func generateCancelReleaseButton(
	space *spaces.Space,
	releaseId int,
	modalType ModalType,
) *slack.ActionBlock {
	cancelBtn := slack.NewButtonBlockElement(
		CancelTempReleaseParkingActionId,
		ActionValues{
			SpaceKey:  space.Key(),
			ModalType: modalType,
			ReleaseId: releaseId,
		}.Encode(),
		slack.NewTextBlockObject("plain_text", "Cancel", true, false),
	)
	cancelBtn = cancelBtn.WithStyle(slack.StyleDanger)
	actionBlock := slack.NewActionBlock("", cancelBtn)
	return actionBlock
}

func ParseCancelActionValue(actionValue string) (spaces.SpaceKey, int, error) {
	if strings.Count(actionValue, CancelActionValueSeparator) != 1 {
		return "", -1, fmt.Errorf(
			"unexpected format of cancel action value: epected %q;actual %q",
			"1st floor 121__5",
			actionValue,
		)
	}
	parts := strings.Split(actionValue, CancelActionValueSeparator)
	parkingSpace := spaces.SpaceKey(parts[0])
	releaseId, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", -1, fmt.Errorf(
			"failed to convert release id to int: id %q; err: %v", parts[1], err,
		)
	}

	return parkingSpace, releaseId, nil
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

	space, noSpaceErr := p.data.ParkingLot.GetOwnedSpaceByUserId(userId)

	// If admin views this page add option to switch to overview.
	// Additionally if somehow user who doesn't have space views
	// this page -> add this button so he can go back to overview
	if p.data.UserManager.IsAdminId(userId) || noSpaceErr != nil {
		switchOverviewBtn := generateSwitchOverviewButton(p.Type)
		allBlocks = append(allBlocks, switchOverviewBtn)
	}

	div := slack.NewDividerBlock()
	allBlocks = append(allBlocks, div)

	if noSpaceErr != nil {
		errorBlock := createErrorTextBlock(noSpaceErr.Error())
		allBlocks = append(allBlocks, errorBlock)
		return allBlocks
	}

	spaceBlock := p.generateParkingSpaceBlock(space, userId)
	allBlocks = append(allBlocks, spaceBlock)

	tempReleaseBtn := generateTempReleaseButton(space, p.Type)
	allBlocks = append(allBlocks, tempReleaseBtn)

	if p.data.UserManager.IsAdminId(userId) {
		allBlocks = append(allBlocks, generateReleaseButton(space, p.Type))
	}

	allBlocks = append(allBlocks, div)

	releases := p.data.ParkingLot.ToBeReleased.GetAll(space.Key())
	if len(releases) == 0 {
		// If not release available -> return early
		return allBlocks
	}

	for _, release := range releases {
		if !release.DataPresent() || !release.Submitted {
			continue
		}
		releaseBlock := generateTemporaryReleaseBlock(release)
		allBlocks = append(allBlocks, releaseBlock)

		if !release.Cancelled {
			cancelBtn := generateCancelReleaseButton(space, release.UniqueId, p.Type)
			allBlocks = append(allBlocks, cancelBtn)
		}
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
