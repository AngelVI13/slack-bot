package slack

import (
	"log"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"

	"github.com/AngelVI13/slack-bot/pkg/config"
)

type Client struct {
	socket *socketmode.Client
}

func NewClient(config *config.Config) *Client {
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
		socket: socketClient,
	}
}

// Listen Listen on incomming slack events
func (c *Client) Listen() {
	for {
		select {
		// inscase context cancel is called exit the goroutine
		case event := <-c.socket.Events:
			// We have a new Events, let's type switch the event
			// Add more use cases here if you want to listen to other events.
			switch event.Type {
			case socketmode.EventTypeEventsAPI:
				// Handle mentions
				// NOTE: there is no user restriction for app mentions
				// TODO: process this
				// bot.processEventApi(event)
			case socketmode.EventTypeSlashCommand:
				// TODO: process this
				// bot.processSlashCommand(event)
			case socketmode.EventTypeInteractive:
				// Handle interaction events i.e. user voted in our poll etc.
				// TODO: process this
				// bot.processEventInteractive(event)
			}
		}
	}
}
