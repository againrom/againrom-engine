# Mod data files

A mod is one folder with `mod.toml` and a Starlark entry script. The script's
`init(game, settings)` loads each data file once with `game.data.add(path)`.
Every refusal stops the launch and names the mod, the file and the line.
`againrom -dump-base <folder>` writes the install's own records in this
format (section Base dump). `againrom --headless <scenario.json>` runs the
game with no window and writes the frame of a town (section Headless runs).
A mod's `files` folder replaces install pictures, sounds and texts (section
Install files). A mod folder packs into one file, `mod-<id>.res`, that the
game reads as the folder (section Packed mods).

## Data kinds

Every kind of data a mod carries is declared once in `pkg/mod/kinds.go`, and
a test holds this table to those declarations. A catalog kind is game data
the base holds as layer 0; each mod is a layer over it, keyed by the record
key. A merged kind combines the mods through one layer step: a record adds,
changes the fields it names, or switches a record off. A kept kind keeps its
own rules until the story named migrates it. A configuration kind has no base
catalog. Data files other than `settings.toml` are loaded by the entry script
with `game.data.add`.

| kind | file | key | layer | two mods on one record | see |
|---|---|---|---|---|---|
| items | `data/items.toml` | the row: `key` (an added item or weapon row), `item` (a shipped row); a mask bit is one field of its row | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; one field is one number of a row or one mask bit (row, tier, material); two mods adding one key need `load-after` too | Items, `DIV-1861`, `DIV-4005`, `DIV-4006`, `DIV-4007`, `docs/1272/story.md`, `docs/1509/story.md` |
| characters | `data/characters.toml` | `key` (a character of the catalog); `target` (an edit of a definition row or a family of tier rows) | catalog, merged | catalog keys: yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a mod's character off; target edits: all operations apply in load order, including overlapping families and rows, without requiring load-after | Guest parties, `docs/1278/story.md`, `docs/1508/story.md`, `DIV-4002`, `DIV-4003` |
| screens | `data/screens.toml` | `key`, within the mod | configuration | no: a screen is its mod's own; one action is added by one mod | `DIV-1894`, `DIV-1906`, `docs/1277/story.md` |
| companions | `data/companions.toml` | `chapter` and `companion` | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a condition off | `docs/1280/story.md`, `DIV-3909` |
| spells | `data/spells.toml` | `target` and the field; `[global]` keys | catalog, kept until an unscheduled spell catalog story | no: one field of one spell, and one global key, is set by one mod | Spell damage columns, `docs/1341/story.md`, `DIV-2364` |
| body sheets | `data/bodies.toml` | `name` | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; a later sheet replaces the whole sheet; `off` switches a sheet off | Body sheets, `DIV-2913`, `DIV-2914`, `DIV-3910` |
| weapon bodies | `data/wpnbodies.toml` (a mod folder may also use `data/weapon-bodies.toml`) | the weapon row, by `weapon`, `row` or both | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; a weapon row is never switched off | Weapon bodies, `DIV-2912`, `DIV-3911` |
| missions | `data/missions.toml` | `number`; `key` within the mod | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; a later mission replaces the whole mission; `off` switches a mission off; one guest mission plays after one win | Missions, `docs/1408/story.md`, `docs/1508/story.md`, `DIV-4001` |
| towns | `data/towns.toml` | `id`; a tavern row by `town`, `chapter` and `npc` | catalog, merged | yes, when `load-after` orders the mods: the later wins and the check names the override; without an order the launch is refused; `off` switches a town or a tavern row off | Towns, Tavern rows, `docs/1486/story.md`, `docs/1513/story.md` |
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
| `author` | optional string for people; the game does not read it |
| `description` | optional string for people; the game does not read it |
| `api` | integer, default 1; this build runs api 1 and refuses any other |
| `applies-to` | a string or list; default `["common"]`. Entries are `common` (every install), an install id such as `rom1-en`, or the family word `rom1` (every first-game install). The mod is refused on an install none of them names |
| `entry` | the Starlark file; default `main.star` when the folder holds one |
| `requires` | ids the mod needs, each with an optional constraint such as `"base>=1.0"` (operators `>=`, `<=`, `>`, `<`, `=`, dotted numbers). The required mod must be enabled and loads before this one |
| `conflicts` | ids that must not be enabled together with this mod |
| `load-after` | ids this mod loads after when they are enabled; it orders and does not require |

```toml
id          = "second-town"
title       = "Second Town"
version     = "1.0"
author      = "A. Modder"
applies-to  = ["rom1"]
requires    = ["base-art>=1.0"]
load-after  = ["other-mod"]
```

The script's `print` output is discarded, in `againrom -check` and in play.
A script reports a problem with `fail("text")`; the refusal names the mod,
the file and the line (`mod "m": main.star:3: fail: text`), and it stops the
check or the launch.

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
(`npc41m50.txt`). A new path anywhere else is refused as before.

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
| sound | `.wav` | a WAV the game decodes |
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

A mod that sets no rule, adds no data file and serves no text file
(`.txt`) changes no game state: it is listed after `state-free=`. A text
feeds the definitions the game builds its state from (the body list, item
names, class and spell tables), so a mod that serves any text is marked even
when that text changes nothing. A picture or a sound is only drawn and
played. A state-free mod is not written into a save's mod mark, so a save
made with a mod of pictures and sounds loads without it. A mod that changes
game state is marked, and its save is refused without it. A save whose mark
names a mod loads while every mod it names is active, whether or not that
mod is state-free now.

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
exits. It needs no install and opens no window. Every folder and file of
the mod is packed except names that start with a dot (`.git`), which no mod
reader takes: a script or data path with a dotted component is refused naming the path, for a
folder and for the packed file alike. Names match without case, as on Windows: ASCII and
Cyrillic letters (`Ё` too) of another case name the same file, and a pack holding two names equal
that way is refused naming both. The pack is refused, writing nothing, and the refusal names
the reason and, for a file, the file:

- the file exists; `-pack-replace` writes over it;
- the `-pack-out` folder is a game folder or lies inside one (a folder
  holding the game's archives);
- a file or folder name is longer than 15 bytes, the longest name a RES
  holds;
- a name holds a character the RES's code page, CP866, has no byte for.
  Latin and Cyrillic letters, digits and `. _ -` are one byte each;
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
- names inside a packed mod match without regard to the case of `A`-`Z`,
  as a folder's names do on Windows.

The map editor saves into a mod folder: `mapedit -mod` refuses a packed
mod. To change a packed mod, change its folder and pack it again.

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
sprite   = "assets/sprites/bone-pike.png"
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
still read in a mod folder, but it is 18 bytes and a packed mod holds names of
at most 15, so use the short name. A mod that holds both names is refused,
naming both files.

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
heroes      = "bodies/hammer2h.png"
heroes_l    = "bodies/hammer2h_l.png"
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

`data/missions.toml` holds `[[mission]]` tables of two kinds. A guest mission
(`kind = "guest"`, the default) adds one mission that a guest party plays
after a campaign mission is won and before the town that win opens, or after
another guest mission is won (see Chains). The party waits: it is kept as the
town SAV of that moment and comes back through the town LOAD when the guest
party wins the last mission of the chain. A replacing mission
(`kind = "replace"`) plays a campaign mission of the install on the mod's
map; see below.

| key | value |
|---|---|
| `kind` | optional; `"guest"` (default) or `"replace"` |
| `key` | required; 1 to 32 of a-z, 0-9 and `_`, starting with a letter |
| `number` | required; the mission number the mission runs as, 1..65535; a `guest` row may not use a campaign number or one of the install's `scenario` container; a `replace` row must (see Replacing a campaign mission) |
| `map` | required; the map file inside the mod folder (at most 1 MiB) |
| `texts` | optional; a folder inside the mod whose `*.txt` files (at most 64) serve as the mission's `main/text/battle/m<number>/` files: `eventNN.txt`, `tipsNN.txt` and `briefmap.txt`, the world map's mission text. Write them in UTF-8; the game encodes each into the running install's bytes, so one folder reads right on the English and the Russian base. Typographic quotes, dashes, the ellipsis and the no-break space are written as their plain forms (`"`, `'`, `-`, `...`, a space); any other character the install has no byte for is dropped, and the map check names it when the editor saves into the mod. A file that is not valid UTF-8 is served byte for byte, as mods written before UTF-8 were. A short install-encoded Russian file whose high bytes happen to form valid UTF-8 (CP866 `e2 a0 ac`) is read as UTF-8; whole sentences are not |
| `title`, `briefing` | optional; text keys of the mod's `text/<lang>/strings.toml`, written into the mission's `title.txt` and `briefing.txt` |
| `after` | required; the mission whose win starts this mission: a campaign mission whose successor is a town, or a guest mission of the active mods |
| `guest` | the guest form: the campaign companion who plays it, as the town of `chapter` would grant him or her over the party |
| `chapter` | the guest form: the town the chain ends in, whose `AddHero` grants `guest` |
| `party` | instead of `guest`, `chapter` and `carry`: 1 to 8 members, an array of `{ character = "...", level = 1..4, weapon = "...", shield = "...", armour = ["...", ...] }` (see Guest parties) |
| `gold` | optional, 0..1000000, default 0; the guest party's purse |
| `carry` | optional; only `"join"`: the guest joins in the town of `chapter` through its own grant, as he or she ended the chain |
| `off` | `true` switches off the mission of `number` another mod declares; the table holds only `number`, `off` and `kind` |

`{lang}` in `map` or `texts` becomes the language the game runs in (`en`,
`ru`). Missions merge by `number`: a mod that loads after another (its
`load-after` names that mod) replaces that mod's mission of the same number
as a whole, and the check names the override; without that order two mods
cannot declare one number. One mission follows one win: two guest missions
after one mission are refused at load, naming both files and lines. Nothing
but the state of the guest who joins carries back: the chain's purse, loot,
payment and every other member stay behind. A SAVE inside a guest mission
names the mod and the mission; a LOAD without that mod refuses.

A map's own pages take priority: `<stem>.text/<lang>/`, then
`<stem>.text/` beside its ALM. The declared `texts` folder supplies missing
pages; a folder ending in `/en` or `/ru` also reads page files in its parent
as a fallback. Other files in that parent are ignored. A replacing campaign mission finally keeps the install's
page for a number the map and mod leave out. Priority applies per file.

The map editor's Save As copies every language folder and unlanguaged page
beside the new ALM into `<stem>.text/`. Changed workspace pages go into
`<stem>.text/<lang>/`. Both destinations must be new and outside the install,
even when the source has no pages. W copies the same tree to `mission/text/` and declares
`texts = "mission/text/{lang}"`. An older package with only `mission/text/`
continues to load. Saving an editor-owned package keeps its recorded
unlanguaged and other-language pages. Unchanged unlanguaged pages stay
unlanguaged. A language folder never substitutes for another language.
Page storage does not change ALM or SAV layout.

```toml
[[mission]]
key      = "prologue"
number   = 25
map      = "mission/{lang}/map.alm"
texts    = "mission/{lang}/text"
title    = "prologue.title"
briefing = "prologue.briefing"
after    = 20
guest    = 22
chapter  = 30
```

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
map     = "mission/{lang}/map.alm"
after   = 20
guest   = 22
chapter = 30

[[mission]]
key    = "second"
number = 26
map    = "mission/{lang}/map.alm"
after  = 25
gold   = 100
party  = [{ character = "npc22" }, { character = "sellsword", level = 2, weapon = "Iron Short Sword", armour = ["Hard Leather Mail"] }]
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
- `weapon` and `shield` replace the row's weapon and shield cell; `armour`
  replaces all eight armour cells, those it does not name left empty. An
  empty string empties the cell. A name no item of the cell's kind has is
  refused at load.

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
`texts`, `title` and `briefing`, with the meanings above; `after`, `guest`,
`chapter`, `gold` and `carry` are refused. When the campaign reaches mission
`number` by ordinary play, the mission is built from the mod's map; the files
of `texts` and the `title` and `briefing` replace the installed mission's files
of those names, and the others stay the install's. Everything else stays the
install's: the scenario rows, the world map, the payment and the town that
follows. The rules:

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
- One mission number belongs to one mod: a second mod that replaces the same
  mission, or uses its number in any way, is refused when it loads. A guest
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
map    = "mission/{lang}/map.alm"
texts  = "mission/{lang}/text"
title  = "ambush.title"
```

The repository holds no map. `cmd/modmission -install <root> -map <n> -out
<mod folder>` copies an installed map and its event, tip, title, briefing and
briefmap texts into `mission/<lang>/` of a mod folder, the texts as UTF-8, and
writes nothing else. When the map editor saves into a mod, its map report warns
about a message the map's script sends for which neither the mod's `texts`
folder nor, for a replacing mission, the install holds an `eventNN.txt`.

## Towns

`data/towns.toml` holds `[[town]]` tables. Each is a layer record over the
first game's town catalog, keyed by `id`, in the catalog's entry format
(`docs/1485/story.md`). The install's catalog is layer 0; each mod's file is
applied in load order. A table adds a town when no earlier layer holds its
`id`, changes only the fields it names when one does, or switches the town
off with `off = true`. The second game's catalog takes no layer.

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
starts with `mod:` reads a file inside the mod folder (`mod:art/east.bmp`);
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
description = "towns/east.json"
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
| `square` | `name` and `enter`, the steps that put the party on the square |
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
`layers`), `art`, `music`, `seats` and `buttons`. A step is `{ "hook": name }`
or `{ "composer": "reset" }`.

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
| `mod.toml` | the manifest: `id` (the folder's name), a title, version `1.0.0`, `applies-to` naming the install it was taken from, and `entry = "main.star"` |
| `main.star` | the entry script: `game.data.add("data/towns.toml")` |
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

The second game reads no towns layer. Its dump still lists its three towns
in `data/towns.toml`, for reading, and says so in the file and in the
printed lines; its `main.star` adds no file. A second-game town has no
`name` key: the second game draws no name from its catalog.

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
`escape`, `esc`, `load`, `l`, `f1` to `f9`, `f11`, `f12`, `shift-f4`, `ctrl-f5`
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
another name lists them.

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

Cell spells appear in the record list as traps, unit spell entries or casting structures. F3 opens the shared Add window: choose an existing cell spell as a template, a unit or structure id, its cell, spell (1..28) and power (0..100). Unit and casting-structure entries require a unique object with that id at its anchor cell; their id and cell are read-only after placement. Add updates a casting structure already recorded at that cell with that id and opens its popup; unit spell entries keep appending. E edits the known fields; Delete removes the selected entry and renumbers later item back-references as the original editor does. Trap kinds choose cell or structure as source and cell or unit as target. Setting a trap, unit entry or casting structure's spell to 0 removes it immediately; undo restores its bytes. Add refuses spell 0 with a reason. Unknown kinds and opaque words remain unchanged. Scan range is shown but editing it, and creating without a template, await research on the original editor's bounds and fresh allocation bytes.

The record list's Add button on filter 5 opens the same cell-spell Add window.

The map editor, `mapedit.exe` (`./mapedit` on macOS), ships beside the game
and the starter. The starter's **Map editor** button (**Редактор карт** on a
Russian base) opens it on the selected base. The starter first runs the
editor's map list on that base; when the editor cannot load it, for example a
second-game install, the starter shows the editor's error and opens nothing.
A folder that is not an install and a missing `mapedit` are refused with the
starter's message. From a terminal, `mapedit -assets <install>` opens the same
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
A new map saved with `mapedit -mod` goes to the mod's one mission entry
whose map file the folder does not hold yet. The party lands on the drop cell
of the map's script, so a map the party is to play needs one (the report
warns when the map has none and its fallback landing cells leave the map).
`mapedit -new <width>x<height> -out <file>` writes the blank map to a new
file and prints that report; `-new` alone opens the window on it, with
`-check` it prints the report.

After the size, N asks for the starting height, 0..126, 64 by default; every
height cell of the new map holds it, and another number is refused with the
question kept. From a terminal, `-new 32x32/altitude40` gives the same.

Ctrl+E (Tools: Random terrain) asks for a seed and redraws every cell's tile
among the tiles that show the same terrain and the same corners, so no
transition moves. Ctrl+Shift+R (Tools: Random heights) asks for a seed and lays
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
speaker and action's page reference as one undo entry. The trigger wizard's
message parameter uses the same New message choice and Open Message link.
Choosing an existing message applies immediately when it is the only pending
field; one Undo restores it. Other pending fields still require Apply, and
New/Open keep the Apply-or-Cancel requirement for an unfinished popup.
The message popup lists every page, including conditional alternatives, with
its text and speaker. Scroll to edit any page. Add after inserts a page after
the chosen page; Add last appends one. Remove deletes the chosen page, and
Up / Down move it. At least one page remains. Apply writes all changes as one
undo entry; Cancel discards them. Structural changes number the pages in
their written order, with conditional alternatives retaining a shared number.
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
`eventNN.txt` and `tipsNN.txt`; numbers use at least two digits and can exceed
99. Shift+Enter inserts a new line, Ctrl+A selects the field and Enter applies
it. Pages use UTF-8; an unrepresentable character or an oversized page is
refused with the entered text retained. Map, page and route changes share
Undo/Redo. Saving only an ALM refuses unsaved page or route changes.

W selects a mod folder or creates an owned package in a new explicit folder.
Its fields set the mod ID/title and the guest or campaign-replacement route;
Tab selects the next field and Up/Down selects available choices. A source
with several missions has a source-mission choice. An imported manual mod
remains read-only: choose a new folder and ID to save the selected mission as
an owned copy. The source files stay unchanged. An owned package contains one
mission; select and copy other source missions separately. Ctrl+S stages the
map, declaration and pages, validates them with the normal loader and then
saves the package. Unrelated files survive updates; conflicts refuse the save.

To build a playable scenario, create its size with N, set D, add its units
and structures, write an `event01.txt` page with X, and use F2 to connect a
message and win action to the intended conditions. Set the route and a new
folder with W, save with Ctrl+S, and reopen the folder with W to check the
checkpoint. F5 enters character generation for the current map. Acknowledging
its Victory panel returns the game to the campaign town; the editor stays
open for further changes.

`mapedit -mod <mod folder>` saves into the mod: Ctrl+S writes the edited map
to the path the mod's mission entry for it names (the entry whose `number` is
the opened installed map's mission, or whose `map` is the opened file), and
refuses a map no entry names.

F5 plays the map as it stands, unsaved edits included. The editor writes the
map into its own test mod, `testplay`, in the editor's player profile (the
`testplay` folder beside `mapedit`, or `Againrom/testplay` when the editor runs
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
author saved changes. A missing or failed game program is named on the
message line. The second game has no test play.
