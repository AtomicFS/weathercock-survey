// SPDX-License-Identifier: AGPL-3.0-or-later

package timezone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zoneTab is a short zone1970.tab in the format of tzdata.
const zoneTab = `# tzdb timezone descriptions, for users who do not care about old timestamps
#
# Europe/Commented
#codes	coordinates	TZ	comments
CZ,SK	+5005+01426	Europe/Prague
UA	+5026+03031	Europe/Kyiv	most of Ukraine

US	+404251-0740023	America/New_York	Eastern (most areas)
XX	+0000+00000
`

// placeWith returns a new directory that contains zone1970.tab with content.
func placeWith(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone1970.tab"), []byte(content), 0o600))
	return dir
}

func TestReadNamesSearch(t *testing.T) {
	// Each place lists a different name, so the test sees which place the search used.
	first := placeWith(t, "XX\t+0000+00000\tFirst/Place\n")
	second := placeWith(t, "XX\t+0000+00000\tSecond/Place\n")
	zoneInfo := placeWith(t, "XX\t+0000+00000\tZone/Info\n")
	empty := t.TempDir()
	regularFile := filepath.Join(first, "zone1970.tab")

	all := []string{"Zone/Info", "First/Place", "Second/Place"}

	testCases := []struct {
		name     string
		options  Options
		wantName string
	}{
		// Pass
		{name: "ZONEINFO first", options: Options{ZoneInfo: zoneInfo, Places: []string{first}}, wantName: "Zone/Info"},
		{name: "first place", options: Options{Places: []string{first, second}}, wantName: "First/Place"},
		{name: "place without the file", options: Options{Places: []string{empty, second}}, wantName: "Second/Place"},
		{name: "place is a file", options: Options{Places: []string{regularFile, second}}, wantName: "Second/Place"},
		// The search ignores ZONEINFO in these rows.
		{name: "ZONEINFO is a file", options: Options{ZoneInfo: regularFile, Places: []string{second}}, wantName: "Second/Place"},
		{name: "ZONEINFO without the file", options: Options{ZoneInfo: empty, Places: []string{second}}, wantName: "Second/Place"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			names := ReadNames(tc.options)
			assert.Empty(t, names.Problem())
			// Only the names of one place: the first place that contains the file ends the search.
			for _, name := range all {
				assert.Equal(t, name == tc.wantName, names.IsCanonical(name), name)
			}
		})
	}
}

func TestReadNamesProblems(t *testing.T) {
	readable := placeWith(t, zoneTab)

	unreadable := placeWith(t, zoneTab)
	require.NoError(t, os.Chmod(filepath.Join(unreadable, "zone1970.tab"), 0))

	directory := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(directory, "zone1970.tab"), 0o700))

	testCases := []struct {
		name   string
		places []string
		want   string
		skip   bool
	}{
		{name: "no place", places: nil, want: ProblemNotFound},
		{name: "no place contains the file", places: []string{t.TempDir()}, want: ProblemNotFound},
		// The first place that contains the file ends the search, also if the next place is readable.
		{name: "not readable", places: []string{unreadable, readable}, want: kindPermissionDenied, skip: os.Geteuid() == 0},
		{name: "not a regular file", places: []string{directory, readable}, want: kindNotRegularFile},
		// A file without names would make each stored name look renamed.
		{name: "empty file", places: []string{placeWith(t, ""), readable}, want: ProblemNoZones},
		{name: "only comments", places: []string{placeWith(t, "# Europe/Prague\n#codes\tcoordinates\tTZ\n"), readable}, want: ProblemNoZones},
		{name: "damaged file", places: []string{placeWith(t, "Europe/Prague\n"), readable}, want: ProblemNoZones},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("root can read each file")
			}
			names := ReadNames(Options{Places: tc.places})
			assert.Equal(t, tc.want, names.Problem())
			assert.False(t, names.IsCanonical("Europe/Prague"), "without a file, only UTC and Etc zones are canonical")
			assert.True(t, names.IsCanonical("UTC"))
		})
	}
}

func TestZeroNamesIsNotRead(t *testing.T) {
	// A Names that does not come from ReadNames MUST NOT count as a file that was read.
	assert.Equal(t, ProblemNotFound, Names{}.Problem())
}

func TestIsCanonical(t *testing.T) {
	names := ReadNames(Options{Places: []string{placeWith(t, zoneTab)}})
	require.Empty(t, names.Problem())

	testCases := []struct {
		name string
		want bool
	}{
		// Pass
		{name: "UTC", want: true},
		{name: "Etc/UTC", want: true},
		{name: "Etc/GMT", want: true},
		{name: "Etc/GMT+1", want: true},
		{name: "Etc/GMT+12", want: true},
		{name: "Etc/GMT-1", want: true},
		{name: "Etc/GMT-14", want: true},
		{name: "Europe/Prague", want: true},
		{name: "Europe/Kyiv", want: true},
		{name: "America/New_York", want: true},

		// Fail
		{name: "Etc/GMT+0"},
		{name: "Etc/GMT-0"},
		{name: "Etc/GMT+13"},
		{name: "Etc/GMT-15"},
		{name: "Etc/GMT-01"},
		{name: "Etc/Universal"},
		{name: "Etc/UTC/x"},
		{name: "utc"},
		{name: "Europe/Prague "},
		{name: " Europe/Prague"},
		{name: "europe/prague"},
		{name: "Europe/Kiev"},
		{name: "Europe/Bratislava"},
		{name: "Europe/Commented"},
		{name: "CZ,SK"},
		{name: "+5005+01426"},
		{name: "most of Ukraine"},
		{name: "TZ"},
		{name: ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, names.IsCanonical(tc.name))
		})
	}
}
