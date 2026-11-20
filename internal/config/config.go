// SPDX-License-Identifier: AGPL-3.0-or-later

// Package config reads the configuration file of the server.
//
// The types in this file are the reference of the keys. Version 0 checks the
// syntax and the types of all keys, but uses only time_zone and form_files.
package config

import (
	"fmt"
	"path/filepath"

	"github.com/AtomicFS/weathercock-survey/internal/yamlfile"
)

// The problems that this package adds to the problems of yamlfile.
// Each ends with "; " and a fixed suggestion.
const (
	ProblemFormFilesCount = "must contain exactly one form file in version 0; give exactly one path"
	ProblemNotSupportedV0 = "not supported in version 0; remove the key"
)

// ANCHOR: config

// Config is the configuration file. Each key has the name in its yaml tag.
type Config struct {
	// TimeZone is one time zone, for example "Europe/Prague". It is required in all modes.
	TimeZone string `yaml:"time_zone" required:"true"`
	// FormFiles are the paths of the form files. It is required in all modes.
	FormFiles []string `yaml:"form_files" required:"true"`
	// EmployeesFile is the path of the list of employees. Without it, there is no list.
	EmployeesFile string `yaml:"employees_file"`
	// TrustedPeople is the public list of the people with access to the server and its data.
	// It is required and not empty, except in development mode.
	TrustedPeople []TrustedPerson `yaml:"trusted_people"`
	// SecurityTxtFile is the path of a security.txt file. Without it, there is no file.
	SecurityTxtFile string `yaml:"security_txt_file"`
	// Host is the host name of the server. It is required, except in development mode.
	Host string `yaml:"host"`
	// OIDC holds the settings of the identity provider.
	OIDC OIDC `yaml:"oidc"`
	// Listen is the address and port of the plain HTTP listener, for example "127.0.0.1:8080".
	Listen string `yaml:"listen"`
	// Backup holds the settings of the backups.
	Backup Backup `yaml:"backup"`
	// SecretFiles gives a different path for a secret.
	SecretFiles SecretFiles `yaml:"secret_files"`
}

// TrustedPerson is a person or a company with access to the server, the reverse proxy, the database, or the backups.
// Each field is required in all modes.
type TrustedPerson struct {
	// ID is a stable short ID, for example "it-jane". Each ID is unique in the list.
	ID string `yaml:"id" required:"true"`
	// Name is the name of the person or the company.
	Name string `yaml:"name" required:"true"`
	// Email is a contact address. A shared mailbox is permitted.
	Email string `yaml:"email" required:"true"`
	// Role is the access of the person, for example "Server admin".
	Role string `yaml:"role" required:"true"`
}

// OIDC holds the settings of the OpenID Connect (OIDC) identity provider.
type OIDC struct {
	// IssuerURL is the issuer URL, for example "https://accounts.google.com".
	// It is required, except in development mode.
	IssuerURL string `yaml:"issuer_url"`
	// ClientID is the client ID of the identity provider. It is required, except in development mode.
	ClientID string `yaml:"client_id"`
	// Scopes are the scopes of the sign-in. They contain "openid". The default is [openid].
	Scopes []string `yaml:"scopes"`
	// UserIDClaim is the claim of the ID token that gives the user ID. The default is "sub".
	UserIDClaim string `yaml:"user_id_claim"`
	// RequiredClaims are pairs of a claim name and a value that each ID token contains.
	// This is the only mapping with free keys.
	RequiredClaims map[string]string `yaml:"required_claims"`
}

// Backup holds the settings of the backups.
type Backup struct {
	// Directory is an absolute path. The default is "backups/" in the directory of the database.
	Directory string `yaml:"directory"`
	// Keep is the number of backups that the server keeps, 1 or more. The default is 1.
	Keep int `yaml:"keep"`
}

// SecretFiles gives the path of each secret. The default of each path is
// $CREDENTIALS_DIRECTORY/<name>. The names of the secrets are an exception to
// snake_case.
type SecretFiles struct {
	// IDPClientSecret is the path of the client secret of the identity provider.
	IDPClientSecret string `yaml:"idp-client-secret"`
}

// ANCHOR_END: config

// Load reads and checks the configuration file at path. Each error is a
// *yamlfile.FileError. The Lines give the line of each key that the file sets.
//
// Load makes each form file path absolute, relative to the configuration
// directory, and checks that the file opens.
func Load(path string) (*Config, yamlfile.Lines, error) {
	// filepath.Abs fails only if the working directory is gone. Then a relative
	// path cannot be read either.
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, yamlfile.Lines{}, &yamlfile.FileError{Path: path, Problem: yamlfile.ProblemCannotRead}
	}
	config, lines, err := read(path)
	if err != nil {
		return nil, yamlfile.Lines{}, err
	}
	if fileErr := checkVersion0(config, lines); fileErr != nil {
		return nil, yamlfile.Lines{}, fileErr
	}
	if fileErr := resolveFormFiles(filepath.Dir(absolute), config, lines); fileErr != nil {
		return nil, yamlfile.Lines{}, fileErr
	}
	return config, lines, nil
}

// resolveFormFiles makes each form file path absolute, and checks that each file opens.
//
// configDir is the directory of the absolute configuration path. A relative
// path is joined to it lexically, so "../" goes up from the directory that
// the admin sees in --config. An absolute path stays as written, and the
// kernel resolves it.
func resolveFormFiles(configDir string, config *Config, lines yamlfile.Lines) *yamlfile.FileError {
	for i, form := range config.FormFiles {
		if !filepath.IsAbs(form) {
			form = filepath.Join(configDir, form)
		}
		// The error gives the line of the key, never the path: the path can be pasted data.
		if problem := yamlfile.FileProblem(form); problem != "" {
			return lines.Error(fmt.Sprintf("form_files[%d]", i), problem)
		}
		config.FormFiles[i] = form
	}
	return nil
}

// read checks the syntax and the types of all keys, and decodes the file.
func read(path string) (*Config, yamlfile.Lines, error) {
	var config Config
	lines, err := yamlfile.Load(path, &config)
	if err != nil {
		return nil, yamlfile.Lines{}, err
	}
	return &config, lines, nil
}

// checkVersion0 applies the rules of version 0 for the keys that it uses or refuses.
// A key counts as set if the file has it, also with an empty value.
func checkVersion0(config *Config, lines yamlfile.Lines) *yamlfile.FileError {
	if len(config.FormFiles) != 1 {
		return lines.Error("form_files", ProblemFormFilesCount)
	}
	if lines.Line("employees_file") != 0 {
		return lines.Error("employees_file", ProblemNotSupportedV0)
	}
	return nil
}
