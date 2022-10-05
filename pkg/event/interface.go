package event

import "fmt"

type EventType int

type Event interface {
	Type() EventType
	User() string
	String() string
	HasContext(context string) bool
}

type ResponseActionType int

const (
	OpenView ResponseActionType = iota
	PushView
	UpdateView
	PostEphemeral
	Post
)

var ResponseActionNames = map[ResponseActionType]string{
	OpenView:      "OpenView",
	PushView:      "PushView",
	PostEphemeral: "PostEphemeral",
	Post:          "Post",
}

type ResponseAction interface {
	String() string
	Action() ResponseActionType
}

type Response interface {
	Event
	Actions() []ResponseAction
}

const (
	BasicEvent EventType = iota
	MentionEvent
	SlashCmdEvent
	ViewSubmissionEvent
	BlockActionEvent
	TimerEvent
	ResponseEvent
	AnyEvent
)

var EventNames = map[EventType]string{
	BasicEvent:          "BasicEvent",
	MentionEvent:        "Mention",
	SlashCmdEvent:       "SlashCmd",
	ViewSubmissionEvent: "ViewSubmission",
	BlockActionEvent:    "BlockAction",
	TimerEvent:          "TimerEvent",
	ResponseEvent:       "ResponseEvent",
	AnyEvent:            "AnyEvent",
}

func DefaultEventString(e Event) string {
	return fmt.Sprintf("%s(%s)", EventNames[e.Type()], e.User())
}
