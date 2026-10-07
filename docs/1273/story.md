# Mod clothing layers, inventory window first

## Intent and authority

A mod item can be worn together with the item in another slot: a gambeson under body armour, boots over the legs, a cloak over the body. Authority is owner direction (`review/mod-milestone/MOD-LAYOUT.md`, `docs/1272/story.md`). The twelve equipment slots stay the SAV model. Both items show on the inventory paperdoll in the right Z order; defence and absorption sum through the loadout; SAVE and LOAD carry the layers in the `AgainromMods` leaf; LOAD without the mod refuses. The map figure is unchanged. Without a mod, every hash, SAV byte, scenario and release witness is unchanged. ROM1 differences are rows `DIV-1870` to `DIV-1873` in `docs/divergences/mods.md`; `DIV-1866` is reworded.

## As built

Data (`pkg/mod/items.go`). Three keys on an `[[item]]`:

- `layer`: `under`, `main` (no layer, the default) or `over`.
- `anchor`: a slot name; the item's own slot by default. It needs `layer = "under"` or `"over"`.
- `figure`: the English name of an original armour or shield item whose worn picture the layer borrows. The borrow crosses class and sex directories, since the shipped sheets are per class (`resolveFigure`).

A bad value is refused with mod, file and line. `LoadStrings` now reads through `ReadInside`, and `SetMods` keeps the already set items.

Placement (`pkg/mapload/layers.go`). `ModItem` carries `Layer` (-1 under, +1 over) and `Anchor`. A worn layer is one unit of its code that stays in the pack, recorded in `PartyMember.Layers`. `ActiveLayers` keeps those the pack still backs, so a sale, drop or trade takes the layer off. `AddLayers` adds the layers to the loadout through `data.FoldLayers`, the same per-item sum `data.FoldWear` uses; `PartyLoadout` and the mission rearm call it. The weight is the pack unit's, counted as carried. A second layer at the same anchor and depth replaces the first (`wearLayer`).

Wear gesture. In the mission inventory window the equip request of a pack cell holding a layer toggles it (`mapWorld.toggleLayer`) before the ordinary equip route; in the town shop the pack cell used on the doll does (`townScreen.shopToggleLayer`), and a layer is never worn through the shelf or the table. The item's `for` row and the wear rule of its table apply. A member whose statistics come from an original save refuses a layer.

Paperdoll (`pkg/game/inventory.go`, `portrait.go`, `modlayers.go`). `composeInventorySubjectLayered` paints an under layer just before, and an over layer just after, the anchor slot's turn in `data.FigureDrawOrder`, whether or not the slot is occupied. A layer owns no slot in the click mask. The mission inventory window, the town shop doll and the shop portrait strip draw layers. The unit's map figure does not.

Saves (`pkg/game/modmark.go`, `currentsave.go`, `originalsave.go`, `currentpartyread.go`). The leaf gains an optional `layers` array (`member`, `code`); the format stays 1. SAVE lists each worn layer the pack backs and writes the layer's pack unit under the stand-in rule of `DIV-1863`; the twelve ordinary slots are written as before. LOAD under the same mod set restores the layers on the party before the world is read and carries them across the mission party's rebuild from the saved records. A layer naming an item no active mod adds as a layer, or a member the save lacks, is refused. LOAD without the mod refuses by `DIV-1802`. An unmodded save has no leaf.

Example mod (`pkg/modrt/testdata/mods/heavy-armor`, test data) now holds three items and the Chain Mail weight change:

| key | slot row | layer | anchor | defence | absorption | weight | price | figure | for |
|---|---|---|---|---|---|---|---|---|---|
| gambeson | armour row 31 | under | body | 6 | 1 | 6 | 120 | Soft Mail | any |
| cloak | shield row 10 | over | body | 2 | 0 | 4 | 60 | Cloak | fighter |
| boots | shield row 11 | over | legs | 1 | 0 | 3 | 40 | Chain Boots | any |

All three are on the armour shelf and carry English and Russian names.

## Proof

- Unit: `pkg/mod` (layer, anchor and figure parsing and refusals, `LoadStrings` containment), `pkg/game/modlayers_test.go` (placement carried into `ModItem`, `ActiveLayers`, the replacement rule, mark layers round trip and refusals, clone ownership).
- Release, one root at a time, EN and RU (`pkg/game/modlayers_release_test.go`, `moditems_release_test.go`, gated by `AGAINROM_ASSETS`):
  - Town: the three layers are bought and worn; the slots are unchanged and the pack still holds the units; the layers add defence 9 and absorption 1 to the loadout. The town SAV holds three layers in the leaf and no mod code in an item object. A cold LOAD under the mod restores all three; LOAD without the mod refuses. A SAVE after taking the layers off loads without them and the witness fails (loss control).
  - Mission: a cold town LOAD opens a mission whose hero has defence 9 above the same hero without layers and the same carried load. Using the cloak's pack cell in the inventory window takes it off (defence +7) and puts it on (defence +9). A mission SAVE holds three layers; a cold mission LOAD under the mod restores them and the defence; LOAD without the mod refuses.
  - Z order: for the gambeson under chain mail, the cloak over chain mail and the boots over bare legs, every pixel of the composed figure equals the painter's rule applied to the anchor-only and layer-only figures. The witness requires pixels where both paint with different colours, pixels only the layer paints, a layer pixel hidden by the anchor item for an under layer, no click-mask slot on layer pixels, and a figure that differs without the layer.
  - An unmodded town SAV has no leaf (`TestReleaseUnmoddedTownSaveHasNoModLeaf`).
- Screenshot (offscreen, never skipped): `review/story1273-mod-layers/paperdoll-<root>.png` shows the figure bare, in chain mail, in chain mail under the three layers, and the twelve-slot panel.

## Open debt

- The map figure of a unit ignores layers (`DIV-1873`).
- A layer is a pack unit: it is not a worn object of the simulation, so another actor reading the hero's equipment (a trade, a script) does not see it.
- A layer needs a free row in the armour or shield table; the armour table has one free row on a shipped install.
- A member whose statistics come from an original save refuses layers.
- Whether the original reader accepts a save whose pack holds a stand-in object for a layer is Unknown, as in `docs/1272/story.md`.
- Original SAV-corpus saves were not loaded under a mod.
