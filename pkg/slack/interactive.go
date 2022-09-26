package slack

import (
	"log"

	"github.com/slack-go/slack"
)

// TODO: maybe move these to separate files
type ViewSubmission struct {
}

func (v *ViewSubmission) Type() EventType {
	return ViewSubmissionEvent
}

func (v *ViewSubmission) Data() any {
	return nil
}

type BlockAction struct {
}

func (v *BlockAction) Type() EventType {
	return BlockActionEvent
}

func (v *BlockAction) Data() any {
	return nil
}

func handleInteractionEvent(interaction slack.InteractionCallback) Event {
	var event Event

	switch interaction.Type {
	case slack.InteractionTypeViewSubmission:
		event = handleViewSubmission(&interaction)
	case slack.InteractionTypeBlockActions:
		event = handleBlockActions(&interaction)
	default:
		log.Printf("Unsupported interaction event: %v, %v", interaction.Type, interaction)
	}

	return event
}

func handleViewSubmission(interaction *slack.InteractionCallback) Event {
	// TODO: extract the data that is needed for later processing
	/*
		// NOTE: we use title text to determine which modal was submitted
		switch interaction.View.Title.Text {
		case modals.MRestartProxyTitle:
			restartProxySubmission(SlackClient, interaction, Data.Devices)
		case modals.MRemoveUsersTitle:
			removeUserSubmission(interaction, Data.Users)
		case modals.MAddUserTitle:
			addUserSubmission(SlackClient, interaction, Data.Users, Data.Reviewers)
		default:
		}
	*/
	return &ViewSubmission{}
}

func handleBlockActions(interaction *slack.InteractionCallback) Event {
	// TODO: extract the data that is needed for later processing
	/*
			if CurrentOptionModalData.Handler == nil {
				log.Fatalf(
					`Did not have a valid pointer to OptionModal,
		                    please make sure to close any open modals before restarting the bot`,
				)
			}

			var updatedView *slack.ModalViewRequest

			switch interaction.View.Title.Text {
			case modals.MDeviceTitle:
				updatedView = handleDeviceActions(bot, interaction)
			case modals.MShowUsersTitle, modals.MRemoveUsersTitle, modals.MAddUserTitle:
				updatedView = handleUserActions(bot, interaction)
			case modals.MParkingTitle:
				updatedView = handleParkingActions(bot, interaction)
			case modals.MParkingBookingTitle: // TODO: Why is this not the parking release title instead?
				updatedView = handleParkingBooking(bot, interaction)
			default:
			}

			// Update view if a handler generated an update
			if updatedView != nil {
				_, err := SlackClient.UpdateView(*updatedView, "", "", interaction.View.ID)
				if err != nil {
					log.Fatal(err)
				}
			}
	*/

	return &BlockAction{}
}
