package slack

import (
	"fmt"
	"log"

	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type Slash struct {
	BaseEvent
	Command   string
	TriggerId string
}

func (s *Slash) Type() event.EventType {
	return event.SlashCmdEvent
}

func (s Slash) String() string {
	return fmt.Sprintf("%s - %s", event.DefaultEventString(&s), s.Command)
}

func (s *Slash) HasContext(c string) bool {
	return true
}

func handleSlashCommand(socketEvent socketmode.Event) event.Event {
	command, ok := socketEvent.Data.(slack.SlashCommand)
	if !ok {
		log.Fatalf("Could not type cast the message to a SlashCommand: %v\n", command)
	}

	return &Slash{
		BaseEvent: BaseEvent{
			UserName: command.UserName,
			UserId:   command.UserID,
		},
		Command:   command.Command,
		TriggerId: command.TriggerID,
	}
}
