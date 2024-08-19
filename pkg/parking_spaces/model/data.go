package model

import (
	"github.com/AngelVI13/slack-bot/pkg/model"
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

type ParkingData struct {
	*model.Data
	SelectedFloor     map[string]string
	SelectedShowTaken map[string]bool
}

func NewParkingData(data *model.Data) *ParkingData {
	return &ParkingData{
		Data:              data,
		SelectedFloor:     map[string]string{},
		SelectedShowTaken: map[string]bool{},
	}
}

func (d *ParkingData) SetDefaultFloorIfMissing(userId, floor string) {
	if _, ok := d.SelectedFloor[userId]; !ok {
		d.SelectedFloor[userId] = floor
	}
}
