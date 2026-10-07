# Tasks — an attack that resolves

**Reading key:** `FR`/`AC`/`P` → `spec.md`; `DD`/`R`/`SC` → `plan.md`. Every entry below is an
*implementation* task: one coherent, test-passing commit each, trailer `SDD-Task:
0064-combat-resolve/T<n>`.

**Recorded deviation (blast-radius heuristic):** T2 exceeds the ~400-line guidance because a
widened record and the hand-written pins that measure it are correct only together; no coherent,
independently testable boundary exists inside it.

## T1 — the attack state on the entity

Kind: implementation. Boundary: the fields exist, the constructor's rules hold, and nothing reads
them; the byte form is untouched.

Files: `pkg/sim/world.go` MODIFY · `pkg/sim/combat.go` ADD · `pkg/sim/nostate_test.go` MODIFY ·
`pkg/sim/combat_test.go` ADD

Covers: FR-7 (the field set), DD-3, DD-4, DD-5, DD-6, DD-13.

Fences: no command kind, no loop in `Step`, no encoder or decoder change, no draw.

Done when: `Entity` carries the attack state and the seven numbers; the phase type has its three
values and a definedness test; a world built with an attacker that is not alive, that names itself,
or that names an id the world does not hold comes back holding no attack order; a world built with
no attack order holds a zero victim, phase and count; a negative count owed and an undefined phase
are refused; the canonical field-set pin names the new fields; the suite is green.

## T2 — the byte form carries the cycle

Kind: implementation. Boundary: the form and its digest; no behaviour changes.

Files: `pkg/sim/binary.go` MODIFY · the seven `pkg/sim` test files holding a hand-written record
width, offset, version byte or pinned digest MODIFY · `pkg/mapload/fromalm_test.go`,
`pkg/mapload/gridform_test.go` MODIFY

Covers: FR-7, FR-8, AC-11, AC-14, P-5, DD-11, DD-12, R-1, R-1a, SC-4, SC-7.

Fences: do not regenerate a pin from the code that writes it — re-derive each from the world it
names. Do not add a migration path for version 8. Leave `pkg/game` alone; it pins no digest.

Done when: the record carries the twelve fields at the offsets the contract's layout gives; the
version constant has moved and version 8 is refused like any other; each refusal of FR-8 has a case
of its own and leaves the receiving world untouched; a round trip is the identity on a world
holding a full attack state; moving any one of the twelve moves the digest; the test whose name
asserts this revision moved no byte of the form says something true again; the tree is green.

## T3 — the order

Kind: implementation. Boundary: an order can be given and is remembered; no tick resolves anything.

Files: `pkg/sim/step.go` MODIFY · `pkg/sim/group.go` MODIFY · `pkg/sim/combat_test.go` MODIFY

Covers: FR-1, FR-2, FR-5, AC-8, AC-9, AC-12, P-4, DD-1, DD-1a, DD-9, DD-10.

Fences: the existing kill and damage arms keep their behaviour and their doc; do not clear the
transit pair or the group rate term on an attack order; do not add the third loop.

Done when: the new kind is a value beside the existing ones and the zero value still means what it
meant; each of the four ignored shapes leaves the world's digest equal to that of the same world
advanced with no command; an attack order clears any walk *and its stored route* while a move order
— single or group — clears any attack; an order naming the victim already held leaves the cycle
untouched; being felled drops the attack order at the one site that drops the rest; suite green.

## T4 — the cycle and the blow

Kind: implementation. Boundary: everything a tick does with an attack order.

Files: `pkg/sim/rng.go` MODIFY · `pkg/sim/combat.go` MODIFY · `pkg/sim/step.go` MODIFY ·
`pkg/sim/doc.go` MODIFY · `pkg/sim/replay_test.go` MODIFY · `pkg/sim/combat_test.go` MODIFY

Covers: FR-3, FR-4, FR-6, AC-1…AC-7, AC-10, AC-13, AC-15, P-1, P-2, P-3, P-6, DD-2, DD-2a, DD-7,
DD-8, DD-8a, DD-14, R-2, R-3, R-4, SC-1, SC-2, SC-3, SC-5, SC-6, SC-8, SC-10, SC-11.

Fences: the loop runs after the move loop and touches neither the occupancy scratch nor any
movement field; the reach test stands before every draw; no arm may skip a draw on a drawn value
except where the contract says a whole cycle is skipped.

Done when: the advance-by-advance trace of a blow matches FR-3's accounting at every charge and
relax including zero and negative; speed moves nothing; a victim ordered attacked dies and its
attacker's order ends; a strike advance costs three draws in reach and one out of it, counted off
the generator's own state; a doubled frame of attack orders yields the identical world, and the same
test states what a doubled damage command does; the replay fixture puts the new fields in motion;
every doc clause the loop falsifies is corrected; suite green.

## Traceability

| Requirement | Criterion | Task |
|---|---|---|
| FR-1 | SC-4, SC-11 | T3 |
| FR-2 | SC-3 | T3 |
| FR-3 | SC-1, SC-2, SC-11 | T4 |
| FR-4 | SC-1, SC-8 | T4 |
| FR-5 | SC-3 | T3, T4 |
| FR-6 | SC-3, SC-10 | T4 |
| FR-7 | SC-5, SC-6, SC-7 | T1, T2 |
| FR-8 | SC-4, SC-7 | T2 |
| SC-9 | — | every task |
