package slack

import (
	"log"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type Slash struct {
	BaseEvent
	Command     string
	TriggerId   string
	ChannelName string
	ChannelId   string
}

func (s *Slash) Type() event.EventType {
	return event.SlashCmdEvent
}

func (s *Slash) Info() map[string]any {
	return map[string]any{
		"cmd": s.Command,
	}
}

func (s *Slash) HasContext(c string) bool {
	return true
}

func handleSlashCommand(socketEvent socketmode.Event) event.Event {
	command, ok := socketEvent.Data.(slack.SlashCommand)
	if !ok {
		log.Printf(
			"ERROR: Could not type cast the message to a SlashCommand: %v\n",
			command,
		)
		return nil
	}

	return &Slash{
		BaseEvent: BaseEvent{
			UserName: command.UserName,
			UserId:   command.UserID,
		},
		Command:     command.Command,
		TriggerId:   command.TriggerID,
		ChannelName: command.ChannelName,
		ChannelId:   command.ChannelID,
	}
}
