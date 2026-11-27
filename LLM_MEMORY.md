# Memory

The agent owns this file. See the global agent instructions, section "Memory".

## Settled

- When a task is done, the agent moves its `TODO.md` item to "Done" itself, in the task's commit. Other `TODO.md` changes are still proposals.
- Errors follow the firmware-action style: package-level sentinel errors with short text, the fix in a `suggestion` attribute of the log line, `errors.Join` to report all problems at once.
- The fix travels with the error (`suggest.Wrap`); only `main` logs, one line per joined problem. Packages do not log errors.
- Paths and OS error text may appear in errors (`PRINCIPLES.md` allows them); wrap with `%w`.
- YAML: plain `yaml.Load(..., yaml.WithKnownFields())`, library errors pass through. Config and form files hold nothing secret. The employees loader must drop the library text (keep path and line) and get a sentinel test: `PRINCIPLES.md` forbids logging the list of employees.
- Config does not check that the form file opens; the form loader reports it, with the path.
- No `go-playground/validator`: six more modules, and the hard form rules need custom code anyway. Plain `if` checks.
- No `zone1970.tab` check: it refused valid names such as `Europe/Oslo`, and UTC planned ends make a stored zone name unneeded. `time.LoadLocation`, refuse `""` and `"Local"`.
- Config holds only the keys that a feature uses today. Add a key with its feature.
- Tests: a `wantErr error` field and `assert.ErrorIs`. `assert.NotNil` is fine where any error is the right answer.
- `summed` surveys take `text` answers too (sorted alphabetically, deleted below k).
- `entries` forms wait until later. Keep the door open with a form type field, no code for it now.
- Each period has a snapshot of its settings and form in the database; config changes apply only to later periods. This stops changes to an open period.
- Database file permissions wider than 0600: fix them and log a warning, do not refuse to start.
- `govulncheck` needs the network, so it runs in its own task and in CI, not in `go-task check`.
- No receipts for employees: a receipt lets a manager demand proof of an answer.
- Choice questions are required, with nothing selected by default; text questions are optional. This makes the `n` check exact.
- "Prefer not to answer": `form.Load` appends option `_no_answer` last to each choice question, unless `include_no_answer: false` (default true; the maintainer wants the option). Server-made IDs start with `_`, which the ID pattern refuses, so they never collide with admin IDs. In `multi`, `_no_answer` must be the only selected option.
- Form schema: uniform `options` lists; scale options carry a `value` (unique, any order, may be negative). `text_limit` on the form (default) and on text questions (override), each 1..65000 (`form.MaxTextLimit`); `Load` stores the final limit on each text question.
- Form integers use `form.Int`: plain decimal only (go-yaml reads `010` as 8, and reads `0x10`, `1_0`), and it tells a missing key from 0.
- Choice questions need at least 2 admin-written options; a form needs at least one question.
- The list of employees is required outside development mode: without it, extra accounts lift a period above k. Publishing the eligible count stays optional.
- No `/version` file hashes: the server reports its own hashes, so they prove nothing. No trusted-people dates.
- Never tell employees to save a copy of their answers (a copy on a monitored laptop links them). State the facts; the employee decides.
- A raised k does not apply to an open period: the admin could use it to delete the answers of that period.
- Below k, delete all data of the period, `n` included; keep only a record that the period ended with fewer than k submissions.
- Ideas from the old `shitshow` branch were reviewed on 2026-11-27; the rejected ones: single-instance `flock`, dev mode stored in the database, file-mode checks for all files, four-attacker table.
- 360-degree feedback lives in `IDEA_INCUBATOR.md`, not `WISHLIST.md`: it can harm people and needs a harm review before any build.
- Templates: `html/template`, not `templ`. About four pages and no JavaScript: no code generator and 14 fewer indirect modules outweigh compile-time types. Tests of each page state make up for the weak typing.
- No ADRs and no long specification. `PRINCIPLES.md`, `WISHLIST.md`, and this file replace them. The mdBook setup stays for future documentation.

## Parked

- Snapshot task: `form.Int` has no marshaler, and a loaded `Form` holds `_no_answer`, which `check` refuses. Store the raw form file bytes in the snapshot and split `form.Load` into a `Parse(data, name)`, so a snapshot loads through the same checks.
- Planned ends at start run before the config is read, so the database path must not come from the config file.
- Submission task: the `http.MaxBytesReader` cap must come from the form's text limits (form encoding can send about 12 bytes per character), not a fixed constant, or a valid long answer gets cut and cannot be refilled.

## State

Facts for the next session:

- go-task 3.54.0: `env -u GOFLAGS` removes a shell `GOFLAGS`; an empty `GOFLAGS=` does not. `GOENV=off` wins over a shell `GOENV`.
- A change of the git `origin` URL alone does not rebuild: run `go-task build --force`.
- `build:css-update` needs the network; `docs:build-and-watch` does not stop.
- Kong exits with 80 on its own usage errors; `run` handles parse errors itself (exit 2) and catches the exit after `--help` (exit 0).
- go-yaml v4: `yaml.Load` refuses an empty file and a second document; unknown keys need `WithKnownFields()`. Errors are `*yaml.LoadErrors`; `errors.As` to `*yaml.LoadError` works. Library error text can hold values.
- Go reads `ZONEINFO` once per process. The `go test` binary has a Go root fallback for tzdata, so `time.LoadLocation` works in tests without host tzdata.
- `serve --dev` returns at once until the HTTP server exists; then `TestRun` must change.
