package slack

type EventType int

const (
	MentionEvent EventType = iota
	SlashCmdEvent
	BlockActionEvent
	ViewSubmissionEvent
)

var EventNames = map[EventType]string{
	MentionEvent:        "Mention",
	SlashCmdEvent:       "SlashCmd",
	BlockActionEvent:    "BlockAction",
	ViewSubmissionEvent: "ViewSubmission",
}

type Event interface {
	Type() EventType
	Data() any
}
