package event

import (
	"fmt"
	"time"
)

type TimerDone struct {
	label string
}

func (t *TimerDone) Type() EventType {
	return TimerEvent
}

func (t TimerDone) String() string {
	return fmt.Sprintf("Timer[%s] Done.", t.label)
}

func (t *TimerDone) User() string {
	return ""
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
						label: label,
					},
				)
			}
		}
	}()
}
