# Mod data files

A mod is one folder with `mod.toml` and a Starlark entry script. The script's
`init(game, settings)` loads each data file once with `game.data.add(path)`.
Every refusal stops the launch and names the mod, the file and the line.

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
next field, Enter: set), and S adds a sack on the next clicked cell; Ctrl+Z
undoes, Ctrl+Y redoes and Ctrl+S asks for a new `.alm` path to save a copy
(Enter saves); Enter opens a typed ALM path; Esc goes back. The
editor never writes into an install.

N lists the new-map sizes above the installed maps (32, 48, 64, 80, 128 and
256 cells a side); a click opens a blank map with two sides, the second named
Enemy and hostile to the first. U, a click on a cell and a class key add a
unit. A new map saved with `mapedit -mod` goes to the mod's one mission entry
whose map file the folder does not hold yet. The party lands on the drop cell
of the map's script, so a map the party is to play needs one (the report
warns when the map has none and its fallback landing cells leave the map).
`mapedit -new <width>x<height> -out <file>` writes the blank map to a new
file and prints that report; `-new` alone opens the window on it, with
`-check` it prints the report.

`mapedit -mod <mod folder>` saves into the mod: Ctrl+S writes the edited map
to the path the mod's mission entry for it names (the entry whose `number` is
the opened installed map's mission, or whose `map` is the opened file), and
refuses a map no entry names.
