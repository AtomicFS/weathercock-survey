# AGENTS.md

`weathercock-survey` is a server for anonymous employee surveys, written in Go.

Read first: `PRINCIPLES.md` (strict), `LLM_MEMORY.md` (yours), `TODO.md` (the maintainer's), `WISHLIST.md` (what to build), and `CONVENTIONS.md` (code, tests, commits).
The global agent instructions apply: your role, finish before you switch, memory, how we work together, and the adversarial review.

## Workflow

1. Propose the next task from `TODO.md` in at most five lines: what, how, and what you will not do. Wait for "go".
2. Build it. Write a test first where a test is worth it. Run `go-task check`, then `go-task test:all`.
3. Ask the `reviewer` subagent for the adversarial review. Give it the diff and `PRINCIPLES.md`, and tell it to check that the change keeps the anonymity rules.
4. Fix the findings that you agree with. Argue back on the others, in one line each.
5. Update `LLM_MEMORY.md`, and commit. Propose changes to `TODO.md` in the report.

## Scaffolding

The maintainer may write a skeleton with instructions in comments that start with `// AGENT:`.
Review the structure first. After approval, fill in the code, and remove each `// AGENT:` comment when it is done.

## Git

- Follow "Commit messages" in `CONVENTIONS.md`, with the AI trailers.
- Use only `git add`, `git rm`, and `git commit` to change the repository.
