package roll

import (
	"fmt"
	"math/rand"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
)

const (
	Identifier   = "Roll: "
	SlashCmd     = "/roll"
	TestSlashCmd = "/test-roll"
)

type Manager struct {
	eventManager  *event.EventManager
	testingActive bool
}

func NewManager(eventManager *event.EventManager, conf *config.Config) *Manager {
	return &Manager{
		eventManager:  eventManager,
		testingActive: conf.TestingActive,
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.SlashCmdEvent:
		data := e.(*slackApi.Slash)
		if !common.ShouldProcessSlash(
			data.Command,
			SlashCmd,
			TestSlashCmd,
			m.testingActive,
		) {
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
