# Story 1137 — single carrier for the ground Sack's own container tail

## Status

Gates run on the exact candidate commit, current main (`354e37d9`, the
engine repository's single re-rooted commit); see "Proof" for the full list
and exact counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...`
(whole repository, asset-free) clean, 49 packages, 0 FAIL, including
`internal/gatedtests`' checked-in `testdata/population.txt` census unchanged
(205 registered names before and after — this story adds and removes no
gated test); `scripts/check-no-game-assets.sh` clean;
`pipeline/check-release-tests.sh` against both lawful roots, 8 packages, 205
gated tests, 205 of 205 ran and 0 lacked a subject on EN and again on RU;
mission 10/20 `-trace -ticks 1` UNSUPPORTED-node count 0 on both missions,
both roots, unchanged from master's own recorded census (no `cannot run` line
for either mission, `pipeline/milestone-baseline.txt`) — this story's own
change is reached only through an original-save resume, a path the fresh
mission drive does not exercise (story 1136's own precedent, "Proof"). No
claim was read, cited beyond story 1136's own three, or created.

## Player result

A mission loaded from an original ROM1 SAV keeps exactly one live record of
each ground Sack's own container tail (`+0x1c` insertion sentinel, `+0x20`
running weight, `SAV-670`/`SAV-671`/`SAV-672`). Story 1136 built the correct
carrier, `sim.SavedObjectContainer` (story 1115) — live, hashed, filled by
`importSavedSackObjects` on both original LOAD doors, enacted by
`putGroundObject`'s sentinel mint and `weight()` on drop/pickup and by a
dying actor's pack spill — but also added a second, unhashed carrier,
`sim.World.savedGroundContainers`, restored from the same two file fields on
the same LOAD doors but read only by the unshipped
`exportOriginalGroundContainers`, which re-exported that second carrier's
file-time value rather than the first carrier's live one (DIV-962). This
story removes the second carrier outright: the type, the field, its staging
priming in `ImportOriginalLivingActors` and `restoreOriginalActorStock`, and
the LOAD-time import that filled it. `exportOriginalGroundContainers` now
re-derives both dwords from the single live `SavedObjectContainer` ledger on
every call, joined to the file's own archive Identity through a new shared
helper, `liveGroundContainerTail` — never a file-time snapshot again, and
never silently stale after a mid-mission drop or pickup.

The removed carrier was never referenced inside `World.encode()` (traced
across its full call chain, ending in `appendSavedObjects`), so removing it
cannot move the world hash; the kept carrier already was inside `encode()`
before this story and is unchanged by it. `TestHashIsPinned`,
`TestThePinnedDigestIsFNV1aOfThePinnedBytes` and
`TestHashIsFNV1aOverExactlyTheByteForm` (`pkg/sim/hash_test.go`) all still
pass unchanged against the same literal pinned digest, `0xf8eeb7a2b2bd91df`
("Form84") — the byte form this story touches zero bytes of.

`applyOriginalGroundContainers` does not become a repositioned cross-check:
`importSavedSackObjects` — the call that actually populates the live
registry — runs near the end of both original LOAD doors, well after the old
call site, so a production check there would need moving, not patching. The
existing corpus audit (`TestGroundContainerCorpusAudit1136`,
`sessioncorpusaudit`) already cross-checks every adopted Sack in the full
preserved corpus against the single carrier at full LOAD-time population
(see "Proof"); adding a second, narrower production check that runs on every
load for no additional coverage was not worth the risk of a second place to
get the join wrong. `applyOriginalGroundContainers` and the file-reading
`originalGroundContainers` it called are deleted, not repositioned.

## Authority

No new claim. The single carrier's own shape is governed by the same three
claims story 1136 already cited and did not change:

- **`SAV-670`** (High, promoted, EXP-0313): the container tail is
  `ITEM-CONT-004`'s own class, distinct from `Item`'s numerically coincident
  fields.
- **`SAV-671`** (High, promoted, EXP-0313): `+0x1c` defaults to sentinel
  `10000`; the corpus census behind this claim finds it on every one of 307
  sampled Sack-class containers, zero exceptions.
- **`SAV-672`** (High, promoted, EXP-0313): `+0x20` is the running
  weight-times-count sum `ITEM-STACK-003` gives five maintenance sites for.

## As-built behaviour

**`pkg/sim`**: `savedgroundcontainer.go` and `savedgroundcontainer_test.go`
deleted — `SavedGroundContainer`, `World.SavedGroundContainers`/
`SetSavedGroundContainers`/`ImportOriginalGroundContainers`, and the three
tests that existed solely to prove the removed type's round-trip/detach and
staging-carry contract. `World.savedGroundContainers` removed from the struct
(`world.go`) and from `UnmarshalBinary`'s carried-not-wire-form composite
literal (`binary.go`, `gofmt -w` re-applied after the edit).
`ImportOriginalLivingActors` (`originalliving.go`) no longer primes it before
its staging round trip. `nostate_test.go`'s field-completeness fixture and
`world_test.go`'s method-set pins both drop their rows for the removed
type/methods.

**`pkg/game`**: `originalgroundcontainers.go` rewritten down to two
functions. `liveGroundContainerTail(registry *sim.SavedObjects, identity
uint32) (sav.GroundContainerTail, bool)` is the new shared join, reused by
LOAD-side proof and by export. `exportOriginalGroundContainers(f *sav.File,
w *sim.World) error` walks the file's own `GroundSacks()`, looks up each by
identity through the helper, and refuses with a named error
(`"original ground containers export: Sack %#x has no live container"`) for
any file Sack `importSavedSackObjects` declined to adopt — a Sack with no
live container is no longer papered over with a stale file-time copy. The old
file-reading `originalGroundContainers` and the LOAD-time
`applyOriginalGroundContainers` are gone. `originalsave.go` loses
`OriginalSaveResume.GroundContainers`/`GroundContainersApplied`, their
`String()` line, and both LOAD doors' calls into the removed importer.
`restoreOriginalActorStock` (`originalholdings.go`) no longer primes the
removed field before its own staging round trip; `resume.go`'s doc comment
naming the same-receiver no-priming case is corrected to say so.

**Tests**: `originalground_test.go`'s assertion for `testGroundPopulation()`
(two Sacks sharing cell `0x0706`) now proves the single carrier invents
nothing for a Sack `importSavedSackObjects` declines
(`savedSackCellAmbiguous`, both Sacks here) — `liveGroundContainerTail`
returns `ok=false` for both identities. The adopted case is proven at scale
by `TestGroundContainerCorpusAudit1136` (51 of 51 real world-half saves, 0
declines in the preserved corpus) and by
`TestReleaseOriginalGroundContainersRestoreOnLoad1136` (`game0021.sav`, 4
real adopted Sacks); building a second, synthetic adopted fixture here would
require faking the separate saved-cell-planes subsystem `registerSavedSackCell`
needs, which this story does not otherwise touch.
`originalgroundcontainers1136_release_test.go` and
`originalgroundcontainers1136_corpus_test.go` both now compare against
`sav.GroundContainerTail` through `liveGroundContainerTail` instead of the
removed type; the corpus test's own primary comparison — previously file
against the second carrier, before ever reaching anything outside that same
decode — is gone, and its "independent carrier" comparison against
`sim.SavedObjects().Containers` (story 1115's own decode, never
`GroundSacks`) is now the sole comparison. `originalholdings_test.go` drops
the staging-carry regression for the removed type.

## Divergence rows

- **DIV-962** (updated; remains OPEN): `sim.SavedObjectContainer` (story
  1115) is now recorded as the SINGLE carrier. The second carrier this row
  used to name — `sim.World.savedGroundContainers`, its priming, and
  `applyOriginalGroundContainers` — is recorded as removed outright, never
  inside `World.encode()`, so its removal changes no world hash.
  `exportOriginalGroundContainers` is recorded as re-deriving from the single
  live ledger on every call, refusing rather than falling back to a stale
  copy for a Sack `importSavedSackObjects` declined — never observed in the
  full preserved corpus. Remaining debt, unchanged by this story: no claim
  names a Sack-side consumer of either field, and neither LOAD path validates
  a hostile tail against the item list. The row's own revisit condition — a
  claim naming a Sack-side consumer, or an owner decision on modeling
  dropped-loot order/weight display — is not met by this story, so the row
  stays OPEN; the "production caller needs a separately scoped story first"
  clause is removed, since a caller no longer would need one.
- **DIV-594** (updated): the sentence recording the container tail's own
  carrier now names `sim.SavedObjectContainer` as the single carrier and
  records that this story removed the second, unhashed one DIV-962 used to
  name.

## Proof

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `go vet ./...` and `go vet -tags sessioncorpusaudit ./...`: clean (a
  handful of pre-existing `BookSpell` unkeyed-field warnings in files this
  story does not touch are unrelated and unchanged).
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean,
  205 registered names before and after this story (`population.txt` is 209
  lines, 4 of them comments) — net zero change, no gated test added or
  removed.
- `scripts/check-no-game-assets.sh`: clean.
- `TestGroundContainerCorpusAudit1136` (`-tags sessioncorpusaudit`, opt-in,
  fresh run on both lawful roots): `audited 72 file(s): 51 world-half, 20
  between-mission, 1 unreadable, 51 non-empty, 0 mismatching`, identical on
  EN and RU, byte-for-byte native re-export unchanged on every save. Every
  one of the 51 non-empty world-half saves logs "0 not adopted by
  importSavedSackObjects" — 100% adoption in the full preserved corpus, so
  the new export refusal path is exercised by no preserved save.
- `TestReleaseOriginalGroundContainersRestoreOnLoad1136` (fresh run, both
  roots): `sacks=4 accumulators=[20 4 11 10]: LOAD and native export both
  reproduce the source file's decoded container tails` on EN and again on
  RU — the same real values story 1136 originally recorded, now read through
  the single carrier.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 205 gated
  tests, 2 roots; **205 of 205 ran and 0 lacked a subject, on EN and again on
  RU.**
- Hashed-state instrument: `pkg/sim/hash.go`'s `Hash()` is
  `FNV-1a(w.encode())`; `encode()`'s own call chain
  (`pkg/sim/binary.go`) ends in `w.appendSavedObjects(...)`, confirmed by
  direct trace to reference `savedObjects` (story 1115's carrier, kept) and
  never `savedGroundContainers` (this story's removed carrier) anywhere in
  its body. `TestHashIsPinned`, `TestThePinnedDigestIsFNV1aOfThePinnedBytes`
  and `TestHashIsFNV1aOverExactlyTheByteForm` (`pkg/sim/hash_test.go`) all
  pass unchanged against the same literal pinned digest,
  `0xf8eeb7a2b2bd91df` ("Form84"). No existing save or scenario's world hash
  moves.
- `pipeline/check-milestone.sh`: not run, on story 1136's own precedent for
  the same population — this story's change is reached only through an
  original-save resume. `go build ./cmd/missionrun` plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master's
  own recorded census, which carries no `cannot run` line for either mission
  (`pipeline/milestone-baseline.txt`).

## Open debt

**No live consumer reads the carried tail** (DIV-962, unchanged by this
story). Neither field feeds any Sack-side rule: `pourSack` always appends at
the live tail regardless of the carried insertion index, and no Sack-side
weight display or capacity check exists to read the accumulator back. A
future story that wants either value to matter needs a claim naming a
Sack-side consumer, which does not exist today.

**Neither LOAD path validates the tail against the item list** (DIV-962,
unchanged). A hostile weight sum or an insertion index outside the list is
carried unchanged on LOAD, because `SAV-671`'s own census finds the `+0x1c`
sentinel on all 307 sampled records with zero exceptions and a refusal here
would reject loads the original itself accepts.

**The new export refusal for a declined Sack is untested against a real
save.** `exportOriginalGroundContainers` now refuses outright if a file Sack
has no live container, but every one of the 51 non-empty world-half saves in
the full preserved corpus is 100% adopted — the refusal path is exercised
only by `originalground_test.go`'s synthetic same-cell fixture, never by a
real save. This is a narrower failure mode than before (a stale copy could
never surface this way), not a new risk, but it has not been observed
against owner data.

**Native mission SAVE still has no interactive path.**
`exportOriginalGroundContainers` remains exercised only by this story's own
corpus audit and release witness, unchanged from story 1136; the owner's own
standing rule against inventing a production mission SAVE entry point still
applies.

## Touched surfaces

- `pkg/sim/savedgroundcontainer.go`, `pkg/sim/savedgroundcontainer_test.go`:
  deleted.
- `pkg/sim/world.go`: removed the `savedGroundContainers` field.
- `pkg/sim/binary.go`: removed it from `UnmarshalBinary`'s composite
  literal; `gofmt -w` realignment.
- `pkg/sim/originalliving.go`: removed its staging prime.
- `pkg/sim/nostate_test.go`, `pkg/sim/world_test.go`: field/method-set pins.
- `pkg/game/originalgroundcontainers.go`: rewritten to the single-carrier
  helper and export.
- `pkg/game/originalgroundcontainers1136_corpus_test.go`,
  `pkg/game/originalgroundcontainers1136_release_test.go`: repointed at the
  single carrier through `liveGroundContainerTail`.
- `pkg/game/originalground_test.go`: replaced assertion, proving the
  declined-Sack case.
- `pkg/game/originalholdings.go`, `pkg/game/originalholdings_test.go`:
  removed staging prime and its regression.
- `pkg/game/originalsave.go`: removed the report fields, `String()` line,
  and both LOAD doors' calls into the removed importer.
- `pkg/game/resume.go`: corrected doc comment.
- `docs/DIVERGENCES.md`: DIV-962, DIV-594 (both updated, both remain OPEN).
