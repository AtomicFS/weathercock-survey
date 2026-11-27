// SPDX-License-Identifier: AGPL-3.0-or-later

package form

import (
	"errors"
	"regexp"
	"strconv"

	"go.yaml.in/yaml/v4"
)

// ErrNotDecimal means that an integer is not written as a plain decimal number.
var ErrNotDecimal = errors.New("must be a plain decimal integer, for example 3 or -2")

// decimal is a plain decimal integer: no leading zero, no "+", no "_", no "-0".
var decimal = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

// Int is an integer from the form file. YAML reads 010 as 8, and also reads
// 0x10 and 1_0, so Int refuses each form except a plain, unquoted decimal.
// Set tells a missing key apart from 0.
type Int struct {
	Value int
	Set   bool
}

// UnmarshalYAML reads a plain decimal integer.
func (i *Int) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.ShortTag() != "!!int" || !decimal.MatchString(node.Value) {
		return ErrNotDecimal
	}
	value, err := strconv.Atoi(node.Value)
	if err != nil {
		// Too large for int. Each range check refuses such a number anyway.
		return ErrNotDecimal
	}
	*i = Int{Value: value, Set: true}
	return nil
}
