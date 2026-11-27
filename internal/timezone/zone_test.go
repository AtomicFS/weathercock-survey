// SPDX-License-Identifier: AGPL-3.0-or-later

package timezone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	testCases := []struct {
		name    string
		zone    string
		wantErr error
	}{
		// Pass
		{name: "UTC", zone: "UTC"},
		{name: "canonical name", zone: "Europe/Prague"},
		{name: "link to another zone", zone: "Europe/Oslo"},

		// Fail
		{name: "empty", zone: "", wantErr: ErrUnknownZone},
		{name: "zone of the host", zone: "Local", wantErr: ErrUnknownZone},
		{name: "not a zone", zone: "Nowhere/City", wantErr: ErrUnknownZone},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			location, err := Load(tc.zone)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, location)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.zone, location.String())
		})
	}
}
