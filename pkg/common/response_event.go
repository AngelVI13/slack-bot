package common

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/event"
)

type Response struct {
	user    string
	actions []event.ResponseAction
}

func NewResponseEvent(user string, actions ...event.ResponseAction) *Response {
	return &Response{
		user:    user,
		actions: actions,
	}
}

func NewAnonResponseEvent(actions ...event.ResponseAction) *Response {
	return &Response{
		user:    "",
		actions: actions,
	}
}

func (r *Response) Type() event.EventType {
	return event.ResponseEvent
}

func (r *Response) User() string {
	return r.user
}

func (r *Response) Info() map[string]any {
	out := map[string]any{}

	for i, action := range r.Actions() {
		actionStr := fmt.Sprintf("%s.%d", event.ResponseActionNames[action.Action()], i)
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
