# Tasks — 0124-equip-from-the-pack

**Reading key.** `FR-n`, `AC-n`, `P-n`, `DD-n` -> `spec.md`. `D-n`, `R-n` -> `plan.md`.

Tasks are ordered, and each leaves `go build ./...`, `go vet ./...`, `gofmt` and
`go test -trimpath -count=1 ./...` green on its own. Three other lanes are inside `pkg/ui` and one
inside `pkg/game`: keep every edit inside the functions named, and reformat nothing else (R-3).

---

## T1 — an item code answers its slot, and a weapon comes back out of one *(implementation)*

**Boundary.** `pkg/data` only. Nothing outside the package calls either function yet.

**Files**

- `pkg/data/equip.go` — `MODIFY` (add `EquipSlotFor`, D-1)
- `pkg/data/weapon.go` — `MODIFY` (add `WeaponFromCode` under `ResolveWeapon`, D-2)
- `pkg/data/equip_test.go`, `pkg/data/weapon_test.go` — `MODIFY` or add a sibling `_test.go`

**Covers** FR-1, FR-2 · D-1, D-2 · DD-5 (AC-1, AC-2, P-2)

**Fences.** `EquipSlotFor` must reuse `validEquipSlot` and must not write a second bound, a
per-class arm or a named constant for any particular slot (P-2). `WeaponFromCode` must call
`ResolveWeapon` and must not recompute a scale, a damage byte or a cadence (DD-5). Do not change
`ResolveWeapon`, `WeaponRange` or any existing signature. Do not touch `pkg/sim`, `pkg/ui` or
`pkg/game`. Fixtures are synthetic; no test reads an install.

**Done when:** `EquipSlotFor` answers `n` for every code whose field B is 1..12 and no slot for B
of 0 and 14; and for a synthetic shape/material/weapon table, every name the resolver answers
round-trips — `WeaponFromCode(w.Code)` equals `w` in every field but `Name`. Reverting the field
order in the joined name reddens the round-trip.

---

## T2 — the world holds equipment, and one command moves an item into it *(implementation)*

**Boundary.** `pkg/sim` only. Nothing issues the new kind yet.

**Files**

- `pkg/sim/equip.go` — `ADD` (`EquipSlots`, the move, `Equipped`)
- `pkg/sim/world.go` — `MODIFY` (the `equipment` field, materialised by the constructor)
- `pkg/sim/step.go` — `MODIFY` (`KindEquip` and its arm)
- `pkg/sim/binary.go` — `MODIFY` (version **34**, the section between carry and purse)
- `pkg/sim/equip_test.go` — `ADD`; `pkg/sim/binary_test.go` — `MODIFY` (fixtures)

**Covers** FR-3, FR-4, FR-5, FR-6 · D-3 · DD-1, DD-2 (AC-3..AC-6, P-1, P-3)

**Fences.** **Version 34 and no other** — 32 is live and 33 is out to another lane. Move no offset above
the new section. Put the version number in fixtures and comments, **never in a test identifier**.
Zero is empty in a slot and is not refused by the decoder, unlike a carried code. Stdlib only;
no float, no clock, no new `Entity` field; no `SetCombat` call here.

**Done when:** `[a,b,c]` with slot 1 empty, given `KindEquip{X:1,Y:1}`, becomes `[a,c]` with slot
1 = `b`; with slot 1 = `d` it becomes `[a,d,c]`; an absent entity, an index past the container and
a slot outside 1..12 each leave the bytes identical; a set world round-trips byte-identically; a
version-32 buffer is refused; two worlds differing in one slot hash differently.

---

## T3 — a double-click on a pack cell raises one request *(implementation)*

**Boundary.** `pkg/ui` only, and inside `inventory.go` and `command.go` alone. Nothing drains the
request yet.

**Files**

- `pkg/ui/inventory.go` — `MODIFY` (`inventoryPackCellAt`, `InventoryDoubleClickFrames`,
  `TakeInventoryEquip`, D-4)
- `pkg/ui/command.go` — `MODIFY` (the count and the press test, inside the existing swallow branch)
- `pkg/ui/viewer.go` — `MODIFY` (three fields, in the existing inventory field block)
- `pkg/ui/invequip_test.go` — `ADD`

**Covers** FR-7, FR-8 · D-4 · DD-3 (AC-7, P-5)

**Fences.** The threshold is **frames**; do not import `time` and do not read a clock. One named
constant, one place. Do not touch `panel.go`, `flow.go`, `app.go` or the map-opener's result list
— no seam is widened by this task (DD-3). Do not name an item code, an archive or a definition
table anywhere in `pkg/ui` (P-5). Do not disturb the swallow branch's existing behaviour: a press
outside the window still reaches the map.

**Done when:** two presses on one pack cell within the threshold raise exactly one request and
draining it twice yields nothing the second time; the same two presses one frame beyond the
threshold raise none; two presses on different cells raise none; a double-click on the figure box
raises none.

---

## T4 — the wiring: drain, issue, recompute, recompose *(implementation)*

**Boundary.** `pkg/game` only. Every tier below is already in place.

**Files**

- `pkg/game/world.go` — `MODIFY` (`equipFromPack`, `refreshEquipment` on `paced`; `rearm` in
  `tick` right after `sim.Step`; two `mapWorld` fields, D-5)
- `pkg/game/inventory.go` — `MODIFY` (the composer takes a `data.Equipment`; rewrite the header's
  "runs once, at mission open")
- `pkg/game/equip_test.go` — `ADD`

**Covers** FR-9, FR-10, FR-11, FR-12 · D-5 · DD-4 · R-1, R-2 (AC-8, AC-9, AC-10, P-4)

**Fences.** `SetCombat` is called from `tick` and nowhere else — not from a draw path, not from
`paced`, not from an input handler (FR-11). The equip reaches the world only as a `sim.Command` in
`mw.pending` (FR-10). Every inapplicable case is a bare `return` (FR-9). Pass the zero
`data.Profile`, as `partyCharacters` does, and say why (R-1). Append before the pace (R-2). Do not
touch `pkg/ui`, `pkg/sim`, `pkg/data` or `pkg/mapload`.

**Done when:** a synthetic mission whose subject carries a resolvable weapon code, given a
double-click on that cell, ends the next frame with the entity's `DamageBase` at the value
`Hero.Recompute` gives for that weapon and not at its old one — and reverting the `SetCombat` call
reddens that on the number. An empty cell and a non-weapon code leave the bytes identical.

---

## T5 — the pick-up log *(implementation)*

**Boundary.** A new surface in `pkg/ui` and one caller in `pkg/game`.

**Files**

- `pkg/ui/pickup.go` — `ADD` (`PickupRow`, `PostPickup`, `PickupDwellFrames`, compose, present)
- `pkg/ui/viewer.go` — `MODIFY` (the rows and countdowns in one field block; draw them)
- `pkg/game/world.go` — `MODIFY` (`grab` reads the sack, counts, names, posts; R-4)
- `pkg/ui/pickup_test.go`, `pkg/game/pickup_test.go` — `ADD`

**Covers** FR-13, FR-14, FR-15, FR-16, FR-17 · D-6 · DD-6 · R-4 (AC-11, AC-12)

**Fences.** It takes **no input**: no press, no dismiss, no advance seam, and it gates no arm
(FR-14, AC-12). Do not widen `pkg/ui/notice.go` or reuse the notice box (FR-15). The dwell is
**frames** in one named constant; do not import `time` for it. `pkg/ui` names no item code, archive
or definition table — a row arrives as text and a count (FR-16 is the caller's job). Do not touch
`pkg/sim`, `pkg/data` or `pkg/mapload`.

**Done when:** a pick-up of three codes, two equal, posts two rows at quantities 2 and 1; after the
dwell no row remains and the composed picture is empty; a press over where a row was drawn still
selects and orders exactly as with no rows. Reverting the decrement leaves a row on screen forever
and reddens the dwell test.

---

## Traceability

FR-1, FR-2 -> T1. FR-3, FR-4, FR-5, FR-6 -> T2. FR-7, FR-8 -> T3. FR-9, FR-10, FR-11, FR-12 -> T4. FR-13, FR-14, FR-15, FR-16, FR-17 -> T5.
D-1, D-2 -> T1. D-3 -> T2. D-4 -> T3. D-5 -> T4.
AC-1, AC-2 -> T1. AC-3..AC-6 -> T2. AC-7 -> T3. AC-8, AC-9, AC-10 -> T4. AC-11, AC-12 -> T5.
P-1, P-3 -> T2. P-2 -> T1 and T2. P-4 -> T4. P-5 -> T3.
DD-1, DD-2 -> T2. DD-3 -> T3. DD-4 -> T4. DD-5 -> T1. DD-6 -> T5.
R-1, R-2 -> T4. R-3 -> every task, by boundary. R-4 -> T5.
D-6 -> T5.
