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

type Response struct {
	actions []event.ResponseAction
}

func NewResponseEvent(actions ...event.ResponseAction) *Response {
	return &Response{
		actions: actions,
	}
}

func (r *Response) Type() event.EventType {
	return event.ResponseEvent
}

func (r *Response) User() string {
	return ""
}

func (r Response) String() string {
	out := "Response ["
	for _, action := range r.Actions() {
		out += action.String() + ", "
	}
	out += "]"
	return out
}

func (r *Response) HasContext(c string) bool {
	return true
}

func (r *Response) Actions() []event.ResponseAction {
	return r.actions
}
