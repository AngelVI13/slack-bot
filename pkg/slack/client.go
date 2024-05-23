package slack

import (
	"log"
	"log/slog"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"

	"github.com/AngelVI13/slack-bot/pkg/common"
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
	c := &Client{
		socket:       socketClient,
		eventManager: eventManager,
	}
	// This actually performs the connection to slack (its blocking)
	go c.socket.Run()

	return c
}

// Listen Listen on incomming slack events
func (c *Client) Listen() {
	for socketEvent := range c.socket.Events {
		var processedEvent event.Event
		// We have a new Events, let's type switch the event
		// Add more use cases here if you want to listen to other events.
		switch socketEvent.Type {
		case socketmode.EventTypeEventsAPI:
			// Handle mentions
			c.socket.Ack(*socketEvent.Request)
			processedEvent = handleApiEvent(socketEvent, c)
		case socketmode.EventTypeSlashCommand:
			// Handle slash commands
			c.socket.Ack(*socketEvent.Request)
			processedEvent = handleSlashCommand(socketEvent)
		case socketmode.EventTypeInteractive:
			// Handle interaction events i.e. user voted in our poll etc.
			c.socket.Ack(*socketEvent.Request)
			processedEvent = handleInteractionEvent(socketEvent, c)
		default:
			// log.Println("Unknown event", socketEvent)
		}

		if processedEvent != nil {
			c.eventManager.Publish(processedEvent)
		}
	}
}

func (c *Client) Consume(e event.Event) {
	data, ok := e.(event.Response)
	if !ok {
		slog.Error("Slack client expected Response but got sth else", "event", e)
		return
	}

	for _, action := range data.Actions() {
		switch action.Action() {
		case event.OpenView, event.PushView, event.UpdateView:
			view := action.(*common.ViewAction)
			viewAction := view.Action()

			var (
				err     error
				newView *slack.ViewResponse
			)

			switch viewAction {
			case event.OpenView:
				newView, err = c.socket.OpenView(view.TriggerId, view.ModalRequest)
			case event.PushView:
				newView, err = c.socket.PushView(view.TriggerId, view.ModalRequest)
			case event.UpdateView:
				_, err = c.socket.UpdateView(view.ModalRequest, "", "", view.ViewId)
			default:
				slog.Error("Unsupported view action", "viewAction", viewAction)
			}

			if newView != nil {
				c.eventManager.Publish(&ViewOpened{
					BaseEvent: BaseEvent{
						UserName: e.User(),
						UserId:   "",
					},
					Title:      view.ModalRequest.Title.Text,
					ViewId:     newView.ID,
					RootViewId: newView.RootViewID,
				})
			}

			if err != nil {
				slog.Error("Error opening view", "err", err)
			}
		case event.PostEphemeral:
			post := action.(*common.PostEphemeralAction)
			c.socket.PostEphemeral(post.ChannelId, post.UserId, post.MsgOption)
		case event.Post:
			post := action.(*common.PostAction)
			c.socket.PostMessage(post.ChannelId, post.MsgOption)
		default:
			slog.Error("Unsupported action", "action", action.Action())
		}
	}
}
