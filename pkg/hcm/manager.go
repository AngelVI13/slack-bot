package hcm

import (
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
)

const HandleHcm = "HandleHCM"

type Manager struct {
	eventManager *event.EventManager
}

func NewManager(eventManager *event.EventManager) *Manager {
	return &Manager{eventManager: eventManager}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.TimerEvent:
		data := e.(*event.TimerDone)
		if data.Label != HandleHcm {
			return
		}

		response := m.handleHcm(data.Time)
		if response == nil {
			return
		}

		m.eventManager.Publish(response)
	}
}

func (m *Manager) Context() string {
	return HandleHcm
}

func (m *Manager) handleHcm(eventTime time.Time) *common.Response {
	var actions []event.ResponseAction

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("HCM", actions...)
}
