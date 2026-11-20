// SPDX-License-Identifier: AGPL-3.0-or-later

package yamlfile

// Lines gives the line of each key path that Load checked. A caller uses it to
// report a problem of a value at the line of its key.
//
// Lines holds the key paths of known keys and list items only. It never holds
// a free key, such as a claim name, because a free key is content from the file.
// A key that the file does not set has no line.
type Lines struct {
	path  string
	lines map[string]int
}

// Line returns the line of the key path, or 0 if the file does not set the key.
func (l Lines) Line(keyPath string) int {
	return l.lines[keyPath]
}

// Error returns a FileError for the key path in the file that Load read.
// The line is 0 if the file does not set the key.
//
// The key path MUST contain only the names of known keys and list indexes,
// for example "form_files[0]". It MUST NOT contain a free key name, such as a
// claim name, or any other content from the file. Error puts it in the message as it is.
func (l Lines) Error(keyPath, problem string) *FileError {
	return &FileError{Path: l.path, Line: l.Line(keyPath), KeyPath: keyPath, Problem: problem}
}
