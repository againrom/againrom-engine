# Story 1257: AGS is dropped (SAV milestone M11)

## Intent

SAV is the only save format. The 114 AGS files in `engine/saves` are converted
once, and after that nothing in the game reads AGS: LOAD, the save list, delete,
automatic save and every shipped tool read SAV only. A state the producer cannot
express is written from its best source and named as debt, never refused.

## Authority

Owner rulings in `pipeline/SAV-COMPLETION.md` (milestone 11) and the seat brief:
the engine is the original in this sense, one producer writes every save point,
AGS is never a fallback. Label policy follows DIV-1339 (install code page).
No ROM1 claim is involved; the AGS format is an Againrom-private record.

## As-built behaviour

### Reader isolation

| Package | Role | Imports `pkg/game` |
|---|---|---|
| `internal/agsenvelope` | AGS header, CRC, length and gob preflight (`Parse`, `Build`, `Preflight`) | no |
| `internal/agsreader` | `Decode`: envelope, gob payload, the two historical field names, then `game.AdoptDecodedAGS` | yes |
| `cmd/saveconvert` | the only production importer of `agsreader` | yes |

`internal/archtest` (`CheckAGSReaderIsolated`) fails when any other production
package imports either reader package. The AGS-reading code is exactly
`internal/agsenvelope` plus `internal/agsreader`; `pkg/game` keeps only the
`Snapshot` type, because `Snapshot` is also the live capture structure, and
`AdoptDecodedAGS`, the validator hook the reader calls after decoding.

`game.EncodeSave`, `DecodeSave`, `SaveStore.Write` and the `.ags` branches of
`SaveStore.List` and of the load seam are removed. A `.ags` file in a save
directory is not listed; loading its name fails with `is not a SAV file`.
Tests that need an AGS fixture use a test-only codec in `pkg/game` and
`cmd/saveconvert` (`agsfixture_test.go`), which production code cannot reach.

### Converter

`saveconvert -in-dir <dir> -out-dir <dir> -manifest <file> -assets <install>`:

1. Decode each `.ags` with `agsreader`, restore it, and write SAV through
   `FrontEnd.ExportCurrentSave` (`ConvertLegacySnapshot`).
2. Read the written bytes back, compare the hash, load them cold on a fresh
   FrontEnd (`LoadObservedSAV`) and compare that state with the AGS-loaded state
   (`CompareConvertedStates`, the comparator the round-trip instrument used).
   Every difference is named in the manifest; differences covered by a
   disclosed row carry its DIV id, any other is `undisclosed` and fails the run.
3. Re-encode the label. An AGS label is UTF-8; a SAV label is install code-page
   bytes (`MigrateLabel`, per rune through the install selector). A rune with no
   code-page byte becomes `?` with a label-debt entry; a non-UTF-8 label keeps
   its bytes with debt; a label past 255 bytes is cut with debt.
4. Refuse an output under `gameversions/en`, `ru`, `rom1-demo` or `rom2-ru`,
   an existing manifest, an existing file, and any path the install fence
   covers. A clashing name takes a `-from-ags` suffix.

### List display (owner direction)

The owner directed that save lists show no file extension, because SAV is the
only format. A SAVE dialog row shows the disk name without its one trailing
`.sav` (`saveListName`); a LOAD row already shows only the label. File names on
disk, selection, Delete, overwrite confirmation and the label rule are
unchanged. Choosing a SAVE row fills the name field with the same shortened
name, except when the remainder still ends in `.sav` or `.ags`: the writer
removes one trailing suffix from a typed name, so such a row fills its full disk
name and the next SAVE targets the selected file. Delete accepts `.sav` only.
`TestSaveAndLoadListsShowNoExtension` covers both lists.

### Retired instruments and tools

| Retired | Replacement |
|---|---|
| AGS round-trip corpus test and its refusal baseline | `TestSAVConvertedCorpusContinuation` (tag `sessioncorpusaudit`): identity by sha256, cold LOAD, resave, reload, no loss admitted, over the converted corpus and its manifest |
| `check-milestone2-acceptance.sh` AGS corpus legs | the same script runs the converted-corpus instrument per root (`review/story1257-ags-migration/out-<root>`, or `AGAINROM_CONVERTED_CORPUS`) |
| `cmd/savemigrate`, `cmd/saverepair` | none: both reported on the AGS byte form; debt in DIV-1681 |
| `SaveAGS` and `SaveBoth` dialog formats | removed; `SaveSAV` only |

## Proof

Counts and receipts are under `review/story1257-ags-migration/`:
`inventory.sha256` (114 files, 21,250,202 bytes, unchanged after the runs),
`convert-en.log`, `convert-ru.log`, `out-en/conversion-manifest.json`,
`out-ru/conversion-manifest.json`, `m2-acceptance.log`, `release-tests-en.log`,
`release-tests-ru.log`.

| Root | Discovered | Converted | Verified | Debt | Undisclosed | Failed | Labels re-encoded |
|---|---|---|---|---|---|---|---|
| EN | 114 | 114 | 114 | 0 | 0 | 0 | 1 |
| RU | 114 | 114 | 114 | 0 | 0 | 0 | 1 |

The re-encoded label is the one UTF-8 label (`пgooo!`, one Cyrillic letter,
stored as the single code-page byte 0xef). Two outputs were renamed
(`mytown-from-ags.sav`, `save-from-ags.sav`) because `engine/saves` already
holds `.sav` files of those names.

Focused tests: `TestMigrateLabelThroughTheInstallCodePage`,
`TestConvertLegacySnapshotWritesTheMigratedLabel`,
`TestProductionLoadRefusesAnAGSName`, `TestSaveStoreListsLocalSAVAndIgnoresAGS`,
the `internal/archtest` isolation rule and its two tests, and the batch tests in
`cmd/saveconvert`. The converted files are not placed under `gameversions/saves`;
the seat decides their location.

## Open debt

- DIV-1679 to DIV-1682 in `docs/divergences/persistence-current-sav.md`.
- `sim.UpgradeSaveForm`, the older-form restore paths and the Snapshot gob wire
  shape remain as converter-only code until the owner confirms the converted
  corpus is final.
- `pipeline/sav-export-census.sh` still enumerates `engine/saves/*.ags`; it is a
  seat script and is not changed here.
- Original-runtime acceptance of a converted file is unproved (DIV-1369).
