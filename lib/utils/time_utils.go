package utils

import (
	"strconv"
	"strings"
	"time"
)

type TimeUnit int

const (
	Years = TimeUnit(iota)
	Months
	Weeks
	Days
	Hours
	Minutes
	Seconds
)

func (tp TimeUnit) String() string {
	switch tp {
	case Years:
		return "year"
	case Months:
		return "month"
	case Weeks:
		return "week"
	case Days:
		return "day"
	case Hours:
		return "hour"
	case Minutes:
		return "minute"
	default:
		return "second"
	}
}

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

type relativeUnitData struct {
	Unit TimeUnit
	Get  func(ru RelativeUnits) (string, int)
}

var relativeUnitTable = []relativeUnitData{
	{Years, func(ru RelativeUnits) (string, int) { return Years.String(), ru.Years }},
	{Months, func(ru RelativeUnits) (string, int) { return Months.String(), ru.Months }},
	{Weeks, func(ru RelativeUnits) (string, int) { return Weeks.String(), ru.Weeks }},
	{Days, func(ru RelativeUnits) (string, int) { return Days.String(), ru.Days }},
	{Hours, func(ru RelativeUnits) (string, int) { return Hours.String(), ru.Hours }},
	{Minutes, func(ru RelativeUnits) (string, int) { return Minutes.String(), ru.Minutes }},
	{Seconds, func(ru RelativeUnits) (string, int) { return Seconds.String(), ru.Seconds }},
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

func NewRelativeTimeUnits(unixSeconds int64) RelativeUnits {
	var isFuture bool
	relativeSeconds := time.Now().Unix() - unixSeconds
	if relativeSeconds < 0 {
		isFuture = true
		relativeSeconds = unixSeconds - time.Now().Unix()
	}

	return RelativeUnits{
		NewDurationUnits(time.Second * time.Duration(relativeSeconds)),
		isFuture,
	}
}

func (ru RelativeUnits) String() string {
	var relativeStr string
	for _, entry := range relativeUnitTable {
		unitStr, value := entry.Get(ru)
		if value > 0 {
			if value > 1 {
				unitStr += "s"
			}
			relativeStr = strconv.Itoa(value) + " " + unitStr
			break
		}
	}

	if len(relativeStr) == 0 {
		return "just now"
	}

	if ru.isFuture {
		return "in " + relativeStr
	}
	return relativeStr + " ago"
}

func (ru RelativeUnits) ToPrecisionString(p TimeUnit) string {
	var parts []string = make([]string, 0, len(relativeUnitTable))
	for _, entry := range relativeUnitTable {
		unitStr, value := entry.Get(ru)
		if entry.Unit > p {
			break
		}
		if value == 0 {
			continue
		}

		if value > 1 {
			unitStr += "s"
		}

		parts = append(parts, strconv.Itoa(value)+" "+unitStr)
	}

	if len(parts) == 0 {
		return "just now"
	}

	if ru.isFuture {
		return "in " + strings.Join(parts, ", ")
	}
	return strings.Join(parts, ", ") + " ago"
}
