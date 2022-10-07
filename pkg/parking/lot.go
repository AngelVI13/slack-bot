package parking

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/config"
)

type SpacesInfo []*ParkingSpace

type ReleaseInfo struct {
	ReleaserId string
	OwnerId    string
	OwnerName  string
	Space      *ParkingSpace
	StartDate  *time.Time
	EndDate    *time.Time
	Submitted  bool

	// These are only used while the user is choosing date range to refer
	// between space selected and release range selected (i.e. between booking modal
	// and corresponding release modal)
	RootViewId string
	ViewId     string
}

func (i *ReleaseInfo) MarkSubmitted() {
	i.Submitted = true

	// TODO: Synchronize to file here
}

func (i *ReleaseInfo) DataPresent() bool {
	return (i.ReleaserId != "" &&
		i.OwnerId != "" &&
		i.OwnerName != "" &&
		i.Space != nil &&
		i.StartDate != nil &&
		i.EndDate != nil)
}

func (i *ReleaseInfo) Check() string {
	if !i.DataPresent() {
		return fmt.Sprintf("Missing date information for temporary release of space (%d)", i.Space.Number)
	}

	return common.CheckDateRange(*i.StartDate, *i.EndDate)
}

func (i ReleaseInfo) String() string {
	return fmt.Sprintf("ReleaseInfo(space=%d, userName=%s, start=%v, end=%v)", i.Space.Number, i.OwnerName, i.StartDate, i.EndDate)
}

type ReleaseQueue struct {
	queue []*ReleaseInfo
}

func (q *ReleaseQueue) Get(space int) *ReleaseInfo {
	for _, item := range q.queue {
		if item.Space.Number == space {
			return item
		}
	}
	return nil
}

func (q *ReleaseQueue) GetByReleaserId(userId string) *ReleaseInfo {
	for _, item := range q.queue {
		if item.ReleaserId == userId {
			return item
		}
	}
	return nil
}

func (q *ReleaseQueue) GetByRootViewId(rootId string) *ReleaseInfo {
	for _, item := range q.queue {
		if item.RootViewId == rootId {
			return item
		}
	}
	return nil
}

func (q *ReleaseQueue) GetByViewId(viewId string) *ReleaseInfo {
	for _, item := range q.queue {
		if item.ViewId == viewId {
			return item
		}
	}
	return nil
}

func (q *ReleaseQueue) RemoveByViewId(viewId string) (int, bool) {
	spaceNum := -1
	removeIdx := -1
	for idx, item := range q.queue {
		if item.ViewId == viewId {
			removeIdx = idx
			spaceNum = item.Space.Number
			break
		}
	}
	if removeIdx == -1 {
		return spaceNum, false
	}

	q.queue[len(q.queue)-1] = q.queue[removeIdx]
	q.queue = q.queue[:len(q.queue)-1]
	return spaceNum, true
}

func (q *ReleaseQueue) Add(
	viewId,
	releaserId,
	ownerName,
	ownerId string,
	space *ParkingSpace,
) (*ReleaseInfo, error) {
	if q.Get(space.Number) != nil {
		return nil, fmt.Errorf("Space already marked for release by someone else.")
	}

	releaseInfo := &ReleaseInfo{
		RootViewId: viewId,
		ReleaserId: releaserId,
		OwnerName:  ownerName,
		OwnerId:    ownerId,
		Space:      space,
		Submitted:  false,
	}

	q.queue = append(q.queue, releaseInfo)
	return releaseInfo, nil
}

type ParkingLot struct {
	ParkingSpaces
	config *config.Config
	// TODO: sync this info to file in case bot restarts
	ToBeReleased ReleaseQueue
}

func NewParkingLot() ParkingLot {
	return ParkingLot{
		ParkingSpaces: make(map[int]*ParkingSpace),
		ToBeReleased:  ReleaseQueue{},
	}
}

// NewParkingLotFromJson Takes json data as input and returns a populated ParkingLot object
func NewParkingLotFromJson(data []byte, config *config.Config) ParkingLot {
	parkingLot := NewParkingLot()
	parkingLot.synchronizeFromFile(data)
	parkingLot.config = config
	return parkingLot
}

func (d *ParkingLot) SynchronizeToFile() {
	data, err := json.MarshalIndent(d, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(d.config.ParkingFilename, data, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("INFO: Wrote parking lot to file")
}

func (d *ParkingLot) synchronizeFromFile(data []byte) {
	// Unmarshal the provided data into the solid map
	err := json.Unmarshal(data, d)
	if err != nil {
		log.Fatalf("Could not parse parking file. Error: %+v", err)
	}
}

// TODO: This is identical to GetDevicesInfo -> refactor it out
func (d *ParkingLot) GetSpacesInfo(user string) SpacesInfo {
	// Group spaces in 2 groups -> belonging to given user or not
	// The group that doesn't belong to user will be sorted by name and by status (reserved or not)
	userSpaces := make(SpacesInfo, 0)
	nonUserSpaces := make(SpacesInfo, 0)
	for _, d := range d.ParkingSpaces {
		if d.Reserved && d.ReservedBy == user {
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
			return nonUserSpaces[i].Number < nonUserSpaces[j].Number
		})
	} else {
		// split spaces into 2 - free & taken
		// sort each sub slice based on space number
		free := nonUserSpaces[:firstTaken]
		taken := nonUserSpaces[firstTaken:]

		sort.Slice(free, func(i, j int) bool {
			return free[i].Number < free[j].Number
		})

		sort.Slice(taken, func(i, j int) bool {
			return taken[i].Number < taken[j].Number
		})
	}

	allSpaces := make(SpacesInfo, 0, len(d.ParkingSpaces))
	allSpaces = append(allSpaces, userSpaces...)
	allSpaces = append(allSpaces, nonUserSpaces...)
	return allSpaces
}

func (l *ParkingLot) Reserve(parkingSpace, user, userId string, autoRelease bool) (errMsg string) {
	spaceNumber, err := strconv.Atoi(parkingSpace)
	if err != nil {
		log.Fatalf("Could not convert parkingSpace %+v to int", parkingSpace)
	}

	space, ok := l.ParkingSpaces[spaceNumber]
	if !ok {
		log.Fatalf("Wrong parking space number %d, %+v", spaceNumber, l)
	}
	// Only inform user if it was someone else that tried to reserved his space.
	// This prevents an unnecessary message if you double clicked the reserve button yourself
	if space.Reserved && space.ReservedById != userId {
		reservedTime := space.ReservedTime.Format("Mon 15:04")
		return fmt.Sprintf("*Error*: Could not reserve *%d*. *%s* has just reserved it (at *%s*)", spaceNumber, space.ReservedBy, reservedTime)
	}
	log.Printf("PARKING_RESERVE: User (%s) reserved space (%d) with auto release (%v)", user, spaceNumber, autoRelease)

	space.Reserved = true
	space.ReservedBy = user
	space.ReservedById = userId
	space.ReservedTime = time.Now()
	space.AutoRelease = autoRelease

	l.SynchronizeToFile()
	return ""
}

func (l *ParkingLot) Release(parkingSpace, user string) (victimId, errMsg string) {
	space := l.GetSpace(parkingSpace)

	log.Printf("PARKING_RELEASE: User (%s) released (%s) space.", user, parkingSpace)

	space.Reserved = false
	l.SynchronizeToFile()

	if space.ReservedBy != user {
		return space.ReservedById, fmt.Sprintf(":warning: *%s* released your (*%s*) space (*%d*)", user, space.ReservedBy, space.Number)
	}
	return "", ""
}

func (l *ParkingLot) GetSpace(parkingSpace string) *ParkingSpace {
	spaceNumber, err := strconv.Atoi(parkingSpace)
	if err != nil {
		log.Fatalf("Could not convert parkingSpace %+v to int", parkingSpace)
	}

	space, ok := l.ParkingSpaces[spaceNumber]
	if !ok {
		log.Fatalf("Incorrect parking space number %s, %+v", parkingSpace, l)
	}
	return space
}

// TODO: Test this
func (l *ParkingLot) ReleaseSpaces(cTime time.Time) {
	for _, space := range l.ParkingSpaces {
		// Simple case
		if space.Reserved && space.AutoRelease {
			log.Println("AutoRelease space ", space.Number)
			space.Reserved = false
			space.AutoRelease = false
			// Fall-through to check if this is also a temporary
			// released space has to be reserved
		}

		// If a scheduled release was setup
		releaseInfo := l.ToBeReleased.Get(space.Number)
		log.Println("ReleaseInfo for space ", space.Number, releaseInfo)
		if releaseInfo == nil {
			continue
		}

		// On the day before the start of the release -> make the space
		// available for selection
		if releaseInfo.StartDate.Sub(cTime).Hours() < 24 && releaseInfo.StartDate.After(cTime) {
			log.Println("TempRelease space ", space.Number, releaseInfo)
			space.Reserved = false
			space.AutoRelease = false
		} else if releaseInfo.EndDate.Sub(cTime).Hours() < 24 && releaseInfo.EndDate.Before(cTime) {
			// On the day of the end of release -> reserve back the space
			// for the correct user
			log.Println("TempReserve space ", space.Number, releaseInfo)
			space.Reserved = true
			space.AutoRelease = false
			space.ReservedBy = releaseInfo.OwnerName
			space.ReservedById = releaseInfo.OwnerId
		}
	}
}

func getParkingLot(config *config.Config) (parkingLot ParkingLot) {
	path := config.ParkingFilename

	fileData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Could not read parking file (%s)", path)
	}

	parkingLot = NewParkingLotFromJson(fileData, config)

	loadedSpacesNum := len(parkingLot.ParkingSpaces)
	if loadedSpacesNum == 0 {
		log.Fatalf("No spaces found in (%s).", path)
	}

	log.Printf("INIT: Parking spaces list loaded successfully (%d spaces configured)", loadedSpacesNum)
	return parkingLot
}
