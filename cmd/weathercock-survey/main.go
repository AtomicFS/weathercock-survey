// SPDX-License-Identifier: AGPL-3.0-or-later

// Package main is the command line of the weathercock-survey server.
package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/alecthomas/kong"

	"github.com/AtomicFS/weathercock-survey/internal/config"
	"github.com/AtomicFS/weathercock-survey/internal/form"
	"github.com/AtomicFS/weathercock-survey/internal/logging"
	"github.com/AtomicFS/weathercock-survey/internal/suggest"
	"github.com/AtomicFS/weathercock-survey/internal/timezone"
)

// sourceURL is the URL of the source code of the running version.
// The build sets it from the git remote "origin" with -ldflags "-X main.sourceURL=...".
// It is the only mutable global variable that the conventions permit.
var sourceURL = "https://github.com/AtomicFS/weathercock-survey"

// license is the license of the project.
const license = "AGPL-3.0-or-later"

// exitRefuse is the exit code when the server refuses to start, or the command line is not valid.
// The systemd unit of v1 uses it in RestartPreventExitStatus.
const exitRefuse = 2

var (
	errDevRequired = errors.New("version 0 runs only in development mode")
	errEmptyConfig = errors.New("--config: the path is empty")
)

type cli struct {
	Serve serveCmd `cmd:"" help:"Run the server."`
}

// configOption is the option --config. Each command that reads the configuration embeds it.
type configOption struct {
	// Config is a plain string, not the kong type "path": errors show the path as the admin typed it.
	Config string `help:"Path of the configuration file." default:"/etc/weathercock-survey/config.yaml"`
}

// load reads and checks the configuration file.
func (o configOption) load() (*config.Config, error) {
	if o.Config == "" {
		return nil, suggest.Wrap(errEmptyConfig, "give the path of the configuration file")
	}
	return config.Load(o.Config)
}

type serveCmd struct {
	configOption `embed:""`

	Dev bool `help:"Start development mode. Not anonymous: do not use with real employees."`
}

// Run starts the server. The logger shows times in UTC, because the time zone
// is not loaded yet.
func (c *serveCmd) Run(logger *slog.Logger) error {
	if !c.Dev {
		return suggest.Wrap(errDevRequired, `use "serve --dev"`)
	}
	for _, warning := range timezone.EnvWarnings(os.Getenv) {
		logger.Warn(warning.Message, slog.String("suggestion", warning.Suggestion))
	}
	cfg, err := c.load()
	if err != nil {
		return err
	}
	_, err = form.Load(cfg.FormFiles[0])
	return err
}

// logErrors writes one log line for each problem in err, with its suggestion.
func logErrors(logger *slog.Logger, err error) {
	for _, problem := range suggest.Problems(err) {
		logger.Error("command failed", slog.String("error", problem.Error()), slog.String("suggestion", suggest.Of(problem)))
	}
}

// printHelp prints the help of kong, and a footer with the source code URL and the license.
func printHelp(options kong.HelpOptions, ctx *kong.Context) error {
	if err := kong.DefaultHelpPrinter(options, ctx); err != nil {
		return err
	}
	_, err := fmt.Fprintf(ctx.Stdout, "\nSource code: %s\nLicense: %s\n", sourceURL, license)
	return err
}

// exitRequest carries the exit code of kong, for example after --help, to run.
type exitRequest int

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// newParser returns the parser of the command line, which fills in parsed.
func newParser(parsed *cli, stdout, stderr io.Writer) *kong.Kong {
	parser, err := kong.New(
		parsed,
		kong.Name("weathercock-survey"),
		kong.Description("Server for anonymous employee surveys."),
		kong.Help(printHelp),
		kong.Writers(stdout, stderr),
		// Kong calls exit after --help. A panic stops the parse, and the recover in
		// run returns the code, so that main keeps the only call of os.Exit.
		kong.Exit(func(c int) { panic(exitRequest(c)) }),
	)
	if err != nil {
		panic(err) // Only a programming error in the cli struct gets here.
	}
	return parser
}

func run(args []string, stdout, stderr io.Writer) (code int) {
	parser := newParser(&cli{}, stdout, stderr)

	defer func() {
		if r := recover(); r != nil {
			exit, ok := r.(exitRequest)
			if !ok {
				panic(r)
			}
			code = int(exit)
		}
	}()

	ctx, err := parser.Parse(args)
	if err != nil {
		parser.Errorf("%s", err)
		return exitRefuse
	}
	logger := slog.New(logging.NewHandler(stderr, time.UTC))
	if err := ctx.Run(logger); err != nil {
		logErrors(logger, err)
		return exitRefuse
	}
	return 0
}
