# 0155 — provenance

## Research claims

None. This story adds no format knowledge and decodes nothing. Every fact it
depends on is already implemented in this repository: the mission start path, the
compiled script and its gap census, the simulation command set, and the party
build. The research submodule pin is unchanged for the duration of the story.

## Ours by choice

The scenario language is entirely this project's own design. Nothing about it is
derived from the original game, and no original behaviour constrains it.

- The stage split, the command table, the condition forms and the assertion
  fields are authored here.
- The unit reference grammar (`uNN`, `pN`) is this project's, introduced by
  `cmd/missionrun`; `eNN` is added by this story.
- `unsupported_at_most` reports a property of **this build**, not of the game: it
  counts the arms of a shipped script that this implementation does not yet run.
  A change in that number is a change in the implementation.
- The default wait ceiling of 40000 ticks is `cmd/missionrun`'s own default,
  carried over so that a transcribed drive waits as long as it used to.

## Open

- Whether the two headless drivers should converge is deliberately unanswered.
  This story states that they do not converge now, shares the reference grammar
  between them, and leaves `cmd/missionrun` untouched because
  `pipeline/check-milestone.sh` drives its argv.
- Per-tick script tracing exists only in `cmd/missionrun`. A mission scenario can
  assert the gap count and the latch state but cannot see which arm fired on
  which tick.

## Nothing removed

No behaviour was withdrawn. Version-1 scenarios keep their vocabulary and their
meaning; `scenarios/0152-save666.json` is unchanged and still validates.
