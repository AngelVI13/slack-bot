package main

import (
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/edit_parking_spaces"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces"
	"github.com/AngelVI13/slack-bot/pkg/parking_users"
	"github.com/AngelVI13/slack-bot/pkg/roll"
	"github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/AngelVI13/slack-bot/pkg/workspaces"
)

func setupLogging(logPath string) *os.File {
	// Configure logger
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	wrt := io.MultiWriter(os.Stdout, logFile)

	logger := slog.New(slog.NewTextHandler(wrt, nil))
	// logger := slog.New(slog.NewJSONHandler(wrt, nil))
	slog.SetDefault(logger)

	return logFile
}

func main() {
	logFile := setupLogging("slack-bot.log")
	defer logFile.Close()

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

	editParkingSpacesManager := edit_parking_spaces.NewManager(
		eventManager,
		userManager,
		parkingSpacesManager,
	)
	eventManager.SubscribeWithContext(
		editParkingSpacesManager,
		event.SlashCmdEvent,
		event.ViewSubmissionEvent,
		event.BlockActionEvent,
		event.ViewOpenedEvent,
		event.ViewClosedEvent,
		event.TimerEvent,
	)
	rollManager := roll.NewManager(eventManager)
	eventManager.Subscribe(rollManager, event.SlashCmdEvent)

	slackClient := slack.NewClient(config, eventManager)
	eventManager.Subscribe(slackClient, event.ResponseEvent)

	go slackClient.Listen()

	eventManager.ManageEvents()
}
