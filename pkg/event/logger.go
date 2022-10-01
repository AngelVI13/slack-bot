package event

import (
	"log"
)

const (
	LogPadding = 30
)

type EventLogger struct {
	padding int
}

func NewEventLogger() *EventLogger {
	return &EventLogger{
		padding: LogPadding,
	}
}

func (ev *EventLogger) Consume(event Event) {
	log.Printf(
		"%*s",
		ev.padding,
		event,
	)
}
