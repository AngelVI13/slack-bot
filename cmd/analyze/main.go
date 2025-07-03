package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/AngelVI13/slack-bot/pkg/common"
	"github.com/AngelVI13/slack-bot/pkg/model/spaces"
)

func main() {
	parkingFilename := flag.String("park", "", "-park=parking.json")
	flag.Parse()

	if strings.TrimSpace(*parkingFilename) == "" {
		fmt.Println("no parking.json file provided")
		os.Exit(-1)
	}

	parkingLot := spaces.GetSpacesLot(*parkingFilename)
	today := common.TodayDate()
	issues := 0

	for spaceKey, releasePool := range parkingLot.ToBeReleased {
		duplicates := map[string]int{}
		active := []string{}
		for _, release := range releasePool.All() {
			dateRange := release.DateRange()
			duplicates[dateRange]++

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
