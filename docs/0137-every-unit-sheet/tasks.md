# Tasks — 0137 every unit states its own character sheet

**Legend.** `impl` — an implementation task. Tasks run in the order given.

---

## T1 — the placement sheet (impl)

**Covers:** FR-1, FR-2, FR-3, FR-4, FR-4a, FR-5, FR-5a; DD-1, DD-2, DD-4, DD-6, DD-7; R-1, R-3.
**Files:** `pkg/mapload/sheet.go` (new), `pkg/mapload/sheet_test.go` (new).

New exported type and function in `pkg/mapload`, keyed by the entity id the world builder mints
for a placement — state that rule from the package that mints it. Resolve each placement with this
package's own resolver; the units arm yields a creature, any other arm reaching an entry yields a
person, and anything else yields no entry in the map. A nil map or a table with neither collection
yields an empty result and no panic.

The value is builtins and fixed-size arrays only, comparable, and names no type of a tier above
this one. It carries the band, the four statistics, the six skill positions, both families and the
experience.

**Done when:** the two arms produce the values FR-3, FR-4, FR-4a and FR-5 require; a test builds a
synthetic units row with ten distinct cells and asserts which family array each block lands in
*and* which skill positions the weapon-kind block reaches (R-1); a test asserts a person placement
carrying equipment and one carrying none state equal families (FR-5a); a test covers the no-entry,
nil-map and empty-table cases.

**Fences:** touch nothing else in `pkg/mapload`. Do not export anything from `pkg/data`. Do not
resolve equipment.

---

## T2 — the band reaches the window (impl)

**Covers:** FR-2, FR-4, FR-7; DD-3, DD-5.
**Files:** `pkg/ui/panel.go`, `pkg/ui/panel_test.go` (or the existing panel test file).

Add the band enumeration and carry it on the character value; the value must stay comparable,
because the panel's refresh key holds one. The skill row's formatter takes the band and prints six
numbers for a person and five for a creature — positions 1 to 5, position 0 not drawn. Every other
row's gate and source is unchanged, the two five-wide family rows included, and an unknown
character still states nothing anywhere.

**Done when:** a subject of each band resolves the skill row to six and to five numbers
respectively; an unknown character still yields no value for every character row; the existing
panel tests pass unchanged.

**Fences:** `pkg/ui` takes no new import. Do not change any row's gate other than the skill row's,
and do not change the layout's rows or labels.

---

## T3 — the driver joins the two (impl)

**Covers:** FR-1, FR-6, FR-7, FR-8; DD-2, DD-8, DD-9; R-2.
**Files:** `pkg/game/panelchars.go`, `pkg/game/world.go`, a test file in `pkg/game`.

Convert T1's value into the window's character, field by field, in `panelchars.go`; merge it with
the party's own lookup so one map holds both, the party written last. Give the party path its band.
Feed the merged lookup to both map-world constructors — the mission door and the plain-map door,
which passes nothing today.

In the draw path, guard the per-slot level write on the person band and leave the experience total
unconditional.

**Done when:** a mission's readout states a character for a placed unit of each band and for each
party member; a party member's character is unchanged in every field but the band; a stepped world
leaves a creature's skill positions equal to its columns while a member's follow his entity's
experience (R-2).

**Fences:** inside the draw path's readout loop, change only the line that writes a skill level and
add only what that guard needs — no other line of that loop, and nothing else in `world.go` beyond
the two constructor arguments and the party path's band. Lanes are editing neighbouring blocks of
that file.

---

## T4 — the tool prints the sheet (impl)

**Covers:** FR-9, FR-10, FR-11; DD-10, DD-11, DD-12, DD-13.
**Files:** `cmd/classdump/sheet.go` (new), `cmd/classdump/databin.go`, `cmd/classdump/campaign.go`,
`cmd/classdump/main.go` (the usage doc), a test file in `cmd/classdump`.

One report function taking a map, a table and the built world's entities, printing one block per
placement in FR-9's order. Call it from the map-dump verb, which already holds all three, and from
the campaign verb's named-map path, which must build the world itself at the ordinary difficulty.
Add the two equipment collections to the campaign verb's table.

Values a built world carries are read off the entity; only the statistics, the two families and the
experience come from T1. Unstated positions print as a dash, with one legend line.

**Done when:** both verbs print the block; a test over a synthetic map and table pins the block's
shape and the dash for an unstated position; the tool's own usage doc states the new section and
the campaign verb's doc no longer claims it prints counts and cells only.

**Fences:** add no third verb and no new flag. Recompute nothing the world already carries.

---

## Traceability

| FR / DD | Task |
|---|---|
| FR-1 | T1, T3 |
| FR-2 | T1, T2 |
| FR-3 | T1 |
| FR-4 | T1, T2 |
| FR-4a | T1 |
| FR-5 | T1 |
| FR-5a | T1 |
| FR-6 | T3 |
| FR-7 | T2, T3 |
| FR-8 | T3 |
| FR-9 | T4 |
| FR-10 | T4 |
| FR-11 | T4 |
| DD-1, DD-4, DD-6, DD-7 | T1 |
| DD-2 | T1, T3 |
| DD-3, DD-5 | T2 |
| DD-8, DD-9 | T3 |
| DD-10, DD-11, DD-12, DD-13 | T4 |
