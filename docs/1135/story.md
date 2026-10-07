# Story 1135 — Diary entries and Player raw tail

## Status

Gates run on the exact commit; see "Proof" for the full list and exact
counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository, asset-free) clean, 49 packages, 0 FAIL, including
`internal/gatedtests`' checked-in `testdata/population.txt` census (204
registered names — `population.txt` itself is 208 lines, 4 of them
comments); `scripts/check-no-game-assets.sh` clean;
`pipeline/check-release-tests.sh` against both lawful roots (8 packages, 204
gated tests, 204 of 204 ran and 0 lacked a subject on each of EN and RU);
mission 10/20 `-trace -ticks 1` UNSUPPORTED-node count 0 on both missions,
both roots, unchanged from master — this story's own restoration is reached
only through an original-save resume, a path the fresh mission drive does
not exercise (story 1132/1133's own precedent, "Proof"). A seat hotfix
closed review findings F-1, F-2, F-5 and F-6 of
`pipeline/reviews/story1135-pass1.md`: two release witnesses now assert real
Diary entry values over `2026-08-24/game9999.sav` instead of counts alone,
the corpus audit compares entry values and the complete re-exported body
byte for byte instead of lengths alone, this Status/Proof carry the actual
post-hotfix gate output, and DIV-956 states the `.ags` resume loss in
DIV-944's own words. F-3 and F-4 remain named debt, unchanged by the
hotfix.

## Player result

A mission SAV written by ROM1 now restores its Diary content per owning
unit instead of as a bare count. The Player's own inline Diary and every
admitted Humanoid's own referenced Diary are present in `sim.World`'s
carried state before the first tick — length, and every element that
departs from the array's own construction default — instead of being
decoded once for a report line and dropped. `RestoredParty`'s own
`JournalChars`/`JournalEntries` count, which previously was the only
surviving trace of this content, is unchanged; the typed content now
carried alongside it is new. The Player block's 32-byte raw save tail also
decodes: its one claim-named field, the formation-mode byte, is a typed
accessor now instead of an anonymous byte inside an opaque span, and the
whole 32-byte block still round-trips through `program.go`'s own raw-member
retention with no further plumbing required, since nothing in this project
ever overwrites it.

No live consumer reads a restored Diary entry: this build has no journal,
quest-counter or UI surface for one (DIV-956), on the same "carried, not
enacted" shape stories 1132 and 1133 already established for the SpellEffect
graph and the Projectiles store. The decoded Formation byte is not bound to
this project's own live per-player formation array either, because no claim
this story cites relates a save file's own Player identity to a live player
index (DIV-957). `exportOriginalDiaries` stays an unshipped round-trip
building block: no production mission SAVE path exists in this project
(owner rule), so it is exercised only by the corpus audit and the release
witness's own native-export half.

## Authority

- **`SAV-DIARY-042`** (active, High for the array shapes): a Diary owns two
  parallel arrays, a 4-byte and a 2-byte element, whose declared lengths
  need not agree in general (this project's own decoder additionally
  refuses a record whose two counts disagree, since `SAV-667` finds them
  always equal across the corpus).
- **`SAV-667`** (cited within promoted `SAV-668`): a fresh Diary sizes both
  arrays once from the live Units table; an array index is a Units-table
  row, i.e. a unit type; the two arrays' element counts are always equal
  across the corpus.
- **`SAV-668`** (promoted): `word[i] = 1024 - dword[i]` exactly on all
  64260 sampled elements, no exception; the one located mutator
  (`R1564`) is a floor-limited decrement touching only the word
  array, whose own caller was not traced further — "the screen or
  tick-phase context that ultimately calls it remains Unknown"; only six of
  119 Units-table indices ever carry a nonzero value anywhere in the
  45-save research corpus, and which six is not named.
- **`SAV-669`** (promoted): a Diary's own `D2C` self-reference resolves to
  its enclosing Player with zero counterexamples across the preserved
  population.
- **`SAV-PLDIARY-054`** (active, High): the Player's own inline Diary is a
  short record programme, reached by a fixed construction path, not a
  terminator.
- **`SAV-EMBED-039`** (active, High for the mechanism, Unknown for content):
  a Humanoid's own Diary reference lives at a fixed embedded offset,
  resolved inside a shared helper rather than `Unit::Serialize` itself.
- **`SAV-662`** (promoted, High): locates `AI-FORM-037`'s own formation byte
  at index `+0x1f` inside `Player+0x30`'s own 32-byte block, which
  `Player::Serialize` writes as one literal `CArchive` call with no
  per-field dispatch; exhaustive corpus census over 410 records finds only
  that one byte ever nonzero, and partially bounds three of the other 31
  (two with a known dead writer, one with three known readers and no
  writer).
- **`SAV-663`** (active): three more functions share that same `+0x1f`
  access and are unreferenced anywhere in the image.
- **`AI-FORM-037`** (active, amended): the formation byte's whole ROM1
  surface — allocation, constructor default, one setter, two group-move
  readers. This story cites only the corpus-untouched parts; the row's own
  once-refuted corpus clause (a single ALM trigger node's own parameter
  reading, corrected by `EXP-0170`) is a different fact about a shipped
  map's own script, not about a `.sav` file's own byte content, and is not
  leaned on here (checked directly against `research/claims/retracted.md`
  before this row was written).

## As-built behaviour

**`pkg/formats/sav`** (`diary.go`, `playertail.go`, both new; `party.go`
extended): `diaryFromRecord`/`PlayerDiary` decode a Diary's two parallel
arrays into a sparse `[]DiaryEntry` (a default element, `Count==0,
Remaining==1024`, carries no information the file distinguishes from
"never touched", so it is never reported) plus `Length`, the arrays' own
shared declared count, carried separately so a byte-for-byte re-export can
reproduce an all-default Diary's own array length. `SetDiary` is the exact
inverse. `Character` gains `HasDiary`/`Diary` (decoded via `PartyWalk`,
which — unlike `Party()` — retains every raw actor-list reference including
repeats) and a new `ArchiveIndex uint16` field, mirroring `Record.Index`
directly: this is the export-direction join key, because `MapUnitID`, the
existing map-placement identity, is unset (0) for the party's own lead
Hero on the release fixture used below — confirmed by direct inspection of
that file, not assumed, and already documented as a known gap on
`actorCharacter`'s own doc comment before this story. `PlayerTail` decodes
`Player+0x30`'s own 32-byte block (`PRaw32`, a `KindRaw` member `program.go`
already retains on every walk) into a `Raw [32]byte` plus one named
accessor, `Formation()`, at index `0x1f`.

**`pkg/sim`** (`saveddiary.go`, new): `SavedDiary`/`SavedDiaryEntry` mirror
the `sav` types with no `pkg/formats/sav` import; `SavedDiaryOwner` names
either the Player or a live `EntityID`. `World.savedDiaries` is
carried-not-wire-form, `savedProjectiles`' own class: `UnmarshalBinary`'s
composite literal carries the receiver's own prior value across a decode,
so a decode into a FRESH receiver needs `SetSavedDiaries` first or the
value is silently lost. `ImportOriginalLivingActors`
(`pkg/sim/originalliving.go`) and `restoreOriginalActorStock`
(`pkg/game/originalholdings.go`) both stage every original actor admission
through exactly that round trip and both now prime it, on
`savedCellRecords`'/`savedSpellEffects`'/`savedProjectiles`' own precedent.
`resumeWorld`'s own `.ags` resume needs no priming call, proven rather than
inferred by `TestUnmarshalBinaryOntoTheSameReceiverNeedsNoDiaryPriming`
(`pkg/sim/saveddiary_test.go`).

**`pkg/game`** (`originaldiaries.go`, new): `applyOriginalDiaries` takes a
second, independent `PartyWalk()` result (deduped by `.Off`, since the walk
repeats archive references) alongside the Player record, resolves each
Humanoid's owner through the existing nil-safe `registryTarget` helper —
skipping one the registry never admitted, the same tolerance
`applyOriginalActorPools` already applies — and calls
`ms.World.ImportOriginalDiaries`. Wired into both original LOAD doors
(`ResumeOriginalSave`, `RestoreOriginal`) between `admitOriginalActorRegistry`
and `applyOriginalSession`, and into `restoreOriginalActorStock`'s staging
round trip. `OriginalSaveResume` reports `Diaries`/`DiariesApplied`
alongside the existing fields; its `String()` no longer claims "the
journal" is NOT CARRIED, which became false as of this story (the same kind
of fix stories 0147 and 1130 made for their own surfaces).
`exportOriginalDiaries` builds two lookup maps — the Player's own owner is
matched directly through the decoded file's own `Refs["Diary"]`, an Actor
owner through `ArchiveIndex` on both sides (the file's own `Character.
ArchiveIndex` against the live `Entity.SourceBinding.ArchiveIndex`, both
already populated at their own respective points) — and calls `f.SetDiary`
per diary. No production caller: this project has no production mission
SAVE path yet (owner rule), so it is an unshipped round-trip building
block, checked only by the corpus audit and the release witness's own
native-export half.

`PlayerTail`/`Formation()` are decode-only: no `pkg/game` or `pkg/sim`
carrier was added for the 32-byte block, because nothing in this project's
own LOAD or export path ever overwrites `PRaw32`'s own bytes in the
decoded file — `program.go`'s raw-member retention already keeps it
verbatim across every walk, so a native re-export (were one wired to
production) reproduces it with no further code. Applying the Formation
byte to a live player would need code this story does not add (DIV-957).

## Divergence rows

- **DIV-956** (UNKNOWN, OPEN): Diary content is carried, never enacted. No
  live journal, quest-counter or UI surface reads it; the located mutator's
  own caller and the meaning of the six ever-nonzero Units-table indices
  are both Unknown to research, on the same standing rule `DIV-939` and
  `DIV-944` already state for the SpellEffect graph and the Projectiles
  store.
- **DIV-957** (UNKNOWN, OPEN): the Formation byte decodes but is not bound
  to this project's own live per-player formation array
  (`pkg/sim/formation.go`), because no claim relates a save file's own
  Player identity to a live player index.
- **DIV-958** (UNKNOWN, OPEN): the other 31 bytes of the Player's own raw
  tail are carried in `PlayerTail.Raw` and not further decoded; three of
  them have partial writer/reader coverage from `SAV-662` with no
  applicable value (a writer that only ever stores 0, or a reader keyed to
  an unidentified second object), and the remaining 28 have no individual
  claim at all.
- DIV-959 through DIV-961 (reserved, unused) are returned unclaimed.
- DIV-939 (story 1132) and DIV-944 (story 1133) are different, narrower
  surfaces (the SpellEffect graph, the Projectiles store) and are unchanged
  by this story; DIV-956 follows the same carried-not-enacted shape for a
  fourth, distinct subtree.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean,
  204 registered names; `population.txt` is 208 lines, 4 of them comments.
- `scripts/check-no-game-assets.sh`: clean.
- Two release witnesses over `2026-08-24/game9999.sav`
  (`TestReleaseOriginalDiaryRealPlayerEntriesRestoreOnLoad1135` and
  `TestReleaseOriginalDiaryRealPlayerEntriesSurviveNativeExport1135`,
  `originaldiary1135_release_test.go`) assert the fixture's four real Player
  Diary entries — `{64,2,1022} {88,2,1022} {96,6,1018} {100,5,1019}` — by
  value on LOAD, and the same entries plus all 63,398 body bytes exactly
  after a native re-export; both pass on EN and RU (review finding F-1).
- `TestDiaryCorpusAudit1135` (`-tags sessioncorpusaudit`, opt-in, both
  lawful roots): identical log line on EN and RU — `audited 72 file(s): 51
  world-half, 20 between-mission, 1 unreadable, 48 with more than one
  diary, 0 mismatching`. The player and actor comparisons now check each
  Diary's decoded entry values, not `Length`/element-count alone, and the
  export comparison diffs the complete re-exported body byte for byte
  against the source file, not the re-decoded Diary's shape alone (review
  finding F-2); the corpus still produces zero mismatches under the
  strengthened check. A verbose run's own per-file entry counts for the
  Player's own Diary span 0 to 6 non-default elements (10 files at 0, 41
  with real content, summing to 51) — real, non-default Diary content is
  present broadly across the corpus, not a single repeated sample; the "48
  with more than one diary" figure above independently confirms actor
  Diaries are likewise present, not merely the Player's own.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 204
  gated tests, 2 roots; **204 of 204 ran and 0 lacked a subject, on EN and
  again on RU.**
- `pipeline/check-milestone.sh`: not run. This story's restoration is
  reached only when resuming an original save, a path the census's own
  fresh, non-resumed mission drive never exercises — stories 1132/1133's
  own reasoning for the same population.
- `go build ./cmd/missionrun` (from this worktree) plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both
  missions, both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged
  from master (`pipeline/milestone-baseline.txt` carries no `cannot run`
  line for either mission), and matching story 1133's own last recorded
  value for this same check. Not rerun for the hotfix: it touches no
  production or simulation code.
- `pipeline/check-div-claims.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree): exit 0, 396 live row(s) of 396 (0 closed), citing 586
  distinct claim id(s); 107 row(s) cite a claim carrying a retraction row
  (informational — DIV-957's own citation of `AI-FORM-037` is one of them,
  whose retraction covers a single shipped-map trigger-node reading this
  row does not lean on, checked directly against
  `research/claims/retracted.md`). Unchanged by the hotfix: DIV-956's
  wording correction (F-6) adds prose, not a new row or citation.

## Open debt

**No live consumer reads a restored Diary entry** (DIV-956). What screen or
tick-phase context calls the one located mutator, and what the six
ever-nonzero Units-table indices name, are both Unknown to research; this
project has no journal, quest-counter or other UI surface that would
consume this state even if they were known.

**The Formation byte is decoded, not applied** (DIV-957). This project's
own live per-player formation array already exists and already implements
`AI-FORM-037`'s whole mechanism; what is missing is a claim relating an
original save's own Player object identity to a live player index, not
implementation effort.

**28 of the Player raw tail's 31 non-Formation bytes have no individual
claim at all** (DIV-958); the other 3 have partial writer/reader coverage
with no applicable value. `SAV-662`'s own corpus-zero finding is a
population fact over 410 preserved records, not a proof the bytes are
structurally inert.

**Native mission SAVE has no interactive path yet.**
`exportOriginalDiaries` is exercised only by this story's own corpus audit
and release witness, on `exportOriginalCellRecords`/`exportOriginalSpell
Effects`/`exportOriginalProjectiles`'s own precedent (stories 1131-1133):
the owner's own standing rule against inventing a production mission SAVE
entry point applies here the same way.

**The `.ags` resume drops carried Diary state while nothing reads it**, the
same gap DIV-932/938/944 already name for `savedCellRecords`/
`savedSpellEffects`/`savedProjectiles`: `World.MarshalBinary` does not
encode `savedDiaries`, so `resumeWorld` (`pkg/game/resume.go`) decodes onto
an empty value regardless of what an `.ags` envelope's own prior session
held. `World.Hash()` is unaffected, since no simulation step reads this
field today.

## Touched surfaces

- `pkg/formats/sav/diary.go` (new): `Diary`/`DiaryEntry`, decode/encode,
  `PlayerDiary`, `SetDiary`.
- `pkg/formats/sav/diary_test.go` (new): unit tests built from claims.
- `pkg/formats/sav/playertail.go` (new): `PlayerTail`, `Formation()`,
  `PlayerTailFromRecord`.
- `pkg/formats/sav/playertail_test.go` (new): unit tests built from claims.
- `pkg/formats/sav/party.go`: `Character.HasDiary`/`Diary`/`ArchiveIndex`.
- `pkg/formats/sav/program.go`: walker support for the new decoded members.
- `pkg/sim/saveddiary.go` (new): `SavedDiary`/`SavedDiaryEntry`/
  `SavedDiaryOwner` and `World` accessors.
- `pkg/sim/saveddiary_test.go` (new): round-trip/detach contract and the
  same-receiver no-priming proof.
- `pkg/sim/world.go`: `savedDiaries` field.
- `pkg/sim/binary.go`: carry it across `UnmarshalBinary`'s composite
  literal.
- `pkg/sim/originalliving.go`: prime it before `ImportOriginalLivingActors`'
  own staging decode.
- `pkg/sim/nostate_test.go`, `pkg/sim/world_test.go`: field/method-set pins.
- `pkg/game/originaldiaries.go` (new): the sav<->sim converters, LOAD and
  export entry points.
- `pkg/game/originaldiary1135_test.go` (new): asset-free nil-guard and
  conversion unit tests.
- `pkg/game/originaldiary1135_corpus_test.go` (new, opt-in): the corpus
  audit.
- `pkg/game/originaldiary1135_release_test.go` (new): the release witness.
- `pkg/game/originalsave.go`: wire both original LOAD doors,
  `OriginalSaveResume`'s report/`String()`.
- `pkg/game/originalsave_test.go`: updated report-text assertions.
- `pkg/game/originalholdings.go`: prime the same carried field before
  `restoreOriginalActorStock`'s own staging decode.
- `pkg/game/originalparty.go`: doc-comment correction (journal content is
  no longer read-and-discarded).
- `internal/gatedtests/testdata/population.txt`: register the new release
  witness.
- `docs/DIVERGENCES.md`: DIV-956 through DIV-958.
