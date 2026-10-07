# Story 1132 — SpellEffect graph importer

## Status

Gates run on the exact candidate commit; see "Proof" for the full list and
exact counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository, asset-free) clean, 49 packages, 0 FAIL, including
`internal/gatedtests`' checked-in `testdata/population.txt` census (net +2:
the two new release witnesses named below, 196 registered names —
`population.txt` itself is 200 lines, 4 of them comments); `scripts/check-no-game-assets.sh`
clean; `pipeline/check-release-tests.sh` against both lawful roots (8
packages, 196 gated tests, 196 of 196 ran and 0 lacked a subject on each of
EN and RU); `pipeline/check-milestone.sh` (seat script, `AGAINROM_MILESTONE_DRIVE`
pointed at a `missionrun` built from this worktree) exit 0, unchanged on both
roots — expected, since this story's own restoration only reaches an
original-save resume, a path the census's fresh mission drive does not
exercise.

## Player result

A mission SAV written by ROM1 carrying a non-empty top-level SpellEffect list
now loads in againrom with every SpellEffect record and its typed children
(SpellTransport, PointEffect, AreaEffect, Effect, Effect_DirectDamage) present
as live state, object identity and cross-references intact, instead of being
walked past. `game0018.sav`'s own witnessed shape — a root SpellTransport
whose first typed child is a PointEffect, whose own typed Effect is an
Effect_DirectDamage, with the transport's AreaEffect arm null
(`SAV-EFFECTGRAPH-366`) — restores exactly, and a native re-export reproduces
the source file byte for byte. Every other reachable owner save carries an
empty list (49 of 51 world-half saves in the available corpus); LOAD now
carries that shape correctly too instead of silently discarding what a
non-empty list would have held. VirtualCaster has a closed decode programme
but no promoted claim or corpus sample names a reachable reference site for
one (`SAV-CLASSSER-177`), so this story projects no VirtualCaster field.

No live simulation step reads or acts on this graph after LOAD: no promoted
claim names a post-load consumer (a live cast, an area-effect tick, a
transport delivery), so none is invented (DIV-939). PE44, a raw archive
identity key `PointEffect` carries, stays an opaque dword: the original's own
repair mechanism (`SAV-CLASSSER-174`) has a documented lookup rule but an
undocumented effect, and this project has no live target to resolve it into
(DIV-938). Native mission SAVE itself is not a wired player-facing feature
yet: no interactive path calls the native writer this story adds (see "Open
debt"). Mid-mission continuation remains exclusively the existing lossless
`.ags` envelope; this story's own native writer is a tested building block for
a later story in the same milestone sequence (`pipeline/SAV-COMPLETION.md`,
owner milestone 2, step 3 of the sequence).

## Authority

- **`SAV-DOC-053`** (High, amended): the whole document Serialize order,
  including the top-level SpellEffect list's own reader chain
  (`R1122` -> `R1109`).
- **`SAV-EFFECTGRAPH-366`** (High for structure): `game0018.sav`'s own
  witnessed graph shape — a root SpellTransport whose ST44 is a PointEffect
  whose own PE48 is an Effect_DirectDamage, ST48 null. Original load and
  resave remain Unknown beyond that structure.
- **`SAV-CLASSSER-172`..`177`**: close the six newly read Token-lineage
  classes' own programmes. 173 gives VirtualCaster (44 bytes: Token + 1 byte +
  6 raw bytes) and SpellEffect (39 bytes) and Effect_DirectDamage (68 bytes:
  Effect's 44 + 24 raw). 174 gives PointEffect (SpellEffect + one typed Effect
  reference + one raw identity dword at +0x44, repaired through the archive
  identity map at load via `R1108`, "what the repaired pointer means to
  simulation is Unknown"). 175 gives AreaEffect (SpellEffect + 4 raw bytes +
  one word + one typed Effect reference) and SpellTransport (SpellEffect + two
  typed references, ST44 then ST48 + one trailing word). 177 is a corpus
  census (55 paths, 31 SHA-distinct saves): one distinct nested
  SpellTransport -> PointEffect -> Effect_DirectDamage graph and zero records
  or raw-name hits for VirtualCaster, bare SpellEffect, AreaEffect, Outpost or
  Tavern.
- **`SAV-EFFCHAIN-046`**: `Effect::Serialize` (`L08329`) is Token(37) + u8
  `+0x3c` + u8 `+0x3d` + u32 `+0x40` + u8 `+0x0c` = 44 bytes; an Effect is an
  element of the `+0x20` list of the object it is attached to.
- **`SAV-SPELL-044`**, **`SAV-EQUIPEFFECT-553`**: adjacent context — the
  unrelated `Spell` class shares the same archive identity map as Token
  (`SAV-SPELL-044`); an Item's own Effect/Effect_DirectDamage list is walked
  forward and incrementally, not batch (`SAV-EQUIPEFFECT-553`) — neither
  names a field of the top-level SpellEffect graph this story projects.

## As-built behaviour

**`pkg/formats/sav`** (`spelleffect.go`, `document.go`): `File.SpellEffects()`
returns the typed top-level list in archive order (`[]SpellEffect`, one entry
per graph root), distinguishing a no-world save (`present=false`) from a
world save with an empty list. `SpellEffect` carries `Class`, `SE40`/`SE41`
(every subtype), and the per-subtype fields named by
`SAV-CLASSSER-174`/`175`: `PE48 *Effect`/`PE44 uint32` for PointEffect,
`AE48 [4]byte`/`AE4C uint16`/`AE44 *Effect` for AreaEffect, and
`ST44 *SpellEffect`/`ST48 *SpellEffect`/`ST4C uint16` for SpellTransport.
`Effect` carries `Class`, `E3C`/`E3D`/`E40`/`E0C`, and `DirectDamage [24]byte`
for Effect_DirectDamage. Object identity at the archive layer is preserved by
the walker's own existing back-reference cache (`document.go` retains
`doc.effects` instead of discarding the decoded list, the only change to that
file): a back-reference and its first introduction share one `*Record`. The
typed `SpellEffect`/`Effect` projection built on top of it does not carry that
identity forward: `spellEffectFromRecord`/`effectFromRecord` allocate a fresh
value at every reference site with no memo keyed on `*Record`, so two graph
positions that share one archive object receive two distinct, equal-content
Go pointers here, and the pointer-keyed converter in `pkg/game` that is
written to restore the identity can never see the repeat either (review F-1;
DIV-939).
`File.SetSpellEffects` patches every subtype's scalar fields in place, row for
row and reference for reference against the file's own current graph shape,
and recurses into a freshly-introduced nested reference
(`refSpan`); a null or back-reference is left exactly as the file already has
it. A layout test (`TestSpellEffectGraphReadsEveryClassAndField`) locks every
class's own byte width; a leave-alone test
(`TestSetSpellEffectsLeavesUnrelatedBytesAlone`) locks that the writer never
touches a byte outside the list's own span. Fixing `patchSpellEffect`/
`patchEffect` required computing each record's field offset as `r.Off +
TokenLen` rather than `r.Off` directly: `Class.Fixed()`'s own rule is that a
`Head:true` class's `Off` is the start of its embedded 37-byte Token head, not
its own first named field, the same fact every other class-programme
consumer in this package already accounts for.

**`pkg/sim`** (`savedspelleffect.go`, `world.go`, `binary.go`,
`originalliving.go`): `SavedSpellEffect`/`SavedEffect` are `pkg/sim`'s own
parallel types (no `pkg/formats/sav` import — that package must never depend
on the decoder). `World.savedSpellEffects` is carried-not-wire-form, the same
choice `savedCellRecords` (story 1131) already made and for the same reason:
no rule states how to recompute this state from anything else, so
`UnmarshalBinary`'s composite literal carries the receiver's own prior value
across a decode rather than reading it from the encoded bytes, and a decode
into a fresh receiver needs priming first (`SetSavedSpellEffects`) or the
value is silently lost. `ImportOriginalLivingActors`
(`pkg/sim/originalliving.go`) and `restoreOriginalActorStock`
(`pkg/game/originalholdings.go`) both stage every original actor admission
through exactly that round trip and both now prime it, on `savedCellRecords`'
own precedent.

**`pkg/game`** (`originalspelleffects.go`, `originalsave.go`):
`originalSpellEffects`/`applyOriginalSpellEffects` decode the file's own list
and convert it into `sim.SavedSpellEffect`/`sim.SavedEffect` values through a
per-call identity-preserving converter (`spellEffectConverter`, a
`map[*sav.Effect]*sim.SavedEffect` / `map[*sav.SpellEffect]*sim.SavedSpellEffect`
cache keyed by pointer, so a shared or back-referenced object converts once
and every reference site shares one Go value); `exportOriginalSpellEffects`
converts the reverse direction (`savedSpellEffectConverter`) and calls
`SetSpellEffects`. Both original LOAD doors in `originalsave.go`
(`ResumeOriginalSave` and `RestoreOriginal`) decode and apply the graph in the
same position `applyOriginalCellRecords` already occupies, and `OriginalSaveResume` reports `SpellEffects`/`SpellEffectsApplied`
alongside the existing cell-record fields. `exportOriginalSpellEffects` is not
called from any production SAVE entry point; it is a tested round-trip
building block for a later story, on `exportOriginalCellRecords`'s own
precedent (see "Open debt").

## Divergence rows

- **DIV-938** (FIDELITY-DEBT, OPEN): PE44's own repair mechanism
  (`SAV-CLASSSER-174`) is documented but not built — no live PointEffect
  target exists in this project to resolve the archive key into, and even a
  built repair would have no documented use, since the same claim leaves what
  a resolved pointer means to simulation Unknown. The raw dword is carried
  opaque and round-trips byte for byte (0 mismatching over the corpus, both
  roots).
- **DIV-939** (UNKNOWN, OPEN): no promoted claim names a post-load consumer
  of any SpellEffect-graph object — what triggers a live cast, an
  area-effect tick, a transport delivery, or a save-time relocation, and
  when. The complete typed graph is imported and re-exported; nothing in this
  project reads or acts on it during play.
- **DIV-940** (UNKNOWN, OPEN): VirtualCaster has a closed decode programme
  (`SAV-CLASSSER-172`/`173`) but zero records or raw-name hits anywhere in
  the permitted corpus and no promoted claim naming a reference site
  (`SAV-CLASSSER-177`). This story's typed projection surfaces no
  VirtualCaster field, since one would be invented reach.
- DIV-941 through DIV-943 (reserved, unused) are returned unclaimed.
- DIV-933/934 (story 1131, cell-record table's own Sack/SpellEffect identity
  keys) are a different, narrower surface — per-cell archive keys, not this
  story's top-level graph — and are unchanged by this story.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean
  (net +2, 196 registered names; `population.txt` is 200 lines, 4 of them
  comments).
- `scripts/check-no-game-assets.sh`: clean.
- `TestSpellEffectCorpusAudit1132` (`-tags sessioncorpusaudit`, opt-in, both
  lawful roots): 72 files audited (51 world-half, 20 between-mission, 1
  unreadable), 2 non-empty SpellEffect list(s), 0 mismatching, on EN and again
  on RU (identical log line both roots: `audited 72 file(s): 51 world-half,
  20 between-mission, 1 unreadable, 2 non-empty SpellEffect list(s), 0
  mismatching`). The 2 non-empty subjects are both copies of `game0018.sav`.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 196 gated
  tests, 2 roots; **196 of 196 ran and 0 lacked a subject, on EN and again on
  RU.**
- `pipeline/check-milestone.sh` (seat script, run because a restored graph
  could in principle change a shipped map's tick outcome;
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

**Native mission SAVE has no interactive path yet.** `exportOriginalSpellEffects`
is exercised only by this story's own corpus audit and release witnesses. No
production menu, hotkey or session flow calls it; the owner's own standing
rule against inventing a production mission SAVE entry point applies here the
same way it did to `exportOriginalCellRecords` in story 1131. The building
block is tested and correct; wiring it to a player-visible SAVE action is a
separate, later decision.

**PE44's repaired meaning and the graph's post-load consumer are both
Unknown** (DIV-938, DIV-939). Neither is simulated by guess. A future story
that needs either behaviour needs a claim first.

**VirtualCaster has no reachable typed site** (DIV-940). Its own decode
programme in `pkg/formats/sav/program.go` is pre-existing and unchanged; this
story's typed projection does not surface it because no evidence names where
it is referenced from.

**The corpus's only non-empty sample is one graph shape.** `game0018.sav`
(both roots) is the sole witnessed non-empty SpellEffect list in the
available corpus (`SAV-CLASSSER-177`'s own census, corroborated by this
story's corpus audit). AreaEffect's own typed reference and a bare
(non-Point/Area/Transport) SpellEffect are decoded and round-trip through the
unit tests in `pkg/formats/sav/spelleffect_test.go`, which are hand-built from
the claims (golden rule 2), but neither has a real corpus witness.

**The `.ags` resume drops this graph while nothing reads it.** `resumeWorld`
(`pkg/game/resume.go:717`) decodes a resumed session's world through a fresh
`UnmarshalBinary`, and `savedSpellEffects` is carried-not-wire-form (see
"As-built behaviour"): the fresh receiver starts empty and nothing primes it
on this path, the same gap DIV-932 names for `savedCellRecords`. Driven end
to end (original LOAD of a non-empty save, ordinary SAVE, fresh LOAD): the
top-level SpellEffect count goes from 1 to 0 while `World.Hash()` is
unchanged, since no simulation step reads this field today. `exportOriginalSpellEffects`
fails closed on such a world (`sav: 0 SpellEffect values for 1 records`)
rather than writing an empty list over a populated one.

**The carried graph holds each class's own fields past its embedded Token
head, not the object.** 41 of the 196 bytes of the only real corpus graph
(`game0018.sav`) are projected; the other 155 are three 37-byte Token heads,
two class-introduction spans, and the 2-byte null `ST48` arm. The native
re-export is byte-identical only because it patches the source file's own
bytes in place — the carried state alone could not rebuild those 155 bytes
(the spell object's own cell, fine position, state and identity fields among
them), so an in-flight spell's own position is not carried.

**Neither staging prime has an asset-free witness.** `checked.SetSavedSpellEffects`
(`pkg/sim/originalliving.go`) and `staged.SetSavedSpellEffects`
(`pkg/game/originalholdings.go`) are both load-bearing — reverting either
fails `TestReleaseOriginalSpellEffectGraphRestoresOnLoad1132` — but only that
install-gated release witness catches it; `go test ./pkg/sim ./pkg/game ./pkg/formats/sav`
stays green on either revert alone. Story 1131 has a `pkg/sim` unit witness
for the same shape (`TestImportOriginalLivingActorsStagingCarriesSavedCellRecords`);
this story has none.

## Touched surfaces

- `pkg/formats/sav/spelleffect.go` (new): typed projection and writer.
- `pkg/formats/sav/spelleffect_test.go` (new): unit tests built from claims.
- `pkg/formats/sav/document.go`: retain the decoded top-level list instead of
  discarding it.
- `pkg/formats/sav/program_test.go`: `walkFixture.spellEffects` hook.
- `pkg/sim/savedspelleffect.go` (new): `SavedSpellEffect`/`SavedEffect` and
  `World` accessors.
- `pkg/sim/world.go`: `savedSpellEffects` field.
- `pkg/sim/binary.go`: carry it across `UnmarshalBinary`'s composite literal.
- `pkg/sim/originalliving.go`: prime it before `ImportOriginalLivingActors`'
  own staging decode.
- `pkg/sim/nostate_test.go`, `pkg/sim/world_test.go`: field/method-set pins.
- `pkg/game/originalspelleffects.go` (new): the sav<->sim converters, LOAD and
  export entry points.
- `pkg/game/originalsave.go`: wire both original LOAD doors and
  `OriginalSaveResume`'s report/`String()`.
- `pkg/game/originalholdings.go`: prime the same carried field before
  `restoreOriginalActorStock`'s own staging decode.
- `pkg/game/originalspelleffects1132_test.go` (new, untagged):
  `sameSpellEffectGraphForAudit`, shared by the corpus audit and the release
  witnesses.
- `pkg/game/originalspelleffects1132_corpus_test.go` (new,
  `sessioncorpusaudit`-tagged): `TestSpellEffectCorpusAudit1132`.
- `pkg/game/originalspelleffects1132_release_test.go` (new):
  `TestReleaseOriginalSpellEffectGraphRestoresOnLoad1132`,
  `TestReleaseEmptySpellEffectListStaysEmptyOnLoad1132`.
- `internal/gatedtests/testdata/population.txt`: register both new witnesses.
- `docs/DIVERGENCES.md`: DIV-938, DIV-939, DIV-940.
- `docs/1132/story.md`: this file.
