# Tasks — the party member is drawn as what he wears

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. `SC-n` is `plan.md` §Success criteria; `R-n` is §Risks.

## T1 the appearance law

Kind: impl. Carries FR-2, FR-3, FR-8, DD-1, DD-2, DD-3, DD-4, DD-5, P-3.
Criteria SC-1, SC-2, SC-3, SC-4.
Files: `pkg/data/appearance.go` ADD, `pkg/data/appearance_test.go` ADD.

Boundary: from a body name and two facts about what is worn to a class key, a composed name, a
directory and a sheet address. Nothing here opens an archive, reads a registry or knows a party
exists.

Scope fence: no existing file in the package is edited; the class record types, the sprite-path
helpers and the chargen values are read or reused, never changed; nothing here maps a weapon, a
shape name, a skill slot or a statistic onto a name.

Done when: every published name answers its published key with the match flag set and an unmatched
name — a suffixed name with no arm among them — answers the fallback with it clear; the suffix and
both substitutions compose as published and neither touches a living fighter; all sixteen material
blocks and both special arms answer their published directory; an address and its sibling are the
published strings; and the whole gate is clean.

## T2 the party's class is derived from the body it wears

Kind: impl. Carries FR-1, FR-5, FR-7, DD-6, DD-7, P-1.
Criteria SC-5, SC-9, SC-10.
Files: `pkg/mapload/start.go` MODIFY, `pkg/game/hero.go` MODIFY, `pkg/game/frontend.go` MODIFY,
`pkg/game/heroappear_test.go` ADD, `pkg/game/frontend_test.go` MODIFY, `pkg/game/hero_test.go`
MODIFY.

Boundary: one field on the type a start places, one authored function beside the two authored values
already there, and the single expression that turns the first into a class key. The retired constant
goes with it.

Scope fence: the entity-minting loop is not edited and no field it writes changes; no simulation
state type, no `formatVersion` and no encoder is touched; the spread, the skill slot, both weapon
tables and the weapon resolver are unchanged; R-3's readers are re-aimed at the derived value and
never at a fresh literal.

Done when: the placed member's class is the law's answer for the authored body and is not the
unmatched fallback; the authored body and the authored skill slot are asserted as one pair; a world
assembled from identical entities encodes to identical bytes at the same version and the same
digest; and the whole gate is clean.

## T3 the body's own sheet, on the class record's geometry

Kind: impl. Carries FR-4, FR-6, DD-8, DD-9, DD-10, R-1.
Criteria SC-6, SC-7.
Files: `pkg/render/terrain/units.go` MODIFY, `pkg/game/heroart.go` ADD, `pkg/game/frontend.go`
MODIFY, `pkg/game/heroart_test.go` ADD.

Boundary: from a bundle, an archive, a directory and a body name to one loaded class carrying the
composed sheet's frames — and the one call that resolves the party's own, beside the unit load.

Scope fence: the unit loader's own walk, its per-path memo, its tier resolution and its corpse
linking are read or reused, never changed; no existing bundle field moves or changes meaning; the
drawing tier gains one plain-keyed map and no import; nothing here is fatal.

R-1's evidence is that the composite carries the record's descriptor and the sheet's frames
unchanged — the agreement between the two is an install's fact and is not asserted here.

Done when: a loaded body draws the composed sheet's frames and carries the record's canvas, centre,
descriptor, name and corpse link, with no tier slice; an absent sheet, an undecodable one and a name
resolving to no record each produce no body and no error; and the whole gate is clean.

## T4 the picture prefers the body

Kind: impl. Carries FR-4, FR-8, DD-11, DD-12, P-2, P-4, R-2.
Criteria SC-8.
Files: `pkg/game/world.go` MODIFY, `pkg/game/heropicture_test.go` ADD, `pkg/game/sheet_test.go`
MODIFY.

Boundary: one lookup on the map driver, resolved when the map opens from the party the mission
started with, and the one statement in the entity push that prefers it.

Scope fence: the push's death, swing and live arms are not edited and their order does not change;
the step memory, death clock, swing clock, odometer and tier lookup are untouched; the facing
translation, the placement builder and the seam's fields are unchanged; the plain constructor keeps
its behaviour with no override for anybody.

R-2 moves three production call sites and one test helper; a nil argument must leave every entity
drawn exactly as before.

Done when: the member the body was resolved for draws the body's frames and every other entity draws
its own class record's, in the live arm and after a fall alike; a driver built with no override
draws what it drew before; and the whole gate is clean.

## Traceability

| Requirement | Task |
|---|---|
| FR-1 | T2 |
| FR-2, FR-3 | T1 |
| FR-4 | T3, T4 |
| FR-5 | T2 |
| FR-6 | T3 |
| FR-7 | T2 |
| FR-8 | T1, T4 |
| P-1 | T2 |
| P-2, P-4 | T4 |
| P-3 | T1 |
