package common

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

type ViewAction struct {
	action       event.ResponseActionType
	TriggerId    string
	ModalRequest slack.ModalViewRequest
}

func NewViewAction(
	action event.ResponseActionType,
	triggerId string,
	modalRequest slack.ModalViewRequest,
) *ViewAction {
	return &ViewAction{
		action:       action,
		TriggerId:    triggerId,
		ModalRequest: modalRequest,
	}
}

func (v ViewAction) String() string {
	return fmt.Sprintf("%s, TriggerID: %s", event.ResponseActionNames[v.Action()], v.TriggerId)
}

func (v *ViewAction) Action() event.ResponseActionType {
	return v.action
}

type PostAction struct {
	action    event.ResponseActionType
	channelId string
	msgOption slack.MsgOption
}

func (p *PostAction) Action() event.ResponseActionType {
	return p.action
}

func (p PostAction) String() string {
	return fmt.Sprintf("%s, ChannelId: %s", event.ResponseActionNames[p.Action()], p.channelId)
}

func NewPostAction(channelId string, msgOption slack.MsgOption) *PostAction {
	return &PostAction{
		action:    event.Post,
		channelId: channelId,
		msgOption: msgOption,
	}
}

type PostEphemeralAction struct {
	PostAction
	userId string
}

func NewPostEphemeralAction(channelId, userId string, msgOption slack.MsgOption) *PostEphemeralAction {
	return &PostEphemeralAction{
		PostAction: PostAction{
			action:    event.PostEphemeral,
			channelId: channelId,
			msgOption: msgOption,
		},
		userId: userId,
	}
}