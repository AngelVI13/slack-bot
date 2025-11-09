package bss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
	"github.com/AngelVI13/slack-bot/pkg/model/user"
)

const (
	StatusUnapproved int = 1
	StatusApproved   int = 2
	StatusCancelled  int = 3
)

type Operation struct {
	OperationNr int    `json:"operationNr"`
	MarkingNr   int    `json:"markingNr"`
	MarkingCode string `json:"markingCode"`
	MarkingName string `json:"markingName"`
	TimeboardNr string `json:"timeboardNo"`
	ValidFrom   string `json:"validFrom"`
	ValidTo     string `json:"validTo"`
	StatusCfgNr int    `json:"statusCfgNr"`
}

type BssResponse struct {
	TotalCount int         `json:"totalCount"`
	PageSize   int         `json:"pageSize"`
	PageNumber int         `json:"pageNumber"`
	PageCount  int         `json:"pageCount"`
	Data       []Operation `json:"data"`
}

const (
	HandleBss                = "HandleBSS"
	LoginEndpoint            = "/auth"
	SearchOperationsEndpoint = "/staff/operations/:search"
)

type Manager struct {
	eventManager    *event.EventManager
	data            *model.Data
	bssConf         config.BssConfig
	debug           bool
	reportPersonId  string
	bssHashFilename string
	vacationsHash   common.VacationsHash
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
	conf *config.Config,
) *Manager {
	return &Manager{
		eventManager:    eventManager,
		data:            data,
		bssConf:         conf.Bss,
		debug:           conf.Debug,
		reportPersonId:  conf.ReportPersonId,
		bssHashFilename: conf.Bss.VacationsHashFilename,
		vacationsHash:   common.LoadVacationsHash(conf.Bss.VacationsHashFilename),
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.TimerEvent:
		data := e.(*event.TimerDone)
		if data.Label != HandleBss {
			return
		}

		response := m.handleBss(data.Time)
		if response == nil {
			return
		}

		m.eventManager.Publish(response)
	}
}

func (m *Manager) Context() string {
	return HandleBss
}

func (m *Manager) handleBss(eventTime time.Time) *common.Response {
	var actions []event.ResponseAction

	quadData, err := m.vacationsInfo(user.Quad)
	if err != nil {
		actions = append(actions, m.reportErrorAction(err.Error()))
		return common.NewResponseEvent("BSS", actions...)
	}

	// qdevData, err := m.vacationsInfo(user.Qdev)
	// if err != nil {
	// 	actions = append(actions, m.reportErrorAction(err.Error()))
	// 	return common.NewResponseEvent("BSS", actions...)
	// }
	// allData := append(quadData, qdevData...)
	allData := quadData

	slog.Info("BSS", "allData", allData)

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("BSS", actions...)
}

type BssTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (m *Manager) login(company user.Company) (*BssTokens, error) {
	fullURL := m.bssConf.Url + LoginEndpoint

	bssCompanyConf := m.bssConf.Quad
	if company == user.Qdev {
		bssCompanyConf = m.bssConf.Qdev
	}

	data := map[string]any{
		"username":      bssCompanyConf.Username,
		"password":      bssCompanyConf.Password,
		"environmentId": bssCompanyConf.EnvironmentId,
		"companyId":     bssCompanyConf.CompanyId,
	}

	b, err := json.Marshal(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request body: %v\n%v", data, err)
	}

	resp, err := makeRequest(fullURL, "", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	var tokens BssTokens
	err = json.Unmarshal(resp, &tokens)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal token response: %v\n%s",
			err,
			string(resp),
		)
	}

	return &tokens, nil
}

func (m *Manager) searchOperations(tokens *BssTokens) (*BssResponse, error) {
	fullURL := m.bssConf.Url + SearchOperationsEndpoint

	// TODO: use correct valid from and valid to dates
	data := map[string]any{
		"Filtering": map[string]any{
			"Filters": []map[string]any{
				{
					"Field":    "statusCfgNr",
					"Value":    2, // status: approved
					"operator": "equal",
				},
				{
					"Field":    "ValidFrom",
					"Value":    "2025-09-15",
					"operator": "greaterOrEqual",
				},
				{
					"Field":    "ValidTo",
					"Value":    "2025-10-01",
					"operator": "lessOrEqual",
				},
			},
		},
		"sorting": []map[string]string{
			{
				"field":     "recordCreationDate",
				"direction": "desc",
			},
		},
	}

	b, err := json.Marshal(&data)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal search ops request body: %v\n%v",
			err,
			data,
		)
	}

	resp, err := makeRequest(fullURL, tokens.AccessToken, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	var bssResp BssResponse
	err = json.Unmarshal(resp, &bssResp)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal bss search response: %v\n%s",
			err,
			string(resp),
		)
	}
	return &bssResp, nil
}

func makeRequest(fullURL, token string, body io.Reader) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create bss request (%q): %v", fullURL, err)
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n\nREQUEST:\n%s\n\n", string(reqDump))

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do bss request (%q): %v", fullURL, err)
	}

	respDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		return nil, err
	}
	fmt.Printf("\n\nRESPONSE:\n%s\n\n", string(respDump))

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read bss response (%q): %v", fullURL, err)
	}

	return b, nil
}

func (m *Manager) reportErrorAction(errTxt string) *common.PostAction {
	postAction := common.NewPostAction(
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
	Key      string
	UserId   string
}

type VacationData []Vacation

func (m *Manager) vacationsInfo(
	company user.Company,
) (VacationData, error) {
	vacationData := VacationData{}

	tokens, err := m.login(user.Quad)
	if err != nil {
		return vacationData, err
	}

	resp, err := m.searchOperations(tokens)
	if err != nil {
		return vacationData, err
	}

	operations := resp.Data

	// filter only current and future vacations
	today := common.TodayDate()
	location := today.Location()
	for _, operation := range operations {
		key := common.MakeBssVacationHash(
			operation.TimeboardNr,
			company,
			operation.ValidFrom,
			operation.ValidTo,
		)
		if _, found := m.vacationsHash[key]; found {
			continue
		}

		userId := m.data.UserManager.GetUserIdFromBssId(operation.TimeboardNr, company)
		if userId == "" {
			// NOTE: we don't add this to vacations hash because if user
			// gets added later, we should process his vacations
			continue
		}

		endDate, parseErr := time.ParseInLocation(
			"2006-01-02",
			operation.ValidTo,
			location,
		)
		if parseErr != nil {
			return nil, fmt.Errorf(
				"failure to parse validTo format %s: %v",
				operation.ValidTo,
				parseErr,
			)
		}

		if endDate.Before(today) {
			m.vacationsHash[key] = true
			continue
		}

		startDate, pErr := time.ParseInLocation(
			"2006-01-02",
			operation.ValidFrom,
			location,
		)
		if pErr != nil {
			return nil, fmt.Errorf(
				"failure to parse firstDay format %s: %v",
				operation.ValidFrom,
				pErr,
			)
		}
		vacationData = append(vacationData, Vacation{
			Type:     operation.MarkingName,
			StartDay: startDate,
			EndDay:   endDate,
			Key:      key,
			UserId:   userId,
		})
	}

	err = m.SynchronizeToFile()
	return vacationData, err
}

func (m *Manager) SynchronizeToFile() error {
	data, err := json.MarshalIndent(m.vacationsHash, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshall BSS vacations hash data: %v", err)
	}

	err = os.WriteFile(m.bssHashFilename, data, 0o666)
	if err != nil {
		return fmt.Errorf(
			"failed to write BSS vacations hash file(%s): %v",
			m.bssHashFilename,
			err,
		)
	}
	slog.Info("Wrote BSS vacations hashes to file", "file", m.bssHashFilename)
	return nil
}
