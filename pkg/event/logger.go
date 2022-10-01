package event

import (
	"log"
)

type EventLogger struct {
}

func NewEventLogger() *EventLogger {
	return &EventLogger{}
}

func (ev *EventLogger) Consume(event Event) {
	log.Printf("\t%s", event)
}
