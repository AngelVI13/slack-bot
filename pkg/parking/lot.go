package parking

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/AngelVI13/slack-bot/pkg/config"
)

type SpacesInfo []*ParkingSpace

type ParkingLot struct {
	ParkingSpaces
	config       *config.Config
	ToBeReleased ReleaseMap
}

func NewParkingLot() ParkingLot {
	return ParkingLot{
		ParkingSpaces: make(ParkingSpaces),
		ToBeReleased:  make(ReleaseMap),
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

	// Do not load any submitted items from to be released map
	for space, info := range d.ToBeReleased {
		if info.Submitted != true {
			delete(d.ToBeReleased, space)
		}
	}
}

func (d *ParkingLot) HasSpace(userId string) bool {
	userAlreadyReservedSpace := false
	for _, space := range d.ParkingSpaces {
		if space.Reserved && space.ReservedById == userId {
			userAlreadyReservedSpace = true
			break
		}
	}
	return userAlreadyReservedSpace
}

func (d *ParkingLot) HasTempRelease(userId string) bool {
	userAlreadyReleasedSpace := false
	for _, releaseInfo := range d.ToBeReleased {
		if releaseInfo.Submitted && releaseInfo.OwnerId == userId {
			userAlreadyReleasedSpace = true
			break
		}
	}

	return userAlreadyReleasedSpace
}

func (d *ParkingLot) GetSpacesByFloor(userId, floor string) SpacesInfo {
	floorSpaces := make(SpacesInfo, 0)
	allSpaces := d.GetSpacesInfo(userId)

	if floor == "" {
		return allSpaces
	}

	for _, space := range allSpaces {
		if strings.HasPrefix(string(space.ParkingKey()), floor) {
			floorSpaces = append(floorSpaces, space)
		}
	}
	return floorSpaces
}

func (d *ParkingLot) GetSpacesInfo(userId string) SpacesInfo {
	// Group spaces in 2 groups -> belonging to given user or not
	// The group that doesn't belong to user will be sorted by name and by status (reserved or not)
	userSpaces := make(SpacesInfo, 0)
	nonUserSpaces := make(SpacesInfo, 0)
	for _, d := range d.ParkingSpaces {
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

	allSpaces := make(SpacesInfo, 0, len(d.ParkingSpaces))
	allSpaces = append(allSpaces, userSpaces...)
	allSpaces = append(allSpaces, nonUserSpaces...)
	return allSpaces
}

func (l *ParkingLot) Reserve(parkingSpace ParkingKey, user, userId string, autoRelease bool) (errMsg string) {
	space := l.GetSpace(parkingSpace)
	if space == nil {
		return fmt.Sprintf("Failed to reserve space: couldn't find the space %s", parkingSpace)
	}

	// Only inform user if it was someone else that tried to reserved his space.
	// This prevents an unnecessary message if you double clicked the reserve button yourself
	if space.Reserved && space.ReservedById != userId {
		reservedTime := space.ReservedTime.Format("Mon 15:04")
		return fmt.Sprintf(
			"*Error*: Could not reserve *%s*. *%s* has just reserved it (at *%s*)",
			parkingSpace,
			space.ReservedBy,
			reservedTime,
		)
	}
	log.Printf(
		"PARKING_RESERVE: User (%s) reserved space (%s) with auto release (%v)",
		user,
		parkingSpace,
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

func (l *ParkingLot) Release(parkingSpace ParkingKey, userName, userId string) (victimId, errMsg string) {
	space := l.GetSpace(parkingSpace)
	if space == nil {
		return userId, fmt.Sprintf("Failed to release space: couldn't find the space %s", parkingSpace)
	}

	log.Printf("PARKING_RELEASE: User (%s) released (%s) space.", userName, parkingSpace)

	space.Reserved = false
	l.SynchronizeToFile()

	if space.ReservedById != userId {
		return space.ReservedById,
			fmt.Sprintf(
				":warning: *%s* released your (*%s*) space (*%s*)",
				userName,
				space.ReservedBy,
				parkingSpace,
			)

	}
	return "", ""
}

func (l *ParkingLot) GetSpace(parkingSpace ParkingKey) *ParkingSpace {
	space, ok := l.ParkingSpaces[parkingSpace]
	if !ok {
		log.Printf("Incorrect parking space number %s", parkingSpace)
		return nil
	}
	return space
}

// TODO: Test this
func (l *ParkingLot) ReleaseSpaces(cTime time.Time) {
	for spaceKey, space := range l.ParkingSpaces {
		// Simple case
		if space.Reserved && space.AutoRelease {
			log.Println("AutoRelease space ", spaceKey)
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
		if releaseInfo.StartDate.Sub(cTime).Hours() < 24 && releaseInfo.StartDate.After(cTime) {
			log.Println("TempRelease space ", spaceKey, releaseInfo)
			space.Reserved = false
			space.AutoRelease = false
		} else if releaseInfo.EndDate.Sub(cTime).Hours() < 24 && releaseInfo.EndDate.Before(cTime) {
			// On the day of the end of release -> reserve back the space
			// for the correct user
			log.Println("TempReserve space ", spaceKey, releaseInfo)
			space.Reserved = true
			space.AutoRelease = false
			space.ReservedBy = releaseInfo.OwnerName
			space.ReservedById = releaseInfo.OwnerId

			ok := l.ToBeReleased.Remove(spaceKey)
			if !ok {
				log.Printf("Failed removing release info for space %s", spaceKey)
			}
		}
	}

	l.SynchronizeToFile()
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

	log.Printf(
		"INIT: Parking spaces list loaded successfully (%d spaces configured)",
		loadedSpacesNum,
	)
	return parkingLot
}
