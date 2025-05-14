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

func (tp TimeUnit) ToShortString() string {
	switch tp {
	case Years:
		return "y"
	case Months:
		return "mo"
	case Weeks:
		return "wk"
	case Days:
		return "d"
	case Hours:
		return "h"
	case Minutes:
		return "m"
	default:
		return "s"
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

type unitData struct {
	Unit             TimeUnit
	GetDurationUnits func(du DurationUnits) int
	GetRelativeUnits func(ru RelativeUnits) int
}

var unitTable = []unitData{
	{
		Years,
		func(du DurationUnits) int { return du.Years },
		func(ru RelativeUnits) int { return ru.Years },
	},
	{
		Months,
		func(du DurationUnits) int { return du.Months },
		func(ru RelativeUnits) int { return ru.Months },
	},
	{
		Weeks,
		func(du DurationUnits) int { return du.Weeks },
		func(ru RelativeUnits) int { return ru.Weeks },
	},
	{
		Days,
		func(du DurationUnits) int { return du.Days },
		func(ru RelativeUnits) int { return ru.Days },
	},
	{
		Hours,
		func(du DurationUnits) int { return du.Hours },
		func(ru RelativeUnits) int { return ru.Hours },
	},
	{
		Minutes,
		func(du DurationUnits) int { return du.Minutes },
		func(ru RelativeUnits) int { return ru.Minutes },
	},
	{
		Seconds,
		func(du DurationUnits) int { return du.Seconds },
		func(ru RelativeUnits) int { return ru.Seconds },
	},
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

func (du DurationUnits) String() string {
	parts := make([]string, 0, len(unitTable))
	for _, entry := range unitTable {
		value := entry.GetDurationUnits(du)
		if value > 0 {
			unitStr := entry.Unit.String()
			if value > 1 {
				unitStr += "s"
			}
			parts = append(parts, strconv.Itoa(value)+" "+unitStr)
		}
	}

	if len(parts) == 0 {
		return "0 seconds"
	}

	return strings.Join(parts, ", ")
}

func (du DurationUnits) ToShortString() string {
	parts := make([]string, 0, len(unitTable))
	for _, entry := range unitTable {
		value := entry.GetDurationUnits(du)
		if value > 0 {
			unitStr := entry.Unit.ToShortString()
			parts = append(parts, strconv.Itoa(value)+unitStr)
		}
	}

	if len(parts) == 0 {
		return "0s"
	}

	return strings.Join(parts, " ")
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
	for _, entry := range unitTable {
		value := entry.GetRelativeUnits(ru)
		if value > 0 {
			unitStr := entry.Unit.String()
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
	var parts []string = make([]string, 0, len(unitTable))
	for _, entry := range unitTable {
		value := entry.GetRelativeUnits(ru)
		if entry.Unit > p {
			break
		}
		if value == 0 {
			continue
		}

		unitStr := entry.Unit.String()
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
