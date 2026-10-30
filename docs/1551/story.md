# A second-game town SAVE writes the original's own `Bsg&` file

## Intent

The owner's direction: every second-game save point writes a native `Bsg&` SAV that the
original loads. The lane narrowed this to the town. A second-game town SAVE now writes a
`Bsg&` file in the second game's records. A mission SAVE still writes the engine's `Asg&`
document; the mission writer is the next story (DIV-3079).

## Authority

- Owner, in the brief: one producer; record shapes are `sav.Profile` data; a field takes its
  value from the current state, then a rule from claims, then the loaded document's bytes,
  then a claimed constructor value; a field with no source is a DIV debt row and never refuses
  a save; older `Asg&` second-game documents keep loading; the first game's SAV bytes do not
  change.
- Coordinator narrowing: town only; the mission `Bsg&` writer is the next story.
- Claims: `R2-SESSION-017` (magic), `R2-SESSION-018` (serializer and trailer),
  `R2-SESSION-021` (store and tail), `R2-SESSION-150` (preference bytes), `R2-SESSION-155`
  (availability), `R2-SESSION-156` (shop grid), `R2-SESSION-165` (bag I/O), `R2-ENGINE-319`
  and `R2-ENGINE-320` (main skill), `R2-ENGINE-402`, `R2-ENGINE-403`, `R2-ENGINE-410` (bag,
  Item, insert index), `R2-ENGINE-675` (Player bytes), `R2-SESSION-020` (the second game's
  Player blocks, their meaning Unknown), `R2-SESSION-285` (the town writer's empty
  `SpellBook/Shortcuts`).

## As built

### Records (Part A, 7d8dd1c5)

`sav.Profile` carries the record programmes of each game. `SecondGame` holds the second game's
Unit, Humanoid, Player and Pointer programmes, its tail writer and its magic. Parse, serialize
and validate take the profile. `DocumentData` holds `Game`, `SecondTail` and the world extras.

### Town SAVE

`ExportCurrentSave` stays the one producer. It builds the engine's document and encodes it as
`Asg&`. Then `campaignService.saveForm` decides the form:

- the first game and a second-game mission keep the engine's records;
- a second-game town writes what the second game's Humans hold that the engine keeps
  elsewhere (`writeSecondTownHolders` in `pkg/game/savform.go`): the hero's main skill, every
  bag's insert index 0x7fff, the campaign's bag as Item records in the hero's Inventory owned
  by the hero's Player, every bag's weight, each magic Item's row from its ID, each Effect's
  T0E 0 and the Player's F48 0;
- the town producer writes each member's U148 party identity (`mapload.PartyIdentity`), the
  hero's Humans row his loaded record names (`mapload.PartyHumanRow`), and each actor's
  loaded town state and the Player colour (`writeTownActors`, `writeTownPlayerColor`;
  DIV-4324) in both games;
- `sav.DocumentInProfile` rewrites the records in the `SecondGame` form (the profile's
  `form` data in `pkg/formats/sav/profile.go`), sets the game, both trailer words 0xBADFACE1,
  writes `SpellBook/Shortcuts` with no element (R2-SESSION-285) and drops the engine's
  supplement leaves;
- `secondTownTail` writes the campaign tail: bank, shop records, available places, current
  ordinal;
- `sav.StoreEngineDocument` stores the engine's `Asg&` document in the store leaf
  `/CurrentState/AgainromForm`, and `sav.EncodeDocumentData`, the one document encoder,
  encodes the `Bsg&` file. `internal/archtest` lists `EngineDocument` as a reader and the
  `savtool document` verb as a caller of the encoder.

### LOAD

`restoreOriginal` first asks `sav.EngineDocument` for the engine's document. A file with the
leaf loads from it, so an engine town file loads to the same state the `Asg&` document did. A
`Bsg&` file with no value record at the leaf's path loads as the original's: a town file as an
original town (DIV-4153), a mission file as mission 10 (DIV-4170 to DIV-4175). A file whose leaf
is present but does not validate (its framing, length, padding or inner document) refuses with
a bounded error. An `Asg&` document loads as before.

A town LOAD of an original keeps each restored actor's town state (`keepLoadedTownActors`)
and the hero's record row (`heroRecordRow`). A hero bag piece with no bag form, such as magic
item 0x0e4d at the price word 0xffffffff in the Kaarg town files, stays in the hero's pack
(`secondHeroBag`, `leaveBagToCampaign`; DIV-4191) instead of refusing the LOAD.

A `Bsg&` mission file's whole document now decodes in the second game's records. The complete
document import binds each Player's formation from the block the document's profile names
(`sav.Profile.PlayerFormation`): the first game's `PRaw32`. No claim places a formation block
in the second game's Player (R2-SESSION-020 leaves its 2,560 and 36 bytes unnamed), so the
`SecondGame` profile names none and `decodeSavedDocument` keeps such a file on the partial
import, the route it took before its document decoded. The five EN mission originals LOAD and
resume as they did on 6fcba414.

### Instruments

`savtool document FILE...` decodes, re-encodes and compares each file by part (file, body,
label, store, tail), names differing ranges and record fields, and tests whether two stores
hold the same entries. `savtool compare A B` compares two files the same way.

### Kinds and their one builder

| Kind | Builder |
|---|---|
| SAV record shapes per game | `sav.Profile` programmes and its `form` data |
| SAV document producer | `ExportCurrentSave` |
| game choice of the saved form | `campaignService.saveForm` |
| bag Item records | `cityObjectProjection` (`packRefs`, `appendPack`) |
| second-game tail | `SecondTail` writer (Part A) |
| SAV LOAD | `restoreOriginal` |
| SAV LOAD import | `decodeSavedDocument`, `importSavedDocument`, `importSavedGroupBindings`, `importSavedFormations`; the Player formation block from `sav.Profile` |
| engine document leaf | `sav.EngineDocument` and `NativeFormPath`, one reader |
| second-game LOAD bag | `secondHeroBag`, `leaveBagToCampaign`, the bag form `secondshop.FromInstance` |
| town actor record and its carrier | `nativeCityUnitData`, `mapload.KeptActor` (`keptFromCharacter`, `writeTownActor`) |
| a member's Humans row | `mapload.PartyHumanRow` |

## Proof

- `TestSecondTownSaveWritesTheSecondGameRecords`: an engine town SAVE is `Bsg&`, game rom2,
  with the tail, the hero's main skill, the insert index, the bag owner and weight; the
  original-format reader `sav.Open().Party()` finds the hero and two Items; a cold LOAD of the
  records alone, of the whole file and of the embedded `Asg&` document each give the same
  town, party, purse, bank and campaign position.
- `TestSaveFormKeepsTheEngineRecords`, `TestEngineTownDocumentWritesTheSecondGameForm`,
  `TestEngineTownDocumentRefusals`, `TestCompareNamesThePartThatDiffers`,
  `TestEqualTextRanges`. On base 712a2783 the new tests fail to compile.
- The nine EN originals, through `savtool document`: every tail is equal. The bodies of
  game0002, game0003, game0005, game0006, game0001 and game9998 are equal; game0000, game0004
  and game9999 differ in the last body byte, an alignment pad the original leaves as debris.
  Labels differ only in debris after the NUL. Every store holds the same entries in another
  order with zeroed name debris.
- The four EN town originals (game0000, game0002, game0003, game9998), LOAD and SAVE with no
  play through `savecheck`: each loads as a town and saves as `Bsg&`; gold, bag, available
  places, main skill, insert index, preference bytes, bag weight and the worn item records apart from their keys and T0E match. The
  fields that differ are the rows DIV-4305 and DIV-4306.
- EN and RU resaves of game9998 are byte-identical.
- `TestSecondMissionFileWithTheWholeStoreLoads`: a synthetic mission file with the nine store
  directories of the EN mission originals decodes whole and LOADs mission 10 with its hero; on
  92b96149 it is refused with "saved Player lacks its formation block".
  `TestEngineDocumentRefusesALeafThatDoesNotValidate`: a town file whose leaf framing word is
  changed refuses; on 92b96149 it was read as the original's.
- The five EN mission originals (game0001, game0004, game0005, game0006, game9999) and game0001
  on RU, `savecheck load`: refused on 92b96149, loaded after, with output equal to 6fcba414's;
  the EN game0001 re-save is byte-equal to 6fcba414's.

- `TestSecondTownLoadKeepsAPieceTheBagHasNoFormFor`: a fixture town file with magic item
  0x0e4d at the price word 0xffffffff loads with it in the hero's pack, and two SAVE and LOAD
  rounds keep it in his Inventory at that price and row 77; on 810022ac the LOAD refuses with
  "bag item 0x0e4d has no bag form". `TestSecondTownSaveKeepsTheLoadedHeroRecord`: a fixture
  hero's T0C 28, U148 21, U50 0x0b, U136 1, U6C 128 and the Player colour survive two rounds;
  on 810022ac each is 0.
- The seven files of the second owner corpus (game0008 to game0012, game9998, game9999), EN and
  RU `savecheck -party -resave load`: every file loads; the towns and missions resave, except
  the won mission game0010, whose SAVE is not available; EN and RU resaves are byte-identical.

## Open debt

- DIV-4304: the original's acceptance of the engine document leaf, its size and the store's
  entry order.
- DIV-4305: town record fields without a source.
- DIV-4306: tail fields without a source.
- DIV-3079: a mission SAVE writes the engine's `Asg&` document.
- The owner's test of the original loading an engine town file is open.
- The second game's Player formation: no claim places it, so a `Bsg&` mission file keeps the
  partial import.
- DIV-4324: the loaded town actor state is carried by the first game's rule.
- A town LOAD drops a joined member such as identity 22 in the Kaarg town file game9998
  (T0C 30, U148 22): `persistentOriginalCharacter` keeps the hero and the first game's hires
  only.
