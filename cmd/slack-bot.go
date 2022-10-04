package main

import (
	"io"
	"log"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking"
	"github.com/AngelVI13/slack-bot/pkg/slack"
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

	parkingManager := parking.NewManager(eventManager, config)
	eventManager.SubscribeWithContext(
		parkingManager,
		event.SlashCmdEvent,
		event.ViewSubmissionEvent,
		event.BlockActionEvent,
		event.TimerEvent,
	)

	slackClient := slack.NewClient(config, eventManager)
	go slackClient.Listen()

	eventManager.ManageEvents()

}
