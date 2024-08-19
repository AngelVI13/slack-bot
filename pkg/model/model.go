package model

import (
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/AngelVI13/slack-bot/pkg/user"
)

type Data struct {
	ParkingLot    *spaces.SpacesLot
	WorkspacesLot *spaces.SpacesLot
	UserManager   *user.Manager
}

func NewData(config *config.Config) *Data {
	userManager := user.NewManager(config)
	parkingLot := spaces.GetSpacesLot(config.ParkingFilename)
	worspacesLot := spaces.GetSpacesLot(config.WorkspacesFilename)
	return &Data{
		UserManager:   userManager,
		ParkingLot:    &parkingLot,
		WorkspacesLot: &worspacesLot,
	}
}
