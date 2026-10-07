# Tasks — 0113-hero-recompute

## T1 — the one recompute *(implementation)*

New `pkg/data/recompute.go`: the four types DD-1, DD-2 and DD-3 name, `Recompute`, `ClampPools`
(DD-7) and the per-slot experience helper. The order is FR-4's and the terms are FR-5 to FR-15; no
step may move. `Protection` and `Resistance` are `[5]int32`, matching `data.UnitDef`'s arrays.

In `pkg/data/hero.go`, reduce `Derive`, `Speed` and `Sight` to one-line accessors over `Recompute`
(DD-5): delete their bodies, keep the load-bearing content of their doc blocks, and leave `capStat`,
`pow11`, `ftol`, `activeSkill` and `cellOr` untouched. `Derive(w)` must return exactly what it
returns today — `pkg/data/hero_test.go` stays as it is and is AC-4's witness.

New `pkg/data/recompute_test.go` witnesses AC-2, AC-3, AC-5, AC-6, AC-7, AC-8, AC-9 and AC-10.

Doc blocks must carry, in the code: the four load-bearing ordering points; that absorption has no
source but armour; the armour seam and its exact missing fact — which `Data.bin` column fills
`Armor::Equip`'s and `Shield::Equip`'s contribution (DD-4); that the pools are produced and not
wired, with the three inputs nobody can state (DD-6); that each family's sixth block slot is filled
by no column (DD-8); and DD-12's unmeasured logarithm margin.

## T2 — a recompute reaches a live entity *(implementation)*

New `pkg/sim/rearm.go` and `pkg/sim/rearm_test.go`, for FR-18 by DD-9.

`type CombatBlock struct` holds, as builtins, the eight numbers a blow reads plus the reach:
`DamageBase`, `DamageSpread`, `ToHit`, `Defence`, `Absorption`, `AttackCharge`, `AttackRelax`,
`AlwaysHits`, `Reach` — the entity's own field names and widths, so the write is a copy and not a
translation.

`func (w *World) SetCombat(id EntityID, c CombatBlock) bool` finds the entity with that id, writes
those nine fields and returns true; returns false and changes nothing when no entity has that id.

It must touch no other field — not health, not speed, not the order block, not the tick — and no
other entity. It is called from nowhere in this story: it is the door `0115` uses.

Its doc block states that it mutates CANONICAL, hashed state, so a caller applies it at a
deterministic point in the frame, and that a caller holding a stale `Entities()` copy must re-read.

Tests witness AC-12: one entity's nine fields move, a second is untouched, an unknown id answers
false and leaves the world's hash unchanged.

`pkg/sim` is behind the determinism wall — stdlib only, no floats, no `os`/`time`/`math/rand`. No
field is added to `Entity`, so the byte form and its version constant are untouched.

## T3 — the full set reaches the info window *(implementation)*

`pkg/ui/panel.go`, `pkg/ui/panel_test.go`, `pkg/game/world.go`, `pkg/game/hero_test.go`, for FR-17
by DD-10 — plus the two row-set pins that move with them, `pkg/ui/sheet_test.go` and
`pkg/game/sheet_test.go`, whose fixtures are extended to the recompute's real numbers, not relaxed.

`ui.UnitCharacter` gains `Experience int`, `Protection [5]int` and `Resistance [5]int`. Add
`PanelFieldExperience = 34`, `PanelFieldProtection = 35` and `PanelFieldResistance = 36`, saying why
the run starts at 34 under this file's own allocation rule; teach `panelText` to state them, a
family as its five numbers space separated; add three rows to `AuthoredPanelLayout` after `ABSORB`,
labelled `XP`, `PROTECT` and `RESIST`.

Gate the three on `UnitCharacter.Known`, as the existing character rows are. Change no geometry,
colour, corner, margin, gap or existing row.

`partyCharacters` calls `p.Hero.Recompute(data.Profile{}, data.Loadout{Weapon: p.Weapon})` **once**
per member and fills the four statistics, the skill, the weapon name, the experience and the two
families out of that one value. The statistics come from the recompute, so the window cannot
disagree with the derivation about a capped one.

Tests witness AC-11: a party member states the three with the recompute's numbers, a placed unit
none.

## Traceability

| Task | Contract | Decisions |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-6a, FR-7, FR-8, FR-9, FR-10, FR-11, FR-12, FR-13, FR-14, FR-15, FR-16 | DD-1, DD-2, DD-3, DD-4, DD-5, DD-6, DD-7, DD-8, DD-11, DD-12, DD-13 |
| T2 | FR-18 | DD-9 |
| T3 | FR-17 | DD-10 |
