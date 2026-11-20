# TODO

The maintainer owns this file. Agents propose changes in their reports.

## In progress

## Next: the prototype (must have, see `WISHLIST.md`)

- [ ] Lean audit of the existing code: errors in the firmware-action style (sentinels, `suggestion` in the log, `errors.Join`), and a smaller YAML loader if the strict walker is more than we need
- [ ] Form file: one `summed` form with `scale`, `single`, `multi`, and `text` questions; stable IDs for questions and options; k of at least 3
- [ ] Periods: every N `week`, `month`, `quarter`, or `year` from an anchor date, in the configured time zone; stable period names
- [ ] Storage: SQLite with choice counters and text answers; no link to the person, no link between answers, no time, no insert order
- [ ] Planned end: close the period, delete who submitted; below k, delete the answers too
- [ ] Development mode: HTTP server on loopback only, fake sign-in, red banner, clean shutdown
- [ ] Submission: one submission per person and period
- [ ] Pages: start page, form page, results page (closed periods, option bars, text answers sorted), limits page
- [ ] End-to-end test: sign in, submit, a second submit fails, planned end, results (or "fewer than k")

## Done

- [x] Configuration file: `--config`, strict YAML loader, `time_zone` and `form_files`
- [x] Time zone: canonical names from `zone1970.tab`, first-start check, `ZONEINFO`/`GOROOT` warnings
- [x] Build: no `time/tzdata` in the binary
- [x] `--help` with the source code URL and the license
