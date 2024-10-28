package slack

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
)

type Client struct {
	socket         *socketmode.Client
	eventManager   *event.EventManager
	reportPersonId string
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
		socket:         socketClient,
		eventManager:   eventManager,
		reportPersonId: config.ReportPersonId,
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

func (c *Client) ReportError(msg string) {
	slog.Error("REPORT", "err", msg)
	if c.reportPersonId == "" {
		return
	}
	post := common.NewPostEphemeralAction(c.reportPersonId, c.reportPersonId, msg, false)
	c.socket.PostEphemeral(post.ChannelId, post.UserId, post.MsgOption)
}

func (c *Client) Consume(e event.Event) {
	data, ok := e.(event.Response)
	if !ok {
		msg := "Slack client expected Response but got sth else"
		slog.Error(msg, "event", e)
		c.ReportError(msg)
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

			/*
				b, marshallErr := json.Marshal(view.ModalRequest)
				if marshallErr == nil {
					slog.Info("Serialized modal request", "req", string(b))
				}
			*/

			switch viewAction {
			case event.OpenView:
				newView, err = c.socket.OpenView(view.TriggerId, view.ModalRequest)
			case event.PushView:
				newView, err = c.socket.PushView(view.TriggerId, view.ModalRequest)
			case event.UpdateView:
				newView, err = c.socket.UpdateView(view.ModalRequest, "", "", view.ViewId)
			default:
				slog.Error("Unsupported view action", "viewAction", viewAction)
			}

			if newView != nil && err == nil {
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
				actionName := event.ResponseActionNames[action.Action()]
				details := newView.ResponseMetadata.Messages
				slog.Error("", "action", actionName, "err", err, "details", details)

				jsonRequest, marshallErr := json.MarshalIndent(
					&view.ModalRequest,
					"",
					"\t",
				)
				var jsonRequestStr string
				if marshallErr == nil {
					jsonRequestStr = string(jsonRequest)
				} else {
					// in the case of error while marshalling request json
					//-> show error in that field
					jsonRequestStr = marshallErr.Error()
				}
				msgTxt := fmt.Sprintf(
					"Slack open view error.\nUser: %s\nAction: %s\nError:%s\nDetails:%s\nJsonRequest: %s\n",
					e.User(),
					actionName,
					err,
					strings.Join(details, "\n"),
					jsonRequestStr,
				)
				c.ReportError(msgTxt)
			}
		case event.PostEphemeral:
			post := action.(*common.PostEphemeralAction)
			timestamp, err := c.socket.PostEphemeral(
				post.ChannelId,
				post.UserId,
				post.MsgOption,
			)
			msgTxt := fmt.Sprintf(
				"Slack post ephemeral error.\nUser: %s\nActions: %s\nTimestamp: %s\nError:%s\nTxt: %s\nChannelId: %s\n",
				e.User(),
				event.ResponseActionNames[post.Action()],
				timestamp,
				err,
				post.Txt,
				post.ChannelId,
			)
			c.ReportError(msgTxt)
		case event.Post:
			post := action.(*common.PostAction)
			respChannel, respTimestamp, err := c.socket.PostMessage(
				post.ChannelId,
				post.MsgOption,
			)
			msgTxt := fmt.Sprintf(
				"Slack post error.\nUser: %s\nAction: %s\nRespChannel: %s\nTimestamp: %s\nError:%s\nTxt: %s\nChannelId: %s\n",
				e.User(),
				event.ResponseActionNames[post.Action()],
				respChannel,
				respTimestamp,
				err,
				post.Txt,
				post.ChannelId,
			)
			c.ReportError(msgTxt)
		default:
			slog.Error("Unsupported action", "action", action.Action())
			c.ReportError(
				fmt.Sprintf(
					"Unsupported action:\nUser: %s\nAction: %s\n",
					e.User(),
					event.ResponseActionNames[action.Action()],
				),
			)
		}
	}
}
