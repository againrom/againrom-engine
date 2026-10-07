# 1347 — ROM2 mission 10 opens

## Result

An asset root is one of two games, decided from its files. A second-game root
(`allods2.exe` and `scenario.dll` present) selects the second game's data file
layout, map reader and placement lookup. Mission 10 (`Scenario\10.alm`) loads
from `rom2-ru` and `rom2-en` headless, builds a world, renders through
`terraintool` and ticks. No first-game behaviour changes. The game window and
the starter do not open a second-game root.

## Authority

Pin k175. Claims read one at a time.

- `R2-ASSET-029`: the data file keeps the first game's grammar with a 14-byte
  armor and shield raw block. Units serverID is slot 55; Humans serverID slot 24.
- `R2-ENGINE-033`: a mission is `<n>.alm` under `Scenario\` (RU build only).
- `R2-ENGINE-036`: the first game's type-7 grammar parses all 46 campaign maps;
  the vocabulary adds instants 35 to 39 and checks 23 to 27; opcode 65538 is
  undeclared.
- `R2-ENGINE-038`: a placement resolves its server id against Units slot 55, or
  Humans slot 24 for the person arm.
- `R2-ENGINE-039`: the first game's sprite grammars parse every second-game frame.
- `R2-ENGINE-037`: mission text is `text/mission<N>.txt`; its conversion is
  Unknown, so no mission text is loaded.

No first-game claim is authority for a second-game rule. The differences are
`DIV-2354` to `DIV-2357` in `docs/divergences/rom2.md`.

## As built

- `pkg/base`: `Game` (`GameROM1`, `GameROM2`), profiles `rom2-en`, `rom2-ru`
  and `rom2`, the build hashes, and `Limits` stating no character generation,
  first mission 10 and no original-save reading. `DetectIn` chooses the game
  from the marker files, then the build and language within it.
- `pkg/formats/databin`: `Layout` with `ROM1Layout` and `ROM2Layout`;
  `ParseWith` takes the raw block width. `Parse` is `ParseWith(ROM1Layout)`.
- `pkg/formats/alm`: `OpenROM2`. Format versions 1300 to 1600; type 0 occupies
  660 bytes though it declares 644; the record tag is not checked; types 10, 11
  and 12 are kept raw after their sizes are checked against the type-0 counts;
  the unit record is 48 bytes and carries the server id. `Open` is unchanged.
- `pkg/data`: `FindUnitByServerID`. `pkg/mapload`: `Table.Game`; `Resolve`
  takes the server id route for a second-game table, flag `0x10` selecting
  Humans.
- `pkg/game`: `LoadDefinitionsFor`, `Archives.Game`, `StartMission` opens a
  second-game map for a second-game table, and `NewFrontEnd` refuses a
  second-game root by name. `cmd/missionrun` and `cmd/terraintool` accept one.
- Item 4 (starter selects a second-game root) is not delivered: the window needs
  the second game's menu art, which differs in size from the first game's.

Unit record offsets and the Units search direction are derived from the 46 maps
per root and are not research readings (`DIV-2357`).

## Proof

Synthetic tests: `pkg/base` (root detection, profile limits),
`pkg/formats/databin` (wide block, each layout refuses the other's file),
`pkg/formats/alm` (660-byte type 0, tag 5 and 7, versions, extension sizes, unit
fields, `Open` still refuses the file), `pkg/data`, `pkg/mapload`.

Release tests, run against one second-game root each with
`AGAINROM_ASSETS=<root> go test -run 'TestReleaseSecondGame' ./pkg/game`:
`TestReleaseSecondGameMissionTenOpens` asserts 80x80, 29 units, 16 objects, one
world entity per unit plus the party, four unsupported arms all instant 36,
and 300 ticks with the outcome undecided. `TestReleaseSecondGameCampaignMapsDecode`
opens every scenario map present. Both skip on a first-game root without naming
a variable, so the unchanged `pipeline/check-release-tests.sh` still runs the
first-game suites on `gameversions/en` and `gameversions/ru` and counts them as
lacking a subject for the second game.

Witness (uncommitted): `review/story1347/` holds the headless render of mission 10.

## Open debt

- The game window and menu art for a second-game root; starter selection.
- Instants 35 to 39, checks 23 to 27, opcode 65538 (`DIV-2354`, `DIV-2355`).
- `scenario.dll` campaign driving, mission text, party carry-over, character
  generation, saves (`DIV-2356`).
- Research questions: placement record layout and lookup order (`DIV-2357`).
