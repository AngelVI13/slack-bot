package bss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
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
	ValidFrom   string `json:"validFrom"`
	ValidTo     string `json:"validTo"`
	StatusCfgNr int    `json:"statusCfgNr"`
}

type Response struct {
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
		bssHashFilename: conf.VacationsHashFilename,
		vacationsHash:   common.LoadVacationsHash(conf.VacationsHashFilename),
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

	tokens := m.login(user.Quad)
	slog.Info("bss login", "tokens", tokens)

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("BSS", actions...)
}

type BssTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// TODO: return error and send it to reportPerson
func (m *Manager) login(company user.Company) BssTokens {
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
	fmt.Println(fullURL, string(b))
	if err != nil {
		log.Fatalf("Failed to marshal login request body: %v\n%v", data, err)
	}

	respStr := makeRequest(fullURL, "", bytes.NewBuffer(b))
	// fmt.Println(respStr)
	var tokens BssTokens
	err = json.Unmarshal([]byte(respStr), &tokens)
	if err != nil {
		log.Fatalf("failed to unmarshal token response: %v", err)
	}

	// fmt.Println(tokens)
	return tokens
}

func makeRequest(fullURL, token string, body io.Reader) string {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, fullURL, body)
	if err != nil {
		log.Fatal(err)
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	// NOTE: if this is missing the the reply is in XML format
	// Might be more useful to use the XML format because it contains escape codes
	// For lithuanian alphabet special characters whereas json returns the literal characters
	// Might be easiest if i replace the xml espace codes with `.` and perform a regex search to match
	// a user in the parking bot users.json
	// req.Header.Set("Accept", "application/json")

	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n\nREQUEST:\n%s\n\n", string(reqDump))

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	respDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n\nRESPONSE:\n%s\n\n", string(respDump))

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(b)
}
