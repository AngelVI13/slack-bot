package slack

import (
	"log"

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
}

type ViewSubmission struct {
	Interaction
}

func (v *ViewSubmission) Type() event.EventType {
	return event.ViewSubmissionEvent
}

type BlockAction struct {
	Interaction
}

func (b *BlockAction) Type() event.EventType {
	return event.BlockActionEvent
}

func handleInteractionEvent(socketEvent socketmode.Event) event.Event {
	interactionCb, ok := socketEvent.Data.(slack.InteractionCallback)
	if !ok {
		log.Fatalf(
			"Could not type cast the message to a Interaction callback: %v\n",
			socketEvent,
		)
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
	}

	switch interactionCb.Type {
	case slack.InteractionTypeViewSubmission:
		event = &ViewSubmission{interaction}
	case slack.InteractionTypeBlockActions:
		event = &BlockAction{interaction}
	default:
		log.Printf("Unsupported interaction event: %v, %v", interactionCb.Type, interaction)
		return nil
	}

	return event
}
