// SPDX-License-Identifier: AGPL-3.0-or-later

package timezone

// Warning is a warning to log at start, with the fix for the admin.
// Neither field contains a value from the environment.
type Warning struct {
	Message    string
	Suggestion string
}

// EnvWarnings returns a warning for ZONEINFO and for GOROOT, if the variable is
// set and not empty. Then the time zone data can come from a place other than
// the system.
func EnvWarnings(getenv func(string) string) []Warning {
	var warnings []Warning
	if getenv("ZONEINFO") != "" {
		warnings = append(warnings, Warning{
			Message:    "ZONEINFO is set: the time zone data can come from the location in ZONEINFO, not from the system",
			Suggestion: "Unset ZONEINFO in the environment of the server, unless you need it",
		})
	}
	if getenv("GOROOT") != "" {
		warnings = append(warnings, Warning{
			Message:    "GOROOT is set: the time zone data can come from the Go installation, not from the system",
			Suggestion: "Unset GOROOT in the environment of the server, unless you need it",
		})
	}
	return warnings
}
