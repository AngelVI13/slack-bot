package views

import (
	"encoding/json"
	"log"

	"github.com/AngelVI13/slack-bot/pkg/spaces"
)

type ActionValues struct {
	SpaceKey  spaces.SpaceKey `json:"space,omitempty"`
	ModalType ModalType       `json:"modalType,omitempty"`
	ReleaseId int             `json:"releaseId,omitempty"`
}

func (av ActionValues) Encode() string {
	b, err := json.Marshal(av)
	if err != nil {
		log.Fatalf("failed to marshal action values: %v; err: %v", av, err)
	}

	return string(b)
}

func (av ActionValues) Decode(value string) ActionValues {
	err := json.Unmarshal([]byte(value), &av)
	if err != nil {
		log.Fatalf("failed to unmarshal action value: %v; err: %v", value, err)
	}
	return av
}
