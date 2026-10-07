# Story 1092 — read-only authored map inspection

## Result

`cmd/mapedit` opens an installed or explicit ALM map with its authored units,
structures, ground sacks and scenery. It has no mission, simulation, fog,
script execution or map-write route. `cmd/mapview` keeps its diagnostic defaults.

## Behaviour

- Installed catalogue and explicit host paths use the byte-preserving
  `pkg/mapedit` model. Failed, missing and malformed opens retain the previous
  view and camera. Escape dismisses path/error presentation.
- Actual installed art is shown in frozen authored poses. No HP filter,
  runtime spawns, party substitution or coalescing of ground-loot records runs.
- The catalogue covers placements, actor stock, groups, script nodes,
  enchantment metadata and every physical section. Unknown, malformed and
  overridden sections remain listed. Missing spatial art gets a visible anchor
  and warning; out-of-map anchors are marked at the edge.
- Selection preserves the exact section and file index, marks its footprint,
  and jumps from the catalogue. Canvas clicks inspect/cycle overlapping cells.
  Selecting an object reveals its row in All; a chosen filter is not replaced.
  The inspector shows decoded item names/codes, gold, raw fields and enchantment
  links. Wheel scrolling reaches every detail line, including at 640x480.
- Drag/arrows pan, canvas wheel zooms, and Fit restores the complete overview,
  including 256x256 maps at 640x480. Only the editor camera receives a lower
  zoom floor. Maps, Open path and the rail wheel remain separate from canvas
  input; fractional wheel travel is retained separately for list and details.

## Design and authority

The owner requested a read-only first editor slice. The frontend-design skill
led to canvas plus a fixed 320-pixel right rail: a bottom strip sacrificed map
height. Native window pixels preserve shipped font1 legibility at 640x480;
heading, body and coordinates use restrained color roles rather than new fonts.
The signature is the selected file record's anchor/footprint and matching row.
Palette: ink #201B16, panel #30291F, parchment #D6C49B, light #F3E5BE,
gold #B8954F, warning #B85E46. Russian item bytes are wrapped without UTF-8
reinterpretation; Unicode host paths are encoded only for display.

Research pin ba21c9a supplies `ALM-REQ-055`, `ALM-UNIT-048`, `ALM-SACK-065`
and the existing terrain/art contracts. DIV-618 discloses the authored interface
and unresolved content. No original-editor fidelity is claimed.

## Proof and exclusions

See [verification.md](verification.md) for counts, gates and render witnesses.
The touched surfaces are the new command, game asset-to-inspection adapter,
UI catalogue/inspector, an opt-in Viewer viewport/HUD branch and per-camera
zoom floor. Ordinary game/mapview limits, saves and hashed state are unchanged.

EN and RU each retain 97 unresolved scenery cells as visible warnings. Undecided
enchantment metadata has no invented world placement. No editing, script preview,
unit animation, general search or original editor reproduction is included.
The headless PNG uses CPU terrain, retained body placements and the exact rail;
it omits projected shadows and is not GPU readback. No live window was driven.
