package slack

import (
	"log/slog"
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

func (i *Interaction) IValue(blockId, actionId string) string {
	var values slack.BlockAction
	if blockId == "" {
		// incase element is provided as attachment object then it has a randomly
		// provided blockId -> search through all of them and select the one which has
		// the same expected actionId
		var found bool
		for _, v := range i.Values {
			values, found = v[actionId]
			if found {
				break
			}
		}
	} else {
		values = i.Values[blockId][actionId]
	}

	if values.Value != "" {
		return values.Value
	} else if values.SelectedOption.Value != "" {
		return values.SelectedOption.Value
	} else if len(values.SelectedOptions) > 0 {
		var out []string
		for _, v := range values.SelectedOptions {
			out = append(out, v.Value)
		}
		return strings.Join(out, ",")
	} else if values.SelectedUser != "" {
		return values.SelectedUser
	} else if len(values.SelectedUsers) > 0 {
		return strings.Join(values.SelectedUsers, ",")
	} else if values.SelectedChannel != "" {
		return values.SelectedChannel
	} else if len(values.SelectedChannels) > 0 {
		return strings.Join(values.SelectedChannels, ",")
	} else if values.SelectedConversation != "" {
		return values.SelectedConversation
	} else if len(values.SelectedConversations) > 0 {
		return strings.Join(values.SelectedConversations, ",")
	} else if values.SelectedDate != "" {
		return values.SelectedDate
	} else if values.SelectedTime != "" {
		return values.SelectedTime
	}
	return ""
}

func (i *Interaction) ActionInfo() map[string]string {
	out := map[string]string{}
	for _, action := range i.Actions {
		value := i.IValue(action.BlockID, action.ActionID)
		if value == "" {
			value = action.Value
		}
		out[action.ActionID] = value
	}

	return out
}

type ViewSubmission struct {
	Interaction
}

func (v *ViewSubmission) Type() event.EventType {
	return event.ViewSubmissionEvent
}

func (v *ViewSubmission) Info() map[string]any {
	return map[string]any{
		"title": v.Title,
	}
}

type BlockAction struct {
	Interaction
	SelectedUserName string
}

func (b *BlockAction) Type() event.EventType {
	return event.BlockActionEvent
}

func (b *BlockAction) Info() map[string]any {
	return map[string]any{
		"info":  b.ActionInfo(),
		"title": b.Title,
	}
}

type ViewClosed struct {
	Interaction
}

func (v *ViewClosed) Type() event.EventType {
	return event.ViewClosedEvent
}

func (v *ViewClosed) Info() map[string]any {
	return map[string]any{
		"title": v.Title,
	}
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

func (v *ViewOpened) Info() map[string]any {
	return map[string]any{
		"title": v.Title,
	}
}

func (v *ViewOpened) HasContext(c string) bool {
	return strings.Contains(v.Title, c)
}

func handleInteractionEvent(socketEvent socketmode.Event, c *Client) event.Event {
	interactionCb, ok := socketEvent.Data.(slack.InteractionCallback)
	if !ok {
		slog.Error(
			"Could not type cast the message to a Interaction callback", "event",
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
		event = &BlockAction{
			Interaction:      interaction,
			SelectedUserName: userNameForSelectedUser(interaction, c),
		}
	case slack.InteractionTypeViewClosed:
		event = &ViewClosed{interaction}
	default:
		slog.Error(
			"Unsupported interaction event",
			"interactionCbType", interactionCb.Type,
			"interaction", interaction,
		)
		return nil
	}

	return event
}

func userNameForSelectedUser(interaction Interaction, c *Client) string {
	if len(interaction.Actions) <= 0 {
		return ""
	}
	userId := interaction.Actions[0].SelectedUser
	if userId == "" {
		return ""
	}
	userData, err := c.socket.GetUserInfo(userId)
	if err != nil || userData == nil {
		return ""
	}
	return userData.Name
}
