# Tasks — 0112-container

**Reading key.** `FR-n`, `D-n` → `spec.md` §The contract, §Divergences. `AC-n`, `P-n` →
`spec.md` §Acceptance, §Derived properties. `DD-n` → `plan.md` §Design decisions. `R-n` →
`plan.md` §Risks. `SC-n` → `plan.md` §Success criteria.

Tasks are ordered, and each leaves `go build ./...`, `go vet ./...` and `go test ./...` green on
its own.

---

## T1 — a world holds what its actors carry *(implementation)*

**Boundary.** The container and the purse exist, are built and are read back. They do not reach
the byte form yet, and nothing moves anything into either.

**Files**

- `pkg/sim/carry.go` — `ADD` (`Stock`, `carryFault`, `normaliseCarried`, `Carried`, `Purse`)
- `pkg/sim/world.go` — `MODIFY` (two `World` fields, `NewStockedWorld`, `newWorld`)
- `pkg/sim/carry_test.go` — `ADD`

**Covers** FR-1, FR-2, FR-3, FR-4, FR-5 · DD-1, DD-2, DD-3, DD-4 · D-1, D-2 (AC-1, AC-2, P-1,
P-2, P-3, P-4)

**Fences.** Do not add a field to `Entity` — DD-1, and `Entities()`'s own doc block says why. Do
not store a load, a weight or a stack count (D-1, D-2). Do not read a field of an item code
beyond testing it against zero (P-2). Do not touch `binary.go` or any pinned digest — T2 owns the form. Nothing outside `pkg/sim`.

**Done when:** a world built with codes answers them in the order given with zeroes dropped; a
world naming no stock answers an empty container for every entity; `Carried` and the caller's
own slice cannot reach each other; a purse is readable per roster slot and an out-of-range slot
answers zero; and `go test ./...` is green tree-wide.

---

## T2 — the container crosses the byte form *(implementation)*

**Boundary.** Version 26 carries the two new sections and the digest covers them. No new
behaviour.

**Files**

- `pkg/sim/binary.go` — `MODIFY` (version 26, the two sections, encode, decode, the offset map)
- `pkg/sim/binary_test.go` — `MODIFY` (DD-9's doc block; recomputed pins)
- other `pkg/sim` `*_test.go` — `MODIFY` only where a pinned digest or version literal moves

**Covers** FR-9, FR-10 · DD-7, DD-8, DD-9 · R-1 · D-4 (AC-2, AC-3, AC-4, AC-5)

**Fences.** Do not take version 27 — another lane holds it. Do not rename
`TestUnmarshalRefusesEveryVersionButTheCurrentOne`: it is already version-free, and DD-9 is
about its BODY. Do not touch any pin outside `pkg/sim`. Do not change what a world holds — T1
settled that. Do not store `+0x1c` (D-4).

**Done when:** marshal/unmarshal/marshal is byte-identical over a world with carried codes and
non-zero purses; two worlds differing only in one entity's codes, and two differing only in one
purse, hash differently; a stream carrying a zero code is refused naming the entity; a
well-formed version-25 stream is refused; and `go test ./...` is green tree-wide.

---

## T3 — a whole sack becomes what one actor carries *(implementation)*

**Boundary.** The transfer primitive, as a `World` method. No order, no walk, no command, no
caller.

**Files**

- `pkg/sim/carry.go` — `MODIFY` (`TakeSack`)
- `pkg/sim/carry_test.go` — `MODIFY`

**Covers** FR-6, FR-7, FR-8 · DD-5, DD-6 (AC-6, AC-7, P-1, P-3)

**Fences.** No distance test, no ownership test, no capacity test — the routine has none
(`ITEM-PICK-009`). No per-item selection: the protocol has none. Do not touch the entity's
position, its order, its state byte or the tick. Do not put the gold in the container (FR-7).
Nothing outside `pkg/sim/carry.go` and its test.

**Done when:** the transfer appends every code in the sack's own order, adds the gold to the
picking entity's owner's purse with a 32-bit wrap, and the sack is absent from `Sacks()`
afterwards; an unknown entity, an owner past the roster and a cell with no sack are each refused
with the world unchanged; and a world that has taken a sack still round-trips byte-identically.

---

## T4 — the map's own stock reaches the actor it stocks *(implementation)*

**Boundary.** A type-8 stock record's elements arrive in the named actor's container at load.
Ground records are untouched.

**Files**

- `pkg/mapload/fromalm.go` — `MODIFY` (`stockFrom`, and the constructor call)
- `pkg/mapload/loot_test.go` — `MODIFY`

**Covers** FR-11, FR-12 · DD-10 · R-2, R-4 · SC-3 (AC-8)

**Fences.** Do not touch `pkg/mapload/start.go` — another lane holds it (R-2). Do not change
`sacksFrom`: it already skips a non-ground record. Do not read the record's coordinates or gold
on the stock arm — the original never does. Do not invent a fallback for a record matching no
unit or more than one: it contributes nothing and the load still succeeds (FR-12).

**Done when:** a synthetic map with one stock record gives the named unit's entity that record's
codes in element order and places no sack at that record's cell; a record whose owner word
matches no unit, and one whose owner word matches two, each place nothing and leave the load
successful; and every existing `pkg/mapload` test is still green.

---

## T5 — the pack area draws what it is given *(implementation)*

**Boundary.** The drawing tier gains a pack and draws it. Nothing supplies one.

**Files**

- `pkg/ui/inventory.go` — `MODIFY` (`Pack` field, one argument in `RenderInventory`)
- `pkg/ui/inventory_test.go` — `MODIFY`

**Covers** FR-13, FR-14 · DD-11, DD-12 · SC-4 (AC-9, P-5, P-6)

**Fences.** `Pack` is an ARRAY, not a slice: `inventoryKey` holds an `InventorySubject` by value
and compares it (DD-11). Do not add a second spelling of the cell count — `invPackCells` is
already the one. Do not import `pkg/data` or any archive reader. Do not change the layout, the
cell size, the frame or the twelve slots.

**Done when:** a subject carrying a pack picture composes differently from the same subject
without one; a subject with an empty pack composes byte-identically to what 0110 composed; a nil
pack picture is still an empty cell; and reverting the one argument in `RenderInventory` turns a
test red.

---

## T6 — the first party member's pack, and the key that fills it *(implementation)*

**Boundary.** The wiring tier composes the pack's pictures from the world's container and gives
the player one key that performs T3's primitive.

**Files**

- `pkg/game/inventory.go` — `MODIFY` (`buildInventoryPack`)
- `pkg/game/world.go` — `MODIFY` (the refresh and the grab dispatch)
- `pkg/ui/app.go` — `MODIFY` (the `Grab` key)
- `pkg/ui/*.go` — `MODIFY` only the input struct the key lands on
- `pkg/game/inventory_test.go`, `pkg/game/world_test.go` — `MODIFY`

**Covers** FR-15, FR-16 · DD-13, DD-14, DD-15 · R-3 · D-3 (AC-10, AC-11, P-5)

**Fences.** Reuse `loadItemIcon` and `data.ItemIconPath` — do not write a second icon reader.
Do not rebuild the figure or the twelve slots per frame: 0110's D-7 stands for those. Do not add
a walk, a path, a queue or an order to the key (FR-16). Do not read the archive when the carried
codes have not changed (R-3). Do not touch `pkg/mapload`.

**Done when:** the builder composes one icon per carried code in carried order, stops at the
array's length, and reports each address it could not read; composing twice over an unchanged
container reads the archive once; the key takes the sack under the first party member and is
inert when that member stands on none; and `go test ./...` is green tree-wide.

---

## Traceability

| Task | FR | DD | AC / P |
|---|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-4, FR-5 | DD-1, DD-2, DD-3, DD-4 | AC-1, AC-2, P-1, P-2, P-3, P-4 |
| T2 | FR-9, FR-10 | DD-7, DD-8, DD-9 | AC-3, AC-4, AC-5 |
| T3 | FR-6, FR-7, FR-8 | DD-5, DD-6 | AC-6, AC-7 |
| T4 | FR-11, FR-12 | DD-10 | AC-8 |
| T5 | FR-13, FR-14 | DD-11, DD-12 | AC-9, P-5, P-6 |
| T6 | FR-15, FR-16 | DD-13, DD-14, DD-15 | AC-10, AC-11 |
