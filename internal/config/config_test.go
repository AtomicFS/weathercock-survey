// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
	"github.com/AtomicFS/weathercock-survey/internal/timezone"
)

const minimal = "time_zone: Europe/Prague\nform_files: [forms/a.yaml]\n"

func writeAt(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestLoad(t *testing.T) {
	t.Run("relative form file path", func(t *testing.T) {
		// The working directory is not the configuration directory.
		root := t.TempDir()
		t.Chdir(root)
		writeAt(t, filepath.Join(root, "etc", "config.yaml"), minimal)

		got, err := Load(filepath.Join("etc", "config.yaml"))
		require.NoError(t, err)
		want := &Config{TimeZone: "Europe/Prague", FormFiles: []string{filepath.Join(root, "etc", "forms", "a.yaml")}}
		assert.Equal(t, want, got)
	})

	t.Run("absolute form file path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		writeAt(t, path, "time_zone: UTC\nform_files: [/srv/form.yaml]\n")
		got, err := Load(path)
		require.NoError(t, err)
		assert.Equal(t, []string{"/srv/form.yaml"}, got.FormFiles)
	})

	t.Run("symbolic link not resolved", func(t *testing.T) {
		// root/b/link points to root/a/real. From the link, ../form.yaml is root/b/form.yaml.
		root := t.TempDir()
		writeAt(t, filepath.Join(root, "a", "real", "config.yaml"), "time_zone: UTC\nform_files: [../form.yaml]\n")
		require.NoError(t, os.MkdirAll(filepath.Join(root, "b"), 0o700))
		require.NoError(t, os.Symlink(filepath.Join(root, "a", "real"), filepath.Join(root, "b", "link")))

		got, err := Load(filepath.Join(root, "b", "link", "config.yaml"))
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, "b", "form.yaml")}, got.FormFiles)
	})
}

func TestLoadErrors(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		// wantErrs are the sentinels that the error wraps. Without them, the error is a YAML error.
		wantErrs []error
	}{
		// YAML errors.
		{name: "empty file", content: ""},
		{name: "syntax error", content: minimal + "a: b: c\n"},
		{name: "unknown key", content: minimal + "employees_file: e.yaml\n"},
		{name: "wrong type", content: "time_zone: [UTC]\nform_files: [a]\n"},
		{name: "two documents", content: minimal + "---\n" + minimal},

		// Values.
		{name: "missing time zone", content: "form_files: [a]\n", wantErrs: []error{ErrMissingTimeZone}},
		{name: "unknown time zone", content: "time_zone: Nowhere/City\nform_files: [a]\n", wantErrs: []error{timezone.ErrUnknownZone}},
		{name: "no form file", content: "time_zone: UTC\nform_files: []\n", wantErrs: []error{ErrFormFilesCount}},
		{name: "two form files", content: "time_zone: UTC\nform_files: [a, b]\n", wantErrs: []error{ErrFormFilesCount}},
		{name: "all problems at once", content: "form_files: []\n", wantErrs: []error{ErrMissingTimeZone, ErrFormFilesCount}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			writeAt(t, path, tc.content)

			got, err := Load(path)
			assert.Nil(t, got)
			require.Error(t, err)
			if len(tc.wantErrs) == 0 {
				var yamlErr *yaml.LoadError
				require.ErrorAs(t, err, &yamlErr)
				assertSuggestions(t, err)
				return
			}
			for _, want := range tc.wantErrs {
				assert.ErrorIs(t, err, want)
			}
			assertSuggestions(t, err)
		})
	}

	t.Run("missing file", func(t *testing.T) {
		got, err := Load(filepath.Join(t.TempDir(), "config.yaml"))
		assert.Nil(t, got)
		assert.ErrorIs(t, err, fs.ErrNotExist)
		assertSuggestions(t, err)
	})
}

// assertSuggestions checks that each problem in err has its own suggestion.
func assertSuggestions(t *testing.T, err error) {
	t.Helper()
	for _, problem := range suggest.Problems(err) {
		assert.NotEqual(t, suggest.Bug, suggest.Of(problem), "no suggestion for: %v", problem)
	}
}

func TestExample(t *testing.T) {
	_, err := Load("../../examples/config-dev.yaml")
	require.NoError(t, err)
}
