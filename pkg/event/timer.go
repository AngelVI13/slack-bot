package event

import (
	"time"
)

type TimerDone struct {
	Label string
	Time  time.Time
}

func (t *TimerDone) Type() EventType {
	return TimerEvent
}

func (t *TimerDone) Info() map[string]any {
	return map[string]any{
		"label": t.Label,
	}
}

func (t *TimerDone) User() string {
	return ""
}

func (t *TimerDone) HasContext(c string) bool {
	return true
}

type Timer struct {
	eventManager *EventManager
}

func NewTimer(eventManager *EventManager) *Timer {
	return &Timer{
		eventManager: eventManager,
	}
}

// AddDaily adds a daily recurring timer/alarm in the form of event which is published
// to the event manager.
func (t *Timer) AddDaily(hour, min int, label string) {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			cTime := <-ticker.C
			if cTime.Hour() == hour && cTime.Minute() == min {
				t.eventManager.Publish(
					&TimerDone{
						Label: label,
						Time:  cTime,
					},
				)
			}
		}
	}()
}
