package slack

import (
	"fmt"
	"log"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Mention struct {
	BaseEvent
	Text      string
	Channel   string
	Timestamp string
}

func (m *Mention) Type() event.EventType {
	return event.MentionEvent
}

func (m Mention) String() string {
	return fmt.Sprintf("%s - %s", event.DefaultEventString(&m), m.Text)
}

func (m *Mention) HasContext(c string) bool {
	return true
}

// handleApiEvent will take an event and handle it properly based on the type of event
func handleApiEvent(socketEvent socketmode.Event, client *Client) event.Event {
	// The Event sent on the channel is not the same as the EventAPI events so we need to type cast it
	apiEvent, ok := socketEvent.Data.(slackevents.EventsAPIEvent)
	if !ok {
		log.Printf("Could not type cast the event to the EventsAPIEvent: %v\n", socketEvent)
		return nil
	}

	var processedEvent event.Event

	switch apiEvent.Type {
	// First we check if this is an CallbackEvent
	case slackevents.CallbackEvent:

		innerEvent := apiEvent.InnerEvent
		// Yet Another Type switch on the actual Data to see if its an AppMentionEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			// The application has been mentioned since this Event is a Mention event
			user, err := client.socket.GetUserInfo(ev.User)
			if err != nil {
				return nil
			}
			processedEvent = &Mention{
				BaseEvent: BaseEvent{
					UserName: user.Name,
					UserId:   user.ID,
				},
				Text:      ev.Text,
				Channel:   ev.Channel,
				Timestamp: ev.TimeStamp,
			}
			return processedEvent
		default:
			log.Printf("unsupported callback event type: %T -> %v", innerEvent.Data, innerEvent.Data)
			return nil

		}
	default:
		log.Printf("unsupported api event type: %T -> %v", apiEvent.Type, apiEvent)
		return nil
	}
}
