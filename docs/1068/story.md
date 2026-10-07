# 1068 — original mission pack grid

## Player result

The mission pack uses the original 80-pixel grid. Installed 80x80 item icons are no longer cut
down to the old 48-pixel cells. Quantity and enchanted-item stars remain attached to the same cell,
and long packs remain reachable through the two end strips.

## Authority and as built

At research pin `e60b8a125f69`, `ITEM-STARCOMP-100` (High) gives rectangle
`(0,H-90,W-160,H)`, origin `left+32+((W-240)%80)/2+80*i, top+6`, 80x80 icons, mission draw order
base/quantity/trail, and only the renderer-global half-open clip. `SESS-INPUT-037` (High) gives
one-position scrolling at the two edge strips plus cell hit, drag and transfer routes.

- One geometry supplies paint, hover, drag, double-click/equip and scroll hit-testing. It gives five
  cells at 640x480 and nine at the shipped 1024x768 mission frame.
- The base picture uses the complete 80x80 cell. Quantity paints after it and the purple trail paints
  last. A full-frame-width layer removes the former compact-bar clip; the decoded trail footprint
  remains inside the layer vertically.
- The existing small triangle cues remain centred in the edge strips. Their art is authored and is
  not claimed as original; the complete strips are clickable.
- Pack order, contents, saves, simulation and hashes are unchanged. Town, shop, worn and spellbook
  grids are outside this story.

## Proof and open limits

Independent literal tests pin both frame sizes, every first/last cell and both strip rectangles.
Synthetic 80x80 corner pixels prove the base is not clipped. Existing pack gesture, popup, drag,
equip, scroll and star-order suites exercise the shared geometry. The EN and RU release witness
decodes `graphics/inventory/0001003.16a`: both roots produce 586 painted pixels and identical
plain/phase-0 composed hashes.

The original decorative strip art remains Unknown. `DIV-497` still records the Unknown initial CRT
coordinate-stream state. This story closes `DIV-498`.

The two older `1005` frontend scenarios remain blocked before their mission-pack coverage:
`1005-doll-and-shop.json` times out at step 2 waiting for the first notice immediately after load,
and `1005-doll-carry-over-worn.json` times out at step 10 waiting for a notice after step 9 has
successfully proved the armour-to-pack transfer. Both failures reproduce at the same steps on EN
and RU on exact serialized base `2e65ad4acc89cf2468370fcbffe4246c630ae0ca`; 1068's candidate
produces the same states. Repairing that stale mission-20-to-town progression is outside this UI
geometry slice.
