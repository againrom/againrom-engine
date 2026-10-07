# Plan — an actor can hold items

## Approach

Six tasks, in dependency order: the container and the purse in `pkg/sim` (T1);
their byte form (T2); the transfer (T3); the map's own stock (T4); the pack in
`pkg/ui` (T5); the pack's pictures and the key in `pkg/game` (T6). T1 and T2 are
two tasks rather than one because a world field outside the form is a landable
state — no test in this tree asserts a container reaches the bytes until T2
writes one — and one commit carrying both came out over the task ceiling, which
is exactly what that ceiling is for.

## Facts verified during planning

- `pkg/sim/sack.go`'s `Items` is carried through the constructor, the form and
  the digest and is read by NO consumer that interprets a code — a grep for
  `.Items` outside `pkg/sim` and `pkg/mapload` returns nothing. The
  orchestrator's premise holds, with that one refinement: it is serialized, not
  merely stored.
- `sim.Entity` carries no item, container or load field. Holds.
- `ui.InventorySubject` is `{ID, Figure, Slots [12]}`. Holds.
- `formatVersion = 25` at `pkg/sim/binary.go:336`. Holds.
- `data.Equipment` reaches no simulation type, no byte form and no digest — a
  grep puts every use in `pkg/data`, `pkg/game/hero.go` and
  `pkg/game/inventory.go`. Holds.
- The version-named test the brief warns about is ALREADY version-free:
  `TestUnmarshalRefusesEveryVersionButTheCurrentOne`, renamed by 0106 DD-11
  after three staleness repairs. There is nothing to rename. Its prose
  enumeration of refused versions is the part that goes stale, and DD-9 below
  deals with it.
- `Entities()` documents its result as "a pointer-free value type: mutating the
  result, or anything in it, cannot reach the world it came from". A slice field
  on `Entity` would make that sentence false. DD-1 follows from it.
- `inventoryKey` holds an `InventorySubject` BY VALUE and is compared. The type
  must stay comparable, so the pack is an array. DD-11 follows from it.

## Design decisions

- **DD-1** The container is a slice on `World` parallel to `entities`, not a
  field on `Entity` — `routes`'s exact shape. `Entity` stays pointer-free, so
  `Entities()` keeps its documented guarantee, every existing `Entity` literal
  in the tree keeps compiling, and no per-entity deep copy has to be threaded
  through the constructor's sort. FR-1, P-3.
- **DD-2** `carryFault(codes)` is the ONE predicate, called by the constructor's
  normaliser and by the decoder — `sackFault`'s split, restated for the same
  reason: a constructor that built what the decoder refuses would build worlds
  this package cannot marshal and read again. The constructor DROPS a zero, the
  decoder REFUSES one. FR-4, P-4.
- **DD-3** The purse is `[relationSlots]uint32` on `World`, indexed by the owner
  word as `Relations` is. Not a map: Go's iteration order has no place on a path
  that builds canonical state. Not per entity: money is `Player+0x38`. Owner 0
  indexes cell 0, so no gold is destroyed by an entity outside the roster; an
  owner at or past `relationSlots` has no purse and the transfer refuses before
  it moves anything. FR-5, FR-7.
- **DD-4** A sixth constructor, `NewStockedWorld`, taking `[]Stock{ID, Items}`
  positionally beside the sacks. `NewLootWorld`'s own stated reason applies
  unchanged: most worlds name no stock, an unnamed list materialises to exactly
  what such a world had, and every name funnels through `newWorld`. FR-1, P-4.
- **DD-5** `Carried(id)` answers a fresh copy and an ok; `Purse(slot)` answers a
  value. No setter of either: the only writer is the transfer. FR-6, P-3.
- **DD-6** `TakeSack(id, x, y) error` is the transfer. It looks the sack up
  through the world's own ordered list, appends every code in the sack's order,
  adds the gold to the purse with 32-bit wrap, and removes the sack entry. It
  applies `carryFault` to nothing: a sack's codes were already normalised when
  the world was built, so a zero cannot be there. FR-6, FR-8.
- **DD-7** Two new sections, both after the sack section and before the script
  section: the CARRY section, one record per entity in entity order, each a
  `uint32` count then that many `uint16` codes — the route section's rule, so
  neither carries a section count that could disagree with the entity count —
  and the PURSE section, a fixed `relationSlots` dwords with no length of its
  own, the relation block's rule. Version 26. FR-9, FR-10.
- **DD-8** The decoder refuses a carry record whose declared count overruns the
  buffer, and a code of zero, naming the entity. It reads the carry section for
  exactly `len(entities)` records — the count it already parsed — so a truncated
  section is a length error rather than a silent short read. FR-4, FR-10.
- **DD-9** The refused-version test's doc block loses its enumeration of every
  refused number instead of gaining a twenty-sixth. The enumeration is prose
  restating what the loop computes from `formatVersion`, and it is the same
  label-that-must-be-edited-at-every-bump that cost three repairs before 0106
  removed it from the NAME. Removing it from the body finishes that decision.
- **DD-10** `stockFrom(m)` in `pkg/mapload/fromalm.go`, beside `sacksFrom`,
  walking the same decoded loot. It builds a `UnitID` to index map over
  `m.Units` once, counting duplicates, and emits a `sim.Stock` only for a record
  matching exactly one. `sacksFrom` is unchanged: it already skips a non-ground
  record. FR-11, FR-12.
- **DD-11** `InventorySubject.Pack` is `[invPackCells]*image.RGBA`, an array so
  the type stays comparable for `inventoryKey`, and `invPackCells` stays the one
  spelling of the count — it is already a constant of this package, unlike the
  slot count `Slots [12]` had to embed. FR-13.
- **DD-12** `RenderInventory` passes `s.Pack[i]` where it passed `nil`. That is
  the whole change: `drawInventoryCell` already draws a nil picture as an empty
  cell, so FR-14 costs one argument and no new path. FR-14.
- **DD-13** `buildInventoryPack(src, codes)` in `pkg/game/inventory.go` composes
  the pictures through `data.ItemIconPath` and `loadItemIcon`, both already
  there, and caches by code so a redraw of an unchanged container reads nothing.
  It stops at the array's length and reports the addresses it could not read, in
  carried order — `buildInventorySubject`'s own contract. FR-15, D-3.
- **DD-14** The pack refreshes on the seam that already runs per frame, guarded
  by a comparison of the carried codes against the last set composed, so an
  unchanged container costs one slice compare and no archive read. The figure
  and the twelve slots stay mission-lifetime: 0110 D-7 is about the FIGURE, and
  the pack is the one part of the subject a tick can change. FR-15.
- **DD-15** The key is `G`, a new `Grab` field on the input struct beside
  `Kill`, dispatched where `Kill` is. It calls `TakeSack` for the first party
  member at that member's own cell and does nothing else — no path, no order, no
  queue. An error is discarded: "there is no sack here" is the ordinary answer.
  FR-16.

## Files to touch

- `pkg/sim/carry.go` (new) — `Stock`, `carryFault`, `normaliseCarried`,
  `Carried`, `Purse`, `TakeSack`.
- `pkg/sim/world.go` — two `World` fields, `NewStockedWorld`, `newWorld`.
- `pkg/sim/binary.go` — version, the two sections, encode, decode, the doc map.
- `pkg/sim/binary_test.go` — DD-9's doc block; the pinned bytes and pin digests.
- `pkg/mapload/fromalm.go` — `stockFrom`, and the constructor call site.
- `pkg/ui/inventory.go` — the `Pack` field and one argument.
- `pkg/game/inventory.go` — `buildInventoryPack`.
- `pkg/game/world.go` — the per-frame refresh and the grab dispatch.
- `pkg/ui/app.go` — the `Grab` key.
- Tests beside each.

## Risks

- **R-1** Pinned digests move. Every pin test in `pkg/sim` compares a hard-coded
  digest; adding two sections changes all of them. They are RECOMPUTED, not
  deleted, and the version bump is what makes that lawful — a pin whose form
  version has moved is measuring a different form. T1 owns the recomputation and
  must not touch a pin outside `pkg/sim`.
- **R-2** `pkg/mapload/start.go` is being edited concurrently by another lane
  around the hero recompute. T3 must not touch that file: the stock join lives
  in `fromalm.go`, which is where the loot already is.
- **R-3** The per-frame refresh could read the archive every frame. DD-14's
  compare-then-compose and DD-13's cache are what stop it; a test that composes
  twice and counts reads is the discriminator.
- **R-4** The join is Medium (analysis.md). If a future decode contradicts
  `UnitID`, FR-12's rule means the cost is stock records not placed, never a
  refused load or an item on the wrong actor — a miss is a miss, not a
  mis-assignment.

## Success criteria

- **SC-1** `go test -trimpath -count=1 ./...` green with no game install
  present.
- **SC-2** A version-25 stream well formed for version 25 is refused, and the
  refusal names the version.
- **SC-3** Both roots' campaign corpora are walked and the stock-record count,
  element count and `UnitID` match rate are recorded in verification.md.
- **SC-4** Reverting DD-12's one argument turns an AC-9 test red — the pack
  drawing is witnessed rather than asserted.
- **SC-5** `check-sdd-audit.sh` shows no FAIL and the trailer set is a bijection
  onto T1..T5, with no co-author trailer anywhere on the branch.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1, FR-2, FR-3 | DD-1, DD-4 | SC-1 |
| FR-4 | DD-2, DD-8 | SC-1 |
| FR-5, FR-7 | DD-3 | SC-1 |
| FR-6, FR-8 | DD-5, DD-6 | SC-1 |
| FR-9, FR-10 | DD-7, DD-8, DD-9 | SC-2 |
| FR-11, FR-12 | DD-10 | SC-3 |
| FR-13, FR-14 | DD-11, DD-12 | SC-4 |
| FR-15 | DD-13, DD-14 | SC-1 |
| FR-16 | DD-15 | SC-1 |
