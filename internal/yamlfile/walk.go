// SPDX-License-Identifier: AGPL-3.0-or-later

package yamlfile

import (
	"reflect"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

// textStyles are the scalar styles of text that a reader sees as a string.
const textStyles = yaml.DoubleQuotedStyle | yaml.SingleQuotedStyle | yaml.LiteralStyle | yaml.FoldedStyle

// walk checks node against typ before the library decodes it. The check runs
// first, because an error of the library can contain content from the file.
// The key path names only known keys. Nothing from the file goes into an error.
// The holder line is the line of the key that holds the node, or 0 if no key does.
func (l loader) walk(node *yaml.Node, typ reflect.Type, keyPath string, holderLine int) *FileError {
	if fileErr := l.checkNode(node, keyPath); fileErr != nil {
		return fileErr
	}
	switch {
	case typ.Kind() == reflect.Struct:
		return l.walkStruct(node, typ, keyPath, holderLine)
	case typ.Kind() == reflect.Slice:
		return l.walkList(node, typ.Elem(), keyPath)
	case isFreeKeyMap(typ):
		return l.walkFreeKeys(node, keyPath)
	case typ.Kind() == reflect.Int:
		return l.checkInteger(node, keyPath)
	default:
		return l.checkString(node, keyPath)
	}
}

// checkNode refuses the YAML features that hide content or make it hard to review.
// The check for an alias is a second line of defense: its anchor fails first.
func (l loader) checkNode(node *yaml.Node, keyPath string) *FileError {
	switch {
	case node.Kind == yaml.AliasNode, node.Anchor != "":
		return l.fail(node.Line, keyPath, ProblemAnchorOrAlias)
	case node.Style&yaml.TaggedStyle != 0:
		return l.fail(node.Line, keyPath, ProblemTag)
	}
	return nil
}

// checkKey checks a key node of a mapping. The key path is the path of the mapping.
func (l loader) checkKey(key *yaml.Node, keyPath string) *FileError {
	if fileErr := l.checkNode(key, keyPath); fileErr != nil {
		return fileErr
	}
	if key.Kind == yaml.ScalarNode && key.ShortTag() == "!!merge" {
		return l.fail(key.Line, keyPath, ProblemMergeKey)
	}
	if key.Kind != yaml.ScalarNode {
		return l.fail(key.Line, keyPath, ProblemNotScalar)
	}
	return nil
}

func (l loader) walkStruct(node *yaml.Node, typ reflect.Type, keyPath string, holderLine int) *FileError {
	if node.Kind != yaml.MappingNode {
		return l.fail(node.Line, keyPath, ProblemNotMapping)
	}
	missingLine := node.Line
	if holderLine > 0 {
		missingLine = holderLine
	}
	fields := structFields(typ)
	seen := make(map[string]bool, len(fields))
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if fileErr := l.checkKey(key, keyPath); fileErr != nil {
			return fileErr
		}
		known := findField(fields, key.Value)
		if known == nil {
			return l.fail(key.Line, keyPath, ProblemUnknownKey)
		}
		fieldPath := joinKey(keyPath, known.name)
		if seen[known.name] {
			return l.fail(key.Line, fieldPath, ProblemDuplicateKey)
		}
		seen[known.name] = true
		l.lines[fieldPath] = key.Line
		if fileErr := l.walk(value, known.typ, fieldPath, key.Line); fileErr != nil {
			return fileErr
		}
	}
	for _, f := range fields {
		if f.required && !seen[f.name] {
			return l.fail(missingLine, joinKey(keyPath, f.name), ProblemMissingKey)
		}
	}
	return nil
}

func findField(fields []field, name string) *field {
	for i := range fields {
		if fields[i].name == name {
			return &fields[i]
		}
	}
	return nil
}

func joinKey(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

func (l loader) walkList(node *yaml.Node, elem reflect.Type, keyPath string) *FileError {
	if node.Kind != yaml.SequenceNode {
		return l.fail(node.Line, keyPath, ProblemNotList)
	}
	for i, item := range node.Content {
		itemPath := keyPath + "[" + strconv.Itoa(i) + "]"
		l.lines[itemPath] = item.Line
		if isNull(item) {
			return l.fail(item.Line, itemPath, ProblemEmptyListItem)
		}
		if fileErr := l.walk(item, elem, itemPath, 0); fileErr != nil {
			return fileErr
		}
	}
	return nil
}

// walkFreeKeys checks a mapping with free keys and string values. Each error
// uses the path of the mapping, because a key name is content.
func (l loader) walkFreeKeys(node *yaml.Node, keyPath string) *FileError {
	if node.Kind != yaml.MappingNode {
		return l.fail(node.Line, keyPath, ProblemNotMapping)
	}
	seen := make(map[string]bool, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if fileErr := l.checkKey(key, keyPath); fileErr != nil {
			return fileErr
		}
		if key.ShortTag() == "!!null" {
			return l.fail(key.Line, keyPath, ProblemNullKey)
		}
		if seen[key.Value] {
			return l.fail(key.Line, keyPath, ProblemDuplicateKey)
		}
		seen[key.Value] = true
		if fileErr := l.walk(value, reflect.TypeFor[string](), keyPath, key.Line); fileErr != nil {
			return fileErr
		}
	}
	return nil
}

// checkString accepts each scalar that is not null. The library keeps the text of the file.
func (l loader) checkString(node *yaml.Node, keyPath string) *FileError {
	if node.Kind != yaml.ScalarNode {
		return l.fail(node.Line, keyPath, ProblemNotScalar)
	}
	if node.ShortTag() == "!!null" {
		return l.fail(node.Line, keyPath, ProblemNotString)
	}
	return nil
}

// checkInteger accepts only a plain decimal integer without leading zeros.
// The library also reads octal, hex, and other forms, so the walker limits them.
func (l loader) checkInteger(node *yaml.Node, keyPath string) *FileError {
	if node.Kind != yaml.ScalarNode || node.Style&textStyles != 0 || !isDecimal(node.Value) {
		return l.fail(node.Line, keyPath, ProblemNotInteger)
	}
	if _, err := strconv.Atoi(node.Value); err != nil {
		return l.fail(node.Line, keyPath, ProblemNotInteger)
	}
	return nil
}

// isDecimal matches "0", or an optional "-" and digits that do not start with "0".
// It refuses "-0", because there is only one way to write zero.
func isDecimal(text string) bool {
	if text == "0" {
		return true
	}
	digits := strings.TrimPrefix(text, "-")
	if digits == "" || digits[0] == '0' {
		return false
	}
	for _, char := range digits {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
