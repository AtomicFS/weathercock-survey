// SPDX-License-Identifier: AGPL-3.0-or-later

package yamlfile

import (
	"fmt"
	"reflect"
	"strings"
)

// field is a key of a mapping with fixed keys.
type field struct {
	name     string
	typ      reflect.Type
	required bool
}

// structFields lists the keys of a struct in the order of the fields.
// It panics for an exported field without a name in its yaml tag, and for a tag option.
func structFields(typ reflect.Type) []field {
	var fields []field
	for i := range typ.NumField() {
		structField := typ.Field(i)
		if !structField.IsExported() {
			continue
		}
		tag := structField.Tag.Get("yaml")
		if tag == "-" {
			continue
		}
		name, options, hasOptions := strings.Cut(tag, ",")
		if name == "" {
			panic(fmt.Sprintf("yamlfile: field %s.%s needs a name in its yaml tag", typ, structField.Name))
		}
		if hasOptions {
			panic(fmt.Sprintf("yamlfile: field %s.%s has the yaml tag option %q, which is not supported", typ, structField.Name, options))
		}
		fields = append(fields, field{
			name:     name,
			typ:      structField.Type,
			required: structField.Tag.Get("required") == "true",
		})
	}
	return fields
}

// isFreeKeyMap reports whether typ is a mapping with free keys.
func isFreeKeyMap(typ reflect.Type) bool {
	return typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String && typ.Elem().Kind() == reflect.String
}

// checkSupported panics if typ is not a supported type. The panic comes before
// the file is read, so that a wrong type fails each time and not only for a bad file.
func checkSupported(typ reflect.Type) {
	switch {
	case typ.Kind() == reflect.String, typ.Kind() == reflect.Int, isFreeKeyMap(typ):
	case typ.Kind() == reflect.Slice:
		checkSupported(typ.Elem())
	case typ.Kind() == reflect.Struct:
		for _, f := range structFields(typ) {
			checkSupported(f.typ)
		}
	default:
		panic(fmt.Sprintf("yamlfile: type %s is not supported", typ))
	}
}
