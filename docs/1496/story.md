# One layer schema for every mod data kind

## Intent

Every kind of data a mod carries is declared once, in one list, with its file names, its record
key, its reader and validator, whether it is a catalog or configuration, and how the records of
several mods combine. Every route that reads mod data walks that list: the script's
`game.data.add`, the mod check, the launch, the folder and packed forms, the save's mod mark and
the base dump. One apply step in the game replaces the per-kind apply calls of `cmd/againrom`.
Catalog kinds whose layer 0 comes from one base decoder, or is empty, combine through `mod.Merge`
with the towns override rule. Every mod that worked before reads and plays as before.

## Authority

- Owner ruling "Mods are patches of the base game" (`pipeline/OWNER-RULINGS.md`): each kind of game
  data is one catalog; layer 0 is filled by the base's decoder; each mod is a layer over it,
  keyed by the record key; with `load-after` the later mod wins and the check names the override;
  without an order the launch refuses.
- Owner decision on audit `pipeline/reviews/arch-audit-0e842efb.md`, item 2: one layer schema for
  every mod data kind, a story after 1495 and before T6 (`pipeline/BACKLOG.md`). It replaces the
  earlier decision that T6 to T9 each move their own kind.
- Towns are the model (`docs/1486/story.md`); the consumers T6 to T9 own are not migrated here.

## As built

### The declarations

`pkg/mod/kinds.go` holds `DataKind` and the list `dataKinds`, in apply order. A kind declares:

- `Name`, `Files` (the first is the name to use; weapon bodies keep the folder-only long name as a
  second name, and a mod holding both is refused when its manifest loads), `Key`;
- `Class` (`CatalogData` or `ConfigurationData`) and `Base`: layer 0 of a catalog kind, or why a
  configuration kind has no base catalog;
- `Layer`: `LayerMerged` (patches through `mod.Merge`), `LayerKept` (the kind keeps its own rules;
  `Owner` names the story that migrates it) or `LayerOwn` (configuration: each mod's own);
- `Shared`, the rule for two mods on one record, as `docs/MODS.md` prints it;
- `Dump`, whether the base dump writes its layer 0; `Script`, whether `game.data.add` loads it;
- an unexported reader (file, line, field and reason on every refusal) and a summary line for the
  mod check.

`pkg/mod/kindread.go` holds the readers, moved from `pkg/modrt`. `mod.Data` holds the records of
every kind; `modrt.Result` embeds it, so `res.Items` and the other fields read as before.

### The kinds

| kind | class | layer | dump | why |
|---|---|---|---|---|
| items | catalog | kept, T9 | no | layer 0 is the item rows of the definition table and new codes take free rows; T9 adds rows and materials and owns the item catalog |
| characters | catalog | kept, T6 | no | the people are read three ways (definition table, `npc.reg`, editor sections); T6 builds the one character catalog |
| screens | configuration | own | no | a screen or menu action is an addition of its mod; the base has none to change. Applied by the window (`app.SetModScreens`), which is presentation outside the front end |
| companions | catalog | merged | no | layer 0 is empty: the base grants every town companion on arrival |
| spells | catalog | kept, unscheduled | no | the spell rows, names, rules and formulas come from several derived reads; no one list yet |
| body sheets | catalog | merged | no | layer 0 is empty: a shipped body is a registry class, and a sheet may not take a shipped name |
| weapon bodies | catalog | merged | no | layer 0 is the shipped body list, one decoder |
| missions | catalog | kept, T6 | no | the campaign's missions come from the campaign decoder and its loads; T6 makes missions a layered catalog |
| towns | catalog | merged | yes | unchanged from `docs/1486/story.md` |
| settings | configuration | own | no | a setting is a knob of its mod, read with the manifest before the script, kept in the mark |

### One route

- `pkg/modrt/data.go`: `game.data.add(path)` finds the kind by `mod.ScriptDataKind` and runs its
  reader through `mod.ReadInside`, which serves the folder and the packed RES alike; an unknown
  path is refused listing `mod.ScriptDataFiles()`. Adding a file still marks the mod as changing
  game state, so the save's mod mark is unchanged.
- `againrom -check` prints one summary per declared kind from the declarations, in today's words.
- `pkg/game/moddata.go`: `FrontEnd.SetModData(mod.Data)` walks the declarations and runs each
  kind's apply step; it returns the overrides to print and the mission warnings. `cmd/againrom`
  calls it once where it called `SetModItems`, `SetModCharacters`, `SetModCompanions`,
  `SetModSpells`, `SetModBodies`, `SetModMissions` and `SetModTowns`. The output order (warnings,
  refusal with exit 2, overrides) is the old one.
- `pkg/game/basedump.go`: the dump writes the kinds whose declaration sets `Dump`, each by its
  first file name; `baseDumpLayers` is keyed by kind name and a test holds it to the declarations.

### The merged kinds

- Companions: `[[join]]` keyed by `chapter` and `companion`. A new key needs `key`, `building` and
  `talk`; a later mod's table changes the keys it gives; `off = true` stands alone.
  `mod.MergeCompanions` runs `Merge` over an empty layer 0; `game.SetModCompanions` then runs the
  campaign checks of `DIV-1912` on the merged list. `DIV-3909`.
- Body sheets: `[[body]]` keyed by `name`; a later sheet replaces the whole sheet, `off = true`
  with `name` alone switches one off. `mod.MergeBodySheets`. `DIV-3910`.
- Weapon bodies: keyed by the weapon row over the shipped body list, the one field `body`;
  `game.layerModBodies` merges them after the sheets and before the shield forms. A row is never
  switched off. One mod naming one row twice keeps its old refusal. `DIV-3911`.
- Towns: unchanged; the declaration names them.

Two mods on one record of a merged kind: with `load-after` the later wins and the check prints
`<kind> <key> <field>: mod "b" (<file>:<line>) overrides mod "a"`; without an order the launch is
refused naming both mods, the file and the line. Before this story companions, body sheets and
weapon bodies refused every second mod on one record; a set that worked before has no such pair
and reads the same.

### Outside the declarations

- The files overlay (`files/`) shadows install files by path. It is a resource boundary, not a
  record catalog, and keeps its own route (`pkg/game/modfiles.go` and `pkg/game/installshare.go`, T5).
- The screens apply step stays on the window application; the declaration's apply step is empty.
- `settings.toml` is read by `mod.LoadSettings` with the manifest; its declaration validates the
  file and gives its row in `docs/MODS.md`.

## Proof

- `pkg/mod/kinds_test.go`: `TestEveryDataKindIsDeclaredWhole` walks the declarations: each first
  file name has segments of at most 15 bytes, no file belongs to two kinds, and each kind has a
  key, a reader, layer 0, a rule for two mods and a fixture mod under `pkg/modrt/testdata/mods`
  that carries its file. `TestModsDocListsTheDeclaredKinds` holds the `docs/MODS.md` Data kinds
  table to the declarations, row by row.
- `pkg/modrt/mergeddata_test.go` `TestFixtureModsGiveTheRecordedMergedData`: every fixture mod
  alone, as a folder and packed, the completed `east-town` and `reniesta`, and all fixture mods
  together, on `rom1-en` and `rom1-ru`. Each line is the SHA-256 of one kind's canonical records,
  the rules, the mod mark, or the refusal. `pkg/modrt/testdata/merged-data.txt` (144 lines) was
  written by this test run on the base source `ffeabd2f`; the fields the schema adds (`Fields`,
  `Off`, `Order`, `OffLine`) are left out of the hash. The test also checks that
  `MergeCompanions` and `MergeBodySheets` return the read records when no two mods share a key.
  Loss control: with the companions declaration removed, this test and
  `TestExampleModHoldsTheTownCompanionUntilTheTalk` fail; the file was restored from the commit.
- Override tests, each with load-after (later wins, override named), no order (refused) and `off`
  where the kind has it: `pkg/mod/kindmerge_test.go` (companions, body sheets),
  `pkg/game/moddata_test.go` (`TestSetModDataMergesCompanionLayers`,
  `TestWeaponBodyLayersFollowLoadAfter`), `pkg/modrt` `TestTwoModsOnOneCompanionNeedLoadAfter` and
  `TestTwoModsOnOneBodyNeedLoadAfter`. `TestModDataStepsCoverTheDeclaredKinds` holds the apply
  steps and the dump to the declarations.
- Headless: the six mods of `builds/current/mods` (archer-girl, heavy-armor, mission-abandon,
  reniesta, reniesta-joins-on-talk, skill-cap), built from `ffeabd2f` and from this story, a new
  game into missions 10 and 30 for 3000 ticks, then a SAVE, on EN and RU. Every step's output is
  byte-equal; the written SAVs are byte-equal except the wall-clock time of the `map_load` leaf.
  The front-end stage prints no World hash and the mission stage takes no mods, so the SAV, which
  holds the World, is the compared form.
- The EN and RU release sets; no frame, World hash or SAV byte moves.

## Open debt

- Items: kept until T9; no dump. Two changes of one shipped row both apply, unnamed.
- Characters: kept until T6 (one character catalog); no dump. Two edits of one target apply in
  load order, unnamed.
- Missions: kept until T6 (missions as a layered catalog); no dump.
- Spells: kept; no story owns the spell catalog yet. One field of one spell per mod set.
- Companions, body sheets, weapon bodies: merged, no dump (T4 dumps items first; these have an
  empty layer 0 except weapon bodies, whose layer 0 is the shipped body list).
- Weapon bodies cannot switch a row off: a later mod cannot restore a row an earlier mod changed.
- Screens and settings are configuration and take no layer.
