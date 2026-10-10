# Mod data files

A mod is one folder with `mod.toml` and a Starlark entry script. The script's
`init(game, settings)` loads each data file once with `game.data.add(path)`.
Every refusal stops the launch and names the mod, the file and the line.

| file | what it changes | keys and rules |
|---|---|---|
| `data/items.toml` | items in free code rows and edits of shipped rows | `DIV-1861`, `docs/1272/story.md` |
| `data/characters.toml` | tavern characters | `docs/1278/story.md` |
| `data/companions.toml` | join conditions | `docs/1280/story.md` |
| `data/screens.toml` | information screens and menu actions | `DIV-1894`, `DIV-1906`, `docs/1277/story.md` |
| `data/spells.toml` | spell values and formula tables | `docs/1341/story.md`, `DIV-2364` |
| `data/weapon-bodies.toml` | the body a weapon is drawn with | below, `DIV-2912` |
| `data/bodies.toml` | body sheets the mod supplies | below, `DIV-2913`, `DIV-2914` |

## Weapon bodies

`data/weapon-bodies.toml` holds `[[weapon]]` tables. Each draws one weapon row
with another body.

| key | value |
|---|---|
| `weapon` | the weapon table row's name, as the install's weapon table spells it (`War Hammer` on every install) |
| `row` | the weapon row number, 1 to 31 |
| `body` | a body the install ships under both `heroes` and `heroes_l`, or a body a loaded mod supplies |

A table gives `weapon`, `row` or both; both must name the same row. `body`
names the base body. A hero with a shield takes the `_` form of a shipped body,
as without the mod; a supplied body's `_` form is used when a mod supplies it,
else the body itself. A row the shipped body list leaves blank, two choices for one
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
| `origin` | `[x, y]` of the pixel that stands on the ground point, inside the frame |
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
