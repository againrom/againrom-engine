# Story 1131 — cell-record keys, baselines and layer count

## Status

Gates run on the exact candidate commit; see "Proof" for the full list and
exact counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository, asset-free) clean, 49 packages, 0 FAIL, including
`internal/gatedtests`' checked-in `testdata/population.txt` census (net +2:
the two new release witnesses named below, 194 registered names —
`population.txt` itself is 198 lines, 4 of them comments); `scripts/check-no-game-assets.sh`
clean; `pipeline/check-release-tests.sh` against both lawful roots (8
packages, 194 gated tests, 194 of 194 ran and 0 lacked a subject on each of
EN and RU); `pipeline/check-milestone.sh` (seat script, `AGAINROM_MILESTONE_DRIVE`
pointed at a `missionrun` built from this worktree) exit 0, unchanged on both
roots — expected, since this story's own restoration only reaches an
original-save resume, a path the census's fresh mission drive does not
exercise.

## Player result

A player resuming an original mid-mission save now gets the complete
per-cell record back, not only its six-byte trigger tail and Building slot
(story 1106, story 1114): the captured terrain cost/static baselines and the
occupied-layer count read back as typed saved-cell state, and the two
movement-domain identity slots (Ground, Air) rebind to the same live actor
Building's own slot already rebinds to, wherever this build's own actor
registry can resolve the saved archive key. Sack and the six SpellEffect keys
carry through unresolved (no live registry for either exists in this
project) but byte-identical, so nothing already-working regresses.

Native mission SAVE itself is not a wired player-facing feature yet: no
interactive path calls the native writer this story adds (see "Open debt").
Mid-mission continuation remains exclusively the existing lossless `.ags`
envelope; this story's own native writer is a tested building block for a
later story in the same milestone sequence (`pipeline/SAV-COMPLETION.md`,
owner milestone 2, step 2 of the sequence).

## Authority

- **`SAV-CELLLOAD-109`** (High): a saved cell table overlays the cell hash
  the ALM already constructed; it does not replace or clear it. A saved key
  overwrites its constructed payload, a saved-only key is inserted, and a
  construction-only node survives untouched.
- **`SAV-CELLLOAD-110`** (High): ten cell-payload dwords (Ground `+0x04`, Air
  `+0x08`, Building `+0x0c`, Sack `+0x10`, six SpellEffect `+0x14..+0x28`)
  are persisted archive identity keys, each later replaced by a live pointer
  on successful lookup in the archive identity map; zero and a failed lookup
  leave the saved value unchanged. Runtime failure behaviour after a
  preserved raw key is Unknown.
- **`SAV-CELLLOAD-111`** (High for the positive baseline, count and identity
  fields and the two conditional trigger consumers; Medium for treating
  `+0x03` and `+0x32..+0x33` as ignored residue; Unknown whether either
  restored shipped trigger completes in a running original): `+0x00`/`+0x01`
  are the captured cost/static baselines recomputation reads; `+0x02` is the
  occupied six-layer count, retained until an ordinary layer attach/detach
  recounts it; `+0x03` and `+0x32..+0x33` are constructor-zero with no known
  consumer in the enumerated population.
- **`SAV-CELLREC-032`** (Medium, table measurement) / **`SAV-TERRKEY-056`**
  (High, table trailer): the cell-record table itself is `count + count × 54`
  bytes with no tail — `R1521`, its own serializer, writes nothing
  past the last record. The four bytes once measured immediately after the
  table (the "unattributed" framing `SAV-CELLREC-032` originally used,
  `claims/retracted.md:449`) are the terrain object's own inline identity
  key in `SAV-PTRMAP-035`'s pointer-map scheme, written by a different
  routine (`R1360`) for a different object. They are not part of any
  per-cell record and are out of scope for this story on that basis, not by
  omission.

## As-built behaviour

**1. `pkg/formats/sav` carries the complete typed record (prior commit
`84550c66`).** `Cell` projects every field of the 54-byte record: both
baselines, layer count, all ten archive-identity keys, the six-byte trigger
tail and both residue spans. `Cells()`/`SetCell()` read and write it
in archive order, key included, on `CellTriggers`/`StructureCells`'s own
narrower-projection precedent — those two accessors remain unchanged
alongside the new complete one.

**2. `sim` carries the saved residue (prior commit `243201c8`).**
`SavedCellRecord{Cell, LayerCount, Residue0, Residue1, Ground, Air
SavedCellActorSlot, Sack, SpellEffects}` and `SavedCellActorSlot{Key,
Entity, Bound}` are new types; `SavedCellRecords()`/`SetSavedCellRecords()`/
`ImportOriginalCellRecords()` are new `*World` methods. `savedCellRecords` is
carried-not-wire-form — the same choice DIV-041 and DIV-926 already name for
other fields with no rebuild rule — sorted ascending by cell key on import,
refusing a duplicate key outright the same way `ImportOriginalStructures`
already refuses one for `StructureCell`.

**3. Both original LOAD doors wire the record in (this story).**
`applyOriginalCellRecords`/`exportOriginalCellRecords`
(`pkg/game/originalcellrecords.go`, new) are the third cell-table overlay,
beside `applyOriginalCellTails` and `applyOriginalStructures`:
`applyOriginalCellRecords` reads `sav.Cell` through both `ResumeOriginalSave`
and `FrontEnd.RestoreOriginal`, dedupes to last-write-wins by cell key
(`applyOriginalStructures`'s own precedent for `StructureCell`, matching
SAV-CELLLOAD-109's own restore order), and rebinds Ground/Air to a live
`EntityID` wherever this build's own actor registry names one by archive
identity, leaving the raw key unchanged otherwise. Sack and the six
SpellEffect keys carry opaque: this project has no live identity registry
for either class (DIV-933). `exportOriginalCellRecords` is the native-writer
counterpart, the same building-block role `exportOriginalMissionSession`
plays for story 1130's session block: it merges `SavedCellRecord`,
`SavedStructureCell` and `CellTails` back into the complete record, row for
row against the file's own archive order.

**4. A real defect found and fixed: `ImportOriginalLivingActors`' own
staging round trip silently dropped the whole saved-cell-record span.**
`ImportOriginalLivingActors` (`pkg/sim/originalliving.go`) stages every
original actor admission through `MarshalBinary`/fresh-`UnmarshalBinary`,
the same detached-staging shape `restoreOriginalActorStock` uses. Before
this story, its fresh `checked` receiver primed only `sourceDerive`;
`savedCellRecords` is carried-not-wire-form (item 2 above), so the fresh
receiver's own value started empty regardless of what the caller's `w`
already held. `admitOriginalActorRegistry` (which calls
`ImportOriginalLivingActors`) runs between `applyOriginalCellRecords` and
`applyOriginalSession` in both original LOAD doors, so this loss was real,
not hypothetical: it reached the live `ms.World` on every original-save
resume, one call after `applyOriginalCellRecords` had restored the span
correctly. Story 1130 never needed to close this exact gap, because its own
carried fields (`rawSessionHead`/`rawSessionMid`) are restored by
`ImportOriginalSession`, which runs *after* `ImportOriginalLivingActors` in
both doors — the two stories' own carried fields cross this staging step in
opposite order, so a fix sized to 1130's own defect would have missed this
one.

This was found by direct probing against a real preserved EN save, before
this story's own corpus instrument existed to also catch it:
`SavedCellRecords()` read back empty despite `applyOriginalCellRecords`
having reported success moments earlier. The fix adds
`checked.SetSavedCellRecords(w.SavedCellRecords())` before `checked`'s own
decode in `ImportOriginalLivingActors`, priming the fresh receiver the same
way `mapload.BindSourceDerive` already primes `sourceDerive` there.
`restoreOriginalActorStock`'s own staging round trip requires the same
priming: reverting it alone leaves `pkg/sim` green but fails both release
witnesses on the story's own fixture (`2026-08-02/game9999.sav`), 52 unbound
Ground keys, one per non-zero Ground key the fixture carries.
`TestImportOriginalLivingActorsStagingCarriesSavedCellRecords`
(`pkg/sim/savedcellrecord_test.go`) is the regression witness: confirmed to
fail with the exact symptom (records present before the call, empty after)
when the fix is reverted, and to pass with it restored.

**5. Acceptance instrument.** `TestCellRecordCorpusAudit1131`
(`pkg/game/originalcellrecords1131_corpus_test.go`, build tag
`sessioncorpusaudit`, opt-in) extends story 1130's own corpus-audit shape to
the complete record: for every preserved owner save under
`gameversions/saves` that carries a world half, it reads the file's own
`Cells()` directly, resumes the same file through the ordinary front-end
path, and independently reconstructs the live comparison value from
`SavedStructures`/`SavedCellRecords`/`Entities` (not by calling
`exportOriginalCellRecords` for both directions), comparing every field:
baselines, Building, layer count, residue, Ground/Air key **and** bind
correctness, Sack, SpellEffects and the trigger tail. It then re-opens a
native export of the live state and compares the complete record-table span
against the file byte for byte. Result: **72 files audited (51 world-half,
20 between-mission, 1 unreadable), 0 mismatching, on both EN and RU**
(identical log line on both roots: `audited 72 file(s): 51 world-half, 20
between-mission, 1 unreadable, 0 mismatching`) — the same population story
1130 found. Across the reachable corpus, LayerCount, Residue0, Residue1 and
all six SpellEffect keys are 0 non-zero everywhere; Ground and Air both have
real non-zero samples (DIV-934).

**6. Release witnesses.** `TestReleaseOriginalCellRecordsRestoreOnLoad1131`
and `TestReleaseOriginalCellRecordsDiagnosticFieldsSurviveLoad1131`
(`pkg/game/originalcellrecords1131_release_test.go`, new) run through the
ordinary App LOAD path (`releaseFront`, `groundCorpusFile`, `groundAppLoad`,
1130's own helpers). The first covers baselines and both identity slots with
real content, against `gameversions/saves/2026-08-02/game9999.sav`
(hash-pinned) — one of five reachable owner saves tied at the corpus
maximum of five non-zero Air keys each (5 of 197 records; 23 of 51
world-half saves carry at least one non-zero Air key, 76 non-zero Air cells
total), so it is a defensible fixture for a natural-content Air witness
rather than a diagnostic mutation. The second covers LayerCount
and the two residue bytes, all-zero in every reachable save (DIV-934): a
named diagnostic byte pattern written into one record (that also carries
real Ground/Air content) before LOAD, on
`TestReleaseOriginalSessionRawSpansSurviveActorStockStaging1130`'s own
precedent for RawMid — a probe of the carry-through mechanism, not a claim
about original runtime content. Both drive a native-export byte check on
top and pass on EN and RU; both are registered in
`internal/gatedtests/testdata/population.txt`.

## Divergence rows

Amended (`docs/DIVERGENCES.md`):

- **DIV-730**: the "layer count, the other nine identity slots and residue
  remain unapplied" clause is replaced with a STORY 1131 paragraph recording
  the complete record's restoration — Ground/Air rebind, layer
  count/residue carry, Sack/SpellEffect opacity — and the corpus/native-export
  result. Narrowed to OPEN for the operation26 relocation consumer alone,
  the one payload owner this story leaves untouched. The remaining-work
  column no longer asks for "remaining payload owners"; only relocation and
  an optional future Sack/SpellEffect registry remain named.
- **DIV-026**: the cell-fields clause in the "remain unrestored" sentence is
  narrowed to name only the operation26 relocation consumer; a STORY 1131
  paragraph is added alongside the existing STORY 1130 one, recording the
  same corpus result for the cell-record table.

New, from the reserved DIV-932 through DIV-937 range
(`pipeline/ALLOCATIONS.md`):

- **DIV-932** (cell-record layer count and residue carry-across): records
  the `ImportOriginalLivingActors` staging defect found and fixed by this
  story (item 4 above) — the same carried-not-wire-form class DIV-926
  already names, reached through a different caller than the one DIV-926
  itself closed. Narrowed, not closed, on DIV-926's own precedent: a
  different fresh-`World` decode this project already has —
  `resumeWorld` (`pkg/game/resume.go`), the `.ags` resume path — still loses
  the span unless it primes it the same way.
- **DIV-933** (cell-record Sack and SpellEffect identity opacity): records
  that this project has no registry mapping the cell-record table's own
  archive-identity keys to a live Sack or SpellEffect object — `sim.Sack`'s
  own `ObjectID` is a separate, project-native identity for holdings/item
  tracking (story 1076), not a rebind target for this archive key space, and
  no SpellEffect runtime type exists at all. The raw keys still round-trip
  byte-identical.
- **DIV-934** (cell-record LayerCount, residue and SpellEffect real-content
  coverage): records that these four fields are 0 non-zero across the
  entire reachable corpus (both roots), so their carry-through fidelity is
  proven only by mechanism and one named diagnostic mutation, not by an
  authentic non-zero sample — DIV-928's own precedent for RawMid, extended
  to these fields. Ground and Air do not share this gap: both have real
  samples in the same corpus.

**DIV-935 through DIV-937 remain unused and retired**: no further named gap
this story found was a new, unclaimed owner-direction choice of its own
distinct from DIV-932 through DIV-934 above.

**Side correction, not this story's scope.** DIV-926 and DIV-927 (story
1130) cited that story's own pre-hotfix population, "26 world-half saves":
the landed figure after story 1130's seat hotfix (`c273db81`) is 72 files,
51 world-half, 20 between-mission, 1 unreadable, 0 mismatching, on both EN
and RU — the same figure DIV-026, DIV-928 and this story's own DIV-932
already carry correctly. Both rows are corrected to the landed figure in
this same ledger edit, as a carried fix, not new evidence from this story.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean
  (net +2, 194 registered names; `population.txt` is 198 lines, 4 of them
  comments).
- `scripts/check-no-game-assets.sh`: clean.
- `TestCellRecordCorpusAudit1131` (`-tags sessioncorpusaudit`, opt-in, both
  lawful roots): 72 files audited (51 world-half, 20 between-mission, 1
  unreadable), 0 mismatching, on EN and again on RU (identical log line both
  roots: `audited 72 file(s): 51 world-half, 20 between-mission, 1
  unreadable, 0 mismatching`).
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 194 gated
  tests, 2 roots; **194 of 194 ran and 0 lacked a subject, on EN and again on
  RU.**
- `pipeline/check-milestone.sh` (seat script, run because a restored
  baseline/layer count can change a shipped map's tick outcome;
  `AGAINROM_MILESTONE_DRIVE` pointed at a `missionrun` built from this
  worktree, `AGAINROM_MILESTONE_ALLOW_STALE=1` because that override's own
  binary has no relationship to the unrelated `implementation/` checkout's
  HEAD timestamp the stale-drive check reads by default): **exit 0,
  unchanged from `pipeline/milestone-baseline.txt` on both roots** — all 28
  campaign maps' compiled-script signatures match and zero `cannot run`
  lines on either root, the same population the baseline already recorded.
  This story's own restoration is reached only when resuming an original
  save, a path the census's own fresh, non-resumed mission drive never
  exercises, so neither the script-gap census nor the mission-10 escort
  drive's own tick/outcome/census line moved.
- `go build ./cmd/missionrun` (from this worktree) plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master,
  whose own recorded census already carries no `cannot run` line for either
  mission (`pipeline/milestone-baseline.txt`).

## Open debt

**Native mission SAVE has no interactive path yet.** `exportOriginalCellRecords`
is a tested building block, not a wired feature: no UI SAVE action calls it,
the same debt class story 1130 named for `exportOriginalMissionSession` and
confirmed the same way — tracing every caller in this codebase before
writing it. This story's own scope (`pipeline/SAV-COMPLETION.md`, owner
milestone 2, step 2 of the same 8-story sequence) stops at the
building-block level by design; a later story in the same sequence wires an
interactive path.

**Sack and SpellEffect identity keys stay opaque (DIV-933).** No live
registry exists in this project for either class, so `SAV-CELLLOAD-110`'s
rebind rule has no target to apply to here; the raw archive keys still carry
through byte-identical. Building one with no consumer that reads it would be
unproved synthesis.

**LayerCount, residue and SpellEffect real-content fidelity is unverified
beyond one diagnostic mutation (DIV-934).** Every preserved owner save
available to this project carries all four as zero. The carry-through
mechanism is proven (same code paths Ground/Air use), but no authentic
non-zero sample exists in the accessible corpus to check byte-for-byte
fidelity against.

**The layer-count/residue carry-across gap is narrowed, not closed
(DIV-932).** This story closes it for both `ImportOriginalLivingActors`' and
`restoreOriginalActorStock`'s own staging round trips — both required, each
separately witnessed: reverting either alone fails a distinct witness (the
unit regression test for the first, both release witnesses for the second).
A different fresh-`World` decode this project already has —
`resumeWorld` (`pkg/game/resume.go`), the `.ags` resume path — still loses
the span unless it primes it with `SetSavedCellRecords` first.

**A second, weaker carrier already holds the same Ground/Air state (review
F-1).** `sim.SavedActorCell` (`pkg/sim/savedmotion.go`) already carries the
complete 52-byte record and the same Ground/Air `Key`/`Entity`/`Bound`
triple, in wire form, measured identical to this story's own
`savedCellRecords` in 8,654 of 8,654 records across the corpus. It survives
the `.ags` resume where `savedCellRecords` does not. Whether
`SavedCellRecord` should fold into the existing carrier, or a second
carrier is deliberately wanted, is undecided.

**Twenty Ground keys naming a dead actor do not rebind, and the acceptance
instrument cannot see it (review F-5).** `planOriginalActors` skips every
dead actor when building the identity map `applyOriginalCellRecords` binds
from; 20 of 2,010 non-zero Ground keys in the corpus name such an actor and
keep their raw key. Whether ROM1 itself binds a cell slot naming a dead
actor is a research question, not measured here. `TestCellRecordCorpusAudit1131`
derives its own expectation from the same live population the production
code binds from, so it reports zero mismatches for this class regardless of
whether the rebind is correct.

**Operation26 relocation stays unimplemented (DIV-730, narrowed).** The
trigger tail's own `+0x2c == 26` relocation coordinates persist through AGS
(story 1106) but no code enacts the relocation itself; this remains the sole
unimplemented payload owner in the complete cell-record table.

Out of scope, unchanged, per the brief: general dead-object graphs,
remaining actor state, orders, casts, area effects, 18 bytes of unpromoted
session gaps (DIV-927) — DIV-026's own pre-existing remainder, unaffected by
this story beyond the cell-fields clause it narrows.

## Touched surfaces

`pkg/formats/sav/cellrecord.go` (new, prior commit), `pkg/formats/sav/cellrecord_test.go`
(new, prior commit).

`pkg/sim/savedcellrecord.go` (new, prior commit: `SavedCellRecord`,
`SavedCellActorSlot`, `SavedCellRecords`/`SetSavedCellRecords`/
`ImportOriginalCellRecords`), `pkg/sim/binary.go`, `pkg/sim/world.go`,
`pkg/sim/world_test.go`, `pkg/sim/nostate_test.go` (prior commit: pins
extended for the new field).

`pkg/game/originalcellrecords.go` (new: `applyOriginalCellRecords`,
`exportOriginalCellRecords`), `pkg/game/originalsave.go` (both original LOAD
doors call `applyOriginalCellRecords`; `OriginalSaveResume` gains
`CellRecords`/`CellRecordsApplied`), `pkg/game/originalholdings.go`
(`restoreOriginalActorStock` defensively primes `SavedCellRecords` before
its own staging decode), `pkg/sim/originalliving.go`
(`ImportOriginalLivingActors` primes `SavedCellRecords` before its own
staging decode — the fix for the real defect), `pkg/sim/savedcellrecord_test.go`
(new: round-trip witness and the staging-loss regression witness).

`pkg/game/originalcellrecords1131_corpus_test.go` (new, build-tag
`sessioncorpusaudit`, opt-in acceptance instrument, later extended with
residue nonzero counters), `pkg/game/originalcellrecords1131_release_test.go`
(new: two release witnesses).

`internal/gatedtests/testdata/population.txt` (net +2: the two new release
witness names). `docs/DIVERGENCES.md` (DIV-730, DIV-026 amended; DIV-932,
DIV-933, DIV-934 new; DIV-926, DIV-927 population figures corrected as a
carried fix, not this story's own evidence).

No `formatVersion` or pinned-digest constant moved.
