package user

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
)

type AccessRight int

// NOTE: Currently access rights are not used
const (
	STANDARD AccessRight = iota
	ADMIN
)

type User struct {
	Id                  string
	Rights              AccessRight
	HasPermanentParking bool `json:"has_parking"`
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

func (m *Manager) IsAdminId(userId string) bool {
	for _, user := range m.users {
		if user.Id == userId {
			return user.Rights == ADMIN
		}
	}
	return false
}

func (m *Manager) SetAccessRights(userId string, rights AccessRight) {
	for _, user := range m.users {
		if user.Id == userId {
			user.Rights = rights
			return
		}
	}
}

func (m *Manager) SetParkingPermission(userId string, hasParking bool) {
	for _, user := range m.users {
		if user.Id == userId {
			user.HasPermanentParking = hasParking
			return
		}
	}
}

// InsertUser Inserts new user to user table with default
// permissions: simple user with no parking.
func (m *Manager) InsertUser(userId, userName string) error {
	if m.Exists(userId) {
		return fmt.Errorf("UserId (%s) already exists", userId)
	}

	if _, ok := m.users[userName]; ok {
		return fmt.Errorf("UserName (%s) already exists", userName)
	}

	m.users[userName] = &User{Id: userId}
	return nil
}

func (m *Manager) Exists(userId string) bool {
	for _, user := range m.users {
		if user.Id == userId {
			return true
		}
	}
	return false
}

func (m *Manager) HasParkingById(userId string) bool {
	for _, user := range m.users {
		if user.Id == userId {
			return user.HasPermanentParking
		}
	}
	return false
}

func (m *Manager) GetNameFromId(userId string) string {
	for name, user := range m.users {
		if user.Id == userId {
			return name
		}
	}
	return ""
}
