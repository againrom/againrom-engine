# Tasks — the player's own units can fight

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. Criterion numbers are `plan.md` §Success criteria; `R-n` is §Risks.

## T1 a chargen literal becomes a weapon's numbers

Kind: impl. Carries FR-3, FR-4, DD-2, DD-3, DD-4, DD-5, DD-13, R-1, R-2. Criteria SC-2, SC-3.
Files: `pkg/formats/databin/databin.go` MODIFY, `pkg/formats/databin/doubles_test.go` ADD,
`internal/synth/databin.go` MODIFY, `pkg/data/weapon.go` ADD, `pkg/data/weapon_test.go` ADD.

Boundary: from a name and three collections to one weapon's numbers. No hero, no entity, no start.

Scope fence: nothing under `pkg/sim`, `pkg/mapload` or `pkg/game` is edited; no armour, shield,
magic item, price or enchantment is decoded; and no shape or material is matched against a name this
tree expects to find.

The nine doubles are read at `8j` of the record, per `ITEM-LADDER-019` — the record begins at the
ladder's own origin. Column numbers are the runtime ones, which are the parser's slot indices
already.

Done when: a literal with a shape word, one without, and one with a two-word material each resolve;
the scaled pair subtracts the already-rounded base; a ranged row and an unknown remainder are each
refused by name with the zero value; and the whole gate is clean.

## T2 a hero's own eight numbers

Kind: impl. Carries FR-2, FR-5, FR-6, DD-1, DD-12, R-2. Criteria SC-1, SC-4, SC-7.
Files: `pkg/data/hero.go` ADD, `pkg/data/hero_test.go` ADD.

Boundary: a pure function of a hero and an optional weapon to eight numbers. It builds nothing,
reads no file and holds no state.

Scope fence: health, mana, speed, sight, capacity, reach, experience and the point-buy cost table
are NOT derived — each is a disclosed divergence and a later story. `pkg/data`'s existing types are
not edited.

The order is normative and is `spec.md` FR-2's six steps. The skill's two terms are gated on the
active skill being a slot in `1..5`; the weapon supplies that slot and a bare hero has none.

Done when: the four measured band edges and their step points reproduce; a bare hero's four values
reproduce; the cap binds at 50; Mind and Spirit move nothing; the skill moves the base and the
to-hit and not the spread; the zero hero with no weapon derives five zeroes and the cadence 8/4; the
margin sweep passes over 0..100 on both stats; and the whole gate is clean.

## T3 the party carries them

Kind: impl. Carries FR-1, FR-7, FR-8, FR-9, DD-7, DD-8, DD-9, DD-10, DD-11, R-3, R-4.
Criteria SC-5, SC-6.
Files: `pkg/mapload/start.go` MODIFY, `pkg/mapload/start_test.go` MODIFY,
`pkg/mapload/hero_test.go` ADD, `pkg/game/hero.go` ADD, `pkg/game/table.go` MODIFY,
`pkg/game/frontend.go` MODIFY, `pkg/game/hero_test.go` ADD, `cmd/missionrun/main.go` MODIFY.

Boundary: from a party member to an entity's eight fields, and from an install to the one weapon the
front end starts with.

Scope fence: no field is added to `Entity`, no byte-form record moves, `formatVersion` is NOT
touched, and placement — the drop cell, the walk, the crowding count — is not edited. If a version
bump looks needed the boundary was crossed and this task stops.

`start.go`'s `PartyMember` comment is REWRITTEN in place to say what a hero now is and what he still
is not; it is not deleted. `LoadTable` keeps its signature and becomes a call to `LoadDefinitions`.

Done when: a started member carries the derived numbers and a zero-value member carries today's; a
world holding such a party round-trips to equal bytes and an equal hash at version 12; the front end
resolves the weapon from the one parse and its check line states the band, or states why there is
none; the mission tool builds its party the same way; and the whole gate is clean.

## T4 the instrument that measures the blow

Kind: impl. Carries FR-10, DD-14. Criteria SC-8.
Files: `cmd/missionrun/main.go` MODIFY, `cmd/missionrun/main_test.go` MODIFY.

Boundary: one repeatable flag, the command it queues and the line it prints.

Scope fence: no waypoint behaviour changes, no default output changes, and nothing outside
`cmd/missionrun` is edited.

Done when: the flag names an attacker and a victim by the references the tool already resolves,
drives until the victim falls or the ceiling is reached, reports which and on what tick; a run with
no flag prints exactly what it printed before; and the whole gate is clean.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1, FR-7, FR-8 | SC-5 | T3 |
| FR-2 | SC-1, SC-4 | T2 |
| FR-3 | SC-2 | T1 |
| FR-4 | SC-3 | T1 |
| FR-5, FR-6 | SC-1 | T2 |
| FR-9 | SC-6 | T3 |
| FR-10 | SC-8 | T4 |
| AC-1, AC-13, AC-14, AC-18 | SC-5 | T3 |
| AC-2, AC-3, AC-4 | SC-1 | T2 |
| AC-5, AC-6, AC-9 | SC-2 | T1 |
| AC-7, AC-8 | SC-3 | T1 |
| AC-10, AC-11, AC-12 | SC-4 | T2 |
| AC-15 | SC-6 | T3 |
| AC-16, P-1 | SC-7 | T2 |
| AC-17 | SC-8 | T4 |
| P-2, P-5 | SC-5 | T3 |
| P-3 | SC-3 | T1 |
| P-4 | SC-6 | T3 |
| DD-1, DD-12 | SC-1, SC-7 | T2 |
| DD-2, DD-3, DD-4 | SC-2 | T1 |
| DD-5, DD-13 | SC-3 | T1 |
| DD-6 | SC-1 | T2 |
| DD-7, DD-8 | SC-5 | T3 |
| DD-9, DD-10, DD-11 | SC-6 | T3 |
| DD-14 | SC-8 | T4 |
| R-1, R-2 | SC-2 | T1 |
| R-3, R-4 | SC-5, SC-6 | T3 |

`verification.md` and the runnable build are pipeline stages, not tasks: neither is an entry above
and neither commit carries a trailer.
