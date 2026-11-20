# Memory

The agent owns this file. See the global agent instructions, section "Memory".

## Settled

- Errors follow the firmware-action style: package-level sentinel errors with short text, the fix in a `suggestion` attribute of the log line, `errors.Join` to report all problems at once.
- Tests: a `wantErr error` field and `assert.ErrorIs`. `assert.NotNil` is fine where any error is the right answer.
- `summed` surveys take `text` answers too (sorted alphabetically, deleted below k).
- `entries` forms wait until later. Keep the door open with a form type field, no code for it now.
- No ADRs and no long specification. `PRINCIPLES.md`, `WISHLIST.md`, and this file replace them. The mdBook setup stays for future documentation.

## Parked

- Code comments still say "the specification" (`internal/timezone/check.go`, `names.go`, `main_test.go`). Fix them in the lean audit.
- The form file loader must not open the file two times (`config.Load` checks it, then the loader opens it again).

## State

Facts for the next session:

- go-task 3.54.0: `env -u GOFLAGS` removes a shell `GOFLAGS`; an empty `GOFLAGS=` does not. `GOENV=off` wins over a shell `GOENV`.
- A change of the git `origin` URL alone does not rebuild: run `go-task build --force`.
- `build:css-update` needs the network; `docs:build-and-watch` does not stop.
- Kong exits with 80 on its own usage errors; `run` handles parse errors itself (exit 2) and catches the exit after `--help` (exit 0).
- go-yaml v4: `yaml.LoadError` has `Mark.Line` (0 = no line). `Decode` into `int` reads `012` as octal. Unknown keys are ignored by default. A plain `<<` key triggers a merge. Library error text can hold values.
- Go reads `ZONEINFO` once per process. The `go test` binary has a Go root fallback for tzdata, so tests use the injected zone loader.
- `serve --dev` returns at once until the HTTP server exists; then `TestRun` must change.
