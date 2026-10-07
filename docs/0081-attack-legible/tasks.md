# Tasks — a fight is legible

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 a facing is simulation state, written by a step

Kind: impl. Carries FR-1, FR-2, FR-3, FR-6, DD-1, DD-2, DD-3, DD-4, DD-6, R-3. Criteria SC-1, SC-2.
Files: `pkg/sim/facing.go` ADD, `pkg/sim/world.go` MODIFY, `pkg/sim/binary.go` MODIFY,
`pkg/sim/step.go` MODIFY, `pkg/sim/facing_test.go` ADD, and every test the record's new width, the
new version byte or a whole-entity comparison moves — in `pkg/sim` and `pkg/mapload` both.

Boundary: the field, its three conversions, the byte form, and the one write a step makes. The attack
path is not touched.

Scope fence: nothing outside `pkg/sim` and the two packages' tests; no route search, no rate law, no
`combat.go`. If the attacker's turn looks needed to make a test pass, the boundary was crossed.

Every pinned digest in both packages moves. Re-pin each beside a backwards derivation onto the
literal that predates this story — one byte per record removed, the version byte put back — which is
the only check that can still fail once a pin and its own digest agree (R-3's shape, one story down).

Done when: a mover faces each of the eight directions it steps in; a blocked, felled, stalled or
arrived unit keeps its last step's facing; all 256 bytes round-trip; version 13 is refused; and the
whole gate is clean.

## T2 an attacker in reach turns toward its victim

Kind: impl. Carries FR-4, FR-5, DD-5. Criteria SC-3.
Files: `pkg/sim/combat.go` MODIFY, `pkg/sim/turn_test.go` ADD.

Boundary: one guarded write on the pursuit arm that stops the walk. The cycle, the blow, the roll,
the damage and the reach test itself are not edited, and the attack cycle takes no facing code.

Scope fence: no field is added, no byte-form offset moves, `formatVersion` is not touched, and
nothing reads the facing. A blow refused for facing away is out of scope: the gate is decoded and its
site is the order machine's act-state entry, which this tree does not have.

Done when: an attacker faces an adjacent victim from all eight relative positions; one out of reach
keeps its walk's facing; a victim on its own cell changes nothing; the same schedule from two worlds
differing only in a facing gives the same health, cells and routes; and the whole gate is clean.

## T3 the attack timeline reaches the descriptor, and a swing is selected

Kind: impl. Carries FR-7, FR-8, DD-7, DD-8, DD-8a, DD-14, R-4. Criteria SC-4, SC-5.
Files: `pkg/data/anim.go` MODIFY, `pkg/render/terrain/unitanim.go` MODIFY, `pkg/game/units.go`
MODIFY, `pkg/data/anim_test.go` MODIFY, `pkg/game/units_anim_test.go` MODIFY,
`pkg/render/terrain/attackanim_test.go` ADD.

Boundary: three descriptor fields, their field-for-field copy, and one new pure selection beside the
two that ship. No caller chooses between them yet — T4 owns the dispatch.

Scope fence: `SelectUnitFrame`'s signature and behaviour are unchanged, `SelectDeathFrame` is
untouched, and no sheet, archive or clock is read.

Done when: the three fields are what a class's keys imply, absent phase and empty track included; the
two tiers hold one field set with the mirror fixture still pairwise distinct; the selection returns
the block's frames for every tick of the run, refuses at and past its length and the four other ways
it must, and is total at any clock and a zero frame count; and the whole gate is clean.

## T4 the seam draws the simulation's facing, and dispatches the swing

Kind: impl. Carries FR-9, FR-10, FR-11, DD-9, DD-10, DD-11, DD-12, R-1, R-2, R-5. Criteria SC-6.
Files: `pkg/game/world.go` MODIFY, `pkg/game/facing_test.go` MODIFY, `pkg/game/swing_test.go` ADD,
and the test files whose `mapWorld` literals name the deleted memory.

Boundary: which direction and which selection each entity is drawn with, and one clock behind the
second of those.

Scope fence: `pkg/ui` gains no field and still names no simulation type; the step delta, `moving`,
the odometer, the death clock and the tier lookup are not touched; and no world is advanced.

The rotation is asserted against the derivation it replaces, over all eight deltas, in the commit
that deletes it — so the two answers are compared while both exist (R-1).

Done when: a swinging entity draws an attack frame and the same entity moving draws a move frame; a
corpse draws a death frame; a class with no attack block falls through; a never-turned unit draws
north and the old south assertion is rewritten rather than dropped; the clock is zero for a pursuing
attacker; a schedule run with and without a viewer gives one digest; and the whole gate is clean.

## T5 the mission tool reports an attacker's facing

Kind: impl. Carries FR-12, DD-13. Criteria SC-7.
Files: `cmd/missionrun/main.go` MODIFY.

Boundary: one compass name appended to the attack line, through the one exported conversion.

Scope fence: no flag, no other line, no second reading of the facing byte, and nothing about how the
tool drives a world changes.

Done when: the attack line states which of the eight the attacker ended facing, and the whole gate is
clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-2 | SC-1 | T1 |
| FR-3 | SC-2 | T1 |
| FR-4, FR-5 | SC-3 | T2 |
| FR-6 | SC-1 | T1 |
| FR-7 | SC-4 | T3 |
| FR-8 | SC-5 | T3 |
| FR-9, FR-10, FR-11 | SC-6 | T4 |
| FR-12 | SC-7 | T5 |
| FR-13 | SC-8 | all |
| AC-1, AC-2, AC-6 | SC-1 | T1 |
| AC-3 | SC-2 | T1 |
| AC-4, AC-5 | SC-3 | T2 |
| AC-7 | SC-4 | T3 |
| AC-8 | SC-5 | T3 |
| AC-9, AC-10, AC-11 | SC-6 | T4 |
| AC-12 | SC-7 | T5 |
| P-1 | SC-1 | T1 |
| P-2 | SC-5, SC-6 | T3, T4 |
| P-3 | SC-6 | T4 |
| P-4 | SC-6 | T4 |
| P-5 | SC-8 | all |
| DD-1..DD-4, DD-6 | SC-1, SC-2 | T1 |
| DD-5 | SC-3 | T2 |
| DD-7, DD-8, DD-8a, DD-14 | SC-4, SC-5 | T3 |
| DD-9..DD-12 | SC-6 | T4 |
| DD-13 | SC-7 | T5 |
| R-1, R-2, R-5 | SC-6 | T4 |
| R-3 | SC-1 | T1 |
| R-4 | SC-5 | T3 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above,
and neither commit carries a trailer.
