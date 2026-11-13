# Story document format

Owner ruling: every story from story 1643 on has one `docs/<NNNN>/story.md` in
this format. Earlier stories keep the form they landed with.
`internal/storydoc` checks it.

## Template

```
# <what the player gets>

## Result

Three to five sentences in plain words: what a player can now do or see that
they could not before. No code identifier, path or backtick.

## Authority

- Owner: «quote», or a claim ID, or a DIV ID with one line on what it says.

## Rules

1. One player-visible rule per line: when X, Y.
2. A reviewer checks these; a later story must not break them.

## Code

| kind | its one builder | change |
|---|---|---|
| settings panel | `cmd/starter/app.go` scrollList | third list of the one list builder |

## Proof

| test | rules |
|---|---|
| `TestEverySettingIsEditedStoredAndPassed` | 2, 4 |

Release tests at the release: `TestReleaseX`, `TestReleaseY` (or none).

## Open debt

- One line each, with the BACKLOG item or hotfix it waits for; or `none`.
```

## Limits

- At most 6144 bytes. The sections appear once each, in this order.
- Result holds no backtick and no paragraph over five lines.
- Rules is a numbered list, one line each. Every rule number appears in the
  rules column of the Proof table.
- Code has at most 8 rows, Proof at most 12. A Proof row names tests, not what
  each assertion compares.
- No line outside a table or code block is longer than 120 characters, and no
  paragraph is longer than five lines.
- Long evidence (loss controls, censuses, measurement tables, frame lists) goes
  into the review report the brief names; the story links it in one line.
