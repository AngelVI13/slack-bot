package user

import (
	"encoding/json"
	"log"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/slack-go/slack"
)

type AccessRight int

// NOTE: Currently access rights are not used
const (
	STANDARD AccessRight = iota
	ADMIN
)

type User struct {
	Id         string
	Rights     AccessRight
	IsReviewer bool `json:"is_reviewer"`
}

type UsersMap map[string]*User

func getUsers(path string) (users UsersMap) {
	fileData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Could not read users file (%s)", path)
	}

	err = json.Unmarshal(fileData, &users)
	if err != nil {
		log.Fatalf("Could not parse users file (%s). Error: %+v", path, err)
	}

	loadedUsersNum := len(users)
	if loadedUsersNum == 0 {
		log.Fatalf("No users found in (%s).", path)
	}

	log.Printf("INIT: User list loaded successfully (%d users configured)", loadedUsersNum)
	return users
}

type Manager struct {
	users         UsersMap
	usersFilename string
}

func NewManager(config *config.Config) *Manager {
	usersMap := getUsers(config.UsersFilename)

	return &Manager{
		users:         usersMap,
		usersFilename: config.UsersFilename,
	}
}

func (m *Manager) synchronizeToFile() {
	data, err := json.MarshalIndent(m.users, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(m.usersFilename, data, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("INFO: Wrote users list to file")
}

func (m *Manager) AddNewUsers(selectedManager []*slack.User, selectedOptions []slack.OptionBlockObject, accsessRightSelection string, reviewerOptionSelection string) {
	accessRights := STANDARD
	isReviewer := false

	for _, selection := range selectedOptions {
		switch selection.Value {
		case accsessRightSelection:
			accessRights = ADMIN
		case reviewerOptionSelection:
			isReviewer = true
		}
	}

	for _, userInfo := range selectedManager {
		userName := userInfo.Name
		log.Printf("Adding %s", userName)

		m.users[userName] = &User{
			Id:         userInfo.ID,
			Rights:     accessRights,
			IsReviewer: isReviewer,
		}
	}

	m.synchronizeToFile()
}

func (m *Manager) IsSpecial(userName string) bool {
	user, ok := m.users[userName]
	if !ok {
		return false
	}
	return user.Rights == ADMIN
}