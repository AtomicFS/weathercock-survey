// SPDX-License-Identifier: AGPL-3.0-or-later

package period

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
)

func loadZone(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	require.NoError(t, err)
	return location
}

func TestUnitUnmarshalText(t *testing.T) {
	testCases := []struct {
		name    string
		text    string
		want    Unit
		wantErr error
	}{
		// Pass
		{name: "week", text: "week", want: Week},
		{name: "month", text: "month", want: Month},
		{name: "quarter", text: "quarter", want: Quarter},
		{name: "year", text: "year", want: Year},

		// Fail
		{name: "empty", text: "", wantErr: ErrUnit},
		{name: "capital letter", text: "Week", wantErr: ErrUnit},
		{name: "unknown", text: "weekly", wantErr: ErrUnit},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var unit Unit
			err := unit.UnmarshalText([]byte(tc.text))
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, unit)
			// String and UnmarshalText must use the same names.
			assert.Equal(t, tc.text, unit.String())
		})
	}
}

func TestNew(t *testing.T) {
	prague := loadZone(t, "Europe/Prague")
	testCases := []struct {
		name     string
		unit     Unit
		every    int
		anchor   string
		location *time.Location
		wantErrs []error
		// wantDates are the dates that the suggestion must show.
		wantDates []string
	}{
		// Pass
		{name: "week starts on Monday", unit: Week, every: 1, anchor: "2027-01-04", location: prague},
		{name: "month", unit: Month, every: 1, anchor: "2027-02-01", location: prague},
		{name: "quarter", unit: Quarter, every: 1, anchor: "2027-04-01", location: prague},
		{name: "year", unit: Year, every: 1, anchor: "2027-01-01", location: prague},
		{name: "largest every", unit: Month, every: MaxEvery, anchor: "2027-01-01", location: prague},

		// Fail
		{name: "missing unit", every: 1, anchor: "2027-01-01", location: prague, wantErrs: []error{ErrUnit}},
		{name: "unknown unit", unit: Year + 1, every: 1, anchor: "2027-01-01", location: prague, wantErrs: []error{ErrUnit}},
		{name: "every 0", unit: Month, every: 0, anchor: "2027-01-01", location: prague, wantErrs: []error{ErrEvery}},
		{name: "every too large", unit: Month, every: MaxEvery + 1, anchor: "2027-01-01", location: prague, wantErrs: []error{ErrEvery}},
		{name: "missing anchor", unit: Month, every: 1, anchor: "", location: prague, wantErrs: []error{ErrAnchorFormat}},
		{name: "anchor without leading zeros", unit: Month, every: 1, anchor: "2027-2-1", location: prague, wantErrs: []error{ErrAnchorFormat}},
		{name: "anchor not a date", unit: Month, every: 1, anchor: "2027-02-30", location: prague, wantErrs: []error{ErrAnchorFormat}},
		{name: "week anchor not a Monday", unit: Week, every: 1, anchor: "2027-01-06", location: prague, wantErrs: []error{ErrAnchorAlign}, wantDates: []string{"2027-01-04", "2027-01-11"}},
		{name: "month anchor not the 1st", unit: Month, every: 1, anchor: "2027-02-10", location: prague, wantErrs: []error{ErrAnchorAlign}, wantDates: []string{"2027-02-01", "2027-03-01"}},
		{name: "quarter anchor not a quarter start", unit: Quarter, every: 1, anchor: "2027-02-01", location: prague, wantErrs: []error{ErrAnchorAlign}, wantDates: []string{"2027-01-01", "2027-04-01"}},
		{name: "year anchor not 1 January", unit: Year, every: 1, anchor: "2027-07-01", location: prague, wantErrs: []error{ErrAnchorAlign}, wantDates: []string{"2027-01-01", "2028-01-01"}},
		{name: "missing time zone", unit: Month, every: 1, anchor: "2027-01-01", wantErrs: []error{ErrNoLocation}},
		{name: "all problems at once", every: 0, anchor: "", wantErrs: []error{ErrUnit, ErrEvery, ErrAnchorFormat, ErrNoLocation}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.unit, tc.every, tc.anchor, tc.location)
			if tc.wantErrs == nil {
				require.NoError(t, err)
				return
			}
			for _, wantErr := range tc.wantErrs {
				assert.ErrorIs(t, err, wantErr)
			}
			assert.Len(t, suggest.Problems(err), len(tc.wantErrs))
			for _, date := range tc.wantDates {
				assert.Contains(t, suggest.Of(err), date)
			}
		})
	}
}

func TestAt(t *testing.T) {
	prague := loadZone(t, "Europe/Prague")
	utc := func(year int, month time.Month, day, hour int) time.Time {
		return time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	}
	testCases := []struct {
		name   string
		unit   Unit
		every  int
		anchor string
		at     time.Time
		want   Period
		wantOK bool
	}{
		// Pass
		{
			name: "week", unit: Week, every: 1, anchor: "2025-01-06",
			at:   time.Date(2025, 3, 5, 12, 0, 0, 0, prague),
			want: Period{Name: "2025-W10", Start: utc(2025, 3, 2, 23), End: utc(2025, 3, 9, 23)}, wantOK: true,
		},
		{
			name: "ISO week of the next year", unit: Week, every: 1, anchor: "2024-12-30",
			at:   time.Date(2025, 1, 1, 12, 0, 0, 0, prague),
			want: Period{Name: "2025-W01", Start: utc(2024, 12, 29, 23), End: utc(2025, 1, 5, 23)}, wantOK: true,
		},
		{
			name: "week 53, every 2 weeks", unit: Week, every: 2, anchor: "2020-12-28",
			at:   time.Date(2021, 1, 5, 12, 0, 0, 0, prague),
			want: Period{Name: "2020-W53", Start: utc(2020, 12, 27, 23), End: utc(2021, 1, 10, 23)}, wantOK: true,
		},
		{
			name: "month with a daylight saving change", unit: Month, every: 1, anchor: "2025-01-01",
			at:   time.Date(2025, 3, 15, 12, 0, 0, 0, prague),
			want: Period{Name: "2025-03", Start: utc(2025, 2, 28, 23), End: utc(2025, 3, 31, 22)}, wantOK: true,
		},
		{
			name: "every 3 months", unit: Month, every: 3, anchor: "2025-02-01",
			at:   time.Date(2025, 6, 15, 12, 0, 0, 0, prague),
			want: Period{Name: "2025-05", Start: utc(2025, 4, 30, 22), End: utc(2025, 7, 31, 22)}, wantOK: true,
		},
		{
			name: "quarter", unit: Quarter, every: 1, anchor: "2025-01-01",
			at:   time.Date(2025, 5, 15, 12, 0, 0, 0, prague),
			want: Period{Name: "2025-Q2", Start: utc(2025, 3, 31, 22), End: utc(2025, 6, 30, 22)}, wantOK: true,
		},
		{
			name: "every 2 years", unit: Year, every: 2, anchor: "2024-01-01",
			at:   time.Date(2025, 6, 1, 12, 0, 0, 0, prague),
			want: Period{Name: "2024", Start: utc(2023, 12, 31, 23), End: utc(2025, 12, 31, 23)}, wantOK: true,
		},
		{
			name: "at the start", unit: Month, every: 1, anchor: "2025-01-01",
			at:   utc(2025, 2, 28, 23),
			want: Period{Name: "2025-03", Start: utc(2025, 2, 28, 23), End: utc(2025, 3, 31, 22)}, wantOK: true,
		},
		{
			name: "just before the end", unit: Month, every: 1, anchor: "2025-01-01",
			at:   utc(2025, 3, 31, 22).Add(-time.Nanosecond),
			want: Period{Name: "2025-03", Start: utc(2025, 2, 28, 23), End: utc(2025, 3, 31, 22)}, wantOK: true,
		},
		{
			name: "at the end is the next period", unit: Month, every: 1, anchor: "2025-01-01",
			at:   utc(2025, 3, 31, 22),
			want: Period{Name: "2025-04", Start: utc(2025, 3, 31, 22), End: utc(2025, 4, 30, 22)}, wantOK: true,
		},
		{
			name: "first period, by the local date", unit: Month, every: 1, anchor: "2025-01-01",
			at:   utc(2024, 12, 31, 23),
			want: Period{Name: "2025-01", Start: utc(2024, 12, 31, 23), End: utc(2025, 1, 31, 23)}, wantOK: true,
		},

		// Fail
		{
			name: "before the anchor", unit: Month, every: 1, anchor: "2025-01-01",
			at: utc(2024, 12, 31, 23).Add(-time.Nanosecond),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cycle, err := New(tc.unit, tc.every, tc.anchor, prague)
			require.NoError(t, err)
			got, ok := cycle.At(tc.at)
			require.Equal(t, tc.wantOK, ok)
			if !ok {
				return
			}
			assert.Empty(t, cmp.Diff(tc.want, got))
			assert.Equal(t, time.UTC, got.Start.Location())
			assert.Equal(t, time.UTC, got.End.Location())
		})
	}
}

func TestMidnight(t *testing.T) {
	// Real daylight saving changes at local midnight. None falls on a Monday or
	// on the 1st of a month, so At cannot reach them. time.Date picks a
	// different zone west and east of UTC, so both sides run each branch.
	testCases := []struct {
		name string
		zone string
		date time.Time
		want time.Time
	}{
		{
			name: "no change",
			zone: "Europe/Prague",
			date: time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2027, 2, 28, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "missing midnight gives the first valid time",
			zone: "America/Havana",
			date: time.Date(2025, 3, 9, 0, 0, 0, 0, time.UTC),
			want: time.Date(2025, 3, 9, 5, 0, 0, 0, time.UTC),
		},
		{
			name: "missing midnight in another zone",
			zone: "America/Santiago",
			date: time.Date(2025, 9, 7, 0, 0, 0, 0, time.UTC),
			want: time.Date(2025, 9, 7, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "missing midnight, east of UTC",
			zone: "Asia/Beirut",
			date: time.Date(2025, 3, 30, 0, 0, 0, 0, time.UTC),
			want: time.Date(2025, 3, 29, 22, 0, 0, 0, time.UTC),
		},
		{
			name: "midnight two times gives the first",
			zone: "America/Havana",
			date: time.Date(2025, 11, 2, 0, 0, 0, 0, time.UTC),
			want: time.Date(2025, 11, 2, 4, 0, 0, 0, time.UTC),
		},
		{
			name: "midnight two times, east of UTC",
			zone: "Asia/Amman",
			date: time.Date(2021, 10, 29, 0, 0, 0, 0, time.UTC),
			want: time.Date(2021, 10, 28, 21, 0, 0, 0, time.UTC),
		},
		{
			name: "missing day gives the next day",
			zone: "Pacific/Apia",
			date: time.Date(2011, 12, 30, 0, 0, 0, 0, time.UTC),
			want: time.Date(2011, 12, 30, 10, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := midnight(tc.date, loadZone(t, tc.zone))
			assert.True(t, tc.want.Equal(got), "want %s, got %s", tc.want, got.UTC())
		})
	}
}
