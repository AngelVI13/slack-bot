package slack

import (
	"log"
	"strings"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type Interaction struct {
	BaseEvent
	Values    map[string]map[string]slack.BlockAction
	Actions   []*slack.BlockAction
	TriggerId string
	ViewId    string
	Title     string
}

func (i *Interaction) HasContext(c string) bool {
	return strings.Contains(i.Title, c)
}

type ViewSubmission struct {
	Interaction
}

func (v *ViewSubmission) Type() event.EventType {
	return event.ViewSubmissionEvent
}

func (v ViewSubmission) String() string {
	return event.DefaultEventString(&v) + " " + v.ViewId
}

type BlockAction struct {
	Interaction
}

func (b *BlockAction) Type() event.EventType {
	return event.BlockActionEvent
}

func (b BlockAction) String() string {
	return event.DefaultEventString(&b)
}

type ViewClosed struct {
	Interaction
}

func (v *ViewClosed) Type() event.EventType {
	return event.ViewClosedEvent
}

func (v ViewClosed) String() string {
	return event.DefaultEventString(&v) + " " + v.ViewId
}

type ViewOpened struct {
	BaseEvent
	ViewId     string
	RootViewId string
	Title      string
}

func (v *ViewOpened) Type() event.EventType {
	return event.ViewOpenedEvent
}

func (v ViewOpened) String() string {
	return event.DefaultEventString(&v) + " " + v.ViewId
}

func (v *ViewOpened) HasContext(c string) bool {
	return strings.Contains(v.Title, c)
}

func handleInteractionEvent(socketEvent socketmode.Event) event.Event {
	interactionCb, ok := socketEvent.Data.(slack.InteractionCallback)
	if !ok {
		log.Printf(
			"ERROR: Could not type cast the message to a Interaction callback: %v\n",
			socketEvent,
		)
		return nil
	}

	var event event.Event

	// Collect common data used by all supported interaction types
	interaction := Interaction{
		BaseEvent: BaseEvent{
			UserName: interactionCb.User.Name,
			UserId:   interactionCb.User.ID,
		},
		Values:    interactionCb.View.State.Values,
		Actions:   interactionCb.ActionCallback.BlockActions,
		TriggerId: interactionCb.TriggerID,
		ViewId:    interactionCb.View.ID,
		Title:     interactionCb.View.Title.Text,
	}

	switch interactionCb.Type {
	case slack.InteractionTypeViewSubmission:
		event = &ViewSubmission{interaction}
	case slack.InteractionTypeBlockActions:
		event = &BlockAction{interaction}
	case slack.InteractionTypeViewClosed:
		event = &ViewClosed{interaction}
	default:
		log.Printf("Unsupported interaction event: %v, %v", interactionCb.Type, interaction)
		return nil
	}

	return event
}
