package slack

import (
	"log"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"

	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
)

type Client struct {
	socket       *socketmode.Client
	eventManager *event.EventManager
}

func NewClient(config *config.Config, eventManager *event.EventManager) *Client {
	client := slack.New(
		config.SlackAuthToken,
		slack.OptionDebug(config.Debug),
		slack.OptionLog(
			log.New(log.Writer(), "client: ", log.Lshortfile|log.LstdFlags),
		),
		slack.OptionAppLevelToken(config.SlackAppToken),
	)

	// Convert simple slack client to socket mode client
	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(config.Debug),
		socketmode.OptionLog(
			log.New(log.Writer(), "socketmode: ", log.Lshortfile|log.LstdFlags),
		),
	)
	return &Client{
		socket:       socketClient,
		eventManager: eventManager,
	}
}

// Listen Listen on incomming slack events
func (c *Client) Listen() {
	for {
		select {
		// inscase context cancel is called exit the goroutine
		case socketEvent := <-c.socket.Events:
			// We need to send an Acknowledge to the slack server
			// TODO: should the ACK be done here before any processing happens?
			c.socket.Ack(*socketEvent.Request)
			log.Println(socketEvent)

			var processedEvent event.Event
			// We have a new Events, let's type switch the event
			// Add more use cases here if you want to listen to other events.
			switch socketEvent.Type {
			case socketmode.EventTypeEventsAPI:
				// Handle mentions
				// NOTE: there is no user restriction for app mentions
				processedEvent = handleApiEvent(socketEvent, c)
			case socketmode.EventTypeSlashCommand:
				// TODO: process this
				// bot.processSlashCommand(event)
			case socketmode.EventTypeInteractive:
				// Handle interaction events i.e. user voted in our poll etc.
				processedEvent = handleInteractionEvent(socketEvent)

			}

			if processedEvent != nil {
				c.eventManager.Publish(processedEvent)
			}
		}
	}
}
