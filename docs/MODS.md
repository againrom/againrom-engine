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

| file | what it changes | keys and rules |
|---|---|---|
| `data/items.toml` | items in free code rows and edits of shipped rows | `DIV-1861`, `docs/1272/story.md` |
| `data/characters.toml` | tavern characters | `docs/1278/story.md` |
| `data/companions.toml` | join conditions | `docs/1280/story.md` |
| `data/missions.toml` | missions a guest party plays between two campaign missions | below, `docs/1408/story.md` |
| `data/screens.toml` | information screens and menu actions | `DIV-1894`, `DIV-1906`, `docs/1277/story.md` |
| `data/spells.toml` | spell values and formula tables | `docs/1341/story.md`, `DIV-2364` |
| `data/weapon-bodies.toml` | the body a weapon is drawn with | below, `DIV-2912` |
| `data/bodies.toml` | body sheets the mod supplies | below, `DIV-2913`, `DIV-2914` |
| `data/towns.toml` | towns: add, change named fields, switch off | below, `docs/1486/story.md` |

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
  holds (`data/weapon-bodies.toml` is 18: a mod with weapon bodies stays a
  folder);
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

`data/weapon-bodies.toml` holds `[[weapon]]` tables. Each draws one weapon row
with another body.

| key | value |
|---|---|
| `weapon` | the weapon table row's name, as the install's weapon table spells it (`War Hammer` on every install) |
| `row` | the weapon row number, 1 to 31 |
| `body` | a body the install ships under both `heroes` and `heroes_l`, or a body a loaded mod supplies |

A table gives `weapon`, `row` or both; both must name the same row. `body`
names the base body. A hero with a shield takes the `_` form of a chosen body when a `_` sheet
exists (under both directories for a shipped body, supplied by the mod for a
supplied one), else the body itself. A row the shipped body list leaves blank, two choices for one
row, and an unknown weapon, row or body are refused.

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

The PNG is a grid of `frame`-sized cells. Row 0 holds the standing frames, one
per facing: 16, or 9 at 5 directions. Then, for `move`, `attack` and `idle` in
that order, one row per direction holds that action's frames from column 0.
A pixel with alpha 0 is background; every other pixel is opaque, and a sheet
is reduced to at most 256 colours. A cell outside the picture, or a cell with
no visible pixel, is refused, naming the action, direction and cell.

A fallen hero is drawn with the shipped dying body of the directory. A
supplied body has no owner shading and no silhouette sheet. A shield form is a
second body named with the `_` suffix.

```toml
# A two-handed body: 8 directions, 16 standing frames on row 0, then one row
# per direction for each of move, attack and idle (25 rows of 64x80 cells).
[[body]]
name        = "hammer2h"
heroes      = "bodies/hammer2h-heroes.png"
heroes_l    = "bodies/hammer2h-heroes_l.png"
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
after a campaign mission is won and before the town that win opens. The party
waits: it is kept as the town SAV of that moment and comes back through the
town LOAD when the guest wins. A replacing mission (`kind = "replace"`) plays
a campaign mission of the install on the mod's map; see below.

| key | value |
|---|---|
| `kind` | optional; `"guest"` (default) or `"replace"` |
| `key` | required; 1 to 32 of a-z, 0-9 and `_`, starting with a letter |
| `number` | required; the mission number the mission runs as, 1..65535; a `guest` row may not use a campaign number or one of the install's `scenario` container; a `replace` row must (see Replacing a campaign mission) |
| `map` | required; the map file inside the mod folder (at most 1 MiB) |
| `texts` | optional; a folder inside the mod whose `*.txt` files (at most 64) serve as the mission's `main/text/battle/m<number>/` files: `eventNN.txt`, `tipsNN.txt` and `briefmap.txt`, the world map's mission text. Write them in UTF-8; the game encodes each into the running install's bytes, so one folder reads right on the English and the Russian base. Typographic quotes, dashes, the ellipsis and the no-break space are written as their plain forms (`"`, `'`, `-`, `...`, a space); any other character the install has no byte for is dropped, and the map check names it when the editor saves into the mod. A file that is not valid UTF-8 is served byte for byte, as mods written before UTF-8 were. A short install-encoded Russian file whose high bytes happen to form valid UTF-8 (CP866 `e2 a0 ac`) is read as UTF-8; whole sentences are not |
| `title`, `briefing` | optional; text keys of the mod's `text/<lang>/strings.toml`, written into the mission's `title.txt` and `briefing.txt` |
| `after` | required; the campaign mission whose win starts this mission; its successor must be a town |
| `guest` | required; the campaign companion who plays it, as the town of `chapter` would grant him or her over the party |
| `chapter` | required; a town chapter whose `AddHero` grants `guest` |
| `gold` | optional, 0..1000000, default 0; the guest's purse |
| `carry` | optional; only `"join"`: the guest joins in the town of `chapter` through its own grant, as he or she ended the mission |

`{lang}` in `map` or `texts` becomes the language the game runs in (`en`,
`ru`). One mod or two cannot add two missions under one number or after one
win. Nothing but the guest's own state carries back: the mission's purse,
loot and payment stay behind. A SAVE inside the mission names the mod and the
mission; a LOAD without that mod refuses.

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
| `data/towns.toml` | every town of the install as a `[[town]]` table with every key of the Towns section: `id`, `description`, `hooks`, `offers`, `map-point`, `name`, `opens-after`, `missions`, `cite`, and `off = false` |

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

X edits mission pages. Set the page name, reopen X and use Tab to reach its
body. Supported pages are `title.txt`, `briefing.txt`, `briefmap.txt`,
`eventNN.txt` and `tipsNN.txt`; numbers use at least two digits and can exceed
99. Shift+Enter inserts a new line, Ctrl+A clears the field and Enter applies
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
