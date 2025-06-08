package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	issues := 0

	for spaceKey, releasePool := range parkingLot.ToBeReleased {
		releases := map[string]int{}
		for _, release := range releasePool.All() {
			releases[release.DateRange()]++
		}

		for dateRange, numOccurances := range releases {
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
