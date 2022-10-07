package parking

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/common"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/slack-go/slack"
)

const (
	tempReleaseParkingActionId = "tempReleaseParking"
	releaseStartDateActionId   = "releaseStartDate"
	releaseEndDateActionId     = "releaseEndDate"
	releaseBlockId             = "releaseBlockId"
)

var parkingReleaseTitle = Identifier + "Temporary release a parking spot"

// NOTE: this is triggered by a block action (i.e. when user presses the
// release button for a parking space)
func generateReleaseModalRequest(
	command *slackApi.BlockAction,
	space *ParkingSpace,
	errorTxt string,
) slack.ModalViewRequest {
	allBlocks := generateReleaseModalBlocks(command, space, errorTxt)
	// NOTE: Since this is a modal thats pushed ontop of sth else,
	// apparently the same title has to be used as the underneath modal.
	return common.GenerateModalRequest(parkingBookingTitle, allBlocks)
}

func generateReleaseModalBlocks(
	command *slackApi.BlockAction,
	space *ParkingSpace,
	errorTxt string,
) []slack.Block {
	description := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf(
				"Temporarily release space: %d (%d floor)",
				space.Number,
				space.Floor,
			), false, false),
		nil,
		nil,
	)

	startDate := slack.NewDatePickerBlockElement(releaseStartDateActionId)
	startDate.Placeholder = slack.NewTextBlockObject("plain_text", "Select START date", false, false)

	endDate := slack.NewDatePickerBlockElement(releaseEndDateActionId)
	endDate.Placeholder = slack.NewTextBlockObject("plain_text", "Select END date", false, false)

	calendarsSection := slack.NewActionBlock(
		releaseBlockId,
		startDate,
		endDate,
	)

	allBlocks := []slack.Block{
		description,
		calendarsSection,
	}

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

	return allBlocks
}
