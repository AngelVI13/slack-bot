package edit_parking_spaces

import (
	"fmt"
	"log"
	"log/slog"
	"slices"
	"strconv"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/event"
	"github.com/AngelVI13/slack-bot/pkg/model"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Edit Parking Spaces: "
	SlashCmd   = "/spaces-parking"
	// SlashCmd = "/test-spaces"

	defaultUserOption = ""
)

type SelectedEditOptionMap map[string]editOption

// TODO: do any of the below methods copy or mutate the map????
func (m SelectedEditOptionMap) Get(userId string) editOption {
	val, found := m[userId]
	if !found {
		return notSelectedOption
	}
	return val
}

func (m SelectedEditOptionMap) Set(userId string, option editOption) {
	if !slices.Contains(editOptions, option) {
		log.Fatalf("Unsupported park space edit option: %q", option)
	}
	m[userId] = option
}

func (m SelectedEditOptionMap) ResetSelectionForUser(userId string) {
	m[userId] = notSelectedOption
}

type Manager struct {
	eventManager       *event.EventManager
	data               *model.Data
	slackClient        *slack.Client
	selectedEditOption SelectedEditOptionMap
}

func NewManager(
	eventManager *event.EventManager,
	data *model.Data,
) *Manager {
	return &Manager{
		eventManager:       eventManager,
		data:               data,
		selectedEditOption: SelectedEditOptionMap{},
	}
}

func (m *Manager) Consume(e event.Event) {
	switch e.Type() {
	case event.SlashCmdEvent:
		data := e.(*slackApi.Slash)
		if data.Command != SlashCmd {
			return
		}

		response := m.handleSlashCmd(data)

		m.eventManager.Publish(response)
	case event.BlockActionEvent:
		data := e.(*slackApi.BlockAction)

		response := m.handleBlockActions(data)
		if response == nil {
			return
		}

		m.eventManager.Publish(response)

	case event.ViewSubmissionEvent:
		data := e.(*slackApi.ViewSubmission)

		if data.Title != parkSpaceManagementTitle {
			return
		}

		response := m.handleViewSubmission(data)
		if response == nil {
			return
		}
		m.eventManager.Publish(response)
	}
}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	if !m.data.UserManager.IsAdminId(data.UserId) {
		errTxt := fmt.Sprintf(
			"You don't have permission to execute '%s' command",
			SlashCmd,
		)
		action := common.NewPostEphemeralAction(data.UserId, data.UserId, errTxt, false)
		return common.NewResponseEvent(data.UserName, action)
	}

	modal := m.generateEditSpacesModalRequest(data, data.UserId)

	action := common.NewOpenViewAction(data.TriggerId, modal)
	response := common.NewResponseEvent(data.UserName, action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	var actions []event.ResponseAction

	for _, action := range data.Actions {
		switch action.ActionID {
		case selectEditOptionId:
			selectedEditAction := data.IValueSingle("", selectEditOptionId)
			m.selectedEditOption.Set(data.UserId, editOption(selectedEditAction))
			modal := m.generateEditSpacesModalRequest(data, data.UserId)

			action := common.NewUpdateViewAction(data.TriggerId, data.ViewId, modal, "")
			actions = append(actions, action)
		}
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
}

func (m *Manager) errorMessageAction(
	data *slackApi.BaseEvent,
	errTxt string,
) *common.PostEphemeralAction {
	slog.Error(errTxt, "requestor", data.UserName)
	action := common.NewPostEphemeralAction(
		data.UserId,
		data.UserId,
		errTxt,
		false,
	)
	return action
}

func (m *Manager) handleRemoveSpaceSubmission(
	data *slackApi.ViewSubmission,
) []event.ResponseAction {
	var actions []event.ResponseAction
	selectedSpaces := data.IValue("", selectSpaceOptionId)
	if len(selectedSpaces) == 0 {
		errTxt := "No spaces selected for removal -> nothing was done"
		actions = append(actions, m.errorMessageAction(&data.BaseEvent, errTxt))
	}

	slog.Info(
		"Removing spaces from DB",
		"requestor",
		data.UserName,
		"spaces",
		selectedSpaces,
	)
	for _, space := range selectedSpaces {
		spaceKey := spaces.SpaceKey(space)
		m.data.ParkingLot.ToBeReleased.RemoveAllReleases(spaceKey)
		delete(m.data.ParkingLot.UnitSpaces, spaceKey)
	}

	m.data.ParkingLot.SynchronizeToFile()

	// TODO: Should I inform the requestor that the action was completed successfully ?
	return actions
}

func (m *Manager) handleAddSpaceSubmission(
	data *slackApi.ViewSubmission,
) []event.ResponseAction {
	var actions []event.ResponseAction
	floorStr := data.IValueSingle("", addFloorActionId)
	spaceNumberStr := data.IValueSingle("", addSpaceActionId)

	// NOTE: slack does a lot of validation for correct inputs
	// so this in theory shouldn't fail
	floor, err := strconv.Atoi(floorStr)
	if err != nil {
		errTxt := fmt.Sprintf(
			"Space was not added - error while trying to convert floor %q to int: %v",
			floorStr,
			err,
		)
		actions = append(actions, m.errorMessageAction(&data.BaseEvent, errTxt))
		return actions
	}

	if floor == 0 {
		errTxt := "Space was not added - invalid floor value (0). allowed values are: -2, -1, 1"
		actions = append(actions, m.errorMessageAction(&data.BaseEvent, errTxt))
		return actions
	}

	spaceNumber, err := strconv.Atoi(spaceNumberStr)
	if err != nil {
		errTxt := fmt.Sprintf(
			"Space was not added - error while trying to convert spaceNumber %q to int: %v",
			spaceNumberStr,
			err,
		)
		actions = append(actions, m.errorMessageAction(&data.BaseEvent, errTxt))
		return actions
	}

	space := spaces.NewSpace(spaceNumber, floor, "")
	spaceKey := space.Key()

	_, found := m.data.ParkingLot.UnitSpaces[spaceKey]
	if found {
		errTxt := fmt.Sprintf(
			"Can't add parking space %q because it already exists",
			spaceKey,
		)
		actions = append(actions, m.errorMessageAction(&data.BaseEvent, errTxt))
		return actions
	}

	m.data.ParkingLot.UnitSpaces[spaceKey] = space
	m.data.ParkingLot.SynchronizeToFile()

	return actions
}

func (m *Manager) handleViewSubmission(data *slackApi.ViewSubmission) *common.Response {
	var actions []event.ResponseAction

	selectedAction := m.selectedEditOption.Get(data.UserId)
	// Reset selected action
	m.selectedEditOption.ResetSelectionForUser(data.UserId)

	switch selectedAction {
	case removeSpaceOption:
		actions = append(actions, m.handleRemoveSpaceSubmission(data)...)
	case addSpaceOption:
		actions = append(actions, m.handleAddSpaceSubmission(data)...)
	case notSelectedOption:
		return nil // do nothing
	default:
		log.Fatalf("unsupported action: %v", selectedAction)
	}

	if len(actions) == 0 {
		return nil
	}

	return common.NewResponseEvent(data.UserName, actions...)
}
