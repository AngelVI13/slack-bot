package hcm

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"
	"unicode"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
)

const (
	HandleHcm                       = "HandleHCM"
	ListEmployeesEndpoint           = "/ext/api/v1/employees"
	VacationsOfAllEmployeesEndpoint = "/ext/api/v1/employees/periods"
)

type Manager struct {
	eventManager *event.EventManager
	data         *model.Data
	hcmUrl       string
	hcmApiToken  string
	debug        bool
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
	conf *config.Config,
) *Manager {
	return &Manager{
		eventManager: eventManager,
		data:         data,
		hcmUrl:       conf.HcmUrl,
		hcmApiToken:  conf.HcmApiToken,
		debug:        conf.Debug,
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.TimerEvent:
		data := e.(*event.TimerDone)
		if data.Label != HandleHcm {
			return
		}

		response := m.handleHcm(data.Time)
		if response == nil {
			return
		}

		m.eventManager.Publish(response)
	}
}

func (m *Manager) Context() string {
	return HandleHcm
}

func (m *Manager) handleHcm(eventTime time.Time) *common.Response {
	var actions []event.ResponseAction

	usersWithoutHcmId := m.data.UserManager.UsersWithoutHcmId()
	if len(usersWithoutHcmId) > 0 {
		err := m.employeesInfo()
		log.Fatal(err)
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("HCM", actions...)
}

func (m *Manager) employeesInfo() error {
	url := m.hcmUrl + ListEmployeesEndpoint
	b, err := makeHcmRequest(url, m.hcmApiToken, m.debug)
	if err != nil {
		return fmt.Errorf("failed to make hcm request: %v", err)
	}

	var info EmployeeInfo
	err = xml.Unmarshal(b, &info)
	if err != nil {
		return fmt.Errorf("failed to unmarshal employees info: %v", err)
	}

	users := m.data.UserManager.AllUserNames()
	for _, employee := range info.Items {
		name := employee.Values[0].Name
		// TODO: some people have multiple names so remove everything but the
		// first and last name and also replace the space in the middle with a
		// escaped dot
		regx, err := MakeRegexFromName(name)
		if err != nil {
			slog.Info("failed to make regex from employee name", "name", name)
		}

		found := false
		for _, user := range users {
			if !regx.MatchString(user) {
				continue
			}
			found = true
			slog.Info("found user for employee", "employee", name, "user", user)
			break
		}

		if !found {
			slog.Info("failed to find user for employee", "employee", name)
		}
	}
	fmt.Println(info)

	return nil
}

// MakeRegexFromName turn a name into regexp pattern. Any non ASCII char is
// replaced with '.'
func MakeRegexFromName(name string) (*regexp.Regexp, error) {
	pattern := ""
	for c := range name {
		if c > unicode.MaxASCII {
			pattern += "."
		} else {
			pattern = fmt.Sprintf("%s%c", pattern, c)
		}
	}
	return regexp.Compile(pattern)
}

/*
<List>

	<item>
	  <id>14</id>
	  <values>
	    <values>
	      <label>name</label>
	      <textValue>Oleg Krukovskij</textValue>
	      <dateValidFrom>2015-04-01</dateValidFrom>
	    </values>
	  </values>
	</item>

</List>
*/

type EmployeeValue struct {
	Name      string `xml:"textValue"`
	StartDate string `xml:"dateValidFrom"`
}

type EmployeeItem struct {
	Id     int             `xml:"id"`
	Values []EmployeeValue `xml:"values>values"`
}
type EmployeeInfo struct {
	Items []EmployeeItem `xml:"item"`
}

func makeHcmRequest(url, token string, debug bool) ([]byte, error) {
	if debug {
		// TODO: remove this later
		return os.ReadFile("out.xml")
	}
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("x-api-key", token)
	// NOTE: if this is missing the the reply is in XML format
	// Might be more useful to use the XML format because it contains escape codes
	// For lithuanian alphabet special characters whereas json returns the literal characters
	// Might be easiest if i replace the xml espace codes with `.` and perform a regex search to match
	// a user in the parking bot users.json
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}
