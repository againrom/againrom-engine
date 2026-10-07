# 0151 — analysis

Seven defects reported by the owner on 2026-08-13. This file records what he observed and where each
defect is believed to live. The locations are premises, not conclusions: each was found by reading
the tree from the orchestrator seat, and none was reproduced under a debugger or on screen. A lane
verifies the evidence named beside a location before building on it.

## Reported defects

1. **The weapon is not the topmost layer of the doll.** Boots cover the sword on the character doll
   and on the doll of a unit wearing clothing.
2. **Gloves and shoulder pieces are drawn on one side only.** Both sides are expected.
3. **The armour layer order is unverified.** Chain mail is expected under a cuirass, and a weapon or
   staff above every other layer.
4. **A worn item cannot be taken off by double-clicking it.** The only route to the pack today is
   replacement through the inventory. The worn box is the required surface; the doll itself may be
   skipped.
5. **No popup states an item's characteristics.** Expected in the pack grid and in the worn box,
   including the item's name.
6. **An item is named by its digits.** Picking up armour prints a code of the form `102430430`
   instead of a name.
7. **A moving unit falls below corpses and sacks.** A unit standing on a corpse or a sack draws
   above it, which is correct. A unit that begins to move north draws below it.

## Believed locations

| Defect | Location | Evidence for the belief |
|---|---|---|
| 1, 3 | `pkg/game/figures.go:103` `composeUnitFigure`; `pkg/game/inventory.go:159` `composeInventorySubject` | Both paint one layer per occupied slot in ascending slot number. `composeUnitFigure`'s own doc states "in ascending slot order"; the loop is `for n := 1; n <= data.EquipSlots; n++`. Paint order is therefore slot number order, and nothing states that slot number is layer order. The two functions are separate sites with the same choice, and `composeUnitFigure`'s doc records that the second was written rather than factored out |
| 2 | not located | Two candidates, neither checked: the twelve slots carry one code per pair rather than one per side, or `data.ItemFigureLayerPath` (`pkg/data/itemcode.go:146`) addresses one sprite per pair and the second side is a separate address |
| 4 | `pkg/ui/inventory.go:830` `InventoryDoubleClickFrames`, `pkg/ui/command.go:649`, `pkg/ui/invequip_test.go` | The double-click gesture already exists and already raises an equip request from the pack. What is absent is the same gesture on the worn box in the opposite direction |
| 5 | `pkg/ui/inventory.go` | The window draws the pack grid and the worn box. No hover or press path produces a text box over either |
| 6 | `pkg/game/world.go:2386` `itemName` | It returns `code.Name()` — the code's digits — whenever `data.WeaponFromCode` fails, and armour does not resolve through `WeaponFromCode`. `pickupRowsFor` (`world.go:2359`) packs that string into `ui.PickupRow.Text`, which is the line the owner reads |
| 7 | `pkg/render/terrain/structures.go:794` `DepthOrder`, called from `pkg/ui/statics.go:259` | Entities and sacks are interleaved by row, sack first at a tie (0111 DD-2). The tie rule is why a unit standing on a sack draws above it. If an entity's placement row follows its destination cell while its drawn position is still interpolating, a northward move places it in an earlier row than the sack it is leaving, which draws it first and therefore below |

## Not looked at

- Whether `research/claims/` carries the meaning of the twelve equipment slots, and whether the
  original's own doll composition order is decoded.
- Whether a corpse is an entity or an object in the depth list. Defect 7 names corpses and sacks
  together; only the sack stream is identified above.
- Whether defect 3's expected order is decoded anywhere or is authored.

## Process

The story is implemented first and its SDD artifacts are written afterwards, on the owner's
instruction of 2026-08-13. Implementation commits carry `SDD-Task: 0151-layers-and-names/T<n>`
trailers from the start, so the bijection the audit checks is complete when the artifacts land.

## Later reports

Four defects were reported after the first pass:

1. **The item popup has no background.** Its text is drawn directly over the inventory and can be
   lost against the picture beneath it.
2. **Weapon damage is stated as a sum.** The popup shows `base + spread`, although the visible
   quantity is the roll's lower and upper bounds.
3. **The first moving-depth rule regresses other crossings.** Holding every crossing at its origin
   row puts a unit walking south onto a sack's cell below that sack, and does not state the sideways
   case.
4. **Russian item names draw two wrong glyphs per letter.** The name loader appears to turn the
   game's bytes into UTF-8 before the font path performs its own byte conversion.

The first three beliefs came from reading the landed implementation and its tests. The fourth came
from the rendered symptom and the two conversion sites; no file byte was changed to test it.
