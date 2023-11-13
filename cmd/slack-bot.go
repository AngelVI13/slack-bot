package main

import (
	"io"
	"log"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces"
	"github.com/AngelVI13/slack-bot/pkg/parking_users"
	"github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/AngelVI13/slack-bot/pkg/workspaces"
)

func setupLogging(logPath string) {
	// Configure logger
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
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

	resetParkingTimer := event.NewTimer(eventManager)
	resetParkingTimer.AddDaily(
		parking_spaces.ResetHour,
		parking_spaces.ResetMin,
		parking_spaces.ResetParking,
	)
	resetWorkspacesTimer := event.NewTimer(eventManager)
	resetWorkspacesTimer.AddDaily(
		workspaces.ResetHour,
		workspaces.ResetMin,
		workspaces.ResetWorkspaces,
	)

	userManager := user.NewManager(config)

	parkingSpacesManager := parking_spaces.NewManager(
		eventManager,
		userManager,
		config.ParkingFilename,
	)
	eventManager.SubscribeWithContext(
		parkingSpacesManager,
		event.SlashCmdEvent,
		event.ViewSubmissionEvent,
		event.BlockActionEvent,
		event.ViewOpenedEvent,
		event.ViewClosedEvent,
		event.TimerEvent,
	)
	workspacesManager := workspaces.NewManager(
		eventManager,
		userManager,
		config.WorkspacesFilename,
	)
	eventManager.SubscribeWithContext(
		workspacesManager,
		event.SlashCmdEvent,
		event.ViewSubmissionEvent,
		event.BlockActionEvent,
		event.ViewOpenedEvent,
		event.ViewClosedEvent,
		event.TimerEvent,
	)

	parkingUsersManager := parking_users.NewManager(
		eventManager,
		userManager,
		parkingSpacesManager,
	)
	eventManager.SubscribeWithContext(
		parkingUsersManager,
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
