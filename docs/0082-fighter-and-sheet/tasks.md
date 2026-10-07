# Tasks — the hero is a real fighter, and his numbers are on screen

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the rules character generation enforces

Kind: impl. Carries FR-1, FR-2, FR-4, DD-1, DD-4. Criteria SC-1.
Files: `pkg/data/chargen.go` ADD, `pkg/data/chargen_test.go` ADD, `pkg/data/hero.go` MODIFY.

Boundary: from a statistic to what it costs, and from four statistics to whether generation could
have produced them. Pure arithmetic — it builds no party, reads no file and holds no state.

Scope fence: nothing outside `pkg/data`; no screen, no click handler, no pool that anything spends
at runtime; `Derive` and every constant it reads are not edited; and the post-generation cap of 50 is
NOT confused with the click ceiling of 45 — they are different bounds from different routines.

`NewHero` becomes the one construction implementation and `NewCampaignHero` is re-expressed through
it; the latter's doc comment is corrected in place to say it is the generation *start*, not the
character a campaign starts with. The five skill names are the shipped column titles' warrior halves.

Done when: the three cost figures, the escalating last step and the refund symmetry hold; the three
reachable maxima come out 42, 43 and 34; the pool identity `4*T(25) + 100 = 140` holds; a spread
under the floor, over the ceiling and over the budget are each refused for their own reason; the
generation start still derives what it derived; and the whole gate is clean.

## T2 the party's hero is a generated fighter

Kind: impl. Carries FR-3, FR-5, FR-8, DD-2, DD-3, DD-9, R-1, R-4. Criteria SC-2, SC-5.
Files: `pkg/game/hero.go` MODIFY, `pkg/game/hero_test.go` MODIFY, `pkg/game/frontend.go` MODIFY,
`pkg/game/frontend_test.go` MODIFY, `pkg/data/hero_test.go` MODIFY, `cmd/againrom/main_test.go`
MODIFY.

Boundary: which spread the party carries, and the one line a headless run prints about it.

Scope fence: `PartySkillSlot`, `startingWeapons` and `highStartingWeapons` are not re-decided —
`0078` chose the ordinary arm and this task carries that choice unexamined. No spread is computed at
run time: the seam returns a value. `pkg/data/hero_test.go` is touched only for a comment this task
falsifies.

Uniqueness is asserted by searching the legal space in the test; the seam itself must not search.

Done when: the seam's spread is the rule's unique answer; the hero's band is 10-16, his to-hit 49 and
his defence 8 against the blade arm's weapon at `(5, 3)` with to-hit 5; that band equals the owner's
measurement at Body 43; the check line's four numbers are asserted **equal to the party member's
own** rather than merely well-formed; a bare hero still states his character and says why he is bare;
and the whole gate is clean.

## T3 the panel states a unit's numbers

Kind: impl. Carries FR-6, FR-7, DD-5, DD-6a, DD-6b, DD-7, R-2. Criteria SC-3.
Files: `pkg/ui/panel.go` MODIFY, `pkg/ui/overlay.go` MODIFY, `pkg/ui/sheet_test.go` ADD.

Boundary: from a subject to the rows a panel states about it, and the two groups the entity seam
carries them in. Nothing here opens a file, reads a clock or names a simulation, format or data type.

Scope fence: `pkg/ui/readout.go` is not edited and no readout field number moves; the twelve new
field numbers are `4..15` and no other; the drawing, measurement and placement path below
`panelItem` is not touched; no existing row, label, colour, corner or margin changes; and **no new
viewer setter is added** — both groups arrive on the seam that already carries an entity.

The two resolvers stay total and disjoint: each must go on answering false for the other's fields.

Done when: a subject with a character states the four statistics in the order Body, Reaction, Mind,
Spirit and then the skill and the weapon; one without states none of those rows; a subject told
neither group composes the picture the story found, row for row and pixel for pixel; the damage row
reads `base-(base+spread)`; the always-hits row appears only when the mark is set; a dropped row
costs no space; the composed box fits the default window; and the whole gate is clean.

## T4 the values arrive from the world and the loader

Kind: impl. Carries FR-9, DD-6, DD-8, DD-10, R-3. Criteria SC-4, SC-6.
Files: `pkg/mapload/start.go` MODIFY, `pkg/mapload/hero_test.go` MODIFY, `pkg/game/mission.go`
MODIFY, `pkg/game/world.go` MODIFY, `pkg/game/sheet_test.go` ADD.

Boundary: from a started world to what the per-tick entity push carries — the eight read off each
entity, and the character resolved at open.

Scope fence: no field is added to `sim.Entity`, no byte-form record moves, `formatVersion` is NOT
touched, and placement — the drop cell, the walk, the crowding count — is not edited. If a version
bump looks needed the boundary was crossed and this task stops. `newMapWorld` keeps its signature, so
no test call site moves.

The character lookup is the tier lookup's shape exactly, down to never being ranged.

Done when: a started world reports the id it minted for each member; a mission carries the party it
was started with; every entity's eight numbers reach the seam and the party's character reaches it
beside them; an entity the lookup has no entry for states no character, and a driver holding no
lookup states none for anyone; the round trip is equal bytes, an equal hash and version 13 by number;
the whole suite is green with no game present; and the whole gate is clean.

## T5 the speed a Reaction buys

Kind: impl. Carries FR-10, DD-11, DD-12. Criteria SC-7.
Files: `pkg/data/hero.go` MODIFY, `pkg/data/chargen_test.go` MODIFY, `pkg/mapload/start.go` MODIFY,
`pkg/mapload/hero_test.go` MODIFY, `pkg/ui/panel.go` MODIFY, `pkg/ui/sheet_test.go` MODIFY.

Boundary: from a Reaction to a rate, onto the entity that already has the field, and onto one row.

Scope fence: nothing under `pkg/sim` — the rate law and `Entity.Speed` are `0081`'s tree and already
exist; no byte-form record moves and `formatVersion` stays 13. `Derive` and `Combat` are not edited.
The class default stays where it is for every unit that is not a party member.

The owner's «19» is TESTIMONY and is a question, not a target. Derive, apply, and report what falls
out — if it disagrees with him, say so plainly rather than tuning toward it.

Done when: the branch changes at 12 and the four sampled Reactions answer 11, 14, 17 and 21; a
started member carries his hero's speed and not the table's; a zero-value member reaches his by the
same arithmetic with no fallback; the sheet states it and a subject told no numbers states no row;
the two omitted terms are omitted with their reasons written where the law is; and the gate is clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-2, FR-4, AC-1..AC-8, AC-12, P-1, DD-1, DD-4 | SC-1 | T1 |
| FR-3, FR-5, FR-8, AC-9, AC-10, AC-11, AC-18, DD-2, DD-3, DD-9, R-1, R-4 | SC-2, SC-5 | T2 |
| FR-6, FR-7, AC-13, AC-14, AC-14a, AC-15, AC-16, P-4, P-5, DD-5, DD-6a, DD-6b, DD-7, R-2 | SC-3 | T3 |
| FR-9, AC-17, AC-19, AC-20, P-2, P-3, DD-6, DD-8, DD-10, R-3 | SC-4, SC-6 | T4 |
| FR-10, AC-21, AC-22, AC-23, DD-11, DD-12 | SC-7 | T5 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
