package event

type EventType int

type Event interface {
	Type() EventType
	User() string
	Info() map[string]any
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
	UpdateView:    "UpdateView",
	PostEphemeral: "PostEphemeral",
	Post:          "Post",
}

type ResponseAction interface {
	Info() map[string]any
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
	ViewOpenedEvent
	ViewClosedEvent
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
	ViewOpenedEvent:     "ViewOpened",
	ViewClosedEvent:     "ViewClosed",
	BlockActionEvent:    "BlockAction",
	TimerEvent:          "TimerEvent",
	ResponseEvent:       "ResponseEvent",
	AnyEvent:            "AnyEvent",
}

func EventName(e Event) string {
	return EventNames[e.Type()]
}
