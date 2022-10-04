package parking

import (
	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/slack"
)

const (
	Identifier = "Parking: "
	// SlashCmd   = "/parking"
	SlashCmd = "/test-park"
)

type Manager struct {
	eventManager *event.EventManager
	parkingLot   *ParkingLot
}

func NewManager(eventManager *event.EventManager, config *config.Config) *Manager {
	parkingLot := GetParkingLot(config)

	return &Manager{
		eventManager: eventManager,
		parkingLot:   &parkingLot,
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.SlashCmdEvent:
		data := e.(*slack.Slash)
		if data.Command != SlashCmd {
			return
		}

		spaces := m.parkingLot.GetSpacesInfo(data.UserName)
		modal := GenerateModalRequest(data, spaces)

		action := common.NewViewAction(event.OpenView, data.TriggerId, modal)
		response := common.NewResponseEvent(action)

		m.eventManager.Publish(response)
	case event.BlockActionEvent:

	}

}

func (m *Manager) Context() string {
	return Identifier
}
