// SPDX-License-Identifier: AGPL-3.0-or-later

// Package suggest attaches the fix for the admin to an error. The caller logs
// the fix in the "suggestion" attribute of the log line.
package suggest

import "errors"

// Bug is the suggestion for each state that only a bug can cause.
const Bug = "this should not happen, report it to the developers"

// Error is an error with the fix for the admin.
type Error struct {
	Err        error
	Suggestion string
}

// Error returns the text of the wrapped error, without the suggestion.
func (e *Error) Error() string { return e.Err.Error() }

// Unwrap returns the wrapped error, so that errors.Is finds the sentinel.
func (e *Error) Unwrap() error { return e.Err }

// Wrap returns err with the suggestion.
func Wrap(err error, suggestion string) error {
	return &Error{Err: err, Suggestion: suggestion}
}

// Problems splits the result of errors.Join into its problems, also a join
// inside a join. A join that another error wraps stays one problem, so that
// the outer text and suggestion stay. fmt.Errorf with two %w also makes an
// error that Problems splits: use one %w in an error with a suggestion.
func Problems(err error) []error {
	joined, ok := err.(interface{ Unwrap() []error }) //nolint:errorlint // Only err itself, not its chain.
	if !ok {
		return []error{err}
	}
	var problems []error
	for _, problem := range joined.Unwrap() {
		problems = append(problems, Problems(problem)...)
	}
	return problems
}

// Of returns the suggestion of err. An error without a suggestion is a bug,
// so Of returns Bug for it.
func Of(err error) string {
	var withSuggestion *Error
	if errors.As(err, &withSuggestion) {
		return withSuggestion.Suggestion
	}
	return Bug
}
