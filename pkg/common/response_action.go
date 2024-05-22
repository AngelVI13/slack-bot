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
	ErrTxt       string
}

func NewUpdateViewAction(
	triggerId,
	viewId string,
	modalRequest slack.ModalViewRequest,
	errTxt string,
) *ViewAction {
	return &ViewAction{
		action:       event.UpdateView,
		TriggerId:    triggerId,
		ViewId:       viewId,
		ModalRequest: modalRequest,
		ErrTxt:       errTxt,
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
	out := map[string]any{}

	if v.ErrTxt != "" {
		out["error"] = v.ErrTxt
	}

	return out
}

func (v *ViewAction) Action() event.ResponseActionType {
	return v.action
}

type PostAction struct {
	action    event.ResponseActionType
	ChannelId string
	MsgOption slack.MsgOption
	Txt       string
}

func (p *PostAction) Action() event.ResponseActionType {
	return p.action
}

func (p *PostAction) Info() map[string]any {
	return map[string]any{
		"txt":       p.Txt,
		"channelId": p.ChannelId,
	}
}

func NewPostAction(channelId, txt string, escape bool) *PostAction {
	return &PostAction{
		action:    event.Post,
		ChannelId: channelId,
		MsgOption: slack.MsgOptionText(txt, escape),
		Txt:       txt,
	}
}

type PostEphemeralAction struct {
	PostAction
	UserId string
}

func NewPostEphemeralAction(
	channelId, userId, txt string,
	escape bool,
) *PostEphemeralAction {
	return &PostEphemeralAction{
		PostAction: PostAction{
			action:    event.PostEphemeral,
			ChannelId: channelId,
			MsgOption: slack.MsgOptionText(txt, escape),
			Txt:       txt,
		},
		UserId: userId,
	}
}
