package slack

import (
	"github.com/AngelVI13/slack-bot/pkg/event"
)

const (
	MentionEvent event.EventType = iota
	SlashCmdEvent
	ViewSubmissionEvent
	BlockActionEvent
)

var EventNames = map[event.EventType]string{
	MentionEvent:        "Mention",
	SlashCmdEvent:       "SlashCmd",
	ViewSubmissionEvent: "ViewSubmission",
	BlockActionEvent:    "BlockAction",
}
