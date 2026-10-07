# Story 1124 — school training animation

## Player result

The playable school now shows both installed training sides. Entering the room
plays the selected fighter or mage transition silently, without blocking the
picker, mode, skill cells, Train, the row list or Escape: the skill panel
already shows the entered class on the first painted frame, because the
column is already resting at that class's own endpoint. Changing to a party
member of the other class swaps the doll immediately, plays that class
transition, starts the existing rotating column at the measured frame
threshold and restores the school controls only after both motions reach
their endpoint.

While each side is free it can independently play its installed idle episode.
The mage ascends to `m0011`, holds that picture and resumes at `m0008`; the
fighter ascends to `m0009` and reverses normally at `m0008`. Idle pictures have
priority over a transition on their own side. The room, column, skills, diamond
and existing school interactions remain in place.

This is presentation only. Character state, training rules, native saves,
simulation bytes and hashes keep their existing contracts.

## Authority and policy

`TOWN-427` through `TOWN-434` at research pin
`682184eb8e2571ee9b23d2ee614b9f9a459a63d3` establish all 62 literal frames,
their dimensions and side anchors, idle-over-transition priority, the shared
strict greater-than-83-ms one-step gate, modulo-23 and modulo-19 transitions,
column thresholds 5 and 6, the exact idle and hold arithmetic, the mage return
skip, the fighter reverse, object-local cleanup, process-static timing fields
and the bounded sound-request surface.

The outer physical paint cadence, random algorithm and seed, focus and dialogue
lifecycle, malformed-family behavior, picture ownership, control enumeration
while blocked and audible device result remain Unknown. Againrom therefore:

- advances only from a focused live App school paint, once at most, with no
  catch-up;
- uses a private presentation generator and separately retained timer and hold
  fields;
- resets local state on entry, reentry, load and new game, keeps a school
  dialogue in the room, and rebases after a focus, menu or cutscene
  suspension, which freezes both the tr/idle sides and any class-change
  column walk already under way (including that walk's own legacy fallback
  on a degraded install);
- loads mage transition, fighter transition, mage idle and fighter idle as four
  independently atomic optional families;
- blocks the school surface only for a changed class's measured
  transition-and-column wait; entry and reentry arm and animate the entering
  class's own transition without ever blocking input (DIV-873);
- requests the installed `Rotate.wav` once at actual column start and invents
  no idle sound.

`DIV-868` through `DIV-877` ledger those client policies and Unknowns.

## Touched surfaces

- `pkg/game`: four-family loading, private controller and random seam, column
  coupling, lifecycle, control blocking, sound request and persistence tests.
- `pkg/ui`: two chosen side pictures, opaque side placement, live App delivery
  and exact composition before the established column, skills and diamond.
- `internal/gatedtests`: installed EN/RU release-test population.

## Proof so far

Focused game and UI tests cover all 62 paths and dimensions, independent
four-family failure, exact opaque draw order, the strict time boundaries and no
catch-up, both transition vectors and thresholds, mage skip, fighter reverse,
idle-over-transition priority, one Rotate request, changed-class input
blocking, lifecycle, read-only views and native/save/simulation-hash
invariance. `TestSchoolTrainingEntryAdmitsInputButClassChangeStillBlocks`
proves the picker, mode, skill cells, Train, the row list and Escape are all
live on the first painted frame after entry and after reentry, that the skill
panel already shows the entered class, that a changed class still blocks the
whole surface until its own tr-and-column wait clears, and that Rotate is
requested exactly once for that changed class and never for entry.
`TestSchoolTrainingSuspendedFreezesTheLegacyColumnFallbackToo` proves a focus,
menu or cutscene suspension freezes a class change's column walk, including
its own legacy fallback for a degraded install, exactly as it already froze
the tr/idle sides. The existing column, diamond, skill-selection and doll
tests remain in the focused package set.

`TestReleaseSchoolTraining1124InstalledAppFramesSoundAndNative` independently
decodes the literal installed paths rather than production constants. A real
App load and school entry reaches live fighter and mage transitions, both idle
returns, retained column and diamond pixels and exactly one waveform-identical
Rotate request. The focused test has passed separately on the preserved EN and
RU roots. Optional ignored captures are written only when
`AGAINROM_STORY1124_FRAMES` names an output directory such as
`review/story1124/en` or `review/story1124/ru`.

The branch is reconciled onto implementation master after story 1123 (town
bird episodes, statue star, repeating crowd). The two stories share no school
surface: 1123 touches the town exterior and ambience files, 1124 touches the
school, column and shell files. Both stories also touched the release-test
population manifest and `pkg/game/frontend.go`, and those combined without a
marked conflict. The merge marked conflicts only in the divergence ledger and
the reset-survivor field list in `frontend_session_test.go`, both resolved as
unions, plus the research gitlink, resolved to this branch's descendant
commit. School training behaviour is unchanged by the reconciliation itself.
The gate is class-change-only (above): entry never blocks, so
`TestReleaseSchoolColumnClassTransitionInstalledPixelsAndSound`,
`TestReleaseSchoolDiamondTrainPaintLifecycle` and
`TestReleaseSchoolCaptionsMatchInstalledWordsAndPixels` click the school
surface on the same painted frame as `CloseTip`, with no settle loop ahead of
that first click. Full final gates and paired release gate output run on the
reconciled candidate.

## Open debt

Original physical paint cadence, generator identity, exact cross-visit static
aliases, missing-art behavior, picture destruction, complete blocked-input
dispatch and audible hardware output remain Unknown. No speaker or GUI-device
result is claimed.
