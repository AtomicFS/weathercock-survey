# Limits of anonymity

This service hides which answers are yours from your colleagues.
It also hides them from a person who reads the database after the planned end of a period.
This page lists the risks that this service does not protect you from.

## The server admin

You must trust the server admin. The admin can link you to your answers:

- Your browser gets your token after you sign in, and sends your answers when you click submit. Both requests come from the same IP address.
  An admin who records the traffic can link the two requests. If few people fill in the form at the same time, also the time can link them.
- From the database alone, the admin knows who got a token, but not who submitted.
- The admin can change the code that the server sends to your browser.
- The admin can sign your token with a key that only you get, and then identify you by the key.
- The admin holds the signing key, so the admin can add forged answers. See [Summed results](#summed-results).

On the server, the end of this page lists the trusted people: the admin, and each other person or company with access to the server, the reverse proxy, the database, or the backups.
The admin writes this list. The service cannot check that it is complete or correct.

## The database before the planned end

- Until the planned end of a period, the database records who got a token for a form with one submission in each period.
  If the server is stopped at the planned end, the server deletes this record at its next start.
- If only 1 or 2 people got a token and submitted, a person who reads the database before the planned end can link them to their answers.
- A person with a copy of the database made before the planned end can add forged answers to that period.

## Backups

- The server makes a backup only after the end of a period.
  A backup does not record who got a token, and contains no data of a period that did not end.
- If the server loses its data before the end of a period, your submission is lost. Keep your own copy of your answers, and submit them again.
- Deleted submissions can stay in backups until the server or the admin deletes the backups.

## The identity provider

The identity provider (for example Google or Microsoft Entra) records when you sign in.
You sign in before you fill in the form, so the time of the sign-in can be close to the time of the submit.
The identity provider does not see your answers.

## Your browser

For a form with one submission in each period, your browser gets your token after you sign in. If the page shows the eligible count, you get the token only after you click "Continue".
If you delete the data of your browser, or change to a different browser, before you submit, you cannot submit again in that period.

## Readers

Forms whose submissions go to readers are for small matters. They are not a whistleblower channel.
To report a breach of law, use the reporting channel of your company or of the authorities.

Readers can read the submissions of these forms.
A reader can give a submission to other people.
Each submission tells the readers if it is confidential, or if they can discuss it openly. The service cannot make the readers respect this choice.
The form page lists the readers of the current period. The admin can remove readers, but cannot add readers who can read this period.

## Your answers

- Your writing style, the language you write in, or details in your text can identify you.
- A rare combination of answers can link your submissions across periods.

## Summed results

The results page shows a period only if it has a minimum number of submissions (k).
At the planned end, the server deletes the answers of a period with fewer submissions. They are not in the results, in the database, or in backups.
The results page and the CSV export show each period separately, and only after its planned end.

- Forged answers can lift a period above k. The results page then shows the real answers of fewer than k people.
  After each period with at least k submissions, the form page shows how many tokens the server issued and how many were spent. More spent tokens than issued tokens show forged answers.
  An admin who changes the database can hide forged answers.

## Participation

For each period with at least k submissions, the results page shows how many people were on the list of employees (the eligible count), and the participation rate.
After you sign in, and before you get your token, the page shows the eligible count of the current period, if the form publishes it. You can then click "Do not take part". The server records nothing until you continue.
Thus, everybody who can read the results knows the number of employees, and how many took part.
This can show more than the counts alone:

- If everybody submitted, each count applies to each person.
  Example: "18 of 18" submitted, and 18 selected "unhappy". Then each person selected "unhappy".
- If few people did not submit, your colleagues can guess who did not take part.
  Example: "17 of 18" tells them that exactly one person did not submit.
- Your colleagues can compare two periods. Example: one period shows 18 of 18. Bob joins, and the next period shows 19 of 19.
  Then the change of each count is Bob's answer, plus the changes of the other people. If people give the same answers each period, the result is close to Bob's answers.
  Your colleagues can find the answers of a person who leaves in the same way. The minimum number of submissions (k) does not stop this.
- If the list has exactly k people, each public period had full participation.
- If the eligible count is equal to the number of issued tokens, each person on the list got a token. Then "I did not take part" is not believable.
- `at least` next to the eligible count shows that a person who got a token left the list during the period. It also shows that a person who got a token joined the list.
  If your colleagues know the one person who left or joined, they know that this person took part.
- The eligible counts over the periods show when people joined or left.
- The eligible count is the highest number of people on the list during the period. A person who left early counts for the whole period.

Some forms do not publish the eligible count. Without a list of employees, the server shows no eligible count.

## Your device

A compromised, monitored, or stolen device is a risk that this service cannot protect you from.
Examples are malware, monitoring software of your company, or a lost or stolen laptop or phone.

- Anyone who can use your browser can read your draft.
  This includes a person who steals your device.
- Copied text can stay in your clipboard history, or sync to other devices.

## The reverse proxy and other programs

This service runs behind a reverse proxy. The reverse proxy handles the encrypted connection (HTTPS) from your browser.
The reverse proxy, network logs, and other programs on the server can record your IP address, the time of your requests, and the pages that you open.
These programs are outside the control of this service.
