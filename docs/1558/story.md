# One SAV saved form per game and document kind

## Intent

A SAVE's saved form is one row per (game, document kind): the records the game's original reads,
built from the engine's own document. The producer applies the row. No SAV the engine writes and
no state a LOAD gives back changes, on either game, town or mission.

## Authority

- Owner: a third of architecture audit `bdac396d`'s multi-way kinds become one builder each; this
  story is its rank 3, the SAV saved form per game and document kind. No player change.
- Owner: SAV is the only save format and has one producer (`AGENTS.md`).
- DIV-4304 (the engine document inside a second-game file), DIV-4350 (the mission file's world
  half) and DIV-3079 (the load and save page list) bound the second game's form; this story moves
  no field.

## As built

| kind | builder | change |
|---|---|---|
| SAV saved form | `savedForm` rows, `firstSavedForms` and `secondSavedForms` (`pkg/game/savform.go`), applied by `savedForm.write` | new: one row per (game, document kind) |
| SAV document producer | `ExportCurrentSave` | asks the game's `saveForm`, which reads the row; stores the engine leaf where the row says so |
| SAV document encoder | `sav.EncodeDocumentData`; the engine leaf `sav.StoreEngineDocument` and its reader `sav.EngineDocument` | unchanged |
| second-game record holders and tail | `writeSecondTownHolders` over the holder rows `secondFormHolders`; the tail writer `secondCampaignTail` | the holders are rows {class, field, site, source}; the tail writer is the row's |
| SAV LOAD | `restoreOriginal` and `decodeSavedDocument` | unchanged |

A row holds:

- `records`: the SAV profile whose record programmes and form rows the records take
  (`sav.DocumentInProfile`); `sav.FirstGame` writes the engine's own records, the identity form.
- `holders`: the fields the profile's records hold that the engine's records keep elsewhere, each
  with its site and source, written in order before the form rows.
- `keeps`: what the records hold that the World keeps (each Pointer's word).
- `tail`: the campaign tail writer. Its loaded-tail source is `secondTailKept`, unchanged.
- `leaf`: the engine's own document travels in the AgainromForm leaf.

| game | kind | records | holders | keeps | tail | leaf |
|---|---|---|---|---|---|---|
| first | mission | `sav.FirstGame` | none | none | none | no |
| first | town | `sav.FirstGame` | none | none | none | no |
| second | mission | `sav.SecondGame` | `secondFormHolders` | `secondFormKeeps` | `secondCampaignTail` | yes |
| second | town | `sav.SecondGame` | `secondFormHolders` | none | `secondCampaignTail` | yes |

- Before: `firstCampaignRules.saveForm` answered the engine's records; `secondCampaignRules.saveForm`
  called `secondTownForm` for both kinds, which ran the holder code, `secondFormKeeps`,
  `sav.DocumentInProfile` and `secondTownTail` in sequence.
- After: each `saveForm` is one line, `<game>SavedForms[savedFormKind(s)].write(f, s, doc)`.
  `secondTownForm` is gone; `secondTownTail` is `secondCampaignTail`.
- A town SAVE's row reads nothing from a World: `exportable` refuses a second-game town snapshot
  that holds one, so `secondFormKeeps` answered nothing for a town before.
- `savedForm.write` refuses a row whose records and leaf disagree, and a missing row.
- Guard: `internal/archtest/savedform_test.go` fails on a declared `secondTownForm`, a `saveForm`
  that is not one return of a saved form row, and a reference to `DocumentInProfile`,
  `writeSecondTownHolders`, `StoreEngineDocument`, `saveForm`, `secondCampaignTail`,
  `secondFormKeeps` or `secondFormHolders` outside its owner. Its self-test names a planted bypass
  of each shape.

## Proof

- Unchanged and passing: `TestEngineTownDocumentWritesTheSecondGameForm`,
  `TestSaveFormKeepsTheEngineRecords`, `TestEngineDocumentRefusesALeafThatDoesNotValidate`, the
  second-game mission form tests (`TestSecondMission*`), `TestCampaignRecordRoutesWriteTheBaseBytes`;
  `go test -count=1 ./pkg/game ./pkg/formats/sav` passes.
- `TestEachGameAndDocumentKindTakesItsSavedFormRow` (`pkg/game/savform_test.go`): for each of the
  four (game, kind) fixtures, deleting that row refuses that SAVE and no other; a marked tail writer
  in one second-game row reaches only that kind's file. `TestSavedFormRecordsAndLeafAgree`: a row
  whose records and leaf disagree, and an empty row, refuse.
- Guard: `TestEverySaveTakesItsSavedFormFromItsRow` finds the two row tables and no bypass;
  `TestSavedFormScanNamesEachBypass` names six planted bypasses.
- Bytes: `savecheck -saves <fresh directory> -party -resave load`, LOAD and SAVE with no play, base
  `3f54727c4` against the candidate, each file copied first and checked against its folder's
  manifest, one install root at a time.

  | population | root | files | resaved, bytes equal | refused by both, same message |
  |---|---|---|---|---|
  | first game, the round-trip gate's files | EN | 128 | 128 (84 mission, 44 town) | 0 |
  | first game, the round-trip gate's files | RU | 128 | 128 (84 mission, 44 town) | 0 |
  | second game (`Bsg&`), story 1554's 22 owner files among them | EN | 48 | 44 (15 mission, 29 town) | 4 |
  | second game (`Bsg&`), story 1554's 22 owner files among them | RU | 48 | 44 (15 mission, 29 town) | 4 |

  The refusals: `rom2-gog-original-2` game0010 and `rom2-gog-original-3` game0026 and game0029
  ("saving is not available for this game yet"); EXP-0261's 1850-byte game9000 ("no saves in").
  No file differs.
- Release witnesses the next release runs on EN and RU: the SAV round-trip gate
  (`scripts/check-sav-roundtrip-gate.sh`), `TestReleaseOwnerSavesWriteOneByteSequence`,
  `TestReleaseSecondOriginalTownLoadFollowsTheFile`, `TestReleaseSecondNewGameMissionTenSaves`,
  `TestReleaseSecondGameSaveKeepsOrderThresholdBytes` and the milestone-2 acceptance
  (`scripts/check-milestone2-acceptance.sh`).

## Open debt

- None new. DIV-4304, DIV-4305, DIV-4306, DIV-4350 to DIV-4355 stand as they were.
