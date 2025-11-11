package common

import (
	"log/slog"

	"github.com/slack-go/slack"
)

func GenerateModalRequest(title string, blocks []slack.Block) slack.ModalViewRequest {
	// Create a ModalViewRequest with a header and two inputs
	titleText := slack.NewTextBlockObject("plain_text", title, false, false)
	closeText := slack.NewTextBlockObject("plain_text", "Close", false, false)
	submitText := slack.NewTextBlockObject("plain_text", "Submit", false, false)

	blockSet := slack.Blocks{
		BlockSet: blocks,
	}

	var modalRequest slack.ModalViewRequest
	modalRequest.Type = slack.ViewType("modal")
	modalRequest.Title = titleText
	modalRequest.Close = closeText
	modalRequest.Submit = submitText
	modalRequest.Blocks = blockSet
	modalRequest.NotifyOnClose = true
	return modalRequest
}

func GenerateInfoModalRequest(title string, blocks []slack.Block) slack.ModalViewRequest {
	// Create a ModalViewRequest with a header and two inputs
	titleText := slack.NewTextBlockObject("plain_text", title, false, false)
	closeText := slack.NewTextBlockObject("plain_text", "Close", false, false)

	blockSet := slack.Blocks{
		BlockSet: blocks,
	}

	var modalRequest slack.ModalViewRequest
	modalRequest.Type = slack.ViewType("modal")
	modalRequest.Title = titleText
	modalRequest.Close = closeText
	modalRequest.Blocks = blockSet
	modalRequest.NotifyOnClose = true
	return modalRequest
}

func MakeTitle(title string, testingActive bool) string {
	if testingActive {
		title = "[Test] " + title
	}

	// NOTE: slack accepts max title length of 25 chars
	if len(title) >= 25 {
		slog.Warn(
			"Truncating modal title to 24 chars",
			"title",
			title,
			"truncated",
			title[0:24],
		)
		title = title[0:24]
	}
	return title
}
