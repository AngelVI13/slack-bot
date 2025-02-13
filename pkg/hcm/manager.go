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
	"strings"
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
	eventManager   *event.EventManager
	data           *model.Data
	hcmUrl         string
	hcmApiToken    string
	debug          bool
	reportPersonId string
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
	conf *config.Config,
) *Manager {
	return &Manager{
		eventManager:   eventManager,
		data:           data,
		hcmUrl:         conf.HcmUrl,
		hcmApiToken:    conf.HcmApiToken,
		debug:          conf.Debug,
		reportPersonId: conf.ReportPersonId,
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
		err := m.updateEmployeesInfo()
		if err != nil {
			errTxt := fmt.Sprintf("Error while trying to obtain employee Ids: %v", err)
			actions = append(actions, m.reportErrorAction(errTxt))
		}
	}

	vacationInfo, err := m.vacationsInfo()
	if err != nil {
		errTxt := fmt.Sprintf("Error while trying to obtain vacation periods: %v", err)
		actions = append(actions, m.reportErrorAction(errTxt))
	}
	fmt.Println(vacationInfo)

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("HCM", actions...)
}

func (m *Manager) reportErrorAction(errTxt string) *common.PostEphemeralAction {
	postAction := common.NewPostEphemeralAction(
		m.reportPersonId,
		m.reportPersonId,
		errTxt,
		false,
	)
	return postAction
}

func (m *Manager) employeesVacations() error {
	url := m.hcmUrl + VacationsOfAllEmployeesEndpoint
	b, err := makeHcmRequest(url, m.hcmApiToken, m.debug)
	if err != nil {
		return fmt.Errorf("failed to make hcm request: %v", err)
	}
	_ = b
	return nil
}

type Vacation struct {
	StartDay time.Time
	EndDay   time.Time
}

type VacationData map[int][]Vacation

// vacationsInfo Fetches employee vacation information (only current and future
// ones)
func (m *Manager) vacationsInfo() (VacationData, error) {
	url := m.hcmUrl + VacationsOfAllEmployeesEndpoint
	b, err := makeHcmRequest(url, m.hcmApiToken, m.debug)
	if err != nil {
		return nil, fmt.Errorf("failed to make hcm request: url=%q err=%v", url, err)
	}

	var info VacationInfo
	err = xml.Unmarshal(b, &info)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal vacations info: %v", err)
	}

	// filter only current and future vacations
	today := common.TodayDate()
	location := today.Location()
	vacationData := map[int][]Vacation{}
	for _, employee := range info.Items {
		var currentVacations []Vacation

		for _, period := range employee.Periods {
			endDate, err := time.ParseInLocation("2006-01-02", period.LastDay, location)
			if err != nil {
				return nil, fmt.Errorf(
					"failure to parse lastDay format %s: %v",
					period.LastDay,
					err,
				)
			}

			if endDate.Before(today) {
				continue
			}

			startDate, err := time.ParseInLocation(
				"2006-01-02",
				period.FirstDay,
				location,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"failure to parse firstDay format %s: %v",
					period.FirstDay,
					err,
				)
			}
			currentVacations = append(currentVacations, Vacation{
				StartDay: startDate,
				EndDay:   endDate,
			})
			vacationData[employee.Id] = currentVacations
		}
	}
	return vacationData, nil
}

func (m *Manager) updateEmployeesInfo() error {
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
	slog.Info("users without HCM id", "users", users)
	for _, employee := range info.Items {
		originalName := employee.Values[0].Name
		parts := strings.Split(originalName, " ")
		// format name pattern as first name & last name separated by a dot
		// this is done as some people have 5 names but slack-bot only cares about
		// first_name.last_name
		name := fmt.Sprintf("%s\\.%s", parts[0], parts[len(parts)-1])
		name = strings.ToLower(name)

		regx, err := MakeRegexFromName(name)
		if err != nil {
			slog.Info("failed to make regex from employee name", "name", name)
		}

		slog.Info("Employee as regex", "originalName", originalName, "reg", regx.String())

		found := false
		for _, user := range users {
			if !regx.MatchString(user) {
				continue
			}
			found = true
			slog.Info("found user for employee", "employee", name, "user", user)
			m.data.UserManager.SetHcmId(user, employee.Id)
			break
		}

		if !found {
			slog.Info("failed to find user for employee", "employee", name)
		}
	}
	/* TODO: users that are not matched correctly are stored in the names_to_be_corrected.txt
		   rerun this code with the users_new.json (replace the users.json with it) and
	       edit the users.json to match the different slack names.
	*/
	m.data.UserManager.SynchronizeToFile()

	return nil
}

// MakeRegexFromName turn a name into regexp pattern. Any non ASCII char is
// replaced with '.'
func MakeRegexFromName(name string) (*regexp.Regexp, error) {
	pattern := ""
	for _, c := range name {
		if c > unicode.MaxASCII {
			pattern += "."
		} else {
			pattern = fmt.Sprintf("%s%c", pattern, c)
		}
	}
	return regexp.Compile(pattern)
}

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

type PeriodValue struct {
	Type     string `xml:"type"`
	FirstDay string `xml:"firstDay"`
	LastDay  string `xml:"lastDay"`
}

type VacationItem struct {
	Id      int           `xml:"id"`
	Periods []PeriodValue `xml:"periods>periods"`
}

type VacationInfo struct {
	Items []VacationItem `xml:"item"`
}

func makeHcmRequest(url, token string, debug bool) ([]byte, error) {
	// TODO: remove this later
	if debug {
		if strings.HasSuffix(url, ListEmployeesEndpoint) {
			return os.ReadFile("example_employee_list.xml")
		} else if strings.HasSuffix(url, VacationsOfAllEmployeesEndpoint) {
			return os.ReadFile("vacations.xml")
		}
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
	// req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}
