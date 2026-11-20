// SPDX-License-Identifier: AGPL-3.0-or-later

// Package yamlfile loads YAML files in a strict way.
//
// The Go type that the caller passes is the only schema. Its yaml tags give the
// key names, and the tag required:"true" marks a required key. An error never
// contains content from the file, because the error could reveal who wrote
// an answer or who is an employee.
package yamlfile

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strconv"

	"go.yaml.in/yaml/v4"

	"github.com/AtomicFS/weathercock-survey/internal/bug"
)

// The problems of a FileError. Each ends with "; " and a fixed suggestion,
// which never contains content from a file.
const (
	ProblemFileNotExist     = "file does not exist; check the path"
	ProblemPermissionDenied = "permission denied; let the service user read the file"
	ProblemNotRegularFile   = "not a regular file; give the path of a regular file"
	ProblemCannotRead       = "cannot read file; check the file and the disk"
	ProblemEmptyFile        = "empty file; add the content of the file"
	ProblemInvalidYAML      = "invalid YAML; fix the YAML syntax"
	ProblemManyDocuments    = "more than one YAML document; keep only one YAML document"
	ProblemAnchorOrAlias    = "anchors and aliases are not allowed; write each value in full"
	ProblemTag              = "tags are not allowed; remove the tag"
	ProblemMergeKey         = "merge key is not allowed; write each key in full"
	ProblemUnknownKey       = "unknown key; remove the key, or check its spelling"
	ProblemDuplicateKey     = "duplicate key; remove one of the keys"
	ProblemMissingKey       = "missing key; add the key"
	ProblemNotMapping       = "must be a mapping; write a mapping, or check the indentation"
	ProblemNotList          = "must be a list; write a list of \"- \" items"
	ProblemNotScalar        = "must be a scalar; write one value"
	ProblemNotString        = "must be a string; write a value after the key"
	ProblemEmptyListItem    = "empty list item; write a value, or remove the item"
	ProblemNullKey          = "key is null; write a name for the key"
	ProblemNotInteger       = "must be an integer; write a decimal integer"
	ProblemInternal         = "internal error; " + bug.Suggestion
)

// FileError describes a problem in a file. It never holds content from the file.
type FileError struct {
	// Path is the path of the file.
	Path string
	// Line is the line of the problem, or 0 if the line is not known.
	Line int
	// KeyPath names the known keys that lead to the problem, for example "form_files[0]".
	KeyPath string
	// Problem is one of the Problem constants.
	Problem string
}

// Error returns the text in the form <path>:<line>: <key path>: <problem>.
func (e *FileError) Error() string {
	text := e.Path
	if e.Line > 0 {
		text += ":" + strconv.Itoa(e.Line)
	}
	text += ": "
	if e.KeyPath != "" {
		text += e.KeyPath + ": "
	}
	return text + e.Problem
}

// Load reads the YAML file at path, checks it strictly against the type of out,
// and decodes it into out. Each error is a *FileError.
//
// The type of out MUST be a pointer to a struct, a slice, a map[string]string,
// a string, or an int. A struct holds fields of these types. Each exported
// field needs a yaml tag. Load panics for any other type, because that is a
// programming error.
func Load(path string, out any) (Lines, error) {
	typ := schemaType(out)
	loader := loader{path: path, lines: make(map[string]int)}

	data, fileErr := loader.read()
	if fileErr != nil {
		return Lines{}, fileErr
	}
	root, fileErr := loader.parse(data)
	if fileErr != nil {
		return Lines{}, fileErr
	}
	if fileErr = loader.walk(root, typ, "", 0); fileErr != nil {
		return Lines{}, fileErr
	}
	// The walker accepts only what the library can decode. A failure here is a
	// bug, for example in a custom unmarshaler. The library text could hold content.
	if err := root.Decode(out); err != nil {
		return Lines{}, loader.fail(0, "", ProblemInternal)
	}
	return Lines{path: path, lines: loader.lines}, nil
}

// loader holds the path of the file that it checks, and the lines that it finds.
type loader struct {
	path  string
	lines map[string]int
}

func (l loader) fail(line int, keyPath, problem string) *FileError {
	return &FileError{Path: l.path, Line: line, KeyPath: keyPath, Problem: problem}
}

func (l loader) read() ([]byte, *FileError) {
	if problem := FileProblem(l.path); problem != "" {
		return nil, l.fail(0, "", problem)
	}
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, l.fail(0, "", ReadProblem(err))
	}
	return data, nil
}

// FileProblem returns "" if path is a regular file that opens for reading.
// Else it returns ProblemFileNotExist, ProblemPermissionDenied,
// ProblemNotRegularFile, or ProblemCannotRead. It never returns the text of
// the operating system error, because the text holds the path.
func FileProblem(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ReadProblem(err)
	}
	if !info.Mode().IsRegular() {
		return ProblemNotRegularFile
	}
	// Opening the path that the admin configured is the purpose. Clean would
	// change which file "link/.." names, so the check would differ from the read.
	file, err := os.Open(path) //nolint:gosec // G304: the path comes from the admin.
	if err != nil {
		return ReadProblem(err)
	}
	if err := file.Close(); err != nil {
		return ProblemCannotRead
	}
	return ""
}

// ReadProblem gives the kind of a read error, without the text of the error:
// ProblemFileNotExist, ProblemPermissionDenied, or ProblemCannotRead.
func ReadProblem(err error) string {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ProblemFileNotExist
	case errors.Is(err, fs.ErrPermission):
		return ProblemPermissionDenied
	default:
		return ProblemCannotRead
	}
}

// parse returns the root node of the only document in data.
func (l loader) parse(data []byte) (*yaml.Node, *FileError) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, l.fail(0, "", ProblemEmptyFile)
		}
		return nil, l.syntaxError(err)
	}

	var second yaml.Node
	switch err := decoder.Decode(&second); {
	case errors.Is(err, io.EOF):
	case err != nil:
		return nil, l.syntaxError(err)
	default:
		return nil, l.fail(second.Line, "", ProblemManyDocuments)
	}

	root := &document
	if document.Kind == yaml.DocumentNode && len(document.Content) == 1 {
		root = document.Content[0]
	}
	if isNull(root) {
		return nil, l.fail(0, "", ProblemEmptyFile)
	}
	return root, nil
}

// syntaxError keeps only the line of a library error.
func (l loader) syntaxError(err error) *FileError {
	line := 0
	var loadErr *yaml.LoadError
	if errors.As(err, &loadErr) {
		line = loadErr.Mark.Line
	}
	return l.fail(line, "", ProblemInvalidYAML)
}

// isNull reports whether the node is a plain null scalar, for example a file with only "---".
func isNull(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Style&yaml.TaggedStyle == 0 && node.ShortTag() == "!!null"
}

// schemaType returns the type that out points to. It panics if out is not a
// pointer, or if the type is not supported.
func schemaType(out any) reflect.Type {
	typ := reflect.TypeOf(out)
	if typ == nil || typ.Kind() != reflect.Pointer || reflect.ValueOf(out).IsNil() {
		panic("yamlfile: out must be a non-nil pointer")
	}
	typ = typ.Elem()
	checkSupported(typ)
	return typ
}
