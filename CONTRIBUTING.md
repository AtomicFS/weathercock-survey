# Contributing

We use GitHub for the code, the issues, and the pull requests: <https://github.com/AtomicFS/weathercock-survey>.

Follow the rules for code, tests, documentation, and commits in [CONVENTIONS.md](CONVENTIONS.md).

## Before you start

- Read `PRINCIPLES.md`. We reject a change that weakens anonymity, or that adds weight without a clear need.
- `WISHLIST.md` lists what the project should become.

## Security and anonymity problems

Do not report a security problem or an anonymity problem in a public issue.
A public report can harm employees before a fix is available.

Report the problem with [GitHub private vulnerability reporting](https://github.com/AtomicFS/weathercock-survey/security/advisories/new).

## Issues

- Before you open an issue, search the [issue tracker](https://github.com/AtomicFS/weathercock-survey/issues) for the same issue.
- Never put real employee data, user IDs, or logs of a production server in an issue.

## Pull requests

1. Fork the project on GitHub.
2. Do your work in a branch of your fork.
3. Run `go-task check`. It formats the code, runs the linters, and runs the tests.
4. Push the branch to your fork.
5. Open a pull request against <https://github.com/AtomicFS/weathercock-survey>.

- Make small commits with one purpose each, so that the review is easy.
- Each commit does not need to build and pass the tests. The full pull request MUST build and pass all tests.
- If the pull request closes an issue, write "Fixes #NNN" in the description.
- Never put real employee data, user IDs, or logs of a production server in a pull request.

## Code review

The maintainer reviews each pull request.
