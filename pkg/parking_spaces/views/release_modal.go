package views

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces/model"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/slack-go/slack"
)

const (
	ReleaseStartDateActionId = "releaseStartDate"
	ReleaseEndDateActionId   = "releaseEndDate"
	ReleaseBlockId           = "releaseBlockId"
)

type Release struct {
	Title       string
	managerData *model.Data
}

func NewRelease(identifier string, managerData *model.Data) *Release {
	return &Release{
		Title:       identifier + "Release a spot",
		managerData: managerData,
	}
}

// NOTE: this is triggered by a block action (i.e. when user presses the
// release button for a parking space)
func (r *Release) Generate(
	space *spaces.Space,
	errorTxt string,
) slack.ModalViewRequest {
	allBlocks := generateReleaseModalBlocks(space, errorTxt)
	// NOTE: Since this is a modal thats pushed ontop of sth else,
	// apparently the same title has to be used as the underneath modal.
	return common.GenerateModalRequest(r.Title, allBlocks)
}

func generateReleaseModalBlocks(
	space *spaces.Space,
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

	startDate := slack.NewDatePickerBlockElement(ReleaseStartDateActionId)
	startDate.Placeholder = slack.NewTextBlockObject(
		"plain_text",
		"Select START date",
		false,
		false,
	)

	endDate := slack.NewDatePickerBlockElement(ReleaseEndDateActionId)
	endDate.Placeholder = slack.NewTextBlockObject(
		"plain_text",
		"Select END date",
		false,
		false,
	)

	calendarsSection := slack.NewActionBlock(
		ReleaseBlockId,
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
		allBlocks = append(allBlocks, errorSection)
	}

	return allBlocks
}
