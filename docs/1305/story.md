# Worn items follow the original's colour pass

## Intent and authority

A figure draws its worn items in the order the original does. Owner direction: close the revisit condition of DIV-1541 (known defect D9). Authority: `HERO-FIGURE-058` (a layer's slot is its drawable index plus one), `HERO-FIGURE-059` (the colour pass is program order, one pass for a mage and one for everyone else; slot 9 is tagged and never blitted for a mage), `HERO-FIGURE-060` (the held-slot predicate), `HERO-APPEAR-050` (the second sheets).

## As-built behaviour

`data.FigureDrawSteps(dir, bodyList, equipment)` returns the figure's steps: an equipment slot, whether it is that slot's second sheet, and a kind (paint, tag only, or none). `composeUnitFigure` (map, hover, dialogue figure) and `composeInventorySubject` (inventory window, shop doll) both range over it. `data.FigureDrawOrder` is the slots in first-reached order.

Non-mage, in slots: 12, 11, 7, 4, 5, 9, 10, 1, 8, the second sheet of 4, 6, the second sheets of 9 and 10, then slot 1 when `FigureHeldLast` answers 1, otherwise slot 2. The tail paints one slot, so slot 2 is not painted under a weapon-last body. Slot 3 has no step.

Mage, in slots: 8 (first sheet behind the body), 12, 10 and its second sheet, 4 and its second sheet, 7, 5, 9 (tag only), 1, 6, then the second sheet of 8. Slots 11, 3 and 2 have no step. Slot 9 claims its pixels in the per-pixel `SlotMask` and the hover coverage without painting colour, as `HERO-FIGURE-059` finding (c) says.

A slot without a painting step keeps a no-op step, so its icon and the mod clothing layers anchored to it keep their place.

Player-visible change: on a fighter the weapon is covered by body armour where they overlap, gauntlets and bracers draw as the original draws them, and a shield is not drawn under a two-handed or ranged body. A mage no longer shows slot 9 or slot 2 items. A slot 3 item is no longer drawn on any figure.

## Proof

- `pkg/data` `TestFigureDrawStepsOfAFighter`, `TestFigureDrawStepsOfAMage`: the step lists.
- `pkg/game` `TestComposeUnitFigurePaintsASlotOneWeaponBeforeSlotEightArmour` and `TestComposeFiguresOfAMagePaintNeitherSlotNineNorSlotTwo`: synthetic sheets, both compositors, picture and hit map.
- `TestReleaseFighterWornItemsFollowTheOriginalsColourPass` (EN and RU installs): a man fighter in a weapon, a slot 8 item, bracers, gauntlets, a ring, a helm and a shield from the installed definition rows, for each outcome of the held-slot predicate. An oracle reads the installed sheets and the claim's pass without calling a compositor; every pixel it paints is compared with the map figure and the inventory doll, colour and slot. A second oracle in the earlier order must differ, which it does on 744 pixels (weapon-last body) and 1 pixel (shield-last body).
- `TestReleaseTownMageGlovesAreDrawnUnderTheBraceletsOnReniesta` still passes on both roots.

## Open debt

- DIV-1541 narrows to the order of the non-mage hit-map pass. No claim states whether it equals the colour pass; this build uses the colour order.
- The installed mage directories ship no sheet for slot 9 or slot 2 (searched: the shop armour and weapon pools on EN and RU, the man and woman mage figure directories), so the mage rule for those slots is witnessed on synthetic sheets only.
- The shield-last witness separates the two orders on one pixel only; the weapon-last witness on 744.
