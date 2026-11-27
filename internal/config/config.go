// SPDX-License-Identifier: AGPL-3.0-or-later

// Package config reads the configuration file of the server.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"

	"github.com/AtomicFS/weathercock-survey/internal/suggest"
	"github.com/AtomicFS/weathercock-survey/internal/timezone"
)

// The problems that Load finds in the values of the keys.
var (
	ErrMissingTimeZone = errors.New("time_zone: missing")
	ErrFormFilesCount  = errors.New("form_files: must contain exactly one path in version 0")
)

// Config is the configuration file. Each key has the name in its yaml tag.
// Add a key only together with the feature that uses it.
type Config struct {
	// TimeZone is one time zone, for example "Europe/Prague".
	TimeZone string `yaml:"time_zone"`
	// FormFiles are the paths of the form files. Load makes each path absolute.
	FormFiles []string `yaml:"form_files"`
}

// Load reads and checks the configuration file at path. It returns all
// problems of the values at once, joined with errors.Join. Each problem has a
// suggestion (see package suggest).
//
// Load makes each form file path absolute, relative to the configuration directory.
func Load(path string) (*Config, error) {
	// filepath.Abs fails only if the working directory is gone.
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, suggest.Wrap(err, "start the server in a directory that exists, or give an absolute --config path")
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: the path comes from the admin.
	if err != nil {
		return nil, suggest.Wrap(err, "check the --config path and the permissions of the file")
	}
	var config Config
	if err := yaml.Load(data, &config, yaml.WithKnownFields()); err != nil {
		return nil, suggest.Wrap(fmt.Errorf("%s: %w", path, err), "fix the YAML at the line in the error")
	}
	if err := config.check(path); err != nil {
		return nil, err
	}
	config.resolveFormFiles(filepath.Dir(absolute))
	return &config, nil
}

// check returns all problems of the values, each with the path of the file.
func (c *Config) check(path string) error {
	var errs []error
	if c.TimeZone == "" {
		errs = append(errs, suggest.Wrap(fmt.Errorf("%s: %w", path, ErrMissingTimeZone),
			`add the key, for example "time_zone: Europe/Prague"`))
	} else if _, err := timezone.Load(c.TimeZone); err != nil {
		errs = append(errs, suggest.Wrap(fmt.Errorf("%s: time_zone: %w", path, err),
			"use a name from the time zone data of the host, for example Europe/Prague or UTC"))
	}
	if len(c.FormFiles) != 1 {
		errs = append(errs, suggest.Wrap(fmt.Errorf("%s: %w", path, ErrFormFilesCount),
			"give exactly one path in form_files"))
	}
	return errors.Join(errs...)
}

// resolveFormFiles joins each relative form file path to configDir, the
// directory of the absolute configuration path. The join is lexical, so "../"
// goes up from the directory that the admin sees in --config.
func (c *Config) resolveFormFiles(configDir string) {
	for i, form := range c.FormFiles {
		if !filepath.IsAbs(form) {
			c.FormFiles[i] = filepath.Join(configDir, form)
		}
	}
}
