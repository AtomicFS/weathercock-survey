// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// taskfilePath is the Taskfile, from the directory of this package.
const taskfilePath = "../../Taskfile.yml"

// taskfile holds the parts of Taskfile.yml that the build rules need.
type taskfile struct {
	Vars  map[string]any  `yaml:"vars"`
	Tasks map[string]task `yaml:"tasks"`
}

// task is one task of the Taskfile. A command is a string, or a mapping with the key "cmd".
type task struct {
	Cmds []any `yaml:"cmds"`
}

// goBuildCommands returns the "go build" commands of each task, by task name.
func (tf taskfile) goBuildCommands() map[string][]string {
	commands := make(map[string][]string)
	for name, tk := range tf.Tasks {
		for _, cmd := range tk.Cmds {
			text, ok := cmd.(string)
			if mapping, isMap := cmd.(map[string]any); isMap {
				text, ok = mapping["cmd"].(string)
			}
			if ok && strings.Contains(text, "go build") {
				commands[name] = append(commands[name], text)
			}
		}
	}
	return commands
}

// buildEnv returns the variables that a command sets before "go build".
func buildEnv(command string) []string {
	before, _, _ := strings.Cut(command, "go build")
	return strings.Fields(before)
}

func parseTaskfile(t *testing.T, data []byte) taskfile {
	t.Helper()
	var parsed taskfile
	require.NoError(t, yaml.NewDecoder(bytes.NewReader(data)).Decode(&parsed))
	return parsed
}

// buildFlags returns the build flags of the Taskfile: the variable BUILD_FLAGS.
func (tf taskfile) buildFlags(t *testing.T) []string {
	t.Helper()
	flags, ok := tf.Vars["BUILD_FLAGS"].(string)
	require.True(t, ok, "Taskfile.yml has no string variable BUILD_FLAGS")
	return strings.Fields(flags)
}

// deps returns the import paths of the server and of all its dependencies.
func deps(t *testing.T, flags ...string) []string {
	t.Helper()
	args := append(append([]string{"list", "-deps"}, flags...), ".")
	cmd := exec.Command("go", args...) //nolint:gosec // G204: the flags come from the Taskfile of this repository.
	// The same environment as the go build commands of the Taskfile: no GOFLAGS
	// (as "env -u GOFLAGS"), and GOENV=off ignores "go env -w".
	env := slices.DeleteFunc(os.Environ(), func(v string) bool { return strings.HasPrefix(v, "GOFLAGS=") })
	cmd.Env = append(env, "CGO_ENABLED=0", "GOENV=off")
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.NoError(t, err, "go list: %s", exitErr.Stderr)
	}
	require.NoError(t, err)
	return strings.Fields(string(out))
}

// TestNoTZData makes sure that the server uses the tzdata of the host, not a copy in the binary.
func TestNoTZData(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go list")
	}
	data, err := os.ReadFile(taskfilePath)
	require.NoError(t, err)
	parsed := parseTaskfile(t, data)
	flags := parsed.buildFlags(t)

	t.Run("build flags", func(t *testing.T) {
		assert.NotContains(t, deps(t, flags...), "time/tzdata")
	})

	t.Run("the check finds time/tzdata", func(t *testing.T) {
		// If Go renames the package or the tag, this fails, and the check above
		// does not pass without a reason.
		assert.Contains(t, deps(t, append(flags, "-tags", "timetzdata")...), "time/tzdata")
	})

	t.Run("no timetzdata tag in the Taskfile", func(t *testing.T) {
		assert.NotContains(t, string(data), "timetzdata")
	})

	t.Run("go build ignores GOFLAGS and go env -w", func(t *testing.T) {
		commands := parsed.goBuildCommands()
		// build has a string command, and test:reproducible a "for" mapping. Both forms must be found.
		assert.Contains(t, commands, "build", "the search does not find a string command")
		assert.Contains(t, commands, "test:reproducible", "the search does not find a mapping command")
		for name, list := range commands {
			for _, command := range list {
				env := buildEnv(command)
				// "env -u GOFLAGS" removes the variable. An empty "GOFLAGS=" does not replace the value of the shell.
				assert.Equal(t, []string{"env", "-u", "GOFLAGS"}, env[:min(3, len(env))], "task %s: the go build command does not start with env -u GOFLAGS", name)
				for _, want := range []string{"CGO_ENABLED=0", "GOENV=off"} {
					assert.Contains(t, env, want, "task %s: the go build command does not set %s", name, want)
				}
			}
		}
	})
}
