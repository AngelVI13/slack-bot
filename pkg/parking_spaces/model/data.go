package model

import (
	"github.com/AngelVI13/slack-bot/pkg/model"
)

var (
	ShowOptions     = [2]string{"Free", "Taken"}
	ShowFreeOption  = ShowOptions[0]
	ShowTakenOption = ShowOptions[1]
)

const (
	ResetHour = 17
	ResetMin  = 0
)

type ParkingData struct {
	*model.Data
	SelectedFloor     map[string]string
	SelectedShowTaken map[string]bool
	DefaultFloor      string
}

func NewParkingData(data *model.Data) *ParkingData {
	allFloors := data.ParkingLot.GetAllFloors()
	defaultFloor := ""
	if len(allFloors) > 0 {
		defaultFloor = allFloors[0]
	}
	return &ParkingData{
		Data:              data,
		SelectedFloor:     map[string]string{},
		SelectedShowTaken: map[string]bool{},
		DefaultFloor:      defaultFloor,
	}
}

func (d *ParkingData) SetDefaultFloorIfMissing(userId string) {
	if _, ok := d.SelectedFloor[userId]; !ok {
		d.SelectedFloor[userId] = d.DefaultFloor
	}
}
