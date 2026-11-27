// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AtomicFS/weathercock-survey/internal/config"
	"github.com/AtomicFS/weathercock-survey/internal/suggest"
	"github.com/AtomicFS/weathercock-survey/internal/timezone"
)

// writeConfig writes a configuration file with content, and a copy of the
// example form as form.yaml next to it. It returns the path of the configuration file.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	example, err := os.ReadFile(filepath.Join("..", "..", "examples", "forms", "example.yaml"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "form.yaml"), example, 0o600)) //nolint:gosec // G703: a fixed name in t.TempDir.
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

const validConfig = "time_zone: UTC\nform_files: [form.yaml]\n"

func TestRun(t *testing.T) {
	valid := writeConfig(t, validConfig)
	twoProblems := writeConfig(t, "form_files: []\n")

	testCases := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout bool
		// wantLines is the number of error log lines, one for each problem.
		wantLines int
	}{
		// Pass
		{name: "serve in development mode", args: []string{"serve", "--dev", "--config", valid}, wantCode: 0},
		{name: "help", args: []string{"--help"}, wantCode: 0, wantStdout: true},

		// Fail
		{name: "serve without development mode", args: []string{"serve", "--config", valid}, wantCode: 2, wantLines: 1},
		{name: "two problems in the configuration file", args: []string{"serve", "--dev", "--config", twoProblems}, wantCode: 2, wantLines: 2},
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
			if tc.wantCode != 0 {
				assert.NotEmpty(t, stderr.String())
			}
			assert.Equal(t, tc.wantLines, strings.Count(stderr.String(), "level=ERROR"), "stderr: %s", stderr.String())
		})
	}
}

func TestServe(t *testing.T) {
	valid := writeConfig(t, validConfig)
	unknownZone := writeConfig(t, "time_zone: Nowhere/City\nform_files: [form.yaml]\n")
	missing := filepath.Join(t.TempDir(), "config.yaml")
	missingForm := writeConfig(t, "time_zone: UTC\nform_files: [missing.yaml]\n")

	testCases := []struct {
		name    string
		cmd     serveCmd
		wantErr error
	}{
		// Pass
		{name: "development mode", cmd: serveDev(valid)},

		// Fail
		{name: "not development mode", cmd: serveCmd{configOption: configOption{Config: valid}}, wantErr: errDevRequired},
		{name: "empty configuration path", cmd: serveDev(""), wantErr: errEmptyConfig},
		{name: "missing configuration file", cmd: serveDev(missing), wantErr: fs.ErrNotExist},
		{name: "unknown time zone", cmd: serveDev(unknownZone), wantErr: timezone.ErrUnknownZone},
		{name: "missing form file", cmd: serveDev(missingForm), wantErr: fs.ErrNotExist},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.Run(slog.New(slog.DiscardHandler))
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.wantErr)
			assert.NotEqual(t, suggest.Bug, suggest.Of(err), "each error has a suggestion")
		})
	}
}

func TestLogErrors(t *testing.T) {
	first := suggest.Wrap(config.ErrMissingTimeZone, "fix one")
	second := suggest.Wrap(config.ErrFormFilesCount, "fix two")
	plain := errors.New("plain")
	// A wrapped join is one problem: the outer text and suggestion stay.
	wrapped := suggest.Wrap(fmt.Errorf("form.yaml: %w", errors.Join(first, second)), "fix three")

	var records []slog.Record
	logErrors(slog.New(recordHandler{records: &records}), errors.Join(first, errors.Join(second, plain), wrapped))

	type line struct{ err, suggestion string }
	var got []line
	for _, record := range records {
		assert.Equal(t, slog.LevelError, record.Level)
		got = append(got, line{attrString(record, "error"), attrString(record, "suggestion")})
	}
	want := []line{
		{first.Error(), "fix one"},
		{second.Error(), "fix two"},
		{plain.Error(), suggest.Bug},
		{wrapped.Error(), "fix three"},
	}
	assert.Equal(t, want, got)
}

func TestServeWarnsAboutZoneEnv(t *testing.T) {
	t.Setenv("ZONEINFO", t.TempDir())
	t.Setenv("GOROOT", "/nonexistent")
	path := writeConfig(t, validConfig)

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

func TestRunLogTimeFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"serve", "--config", writeConfig(t, validConfig)}, &stdout, &stderr)
	assert.Equal(t, exitRefuse, code)
	assert.Regexp(t, `time="\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}" level=ERROR`, stderr.String())
}

// serveDev returns the command "serve --dev --config <path>".
func serveDev(path string) serveCmd {
	return serveCmd{configOption: configOption{Config: path}, Dev: true}
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
