package spaces

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"
)

type SpacesInfo []*Space

type SpacesLot struct {
	UnitSpaces
	Filename     string
	ToBeReleased ReleaseMap
}

func NewSpacesLot() SpacesLot {
	return SpacesLot{
		UnitSpaces:   make(UnitSpaces),
		ToBeReleased: make(ReleaseMap),
	}
}

// NewSpacesLotFromJson Takes json data as input and returns a populated ParkingLot object
func NewSpacesLotFromJson(data []byte, filename string) SpacesLot {
	spacesLot := NewSpacesLot()
	spacesLot.synchronizeFromFile(data)
	spacesLot.Filename = filename
	return spacesLot
}

func (d *SpacesLot) SynchronizeToFile() {
	data, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(d.Filename, data, 0o666)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Wrote spaces lot to file", "file", d.Filename)
}

func (d *SpacesLot) synchronizeFromFile(data []byte) {
	// Unmarshal the provided data into the solid map
	err := json.Unmarshal(data, d)
	if err != nil {
		log.Fatalf("Could not parse spaces file. Error: %+v", err)
	}

	// Do not load any submitted items from to be released map
	for space, info := range d.ToBeReleased {
		if !info.Submitted {
			delete(d.ToBeReleased, space)
		}
	}
}

func (d *SpacesLot) HasSpace(userId string) bool {
	userAlreadyReservedSpace := false
	for _, space := range d.UnitSpaces {
		if space.Reserved && space.ReservedById == userId {
			userAlreadyReservedSpace = true
			break
		}
	}
	return userAlreadyReservedSpace
}

func (d *SpacesLot) OwnsSpace(userId string) bool {
	if d.HasSpace(userId) {
		return true
	}

	if d.HasTempRelease(userId) {
		return true
	}

	return false
}

// GetOwnedSpaceByUserId Returns owned space by user even if currently that
// space is temporarily reserved by someone else.
func (d *SpacesLot) GetOwnedSpaceByUserId(userId string) *Space {
	for _, space := range d.UnitSpaces {
		if space.Reserved && space.ReservedById == userId {
			return space
		}
	}

	for _, releaseInfo := range d.ToBeReleased {
		if releaseInfo.Submitted && releaseInfo.OwnerId == userId {
			return releaseInfo.Space
		}
	}

	return nil
}

func (d *SpacesLot) HasTempRelease(userId string) bool {
	userAlreadyReleasedSpace := false
	for _, releaseInfo := range d.ToBeReleased {
		if releaseInfo.Submitted && releaseInfo.OwnerId == userId {
			userAlreadyReleasedSpace = true
			break
		}
	}

	return userAlreadyReleasedSpace
}

func (d *SpacesLot) GetSpacesByFloor(
	userId, floor string,
	onlyTaken bool,
) SpacesInfo {
	floorSpaces := make(SpacesInfo, 0)
	allSpaces := d.GetSpacesInfo(userId)

	if floor == "" || len(allSpaces) == 0 {
		// NOTE: This should never happen
		return SpacesInfo{}
	}

	for _, space := range allSpaces {
		if !strings.HasPrefix(string(space.Key()), floor) {
			continue
		}

		if space.Reserved && space.ReservedById == userId {
			floorSpaces = append(floorSpaces, space)
			// its already added so skip it
			continue
		}

		if onlyTaken && space.Reserved {
			floorSpaces = append(floorSpaces, space)
		} else if !onlyTaken && !space.Reserved {
			floorSpaces = append(floorSpaces, space)
		}
	}
	return floorSpaces
}

func (d *SpacesLot) GetSpacesInfo(userId string) SpacesInfo {
	// Group spaces in 2 groups -> belonging to given user or not
	// The group that doesn't belong to user will be sorted by name and by status (reserved or not)
	userSpaces := make(SpacesInfo, 0)
	nonUserSpaces := make(SpacesInfo, 0)
	for _, d := range d.UnitSpaces {
		if d.Reserved && d.ReservedById == userId {
			userSpaces = append(userSpaces, d)
		} else {
			nonUserSpaces = append(nonUserSpaces, d)
		}
	}

	// NOTE: This sorts the spaces list starting from free spaces
	sort.Slice(nonUserSpaces, func(i, j int) bool {
		return !nonUserSpaces[i].Reserved
	})

	firstTaken := -1 // Index of first taken space
	for i, space := range nonUserSpaces {
		if space.Reserved {
			firstTaken = i
			break
		}
	}

	// NOTE: this might be unnecessary but it shows spaces in predicable way in UI so its nice.
	// If all spaces are free or all spaces are taken, sort by number
	if firstTaken == -1 || firstTaken == 0 {
		sort.Slice(nonUserSpaces, func(i, j int) bool {
			return nonUserSpaces[i].Smaller(nonUserSpaces[j])
		})
	} else {
		// split spaces into 2 - free & taken
		// sort each sub slice based on space number
		free := nonUserSpaces[:firstTaken]
		taken := nonUserSpaces[firstTaken:]

		sort.Slice(free, func(i, j int) bool {
			return free[i].Smaller(free[j])
		})

		sort.Slice(taken, func(i, j int) bool {
			return taken[i].Smaller(taken[j])
		})
	}

	allSpaces := make(SpacesInfo, 0, len(d.UnitSpaces))
	allSpaces = append(allSpaces, userSpaces...)
	allSpaces = append(allSpaces, nonUserSpaces...)
	return allSpaces
}

func (l *SpacesLot) Reserve(
	unitSpace SpaceKey,
	user, userId string,
	autoRelease bool,
) (errMsg string) {
	space := l.GetSpace(unitSpace)
	if space == nil {
		return fmt.Sprintf(
			"Failed to reserve space: couldn't find the space %s",
			unitSpace,
		)
	}

	// Only inform user if it was someone else that tried to reserved his space.
	// This prevents an unnecessary message if you double clicked the reserve button yourself
	if space.Reserved && space.ReservedById != userId {
		reservedTime := space.ReservedTime.Format("Mon 15:04")
		return fmt.Sprintf(
			"*Error*: Could not reserve *%s*. *%s* has just reserved it (at *%s*)",
			unitSpace,
			space.ReservedBy,
			reservedTime,
		)
	}
	slog.Info(
		"SPACE_RESERVE",
		"user",
		user,
		"space",
		unitSpace,
		"autoRelease",
		autoRelease,
	)

	space.Reserved = true
	space.ReservedBy = user
	space.ReservedById = userId
	space.ReservedTime = time.Now()
	space.AutoRelease = autoRelease

	l.SynchronizeToFile()
	return ""
}

func (l *SpacesLot) Release(
	unitSpace SpaceKey,
	userName, userId string,
) (victimId, errMsg string) {
	space := l.GetSpace(unitSpace)
	if space == nil {
		return userId, fmt.Sprintf(
			"Failed to release space: couldn't find the space %s",
			unitSpace,
		)
	}

	slog.Info("SPACE_RELEASE", "user", userName, "space", unitSpace)

	space.Reserved = false
	l.SynchronizeToFile()

	if space.ReservedById != userId {
		return space.ReservedById,
			fmt.Sprintf(
				":warning: *%s* released your (*%s*) space (*%s*)",
				userName,
				space.ReservedBy,
				unitSpace,
			)
	}
	return "", ""
}

func (l *SpacesLot) GetSpace(unitSpace SpaceKey) *Space {
	space, ok := l.UnitSpaces[unitSpace]
	if !ok {
		slog.Error("Incorrect space number", "space", unitSpace)
		return nil
	}
	return space
}

// TODO: Test this
func (l *SpacesLot) ReleaseSpaces(cTime time.Time) {
	for spaceKey, space := range l.UnitSpaces {
		// Simple case
		if space.Reserved && space.AutoRelease {
			slog.Info("AutoRelease", "space", spaceKey)
			space.Reserved = false
			space.AutoRelease = false
			// Fall-through to check if this is also a temporary
			// released space has to be reserved
		}

		// If a scheduled release was setup
		releaseInfo := l.ToBeReleased.Get(spaceKey)
		if releaseInfo == nil {
			continue
		}

		// On the day before the start of the release -> make the space
		// available for selection
		if releaseInfo.StartDate.Sub(cTime).Hours() < 24 &&
			releaseInfo.StartDate.After(cTime) {
			slog.Info("TempRelease", "space", spaceKey, "releaseInfo", releaseInfo)
			space.Reserved = false
			space.AutoRelease = false
		} else if releaseInfo.EndDate.Sub(cTime).Hours() < 24 && releaseInfo.EndDate.Before(cTime) {
			// On the day of the end of release -> reserve back the space
			// for the correct user
			slog.Info("TempReserve (return to owner)", "space", spaceKey, "releaseInfo", releaseInfo)
			space.Reserved = true
			space.AutoRelease = false
			space.ReservedBy = releaseInfo.OwnerName
			space.ReservedById = releaseInfo.OwnerId

			ok := l.ToBeReleased.Remove(spaceKey)
			if !ok {
				slog.Error("Failed removing release info", "space", spaceKey)
			}
		}
	}

	l.SynchronizeToFile()
}

func GetSpacesLot(filename string) (spacesLot SpacesLot) {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not read spaces file (%s)", filename)
	}

	spacesLot = NewSpacesLotFromJson(fileData, filename)

	loadedSpacesNum := len(spacesLot.UnitSpaces)
	if loadedSpacesNum == 0 {
		log.Fatalf("No spaces found in (%s).", filename)
	}

	slog.Info(
		"INIT: Spaces list loaded successfully",
		"file", filename, "spaces", loadedSpacesNum,
	)
	return spacesLot
}
