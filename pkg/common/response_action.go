package common

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
)

type ViewAction struct {
	action       event.ResponseActionType
	TriggerId    string
	ViewId       string
	ModalRequest slack.ModalViewRequest
}

func NewUpdateViewAction(
	triggerId,
	viewId string,
	modalRequest slack.ModalViewRequest,
) *ViewAction {
	return &ViewAction{
		action:       event.UpdateView,
		TriggerId:    triggerId,
		ViewId:       viewId,
		ModalRequest: modalRequest,
	}
}

func NewOpenViewAction(
	triggerId string,
	modalRequest slack.ModalViewRequest,
) *ViewAction {
	return &ViewAction{
		action:       event.OpenView,
		TriggerId:    triggerId,
		ViewId:       "",
		ModalRequest: modalRequest,
	}
}

func NewPushViewAction(
	triggerId string,
	modalRequest slack.ModalViewRequest,
) *ViewAction {
	return &ViewAction{
		action:       event.PushView,
		TriggerId:    triggerId,
		ViewId:       "",
		ModalRequest: modalRequest,
	}
}

func (v ViewAction) String() string {
	return fmt.Sprintf(
		"%s, TriggerID: %s ViewId: %s",
		event.ResponseActionNames[v.Action()],
		v.TriggerId,
		v.ViewId,
	)
}

func (v *ViewAction) Action() event.ResponseActionType {
	return v.action
}

type PostAction struct {
	action    event.ResponseActionType
	ChannelId string
	MsgOption slack.MsgOption
}

func (p *PostAction) Action() event.ResponseActionType {
	return p.action
}

func (p PostAction) String() string {
	return fmt.Sprintf("%s, ChannelId: %s", event.ResponseActionNames[p.Action()], p.ChannelId)
}

func NewPostAction(channelId string, msgOption slack.MsgOption) *PostAction {
	return &PostAction{
		action:    event.Post,
		ChannelId: channelId,
		MsgOption: msgOption,
	}
}

type PostEphemeralAction struct {
	PostAction
	UserId string
}

func NewPostEphemeralAction(channelId, userId string, msgOption slack.MsgOption) *PostEphemeralAction {
	return &PostEphemeralAction{
		PostAction: PostAction{
			action:    event.PostEphemeral,
			ChannelId: channelId,
			MsgOption: msgOption,
		},
		UserId: userId,
	}
}
