# A second-game mission SAVE writes the original's mission file

## Intent

Every second-game mission save point writes the original's `Bsg&` file through the one producer:
the save page, quick save, the timed autosave and the mission start autosave. The file holds the
second game's records with a world half, `CurrentState/InBattle` 1, and the engine's own document
in the store leaf `/CurrentState/AgainromForm`. Its LOAD gives back the state the save point held.
A Token row above 255 (a Human's, a Building's) and a loaded Pointer are written back as read.

## Authority

- Owner, in the brief: each field takes its value from the current state, then a rule from claims,
  then the loaded document's bytes, then a claimed constructor value; a field with none of these is
  a DIV debt row, never a refused save.
- Claims: `R2-SESSION-017` (the `Bsg&` magic), `R2-SESSION-018` (the serializer keeps the first
  game's head, Player list, world byte and trailer and adds one object list), `R2-SESSION-019` (the
  62-byte cell and the 10,374-byte session block), `R2-SESSION-020` (the Player's raw blocks),
  `R2-SESSION-285` (an empty `SpellBook/Shortcuts`), `R2-ENGINE-209` (the Token +0x0c word),
  `R2-ENGINE-603` (Pointer is a Building and one more DWORD).
- The knowledge pin is k395. `R2-SESSION-019`, `R2-SESSION-310` and `R2-SESSION-311` (the key list,
  the entry map's bucket order and +0x3a), `R2-SESSION-313` (register 901) and `R2-ENGINE-760`
  (the diplomacy constructor bytes) are applied.
- The pin was k392 before. `R2-ENGINE-739` (a Human's +0x0c is a word; rows run to 289) and
  `R2-SESSION-297` (missions 73 and 86 place Humans rows 256 and 257) are applied.

## As built

| kind | builder | change |
|---|---|---|
| SAV document producer | `ExportCurrentSave` | unchanged route; `secondCampaignRules.saveForm` forms a mission SAVE as it forms a town SAVE |
| second-game record form | `sav.DocumentInProfile` with `secondForm` | a world half is written in the second game's widths (`worldForm`); `dropClasses` leave out the Diary records (`dropClassRecords`); the form's `pointer` row retypes the Building records the World keeps as Pointers to `Pointer` with `P6C` (`formRetype`, words passed as `ProfileKeeps`) |
| World continuation form | `formWrappers`, wrapper `TKR1` | carries the high byte of an entity's row and, with the index flagged, of a saved structure's row; written only when such a row exists |
| supplement leaf | the leaf table, `/CurrentState/AgainromRows` | Building, Tavern, Shop and Outpost rows ride beside Human rows (`wideRowClasses`) |
| saved structure | `sim.SavedStructure` | `Token0C` is a word (the section keeps its low byte); class `SavedPointer` with `Pointer6C` |
| native city row | `nativeCityDefRow` / `nativeCityToken` | the town head keeps the low byte; `nativeCityHeadRows` writes the row in the record of a town SAVE; a head-only writer refuses (`nativeCityWideRows`) |
| save points | `playerMissionSave` and `SaveDialogSeams`, both through `ExportCurrentSave` | unchanged |

- The world half: the key list holds the cells 0x807 up to 0xedee above 0x0f; the entries go
  bucket by bucket, (key >> 4) mod 100, each with its key at +0x3a (`terrainForm`, a row of
  `secondForm`); result slot 901 is 1 and the diplomacy block opens 1 1 0 0. The other cell
  bytes past the first game's fields, result slots past 100, stances outside 50x50 and each
  cell's first-game trigger tail are written as zero (DIV-4350, DIV-4351, DIV-4352).
- `savtool compare` names the world parts that differ: key list, entries by byte group, entry
  order, terrain key and session parts.
- The records alone open the mission; LOAD reads the engine document from the leaf.
- A World without a wide row or a Pointer encodes to the bytes and hash it had before.

## Proof

- `TestSecondMissionSavePointsWriteTheSecondGameFile` (`pkg/game/secondmissionfile_test.go`): the
  four save points each write a `Bsg&` file; the save page's file LOADs back to the same World.
- `TestSecondMissionSavesAPlacedHumanAboveRow255`: a Humans row 256 is written whole.
- `TestSecondMissionKeepsABuildingRowAbove255` and `TestSecondMissionWritesAPointerBackAsAPointer`
  (`secondmissionbuilding_test.go`): a loaded Building row 330 and a loaded Pointer with its word
  are written back and LOAD again with the same World hash. Both fail on `06a15cbf`.
- `TestSecondTownSaveKeepsAHireRowAbove255` (`secondtownwiderow_test.go`): a town SAVE holds a
  hire's row 300. It fails on `06a15cbf` with the town head refusal.
- `TestStructureRowsWithinAByteKeepTheirForm` (`pkg/sim/structurerowform_test.go`) pins the form
  and hash of a World with a Building row 74, computed on `06a15cbf`.
- `savecheck -resave` over four original files on EN and on RU: no Building row is truncated; the
  Pointer of `rom2-gog-original-2` game9999 stays a Pointer with its word.
- `TestSecondMissionWorldFollowsTheTerrainAndSessionRules` (`pkg/formats/sav/engine_form_test.go`):
  the key list rule, bucket order, +0x3a key, slot 901 and the diplomacy head. It fails on
  `b03aaaf5`.
- `savecheck -resave` over the ten original files that resave, EN and RU (game0010 refuses):
  differing field lines fall by 1 to 3 per file; game0001's key list equals the original's.
- `TestReleaseSecondNewGameMissionTenSaves` writes the engine's own mission 10 file after a new
  game, a walk and a fight; its cold LOAD has the World of the play that saved it.

## Open debt

- DIV-4350: the world half's fields with no holder are zero.
- DIV-4172: the Player's and the Unit's raw blocks and the extra list's records are not written
  back; a resaved Pointer's raw +52 image holds its width, height and blocking bytes.
- A siege hire's Units row above 255 still refuses in a town SAVE.
- DIV-4351: footprint DWORDs, cells the engine's planes leave at 0x0f or below, chain order.
- DIV-4352: registers 999 and 902, +0xa4c and the diplomacy DWORD are zero.
- DIV-4353: the party hero's actor Effect record is dropped in `rom2-gog-original-2` game0012
  and `rom2-owner-kaarg-0051` game0017.
- DIV-4354: a Building's and a Pointer's raw +52 bytes take the live geometry.
- DIV-4355: the record fields with no source (self keys, Token words, Unit raw blocks).
