package reminder

import (
	"log"
	"math/rand"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/parking_spaces"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier     = "Reminder: "
	UseAppReminder = "UseAppReminder"
)

type Manager struct {
	eventManager   *event.EventManager
	userManager    *user.Manager
	parkingManager *parking_spaces.Manager
	slackClient    *slack.Client
}

func NewManager(
	eventManager *event.EventManager,
	userManager *user.Manager,
	parkingManager *parking_spaces.Manager,
) *Manager {
	return &Manager{
		eventManager:   eventManager,
		userManager:    userManager,
		parkingManager: parkingManager,
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.TimerEvent:
		data := e.(*event.TimerDone)
		if data.Label != UseAppReminder {
			return
		}

		log.Println("Reminder")
		m.handleAppReminder()
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleAppReminder() *common.Response {
	reminderTxt := `This is a reminder that the company uses a parking reservation 
        system QDParking. Please make sure to use it to reserve/release 
        a parking space. More information how to use parking reservation 
        system in Sharepoint`
	log.Println(reminderTxt)

	usersMap := m.userManager.Users()

	numUsers := len(usersMap)
	allUsers := make([]string, 0, numUsers)
	for name := range usersMap {
		allUsers = append(allUsers, name)
	}

	// Don't spam everyone every single week
	groupSize := numUsers / 4
	selectedUsers := make(map[string]user.User, groupSize)
	for len(selectedUsers) < groupSize {
		chosenIdx := rand.Intn(numUsers)
		chosenName := allUsers[chosenIdx]
		selectedUsers[chosenName] = usersMap[chosenName]
	}
	log.Println(selectedUsers)
	// TODO: finish and test this

	/*
		action := common.NewPostEphemeralAction(
			data.UserId,
			data.UserId,
			slack.MsgOptionText(reminderTxt, false),
		)
	*/
	// return common.NewResponseEvent(action)
	return nil
}
