# 1059 — animated enchanted-item stars

## Intent

Replace the authored static three-star mark with ROM1's procedural magenta trail on every measured
item-cell consumer, without adding presentation state to simulation, saves, hashes or item icons.
The authority is `ITEM-STARFLAG-096`, `ITEM-STARSURF-097`, `ITEM-STARPIX-098`,
`ITEM-STARPHASE-099` and `ITEM-STARCOMP-100` at research pin `e60b8a125f69`.

## As built

- A resolved picture plus nonempty effects enables the trail; Potion kind 3 suppresses it. Mission
  carried pack, shop shelf, trade table and shown-member pack are the complete positive consumer
  set. Worn equipment, world figures and mission/shop held cursors retain the shared base icon.
- Each grid owns 1,024 CRT-generated coordinate pairs. A phase selects one coordinate every two
  increments and paints the seven-centre RGB `(255,0,255)` trail with the decoded 13-pixel kernel.
- Mission phases belong to visible slots and advance only inside one actual world tick, so ordinary
  pause and repaint hold them. Shop phases advance before the cells on each App paint. Scroll
  applies the decoded asymmetric slot shifts; a selected singleton disappears with its trail while
  a remaining stack continues. A wheel during drag keeps the absolute shelf/pack source bound.
- Mission paints base, quantity, then trail. Shop paints background, base, trail, quantity, then
  price. Shop uses its decoded `top+1` origin and the full-frame clip with no per-cell clip.

## Proof

- Focused clean-room tests independently pin the normalized coordinate prefix, 13 kernel pixels,
  alpha ages, two-increment cadence, seven-centre lifetime, asymmetric scroll, Potion exception,
  all positive and negative consumers, opposite mission/shop quantity order, selected-stack rule,
  mission tick clock, shop advance-before-paint clock and wheel-drag source identity.
- The installed-art release witness changes pixels inside all three shop footprints on both EN and
  RU. Its plain/phase-0/phase-2 SHA-256 values are recorded by
  `TestReleaseEnchantedItemStarsChangeEveryMeasuredShopGrid`.
- Final full-suite, no-assets, release and mission-drive results are recorded on the candidate
  commit in the lane handoff.

## Open debt

- `DIV-497`: exact CRT seed, constructor order and intervening draws remain Unknown; this build
  discloses a deterministic normalized zero start rather than claiming a ROM1 coordinate stream.
- `DIV-498`: the old mission pack is an authored 48-pixel grid rather than ROM1's 80-pixel grid, so
  the mission origin follows the base icon actually drawn and its intermediate bar clip. Shop
  geometry is exact. Replacing the whole mission bar is separate layout/input work.
- Surface exclusivity remains Medium for a runtime-computed indirect target. Active ROM1 global
  clip values, runtime occlusion, shop repaint scheduling and paused-shop coexistence remain
  Unknown. `DIV-499` and `DIV-500` were unused and are permanently retired.
