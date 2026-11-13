# Mod data files

A mod is one folder with `mod.toml` and optional data, assets and Starlark logic.
Every held data file loads automatically; an optional script runs `init(game, settings)`.
Every refusal stops the launch and names the mod, the file and the line.
`againrom -dump-base <folder>` writes the install's own records in this
format (section Base dump). `againrom --headless <scenario.json>` runs the
game with no window and writes the frame of a town (section Headless runs).
A mod's `files` folder replaces install pictures, sounds and texts (section
Install files). A mod folder packs into one file, `mod-<id>.res`, that the
game reads as the folder (section Packed mods).

## Fixed tree

This table is normative. `pkg/mod/kinds.go` defines it; `TestModsDocListsFixedTree`
checks these rows. `*` matches one path component, `**` any number of folders,
`{lang}` is `en` or `ru`, `{layer}` also admits `common`, and `<N>` is a campaign
mission number. Only the manifest is required.

<!-- fixed-tree:start -->
| kind | paths | formats | found by | language |
|---|---|---|---|---|
| items | `data/items.toml`, `assets/items/**/*.png` | TOML subset; PNG | held file; row sprite | shared |
| characters | `data/characters.toml` | TOML subset | held file | shared |
| screens | `data/screens.toml` | TOML subset | held file | shared |
| companions | `data/companions.toml` | TOML subset | held file | shared |
| spells | `data/spells.toml` | TOML subset | held file | shared |
| body sheets | `data/bodies.toml`, `assets/bodies/**/*.png` | TOML subset; PNG sheets | held file; row heroes and heroes_l | shared |
| weapon bodies | `data/wpnbodies.toml`, `data/weapon-bodies.toml` | TOML subset | held file | shared |
| terrain | `data/terrain.toml` | TOML subset; 8-bit BMP strips | held file; strip group via install path | shared |
| missions | `data/missions.toml` | TOML subset | held file; row map and texts | shared; {lang} in map |
| towns | `data/towns.toml`, `assets/towns/**/*.json`, `assets/towns/**/*.bmp` | TOML subset; JSON; BMP; UTF-8 TXT | held file; description; mod: art; talk install path | shared; talk texts per language |
| speakers | `data/speakers.toml`, `assets/speakers/*.png` | TOML subset; PNG faces | held file; row picture | shared |
| settings | `settings.toml` | TOML subset | held file | shared; labels per language |
| manifest | `mod.toml` | TOML subset | required fixed path | shared |
| entry script | `main.star` | Starlark | optional fixed path; existing manifest entry | shared |
| script modules | `**/*.star` | Starlark | script load | shared |
| strings | `text/{lang}/strings.toml` | TOML subset | fixed path per language | en or ru |
| mission maps | `maps/*.alm`, `maps/*.almx`, `maps/{lang}/*.alm`, `maps/{lang}/*.almx` | ALM or ALMX | mission row | shared or en/ru |
| mission pages | `maps/*.text/*.txt`, `maps/*.text/{lang}/*.txt`, `maps/{lang}/*.text/*.txt`, `maps/{lang}/*.text/{lang}/*.txt`, `<N>.text/*.txt`, `<N>.text/{lang}/*.txt` | UTF-8 TXT | beside map; row texts as lower layer | shared or en/ru |
| map variable labels | `maps/*.variables`, `maps/{lang}/*.variables` | JSON | beside map; editor only | shared |
| install pictures | `files/{layer}/**/*.bmp` | BMP | install path | common or en/ru |
| install sounds | `files/{layer}/**/*.wav` | PCM or Microsoft ADPCM WAV | install path | common or en/ru |
| install texts | `files/{layer}/**/*.txt` | UTF-8 TXT | install path | common or en/ru |
| map by number | `<N>.alm`, `<N>.almx`, `maps/<N>.alm`, `maps/<N>.almx` | ALM or ALMX | campaign mission number | shared |
| read-me | `README.md` | Markdown | fixed path; for people | shared |
<!-- fixed-tree:end -->

Every held catalog or configuration file loads without a script. Holding any of
those files except settings marks the mod as changing state, including empty data
files and speaker-only mods. Settings alone do not mark it. A script is optional;
an existing manifest `entry` remains supported. One explicit `game.data.add` of a
held file is accepted without adding records again; a second explicit call refuses.
Settings resolve before the files load, so a conditional script does not prevent a
held file loading.

Old paths named by rows remain readable and packed. Old mission map and page paths,
and the second weapon bodies name, produce check warnings naming the fixed place.
A title or briefing key with a page produces a warning because
the key wins. Maps not used by any mission and missing map declarations are left
to the map-list work. The packed name table supports components through 255 UTF-8
bytes, so names over 15 bytes need no warning.

Files outside this table and outside paths named by rows are warned about and
excluded from packing and the digest. Dotted names are ignored. The top-level
read-me is packed but excluded from the digest. Removing an unread file from an
old mod's digest changes its save mark once: a save made with the old digest will
report different files. A previously state-free speaker-only mod now enters the
save mark and can likewise make an older save report a different active mod set.

## Files a row names

A row that names a file of the mod (an item's `sprite`, a body's `heroes` and
`heroes_l`, a speaker's `picture`, a town's `description`, a mission's `map` and
`texts`) and a `mod:` art key of a town description all pass one rule:

- the value is a relative path inside the mod folder, written with `/`;
- no `\`, no `:`, no leading `/`, no `..` that leaves the folder;
- no component starts with a dot;
- each component holds at most 255 bytes and no control character;
- the value is UTF-8; a name that is not UTF-8 is refused;
- an item's sprite and a body's sheets end in `.png`, and a town's description of
  the mod ends in `.json`; case does not matter;
- the mod holds the file. A missing file is refused at the row's line.

An item's sprite and a body's sheets must be written clean (`assets/items/a.png`,
not `./assets/items/a.png`); the other values are read as their clean path.

A mod's own pictures and descriptions belong under `assets/<kind>/`:
`assets/items/`, `assets/bodies/`, `assets/speakers/`, `assets/towns/`. A file a row
names from another place still loads, plays and packs. `againrom -check` and the map
editor's overview of the mod print two warnings about these files:

```text
warning: mod "east": data/items.toml:7: sprite art/pike.png uses an old path; use assets/items/<name>.png
warning: mod "east": assets/items/lost.png: no row names this file of items; the game does not read it
```

The first names a file a row reads from outside `assets/`; a `mod:` art key outside
`assets/towns/` gets the same warning at its description file. The second names a
file under `assets/` that no row and no art key names. Neither stops the mod.

### Importing a file in the map editor

The map editor copies a picked file into the open mod at `assets/<kind>/<name>`.
`<name>` is the picked file's own name in lower case, cut to 255 bytes with its
extension kept. The file is checked at once by the reader the game uses for that
kind, and a file the game would refuse is refused there, with the same message:

| kind | file | checked as |
|---|---|---|
| items | `.png` | an item picture, 1 to 80 pixels a side |
| bodies | `.png` | a readable PNG sheet |
| speakers | `.png` | a face of the message window's size |
| towns | `.json` | a town description |
| towns | `.bmp` | a 24-bit or 8-bit BMP the game decodes |

The import is one undo step and writes nothing: the file is a draft of the project,
and Ctrl+S writes it with the rest of the mod. Undo takes the draft away; after a
save, undo and another Ctrl+S remove the file from the folder. When the mod already
holds a file of that name, the editor asks before it replaces it. A packed mod and
an imported read-only mod take no import.

Adding a speaker in the editor is this import: the face is copied to
`assets/speakers/<id>.png`. The other kinds have no page in the mod window yet; the
import of their files exists and is waiting for those pages.

## Data kinds

Every kind of data a mod carries is declared once in `pkg/mod/kinds.go`, and
a test holds this table to those declarations. A catalog kind is game data
the base holds as layer 0; each mod is a layer over it, keyed by the record
key. A merged kind combines the mods through one layer step: a record adds,
changes the fields it names, or switches a record off. A kept kind keeps its
own rules until the story named migrates it. A configuration kind has no base
catalog. Every held data file is read automatically before the optional entry script.

| kind | file | key | layer | two mods on one record | see |
|---|---|---|---|---|---|
| items | `data/items.toml` | the row: `key` (an added item or weapon row), `item` (a shipped row); a mask bit is one field of its row | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; one field is one number of a row or one mask bit (row, tier, material); two mods adding one key need `load-after` too | Items, `DIV-1861`, `DIV-4005`, `DIV-4006`, `DIV-4007`, `docs/1272/story.md`, `docs/1509/story.md` |
| characters | `data/characters.toml` | `key` (a character of the catalog); `target` (an edit of a definition row or a family of tier rows) | catalog, merged | catalog keys: yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a mod's character off; target edits: all operations apply in load order, including overlapping families and rows, without requiring load-after | Guest parties, `docs/1278/story.md`, `docs/1508/story.md`, `DIV-4002`, `DIV-4003` |
| screens | `data/screens.toml` | `key`, within the mod | configuration | no: a screen is its mod's own; one action is added by one mod | `DIV-1894`, `DIV-1906`, `docs/1277/story.md` |
| companions | `data/companions.toml` | `chapter` and `companion` | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a condition off | `docs/1280/story.md`, `DIV-3909` |
| spells | `data/spells.toml` | `target` and the field; `[global]` keys | catalog, kept until an unscheduled spell catalog story | no: one field of one spell, and one global key, is set by one mod | Spell damage columns, `docs/1341/story.md`, `DIV-2364` |
| body sheets | `data/bodies.toml` | `name` | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; a later sheet replaces the whole sheet; `off` switches a sheet off | Body sheets, `DIV-2913`, `DIV-2914`, `DIV-3910` |
| weapon bodies | `data/wpnbodies.toml` (also `data/weapon-bodies.toml`) | the weapon row, by `weapon`, `row` or both | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; a weapon row is never switched off | Weapon bodies, `DIV-2912`, `DIV-3911` |
| terrain | `data/terrain.toml` | `group` | configuration | no: a group is one mod's | New terrain tiles, `docs/1573/story.md`, `DIV-4412`, `DIV-4413` |
| missions | `data/missions.toml` | `number`; `key` within the mod | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; a later mission replaces the whole mission; `off` switches a mission off; one guest mission plays after one win | Missions, `docs/1408/story.md`, `docs/1508/story.md`, `DIV-4001` |
| towns | `data/towns.toml` | `id`; a tavern row by `town`, `chapter` and `npc`; an inn row by `topic` and `npc` | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a town, a tavern row or an inn row off | Towns, Tavern rows, Inn rows, `docs/1486/story.md`, `docs/1513/story.md`, `docs/1532/story.md` |
| speakers | `data/speakers.toml` | `id`, a record number from 20000 to 99999 | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a record off | Speakers, `docs/1624/story.md` |
| settings | `settings.toml` | the table name | configuration | no: a setting is its mod's own | `docs/1267/story.md` |

The base dump writes layer 0 of towns only; the other catalog kinds have no
dump yet (`docs/1496/story.md`).

## mod.toml

`mod.toml` holds top-level `key = value` lines; a `[table]` header ends the
keys. A key twice, a key not listed here and a wrong type are refused with the
file and line. A list may span lines.

| key | value |
|---|---|
| `id` | required string; the mod's name, equal to its folder name (a packed mod: the `<id>` of `mod-<id>.res`), 1 to 64 characters of `a-z 0-9 . _ -` starting with a letter or digit |
| `title` | required string; the name shown to the player |
| `version` | required string, not empty |
| `author` | optional string for people; shown and edited in W |
| `description` | optional string for people; shown and edited in W |
| `api` | integer, default 1; this build runs api 1 and refuses any other |
| `applies-to` | a string or list; default `["common"]`. Entries are `common` (every install), an install id such as `rom1-en`, or the family word `rom1` (every first-game install). The mod is refused on an install none of them names |
| `entry` | the Starlark file; default `main.star` when the folder holds one |
| `requires` | ids the mod needs, each with an optional constraint such as `"base>=1.0"` (operators `>=`, `<=`, `>`, `<`, `=`, dotted numbers). The required mod must be enabled and loads before this one |
| `conflicts` | ids that must not be enabled together with this mod |
| `load-after` | ids this mod loads after when they are enabled; it orders and does not require |

For compatibility, ordinary loading ignores non-string documentary `author`
and `description` values and non-ID names in `conflicts` and `load-after`.
W's authoring reader requires strings and valid dependency syntax when editing
or saving. Re-selecting a displayed target list outside its six choices keeps
that list unchanged.

```toml
id          = "second-town"
title       = "Second Town"
version     = "1.0"
author      = "A. Modder"
applies-to  = ["rom1"]
requires    = ["base-art>=1.0"]
load-after  = ["other-mod"]
```

### Load order and the starter's list

The game loads the enabled mods in one order: mods whose `applies-to` is
`common` first, then the others, each group by id, except that a mod always
follows the mods it `requires` and the enabled mods it names in `load-after`.
The order the mods were ticked in and the order of `-mods` do not count. The
launch and `againrom -check` refuse a set holding an `api` this build does not
run, an `applies-to` that does not name the base, an enabled mod a `conflicts`
names, a `requires` that is not enabled, cannot be used or is too old, or a cycle of `requires`
and `load-after`; they print the first refusal.

The starter's mod list shows, for the selected base, the enabled mods in that
load order, numbered from 1; then any enabled mod whose folder is missing or
whose `mod.toml` cannot be read; then the mods that are not enabled, by folder
name. Every refusal stands in red on the row of the mod it names, worded in
the language of the selected base, and Play and Check are refused while an
enabled mod carries one. A mod that is not enabled and does not fit the
selected base (its `applies-to` or its `api`) says so on its row and can still
be ticked. Selecting another base or ticking a mod refreshes the rows. Without
a selected base the enabled mods are listed by name, unnumbered.

### The mods folder, and starting a mod's mission from the starter

The starter, the game and the map editor use one mods folder. It is the folder
the starter's **Folder** field names (`dir` of `[mods]` in `starter.ini`), else
the folder `mods` beside the programs. The starter passes it to the game and to
the map editor as `-mods-dir`. A game or an editor started without the starter
and without `-mods-dir` reads the `starter.ini` beside it and resolves the same
folder. A `starter.ini` kept elsewhere (`starter -ini <path>`) is known only to
the starter and to the programs it starts.

The starter reads the mods folder again when its window regains focus, and on
**Rescan**. A mod published or copied there while the starter was open is
listed; the ticks, the order, the selected mod and the selected mission stay.

Tick a mod and the **Missions** list under the status line shows the missions the
mod declares in `data/missions.toml`: each numbered row that is not `off`, by
its number and its title in the starter's language. The starter reads the rows
as the game does and opens no map. A mod that is not ticked shows no missions
and says so; a mod with no missions file says it has none, on either game; a
missions file the game would refuse shows the refusal.

Select a mission and press **Start mission**: the game starts with the enabled
mods on that mission (`-mods <ids> -mission <n>`). **Play** starts the game with
the enabled mods and no mission of the list; the typed **mission** setting still
applies to Play. Play and Start mission write `starter.ini` first, so the
enabled list needs no **Save settings**. A start the starter refuses writes
nothing. The list holds three rows in view and scrolls with the wheel; while
it is shown the Check output keeps two lines. With no mod selected the window
is as it was, with six lines of Check output.

The script's `print` output is discarded, in `againrom -check` and in play.
A script reports a problem with `fail("text")`; the refusal names the mod,
the file and the line (`mod "m": main.star:3: fail: text`), and it stops the
check or the launch.

## Mission abandon

Both base games offer Abandon mission over an undecided mission entered from
a town. Confirming returns the party, money and items held at entry; it records
no victory or payment, and the mission stays offered. The first game's loaded
mission uses its current party as DIV-1907 states. A loaded second-game mission
has no retained entry town and shows no abandon entry.

`game.rules.abandon` is an integer rule, default `1` (on), with values `0` and
`1`. A mod that does not set it keeps the default. To remove the base action:

```python
def init(game, settings):
    game.rules.abandon = 0
```

A mod's `[[action]]` with `action = "abandon"` replaces the base action with its
own labels and confirmation; it does not add another abandon row. The switch
controls the base action, so an explicit mod action remains available even with
the switch off. Base actions take no mod menu slot. The `mission-abandon`
example remains the first-game fixture for this replacement.

## Install files

A mod replaces an install's picture, sound or text by carrying a file at the
same install path under its `files` folder. No script line is needed: the
game reads the mod's file wherever it reads the install's.

```
my-mod/
  mod.toml
  files/
    common/graphics/interface/town/townmain.bmp   every install
    common/sfx/<path of a sound>.wav               every install
    en/main/text/sites.txt                         English installs
    ru/main/text/sites.txt                         Russian installs
```

The install path is the archive name and the path inside it, as the
install's resource files hold it: `main/`, `graphics/`, `scenario/`,
`world/`, `movies/` or `sfx/`, then the path. Case does not matter.

A file replaces a file the install holds. One folder also takes new files:
`main/text/inn/npc/`, for the talk texts of the tavern rows a mod adds or
changes (see "Tavern rows"). A new file there is a text named as the game
names a talk text, `npc<NN>m<M>.txt`: the NPC record in at least two digits
and the mission, or the chapter for a talk with nothing to give
(`npc41m50.txt`). The terrain strips of a group the mod adds are new
files too (see "New terrain tiles"). A new path anywhere else is refused as
before.

Language folders are `common`, `en` and `ru`. On an install the game reads
the folder of the install's language (`en` on `rom1-en` and `rom2-en`, `ru`
on `rom1-ru` and `rom2-ru`) and `common`; for one path the language folder's
file wins over `common`'s. The other language's folder is held to the folder
rules below and not read; only the files the install reads are
checked against the install. Names that start with a dot are skipped in this folder, and no
mod reader takes a path with such a component (see "Packed mods").

| kind | extension | rule |
|---|---|---|
| picture | `.bmp` | the bit depth (8 or 24) and the size of the install's picture at that path |
| sound | `.wav` | 8-bit or 16-bit PCM, or 4-bit Microsoft ADPCM; mono or stereo |
| text | `.txt` | UTF-8; the game writes it into the install's bytes, so one file reads right on English and Russian. A character the install's alphabet has no byte for is refused |

Each file is at most 16 MiB. `againrom -check` and the launch refuse, naming
the mod and the file: a file outside a language folder, a folder that is not
a language, a file directly under a language folder, another extension, a
path the install does not hold, a picture of another depth or size, a sound
the game cannot decode, a text that is not UTF-8 or holds a character the
install cannot write. On success `-check` prints
`againrom: install files=<count> state-free=<mods> (<the rule>)`.

Two mods may serve one path only when one loads after the other (`requires`
or `load-after`). The later mod's file is read and the check prints
`againrom: override: install file <path>: mod "b" (<file>) overrides mod "a"
(<file>)`. Without an order the launch is refused naming both mods.

A mod that sets no rule, holds no state data file and serves no text file
(`.txt`) changes no game state: it is listed after `state-free=`. A text
feeds the definitions the game builds its state from (the body list, item
names, class and spell tables), so a mod that serves any text is marked even
when that text changes nothing. A picture or a sound is only drawn and
played. A state-free mod is not written into a save's mod mark, so a save
made with a mod of pictures and sounds loads without it. A mod that changes
game state is marked, and its save is refused without it. A save whose mark
names a mod loads while every mod it names is active, whether or not that
mod is state-free now.

A save made in town also loads when the active mods are the mods its mark
names, in the same order, and only their files, versions or settings changed.
A changed setting takes the same load as changed files. What the
active mods no longer hold is taken away on the load: an item they no longer
define leaves its holder's pack or worn slot, a tavern hire's included (the
slot stays empty), a skill above their cap falls to the cap, a clothing layer
they no longer define comes off, a waiting guest whose mission they no longer
declare is dropped (one whose mission moved takes its new number), and a
campaign mission they no longer declare leaves the town's lists. The Load
page lists each of these in the mod window, titled "The save was loaded under
changed mods", naming a removed item by its key and mod, and the town opens
when it closes; a quick load opens the same window. The next save carries the
active mods' mark. A save made inside a mission, another mod, another order
or another base is refused as below.

The mark names each mod item and clothing layer the save holds by its mod's
id and its key (`"mod"` and `"key"`). Item codes are handed out in load order,
so adding or removing an item moves the codes of the items after it; under
changed mods each held item keeps the item its key names, at whatever code
that item has now. A save written before marks named their items, under
changed mods, loads only if it holds no mod item or layer; otherwise the
window says it was written before saves stored the names of mod items.

A LOAD the mark refuses opens a window on the Load page of either game that
lists every difference, one row each: a mod the save used that is not active,
an active mod the save did not use, another version (both shown), the same
version with other files, a setting with another value (both shown), another
base game, another load order, a save without a mark while mods are active, and
a mark that cannot be read. The differing values are drawn in red. The window
says how to start the game to load the save (with the save's mods, or with
`-mods-accept-unmarked` for a save without a mark) and closes with OK or
Escape. A refusal outside the Load page, such as a headless run's, keeps
the one line of text that names the same differences.

Not served: `speech.res`, `music.res`, loose files beside the install's
resource files, and pictures in `.256`, `.16a`, `.16`, `.pal`, `.reg`,
`.alm` form.

## New terrain tiles

A mod adds terrain tile groups 13 to 31; the install's groups 0 to 12 stay as
they are. A group is one row of the mod's `data/terrain.toml` and four strips
under `files/common/`, one for each blend column, at the tile loader's path:
group g, column b is `graphics/terrain/tile<G>-<VV>.bmp` with G = g/4 + 1 and
VV = 4 * (g mod 4) + b in two digits.

```
snow-field/
  mod.toml
  main.star                                    game.data.add("data/terrain.toml")
  data/terrain.toml
  files/common/graphics/terrain/tile4-04.bmp   group 13, column 0
  files/common/graphics/terrain/tile4-05.bmp   group 13, column 1
  files/common/graphics/terrain/tile4-06.bmp   group 13, column 2
  files/common/graphics/terrain/tile4-07.bmp   group 13, column 3
  files/common/graphics/terrain/tile8-12.bmp   group 31, columns 0 to 3:
  files/common/graphics/terrain/tile8-13.bmp   tile8-12 to tile8-15
  files/common/graphics/terrain/tile8-14.bmp
  files/common/graphics/terrain/tile8-15.bmp
```

```toml
[[group]]
group = 13
kind = "grass"
name = "Snow"
minimap = "#e8f0f8"

[[group]]
group = 31
kind = "mountain"
name = "Ice"
```

| field | rule |
|---|---|
| `group` | 13..31, required; one row per group |
| `kind` | required: `land`, `grass`, `flowers`, `sand`, `cracked`, `stones`, `savanna`, `mountain`, `water` or `road`. A cell of the group takes that kind's cost, passability and block: `mountain` and `water` block (DIV-4413) |
| `name` | required, 1 to 32 bytes; the map editor's terrain list shows it |
| `minimap` | optional `#rrggbb`; without it the minimap uses the kind's colour |

A strip is an 8-bit BMP with its own colour table, which the game draws
with; it is 32 px wide and 1 to 14 cells of 32 px tall. Cell f from the top
is frame f of that column, and a cell's tile word names its frame. A strip
that replaces one of the install's groups 0 to 12 keeps the rule of "Install
files" instead.

`againrom -check` and the launch refuse, naming the mod and the file: a strip
outside the strip rule, a strip of a group no row of the same mod adds, a row
whose group lacks one of its four strips, a row outside 13..31, a second row
for a group, a row without `group`, `kind` or `name`, an unknown kind, and a
group that two mods add.

A mod that adds a group changes game state: a save made with it holds the
mod in its mark and is refused without it.

With the mod open in the map editor (`againrom --editor -mod <mod folder>` or the mod
chosen in the editor), the terrain list shows each group after the install's
terrains, by its `name` and with its first frame as the swatch. The brush
paints whole cells of the group. Its words are every frame of the group's
four strips, so a stroke mixes the four strips: each cell takes one word,
chosen by the brush's fill (the cell's position, a draw, or the one variant
chosen). It leaves its neighbours as they are: a mod group's cell takes no
corner blend, and Random terrain (Ctrl+E) leaves it unchanged. The saved map keeps the words and sets the group's bit in the
map's group mask (DIV-4412). The game refuses a map whose mask or cells name
a group no active mod adds, or a frame the group's strip lacks, naming the
group and the map. Publish packs the strips and `data/terrain.toml` with the
rest of the mod.

## Packed mods

A mod ships as its folder or as one file: the folder packed into a RES
file, the archive format of the game's own `.res` files, named
`mod-<id>.res` and put in the mods folder beside the mod folders. The game
reads the file as the folder it was packed from: the same `mod.toml`,
scripts, settings, data files, texts and `files` folder, under the same
checks, with the same refusals. `-mods <id>` finds either form, and
`againrom -check` reports a packed mod as its folder, with `dir=` naming the
file. A save's mod mark records the mod's files, not its form, so a save
made with the folder loads with the packed file and the reverse.

Pack a mod folder from a terminal:

```
againrom.exe -pack-mod C:\Mods\east-town -pack-out C:\Games\Againrom\mods
```

`-pack-out` names an existing folder; the game writes
`mod-<id>.res` there, prints `againrom: packed <folder> into <file>`, and
exits. It needs no install and opens no window. The fixed-tree files and old paths
named by rows are packed; unread files outside that description are left out.
Dotted names (`.git`) are ignored, which no mod reader takes: a script or data path with a dotted component is refused naming the path, for a
folder and for the packed file alike. Names match without case by one rule, for a folder and a pack alike
and on every host: ASCII `A`-`Z` and the Cyrillic letters of CP866 (`Ё`, `Є`, `Ї`, `Ў` too) of another case name the same file. The case of any other
letter does not fold: `éclair.star` does not name `Éclair.star`, in a folder on Windows as in a pack. A pack holding two names equal under the rule is refused naming both. The pack is refused, writing nothing, and the refusal names
the reason and, for a file, the file:

- the file exists; `-pack-replace` writes over it;
- the `-pack-out` folder is a game folder or lies inside one (a folder
  holding the game's archives);
- a file or folder component exceeds 255 UTF-8 bytes, is invalid UTF-8,
  or holds a control character; Cyrillic letters usually take two UTF-8 bytes;
- the root holds `mod-names.json`, a name reserved without case for the
  packed mod's name table, as either a file or a folder; folder mods refuse
  that name too;
- a link or another non-regular file, or a file larger than 16 MiB;
- the folder is not a mod `-check` would take: no `mod.toml`, or an `id`
  other than the folder's name.

In the mods folder:

- a folder and a packed file of one `id` are both refused, naming both;
  keep one of them;
- a file named `mod*.res` that is not `mod-<id>.res` with a valid `id`, a
  packed file whose `mod.toml` names another `id`, and a file the game
  cannot read as a RES are refused naming the file; other `.res` files are
  not mods and are left alone;
- names inside a packed mod and inside a folder match without regard to the
  case of `A`-`Z` and the Cyrillic letters of CP866 (`Ё`, `Є`, `Ї`, `Ў`
  too), and of no other letter, whatever the host file system folds.

### Name table

The RES container and the install's RES reader keep their format. A mod
component over 15 UTF-8 bytes, or one CP866 cannot encode, is stored under a
short name. Other components keep their names. The packer chooses short
names in tree order: `~` and a 14-digit lowercase hexadecimal counter,
starting at 1 and skipping a name already held by that folder, without case.

When at least one component needs a short name, the root RES file
`mod-names.json` holds UTF-8 JSON of this form:

```json
{"version":1,"entries":[{"path":"data/~00000000000001","name":"weapon-bodies.toml"}]}
```

Each entry names one renamed file or directory. `path` is its full path in
the RES, including short names of its parents; `name` is its original UTF-8
component. The table is at most 16 MiB. The packed tree reader restores
components before indexing and sorting the tree. Every mod reader sees the
original paths; the table itself is absent from that tree and its digest.
Duplicate entries, missing target nodes, invalid components, restored names
equal without case and an unsupported version are refused. A root entry
with the reserved name must be a valid table, never author content.

Without a table, an existing pack reads by its stored names as before.
When no component needs renaming, packing emits no table and keeps the
previous pack bytes. The digest and save mark always use the original
folder names and contents.

The map editor saves into a mod folder only. It opens a packed mod read-only
(see The map editor): every change and every save into it is refused, and the
packed file stays as it was. To change a packed mod, change its folder and
pack it again.

## Items

`data/items.toml` holds four kinds of table. `[[item]]` adds an armour or
shield item and `[[change]]` sets the price, weight, defence or absorption of
a shipped row (`docs/1272/story.md`). `[[material]]` allows or forbids one
material for one tier of a row. `[[weapon]]` adds a weapon row as a copy of a
shipped one. Row, tier and material names are those of the install's
`Data.bin`, which every first-game install spells in English.

### Materials on a row

Each armour, shield and weapon row holds one material mask per tier
(`DAT-MATMASK-020`). The shop's candidates, the item picture set and every
other reader of the mask read the mask after the mods.

| key | value |
|---|---|
| `item` | an armour, shield or weapon row's name, or the `key` of a weapon row this mod or one it loads after adds |
| `tier` | `Common`, `Uncommon`, `Rare`, `Very Rare` or `Elven` |
| `material` | a material's name: `Iron` to `Crystal`, or `None` |
| `allow` | `true` adds the pair to the row, `false` removes it |
| `sprite` | optional: an 80x80 PNG, the pair's inventory picture |
| `name` | optional: a text key, the pair's shown name |

An allowed pair the install has no inventory picture of needs `sprite`; without
it the mod is refused when it loads, naming the file, the line, the row, the
tier and the material. A forbidden pair takes no `sprite` and no `name` and
never rolls in a shop; LOAD tests no mask (`ITEM-224`), so an item of it a
save already holds still loads. The doll paints the figure layers the install
holds for the pair's code and paints none where the install has none.

```toml
# A common bone pike, with its own picture and name.
[[material]]
item     = "Pike"
tier     = "Common"
material = "Bone"
allow    = true
sprite   = "assets/items/bone-pike.png"
name     = "item.bone_pike"
```

### Weapon rows

A `[[weapon]]` table adds a weapon row. It starts as a copy of the shipped row
`base` names: every parameter, the material masks, the pictures of each pair
and the body it is drawn with are the base row's. The mod sets only the
parameters below; any other key is refused naming the file and the line. The
row takes the lowest free row of the weapon table (rows 28 to 31 on a shipped
install, `DAT-WEAPCELLS-023`); a fifth added row is refused.

| key | parameter | meaning | claim |
|---|---|---|---|
| `key` | | the row's key: 1 to 32 of `a`-`z`, `0`-`9` and `_`, starting with a letter | |
| `name` | | a text key, the shown name of every pair | |
| `base` | | the shipped weapon row it copies | |
| `price` | 2 | price of the common iron item | `SHOP-103`, `ITEM-WEAPCOL-021` |
| `weight` | 3 | weight | `ITEM-SCALE-017` |
| `damage-min`, `damage-max` | 6, 7 | physical damage | `ITEM-DMGCOL-018`, `ITEM-WEAPCOL-021` |
| `to-hit` | 8 | to-hit | `ITEM-SCALE-017` |
| `defence` | 9 | defence | `ITEM-SCALE-017` |
| `range` | 11 | reach in cells, 1 to 255, not scaled | `ITEM-221` |
| `sprite` | | optional: an 80x80 PNG, the inventory picture of every pair | |

Price, weight, damage, to-hit and defence scale by tier and material as the
shipped rows' do. The damage pair must stay in order and in a byte:
`damage-min` may not exceed `damage-max` (after the base row's value fills a
key you leave out), and for every tier and material the row's masks allow, the
scaled base and spread must stay within 0..255 (a pair whose factor is above 1
lowers the largest value you may write). A row that breaks either is refused
when the mod loads, naming the row, both values and the pair. Parameters 0, 1,
4, 10 and 16 have no known reader
(`ITEM-219`) and parameters 5, 12 to 15 choose the attack kind, the cadence,
the two-handed rule and who may use the weapon; all of them stay the base
row's. `[[material]]` tables name the new row by its `key`. A
`data/wpnbodies.toml` record names it by its `key` as `weapon` to draw it with
another body; the hero's class key stays the base row's.

```toml
# A war glaive: a Pike that hits harder and reaches three cells.
[[weapon]]
key        = "war_glaive"
name       = "item.war_glaive"
base       = "Pike"
damage-min = 40
damage-max = 80
range      = 3
price      = 900
```

### Two mods on one row

Items merge through the layer step of the merged kinds. A field is one
number of one row or one mask bit (row, tier, material). Two mods that set
one field, or add one `key`, need `load-after`: the later wins and
`againrom -check` names the override; without an order the launch is
refused naming both mods. Two changes of one field once both applied, the
later's value last, with no message; such a pair of mods now needs
`load-after` or is refused.

### Saves

A SAVE under a mod that adds or allows items names the mod set in the
`AgainromMods` leaf, and a LOAD without the mod refuses. A pair a mod allows
on a shipped row keeps its own code in the SAV: the code is valid and LOAD
tests no mask (`ITEM-224`). An item of an added weapon row is written as the
same pair of the base row (code and row) and the leaf lists the object with
its true code and row; LOAD under the same mod set writes them back. A save
whose leaf lists no such object, as every save written before this, holds
shipped items only.

## Spell damage columns

For the first game, `data/spells.toml` accepts `damage = { min = N, max = N }`
with each column in 0..6898, ordered and not both zero. The second game retains
its 0..255 column domain. Global damage and healing multipliers clamp their
columns to the corresponding game's bound. At power 255, 6898 scales to 65531,
inside the original random wrapper's non-overflowing spread domain.

For first-game rows without a `damage_factor` table, columns at or below
255 retain the original binary64 record bytes. If either column is above
255, both scaled products and their spread stay whole, with the same
binary64 rounding. A `damage_factor` table keeps its integer formula.
For a row with either column above 255, the popup states the whole pair and
fresh immediate, queued and area casts apply it. Byte-column factor-table
rows retain their existing delivery and popup routes. SAVE saturates an
in-flight base or spread above 255 to 255;
cold LOAD applies those saved bytes (`DIV-3002`, `DIV-2224`, `DIV-2327`).

## Weapon bodies

`data/wpnbodies.toml` holds `[[weapon]]` tables. Each draws one weapon row
with another body. The first name of this file, `data/weapon-bodies.toml`, is
also read in both a mod folder and a packed mod. A mod that holds both names
is refused, naming both files.

| key | value |
|---|---|
| `weapon` | the weapon table row's name, as the install's weapon table spells it (`War Hammer` on every install) |
| `row` | the weapon row number, 1 to 31 |
| `body` | a body the install ships under both `heroes` and `heroes_l`, or a body a loaded mod supplies |

A table gives `weapon`, `row` or both; both must name the same row. `body`
names the base body. A hero with a shield takes the `_` form of a chosen body when a `_` sheet
exists (under both directories for a shipped body, supplied by the mod for a
supplied one), else the body itself. A row the shipped body list leaves blank, two choices for one
row in one mod, and an unknown weapon, row or body are refused. Two mods that
choose a body for one row need `load-after`: the later choice is drawn and
`againrom -check` names the override; without an order the launch is refused
naming both mods.

The choice changes the picture only: the sheet, its geometry and the order the
doll paints the weapon and shield layers in. Sound, attack timing, casting,
shooting and everything a save or the world hash holds stay those of the
weapon's shipped body. A save holds the shipped body's name.

```toml
# The War Hammer is drawn with the mace body.
[[weapon]]
weapon = "War Hammer"
body   = "clubman"
```

## Body sheets

`data/bodies.toml` holds `[[body]]` tables. Each supplies a body under both
directories, one PNG per directory.

| key | value |
|---|---|
| `name` | the body's name, 1 to 32 of `a-z`, `0-9` and `_`, starting with a letter; a name the install ships is refused |
| `heroes`, `heroes_l` | the PNG paths inside the mod folder, for the plain and the light armour directory |
| `frame` | `[width, height]` of one frame, each 1 to 512 pixels |
| `origin` | `[x, y]` of the pixel that stands on the ground point, inside the frame (`x` below its width, `y` below its height) |
| `directions` | `8`, or `5` with directions 5 to 7 drawn mirrored |
| `move`, `attack` | `{ frames = n, ticks = t }`: 1 to 64 frames; `ticks` is one count for every frame or an array of one count per frame, each 1 to 255 |
| `idle` | optional, as `move`, with 0 to 64 frames |
| `weapon-last` | optional, `true` to paint the weapon layer after the shield layer on the doll |
| `selection` | optional `[x1, y1, x2, y2]`, the selection box inside the frame |
| `off` | `true` switches off a sheet an earlier mod supplies and stands alone with `name` |

The PNG is a grid of `frame`-sized cells. Row 0 holds the standing frames, one
per facing: 16, or 9 at 5 directions. Then, for `move`, `attack` and `idle` in
that order, one row per direction holds that action's frames from column 0.
A pixel with alpha 0 is background; every other pixel is opaque, and a sheet
is reduced to at most 256 colours. A cell outside the picture, or a cell with
no visible pixel, is refused, naming the action, direction and cell.

Two mods that supply one `name` need `load-after`: the later mod's sheet
replaces the whole sheet and `againrom -check` names the override; without an
order the launch is refused naming both mods.

A fallen hero is drawn with the shipped dying body of the directory. A
supplied body has no owner shading and no silhouette sheet. A shield form is a
second body named with the `_` suffix.

```toml
# A two-handed body: 8 directions, 16 standing frames on row 0, then one row
# per direction for each of move, attack and idle (25 rows of 64x80 cells).
[[body]]
name        = "hammer2h"
heroes      = "assets/bodies/hammer2h.png"
heroes_l    = "assets/bodies/hammer2h_l.png"
frame       = [64, 80]
origin      = [32, 70]
directions  = 8
move        = { frames = 6, ticks = 2 }
attack      = { frames = 5, ticks = [2, 2, 3, 3, 2] }
idle        = { frames = 2, ticks = 8 }
weapon-last = true
selection   = [16, 10, 48, 74]
```

A mod draws a weapon with its own body by loading both files and naming the
body in the choice:

```toml
[[weapon]]
weapon = "War Hammer"
body   = "hammer2h"
```

A mod that loads on both games lists `"rom1"`, `"rom2-en"` and `"rom2-ru"` in
`applies-to`. `-check` prints `againrom: weapon bodies=N body sheets=M`.

## Missions

`data/missions.toml` holds `[[mission]]` tables. A guest mission
(`kind = "guest"`, the default) adds one mission that a guest party plays
after a campaign mission is won and before the town that win opens, or after
another guest mission is won (see Chains). The party waits: it is kept as the
town SAV of that moment and comes back through the town LOAD when the guest
party wins the last mission of the chain. A replacing mission
(`kind = "replace"`) plays a campaign mission of the install on the mod's
map. A campaign row (`kind = "campaign"`) sets its successor or party and may
add a mission; see Campaign flow below.

| key | value |
|---|---|
| `kind` | optional; `"guest"` (default), `"replace"` or `"campaign"` |
| `key` | required; 1 to 32 of a-z, 0-9 and `_`, starting with a letter |
| `number` | required; the mission number the mission runs as, 1..65535; a `guest` row may not use a campaign number or one of the install's `scenario` container; a `replace` row must (see Replacing a campaign mission) |
| `map` | the map file inside the mod folder (at most 1 MiB); optional for a campaign row of an installed mission |
| `texts` | optional; a folder inside the mod whose `*.txt` files (at most 64) serve as the mission's `main/text/battle/m<number>/` files: `eventNN.txt`, `tipsNN.txt`, `briefmap.txt`, the world map's mission text, and `units.txt`, the names of the map's placed units. Write them in UTF-8; the game encodes each into the running install's bytes, so one folder reads right on the English and the Russian base. Typographic quotes, dashes, the ellipsis and the no-break space are written as their plain forms (`"`, `'`, `-`, `...`, a space); any other character the install has no byte for is dropped, and the map check names it when the editor saves into the mod. A file that is not valid UTF-8 is served byte for byte, as mods written before UTF-8 were. A short install-encoded Russian file whose high bytes happen to form valid UTF-8 (CP866 `e2 a0 ac`) is read as UTF-8; whole sentences are not |
| `title`, `briefing` | optional; text keys of the mod's `text/<lang>/strings.toml`, written into the mission's `title.txt` and `briefing.txt` |
| `after` | guest rows only, required; the mission whose win starts this mission: a campaign mission whose successor is a town, or a guest mission of the active mods |
| `guest` | the guest form: the campaign companion who plays it, as the town of `chapter` would grant him or her over the party |
| `chapter` | the guest form: the town the chain ends in, whose `AddHero` grants `guest` |
| `party` | instead of `guest`, `chapter` and `carry`: 1 to 8 members, an array of `{ character = "...", level = 1..4, weapon = "...", shield = "...", armour = ["...", ...] }` (see Guest parties) |
| `AutoGetMission` | campaign and replace rows only; the successor mission number; `-1` removes the automatic link |
| `Payment` | campaign and replace rows only; optional payment to the main party on completion, 0..1000000 |
| `gold` | optional, 0..1000000, default 0; the guest party's purse |
| `carry` | optional; only `"join"`: the guest joins in the town of `chapter` through its own grant, as he or she ended the chain |
| `class` | optional, guest missions only; `"fighter"` or `"mage"`: the mission plays only when the starting hero is of that class (see Missions by starting hero) |
| `sex` | optional, guest missions only; `"male"` or `"female"`: the mission plays only when the starting hero is of that sex |
| `off` | `true` switches off the mission of `number` another mod declares; the table holds only `number`, `off` and `kind` |

`{lang}` in `map` or `texts` becomes the language the game runs in (`en`,
`ru`). Missions merge by `number`: a mod that loads after another (its
`load-after` names that mod) replaces that mod's mission of the same number
as a whole, and the check names the override; without that order two mods
cannot declare one number. One mission follows one win for each starting
hero: two guest missions after one mission whose `class` and `sex` admit one
hero are refused at load, naming both files and lines. Nothing
but the state of the guest who joins carries back: the chain's purse, loot,
payment and every other member stay behind. A SAVE inside a guest mission
names the mod and the mission; a LOAD without that mod refuses.

A map's own pages take priority: `<stem>.text/<lang>/`, then
`<stem>.text/` beside its ALM. The declared `texts` folder supplies missing
pages; a folder ending in `/en` or `/ru` also reads page files in its parent
as a fallback. Other files in that parent are ignored. A replacing campaign mission finally keeps the install's
page for a number the map and mod leave out. Priority applies per file. A page
name that none of these sources supplies for the install's language is taken
from the same folders of the other language (`en` or `ru`), in the same order;
the install's own page of that name still beats it.

The map editor's Save As copies every language folder and unlanguaged page
beside the new ALM into `<stem>.text/`. Changed workspace pages go into
`<stem>.text/<lang>/`. Both destinations must be new and outside the install,
even when the source has no pages. Saving an editor-owned package keeps its recorded
unlanguaged and other-language pages. Unchanged unlanguaged pages stay
unlanguaged. A page that exists only in the other language's folder shows in
the editor and in the game, and Save As and W keep it in that folder: only a
page changed in the workspace is written under the install's language.
Page storage does not change ALM or SAV layout.

One place rule puts a mission's files into a mod. The map lies at
`maps/<stem>.alm`, or `maps/<stem>.almx` when it holds the extended record;
`<stem>` is the source map's file name without its extension, `m<N>` for a
numbered name, `map` for a new map, with `-2`, `-3` and on added when the
name is taken. The pages lie beside it in `maps/<stem>.text/` and
`maps/<stem>.text/<lang>/`, the editor's variable labels in
`maps/<stem>.variables`. The row names `map` and no `texts`. W's package
save, Add open map and F5 test play write by this rule; `cmd/modmission`
writes its per-language form, `maps/<lang>/m<N>.alm` with the pages in
`maps/<lang>/m<N>.text/<lang>/`.

A mod written in an older layout opens, checks, plays and packs as it
stands; `againrom -check` prints a warning for each old path. The older
layouts are `mission/map.alm` with `mission/text/`, Add's
`mission/map<N>/map.alm` with `mission/map<N>/text/`, and
`mission/{lang}/map.alm` with `mission/{lang}/text`. Its next save through W
moves each such row's map, labels and pages under the place rule and
rewrites the row's `map` and `texts` keys in place:

- The moved files keep their bytes and names. The pages of the old `texts`
  folder, and of its parent when the folder ends in a language, join the
  pages beside the map unless a page read before them holds the name.
- A per-language map keeps its language folder, `maps/{lang}/<stem>.alm`,
  and its `texts` key names the new pages folder.
- A map that holds the extended record but is named `.alm` is renamed `.almx`.
- A speaker record bound to a moved map names the map's new path.
- Old folders the move empties are removed. Every other file and key stays as
  written. Each language reads the same map and pages before and after.
- A row whose new file a held file already takes, and rows that share a map
  but not a `texts` folder, stay where they are.

A mod already in the tree, opened and saved without an edit, keeps its bytes.
F5 writes the moved layout only into its private copy. The move changes the
mod's content digest, as any edit does.

```toml
[[mission]]
key      = "prologue"
number   = 25
map      = "maps/{lang}/prologue.alm"
title    = "prologue.title"
briefing = "prologue.briefing"
after    = 20
guest    = 22
chapter  = 30
```

### Campaign flow

A campaign row takes `key` and `number`. Without `map` it uses the install's
map. With `map` it replaces that map, or adds a side-numbered mission when
the install declares no such number. An added number must not be a multiple
of ten, must not occupy an installed map and must be reached by an automatic
link. It has no payment unless `Payment` sets one.

`AutoGetMission` uses the scenario registry's spelling. Winning the source
opens that mission without a town between. The target must be an installed
main mission above the current main mission, or a mission the mod adds
above it. Installed side missions are refused, even with their own row;
supporting them as automatic targets is open debt. The load checks the whole chain and the
retention of installed side missions; cycles, missing targets and backward
requests refuse launch with the file and line. The install's links remain
where no row overrides them. Removing a link (`AutoGetMission = -1`) runs
the normal completion and returns to the town with the next main mission
available. An added mission without a link does the same.

For example, the main party plays 10, then the added 11, then the install's
20. Winning 20 takes the install's normal route:

```toml
[[mission]]
kind = "campaign"
key = "first"
number = 10
AutoGetMission = 11

[[mission]]
kind = "campaign"
key = "detour"
number = 11
map = "maps/detour.alm"
texts = "maps/detour.text/{lang}"
AutoGetMission = 20
party = "main"
Payment = 75
```

Campaign and replace rows default to the main party; `party = "main"` states
that explicitly. Either may instead set `guest` with `chapter`, or the same
`party` array a guest row takes. `gold` is that guest party's purse. A guest
plays while the main party waits in a SAV. Only the joining guest's state
comes back. Up to 64 guests may await their separate town grants; entering
another guest mission at that limit is refused. Other members, purse and loot
stay behind. The campaign win then
completes for the waiting main party, including its payment, town, documents
and mercenary unlocks. The main party in an added mission keeps its loot,
experience and inventory through the ordinary campaign carry.
A joining guest's chapter must be the town reached after the mission and
its automatic successors, and that town must grant the companion. A chapter
already passed, or a chain ending in another town, refuses the mod with its
file and line.

Campaign rows merge as whole missions by number, under the same `load-after`
rule as replacements. Combine a map, flow and party in one row of a mod.
The existing numbered-map form still replaces maps; use a row that names that
file to add flow or party data. `againrom -check` and the starter check print
one line for every campaign row, including payment-only and party-only rows.
For example: `mission 10 AutoGetMission = 11, party = main by mod "m",
declared at data/missions.toml:5` and `mission 20 Payment = 999, party = main
by mod "m", declared at data/missions.toml:1`.

SAVE and LOAD inside these missions use SAV and the same mod mark as other
mod state. An added mission occupies a selected child record beside the
current main record. Its successor is written through the campaign's
`autoGetMissionValue`. Guest state uses the existing guest record; the
internal waiting-party SAV may precede the first town. Existing `after`
chains keep their saved representation. No SAVX extension is needed for
these fields. Original-runtime compatibility of a side mission in an
automatic chain is Unknown (`DIV-4854`, `DIV-4856`).
The early town after removing the prologue link is also authored behaviour
(`DIV-4857`).

The map editor does not author these flow and party fields yet. Edit the
missions file directly. The second game's bank-driven campaign routing is
also outside this implementation: the shared scenario value receives the
override, but the bank does not route by it.

### Chains

A guest mission whose `after` is another guest mission plays when that one
is won, with no town between, as the campaign's `AutoGetMission` opens the
next mission after missions 10 and 20 (REG-SCN-063, SAV-1329). The missions
make one chain of any length; the first plays after a campaign mission, and
the chain ends in the town that campaign mission's win opens, where the
waiting party comes back when the last mission is won (DIV-4001). A member
a mission's party names again comes as he or she ended the previous mission;
the others are built anew. At most one mission of a chain uses the guest
form, and its guest is the one who joins. A cycle (`after` naming a later
link of the same chain) or a missing link (`after` naming neither a campaign
mission nor a guest mission) is refused at load with the file and the line.
A SAVE in any mission of the chain keeps the mod, the mission, the party the
mission began with and the waiting party; a LOAD continues the chain.

```toml
[[mission]]
key     = "first"
number  = 25
map     = "maps/first.alm"
after   = 20
guest   = 22
chapter = 30

[[mission]]
key    = "second"
number = 26
map    = "maps/second.alm"
after  = 25
gold   = 100
party  = [{ character = "npc22" }, { character = "sellsword", level = 2, weapon = "Iron Short Sword", armour = ["Hard Leather Mail"] }]
```

### Missions by starting hero

`class` and `sex` choose a guest mission by the starting hero the player
made: a fighter or a mage, male or female. A mission that names neither
plays for every hero; one that names one plays for the two heroes of that
class or sex; one that names both plays for one hero. Several guest missions
may follow one win when no hero is admitted by two of them; each has its own
`number`, map and texts. When the win comes, the hero's own mission opens.
A hero no mission admits goes on to the town as if no mod mission followed
the win. Two missions that admit one hero are refused at load with both
files and lines and the hero they share, for example "a mission already
plays after mission 20 for a female fighter at line 1". A later link of a
chain is chosen by the starting hero of the waiting party, not by the guest.
A SAVE inside any of the missions names that mission, so a LOAD plays the
same map; the town SAV after its win keeps that mission's guest. A replacing
mission plays for every hero and takes neither key.

```toml
[[mission]]
key     = "fighter"
number  = 25
map     = "maps/fighter.alm"
after   = 20
guest   = 22
chapter = 30
class   = "fighter"

[[mission]]
key     = "mage_male"
number  = 27
map     = "maps/mage_male.alm"
after   = 20
guest   = 22
chapter = 30
class   = "mage"
sex     = "male"

[[mission]]
key     = "mage_female"
number  = 28
map     = "maps/mage_female.alm"
after   = 20
guest   = 22
chapter = 30
class   = "mage"
sex     = "female"
```

### Guest parties

A party member names a character of the one character catalog: `npc<n>` is
record `n` of the install's `npc.reg`, and any other key is a `[[character]]`
a mod adds in `data/characters.toml`. The first member leads the party. Every
member uses the same party builder as a town companion (DIV-4002).

Every installed catalog record can be named, including `npc1`, `npc31` and
`npc100`. A record without `DataBinID` has no installed party template. It
uses a stated default (DIV-4003): the `PC_Danath`, `PC_Naira`, `PC_Fergard`
or `PC_Reniesta` family selected by its `Female` and `Mage` flags, with an
absent flag meaning male or fighter. Its level picks the family tier (the
bare name for level 1, `_2` to `_4` for later levels), by default the band
of the chain's town. Statistics, skills and equipment come
from that row; the declaration's equipment replaces its cells. A human
figure keeps the record's face; a flat portrait uses the row's human face
and body. Its name is the record's `Name`, else its installed NPC name,
else its catalog key. This default does not create a playable creature.

- `level` 1 to 4 picks the row of the level's band. A composed `npc.reg`
  record takes the row of the band of mission `(level - 1) * 40`
  (HERO-JOIN-120); a mod character whose row is a family takes
  `<family>_<level>`. Without `level`, a composed record takes the row the
  chain's town grants and a family takes the band of that town's chapter.
  An `npc.reg` record with one row, or a mod character with one row that is
  no tier of a family, takes no `level`.
- A member's name follows his record. A composed record (`npc22` to `npc24`
  of the first game) is the archetype row its flags choose for the starting
  hero's class and sex, and plays under that archetype's name: beside a male
  fighter `npc22` is Reniesta, beside a female fighter Fergard. A record
  that is one definition row is one person and plays under the record's own
  name, its `Name`, else its installed NPC name: `npc25` is Brian and
  `npc26` Rood Glaen beside every hero, not the archetype of their class
  and sex.
- `weapon` and `shield` replace the row's weapon and shield cell; `armour`
  replaces all eight armour cells, those it does not name left empty. An
  empty string empties the cell. A name no item of the cell's kind has is
  refused at load.
- `hero = true` in place of `character` is the player's own hero as he
  stands: the hero character generation made, or the hero the link before
  brought. The hero takes no `level`, `weapon`, `shield` or `armour`, and a
  party names him once. A mission after another in a chain names the hero
  only when the mission before it does.
- `skills = 100` opens the mission with every skill of the member at the
  rules' skill cap, with the experience the school asks for that level. No
  other value is taken. A cap below 100 gives the cap.
- `spells = "all"` opens the mission with every spell the member can learn
  in his book: each spell goes through the game's one read of a spell book,
  so a fighter, or a spell no book of the game teaches, gains nothing. No
  other value is taken.

```toml
party = [
    { hero = true, skills = 100, spells = "all" },
    { character = "npc22", spells = "all" },
    { character = "npc23", level = 2 },
]
```

A mod's own mission with a `party` array also starts by its number
(`-mission <n>`) with no won mission before it: character generation makes
the hero, the array builds the party, and its win ends as a campaign
mission's does. A mission that names `class` or `sex` starts this way only
for a hero of that kind; for another hero the start is refused in a line
that names both.

A `[[character]]` table with `key` adds a character to the catalog. Its keys
are `key` (1 to 32 of a-z, 0-9 and `_`, starting with a letter; not `npc<n>`), `row` (a definition
row, or a family of tier rows), `name` (a text key of the mod) and `face`.
`off = true`, alone with `key`, switches a mod's character off. Characters
merge by `key` the way missions merge by `number`. A `[[character]]` table
with `target` edits definition rows as before. Every operation applies in
load order, including family and individual-row edits that overlap. No
`load-after` is required for target edits: a later `kind` replaces the
earlier face with its own, then its explicit `face` applies if present.

```toml
[[character]]
key  = "sellsword"
row  = "NPC06"
name = "character.sellsword"
```

### Replacing a campaign mission

A `kind = "replace"` table takes `key`, `number`, `map` and the optional
texts, flow, payment and party fields above; `after`, `class` and `sex` are
refused. When the campaign reaches mission
`number` by ordinary play, the mission is built from the mod's map; the files
of `texts` and the `title` and `briefing` replace the installed mission's files
of those names, and the others stay the install's. Everything else stays the
install's: the scenario rows, the world map, the payment and the town that
follows, unless the row overrides them. The rules:

- `number` must be a mission of the install's campaign whose map the install
  holds; another number is refused. The replaced mission is built and proven
  on the first game; a mod names the bases it applies to in `applies-to`.
- The map must decode as a map; a map the mission build refuses is refused
  when the mod loads, naming the file and line.
- The map editor's rules judge the map of every mission, guest and replacing
  (`DIV-3256`). A rule at error level refuses the mod when it loads, naming the
  file, the line, the rule and its message. A rule at warning level refuses
  nothing; its finding prints as a `warning:` line on standard error, which the
  starter's check shows. The rule for message texts reads the mission's own
  `texts` folder.
- A later mod may replace a whole mission row when `load-after` orders the
  two mods. Without that order a second use of one number is refused. A guest
  mission never uses a campaign number, so the two kinds cannot collide; a
  guest after a replaced mission plays after its win.
- A SAVE inside a replaced mission carries the mod mark, whose content digest
  covers the map: a LOAD without the mod, or under another mod set, refuses.
  A save of that mission without a mod mark is refused while the mod is
  active, even with `-mods-accept-unmarked`: its world was built from another
  map.

```toml
[[mission]]
kind   = "replace"
key    = "ambush"
number = 20
map    = "maps/{lang}/ambush.alm"
title  = "ambush.title"
```

A map file named by its mission number replaces that mission with no table
and no script line: a mod of `mod.toml` and `10.alm` replaces the map of
campaign mission 10. For the game it is the replacement a `kind = "replace"`
table makes, with only `number` and `map`: the same build, the same map
rules, the same mod mark, the same rule of one mission number per mod and the
same order between two mods. The mission's title, briefing and texts stay the
install's; a page folder beside the map (`10.text/`) is read as for any map.
The rules:

1. Only the mod's top folder and its `maps/` folder are read, and only a name
   that is a mission number with no leading zero followed by `.alm` or
   `.almx`, in any case (`10.alm`, `maps/20.ALMX`; not `010.alm`).
2. A file that a table of `data/missions.toml` names as its `map` is that
   table's map and nothing else: a `replace` table of mission 10 with
   `map = "10.alm"` or `map = "maps/10.alm"` is one replacement and takes the
   table's `texts`, `title` and `briefing`; a guest mission whose map is
   `maps/10.alm` stays a guest mission and replaces nothing.
3. A `replace` table of mission 10 whose `map` is another file, while an
   unnamed `10.alm` lies in the top folder or `maps/`, is refused when the mod
   loads, naming both files. So are two files of one number: one in the top
   folder and one in `maps/`, or `.alm` and `.almx`.
4. A numbered file whose number is no campaign mission whose map the install
   holds replaces nothing and refuses nothing; the launch and `againrom -check`
   print one `warning:` line naming the file. The mod still enters a save's
   mod mark.
5. One file serves the English and the Russian install. A mod that needs a map
   per language uses a `replace` table with `{lang}` in `map`.
6. `againrom -check` prints one line per replaced mission, of either form:
   `againrom: mission 10 replaced by mod "m": map 10.alm, named by its file
   name`, or `..., declared at data/missions.toml:4` for a table.

The repository holds no map. `cmd/modmission -install <root> -map <n> -out
<mod folder>` copies an installed map to `maps/<lang>/m<n>.alm` of a mod
folder and its event, tip, title, briefing and briefmap texts beside it into
`maps/<lang>/m<n>.text/<lang>/`, the texts as UTF-8, and writes nothing else;
a row `map = "maps/{lang}/m<n>.alm"` names them. When the map editor saves into a mod, its map report warns
about a message the map's script sends for which neither the mod's `texts`
folder nor, for a replacing mission, the install holds an `eventNN.txt`.

## Extended map scripts (.almx)

A first-game map whose script needs more than the original trigger record
holds is an `.almx` file: an ALM file with one more record, the extended
script record, of typeId 100 (DIV-4599). The original game's loader steps
over every record of typeId 10 or above, and the second game reads typeIds
10 to 12, so 100 stays clear of both. An `.almx` lists in the game's map list
beside the `.alm` files, and a mission's `map` in `data/missions.toml` may
name one (`map = "maps/{lang}/ambush.almx"`). W saves an extended map into a
mod as `maps/<stem>.almx`; F5 plays it, Publish packs it, and the packed mod
reads it as its folder does. A mod mission's map that carries
the record but is named `.alm` loads and plays with the record, and the
map report warns that the original game does not run its triggers under that
name (DIV-4603). A loose map in
the game's map list is read by its bytes, not its name: a loose `.alm` that
carries the record lists and plays as an `.almx`, and the original game would
run its nodes with no trigger. No Againrom writer makes such a file. A mission
plays on its map as always; the extended record only changes how its script
runs.

What the extended machine holds (DIV-4601):

| limit | original record | extended record |
|---|---|---|
| variables | 100, shared with the check results | 0..4096, an array of their own |
| triggers | 1000 latches | 10000 |
| conditions per trigger | 3 pairs, all must hold | 64, in groups |
| actions per trigger | 4 | 64 |
| condition and action ids | | 1..5000 |
| comparison codes | 0..5 | 0..5 |

A trigger's conditions are numbered into groups. It fires when any group
holds (OR), and a group holds when all of its conditions hold (AND)
(DIV-4600). The first condition is in group 0 and each next one is in its
predecessor's group or the one after it. A trigger with no condition fires on
every pass. Triggers run in map order on the original's pass: once every 16
ticks, every check first, then every trigger. A fire-once trigger fires once
for the whole mission.

The variables are their own array (DIV-4602). Check 19 reads variable p0;
actions 3 and 8 set and raise one. A check's result is its own slot, read
only by the conditions that name the check's id, so a variable never takes a
check's result. Variable 93 counts the passes, as register 93 does in an
original map, when the map has at least 94 variables. A condition id that
names no condition node reads 0; an action id that names no action node, or
the drop location, runs nothing. A subscript past the map's variable count
reads 0 and writes nothing.

The type-7 record keeps the action and condition nodes as in any ALM, and
its trigger count is 0: the triggers live in the extended record alone. A map
carrying the extended record and a type-7 trigger, two extended records, or a
value past a limit above is refused, and the message names the trigger and
the entry.

### The record's bytes

Every word is a little-endian unsigned 32-bit integer. Append the record
after the file's last record and add 1 to the record count at file header
+0x0c. The record header is the usual 20 bytes: 7, 20, the payload size, 100,
and 0.

| offset | payload field |
|---|---|
| +0x00 | version, 1 |
| +0x04 | variable count, 0..4096 |
| +0x08 | trigger count, 0..10000 |
| +0x0c | the triggers, back to back |

| offset in a trigger | field |
|---|---|
| +0x00 | name, 64 bytes, Windows-1251, padded with zero bytes |
| +0x40 | fire-once word: 1 fires once, 0 on every pass it holds |
| +0x44 | condition count C, 0..64 |
| +0x48 | action count A, 0..64 |
| +0x4c | C conditions, 16 bytes each: group, left condition id, right condition id, comparison code |
| +0x4c + 16C | A action ids, 4 bytes each |

The payload ends after the last trigger's last action id. The comparison
codes are 0 `==`, 1 `!=`, 2 `>`, 3 `<`, 4 `>=`, 5 `<=`, applied as
left `op` right. `almtool script <file>` prints the record's head and each
trigger's conditions with their groups.

### Texts inside the map

An `.almx` may also hold the map's text pages in one more record, the texts
record, of typeId 101. It holds the pages a text folder beside the map holds,
under the same names: `name.txt` for a page of every language and
`<lang>/name.txt` for one language's page. A mission reads the record's pages
first, then the text folder beside the map, then the mission's declared
`texts` folder: a page the record holds wins, and a page only a folder holds
is still read. For a name the language lacks, the record's pages of the other
language fill in before the folders' do. A map that holds the record needs no
text folder, also when its mission declares one. A packed mod reads the record
as its folder does. A map with the record, in a mod or loose, stays within the
mod reader's 1 MiB per file: the game reads no page from a larger one, and the
map editor refuses to save one.

The map editor's File menu has the switch "Texts inside the map", off by
default, and the in-game save window shows the same switch. Switching it marks
the map changed. Saved with it on, the pages go into the record and no text
folder is written; a folder already beside the map is left as it is, and it
still supplies a page the record lacks. The record is the switch's state: a map
that holds it opens with the switch on. Saved with the switch off, the map and
its folder are written as before, and the pages a record held go into the
folder. A map whose file holds the record is saved with its pages or not at
all: when the switch is off and the pages cannot be read or the folder cannot
be written, the save is refused and neither the map nor the folder changes. In
a mod's mission list the same holds for a map that is not the one shown: its
draft keeps the record when the record cannot be made, and with the switch off
the record's pages go into the map's folder with it.
Switching it on for an original map asks whether to make the map extended, as
an edit the original record cannot hold does; Yes makes it extended and the
map saves as `.almx`. An `.alm` never holds the record: with the switch on,
Save As to an `.alm` name is refused, and Save over an own `.alm` writes
`<name>.almx` beside it. A mod mission's map that holds the record under an
`.alm` name gets the report warning of the script record. The mod package
save (W) writes the pages into the record and, as before, as files.

The record header is 7, 20, the payload size, 101 and 0; the record is
appended after the script record and the record count rises by 1.

| offset | payload field |
|---|---|
| +0x00 | version, 1 |
| +0x04 | page count N, 0..1024 |
| +0x08 | N pages, back to back, in byte order of their names |

| offset in a page | field |
|---|---|
| +0x00 | name length L, 1..128 |
| +0x04 | name, L bytes of UTF-8 with `/`, one or two parts, no `\`, `:`, control character, empty, `.` or `..` part |
| +0x04 + L | text length T, 0..1048576 |
| +0x08 + L | text, T bytes, as the page file holds them |

The payload ends after the last page's text. A name out of order or twice,
a byte after the last page, or two texts records is refused.

### Saves of an extended map

A SAV of an extended map holds the first 100 variables and the first 1000
trigger latches in the original session block, and the program with the rest
of its state, every variable past 99, every check result and every latch past
999, in the Againrom supplement (DIV-4604). LOAD resumes the same mission
state.

### The map editor

The map editor opens `.alm` and `.almx` maps. An extended map's trigger
window shows its conditions in OR groups: Add condition pair adds a condition
to the last group, Add OR group starts a group, and Remove on a condition's
comparison row or on an action's row takes it out. A trigger holds up to 64 conditions and 64
actions, and the variable pickers offer variables 0..4095.

To make an extended map, open an `.alm` and make an edit the original record
cannot hold: a fourth condition pair, an OR group, a fifth action, a variable
at 100 or above, or a trigger past 1000. The editor asks once, in EN or RU,
whether to make the map extended; it then saves as `.almx` and the original
game will not open it (DIV-4615). Cancel changes nothing. Yes moves every
trigger into the extended record as one history entry, applies the edit, and
asks nothing more for that opened map.

An extended map saves as `.almx`. Save over the map's own `.alm` writes
`<name>.almx` beside it, and the `.alm` stays as it was. When the script fits
the original records again (one group per trigger, at most 3 pairs and 4
actions, variables 0..99 free of check results, at most 1000 triggers),
Save or Save As to an `.alm` name asks whether to save it as an original map;
Yes writes the triggers back into 184-byte records, Cancel writes nothing.
Save As to an `.alm` name of a map that does not fit is refused (DIV-4616).
An original map with no extended edit saves byte for byte as before.

The mod window (W) opens any mod of the first game, whatever kinds it holds:
a mod folder, or a packed `mod-<id>.res` read-only, chosen in W's folder field
or with `againrom --editor -mod <folder or file>`. The editor builds the mod's
data the way the game applies it, so its terrain groups are on the terrain
list, its characters and speakers in W's pickers, and its items in the party's
equipment. A mod with an enabled mission opens that mission's map, as before.
Any other mod, and every packed mod, opens over the map in the editor and
shows W's Overview page first: the kinds whose files the mod holds, the lines
`againrom -check` prints for it, its files, the overrides and warnings its data
gave, and what refuses it. The Missions and Overview tabs at the top of W, or
W again, turn the page. The fields page also edits the manifest's version,
author, description, target base, requirements, conflicts and load order hints.
The target is a choice of `common`, `rom1`, `rom1-en`, `rom1-ru`, `rom2-en`
and `rom2-ru`. Dependency fields take comma-separated IDs; a requirement may
include a version constraint. Invalid values show the manifest reader's error.
ID stays fixed after opening. Edits share the project draft and Undo/Redo;
Ctrl+S writes them. API is written as 1 and is not a field. A packed mod refuses every change in W and every save
into it, naming the packed file; opening another mod is allowed. New packages contain no generated entry script. An opened packed file stays read-only.

## Towns

`data/towns.toml` holds `[[town]]` tables. Each is a layer record over the
first game's town catalog, keyed by `id`, in the catalog's entry format
(`docs/1485/story.md`). The install's catalog is layer 0; each mod's file is
applied in load order. A table adds a town when no earlier layer holds its
`id`, changes only the fields it names when one does, or switches the town
off with `off = true`. The second game's catalog takes no `[[town]]` or
`[[tavern]]` layer; its inn rows take one (see "Inn rows").

| key | value |
|---|---|
| `id` | required; the town ID, 1 or more; one table per ID in a file |
| `description` | an installed description (`rom1`), or a `.json` file inside the mod read by the game's one town description decoder |
| `hooks` | `"chapters"`; any other value is refused |
| `offers` | `"chapter-keys"`; any other value is refused |
| `map-point` | one source: `{ object = n }` (an object of the world map), `{ record = true }` (the map description's record), or `{ point = [x, y], rect = [x, y, w, h] }`, a point and its click rect stated in world-map pixels |
| `name` | the town's title on the map and on the home scroll, in UTF-8; the game draws it in the install's font, and a character the font has no glyph for is refused |
| `opens-after` | the mission whose win opens the town; 0 is open from the start |
| `missions` | the missions the town's buildings offer |
| `cite` | the sources the record rests on, a list of strings |
| `off` | `true` switches the town off and stands alone in its table; `false` with other keys switches a town an earlier layer switched off back on |

A new town names `description`, `hooks`, `offers` and `map-point`. The merged
catalog passes the same checks as a shipped one, after each mod: the entry
format, the description and its rooms, the campaign's opening missions, one
town per mission, and a stated point and rect inside the picture of every
world-map mode.

Every town must be able to open. The towns with `opens-after = 0` are open
from the start, and they must offer at least one mission. A town opens when a
town already open offers its `opens-after` mission. A town that lists
`missions` offers those; a town that lists none offers every mission no town
lists. A mission no building of the campaign offers (missions 10 and 20 of
the stock campaign) is the campaign's own and counts as open from the start.
A town that lists the mission it opens after, or whose mission only such a
town offers, never opens and is refused.

A description file of the mod names its pictures by art key. A key that
starts with `mod:` reads a file inside the mod folder (`mod:assets/towns/east.bmp`);
every other key reads the install. A picture is a 24-bit BMP and a mask an
8-bit BMP, as in the install. A run-time-bound art entry cannot name a mod
file.

```toml
# Town 1 keeps everything but its name.
[[town]]
id   = 1
name = "West"

[[town]]
id          = 2
description = "assets/towns/east.json"
hooks       = "chapters"
offers      = "chapter-keys"
map-point   = { point = [430, 230], rect = [410, 205, 40, 30] }
name        = "East"
opens-after = 20
missions    = [31]
```

### Tavern rows

The same file holds `[[tavern]]` tables: the talks and quests of a town's
tavern. A row is one NPC record speaking in one chapter, and the mission it
offers. Layer 0 is the scenario's: each `[Mission<n>]` section's `InnNPC`
and `InnMission` pairs (REG-SCN-064). A row of a town the catalog holds as
layer 0 stands in the first town that offers its mission. A table adds a
row when no layer holds its key, changes the fields it names when one does,
or switches the row off with `off = true`.

| key | value |
|---|---|
| `town` | part of the key; the town whose tavern shows the row |
| `chapter` | part of the key; the main mission whose chapter the row belongs to (30, 40, ...) |
| `npc` | part of the key; the NPC record that speaks; a chapter names a record once over every town |
| `mission` | the mission the talk offers; 0 is a talk with nothing to give (REG-SCN-064). A heard mission is queued and registered on leaving the tavern (TAVERN-023, TAVERN-024) |
| `off` | `true` switches the row off and stands alone in its table |

No other key is read; `text`, a speaker name or a picture is refused as an
unknown key. The talk speaks the install's text at
`main/text/inn/npc/npc<NN>m<M>.txt`, with the mission, or the chapter for a
talk with nothing to give, as M. A row a mod adds or changes is refused at
launch, naming the mod, the file, the line and the row, when:

- its chapter is not a main mission of the campaign;
- its mission is not a mission section of the campaign, or its town does not
  offer the mission (see the rule above: a town that lists `missions`
  offers those, a town that lists none offers every other mission);
- its NPC record is not in the install's NPC table;
- its talk text is in neither the install nor a loaded mod. A mod carries a
  new talk text as an install file (see "Install files").

A row switched off is refused at launch when a companion join of a loaded
mod (`data/companions.toml`) waits for that talk in that chapter; the
refusal names the towns mod, its file, line and row, and the companions
mod with its file and line. The companions kind applies first, so the
towns layer sees every join.

Two mods that set one row need `load-after` or `requires`, as for towns; the
check prints each override. A row a mod adds shows in its own town's
tavern; a row of layer 0 shows in the town that offers its mission. A
quest accepted from a mod row is a mission record as any other, and a save
carries its talk and quest as the original's offer arrays do: the mod is
named in the save's mod mark, and a LOAD without it is refused.

A save keeps the current chapter's offers as they were. A LOAD adds no row,
so a mod row of the loaded chapter shows from the next chapter or a new game.
This holds for the tavern's talk cells too: a save made without the mod
shows the same talkers with it as without it.

```toml
# The talk of NPC record 41 joins chapter 50's tavern of town 1.
[[tavern]]
town    = 1
chapter = 50
npc     = 41
mission = 0

# NPC record 32 of chapter 50 offers mission 31 instead of 51.
[[tavern]]
town    = 1
chapter = 50
npc     = 32
mission = 31

# NPC record 25's talk of chapter 50 is switched off.
[[tavern]]
town    = 1
chapter = 50
npc     = 25
off     = true
```

The mod carries `files/en/main/text/inn/npc/npc41m50.txt` and
`files/en/main/text/inn/npc/npc32m31.txt` (and the `ru` ones for a Russian
install), each in the part form of the install's talk texts
(`<part=1 npc=41>` and the words).

### Inn rows

On the second game the same file holds `[[inn]]` tables: the option rows of
the towns' inns. A `[[tavern]]` row and an `[[inn]]` row are one row type,
read by one parser and judged by one validator; the tavern row is its reduced
shape, one chapter of one town. A catalog names the table its rows take
(`offer-table`), and a table of the other game is refused. Layer 0 is the
catalog's own rows, `pkg/game/towncatalogs/rom2.json` key `offers` under
`"offer-table": "inn"`: the stage bodies of the inn's
entry (stages 10, 30, 40, 50, the body stages 60, 70 and 80 share, 90, 100
and 110), the town 3 talks of NPC records 5 and 675 after stage 60, and the
town 2 mission talks of NPC records 2022 and 22 (R2-ENGINE-161,
R2-ENGINE-215 to R2-ENGINE-228). The inn shows, in row order, every row the
party's town, the stage (bank slot 768) and the bank admit, then the hire
actors of the hire bank, which no row states. The four-way mission talks of
NPC record 2022 (topics 74 to 77, 84 to 87 and 93 to 96) and topics 81 and
82 of layer 0 also name the starting hero they show for, `class` and `sex`,
read in the bank slots the catalog's `hero-slots` names (776 nonzero for a
mage, 781 for a female hero, R2-ENGINE-217, R2-ENGINE-424); the same chooser
picks a guest mission by `class` and `sex` (see Missions by starting hero).
A mod's table does not set `class` or `sex`. A row of the table that names
neither `zero` nor `nonzero` keeps a layer 0 row's `class` and `sex`. The
hero slots hold a fighter or male hero in `zero` and a mage or female hero in
`nonzero`, so a row that names `zero`, `nonzero` or both replaces that part
of the hero condition with the list it names: the part in a list it names
goes from `class` and `sex`, and the part in a list it does not name stays
there. A table adds a row when no
layer holds its key, changes the fields it names when one does, or switches
the row off with `off = true`. A row a mod adds comes after the rows of
layer 0.

| key | value |
|---|---|
| `topic` | part of the key; the talk's topic, and for kind 3 the mission it makes available |
| `npc` | part of the key; the NPC record that speaks, from 1 |
| `kind` | 0, a talk; 3, a talk that makes mission `topic` available. A row that does not set it is a talk |
| `town` | the town whose inn shows the row; 0 or absent shows it in every town |
| `stages` | the stages the row shows at, each from 1; absent shows it at every stage |
| `after` | from 1, the row shows only at a stage above it |
| `zero` | bank slots (0 to 1023) that must hold 0 |
| `nonzero` | bank slots that must hold a value other than 0 |
| `one` | bank slots that must hold 1 |
| `any` | bank slots of which at least one must hold a value other than 0 |
| `off` | `true` switches the row off and stands alone in its table |

A list a table sets replaces the row's list. The talk speaks the section
`npc<npc>talk<topic>` of the install's town text, `main/text/town.txt`, as a
row of layer 0 does. A merged row is refused at launch, naming the mod, the
file, the line and the row, when its kind is not 0 or 3, its town is not a
town of the catalog, a stage is not positive or a slot is outside the bank.
A row a mod adds or changes is also refused, as a tavern row is, when:

- its kind is 3 and `topic` is not a mission record of the world map;
- its NPC record is not in the install's NPC table;
- its talk section is in neither the install's town text nor a town text a
  loaded mod carries (`files/<language>/main/text/town.txt`, see "Install
  files"). The English and the Russian town texts do not hold the same
  sections: `npc520talk7` and `npc2006talk110` are in the English text
  alone (the rom2-en and rom2-ru installs, `main.res`).

The second game loads these rows with no scenario of the first game's kind:
its install holds no `scenario/scenario.reg`. The first game refuses
`[[inn]]` tables; its tavern talks are `[[tavern]]` rows.

```toml
# Town 2's inn at stage 30 also hears NPC record 520 on topic 7. The
# English town text holds that section; on a Russian install the mod also
# carries a town text that holds it.
[[inn]]
topic  = 7
npc    = 520
kind   = 0
town   = 2
stages = [30]

# NPC record 2108's stage 30 talk shows only while bank slot 927 holds 0
# and slot 533 does not.
[[inn]]
topic   = 31
npc     = 2108
nonzero = [533]

# NPC record 2110's stage 30 talk is switched off.
[[inn]]
topic = 39
npc   = 2110
off   = true
```

### Town description format

A description is one JSON object, read by the game's one town description
decoder. A field the format does not name is refused with the file and the
reason.

| field | what it holds |
|---|---|
| `town` | the description's name, a string |
| `view` | `{ "size": [w, h] }`, the picture size in pixels |
| `clock` | `{ "period-ms": n, "compare": "greater" }`; `compare` is `greater` or `at-least` |
| `random` | named random sources: `name`, `source` (`host`, or a generator with `multiplier`, `increment`, `shift`, `mask`) |
| `art` | named pictures: `name`, `format`, `key`, and for a series `first` and `count` |
| `mask` | `art` (a mask picture) and `bytes`: each `{ "byte": n, "hotspot": name }` maps a mask byte to a hotspot |
| `hotspots` | `name`, `row` (the row of the keyboard order), `tip` or `no-tip`, `click` and `hover` lists of operations |
| `pointer`, `sounds`, `actors`, `step` | the pointer shapes, sound slots and triggers, animated actors and the per-tick steps of a scene |
| `layers` | the order the scene is drawn in: `art` or `actor`, `mode`, `at`, `clip`, `when` |
| `music` | `track`, or `tracks` with `by` naming the value that chooses |
| `square` | `name` and `enter`, the steps that put the party on the square, and an optional `pane`, as a room's |
| `rooms` | the rooms reached from the square, below |
| `save` | `admitted-when`, a condition |

Art `format` is one of `picture`, `series`, `sprites`, `frame`, `mask`. A
layer `mode` is `copy` or `over`. A key that starts with `mod:` reads a file of
the mod; any other key must be an entry of the install, and a key the install
does not hold is refused at `-check`. A `music` track and a sound slot `key`
are likewise checked against the install's music and sounds.

A click operation is an object with one of `room` (go to the named room),
`hook`, `menu`, `dialogue`, or `if` with `then` and `else` lists. A room has
`name`, `page`, `enter` and `exit` step lists, an optional `scene` (the same
fields as the top level: `view`, `random`, `art`, `sounds`, `clocks`, `actors`,
`layers`), `art`, `music`, `seats`, `buttons` and `pane`. A step is
`{ "hook": name }` or `{ "composer": "reset" }`.

A button named `sleep` is the town's Sleep, which restocks the shop; its word
is `tavern.sleep`. The first game's tavern always takes the shipped first
town's buttons, whose first is Sleep; a `tavern` room's own `buttons` do not
change them.

A room's optional `pane` is the character pane its page shows: `rect`, the
pane's rectangle `[x0, y0, x1, y1]`; `book`, `true` when the pane has the
spellbook corner; and `font`, where `"install"` draws the pane's arrows and
mode box in the game's own font and leaving it out draws them in the character
generator's font. The pane's drawing, its doll and item drop, its picker
arrows, its mode corner and its book corner all stand on `rect`. A shop pane
with `"book": false` has no spellbook: no corner, the B key does nothing and
the spell strip does not open. In a `chapters` town a `tavern`, `shop` or
`school` room without `pane` takes the same room's pane of the shipped first
town: `[480, 238, 640, 480]`, the book in the shop only, the game's own font.

`page` is `tavern`, `shop`, `school` or `world-map`. Hooks are
`gate-closed`, `release-audio`, `set-room`, `talk-clear`, `tip`,
`school-reset`, `shop-shelf`, `shop-interior`, `faces`, `offer-on-entry`,
`tavern-interior`, `tavern-detail-clear`, `tavern-selection`, `navigate`,
`inn-queue-commit`, `trade-cancel`, `dialogue-reset`, `tavern-leave`,
`shop-reset` and `school-leave`. The conditions are `gate-open` and
`save-admitted`; the values are `shop-chosen`, `member-class` and
`school-hover`; the events are `rack`, `merchant-yes`, `merchant-no`,
`train` and `class-change`. An unknown name is refused with this list.

A town with `hooks = "chapters"` needs the rooms `tavern`, `shop`, `school`
and `gates`, and the check refuses a description the game cannot play:

- the square's `enter` and every room's `enter` run the hook `set-room`, or
  the party never stands on the square or in the room;
- the `gates` room's `enter` runs the hook `navigate`, or the gates never
  open the world map;
- the `tavern`, `shop` and `school` rooms each have a `scene`, because their
  pages are drawn from it;
- a `pane` of those rooms has `rect` `[480, 238, 640, 480]`, because the
  pages' own controls fill the rest of the screen; only the `shop` pane may
  have `"book": true`, and no pane has a `book_offset`;
- a `pane` of the square has the same `rect`, no `"book": true` and no
  `book_offset`;
- the `school` scene has the actors `column` (program `target`), `training`
  (`training`), `diamond` (`bounce`) and `shine` (`cycle`).

The smallest working description follows. It draws the square from
`art/square.bmp` (24-bit BMP), the rooms from `art/room.bmp`, the hotspot mask
from `art/mask.bmp` (8-bit BMP, using the bytes 128, 144, 160 and 176) and the
school actors from `art/frame0000.bmp` and `art/frame0001.bmp`. The square,
room and mask pictures are the full 640 by 480 view; the frames may be any size.

```json
{
  "town": "east",
  "view": { "size": [640, 480] },
  "clock": { "period-ms": 100, "compare": "greater" },
  "art": [
    { "name": "square", "format": "picture", "key": "mod:art/square.bmp" },
    { "name": "mask", "format": "mask", "key": "mod:art/mask.bmp" }
  ],
  "layers": [{ "art": "square", "mode": "copy" }],
  "mask": {
    "art": "mask",
    "bytes": [
      { "byte": 128, "hotspot": "tavern" },
      { "byte": 144, "hotspot": "shop" },
      { "byte": 160, "hotspot": "school" },
      { "byte": 176, "hotspot": "gates" }
    ]
  },
  "hotspots": [
    { "name": "tavern", "row": 0, "no-tip": true, "click": [{ "room": "tavern" }] },
    { "name": "shop", "row": 1, "no-tip": true, "click": [{ "room": "shop" }] },
    { "name": "school", "row": 2, "no-tip": true, "click": [{ "room": "school" }] },
    { "name": "gates", "row": 3, "no-tip": true, "click": [{ "room": "gates" }] }
  ],
  "square": { "name": "square", "enter": [{ "hook": "set-room" }] },
  "rooms": [
    {
      "name": "tavern", "page": "tavern",
      "enter": [{ "hook": "set-room" }],
      "scene": {
        "view": { "size": [640, 480] },
        "art": [{ "name": "inside", "format": "picture", "key": "mod:art/room.bmp" }],
        "layers": [{ "art": "inside", "mode": "copy" }]
      }
    },
    {
      "name": "shop", "page": "shop",
      "enter": [{ "hook": "set-room" }],
      "scene": {
        "view": { "size": [640, 480] },
        "art": [{ "name": "inside", "format": "picture", "key": "mod:art/room.bmp" }],
        "layers": [{ "art": "inside", "mode": "copy" }]
      }
    },
    {
      "name": "school", "page": "school",
      "enter": [{ "hook": "set-room" }],
      "scene": {
        "view": { "size": [640, 480] },
        "random": [{ "name": "training", "source": "host" }],
        "art": [
          { "name": "inside", "format": "picture", "key": "mod:art/room.bmp" },
          { "name": "column", "format": "series", "key": "mod:art/frame%04d.bmp", "first": 0, "count": 2 },
          { "name": "diamond", "format": "series", "key": "mod:art/frame%04d.bmp", "first": 0, "count": 2 },
          { "name": "fighter-tr", "format": "series", "key": "mod:art/frame%04d.bmp", "first": 0, "count": 1 },
          { "name": "fighter-m", "format": "series", "key": "mod:art/frame%04d.bmp", "first": 0, "count": 1 },
          { "name": "mage-tr", "format": "series", "key": "mod:art/frame%04d.bmp", "first": 0, "count": 1 },
          { "name": "mage-m", "format": "series", "key": "mod:art/frame%04d.bmp", "first": 0, "count": 1 }
        ],
        "sounds": { "slots": [{ "name": "rotate", "source": "school-training", "key": "town/school/rotate.wav", "untracked": true }] },
        "clocks": [{ "name": "gate", "period-ms": 83, "compare": "greater", "rebase": "every" }],
        "actors": [
          { "name": "column", "program": "target", "art": "column", "frames": 2, "endpoints": [0, 1],
            "value": "member-class", "wait": { "base-ms": 83, "compare": "greater" } },
          { "name": "training", "program": "training", "clock": "gate", "column": "column", "slot": "rotate",
            "value": "member-class", "delay": { "draw": "training", "raw": 32768, "divide": 10 }, "delay-ms": 3000,
            "hold-pick": { "draw": "training", "raw": 32768, "form": "scaled", "n": 20, "base": 20 },
            "variants": [
              { "name": "fighter", "transition": "fighter-tr", "transition-count": 1, "idle": "fighter-m", "idle-count": 1, "column-at": 0 },
              { "name": "mage", "transition": "mage-tr", "transition-count": 1, "idle": "mage-m", "idle-count": 1, "column-at": 0 }
            ] },
          { "name": "diamond", "program": "bounce", "art": "diamond", "frames": 2 },
          { "name": "shine", "program": "cycle", "frames": 1, "wait": { "base-ms": 300, "compare": "greater" } }
        ],
        "layers": [{ "art": "inside", "mode": "copy" }]
      }
    },
    {
      "name": "gates", "page": "world-map",
      "enter": [{ "hook": "set-room" }, { "hook": "navigate" }]
    }
  ]
}
```

Two mods that set one field of one town, or add the same new `id`: the later
mod wins only when it loads after the earlier one through `load-after` or
`requires`, directly or through another mod, and `againrom -check` prints
an `override:` line naming the town, the field and both mods. Otherwise the
launch is refused naming both mods and the field. Two mods that set
different fields of one town need no order, except a town another mod adds:
a mod that changes that town must load after the mod that adds it.

Refused, naming the mod, the file and the line:

- switching off town 1, the campaign's starting town, or switching off a town
  no layer holds;
- an unknown key, a wrong type, a table that changes nothing, `off = true`
  with other keys, and one `id` twice in a file;
- a description file the mod does not hold, a description that is not shipped,
  a `mod:` picture the mod does not carry, a picture key, music track or sound
  key the install does not hold, and a description the game cannot play (see
  Town description format);
- `hooks` other than `"chapters"`, `offers` other than `"chapter-keys"`, and
  a `name` with a character the install's font has no glyph for;
- a merged catalog that fails a catalog check, a town that can never open,
  towns open from the start that offer no mission, and a stated point or
  rect outside the world map.

A save written with a towns mod carries the mod mark. A LOAD without the mod
refuses with the mod named.

## Speakers

`data/speakers.toml` holds a mod's own speaker records. The game reads the file beside `mod.toml`;
the entry script names it nowhere. A message tag `<npc=N>` names a record by its `id`.

```toml
[[speaker]]
id = 20000                     # 20000 to 99999; the numbers below are the game's records
name = "Karl"                  # shown in the editor's speaker lists
picture = "assets/speakers/karl.png"  # a PNG of 72x96 pixels, the message window's face
map = "maps/en/ambush.alm"     # by choice, with unit: the map's path in the mod
unit = 12                      # the unit's script number on that map, 1 to 65535
```

A record with `map` and `unit` is the speaker the map editor names for that unit. `unit` is the
number a script names the unit by, not its place in the map's unit list, so the record stays with
the unit when other units are added, moved or deleted. A number held by more than one unit of the
map names none of them. In the game a message that names the record's number draws its picture. A
save holds that number in the map's message text and no other field. In the map editor, on a map
that lives in a mod, clicking a unit no speaker record shows with the speaker pick asks for a name
and opens the file window for a PNG. The record and picture at `assets/speakers/<id>.png` enter the
project draft together, as one undo step. The speaker is available at once; Ctrl+S writes the
record and picture with the project. No file is written by the add. A unit with script number 0, or one another unit shares, is refused with a message that
says so. A map outside a mod keeps the refusal text. While the mod's records did not load (a bad
picture, a failing script) the editor adds no record and states why.

The highest number is 99999: a message tag in the second game is read for five digits after `npc=`.
The editor gives a new record a number the mod's id decides: a stable hash of the id into the band
20000 to 99999 is the first number tried, and the search walks up from there, round to 20000 at the
top, past every number held by this mod's file, by the records the editor has loaded, and by the
`data/speakers.toml` of every other mod in the same mods folder. So two mods made apart do not get
one number by accident. A mod that names a number by hand keeps the merge rule above: the number is
one record and `load-after` decides which mod's record stands.

## Base dump

The base dump writes what the installed game holds, in the format a mod
reads, so a mod starts from the game's own values instead of guessed ones.
Run the game from a terminal with `-dump-base` and a folder outside the
game folder:

```
againrom.exe -assets C:\Games\Rage -dump-base C:\Mods\base-dump
```

Without `-assets` the game uses the install it finds in the working folder
or beside `againrom.exe`. The game opens no window: it writes the folder,
prints one line per file, and exits.

The folder is a mod. Its name is the mod's `id`, so it may hold only `a-z`,
`0-9`, `.`, `_` and `-`. It holds:

| file | what it holds |
|---|---|
| `mod.toml` | the manifest: `id` (the folder's name), a title, version `1.0.0`, `applies-to` naming the install it was taken from; no `entry` key is written |
| `data/towns.toml` | every town of the install as a `[[town]]` table with every key of the Towns section: `id`, `description`, `hooks`, `offers`, `map-point`, `name`, `opens-after`, `missions`, `cite`, and `off = false`; on the first game, then every tavern row of the scenario as a `[[tavern]]` table: `town`, `chapter`, `npc`, `mission` and `off = false` |

Values are the game's own. `description` names the installed description
(`rom1`) and `map-point` the world-map object (`{ object = 1 }`); no picture
or text is copied out of the install. `name` is the title the world map
draws for the town, written in UTF-8, so the Russian game's dump holds it in
Cyrillic letters. Loaded unchanged, the dump changes nothing a player sees
or a save holds, besides the mod mark every active mod puts in a save.

To make a mod from the dump:

1. Copy the folder and give the copy the mod's name. Set `id` in `mod.toml`
   to that name and change `title`.
2. In `data/towns.toml`, keep the `[[town]]` tables you change and delete
   the rest. In a table you keep, keep `id` and the keys you change and
   delete the others: each key a table names counts as set by your mod, and
   two mods that set one key of one town need `load-after`.
   Keep a `[[tavern]]` table you change with `town`, `chapter` and `npc`.
3. Change the values, and add tables for new towns as the Towns section
   shows.
4. Run `againrom.exe -check -mods <name> -mods-dir <folder holding it>`; a
   refusal names the file and the line.

The second game reads no towns layer of `[[town]]` tables. Its dump still
lists its three towns in `data/towns.toml`, as comment lines for reading, and
says so in the file and in the printed lines; the dump loads as a mod that
changes nothing. A second-game
town has no `name` key: the second game draws no name from its catalog. The
dump writes no `[[inn]]` table; a second-game mod that changes an inn row
writes its own `data/towns.toml` holding only `[[inn]]` tables (see "Inn
rows"); the held file loads without a script.

The dump refuses, writes nothing and names the reason when the folder:

- is the game folder, lies inside it, or lies inside any folder holding a
  game's archives;
- exists and is not empty;
- is an existing file;
- has a name that is not a mod `id`.

## Headless runs

A headless run drives the game from a scenario file, opens no window, plays
no sound, and prints what the game shows. It runs a mod's town through the
game's own screens and writes the frame the game draws, so a mod can be
checked on a machine with no display.

```
againrom.exe -assets C:\Games\Rage -mods east-town -mods-dir C:\Mods --headless scenario.json
```

`-assets`, `-mods`, `-mods-dir` and `-mods-accept-unmarked` mean what they mean
for a normal launch. Without `-assets` the game uses the install it finds in
the working folder or beside `againrom.exe`. The scenario file is read first:
a scenario the game refuses stops the run with exit code 2 before any install
is read. A step that fails stops the run with exit code 1 and the step's
number and command. A run that ends exits 0.

The output is split. The standard output holds one JSON line per step, in
order. The error output holds one trace line per step and the lines the game
itself prints when it loads, and begins with
`againrom: headless saves in <folder>` when the run has a saves folder.

### The scenario file

One JSON object. A key the format does not name is refused with the keys the
format accepts at that place. A path that is not absolute is read relative to
the folder of the scenario file.

| key | value |
|---|---|
| `version` | required; 1 to 10 in this build. A file keeps the commands of its version, so write 10 |
| `stage` | `frontend` (default) or `mission` |
| `assets` | `install` (default) or `synthetic`; `synthetic` is for the mission stage and reads no install |
| `window` | `{ "w": 1280, "h": 960 }`, the window's size in pixels; front-end stage, version 4 and later |
| `original_saves` | a folder; front-end stage only |
| `saves` | a folder; front-end stage only |
| `steps` | required; a list of steps, run in order |
| `mission`, `difficulty`, `mage`, `world` | the mission stage's own inputs, below |

The front-end stage drives the game as a player does: the main menu, the LOAD
list, the town, the world map, the generation screen, the game menu. It is
the stage a mod's town is checked on. The mission stage plays one campaign
mission's simulation with no screen above it: `mission` is the campaign
mission number (required with `assets` `install`), `difficulty` is `easy`,
`normal` or `hard`, `mage` is `true` when the hero opens as a caster, and
`world` is the authored world of an `assets` `synthetic` scenario. `-mods` does
not apply to the mission stage, and `original_saves`, `saves` and `window`
are refused there.

### Saves

The `saves` key is the folder the run saves into and lists first in the LOAD
list. Without it the run uses the player profile's saves folder as the game
does, the `Againrom` folder beside the program when the program sits in the
install folder, and the run prints that folder on its first error-output line.
Name a folder of your own so that a test run writes nothing beside the program.

The `original_saves` key is the folder of original-format saves the LOAD list
shows after yours; the run only reads it. Without it that folder is the install
folder, so the install's own saves are listed. The `-saves` option sets the
saves folder too; the `saves` key wins when both are given.

A save row reads `<name> - between missions` for a save made in a town and
`<name> - mission <n>` for one made in a mission. The `load` command names a
row by that text or by the start of it (below).

### Steps

A step is one object with a `command` and the keys that command takes. A key
a command does not take is refused. Commands of the front-end stage:

| command | keys | what it does |
|---|---|---|
| `load` | `target` | opens the LOAD list and loads the row `target` names; refused when the game does not open |
| `activate` | `target` | presses the control `target` names on the screen shown; `select_control` is the same |
| `open_menu` | `target` | `game` opens the game menu from the map or a town; `load` opens the LOAD list |
| `key` | `key` | presses a key |
| `pointer` | `action`, `at` | one edge of a mouse gesture at a place the game's own hit test finds |
| `select_member` | `id` | selects a party member on the map by the `id` a capture shows |
| `wait_ticks` | `ticks` | steps the map screen that many ticks; the map screen only |
| `wait_until` | `until`, `ticks` | steps the game until `until` holds or `ticks` steps pass (default 40000) |
| `create_character` | `character` | sets the generation screen's character |
| `save`, `save_open`, `save_edit`, `save_action`, `save_select` | `directory`, `name`, `target` | the SAVE dialog: `save` saves as `scenario-save-<time>` in one step; the others drive each step of the dialog |
| `abort_game` | none | ABORT GAME from the game menu, back to the main menu |
| `capture` | `name` | records the state under `name`, once per name, and prints it with the town (below) |
| `frame` | `path` | writes the frame the game shows as a PNG (below) |
| `assert_state`, `assert_member`, `assert_inventory`, `assert_shop` | `state`, `member`, `inventory`, `shop` | fail the run when the state differs from what the key names |

Commands of the mission stage, which the refusal of an unknown command also
lists: `order`, `kill`, `kill_player`, `place`, `pick_item`, `heal`,
`assert_unit`, `assert_world`, `report`, `wait_ticks` and `wait_until`. A
refusal of an unknown key inside one of them lists its keys.

### Forms of each argument

`target` of `load` and `activate` on a list (the LOAD list, the game menu, the
picker, a town's rows): the text of the row, matched without regard to case,
or the start of it when the next character is a space (`GATES` for
`GATES     1 mission(s) available`); `@first` is the first row that can be
chosen. A row that cannot be chosen, or two rows that match, are refused with
the rows on the screen.

`target` of `activate` elsewhere:

| screen | forms |
|---|---|
| main menu | the name of a button; the refusal lists them |
| a town's square | the row text, as above: `TAVERN`, `SHOP`, `SCHOOL`, `GATES` |
| a town's tavern, shop and school | the label of a visible cell or button; the refusal lists them |
| the world map (after `GATES`) | `mission N` or `town N`, the number the card shows; `walk out to mission N` and `travelling to mission N` mean `mission N`. The party travels and the step ends when it arrives |
| a town's conversation | `dialogue` or `notice` closes it |
| the mission screen's outcome panel | `victory`, `continue`, `exit to main menu`, `load game`, or `notice` |

`target` of `open_menu`: `game` or `load`.

`key` takes a name from this list, not case-sensitive: `enter`, `numpad-enter`,
`escape`, `esc`, `load`, `l`, `f1` to `f9`, `f11`, `f12`, `ctrl-shift-f4`, `ctrl-f5`
to `ctrl-f8`, `help`, `readout`, `fps`, `alt-enter`, `alt-backspace`,
`alt-f12`, `numpad-plus`, `numpad-minus`, `ctrl-numpad-plus`,
`ctrl-numpad-minus`, `c`, `cast`, `pageup`, `pagedown`, `up`, `down`, `left`,
`right`, `shift-left`, `shift-right`, `shift-home`, `shift-end`, `space`, `0`,
`numpad-0`, `ctrl-n`, `ctrl-o`, `ctrl-u`, `ctrl-h`, `ctrl-l`, `ctrl-f`,
`ctrl-w`, `ctrl-a`, `a`, `attack`, `r`, `retreat`, `d`, `defend`, `p`,
`patrol`, `f`, `grab`, `e`, `select-all`, `tab`, `shift-tab`, `backspace`,
`delete`, `home`, `end`, `autocast`, `w`, `weapon-swap`, `zoom-reset`,
`speed-reset`, `book`, `b`, `q`, `inventory`, `i`, `backquote`, `` ` ``,
`doll`, `j`, and `alt-` followed by a letter `a` to `z`. The refusal of
another name lists them. In a mission, `readout` (`f11`) and `ctrl-shift-f4`
act only while the cheats are unlocked.

`action` of `pointer`: `hover`, `press`, `move`, `release`, `right-press`,
`right-move`, `right-release`, `middle-press`, `middle-move`,
`middle-release`, `wheel-up` and `wheel-down`, each with an optional `shift-`
prefix and then an optional `ctrl-` prefix (the map reads `ctrl-`; other
screens ignore it). The scenario runner and the game take this one list.

`at` of `pointer` names one place with exactly one of these keys, and may add
`offset_x` and `offset_y`, pixels added to the place it found. Naming none or
two is refused with this list.

| key | place |
|---|---|
| `spell` | the spellbook cell of this spell number (1 to 65535) |
| `doll_slot` | slot 1 to 12 of the map screen's doll |
| `doll_box` | the doll box, on no slot |
| `pack_cell` | the pack bar's cell, counted from 0 |
| `pack_code` | the pack bar's cell holding this item code |
| `ground` | a place on the map no box claims |
| `shop` | `{ "surface": s, "index": n, "code": c }`; `s` is `doll`, `doll_box`, `shelf`, `pack`, `table`, `picker_prev`, `picker_next`, `shelf_pick`, `button` or `merchant`; `index` counts from 0 (from 1 for `doll`) and `code` names a `shelf`, `pack` or `table` cell by its item |
| `shop_idle` | a place on the shop's frame that is no control |
| `world_map_mission` | the world-map card of this mission number |
| `world_map_town` | the world map's return-to-town card |
| `world_map_miss` | a place on the world map that is no card and no region |
| `document` | `left`, `right` or `ok` of the documents panel |
| `tip_close` | the Close button of the tip panel a town opens with |

`until` of `wait_until` names exactly one of `screen` (`menu`, `picker`,
`map`, `chargen`, `town`, `gamemenu`, `load`, `documents`, `ending`),
`notice` (`true` or `false`: a notice is open over the map) or `control` (a
row text, as for `target`).

`character` of `create_character` takes `difficulty`, `name`, `sex`, `class`,
`skill` (the labels the generation screen shows) and `stats` (a statistic's
name and the value to buy it to); it names at least one.

`save_edit` takes `name` (the file name), `directory` (the folder) and `target`
(the format, `SAV`). `save_action` takes `save`, `overwrite`, `confirm`,
`delete`, `back`, `cancel`, `up`, `open` or `select-all` in its `target`;
`save_select` takes the row index, as a string, in its `target`.

`state` of `assert_state` takes any of `screen`, `mission`, `notice`,
`town_place`, `purse`, `documents`, `collected_documents`, `member_count`,
`difficulty`, `quick_spells`, `current_spell`, `spell_armed`, `members` and
`document`; a key it does not name is not checked. `town_place` is `square`
or `world_map`.

### What capture prints

Each step prints one line: `step`, `command`, and `state`, the game's state
after the step: `screen` (`menu`, `town`, `map`, `chargen`, `gamemenu`,
`load` and the others above), `town_place` on a town screen (`square`,
`world_map`), `purse`, `members` (each with `id`, `name`, `xp`, skills,
equipment, load and capacity) and the open dialog or panel, if any.

A `capture` step also prints `name`, and from version 10, on a town screen and
on the game menu opened over a town, `town`:

```
"town":{"id":2,"name":"East","art":["mod:art/east-square.bmp","graphics/interface/town/town_add.bmp","graphics/interface/town/shop_l.bmp","graphics/interface/town/tavern_l.bmp","graphics/interface/town/trener_l.bmp"]}
```

`id` is the ID of the town the party stands in, `name` is that town's name as
the game draws it, in UTF-8 (the install's own town word when the town has no
`name`), and `art` lists the keys of every picture the square can draw, conditional ones
included (a label drawn only while its building is hovered, a picture drawn
only while an actor is active), the square's own picture first: an install
key, or `mod:` and the path inside the mod folder. A version 9 or earlier file
prints no `town`, and the second game prints none at any version.

### The frame

A `frame` step, version 10 and later, writes the game's current frame as a PNG
to `path`, which must end in `.png`, and prints `frame`: the full `path`, the
`width` and the `height` in pixels. The frame is the one the game's headless
renderer composes for that state: a town, the world map, the main menu, the
LOAD list, the generation screen. The mission screen and the mission picker are
drawn straight to the window and have no frame; the step is refused on both.
The game menu over a town is drawn with its authored panel; the error output
carries a note only when the install has no authored panel to draw.

The folders of `path` are made. The step is refused, and nothing is written,
when `path` is inside the install folder, inside a preserved install beside
it, or inside any folder that holds a game's archives; when a file or folder
already stands at `path`; when two `frame` steps name one path; or when it
does not end in `.png`. The paths are judged before the first step runs, so a
refused scenario saves nothing either.

The first visit to a town opens a welcome tip over its square. To see the
square's picture, close it with a `pointer` `press` and `release` at
`tip_close`, as the example does.

### Example: a mod's town

The scenario below goes to the second town of the `east-town` mod and writes
its square and the world map seen from it. It needs a save made in the first
town after the mission that opens the second was won, named `east test` in
the game's SAVE dialog and kept in a folder `saves` beside the scenario; the
mod's `opens-after` decides which mission that is. The save holds the mod's
mark, so run it with the same `-mods`.

```json
{
  "version": 10,
  "saves": "saves",
  "steps": [
    { "command": "load", "target": "east test" },
    { "command": "activate", "target": "GATES" },
    { "command": "activate", "target": "town 2" },
    { "command": "capture", "name": "east" },
    { "command": "pointer", "action": "press", "at": { "tip_close": true } },
    { "command": "pointer", "action": "release", "at": { "tip_close": true } },
    { "command": "frame", "path": "east-square.png" },
    { "command": "activate", "target": "GATES" },
    { "command": "frame", "path": "east-map.png" }
  ]
}
```

The steps load the save, press `GATES`, travel to town 2, capture (the
`town` it prints names town 2, its name and the mod's picture), close the
welcome tip, write `east-square.png`, return to the world map, and write
`east-map.png`, in which the name of the town the party stands in is on the
card at the top right. A mod town that does not open, a name the font cannot
draw or a picture that is not found shows in the capture or the frame, or the
run stops at the step that failed.

## Map editor

The Traps and spells palette and tree branch list traps, unit spells and casting
structures. Each has a named mark on the map. A trap joins its trigger cell to
its source and target cells. Click its trigger cell to open the field popup;
drag it to move only that cell. Unit spells and casting structures follow the
unit or structure they name and cannot be dragged away from it.

Trap area, in Tools and that palette, lays a trap on every cell of a dragged
rectangle, with one spell and power. Source and target start on each trap's own
cell. Select any member and press Delete, or Remove trap set in the palette, to
remove that stroke together. One Undo restores the entire creation or removal.
The set keeps its identity through field edits, shifted record indices and
Undo/Redo. Sets last only in the open editing session: reopening the map leaves
independent traps. The file contains only the original per-cell records
(DIV-4484). Setting one trap's spell to None retains the existing single-entry
removal behaviour.

An untouched map saves byte for byte. Edits confined to cell spells also keep
every unrelated byte; other combinations of edits retain the existing save-form
normalisation (DIV-4485).

Cell spells appear in the record list as traps, unit spell entries or casting structures. F3 opens the shared Add window. Its first choice is New trap, New unit spell, New casting structure, or an existing cell spell to copy; a new record works on every map, including one with no cell spell yet. Then give a unit or structure id, the cell, spell (1..28) and power; a trap also asks its source and target kind, a casting structure its scan range. A new record holds what the original editor writes: every field zero except the cell, the kind, the id, spell, power and scan range. A new trap's source and target both start at its cell, and the third word of each is 0 (the original leaves an unset value there that the game never reads, DIV-4045); edit them in the popup. A copy keeps every other byte of its template. Unit spells and casting structures find their unit or structure by id alone, as the game does; their cell need not be the object's anchor cell, so most shipped casting structures, which sit on another cell of their building, edit and copy like any other. A unit spell or casting structure whose id names no object, for example after its structure's script target id changed, is listed as unbound: the game skips it, its fields are read-only, and Delete or spell 0 removes it. Add takes any cell of the map for a unit spell or casting structure; the original editor writes one only on a cell of its object, and what the game does with that cell is not established (DIV-4048). Unit and casting-structure ids and cells are read-only after placement. Add updates a casting structure already recorded at that cell with that id and opens its popup; another cell of the same structure gets a second record, as in the original editor; unit spell entries keep appending. E edits the known fields: trap and unit spell power 0..100, casting structure power and scan range 0..255. A larger value is refused with a message (DIV-4046): the game keeps only the low byte, so 256 would act as 0. The scan range is the field the original editor's dialog labels Scan Range; that name as its meaning in the game is Medium confidence, and what the game does with it is not established. Delete removes the selected entry and renumbers later item back-references as the original editor does. Deleting a structure or a unit, alone or in a selection, also deletes the casting structures or unit spell entries whose id is its id, as the original editor does; one Undo restores them with the object. Trap kinds choose cell or structure as source and cell or unit as target. Setting a trap, unit entry or casting structure's spell to 0 removes it immediately; undo restores its bytes. Add refuses spell 0 with a reason. Unknown kinds and opaque words remain unchanged.

The record list's Add button on filter 5 opens the same cell-spell Add window.

The map editor is the `--editor` mode of the game program, `againrom.exe
--editor` (`./againrom --editor` on macOS). The starter's **Map editor** button (**Редактор карт** on a
Russian base) opens it on the selected base and passes it the starter's mods
folder (`-mods-dir`), which Publish writes into. The starter first runs the
editor's map list on that base; when the editor cannot load it, for example a
second-game install, the starter shows the editor's error and opens nothing.
A folder that is not an install and a missing game program are refused with the
starter's message. From a terminal, `againrom --editor -assets <install>` opens the same
window. It opens with the install's map list. The editor's own help names its
keys: PgUp and PgDn page the list; dragging a unit, structure or sack moves
it and dragging the ground pans; C clones the selected unit, structure or
sack (click a cell; Esc cancels), Del deletes it, E edits its fields (Tab:
next field, Enter: set), and S adds a sack on the next clicked cell. F1 opens
the scrollable key reference. Ctrl+Z undoes, Ctrl+Y redoes and Ctrl+S saves
the selected mod package, or asks for a new `.alm` path when no mod is
selected. Enter opens a typed ALM path; Esc goes back. The editor never writes
into an install.

N asks for width x height. Each side can be 32..256 cells in steps of 8;
rectangles are supported. The blank map has two sides, the second named
Enemy and hostile to the first. The Maps button retains the installed-map
catalogue. U, a click on a cell and a class key add a unit. M edits the map's
name and description. The other metadata fields are labelled raw: their
stored bits can be edited, but this does not promise a light or time effect.
A new map saved with `againrom --editor -mod` goes to the mod's one mission entry
whose map file the folder does not hold yet. The party lands on the drop cell
of the map's script, so a map the party is to play needs one (the report
warns when the map has none and its fallback landing cells leave the map).
`againrom --editor -new <width>x<height> -out <file>` writes the blank map to a new
file and prints that report; `-new` alone opens the window on it, with
`-check` it prints the report.

A unit's window, in both views, sets the unit's Name, Speed, Sight and
Facing (DIV-4481, DIV-4482). An empty Name, Speed or Sight shows the kind's
value and keeps it. Speed is 1..255 and Sight 1..19 cells; both replace the
kind's value, and a person's equipment still adds to them. Facing is the
original's random draw or one of the eight directions; the Add window asks
it too, and the map draws the unit facing that way. The three are bytes of
the unit's record (+0x30 speed, +0x33 facing, +0x34 sight), which a shipped
map holds at 0. The name, 1 to 32 characters, goes into the mission page
`units.txt`: one line `<unit id>=<name>` per named unit, in UTF-8, written
to `<stem>.text/<lang>/units.txt` beside the map and among a mod's mission
texts. The game's card and message lines show that name, and a saved game
shows it again after loading. A person keeps an authored speed and sight
only until his first equip, training or potion after a load.

After the size, N asks for the starting height, 0..126, 64 by default; every
height cell of the new map holds it, and another number is refused with the
question kept. From a terminal, `-new 32x32/altitude40` gives the same.

Ctrl+E (Tools: Random terrain) asks for a seed and redraws every cell's tile
among the tiles that show the same terrain and the same corners, so no
transition moves; a cell of a mod's terrain group stays as it is.
Ctrl+Shift+R (Tools: Random heights) asks for a seed and lays
one hill or hollow per 64 cells over the heights the map has, through the
height brush. The prompt offers the last seed plus one; the same seed on the
same map gives the same map. Each command changes only its own plane and is
one undo step. Drag a selection box on the map and
release it: Ctrl+C then copies the tiles, heights and objects (trees and
other cell objects) under the box together with the records it selects.
Ctrl+V with the pointer on a cell writes the patch with its top-left corner
there, cut at the map's edge, replacing the objects it covers, and pastes the
records as before; one paste is one undo step. V (View: Height numbers) writes
on each cell the height of its top-left corner minus 64, so the new-map
default shows 0; the numbers appear only when the view is zoomed in far
enough for them to fit, and follow every height edit at once.

J, a click and a structure kind add a structure. C clones it with a new
identity; E edits its owner, kind, identity and placement words. U adds units;
their E fields include group membership. Select a unit and press K to create
its stock, then E to add items. S creates a ground sack. Select a stock or
sack and press I to choose the item number for a magic recipe; E on the new
recipe sets its spell and ordered effects. Up/Down cycles installed names
and current map references in these field prompts; numeric input also works.
Deleting a referenced recipe or a unit that still owns a stock is refused.

F2 lists actions, conditions and triggers. A adds one of those kinds; E and
Tab edit its fields. Choose operations, parameter kinds and references with
Up/Down or type their numeric values. Changing an operation keeps its current
parameters, so set each required kind and value. A trigger compares up to
three pairs of condition IDs with AND, then runs its four action slots in
order. Its fields also set once/repeat and trigger order. Constant/preset
conditions initialize variable registers by their zero-based condition
position; read-variable conditions and set/increment actions use those
registers or free slots up to 99. Runtime checks also write registers, so
the editor refuses a variable reference that would collide with one.
Clear trigger references before deleting or renaming their nodes. Unknown
imported operations and parameter bytes are retained; authoring choices list
the operations supported by the current runtime. D sets the party drop cell;
E on a drop action can edit another drop independently. G retains the goal
presets and target picker.

Two operations exist only in Againrom. The action "Set undying (Againrom)"
names a unit and a state, 1 to set or 0 to clear. While set, the unit does
not die: damage keeps lowering its health below zero, down to -32768, and
the unit stays alive, keeps acting and can still be attacked and healed.
Clearing the state on a unit at zero health or below starts its ordinary
death in that trigger pass, and the unit's killer gets the kill experience
and the Diary count once, as for any other death. A blow on an undying unit
at health -10 or below pays its striker no blow experience, the same rule as
for a body. Prismatic Spray takes an undying unit as its target at any
health. The condition "Health below (Againrom)" names a
unit and a signed number X from -32768 to 32767. Its value is 1 while the
unit's current health is below X, else 0; it is evaluated in the trigger
pass with the other conditions, once per 16 ticks. Compare it with a
constant 1 by "equals". A map holding either operation is an Againrom-only
map: the editor marks it in the status strip and in the script notice. The
mark adds no bytes. The original game reads such a map but runs neither
operation: the action does nothing and the condition stays 0, so a trigger
that waits for it to be 1 never fires. Example: trigger "Undying on" pairs constant
1 with constant 1 and runs "Set undying" on the boss with state 1; trigger
"Below" pairs "Health below" on the boss with X = -25 against constant 1 and
runs "Set undying" on the boss with state 0, then a message. The boss
survives every blow until its health falls below -25; on the next trigger
pass the message shows and the boss falls.

Every map editor input supports a caret, selection and text copy/paste.
Hold Backspace, Delete or an arrow to repeat after 400 ms, once every 40 ms.
Ctrl+Left/Right moves by words: a word is letters in any alphabet, numbers
or underscores; punctuation and spaces separate words. Shift with arrows,
Home or End selects, Ctrl+Shift+Left/Right selects by words, and Ctrl+A
selects the whole input. Drag inside a text field to select. Typing replaces
the selection; Backspace and Delete remove it.

While Add waits for a map cell, arrows pan the map without editing the
coordinates; Home, End and Backspace still edit them. Ctrl+Up/Down keeps
plain Up/Down in single-line prompts. In an Add choice list with cell
picking off, Shift+Up/Down travels the list; Ctrl+Home/End selects its ends.
Previously, arrows panned the map even with an object popup field focused.
They now move the field's caret (single-line Up/Down still travels choices).
When the field loses focus, arrows pan the map again.

With an input focused, Ctrl+C copies the selection, Ctrl+X cuts it and
Ctrl+V replaces it with pasted text. Windows and macOS use the system
clipboard; other systems keep text in this process. Without a focused input,
the map's object copy/paste keys keep their map meaning. Paste changes CRLF
and CR to LF, keeps LF in multiline bodies, and changes it to a space in
single-line inputs. It drops controls and characters the field cannot store,
then truncates at the field's byte limit without splitting a character.
Map, player and script names use their ALM Windows-1251 codec; message/page
bodies use the selected install's text codec and existing plain-form table.
Paths and package text accept printable Unicode. Numeric fields accept
digits and signs; raw metadata and package numbers also keep hexadecimal
notation. Ordinary typing retains the existing Apply validation.
Enter applies single-line fields as before. Message bodies use Enter for a
new line; page bodies and descriptions use Shift+Enter for a new line.

New message opens with its body ready for typing. Apply writes the body,
speaker and action's page reference as one undo entry. In an action's
details, choosing an existing message applies immediately when it is the
only pending field; one Undo restores it. Other pending fields still require
Apply, and New/Open keep the Apply-or-Cancel requirement for an unfinished
popup.

The trigger wizard's dialog action is Start dialog. Its row shows no number
and no text, only Configure, which opens the dialog editor. A trigger holds
at most one Start dialog: the player sees only the first. In a trigger that
has one, the action list shows Start dialog shut with that reason; other
triggers still offer it. A map whose trigger already has two keeps both,
and the Map panel still warns. Configure on an action that names no message
opens a New dialog that is made on Apply. A trigger with changes not yet
applied is applied first, as its own undo entry. At the top of the dialog
editor, Dialog chooses which dialog the action starts: New dialog or an
existing message by its first line, so triggers can share a dialog. When
another trigger also starts the shown dialog, a note names it.

The dialog editor, opened from Configure or from any Open Message link,
shows one card per page, in order: its number, its speaker and its text. A
conditional alternative is its own card with its page's number. Drag a card
by its grip to move it; a marker shows where it drops. Alt+Up and Alt+Down
move the card being edited. The card's x turns it into a Message removed
strip with Restore, until Apply or Cancel; at least one card remains. + after
the last card adds an empty card at the end with the cursor in its text.
Apply writes all changes as one undo entry; Cancel discards them.
Structural changes number the pages in their written order, with
conditional alternatives retaining a shared number.
Changing only text or a speaker preserves every other page and attribute,
including tips and conditional tags. Save As and mod export use the existing
map text folder and mod paths.

Speaker choices show catalog numbers and names, with 21 to 24 labelled by
role: Main hero, Traitor mage, Monk and Prisoner. Their person depends on the
hero chosen first. The Map panel warns if a trigger sends several messages:
the player will not see the second while the first is open. Use several pages
of one message for a conversation.

X edits mission pages. Enter after the page name continues directly into its
body. Tab still switches the fields. Supported pages are `title.txt`, `briefing.txt`, `briefmap.txt`,
`units.txt`, `eventNN.txt` and `tipsNN.txt`; numbers use at least two digits and can exceed
99. Shift+Enter inserts a new line, Ctrl+A selects the field and Enter applies
it. Pages use UTF-8; an unrepresentable character or an oversized page is
refused with the entered text retained. Map, page and route changes share
Undo/Redo. Saving only an ALM refuses unsaved page or route changes.

W opens the mod window. Its left column holds the working folder, ID, name,
mission list and Publish folder. Its scrolling right column edits the manifest's
version, author, description, target base, requirements, conflicts and load order
hints, followed by the selected mission's route, After win, companion or party, purse, the starting hero's
class and sex (Any writes no key), and read-only lines for carry and off. Enter accepts a value; Tab and
Shift+Tab move focus. Choice lists search by name as text is typed. Select a
member's detail button to edit level and equipment. Template retains the
character's equipment; Empty removes it. Override all armour replaces every
armour cell. The first member leads; a party has one to eight distinct members.
Existing declared characters are choices. Creating character definitions and
adding new tiles are outside this window.

A mod with no folder yet shows the working folder the editor proposes:
`workbench/<id>` in the player profile, the folder that holds `saves` and `maps`
(`Againrom/workbench/<id>` beside the programs when they stand in an install).
Save makes that folder and saves the mod into it; a folder already there is
refused and nothing is written. Another folder can be typed instead.

Open an unregistered map and choose Add map to append it. Move up/down or
Alt+Up/Down changes list order; After win changes the chain. Removing a chain
link asks before reconnecting its successors. Undo after Save makes the
project dirty again.

With a mod open, several of its maps stay open at once. The mission choice,
a mission of the campaign graph, the map list and File > Open each open the
map beside the ones already open; a map that is already open is shown, never
opened twice. From the second open map on, a strip of tabs stands under the
menu bar: one tab per open map, with its mission number and name, and a star
while the map has unsaved changes of its own. A click on a tab shows its
map. The x at the right end of the strip closes the map shown, and asks the
unsaved-changes question first when that map has changes. The tabs have no
keys. At most 8 maps are open at once; opening a ninth is refused with a
message that says to close one first. With one map open, or no mod, there is
no strip.

Each open map keeps its own view, selection, brush, unsaved changes and
Ctrl+Z/Ctrl+Y history; the history of one map never undoes a change of
another. The mod itself is one: W, the campaign graph and
test play read the manifest, the mission rows and the unsaved draft of every
open map, whichever map is shown. A step of a map's history made before
another tab was shown restores that map's own cells, pages, labels and
mission row, and leaves the rest of the mod as it stands. Ctrl+S saves the
map shown and the mod's rows and manifest; the files of the other open maps
stay as they are until their own tab is saved. Closing the editor or opening
another mod asks about each open map with unsaved changes in turn, showing
the map it asks about.

A mission the mod replaces by a map file named as its number is a row of the
mission list too, after the rows of `data/missions.toml` and named by its file
(`10.alm`). The list and the campaign graph open its map as they open any
row's. The file's name is its route: Route and Mission number are read-only,
Move and Remove are refused, and Save writes the map and its pages and no row
for it. An extended map saved under such an `.alm` name is refused; rename the
file to `.almx` in the mod folder.

A handwritten mod's first Save makes a complete copy under another parent
with the same ID. ID stays fixed. Its scripts, translations, hidden files,
replacement/disabled declarations and unexposed fields are retained. Disabled
declarations are marked read-only. Changes to source files outside the editor
require a reopen. Ctrl+S stages all drafts, checks every mission and party
through the normal loader, and replaces the working folder only after admission.
The source-preserving manifest writer retains comments, order, unedited fields
and an existing `entry`. New packages and base dumps need no entry script.
Project admission checks the data in the editor's context; the target base and
companion requirements are checked when the mod is activated by the normal
loader. F5 and Publish retain activation checks. Custom scripts still run during
admission and must load every edited mission under their default settings.

Publish uses the same packer as `-pack-mod`. The Publish folder shows the mods
folder the starter and the game read, and Publish writes `mod-<id>.res` there
with no path typed; it makes that folder when it is absent. Another existing
folder can be typed and stays selected for the session; a blank field returns to
the mods folder. Publish refuses a handwritten mod that has no complete copy
yet, a map that is no mission of the mod, a mod changed outside the editor, a
folder inside an install or inside the working mod, a typed folder that does not
exist, and the case where no mods folder is known and none is typed.
Publish asks Save and publish for unsaved changes, then separately asks Replace
for an existing RES. Cancel writes nothing at that step. A folder with the same
ID in the output directory is refused to avoid ambiguous discovery. The status
shows the full `mod-<id>.res` path and declaration count. A failed Publish after
Save reports that the mod was saved. Publish checks missing maps, chain links,
parties and script admission before replacing the RES.

To build a playable scenario, create its size with N, set D, add its units
and structures, write an `event01.txt` page with X, and use F2 to connect a
message and win action to the intended conditions. Set the route and a new
folder with W, save with Ctrl+S, and reopen the folder with W to check the
checkpoint. F5 enters character generation for the current map. Acknowledging
its Victory panel returns the game to the campaign town; the editor stays
open for further changes.

`againrom --editor -mod <mod folder>` saves into that mod through the same staged
project writer. The edited map keeps the path its mission entry names (the entry whose `number` is
the opened installed map's mission, or whose `map` is the opened file), and
refuses a map no entry names.
An existing mod save replaces only changed files, through temporary files
beside their targets. An external change conflicts only when the editor would
write that file. Unwritten files keep their attributes and modification times.
Saving into a handwritten mod leaves it handwritten; W's explicit new working
copy receives the editor ownership record. That record names editor-written
paths. Save As into a mod uses the same admitted writer.

F5 plays the map as it stands, unsaved edits included. For an open mod project
it copies the complete tree and drafts under the original ID inside the private
test-play directory. Scripts, characters, other missions and the chain reach
the normal loader. The selected mission is passed to `-mission`; F5 does not
save the working mod or clear its dirty state.

Without an open project, the editor writes the map into its own test mod,
`testplay`, in the editor's player profile (the
`testplay` folder beside the game program, or `Againrom/testplay` when the editor runs
from inside an install), and starts the game beside the editor on the same
base with `-mods testplay -mods-dir <that folder> -mission <n>`: character
generation, then the map as guest mission `n`. The test mod declares the map
as a guest mission after the campaign's first won mission that opens a town
granting a companion; `n` is the lowest number after that mission the game
accepts for a guest (21 on the first game's installs). It carries the text
pages of the installed mission the map was opened from, or of its mod entry,
including current text edits.
Each F5 replaces the test mod's map and pages. A map the report finds an error
in is not played; the report shows. The editor stays open, and nothing the
author saved changes. The map may lie in any folder, the game folder
included: F5 refuses only when the test folder is or contains the opened
map's folder, or is, contains or lies inside the selected mod folder. A
missing or failed game program is named on the message line. The second game
has no test play.

On the first game F5 first opens one window that asks who plays the map. It
holds six cards in a row: the hero character generation makes, then the
campaign's five companions, `npc22` to `npc26` of the character catalog (the
second and third at levels 2 and 3). A click on a card's top box puts the
character in the party or takes him out; the hero alone is in at first.
Under each card, Skills 100 writes `skills = 100` and All spells writes
`spells = "all"` for that member; a character who casts nothing has no All
spells box. Start, or Enter, plays the map with exactly the chosen party: the
test mod's mission declares it as its `party` array and no guest. Start takes
no press while nobody is chosen. Cancel, or Esc, plays nothing and writes
nothing. The window opens with the last choice until the editor closes. The
warnings the notice before a test play listed stand in this window, and no
separate notice follows.

The game composes `npc22` to `npc24` by the hero's class and sex, and the
hero is made only after the game starts. So the window names the kind of hero
its cards stand beside on the button between Start and Cancel: man fighter at
first, and a press takes the next of the four kinds. Each companion's card is
the member the game builds beside that hero: his name, his portrait and
whether he has an All spells box. The hero's card has that box when the hero
is a mage. The test mission is written with that kind's `class` and `sex`, so
the game plays it for that hero alone: when character generation makes
another, the start is refused in a line and the map maker picks the kind the
window showed. A name wider than its card stands on two lines.

The window stands for a map with no open project and for a project's own
mission that declares a guest and no `party` array; such a mission keeps its
number and `gold` in the test copy, and the working mod is not changed. It
keeps its `after` too, unless `after` names another mission of the project:
the test copy then plays it as a first link, after the first main mission
whose win opens a town and that no other mission of the project follows, so
the hero alone can start it. A mission that names `class` or `sex` offers the
kinds it plays for. A mission that replaces a campaign mission, declares its
own `party` array or is selected by its map file plays as declared, with the
notice alone.
