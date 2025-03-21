package hcm

import (
	"encoding/json"
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

type VacationsHash map[string]bool

func LoadVacationsHash(filename string) VacationsHash {
	data := VacationsHash{}

	b, err := os.ReadFile(filename)
	if err != nil {
		slog.Info("Could not read vacations hash file.", "err", err)
		return data
	}

	// Unmarshal the provided data into the solid map
	err = json.Unmarshal(b, &data)
	if err != nil {
		slog.Info("Could not parse vacations hash file.", "err", err)
		return data
	}

	slog.Info("Loaded vacations hash file.", "filename", filename, "hashNum", len(data))
	return data
}

func MakeVacationHash(id int, company user.HcmCompany, startDate, endDate string) string {
	return fmt.Sprintf("%d_%s_%s_%s", id, company, startDate, endDate)
}

type Manager struct {
	eventManager    *event.EventManager
	data            *model.Data
	hcmQdevUrl      string
	hcmQuadUrl      string
	hcmApiToken     string
	debug           bool
	reportPersonId  string
	hcmHashFilename string
	vacationsHash   VacationsHash
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
	conf *config.Config,
) *Manager {
	return &Manager{
		eventManager:    eventManager,
		data:            data,
		hcmQdevUrl:      conf.HcmQdevUrl,
		hcmQuadUrl:      conf.HcmQuadUrl,
		hcmApiToken:     conf.HcmApiToken,
		debug:           conf.Debug,
		reportPersonId:  conf.ReportPersonId,
		hcmHashFilename: conf.HcmHashFilename,
		vacationsHash:   LoadVacationsHash(conf.HcmHashFilename),
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
		if len(usersWithoutHcmId) > 0 {
			actions = append(
				actions,
				m.reportErrorAction(
					fmt.Sprintf(
						"after updating users' hcm ids, there are still users without HCM id: %v",
						usersWithoutHcmId,
					),
				),
			)
		}
	}

	/* TODO: when we are adding user to the users.json we are taking the username from
	   slack but that does not always correlate with HCM (like the examples below)
	   * One solution is to correct the users.json later
	   * Second is to get email from slack and take the names before @ from There
	   * Third is to add the username field in the `/users` modal so that admins
	   can change it later??
	*/
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

	todayDate := common.TodayDate()

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
			if release.StartDate.Before(todayDate) {
				// NOTE: we only create requests for the future. so
				// if a vacation period started 5 days ago and it continues for
				// 3 more days then here we create the release from today
				// till the end of the vacation.
				release.StartDate = &todayDate
			}
			release.EndDate = &vacation.EndDay

			overlaps := m.data.ParkingLot.ToBeReleased.CheckOverlap(release)
			if len(overlaps) > 0 {
				m.data.ParkingLot.ToBeReleased.Remove(release)
				continue
			}

			key := MakeVacationHash(
				employee.Id,
				employee.Company,
				vacation.StartDay.Format("2006-01-02"),
				vacation.EndDay.Format("2006-01-02"),
			)
			m.vacationsHash[key] = true
			release.MarkSubmitted("HCM")

			if common.EqualDate(*release.StartDate, todayDate) {
				// Directly release space if release start from today
				space.Reserved = false
				release.MarkActive()
			}

			slog.Info(
				"HCM add temporary release",
				"user", m.data.UserManager.GetNameFromId(userId),
				"space", space.Key(),
				"HCM request", vacation.Type,
				"HCM date range (clamped)", release.DateRange(),
			)
			info := fmt.Sprintf(
				"Parking bot added a temporary release for your space (%s): "+
					"HCM %s request for %s. "+
					"If that's not correct please contact the system administrator.",
				space.Key(),
				vacation.Type,
				release.DateRange(),
			)
			postAction := common.NewPostEphemeralAction(userId, userId, info, false)
			actions = append(actions, postAction)
		}
	}

	syncErr := m.SynchronizeToFile()
	if syncErr != nil {
		actions = append(actions, m.reportErrorAction(syncErr.Error()))
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

type VacationData map[string][]Vacation

// vacationsInfo Fetches employee vacation information (only current and future
// ones)
func (m *Manager) vacationsInfo(
	hcmUrl string,
	hcmCompany user.HcmCompany,
) (VacationData, error) {
	url := hcmUrl + VacationsOfAllEmployeesEndpoint
	b, err := makeHcmRequest(url, m.hcmApiToken)
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
			key := MakeVacationHash(
				employee.Id,
				hcmCompany,
				period.FirstDay,
				period.LastDay,
			)
			if _, found := m.vacationsHash[key]; found {
				continue
			}

			endDate, parseErr := time.ParseInLocation(
				"2006-01-02",
				period.LastDay,
				location,
			)
			if parseErr != nil {
				return nil, fmt.Errorf(
					"failure to parse lastDay format %s: %v",
					period.LastDay,
					parseErr,
				)
			}

			if endDate.Before(today) {
				m.vacationsHash[key] = true
				continue
			}

			startDate, pErr := time.ParseInLocation(
				"2006-01-02",
				period.FirstDay,
				location,
			)
			if pErr != nil {
				return nil, fmt.Errorf(
					"failure to parse firstDay format %s: %v",
					period.FirstDay,
					pErr,
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

	err = m.SynchronizeToFile()
	return vacationData, err
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
	b, err := makeHcmRequest(url, m.hcmApiToken)
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
			err := m.data.UserManager.SetHcmId(user, employee.Id, hcmCompany)
			if err != nil {
				errs = append(errs, err)
			}
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

func makeHcmRequest(url, token string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create hcm request for url=%q: %v", url, err)
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
		return nil, fmt.Errorf("failed to perform hcm request for url=%q: %v", url, err)
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func (m *Manager) SynchronizeToFile() error {
	data, err := json.MarshalIndent(m.vacationsHash, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshall vacations hash data: %v", err)
	}

	err = os.WriteFile(m.hcmHashFilename, data, 0o666)
	if err != nil {
		return fmt.Errorf(
			"failed to write vacations hash file(%s): %v",
			m.hcmHashFilename,
			err,
		)
	}
	slog.Info("Wrote vacations hashes to file", "file", m.hcmHashFilename)
	return nil
}
