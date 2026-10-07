# Spec — an actor can hold items

## Terms

**Container** — the thing that holds items. In the original it is one C++ class,
a `CObList` with two dwords bolted on, and a unit and a sack hold the *same* one
(ITEM-CONT-004): the actor's at `actor+0x7c`, the sack's at `sack+0x40`.

**Item code** — the authored `u16` a map's type-8 element carries. Already
decoded and already carried: `alm.LootElement.ItemCode`, `sim.Sack.Items`.

**Stock record** — a type-8 record whose `+0x04` is NON-zero. It is not a sack:
it names an actor that already exists and puts its elements into that actor's
container (ITEM-OWNED-028).

**Purse** — a roster slot's money. Money is `Player+0x38` and is never an item
(ITEM-DROP-008); a pick-up credits the picking actor's owner (ITEM-PICK-009).

**Pack** — the inventory window's area beneath the twelve equipment cells.
0110 drew it as a fixed count of empty cells with nothing behind it.

## Problem

Nothing in this build can HOLD an item. `sim.Sack` carries codes on the ground;
`sim.Entity` has no item field of any kind; `ui.InventorySubject` has no pack
field. 0111's lane ruled that a pick-up is all-or-nothing and therefore needs
somewhere for a whole sack to land. This story builds that place, the one act
that fills it, and the one view that shows it.

## Scope

The container, its byte form, the transfer, the map's own stock, and the pack
drawing. NOT the pick-up order — the walk, the command and the ordering are a
later story's (ITEM-CMD-007 source 3). NOT equipment: equipped is a different
place from carried (ITEM-EQUIP-006) and `data.Equipment` already models the
twelve slots.

## The contract

### The container

- **FR-1** `pkg/sim` holds, per entity, the item codes that entity carries, in
  a slice parallel to the entity list — the shape `routes` already has. It is
  simulation state: a transfer changes it and a save carries it.

  *Folded from hotfix `c9d0a2c` — see `docs/hotfix/ARCHIVE.md#c9d0a2c`.* The
  record states a starting loadout as well as a container, and every world
  rebuild names it — the field set a rebuild must carry is the record's, not
  the caller's memory of it.
- **FR-2** Order is INSERTION ORDER and nothing sorts it. `+0x1c` is the index
  the next `Add` inserts at and `AddTail` is what an index at or past the count
  reaches (ITEM-CONT-004), so appended-at-the-tail is what the original does.
- **FR-3** It is UNBOUNDED in count. No routine of the class compares against a
  limit and neither insert path reads a maximum (ITEM-CONT-004). That clause is
  graded Medium with a named blind spot — a caller that tests the load before
  calling — so this is a disclosure and not a proof.
- **FR-4** A code of ZERO is not an item. Class is bits 8..11 and a class of 0
  resolves to a null the map-load caller SKIPS (ITEM-CODE-029); `data.Equipment`
  reads the same zero as empty. The CONSTRUCTOR DROPS a zero, exactly as that
  caller skips one; the DECODER REFUSES it, because a decoded record is a claim
  about a saved actor and this package never writes one. This is `sackFault` /
  `normaliseSacks`'s own split.
- **FR-5** A world holds a PURSE per roster slot, `relationSlots` of them,
  indexed by the owner word exactly as the relation is.

### The transfer

- **FR-6** One `World` method takes the sack at a named cell into a named
  entity's container. It is ALL-OR-NOTHING: every element in the sack's own
  order, all of the gold, and the sack ceases to exist (ITEM-PICK-009).
- **FR-7** The gold credits the PICKING ENTITY'S OWNER's purse, wrapping at 32
  bits as the sack merge already does. It does not enter the container.
- **FR-8** It has NO ORDER, NO WALK AND NO COMMAND attached. No distance test,
  no ownership test, no capacity test — the routine has none (ITEM-PICK-009).
  It refuses only what it cannot do: an unknown entity, or no sack at the cell.

### The form

- **FR-9** Byte-form version 26 carries the container and the purse. The carry
  section is one record per entity in entity order — the route section's own
  rule — after the sack section; the purse section is a fixed block after it.
  The digest covers both, through the one traversal that already writes them.
- **FR-10** The decoder refuses a version-25 stream and every other version.

### The map's own stock

- **FR-11** `pkg/mapload` puts a stock record's elements into the container of
  the actor it names, and places no sack for it. The join is the record's owner
  word to `alm.Unit.UnitID`, and the entity is that unit's own index.
- **FR-12** A stock record naming no unit, or naming more than one, contributes
  NOTHING and the load still succeeds — `sacksFrom`'s own rule for a record it
  cannot place.

### The pack

- **FR-13** `ui.InventorySubject` carries a pack: one picture per pack cell, a
  fixed-length array like `Slots`, so the type stays comparable and the array's
  own length is the count.
- **FR-14** `RenderInventory` draws each pack cell's picture where it drew an
  empty cell, by the same call. A nil picture is an empty cell, unchanged.
- **FR-15** `pkg/game` composes those pictures from the first party member's
  carried codes, through `data.ItemIconPath` and `loadItemIcon` — the readers
  0110 already built, not a second one — and refreshes them as the container
  changes rather than once at mission open.
- **FR-16** A key takes the sack under the first party member. It is the FR-6
  primitive exposed and nothing more: no walk, no path, no queue.

### Divergences, disclosed

- **D-1** THE LOAD IS NOT BUILT. `+0x20` is Σ(per-unit weight × count)
  (ITEM-STACK-003) and its only consumer is the overload penalty
  (ITEM-LOAD-005, HERO-SPEED-008). A weapon's weight is the Weapons row's slot 3
  (HERO-EQUIP-017); a non-weapon's sits behind a fill no consumer has been read
  for. A load computed here would be a column this project invented, so the
  container carries codes alone and the load is a named seam.
- **D-2** THE STACK COUNT IS NOT BUILT. A stack is one object with a count
  (ITEM-STACK-003); an authored element carries one code and this build stores
  one code. Two of the same code are two entries.
- **D-3** THE PACK DRAWS THE FIRST CELLS ONLY. The container is unbounded and
  the window has a fixed cell count, so a carrier with more items than cells has
  the remainder undrawn. That is a limit of the WINDOW, not of the container —
  the G2 distinction, and the cell count is one edit wide.
- **D-4** `+0x1c`, the next-insert index, is not stored. With every add at the
  tail it is always the count, so storing it would be a second spelling of a
  number already in the form.

## Acceptance

- **AC-1** A world built with an entity carrying codes answers those codes, in
  the order given, with the zeroes dropped.
- **AC-2** A world built with a code of zero among real ones answers the real
  ones alone; a stream carrying a zero code is refused with the entity named.
- **AC-3** Marshal → unmarshal → marshal is byte-identical for a world whose
  entities carry items and whose purses hold gold.
- **AC-4** Two worlds differing only in one entity's carried codes hash
  differently; two differing only in one purse hash differently.
- **AC-5** A version-25 stream, well formed for version 25, is refused.
- **AC-6** The transfer moves every code and all the gold, in sack order, and
  the sack is gone from `Sacks()` afterwards.
- **AC-7** The transfer refuses an unknown entity and a cell with no sack, and
  changes nothing when it refuses.
- **AC-8** Loading a map with a stock record gives the named actor that record's
  codes and places no sack at that record's cell.
- **AC-9** `RenderInventory` on a subject with a pack picture differs from the
  same subject without it, and composing a subject with no pack at all is
  unchanged from 0110's own output.
- **AC-10** The builder composes an icon per carried code, in carried order, and
  reports the address of each icon it could not read.
- **AC-11** The key path performs exactly FR-6 for the first party member and is
  inert when that member stands on no sack.

## Derived properties

- **P-1** A container is per ENTITY and not per group or per owner: an actor
  holds its own, which is what `actor+0x7c` is.
- **P-2** Nothing in `pkg/sim` reads a field OF a code. Zero is tested as a
  value; class and index stay the format leaf's business, as `sim.Sack` says.
- **P-3** The accessor copies: mutating what it returns cannot reach the world.
  `Entities()` stays a pointer-free value type, so the container is NOT a field
  on `Entity`.
- **P-4** Every constructor funnels through one body, so no rule about what a
  world may hold differs by the way it was built.
- **P-5** Composing the window never fails: no code, no icon and no archive
  still compose the frame and the cells.
- **P-6** `pkg/ui` gains no archive and no item knowledge — it receives pictures.

## Constraints

- `pkg/sim` is behind the determinism wall: stdlib only, no floats, no `os`,
  `time` or `math/rand`; its tests import only the standard library.
- Byte-form version 26. 25 is live, 27 is allocated elsewhere.
- Tests are synthetic and read no game install.
- `pkg/ui` may not import `pkg/data` or any archive reader.

## Out of scope

The pick-up ORDER and its command. Equipment. Dropping. Weight and the overload
penalty. Stacking, splitting and merging. Trade, shops and the mission boundary
(ITEM-CARRY-015). Any second character's window.

## Verification mapping

AC-1..AC-5 and AC-7 are `pkg/sim` tests; AC-6 is a `pkg/sim` test over a world
carrying a sack; AC-8 is a `pkg/mapload` test over a synthetic map; AC-9 is a
`pkg/ui` test; AC-10 and AC-11 are `pkg/game` tests. P-1..P-6 are witnessed in
verification.md against the code that establishes them.

## Gate check

`go build ./... && go vet ./... && gofmt -l` clean, `go test -trimpath -count=1
./...` green with no game present, plus `check-no-game-assets.sh`,
`check-doc-budget.sh`, `check-sdd-audit.sh`.
