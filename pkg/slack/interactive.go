package slack

import (
	"fmt"
	"log"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

type Interaction struct {
	UserName  string
	UserId    string
	Values    map[string]map[string]slack.BlockAction
	Actions   []*slack.BlockAction
	TriggerId string
	ViewId    string
}

func (i Interaction) String() string {
	return fmt.Sprintf("%s(%s)", EventNames[BlockActionEvent], i.UserName)
}

type ViewSubmission struct {
	Interaction
}

func (v *ViewSubmission) Type() event.EventType {
	return ViewSubmissionEvent
}

func (v ViewSubmission) String() string {
	return fmt.Sprintf("---> %s(%s)", EventNames[ViewSubmissionEvent], v.UserName)
}

type BlockAction struct {
	Interaction
}

func (b *BlockAction) Type() event.EventType {
	return BlockActionEvent
}

func handleInteractionEvent(interactionCb slack.InteractionCallback) event.Event {
	var event event.Event

	// Collect common data used by all supported interaction types
	interaction := Interaction{
		UserName:  interactionCb.User.Name,
		UserId:    interactionCb.User.ID,
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
