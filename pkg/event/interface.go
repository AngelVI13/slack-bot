package event

import "fmt"

type EventType int

type Event interface {
	Type() EventType
	User() string
	String() string
}

const (
	BasicEvent EventType = iota
	MentionEvent
	SlashCmdEvent
	ViewSubmissionEvent
	BlockActionEvent
	TimerEvent
	AnyEvent
)

var EventNames = map[EventType]string{
	BasicEvent:          "BasicEvent",
	MentionEvent:        "Mention",
	SlashCmdEvent:       "SlashCmd",
	ViewSubmissionEvent: "ViewSubmission",
	BlockActionEvent:    "BlockAction",
	TimerEvent:          "TimerEvent",
	AnyEvent:            "AnyEvent",
}

func DefaultEventString(e Event) string {
	return fmt.Sprintf("%s(%s)", EventNames[e.Type()], e.User())
}
