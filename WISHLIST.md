# Wishlist

What the project should become, one line per feature. `PRINCIPLES.md` decides how.

## Must have: the prototype

- One form type for now: a `summed` survey from a YAML file, with one submission per person per period; `single` (radio buttons), `scale` (radio buttons with numeric value), `multi` (checkbox), and free-form `text` questions; questions and options with stable IDs; and `k` minimum submissions (at least 3).
- Periods by cycle (repeats every N `week`, `month`, `quarter`, `year`) from an anchor date, in a fixed time zone, with stable names (for example `2027-W10`, `2027-03`, `2027-Q1`, or `2027`).
- When a period closes, delete the sensitive records, such as who submitted. Below k, delete the answers too.
- Development mode: a fake sign-in, the server listens only on loopback, and a red banner says "DEVELOPMENT MODE - NOT ANONYMOUS - DO NOT USE".
- Storage in SQLite: choice counters and text answers, with no link to the person, no link between the answers of one submission, no time, and no insert order.
- Pages: start page, form page, results page (closed periods only, option bars, text answers in alphabetical order), and limits page (what we do not protect against).
- Each error has a suggestion on how to fix it.


## Should have: first production version

- Sign-in with the company SSO (OpenID Connect, for example Google), with required claims such as the company domain. Keep only the user ID; drop the ID token and the access token after sign-in.
- List of employees, required outside development mode: without it, one person with extra accounts can lift a period above k. It gives the eligible count and the participation rate of each period. Open question: a check "`n` is at most the eligible count" fails when people join or leave during a period.
- A separate admin tool that builds the list of employees from the SSO directory, and warns about shared accounts. It runs on the admin's PC, never in the server.
- Several forms
- Submission window (the first N days of a period)
- Optional end date of a form (to prevent form from opening new period)
- Form changes between periods
- Automatic backups after each planned end, sanitized and stripped of open periods: copy only the allowed rows into a new file (`ATTACH`, `INSERT ... SELECT`), `fsync`, rename. Never copy all and delete after, so no file with user IDs exists.
- CSV export of the results, with protection against spreadsheet formulas.
- Charts as server-side SVG, and a trend over the periods.
- Limits page lists the trusted people (who can access the server and the data, for example admins); outside development mode, refuse to start with an empty list; `security.txt`.
- Operations: run behind a TLS reverse proxy, `systemd` unit (`UMask=0077`, `StateDirectoryMode=0700`, `RestartPreventExitStatus=2`), a `validate` command, `/version`, reproducible build.
- Operations docs: leave the state directory out of host backups and snapshots (a copy of an open period, compared with another copy or with the results, can link people); turn off the access log of the proxy; never give each employee an own account at the proxy.
- Pages without JavaScript, with a strict Content Security Policy.
- Question order: `random` or `file` (random order removes order bias, but sometimes order matters)


## Later: the long-term design

- Second form type: `entries` = free-form `text` questions, for example a grievance box: no limit of submissions per person; named readers fixed at submit time (they can only become fewer, and a new reader never sees earlier periods); a sharing question (confidential or open); a retention time that can only get shorter.
- Token protocol (Privacy Pass, blind RSA): the server never links an account to its answers. Employees still trust the admin for the code that the server runs, and against forged submissions.
- Drafts that stay only in the browser (`localStorage`). This needs plain JavaScript, with no framework and no build step.
- Start page sections: open, coming soon, ended.
- A configuration change that deletes data needs an explicit confirmation from admin


## Nice to have

- Privacy screen (hide the form when the employee switches away).
- Demo page of the `entries` page (demonstrate what readers see with dummy data)
- Pop-up before the first sign-in that links to the limits page.
- Each text field had "Copy to clipboard" button.
- Button to "Delete all drafts".
- A cache of published results (closed periods with at least k submissions), never of data of an open period.


## Open questions

- Does the sign-in state survive the return from the identity provider with `SameSite=Strict` cookies, or does it need a short-lived `SameSite=Lax` cookie?
- Google: is the `sub` claim the same as the user `id` of the Directory API?
- Token protocol: size and browser support of the `privacypass-ts` and `blindrsa-ts` bundle; how tests run the browser code.
