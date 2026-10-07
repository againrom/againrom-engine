# Verification

Correction commit `75c6e977bad853fc80ae7492ad1a9697be13046b` (merge) on top of
`0005ceb` (the Cluster A/B fix) reconciles engine main
`5a30e37be25e9ac51bbab9e173487f119f384eae`. Knowledge pin is
`c4073aef7e22bbc849b0efdd2a04a13047e37ae5` (k37) on both sides of the merge —
unchanged, not merely non-regressed.

## Gates (all run on `75c6e97`)

- `gofmt -l .`: no output (clean).
- `go build ./...`: clean.
- `GOCACHE=.../.gocache go test -trimpath -count=1 ./...`: exit 0, every
  package `ok`, zero FAIL, including `pkg/video/smacker` and the other
  packages story1178/1179 added since this branch's prior verification pass.
- `scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree
  scan)`.
- `pipeline/check-div-claims.sh` (`AGAINROM_IMPL` set to this worktree): exit
  0. `check-div-claims: .../wt-story-1171-generated-world-sav at 75c6e97,
  knowledge pin k37`; 459 of 461 live rows selected (2 closed), citing 685
  claim ids; 130 rows cite a claim carrying a retraction row — the same
  standing informational advisory this branch carried before, unrelated to
  this story's own edits. DIV-1203 (the only row this session's fix touches)
  is not among the retraction-advisory rows and now reads `ACCEPTED`, not
  `OPEN`; see `docs/DIVERGENCES.md`.
- `pipeline/check-release-tests.sh` (`AGAINROM_IMPL` set to this worktree,
  one foreground invocation, EN then RU): exit 0.
  `check-release-tests: 9 package(s), 285 gated test(s), 2 root(s)`;
  `check-release-tests: ok (285 of 285 ran, 0 lacked a subject)` on **both**
  roots. 285, not the 283 last measured on this branch, because story1178
  and story1179 registered new gated tests in the interim; nothing in this
  correction pass narrowed the population. All six tests the review returned
  this story over are in that count and pass on both roots:
  `TestReleaseLegacyMission20EndpointScenarios` (`cmd/scenariofixture`),
  `TestReleaseDocuments1176ActualGrantsReturnShopColdSave`,
  `TestReleaseNativeConsumedCompanionGrant`,
  `TestReleaseTownReturnCurrentCampaign1168` (`fresh-mission20` subtest),
  `TestReleaseEngagement1163OriginalAndNativeContinuation`,
  `TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect`
  (`synthetic-terminal-zero-timer` subtest). Individually re-confirmed with
  `-run`/`-v` before the combined run, on `pkg/game` and
  `cmd/scenariofixture`, all `--- PASS`.
- `scripts/check-milestone2-acceptance.sh <EN> <RU>`: exit 0. `ok` on both
  roots (EN 32.428s, RU 32.509s); `TestMilestone2Players` (the story's
  earlier own fixture-count fix) and `TestMilestone2*` unit-scalar audits
  both clean, matching the branch's prior clean state — unaffected by this
  session's Cluster A/B fix, which touches a disjoint code path
  (`nativeCityHumanState`/`nativeCityAttachItems`, not
  `milestone2_players_compare_test.go`).
- `pipeline/check-preserved-installs.sh`: `ok — 554 file(s), every root as
  recorded`.
- No commit trailer of any kind on any commit from `75a43fe` (this branch's
  starting point) through `HEAD`, including the merged-in `origin/main`
  commits: `git log --format='%h %(trailers:key=Co-Authored-By)'
  75a43fe..HEAD` prints every short hash with an empty trailer field.

## The six tests: mechanism and fix

**Cluster B — `importSavedActorActions1171` phase/countdown decode**
(`TestReleaseEngagement1163OriginalAndNativeContinuation`,
`TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect`'s
`synthetic-terminal-zero-timer` subtest). The reader forwarded a raw
continuation value with no membership in `sim.Entity.ActorState`'s enum, and
separately resolved a legitimate boundary-phase idle actor's self-referential
`U5C` convention as a live attack target — the second defect is what made
`TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect` fail (a graph the
test expects to stay dead was read as re-engaging). Fixed by removing the
undefined forwarded value and gating the target/phase/countdown decode on
`OrdinaryTargetable()` in `pkg/game/savactoractions1171.go`. With the decode
correct, `TestReleaseEngagement1163OriginalAndNativeContinuation`'s own
oracle needed correcting: the pinned actor's raw order byte already reads 1,
so `AI-ORDER-039`'s arm2 does not fire again on resume; what governs
continuation is `HERO-CADENCE-023`'s phase/countdown pair, which the decode
now restores. The pinned phase/countdown reaches that machine's own
recovery-zero boundary one tick after resume (tick2987), not the
twelve-tick fresh charge from Ready (tick2999) the test asserted when the
decode did not exist yet. `pkg/game/engagement1163_release_test.go` now
asserts 2987, with the derivation recorded inline in the test's own comment;
no other assertion in that test changed.

**Cluster A — native TOWN/CITY save vs. a mission20-generated party member's
live pickups** (`TestReleaseDocuments1176ActualGrantsReturnShopColdSave`,
`TestReleaseNativeConsumedCompanionGrant`,
`TestReleaseTownReturnCurrentCampaign1168`'s `fresh-mission20` subtest,
`TestReleaseLegacyMission20EndpointScenarios`). `nativeCityHumanState`
(`pkg/game/nativecityhuman.go`) and `nativeCityAttachItems`
(`pkg/game/nativecityitems.go`) both used "`Carry.LiveLoad` set /
`ActorLoad.Source.Class != 0`" as a proxy for "retains an untranslatable
*imported* basis, refuse". That proxy was correct before this story existed
(only a genuinely-imported Human could ever reach `Source.Class` 2 back
then) but became wrong once this story's own `ConstructActorBasis`
(`pkg/mapload/actorconstructor.go`) started giving *every* generated Human
that same `Source.Class`, donor or not. Fixed by:

- `nativeCityHumanState` computing the member's own fresh re-derive first,
  then — only when `Carry.LiveLoad != nil` — verifying the retained
  `sim.SourceActor` against `mapload.HumanSourceActor` of that re-derive by
  field equality, refusing only on a genuine mismatch. `TypeID` is excluded
  (a legitimate cross-document numbering convention nothing reads across
  that boundary); the four fields `HumanSourceActor` never sets (`Reach`,
  `AttackCharge`, `AttackRelax`, `EquipmentRuntimePresent`) are populated on
  the rebuilt side from the same `data.Derived` values `ConstructActorBasis`
  itself uses, before comparing.
- The same fresh re-derive now inherits `ManaReservePercent` from the
  retained basis when one exists, instead of always writing the
  fresh-actor constant. This field is per-player, not per-character
  (`sav/actorgraph.go`), and the sim actively recomputes `ManaFloor` from it
  at runtime — the byte-equality check above caught this as a genuine,
  independent pre-existing bug (a real imported companion differed from its
  own fresh re-derive only in this one field), not a fixture artifact.
- `nativeCityAttachItems` now routes a `Carry.LiveLoad`-carrying member
  through `nativeCityAttachItemsConstruct` directly, instead of delegating to
  story1173's `nativeMissionItems1173`, whose own `Source.Class != 0`
  refusal is deliberately tested for a different population
  (`pkg/game/missioncity1173_controls_test.go`'s `source_human` case) and
  would incorrectly reject every generated actor too. The pre-existing
  refusal for `OrderedStacks` set with no `LiveLoad` (no current-load
  snapshot to verify a container total against) is unchanged — regressing it
  was caught by the pre-existing, untouched
  `TestNativeCityItemsRefuseMissingAndRetainedOperands` during this session's
  own full-suite check and fixed before commit.

A genuinely retained ORIGINAL basis (`member.OriginalHuman != nil`) still
requires lossless `.ags`, exactly as before this story.

## Corrections made this pass

- `docs/1171/story.md`'s Open debt section previously stated both
  Cluster-B tests load `game0013.sav`. Only
  `TestReleaseEngagement1163OriginalAndNativeContinuation` does;
  `TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect` loads
  `2027-09-07/game0031.sav` and `game0032.sav`
  (`pkg/game/originaldying1144_release_test.go`). Corrected.
- `docs/DIVERGENCES.md`'s DIV-1203 described the two clusters as unresolved
  regressions under `OPEN`. Rewritten to name the fix and moved to
  `ACCEPTED`, mirroring DIV-1202.
- The stale main SHA in `story.md`'s Result section is replaced with the
  actual reconciliation point of this pass (`5a30e37`).

## Milestone census

`missionrun -mission {10,20} -trace -ticks 1 | grep -c UNSUPPORTED`, built
from this commit (`75c6e97`), reports 0 for both missions on both EN and RU —
identical to `pipeline/milestone-baseline.txt` (which records no
`UNSUPPORTED` count for either mission, i.e. 0). Unchanged, as expected:
this story and this correction pass touch no `pkg/script` or map-script-
routing file.

## Remaining named debt

- `milestone2_players_controls_test.go` has no deterministic unit coverage
  isolating the `population.excluded` code path in
  `milestone2_players_compare_test.go` (an F58 actor archive with zero or
  several direct Player containers, or no live actor binding); the
  corpus-driven `TestMilestone2Players` exercises it but does not isolate
  it. Left as debt (review Finding 2), not closed under this pass's budget.
- Completed original persistence stays Medium/Unknown per `story.md`'s
  Authority and qualifications section. No RU original-runtime acceptance is
  claimed; RU proof here is limited to the same fixture-only witnesses as
  EN.

## What this lane did not do

No desktop input was sent (no window was driven). No original-runtime
process SAVE/LOAD acceptance was exercised. No merge to `engine/main` was
performed; this lane pushes its own branch and stops. No build-bundle
promotion (`builds/current/`) was performed. `pipeline/` and
`PIPELINE-STATUS.md` were not touched, other than reading
`pipeline/check-div-claims.sh`, `pipeline/check-release-tests.sh`,
`pipeline/check-preserved-installs.sh` and `pipeline/check-pin-forward.sh`
(the last found unable to resolve a linked worktree, noted above, not
fixed — it is seat-owned).
