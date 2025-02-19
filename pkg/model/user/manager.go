package user

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/config"
)

type AccessRight int

// NOTE: Currently access rights are not used
const (
	STANDARD AccessRight = iota
	ADMIN
)

type HcmCompany string

const (
	HcmQdev    HcmCompany = "Qdev"
	HcmQuad    HcmCompany = "Quad"
	HcmUnknown HcmCompany = ""
)

type User struct {
	Id                  string
	Rights              AccessRight
	HasPermanentParking bool `json:"has_parking"`
	HcmId               int
	HcmCompany          HcmCompany
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

	slog.Info("INIT: User list loaded successfully", "users", loadedUsersNum)
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

func (m *Manager) SynchronizeToFile() {
	data, err := json.MarshalIndent(m.users, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(m.usersFilename, data, 0o666)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Wrote users list to file")
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

	m.users[userName] = &User{Id: userId, HcmId: -1, HcmCompany: HcmUnknown}
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

func (m *Manager) GetUserIdFromHcmId(hcmId int, hcmCompany HcmCompany) string {
	for _, user := range m.users {
		if user.HcmId == hcmId && user.HcmCompany == hcmCompany {
			return user.Id
		}
	}
	return ""
}

func (m *Manager) AllUserNames() []string {
	var users []string

	for name := range m.users {
		users = append(users, name)
	}
	return users
}

func (m *Manager) SetHcmId(userName string, hcmId int, hcmCompany HcmCompany) error {
	// m.users[userName].HcmId = hcmId
	user, found := m.users[userName]
	if !found {
		return fmt.Errorf("failed to find user by username: %q", userName)
	}

	user.HcmId = hcmId
	user.HcmCompany = hcmCompany
	return nil
}

func (m *Manager) UsersWithoutHcmId() []string {
	var users []string

	for name, user := range m.users {
		if user.HcmId == -1 {
			users = append(users, name)
		}
	}
	return users
}
