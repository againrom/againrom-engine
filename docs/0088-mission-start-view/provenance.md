# Provenance — where a mission opens, and how much of it is visible

## Backing — what a source asserts

| Spec anchor | Source | Confidence | What it supplies |
|---|---|---|---|
| FR-2 (the anchor is read, not re-derived) | `MISSION-DROP-002` | High | The map's whole contribution to a start is one packed cell, and the engine picks from the array with a **uniform random index**, not `[0]`. A consumer that re-derives the cell reproduces a draw; one that reads the placed party does not. |
| FR-2 | `MISSION-START-001` | High for the positive half (the absence clause is retracted) | The routine that puts the player on the map never reads the map's own unit array: it walks the **player's own container** and places the hero at the drop cell with radius 0, everything else around it. A campaign map *usually* places nobody for the player, but 5 of 28 do — and the walk then moves those to the drop as well. A position read off the placed party is right in both cases; one read off the drop table is right in only one. |
| FR-2 | `PARTY-ORIGIN-010` | High | "The party" is the flat actor index the placement walk reads, not the saved group list — which is why "where the party is" is a property of the load's own result rather than of any stored table. |
| FR-3 (the extent is stated in one place), FR-4 | `MENU-ASSET-001`, `MENU-GEOM-005` | High | The original's frame is **640x480**: the menu's base bitmap and hit mask are both that size, and the overlay placement tables are consumed against a full-screen mask indexed from `(0,0)`. This is the anchor the authored column count is derived from, together with the tileset's own 32-pixel cell. It is a fact about the original's **screen**, not about its map viewport. |

## Ours by choice — what the spec fixes that no source asserts

| Spec anchor | The choice | Why it is ours |
|---|---|---|
| FR-3, FR-4, SC-1 | **The mission opens spanning 20 map columns across the view's width.** | Nothing published states what the original sets its view to at map load, or what its viewport spans in tiles. 20 is 640 divided by the tileset's own 32-pixel cell: the count of native cells that fit the original's whole frame. It is an **upper bound and not a measurement** — the battle screen's panel is undecoded, so the original's map area is at most the frame and certainly less. The number is authored, disclosed, and placed behind one named function so that replacing it is one edit. **Verdict: AUTHORED.** |
| FR-3 | The extent is fixed as a **column count**, and the row count follows from the window's own proportions. | A window's aspect ratio is the player's, not ours, and the original shipped one resolution while this tree has continuous zoom — so there is no pixel answer to recover, only a ratio to choose. A column count is also the shape the open research question is asking in, so its answer replaces this value rather than reshaping the seam. |
| FR-2 | The anchor is the **first party member's** cell, and the cell the start decided when there is no party. | The engine places the hero at the drop cell exactly and crowds everyone else around it (`MISSION-START-001`), so the first member's cell is the spawn in the sense the requirement means. Nothing published says a view should be centred on a party's extent rather than on its hero, so the simpler of the two is chosen and written down. |
| FR-5 | The start view is applied **once**, when a view size is first adopted, and never again. | No source describes when the original establishes its scroll origin relative to its window. Applying once is the weakest behaviour that satisfies the requirement, and it is what keeps FR-6 true. |
| FR-8 | A start view is **clamped** like every other camera movement. | The camera's own bound is this tree's, from an earlier story; no source describes the original's clamp beyond `TERR-EDGE-026` locating one. |

## Open — undecoded, and deliberately given no meaning

- **What the original sets its view to at map load.** `TERR-EDGE-026` (High for the over-scan, Medium
  for the exact visible extent) establishes that the draw loops are viewport-relative and offset by a
  scroll origin at `+0x5c/+0x60`, and that a view clamp exists in the draw path. It publishes neither
  the origin's value at load nor the viewport's extent in tiles. This is the seam FR-3 names, and a
  research lane is open on exactly it.
- **The battle screen's panel geometry.** Nothing published gives it, so the fraction of the 640x480
  frame the original's map occupied is unknown, and the authored count cannot be narrowed from an
  upper bound to a value.
- **The per-player start override.** `MISSION-DROP-011` (High for the writer, Medium for the meaning)
  identifies `player+0x60` as "start the next map where this unit is standing now", written by
  session opcode `0x32`. It is a **start position and not a camera**, and `MISSION-DROP-002` shows it
  is never read on the campaign path. It is named here so that a later reader does not mistake it for
  the answer to FR-3; the tree does not model it.

## Removed — statements dropped, and why

| Dropped | Why |
|---|---|
| Any requirement that the start view reproduce the original's | It would assert something no source establishes. The contract states an authored value and the divergence, which is what the evidence supports. |
| Centring on the party's bounding box rather than on its first member | No source distinguishes the two, and the difference is at most the crowding radius. Asserting the more elaborate of two unsupported readings buys nothing. |
| Deriving the anchor from the map's drop table | `MISSION-DROP-002`'s random index makes this a reproduction of a draw, and `MISSION-START-001`'s five maps make it wrong even where the draw is degenerate. |
