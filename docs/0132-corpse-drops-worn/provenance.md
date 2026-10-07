# 0132-corpse-drops-worn — provenance

**AMBER threshold: High.** This story changes hashed simulation state, so every clause the
contract takes from research is carried by a High grade or is taken at its narrowest reading and
named below as ours.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — every worn field reaches the container before the sack is built | `ITEM-CORPSE-034` | High for the path: the dispatch `CALL dword ptr [EDX + 0x44]` is a named instruction, both possible targets are fixed by a raw dword scan over the whole image, and the humanoid target's loop was read whole |
| FR-1 — which fields the strip covers | `ITEM-CORPSE-034` (the array is thirteen dwords, used 1..12), `ITEM-EQUIP-006` (two base-actor fields at `actor+0x74`/`+0x78`, twelve more at `actor+0x198 + 4i`) | High. `ITEM-EQUIP-006` left `+0x19c` and `+0x1a0` Unknown; `ITEM-CORPSE-034` closes them — ordinary slots, stripped like the rest |
| FR-2 — the strip runs after the two base-actor fields and before the sack | `ITEM-CORPSE-034`: the four instructions sit between the weapon arm and the `"NPC"` find, and the branch target is pinned by two `rel8` in its own listing | High |
| FR-2 — the strip's own order is ascending | `ITEM-CORPSE-034`: counter set to 1, compared against 13, `JGE` to exit | High |
| FR-2 — slot 2 then slot 1 precede it | `ITEM-DEATH-012` | High for the positive clauses; the row's completeness warrant is retracted, which is what this story exists to repair |
| FR-3 — a body that wore anything leaves a sack even carrying nothing | `ITEM-CORPSE-034`, stated in the row: the strip precedes the emptiness test at `L04753` | High |
| FR-4 — an empty field is passed over rather than dropped as a zero | `ITEM-CORPSE-034`: humanoid `vt+0x40` returns 0 for a null argument and the append returns before storing when its item is 0 | High |
| FR-5 — the sack merges per cell rather than a second sack appearing | `ITEM-SACK-010` | High |
| FR-6 — a planted sack stays until it is taken | `ITEM-SACK-011` | High for the two stub slots and the absence from the tick list |

## Ours by choice

| Statement | Why it is ours |
|---|---|
| Slot 1 is dropped unconditionally | The original gates it on the weapon's `Data.bin` parameter 15 (`sutableFor`, `ITEM-DEATH-012`, `ITEM-SUIT-035`). That column lives in `pkg/data`, which the determinism wall puts out of `pkg/sim`'s reach. Carried in from `0123` unchanged and disclosed again here because this story widens the set around it |
| No template-name suppression | `ITEM-DEATH-012` deletes the whole container when the template's name contains `"NPC"`, and `ITEM-CORPSE-034` adds that a suppressed template's armour is destroyed with it. No entity in this tier carries a name, and giving one a suppression bit would widen the byte form. So a suppressed body here drops what it wore. Pre-existing for the container and the two base slots; **this story widens the divergence to the ten armour fields** |
| Array indices 1 and 2 have no representation | `ITEM-EQUIP-006`'s take-off arm addresses `3 <= slot <= 12` only, so this tree's twelve slots are the two base-actor fields plus array indices 3..12. Nothing in this tree can fill `+0x19c`/`+0x1a0`, so nothing drops out of them |
| Nothing is recomputed when a field empties | This tier holds a combat block as state and computes it nowhere (`SetCombat`); the existing death path already empties two fields without recomputing. Stripping ten more changes no derived number here |
| The order two bodies dying on one cell in one tick append in | No source read here says what the original does with a same-tick, same-cell pair. The contract fixes the property instead of the number: the order is a consequence of the tick's own resolution order and of nothing unordered |
| The drive tool's report shape | `missionrun`'s new lines are ours: no source describes a developer tool |

## Open

- What a worn piece contributes to defence or absorption, and therefore what leaving the corpse
  costs it. Undecoded; deliberately assigned no meaning here.
- Whether the player can equip a dropped armour piece. Out of scope, and no claim is read for it.
- `ITEM-CORPSE-034`'s **Medium** clause — that the strip is the *only* route by which a worn piece
  reaches a container. Not relied on: the contract says the strip reaches the container, never that
  nothing else could.

## Removed

| Dropped | Why |
|---|---|
| A gate on the dropped piece's class or slot | No source asserts one. `ITEM-CORPSE-034` reports all 30 shipped `Armors` rows carrying `Slot` in 1..12, so no shipped piece is outside the strip's range and a gate would discriminate nothing |
| Gold at death | `ITEM-DEATH-012` rolls it from three `Data.bin` parameters for `typeID > 0x40`. Out of this story's scope and already on record elsewhere |
