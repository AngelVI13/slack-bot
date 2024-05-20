package common

import (
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

func (v *ViewAction) Info() map[string]any {
	return map[string]any{
		"action":    event.ResponseActionNames[v.Action()],
		"triggerId": v.TriggerId,
		"viewId":    v.ViewId,
	}
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

func (p *PostAction) Info() map[string]any {
	return map[string]any{
		"action":    event.ResponseActionNames[p.Action()],
		"channelId": p.ChannelId,
	}
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

func NewPostEphemeralAction(
	channelId, userId string,
	msgOption slack.MsgOption,
) *PostEphemeralAction {
	return &PostEphemeralAction{
		PostAction: PostAction{
			action:    event.PostEphemeral,
			ChannelId: channelId,
			MsgOption: msgOption,
		},
		UserId: userId,
	}
}
