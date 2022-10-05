package parking

import (
	"log"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/event"
	slackApi "github.com/AngelVI13/slack-bot/pkg/slack"
	"github.com/AngelVI13/slack-bot/pkg/user"
	"github.com/slack-go/slack"
)

const (
	Identifier = "Parking: "
	// SlashCmd   = "/parking"
	SlashCmd = "/test-park"
)

type Manager struct {
	eventManager *event.EventManager
	parkingLot   *ParkingLot
	userManager  *user.Manager
}

func NewManager(
	eventManager *event.EventManager,
	config *config.Config,
	userManager *user.Manager,
) *Manager {
	parkingLot := getParkingLot(config)

	return &Manager{
		eventManager: eventManager,
		parkingLot:   &parkingLot,
		userManager:  userManager,
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
	}

}

func (m *Manager) Context() string {
	return Identifier
}

func (m *Manager) handleSlashCmd(data *slackApi.Slash) *common.Response {
	spaces := m.parkingLot.GetSpacesInfo(data.UserName)
	modal := generateBookingModalRequest(data, spaces)

	action := common.NewViewAction(event.OpenView, data.TriggerId, modal)
	response := common.NewResponseEvent(action)
	return response
}

func (m *Manager) handleBlockActions(data *slackApi.BlockAction) *common.Response {
	isSpecialUser := m.userManager.IsSpecial(data.UserName)

	var actions []event.ResponseAction

	for _, action := range data.Actions {
		switch action.ActionID {
		case ReserveParkingActionId:
			parkingSpace := action.Value
			actions = m.handleReserveParking(data, parkingSpace, isSpecialUser)
		case ReleaseParkingActionId:
			parkingSpace := action.Value
			actions = m.handleReleaseParking(data, parkingSpace, isSpecialUser)
		}
	}

	/*
		case modals.MParkingBookingTitle: // TODO: Why is this not the parking release title instead?
			updatedView = handleParkingBooking(bot, interaction)
	*/

	if actions != nil && len(actions) > 0 {
		return common.NewResponseEvent(actions...)
	}

	return nil
}

func (m *Manager) handleReserveParking(
	data *slackApi.BlockAction,
	parkingSpace string,
	isSpecialUser bool,
) []event.ResponseAction {
	// Check if an admin has made the request
	autoRelease := true // by default parking reservation is always with auto release
	if isSpecialUser {  // unless we have a special user (i.e. user with designated parking space)
		autoRelease = false
	}

	errStr := m.parkingLot.Reserve(parkingSpace, data.UserName, data.UserId, autoRelease)
	if errStr != "" {
		log.Println(errStr)
		// If there device was already taken -> inform user by personal DM message from the bot
		action := common.NewPostEphemeralAction(data.UserId, data.UserId, slack.MsgOptionText(errStr, false))
		return []event.ResponseAction{action}
	}
	return nil
}

func (m *Manager) handleReleaseParking(
	data *slackApi.BlockAction,
	parkingSpace string,
	isSpecialUser bool,
) []event.ResponseAction {
	// Handle general case: normal user releasing a space
	if !isSpecialUser {
		victimId, errStr := m.parkingLot.Release(parkingSpace, data.UserName)
		if victimId != "" {
			log.Println(errStr)
			action := common.NewPostEphemeralAction(victimId, victimId, slack.MsgOptionText(errStr, false))
			return []event.ResponseAction{action}
		}

		return nil
	}

	// Special User handling
	chosenParkingSpace := m.parkingLot.GetSpace(parkingSpace)
	// TODO: specialUser should only be allowed to release their own place ?
	err := m.parkingLot.ToBeReleased.Add(data.UserId, chosenParkingSpace)
	if err != nil {
		// TODO: this should just show an error in modal but not fail the program
		log.Fatal(err)
	}

	releaseModal := generateReleaseModalRequest(data, chosenParkingSpace, "")
	action := common.NewViewAction(event.PushView, data.TriggerId, releaseModal)
	return []event.ResponseAction{action}
}

/*
func handleParkingBooking(bot *SlackBot, interaction *slack.InteractionCallback) *slack.ModalViewRequest {
	var releaseInfo *parking.ReleaseInfo
	// handle button actions
	for _, action := range interaction.ActionCallback.BlockActions {
		switch action.ActionID {
		case modals.ReleaseStartDateActionId, modals.ReleaseEndDateActionId:
			date, err := time.Parse("2006-01-02", action.SelectedDate)
			if err != nil {
				// TODO: replace with proper handling
				log.Fatal(err)
			}
			releaseInfo = bot.Data.ParkingLot.ToBeReleased.GetByUserId(interaction.User.ID)
			if releaseInfo == nil {
				log.Fatalln(bot.Data.ParkingLot.ToBeReleased)
			}

			if action.ActionID == modals.ReleaseStartDateActionId {
				releaseInfo.StartDate = &date
			} else {
				releaseInfo.EndDate = &date
			}

		default:
		}
	}

	if releaseInfo != nil {
		parkingReleaseHandler := &modals.ParkingReleaseHandler{}
		updatedView := parkingReleaseHandler.GenerateModalRequest(
			bot.CurrentOptionModalData.Command,
			releaseInfo.Space,
			releaseInfo.Error().Error(),
		)
		_, err := bot.SlackClient.UpdateView(updatedView, "", "", interaction.View.ID)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
*/
