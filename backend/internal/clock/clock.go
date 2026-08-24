package clock

import (
	"time"
)

// Beijing is GMT+8. All engine timestamps go through this location.
var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing)
}

func Format(t time.Time) string {
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

func FormatNow() string {
	return Format(Now())
}

func Date(t time.Time) (year int, month time.Month, day int) {
	return t.In(Beijing).Date()
}
