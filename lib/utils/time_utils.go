package utils

import (
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
