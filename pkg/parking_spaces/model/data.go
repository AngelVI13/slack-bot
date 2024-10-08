package model

import (
	"github.com/AngelVI13/slack-bot/pkg/spaces"
	"github.com/AngelVI13/slack-bot/pkg/user"
)

var (
	Floors             = [3]string{"-2nd floor", "-1st floor", "1st floor"}
	DefaultFloorOption = Floors[0]
	ShowOptions        = [2]string{"Free", "Taken"}
	ShowFreeOption     = ShowOptions[0]
	ShowTakenOption    = ShowOptions[1]
)

const (
	ResetHour = 17
	ResetMin  = 0
)

type Data struct {
	ParkingLot        *spaces.SpacesLot
	UserManager       *user.Manager
	SelectedFloor     map[string]string
	SelectedShowTaken map[string]bool
}

func NewData(filename string, userManager *user.Manager) *Data {
	parkingLot := spaces.GetSpacesLot(filename)
	return &Data{
		UserManager:       userManager,
		ParkingLot:        &parkingLot,
		SelectedFloor:     map[string]string{},
		SelectedShowTaken: map[string]bool{},
	}
}

func (d *Data) SetDefaultFloorIfMissing(userId, floor string) {
	if _, ok := d.SelectedFloor[userId]; !ok {
		d.SelectedFloor[userId] = floor
	}
}
