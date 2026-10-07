# 0124 — equip from the pack: plan

Four tasks, in dependency order: the data tier answers *what an item is*, the simulation tier holds
and moves it, the drawing tier reports *which cell was double-clicked*, and the wiring tier joins
the three. Each is one commit and each leaves the tree green on its own.

## D-1 — `pkg/data`: the slot an item names (FR-1, AC-1)

`EquipSlotFor(c ItemCode) (int, bool)` in `equip.go`, beside `validEquipSlot`, which it reuses
rather than re-spelling the bound: the slot is `c.B()`, and it is a slot exactly when
`validEquipSlot` says so. `ItemClassCarried` (14) and a B of 0 fall out of that test with no arm of
their own — 14 is past `EquipSlots` and 0 is below it — so the two named cases are consequences of
one comparison rather than two special cases beside it. Its doc block must say that B is *both* the
class and the slot, because a reader who knows only one of the two will read the function as a
category error.

## D-2 — `pkg/data`: a weapon from its code (FR-2, AC-2)

`WeaponFromCode(c ItemCode, shapes, materials ScaleTable, weapons Collection) (Weapon, error)` in
`weapon.go`, immediately under `ResolveWeapon`. It refuses a code whose `B()` is not
`weaponItemClass`, recovers `shapes.EntryName(c.C())`, `materials.EntryName(c.A())` and
`weapons.EntryName(c.D())`, joins the non-empty ones with a single space in **that** order — which
is the order `ResolveWeapon` consumes them in, shape prefix then material prefix then the row's own
name — and returns `ResolveWeapon(joined, ...)`.

**It re-enters the resolver rather than recomputing** (DD-5). The pay-off is the test: for every
name the resolver answers, `WeaponFromCode(w.Code)` must answer a struct equal to `w` in every
field but `Name`, and that is one loop over a synthetic table rather than a second arithmetic to
keep in step. A row index of 0, an empty entry name, or a name that does not resolve, all come back
as the resolver's own error.

## D-3 — `pkg/sim`: equipment, the command, the byte form

FR-3, FR-4, FR-5, FR-6, P-1, P-3; AC-3..AC-6. A new file `equip.go` beside `carry.go`, and the byte
form's own edits in `binary.go`.

- `EquipSlots = 12` and `equipment [][EquipSlots]uint16` on `World`, parallel to `entities` exactly
  as `carried` is, materialised for every entity by the constructor. Zero is empty (DD-1).
- `Equipped(id) ([EquipSlots]uint16, bool)` — `Carried`'s own shape: an array is copied by
  assignment, so the copy rule costs nothing here.
- `KindEquip`, the next free kind after `KindAttack`. `X` is the container index, `Y` the
  destination slot in the original's own 1..12 numbering. Its arm sits in `step.go`'s command
  switch with the others, and it is TOTAL: an index outside the container, a slot outside 1..12 and
  an entity the world does not hold each leave the world untouched, on the switch's own existing
  rule that a command it cannot apply is ignored rather than an error.
- The move itself, in one function in `equip.go` so the arm is one call: read the code at the
  index, read what the slot holds, write the code into the slot, and **write the displaced code
  back at that same index** — or delete the index when the slot was empty. That ordering is the
  claim's own and it is what makes AC-3 and AC-4 one function rather than two arms.
- The byte form: `formatVersion` to **34** (allocated to this lane; 32 is live, 33 is out to
  another lane). `equipRecordLen = EquipSlots * 2`, a section of
  `len(entities)` such records between the carry section and the purse section (DD-2), encoded in
  entity order, decoded with the purse section's own upfront span check. **No offset above it
  moves.** The pinned-bytes fixtures gain the zeroed section and the version byte changes.

**The version test must not spell the number in its name.** `binary_test.go` names no test after
the live version today and must not start now: `TestThePreviousVersionFormIsRefused` is the shape,
and the previous-version fixture it builds moves to 32.

## D-4 — `pkg/ui`: the double-click and the request (FR-7, FR-8, P-5, AC-7)

All of it in `inventory.go` and `command.go`, tight and local — three other lanes are inside this
package.

- `inventoryPackCellAt(x, y) (int, bool)`: the pack rectangles from `inventoryLayoutRects`, offset
  by `inventoryWindowRect`'s own origin. One geometry, already single-spelled by the click hotfix
  this branch carries under the story.
- `InventoryDoubleClickFrames`, one exported named constant with the authored disclosure on it
  (FR-8). Frames, not time.
- Three fields on `Viewer`: the cell the last first-click landed on, how many frames of the window
  are left, and the pending request. `command` decrements the countdown once per call — which is
  once per map-screen frame — and, inside the swallow branch the hotfix already put at the top of
  it, tests a primary press against `inventoryPackCellAt`. A press on the same cell with the
  countdown still up raises the request; any other press restarts the count on the cell it landed
  on.
- `TakeInventoryEquip() (int, bool)` — reads the request and clears it in one statement, so a
  drained request cannot be drained twice (DD-3).

The swallow branch is where this lives, and that matters: a double-click is only ever a
double-click on a window that is already taking the press, so nothing here can reach a frame the
map is handling.

## D-5 — `pkg/game`: drain, issue, recompute, recompose

FR-9, FR-10, FR-11, FR-12, P-2, P-4; AC-8..AC-10.

- `mapWorld` gains the subject's current `data.Equipment` and the last equipment it composed a
  figure from, beside `invCodes`.
- `equipFromPack()` on `paced()`, beside `refreshPack`: drain the viewer's request; return unless
  there is a subject; read `world.Carried(subject)`; return unless the index is in range;
  `data.EquipSlotFor` the code, return unless it names a slot; `data.WeaponFromCode`, return unless
  it resolves; then append a `sim.Command{Kind: sim.KindEquip, ...}` to `mw.pending`. Every refusal
  is a bare `return` (FR-9).
- In `tick()`, immediately after `sim.Step` and before `settleNotices` (DD-4), `mw.rearm()`: for the
  subject alone, read `world.Equipped`, and when it differs from what was last applied, build the
  `data.Loadout` — `WeaponFromCode` of slot 1, falling back to the party member's own weapon when
  slot 1 is empty (P-4) — call `Hero.Recompute`, and hand the nine numbers to `world.SetCombat`.
  This is `SetCombat`'s one call site in the tree (FR-11).
- `buildInventorySubject` (`inventory.go`) splits: the composition takes a `data.Equipment` and an
  entity id, and the existing entry point derives that equipment from the party member exactly as
  it does today. A second caller — `refreshEquipment`, beside `refreshPack` on `paced()` — passes
  the equipment read back from the world, and pushes the rebuilt figure and slots (FR-12). The
  guard is the same compare shape `refreshPack` already uses, so an unchanged loadout reads no
  archive.

The file header of `pkg/game/inventory.go` says in capitals that it **runs once, at mission open**.
That sentence stops being true here and must be rewritten in the same commit, not left standing.

## Risks

- **R-1** The recompute needs a `Profile`, and a generated character has no shipped row for one —
  the zero profile is what `partyCharacters` already passes, and it reaches only the two pools,
  which nothing here reads. Pass the same zero and say so.
- **R-2** `mw.pending` is truncated by `tick()` before the step. `equipFromPack` runs on `paced()`,
  which calls `paceTo` and therefore `tick` — so it must append **before** the pace, or the command
  waits a frame. Not fatal, but it should be deliberate.
- **R-3** Three other lanes are inside `pkg/ui` and one is inside `pkg/game`. Every edit stays
  inside the functions named above; nothing is reformatted or tidied in passing.

## D-6 — the pick-up log (FR-13, FR-14, FR-15, FR-16, FR-17, AC-11, AC-12, DD-6)

A surface of its own in `pkg/ui`, and one caller in `pkg/game`.

- `pkg/ui/pickup.go`: `PickupRow{Text string, Count int}` crossing in, a `PostPickup([]PickupRow)`
  setter, `PickupDwellFrames` as the one authored constant, and a per-frame decrement that drops
  expired rows. It draws at the top of the view through the font the viewer already holds, on
  `notice.go`'s own "everything is a function of its arguments" rule — a composer taking rows and a
  font, and a presenter reading viewer state, so a test asserts the composed pixels with no window.
- **It takes no input and gates nothing.** No press reaches it, no dismiss, no advance seam. That
  is FR-14 and it is a property of there being no statement anywhere that reads a button here.
- The decrement rides the same per-frame statement the inventory's own double-click count does, so
  there is one frame counter in this package and not two.
- `pkg/game`: `grab()` (world.go) reads the sack's codes **before** `TakeSack` consumes it, since
  the sack is gone afterwards; counts equal codes; names each through the collections the front-end
  already parsed; and posts the rows. A code that names nothing posts its own seven-digit name.

## R-4

`grab()` currently discards `TakeSack`'s error and answers nothing. Reading the sack first means
`grab` has to find it — the world's own sack list, at the subject's cell — and the two lookups must
agree, or the log will name a sack the take did not consume. Read the cell once and use it for
both.
