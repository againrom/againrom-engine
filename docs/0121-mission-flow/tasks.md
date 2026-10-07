# 0121 — mission flow: tasks

## T1 — the map list carries campaign missions, and choosing one starts it

Files: `pkg/game/maplist.go`, `pkg/game/frontend.go`, `pkg/ui/picker.go`, and tests beside them.

Add `Mission int` to `MapEntry` (DD-2) and one leading branch to `Text()` (FR-6). Beside
`BuildMapList` add one exported function taking a finished list and returning the mission rows
followed by that list unchanged (DD-1, FR-1): one row per `FromArchive` entry whose stem parses as
a positive decimal integer (DD-3, FR-2, P-3), carrying that entry's `Source`, `Name` and `Err`
(FR-5), ordered by the parsed number, `sort.SliceStable` (DD-4). Compose it where `FrontEnd.Maps`
is built.

In `loadMap`, right after the row is taken and before anything is read, return
`f.MissionOpener(e.Mission)()` when `e.Mission > 0` — that call and nothing else (DD-5, FR-3, P-4).
Other rows keep their path (FR-4). Set `pickerTitle` to `SELECT A MISSION OR MAP` (DD-6).

Tests, synthetic: the composed order and count over a stub container; a non-numeric stem yielding
no mission row; a loose `10.alm` yielding none; row text for both kinds; a mission row over an
undecodable entry present and unchoosable; and — the one that must go red when the branch is
removed — `loadMap` on a mission row handing back a non-nil `ui.MapAdvance` where a map row hands
back nil.

`SDD-Task: 0121-mission-flow/T1`

## T2 — a mission with no living hero is lost, ahead of both counters

Files: `pkg/game/world.go` and tests beside it.

Give `missionNotices` a hero entity id and a flag saying one was recorded, filled in `openMission`
from the start's first recorded party id when there is one (DD-7, FR-7, FR-10). In `settleNotices`,
inside the not-yet-announced arm and **before** the `Outcome()` read, decide the outcome: a recorded
hero the world no longer holds, or holds and `Alive()` is false, gives `sim.OutcomeLost`; else the
world's own outcome (DD-8, DD-9, FR-8). Feed both into the one existing block that latches the
announcement and opens the banner (FR-9). Add nothing to the plain-map path (DD-10, FR-11). Touch
no file under `pkg/sim`, add no field to a serialised record, move no byte-form version (P-1, P-2).

Correct `openMapWorld`'s comment claiming a picker-opened map "advances under its own script"
(DD-11, FR-12).

Tests, synthetic: a mission world stepped with the hero removed reports the loss; with the hero at
zero health and not dead, the same; hero alive and the winning counter satisfied, a win; a win
already latched and the hero dead, the loss; no party, nothing reported; a plain `openMapWorld`
map opening no notice and reporting no outcome after many steps; the losing banner opened once and
dismissing to the menu.

`SDD-Task: 0121-mission-flow/T2`
