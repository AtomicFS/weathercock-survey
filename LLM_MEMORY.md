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
- Periods (`internal/period`): `Unit` is a `uint8` enum with an invalid zero value and only `String` and `UnmarshalText`; `Cycle` has unexported fields and only `New` makes one; the anchor must be the start of a unit, and the error suggests the unit start before and the one after; `every` is 1..100; no period before the anchor; `period` does not import YAML, so the form loader calls `period.New`.
- Fast and slow tests: `go-task check` runs `go test -short`; `go-task test:all` runs all tests. The maintainer keeps the split, because slow tests will come. The agent runs both before each commit (`AGENTS.md` step 2).
- Where it is cheap, test against a plain definition over all inputs (for example `TestAllZones`: each zone change of each host zone), not only picked cases. Keep such tests short.
- Form lifecycle (decided 2026-11-28): a form's `unit`, `every`, `anchor`, and time zone are frozen once its first period opens; to change them, the admin uses a new form ID. The server stores them in a `forms` row and refuses to start on a change, and the error names the old values. So periods of a form never overlap, and periods are keyed by `(form_id, name)`.
- Form `end` (optional): a period boundary (the error suggests the boundary before and after) or `asap` (the planned end of the open period; before the anchor, the form ends without running). A new period opens only if it fits fully before the end; else the form ends for good. Refuse an end in the past or inside the open period, and any change after the form ended. Check `end` only when it differs from the stored value, because a valid date becomes past while it stays in the file. Store the accepted value and the date that `asap` resolved to, so errors can name them.
- No ADRs and no long specification. `PRINCIPLES.md`, `WISHLIST.md`, and this file replace them. The mdBook setup stays for future documentation.

## Parked

- Snapshot task: `form.Int` has no marshaler, and a loaded `Form` holds `_no_answer`, which `check` refuses. Store the raw form file bytes in the snapshot and split `form.Load` into a `Parse(data, name)`, so a snapshot loads through the same checks.
- Planned ends at start run before the config is read, so the database path must not come from the config file.
- When CI comes, it must run `go-task test:all`, not only `go-task check`.
- Submission task: the `http.MaxBytesReader` cap must come from the form's text limits (form encoding can send about 12 bytes per character), not a fixed constant, or a valid long answer gets cut and cannot be refilled.

## State

Facts for the next session:

- Next work, agreed with the maintainer on 2026-11-28: the "Form lifecycle" and "Storage" tasks in `TODO.md` depend on each other. Do them in three steps:
  1. Form file: `id`, `unit`, `every`, `anchor`, and `end` (period boundary or `asap`), with the checks that need only the file. No database.
  2. Storage: the tables, with the `forms` row (frozen cycle, time zone, accepted `end`, the date `asap` resolved to, ended state).
  3. Lifecycle at start: compare the file with the `forms` row, refuse forbidden changes with the old values, and open a period only if it fits before the end.

- go-task 3.54.0: `env -u GOFLAGS` removes a shell `GOFLAGS`; an empty `GOFLAGS=` does not. `GOENV=off` wins over a shell `GOENV`.
- A change of the git `origin` URL alone does not rebuild: run `go-task build --force`.
- `build:css-update` needs the network; `docs:build-and-watch` does not stop.
- Kong exits with 80 on its own usage errors; `run` handles parse errors itself (exit 2) and catches the exit after `--help` (exit 0).
- go-yaml v4: `yaml.Load` refuses an empty file and a second document; unknown keys need `WithKnownFields()`. Errors are `*yaml.LoadErrors`; `errors.As` to `*yaml.LoadError` works. Library error text can hold values.
- Go reads `ZONEINFO` once per process. The `go test` binary has a Go root fallback for tzdata, so `time.LoadLocation` works in tests without host tzdata.
- `time.Date` in a gap or at a doubled midnight picks the earlier zone west of UTC and the later zone east of UTC. Test both sides: Havana/Santiago and Beirut 2025-03-30/Amman 2021-10-29.
- Go `ZoneBounds` after the last change in the tzdata (rule years): away from a change it gives the year as bounds, and on 31 December of a leap year it returns `t` itself as the end (`time/zoneinfo.go` `tzset`). A loop over zones must step past it.
- `-race` makes the all-zones test about 7 times slower; `t.Parallel` per zone brings it back to about 3 s.
- Commit trailers: put both AI trailers in one `-m`, with a newline between them, and use `-s`: `git commit -s -m "<subject>" -m "<body>" -m "AI-Generated: true` newline `AI-Model: <model>"`. Then `Signed-off-by` comes last in the same block. One `-m` per trailer puts a blank line between them, and git does not read `AI-Generated` as a trailer (this broke the commits from `e6319b9` to `119f630`; they stay as they are). `--trailer` with `-s` puts `Signed-off-by` first, which is also wrong.
- `serve --dev` returns at once until the HTTP server exists; then `TestRun` must change.
