# Story 1072 — wide mission interface

## Player result

Modern wide windows reveal more mission map at the same zoom instead of
centering a fixed 4:3 mission picture between side bars. The main menu and the
complete town family retain their native 640x480 composition and uniform fit.
Neither family targets micro-resolutions.

## In scope

- Mission logical width is `max(1024, ceil(window width * 768 / window
  height))`: at 1920x1080 it is 1366x768. A 4:3 window keeps 1024x768.
- Main menu, picker, character generation, town, shop, tavern, school and the
  town in-game menu remain a byte-identical 640x480 logical frame at every
  window aspect. Wide windows therefore letterbox those screens.
- Mission HUD remains 160 pixels wide at the right edge. Extra width reveals
  more map at the existing starting zoom instead of enlarging the same 27
  columns.
- At ordinary zoom, the top and side camera motion stops at the non-black
  terrain interior. The lower edge admits one existing textured but impassable
  margin row, so the final playable projected terrain cell is never clipped.
  On displaced
  terrain the vertical edge is resolved only over the columns currently on
  screen, so relief elsewhere cannot crop reachable rows. Larger black
  map-edge space appears only when zoom-out makes it unavoidable and is centred.
- The live game surface ends above an open inventory or spellbook bar. The
  spellbook sits on the bottom edge when it is open alone. Closing a bar keeps
  the camera origin and reveals terrain below, except at the bottom map edge,
  where the camera clamps upward by only the required distance.
- Input follows the same placement: non-mission side bars are inert, native
  controls round-trip unchanged, and the mission viewport grows with its frame.
  The right column and the bottom panel band never resolve to map input.
- Repeated non-mission Layout calls reuse the same 640x480 canvas.

## Authority

This is owner-directed modern presentation policy, corrected by the owner's
2026-08-27 playtest: only missions expand. `SESS-VIEW-028` describes ROM1's
fixed screen sizes, not arbitrary modern window shapes. `DIV-211` owns the
policy and `DIV-212` owns input outside the active surfaces.

## Proof

Focused placement, camera and input tests prove the four bottom-panel layouts,
the close-without-reanchoring rule, edge-only bounce, a local projected-edge
clamp, the shared lower render-only apron used by camera and minimap, and centred
unavoidable zoom-out space. The paired EN/RU release witness leaves the
installed menu and town byte-identical at 640x480, opens real mission 10 at
1920x1080, verifies that both camera extremes retain the installed map's outer
black engine margin outside the game surface, and reaches its last playable row
plus the one textured impassable apron row. Full Go and the no-game-assets guard
run on the exact hotfix.

## Exclusions

The story does not change simulation, saves, the 480/768 logical heights, the
160-pixel mission HUD width, or support for a logical width below 640/1024.
