# TODO

The maintainer owns this file. Agents propose changes in their reports.

## In progress

## Next: the prototype (must have, see `WISHLIST.md`)

- [ ] Lean audit of the existing code: errors in the firmware-action style (sentinels, `suggestion` in the log, `errors.Join`), and a smaller YAML loader if the strict walker is more than we need
- [ ] Form file: one `summed` form with `scale`, `single`, `multi`, and `text` questions; stable IDs for questions and options; k of at least 3
- [ ] Periods: every N `week`, `month`, `quarter`, or `year` from an anchor date, in the configured time zone; stable period names
- [ ] Storage: SQLite with choice counters and text answers; no link to the person, no link between answers, no time, no insert order; `UNIQUE(user, period)` on the record of who submitted; the record and the counters change in one transaction
- [ ] Database file: `umask 077` at start; if the file permissions are wider than 0600, fix them and log a warning
- [ ] Period snapshot: copy the period settings and the form into the database when a period opens; config changes apply only to later periods
- [ ] Planned end: close the period, delete who submitted; below k, delete the answers too; before the delete, store the submission count `n` of the period; the results page warns when the counts do not match `n` (each required `single` or `scale` question sums to `n`, `multi` and `text` counts are at most `n`, `n` is at most the eligible count); publish results only when k is at least 3, also if the config says less
- [ ] Results hash chain: at the planned end, hash the canonical results of the period (counts and sorted text answers) with the snapshot and the hash of the previous period; show the hash on the results page, with the text "Save this hash. If it changes later, the results changed."
- [ ] Development mode: HTTP server on loopback only, fake sign-in, red banner, clean shutdown; session cookie with `HttpOnly`, `Secure`, and a short lifetime, and the session ends after a submission; protection against cross-site request forgery (`SameSite=Strict` cookies, `Origin` check, for example `net/http.CrossOriginProtection`)
- [ ] Submission: one submission per person and period; accept only known question and option IDs, reject unknown fields, limit the length of text answers, render only through `html/template`
- [ ] Pages (templates and CSS built in with `//go:embed`): start page, form page, results page (closed periods, option bars, text answers sorted), limits page
- [ ] End-to-end test: sign in, submit, a second submit fails, planned end, results (or "fewer than k")
- [ ] Checks: `go mod verify` in `go-task check`; `govulncheck` in its own task and in continuous integration

## Done

- [x] Configuration file: `--config`, strict YAML loader, `time_zone` and `form_files`
- [x] Time zone: canonical names from `zone1970.tab`, first-start check, `ZONEINFO`/`GOROOT` warnings
- [x] Build: no `time/tzdata` in the binary
- [x] `--help` with the source code URL and the license
