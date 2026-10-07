# provenance — 0158 in-game menu

## Claims this story is built on

| Claim | Confidence | What is taken from it |
|---|---|---|
| `MENU-ESC-010` | High | Escape raises two different surfaces, one per UI state. Mission panel `(100,60)-(440,400)` = 340x340; town panel `(100,100)-(440,340)` = 340x240. Neither path reaches the main-menu window. |
| `MENU-ITEM-011` | High for the list, Medium for the disables | The mission surface: Save Game, Load Game (or Diplomacy), Game Options, Sound Options, Quest Objectives, End Quest, Return to Game — seven shown of eight constructed. Four rows carry a disable predicate. |
| `MENU-ITEM-012` | High for the list, Unknown for the exit target | The town surface: Save Game, Load Game, Sound Options, Abort Game, Return to Game. Every row is built enabled. Save precedes Load. |
| `MENU-KEY-013` | High | The effective accelerator is the letter after a single `~` in the label; `~~` is an escape; a label with no `~` keeps the constructor's immediate. `Quest Objectives` passes `M` and ships `Q`. `Diplomacy` and `Abort Game` carry no `~` on the English root. |
| `MENU-ART-014` | High for the row layout, Unknown for the tiling | Row layout: each row is panel-local `(40, 40+30(n-1), width-48, 40+30n)`. The frame is a nine-patch out of `graphics\interface\lm.256`. |
| `MENU-STOP-015` | High | Both surfaces stop the world through the idle-handler gate: no pacer arm runs at all. Compositing is `DLG-DIM-013` unchanged — one destructive remap of the whole screen at shroud level 3, a per-channel gain of 13/16. |
| `MENU-INPUT-016` | High for the routing and the capture rule, Unknown for the panel's own handlers beyond the slots read | Escape opens and Escape closes; `0x1b` posts the same message `Return to Game` posts. Up and Down move the focused row. While the panel is up it is the root's capture object, so a click outside it reaches no other child of the root. |
| `DLG-DIM-013` | High | The shade level and the gain, applied here through this repository's existing authored value (`AuthoredNoticeBackdrop`, 0074). |
| `DLG-STOP-012` | High | What the gate does: no sub-tick, no presentation tick, no map-view drain, no command drain. |
| `DLG-CLOCK-014` | High | The close arm discards the stopped span rather than catching it up. |

## Open in the decode, authored here

- `MENU-ITEM-012` grades the exit rows' target **Unknown**: both post `0x41e` and differ only in control id.
- `MENU-ART-014` grades the nine-patch tiling of the 44-px uncovered band **Unknown**.
- `MENU-INPUT-016` grades what the panel's own handlers do beyond the slots read **Unknown**. A click outside the panel reaches no other child of the root, so this build's silence there matches it.
- `campaign+0x6bc`, the word three mission disables and the Load/Diplomacy exclusion read, is not decoded. `MENU-ITEM-011` cites its comparisons and not its meaning.

Each is authored in `spec.md` under SC-* with the divergence stated in the reader's own units.

## Ours by choice

The panel frame, the row colours and the row labels are this project's own. The original's frame art
and its `main\text\dialogs.txt` labels are named as seams in `spec.md` and are not loaded.

## Not used

`MENU-ASSET-001`, `MENU-ASSET-002`, `MENU-MASK-003`, `MENU-MASK-004`, `MENU-GEOM-005`,
`MENU-GEOM-006`, `MENU-STATE-007`, `MENU-STRTAB-008`, `MENU-DOC-009` describe the **main menu**
overlay and its hit mask, which is a different surface from the in-game panel and is already built.
