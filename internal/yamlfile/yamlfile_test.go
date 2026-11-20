// SPDX-License-Identifier: AGPL-3.0-or-later

package yamlfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// sentinel marks file content that MUST NOT appear in an error.
const sentinel = "USERID-SENTINEL"

type testBackup struct {
	Directory string `yaml:"directory" required:"true"`
	Keep      int    `yaml:"keep"`
}

type testPerson struct {
	Email string `yaml:"email" required:"true"`
}

type testConfig struct {
	Name    string            `yaml:"name" required:"true"`
	Count   int               `yaml:"count"`
	Tags    []string          `yaml:"tags"`
	Backup  testBackup        `yaml:"backup"`
	People  []testPerson      `yaml:"people"`
	Claims  map[string]string `yaml:"claims"`
	Ignored string            `yaml:"-"`
	hidden  string
}

// writeFile writes content to a new file and returns its path.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadDecodes(t *testing.T) {
	content := strings.Join([]string{
		"# comment",
		"name: yes",
		"count: -2",
		`tags: [1.0, 0x10, 2027-01-01, "quoted"]`,
		"backup: {directory: /backup, keep: 0}",
		"people:",
		"- email: a@example.com",
		"- email: b@example.com",
		"claims: {hd: example.com, yes: no}",
		"",
	}, "\n")

	var got testConfig
	_, err := Load(writeFile(t, content), &got)
	require.NoError(t, err)

	want := testConfig{
		Name:   "yes",
		Count:  -2,
		Tags:   []string{"1.0", "0x10", "2027-01-01", "quoted"},
		Backup: testBackup{Directory: "/backup", Keep: 0},
		People: []testPerson{{Email: "a@example.com"}, {Email: "b@example.com"}},
		Claims: map[string]string{"hd": "example.com", "yes": "no"},
	}
	assert.Equal(t, want, got)
	assert.Empty(t, got.hidden)
}

func TestLoadAcceptsEmptyContainers(t *testing.T) {
	var got testConfig
	_, err := Load(writeFile(t, "name: x\ntags: []\nclaims: {}\npeople: []\n"), &got)
	require.NoError(t, err)
	assert.Equal(t, "x", got.Name)
	assert.Empty(t, got.Tags)
	assert.Empty(t, got.Claims)
	assert.Empty(t, got.People)
}

func TestLoadRootList(t *testing.T) {
	var got []testPerson
	_, err := Load(writeFile(t, "- email: a\n- email: b\n"), &got)
	require.NoError(t, err)
	assert.Equal(t, []testPerson{{Email: "a"}, {Email: "b"}}, got)

	path := writeFile(t, "- email: a\n- {}\n")
	_, err = Load(path, &got)
	assertFileError(t, FileError{Path: path, Line: 2, KeyPath: "[1].email", Problem: ProblemMissingKey}, err)
}

func TestLoadIntegers(t *testing.T) {
	testCases := []struct {
		name  string
		value string
		want  int
	}{
		{name: "zero", value: "0", want: 0},
		{name: "positive", value: "5", want: 5},
		{name: "negative", value: "-2", want: -2},
		{name: "large", value: "1234567890", want: 1234567890},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got testConfig
			_, err := Load(writeFile(t, "name: x\ncount: "+tc.value+"\n"), &got)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Count)
		})
	}
}

func TestLoadBadIntegers(t *testing.T) {
	values := []string{
		"010", "0x10", "0o10", "1_000", "+5", "1.0", `"5"`, `'5'`, "true", "-x", "-0x1", "--5",
		"-0", "99999999999999999999", "~", "", sentinel, `""`, "[1]", "{a: 1}",
	}
	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			path := writeFile(t, "name: x\ncount: "+value+"\n")
			var got testConfig
			_, err := Load(path, &got)
			assertFileError(t, FileError{Path: path, Line: 2, KeyPath: "count", Problem: ProblemNotInteger}, err)
		})
	}
}

// assertFileError checks that err is a FileError with the fields of want, and
// that its text after the path does not contain the sentinel.
func assertFileError(t *testing.T, want FileError, err error) {
	t.Helper()
	var fileErr *FileError
	require.ErrorAs(t, err, &fileErr)
	assert.Equal(t, want, *fileErr)
	assert.NotContains(t, strings.TrimPrefix(err.Error(), fileErr.Path), sentinel, "an error MUST NOT contain file content")
}

func TestLoadErrors(t *testing.T) {
	// want has no Path: the test sets it.
	testCases := []struct {
		name    string
		content string
		want    FileError
	}{
		{name: "empty file", content: "", want: FileError{Problem: ProblemEmptyFile}},
		{name: "only comments", content: "# " + sentinel + "\n", want: FileError{Problem: ProblemEmptyFile}},
		{name: "document marker only", content: "---\n", want: FileError{Problem: ProblemEmptyFile}},
		{name: "null document", content: "~\n", want: FileError{Problem: ProblemEmptyFile}},
		{name: "syntax error", content: "name: x\nbad: " + sentinel + ": x\n", want: FileError{Line: 2, Problem: ProblemInvalidYAML}},
		{name: "undefined alias", content: "name: *" + sentinel + "\n", want: FileError{Line: 1, Problem: ProblemInvalidYAML}},
		{name: "second document", content: "name: a\n---\nname: " + sentinel + "\n", want: FileError{Line: 2, Problem: ProblemManyDocuments}},
		{name: "second document is empty", content: "name: a\n---\n", want: FileError{Line: 2, Problem: ProblemManyDocuments}},
		{name: "anchor", content: "name: &" + sentinel + " x\n", want: FileError{Line: 1, KeyPath: "name", Problem: ProblemAnchorOrAlias}},
		{name: "anchor on a key", content: "&" + sentinel + " name: x\n", want: FileError{Line: 1, Problem: ProblemAnchorOrAlias}},
		{name: "tag", content: "name: !" + sentinel + " x\n", want: FileError{Line: 1, KeyPath: "name", Problem: ProblemTag}},
		{name: "explicit int tag", content: "name: x\ncount: !!int \"5\"\n", want: FileError{Line: 2, KeyPath: "count", Problem: ProblemTag}},
		{name: "tag in unused list", content: "name: x\ntags:\n  - !!str a\n", want: FileError{Line: 3, KeyPath: "tags[0]", Problem: ProblemTag}},
		{name: "merge key in mapping", content: "name: x\n<<: {a: " + sentinel + "}\n", want: FileError{Line: 2, Problem: ProblemMergeKey}},
		{name: "merge key in free mapping", content: "name: x\nclaims:\n  <<: {a: " + sentinel + "}\n", want: FileError{Line: 3, KeyPath: "claims", Problem: ProblemMergeKey}},
		{name: "unknown key", content: "name: x\n" + sentinel + ": 1\n", want: FileError{Line: 2, Problem: ProblemUnknownKey}},
		{name: "unknown nested key", content: "name: x\nbackup:\n  directory: /b\n  " + sentinel + ": 1\n", want: FileError{Line: 4, KeyPath: "backup", Problem: ProblemUnknownKey}},
		{name: "unknown key in list item", content: "name: x\npeople:\n  - email: a\n  - email: b\n    " + sentinel + ": 1\n", want: FileError{Line: 5, KeyPath: "people[1]", Problem: ProblemUnknownKey}},
		{name: "skipped field is unknown", content: "name: x\nignored: " + sentinel + "\n", want: FileError{Line: 2, Problem: ProblemUnknownKey}},
		{name: "duplicate key", content: "name: x\nname: " + sentinel + "\n", want: FileError{Line: 2, KeyPath: "name", Problem: ProblemDuplicateKey}},
		{name: "duplicate unknown key", content: "name: x\n" + sentinel + ": 1\n" + sentinel + ": 2\n", want: FileError{Line: 2, Problem: ProblemUnknownKey}},
		{name: "duplicate free key", content: "name: x\nclaims:\n  " + sentinel + ": a\n  " + sentinel + ": b\n", want: FileError{Line: 4, KeyPath: "claims", Problem: ProblemDuplicateKey}},
		{name: "missing key", content: "count: 1\n", want: FileError{Line: 1, KeyPath: "name", Problem: ProblemMissingKey}},
		{name: "missing nested key", content: "name: x\nbackup:\n  keep: 3\n", want: FileError{Line: 2, KeyPath: "backup.directory", Problem: ProblemMissingKey}},
		{name: "missing key in flow mapping", content: "name: x\n\nbackup: {keep: 3}\n", want: FileError{Line: 3, KeyPath: "backup.directory", Problem: ProblemMissingKey}},
		{name: "second document has a syntax error", content: "name: a\n---\nx: " + sentinel + ": y\n", want: FileError{Line: 3, Problem: ProblemInvalidYAML}},
		{name: "map tag", content: "name: x\nbackup: !!map {directory: /b}\n", want: FileError{Line: 2, KeyPath: "backup", Problem: ProblemTag}},
		{name: "anchor on a list", content: "name: x\ntags: &" + sentinel + " [a]\n", want: FileError{Line: 2, KeyPath: "tags", Problem: ProblemAnchorOrAlias}},
		{name: "missing key in list item", content: "name: x\npeople:\n  - email: a\n  - {}\n", want: FileError{Line: 4, KeyPath: "people[1].email", Problem: ProblemMissingKey}},
		{name: "string is a list", content: "name: [" + sentinel + "]\n", want: FileError{Line: 1, KeyPath: "name", Problem: ProblemNotScalar}},
		{name: "null string", content: "name:\n", want: FileError{Line: 1, KeyPath: "name", Problem: ProblemNotString}},
		{name: "mapping is a scalar", content: "name: x\nbackup: " + sentinel + "\n", want: FileError{Line: 2, KeyPath: "backup", Problem: ProblemNotMapping}},
		{name: "mapping is null", content: "name: x\nbackup:\n", want: FileError{Line: 2, KeyPath: "backup", Problem: ProblemNotMapping}},
		{name: "list is a scalar", content: "name: x\ntags: " + sentinel + "\n", want: FileError{Line: 2, KeyPath: "tags", Problem: ProblemNotList}},
		{name: "list item is a list", content: "name: x\ntags:\n  - [" + sentinel + "]\n", want: FileError{Line: 3, KeyPath: "tags[0]", Problem: ProblemNotScalar}},
		{name: "list item is null", content: "name: x\ntags:\n  -\n", want: FileError{Line: 3, KeyPath: "tags[0]", Problem: ProblemEmptyListItem}},
		{name: "mapping list item is null", content: "name: x\npeople:\n  -\n", want: FileError{Line: 3, KeyPath: "people[0]", Problem: ProblemEmptyListItem}},
		{name: "list item with a null tag", content: "name: x\ntags:\n  - !!null\n", want: FileError{Line: 3, KeyPath: "tags[0]", Problem: ProblemTag}},
		{name: "root is a scalar", content: sentinel + "\n", want: FileError{Line: 1, Problem: ProblemNotMapping}},
		{name: "root is a list", content: "- " + sentinel + "\n", want: FileError{Line: 1, Problem: ProblemNotMapping}},
		{name: "key is a list", content: "name: x\n? [" + sentinel + "]\n: a\n", want: FileError{Line: 2, Problem: ProblemNotScalar}},
		{name: "free mapping is a list", content: "name: x\nclaims: [" + sentinel + "]\n", want: FileError{Line: 2, KeyPath: "claims", Problem: ProblemNotMapping}},
		{name: "free value is a mapping", content: "name: x\nclaims:\n  " + sentinel + ": {a: b}\n", want: FileError{Line: 3, KeyPath: "claims", Problem: ProblemNotScalar}},
		{name: "free value is null", content: "name: x\nclaims:\n  " + sentinel + ":\n", want: FileError{Line: 3, KeyPath: "claims", Problem: ProblemNotString}},
		{name: "free key is a list", content: "name: x\nclaims:\n  ? [" + sentinel + "]\n  : a\n", want: FileError{Line: 3, KeyPath: "claims", Problem: ProblemNotScalar}},
		{name: "free key is null", content: "name: x\nclaims:\n  ~: " + sentinel + "\n", want: FileError{Line: 3, KeyPath: "claims", Problem: ProblemNullKey}},
		{name: "free value has an anchor", content: "name: x\nclaims:\n  a: &" + sentinel + " b\n", want: FileError{Line: 3, KeyPath: "claims", Problem: ProblemAnchorOrAlias}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, tc.content)
			var got testConfig
			_, err := Load(path, &got)
			want := tc.want
			want.Path = path
			assertFileError(t, want, err)
		})
	}
}

func TestLoadErrorFields(t *testing.T) {
	path := writeFile(t, "name: x\npeople:\n  - {}\n")
	_, err := Load(path, &testConfig{})

	var fileErr *FileError
	require.ErrorAs(t, err, &fileErr)
	assert.Equal(t, &FileError{Path: path, Line: 3, KeyPath: "people[0].email", Problem: ProblemMissingKey}, fileErr)
}

func TestFileProblem(t *testing.T) {
	dir := t.TempDir()
	readable := filepath.Join(dir, "readable.yaml")
	require.NoError(t, os.WriteFile(readable, []byte("name: x\n"), 0o600))
	unreadable := filepath.Join(dir, "unreadable.yaml")
	require.NoError(t, os.WriteFile(unreadable, []byte("name: x\n"), 0o600))
	require.NoError(t, os.Chmod(unreadable, 0))

	testCases := []struct {
		name string
		path string
		want string
		skip bool
	}{
		// Pass
		{name: "readable", path: readable, want: ""},

		// Fail
		{name: "missing", path: filepath.Join(dir, "missing.yaml"), want: ProblemFileNotExist},
		{name: "directory", path: dir, want: ProblemNotRegularFile},
		{name: "unreadable", path: unreadable, want: ProblemPermissionDenied, skip: os.Geteuid() == 0},
		// A path below a regular file gives neither "does not exist" nor "permission denied".
		{name: "cannot read", path: filepath.Join(unreadable, "x"), want: ProblemCannotRead},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("root can read each file")
			}
			assert.Equal(t, tc.want, FileProblem(tc.path))
		})
	}
}

func TestLoadFileProblem(t *testing.T) {
	// A file that does not open has no line.
	path := filepath.Join(t.TempDir(), "missing.yaml")
	_, err := Load(path, &testConfig{})
	assertFileError(t, FileError{Path: path, Problem: ProblemFileNotExist}, err)
}

func TestLoadLines(t *testing.T) {
	content := strings.Join([]string{
		"name: x",
		"backup:",
		"  directory: /b",
		"  keep: 3",
		"people:",
		"- email: a",
		"- email: b",
		"claims:",
		"  " + sentinel + ": v",
		"tags: [a, b]",
		"",
	}, "\n")
	path := writeFile(t, content)
	lines, err := Load(path, &testConfig{})
	require.NoError(t, err)

	want := map[string]int{
		"name": 1, "backup": 2, "backup.directory": 3, "backup.keep": 4,
		"people": 5, "people[0]": 6, "people[0].email": 6, "people[1]": 7, "people[1].email": 7,
		"claims": 8, "tags": 10, "tags[0]": 10, "tags[1]": 10,
	}
	for keyPath, line := range want {
		assert.Equal(t, line, lines.Line(keyPath), keyPath)
	}
	assert.Zero(t, lines.Line("count"), "a key that the file does not set")
	assert.Zero(t, lines.Line("claims."+sentinel), "a free key is content")
	for keyPath := range lines.lines {
		assert.NotContains(t, keyPath, sentinel)
	}
}

func TestLinesError(t *testing.T) {
	path := writeFile(t, "name: x\nbackup:\n  directory: /b\n  keep: 3\n")
	lines, err := Load(path, &testConfig{})
	require.NoError(t, err)

	assert.Equal(t, &FileError{Path: path, Line: 2, KeyPath: "backup", Problem: "p"}, lines.Error("backup", "p"))
	assert.Equal(t, &FileError{Path: path, Line: 4, KeyPath: "backup.keep", Problem: "p"}, lines.Error("backup.keep", "p"))
	assert.Equal(t, &FileError{Path: path, KeyPath: "count", Problem: "p"}, lines.Error("count", "p"), "no line for an unset key")
}

// failingUnmarshaler passes the walker but fails in the library.
type failingUnmarshaler struct {
	Value string `yaml:"value"`
}

func (failingUnmarshaler) UnmarshalYAML(*yaml.Node) error {
	return errors.New("library text " + sentinel)
}

func TestLoadInternalError(t *testing.T) {
	path := writeFile(t, "value: x\n")
	var got failingUnmarshaler
	_, err := Load(path, &got)
	assertFileError(t, FileError{Path: path, Problem: ProblemInternal}, err)
}

func TestLoadPanicsOnUnsupportedType(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	testCases := []struct {
		name string
		out  any
	}{
		{name: "not a pointer", out: testConfig{}},
		{name: "nil pointer", out: (*testConfig)(nil)},
		{name: "int64", out: &struct {
			F int64 `yaml:"f"`
		}{}},
		{name: "map of int", out: &struct {
			F map[string]int `yaml:"f"`
		}{}},
		{name: "pointer to struct", out: &struct {
			F *testBackup `yaml:"f"`
		}{}},
		{name: "slice of bool", out: &struct {
			F []bool `yaml:"f"`
		}{}},
		{name: "exported field without a yaml tag", out: &struct {
			F string
		}{}},
		{name: "inline option", out: &struct {
			F testBackup `yaml:"f,inline"`
		}{}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() { _, _ = Load(missing, tc.out) }, fmt.Sprintf("%T", tc.out))
		})
	}
}
