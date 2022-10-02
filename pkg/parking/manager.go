package parking

import "github.com/AngelVI13/slack-bot/pkg/event"

const (
	Identifier = "parking"
)

type Manager struct {
	eventManager *event.EventManager
}

func NewManager(eventManager *event.EventManager) *Manager {
	return &Manager{
		eventManager: eventManager,
	}
}

func (m *Manager) Consume(e event.Event) {
}

func (m *Manager) Context() string {
	return Identifier
}
