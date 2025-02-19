package hcm

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
	"github.com/AngelVI13/slack-bot/pkg/model/user"
)

const (
	HandleHcm                       = "HandleHCM"
	ListEmployeesEndpoint           = "/ext/api/v1/employees"
	VacationsOfAllEmployeesEndpoint = "/ext/api/v1/employees/periods"
)

type HcmEmployee struct {
	Id      int
	Company user.HcmCompany
}

func (e *HcmEmployee) ToKey() string {
	return fmt.Sprintf("%d__%s", e.Id, e.Company)
}

func NewHcmEmployeeFromKey(key string) *HcmEmployee {
	parts := strings.Split(key, "__")
	if len(parts) != 2 {
		log.Fatalf("failed to parse hcm employee key: number of parts %d", len(parts))
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Fatalf(
			"failed to parse hcm employee key: failed to convert hcm id to int %d",
			err,
		)
	}

	return &HcmEmployee{
		Id:      id,
		Company: user.HcmCompany(parts[1]),
	}
}

type Manager struct {
	eventManager   *event.EventManager
	data           *model.Data
	hcmQdevUrl     string
	hcmQuadUrl     string
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
		hcmQdevUrl:     conf.HcmQdevUrl,
		hcmQuadUrl:     conf.HcmQuadUrl,
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
		err := m.updateAllEmployeesInfo()
		if err != nil {
			errTxt := fmt.Sprintf("Error while trying to obtain employee Ids: %v", err)
			actions = append(actions, m.reportErrorAction(errTxt))
		}

		usersWithoutHcmId = m.data.UserManager.UsersWithoutHcmId()
		actions = append(
			actions,
			m.reportErrorAction(
				fmt.Sprintf("There are users without HCM id's: %v", usersWithoutHcmId),
			),
		)
	}

	vacationInfo, err := m.vacationsInfo(m.hcmQdevUrl, user.HcmQdev)
	if err != nil {
		errTxt := fmt.Sprintf(
			"Error while trying to obtain vacation periods for qdev: %v",
			err,
		)
		actions = append(actions, m.reportErrorAction(errTxt))
		return common.NewResponseEvent("HCM", actions...)
	}
	quadVacationInfo, err := m.vacationsInfo(m.hcmQuadUrl, user.HcmQuad)
	if err != nil {
		errTxt := fmt.Sprintf(
			"Error while trying to obtain vacation periods for quadigi: %v",
			err,
		)
		actions = append(actions, m.reportErrorAction(errTxt))
		return common.NewResponseEvent("HCM", actions...)
	}

	// merge both vacation maps into 1
	for k, v := range quadVacationInfo {
		vacationInfo[k] = v
	}

	actions = append(actions, m.addVacationReleases(vacationInfo)...)

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("HCM", actions...)
}

func (m *Manager) addVacationReleases(
	vacationInfo VacationData,
) []event.ResponseAction {
	var actions []event.ResponseAction

	tomorrowDate := common.TodayDate().AddDate(0, 0, 1)

	// TODO: do we need to do something with the company name here?
	// TODO: finalize this and test it.
	// TODO: add special handling for sergey who is in both companies but i
	// think uses the quadigi email for vacations
	// TODO: exclude school half-days and potentially other types of vacations
	for hcmKey, vacations := range vacationInfo {
		employee := NewHcmEmployeeFromKey(hcmKey)
		userId := m.data.UserManager.GetUserIdFromHcmId(employee.Id, employee.Company)
		if userId == "" {
			continue
		}

		space := m.data.ParkingLot.OwnsSpace(userId)
		if space == nil {
			continue
		}

		for _, vacation := range vacations {
			release := m.data.ParkingLot.ToBeReleased.Add(
				"hcmViewId",
				"ParkingBot",
				"ParkingBotId",
				space,
			)
			release.StartDate = &vacation.StartDay
			if release.StartDate.Before(tomorrowDate) {
				// NOTE: we only create requests for the future. so
				// if a vacation period started 5 days ago and it continues for
				// 3 more days then here we create the release from tomorrow
				// till the end of the vacation. This is because this relies
				// on the automatic release/reserve functionality that happens
				// in the ParkingLot object (every day at 17:00).
				release.StartDate = &tomorrowDate
			}
			release.EndDate = &vacation.EndDay

			overlaps := m.data.ParkingLot.ToBeReleased.CheckOverlap(release)
			if len(overlaps) > 0 {
				m.data.ParkingLot.ToBeReleased.Remove(release)
				continue
			}

			release.MarkSubmitted("HCM")
			slog.Info(
				"HCM add temporary release",
				"user", m.data.UserManager.GetNameFromId(userId),
				"space", space.Key(),
				"HCM request", vacation.Type,
				"HCM date range (clamped)", release.DateRange(),
			)
			info := fmt.Sprintf(
				"Parking bot added a temporary release for your space (%s): "+
					"HCM %s request for %s."+
					"If that's not correct please contact the system administrator.",
				space.Key(),
				vacation.Type,
				release.DateRange(),
			)
			// TODO: send these messages to me during testing!!! change the userId here
			postAction := common.NewPostEphemeralAction(userId, userId, info, false)
			actions = append(actions, postAction)
		}
	}

	m.data.ParkingLot.SynchronizeToFile()

	return actions
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

type Vacation struct {
	Type     string
	StartDay time.Time
	EndDay   time.Time
}

// func (v Vacation) String() string {
// 	return fmt.Sprintf("%s request for %s-%s", v.Type,
// 		v.StartDay.Format("2006-01-02"),
// 		v.EndDay.Format("2006-01-02"))
// }

// TODO: What if employee IDs can be the same for both companies
// i.e. id 33 means one person in QDev and another in Quadigi ???
type VacationData map[string][]Vacation

// vacationsInfo Fetches employee vacation information (only current and future
// ones)
func (m *Manager) vacationsInfo(
	hcmUrl string,
	hcmCompany user.HcmCompany,
) (VacationData, error) {
	url := hcmUrl + VacationsOfAllEmployeesEndpoint
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
	vacationData := VacationData{}
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
				Type:     period.Type,
				StartDay: startDate,
				EndDay:   endDate,
			})
			hcmEmployee := HcmEmployee{
				Id:      employee.Id,
				Company: hcmCompany,
			}
			vacationData[hcmEmployee.ToKey()] = currentVacations
		}
	}
	return vacationData, nil
}

func (m *Manager) updateAllEmployeesInfo() error {
	var errs []error
	err := m.updateEmployeesInfo(m.hcmQdevUrl, user.HcmQdev)
	if err != nil {
		errs = append(errs, fmt.Errorf("error updating Qdev employees info: %w", err))
	}

	err = m.updateEmployeesInfo(m.hcmQuadUrl, user.HcmQuad)
	if err != nil {
		errs = append(errs, fmt.Errorf("error updating Quadigi employees info: %w", err))
	}

	return errors.Join(errs...)
}

func (m *Manager) updateEmployeesInfo(hcmUrl string, hcmCompany user.HcmCompany) error {
	var errs []error

	url := hcmUrl + ListEmployeesEndpoint
	b, err := makeHcmRequest(url, m.hcmApiToken, m.debug)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to make hcm request: %v", err))
		return errors.Join(errs...)
	}

	var info EmployeeInfo
	err = xml.Unmarshal(b, &info)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to unmarshal employees info: %v", err))
		return errors.Join(errs...)
	}

	users := m.data.UserManager.AllUserNames()
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
			errs = append(
				errs,
				fmt.Errorf("failed to make regex from employee name: %q. %v", name, err),
			)
			continue
		}

		for _, user := range users {
			if !regx.MatchString(user) {
				continue
			}
			slog.Info("found user for employee", "employee", name, "user", user)
			m.data.UserManager.SetHcmId(user, employee.Id, hcmCompany)
			break
		}
	}
	m.data.UserManager.SynchronizeToFile()

	return errors.Join(errs...)
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
