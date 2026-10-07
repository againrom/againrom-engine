# Credits roll and hall input

## Intent and authority

The campaign ending's credits roll and hall of fame take the pace and input the
original has, where the claims are High. Authority: `FAME-030` (step rule, start
offset, end count), `FAME-031` (credits and hall input), `FAME-032` (message 442
is not a second input), `FAME-033` and `SAV-1159` (shared reset), `FAME-026` and
`FAME-034` (score source), `SAV-971`. Owner direction: change state only where
a claim is High and state the rest.

## As built

- The roll steps one pixel when 23 ms or more have passed since its last step,
  once per engine tick at most. The first step waits for nothing. The start
  offset is 480. The roll ends when the step count reaches 480 plus the line
  count times the line pitch. The pitch is the installed menu font's line height
  (High step rule, Medium pitch of 15). The main-menu Credits uses the same
  screen and the same rule.
- Any key-down or a left-button press ends the roll. The right button no longer
  does.
- The hall ends on a left-button release inside the rectangle
  `0x230,0x1a0,0x25c,0x1c0` (High), whether or not the press began there. That
  rectangle replaces the engine's larger button rectangle on the hall layout
  that has the installed background and at most ten rows. Other hall layouts keep
  the engine's buttons.
- Enter on the focused button stays (Medium in `FAME-031`). Escape stays an
  end of the hall; the native key slot, button and base handler ignore it, but
  other root handlers were not enumerated (Medium), so no change is made.
- The terminal route and the new-game install reset the same session fields.
  The terminal route calls `resetSessionForNewGame`; the install assigns the same
  fields. No field changed: the native field lists are Medium and name nothing
  the engine's reset lacks.
- The score source stays the StartingHero Human (DIV-2064 restated: the new-game
  first drawable is Unknown).

## Proof

- `TestCreditsRollStepsOnePixelPer23Milliseconds`,
  `TestEndingCreditsRollEndsAfterStartPlusLinesTimesPitch`,
  `TestCreditsRightButtonDoesNotEndTheRoll`,
  `TestEndingOpensCreditsRollThenHallAndKeysEndTheRoll`,
  `TestEndingHallEndsOnReleaseInsideItsRectangle`,
  `TestCreditsScrollLogoPauseClose1184`.
- `TestTerminalResetDropsTheNewGameSessionFields`.
- Release, EN and RU: the ending and campaign tests named in `docs/1295/story.md`.

## Open debt

DIV-2060: draw-call cadence and the native segment count of the credits text
(Unknown, Medium). DIV-2061: grade of Enter and Escape on the hall. DIV-2063:
field lists and the town screen's room state. DIV-2064: the new-game first
drawable. DIV-1264, DIV-1267 and DIV-1268 are restated, not closed.
