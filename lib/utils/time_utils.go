package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	second = time.Second
	minute = time.Minute
	hour   = time.Hour
	day    = time.Hour * 24
	month  = day * 30
	year   = month * 12
)

type durationData struct {
	unit time.Duration
	set  func(*DurationUnits, int)
}

var durationTable []durationData = []durationData{
	{year, func(du *DurationUnits, i int) { du.Years = i }},
	{month, func(du *DurationUnits, i int) { du.Months = i }},
	{day, func(du *DurationUnits, i int) { du.Days = i }},
	{hour, func(du *DurationUnits, i int) { du.Hours = i }},
	{minute, func(du *DurationUnits, i int) { du.Minutes = i }},
	{second, func(du *DurationUnits, i int) { du.Seconds = i }},
}

type DurationUnits struct {
	Seconds int
	Minutes int
	Hours   int
	Days    int
	Months  int
	Years   int
}

func (tu DurationUnits) String() string {
	var sb strings.Builder

	writeUnit := func(s string, u int) {
		if u == 0 {
			return
		}
		sb.WriteString(strconv.Itoa(u))
		sb.WriteString(" ")
		sb.WriteString(s)
		if u > 1 {
			sb.WriteString("s")
		}
		sb.WriteString(", ")
	}

	writeUnit("year", tu.Years)
	writeUnit("month", tu.Months)
	writeUnit("day", tu.Days)
	writeUnit("hour", tu.Hours)
	writeUnit("minute", tu.Minutes)
	writeUnit("second", tu.Seconds)

	str := sb.String()
	return str[:len(str)-2]
}

func (tu DurationUnits) ToShortString() string {
	var sb strings.Builder
	writeUnit := func(s string, u int) {
		if u == 0 {
			return
		}
		sb.WriteString(strconv.Itoa(u))
		sb.WriteString(s)
		sb.WriteString(" ")
	}

	writeUnit("y", tu.Years)
	writeUnit("mo", tu.Months)
	writeUnit("d", tu.Days)
	writeUnit("h", tu.Hours)
	writeUnit("m", tu.Minutes)
	writeUnit("s", tu.Seconds)

	return strings.TrimSpace(sb.String())
}

func (tu DurationUnits) ToRelativeString() string {
	toUnitStr := func(s string, u int) string {
		str := fmt.Sprintf("%d %s", u, s)
		if u > 1 {
			str += "s"
		}
		return str + " ago"
	}
	switch {
	case tu.Years > 0:
		return toUnitStr("year", tu.Years)
	case tu.Months > 0:
		return toUnitStr("month", tu.Months)
	case tu.Days > 0:
		return toUnitStr("day", tu.Days)
	case tu.Hours > 0:
		return toUnitStr("hour", tu.Hours)
	case tu.Minutes > 0:
		return toUnitStr("minute", tu.Minutes)
	case tu.Seconds > 0:
		return toUnitStr("second", tu.Seconds)
	default:
		return "invalid duration format"
	}
}

func DurationToUnits(d time.Duration) DurationUnits {
	tu := DurationUnits{}

	for _, entry := range durationTable {
		delta := d / entry.unit
		if delta > 0 {
			d -= delta * entry.unit
			entry.set(&tu, int(delta))
		}
	}

	return tu
}
