package model

import (
	"github.com/AngelVI13/slack-bot/pkg/config"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
	"github.com/AngelVI13/slack-bot/pkg/model/user"
)

type Data struct {
	ParkingLot    *spaces.SpacesLot
	WorkspacesLot *spaces.SpacesLot
	UserManager   *user.Manager
}

func NewData(config *config.Config) *Data {
	userManager := user.NewManager(config.UsersFilename)
	parkingLot := spaces.GetSpacesLot(config.ParkingFilename)
	worspacesLot := spaces.GetSpacesLot(config.WorkspacesFilename)
	return &Data{
		UserManager:   userManager,
		ParkingLot:    &parkingLot,
		WorkspacesLot: &worspacesLot,
	}
}
