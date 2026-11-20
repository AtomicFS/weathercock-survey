// SPDX-License-Identifier: AGPL-3.0-or-later

package timezone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvWarnings(t *testing.T) {
	testCases := []struct {
		name string
		env  map[string]string
		want []string // the variables that get a warning
	}{
		// No warning.
		{name: "not set", env: map[string]string{}},
		{name: "empty", env: map[string]string{"ZONEINFO": "", "GOROOT": ""}},

		// A warning for each variable that is set.
		{name: "ZONEINFO", env: map[string]string{"ZONEINFO": "/" + sentinel}, want: []string{"ZONEINFO"}},
		{name: "GOROOT", env: map[string]string{"GOROOT": "/" + sentinel}, want: []string{"GOROOT"}},
		{name: "both", env: map[string]string{"ZONEINFO": "/" + sentinel, "GOROOT": "/" + sentinel}, want: []string{"ZONEINFO", "GOROOT"}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			warnings := EnvWarnings(func(key string) string { return tc.env[key] })
			require.Len(t, warnings, len(tc.want))
			for i, warning := range warnings {
				assert.Contains(t, warning.Message, tc.want[i])
				assert.NotEmpty(t, warning.Suggestion)
				assert.NotContains(t, warning.Message, sentinel, "a warning MUST NOT contain the value of the variable")
				assert.NotContains(t, warning.Suggestion, sentinel, "a suggestion MUST NOT contain the value of the variable")
			}
		})
	}
}
