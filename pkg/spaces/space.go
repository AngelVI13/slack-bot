package spaces

import (
	"fmt"

	"github.com/AngelVI13/slack-bot/pkg/common"
)

type SpaceKey string

type Space struct {
	Number int
	Floor  int
	common.ReservedProps
}

func (p *Space) GetPropsText() string {
	return fmt.Sprintf("(%d floor)", p.Floor)
}

func (p *Space) GetStatusEmoji() string {
	emoji := ":large_green_circle:"
	if p.Reserved {
		emoji = ":large_orange_circle:"
	}
	return emoji
}

// GetStatusDescription Get space status description i.e. reserved, by who, when, etc.
// Returns empty string if space is free
func (p *Space) GetStatusDescription() string {
	status := ""
	if p.Reserved {
		// timeStr := p.ReservedTime.Format("Mon 15:04")
		// status = fmt.Sprintf("_:bust_in_silhouette:*<@%s>*\ton\t:clock1: *%s*_", p.ReservedById, timeStr)
		status = fmt.Sprintf("<@%s>", p.ReservedById)
	}
	return status
}

func (p *Space) Key() SpaceKey {
	return MakeSpaceKey(p.Number, p.Floor)
}

func (p *Space) Smaller(other *Space) bool {
	if p.Floor < other.Floor {
		return true
	} else if p.Floor > other.Floor {
		return false
	} else { // p.Floor == other.Floor
		return p.Number < other.Number
	}
}

func MakeSpaceKey(number, floor int) SpaceKey {
	postfix := "th"

	postfixes := []string{"st", "nd", "rd"}
	absFloor := abs(floor)
	if 1 <= absFloor && absFloor <= 3 {
		postfix = postfixes[absFloor-1]
	}

	return SpaceKey(fmt.Sprintf("%d%s floor %d", floor, postfix, number))
}

func abs(n int) int {
	if n >= 0 {
		return n
	}
	return n * -1
}

type UnitSpaces map[SpaceKey]*Space
