// SPDX-License-Identifier: AGPL-3.0-or-later

package form

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
)

// header is the part of a form file before the questions.
const header = "type: summed\ntitle: T\nk: 3\n"

// single is a valid single question.
const single = "- {id: q, type: single, text: Q, options: [{id: a, text: A}, {id: b, text: B}]}\n"

// formOf returns a form file with the header and the questions.
func formOf(questions ...string) string {
	return header + "questions:\n" + strings.Join(questions, "")
}

func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "form.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadExample(t *testing.T) {
	form, err := Load("../../examples/forms/example.yaml")
	require.NoError(t, err)

	var options [][]string
	var limits []int
	for _, question := range form.Questions {
		if question.Type == TypeText {
			limits = append(limits, question.TextLimit.Value)
			continue
		}
		var ids []string
		for _, option := range question.Options {
			ids = append(ids, option.ID)
		}
		options = append(options, ids)
	}
	want := [][]string{
		{"too_low", "low", "ok", "high", "too_high", NoAnswerID},
		{"dev", "ops", NoAnswerID},
		{"drill", "screwdriver", NoAnswerID},
	}
	assert.Equal(t, want, options, "the option NoAnswerID is last in each choice question")
	assert.Equal(t, []int{500, 2000}, limits, "the question limit, then the form limit")
	assert.Equal(t, -2, form.Questions[0].Options[0].Value.Value)
}

func TestLoadDefaults(t *testing.T) {
	text := "- {id: t, type: text, text: T}\n"
	testCases := []struct {
		name         string
		content      string
		wantLimit    int
		wantNoAnswer bool
	}{
		{name: "no limits", content: formOf(single, text), wantLimit: MaxTextLimit, wantNoAnswer: true},
		{name: "form limit", content: "text_limit: 5\n" + formOf(single, text), wantLimit: 5, wantNoAnswer: true},
		{name: "question limit", content: "text_limit: 5\n" + formOf(single, "- {id: t, type: text, text: T, text_limit: 7}\n"), wantLimit: 7, wantNoAnswer: true},
		{name: "without no answer", content: formOf("- {id: q, type: multi, text: Q, include_no_answer: false, options: [{id: a, text: A}, {id: b, text: B}]}\n", text), wantLimit: MaxTextLimit},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form, err := Load(write(t, tc.content))
			require.NoError(t, err)
			assert.Equal(t, tc.wantLimit, form.Questions[1].TextLimit.Value)
			options := form.Questions[0].Options
			assert.Equal(t, tc.wantNoAnswer, options[len(options)-1].ID == NoAnswerID)
		})
	}
}

func TestLoadErrors(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		// wantErrs are the sentinels that the error wraps.
		wantErrs []error
		// wantYAML means a YAML error, which stops the check before the values.
		wantYAML bool
	}{
		// Fail: YAML and integers.
		{name: "unknown key", content: formOf(single) + "extra: 1\n", wantYAML: true},
		{name: "quoted k", content: strings.Replace(formOf(single), "k: 3", `k: "3"`, 1), wantYAML: true, wantErrs: []error{ErrNotDecimal}},
		{name: "negative zero", content: formOf("- {id: s, type: scale, text: S, options: [{id: a, text: A, value: -0}, {id: b, text: B, value: 1}]}\n"), wantYAML: true, wantErrs: []error{ErrNotDecimal}},
		{name: "octal k", content: strings.Replace(formOf(single), "k: 3", "k: 010", 1), wantYAML: true, wantErrs: []error{ErrNotDecimal}},
		{name: "hex k", content: strings.Replace(formOf(single), "k: 3", "k: 0x10", 1), wantYAML: true, wantErrs: []error{ErrNotDecimal}},
		{name: "k with underscore", content: strings.Replace(formOf(single), "k: 3", "k: 1_0", 1), wantYAML: true, wantErrs: []error{ErrNotDecimal}},

		// Fail: form.
		{name: "missing type", content: strings.Replace(formOf(single), "type: summed\n", "", 1), wantErrs: []error{ErrFormType}},
		{name: "unknown type", content: strings.Replace(formOf(single), "summed", "entries", 1), wantErrs: []error{ErrFormType}},
		{name: "empty title", content: strings.Replace(formOf(single), "title: T", "title: ' '", 1), wantErrs: []error{ErrEmptyText}},
		{name: "missing k", content: strings.Replace(formOf(single), "k: 3\n", "", 1), wantErrs: []error{ErrK}},
		{name: "small k", content: strings.Replace(formOf(single), "k: 3", "k: 2", 1), wantErrs: []error{ErrK}},
		{name: "form text limit 0", content: "text_limit: 0\n" + formOf(single), wantErrs: []error{ErrTextLimit}},
		{name: "form text limit too large", content: "text_limit: 65001\n" + formOf(single), wantErrs: []error{ErrTextLimit}},
		{name: "no questions", content: header + "questions: []\n", wantErrs: []error{ErrNoQuestions}},
		{name: "missing questions", content: header, wantErrs: []error{ErrNoQuestions}},

		// Fail: questions.
		{name: "bad question ID", content: formOf(strings.Replace(single, "id: q", "id: Q", 1)), wantErrs: []error{ErrID}},
		{name: "missing question ID", content: formOf(strings.Replace(single, "id: q, ", "", 1)), wantErrs: []error{ErrID}},
		{name: "duplicate question ID", content: formOf(single, single), wantErrs: []error{ErrDuplicateID}},
		{name: "unknown question type", content: formOf(strings.Replace(single, "single", "rating", 1)), wantErrs: []error{ErrQuestionType}},
		{name: "empty question text", content: formOf(strings.Replace(single, "text: Q", "text: ''", 1)), wantErrs: []error{ErrEmptyText}},
		{name: "text limit on a choice question", content: formOf(strings.Replace(single, "text: Q", "text: Q, text_limit: 5", 1)), wantErrs: []error{ErrNotAllowed}},
		{name: "options on a text question", content: formOf("- {id: t, type: text, text: T, options: [{id: a, text: A}]}\n"), wantErrs: []error{ErrNotAllowed}},
		{name: "no answer on a text question", content: formOf("- {id: t, type: text, text: T, include_no_answer: true}\n"), wantErrs: []error{ErrNotAllowed}},
		{name: "question text limit too large", content: formOf("- {id: t, type: text, text: T, text_limit: 65001}\n"), wantErrs: []error{ErrTextLimit}},

		// Fail: options.
		{name: "one option", content: formOf("- {id: q, type: single, text: Q, options: [{id: a, text: A}]}\n"), wantErrs: []error{ErrTooFewOptions}},
		{name: "no options", content: formOf("- {id: q, type: multi, text: Q}\n"), wantErrs: []error{ErrTooFewOptions}},
		{name: "reserved option ID", content: formOf(strings.Replace(single, "id: a", "id: "+NoAnswerID, 1)), wantErrs: []error{ErrID}},
		{name: "duplicate option ID", content: formOf(strings.Replace(single, "id: b", "id: a", 1)), wantErrs: []error{ErrDuplicateID}},
		{name: "empty option text", content: formOf(strings.Replace(single, "text: A", "text: ''", 1)), wantErrs: []error{ErrEmptyText}},
		{name: "value 0 on a single option", content: formOf(strings.Replace(single, "text: A", "text: A, value: 0", 1)), wantErrs: []error{ErrNotAllowed}},
		{name: "scale option without value", content: formOf("- {id: s, type: scale, text: S, options: [{id: a, text: A, value: 0}, {id: b, text: B}]}\n"), wantErrs: []error{ErrMissingValue}},
		{name: "duplicate scale value", content: formOf("- {id: s, type: scale, text: S, options: [{id: a, text: A, value: 1}, {id: b, text: B, value: 1}]}\n"), wantErrs: []error{ErrDuplicateValue}},

		// Fail: all problems at once.
		{name: "two problems", content: "type: summed\ntitle: T\nquestions: []\n", wantErrs: []error{ErrK, ErrNoQuestions}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form, err := Load(write(t, tc.content))
			assert.Nil(t, form)
			require.Error(t, err)
			if tc.wantYAML {
				var yamlErr *yaml.LoadError
				assert.ErrorAs(t, err, &yamlErr)
			}
			for _, want := range tc.wantErrs {
				assert.ErrorIs(t, err, want)
			}
			problems := suggest.Problems(err)
			if !tc.wantYAML {
				assert.Len(t, problems, len(tc.wantErrs), "only the expected problems: %v", err)
			}
			for _, problem := range problems {
				assert.NotEqual(t, suggest.Bug, suggest.Of(problem), "no suggestion for: %v", problem)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "form.yaml"))
		assert.ErrorIs(t, err, fs.ErrNotExist)
		assert.NotEqual(t, suggest.Bug, suggest.Of(err))
	})
}
