package common

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/AngelVI13/slack-bot/pkg/model/user"
)

type VacationsHash map[string]bool

func LoadVacationsHash(filename string) VacationsHash {
	data := VacationsHash{}

	b, err := os.ReadFile(filename)
	if err != nil {
		slog.Info("Could not read vacations hash file.", "err", err, "filename", filename)
		return data
	}

	// Unmarshal the provided data into the solid map
	err = json.Unmarshal(b, &data)
	if err != nil {
		slog.Info(
			"Could not parse vacations hash file.",
			"err",
			err,
			"filename",
			filename,
		)
		return data
	}

	slog.Info("Loaded vacations hash file.", "filename", filename, "hashNum", len(data))
	return data
}

func MakeHcmVacationHash(id int, company user.Company, startDate, endDate string) string {
	return fmt.Sprintf("%d_%s_%s_%s", id, company, startDate, endDate)
}

func MakeBssVacationHash(
	id string,
	company user.Company,
	startDate, endDate string,
) string {
	return fmt.Sprintf("%s_%s_%s_%s", id, company, startDate, endDate)
}
