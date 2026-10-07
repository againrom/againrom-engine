# tasks — 0128 a person wears his whole row

T1 → FR-2, FR-2a, FR-2b, FR-3, P-3, P-5, DD-4, DD-5, DD-6, DD-7. T2 → FR-1, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9,
P-1, P-2, P-4, DD-1, DD-2. T3 → FR-10, DD-3. T4 → FR-11, DD-3.

## T1 — the shared parse and the two new resolvers (plan D-1, D-5, D-6)

`pkg/data`. Add the walk of FR-2a and FR-2b and make it the ONE parse: `takePrefix`/`hasWordPrefix`
in `pkg/data/weapon.go` are replaced by it, and `ResolveWeapon` calls it (a weapon takes no implied
shape word). Add `ResolveShield` and `ResolveArmor` beside it, each returning a small type carrying
Name, Row and Code — plus, for armour, its Slot column, which is param **4** of the row. A shield
drops a trailing ` Shield` from the residue before its lookup; a shield's code class is **2**, an
armour's is its Slot. Refuse a residue naming no row. Do NOT refuse a Slot out of range: return the
value and let the caller read it (plan D-1). Two landed tests assert the authored rule the walk
replaces and are corrected with it (DD-6): `pkg/data/weapon_test.go`'s word-boundary case, and
`pkg/game/hero_test.go`'s fixture, whose shape table must carry the shape word its weapon name
actually leads with and whose row must be named as an install names it. No other test may move. New
tests for FR-2a's descending walk, its positional rebuild on an out-of-order name, FR-2b both ways,
and P-3's totality on nil tables.

## T2 — the loadout, in the loader (plan D-2)

`pkg/mapload`. `Table` gains the armour and shield collections; `pkg/game/table.go` fills them off
the same walk the other three come from. Replace `firstWeapon` in `pkg/mapload/spawn.go` with one
function turning a row's ten cells into a worn set (`[sim.EquipSlots]uint16`) plus a container
overflow, per FR-1, FR-3, FR-4, FR-5, FR-6, FR-7 and FR-9; it still returns the weapon pointer the combat
fold reads. Widen `spawnBlock` and the placement build in `pkg/mapload/fromalm.go` so both reach
`sim.Stock`. Extend the `NPC`-template gate to slots 1 and 2 (FR-8). Keep `carriable`'s existing
refusal of a non-item weapon. `pkg/mapload/human_test.go`'s "first cell that resolves" test asserts
the humans-band search FR-1 replaces, down to its name; rewrite and rename it to the positional rule.
The units band keeps its own search, which is a different arm and is not this story's. Tests: AC-1
through AC-5, AC-7, and P-4 with each collection absent in turn.

## T3 — the panel row (plan D-3)

`pkg/ui`. Add a panel field for the worn set — the next free field number, which is 30 — a
`[]string` member on the panel subject holding the already-formatted names, and a row in
`AuthoredPanelLayout` labelled `WORN`, placed directly under the weapon row. An empty list states
nothing and contributes no row, which is the layout's existing rule. Value formatting: the names
joined by `, ` in slot order. `pkg/ui/sheet_test.go`'s field census must gain the new field; **rename
it so its name no longer spells the count** and take the count from the census itself, because a name
carrying a number is a name that goes stale every time a row is added. Test AC-8 both halves: a subject with names states them, and a subject
without composes exactly what it composed before.

## T4 — the instrument (plan D-4, D-7, D-8)

`cmd/`. A new tool reading a lawful install (`-assets`, or `AGAINROM_ASSETS`) that walks the shipped
person collection and prints, per cell class: cells used, cells resolved, cells refused, and where
each resolved cell landed — worn in slot n, or carried. Print every anomaly in full and the ordinary
rows as a bounded sample; it writes no game data anywhere. Also make `cmd/paneldump` fill the panel
subject's worn list, turning each nonzero worn code into a row name through the collection its slot
names — slot 1 the weapons collection, slot 2 the shields collection, any other slot the armours
collection. No test reads an install; the tool's own output is `verification.md`'s evidence.
