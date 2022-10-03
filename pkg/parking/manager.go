package parking

import (
	"log"

	"github.com/AngelVI13/slack-bot/pkg/event"
)

const (
	Identifier = "Parking: "
	SlashCmd   = "/parking"
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
	log.Println("PARKING: ", e)
	switch e.Type() {
	case event.SlashCmdEvent:

	case event.BlockActionEvent:

	}

}

func (m *Manager) Context() string {
	return Identifier
}
