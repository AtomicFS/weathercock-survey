// SPDX-License-Identifier: AGPL-3.0-or-later

package timezone

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sentinel marks a value that MUST NOT appear in an error.
const sentinel = "USERID-SENTINEL"

// fakeLoad loads each name except "Broken/Zone" and the sentinel, and fails with the name in
// the error text, as time.LoadLocation does.
func fakeLoad(name string) (*time.Location, error) {
	if name == "Broken/Zone" || name == sentinel {
		return nil, errors.New("unknown time zone " + name)
	}
	return time.FixedZone(name, 0), nil
}

func TestCheckNewName(t *testing.T) {
	// The sentinel is listed, so that the check reaches the loader, whose error contains the name.
	listed := ReadNames(Options{Places: []string{placeWith(t, zoneTab+"XX\t+0000+00000\tBroken/Zone\nXX\t+0000+00000\t"+sentinel+"\n")}})
	require.Empty(t, listed.Problem())

	missing := ReadNames(Options{Places: []string{t.TempDir()}})

	notRegular := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(notRegular, "zone1970.tab"), 0o700))
	notReadable := ReadNames(Options{Places: []string{notRegular}})

	noZones := ReadNames(Options{Places: []string{placeWith(t, "# no zones\n")}})

	testCases := []struct {
		name    string
		zone    string
		names   Names
		wantErr error
		// wantKind is the kind of a ZoneTabNotReadableError.
		wantKind string
	}{
		// Pass
		{name: "UTC", zone: "UTC", names: listed},
		{name: "Etc zone", zone: "Etc/GMT+1", names: listed},
		{name: "listed name", zone: "Europe/Prague", names: listed},

		// Fail
		{name: "file not found", zone: "UTC", names: missing, wantErr: ErrZoneTabNotFound},
		// The file rows come before "unknown time zone": the name can be valid.
		{name: "file not found, listed name", zone: "Europe/Prague", names: missing, wantErr: ErrZoneTabNotFound},
		{name: "file not readable", zone: "Europe/Prague", names: notReadable, wantKind: kindNotRegularFile},
		{name: "file lists no zones", zone: "Europe/Prague", names: noZones, wantErr: ErrZoneTabNoZones},
		{name: "not canonical", zone: "Europe/Kiev", names: listed, wantErr: ErrUnknownZone},
		{name: "does not load", zone: "Broken/Zone", names: listed, wantErr: ErrUnknownZone},
		{name: "name in the error of the loader", zone: sentinel, names: listed, wantErr: ErrUnknownZone},
		{name: "zero Names", zone: "UTC", names: Names{}, wantErr: ErrZoneTabNotFound},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			location, err := CheckNewName(tc.zone, tc.names, fakeLoad)
			if tc.wantErr == nil && tc.wantKind == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.zone, location.String())
				return
			}
			require.Error(t, err)
			assert.Nil(t, location)
			if tc.wantKind != "" {
				var notReadableErr *ZoneTabNotReadableError
				require.ErrorAs(t, err, &notReadableErr)
				assert.Equal(t, tc.wantKind, notReadableErr.Kind)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
			assert.NotContains(t, err.Error(), tc.zone, "an error MUST NOT contain a time zone name")
		})
	}
}
