# 1204 -- story numbers purged from the engine's own names

## Intent

The owner found story numbers inside shipped identifiers and demanded three
things: fix every one now, cut the reverse-engineering narrative out of code
comments, and install a gate so neither comes back. This story does the
first fully, the gate fully, and the comment cut only for symbols it renamed;
the remaining comment volume is named as open debt, not attempted.

## As-built behaviour

### Identifiers (production code)

Every story-numbered identifier in non-test code is renamed, AST-aware
(`go/parser` + `go/ast` + `go/format`, not `sed`), across 69 files including
every test-file reference to them. Zero remain outside the one exception list
below: the guard measures `NonTestIdents: 0`.

Counted on a clean checkout of the base commit with this pass's own `go/ast`
walker (top-level and local declarations plus struct field names, the
`notAStoryNumber` entries excluded): **81 story-numbered declarations in 24
non-test files**. The adversarial pass's independently written walker read 82
in 25 files under its own definition of a declaration. Neither is 84, which is
what this document claimed before the review. Adding the three out-of-range
`effectTan` angle constants makes 84 or 85 renamed declarations depending on
the same definition.

The rename rule is not mostly "drop the number". **31 of the 81 map to a bare
number-strip** (34 of 82 by the review's walker); the other 50 changed the
stem. The three families below cover 23 of those 50; the remaining 27 are
one-off improvements in the same spirit -- `citySnapshot1173` to
`citySnapshotFromMission`, `savLabelRefusal1173` to `encodeSaveLabel`,
`projectilePath1162` to `foldStatePath` (the function ASCII-folds a state path
and has nothing to do with projectiles), `admitWorld1170Effects` to
`admitEffects`, `generatedWorld1171` to `generatedDocumentBuilder`, and two
dozen more.

| Family | Old shape | New shape | Why |
|---|---|---|---|
| `construct1171*`, 11 declarations | `construct1171Value`, `construct1171Record`, ... | `mustSetValue`, `mustNewRecord`, ... | panic-on-invariant-violation setters; `must*` names what they do everywhere else in this codebase. Two are not `must*`: `construct1171Item`/`construct1171Position` became `constructedGeneratedItem`/`constructedPositionBlock`, which build a value rather than assert one |
| `*Application1170*`, 11 declarations | `snapshotApplication1170`, `restoreApplication1170` | `captureApplicationState`, `restoreApplicationState` | "Application" alone read as an act; the SAV/session snapshot type is a *state*, so every derivative gained the suffix. One of the 11 is the wire field name, kept |
| `effectTan1125/3375/5625/7875`, 4 renames | a bare number that coincidentally read as a story id | `effectTan11Deg25/33Deg75/56Deg25/78Deg75` | spells out the tangent-boundary angle the constant actually is, instead of a number. Only `effectTan1125` falls in the guard's range, so only it is one of the 81 |

One rename was forced by a collision, not by clarity: `namedSaveBase1173`
(a type) would have collided with the pre-existing function
`namedSaveBase(name string) (namedSaveBase1173, error)` in the same file if
simply stripped, so the type became `validatedSaveName`.

Three verified false positives from the seat census's own regex, none renamed:
`x1000`, `x1001`, `x1100` in `pkg/video/audioprobe_windows_386.go` /
`native_windows_386.go` and `pkg/sim/effectwitness.go` are substrings of hex
literals (`0x1000` etc.), never Go identifiers -- `go/ast` never sees them as
`*ast.Ident`.

Two families are documented exceptions in `internal/storyguard`'s
`notAStoryNumber`, not renames:

1. `decodeCP1251`, `encodeCP1251`, `Windows1251` (`pkg/formats/alm/alm.go`,
   `encode.go`; `pkg/formats/textinput/input.go`) name the Windows-1251
   Cyrillic code page ALM/BIN text actually uses. 1251 is a code page number.
2. `Application1170`, `GameOptions1186`, `LocalOnly1186` are gob wire-format
   field names, spelled only in `pkg/game/savehistoricalwire.go` and its test.
   The next section is why.

### Saved games keep their names

`Snapshot` is persisted with `encoding/gob`, **which matches struct fields by
name and silently ignores a field present in the stream that the receiving
struct does not declare.** Three of the renamed identifiers were exported
fields of that struct graph:

| old wire name | current Go name |
|---|---|
| `Snapshot.Application1170` | `Snapshot.ApplicationState` |
| `SnapshotResidue.GameOptions1186` | `SnapshotResidue.PendingGameOptions` |
| `SnapshotApplicationState.LocalOnly1186` | `.LocalOnly` |

Renaming a gob **type** is free -- it changes what a fresh encode writes and
nothing about what a decode reads. Renaming a gob **field** is a format break:
every save already written carries the old name, so the value is dropped with
no error and, once saved again, is gone for good. Measured before this
correction: **20 of the owner's 114 `engine/saves/*.ags` lost their mission UI
record** -- saved selection, camera, open panels, health bars, flying damage
numbers, time flow, game speed and retreat baseline -- and the EN/RU release
gate failed on `TestReleasePortraitInputHotfix/mission111-hover-and-tab`
because the saved camera rides in that record and the hero opened off screen.
The three committed envelope fixtures could not have caught it: all three
predate these fields and name none of them.

The fix keeps the clean names on the live structs and adds one explicit legacy
decode path, `pkg/game/savehistoricalwire.go`. Its doc says what it is: a
record of the wire as it was already written, not engine vocabulary. It
declares the three historical field names in types nothing else in the engine
names, decodes the same payload a second time on its own `gob.Decoder`, and
adopts the two values into the live `Snapshot`. The alternative -- leaving the
three field names unrenamed -- was rejected because it puts the numbers back
into `Snapshot` and `SnapshotApplicationState`, the two structs every
persistence reader opens, instead of into one file whose subject is the
historical wire.

The second decode runs only when the current name produced nothing **and** the
historical name is present in the payload bytes. gob writes a field name as an
uncompressed length-prefixed ASCII string in the struct's type descriptor, so
a stream that declares the field contains those bytes and a stream that does
not cannot; the test can only over-trigger, and an over-trigger costs one
decode that adopts nothing. An ordinary current save pays two `bytes.Contains`
and no second decode.

**Direction.** This build reads both spellings and writes only the current
one. A save written here and read by a build older than this file loses the
same two values this file recovers. The envelope version does not move for it:
both fields are additive and both already have a defined absent meaning, so an
older build refuses nothing and reports nothing -- it reads the save as one
written before the fields existed. The seat keeps one rollback build; this is
the bound on what a rollback costs.

**The audit behind "three".** Not taken from the review. A `reflect` walker
enumerated every exported field path reachable from `game.Snapshot` in both
trees -- 1,806 paths each -- and diffed them. Exactly three field names
changed; every other old-only path is a child of one of the three. Nothing
else in the persisted graph moved.

**The regression test.** `pkg/game/savehistoricalwire_test.go` declares the
historical field names a second time, independently of production, frames a
gob payload through the real envelope header, and asserts `DecodeSave`
recovers the application record, its `LocalOnly` flag and the pending game
options. Disabling the adoption path makes two of its three tests fail with
"application record was dropped". A third test asserts the write side did not
move: a fresh `EncodeSave` contains neither historical name and round-trips.

### File names (production code)

All 25 production `.go` files whose *name* carried a story number are
`git mv`-ed to a name that states what the file holds, colliding with nothing:
`cityholdings1172.go` to `cityholdings.go`, `worldsaveprojectiles1170.go` to
`worldsaveprojectiles.go`, `savitemobjects1115_scroll.go` to
`savitemobjects_scroll.go`, and 22 more of the same shape. The paired
`_test.go` files are untouched -- Go does not require a test file's name to
match its package's production file, and the 272 numbered test files are this
story's declared open debt.

### Living ledgers

A renamed symbol or path cited in a living document is a wrong document. Every
`.md` outside `docs/<NNNN>/` was swept for identifiers that exist at the base
commit and nowhere in this tree, and for the 25 old file paths. Corrected:

- `docs/divergences/persistence-current-sav.md` (DIV-1230, DIV-1334, DIV-1339,
  DIV-1344): `savLabelRefusal1173`, `playerMissionSave1198`,
  `missionBattlefieldFallbackNotice1198`, `citySaveNotice1194`,
  `playerCitySave1173`, `citySnapshot1173`, `projectWorld1170Dead`,
  `Application1170.LocalOnly1186`, and the paths `savedialog1173.go`,
  `missionsave1198.go`, `worldsave1170.go`.
- `docs/divergences/persistence-city-sav-and-imports.md`:
  `importSavedActorActions1171`.
- `docs/HOTFIXES.md` rows `2752c5e` and `3efa04f`: `applicationCurrentRaw1170`,
  `restoreApplication1170`, `initialiseGeneratedApplication1171`,
  `localGameOptionView1186`, `Application1170`, `LocalOnly1186`, and
  `worldsave1170.go`. The camera row's `worldsave1170.go:60` pointer is
  replaced by the function name, `currentWorldDocument`, which does not go
  stale on the next edit above it.

`docs/hotfix/LEDGER.md` still cites `pkg/game/savitemobjects1115.go` in row
`f1053f5` and is deliberately left alone: its own header calls it "the only
record that they happened", each row names the commit it describes, and the
path was correct in that commit's tree. It is a frozen record, not a living
document.

### The guard: `internal/storyguard`

Modeled on `internal/archtest`'s own split of a pure evaluator (`Check`) a
synthetic-graph test proves, from a tree walker (`Scan`) a live-tree test
runs. `TestLiveTreeClean` fails the build the same way
`archtest.TestLiveTreeClean` does.

It measures four things over every `.go` file: identifiers whose name carries
a 4-digit run in `[1000, 2000]`, `.go` base names with the same run, six
forbidden comment forms matched inside `go/ast` comment text only (retired
spec clauses, `story NNNN` mentions, calendar dates, raw ROM1 addresses,
`FUN_xxxxxxxx` names, `EXP-NNNN` citations), and total comment bytes. Non-test
identifiers and file names must be zero; the rest are committed counts that
must equal the tree exactly, so drift either way fails. A bare provenance id
such as `DIV-1359` or `SAV-1011` matches nothing.

**What the guard does not read**, all confirmed by construction on a copy
during the adversarial pass: directory names (`pkg/game/legacy1172/` with
`package legacy` scans clean), non-`.go` file names (`testdata/fixture1172.bin`,
`scripts/check-story1204.sh`), struct tags and string literals
(`json:"field1172"`, `const S = "written by story 1204"`). `AGENTS.md` now
states that boundary instead of implying the guard covers every name.

**The exception list is the one door, so it has a lock.** `notAStoryNumber` is
applied to identifiers in test and non-test files alike, keyed by the whole
identifier name. Appending one line to it turns a live production leak into
`CHECK: clean`, which the adversarial pass demonstrated. `TestNotAStoryNumber`
now asserts the map's exact contents, so growing it fails the build until the
same commit says what was added and why; `TestScanAppliesTheExceptionList`
scans a temp directory and proves the exception is applied in the walker and
is keyed by the name rather than by the digit run.

**Range choice**, restated on evidence that exists. `1000..2000` is a bound on
this project's own numbering (`docs/<NNNN>` runs 1058-1204) with room past a
doubling, not "any 4 digits". The earlier rationale cited
`effectTan3375/5625/7875` as the unrelated 4-digit values the ceiling
excluded; this same story renamed those constants, so that evidence no longer
exists. Measured over every `.go` file in the module, the only 4-digit
identifier run outside the range is `0002`, in
`TestReleaseGame0002PotionRetainsItsSavedEffect` -- a game content id the
floor keeps out. The exactly-4 rule carries the rest of the separation, and
`TestDigitRun` fixes both boundaries.

**Proof each check discriminates.** `TestCheckDiscriminates` drives `Check`
with synthetic `Report` values, the same route
`archtest.TestCheckNamesOffendingEdge` uses rather than a transient live-tree
edit: a clean report passes; one non-test identifier fails and the message
names the offender and its one exception list; one non-test file name fails,
names the file, and says file names have no allowlist; a ratcheted count
rising is reported as a regression; the same count falling names the exact
number to commit; a forbidden comment form rising names the form; comment
bytes rising and falling get their two distinct messages.

**Comment bytes may rise for new code, and only by saying so.** The ratchet's
original instruction was "may only fall", which no new file with a doc comment
can satisfy -- this correction pass is the first to hit it. `baseline.go` now
states the one exception, and the rise message names the number to commit
instead of only forbidding the edit. The adversarial pass's related warning is
why `measure`'s output was read field by field rather than pasted: seven of
the eight numbers are byte-identical to the previous baseline
(`TestIdentCount` 6072, `TestFileCount` 272, and all six comment forms), and
only `CommentBytes` moved, 9,467,172 to 9,473,680, which is this pass's own
new file and test.

### AGENTS.md (this worktree)

Three fixes:

1. A "Naming and comments" section: no story or experiment number in an
   identifier, file name, or comment; exactly which of those
   `internal/storyguard` enforces and which it does not; the one exception
   list and the test that pins it; what a comment is for versus what is not
   its job; a proposed 12-line ceiling per comment group.
2. The rule this story's own defect earned: an exported field name on a
   gob-persisted struct is a wire-format identifier, renaming the type is
   free, renaming the field is a format break, and a frozen fixture written
   before the field existed cannot discriminate the change.
3. The divergence-routing defect story1203's review found: a new row goes into
   the subsystem file under `docs/divergences/` its own Subsystem cell names,
   not into `docs/DIVERGENCES.md`, which is the index and row-format spec.
   Every instrument (`check-div-claims.sh`, `internal/divledger`,
   `cmd/divcensus`) reads `docs/divergences/*.md` directly.

### Comment ceiling: measured distribution and proposal

Tree-wide `go/ast` comment-group line spans (span = last line minus first line
plus one), 29,071 groups: median 3, 75th percentile 6, 90th 12, 95th 17, 99th
34, worst case 908 lines (`pkg/sim/binary.go:97`). Twelve lines is the natural
cut at p90: it asks nothing of 90% of the tree's comments and bounds the rest,
which run 15-80x longer for no reason a reader needs. 2,582 groups (8.9%)
exceed it today and are named as debt, not fixed. An independently written
walker in the adversarial pass read 29,052 groups and 2,580 over twelve on the
pre-correction tree -- a 0.07% difference in what counts as a group, which
moves no percentile.

### Comments on renamed symbols

Five files had a doc comment carrying cuttable narrative and were trimmed to
what the function does plus, where provenance mattered, a bare id:
`citysavenotice.go`, `missionsave.go`, `savedialog.go`, `worldsave.go`,
`generatedworld.go`. Every load-bearing constraint survives -- the
`rawOffered`-before-normalisation trap, the both-branches-encode-first rule,
the `MapUnitID` 0 corpse guard with `DIV-997`/`DIV-1344`, `TEXT-SAVELABEL-057`,
`DIV-019`. What was cut is review-record and story pointers. The remaining
declaration sites carried no comment or one already free of the forbidden
forms.

### User-visible strings

One real leak, unrelated to any numbered identifier.
`pkg/game/townscreen.go`'s shop and school rows said `"(trade is a later
story)"` and `"training is a later story - ..."` -- development-roadmap
language shown to the player. Fixed to `"(trade is not yet available)"` and
`"training is not yet available - ..."`. No test, fixture or baseline asserts
the old text. These two rows carry no localisation table, so EN and RU both
show the English string; that is pre-existing and not this story's doing.

The `DIV-1319`/`DIV-1320`/`DIV-1334` citations inside
`cityExportNotice`/`missionBattlefieldFallbackNotice`'s player-facing "SAV
approximated" disclosures are a distinct, pre-existing, deliberate pattern
documented in `docs/DIVERGENCES.md`, not a story-number leak.

### Non-rename edits, named

Hashing every `.go` file's token stream with identifiers normalised and
comments dropped, on both trees, leaves exactly **three** files differing
beyond renames, comment cuts, file moves and the new package:

1. `pkg/game/townscreen.go`, the two string literals above.
2. `pkg/game/save_test.go`, three `sha256` literals (below).
3. `pkg/game/savlabel1198_test.go`, whose `t.Fatalf` message strings moved
   with the function they name (`savLabelRefusal1173(%q, %d)` to
   `encodeSaveLabel(%q, %d)`).

## Proof

### The save corpus, before and after

All 114 `engine/saves/*.ags`, decoded through `game.DecodeSave` and passed to
`(&game.FrontEnd{}).ExportCurrentWorldSave`, by an external module with
`replace againrom => <tree>` built from each commit in turn. Nothing was
written into `engine/saves` or any install.

| | base `6a26c1c` | before this pass | after this pass |
|---|---|---|---|
| saves with an application record | 20 (15 `LocalOnly`, 5 full) | **0** | 20 (15 `LocalOnly`, 5 full) |
| decode errors | 0 | 0 | 0 |
| `never-imported worlds have no source-backed constructors` | 79 | 94 | 79 |
| `local UI settings have no admitted original application record` | 15 | 0 | 15 |
| `one mission World is required` | 15 | 15 | 15 |
| `current Group graph requires lifecycle support` | 4 | 4 | 4 |
| `current camera is not an integer-cell origin at original zoom` | 1 | 0 | 1 |
| `current application state is absent` | 0 | 1 | 0 |

The after-run's full output, including the sorted list of the 20 files
carrying a record, is byte-identical to the base-commit run.

### The three `sha256` literals

All three fresh-encode hashes in
`TestReleasedEnvelopeAtTheCurrentSimulationFormFixture` move:
`current72`'s expected hash, `currentReleasedSaveFixtureSHA256`, and
`old86`'s expected hash. The other five compare frozen historical byte blobs
that are never re-encoded and are unchanged; the blobs still decode into the
current types.

The cause was measured rather than argued: a zero `Snapshot` was gob-encoded
on each tree and every ASCII name run in the payload extracted and diffed.
Five names changed, and each of the three hashes moves for both reasons at
once, because one fresh encode transmits the whole type graph:

| changed name | kind | effect |
|---|---|---|
| `Application1170` to `ApplicationState` | field | fresh bytes **and** what a decode reads |
| `GameOptions1186` to `PendingGameOptions` | field | fresh bytes **and** what a decode reads |
| `LocalOnly1186` to `LocalOnly` | field | fresh bytes **and** what a decode reads |
| `SnapshotApplication1170` to `SnapshotApplicationState` | type | fresh bytes only |
| `PendingGameOption1186` to `PendingGameOption` | type | fresh bytes only |

A zero `Snapshot`'s payload goes from 14,498 to 14,491 bytes. The read half of
the field rows is what `savehistoricalwire.go` restores; the write half is
not restored and is not meant to be, so the three committed hashes are the
post-rename values and the test passes on them unchanged.

### Gates

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...`: every package `ok`, including
  `internal/storyguard` and the three new `pkg/game` tests. Pre-existing
  `go vet` warnings (`BookSpell` unkeyed struct literals in five `pkg/game`
  test files) are outside this story's touched-file set.
- `bash scripts/check-no-game-assets.sh`: clean.
- `bash pipeline/check-div-claims.sh`: clean, after the ledger sweep.
- `bash pipeline/check-release-tests.sh <EN> <RU>`: `ok (320 of 320 ran, 0
  lacked a subject)` on each root, exit 0.
  `TestReleasePortraitInputHotfix/mission111-hover-and-tab`, which failed
  before this pass with `headless select entity 81: entity is outside the
  current view`, is inside that population and passes.
- `bash engine/scripts/check-milestone2-acceptance.sh <EN> <RU>`: exit 1, on
  both roots, on exactly the two reasons untouched main exits 1 on and with
  the same counts:

  ```
  "... local UI settings have no admitted original application record ..." now 15, baseline 8
  "... original-compatible town save member \"hero\" changed ..."          now  2, baseline 0
  ```

  Both are pre-existing corpus drift the seat already tracks; the gate reads a
  discovered corpus rather than a pinned one. The three reasons this story had
  added to that set before the correction -- `never-imported...` risen to 94,
  and a newly minted `current application state is absent` -- are gone.
- Milestone script census, `AGAINROM_ASSETS=gameversions/en missionrun
  -mission N -trace -ticks 1 | grep -c UNSUPPORTED`: mission 10 and mission 20
  both **0 before, 0 after**, and each mission's script population printed
  exactly its `pipeline/milestone-baseline.txt` line (`m10: 16 checks, 27
  instants, 12 triggers`; `m20: 14 checks, 15 instants, 11 triggers`). Nothing
  in this story reaches `pkg/sim`'s script or trigger dispatch.
- Engine `origin/main` is `6a26c1c`, identical to this branch's base.

## Open debt

- **Guard coverage gaps**, all verified by construction: directory names,
  non-`.go` file names, struct tags and string literals are not read. Closing
  them means a second walker over the filesystem rather than over `go/ast`,
  and is a separate scope.
- **Exact-match `CommentBytes` stays high-friction.** Any comment edit
  anywhere in 9.5 MB fails the build until `baseline.go` changes, and
  `measure` regenerates all eight counts at once, so a routine paste can
  launder a rise in `storymention` or `TestIdentCount`. This pass's remedy was
  to read every number before pasting; the durable remedy would be for
  `measure` to emit a diff against the committed baseline rather than a fresh
  block. `Scan` also walks the filesystem rather than Git, so an untracked
  scratch `.go` file with a comment fails `TestLiveTreeClean`.
- **`TestScanAppliesTheExceptionList` is the only temp-dir `Scan` test.** The
  other five live-tree break categories were exercised by hand in the
  adversarial pass and are not committed as re-runnable tests.
- **`AGENTS.md`'s divergence-routing paragraph is incomplete**: it omits
  `pipeline/next-div-id.sh`, which also reads `docs/DIVERGENCES-CLOSED.md` and
  `docs/hotfix/LEDGER.md`, and it says the file is chosen by the row's own
  Subsystem text while `docs/DIVERGENCES.md` declares Subsystem free text. The
  actual mapping lives in `docs/1203/story.md`. No instrument enforces
  routing.
- **`pkg/game/worldsave.go`'s comment no longer names
  `TestReleaseNewCorpse1200MapUnitZero`** as the test that proves its
  `MapUnitID` 0 corpse guard. The guard and its `DIV-997`/`DIV-1344` citations
  survived the comment cut; the pointer to its proving test did not.
- **6,072 story-numbered identifiers remain in test files across 272 numbered
  test file names**; both are committed baselines that may only fall.
- **Forbidden comment forms remain tree-wide**, all ratcheted: 10,401
  spec-clause citations, 977 `story NNNN` mentions, 517 calendar dates, 70 raw
  ROM1 addresses, 142 `FUN_xxxxxxxx` names, 54 `EXP-NNNN` citations. These use
  `internal/storyguard`'s own word-bounded regexes and differ from the rougher
  pre-story seat census by methodology.
- **Total comment bytes tree-wide: 9,473,680**, ratcheted. The comment cleanup
  beyond symbols this story renamed is explicitly not attempted.
- **The 12-line comment ceiling is guidance in `AGENTS.md`, not a coded
  gate**; 2,582 existing groups exceed it.
