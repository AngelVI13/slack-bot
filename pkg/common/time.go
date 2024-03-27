package common

import (
	"fmt"
	"time"
)

func CheckDateRange(start, end time.Time) string {
	today := time.Now()
	todayDate := time.Date(
		today.Year(),
		today.Month(),
		today.Day(),
		0,
		0,
		0,
		0,
		today.Location(),
	)

	if start.Before(todayDate) {
		return fmt.Sprintf("Start date is in the past: %s", start.Format("2006-01-02"))
	}

	if end.Before(start) {
		return fmt.Sprintf(
			"End date is before start date: Start(%s) - End(%s)",
			start.Format("2006-01-02"),
			end.Format("2006-01-02"),
		)
	}

	return ""
}

func EqualDate(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.Month() == date2.Month() &&
		date1.Day() == date2.Day()
}
