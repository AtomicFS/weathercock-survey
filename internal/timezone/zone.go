// SPDX-License-Identifier: AGPL-3.0-or-later

// Package timezone loads the time zone of the configuration file from the
// time zone data (tzdata) of the host.
package timezone

import (
	"errors"
	"fmt"
	"time"
)

// ErrUnknownZone means that the name is not a time zone of the host.
var ErrUnknownZone = errors.New("unknown time zone")

// Load returns the time zone with the name. It refuses "" and "Local":
// time.LoadLocation reads them as UTC and as the zone of the host, so the
// periods would move when the host changes.
func Load(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, fmt.Errorf("%w: %q", ErrUnknownZone, name)
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		// The text of err only repeats the name.
		return nil, fmt.Errorf("%w: %q", ErrUnknownZone, name)
	}
	return location, nil
}
