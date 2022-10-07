package main

import (
	"io"
	"log"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking"
	"github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
)

func setupLogging(logPath string) {
	// Configure logger
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()

	wrt := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(wrt)
}

func main() {
	setupLogging("slack-bot.log")
	config := config.NewConfigFromEnv(".env")

	eventManager := event.NewEventManager()

	logger := event.NewEventLogger()
	eventManager.Subscribe(logger, event.AnyEvent)

	timer := event.NewTimer(eventManager)
	timer.AddDaily(17, 00, "Reset parking status")

	userManager := user.NewManager(config)

	parkingManager := parking.NewManager(eventManager, config, userManager)
	eventManager.SubscribeWithContext(
		parkingManager,
		event.SlashCmdEvent,
		event.ViewSubmissionEvent,
		event.BlockActionEvent,
		event.ViewOpenedEvent,
		event.ViewClosedEvent,
		event.TimerEvent,
	)

	slackClient := slack.NewClient(config, eventManager)
	eventManager.Subscribe(slackClient, event.ResponseEvent)

	go slackClient.Listen()

	eventManager.ManageEvents()
}
