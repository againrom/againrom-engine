# Tasks — the player can order an attack

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 an attack order closes the distance

Kind: impl. Carries FR-6, DD-7, DD-8, DD-9, DD-10, R-1, R-2. Criteria SC-5, SC-6.
Files: `pkg/sim/step.go` MODIFY, `pkg/sim/combat.go` MODIFY, `pkg/sim/combat_test.go` MODIFY,
`pkg/sim/pursuit_test.go` ADD.

Boundary: what an attacker does with its destination while it holds a victim. The cycle itself, the
blow, the roll, the damage and the reach are not edited — only who writes a destination and when.

Scope fence: no field is added to `Entity`, no byte-form record moves, `formatVersion` is not
touched, and nothing outside `pkg/sim` is edited. If a new field or a version bump looks needed the
boundary was crossed and this task stops.

0064 FR-2's superseded half is asserted by name in `pkg/sim/combat_test.go`; the assertion is
rewritten in place to the new form in this commit, not deleted (R-1).

Done when: an attacker ordered onto a distant victim walks to it and lands a blow; one ordered onto
an adjacent victim never moves; a victim that walks away is followed; a victim that is killed or
leaves the world ends the walk with no residue; a move order still ends a fight whole; a world
holding an attacker mid-approach round-trips; and the whole gate is clean.

## T2 the seam, the armed mode and the press

Kind: impl. Carries FR-1, FR-2, FR-3, FR-4, FR-5, FR-8, FR-9, DD-1, DD-2, DD-3, DD-4, DD-5, DD-6,
DD-11, DD-13, DD-14, R-3. Criteria SC-1, SC-2, SC-3, SC-4, SC-8, SC-9.
Files: `pkg/ui/flow.go` MODIFY, `pkg/ui/command.go` MODIFY, `pkg/ui/app.go` MODIFY,
`pkg/ui/viewer.go` MODIFY, `pkg/ui/overlay.go` MODIFY, `pkg/game/world.go` MODIFY,
`pkg/game/frontend.go` MODIFY, `pkg/ui/attack_test.go` ADD, `pkg/game/attack_test.go` ADD, and every
existing test the loader tuple's new member moves.

Boundary: from the arming key to one command on the pending queue. Nothing under `pkg/sim` is
edited, and no world is advanced anywhere on this path.

Scope fence: no cursor art, no panel widget, no drawing at all; the tap, the box, the slop
accumulator and the unarmed press are not touched; and the far side makes no ownership comparison.

Done when: the key arms only through the gate and disarms on a second press; an armed press over a
drawn unit issues attacks and no moves and one over ground issues exactly the unarmed press's moves;
every secondary press made while armed spends the arm; the far side appends one attack command per
call, marks the attacker and advances nothing; a nil seam is safe; and the whole gate is clean.

## T3 the readout states the armed flag

Kind: impl. Carries FR-7, DD-12. Criteria SC-7.
Files: `pkg/ui/readout.go` MODIFY, `pkg/ui/readout_attack_test.go` ADD.

Boundary: one field, one row, one resolver arm and the subject value behind them.

Scope fence: no other row moves, no geometry, palette or font changes, and the arming rule itself is
not touched — T2 owns it.

Done when: the readout carries a row stating the flag, its picture changes when the flag changes,
and leaving the map screen leaves it down; and the whole gate is clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1, SC-7 | T2, T3 |
| FR-2, FR-3 | SC-2, SC-3 | T2 |
| FR-4, FR-5 | SC-4 | T2 |
| FR-6 | SC-5, SC-6 | T1 |
| FR-7 | SC-7 | T3 |
| FR-8 | SC-8 | T2 |
| FR-9 | SC-9 | T2 |
| AC-1, AC-1a, AC-15 | SC-1 | T2 |
| AC-2, AC-3, AC-6 | SC-2 | T2 |
| AC-4, AC-5 | SC-3 | T2 |
| AC-7 | SC-4 | T2 |
| AC-8, AC-9, AC-10 | SC-5 | T1 |
| AC-11 | SC-6 | T1 |
| AC-12, AC-13 | SC-7 | T3 |
| AC-14 | SC-9 | T2 |
| P-1, P-2, P-4 | SC-8 | T2 |
| P-3 | SC-6 | T1 |
| DD-1, DD-2, DD-3, DD-4 | SC-2, SC-4 | T2 |
| DD-5, DD-6 | SC-1 | T2 |
| DD-7, DD-8, DD-9 | SC-5 | T1 |
| DD-10 | SC-6 | T1 |
| DD-11 | SC-4 | T2 |
| DD-12 | SC-7 | T3 |
| DD-13, DD-14 | SC-8, SC-9 | T2 |
| R-1 | SC-5 | T1 |
| R-2 | SC-5 | T1 |
| R-3 | SC-3 | T2 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
