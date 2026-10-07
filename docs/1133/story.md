# Story 1133 — Projectile store importer

## Status

Gates run on the exact candidate commit; see "Proof" for the full list and
exact counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository, asset-free) clean, 49 packages, 0 FAIL, including
`internal/gatedtests`' checked-in `testdata/population.txt` census (net +2:
the two new release witnesses named below, 198 registered names —
`population.txt` itself is 202 lines, 4 of them comments);
`scripts/check-no-game-assets.sh` clean; `pipeline/check-release-tests.sh`
against both lawful roots (8 packages, 198 gated tests, 198 of 198 ran and 0
lacked a subject on each of EN and RU); mission 10/20 `-trace -ticks 1`
UNSUPPORTED-node count 0 on both missions, both roots, unchanged from master
— this story's own restoration is reached only through an original-save
resume, a path the fresh mission drive does not exercise (story 1132's own
precedent, "Proof").

## Player result

A mission SAV written by ROM1 with a projectile in flight now loads in
Againrom with that projectile's store restored: the manager's allocator
(`Projectiles/FreeIndex`), the live id set (`Projectiles/IDs`) and every
`Prj<id>` node's sixteen leaves are present in `sim.World`'s carried state
before the first tick, instead of being decoded and dropped. The corpus's
one witnessed non-empty case, `game0018.sav` (id 266, FreeIndex 267),
restores exactly on both original LOAD doors, and a native re-export
reproduces the decoded value. Every other reachable owner save carries an
empty store (49 of 51 world-half saves in the corpus this project can read);
LOAD now carries that shape correctly too instead of silently discarding
what a non-empty store would have held.

No live simulation step reads or acts on a restored projectile: this build
has no persistent projectile entity (`pkg/game/projectiles.go` is a cosmetic
art loader and cast-event renderer, not a live object), so every leaf stays
carried opaquely (DIV-944). Native mission SAVE itself is not a wired
player-facing feature yet: no interactive path calls the native writer this
story adds (see "Open debt"). Mid-mission continuation remains exclusively
the existing `.ags` envelope, which does not carry this store across a
resume (see "Open debt").

## Authority

- **`SAV-PROJSTORE-428`** (High, promoted): the producer's own canonical
  shape — `FreeIndex` kind 2 from the manager word at `world+0xa0c`, one
  `Prj<id>` section per live node with sixteen kind-2 leaves, `IDs` kind 6
  (one u32 per live u16 id, zero-extended) — always emitted, even for an
  empty collection.
- **`SAV-PROJLOAD-429`** (High, promoted): the loader is stricter about wire
  type than about record presence — `FreeIndex`/`IDs` each default
  independently when absent, `IDs` accepts kind 6 or a kind-2 singleton
  compatibility arm, a present id's section defaults a missing leaf from the
  constructed object rather than refusing.
- **`SAV-PROJCORP-430`** (High for the census, Medium for the allocator
  relation, promoted): the lawful research corpus's 29 distinct world-state
  documents all carry canonical `FreeIndex`/`IDs` kinds; 28 are empty; the
  sole nonempty case is `en/game0018.sav`, id 266, `FreeIndex` 267.
- **`ANIM-PROJ-025`, `ANIM-PHASECLOCK-028`, `MAGIC-CASTSPAWN-033`,
  `MAGIC-CASTTICK-030`, `REG-PROJ-086`** (High per-instruction, all active,
  none promoted): together name every one of the sixteen `Prj<id>` leaves'
  own live meaning — a spawned `CProjectile` driven every tick by
  `R0558` (position, facing, a per-picture `switch`, homing re-aim
  against a live target), its sheet advancing on a two-tick phase clock
  against `projectiles.reg`'s per-picture registry, spawned by a cast or a
  unit's shot, applied at a separate simulation-side tick from the one the
  client spawns the visible object on. Consulted for leaf semantics, not
  built against: see "Divergence rows".

## As-built behaviour

**`pkg/formats/sav`** (`projectile.go`, new): `File.Projectiles()` reads the
state-store subtree through `SAV-PROJLOAD-429`'s own tolerant path —
`FreeIndex` and `IDs` each default independently, `IDs` accepts either its
canonical kind-6 array or the kind-2 singleton compatibility arm, and a
between-mission save (or a world save with neither leaf present) reports
`present=false`, not an error. One `ProjectileStore.Items` entry is built per
DISTINCT id `IDs` names, first-occurrence order, mirroring the loader's own
dedupe. `File.SetProjectiles` writes `SAV-PROJSTORE-428`'s own canonical
shape — `FreeIndex` present, `IDs` always kind 6 even where the loader would
also accept the singleton form — through the existing
`parseWorldState`/`serializeWorldState` codec: a kept `Prj<id>` section's own
directory KIND WORD is left exactly as parsed (only a newly created section
gets a fresh one), a removed id's section is deleted outright. A layout test
(`TestProjectilesLayout...`) locks kinds and leaf order against
`worldProjectileLeaves`; a leave-alone test locks the writer's own span.
`SetProjectiles` is the first `File.Set*` writer that rewrites the whole
world state store rather than patching the archive body at an offset —
`serializeWorldState` re-lays every record and the pool in
`worldStateShape`'s own stable name sort, so 51 of 51 world-half corpus
saves differ in byte span at unchanged length even though every leaf outside
`Projectiles` is unchanged and 40 successive exports of the same file are
byte-identical; whether ROM1 tolerates a reordered store is Unknown.

**`pkg/sim`** (`savedprojectile.go`, new): `SavedProjectiles`/`SavedProjectile`
are `pkg/sim`'s own mirror of `sav.ProjectileStore`/`sav.Projectile` (no
`pkg/formats/sav` import). `World.savedProjectiles` is carried-not-wire-form,
`savedCellRecords`'/`savedSpellEffects`' own class: no rule recomputes it
from other state, so `UnmarshalBinary`'s composite literal carries the
receiver's own prior value across a decode rather than reading it from the
encoded bytes, and a decode into a FRESH receiver needs priming
(`SetSavedProjectiles`) first or the value is silently lost.
`ImportOriginalLivingActors` (`pkg/sim/originalliving.go`) and
`restoreOriginalActorStock` (`pkg/game/originalholdings.go`) both stage
every original actor admission through exactly that round trip and both now
prime it, on `savedCellRecords`'/`savedSpellEffects`' own precedent.
`resumeWorld`'s own `.ags` resume (`pkg/game/resume.go`) needs NO priming
call, unlike those two: it calls `ms.World.UnmarshalBinary(form)` directly on
the SAME receiver the composite literal reads `savedProjectiles` off, so the
value already survives with no external call — proven by
`TestUnmarshalBinaryOntoTheSameReceiverNeedsNoProjectilePriming`
(`pkg/sim/savedprojectile_test.go`), not merely inferred. Today
`ms.World.savedProjectiles` is always empty entering that call regardless,
since no original-SAV overlay composes with the `.ags` resume path in this
project.

**`pkg/game`** (`originalprojectiles.go`, new): `applyOriginalProjectiles`
mirrors `applyOriginalCellRecords`' own shape for a section with no rebind —
every leaf stays carried opaquely, on `originalCellRecords`' own Sack/
SpellEffect-key precedent (DIV-933) — and is wired into both original LOAD
doors (`ResumeOriginalSave`, `RestoreOriginal`) right after
`applyOriginalCellRecords`/`applyOriginalSpellEffects`, and into
`restoreOriginalActorStock`'s staging round trip. `OriginalSaveResume`
reports `Projectiles`/`ProjectilesApplied` alongside the existing fields.
`exportOriginalProjectiles` re-exports the carried store into a decoded
file's own world half — REPLACING its `Projectiles` subtree outright, since
the store carries its own complete id set rather than a fixed per-cell table
— but has NO production caller: this project has no production mission SAVE
path yet (owner rule), so it is an unshipped round-trip building block,
checked only by the corpus audit and the release witnesses' own native-export
half.

## Divergence rows

- **DIV-944** (FIDELITY-DEBT, OPEN): the store is carried, never enacted.
  This build has no live projectile registry to bind a restored `Prj<id>`
  section to, and the five leaf-semantics claims describing what a live
  driver would do (`ANIM-PROJ-025` and the other four) are active, not
  promoted — building one would be a new persistent-entity simulation
  feature well outside a store-restore story.
- **DIV-945** (FIDELITY-DEBT, OPEN): real-content coverage is a single
  distinct sample. Both the research corpus (`SAV-PROJCORP-430`, one
  nonempty of 29 documents) and this project's own wider 72-file preserved
  owner-save corpus (2 non-empty of 51 world-half saves per root, both
  `game0018.sav`'s own content) show exactly one distinct nonempty case; the
  loader's own tolerant arms (the kind-2 `IDs` singleton form, a partial leaf
  set, more than one live id) are exercised only by hand-built fixtures.
- DIV-946 through DIV-949 (reserved, unused) are returned unclaimed.
- DIV-933 (story 1131, cell-record table's own Sack/SpellEffect identity
  keys) and DIV-939 (story 1132, the top-level SpellEffect graph) are
  different, narrower surfaces and are unchanged by this story; DIV-944
  follows the same carried-not-enacted shape for a third, distinct subtree
  of the same state store.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean
  (net +2, 198 registered names; `population.txt` is 202 lines, 4 of them
  comments).
- `scripts/check-no-game-assets.sh`: clean.
- `TestProjectileCorpusAudit1133` (`-tags sessioncorpusaudit`, opt-in, both
  lawful roots): 72 files audited (51 world-half, 20 between-mission, 1
  unreadable), 2 non-empty, 0 mismatching, on EN and again on RU (identical
  log line both roots: `audited 72 file(s): 51 world-half, 20
  between-mission, 1 unreadable, 2 non-empty, 0 mismatching`). The 2
  non-empty subjects are both copies of `game0018.sav`'s own content
  (`FreeIndex` 267, one live id 266). The decoded `Projectiles` value matches
  on all 51; the whole state-store span does not reproduce byte for byte on
  any of them, because `serializeWorldState`'s pool order follows the
  shape's stable name sort rather than the original producer's own layout
  for the OTHER sections sharing the store — a pre-existing property of this
  codec, not particular to this story or its subtree.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 198 gated
  tests, 2 roots; **198 of 198 ran and 0 lacked a subject, on EN and again on
  RU.**
- `pipeline/check-milestone.sh`: not run. This story's restoration is
  reached only when resuming an original save, a path the census's own
  fresh, non-resumed mission drive never exercises — story 1132's own
  reasoning for the same population.
- `go build ./cmd/missionrun` (from this worktree) plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master,
  whose own recorded census already carries no `cannot run` line for either
  mission (`pipeline/milestone-baseline.txt`).
- `pipeline/check-div-claims.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree): exit 0, 105 rows flagged (informational; DIV-944 cites
  `ANIM-PHASECLOCK-028`, which carries an unrelated partial retraction — the
  13-entry ramp's own constant enumeration, not the two-tick divisor this row
  leans on).

## Open debt

**Native mission SAVE has no interactive path yet.** `exportOriginalProjectiles`
is exercised only by this story's own corpus audit and release witnesses. No
production menu, hotkey or session flow calls it; the owner's own standing
rule against inventing a production mission SAVE entry point applies here
the same way it did to `exportOriginalCellRecords`/`exportOriginalSpellEffects`
in stories 1131/1132. The building block is tested and correct; wiring it to
a player-visible SAVE action is a separate, later decision.

**No live driver is built** (DIV-944). The five leaf-semantics claims this
story consulted describe a whole tick-driven flight/homing/phase-clock
mechanism; none of it is simulated. A future story that wants a visible,
ticking restored projectile needs a live persistent-projectile entity this
project does not have anywhere yet, and ideally promoted versions of the
five claims.

**The corpus's only non-empty sample is one distinct case** (DIV-945): one
live id, one set of sixteen leaf values, `FreeIndex` one past it. The
loader's own tolerant arms (singleton `IDs`, a partial `Prj<id>` section,
more than one simultaneously live id) are covered by
`pkg/formats/sav/projectile_test.go`'s hand-built fixtures, built from the
claims (golden rule 2), not by any real corpus sample.

**The `.ags` resume drops this store while nothing reads it.**
`World.MarshalBinary` does not encode `savedProjectiles`, so no `.ags` byte
form carries it; `resumeWorld` (`pkg/game/resume.go:731`) decodes a resumed
session's world through `ms.World.UnmarshalBinary(form)` on the same
receiver, whose composite literal carries that receiver's own prior (empty)
value, the same gap DIV-932 and DIV-938 name for `savedCellRecords` and
`savedSpellEffects`. Driven end to end (original LOAD of `game0018.sav`,
ordinary SAVE, fresh LOAD): `FreeIndex` goes from 267 to 0, `IDs` from
`[266]` to `[]`, `Items` from 1 to 0, while `World.Hash()` is unchanged,
since no simulation step reads this field today.

**Both staging primes now have an asset-free witness.**
`TestImportOriginalLivingActorsStagingCarriesSavedProjectiles`
(`pkg/sim/savedprojectile_test.go`) already covered
`checked.SetSavedProjectiles` (`pkg/sim/originalliving.go`);
`TestRestoreOriginalActorStockStagingCarriesSavedProjectiles`
(`pkg/game/originalholdings_test.go`, added by this hotfix) now covers
`staged.SetSavedProjectiles` (`pkg/game/originalholdings.go`), the one of
the two that previously had none — only the install-gated
`TestReleaseOriginalProjectilesRestoreOnLoad1133` caught a revert of it
before.

## Touched surfaces

- `pkg/formats/sav/projectile.go` (new): typed projection and writer.
- `pkg/formats/sav/projectile_test.go` (new): unit tests built from claims,
  a layout test and a leave-alone test.
- `pkg/sim/savedprojectile.go` (new): `SavedProjectiles`/`SavedProjectile`
  and `World` accessors.
- `pkg/sim/savedprojectile_test.go` (new): round-trip/detach contract, the
  staging-carry regression, and the same-receiver no-priming proof.
- `pkg/sim/world.go`: `savedProjectiles` field.
- `pkg/sim/binary.go`: carry it across `UnmarshalBinary`'s composite
  literal.
- `pkg/sim/originalliving.go`: prime it before `ImportOriginalLivingActors`'
  own staging decode.
- `pkg/sim/nostate_test.go`, `pkg/sim/world_test.go`: field/method-set pins.
- `pkg/game/originalprojectiles.go` (new): the sav<->sim converters, LOAD
  and export entry points.
- `pkg/game/originalsave.go`: wire both original LOAD doors and
  `OriginalSaveResume`'s report/`String()`.
- `pkg/game/originalholdings.go`: prime the same carried field before
  `restoreOriginalActorStock`'s own staging decode.
- `pkg/game/originalholdings_test.go` (seat hotfix): asset-free witness for
  that priming, `TestRestoreOriginalActorStockStagingCarriesSavedProjectiles`.
- `pkg/game/resume.go`: document why `resumeWorld` needs no priming call.
- `pkg/game/originalprojectiles1133_corpus_test.go` (new,
  `sessioncorpusaudit`-tagged): `TestProjectileCorpusAudit1133`.
- `pkg/game/originalprojectiles1133_release_test.go` (new):
  `TestReleaseOriginalProjectilesRestoreOnLoad1133`,
  `TestReleaseOriginalProjectilesEmptyStoreSurvivesLoad1133`.
- `internal/gatedtests/testdata/population.txt`: register both new
  witnesses.
- `docs/DIVERGENCES.md`: DIV-944, DIV-945.
- `docs/1133/story.md`: this file.
