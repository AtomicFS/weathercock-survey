// SPDX-License-Identifier: AGPL-3.0-or-later

package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerTime(t *testing.T) {
	at := time.Date(2027, 3, 1, 23, 30, 5, 123, time.UTC)

	testCases := []struct {
		name     string
		location *time.Location
		want     string
	}{
		// Pass
		{name: "UTC", location: time.UTC, want: `time="2027-03-01 23:30:05"`},
		// The time moves to the next day in a zone east of UTC.
		{name: "zone east of UTC", location: time.FixedZone("East", 2*60*60), want: `time="2027-03-02 01:30:05"`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			record := slog.NewRecord(at, slog.LevelWarn, "message", 0)
			require.NoError(t, NewHandler(&out, tc.location).Handle(context.Background(), record))
			assert.Contains(t, out.String(), tc.want)
		})
	}
}

func TestHandlerAttrNamedTime(t *testing.T) {
	// A log call MUST NOT crash the server: an attribute named "time" that is not a time stays as it is.
	var out bytes.Buffer
	record := slog.NewRecord(time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC), slog.LevelWarn, "message", 0)
	record.AddAttrs(slog.String("time", "not a time"))
	require.NotPanics(t, func() {
		require.NoError(t, NewHandler(&out, time.UTC).Handle(context.Background(), record))
	})
	assert.Contains(t, out.String(), `time="not a time"`)
}
