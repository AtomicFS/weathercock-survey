// SPDX-License-Identifier: AGPL-3.0-or-later

package timezone

import (
	"errors"
	"time"
)

// The errors of CheckNewName. Their text is fixed by the specification, and
// contains no time zone name.
var (
	ErrZoneTabNotFound = errors.New("cannot check the time zone: zone1970.tab not found; install the full time zone data, for example the package tzdata")
	ErrZoneTabNoZones  = errors.New("cannot check the time zone: zone1970.tab lists no zones; install the full time zone data, for example the package tzdata")
	ErrUnknownZone     = errors.New("unknown time zone; use a name from zone1970.tab, UTC, or an Etc zone")
)

// ZoneTabNotReadableError is the error of CheckNewName when zone1970.tab is not readable.
type ZoneTabNotReadableError struct {
	// Kind is the kind of problem, for example "permission denied". It never holds a path.
	Kind string
}

// Error returns the text of the specification, with the kind of problem.
func (e *ZoneTabNotReadableError) Error() string {
	return "cannot check the time zone: cannot read zone1970.tab: " + e.Kind + "; check the file modes of the time zone data"
}

// Loader loads a time zone. Production code passes time.LoadLocation.
type Loader func(name string) (*time.Location, error)

// CheckNewName checks a time zone name that the database does not store yet:
// zone1970.tab must be usable, and the name must be canonical and must load.
// The checks follow the order of the rows in the specification, so that a
// valid name does not get "unknown time zone" when zone1970.tab is missing.
func CheckNewName(name string, names Names, load Loader) (*time.Location, error) {
	switch problem := names.Problem(); problem {
	case "":
	case ProblemNotFound:
		return nil, ErrZoneTabNotFound
	case ProblemNoZones:
		return nil, ErrZoneTabNoZones
	default:
		return nil, &ZoneTabNotReadableError{Kind: problem}
	}
	if !names.IsCanonical(name) {
		return nil, ErrUnknownZone
	}
	location, err := load(name)
	if err != nil {
		// The error of the loader contains the name, so it is not wrapped.
		return nil, ErrUnknownZone
	}
	return location, nil
}
