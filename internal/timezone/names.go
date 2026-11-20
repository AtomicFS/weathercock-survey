// SPDX-License-Identifier: AGPL-3.0-or-later

// Package timezone checks the time zone of the configuration file against the
// time zone data (tzdata) of the host.
package timezone

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/AtomicFS/weathercock-survey/internal/yamlfile"
)

// The problems of Names that are not the problems of reading a file.
const (
	// ProblemNotFound means that no place contains zone1970.tab.
	ProblemNotFound = "not found"
	// ProblemNoZones means that zone1970.tab lists no zones, for example because it is empty.
	ProblemNoZones = "lists no zones"
)

// zoneTabName is the file of tzdata that lists the canonical zone names since 1970.
const zoneTabName = "zone1970.tab"

// etcZone matches the Etc zones that are canonical. It is the pattern of the specification.
var etcZone = regexp.MustCompile(`^Etc/(UTC|GMT|GMT\+([1-9]|1[0-2])|GMT-([1-9]|1[0-4]))$`)

// DefaultPlaces are the directories that ReadNames searches after ZONEINFO.
// They are the places where Go reads the zones.
func DefaultPlaces() []string {
	return []string{"/usr/share/zoneinfo/", "/usr/share/lib/zoneinfo/", "/usr/lib/locale/TZ/", "/etc/zoneinfo"}
}

// Options say where ReadNames searches zone1970.tab.
type Options struct {
	// ZoneInfo is the value of the environment variable ZONEINFO. ReadNames
	// uses it first, if it is a directory.
	ZoneInfo string
	// Places are the directories to search after ZoneInfo, in order. Use DefaultPlaces.
	Places []string
}

// Names are the canonical names from zone1970.tab.
type Names struct {
	names   map[string]bool
	problem string
}

// ReadNames reads zone1970.tab from the first place that contains it.
// A missing or unreadable file is not an error: Problem tells it.
func ReadNames(opts Options) Names {
	for _, dir := range searchDirs(opts) {
		path := filepath.Join(dir, zoneTabName)
		// A place that is not a directory does not contain the file.
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			continue
		}
		// The first place that contains the file ends the search, also if the
		// file is not readable. A file in a later place can come from other
		// tzdata, and a refusal is safer than a check against the wrong list.
		return readFile(path)
	}
	return Names{problem: ProblemNotFound}
}

// searchDirs returns the directories in the order of the search.
func searchDirs(opts Options) []string {
	dirs := make([]string, 0, len(opts.Places)+1)
	if opts.ZoneInfo != "" {
		if info, err := os.Stat(opts.ZoneInfo); err == nil && info.IsDir() {
			dirs = append(dirs, opts.ZoneInfo)
		}
	}
	return append(dirs, opts.Places...)
}

// readFile reads the names. The problem never holds the path, because the path
// can be the value of ZONEINFO.
func readFile(path string) Names {
	if problem := yamlfile.FileProblem(path); problem != "" {
		return Names{problem: fileKind(problem)}
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: the directory is ZONEINFO or a place where Go reads the zones; the admin controls both.
	if err != nil {
		return Names{problem: fileKind(yamlfile.ReadProblem(err))}
	}
	names := parse(data)
	if len(names) == 0 {
		// An empty or damaged file would make each stored name look renamed.
		return Names{problem: ProblemNoZones}
	}
	return Names{names: names}
}

// The kinds of a problem to read zone1970.tab. The problems of yamlfile carry
// a suggestion for a file of the admin, which does not fit the time zone data.
const (
	kindPermissionDenied = "permission denied"
	kindNotRegularFile   = "not a regular file"
	kindCannotRead       = "cannot read file"
)

// fileKind limits the problem to the kinds of the specification. The file
// existed a moment before, so a missing file now is a problem of the read.
func fileKind(problem string) string {
	switch problem {
	case yamlfile.ProblemPermissionDenied:
		return kindPermissionDenied
	case yamlfile.ProblemNotRegularFile:
		return kindNotRegularFile
	default:
		return kindCannotRead
	}
}

// parse returns the names in the third column. Lines that start with # and
// empty lines are not names.
func parse(data []byte) map[string]bool {
	names := make(map[string]bool)
	for line := range strings.Lines(string(data)) {
		line = strings.TrimSuffix(line, "\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if columns := strings.Split(line, "\t"); len(columns) >= 3 {
			names[columns[2]] = true
		}
	}
	return names
}

// Problem returns "" if ReadNames read zone1970.tab, and the file lists at
// least one name. Else it returns ProblemNotFound, ProblemNoZones,
// "permission denied", "not a regular file", or "cannot read file".
//
// A Names that does not come from ReadNames has the problem ProblemNotFound,
// so that it never counts as a file that was read.
func (n Names) Problem() string {
	if n.problem == "" && len(n.names) == 0 {
		return ProblemNotFound
	}
	return n.problem
}

// IsCanonical reports if name is UTC, a canonical Etc zone, or a name in
// zone1970.tab. It compares exact strings. Without a file, only UTC and the
// Etc zones are canonical.
func (n Names) IsCanonical(name string) bool {
	return name == "UTC" || etcZone.MatchString(name) || n.names[name]
}
