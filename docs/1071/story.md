# 1071 — bounded mission-map ambience

## Player result

Missions now sound alive around the camera. Visible river terrain retains a
positional river loop, and live Wall of Fire cells retain an independent fire
loop; both can play under mission music at once. Moving the camera updates each
placement without restarting its audible phase, and a loop stops as soon as its
source leaves the view or the map closes.

Visible ambient object classes also drive the recovered deadline call: birds
select slots 60 through 62 and burnt remains select crow slot 70. Missing slot
62 is deliberate silence. Sound Options controls ambience together with the
other game audio, including live mute and volume changes.

## Authority and as-built behaviour

`VIDEO-SFX-013`, `VIDEO-SFX-017`, `VIDEO-SFX-020`, `REG-OBJ-047` and
`REG-SFX-057` establish the two camera-presence loop selectors, the deadline
formula and count-ratio branch, exact slot identities, `FireObject` comparisons,
and the shipped missing leaves. `MAGIC-MAPLAYER-040` and
`TERR-CELLREC-146` establish Wall of Fire as spell 3 and the original loop's
dynamic-cell bit; Againrom projects its live spell-3 coverage across that seam.

- Visible terrain groups 8 through 11 retain slot 50.
- Visible live spell-3 cells retain slot 90 independently of slot 50.
- At a due deadline, visible `FireObject >= 0` and `FireObject == -2` classes
  feed the decoded count-ratio draw. The bird arm uses the executable's
  `rand()/16383` selector for slots 60 through 62; the other arm uses slot 70.
- A successful source census schedules the next attempt at
  `now + 10000 + rand()/2` milliseconds. With no candidate, the deadline
  remains due.
- One presentation RNG owns ambience. It is not simulation RNG and is neither
  saved nor hashed.
- The selected lawful install's existing `SoundBank` supplies every sample.
  Missing bank, leaf, device, or invalid PCM is silence and never blocks play.

`DIV-506` discloses the modern projection: a camera midpoint, a source-cell
centroid, the existing `audio.Place` law, phase-preserving loop replacement,
and a first-frame deadline schedule instead of an immediate call. Wind slots
80/81, direct filename forms and the unresolved non-music call population are
not inferred into this slice.

## Proof

- `pkg/ui/ambient_test.go` covers visible source classification, simultaneous
  loops, idempotence, placement updates without restart, independent stop,
  deadline arithmetic, missing-sample silence, and synthetic non-negative
  `FireObject` classes.
- `pkg/game/ambient_test.go` proves only live spell-3 coverage crosses the
  presentation seam without mutation.
- `pkg/game/statics_test.go` pins `FireObject` through the production registry
  loader.
- `TestReleaseMissionAmbientPopulation` resolves slots 50, 60, 61, 70 and 90,
  refuses 62, 80 and 81, and measures the exact `-2 × 21`, `-1 × 61`,
  non-negative `0` class population on both preserved roots.

## Open debt

The exact ROM1-to-Againrom dynamic-cell identity and original gain/pan reduction
remain open under `DIV-506`. The 85 unresolved direct calls, wind leaves and
direct filename forms remain research backlog, not guessed gameplay events.
