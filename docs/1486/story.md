# Towns in a mod

## Intent

A mod adds, changes and switches off towns of the first game. Its `data/towns.toml` is a layer of
town records over the install's town catalog, in the catalog's entry format, keyed by town ID. The
engine reads only the merged catalog. A mod town has its own description, art, map point,
opening mission and offers. Without a mod nothing changes.

## Authority

Owner direction: "Mods are patches of the base game" (`pipeline/OWNER-RULINGS.md`). Each kind of
game data is one catalog; layer 0 is filled by the base's decoder; each mod is a layer over it in
the same format, applied in load order: add a record, change fields of a record by its key, or
switch a record off. When two mods change one field, the later wins if `load-after` orders them
and the mod check names every override; without an order the launch refuses. Towns are the first
kind built this way (`pipeline/BACKLOG.md` T3, after T1 `docs/1484/story.md` and T2
`docs/1485/story.md`). One builder per kind: the catalog decoder, the description decoder
(`town.Decode`), `newTownState` and the mod loader stay the only builders of their kinds.

## As built

### The layer step

`pkg/mod/layer.go` is the one layer step, generic over key and record (`Merge[K, R]`). Its input
is layer 0's records, a key function, the patches in load order and the load order the mods state
(`Precedence`: the transitive edges of `requires` and `load-after` among enabled mods,
`PrecedenceOf`). A patch with a new key adds a record after layer 0's. A patch with a held key
runs its `Apply` over the record, which writes only the fields the patch names. `off = true`
removes the record from the merged list and `off = false` restores it. A check callback runs
after each mod's last patch over the records so far. The result is the merged records, the
`Override` list, and the source (mod, file, line) of each named and switched-off key.

Two mods on one key: a field both set, or a new key both name, is refused with both mods and the
field named, unless the later loads after the earlier. Then the later wins, and an `Override`
names the kind, key, field, both mods and the later's file and line. Fields set apart need no
order.

### The towns file

`pkg/mod/towns.go` reads `data/towns.toml` (`ParseTowns`): `[[town]]` tables with the keys `id`,
`description`, `hooks`, `offers`, `map-point`, `name`, `opens-after`, `missions`, `cite` and `off`,
each into `town.Entry`. A record lists the fields it names with their lines. `pkg/modrt` loads the
file through `game.data.add`. A `description` ending in `.json` is read from the mod folder
(`mod.ReadInside`) and named `<mod id>:<path>` in the merged catalog. Keys and refusals are listed
in `docs/MODS.md`, section Towns.

### The merged catalog

`FrontEnd.SetModTowns` (`pkg/game/townlayer.go`) runs once after `SetMods`. `layerTownBook` takes
the install's book (`townBookOf`) as layer 0 and refuses a catalog that is not a chapter catalog,
so the second game's catalog takes no layer. Before the layer step each record is read against the
first game: `hooks` other than `chapters` and `offers` other than `chapter-keys` are refused at
their line, and a `name` is encoded from UTF-8 into the install's alphabet by `encodeSaveLabel`
with the font's selector, the route mod item and character names take. A character the alphabet
lacks is refused at the `name` line. The merged catalog holds the encoded bytes, which the home
scroll and the travel card draw. After each mod the check runs:

- town 1, the campaign's starting town, cannot be switched off;
- `Catalog.Check`, the catalog decoder's own entry and vocabulary rules;
- `checkTownCatalog`, the description, hook and offer rules `decodeTownCatalog` applies to a
  shipped catalog; mod descriptions decode through `town.Decode` with the shipped ones;
- the campaign's `checkTowns`: each opening mission is one the campaign holds, and, with more
  than one chapter town, every town can open. The towns with `opens-after = 0` are open from the
  start and must offer a mission of the campaign's `Offered` set. A town opens once a town already
  open offers its opening mission. A town offers the missions `offeredHere` gives it: its own
  list, or, with no list, every mission no town lists. A mission outside `Offered` (10 and 20 on
  the stock campaign) is the campaign's own and counts as open from the start;
- a stated map point and its rect inside the background picture of every world-map mode.

A refusal names the record's line when the failing town is one this mod names (`town.EntryError`
carries the town), and the mod's first record otherwise. A reachability refusal
(`townCheckError`) carries the towns that bear on it in order: the town that never opens, the
towns listing its opening mission, the towns open from the start, then the rest. The refusal names
the first of them this mod names. The campaign and the session's `Town` are
rebuilt from the merged book, so `newTownState`, travel, offers and the map read it unchanged.
`cmd/againrom` prints `town records=N` in the check and one `override:` line per override.

### Description, art and map point

A description is either an installed one (`rom1`, `rom2-town1` and the others in
`townDescriptionData`) or a `.json` file of the mod. A shipped description keeps the hook set its
shipped catalog names. In a mod's description an art key `mod:<path>` names a file inside the mod
folder; every other key reads the install. The files are read when the mod loads
(`readModTownArt`). A missing file is refused at the record's `description` line, and a bound
entry cannot name a mod file. The art is read through `modTownArtSource` and held by the front
end's book (`townBook.modTownArtOf`), not by the install's shared archives, so no other front end
of the install sees it.

`town.MapPoint` gains `point = [x, y]` with `rect = [x, y, w, h]`, a point stated in world-map
pixels, beside `object` and `record`. Exactly one source is named. `townMapPoint` returns a stated
point first.

### SAV

The town record of `docs/1485/story.md` keeps the party's town ID. The mod mark lists every active
mod, so a SAVE made with a towns mod is marked, and a LOAD without the mod refuses with
`ErrModMark` naming it. No SAV field changed.

## Proof

- `pkg/mod/layer_test.go`: add, change, switch off and back on; an unknown key switched off; two
  mods on one field ordered (directly and through a third mod), unordered and ordered the other
  way; fields apart; one new key from two mods; the check after each mod; `PrecedenceOf`.
- `pkg/mod/towns_test.go`, `pkg/modrt/towns_test.go`: every key reads into the entry format;
  table, key and type refusals name the file and line; the fixture mod loads its description file.
- `pkg/town/catalog_test.go`: stated point forms and their refusals.
- `pkg/game/townlayer_test.go` on the travel fixture map:
  - `TestModTownOpensTravelsAndDrawsItsOwnArt`: East is closed until mission 20, then stands at its
    stated point titled East; West's home scroll; travel to East's square composed from the mod's
    description; shop offer 31 alone; the mod's art bytes. Its loss control is the same front end
    without the layer step.
  - `TestModSwitchesATownOff`: East leaves the catalog and the map; town 1, an unheld town and a
    second-game catalog are refused.
  - `TestTwoModsOnOneTown`: the ordered override with its line; the loss control drops the order
    and is refused; the unordered refusal; one new ID from two mods.
  - `TestTownLayerRefusals`, including `hooks = "destinations"` and `offers = "scenario-bank"` at
    their lines. `TestTwoModsOnOneTown` also pins that a mod changing a town another mod adds is
    accepted with `load-after` and refused without it.
  - `TestModTownNameDrawsInTheInstallAlphabet`: under the converting selector `Запад` and `Восток`
    reach the home scroll and East's card as `87 a0 af a0 a4` and `82 ae e1 e2 ae aa`; `East Ω`
    is refused at line 15.
  - `TestModTownThatCanNeverOpenIsRefused`: town 2 opening after mission 30 and listing 30 (with or
    without 31) is refused; town 4 opening after 41, which only the never-opening town 3 lists, is
    refused at town 4's table; town 1 listing only mission 99 leaves the start offering nothing and
    is refused; a chain (town 3 after 31 offering 40, town 4 after 40 offering 41) is accepted.
- Fixture mod `pkg/modrt/testdata/mods/east-town`: names town 1 West and adds East (town 2, opens
  after mission 20, offers mission 31, stated point (430, 230)). Its description and three BMP
  pictures are generated at test time; the repository holds no picture.
- `TestReleaseModTownOpensTravelsSavesAndLoads` (EN and RU): the fixture over the install. East's
  square, mask and tavern are the mod's pictures. A town SAVE in East carries the mod mark. A cold
  LOAD with the mod lands in East with its description. A LOAD without the mod refuses with
  `ErrModMark` naming `east-town`. Also `TestReleaseModTownSwitchedOff` (EN and RU), and
  `cmd/againrom` `TestReleaseCheckReportsTownOverridesAndRefusals` (EN and RU).
- `TestReleaseModTownNameDrawsInTheInstallAlphabet` (EN and RU): the fixture named `Запад` and
  `Восток`. The home scroll and the travel card hold the install-encoded name from West and, after
  travel, from East; on RU the bytes are `87 a0 af a0 a4` and `82 ae e1 e2 ae aa`. `East Ω` is
  refused at `data/towns.toml:15`. `TestReleaseModTownThatCanNeverOpenIsRefused` (EN and RU): the
  self-opening town 2 is refused on the install's campaign.
- Unchanged without a mod: the EN and RU release sets, the mission-stage scenarios and the second
  game's SAVE selection. The lane's return lists each run. Headless frames are in the lane's notes.

## Open debt

- T4: a tool that writes layer 0 of a kind in the mod format, so a modder starts from the
  install's records.
- T5: install files shadowed by a mod. A mod town's pictures are named by `mod:` keys in its own
  description; an installed description's art cannot be replaced.
- The second game's catalog takes no layer; its towns open by `completeBank`'s per-mission switch.
- A mod town offers missions from the chapter keys only; a town with its own mission source has no
  offer source yet. The shop goods stay shared by every first-game town.
- A whole-catalog rule (a mission listed by two towns) is blamed on the table that names the
  failing town, not on the field's line.
