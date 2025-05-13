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
	week   = day * 7
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
	{week, func(du *DurationUnits, i int) { du.Weeks = i }},
	{day, func(du *DurationUnits, i int) { du.Days = i }},
	{hour, func(du *DurationUnits, i int) { du.Hours = i }},
	{minute, func(du *DurationUnits, i int) { du.Minutes = i }},
	{second, func(du *DurationUnits, i int) { du.Seconds = i }},
}

type DurationUnits struct {
	d       time.Duration
	Seconds int
	Minutes int
	Hours   int
	Days    int
	Weeks   int
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
	writeUnit("week", tu.Weeks)
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
	writeUnit("wk", tu.Weeks)
	writeUnit("d", tu.Days)
	writeUnit("h", tu.Hours)
	writeUnit("m", tu.Minutes)
	writeUnit("s", tu.Seconds)

	return strings.TrimSpace(sb.String())
}

func NewDurationUnits(d time.Duration) DurationUnits {
	tu := DurationUnits{d: d}

	for _, entry := range durationTable {
		delta := d / entry.unit
		if delta > 0 {
			d -= delta * entry.unit
			entry.set(&tu, int(delta))
		}
	}

	return tu
}

type RelativeUnits struct {
	DurationUnits
	isFuture bool
}

func (ru RelativeUnits) String() string {
	toUnitStr := func(s string, u int) string {
		str := fmt.Sprintf("%d %s", u, s)
		if u > 1 {
			str += "s"
		}
		if ru.isFuture {
			return "in " + str
		}
		return str + " ago"
	}

	switch {
	case ru.Years > 0:
		return toUnitStr("year", ru.Years)
	case ru.Months > 0:
		return toUnitStr("month", ru.Months)
	case ru.Weeks > 0:
		return toUnitStr("week", ru.Weeks)
	case ru.Days > 0:
		return toUnitStr("day", ru.Days)
	case ru.Hours > 0:
		return toUnitStr("hour", ru.Hours)
	case ru.Minutes > 0:
		return toUnitStr("minute", ru.Minutes)
	case ru.Seconds > 0:
		return toUnitStr("second", ru.Seconds)
	case ru.Seconds == 0:
		return "just now"
	default:
		return "invalid duration format"
	}
}

func NewRelativeTimeUnits(unixSeconds int64) RelativeUnits {
	var isFuture bool
	relativeSeconds := time.Now().Unix() - unixSeconds
	if relativeSeconds < 0 {
		isFuture = true
		relativeSeconds = unixSeconds - time.Now().Unix()
	}
	du := NewDurationUnits(time.Second * time.Duration(relativeSeconds))
	return RelativeUnits{du, isFuture}
}
