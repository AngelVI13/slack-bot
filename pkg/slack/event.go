package slack

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/event"
)

type BaseEvent struct {
	UserName string
	UserId   string
}

func (e *BaseEvent) Type() event.EventType {
	return BasicEvent
}

func (e BaseEvent) String() string {
	return fmt.Sprintf("%s(%s)", EventNames[e.Type()], e.UserName)
}

const (
	BasicEvent event.EventType = iota
	MentionEvent
	SlashCmdEvent
	ViewSubmissionEvent
	BlockActionEvent
)

var EventNames = map[event.EventType]string{
	BasicEvent:          "BasicEvent",
	MentionEvent:        "Mention",
	SlashCmdEvent:       "SlashCmd",
	ViewSubmissionEvent: "ViewSubmission",
	BlockActionEvent:    "BlockAction",
}
