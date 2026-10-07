# Shop item release over the character panel

## Intent and authority

A button-up over the shop's character panel with an object in hand is routed
by the object's stamp, not by what the object is. Authority: knowledge pin
k109, `SHOP-096` to `SHOP-101` and `formats/shop/trade.md`.

- `SHOP-097` (High): the stamp is set by the object's source. Shelf contents
  load index + 5, new party equipment 1, backpack insert 2; a tray insert keeps
  any stamp other than 1.
- `SHOP-098` (High for 1, 2, 5 to 8): stamps 1 and 2 go to the equip arm,
  stamps 5 to 8 to the return routine, whatever the origin code.
- `SHOP-099` (High dispatch, Medium bodies): the return routine dispatches on
  origin code and contains no purchase command.
- `SHOP-101` (High gate list, Medium coverage, Unknown bodies): the equip arm
  equips through command `0x22` from the origin container or rejects.
- `SHOP-100` (High): a button-up reaches the panel only with no capture child
  and a point in `(480,238)-(640,480)`.

## As-built behaviour

Before: a shelf object released on the figure or anywhere on the panel was
bought and worn (`DIV-087`, `DIV-1553`), and so was a table place the merchant
still owned.

After, for a drag released over the panel (`ShopDrag`,
`pkg/game/shopview.go`):

| held object | stamp | result |
|---|---|---|
| shelf object | 5 to 8 | stays on its shelf; no purchase, no wear (`DIV-1667`) |
| table place, merchant-owned | shelf stamp kept | stays on the table (`DIV-1668`) |
| table place, customer-owned | 2 | worn by the wear rule, no charge (`DIV-1669`) |
| pack object | 2 | worn by the wear rule, no charge (`DIV-1669`) |
| object picked off the figure | 1 or 2 | stays worn (`DIV-1670`) |

While a shelf object or a merchant-owned table place is held inside the panel
rectangle the cursor is the registered `cantput` picture, a red cross: 64x64
frame, hotspot (38,36) (`SPR16A-CURSOR-046`), selected for a held object whose
stamp is not 1 by routine `R0338` (`AI-CURSOR-205`). Owned objects keep the
arrow (owner direction; see `DIV-1667`). The cross ends on release.

The owner flag of a staged table place stands for the stamp: a place the
merchant owns was staged from a shelf and carries a shelf stamp; a place the
customer owns was staged from the pack. The panel rectangle test in
`pkg/ui/app.go` is unchanged (`shopPaneTakesItemAt`).

## Removed purchase

The shelf-to-figure purchase (`DIV-087`) was an owner-directed feature. The
owner approved removing it in favour of the return-to-shelf routing above.

## Proof

- `TestShopCrossCursorFollowsAHeldShelfStamp` (pkg/ui): the cross shows for a
  shelf object and a merchant-owned table place held over the panel, and not
  for an owned table place, a worn object, a pack object, or outside the panel.
- `TestReleaseShopPanelReleaseRoutesByStamp` also checks, on EN and RU, that
  the mid-drag cursor is the 38,36 hotspot picture with red pixels for a shelf
  object, is absent for an owned pack object, and ends on release.

- `TestShopShelfObjectReleasedOnTheDollStaysOnItsShelf`,
  `TestShopTableNonMinePlaceReleasedOnTheDollReturnsToTheTable`,
  `TestShopDragDispatchesTheFourRecognisedPairs`,
  `TestShopEquipFromTableWearsAMinePlaceForFree` (fixture tests with controls:
  shelf-to-table staging, pack-to-doll wear, owned table-to-doll wear).
- `TestReleaseShopPanelReleaseRoutesByStamp` (EN and RU installs, production
  App pointer input): shelf release over the figure and over a corner
  rectangle changes nothing; the control drag shelf-to-table stages the object;
  the staged merchant place released over the panel stays; an owned pack object
  and an owned table place are worn with the purse unchanged; SAVE and cold LOAD
  restore the purse, worn set and pack. The test fails with the old routing
  (checked by restoring the previous `ShopDrag`).
- `TestReleaseShopItemReleasedOnEachPaneRectangleIsWorn` (existing) keeps the
  pack and figure cases.

## Open debt

`DIV-1667` to `DIV-1670` carry the Medium and Unknown parts: the return and
drop bodies, command `0x22` acceptance with a tray source, the equip arm's
gates, and stamps from a SAV load. `shopEquipFromShelf` and its city variant
are no longer reached from production input; fixtures still call them to wear a
shelf object, so they remain until those fixtures move.
