# weathercock-survey

Named after `weathercock`, also called `wind vane` and `weather vane`, as means to show direction of the wind. In this case it is used to show trends in people.

![A weathercock](weathercock.jpg)

Image source: [Wikimedia Commons](https://en.wikipedia.org/wiki/Weather_vane#/media/File:Wind_vane_05643.jpg).

`weathercock-survey` is a self-hosted service for anonymous, repeating or continuous employee surveys.

> [!WARNING]
> The project is in early stages. Just a proof of concept. Do not use in production.

> [!WARNING]
> The project is currently relying heavily on LLM/AI generated code to reach first prototype.
>
> I would like this to later change into more hands-on approach, once the concept is proven to work.

## Features

### Version 1

Transparency is a requirement, not an extra:

- License `AGPL-3.0-or-later`. Each page links to the source code of the running version.
  `/version` shows the SHA-256 hash of each embedded file. The build is reproducible.
- A public list of trusted people: each person or company with access to the server, the reverse proxy, the database, or the backups, with the date when the admin added them.
- A public "Limits of anonymity" page. It tells employees in plain words what the service does not protect them from.
- Clear warnings before an employee answers, for example "You must trust the admin" and "Your writing style can identify you".
- Public results. Each employee can read them.

Anonymity by design:

- The server stores counts, not responses. Nothing links the answers of one person.
- If fewer than the configured minimum number of people submit in a period, the server deletes the answers of that period. It does not only hide them.
- The server stores no time data: no timestamps, no insert order, and no request logs.
- The server deletes the record of who submitted at the end of each period.
- The results of a period become public only after the period ends. Nobody can see the results change after a colleague submits.
- Backups contain only closed periods, and no record of who submitted.

Simple to run:

- One Go binary, SQLite, and sign-in with the company SSO. The pages use no JavaScript.

### Long-term

The long-term design keeps the other features of version 1, and adds:

- Blind tokens (Privacy Pass, RFC 9578). The database cannot link a person to their answers, also while a period is open.
- Public token counts for each period. They show forged answers.
- A grievance box. The form page lists the readers of the submissions. The admin cannot give a new reader access to older submissions.
- JavaScript in the browser: plain JavaScript, without a framework and without a build step.
  It runs the token protocol, keeps a draft of the answers, and checks the answers before the submit. The server checks them again.

## Goals of version 1

- Low friction: sign in with Single Sign-On (SSO), answer a few questions, done.
- Anonymous: after the planned end of a period, the stored data cannot link a person to their answers. See [Anonymity and trust](#anonymity-and-trust).
- One submission per person in each period of a form. Monthly periods by default, weekly, quarterly, and yearly as options.
  An optional window limits the submissions to the first days of each period, for example 14 days of each year.
- Configurable questions in one YAML file for each form: scale, single choice (radio buttons), multiple choice (checkboxes), and free text.
- Plain HTML pages, no JavaScript.
- One flat group of people, with no split into departments or teams. Use it for a whole company, or for one team.
- If too few submissions arrive in a period, the server deletes their answers at the end of the period, to protect privacy.
- Results page with trends, without sign-in. It is meant for the intranet. If the internet can reach the server, protect the results with one shared account at the reverse proxy.
- The results show each question separately. They cannot show how the answers to two questions relate, for example satisfaction against workload. This is a deliberate choice: linked answers can identify a person.

The [wishlist](https://github.com/AtomicFS/weathercock-survey/blob/main/WISHLIST.md) adds blind signatures, a grievance box, and more.

## Anonymity and trust

> [!WARNING]
> You must trust the server administrator.
> The server sees the account and the answers together when a person submits.
> The limits page lists the admin and each other person with access to the server, the reverse proxy, the database, or the backups.

The server stores the answers as counters and as separate text answers. Nothing links the answers of one submission, or the answers to the person.

- Until the planned end of a period, the database records who submitted, so that each person submits only one time. At the planned end, the server deletes these records.
- The server makes backups automatically, only after the end of a period. A backup contains only closed periods, and no record of who submitted.
- If the server loses its data, the submissions of the current period are lost.
- If only 1 or 2 people submitted, someone who reads the database before the planned end can link them to their answers.
- Application logs contain no identities and no answers.
- Logs from a reverse proxy or from other programs are outside the control of this project.

Read [Limits of anonymity](https://github.com/AtomicFS/weathercock-survey/blob/main/web/limits_of_anonymity/v1.md) for what employees are not protected from in version 1.

## Documentation

- [Principles](https://github.com/AtomicFS/weathercock-survey/blob/main/PRINCIPLES.md): the core principles and the anonymity rules.
- [Wishlist](https://github.com/AtomicFS/weathercock-survey/blob/main/WISHLIST.md): what the project should become.

Build the docs with `go-task docs:build`.

## License

[GNU Affero General Public License v3.0 or later](https://github.com/AtomicFS/weathercock-survey/blob/main/LICENSE) (`AGPL-3.0-or-later`).
