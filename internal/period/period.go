// SPDX-License-Identifier: AGPL-3.0-or-later

// Package period splits time into survey periods: every N weeks, months,
// quarters, or years from an anchor date, in a time zone.
//
// The package does calendar math on dates, held as time.Time at midnight
// UTC, and turns a date into an instant only at the end, with midnight.
package period

import (
	"errors"
	"fmt"
	"time"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
)

// MaxEvery is the largest number of units in one period. A larger value is
// almost surely a typing error of the admin.
const MaxEvery = 100

// The problems that New finds.
var (
	ErrEvery        = fmt.Errorf("every must be from 1 to %d", MaxEvery)
	ErrAnchorFormat = errors.New("anchor must be a date in the form YYYY-MM-DD")
	ErrAnchorAlign  = errors.New("anchor must be the start of a unit")
	ErrNoLocation   = errors.New("missing time zone")
)

// Cycle is a checked cycle of periods. Only New makes one.
type Cycle struct {
	unit     Unit
	every    int
	anchor   time.Time
	location *time.Location
}

// Period is one period. It holds each time from Start up to, but not
// including, End. Both are in UTC.
type Period struct {
	// Name is stable, for example "2027-W10", "2027-03", "2027-Q1", or "2027".
	Name  string
	Start time.Time
	// End is the planned end.
	End time.Time
}

// New returns the cycle of every units from the anchor date in the location.
// The anchor must be the start of a unit: a Monday, the 1st of a month, the
// 1st of a quarter, or 1 January. New returns all problems at once, joined
// with errors.Join. Each problem has a suggestion.
func New(unit Unit, every int, anchor string, location *time.Location) (Cycle, error) {
	var problems []error
	if !unit.valid() {
		problems = append(problems, suggest.Wrap(ErrUnit, "set unit to week, month, quarter, or year"))
	}
	if every < 1 || every > MaxEvery {
		problems = append(problems, suggest.Wrap(fmt.Errorf("%w, not %d", ErrEvery, every),
			fmt.Sprintf("set every to a whole number from 1 to %d", MaxEvery)))
	}
	date, err := time.Parse(time.DateOnly, anchor)
	if err != nil {
		problems = append(problems, suggest.Wrap(fmt.Errorf("%w, not %q", ErrAnchorFormat, anchor),
			"write the anchor as YYYY-MM-DD, for example 2027-01-01"))
	} else if unit.valid() {
		if before := unit.start(date); !before.Equal(date) {
			after := unit.add(before, 1)
			problems = append(problems, suggest.Wrap(
				fmt.Errorf("%w: %s is not the start of a %s", ErrAnchorAlign, anchor, unit),
				fmt.Sprintf("set the anchor to %s or %s", before.Format(time.DateOnly), after.Format(time.DateOnly)),
			))
		}
	}
	if location == nil {
		problems = append(problems, suggest.Wrap(ErrNoLocation, suggest.Bug))
	}
	if len(problems) > 0 {
		return Cycle{}, errors.Join(problems...)
	}
	return Cycle{unit: unit, every: every, anchor: date, location: location}, nil
}

// At returns the period that holds t. It returns false if t is before the
// anchor: no period exists then.
func (c Cycle) At(t time.Time) (Period, bool) {
	if t.Before(midnight(c.anchor, c.location)) {
		return Period{}, false
	}
	// Clocks that go back across midnight can show the day before after the
	// start of a period. So the local date can point to an earlier period
	// than t: the loop moves on until t is before the end.
	year, month, day := t.In(c.location).Date()
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if date.Before(c.anchor) {
		date = c.anchor
	}
	index := c.unit.between(c.anchor, c.unit.start(date)) / c.every
	for {
		period := c.period(index)
		if t.Before(period.End) {
			return period, true
		}
		index++
	}
}

// period returns the period with the index; the first period has index 0.
func (c Cycle) period(index int) Period {
	start := c.unit.add(c.anchor, index*c.every)
	end := c.unit.add(c.anchor, (index+1)*c.every)
	return Period{
		Name:  c.unit.name(start),
		Start: midnight(start, c.location).UTC(),
		End:   midnight(end, c.location).UTC(),
	}
}

// midnight returns the first instant of the date in the location. If a
// daylight saving change skips local midnight, it returns the first valid
// local time after it. If local midnight occurs two times, it returns the
// first. time.Date alone does not say which of the two zones it picks.
func midnight(date time.Time, location *time.Location) time.Time {
	t := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
	start, end := t.ZoneBounds()
	switch {
	case wallClock(t).Before(date):
		// In a gap, before it: the first valid time is the end of the zone.
		return end
	case wallClock(t).After(date):
		// In a gap, after it: the first valid time is the start of the zone.
		return start
	}
	// Midnight exists. If the zone before t also had this midnight, that one
	// came first.
	_, offset := t.Zone()
	_, offsetBefore := start.Add(-time.Nanosecond).Zone()
	earlier := t.Add(time.Duration(offset-offsetBefore) * time.Second)
	if earlier.Before(start) && wallClock(earlier).Equal(date) {
		return earlier
	}
	return t
}

// wallClock returns the local date and time of t, written as UTC, so that
// it compares with a date.
func wallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}
