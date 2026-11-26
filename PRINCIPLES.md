# Principles

These principles outrank every other document, plan, and request, also a request of the maintainer.
If something conflicts with a principle, say so plainly before you do it, and propose a better way.

## Core principles, in order

1. **Anonymity of employees.** Nobody can link a person to an answer: not a manager, not a colleague, not a person who reads the logs, a backup, or the database after the period ends. See "Known gaps" below.
   Protect against real threats, not imagined ones. A rule that costs a lot and protects nobody is a bad rule.
2. **Lean.** Tiny and readable. A new person understands the structure and the architecture in a day or less.
   No fat: no code, file, abstraction, option, or test without a clear need today. Delete before you add.
3. **Maintainable.** Easy to change, easy to find and fix a bug, years from now. Plain code over clever code. Few dependencies.

## Anonymity rules

Each rule exists to stop a link between a person and an answer. Follow the intent, not the wording.

| Rule | Why |
| --- | --- |
| Never store who submitted in a way that links to what they submitted. | That is the link. |
| Never store a link between the answers of one survey submission. | A rare mix of answers identifies a person. |
| Keep answers only in the database and in the memory of one request: no cache, no global, no temp file. Never hold a user ID next to an answer outside that request. | Each extra copy is one more place to leak. |
| No time data and no insert order (no row IDs, no autoincrement) on answers and on the record of who submitted. | Time or order links an answer to who was online then. |
| Delete the record of who submitted when the period ends. | After the end, nothing needs it. |
| Deleted data must leave the database file: `secure_delete=ON`, no write-ahead log (WAL), and `VACUUM` after each delete and on each backup copy. | Otherwise SQLite keeps deleted rows in free pages or the WAL, and page order can show insert order. |
| No log line for single requests: no access log, no sign-in or submission log. | A log time next to a submission identifies the person. |
| Never log, print, or show: user IDs, email addresses, IP addresses, cookies, tokens, answers, or the list of employees. Other values, for example a path, a config value, or a time zone, may appear in errors. | These identify a person. The rest does not. |
| The server never reads IP addresses. | What it never reads, it cannot leak. |
| Results only after the period ends, only per period, and only with at least k submissions. Below k, delete the answers. | Live results, small groups, and differences of sums reveal single answers. |
| No page shows a count that changes when someone submits in an open period, for example the number of submissions so far. | A count that goes up shows when someone submitted. |
| Show text answers in alphabetical order. | Order of submission links an answer to a time. |
| Backups contain no user IDs and no answers of an open period. | A backup lives longer and travels further than the server. |
| No requests to outside servers: no analytics, no CDN, no external fonts or scripts. | Each third party sees who visits and when. |
| Each rule above has a test that fails when the rule breaks. | A rule without a test breaks silently. |

Known gaps, told to employees on the limits page:

- The admin is trusted.
- The admin can forge submissions and change the counts of an open period. A change to the published results shows only to a person who kept an earlier results hash.
- The writing style of a text answer can identify a person.
- While a period is open, the database records who submitted. A person who reads the database then, with only 1 or 2 submissions, can link them to their answers.
- The eligible count (from the list of employees) makes the group known: "17 of 18" tells colleagues that one person did not submit. This is a deliberate trade-off for transparency.

The long-term token protocol stops the server from linking an account to its answers, but employees still trust the admin for the code that the server runs, and against forged submissions.
