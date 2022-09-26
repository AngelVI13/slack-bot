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

func (c *Client) Listen() {

}
