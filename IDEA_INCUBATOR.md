# Idea incubator

Ideas that are not planned. Each idea can cause harm. Do not build an idea from this file without a harm review. `PRINCIPLES.md` applies.

## 360-degree feedback

Employees rate each other, peer to peer. Each subject is the only reader of their own results.

### Base

- One `summed` form per subject, made from one template.
- Results only after the period ends, and only with at least k submissions.
- Any employee can rate any subject. A large rater pool hides the raters.
- The subject never rates themselves. Otherwise the subject can subtract their own answers and see the sum of k-1 raters.
- No rater groups (manager, peer, report). Each group needs its own k, and a manager group has only one person.

### Harms

- Abuse: free text about a named colleague. The subject cannot answer.
- Uneven feedback: one person gets many ratings, another gets none.
- "Fewer than k" tells the subject that almost nobody rated them.
- A manager can demand to see the results of a subject. The same reason stopped receipts for employees.

### Mitigations to explore

- Opt-in: an employee asks for feedback. Nobody gets rated without a request.
- Structured answers: "one best quality" and "one thing to improve", each picked from a fixed list. A free-text box for these brings back the abuse risk.
- Nudges: "Would you write feedback for X?", with a "skip: I don't know this person" button.
  - At the start of each period, the server copies the list of raters and the list of subjects, and shuffles each list with a secret random seed.
  - Rater i gets m nudges: subjects i, i+1, ..., i+m-1 in the shuffled list, never themselves. Use m of at least k.
  - Each subject gets the same number of nudges.
  - Delete the seed when the period ends.
  - Never use a public order, such as the alphabet. The subject could then work out who was asked to rate them, and press them for proof.
  - Rejected: nudge the subject with the fewest ratings so far. Reloading the page shows live counts, a change links a rater to a subject, and everyone sees who gets few ratings.

### Open questions

- Do results disappear after the subject reads them? This weakens the manager's demand a little, but does not stop it.
- Is k for 360 feedback higher than k for company surveys?
- Free text, structured answers, or both?
