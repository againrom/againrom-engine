# Story 1139 — the native `.ags` resume carries every original-LOAD structure

## Status

Gates run on the exact candidate commit, `dbd23a6` on
`story/1139-native-resume-carried-state`, rebased onto current `origin/main`
(`c73801d`, story 1138's own landing — an ancestor of this branch, confirmed
by `git merge-base --is-ancestor origin/main HEAD`). See "Proof" for the full
list and exact counts. `gofmt -l .` clean; `go build ./...` clean;
`go test -trimpath -count=1 ./...` (whole repository, asset-free) clean, 75
packages (49 `ok`, 26 `[no test files]`), 0 FAIL; `go vet ./...` clean except
eleven pre-existing `BookSpell` unkeyed-field warnings in five files this
branch does not touch (confirmed byte-identical to `origin/main`);
`scripts/check-no-game-assets.sh` clean; `pipeline/check-release-tests.sh`
against both lawful roots (`AGAINROM_IMPL` pointed at this worktree): 8
packages, 208 gated tests, 2 roots, **208 of 208 ran and 0 lacked a subject on
EN and again on RU**; `TestNativeResumeCorpusCycle1139`
(`-tags sessioncorpusaudit`, fresh run, both roots): `audited 72 file(s): 51
world-half, 20 between-mission, 1 unreadable, 0 mismatching`, identical on EN
and RU; mission 10/20 `-trace -ticks 1` UNSUPPORTED-node count 0 on both
missions, both roots, unchanged from master's own recorded census (no
`cannot run` line for either mission, `pipeline/milestone-baseline.txt`) —
this story's own change is reached only through an original-save resume, a
path the fresh mission drive does not exercise (stories 1136/1137's own
precedent). No claim was read, cited, or created; this story changes only the
engine's own persistence architecture.

One defect was found and fixed while running the seat's own release gate
(not by `go test ./...`, which cannot reach it): see "As-built behaviour" and
commit `dbd23a6`.

## Player result

A mission loaded from an original ROM1 SAV, then saved through the ordinary
in-game menu SAVE (a native `.ags`), then loaded again from that `.ags` in a
fresh process, now keeps every structure the original LOAD imported that
`pkg/sim/world.go` marks carried-not-wire-form: the two raw session spans
(story 1130), the cell-record residue (story 1131, DIV-932), the SpellEffect
graph (story 1132, DIV-938/DIV-939), the Projectiles store (story 1133,
DIV-944) and Diaries (story 1135, DIV-956). Before this story, `UnmarshalBinary`
carried each of these across a decode from the RECEIVER's own prior value
rather than reading it from the byte form, so any decode onto a fresh
receiver — every genuine cross-process native `.ags` resume, and every
read-only save instrument — silently zeroed all five; only the
same-receiver decode inside `resumeWorld` (`pkg/game/resume.go`) kept them,
and only because nothing else touched that receiver first. A later native
export of the resumed world (`exportOriginalCellRecords`,
`exportOriginalSpellEffects`, `exportOriginalProjectiles`,
`exportOriginalDiaries`, `exportOriginalMissionSession`) is now byte-identical
to the export taken before the SAVE/LOAD cycle, proven over the full
preserved save corpus and by one full-Go release witness (see "Proof").

Format 85 (`formatVersion` 84 → 85) is the wire-form change that makes this
true: it appends every one of the five fields as one outermost span after
form84's own payload, on the same "absent span costs 4 bytes" convention
form78 through form84 already established. An older `.ags` (any form 1
through 84) still loads, through `UpgradeSaveForm`, exactly as before; this
story registers form84 in `upgradeSteps` so form85 correctly reports it
readable.

## Authority

No new claim. This story is an engine-internal persistence-architecture
change: it neither derives nor asserts any new fact about ROM1. Each field it
gives a wire position keeps the authority its own introducing story already
cited, unchanged by this story: `SAV-SESS-031` for the two raw session spans
(docs/1130/story.md), `SAV-CELLLOAD-111` for the cell-record residue
(docs/1131/story.md), `SAV-DOC-053`/`SAV-CLASSSER-172`..`177` for the
SpellEffect graph (docs/1132/story.md), `SAV-PROJSTORE-428`/`SAV-PROJLOAD-429`
for the Projectiles store (docs/1133/story.md), and `SAV-667`/`SAV-668`/
`SAV-669` for Diary content (docs/1135/story.md). Form85's own byte layout is
an engine serialization decision, not a claim about ROM1, and cites none.

## As-built behaviour

**`pkg/sim/carriedresumebinary.go`** (new): `appendCarriedResumeState`/
`splitCarriedResumeState` write and read Form85's own outermost span, on
form84's own trailing-span convention (payload, then a `u32` little-endian
byte count of that payload as the very last four bytes; a `u32` zero alone
means every field below is absent/empty). `carriedResumeStateEmpty` is what
`MarshalBinary` asks before choosing the four-byte empty form, so a mission
that was never resumed from an original save — the overwhelming common case —
costs four bytes, not this section's own non-empty minimum of 470. Present
payload, in this fixed order:

| Field | Wire shape |
|---|---|
| RawSessionHead | 48 raw bytes |
| RawSessionMid | 400 raw bytes |
| CellRecords | `u32` count, then that many 52-byte records (Cell `u16`, LayerCount `u8`, Residue0 `u8`, Residue1 2 raw bytes, Ground and Air: Key `u32`, Entity `u32`, Bound `u8` each, Sack `u32`, six `u32` SpellEffect keys), strictly ascending by Cell, no duplicate key |
| SpellEffects | `u32` count, then that many spell-effect nodes: 15 fixed bytes (class tag `u8`, SE40, SE41 `u8` each, PE44 `u32`, AE48 4 raw bytes, AE4C `u16`, ST4C `u16`), then four pointer slots (PE48, AE44, ST44, ST48 order), each a presence byte and, if present, PE48/AE44's fixed 33-byte Effect body or ST44/ST48's own nested spell-effect node |
| Projectiles | `u16` FreeIndex, `u32` IDs count + that many `u16` ids, `u32` Items count + that many 66-byte records; no two Items share an ID, IDs may repeat or name an id Items lacks |
| Diaries | `u32` count, then that many records: Player `u8` (0/1), Actor `u32` (zero when Player), Length `u32`, Entries count `u32` + that many (Index `u32`, Count `u32`, Remaining `u16`) triples, strictly ascending by Index, `0<=Index<Length`; at most one Player owner, no repeated Actor owner |

A decoded node carrying nonzero off-class state is refused
(`savedSpellEffectShapeFault`), the same rule every other section in this
form already applies to a byte the encoder could not have written. Shared
SpellEffect identity inside the graph is not reconstructed: `carriedresumebinary.go`'s
own doc records that the archive's own back-references are already lost
upstream, before Form85 ever sees the graph
(`spellEffectFromRecord`/`effectFromRecord` in `pkg/formats/sav` allocate a
fresh value per reference site), so the wire form neither preserves nor
invents sharing — this independently confirms DIV-939 review F-1's own
finding at the wire-form level (see "Divergence rows").

**`pkg/sim/upgrade.go`**: registers form84 in `upgradeSteps` so
`UpgradeSaveForm` can still carry an old form84 `.ags` up to form85 (widening
by exactly the new empty span), the same step every prior format bump added
for its own predecessor.

**Priming removed** (`ImportOriginalLivingActors`, `pkg/sim/originalliving.go`;
`restoreOriginalActorStock`, `pkg/game/originalholdings.go`): both stage
their own recompute through a `MarshalBinary`/fresh-`UnmarshalBinary` round
trip, and both used to prime the fresh receiver by hand before that decode,
because the five fields above were not yet in the byte form.
`restoreOriginalActorStock` primed `SetRawSessionHead`, `SetRawSessionMid`,
`SetSavedCellRecords`, `SetSavedSpellEffects`, `SetSavedProjectiles` and
`SetSavedDiaries`; `ImportOriginalLivingActors` primed
`SetSavedCellRecords`, `SetSavedSpellEffects`, `SetSavedProjectiles` and
`SetSavedDiaries` (never the two raw session spans, which that narrower
staging check never carried). All ten calls are removed: `staged.UnmarshalBinary`/
`checked.UnmarshalBinary` now read every field from the same bytes the
preceding `MarshalBinary` just produced, so priming is redundant. Of the six
setters, `SetSavedSpellEffects` keeps its one remaining production caller,
`applyOriginalSpellEffects` (the original LOAD import path, never a staging
round trip); `SetSavedCellRecords`, `SetSavedDiaries` and `SetSavedProjectiles`
keep no production caller but each keep one direct-construction test caller
in their own package; `SetRawSessionHead` and `SetRawSessionMid` keep no
caller at all, production or test — referenced only by name in
`world_test.go`'s reflective method-set completeness pin. None of the six are
deleted (see "Open debt").

Every doc comment that described a removed caller or called one of these six
fields carried-not-wire-form was corrected in the same change, in
`pkg/sim/world.go`, `pkg/sim/script.go`, `pkg/sim/savedcellrecord.go`,
`pkg/sim/saveddiary.go`, `pkg/sim/savedprojectile.go`,
`pkg/sim/savedspelleffect.go`, `pkg/game/originaldiaries.go`,
`pkg/game/originalsave.go`, `pkg/game/resume.go` and
`pkg/game/originalsave_test.go` — describing the current mechanism, not
rewriting the historical narrative of the defects story 1130/1131/1132/1133/1135
each found and fixed (PROSE.md's frozen-history rule applies to those
records, not to live doc comments about current behaviour).

**Hash pin** (`pkg/sim/hash_test.go`): `pinDigest` moves from
`0xf8eeb7a2b2bd91df` (form84) to `0x8a439cba352d8ae0` (form85), in the same
commit as the format bump (`b539e4b`). `Hash()` is `FNV-1a` over
`w.encode()`, which always encodes the current format version, so every
stored pin moves with a format bump regardless of whether the pinned world
itself exercises the new span; the pinned world here carries none, so only
the version byte and the new four-byte empty footer move the digest.
`TestHashIsPinned`, `TestThePinnedDigestIsFNV1aOfThePinnedBytes` and
`TestHashIsFNV1aOverExactlyTheByteForm` all pass against the new literal.

**A stale form-check regression, found and fixed** (`dbd23a6`,
`pkg/game/originaldead_release_test.go`): `assertTerminalWeapons1100` reads
its own live world's `MarshalBinary()` output and manually peels the
form84 object-registry span before delegating into the shared
`beforeStrideStateForm1115` peel chain, because that chain's own
`beforeObjects1115Form` step asserts the object-registry span is exactly
absent — correct for its actual callers (frozen pre-1139 predecessor
fixtures) but wrong for this witness, whose world may now legitimately carry
a live gold Sack after a fresh original import (the existing comment already
says so). The peel's own version check was `form[0] == 84`; since this story
every `MarshalBinary()` output is form85, that check never fired, so `form`
reached the shared chain still carrying BOTH the unpeeled object-registry
span AND Form85's own new carried-resume span — and this specific fixture
(`game0009.sav`, mission 20, resumed through `ResumeOriginalSave`) genuinely
carries non-empty cell-record residue and Diaries in that span, so
`beforeObjects1115Form`'s absence assertion fired on real, correct content,
failing with "pre-carried-resume native state acquired a populated resume
footer" on both EN and RU. The fix adds the same unconditional strip-whatever-
it-holds peel for Form85's own span, ahead of the existing form84 peel,
mirroring its shape exactly. `pipeline/check-release-tests.sh` found this
because it runs the install-gated test population; plain
`go test -trimpath -count=1 ./...` cannot reach it, since this test is gated
behind `AGAINROM_ASSETS`/`AGAINROM_SAVE_CORPUS` and is skipped without them.
A repository-wide grep for the same `form[0] == 84`/`raw[0] == 84` pattern
found three other matches, all inside the shared `before...Form1115` peel-chain
family itself (already correctly widened to accept forms up to 85 by the
foundational commit) — no other call site shares this defect.

**Two new acceptance instruments**:
`TestNativeResumeCorpusCycle1139` (`pkg/game/nativeresumecycle1139_corpus_test.go`,
`-tags sessioncorpusaudit`, on `TestGroundContainerCorpusAudit1136`'s own
shape) drives the exact cycle — original LOAD, `MarshalBinary`, fresh
`UnmarshalBinary` onto a receiver that ran no `Import*`/`apply*` call of its
own — over every preserved save with a world half, and compares
`exportOriginalMissionSession`/`exportOriginalCellRecords`/
`exportOriginalSpellEffects`/`exportOriginalProjectiles`/`exportOriginalDiaries`'s
own output byte for byte between a pre-cycle and a post-cycle decode of the
same source file. None of the five export functions loosens a check to pass
this comparison: each is unconditional, content-agnostic projection code, so
a pass here is proof the underlying carried values are equal, not that a
guard was relaxed. `TestReleaseNativeResumeCarriesSpellEffectsAndCellRecords1139`
(`pkg/game/nativeresume1139_release_test.go`) is the one full-Go release
witness, on `originalholdings_release_test.go`'s own `holdingsNativeFresh`
shape (original LOAD, ordinary App menu SAVE, fresh App LOAD in a brand-new
FrontEnd/App/World, with its own `Hash()` equality check), additionally
naming the SpellEffect count and cell-record residue count explicitly so a
future regression here fails on a legible structure, not only an opaque hash
mismatch.

## Divergence rows

- **DIV-926** (raw session span carry-across; narrowed, remains OPEN): AS-BUILT
  now records that Form85 gives both spans a wire position outright, so no
  decode loses them, with `TestNativeResumeCorpusCycle1139`'s corpus result
  cited. REMAINDER no longer asks for the wire-position gap (closed); it now
  names only DIV-041's own ghost template as a separate, unrelated field this
  story does not touch. STATUS: `OPEN (narrowed: the wire-position gap this
  row's own remainder column asked for is closed ... DIV-041's own ghost
  template is a different field this story does not touch and remains open
  on its own)`.
- **DIV-932** (cell-record layer count/residue carry-across; narrowed, remains
  OPEN): same pattern. REMAINDER now names only the two pre-existing,
  unrelated open questions (SavedActorCell folding, review F-1; the dead-actor
  Ground key question, review F-5) instead of the now-closed wire-position ask.
- **DIV-938** (SpellEffect PE44 identity-key opacity; STATUS parenthetical
  updated, DEVIATION/remainder unchanged): records that Form85 closes the
  `resumeWorld`-drops-the-graph gap and corrects the now-false claim that "the
  world's digest is unaffected" — the digest now covers this state, since
  `Hash()` is FNV-1a over the current byte form. PE44's own resolved meaning
  is exactly as unproven as before this story.
- **DIV-939** (SpellEffect shared identity; stays plain `OPEN`, unnarrowed):
  AS-BUILT gains one sentence recording that Form85 independently
  confirms review F-1's own finding — no live sharing remains in memory for
  the wire form to preserve or invent — at the wire-form level too.
  Classification (`UNKNOWN`) and remainder unchanged.
- **DIV-944** (Projectiles store; STATUS parenthetical updated): stale
  "primed across both staging round trips" language removed from AS-BUILT;
  records the wire position and the unchanged DEVIATION (no live driver for
  any of the sixteen leaves).
- **DIV-956** (Diary content; STATUS parenthetical updated): same pattern as
  DIV-944.

All six rows were checked for the ledger's own pipe-count invariant (10 pipes
per row) after editing, and no other row in the file references the
carried-not-wire-form gap, the fresh-receiver zeroing, or a `.ags` resume drop
(repository-wide grep, zero further matches). `DIV-974` and `DIV-975` were
reserved for this story (`pipeline/ALLOCATIONS.md`) and are **both unused**:
this story updates six existing rows and creates none: `pipeline/next-div-id.sh`
was not run.

## Proof

- `gofmt -l .`: clean.
- `go build ./...`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  75 packages (49 `ok`, 26 `[no test files]`), 0 FAIL.
- `go vet ./...`: clean except eleven pre-existing `BookSpell` unkeyed-field
  warnings across five files (`originalactorbooks_release_test.go`,
  `originalactorbooks_test.go`, `originalholdings_composition_test.go`,
  `originalprofile_holdings_test.go`, `originalspellbook1101_test.go`); `git
  diff --stat origin/main..HEAD` on each confirms zero changes, so these
  predate this story.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean.
  `population.txt` moves 207 → 208 registered names (one addition: this
  story's own release witness; the other two names already at the file's own
  tail, `TestReleaseOriginalTrailerRoundTripsOnLoad1138` and
  `TestReleaseOriginalUnitResidualFieldsRestoreOnLoad1138`, are story 1138's,
  already landed on `origin/main` before this branch was cut).
- `scripts/check-no-game-assets.sh`: clean.
- `TestNativeResumeCorpusCycle1139` (`-tags sessioncorpusaudit`, fresh run,
  both roots): `audited 72 file(s): 51 world-half, 20 between-mission, 1
  unreadable, 0 mismatching`, identical on EN and RU — every reachable
  world-half preserved save round-trips through original LOAD → native SAVE →
  native LOAD with byte-identical `exportOriginalMissionSession`/
  `exportOriginalCellRecords`/`exportOriginalSpellEffects`/
  `exportOriginalProjectiles`/`exportOriginalDiaries` output before and after.
- `TestReleaseNativeResumeCarriesSpellEffectsAndCellRecords1139` (fresh run,
  both roots, `game0018.sav`): `SpellEffect count=1 cell-record residue
  count=116: both equal before and after the native SAVE/LOAD cycle`,
  identical on EN and RU, plus `holdingsNativeFresh`'s own `Hash()` equality
  check across the whole native SAVE/LOAD cycle.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 208 gated
  tests, 2 roots; **208 of 208 ran and 0 lacked a subject, on EN and again on
  RU.** This run is what found the `assertTerminalWeapons1100` regression
  (see "As-built behaviour"); after the fix, both roots are clean.
- `go build -o <tmp>/mr ./cmd/missionrun` plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master's
  own recorded census, which carries no `cannot run` line for either mission
  (`pipeline/milestone-baseline.txt`). This story's change is reached only
  through an original-save resume, a path the fresh mission drive does not
  exercise, on stories 1136/1137's own precedent for the same population.

## Open debt

**Five setters keep no production caller.** `SetRawSessionHead`,
`SetRawSessionMid`, `SetSavedCellRecords`, `SetSavedDiaries` and
`SetSavedProjectiles` (all `pkg/sim`) lost their only production caller when
priming was removed. `SetSavedCellRecords`, `SetSavedDiaries` and
`SetSavedProjectiles` each keep exactly one direct-construction test caller
in their own package; `SetRawSessionHead` and `SetRawSessionMid` keep no
caller at all, referenced only by name in `world_test.go`'s reflective
method-set completeness pin. None are deleted: each remains a legitimate
direct-set primitive for a future caller (e.g., a from-scratch World
constructor that does not decode from bytes), and removing an exported method
with an existing pinned method-set test was judged out of this story's scope.
A future story that finds no remaining reason for one may remove it and its
pin row together.

**Shared SpellEffect identity is still not reconstructed** (DIV-939,
unchanged, stays OPEN). Form85 independently confirms review F-1's own
finding — the archive's own back-references are already lost upstream of the
wire form, so nothing is preserved or invented at this layer either. This
story does not attempt a rebind mechanism; DIV-939's own open question (a
claim naming a live post-load consumer of the graph) is unmet and unrelated
to this story's own scope.

**PE44's resolved meaning stays unproven** (DIV-938, unchanged). The wire
form now carries it through every resume path, but what a resolved PE44
pointer would mean to simulation remains Unknown per `SAV-CLASSSER-174`
itself; this story changes only whether the raw dword survives a native
resume, not what it means.

**No live consumer reads the Projectiles store or Diary content**
(DIV-944/DIV-956, unchanged). Both are carried and now survive a native
resume losslessly, but this build has no persistent-projectile entity and no
journal/quest-counter UI surface; neither store is read, ticked or displayed
during play.

**Native mission SAVE still has no interactive path** (unchanged, stories
1130-1137's own standing note). Every `exportOriginalXXX` function above,
including the mission-session writer, remains exercised only by the corpus
audit and the release witness; the owner's own standing rule against
inventing a production mission SAVE entry point still applies.

## Touched surfaces

- `pkg/sim/carriedresumebinary.go`: new — Form85's own append/split.
- `pkg/sim/binary.go`, `pkg/sim/upgrade.go`: format bump, `UnmarshalBinary`
  no longer carries the five fields across a decode, form84 registered in
  `upgradeSteps`.
- `pkg/sim/world.go`, `pkg/sim/script.go`, `pkg/sim/savedcellrecord.go`,
  `pkg/sim/saveddiary.go`, `pkg/sim/savedprojectile.go`,
  `pkg/sim/savedspelleffect.go`: doc-comment correction only (no behaviour
  change) for the fields Form85 now carries.
- `pkg/sim/originalliving.go`: `ImportOriginalLivingActors` no longer primes
  four setters before its staging round trip.
- `pkg/sim/sourcebinding.go`: `widenSourceBindings` (the pre-71 widening
  helper) appends Form85's own absent carried-resume span after form84's, on
  the same pattern as every span before it.
- `pkg/sim/hash_test.go`: `pinDigest` moved to the form85 literal; doc comment
  extended with the Form85 predecessor-pin note.
- 27 more `pkg/sim/*_test.go` files (byte-length/hash-constant fixtures that
  shift by exactly one version byte and, where the fixture's own form reaches
  84 or higher, four more for the new empty span): mechanical constant
  updates only, no assertion logic changed; `git diff --stat origin/main..HEAD`
  is the exact list.
- `pkg/game/originalholdings.go`: `restoreOriginalActorStock` no longer
  primes six setters before its staging round trip.
- `pkg/game/originaldiaries.go`, `pkg/game/originalsave.go`,
  `pkg/game/resume.go`, `pkg/game/originalsave_test.go`: doc-comment
  correction only.
- `pkg/game/originaldead_release_test.go`: `assertTerminalWeapons1100` fixed
  to peel Form85's own span (the regression found by
  `pipeline/check-release-tests.sh`; see "As-built behaviour").
- `pkg/game/nativeresumecycle1139_corpus_test.go`,
  `pkg/game/nativeresume1139_release_test.go`: new — this story's own
  acceptance instruments.
- 9 more `pkg/game/save_before_*_test.go`/`sav*1115_test.go`/
  `savcellstate1115_test.go`/`savdocument_composed_test.go`/`save_test.go`
  files, plus `cmd/savemigrate/main_test.go`, `cmd/saverepair/main_test.go`
  and `pkg/mapload/{fromalm,gridform}_test.go` (13 total): mechanical
  fixture-length/hash adaptation for the format bump, and
  (`save_before_objects1115_test.go`, `save_before_cell_state_test.go`,
  `save_before_crossing_state_test.go`, `save_before_stride_state_test.go`)
  the shared peel-chain widened to accept forms up to 85.
- `internal/gatedtests/testdata/population.txt`: +1 registered name (this
  story's own release witness).
- `docs/DIVERGENCES.md`: DIV-926, DIV-932, DIV-938, DIV-939, DIV-944,
  DIV-956 (all updated; all remain OPEN).
- `docs/1139/story.md`: this file.
