# 0151 — plan

## Approach

Keep each rule at the lowest tier that owns it. `pkg/data` describes item codes and figure order;
`pkg/formats/itemname` pairs the two name streams without interpreting text; `pkg/sim` owns the
unequip transition; `pkg/game` joins definitions and simulation state into display text and
commands; `pkg/ui` owns hit testing, depth presentation and popup pixels. The map figure and
inventory doll share the ordering function, while retaining separate composition loops for their
different outputs.

## Facts verified during planning

- Both figure composers painted occupied slots in ascending numeric order. Each already skipped a
  missing sheet and retained the rest of the figure.
- The map entity's simulation cell is its destination during interpolation. Its non-zero step is the
  destination-minus-origin delta for the whole crossing.
- The item display path had weapon recovery and a seven-digit fallback but no stored-name table.
  The font already converts each string byte according to the selected install.
- The pack already had independent double-click state and a one-shot equip request. Equipment and
  carried stacks were already canonical simulation fields and already contributed to the digest.
- Equipment-derived combat values were recomputed after equip, and inventory figure/pack refreshes
  already occurred after the deterministic advance.
- The pickup text composer intentionally produced bare text. The inventory panel already supplied
  the fill, border and font colours needed by a separate popup.

## Design decisions

**DD-1 — One data-level function returns the complete slot order.**

`data.FigureDrawOrder` returns a permutation of all twelve slots. It places the decoded non-mage
primary sequence first, the authored slot 3 next, and the two held slots last in the order selected
by `FigureHeldLast`. Both composition sites iterate this result.

Rejected: separate literal orders in `pkg/game`. They would make agreement between the map figure
and doll a convention rather than a tested boundary. Rejected also: an ordered list containing
primary and secondary draw operations. The current product contract paints the paired sheet beside
its primary and does not reproduce the separate mage programme.

**DD-2 — Held-layer selection reuses the existing body-name derivation.**

`FigureHeldLast` calls `HeroBodyFor` with the same body list and equipment already used to derive an
appearance. It compares the result with the six literal body names. A nil list, empty weapon slot or
out-of-range row follows the ordinary non-match arm and selects slot 2.

Each composer reads the body list from its entry source. The map composer is bounded by its figure
cache; the inventory composer by its equipment-change refresh. Rejected: storing another copy on
`Viewer`, which would move game definitions into the UI tier.

**DD-3 — Secondary-sheet addressing stays beside primary-sheet addressing.**

`pkg/data` owns the four-slot set and the path substitution from `primary` to `secondary`. Each
composer asks for the secondary only for those slots and paints it immediately after the primary.
A missing secondary is handled by the same miss path as a missing primary.

Rejected: probing `secondary` for all twelve slots. Archive absence would then determine slot
semantics at runtime and add eight known misses to every fully occupied composition.

**DD-4 — Item-name parsing is a format leaf that preserves bytes.**

`pkg/formats/itemname.Parse` splits the text on LF, removes a preceding CR, reads little-endian
`u16` keys and walks only the shared prefix. Integer division ignores an odd final key byte. Map
assignment makes the last non-empty occurrence of a duplicate key win. A retained line is converted
directly from its byte slice to a Go string; no character decoder is imported. `internal/archtest`
registers the new leaf and holds it to no external format dependency.

Rejected: decoding to Unicode in the format package. The renderer indexes one glyph per byte and
already owns language conversion. Rejected: matching the original's unbounded line access. A short
input loses names without making the game reader unsafe.

**DD-5 — The loaded table joins definitions and naming in the game tier.**

`game.ReadItemNames` reads the two `main/text` addresses and converts the raw map to
`data.ItemNames`. `LoadDefinitions` stores it on `mapload.Table`, beside the collections needed by
fallback resolution. `itemName` checks stored names, then the existing weapon resolver, then the
code's digits. Missing inputs return no table and no fatal error.

Rejected: putting archive paths or VFS reads in `pkg/data` or `pkg/formats/itemname`; both are leaf
contracts over values handed in by a caller. Rejected: replacing fallback naming. Most possible
codes have no stored line, so a miss must remain visible.

**DD-6 — Game code composes semantic popup lines; UI code composes pixels.**

`pkg/game/iteminfo.go` builds lines from the selected code and definition table. Slot and pack line
arrays are parallel to the existing slot and pack pictures, and refresh from the same equipment or
stack snapshot as those pictures. `pkg/ui` hit-tests those parallel arrays and knows nothing about
item codes, weapon tables or armour tables.

Rejected: resolving definitions in `pkg/ui`, which would reverse the package DAG. Rejected: handing
pre-rendered popup images from `pkg/game`, which would move cursor placement and panel styling out
of the UI tier.

**DD-7 — The hover popup has its own framed composer.**

`composeItemPopup` uses the pickup text face/shadow measurements over an inventory-coloured framed
canvas. It does not add a frame to the shared bare-text composer. `itemPopupPresent` owns all refusal
conditions: cursor observed, an info-bearing cell and a usable composed picture. Presentation is
recomputed for the hovered lines; the uploaded texture is reused while dimensions agree. The final
draw pass places it above all existing elements. Each coordinate is reduced to the far-edge fit and
then floored at zero; a popup larger than the view starts at zero on that axis and may overflow.

Rejected: widening `composeTextLines` with a frame option. That function is the pickup log's plain
presentation; a mode flag would let one caller silently change the other's product.

**DD-8 — Unequip is one deterministic command with a UI request seam.**

The worn box gets its own double-click cell, countdown and one-shot request, independent from the
pack fields. `pkg/game` drains that request before the frame's advance and translates zero-based UI
index to the simulation's one-based slot. `KindUnequip` clears the slot, appends one code and applies
the existing container fold. Invalid entity and slot cases are handled inside the step without
partial mutation. The existing post-step refresh and combat re-derivation are widened to carry the
new state to the screen.

Rejected: moving the item directly from UI or game code. That would bypass command ordering,
replay and the digest. Rejected: checking occupancy in hit testing; geometry should not duplicate
the simulation's totality rule.

**DD-9 — Moving depth changes only the placement row used by ordering.**

`entityLayer` reconstructs origin as `destination - step` and raises the placement's depth row to
the greater of the two. The sprite anchor, interpolated top-left, fog decision, simulation cell and
shadow inputs remain unchanged. Existing tie tiers then place a living unit after a sack or corpse.

Rejected: moving the simulation cell back to origin or changing global tie priorities. Either would
change more than a crossing entity's row and could regress stationary ordering.

**DD-10 — No serialized shape changes.**

The command uses equipment and carried-stack fields already in the canonical form. Display names,
popup lines and render ordering stay outside `pkg/sim`. No format version is allocated.

Rejected: persisting popup or name state. Both derive from definitions and current equipment and
would make a display concern alter world hashes.

## Files to touch

| Area | Paths and intent | Requirements |
|---|---|---|
| Architecture and format | ADD `pkg/formats/itemname/itemname.go`, `pkg/formats/itemname/itemname_test.go`; MODIFY `internal/archtest/dag.go` | FR-3, FR-6 |
| Data rules | MODIFY `pkg/data/equip.go`, `pkg/data/equip_test.go`, `pkg/data/itemcode.go`, `pkg/data/itemcode_test.go`; ADD `pkg/data/itemname.go`, `pkg/data/itemname_test.go` | FR-1, FR-3 |
| Definition join | MODIFY `pkg/mapload/spawn.go`, `pkg/game/table.go`; ADD `pkg/game/itemnames.go`, `pkg/game/itemnames_test.go`; MODIFY `pkg/game/world.go`, `pkg/game/pickup_test.go` | FR-3, FR-4 |
| Figure composition | MODIFY `pkg/game/figures.go`, `pkg/game/figures_test.go`, `pkg/game/inventory.go`, `pkg/game/inventory_test.go` | FR-1 |
| Popup semantics | ADD `pkg/game/iteminfo.go`, `pkg/game/iteminfo_test.go`; MODIFY `pkg/game/world.go` | FR-4 |
| Unequip simulation and join | MODIFY `pkg/sim/equip.go`, `pkg/sim/equip_test.go`, `pkg/sim/step.go`, `pkg/game/world.go`, `pkg/game/equip_test.go`, `pkg/game/rearm_test.go` | FR-5, FR-6 |
| Inventory interaction and popup pixels | MODIFY `pkg/ui/command.go`, `pkg/ui/inventory.go`, `pkg/ui/overlay.go`, `pkg/ui/pickup.go`, `pkg/ui/viewer.go`; ADD `pkg/ui/invunequip_test.go`, `pkg/ui/itempopup.go`, `pkg/ui/itempopup_test.go` | FR-4, FR-5 |
| Crossing depth | MODIFY `pkg/ui/overlay.go`; ADD `pkg/ui/depth_move_test.go` | FR-2 |

## Risks

**R-1 — One compositor could retain the old order.** Mitigation: both call the same permutation and
each has a pixel-overlap test that identifies the last layer (FR-1).

**R-2 — Correct Russian bytes could be converted twice.** Mitigation: the format test uses byte
values from both moved source blocks and requires byte identity; a live run uses the selected
install's existing font path (FR-3, P-3).

**R-3 — A hover could share or mutate click state.** Mitigation: hover reads only cursor position
and parallel info arrays; worn and pack click windows have an independence test (FR-4, FR-5, P-4).

**R-4 — Unequip could clear a slot without returning the item, or leave stale combat values.**
Mitigation: simulation tests compare both equipment and container, whole-byte no-op tests cover
refusals, and a game test removes armour with non-zero contributions (FR-5, P-5).

**R-5 — Fixing northward depth could regress southward movement.** Mitigation: derive the later row
rather than privileging one endpoint, with directional sacks and a corpse as separate fixtures
(FR-2, P-2).

## Success criteria

**SC-1** Named figure-composition tests prove the shared permutation, both held-order arms and the
paired-sheet boundary; directional depth tests prove north, south, sideways, corpse and stationary
ordering.

**SC-2** Name-table and popup tests prove positional pairing, byte preservation, fallback priority,
damage bounds, decoded class lines, hover refusals and the framed background. Six names from the
lawful Russian install render legibly through the selected install's font path.

**SC-3** Unequip tests prove move, merge, isolation, invalid-command byte identity, round trip,
double-click boundaries and combat recomputation.

**SC-4** The repository build, tests and SDD gates are clean; the implementation range deletes no
file.

## Traceability

| Requirement | Decisions | Success criteria |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-3 | SC-1 |
| FR-2 | DD-9 | SC-1 |
| FR-3 | DD-4, DD-5 | SC-2 |
| FR-4 | DD-6, DD-7 | SC-2 |
| FR-5 | DD-8 | SC-3 |
| FR-6 | DD-4, DD-8, DD-9, DD-10 | SC-2, SC-3, SC-4 |
