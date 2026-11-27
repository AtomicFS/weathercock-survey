# Conventions

Rules for code, tests, dependencies, documentation, and commits. `PRINCIPLES.md` outranks this file.

## Go code

- Use the Go version in `go.mod`. Start each source file with `// SPDX-License-Identifier: AGPL-3.0-or-later`.
- Run `go-task check` before each commit. It formats (`gofumpt`), lints (`.golangci.yml`), and tests.
- Write short, plain code. Prefer clear code over clever or fast code. A wait of 1 or 2 seconds is fine in a survey.
- Small functions with one task, small packages with one responsibility. Names that show the intent.
- Use the standard library first, then the approved dependencies. Do not write your own cryptography.
- Accept interfaces, return concrete types. Define an interface where it is used.
- No global mutable state. The one exception is the URL of the source code, which the build sets.
- Comments explain why, not what. Each exported identifier has a doc comment.

## Errors and logs

- Return errors with context: `fmt.Errorf("...: %w", err)`. Use sentinel errors for conditions that callers check.
- Use `log/slog`. Give each warning and error log line a `suggestion` attribute with the fix.
- For a state that only a bug can cause, use the suggestion `bug.Suggestion`.
- Never log the data that `PRINCIPLES.md` forbids. Wrap such values in types whose `LogValue()` returns `[redacted]`.

## Tests

- Write the test first where a test is worth it. Start each bug fix with a test that shows the bug.
- Test behavior through public APIs. Test each anonymity rule, each rule that refuses bad input, and the code that other code relies on.
- Do not test the fixed words of an error, a message, or a page. Check errors with `assert.ErrorIs` and `assert.ErrorAs`.
- Do test the data that a page or an error shows or must not show, for example a count or the absence of a user ID.
- Fewer, stronger tests. Each test must be able to fail for a real reason.
- Table-driven tests with `t.Run`, rows grouped below `// Pass` and `// Fail`. `testify` for assertions, `go-cmp` for structs.
- Isolate with `t.TempDir()` and `t.Setenv()`. Use a real SQLite database, not a mock. Guard slow tests with `testing.Short()`.

## Dependencies

Ask the maintainer before you add a dependency that is not in this list.

- `alecthomas/kong`: the command line.
- `go.yaml.in/yaml/v4`: the YAML files.
- `modernc.org/sqlite`: the database.
- `coreos/go-oidc`, `golang.org/x/oauth2`: the sign-in.
- `cloudflare/circl`: the token protocol (later).
- `yuin/goldmark`: build time only, for the limits page.
- `stretchr/testify`, `google/go-cmp`: tests.
- Pico CSS, classless version: `web/static/pico.classless.min.css`.

## Web pages

- No JavaScript for now. Render HTML on the server with `html/template`. Embed all templates and static files with `embed`.
- Parse all templates at start, and refuse to start on an error. Build one template set for each page: clone the layout, then parse the page. One set for all pages lets a later `{{define}}` replace an earlier one without an error.
- Set `Option("missingkey=error")`. Give each page its own view struct, never a map.
- Render into a buffer, then write the status and the body. Otherwise an error halfway through sends half a page with status 200.
- Test each page state that changes a branch, for example signed in or not, and below k or not. A template finds a wrong field name only in a branch that runs.
- Use `template.HTML` only for the limits page that `goldmark` makes at build time. Never make it from configuration or input: it skips the escaping.

## Documentation

- Write documentation in Simplified Technical English: short sentences, active voice, one instruction per sentence.
- Show a small example instead of a long description.
- Include code and configuration from real files with mdBook `{{#include}}`, not by hand. Build with `go-task docs:build`.

## Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), with these types: `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`.

- Do not commit secrets, credentials, or real employee data.
- Sign off with `git commit -s` ([Developer Certificate of Origin](https://developercertificate.org/)).
- Mark content made with AI with the trailers `AI-Generated: true` and `AI-Model: <model name>`.
