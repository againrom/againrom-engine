# The import-side late-corpse constructor (DIV-1344's own named continuation)

A party member or hired mercenary dies during a mission. ROM1 writes his
death into the dead list with `MapUnitID` 0 -- he was never an authored map
placement, so there is no ALM unit to join him to. The owner's own preserved
saves already carry this shape twice: `game0032.sav` and `game0033.sav`
(2027-09-07) both hold dead record `0x123fbb48`, Class Human, stage 4, no
authored map unit -- a hired mercenary, confirmed by
`TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage`
(`pkg/game/originaldead1143_release_test.go`) before this story.

Before this story that record imported as data and stayed invisible forever.
`ImportOriginalDeadActors`'s `MapUnitID`0 arm (`pkg/sim/originaldead.go`)
appends the record to `w.originalDead` and `continue`s before any entity
exists. `mapWorld.entityDraws` (`pkg/game/world.go`) built its whole returned
slice from `mw.world.Entities()` alone and consulted `OriginalDeadActors()`
nowhere. A map-bound corpse (`MapUnitID` != 0) is an entity, so it drew; this
one never became an entity, so it never drew, no matter what its own
`Current` tuple said. Story1200 (DIV-1344) found this gap while looking at
the writer side, named it "the late-corpse constructor", and left it for its
own story. This is that story.

## In-scope behaviour

- **The record draws, but only once its body resolves.** New
  `mw.virtualDeadDraws` (`pkg/game/world.go`) reads `OriginalDeadActors()`
  directly and, for every `MapUnitID`0 record at `Current.Stage` 2 through 4
  -- the pre-terminal window read out of `SAV-DEADLOAD-128` (see Authority) --
  resolves its body through `mw.art` first and appends one `ui.MapEntity`
  only when that resolution succeeds, at the record's own
  `Current.Cell`/`FineX`/`FineY`, `Life: ui.LifeDead`, `HP` from the frozen
  tuple, `Untargetable: true`. A record whose body does not resolve --
  including every record after a native `.ags` reload, where `ms.DeadArt`
  carries no entry (see next bullet and Open debt) -- draws nothing, the
  same "nothing" main draws for it today. This is the corrected shape: an
  earlier version of this function built the `ui.MapEntity` unconditionally
  and filled `Art`/`Frame` afterward only when resolution succeeded, which
  put an opaque diagnostic placeholder on the map for the unresolved case --
  a player-visible defect the sole adversarial pass found and this story's
  one correction pass removed (Finding 1,
  `pipeline/reviews/story1201-adversarial.md`; see Correction pass below).
- **The body resolves to a real sprite when it resolves at all.**
  `OriginalDeadSource` carries only the coarse 1/2/3 Unit/Human/Humanoid
  category -- not enough to pick a class. The underlying SAV archive record
  does carry a class-resolving field: its `T0E` word, which
  `pkg/formats/sav/actorgraph.go`'s `ActorRecord` already reads generically
  for both `Player` and `doc.dead` records, but `pkg/formats/sav/dead.go`
  had never surfaced it. `sav.DeadActor` now carries it as `TypeID`.
  `applyOriginalDead` (`pkg/game/originaldead.go`) records it, against the
  same freshly minted entity id `ImportOriginalDeadActors` admits, in a new
  `Mission.DeadArt` map -- presentation provenance only, never entered into
  `sim.World`. `missionAppearanceArt` (`pkg/game/world.go`) merges it through
  the exact `units.Classes[int32(uint8(typeID))]` lookup every other
  entity's class already resolves through, so a virtual corpse draws by the
  same rule as everything else on screen, nothing invented for it.
- **Neither ticked nor cell-occupying.** `virtualDeadDraws` reads
  `OriginalDeadActors()`'s frozen `Current` tuple and nothing else -- never
  `Entities()`, `Relations()` or the tick list. `SAV-DEADLOAD-132` finds no
  stage-specific cell insertion in ROM1's own post-load chain and states
  actual loaded cell occupancy is Unknown; nothing here claims one. This is
  the narrower of the two readings `DIV-1344`'s own Recommendation admits (a
  "live, drawn, still-decaying entity"); the decaying/ticking half is not
  implemented and is named as debt below, not silently dropped.

## Out of scope, and why

- **Advancing the record's own decay/stage after load** is not implemented.
  No claim -- not `SAV-DEADLOAD-125`, `126`, `128` or `132`, and DIV-1344's
  own search found none else -- establishes whether ROM1 itself continues
  advancing a `MapUnitID`0 record's stage once no live entity backs it, or
  what identity a reconstruction would carry. Guessing a tick/decay rule with
  no evidence behind it was rejected; the record stays exactly at the
  `Current` tuple `ImportOriginalDeadActors` freezes it at on import.
- **Cell occupancy** (pathing, targeting, collision against this cell) is not
  granted, for the same reason `SAV-DEADLOAD-132` names: Unknown, not
  claimed either way.
- **Persisting the resolved appearance through a native reload.** `ms.DeadArt`
  is rebuilt fresh every time `applyOriginalDead` runs (an original-SAV
  resume) and is never entered into `sim.World`, so a body resolved on one
  load does not survive this project's own native `.ags` save/reload of the
  same mission: after such a reload `virtualDeadDraws` finds no `mw.art`
  entry for the record and draws nothing for it -- see Open debt.

## Authority

`SAV-DEADLOAD-125` (promoted, High) establishes one virtual post-load rebind
for every loaded dead actor, with no authored-map-unit join, for a
`MapUnitID`0 record the same as a bound one. `SAV-DEADLOAD-126` (promoted,
High) establishes stage, signed HP and timer as three independent stored
scalars, restored directly rather than derived. `SAV-DEADLOAD-128` (active,
High) states ROM1's own full-state broadcast "includes dead actors only
below stage 5"; this lane reads that boundary as meaning a stage 2-4 record
is a present actor after load, not a bookkeeping row -- an inference, not the
claim's own text -- and the stage-2-4 draw window is read from it.
`SAV-DEADLOAD-132` (active, High for the
post-load chain's absence / Unknown for runtime occupancy) finds no
stage-specific cell insertion in that chain and leaves actual occupancy
Unknown -- the claim the "neither ticked nor cell-occupying" boundary is read
from. `TypeID`'s own resolution is not a research question: `actorgraph.go`
already reads the identical `T0E` field generically for both `Player` and
`doc.dead` records, so surfacing it in `dead.go` is a decoder-completeness
fact derivable entirely from this project's own source, not new ROM1
evidence. `DIV-1353` (`docs/DIVERGENCES.md`) records this story's own
divergence: the bounded "drawn" half of DIV-1344's wider Recommendation,
FIDELITY-DEBT, OPEN.

## Touched surfaces

- `pkg/formats/sav/dead.go` -- `DeadActor` gains `TypeID uint16`, read off
  `T0E`, populated in `DeadActors()`.
- `pkg/game/mission.go` -- `Mission` gains `DeadArt map[sim.EntityID]uint16`,
  presentation provenance only.
- `pkg/game/originaldead.go` -- `applyOriginalDead`'s `MapUnitID`0 arm records
  `ms.DeadArt[id] = d.TypeID` against the freshly minted id.
- `pkg/game/world.go` -- `missionAppearanceArt` merges `ms.DeadArt` into the
  returned class map; `mw.virtualDeadDraws` appends one draw per `MapUnitID`0
  record at stage 2-4, called from `entityDraws` after its existing
  `Entities()` loop, and only once the body resolves through `mw.art` (the
  correction pass's Finding 1 fix); `entityDraws`' own compaction comment
  corrected to describe its returned order (correction pass).
- `pkg/game/latecorpse1201_release_test.go` -- `TestReleaseLateCorpseConstructor1201`;
  `TestReleaseLateCorpseConstructor1201NativeReloadDrawsNothing` (new,
  correction pass).
- `pkg/game/latecorpse1201_test.go` (new, correction pass) --
  `TestLateCorpseConstructor1201ResolvedAndUnresolvedDraws`, a synthetic
  world discriminating the resolved/unresolved arms and the stage window.
- `internal/gatedtests/testdata/population.txt` -- the new release test name
  registered in alphabetical position.
- `docs/DIVERGENCES.md` -- new row `DIV-1353`; corrected in the correction
  pass (see below).
- Picked up by merge, not authored here: `dcc10f5` (comment-only correction
  to `TestReleaseCurrentWorldDeadRandomRefusal1170`, an unrelated pre-existing
  test whose comments misdescribed a *map-bound* corpse; that fix is the
  seat's, reconciled onto this branch, not touched further).

## Proof

This section reflects the correction pass's own final gate run, on the
reconciled merge commit `8a916dc` (origin/main `21e67eb` merged in). The
review that returned the story is `pipeline/reviews/story1201-adversarial.md`;
its Finding 1 disposition, and N1-N5, are under "Correction pass" below.

`TestReleaseLateCorpseConstructor1201` loads `game0032.sav`
(2027-09-07, hash-pinned), finds its one `MapUnitID`0 dead record (id 43 in
this fixture, stage 4, HP -58, `Current.Cell` decoding to `(52,63)`),
confirms it holds no live entity in `Entities()` (SAV-DEADLOAD-125/132 both
hold), then confirms `entityDraws()` produces exactly one draw for that id:
matching `Cell`, `Life == ui.LifeDead`, matching `HP`, `Untargetable == true`,
and non-nil `Art`/`Frame` -- the fixture's own `TypeID` resolves to a real
sheet.
`TestReleaseLateCorpseConstructor1201NativeReloadDrawsNothing` (new, this
correction pass) forces the same record through story1200's own guarded
native-`.ags` SAVE fallback and reloads it: `ms.DeadArt` is nil on that
reload, and `entityDraws()` now produces zero draws for the record, matching
what `origin/main` draws for it -- nothing -- instead of the opaque magenta
`terrain.EntityMarkerColor` placeholder the pre-correction code drew. Reverted
by hand against a disposable `git archive` copy of the pre-correction tree,
both tests fail: the first with `entityDraws() produced 0 draws for the
MapUnitID-0 record` (the original defect), the second with `native reload
produced 1 draws for record 43, want 0` (Finding 1's own defect), confirming
both tests exercise their fix and not a tautology.
`TestLateCorpseConstructor1201ResolvedAndUnresolvedDraws` (new, asset-free,
`pkg/game/latecorpse1201_test.go`) is N2's own fix: a synthetic
`sim.NewStockedWorld` plus `ImportOriginalDeadActors` batch with four
records at stages 2, 4, 5 and 3 (the last with no resolving `mw.art` entry),
pinning the stage-[2,4] resolved arm, the stage-5 terminal exclusion
(SAV-DEADLOAD-128's own window), and the unresolved arm drawing nothing, all
in one process -- the corpus-only `TestReleaseLateCorpseConstructor1201` could
never by itself discriminate a record inside stage [2,4] from one outside it,
since `game0032.sav` only ever supplied one MapUnitID-0 record at a time.

```
go test ./pkg/game/... -run 'TestLateCorpseConstructor1201|TestReleaseLateCorpseConstructor1201' -tags sessioncorpusaudit -v
--- PASS: TestLateCorpseConstructor1201ResolvedAndUnresolvedDraws
--- PASS: TestReleaseLateCorpseConstructor1201
--- PASS: TestReleaseLateCorpseConstructor1201NativeReloadDrawsNothing
```

`gofmt -l` on every changed/new Go file (`pkg/game/world.go`,
`pkg/game/latecorpse1201_test.go`, `pkg/game/latecorpse1201_release_test.go`)
lists nothing.
`go build ./...` clean.
`go test -trimpath -count=1 ./...` (no install required, golden rule 2), run
once with `GOCACHE` pointed at the seat's shared cache: `EXITCODE=0`, zero
`^FAIL` lines.
`scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree scan)`,
exit 0.
`pipeline/check-div-claims.sh` (the seat's named gate for a divergence-ledger
change, run with `AGAINROM_IMPL` pointed at this checkout): exit 0, `DIV-1353`
and `DIV-1358` both absent from the rows citing a retracted claim; knowledge
pin reported at k58 (`79d0d5e`), matching `origin/main`'s own pin --
`scripts/check-claim-citations.sh` also clean against the same pin.
`git log --format='%h %(trailers:key=Co-Authored-By) %(trailers:key=Claude-Session)' d1e3bd2..HEAD`
lists six commits on this branch's own range (three reconciled in from
`origin/main`'s sack/glow-disc hotfixes, the correction-pass commit, the
reconciliation merge, and this docs commit) -- no trailer on any.

**mission10/mission20 census, measured before and after.** Built
`cmd/missionrun` from this branch's final candidate commit (`8a916dc`) and
compared against the count `origin/main` itself produces at the same script
paths (unaffected by this story's diff, which never touches
`cmd/missionrun`, `pkg/sim/script.go` or `pkg/sim/step.go`):

```
AGAINROM_ASSETS=<againrom>/gameversions/en missionrun -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
AGAINROM_ASSETS=<againrom>/gameversions/en missionrun -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
```

mission10 = 0, mission20 = 0, unchanged -- this story's diff is confined to
`pkg/game` presentation (`entityDraws`/`virtualDeadDraws`/`missionAppearanceArt`)
and documentation; nothing it touches sits on the script-node path.
`pipeline/milestone-baseline.txt` does not carry this exact UNSUPPORTED-node
count under either mission row.

**check-release-tests.sh and check-milestone2-acceptance.sh, EN and RU.**
`check-milestone2-acceptance.sh <en> <ru>` passes clean on both roots (exit
0 each): `SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=102 round-tripped=94
disclosed=0 refused=8 mismatched=0` on both, identical to story1200's own
reported baseline -- this correction pass changed no round-trip outcome.
`check-release-tests.sh <en> <ru>` (single invocation, both roots, using the
gate's own `-timeout` now that it no longer relies on Go's 600s default)
reports `check-release-tests: ok (317 of 317 ran, 0 lacked a subject)` on
both `gameversions/en` and `gameversions/ru`, including the new
`TestReleaseLateCorpseConstructor1201NativeReloadDrawsNothing`, registered in
`internal/gatedtests/testdata/population.txt`. The earlier 600s Go-default
false failure this story had previously worked around by hand is moot: the
gate now sets its own timeout.

## Which EN/RU release tests the seat owes

This is save/screen behaviour (golden-rule table row "screen, shipped data,
or save behaviour"), so the seat owes `check-release-tests.sh` on both roots
on the merge commit. This lane already ran it locally, both roots, one
invocation, clean: `check-release-tests: ok (317 of 317 ran, 0 lacked a
subject)` on each root -- see Proof above. It also owes
`check-milestone2-acceptance.sh <en> <ru>` (original-save-resume gate table
row, since Finding 1's fix sits on the original-save-resume path) -- this
lane ran that too, clean on both roots. Locally, the full sessioncorpusaudit
`TestRelease*` set in `pkg/game`, including both new tests plus every
existing test this change could plausibly touch --
`TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage`,
`TestReleaseNewCorpse1200MapUnitZero`,
`TestReleaseCurrentWorldDeadRandomRefusal1170`,
`TestReleaseMilestone2MapReopen1140` among them -- passed against both the EN
and RU installs and save corpora, folded into the `check-release-tests.sh`
run above.

## Unknowns carried forward, not answered here

- Whether ROM1 itself continues to advance a `MapUnitID`0 corpse's decay
  stage once no live map-joined entity backs it, and what identity a
  reconstruction would carry -- DIV-1344's own gap, unchanged by this story.
- What ROM1's own runtime cell occupancy for such a record actually is
  (`SAV-DEADLOAD-132`'s own Unknown).

## Correction pass

The sole adversarial pass returned this story on one finding, and this is its
one correction pass.

**The finding.** `virtualDeadDraws` built its `ui.MapEntity` unconditionally
for every `MapUnitID`0 record inside the stage 2-4 window, then resolved the
body afterward from `mw.art[r.ID]`. On a native `.ags` restore `ms.DeadArt`
is nil (it is written only by `applyOriginalDead`, on an original-SAV
resume), so `Art`/`Frame` stayed nil, `terrain.UnitPlace` returned not-ok, and
`pkg/ui`'s `entityLayer` routed the draw to its `squares` half, which paints
an opaque `terrain.EntityMarkerColor` square with no debug toggle -- where
main draws nothing. The review reproduced the whole chain through the
production menu SAVE seam: resume `game0032.sav`, damage its own
source-bound `MapUnitID`0 actor, decay it past stage 2 (which forces
`projectWorld1170Dead`'s own native `.ags` fallback, story1200's guard), SAVE,
reload -- record 43 drew once with nil `Art`/`Frame`.

**The fix.** `virtualDeadDraws` now builds the `ui.MapEntity` after the
`mw.art`/`Corpse`/`SelectBoneFrame` lookup and `continue`s when that lookup
yields no frame. First-load behaviour is unchanged; every unresolved path --
native reload, an unresolvable truncated `TypeID`, a class with no `Corpse`
block -- returns to main's "nothing drawn".
`TestLateCorpseConstructor1201ResolvedAndUnresolvedDraws`
(`pkg/game/latecorpse1201_test.go`, new) discriminates the function directly
with a synthetic world: a resolved stage 2 and stage 4 record still draw
with a nonzero `DrawCategory`, a resolved stage 5 record does not (the
window, not the resolution, excludes it), and a stage 3 record with no
`mw.art` entry does not. Confirmed by mutation: reverting the fix makes this
test fail with the pre-fix nil-`Art` draw. `stage` 0/1 cannot be constructed
through `ImportOriginalDeadActors` at all -- `deadStateFault`
(`pkg/sim/originaldead.go`) refuses both for every caller -- so the window's
lower bound stays unpinned by any test; the shipped filter itself is
unchanged.
`TestReleaseLateCorpseConstructor1201NativeReloadDrawsNothing`
(`pkg/game/latecorpse1201_release_test.go`, new) reproduces the review's own
production route end to end -- the same `game0032.sav` fixture, the same
damage/decay/SAVE/reload chain -- and pins zero draws for record 43 after
the reload where the unfixed build drew one with nil `Art`/`Frame`.

**Notes folded in.** The stale comment above `entityDraws`' compaction
("The order is unchanged: this is a stable filter over a slice already in
the world's ascending id") described the loop, not the function's own
result once `virtualDeadDraws`' entries are appended after it; corrected to
say the appended slice is no longer globally ascending while the
compaction's own per-index invariant still holds. `DrawCategory` being left
at zero on the unresolved path is now moot: the only branch that appends a
draw is the resolved one, so `DrawCategory` is always assigned together with
the draw itself. `docs/DIVERGENCES.md`'s `DIV-1353` row attributed
`SAV-DEADLOAD-126`'s three-independent-scalars content to both `125` and
`126`; `125` is the virtual post-load rebind claim and is now cited for that
alone. The row's "so a stage 2-4 record is a present actor after load" was
stated inside the claim-citation cell as if it were the claim's own text; it
is this lane's own reading of `SAV-DEADLOAD-128`'s stage-5 boundary and is
now marked as such, in both `DIV-1353` and this story's own Authority
section above. `game0032.sav` was called a "preserved install" in
`DIV-1353`; it is in the owner save corpus
(`gameversions/saves/2027-09-07/`), and `gameversions/{en,ru}` are the
preserved installs -- corrected.

## Open debt

- **The decaying/ticking half of DIV-1344's Recommendation remains
  unimplemented.** This record never advances past the `Current` tuple it
  was imported with, and it never occupies its cell for pathing, targeting or
  collision. `DIV-1353` records this as the bounded, deliberate scope of this
  story, not an oversight.
- **`Mission.DeadArt` does not survive a native (.ags) save/reload of the
  same mission, so a resolved record's presentation is lost on reload.** It
  is presentation-only, rebuilt by `applyOriginalDead` on every original-SAV
  resume and never entered into `sim.World`. The underlying dead-actor
  record itself round-trips through `sim.World`'s native byte form intact --
  `TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage` already
  proves `MarshalBinary`/`UnmarshalBinary` preserves `OriginalDeadActors()`
  exactly -- but `virtualDeadDraws` finds no `mw.art` entry for it after such
  a reload and draws nothing, the same "nothing" main draws for the record
  today. An earlier version of this function instead built the
  `ui.MapEntity` before that lookup and put an opaque diagnostic placeholder
  on the map for this exact case -- a player-visible defect the sole
  adversarial pass found and this story's one correction pass removed
  (Finding 1, `pipeline/reviews/story1201-adversarial.md`;
  `TestReleaseLateCorpseConstructor1201NativeReloadDrawsNothing` reproduces
  the reload through the real production SAVE seam and pins the corrected
  "draws nothing" answer). Persisting `TypeID` through the native form was
  weighed and set aside: it would need a `formatVersion` bump (95 to 96),
  widening `originalDeadRecordLen` (173 bytes, a fixed fully-packed layout)
  and touching `upgrade.go`'s bridge chain for every version since the
  94-pivot, disproportionate to this story's own bounded scope.
- **The draw carries owner 0, undisclosed until this correction.**
  `virtualDeadDraws` leaves `MapEntity.Owner` at its zero value rather than
  any owner slot; `OriginalDeadSource.OwnerKey` is a foreign identity, not a
  slot, so no claim establishes what owner such a record should carry, and
  this is an authored choice rather than a researched one. At owner 0 the
  record loses `SelfSlot`'s fog exemption (drawn only while its cell is
  currently visible, never once merely explored, unlike a player-owned
  body), draws in the minimap's non-local colour, and is tinted through
  `OwnerPalettes[0]` rather than a player-owned body's own palette (N1,
  `pipeline/reviews/story1201-adversarial.md`). No claim fixes the correct
  owner.
