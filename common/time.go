package common

import (
	"time"
)

func GetTimeStamp(year int, month time.Month, day int, hour int, minute int, sec int, nanosec int, loc *time.Location) uint64 {
	return uint64(time.Date(year, month, day, hour, minute, sec, nanosec, loc).Unix() * 1000)
}

func GetMonthTimeStamp(timePoint uint64) uint64 {
	t := time.Unix(int64(timePoint/1000), 0).UTC()
	month, year := t.Month(), t.Year()
	return GetTimeStamp(year, month, 1, 0, 0, 0, 0, time.UTC)
}

func IsCurrentMonth(timePoint uint64) bool {
	t := time.Unix(int64(timePoint/1000), 0).UTC()
	month := t.Month()
	currentMonth := time.Now().UTC().Month()
	if month == currentMonth {
		return true
	}
	return false
}
