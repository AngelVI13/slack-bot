package hcm

import (
	"io"
	"log"
	"net/http"
	"time"

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

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("HCM", actions...)
}

func (m *Manager) makeHcmRequest(endpoint string, token string) ([]byte, error) {
	fullUrl := m.hcmUrl + endpoint

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, fullUrl, nil)
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
