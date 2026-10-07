# Dead and destroyed stay so after LOAD

## Intent and authority

Result: a creature or building the player killed or destroyed is still dead
or destroyed after SAVE, a cold LOAD in a fresh process, and the next SAVE
and LOAD. Tester defects: 90 (a dead black bat alive after LOAD) and 102 (a
destroyed building intact after LOAD).

Authority: knowledge k165. `SAV-BLDG-037` (Building +0x42 is health; LOAD
restoring it is inference), `SAV-1163`, `SAV-1165`, `TERR-STRUCT-210` to
`TERR-STRUCT-212`, `SAV-DEADLOAD-*`. The original keeps a destroyed
structure as an ordinary 77-byte Building record with +0x42 = 0.
Unknown to research (`TERR-STRUCT-211`): mission-start drawing of
authored-zero placements. No change was made there.

## As built

- Item 102, lost seam: the SAV writer. A world loaded from a SAV keeps the
  retained document and projects live state into it. With no saved-structure
  registry the projection returned early, so a building destroyed after LOAD
  was written with its loaded +0x42 and was whole again at the next LOAD.
  `projectUnregisteredStructureHealth` (`pkg/game/savstructureproject.go`)
  writes Field42 and max health into the unique Building record of the same
  cell and kind.
- LOAD refusal 1, mission-open and reader seam: `ImportOriginalDyingActors`
  refused a dying actor a script had taken off the map (`OffMap`). The
  refusal is removed (`pkg/sim/originaldying.go`). Mission 120 spirits
  killed while hidden gave "no restored construction binding".
- LOAD refusal 2, writer seam: `Actions()` wrote attached-effect rows whose
  target had left the world, so `RestoreActions` failed its count check
  ("incomplete current effect caster population"). Those rows are now
  omitted (`pkg/sim/actioncontinuation.go`). Mission 90 Dragon (entity 139,
  authored effect spell 20) reproduces it.
- Item 90: no route reproduced a dead bat alive after LOAD. Repeated
  SAVE and LOAD of mission 120 kills at 2, 12 and 40 ticks over three
  generations keep every killed map unit dead. The two LOAD refusals above
  are the adjacent defects found; either can present to a tester as a wrong
  state.

## Proof

Release tests in `pkg/game/loadrestore_release_test.go`, run on EN and RU:

- `TestReleaseDestroyedStructureStaysDestroyedAcrossSaveLoadChain`: source
  Building +0x42 = max, SAV +0x42 = 0, cold LOAD health map equals the saved
  world, ruin draw, 20 ticks, second SAVE and LOAD. Subtest "source word
  restored" patches +0x42 back to max and shows LOAD restores it whole.
- `TestReleaseDeadHiddenActorStaysDeadAcrossSaveLoad`
- `TestReleaseDeadActorWithAttachedEffectStaysDeadAcrossSaveLoad`
- `TestReleaseKilledActorsStayDeadThroughRepeatedSaveLoad`

Loss controls, each with its fix reverted on EN: without the structure fix
the first test fails with SAV +0x42 = 300, want 0; without the effect
filter the third fails with "incomplete current effect caster population";
with the `OffMap` refusal restored the second fails with "no restored
construction binding".

## Open debt

- Item 90 is not reproduced. Falsifiable question for research: does the
  original let a ground sword attack the class-70 Bat (Domain 2), and which
  unit is the black bat of mission 120.
- Mission-start drawing of authored-zero placements is Unknown
  (`TERR-STRUCT-211`).
- Other mutable structure state (use amount) is not projected into a
  retained document.
- An effect on a removed creature is not written; the original's handling
  is not measured.
