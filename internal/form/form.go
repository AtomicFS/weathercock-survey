// SPDX-License-Identifier: AGPL-3.0-or-later

// Package form reads and checks a form file.
package form

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
)

// MaxTextLimit is the largest number of characters in one text answer. It
// keeps the size of the database predictable.
const MaxTextLimit = 65000

// MinK is the smallest k. Below it, results reveal single answers.
const MinK = 3

// NoAnswerID is the ID of the "prefer not to answer" option that Load adds.
// A form file cannot use it: the ID pattern refuses a leading "_".
const NoAnswerID = "_no_answer"

// noAnswerText is the text of the option with NoAnswerID.
const noAnswerText = "Prefer not to answer"

// TypeSummed is the only form type. The type field keeps the door open for others.
const TypeSummed = "summed"

// The question types.
const (
	TypeScale  = "scale"
	TypeSingle = "single"
	TypeMulti  = "multi"
	TypeText   = "text"
)

// Form is a form file. Each key has the name in its yaml tag.
type Form struct {
	// Type is the form type. Only TypeSummed exists.
	Type  string `yaml:"type"`
	Title string `yaml:"title"`
	// K is the smallest number of submissions that a period needs to publish results.
	K Int `yaml:"k"`
	// TextLimit is the default limit of text questions. Load sets it to
	// MaxTextLimit if the file does not.
	TextLimit Int        `yaml:"text_limit"`
	Questions []Question `yaml:"questions"`
}

// Question is one question. Choice questions (scale, single, multi) are
// required; text questions are optional.
type Question struct {
	ID   string `yaml:"id"`
	Type string `yaml:"type"`
	Text string `yaml:"text"`
	// IncludeNoAnswer adds the option NoAnswerID to a choice question. The default is true.
	IncludeNoAnswer *bool `yaml:"include_no_answer"`
	// TextLimit is the limit of a text question. Load sets it to the limit of
	// the form if the file does not.
	TextLimit Int      `yaml:"text_limit"`
	Options   []Option `yaml:"options"`
}

// Option is one option of a choice question.
type Option struct {
	ID   string `yaml:"id"`
	Text string `yaml:"text"`
	// Value is the number of a scale option. Other options have none. The
	// option NoAnswerID has none, so it stays out of averages.
	Value Int `yaml:"value"`
}

// Load reads and checks the form file at path. It returns all problems of the
// values at once, joined with errors.Join. Each problem has a suggestion.
//
// Load fills in the defaults: the text limits, and the option NoAnswerID as
// the last option of each choice question that includes it.
func Load(path string) (*Form, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the path comes from the admin.
	if err != nil {
		return nil, suggest.Wrap(err, "check the path in form_files and the permissions of the file")
	}
	var form Form
	if err := yaml.Load(data, &form, yaml.WithKnownFields()); err != nil {
		return nil, suggest.Wrap(fmt.Errorf("%s: %w", path, err), "fix the YAML at the line in the error")
	}
	if err := check(path, &form); err != nil {
		return nil, err
	}
	form.fillDefaults()
	return &form, nil
}

func (f *Form) fillDefaults() {
	if !f.TextLimit.Set {
		f.TextLimit = Int{Value: MaxTextLimit, Set: true}
	}
	for i := range f.Questions {
		question := &f.Questions[i]
		if question.Type == TypeText {
			if !question.TextLimit.Set {
				question.TextLimit = f.TextLimit
			}
			continue
		}
		if question.IncludeNoAnswer == nil || *question.IncludeNoAnswer {
			question.Options = append(question.Options, Option{ID: NoAnswerID, Text: noAnswerText})
		}
	}
}
