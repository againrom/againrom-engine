# Story 1143 — a dead actor that never was a map unit

## Why

Three ROM1-written mission-30 saves the owner made from our own build
(`gameversions/saves/2027-09-07/game0011.sav`, `game0032.sav`, `game0033.sav`)
refused to load. Both refusals sat in guards this project wrote, both
narrower than published law:

- `pkg/game/originaldead.go`'s `applyOriginalDead` required every saved dead
  actor to join exactly one authored map placement, refusing `MapUnitID` 0.
  `SAV-DEADLOAD-125` (promoted, High): ROM1's own complete post-load rebind
  resolves position and five object references "without joining an authored
  map unit" — no such join exists in the load chain at all.
- `pkg/sim/originaldead.go`'s `deadStateFault` admitted only stage 3 through
  5. `SAV-DEADLOAD-128` (High) enumerates the decay ladder's actor-proven
  stage routes as "2/3/4/5," naming stage 2 a route the law carries and this
  guard refused.

## In-scope behaviour

**`deadStateFault`** now admits stage 2 through 5 (`pkg/sim/originaldead.go`).
Every other field of the tuple is unchanged: `Timer` still must be 0
(`SAV-DEADLOAD-131`, High, all eight live writers sit outside the dead-load
and decay routes and all 101 archival records carry zero), and the
stage/HP/RuntimeID pairing, `FineX`/`FineY` and cell bounds are read exactly
as before (`SAV-DEADLOAD-126`, High, three independent stored scalars, no
cross-field derivation). Stage 0 and 1 stay refused: `SAV-DEADLOAD-128` names
stage 1 ("first dying tick") too, but no saved record in this corpus or
`SAV-DEADLOAD-127`'s 101-observation census carries it, so nothing widens a
guard past what an actual saved record has shown.

**A `MapUnitID` 0 dead record** is no longer refused. `deadSourceFault`
(`pkg/sim/originaldead.go`) drops the `MapUnitID == 0` rejection entirely;
`applyOriginalDead` (`pkg/game/originaldead.go`) mints a fresh `EntityID` for
it from the new `World.NextEntityID()` instead of searching `ms.Map.Units`
for a placement, and carries it through `ImportOriginalDeadActors`
(`pkg/sim/originaldead.go`) as a virtual dead actor with **no live entity**:
its tuple is written straight into `originalDeadRecord.terminal` and the
record is appended to `w.originalDead`; `w.entities` is never touched, so
`clearFelled`, kill-credit clearing, loot suppression, and the
carried/equipment wipe that a bound corpse gets all stay unreached for this
shape — there is no entity for them to act on. `OriginalDeadRecord.Current`
already expressed "held tuple, no live projection" through `terminal.Stage !=
0` before this story (`OriginalDeadActors()`'s own branch); a `MapUnitID` 0
record uses that same shape, not a new one.

**Uniqueness.** `ImportOriginalDeadActors`'s `seenMap`/decode-side `maps` set
is now consulted and set only when `Source.MapUnitID != 0`: two actors both
carrying `MapUnitID` 0 are not the same ambiguity as two actors claiming the
same authored placement, and a corpus with more than one such actor (none in
this corpus, but nothing rules it out) must not collide on that account. A
`MapUnitID` 0 actor's own `EntityID` is still checked against every live
entity, so it cannot alias one.

**Occupancy, drawing, hashing (SAV-DEADLOAD-132's own Unknown).** Never
entering `w.entities` means a virtual dead actor is automatically excluded
from the occupancy grid, the draw list, the live tick list, and every
downstream `originalgroups.go`/`savedgroups.go`/`savedgroupsai.go` consumer
of `w.originalDead` — each of those already gates on
`indexOfEntity(w.entities, ...) >= 0`, so no new filter was needed anywhere
in that family. It is still part of the persisted byte form and `Hash()`
(the dead-actor section `w.encode()` already carried), since it is saved
state. Recorded as `DIV-997`, category UNKNOWN: SAV-DEADLOAD-132 itself calls
runtime occupancy Unknown, so never inserting the record asserts nothing
beyond what that claim already leaves open.

**Not touched:** the Effects/Carried/Worn accept set in `applyOriginalDead`
is unchanged — it already refused nonzero Effects/Carried, and nonzero Worn
unless it is the one proven terminal-weapon shape, before this story, and
none of the three failing files needs it widened (`SAV-DEADLOAD-130`, High:
all 101 observed bodies carry container presence 1, count 0, tails
`(10000,0)`; the two `MapUnitID` 0 records here match that shape exactly).

No format-version move: `MapUnitID` was already an encoded field at its
existing wire offset in `originalDeadRecordLen` (173 bytes); only its
interpretation (0 now means "no join," not "invalid") widened.

## Touched surfaces

`pkg/sim/originaldead.go`, `pkg/sim/originaldeadbinary.go`,
`pkg/sim/originalliving.go`, `pkg/sim/spell.go` (the last two only for
`nextEntityID` → exported `NextEntityID`, needed so `pkg/game` can mint an
ID), `pkg/sim/world_test.go` (pinned exported-method-set list),
`pkg/game/originaldead.go`, `pkg/game/originalsave.go` (`UnboundRestored`
counter and report line), `pkg/game/originaldead1143_release_test.go` (new
regression test), `internal/gatedtests/testdata/population.txt`.

Also corrected, on the coordinator's instruction, two prose passages in story
1140's own files that this story's fix makes false — the exclusion mechanism
they describe is unchanged, only the present-tense claim that the corpus
currently carries this refusal family: `scripts/check-milestone2-acceptance.sh`,
`pkg/game/milestone2_acceptance_reader_test.go`, `docs/1140/story.md` (a
closure paragraph appended, the frozen prose above it untouched).

## Proof

`TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage`
(`pkg/game/originaldead1143_release_test.go`) resumes all three files,
checks each dead record's source/current tuple against an independent
transcription, checks entity binding matches the expected shape (bound for
game0011.sav's stage-2 corpse, unbound for game0032.sav/game0033.sav's
`MapUnitID` 0 mercenary), resumes each file a second time and requires equal
`World.Hash()`, and round-trips the resumed world through
`MarshalBinary`/`UnmarshalBinary` requiring an identical dead-actor slice and
hash. Proven to fail both ways: reverting either guard individually
reproduces the exact original refusal text (see report), and the widened
code passes.

Five-file `savecheck ... load` result, before and after, on
`AGAINROM_ASSETS=gameversions/en`:

| file | before | after |
|---|---|---|
| game0005.sav | resumed, hash `b723baccc4a075c1` | unchanged |
| game0011.sav | refused: `unsupported late-dead state: stage 2 HP -14 timer 0 runtime 90 cell 0x3332 fine 128,128` | resumed, hash `8465427295e90c5f`, 39 entities |
| game0032.sav | refused: `invalid original dead identity/class` | resumed, hash `5bc98124a1661e41`, 38 entities |
| game0033.sav | refused: `invalid original dead identity/class` | resumed, hash `5ba2e820a603f314`, 38 entities |
| game0034.sav | town, chapter 40, gold 4917818 | unchanged |

`scripts/check-milestone2-acceptance.sh` on both lawful roots: all four
`TestMilestone2*` instruments report 0 refused (was 3, all in this family)
and every mismatch counter 0 on EN and RU — exact counts in the story's
return report.

## Open debt / Unknowns

- `DIV-997`: virtual dead actor occupancy/drawing/hashing (SAV-DEADLOAD-132's
  own Unknown), conservative choice disclosed, OPEN.
- Whether a `MapUnitID` 0 record could ever carry nonzero Effects/Carried/Worn
  is unevidenced by any file this story read; `applyOriginalDead` still
  refuses it rather than guess a shape.
- Whether stage 0 or 1 could ever appear in a saved dead record is
  unevidenced by this corpus or `SAV-DEADLOAD-127`'s own census; both stay
  refused.
- The `DIV-991` through `DIV-996` slots the allocation ledger reserved for
  this story's twin, story 1142, are that story's own; this story's own
  reservation was `DIV-997` through `DIV-1002`. Only `DIV-997` is used;
  `DIV-998` through `DIV-1002` retire unused — no second decision arose past
  the one occupancy/drawing/hashing choice.
- The tavern hire-offer discrepancy the owner reported for `game0034.sav` is
  measured, not fixed, in this story's return report: it is a separate
  vertical this story was explicitly told not to touch.
