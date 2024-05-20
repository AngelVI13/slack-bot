package event

import "log/slog"

type EventLogger struct{}

func NewEventLogger() *EventLogger {
	return &EventLogger{}
}

func (ev *EventLogger) Consume(event Event) {
	args := []any{"user", event.User()}
	for k, v := range event.Info() {
		args = append(args, k, v)
	}

	slog.Info(EventName(event), args...)
}
