package spaces

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"
	"time"
)

type SpaceType int

const (
	SpaceFree SpaceType = iota
	SpaceTaken
	SpaceAny
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
	for space, pool := range d.ToBeReleased {
		releases := pool.All()
		for _, release := range releases {
			if !release.Submitted {
				pool.Remove(release.UniqueId)
			}
		}

		// Check if no finalized releases are associated with a space and
		// remove them from the ToBeReleased map.
		// NOTE: have to recompute pool.All() to get new value of releases
		if len(pool.All()) == 0 {
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

func (d *SpacesLot) HasPermanentSpace(userId string) *Space {
	for _, space := range d.UnitSpaces {
		if space.Reserved && space.ReservedById == userId && !space.AutoRelease {
			return space
		}
	}
	return nil
}

func (d *SpacesLot) OwnsSpace(userId string) *Space {
	var foundSpace *Space
	foundSpace = d.HasPermanentSpace(userId)
	if foundSpace != nil {
		return foundSpace
	}

	foundSpace = d.HasTempRelease(userId)
	return foundSpace
}

// GetOwnedSpaceByUserId Returns owned space by user even if currently that
// space is temporarily reserved by someone else.
func (d *SpacesLot) GetOwnedSpaceByUserId(userId string) (*Space, error) {
	for _, space := range d.UnitSpaces {
		if space.Reserved && space.ReservedById == userId {
			return space, nil
		}
	}

	for _, pool := range d.ToBeReleased {
		releases := pool.All()
		if len(releases) == 0 {
			continue
		}

		release := releases[0]
		if release.Submitted && release.OwnerId == userId {
			// NOTE: here release contains an outdated copy of a space
			// therefore have to take an up to date version from UnitSpaces
			s, found := d.UnitSpaces[release.Space.Key()]
			if !found {
				return nil, fmt.Errorf(
					"failed to get original space from release: %v space: %q",
					release,
					release.Space.Key(),
				)
			}
			return s, nil
		}
	}

	return nil, fmt.Errorf("no space for user: <@%s>", userId)
}

func (d *SpacesLot) HasTempRelease(userId string) *Space {
	for _, pool := range d.ToBeReleased {
		release := pool.Active()
		if release != nil && release.OwnerId == userId {
			return release.Space
		}
	}

	return nil
}

func (d *SpacesLot) GetSpacesByFloor(
	userId, floor string,
	spaceType SpaceType,
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

		addCondition := false
		switch spaceType {
		case SpaceFree:
			addCondition = !space.Reserved
		case SpaceTaken:
			addCondition = space.Reserved
		case SpaceAny:
			addCondition = true
		default:
			log.Fatalf("unsupported space type: %v", spaceType)
		}
		if addCondition {
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

// NOTE: the implementation looks stupid but can't think of better way
func (d *SpacesLot) GetAllFloors() []string {
	floorMap := map[int]int{}
	var floorsNum []int
	var floors []string

	for _, space := range d.UnitSpaces {
		floorMap[space.Floor] = 1
	}

	for floor := range floorMap {
		floorsNum = append(floorsNum, floor)
	}
	slices.Sort(floorsNum)

	for _, floor := range floorsNum {
		floors = append(floors, MakeFloorStr(floor))
	}

	return floors
}

// GetExistingFloors Find all existing floors that are present in the allowed
// floors slice. For example if SpacesLot contains spaces on floors: 4, 5, 7
// and the list of allowed floors is {4, 6} then this returns only 4th floor as
// a formatted string.
func (d *SpacesLot) GetExistingFloors(allowedFloors []int) []string {
	floorMap := map[int]int{}
	var floorsNum []int
	var floors []string

	for _, space := range d.UnitSpaces {
		if !slices.Contains(allowedFloors, space.Floor) {
			continue
		}
		floorMap[space.Floor] = 1
	}

	for floor := range floorMap {
		floorsNum = append(floorsNum, floor)
	}
	slices.Sort(floorsNum)

	for _, floor := range floorsNum {
		floors = append(floors, MakeFloorStr(floor))
	}

	return floors
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

func (l *SpacesLot) ReleaseSpaces(cTime time.Time) error {
	var errs []error

	for spaceKey, space := range l.UnitSpaces {
		if space == nil {
			err := fmt.Errorf("[SKIP] ReleaseSpaces space is nil. spaceKey=%q", spaceKey)
			errs = append(errs, err)
			continue
		}
		// Simple case
		if space.Reserved && space.AutoRelease {
			slog.Info("AutoRelease", "space", spaceKey)
			space.Reserved = false
			space.AutoRelease = false
			// Fall-through to check if this is also a temporary
			// released space has to be reserved
		}

		// If a scheduled release was setup
		for _, release := range l.ToBeReleased.GetAll(spaceKey) {
			if release == nil {
				err := fmt.Errorf(
					"[SKIP] ReleaseSpaces release is nil. spaceKey=%q",
					spaceKey,
				)
				errs = append(errs, err)
				continue
			}

			if !release.DataPresent() {
				err := fmt.Errorf(
					"[SKIP] ReleaseSpaces release is not fully filled. spaceKey=%q; release=%v",
					spaceKey,
					release,
				)
				errs = append(errs, err)
				continue
			}
			l.ReleaseTemp(space, cTime, release)
		}
	}

	l.SynchronizeToFile()
	return errors.Join(errs...)
}

func (l *SpacesLot) ReleaseTemp(
	space *Space,
	cTime time.Time,
	releaseInfo *ReleaseInfo,
) {
	spaceKey := releaseInfo.Space.Key()
	// On the day before the start of the release -> make the space
	// available for selection
	if releaseInfo.StartDate.Sub(cTime).Hours() < 24 &&
		releaseInfo.StartDate.After(cTime) {
		slog.Info("TempRelease", "space", spaceKey, "releaseInfo", releaseInfo)
		space.Reserved = false
		space.AutoRelease = false
		releaseInfo.MarkActive()
	} else if releaseInfo.EndDate.Sub(cTime).Hours() < 24 && releaseInfo.EndDate.Before(cTime) {
		// On the day of the end of release -> reserve back the space
		// for the correct user
		slog.Info("TempReserve (return to owner)", "space", spaceKey, "releaseInfo", releaseInfo)
		space.Reserved = true
		space.AutoRelease = false
		space.ReservedBy = releaseInfo.OwnerName
		space.ReservedById = releaseInfo.OwnerId

		err := l.ToBeReleased.Remove(releaseInfo)
		if err != nil {
			slog.Error("Failed removing release info", "space", spaceKey, "err", err)
		}
	}
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
