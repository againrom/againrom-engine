# Generated mission SAV the original plays

## Result

A mission SAV the engine writes for a generated/natural campaign mission (the
M7 path, `FrontEnd.ExportCurrentSave` → `currentMissionDocument` →
`materializeCurrentWorld`/`currentSpatial`) now carries the field values
SAV-1091..1098 establish an original mission-10 LOAD requires, for a freshly
opened, never-saved mission: no row-0 Weapon/Armor/Shield definition,
nonzero mover RotationSpeed, publication mask 2 on every Human/Unit/
Building/Sack token, a nonzero Building type word, sack/ground-actor
block-plane occupancy bits, and AutoGetMission naming the actual next
mission. Five owner rounds (EXP-0397, landed at knowledge k74/k75) took a
generated mission-10 SAV from a LOAD fault to a playable, owner-corrected
file; this story adopts that evidence in the engine's own generation path
rather than the owner's hand-corrected copy.

Base: engine main `bda0711`, reconciled to `e33306f` (docs-only) before the
final gates. Knowledge pin: k75
(`0ea8f8429ad24461c02c7c4fd9ae635f22143409`), moved forward from k74 per a
seat note mid-task; k75 only amends SAV-1097/SAV-1098 and
`formats/sav/world.md` (a sack click now both walks the hero to the sack and
completes the pick-up, and message 17 appears at 38,64) and does not change
any of the fields this story fixes.

## The seven points

1. **Weapon/Armor/Shield definition row 0** (SAV-1091). Not a live bug: the
   diagnostic run found 0 of 41 Weapon/Armor/Shield records with definition
   row 0 in a fresh mission-10 document. No change.
2. **Mover RotationSpeed nonzero** (SAV-1092). Was a live bug: all 36
   Human/Unit mover blocks exported byte `+0x0a` as 0, although the live
   `Entity.RotationSpeed` was correctly nonzero for all 36 entities.
   Root cause: `sim.ProjectActorMotion` (`pkg/sim/currentmotion.go`) read
   `e.SourceNow().MoverSpeed` directly. A generated mission's NPCs never
   populate `ActorLoad.Source` on the live `sim.World` entity (that basis is
   only ever built inside SAV-time DTO-adapter code, in a local copy that is
   never written back), so `SourceNow()` returns its early-return zero value
   for `Class == 0`, carrying `MoverSpeed == 0` with it. `pkg/game`'s own
   `currentActorSource`/`savedActorValueRecord` already guarded this case by
   falling back to `uint8(e.RotationSpeed)`, but `ProjectActorMotion` — which
   runs afterward and overwrites the whole 180-byte mover block wholesale —
   did not. Fixed by adding the same `Class == 0` fallback at that one
   remaining call site.
3. **Token+0x18 publication mask 2, Building type word** (SAV-1093). Was a
   live bug: all 18 Building, 17 Human, 19 Unit and 4 Sack tokens exported
   mask 0. Root cause: `nativeCityToken` (`pkg/game/nativecity.go`), the
   shared token constructor for Human/Unit/Sack, never wrote bytes 23:25 (the
   T18 field, per the existing `mustSetToken` raw-offset mapping); the live
   Building constructor (`pkg/game/currentworldspatial.go`'s `currentSpatial`)
   never set `SavedStructure.Token18`.

   First pass wrote mask 2 inside `nativeCityToken` itself. That function is
   shared with the native-city/town SAV builder, where a fresh Human token
   must legitimately stay unpublished; the unconditional write broke the
   pre-existing `TestReleaseNativeCityPlayerConstruction`, caught by a direct,
   unbuffered rerun of the gated population (the wrapper's own
   `check-release-tests.sh` buffers all `go test -json` output behind two
   layers of command substitution and gives no incremental signal, so a
   direct `go test -v` run against the reconstructed `-run` pattern was used
   for this comparison instead). Corrected by reverting
   `nativeCityToken` to leave T18 untouched and instead writing mask 2
   explicitly at each of the mission-context call sites that construct a
   Human/Unit/Sack token for the current-SAV producer:
   `pkg/game/currentworldbuild.go` (`currentRecordActor`, Human/Unit),
   `pkg/game/currentworldspatial.go` (Sack, placed after the bound/unbound
   branch converges, since the bound path fully replaces the record) and
   `pkg/game/savcurrentitems.go` (`projectCurrentUnboundSackRoots`, Sack); the
   dead-code `pkg/game/generatedworldspatial.go` received the same scoped
   write for consistency. The pre-existing
   `pkg/game/nativecityplayer_test.go`'s shared `checkPlayerWireN4` helper
   already distinguished mission-context calls from native-city calls by
   `len(current)` for its ownership check; its T18 assertion now uses the
   same discriminator: mask 2 only from `ExportCurrentWorldSave`
   (`current_mission_and_pending_choice`), 0 from `ExportNativeCitySave`
   (every other subtest). The live Building constructor's `Token18` field is
   a separate, single-purpose struct field, not shared with the city path,
   and needed no such scoping; it now writes mask 2 directly.
4. **AutoGetMission names the next mission** (SAV-1094). Was a live bug:
   `Campaign+0x110` exported as 0; every original mission-10 save carries 20.
   `nativeCampaignProjectionForChapter`, shared with the city-SAV path, has a
   long-documented gap here (DIV-881..883: "no method in campaignprogress.go
   reads them"). Fixed in the mission-SAV-only branch
   (`pkg/game/currentsave.go`'s `currentMissionDocument`) by reading the
   already-loaded `Campaign.AutoAdvance(mission)` — the current-state
   successor scenario.reg already declares for `[Mission10]` — the same
   mechanism `frontend.go` already uses for live progression. Mission 20 has
   no declared successor in the shipped `Campaign.Auto` table (DIV-1360), so
   the owner kit's mission-20 file correctly carries AutoGetMission 0; that
   is not a defect.
5. **Control state group** (SAV-1095). Not fixed: research grades this
   Medium for the group as a whole and Unknown for which single byte drives
   first-LOAD control (hero Token sub-cell, cleared pending order, the
   `+0x178` word list, the 462-byte and 19-byte blocks, Player body `+57`,
   group AI `+0x48`, session wire 1400..1415, changed together in one
   round). The rest position (128,128) and cleared pending-order fields the
   engine already tracks are correct by construction (DIV-1183/1189); the
   remaining fields have no current-state source or research-established
   constructor value. Disclosed as DIV-1381.
6. **AI-start state** (SAV-1096). Not fixed: same Medium/Unknown shape —
   post, the 19-byte block, order/mover start bytes, group centre/guard
   radius and the diplomacy template were copied together in one round with
   the driving field unidentified, and the live World has no group-centre,
   guard-radius or diplomacy-template state to derive them from. Disclosed
   as DIV-1382.
7. **Block-plane rows: static 0x20 at record cells, dynamic 0x40 at
   ground-actor cells** (SAV-BLOCK-011, TERR-PASS-051). Was a live bug: sack
   cells exported static/dynamic 0, and 0 of 36 ground-actor cells carried
   dynamic bit 0x40 (all 36 do now). Root cause:
   `currentSpatial` (`pkg/game/currentworldspatial.go`) built `doc.World.Blocks`
   directly from the ingested terrain plane's own `Static`/`Dynamic` bytes,
   with no per-cell overlay for the sack/actor occupancy this generator adds.
   The `sim.ConstructSavedCellPlanes`/`recomputeSavedCell` machinery that
   already implements the correct TERR-PASS-051 rule
   (`static = payload|0x20`, `dynamic = static|0x40` ground /`|0x80` air) is
   wired only into the original-resume path
   (`pkg/game/originalcellrecords.go`) and into `pkg/game/generatedworldspatial.go`'s
   `spatial()` method, which this investigation found is dead code — nothing
   in the tree calls it. Fixed with a local, minimal inline overlay in
   `currentSpatial`'s own final cell/block loop, applying the same bit rule
   only to cells this generator already knows have a sack, ground actor or
   air actor.

## Correction pass (review f874c23)

The seat's sole adversarial pass (`pipeline/reviews/story1222-review-f874c23.md`)
returned five findings against the state above. All five are fixed; none
returns to review a second time under this story's own one-correction rule.

- **F1 mover mask** (`U154+0x05`): unaffected by this pass; already derived
  from the domain byte at `U4A` in the prior round.
- **F2 Unit Capacity/own-weight fallback** (Capacity=300,
  `U8E==U90`). The prior round's fallback in
  `savedActorValueRecord` (`pkg/game/savactorproject.go`) was live but was
  being silently discarded: `currentWorldDocument`'s own unconditional
  recompute pass (`worldsave.go`), which runs immediately after
  `materializeCurrentWorld`'s fresh construction inside the same
  `ExportCurrentSave` call, re-derives every actor's stats from
  `Entity.SourceNow()` — naturally zero for a generated actor — and
  overwrote the just-written repair before it ever reached the document
  bytes. Found only by running `TestReleaseGeneratedMissionSAVOriginalConstraints`
  with `AGAINROM_ASSETS` set; every prior "green" `go test ./...` in this
  story's history had silently skipped that test population for want of
  installed assets. An intermediate fix threaded a broader `freshMission
  bool` signal (true whenever the caller supplied no prior document) down to
  the recompute pass; that broke two dozen pre-existing fixture tests
  (`TestCurrentProfileZeroPeriodSurvivesSAVAndRegen` and others under
  `pkg/game`) that build a synthetic `sim.World` with a deliberately zero
  live Capacity and assert an exact SAVE/LOAD round trip, since a session's
  first SAVE of such a world also has no prior document. The recompute pass
  (`currentWorldDocument`, `worldsave.go`) instead reuses the same
  `Snapshot.NativeMissionTerrain` signal F3's own block-plane scoping already
  threads through it: the repair belongs to exactly the ROM1-byte-matching
  kit export F2 names, the same scope F3 names, not every first save of a
  session. An ordinary re-save, city return, mission-city capture or fixture
  round trip leaves `NativeMissionTerrain` false and keeps the repair off,
  because those documents may carry an original's own loaded
  Capacity/own-weight bytes, or a deliberate test value, this pass must not
  overwrite.
- **F3 Building T0E, block-plane scoping and values**. `T0E` (not `T0C`) was
  already correct. Row *scoping* (the sweep window, `dyn > 0x0f` only) was
  already correct via the prior round's `Snapshot.NativeMissionTerrain` opt-in.
  This pass closes the remaining byte-*value* gap the review's own reference
  file (`review/owner-sav-exp0397-r5/candidates/game9237.sav`) exposes:
  - The border plane read `03/03` where the original writer's own encoding
    is `1f/1f`. `currentSpatialPlanes` (`currentworldspatial.go`) folded
    domain 2's scenery/border bit into a fresh world's raw grid the same way
    `originalstructures.go`'s LOAD path already folds it for a previously
    loaded document, using the same `grid&2!=0 || Overlay[at]!=0` test.
    Verified exact: 1,657 in-window border cells, all `1f/1f`, against the
    reference file.
  - A structure's own attachment cells read `01/01` (open ground-block only)
    or `00/00` where the reference reads `25/25` (footprint) or `20/20`
    (record-only). `currentSpatial`'s building-attachment loop now ORs
    static-object bit 2 into both the cell's own `DocumentCellData.Static`
    and the plane array `currentBlockPlaneDelta` reads from, but only for an
    attachment cell the terrain arms computation already reads as
    ground-blocked (`v.Static&1 != 0`); an attachment cell the terrain
    leaves open stays a plain record cell, matching the reference's own
    split exactly: 112 cells at `25/25`, 35 at `20/20`, plus the pre-existing
    26 ground-occupant (`60/20`) and 10 (`61/21`) rows the prior round
    already got right. `pkg/game/generatedworldspatial.go`, confirmed
    dead code (no caller anywhere, including tests), is deleted rather than
    kept as a second, unreferenced builder, per the review's own note.
  - **Residual, named debt**: 3 of 1,841 m10 sweep-window cells (0x3f15,
    0x4016 present in the reference but absent from this engine's output;
    0x143f present here but absent from the reference) do not match. `0x143f`
    is inside structure 9's own 3x3 `Attach` bitmap as this engine computes
    it; the other two are not attributable to any structure this engine
    places. This is a structure-footprint-shape or actor/sack-placement
    discrepancy, not a block-plane encoding defect — the encoding rule
    itself (border/footprint/record/occupant values) now matches the
    reference on every cell both sides agree is populated. Not chased
    further in this pass: it is a single-digit-cell, sub-0.2% gap outside
    what F3 named, and this story's one-correction rule does not extend to
    open-ended structure-shape research.
  - `TestReleaseGeneratedMissionSAVOriginalConstraints` now sets
    `snapshot.NativeMissionTerrain = true` before export, so it asserts F1-F5
    as class constraints against the actual narrow, ROM1-byte-matching path
    this story exists to fix, not the ordinary round-trip-safe path it was
    silently asserting against before.
- **F4 AutoGetMission**: unaffected by this pass; already the scenario's own
  declared successor.
- **F5 AI-start post state**: unaffected by this pass; DIV-1382 already
  narrowed to `SAV-1101`'s isolated personal AI-start record in the prior
  round.

## Evidence

SAV-1091 through SAV-1098 (`knowledge/claims/sav.md`, EXP-0397), TERR-PASS-051
and SAV-BLOCK-011 (`knowledge/claims/terrain.md`), DIV-881..883, DIV-888,
DIV-1360. Read via `go run ./tools/claim <ID>` against the pinned k75
snapshot, never the raw claim files.

## Proof

`TestReleaseGeneratedMissionSAVOriginalConstraints`
(`pkg/game/generatedmissionsavfields_test.go`) opens mission 10 through the
owner's exact `App`/`OpenMission` path with no prior save, exports the
current document, decodes it and asserts each of points 1-4 and 7 as field
CLASS constraints (not byte identity against any owner file): no row-0
Weapon/Armor/Shield; nonzero mover RotationSpeed on every Human/Unit; mask 2
at T18 on every Human/Unit/Building/Sack; static bit 0x20 on every sack
cell; dynamic bit 0x40 on every ground-actor cell; AutoGetMission == 20.
It replaces a throwaway diagnostic test used only to find these gaps.

`go test -trimpath -count=1 ./...` is clean on the full tree; `gofmt` clean;
`internal/archtest`, `internal/gatedtests` (population.txt updated with the
new test) and `internal/storyguard` (CommentBytes raised to 7714768,
documented in `internal/storyguard/baseline.go`, for this story's own new
code, the publication-mask scoping correction, and its own comments) all
pass. `scripts/check-no-game-assets.sh` clean.

The gated release population was run directly (`go test -v` against the
`-run` pattern `internal/gatedtests/testdata/population.txt` builds, the same
pattern `check-release-tests.sh` constructs) against both EN and RU roots,
covering all nine gated packages. EN: 30 failing root tests, RU: the same
population; both sets are byte-identical to the pre-existing baseline
recorded in `review/story1221-land-bda0711/release-tests.log` (49 baseline
root/subtest lines collapsing to the same 30 root test names) — zero new
failures, zero newly-resolved failures, on either root. Logs at
`review/story1222-gates-e33306f/direct-en-fresh.log` and
`direct-ru-fresh.log`. `scripts/check-milestone2-acceptance.sh` (fresh rerun
after the point-3 correction) is clean on both roots: 0 mismatches/refused
across every instrument.

The pre-existing `TestReleaseGeneratedCampaignMissionsSAV/10` predicts
SAVE's campaign shape from a pre-save sample of the live, unexported
`Snapshot.Campaign` (`generatedMissionSample`,
`pkg/game/generatedmissions_release_test.go`); it needed the same
`Campaign.AutoAdvance` derivation point 4 added to
`currentMissionDocument`, or its own prediction stays at the pre-fix
AutoGetMission 0 while the reloaded SAV now correctly carries 20. Fixed
by mirroring the production derivation in the test helper; confirmed
passing locally (`go test ./pkg/game/ -run
TestReleaseGeneratedCampaignMissionsSAV/10 -v`, all four subtests PASS).

missionrun UNSUPPORTED census (`cmd/missionrun -trace -ticks 1 | grep -c
UNSUPPORTED`), before (engine main `e33306f`) and after (this branch):
mission 10, 0 → 0; mission 20, 0 → 0. Unchanged — this story is SAV
field-export correctness, not script/trigger routing, and neither mission
had an unsupported script node before or after.

### Correction pass proof

`go test -trimpath -count=1 ./...` is clean on the full merged tree (branch
merged with origin/main's sharp-bilinear hotfix, `28b3b0c`); `gofmt` clean;
`internal/archtest` and `internal/storyguard` (`CommentBytes` re-measured to
7741655 after the merge and this pass's own edits) pass.
`scripts/check-no-game-assets.sh` clean.
`TestReleaseGeneratedMissionSAVOriginalConstraints` now sets
`snapshot.NativeMissionTerrain = true` before export and passes for both
missions, asserting F1-F5 against the actual narrow export path. A
kit-generation tool (never committed; deleted before every gate run, per
`internal/archtest`'s unregistered-package check) regenerated
`review/owner-sav-story1222/game9248.sav` (m10, 141,894 B,
`975578fc61ffc09356de90bb589c3d3748e715e1818209ffee14c6ec875907dc`) and
`game9249.sav` (m20, 211,509 B,
`50269ea4d1812982f71aecc35711cd01bdd32035760c52c18cab556d698cc8af`), verified
against `cmd/savtool blocks -n 0` cell by cell.
`scripts/check-milestone2-acceptance.sh` and `pipeline/check-div-claims.sh`
were re-run on this pass; `pipeline/check-preserved-installs.sh` is clean
(569 files, every root as recorded).

`pipeline/check-release-tests.sh` was re-run against both the EN and RU
roots on the merged, corrected tree. Both roots produce the same 30
root-level failing test names (47 lines counting subtests); this set is
byte-identical to `review/story1221-land-bda0711/release-tests.log`'s own
30/47 — zero new failures, zero newly-resolved failures, on either root.
These are pre-existing AGS-codec/native-continuation gaps this story does
not touch.

An intermediate attempt at the F2 fix threaded a new `freshMission bool`
signal (true whenever the export's own caller supplied no prior document)
instead of reusing `Snapshot.NativeMissionTerrain`; that broke two dozen
pre-existing `pkg/game` fixture tests that build a synthetic zero-Capacity
`sim.World` and assert an exact SAVE/LOAD round trip, because a fixture
test's first SAVE within its own session also has no prior document. Found
by running the full `go test -trimpath -count=1 ./...`, not the targeted
release test alone. Corrected by removing the separate signal and reusing
`NativeMissionTerrain` (`pkg/game/worldsave.go`, `pkg/game/currentsave.go`);
the full suite is clean afterward.

missionrun UNSUPPORTED census reconfirmed on this merged, corrected tree:
mission 10, 0; mission 20, 0 — unchanged from both engine main and the
pre-correction branch state recorded above.

## Open debt

DIV-1381 (control state group) and DIV-1382 (AI-start state), both in
`docs/divergences/persistence-current-sav.md`, both OPEN and narrowed by the
review correction pass (see above). DIV-1383 and DIV-1384 of the story's
DIV-1381..1384 reservation are returned unused.

A 3-of-1,841-cell residual gap on mission 10's block-plane delta (cells
0x3f15 and 0x4016 present in the reference file but absent from this
engine's output; 0x143f present here but absent from the reference) is
named debt in the Correction pass section above: a structure-footprint-shape
or actor/sack-placement discrepancy this pass did not chase further, since
it is outside what F3 named and the story's one-correction rule does not
extend to open-ended structure-shape research.

`pkg/game/generatedworldspatial.go`'s `spatial()` method (and the
`sim.ConstructSavedCellOccupants`-shaped machinery it exercised via
`ImportOriginalCellRecords`/`ConstructSavedCellPlanes`) was confirmed
pre-existing dead code (no production or test call site reached it) and is
deleted, per the review's own note against keeping two builders.
