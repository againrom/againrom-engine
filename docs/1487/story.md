# Base dump for modders

## Intent

A modder who has only the shipped game reaches the install's own records in the format a mod
reads. `againrom -dump-base <folder>` writes a mod folder: the manifest, an entry script, and layer
0 of each layered catalog. Today the town catalog is the only layered catalog, so the folder holds
`data/towns.toml`. Without the flag nothing changes.

## Authority

Owner direction: "Mods are patches of the base game" (`pipeline/OWNER-RULINGS.md`): each kind of
game data is one catalog, layer 0 is filled by the base's decoder, each mod is a layer over it in
the same format, and a tool dumps layer 0 in the mod format. This is T4 of the mod track
(`pipeline/BACKLOG.md`), after T3 `docs/1486/story.md`. One builder per kind: a kind's dump is
written by that kind's own code beside its layer. Clean room: the dump runs on the player's
machine from the player's install, and no dump output enters the repository.

## As built

### The flag

`cmd/againrom` takes `-dump-base <folder>`. It excludes `-check`, `-headless`, `-mods` and the
other check modes. It builds the front end as `-check` does, with sound off and no profile read,
calls `FrontEnd.WriteBaseDump`, prints `againrom: base dump of <base> in <folder>`, one
`againrom: wrote <file>` line per file and one `againrom: note:` line per layer the base does not
read, and exits 0. A refusal prints `againrom: -dump-base: <reason>` and exits 1.

### The folder

`pkg/game/basedump.go` writes the folder. `baseDumpLayers` lists one function per layered kind;
each returns its file, bytes, whether the base reads that layer, and a note when it does not. The
folder gets `mod.toml` (`id` = the folder's name, `applies-to` = the install's base ID,
`entry = "main.star"`), `main.star` (one `game.data.add` per layer the base reads, a comment for
each it does not), and the layer files.

Refusals, each checked before anything is written and wrapped with the path:

- `ErrBaseDumpInstall`: the folder is the install, lies inside it or inside a preserved install
  beside it (`preservedPairRoots`), or lies below any folder holding the five required archives
  (`requiredArchiveAncestor`);
- `ErrBaseDumpNotEmpty`: the folder exists and holds anything;
- `ErrBaseDumpExists`: the path is a file, or a file appears while writing;
- `ErrBaseDumpName`: the folder's name is not a mod ID.

Files are created with `O_EXCL`. On a failure the files and folders made so far are removed.

### The towns file

The keys of a `[[town]]` table other than `id` and `off` are one list, `townKeys` in
`pkg/mod/towns.go`. Each key carries how the decoder reads it, how a record copies it into a
catalog entry, and how a dump writes it. `ParseTowns`, `TownRecord.Apply` and `WriteTowns` read
only this list, and the decoder's unknown-key refusal lists `townFields`, which is derived from it.

`WriteTowns` writes one table per entry: `id`, every key the entry holds a value for, and
`off = false`. Strings are written as TOML basic strings; text that is not UTF-8 or holds a
control character is not written. The file is read back through `ParseTowns`, and an entry that
does not come back equal (an empty list equal to none) is an error, so the file never holds a
value its entry does not.

`FrontEnd.baseTownLayer` (`pkg/game/townlayer.go`) supplies the entries: the shipped catalog the
install's edition names (`townBookOf`). A first-game town with no name of its own is drawn with
the install's home title, so its `name` is that title decoded to UTF-8 by `decodeSaveLabel`, the
inverse of `encodeSaveLabel` that a mod's name takes into the install's alphabet. The decoded name
must encode back to the same bytes; a title that does not is left out with a note. `description`
stays the installed description's name and `map-point` the world-map object: no install bytes are
copied.

The second game's catalog takes no layer (`townCatalogTakesLayer`, shared with `layerTownBook`).
Its dump lists its three towns without `name` (the second game draws none from its catalog), and
the note "the second game reads no towns layer" stands in the file's head and in the printed
lines. Its `main.star` adds no file.

## Proof

- `pkg/mod/townsdump_test.go`: a written file holds, in each table, every key the decoder's own
  unknown-key refusal lists, and reads back to its entries. Loss control: the same file with any
  one key's lines cut is reported as missing exactly that key. An entry with no name writes every
  other key; a name with a line break or bytes that are not UTF-8 is refused.
- `pkg/game/basedump_test.go`: the dump layered over the shipped catalog on the travel fixture
  gives town 1 unchanged but for the name the home scroll already draws, and an edited name shows.
  Refusals: the install, a folder inside it, a folder below another install's archives, a
  non-empty folder, an existing file, a name that is not an ID; the tree is unchanged after each.
  A file found while writing is refused and the files written before it are removed. The second
  game's layer lists towns 1, 2, 3 with the note. `decodeSaveLabel` reads the converting
  selector's `87 a0 af a0 a4` as `Запад` and refuses a byte whose letter encodes elsewhere.
- `cmd/againrom/dumpbase_test.go`: the flag's exclusions; the folder written on a synthetic
  install, a second run into it and a run into the install refused with nothing left.
- Release tests (EN and RU): `TestReleaseBaseDumpListsTheTownWithEveryKey` (town 1 holds the
  decoder's keys; the name encodes back to the home title; Cyrillic on RU),
  `TestReleaseBaseDumpAsAModChangesNothing`, `cmd/againrom` `TestReleaseDumpBaseWritesAModTheCheckAccepts`
  (`-check` with the dump prints the plain summary, then the mod's lines and `town records=1`, and
  no override). ROM2 (EN and RU): `TestReleaseSecondTownBaseDumpListsItsThreeTowns`.
- `TestReleaseBaseDumpAsAModChangesNothing` compares one session without a mod against the same
  session with the whole dump and with its one record cut out as a mod: the town square and world
  map frames, the home scroll title, the town SAV bytes, and mission 10's World hash, world bytes
  and mission SAV bytes after 40 ticks. Two sessions without a mod agree first. With the town
  layer alone the SAV bytes are equal. Through the launcher's steps (`SetMods`) each SAV carries
  the mod mark naming the mod and no value, and the bytes are equal with the mark cleared. Loss
  control: one edited key, the name, changes exactly the world map frame and the home title.

## Open debt

- Each later layered kind (T6 to T9: missions, characters, tavern talks, items) adds its function
  to `baseDumpLayers` beside its own layer.
- T5: install files shadowed by a mod. The dump names installed descriptions and art by reference
  and copies no file. A mod made of the unchanged dump still puts the mod mark in a SAV; T5 makes
  the mark list only mods that change game state.
- The second game reads no towns layer (`docs/1486/story.md`), so its dump is for reading.
