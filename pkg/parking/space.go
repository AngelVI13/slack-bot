package parking

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/common"
)

type ParkingKey string

type ParkingSpace struct {
	Number int
	Floor  int
	common.ReservedProps
}

func (p *ParkingSpace) GetPropsText() string {
	return fmt.Sprintf("(%d floor)", p.Floor)
}

func (p *ParkingSpace) GetStatusEmoji() string {
	emoji := ":large_green_circle:"
	if p.Reserved {
		emoji = ":large_orange_circle:"
	}
	return emoji
}

// GetStatusDescription Get space status description i.e. reserved, by who, when, etc.
// Returns empty string if space is free
func (p *ParkingSpace) GetStatusDescription() string {
	status := ""
	if p.Reserved {
		// timeStr := p.ReservedTime.Format("Mon 15:04")
		// status = fmt.Sprintf("_:bust_in_silhouette:*<@%s>*\ton\t:clock1: *%s*_", p.ReservedById, timeStr)
		status = fmt.Sprintf("<@%s>", p.ReservedById)
	}
	return status
}

func (p *ParkingSpace) ParkingKey() ParkingKey {
	return MakeParkingKey(p.Number, p.Floor)
}

func MakeParkingKey(number, floor int) ParkingKey {
	postfix := "th"

	postfixes := []string{"st", "nd", "rd"}
	absFloor := abs(floor)
	if 1 <= absFloor && absFloor <= 3 {
		postfix = postfixes[absFloor-1]
	}

	return ParkingKey(fmt.Sprintf("%d%s floor %d", floor, postfix, number))
}

func abs(n int) int {
	if n >= 0 {
		return n
	}
	return n * -1
}

type ParkingSpaces map[ParkingKey]*ParkingSpace
