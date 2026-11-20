// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AtomicFS/weathercock-survey/internal/timezone"
	"github.com/AtomicFS/weathercock-survey/internal/yamlfile"
)

// sentinel marks file content that MUST NOT appear in an error.
const sentinel = "USERID-SENTINEL"

// writeConfig writes a configuration file with content, and the form file
// form.yaml next to it. It returns the path of the configuration file.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "form.yaml"), nil, 0o600))
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

const validConfig = "time_zone: UTC\nform_files: [form.yaml]\n"

// shortZoneTab is a zone1970.tab with one zone.
const shortZoneTab = "CZ,SK\t+5005+01426\tEurope/Prague\n"

// useZoneTab points ZONEINFO to a directory with a short zone1970.tab, so that
// the result does not depend on the tzdata of the host.
func useZoneTab(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone1970.tab"), []byte(shortZoneTab), 0o600))
	t.Setenv("ZONEINFO", dir)
}

// useZoneEnvWithSentinel sets ZONEINFO and GOROOT to paths that contain the
// sentinel, and returns the path of a valid configuration file.
func useZoneEnvWithSentinel(t *testing.T) string {
	t.Helper()
	zoneInfo := filepath.Join(t.TempDir(), sentinel)
	require.NoError(t, os.Mkdir(zoneInfo, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(zoneInfo, "zone1970.tab"), []byte(shortZoneTab), 0o600))
	t.Setenv("ZONEINFO", zoneInfo)
	t.Setenv("GOROOT", "/"+sentinel)
	return writeConfig(t, validConfig)
}

func TestRun(t *testing.T) {
	useZoneTab(t)
	valid := writeConfig(t, validConfig)
	unknownZone := writeConfig(t, "time_zone: "+sentinel+"\nform_files: [form.yaml]\n")
	missing := filepath.Join(t.TempDir(), "config.yaml")
	unknownKey := writeConfig(t, validConfig+sentinel+": x\n")

	testCases := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout bool
	}{
		// Pass
		{name: "serve in development mode", args: []string{"serve", "--dev", "--config", valid}, wantCode: 0},
		{name: "help", args: []string{"--help"}, wantCode: 0, wantStdout: true},

		// Fail
		{name: "serve without development mode", args: []string{"serve", "--config", valid}, wantCode: 2},
		{name: "missing configuration file", args: []string{"serve", "--dev", "--config", missing}, wantCode: 2},
		{name: "empty configuration path", args: []string{"serve", "--dev", "--config", ""}, wantCode: 2},
		{name: "unknown key in the configuration file", args: []string{"serve", "--dev", "--config", unknownKey}, wantCode: 2},
		{name: "unknown time zone", args: []string{"serve", "--dev", "--config", unknownZone}, wantCode: 2},
		{name: "no command", args: []string{}, wantCode: 2},
		{name: "unknown command", args: []string{"validate"}, wantCode: 2},
		{name: "unknown flag", args: []string{"serve", "--dev", "--nope"}, wantCode: 2},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, &stdout, &stderr)
			assert.Equal(t, tc.wantCode, code, "stderr: %s", stderr.String())
			if tc.wantStdout {
				assert.NotEmpty(t, stdout.String())
			}
			// Only the row "serve without development mode" shows that run prints
			// the error: the other serve rows also log the ZONEINFO warning.
			if tc.wantCode != 0 {
				assert.NotEmpty(t, stderr.String())
			}
			assert.NotContains(t, stderr.String(), sentinel, "an error MUST NOT contain file content")
		})
	}
}

func TestServe(t *testing.T) {
	useZoneTab(t)
	valid := writeConfig(t, validConfig)
	unknownZone := writeConfig(t, "time_zone: "+sentinel+"\nform_files: [form.yaml]\n")
	missing := filepath.Join(t.TempDir(), "config.yaml")
	unknownKey := writeConfig(t, validConfig+sentinel+": x\n")

	testCases := []struct {
		name        string
		cmd         serveCmd
		wantErr     error
		wantFileErr *yamlfile.FileError
	}{
		// Pass
		{name: "development mode", cmd: serveDev(valid)},

		// Fail
		{name: "not development mode", cmd: serveCmd{configOption: configOption{Config: valid}}, wantErr: errDevRequired},
		{name: "empty configuration path", cmd: serveDev(""), wantErr: errEmptyConfig},
		{name: "missing configuration file", cmd: serveDev(missing), wantFileErr: &yamlfile.FileError{Path: missing, Problem: yamlfile.ProblemFileNotExist}},
		{name: "unknown key", cmd: serveDev(unknownKey), wantFileErr: &yamlfile.FileError{Path: unknownKey, Line: 3, Problem: yamlfile.ProblemUnknownKey}},
		{name: "unknown time zone", cmd: serveDev(unknownZone), wantFileErr: &yamlfile.FileError{Path: unknownZone, Line: 1, KeyPath: "time_zone", Problem: timezone.ErrUnknownZone.Error()}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.Run(slog.New(slog.DiscardHandler))
			assertServeError(t, tc.wantErr, tc.wantFileErr, err)
		})
	}
}

func TestServeReadsZoneInfo(t *testing.T) {
	// ZONEINFO points to a zone1970.tab without zones. If serve ignored
	// ZONEINFO, it would read the file of the host, and UTC would pass.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "zone1970.tab"), []byte("# comment\n"), 0o600))
	t.Setenv("ZONEINFO", dir)
	path := writeConfig(t, validConfig)

	cmd := serveDev(path)
	err := cmd.Run(slog.New(slog.DiscardHandler))
	want := &yamlfile.FileError{Path: path, Line: 1, KeyPath: "time_zone", Problem: timezone.ErrZoneTabNoZones.Error()}
	assertServeError(t, nil, want, err)
}

func TestServeWarnsAboutZoneEnv(t *testing.T) {
	path := useZoneEnvWithSentinel(t)

	var records []slog.Record
	cmd := serveDev(path)
	require.NoError(t, cmd.Run(slog.New(recordHandler{records: &records})))

	want := timezone.EnvWarnings(os.Getenv)
	require.Len(t, want, 2, "both variables are set")
	var got []timezone.Warning
	for _, record := range records {
		assert.Equal(t, slog.LevelWarn, record.Level)
		got = append(got, timezone.Warning{Message: record.Message, Suggestion: attrString(record, "suggestion")})
	}
	assert.Equal(t, want, got)
}

func TestRunLogsWarnings(t *testing.T) {
	// The printed warnings MUST NOT show the values of the variables.
	path := useZoneEnvWithSentinel(t)

	var stdout, stderr bytes.Buffer
	code := run([]string{"serve", "--dev", "--config", path}, &stdout, &stderr)
	assert.Equal(t, 0, code, "stderr: %s", stderr.String())
	// The logger of run uses the time format of the specification.
	assert.Regexp(t, `time="\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}" level=WARN`, stderr.String())
	assert.NotContains(t, stderr.String(), sentinel, "a warning MUST NOT contain the value of the variable")
}

// serveDev returns the command "serve --dev --config <path>".
func serveDev(path string) serveCmd {
	return serveCmd{configOption: configOption{Config: path}, Dev: true}
}

// assertServeError checks err: no error if both wants are nil, a FileError
// with the fields of wantFileErr, or else an error that wraps wantErr.
func assertServeError(t *testing.T, wantErr error, wantFileErr *yamlfile.FileError, err error) {
	t.Helper()
	switch {
	case wantFileErr != nil:
		var fileErr *yamlfile.FileError
		require.ErrorAs(t, err, &fileErr)
		assert.Equal(t, wantFileErr, fileErr)
		assert.NotContains(t, err.Error(), sentinel, "an error MUST NOT contain file content")
	case wantErr != nil:
		assert.ErrorIs(t, err, wantErr)
	default:
		assert.NoError(t, err)
	}
}

// recordHandler keeps each log record, so that a test can check the records.
// It ignores the attributes of logger.With, so a test sees only the attributes of each call.
type recordHandler struct {
	records *[]slog.Record
}

func (h recordHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h recordHandler) Handle(_ context.Context, record slog.Record) error {
	*h.records = append(*h.records, record.Clone())
	return nil
}

func (h recordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h recordHandler) WithGroup(string) slog.Handler { return h }

// attrString returns the string value of the attribute key of the record, or "".
func attrString(record slog.Record, key string) string {
	value := ""
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			value = attr.Value.String()
			return false
		}
		return true
	})
	return value
}

func TestConfigDefault(t *testing.T) {
	var parsed cli
	var stdout, stderr bytes.Buffer
	_, err := newParser(&parsed, &stdout, &stderr).Parse([]string{"serve", "--dev"})
	require.NoError(t, err)
	assert.Equal(t, "/etc/weathercock-survey/config.yaml", parsed.Serve.Config)
}

// sourceURLPattern is the same pattern as the task build:check-source-url in Taskfile.yml.
const sourceURLPattern = `^https://[A-Za-z0-9.-]+(/[A-Za-z0-9._~-]+)+$`

func TestSourceURLIsHTTPS(t *testing.T) {
	assert.Regexp(t, sourceURLPattern, sourceURL)
}

// TestSourceURLPatternMatchesTaskfile makes sure that TestSourceURLPattern tests
// the pattern that the build uses.
func TestSourceURLPatternMatchesTaskfile(t *testing.T) {
	taskfile, err := os.ReadFile(filepath.Join("..", "..", "Taskfile.yml"))
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(taskfile), "grep -Eq '"+sourceURLPattern+"'"),
		"the pattern in Taskfile.yml (build:check-source-url) is different from sourceURLPattern")
}

func TestSourceURLPattern(t *testing.T) {
	testCases := []struct {
		name string
		url  string
		want bool
	}{
		// Pass
		{name: "repository", url: "https://github.com/AtomicFS/weathercock-survey", want: true},
		{name: "nested groups", url: "https://git.example.com/group/sub_group/repo.name~1", want: true},

		// Fail
		{name: "plain http", url: "http://github.com/a/b"},
		{name: "ssh", url: "ssh://github.com/a/b"},
		{name: "user", url: "https://git@github.com/a/b"},
		{name: "port", url: "https://github.com:8443/a/b"},
		{name: "no path", url: "https://github.com"},
		{name: "trailing slash", url: "https://github.com/a/b/"},
		{name: "query", url: "https://github.com/a/b?token=x"},
		{name: "fragment", url: "https://github.com/a/b#x"},
	}
	pattern := regexp.MustCompile(sourceURLPattern)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pattern.MatchString(tc.url), tc.url)
		})
	}
}
