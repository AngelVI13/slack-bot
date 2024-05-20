package common

import (
	"github.com/AngelVI13/slack-bot/pkg/event"
)

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

func (r *Response) Info() map[string]any {
	out := map[string]any{}

	for _, action := range r.Actions() {
		actionStr := event.ResponseActionNames[action.Action()]
		out[actionStr] = action.Info()
	}
	return out
}

func (r *Response) HasContext(c string) bool {
	return true
}

func (r *Response) Actions() []event.ResponseAction {
	return r.actions
}
