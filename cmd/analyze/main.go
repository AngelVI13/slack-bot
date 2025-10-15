package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
	"github.com/AngelVI13/slack-bot/pkg/model/user"
)

func main() {
	parkingFilename := flag.String("park", "", "-park=parking.json")
	usersFilename := flag.String("users", "", "-users=users.json")
	flag.Parse()

	if strings.TrimSpace(*parkingFilename) == "" {
		fmt.Println("no parking.json file provided")
		os.Exit(-1)
	}

	if strings.TrimSpace(*usersFilename) == "" {
		fmt.Println("no users.json file provided")
		os.Exit(-1)
	}

	parkingLot := spaces.GetSpacesLot(*parkingFilename)
	usersManager := user.NewManager(*usersFilename)
	today := common.TodayDate()
	issues := 0

	for spaceKey, space := range parkingLot.UnitSpaces {
		if space.Reserved && !space.AutoRelease {
			if !usersManager.HasParkingById(space.ReservedById) {
				issues++
				fmt.Printf(
					"ERROR: %s is permanently reserved by user %q who doesn't have permanent space.\n",
					spaceKey,
					space.ReservedBy,
				)

				if parkingLot.ToBeReleased.HasActiveRelease(spaceKey) {
					issues++
					fmt.Printf(
						"\t and the space has an active temporary release\n",
					)
				}

				releases := parkingLot.ToBeReleased.GetAll(spaceKey)
				for _, release := range releases {
					if release.Submitted && release.DataPresent() &&
						today.Before(*release.StartDate) {
						issues++
						fmt.Printf(
							"\t and the space has temp releases for the future %s\n",
							release.String(),
						)
					}
				}
			}
		}
	}

	for spaceKey, releasePool := range parkingLot.ToBeReleased {
		duplicates := map[string]int{}
		active := []string{}
		for _, release := range releasePool.All() {
			dateRange := release.DateRange()
			duplicates[dateRange]++

			// if today.Before(*release.StartDate) &&
			// 	!usersManager.HasParkingById(release.OwnerId) {
			if !usersManager.HasParkingById(release.OwnerId) {
				issues++
				fmt.Printf(
					"ERROR: Temp release %s is set to return to owner %q who doesn't have permanent space.\n",
					release.String(),
					release.OwnerName,
				)
			}

			if release.Active {
				active = append(active, dateRange)

				if today.After(*release.EndDate) {
					issues++
					fmt.Printf(
						"ERROR: %q has active releases with end date in the past: %s (today: %s)\n",
						spaceKey,
						dateRange,
						today.Format("2006-01-02"),
					)
				}
			}
		}

		if len(active) > 1 {
			issues++
			fmt.Printf(
				"ERROR: %q has %d active releases: %v\n",
				spaceKey,
				len(active),
				active,
			)
		}

		for dateRange, numOccurances := range duplicates {
			if numOccurances > 1 {
				issues++
				fmt.Printf(
					"ERROR: %q has %d occurances of %q\n",
					spaceKey,
					numOccurances,
					dateRange,
				)
			}
		}
	}
	if issues == 0 {
		fmt.Printf("SUCCESS: No issues found in %q\n", *parkingFilename)
	} else {
		fmt.Printf("FAIL: %d issues found in %q\n", issues, *parkingFilename)
	}
}
