package event

type EventType int

type Event interface {
	Type() EventType
	User() string
}

const (
	BasicEvent EventType = iota
	MentionEvent
	SlashCmdEvent
	ViewSubmissionEvent
	BlockActionEvent
	AnyEvent
)

var EventNames = map[EventType]string{
	BasicEvent:          "BasicEvent",
	MentionEvent:        "Mention",
	SlashCmdEvent:       "SlashCmd",
	ViewSubmissionEvent: "ViewSubmission",
	BlockActionEvent:    "BlockAction",
	AnyEvent:            "AnyEvent",
}
