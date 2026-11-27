// SPDX-License-Identifier: AGPL-3.0-or-later

package form

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
)

// The problems that Load finds in the values of a form file.
var (
	ErrFormType       = errors.New(`must be "summed"`)
	ErrK              = fmt.Errorf("must be at least %d", MinK)
	ErrTextLimit      = fmt.Errorf("must be from 1 to %d", MaxTextLimit)
	ErrNoQuestions    = errors.New("must contain at least one question")
	ErrQuestionType   = errors.New(`must be "scale", "single", "multi", or "text"`)
	ErrID             = errors.New("must match " + idPattern)
	ErrDuplicateID    = errors.New("duplicate ID")
	ErrEmptyText      = errors.New("must not be empty")
	ErrTooFewOptions  = errors.New("must contain at least 2 options")
	ErrNotAllowed     = errors.New("not allowed for this question type")
	ErrMissingValue   = errors.New("missing: each scale option needs a value")
	ErrDuplicateValue = errors.New("duplicate value")
)

// idPattern is the pattern of question and option IDs. IDs are stable: the
// database and the results use them.
const idPattern = `^[a-z0-9][a-z0-9_-]{0,63}$`

var validID = regexp.MustCompile(idPattern)

// checker collects the problems of one form file.
type checker struct {
	path     string
	problems []error
}

// add records a problem at where, for example "questions[1] (team).options[0].id".
func (c *checker) add(where string, err error, suggestion string) {
	c.problems = append(c.problems, suggest.Wrap(fmt.Errorf("%s: %s: %w", c.path, where, err), suggestion))
}

// check returns all problems of the values, joined with errors.Join.
func check(path string, form *Form) error {
	c := &checker{path: path}
	if form.Type != TypeSummed {
		c.add("type", ErrFormType, `write "type: summed"`)
	}
	c.text("title", form.Title)
	if form.K.Value < MinK {
		c.add("k", ErrK, fmt.Sprintf("set k to %d or more", MinK))
	}
	c.textLimit("text_limit", form.TextLimit)
	if len(form.Questions) == 0 {
		c.add("questions", ErrNoQuestions, "add a question")
	}
	ids := make(map[string]bool)
	for i, question := range form.Questions {
		where := fmt.Sprintf("questions[%d]", i)
		if validID.MatchString(question.ID) {
			where += " (" + question.ID + ")"
		}
		c.id(where+".id", question.ID, ids)
		c.text(where+".text", question.Text)
		c.question(where, question)
	}
	return errors.Join(c.problems...)
}

// question checks the keys that depend on the question type.
func (c *checker) question(where string, question Question) {
	switch question.Type {
	case TypeText:
		if question.IncludeNoAnswer != nil {
			c.add(where+".include_no_answer", ErrNotAllowed, "remove the key: a text question is optional")
		}
		if len(question.Options) > 0 {
			c.add(where+".options", ErrNotAllowed, "remove the options, or change the question type")
		}
		c.textLimit(where+".text_limit", question.TextLimit)
	case TypeScale, TypeSingle, TypeMulti:
		if question.TextLimit.Set {
			c.add(where+".text_limit", ErrNotAllowed, "remove the key: only a text question has a text limit")
		}
		c.options(where, question)
	default:
		c.add(where+".type", ErrQuestionType, "use one of the question types")
	}
}

// options checks the options of a choice question.
func (c *checker) options(where string, question Question) {
	if len(question.Options) < 2 {
		c.add(where+".options", ErrTooFewOptions, "add options, or change the question type to text")
	}
	ids := make(map[string]bool)
	values := make(map[int]bool)
	for i, option := range question.Options {
		at := fmt.Sprintf("%s.options[%d]", where, i)
		c.id(at+".id", option.ID, ids)
		c.text(at+".text", option.Text)
		switch {
		case question.Type != TypeScale && option.Value.Set:
			c.add(at+".value", ErrNotAllowed, "remove the value: only a scale option has a value")
		case question.Type == TypeScale && !option.Value.Set:
			c.add(at+".value", ErrMissingValue, "add a value, for example 1")
		case question.Type == TypeScale && values[option.Value.Value]:
			c.add(at+".value", ErrDuplicateValue, "give each option of the scale its own value")
		}
		if option.Value.Set {
			values[option.Value.Value] = true
		}
	}
}

// id checks the pattern of id, and that seen does not hold it yet.
func (c *checker) id(where, id string, seen map[string]bool) {
	if !validID.MatchString(id) {
		c.add(where, ErrID, "use lowercase letters, digits, \"_\", and \"-\"; start with a letter or a digit")
		return
	}
	if seen[id] {
		c.add(where, ErrDuplicateID, "give each question, and each option of a question, its own ID")
	}
	seen[id] = true
}

func (c *checker) text(where, text string) {
	if strings.TrimSpace(text) == "" {
		c.add(where, ErrEmptyText, "write the text")
	}
}

// textLimit checks a text limit, if the file sets it.
func (c *checker) textLimit(where string, limit Int) {
	if limit.Set && (limit.Value < 1 || limit.Value > MaxTextLimit) {
		c.add(where, ErrTextLimit, fmt.Sprintf("set a limit from 1 to %d, or remove the key", MaxTextLimit))
	}
}
