// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AtomicFS/weathercock-survey/internal/yamlfile"
)

// sentinel marks file content that MUST NOT appear in an error.
const sentinel = "USERID-SENTINEL"

// minimal is the shortest valid file. It has 2 lines.
var minimal = []string{"time_zone: Europe/Prague", "form_files: [forms/a.yaml]"}

func fileOf(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

// writeFile writes the configuration file, and the form file forms/a.yaml next to it.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "forms", "a.yaml"), "")
	path := filepath.Join(dir, "config.yaml")
	writeAt(t, path, content)
	return path
}

func writeAt(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// formFile is the path of forms/a.yaml next to the configuration file.
func formFile(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "forms", "a.yaml")
}

// fullFile sets each key, except employees_file, which version 0 refuses.
func fullFile() string {
	return fileOf(
		"time_zone: Europe/Prague",
		"form_files: [forms/a.yaml]",
		"trusted_people:",
		"- id: it-jane",
		"  name: Jane",
		"  email: jane@example.com",
		"  role: Server admin",
		"- {id: it-bob, name: Bob, email: bob@example.com, role: Backup holder}",
		"security_txt_file: security.txt",
		"host: survey.example.com",
		"oidc:",
		"  issuer_url: https://accounts.google.com",
		"  client_id: client",
		"  scopes: [openid, email]",
		"  user_id_claim: sub",
		"  required_claims: {hd: example.com}",
		"listen: 127.0.0.1:8080",
		"backup:",
		"  directory: /var/backups",
		"  keep: 3",
		"secret_files:",
		"  idp-client-secret: secrets/idp",
	)
}

// fullConfig is the result of fullFile. Version 0 resolves only the form files.
func fullConfig(configPath string) *Config {
	return &Config{
		TimeZone:  "Europe/Prague",
		FormFiles: []string{formFile(configPath)},
		TrustedPeople: []TrustedPerson{
			{ID: "it-jane", Name: "Jane", Email: "jane@example.com", Role: "Server admin"},
			{ID: "it-bob", Name: "Bob", Email: "bob@example.com", Role: "Backup holder"},
		},
		SecurityTxtFile: "security.txt",
		Host:            "survey.example.com",
		OIDC: OIDC{
			IssuerURL:      "https://accounts.google.com",
			ClientID:       "client",
			Scopes:         []string{"openid", "email"},
			UserIDClaim:    "sub",
			RequiredClaims: map[string]string{"hd": "example.com"},
		},
		Listen:      "127.0.0.1:8080",
		Backup:      Backup{Directory: "/var/backups", Keep: 3},
		SecretFiles: SecretFiles{IDPClientSecret: "secrets/idp"},
	}
}

func TestLoad(t *testing.T) {
	t.Run("full file", func(t *testing.T) {
		// security.txt and secrets/idp do not exist. Version 0 does not use them.
		path := writeFile(t, fullFile())
		got, lines, err := Load(path)
		require.NoError(t, err)
		if diff := cmp.Diff(fullConfig(path), got); diff != "" {
			assert.Fail(t, "config mismatch", diff)
		}
		assert.Equal(t, 2, lines.Line("form_files[0]"))
	})

	t.Run("empty mappings", func(t *testing.T) {
		// No key inside oidc or backup is required in version 0.
		path := writeFile(t, fileOf(append(minimal, "oidc: {}", "backup: {}")...))
		got, _, err := Load(path)
		require.NoError(t, err)
		want := &Config{TimeZone: "Europe/Prague", FormFiles: []string{formFile(path)}}
		if diff := cmp.Diff(want, got); diff != "" {
			assert.Fail(t, "config mismatch", diff)
		}
	})

	t.Run("values that version 0 does not use", func(t *testing.T) {
		// Version 0 checks the syntax and the types of these keys, but not
		// their values. A v1 value check MUST NOT refuse this file in version 0.
		content := fileOf(append(
			minimal,
			"trusted_people:",
			"- {id: same, name: A, email: a, role: r}",
			"- {id: same, name: B, email: b, role: r}",
			"oidc: {scopes: [email]}",
			"listen: 0.0.0.0:9999",
			"backup: {directory: relative/dir, keep: 0}",
		)...)
		_, _, err := Load(writeFile(t, content))
		require.NoError(t, err)

		_, _, err = Load(writeFile(t, fileOf(append(minimal, "trusted_people: []")...)))
		require.NoError(t, err, "an empty list of trusted people")
	})

	t.Run("minimal file", func(t *testing.T) {
		path := writeFile(t, fileOf(minimal...))
		got, lines, err := Load(path)
		require.NoError(t, err)
		want := &Config{TimeZone: "Europe/Prague", FormFiles: []string{formFile(path)}}
		if diff := cmp.Diff(want, got); diff != "" {
			assert.Fail(t, "config mismatch", diff)
		}
		assert.Equal(t, 2, lines.Line("form_files"))
		assert.Equal(t, 2, lines.Line("form_files[0]"))
	})
}

func TestLoadErrors(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    yamlfile.FileError
	}{
		// The rules of version 0.
		{name: "missing time zone", content: fileOf("form_files: [a]"), want: at(1, "time_zone", yamlfile.ProblemMissingKey)},
		{name: "missing form files", content: fileOf("time_zone: UTC"), want: at(1, "form_files", yamlfile.ProblemMissingKey)},
		{name: "no form file", content: fileOf("time_zone: UTC", "form_files: []"), want: at(2, "form_files", ProblemFormFilesCount)},
		{name: "two form files", content: fileOf("time_zone: UTC", "form_files:", "- /"+sentinel+"/a.yaml", "- /"+sentinel+"/b.yaml"), want: at(2, "form_files", ProblemFormFilesCount)},
		{name: "employees file", content: fileOf(append(minimal, "employees_file: /"+sentinel+"/e.yaml")...), want: at(3, "employees_file", ProblemNotSupportedV0)},
		{name: "empty employees file", content: fileOf(append(minimal, `employees_file: ""`)...), want: at(3, "employees_file", ProblemNotSupportedV0)},
		{name: "null employees file", content: fileOf(append(minimal, "employees_file:")...), want: at(3, "employees_file", yamlfile.ProblemNotString)},

		// A wrong type in a key that version 0 does not use.
		{name: "oidc scopes", content: fileOf(append(minimal, "oidc: {scopes: "+sentinel+"}")...), want: at(3, "oidc.scopes", yamlfile.ProblemNotList)},
		{name: "secret files is a string", content: fileOf(append(minimal, "secret_files: "+sentinel)...), want: at(3, "secret_files", yamlfile.ProblemNotMapping)},
		{name: "trusted person name is a list", content: fileOf(append(minimal, "trusted_people:", "- {id: a, name: ["+sentinel+"], email: e, role: r}")...), want: at(4, "trusted_people[0].name", yamlfile.ProblemNotScalar)},
		{name: "trusted person email is a list", content: fileOf(append(minimal, "trusted_people:", "- {id: a, name: n, email: ["+sentinel+"], role: r}")...), want: at(4, "trusted_people[0].email", yamlfile.ProblemNotScalar)},
		{name: "trusted person role is a list", content: fileOf(append(minimal, "trusted_people:", "- {id: a, name: n, email: e, role: ["+sentinel+"]}")...), want: at(4, "trusted_people[0].role", yamlfile.ProblemNotScalar)},
		{name: "trusted person without id", content: fileOf(append(minimal, "trusted_people:", "- {name: n, email: e, role: r}")...), want: at(4, "trusted_people[0].id", yamlfile.ProblemMissingKey)},
		{name: "trusted person without name", content: fileOf(append(minimal, "trusted_people:", "- {id: a, email: e, role: r}")...), want: at(4, "trusted_people[0].name", yamlfile.ProblemMissingKey)},
		{name: "trusted person without email", content: fileOf(append(minimal, "trusted_people:", "- id: a", "  name: n", "  role: r")...), want: at(4, "trusted_people[0].email", yamlfile.ProblemMissingKey)},
		{name: "second trusted person without role", content: fileOf(append(minimal, "trusted_people:", "- {id: a, name: n, email: e, role: r}", "- {id: b, name: n, email: e}")...), want: at(5, "trusted_people[1].role", yamlfile.ProblemMissingKey)},

		// Anonymity: no content in the error.
		{name: "unknown key", content: fileOf(append(minimal, sentinel+": x")...), want: at(3, "", yamlfile.ProblemUnknownKey)},
		{name: "unknown key in oidc", content: fileOf(append(minimal, "oidc:", "  "+sentinel+": x")...), want: at(4, "oidc", yamlfile.ProblemUnknownKey)},
		{name: "unknown key in backup", content: fileOf(append(minimal, "backup:", "  "+sentinel+": 1")...), want: at(4, "backup", yamlfile.ProblemUnknownKey)},
		{name: "unknown secret", content: fileOf(append(minimal, "secret_files:", "  "+sentinel+": /x")...), want: at(4, "secret_files", yamlfile.ProblemUnknownKey)},
		{name: "unknown key in trusted person", content: fileOf(append(minimal, "trusted_people:", "- "+sentinel+": x")...), want: at(4, "trusted_people[0]", yamlfile.ProblemUnknownKey)},
		{name: "claim value next to a bad key", content: fileOf(append(minimal, "oidc:", "  required_claims:", "    hd: "+sentinel, "  "+sentinel+": 1")...), want: at(6, "oidc", yamlfile.ProblemUnknownKey)},
		{name: "duplicate claim", content: fileOf(append(minimal, "oidc:", "  required_claims:", "    "+sentinel+": a", "    "+sentinel+": b")...), want: at(6, "oidc.required_claims", yamlfile.ProblemDuplicateKey)},
		{name: "claim value is a list", content: fileOf(append(minimal, "oidc:", "  required_claims:", "    "+sentinel+": [x]")...), want: at(5, "oidc.required_claims", yamlfile.ProblemNotScalar)},
		{name: "claims are a list", content: fileOf(append(minimal, "oidc:", "  required_claims: ["+sentinel+"]")...), want: at(4, "oidc.required_claims", yamlfile.ProblemNotMapping)},
		{name: "secret path is a list", content: fileOf(append(minimal, "secret_files: {idp-client-secret: ["+sentinel+"]}")...), want: at(3, "secret_files.idp-client-secret", yamlfile.ProblemNotScalar)},
		{name: "time zone is a list", content: fileOf("time_zone: ["+sentinel+"]", "form_files: [a]"), want: at(1, "time_zone", yamlfile.ProblemNotScalar)},
		{name: "form file is a mapping", content: fileOf("time_zone: UTC", "form_files: [{a: "+sentinel+"}]"), want: at(2, "form_files[0]", yamlfile.ProblemNotScalar)},
		{name: "form files is a string", content: fileOf("time_zone: UTC", "form_files: "+sentinel), want: at(2, "form_files", yamlfile.ProblemNotList)},
		{name: "employees file is a list", content: fileOf(append(minimal, "employees_file: ["+sentinel+"]")...), want: at(3, "employees_file", yamlfile.ProblemNotScalar)},
		{name: "trusted people is a string", content: fileOf(append(minimal, "trusted_people: "+sentinel)...), want: at(3, "trusted_people", yamlfile.ProblemNotList)},
		{name: "trusted person is a string", content: fileOf(append(minimal, "trusted_people:", "- "+sentinel)...), want: at(4, "trusted_people[0]", yamlfile.ProblemNotMapping)},
		{name: "trusted person id is a list", content: fileOf(append(minimal, "trusted_people:", "- {id: ["+sentinel+"], name: n, email: e, role: r}")...), want: at(4, "trusted_people[0].id", yamlfile.ProblemNotScalar)},
		{name: "security txt is a list", content: fileOf(append(minimal, "security_txt_file: ["+sentinel+"]")...), want: at(3, "security_txt_file", yamlfile.ProblemNotScalar)},
		{name: "host is a mapping", content: fileOf(append(minimal, "host: {a: "+sentinel+"}")...), want: at(3, "host", yamlfile.ProblemNotScalar)},
		{name: "oidc is a string", content: fileOf(append(minimal, "oidc: "+sentinel)...), want: at(3, "oidc", yamlfile.ProblemNotMapping)},
		{name: "issuer url is a list", content: fileOf(append(minimal, "oidc: {issuer_url: ["+sentinel+"]}")...), want: at(3, "oidc.issuer_url", yamlfile.ProblemNotScalar)},
		{name: "client id is a list", content: fileOf(append(minimal, "oidc: {client_id: ["+sentinel+"]}")...), want: at(3, "oidc.client_id", yamlfile.ProblemNotScalar)},
		{name: "scope is a list", content: fileOf(append(minimal, "oidc: {scopes: [["+sentinel+"]]}")...), want: at(3, "oidc.scopes[0]", yamlfile.ProblemNotScalar)},
		{name: "user id claim is a list", content: fileOf(append(minimal, "oidc: {user_id_claim: ["+sentinel+"]}")...), want: at(3, "oidc.user_id_claim", yamlfile.ProblemNotScalar)},
		{name: "backup is a string", content: fileOf(append(minimal, "backup: "+sentinel)...), want: at(3, "backup", yamlfile.ProblemNotMapping)},
		{name: "backup directory is a list", content: fileOf(append(minimal, "backup: {directory: ["+sentinel+"]}")...), want: at(3, "backup.directory", yamlfile.ProblemNotScalar)},
		{name: "backup keep is the sentinel", content: fileOf(append(minimal, "backup: {keep: "+sentinel+"}")...), want: at(3, "backup.keep", yamlfile.ProblemNotInteger)},
		{name: "listen is a mapping", content: fileOf(append(minimal, "listen: {a: "+sentinel+"}")...), want: at(3, "listen", yamlfile.ProblemNotScalar)},
		{name: "syntax error", content: fileOf(append(minimal, "host: "+sentinel+": x")...), want: at(3, "", yamlfile.ProblemInvalidYAML)},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, tc.content)
			got, _, err := Load(path)
			assert.Nil(t, got)
			want := tc.want
			want.Path = path
			assertFileError(t, want, err)
		})
	}
}

// at returns a FileError without a path. The test sets the path.
func at(line int, keyPath, problem string) yamlfile.FileError {
	return yamlfile.FileError{Line: line, KeyPath: keyPath, Problem: problem}
}

// assertFileError checks that err is a FileError with the fields of want, and
// that its text does not contain the sentinel.
func assertFileError(t *testing.T, want yamlfile.FileError, err error) {
	t.Helper()
	var fileErr *yamlfile.FileError
	require.ErrorAs(t, err, &fileErr)
	assert.Equal(t, want, *fileErr)
	assert.NotContains(t, err.Error(), sentinel, "an error MUST NOT contain file content")
}

// exampleKeys are the key paths that examples/config.yaml MUST set, so that the
// documentation shows each key. Add a new key of Config here. employees_file is
// not in the list, because version 0 refuses it.
var exampleKeys = []string{
	"time_zone",
	"form_files[0]",
	"trusted_people[0].id",
	"trusted_people[0].name",
	"trusted_people[0].email",
	"trusted_people[0].role",
	"security_txt_file",
	"host",
	"oidc.issuer_url",
	"oidc.client_id",
	"oidc.scopes[0]",
	"oidc.user_id_claim",
	"oidc.required_claims",
	"listen",
	"backup.directory",
	"backup.keep",
	"secret_files.idp-client-secret",
}

func TestExamples(t *testing.T) {
	testCases := []struct {
		name string
		path string
		keys []string
	}{
		{name: "production", path: "../../examples/config.yaml", keys: exampleKeys},
		{name: "development", path: "../../examples/config-dev.yaml", keys: []string{"time_zone", "form_files[0]"}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, lines, err := Load(tc.path)
			require.NoError(t, err)
			for _, key := range tc.keys {
				assert.NotZero(t, lines.Line(key), "the example MUST set %s", key)
			}
		})
	}
}

func TestLoadFormFilePaths(t *testing.T) {
	t.Run("relative to the configuration directory", func(t *testing.T) {
		// The working directory has no forms/a.yaml, so a path relative to it fails.
		root := t.TempDir()
		t.Chdir(root)
		writeAt(t, filepath.Join(root, "etc", "forms", "a.yaml"), "")
		writeAt(t, filepath.Join(root, "etc", "config.yaml"), fileOf(minimal...))

		got, _, err := Load(filepath.Join("etc", "config.yaml"))
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, "etc", "forms", "a.yaml")}, got.FormFiles)
	})

	t.Run("absolute path", func(t *testing.T) {
		form := filepath.Join(t.TempDir(), "form.yaml")
		writeAt(t, form, "")
		got, _, err := Load(writeFile(t, fileOf("time_zone: UTC", "form_files:", "- "+form)))
		require.NoError(t, err)
		assert.Equal(t, []string{form}, got.FormFiles)
	})

	// root/b/link points to root/a/real. From the link, ../form.yaml is
	// root/b/form.yaml. With the link resolved, it is root/a/form.yaml, which does not exist.
	linkedRoot := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		writeAt(t, filepath.Join(root, "a", "real", "config.yaml"), fileOf("time_zone: UTC", "form_files: [../form.yaml]"))
		writeAt(t, filepath.Join(root, "b", "form.yaml"), "")
		require.NoError(t, os.Symlink(filepath.Join(root, "a", "real"), filepath.Join(root, "b", "link")))
		return root
	}

	t.Run("symbolic link not resolved, relative path", func(t *testing.T) {
		root := linkedRoot(t)
		got, _, err := Load(filepath.Join(root, "b", "link", "config.yaml"))
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, "b", "form.yaml")}, got.FormFiles)
	})

	t.Run("absolute path stays as written", func(t *testing.T) {
		// The kernel resolves link/.. to root/a, so root/a/form.yaml is the file.
		root := linkedRoot(t)
		writeAt(t, filepath.Join(root, "a", "form.yaml"), "")
		absolute := root + "/b/link/../form.yaml"
		got, _, err := Load(writeFile(t, fileOf("time_zone: UTC", "form_files:", "- "+absolute)))
		require.NoError(t, err)
		assert.Equal(t, []string{absolute}, got.FormFiles)
	})
}

func TestLoadFormFileProblems(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(t *testing.T, form string)
		want  string
		skip  bool
	}{
		{name: "missing", setup: func(*testing.T, string) {}, want: yamlfile.ProblemFileNotExist},
		{name: "directory", setup: func(t *testing.T, form string) {
			require.NoError(t, os.MkdirAll(form, 0o700))
		}, want: yamlfile.ProblemNotRegularFile},
		{name: "unreadable", setup: func(t *testing.T, form string) {
			writeAt(t, form, "")
			require.NoError(t, os.Chmod(form, 0))
		}, want: yamlfile.ProblemPermissionDenied, skip: os.Geteuid() == 0},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("root can read each file")
			}
			dir := t.TempDir()
			tc.setup(t, filepath.Join(dir, sentinel, "form.yaml"))
			path := filepath.Join(dir, "config.yaml")
			writeAt(t, path, fileOf("time_zone: UTC", "form_files:", "- "+sentinel+"/form.yaml"))

			got, _, err := Load(path)
			assert.Nil(t, got)
			want := at(3, "form_files[0]", tc.want)
			want.Path = path
			// The sentinel is in the path of the form file, which the error MUST NOT give.
			assertFileError(t, want, err)
		})
	}
}
