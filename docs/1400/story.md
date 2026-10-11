# Loot dots on the minimap

## Intent

The mission minimap marks every ground sack with an amber dot. A Game Options
entry, "Loot dots on minimap", switches the dots; it is on by default. Owner
direction (tester's mod list); the original draws no sack on the minimap as far
as the cited claims go (DIV-2954). Base: public main `058e706f`, merged with
`649a5512`; knowledge pin unchanged (k238).

## Authority

Owner direction. `MISSION-063` lists the classes the original's minimap paint
draws (structure, bridge, unit, air unit) and is silent on whether any of three
unnamed classes is a sack. No ROM1 behaviour is claimed.

## As built

### Visibility test

A sack gets a dot when `sackLayer` would admit it to the mission map
(`Viewer.minimapLootMarks`, `pkg/ui/minimap.go`):

1. its cell is inside the grid;
2. `fogGateSack` holds: the cell is `FogVisible` now (an explored or unseen cell
   draws no sack, and no dot);
3. its frame index selects a frame of the sack sheet that is not nil.

Seat ruling: the dot follows the fog test, not final map pixels. A sack in a
cell the player's sight makes visible shows its dot also when later map art (a
tree canopy, a building) covers the sack's pixels, as unit marks show under a
canopy. The fog rule stays: a hidden cell shows no dot.

`TestMinimapLootDotsMatchTheMapsSacks` compares the two over every fog state,
an out-of-grid cell and two bad frame indices.

### Marks

`minimapMark` gains `Loot`. `composeMinimapLayer` draws every loot mark first,
then the unit marks in their existing input order, then the view outline. A unit on a sack
shows the unit's colour. `minimapLootColor` is amber `(255,200,32)`, apart from
the local green, the other red, the pale green view outline and the unseen black.
There is one painter; no second one was added.

### Option

`GameOptionLootDots` is the twelfth `GameOption`. Its row is the last option row
of the Game Options page, built by the page's one row builder
(`gameOptionsPageRows`); its caption is the word `options.loot_dots` in the
engine word tables (EN "Loot dots on minimap", RU "Добыча на миникарте"). It is
stored in the profile options file under `LootDots` and read at start by
`LoadOptions`. A profile without the key shows the dots. The value rides in
`ui.GraphicsOptions.HideLootDots` (zero value: dots shown), which `FrontEnd`
already holds, passes to every mission viewer and updates on the map when the
option changes. It is application presentation state: it is in no World, World
hash or SAV.

The Game Options frame grows one frame-tile row (64 pixels): the dialog is
built from a 560 by 544 argument rectangle, snaps to 488 by 488 and sits with its
top edge on the screen's top edge (`newTopDialogGeometry`), since a frame as tall
as the screen cannot be centred. The loot row is a left-column check box at
y 364..388 from the frame's top, under Timed Autosave. Every other control, the
speed caption and the title keep their offsets from the frame's top-left corner
(`TestGameOptionsPageKeepsEveryOriginalControlOffset`); OK and Cancel keep their
24-pixel height and move down 64 pixels with the frame's bottom. On the screen
every control is 28 pixels higher than before, because the frame is. MENU-073
gives no pixel position the engine's own controls are bound to.

### ROM2

The ROM2 mission minimap is the same viewer and composer. On both ROM2 roots
the first mission's minimap shows the dot for a dropped sack (witness below), so
the change is `[BASE]`. The ROM2 original's minimap is not researched; the
divergence row covers both games.

## Proof

Focused tests: `pkg/ui/minimaploot_test.go` (visible sack dot, loss control no
sack; explored and unseen sack, loss control visible sack; parity with
`sackLayer`; option off, loss control option on; unit over sack in both mark
orders, loss control sack alone; colour distinct) and `pkg/game/lootdots_test.go`
(fresh profile shows dots; off persists to a second front end reading the same
file; back on persists; a graphics write keeps the value).

Installed witness `TestReleaseMinimapLootDotsFollowTheOption`: the first
mission of each root, a gold drop on the nearest unit-free visible cell, the
framed minimap read through `HeadlessMinimap` and `HeadlessMinimapLootAt`.

| root | sack cell | option on | option off | unseen sack (38,64) on | World hash on = off | SAV bytes on = off |
|---|---|---|---|---|---|---|
| EN | (17,65) | loot colour | none | none | 2a3007d41e36757d | 216130, equal |
| RU | (17,65) | loot colour | none | none | 2a3007d41e36757d | 215615, equal |
| ROM2 EN | (11,40) | loot colour | none | no unseen sack in the mission | 771c898e75500b24 | not exported in this route (the direct first-mission route has no second-game save); hash only |
| ROM2 RU | (11,40) | loot colour | none | no unseen sack in the mission | 771c898e75500b24 | as ROM2 EN |

The second game has a current-SAV producer through its campaign route.
`TestReleaseSecondGameLootDotsLeaveTheSaveUntouched` enters the first campaign
mission through the town route, draws the minimap with the option on and off and
compares the World hash and the exported SAV bytes: equal on both ROM2 roots
(World hash 28436e9ba4216677; SAV 190401 bytes on ROM2 EN, 190792 on ROM2 RU).
The witness writes JSON only, no picture.

Release tests (`Minimap|GameMenu|GameOptions|MapReveal|Census|TimedAutosave|
SettingNotice|FrameKit|LootDots`, `pkg/game`), RU then EN, by direct
`go test -run` because `check-release-tests.sh` selects no tests.

## Moved goldens and tests

No pixel golden or digest moved. The Game Options frame moved with its page, so
three tests follow the frame's new origin (76,0) instead of (76,28):
`TestReleaseGraphicsOptions1190InstalledMenusAndColdLoad` clicks (164,144),
(164,170), (164,196), (164,222) instead of y 172, 198, 224, 250;
`TestReleasePauseMenuVisitsEveryDestination` clicks the slider at (227,88)
instead of (227,116); `TestGameOptionsContentHasEqualMarginsAndClearsItsFrame`
takes the frame body from `gameOptionsDialog`. `TestReleaseGameOptions1186PreferencesAndCheckpoints` moves its four
control points and rectangles up 28 pixels (retreat Medium (400,356) to (400,328),
day/night (196,146) to (196,118), health (424,88) to (424,60), damage (424,116)
to (424,88)). Because the page has one more
row: `TestGameOptionsClicksWriteNothingUntilOK` (19 rows),
`TestGameOptionsOKWritesInOrderAndForcesLighting` (one more write),
`TestOptionsDialogControlsLieInsideTheirFrames` (OK and Cancel at y 436..460),
and the timed-caption row index in `TestReleaseTimedAutosaveCurrentStateAndOptions`.

## Open debt

- The dot is one cell wide at the minimap scale and stacks with nothing; many
  sacks in one cell are one dot.
- The ROM2 original's minimap is unresearched (DIV-2954 stays ACCEPTED on owner
  direction).
- The option has no shortcut key.
