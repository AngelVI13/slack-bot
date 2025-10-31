package bss

import (
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
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
	HandleBss = "HandleBSS"
)

type Manager struct {
	eventManager         *event.EventManager
	data                 *model.Data
	BssUrl               string
	BssQuadUsername      string
	BssQuadPassword      string
	BssQuadEnvironmentId int
	BssQuadCompanyId     int
	debug                bool
	reportPersonId       string
	bssHashFilename      string
	vacationsHash        common.VacationsHash
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
	conf *config.Config,
) *Manager {
	return &Manager{
		eventManager:         eventManager,
		data:                 data,
		BssUrl:               conf.BssUrl,
		BssQuadUsername:      conf.BssQuadUsername,
		BssQuadPassword:      conf.BssQuadPassword,
		BssQuadEnvironmentId: conf.BssQuadEnvironmentId,
		BssQuadCompanyId:     conf.BssQuadCompanyId,
		debug:                conf.Debug,
		reportPersonId:       conf.ReportPersonId,
		bssHashFilename:      conf.VacationsHashFilename,
		vacationsHash:        common.LoadVacationsHash(conf.VacationsHashFilename),
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

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent("BSS", actions...)
}
