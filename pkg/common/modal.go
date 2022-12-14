package common

import (
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
