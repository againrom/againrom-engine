# Headless scenarios

`againrom --headless` runs a versioned JSON scenario without creating a window,
renderer, sound device, or physical-input adapter.

A scenario declares a **stage**, which is the driver its steps are dispatched to.
The two stages are disjoint and a file names exactly one of them.

| Stage | Driver | What a step reaches |
|---|---|---|
| `frontend` (default) | the production `ui.App` and controller | menus, saves, the party sheet, the live map screen |
| `mission` | one campaign mission's `sim.World` | units, orders, ticks, the mission's own verdict |

They are not mixed in one file. A front-end scenario reaches a live world only
where the controller put one there, and which world that is depends on the menu
path the file walked; a mission scenario names its world outright.

`cmd/missionrun` remains a separate developer tool. It is the instrument
`pipeline/check-milestone.sh` drives and its argv is not changed by this
directory. What the two share is the **unit reference grammar**, so a drive
written as `missionrun` flags transcribes into a mission scenario unchanged.

## Running one

```powershell
$env:AGAINROM_ASSETS='C:\path\to\lawful-install'
go run ./cmd/againrom --headless scenarios\0155-mission10-escort.json
```

The asset root comes from `-assets` or `AGAINROM_ASSETS`; the two are resolved in
that order and no path is compiled in.

Standard output is JSON Lines, one event per step. A front-end event carries a
`state` object and a mission event carries a `world` object. Standard error is a
compact trace with one row per step. The process exits non-zero on the first
failed step, naming the step number and the command.

## Asset providers

A scenario also declares **where its bytes come from**, with `assets`.

| `assets` | Source | Runs under `go test` |
|---|---|---|
| `install` (default) | a lawful root from `-assets` or `AGAINROM_ASSETS` | no |
| `synthetic` | the world the scenario itself authors, under `world` | yes |

They prove different things. `synthetic` proves mechanism: the rule fired, the
state changed, the order is right. It cannot prove this build reads the
original's bytes correctly, because the bytes were authored here from the same
belief the decode holds. `install` proves fidelity of ingestion: the real archive
opens, the real map loads, the counts come out. Neither states what the original
game does; only a research claim does.

A `synthetic` scenario needs no asset root at all:

```powershell
go run ./cmd/againrom --headless scenarios/0155-synthetic-melee.json
```

Its `world` block states `width`, `height`, an optional `seed`, a `units` list,
an optional `hostile` list of `[from, to]` owner-slot pairs and an optional
`spells` list. Each unit declares the `ref` its own steps address it by, its
cell, `hp`, and optionally `max_hp`, `owner`, `group`, `damage`, `scan_range`,
`always_hits`, `mana`, `max_mana`, `mind`, `known_spells` and `autocast`. The
world is composed from that declaration; nothing is ever cut out of a real
archive.

`max_hp` defaults to `hp`, so a unit that states none starts at full health; a
unit that states a larger `max_hp` starts wounded, which is what a heal is aimed
at. `known_spells` is a list of ids rather than the bitmask the simulation
holds, so a file names spells rather than bits.

Each row of `spells` states `id` and optionally `mana_cost`, `school`,
`max_range`, `damage_min`, `damage_max`, `targets_unit` and `arm`. `arm` is
`damage` or `heal`; a row that states neither is one this build has no arm for
and every cast at it is refused, which is the state the shipped table's own
sixteen non-damage rows are in.

The front-end stage runs on `install` only.

## Versions

`version` is the vocabulary the file is written in. This build reads versions 1
through 8.

A file keeps exactly the vocabulary of the version it declares. A version-1 file
is refused the moment it names a stage, an asset source, a mission number, a
difficulty, a world, or a later command; a version-2 file is refused the moment
it names a version-3 command; a version-3 file is refused the moment it names
`pointer`, `assert_inventory`, `assert_shop` or `window`; a version-4 file is
refused the moment it names `abort_game`; a version-5 file is refused the moment
it names a direct mission action. An older scenario therefore runs unchanged
and cannot acquire behaviour its author never wrote. New scenarios are written
at version 8. Version 7 adds only frontend transition observations: the
`control` wait and the `mission`, `notice`, and `town_place` state assertions.
Older files cannot name those fields.

Version 8 adds `quick_spells` (four real IDs, zero unbound), `current_spell`
and `spell_armed` state assertions, plus pointer action `hover` and the
`{"spell": 16}` form for
a visible book cell. It addresses the actual cell, including an unavailable
one, and never sets a binding or cast mode directly. F5–F8 and Ctrl+F5–F8 are
ordinary `key` aliases (`f5` / `ctrl-f5`).

**The version is recorded per stage, not per command.** `wait_until` reached the
mission stage in version 2 and the front-end stage in version 3, so a version-2
mission file still uses it and a version-2 front-end file cannot.

## Version 1 commands — stage `frontend`

Top-level `original_saves` and `saves` name the save directories, relative to the
scenario file.

| Command | Parameters | Effect |
|---|---|---|
| `key` | `key` | Dispatches `enter`, `escape`, `load`, or an arrow key through `ui.App`. |
| `activate` or `select_control` | `target` | Selects and activates one current production control. |
| `open_menu` | `target` | Opens the `game` or `load` menu. |
| `wait_ticks` | `ticks` | Advances the production simulation by exactly this many ticks. |
| `capture` | `name` | Stores the current state for later comparisons. |
| `select_member` | `id` | Selects a live actor through the production screen hit test. |
| `save` | none | Uses the production game menu save action. |
| `load` | `target` | Uses the production load menu. `@first` selects its first enabled row. |
| `assert_member` | `member` | Checks one stable member ID, XP, skills, membership, carried load, capacity, or a prior capture. |
| `assert_state` | `state` | Checks screen, purse, documents, exact member count, and zero or more members. |

`activate` on the main menu presses one brooch button: `NEW GAME`, `LOAD GAME`
or `EXIT`. The point is found by asking the production hit test, so a scenario
names the button rather than a coordinate. `EXIT` ends the run with an error,
because a scenario that quits the application has no next step.

`scenarios/0152-save666.json` is the worked example.

## Version 3 commands — stage `frontend`

| Command | Parameters | Effect |
|---|---|---|
| `create_character` | `character` | Generates one character on the production generation screen and begins the mission it was armed for. |
| `wait_until` | `until`, `ticks` | Steps the application until the condition holds. |

### `create_character`

The generation screen is reached the way a player reaches it: `activate
"NEW GAME"`, then `activate` the mission row, which arms the generator for that
mission.

The step drives that screen through the same focus, Enter and typed-character
dispatch a keyboard session produces. It presses no control this package invented
and calls no method of the generator directly, so every refusal it reports is the
screen's own refusal, in the screen's own words.

```json
{"command": "create_character", "character": {
  "name": "Aeryn", "sex": "Female", "class": "Mage", "skill": "Air",
  "stats": {"Mind": 30, "Body": 20}}}
```

`sex`, `class` and `skill` are the option labels the screen shows, matched
without regard to case or surrounding space. `stats` maps a statistic's own name
to the value it is bought up or down to, one press of that statistic's own
control at a time. Every field is optional and a field not named keeps what the
generator opened with, so a file naming a class alone asserts nothing about the
spread.

The step fails, and hands nothing over, when the screen refuses: a statistic the
budget cannot pay for, a value outside its floor or ceiling, an option label no
row offers, or a name the screen reserves. A refusal names the offered labels.

The order inside the step is the screen's own: the name, the picture, forward,
the skill, the statistics, play. The skill is set before the statistics because
moving forward reseeds the skill row off the class it commits.

Every front-end event carries a `chargen` object while the generation screen is
showing and carries none otherwise. It holds the stage (`precreate` or
`detailed`), the focus, the name, the remaining budget, whether the spread is
legal, and one row per control with its `kind`, `label`, `value` and whether it
is `chosen`.

### Front-end conditions — `until`

| Form | Holds when |
|---|---|
| `{"screen": "town"}` | that screen is showing — `menu`, `picker`, `map`, `chargen`, `town`, `gamemenu`, `load` |
| `{"notice": true}` | a notice is open over the map, which is how a decided mission waits |

The condition is tested before the first step, so a condition that already holds
costs no tick. `ticks` defaults to 40000 and a ceiling reached without the
condition fails the run.

**The wait presses nothing.** A notice it stopped on is still open when the next
step runs, so a scenario crossing out of a mission alternates `wait_until` with
`activate "notice"` rather than relying on a tick count it had to know in
advance.

`scenarios/0163-chargen-mission10.json` generates a character and enters mission
10 with it. `scenarios/1025-mission10-weight.json` does the same and asserts the
hero's carried load and carrying capacity inside that mission. The capacity is
the formula's own value for the body statistic the file states (`Body x 10 + 1`,
so 201 for a body of 20) and does not depend on any shipped table. The load is
what the shipped weight column resolves to for the starting weapon the generator
gives him, and it is therefore an ingestion pin: it is the same on both roots.

## Controlled mission-20 transition fixture

`0163-mission-to-town.json` and `1013-world-map-one-click.json` require the
explicit native fixture below. They check a fresh mission's completion pages,
Victory, the homeward world-map route, automatic town arrival, party state,
purse, documents, tavern work and departure to mission 30. The `1013` departure
uses exactly one pointer press/release. Both preserve Danath and Reniesta's
independent character state through the next mission.

From the implementation checkout, with an empty output directory outside the
lawful install and source directory:

```powershell
New-Item -ItemType Directory ../review/legacy-notice-hotfix/endpoint-en
go run ./cmd/scenariofixture -assets ../gameversions/en -source ../gameversions/saves/2026-08-02/game0009.sav -out ../review/legacy-notice-hotfix/endpoint-en
go run ./cmd/againrom -assets ../gameversions/en -saves ../review/legacy-notice-hotfix/endpoint-en -headless scenarios/0163-mission-to-town.json
go run ./cmd/againrom -assets ../gameversions/en -saves ../review/legacy-notice-hotfix/endpoint-en -headless scenarios/1013-world-map-one-click.json
```

Repeat with the RU root and a new empty output directory. The generator refuses
an existing output population and never overwrites it. It prints source/output
SHA-256 values and labels the save `fresh mission 20 endpoint - save666 party`.
The source is pinned to SHA-256
`60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6`.

The generator imports the saved party, campaign and purse, then opens a **new**
installed mission 20. In a detached native snapshot it places escort `u136`
from `(10,13)` at `(111,132)` with the existing headless relocation seam.
It advances no tick and changes no fresh script, latch, counter or outcome.
The scenario then lets that mission's script produce its three completion
dialogue pages and Victory. The purse is 600 before the script, 1100 after its
500 reward, and 1600 after campaign completion. Generated saves stay outside Git.

This proves the UI transition and character continuity from a controlled
endpoint, **not original-world continuation, escort gameplay or a campaign
playthrough**. Save 666 already carries spent completion latch 7 as well as
spent message latches. Its original outcome/counters are not imported. Resetting
those latches or replaying messages would hide that separate SAV debt. The
fixture does neither; the original-import regression still checks no replay.

## Version 7 observations — stage `frontend`

`wait_until` with `{"control":"TAVERN"}` waits for a choosable list row whose
label matches exactly or starts with that whitespace-delimited name. It only
steps the App; it does not activate a row. Town's `screen` value also covers the
world map, so waiting for `screen:town` alone cannot prove automatic arrival.

`assert_state` additionally accepts `mission` (live mission number), `notice`
(`none`, `dialogue`, `victory`, `defeat`, or legacy `outcome` on the map), and
`town_place` (`world_map` or `square`). These are read-only observations. They
add no mission action to the frontend stage.

## Version 4 commands — stage `frontend`

| Command | Parameters | Effect |
|---|---|---|
| `pointer` | `action`, `at` | Dispatches one pointer edge — `press`, `move` or `release` — at one named surface. |
| `assert_inventory` | `inventory` | Checks the map screen's doll, worn array, container and the ground's sacks. |
| `assert_shop` | `shop` | Checks the open shop room's doll, worn array, pack and table. |

Top-level `window` states the window size in pixels: `{"w": 1600, "h": 1200}`.
It reaches `ui.App.Layout`, the same door a real window creation or resize uses.
A screen that has no room at the stated size refuses to open, so a scenario
driving an inventory states a size large enough for one. `window` belongs to the
front-end stage; the mission stage has no screen and refuses the field.

### The pointer step

Before version 4 every step named a control by name — a row, a button, a member
id. The inventory has no names: its surfaces are pixels of a composed figure, and
the gesture under test is a press, a move and a release over them.

A point names exactly one surface. The pixel is asked of the production hit test
each time, never computed in the scenario, so a file keeps meaning the same thing
when a layout constant moves.

| Form | Surface |
|---|---|
| `{"doll_slot": 1}` | equipment slot 1..12 of the map screen's doll, at a pixel that figure's own slot mask answers for |
| `{"doll_box": true}` | inside the doll box, on no particular slot — where a pack-origin drag is released to equip |
| `{"pack_cell": 3}` | a pack bar cell, named by the container element index it stands for (the scroll is folded in) |
| `{"pack_code": 33076}` | that same cell named by what it holds |
| `{"ground": true}` | a map pixel no inventory box, HUD toggle bar or spellbook strip claims — where a release becomes a ground drop |
| `{"shop": {"surface": "doll", "index": 1}}` | a shop room surface: `doll` (slot 1..12), `shelf`, `pack`, `table` (0-based), `picker_prev`, `picker_next` |
| `{"shop": {"surface": "pack", "code": 265}}` | a `pack` or `table` cell named by what it holds |
| `{"shop_idle": true}` | a shop frame pixel naming no control |
| `{"world_map_mission": 30}` | the world map's own scroll card for mission number 30, named by campaign number, not display slot — `WorldMapClick`'s selection surface |
| `{"world_map_town": true}` | the world map's separate return-to-town card |
| `{"world_map_miss": true}` | a world map pixel naming neither a card nor a mission region — `WorldMapClick`'s skip arm |

Prefer `pack_code` and `shop.code` to an index. A container's order is a fact
about the run — an unequip appends, a sale removes — so a file naming a position
asserts a layout it did not choose.

A drag is three steps; a tap is a press and a release at the same point.

```json
{"command": "pointer", "action": "press",   "at": {"pack_code": 33076}},
{"command": "pointer", "action": "move",    "at": {"doll_box": true}},
{"command": "pointer", "action": "release", "at": {"doll_box": true}}
```

**A popup stops every gesture.** An open notice clears the map screen's drag
state and the pointer never reaches the inventory. An original save can already
contain Victory: assert that state, then choose Victory or use Escape for
Continue before inventory gestures. Do not replay a fixed count of historical
notices. After Victory, wait for the destination town control while the automatic
return completes. The shop room opens behind its own dialogue for the same
reason: press `dialogue` until the room is reachable.

The1005 carry scenario takes Continue, moves the armour, and saves/reloads that
changed world before choosing the restored Victory. The1090 consumable scenario
takes Continue after each original/native load. The1020 abort witness completes
the original saved victory once, so purse600 becomes1100; the separate fresh
endpoint fixture for1013/0163 runs its own script and ends with1600.

**A selection is visible one tick later.** The inventory subject is composed from
the selection of the previous frame, so `select_member` is followed by
`wait_ticks 1` before anything asserts what the doll shows.

Idle frames and key edges retain the last observed map pointer position; they
do not invent a pointer at the top-left window edge. An explicit edge pointer
still scrolls. The semantic ground target avoids HUD and edge-scroll bands.
An open bottom panel can cover another actor: `key space` closes the panels
before selecting that actor, then opens them again for inventory gestures.
The1005 shop witness uses a wide window so the hero's last pack cell is visible;
selection and item targets still require ordinary visible hit regions.

### Inventory and shop assertions

`assert_inventory` takes `subject` (the entity id), `weapon_fallback` (whether
the doll is drawing a starting weapon the equipment array does not hold),
`figure` and `equipment` (each a list of `{"slot": n, "code": c}` or
`{"slot": n, "empty": true}`), and `carries` and `sacks` (each a list of
`{"code": c}` with an optional exact `count` or `absent: true`).

`figure` is what the doll DRAWS and `equipment` is what the array HOLDS. The two
differ exactly where a starting weapon is drawn without being equipped, which is
the state `weapon_fallback` names.

`assert_shop` takes `member` (the roster index the room is showing), `gold`,
`doll` and `worn` (slot clauses), and `carries` and `table` (code clauses).

`scenarios/1005-doll-and-shop.json` is the worked example: it loads an original
save, trades in the town's shop room with a real shipped weapon, walks into the
next mission, and equips, unequips and ground-drops through the map screen's own
doll, asserting both a real hero and a companion that arrives with a starting
weapon drawn.

`scenarios/1013-world-map-one-click.json` presses and releases once on a
mission's own world map scroll card, then `wait_until`s the map screen with no
further pointer step of any kind: the reveal ticks and the mission opens on
their own, which is the one-click arrival contract (`TOWN-121`, `DIV-135`
closed). The wait's own `HeadlessStep` loop is what advances the cadence gate
`worldMapTickInterval` reads (`app.go`), so this file's `ticks` bound is in
`HeadlessStep` calls, not in `WorldMapTick` firings.

## Version 5 commands — stage `frontend`

| Command | Parameters | Effect |
|---|---|---|
| `abort_game` | none | Escapes to the game menu if not already there, then uses the production ABORT GAME action. |

`abort_game` is shaped like `save`: it escapes to the game menu from the map or
the town if the run is not already showing it, then chooses the ABORT GAME row
by its production action (`ui.gameMenuAbortGame`) rather than by its displayed
text, which is the install's localized `MenuAbort` word. The step fails unless
the screen afterward is `menu`.

`scenarios/1020-abort-witness.json` is the worked example: it loads an original
save into a town, aborts the running game from the game menu, starts a NEW GAME
into a different mission, and asserts that the purse and party are the new
game's own rather than the aborted one's.

## Sweeping the campaign

`scripts/campaign-sweep.sh` drives every campaign mission in ascending order and
prints one row each: the tick it stopped at, the outcome, the compiled and
reached script-gap counts, the live and fallen entity counts, and the digest.

```powershell
$env:AGAINROM_ASSETS='C:\path\to\lawful-install'
bash scripts/campaign-sweep.sh
bash scripts/campaign-sweep.sh --ticks 20000 --census
bash scripts/campaign-sweep.sh --check      # run twice and diff
```

It issues no orders, so the drive is unattended and most missions end
`undecided`. Its exit code ignores the outcome word entirely and is non-zero only
when a mission failed to load or the drive returned an error. What it measures is
the census and whether 28 campaign maps run without failing.

## Version 2 commands — stage `mission`

Top-level `mission` is required. `difficulty` is `easy`, `normal` (the default)
or `hard`. `mage` starts the party's hero as a caster.

| Command | Parameters | Effect |
|---|---|---|
| `order` | `unit`, `order`, and `x`/`y` or `target` | Issues one production simulation command. |
| `wait_until` | `until`, `ticks` | Steps until the condition holds. Failing to reach it inside `ticks` fails the run. |
| `wait_ticks` | `ticks` | Steps exactly this many ticks, stopping early if the mission is decided. |
| `assert_unit` | `unit`, `expect` | Checks one unit. |
| `assert_world` | `world` | Checks the mission as a whole. |
| `report` | `name` | Emits a named event carrying every unit's position, health, slot and group. |

### Unit references

`uNN` is the identifier the map's own script uses for a unit. `pN` is the
party's Nth member, in start order. `eNN` is a raw entity id, which is what a
world with neither a script table nor a party is addressed by.

### Orders

`move`, `patrol` and `march` take `x` and `y`. `attack` takes `target`, a second
unit reference. `guard` and `stand` take neither.

`cast` takes both `target` and `spell`, a spell id the world's own table holds.
`autocast` takes `spell` alone and stores it as the unit's autocast setting; a
`spell` of `0` clears it, so the field is required rather than defaulted — a
step with no `spell` is refused rather than read as a clear.

Both reach exactly the production commands a player's own click and key produce.
Whether a cast is affordable, in range, known and aimed at something the row
admits is decided inside the simulation, so a refused cast is a step that
changes nothing rather than a step that fails.

### Conditions — `until`

Exactly one form per condition.

| Form | Holds when |
|---|---|
| `{"unit": "u21", "within": {"x": 56, "y": 21, "radius": 3}}` | the unit is within that Chebyshev radius of that cell |
| `{"unit": "u21", "dead": true}` | the unit has fallen or the world no longer holds it |
| `{"outcome": "decided"}` | the mission has a verdict; also `won`, `lost`, `undecided` |

`ticks` defaults to 40000, which is `cmd/missionrun`'s own ceiling.

### Unit assertions — `expect`

`alive`, `x`, `y`, `within`, `hp_at_least`, `hp_at_most`, `owner`, `group`,
`mana_at_least`, `mana_at_most`, `autocast` and `carries`. Every field is
optional and an assertion must state at least one.

`autocast` is the spell id the unit casts unbidden, `0` asserting that it casts
none.

`carries` is a list of clauses about the unit's own container. Each names a
packed item `code`, and optionally either an exact `count` or `absent: true`;
a clause naming neither holds when the unit carries at least one of the code.
The code is a number and not a name: the container this reaches holds codes and
the simulation tier carries no item-name table. Mission 30's quest item is 3614.

```json
{"command": "assert_unit", "unit": "p0", "expect": {"carries": [{"code": 3614, "count": 1}]}}
```
### World assertions — `world`

`outcome`, `tick_at_least`, `tick_at_most`, `alive_at_least`, `fallen_at_most`,
`latched`, `not_latched`, and `unsupported_at_most`.

`unsupported_at_most` is the **compiled** script-gap census: how many arms of this
mission's own compiled script this build cannot run, whether or not anything
walks into them. It is the number `pipeline/check-milestone.sh` tracks.

`reached_unsupported_at_most` bounds the **reached** census: how many unrunnable
arms the drive actually arrived at. Every event reports both as `unsupported` and
`reached_unsupported`, and a `report` step carries the per-opcode breakdown under
`census`, including `unresolved` — checks whose unit, group or player reference
did not resolve, which leave a live trigger comparing a stale register.

Every event also carries `hash`, the simulation's own digest. `pkg/sim` steps on
a fixed integer tick with no clock, no `math/rand` and no floats, so the same
scenario over the same assets yields the same digest. It is reported, not
asserted: a digest is meaningful only against byte-identical assets.

An outcome is **recorded, not asserted**, unless the scenario's own orders make
it deterministic. An unattended drive that loses is not a defect — mission 10 is
an escort and an unattended drive loses by design.

`scenarios/0155-mission10-escort.json` and `scenarios/0155-mission20-sweep.json`
are the install-tier worked examples, and `scenarios/0155-synthetic-melee.json`
is the synthetic one. The two mission scenarios produce the same digest over both
preserved roots.

`scenarios/0159-mission40-join.json` drives mission 40's own script to the
hand-over: the player walks to the paladin, the map's `Paladin` trigger fires,
and the paladin changes from roster slot 2 and group 14 to slot 1 and a group of
his own. It asserts that fresh group by number, so it is a scenario about what
the hand-over writes and not only about the owner.

## Version 6 commands — stage `mission`

Version 6 adds direct semantic actions for campaign-script reachability. Each
action changes ordinary world state and then advances one production mission
tick. None can set a script register, trigger latch, counter or outcome; a route
passes only when the mission's own compiled script raises Victory.

| Command | Parameters | Effect |
|---|---|---|
| `kill` | `unit` | Completes that creature's ordinary terminal death, including loot and decay transition. |
| `kill_player` | `player` | Applies the same terminal death to every creature owned by that script player. |
| `teleport` | `unit`, `x`, `y` | Places an on-map creature at an exact in-bounds cell and clears its pending action state. |
| `pick_item` | `unit`, `x`, `y` | Takes the whole sack at that cell through the world's ordinary sack-transfer path. |
| `heal` | `unit` | Restores a restorative creature to authored maximum health through the ordinary revival transition. |

`teleport` deliberately bypasses terrain, occupancy, sight, range and mana. It
is a test-driver relocation, not the gameplay Teleport spell; bounds and actor
presence still apply. `kill_player` refuses an owner absent from the world, so a
mistyped player slot cannot silently pass as an empty faction.

The 28 `1060-campaign-*-reachable.json` scenarios cover every shipped campaign
mission: 10, 20, 30/31, 40/41, 50/51, 60/61, 70/71, 80/81, 90/91, 100/101,
110/111, 120/121, 130/131, 140/141 and 150/151. They use the smallest direct
state changes needed to walk authored trigger prerequisites, then wait for and
assert actual `won`. The suite proves script reachability, not combat balance,
terrain traversal or a human-speed playthrough.

## Strictness

The parser rejects unknown JSON fields, unknown commands, unknown stages,
parameters that do not belong to a command, a command from the other stage, a
command from a later version, unsupported versions, and trailing JSON values.
`wait_ticks` and `wait_until` use fixed tick counts and read no wall clock.
