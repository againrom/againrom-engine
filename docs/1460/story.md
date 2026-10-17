# The map editor's Map panel and its checks

## Intent

The map editor opens each map on one Map panel that states the map in player
words and opens the place that fixes each line. Two checks name the maps an
author most often leaves unplayable: no start point, and nothing that wins
the map. A new map asks single-player or multiplayer and starts with a start
point. The story started at engine main `d85eb214` (game 0.126.0, pin k326);
the pin does not move.

## Authority

Owner direction: one Map panel opened first with each map; the two warnings
show in the panel, the report and one notice before Test play and Save into a
mod, and never refuse a save; a new map is single-player by default with its
drop cell at the centre; an untouched shipped map saves byte-equal.

Claims, read from the pinned snapshot:

- `MISSION-DROP-002` (High): with no usable drop cell the party lands on a
  random cell of 30 to 100 on each axis. `TRIG-DROP-013`: the drop action.
- `TRIG-END-009` and `MISSION-WIN-003`: only instant 4 wins a map; every
  campaign map is won, no loose map is.
- `MISSION-LATCH-014`: two wins in one pass are no win. Not checked here.
- `ALM-MODE-070` (High): type-0 `+0x70` above 1 is multiplayer; every
  scenario map carries 1.
- `ALM-NEWMAP-236`: the original editor writes 4 into a new map's `+0x70`.

## As built

- The panel (`pkg/ui/mapeditormappanel.go`) is the editor's list popup in a
  fifth kind, opened by `openPanel` in `mapeditormode.go` whenever a map opens
  on a host that states `MapFacts`, and by the Map menu's "Map panel" row.
  Esc closes it. Its lines and links:

  | Line | Value | Opens |
  |---|---|---|
  | Size | W x H | nothing |
  | Name, Description | the text, or "none" | the metadata prompt (M) at that field |
  | Players | players and sides (mutual allies join a side) | the Players list |
  | Mode | Single-player, Multiplayer, or the stored number named single-player or multiplayer | the metadata prompt at Mode |
  | Start point | the drop cell, the cells, or "none: the party lands at a random cell" | the drop-cell step |
  | Victory | the winning trigger, the count, or "nothing wins this map: it is endless" | the script goals |
  | Defeat | the losing triggers and the hero's fall | the script goals |
  | Report | errors and warnings | the report |
  | Test play | F5 | test play |

- `mapRules` (`pkg/game/mapcheck.go`) holds 14 rules. The `drop-cell` rule
  now warns on every map with no drop cell ("the map has no start point"),
  with the lost fallback cells counted when any are lost; no second rule
  names a missing start point. The new `victory` warning names a map whose
  script runs no instant 4 (`pkg/game/mapinspectionpanel.go`); a script that
  does not decode is the `mission-build` rule's, not this one's.
- The Start point and Victory lines are drawn in warning ink when their rule
  has a finding. Before Test play and Save into a mod the same two findings
  form one notice; Continue goes on, Cancel and Esc return. A map whose report
  holds an error gets no notice: the save refuses as before.
- A new map asks "single-player or multiplayer?" (Single-player first and on
  Enter). `newAuthorMap` writes `+0x70` = 1 or 4 and one Drop location action
  at cell W/2,H/2 as the map's first bytes (`DIV-3597`); `mapedit -new` writes
  the same map.
- The metadata field of `+0x70` is Mode with Single-player (1) and Multiplayer
  (4); another stored value is offered as its number and kept.

## Proof

Census (`TestReleaseMapPanelCensusOfTheInstalledMaps`, the win count read
from the decoded script without the editor's helpers):

| Root | Maps | No drop cell | No win | `+0x70` |
|---|---|---|---|---|
| EN | 38 | 0 | 10 loose: Beast, Cross, Forester, Horror, Islands, Kids, Kids2, LuMoir, Tomb, Waters | 1: 28 campaign; 4: Kids, Kids2; 8: LuMoir, Waters; 12: Cross; 16: Beast, Forester, Horror, Islands, Tomb |
| RU | 34 | 1: loose/Horror.alm | 6 loose: FORESTER, Horror, ISLANDS, KIDS, LUMOIR, WATERS | 1: 28 campaign and Horror; 4: KIDS; 8: LUMOIR, WATERS; 16: FORESTER, ISLANDS |

The rules name exactly these maps. The single-player value 1 is the value of
all 28 campaign maps on both roots.

Witnesses, each with a loss control:

- `TestMapRuleVictoryNamesAMapNothingWins` and
  `TestMapRuleStartPointIsTheDropCellRule`: a table without the row names
  nothing.
- `TestMapPanelChecksMarkTheStartAndVictoryLines`: without the findings no
  line is marked. `TestMapPanelNoticeBeforeTestPlayAndModSave`: a playable
  map and a map with an error get no notice.
- `TestNewMapIsSingleByDefaultWithADropAtTheCentre`: the bytes equal the
  original new map plus the mode and the drop action; an unknown mode suffix
  is refused.
- Release, EN and RU: `TestReleaseMapPanelLinesOfInstalledMaps`,
  `TestReleaseMapPanelNewMapAsksTheModeAndWritesItsBytes`,
  `TestReleaseMapPanelSaveIntoModWithBothWarningsWritesTheMap` and
  `TestReleaseMapPanelUntouchedMapsSaveTheirBytes` (every installed map, read
  through the panel and the report, saves its installed bytes).
- Frames: `TestReleaseMapPanelFrames`, EN and RU, viewed.

## Open debt

- `MISSION-LATCH-014`: a map that runs two wins in one pass is not checked.
- A new single-player map keeps the original's name "Multiplayer Map".
- The report's finding texts are English on both roots.
- The panel covers the map's top-left corner while it is open.
- The installed RU font draws no `ё`; the panel's words use `е`, and the
  existing "Отчёт (R)" menu word still shows the gap.
