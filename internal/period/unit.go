// SPDX-License-Identifier: AGPL-3.0-or-later

package period

import (
	"errors"
	"fmt"
	"time"
)

// ErrUnit means that the unit of a cycle is missing or unknown.
var ErrUnit = errors.New("unit must be week, month, quarter, or year")

// Unit is the length of one step of a cycle. The zero value is no unit, so
// that a missing key does not become a weekly cycle.
type Unit uint8

// The units of a cycle.
const (
	Week Unit = iota + 1
	Month
	Quarter
	Year
)

func (u Unit) String() string {
	switch u {
	case Week:
		return "week"
	case Month:
		return "month"
	case Quarter:
		return "quarter"
	case Year:
		return "year"
	default:
		return fmt.Sprintf("Unit(%d)", u)
	}
}

// UnmarshalText reads a unit from a form file.
func (u *Unit) UnmarshalText(text []byte) error {
	switch string(text) {
	case "week":
		*u = Week
	case "month":
		*u = Month
	case "quarter":
		*u = Quarter
	case "year":
		*u = Year
	default:
		return fmt.Errorf("%w, not %q", ErrUnit, text)
	}
	return nil
}

// valid tells if u is one of the units.
func (u Unit) valid() bool {
	return u >= Week && u <= Year
}

// start returns the first day of the unit that holds the date.
func (u Unit) start(date time.Time) time.Time {
	year, month, day := date.Date()
	switch u {
	case Week:
		sinceMonday := (int(date.Weekday()) + 6) % 7
		return time.Date(year, month, day-sinceMonday, 0, 0, 0, 0, time.UTC)
	case Month:
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	case Quarter:
		return time.Date(year, (month-1)/3*3+1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	}
}

// add returns the date n units after the date. The date must be the start
// of a unit, so that a month never ends on a missing day 31.
func (u Unit) add(date time.Time, n int) time.Time {
	switch u {
	case Week:
		return date.AddDate(0, 0, 7*n)
	case Month:
		return date.AddDate(0, n, 0)
	case Quarter:
		return date.AddDate(0, 3*n, 0)
	default:
		return date.AddDate(n, 0, 0)
	}
}

// between returns the number of units from the start of a unit to the start
// of a later unit.
func (u Unit) between(from, to time.Time) int {
	months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month())
	switch u {
	case Week:
		return int((to.Unix() - from.Unix()) / (7 * 24 * 60 * 60))
	case Month:
		return months
	case Quarter:
		return months / 3
	default:
		return to.Year() - from.Year()
	}
}

// name returns the stable name of the period that starts at the date.
func (u Unit) name(date time.Time) string {
	switch u {
	case Week:
		year, week := date.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case Month:
		return fmt.Sprintf("%04d-%02d", date.Year(), date.Month())
	case Quarter:
		return fmt.Sprintf("%04d-Q%d", date.Year(), (date.Month()-1)/3+1)
	default:
		return fmt.Sprintf("%04d", date.Year())
	}
}
