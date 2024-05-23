package roll

import (
	"fmt"
	"math/rand"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
)

const (
	Identifier = "Roll: "
	SlashCmd   = "/roll"
	// SlashCmd = "/test-roll"
)

type Manager struct {
	eventManager *event.EventManager
}

func NewManager(eventManager *event.EventManager) *Manager {
	return &Manager{eventManager: eventManager}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.SlashCmdEvent:
		data := e.(*slackApi.Slash)
		if data.Command != SlashCmd {
			return
		}
		response := m.handleSlashCmd(data)
		m.eventManager.Publish(response)
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	roll := rand.Intn(100) + 1
	text := fmt.Sprintf("%s rolled %d", data.UserName, roll)
	action := common.NewPostAction(data.ChannelId, text, true)
	response := common.NewResponseEvent(data.UserName, action)
	return response
}
