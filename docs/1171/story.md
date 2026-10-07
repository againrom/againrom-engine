# Generated first-world SAV

## Result

A generated fighter enters full mission20 with no donor save, performs real
movement and melee damage, writes an original-format world SAV, cold-loads
that SAV with no donor, continues the same native checkpoint, and saves again
through both the App SAVE route and the CLI `ConvertSave` route, on EN and RU.
`TestReleaseGeneratedWorldSAV1171` (`pkg/game/generatedworld1171_release_test.go`)
carries this end to end and is registered in
`internal/gatedtests/testdata/population.txt`. The owner's earlier stop
instruction is superseded; this document replaces the prior paused-WIP draft.

The branch reconciled engine main `5a30e37be25e9ac51bbab9e173487f119f384eae`
(current `origin/main` at reconciliation time, which had by then landed
story1177's stable actor identity/audio hotfix, the `saverepair -w`
documentation hotfix, story1178's movie-audio landing and story1179's
portable Smacker decoder). Knowledge pin k37, `c4073ae`, unchanged on both
sides of the merge — confirmed directly by comparing the `knowledge`
gitlink at this branch's tip and at `origin/main`'s tip, since
`pipeline/check-pin-forward.sh` cannot resolve a linked worktree's `.git`
file as a repository root and reports "not a git repository" there.

## As-built behaviour

`constructGeneratedWorld1171` (`pkg/game/generatedworld1171.go`) builds one
complete Document and live `sim.World` before tick0: for mission20 only, given
a chargen party at least as long as `ms.Start.IDs`, it clones the installed
world, runs each entity through `mapload.ConstructActorBasis` and
`ConstructActorLoad`, admits the result with `world.ImportOriginalLivingActors`,
and reassigns the live `ms.World` pointer to that clone at the end — mission20
gameplay, rendering and the SAV export/import round trip all share the same
constructed world, not a side-channel export-only copy. Generated actors carry
`SourceBinding.Class` 4 (Unit) or 5 (Human), distinct from imported 1..3;
`ActorClass()` maps both back to the imported Unit/Human semantics downstream.

An empty or short party (the production mission door can open mission20 with
none at all) and any `mapload.ConstructActorBasis` refusal (absent Human
basis, unset table row, invalid portrait) both degrade to the same
`Unavailable` SAV-document fallback the missing-installed-table guard already
used, rather than a hard `OpenMission` failure — nothing has been committed to
`ms.World` at that point in any of the three cases.

Settling each entity's `Book.State` away from `BookLegacy` also sets
`ms.Party[i].SpellbookRestored`/`SpellbookPresent` to match, so the
pre-existing, unmodified `mapload.CarryParty` (which mirrors `e.Book` off the
live entity at every later mission boundary) does not trip
`validateSnapshotBooks`'s consistency check the first time a generated party
reaches its next mission.

Ordinary SAVE and the CLI `ConvertSave` "ags->sav" direction share the same
current-projection/remint/Document-encoding writer already built for story1170
(`ExportCurrentWorldSave`); no donor, embedded File.Body, second graph or
native save-schema field was added for the generated origin. The optional
`/CurrentState/AgainromRng` leaf (DIV-1202, kind6, 12 bytes: version1 plus the
low/high dwords of the native SplitMix64 state) carries the RNG stream through
every cold cycle below with no sidecar.

### Four regressions fixed this session

`importSavedActorActions1171` (`pkg/game/savactoractions1171.go`), the reader
this story added alongside the story1170 writer, had two defects that failed
original SAV LOAD outright across the wider corpus, not just mission20:

- It forwarded a raw actor-state continuation value of 1, which has no
  membership in `sim.Entity.ActorState`'s own enum (pickup-completion is 2;
  `savedActorStateSupported`'s different, pre-existing raw order-state space
  in `savedgroupsai.go` is where 1 actually belongs) — `actorStateDefined`
  rejected it downstream. Removed from the forwarded case set.
- It resolved a legitimate boundary-phase idle actor's self-referential `U5C`
  convention (witnessed at full HP with `U54=U58=U6C=0, U136=1` in the
  AreaDamage1164/1111/1170 corpus fixtures) as an attack target, which
  `sim.ImportOriginalActorActions` correctly treats as definitionally invalid.
  Now normalized to "no restored target".

Both were hard LOAD failures, not subtle drift; fixing them turned
`TestReleaseWorldEffectsContinue1162` from failing to passing — it was a
regression from this story's own earlier work, not pre-existing or unrelated
debt as an earlier draft of this document claimed.

The generated-world Book-restore gap above (`SpellbookRestored` unset) and the
`ConstructActorBasis`-refusal hard failure above are the third and fourth:
both were also confirmed as this story's own regressions by A/B testing
against pristine `origin/main` in a disposable worktree, and fixing the third
turned `TestReleaseDifficultyCampaignThroughProductionUI` passing; fixing the
fourth turned `TestReleaseQuickSpellsSaveFreshLoadAndCast` passing (a
controlled-input test party with no `FigureFace` hit exactly that gap once it
reached mission20).

`engine/scripts/check-milestone2-acceptance.sh` (the `sessioncorpusaudit`
family; not part of `check-release-tests.sh`'s own population and never run
against this branch before this session) failed `TestMilestone2Players` on
one corpus file (`2026-08-15/game0017.sav`, both roots) once run. Root cause
was not in this story's own `savdiaries1171.go`: `originalActorRegistry`
(pre-existing, `originalactorregistry.go`) numbers the first retained map
unit's live entity `0`, a genuine identity, not a "missing" sentinel, and
`importSavedDiaries1171`'s completeness (a Diary for every live archive-bound
actor, superseding the older party-only `applyOriginalDiaries`) was the first
caller to actually reach that actor's Diary. The comparison test itself
(`milestone2_players_compare_test.go`, pre-existing, not a story-1171 file)
read its `map[uint16]sim.EntityID` membership with a bare `==0`/`!=0` check at
three sites, so a real actor whose EntityID is `0` looked unbound. Fixed by
switching all three to the comma-ok form; `TestMilestone2*` passes in full on
both roots after.

## Proof

EN, `AGAINROM_ASSETS=gameversions/en AGAINROM_SAVE_CORPUS=gameversions/saves`:

```
go test ./pkg/game/ -run TestReleaseGeneratedWorldSAV1171 -v
```

- initial generated population: actors=57 structures=30 sacks=4
- hero (entity 56) at (13,14) targets entity 1 at (34,44), HP20
- melee damage at tick 519: HP 20 -> 12
- ordinary App SAVE: actors=57 structures=30, SAV=37535 bytes
- cold LOAD with no donor: actors=57 structures=30
- checkpoint RNG native=cold=`0x87d4fba1d0c84897`
- 80 successor ticks: native App session, its byte-identical AGS copy and the
  cold session stay `sim.World.MarshalBinary`-equal; independent per-entity
  `SourceNow`/HP/phase/countdown/regeneration/XP checks all pass
- second SAVE from the cold session, second cold LOAD, 80 more successor
  ticks: same independent checks pass a second time
- CLI `ConvertSave` ags->sav route, from the second cold session's own AGS:
  37449-byte SAV, opens as a SAV, cold-loads to actors=57 structures=30 with
  full `SourceNow`/HP/phase/countdown/`Book` equality against its source

RU, same command with `AGAINROM_ASSETS=gameversions/ru`: identical population,
hero/target/tick/RNG and per-entity checks; ordinary SAV=37531 bytes,
CLI-converted SAV=37445 bytes. Byte counts differ from EN by locale text only.

Advanced-RNG refusal boundary (dead-actor scenario, not generated but sharing
the same writer/reader): `TestReleaseCurrentWorldDeadRandomRefusal1170` — RNG
`0b98ea18ebb9dd79` retained through 161 SAV samples, a second cold SAVE and
the CLI ags->sav converter.

Initial graph/cell/identity loss and provenance-corruption controls: the same
test's seven subtests (`initial_position`, `identity_permutation`,
`provenance_Identity`, `provenance_RuntimeID`, `provenance_T0C`,
`provenance_T0E`, `provenance_U4C`) all pass.

Registration: `go test ./internal/gatedtests/... -run
TestScanMatchesTheCheckedInPopulationList` passes with
`TestReleaseGeneratedWorldSAV1171` listed.

## Authority and qualifications

REG099..102 establish the bounded raw registry/name/type constraints; SAV914..916
show named temporary application readers and fresh registry construction on
original SAVE (DIV-1202). The extension is authored Againrom state; completed
original persistence remains Medium/Unknown, and original resave must not be
reported as lossless native-RNG carry. SAV908 supplies Unit+40/+64/+68 repair;
MAGIC-ITEMKILL-117's portable-credit inference and ITEM-CASTSTATE-056's
only+68 clause are partially retracted; SAV776's alleged caller chain is
withdrawn. Effect+44 is separate and original LOAD-to-next-consumer lifetime
is Unknown. HERO-CADENCE-023, as currently amended (High for the phase
machine itself), qualifies the actor+0x58/+0x6c physical phase/countdown
bytes `importSavedActorActions1171` decodes; it does not by itself establish
that decode is safe to apply to every original actor regardless of how its
current attack cycle began — see Open debt. ALM-PLAYER-069, SAV664/667/844,
terrain/cell claims and physical cadence claims remain the constructor's
qualified sources. Policies are in DIV-1198..1203. No original runtime was
started and no original-runtime acceptance is claimed.

## Milestone census

`missionrun -mission 10 -trace -ticks 1` and `-mission 20 -trace -ticks 1`
each report 0 `UNSUPPORTED` script nodes, on both EN and RU, matching
story1168's own `pipeline/milestone-baseline.txt`-referenced measurement of
the same two missions. This story touches no `pkg/script` or map-script-
routing file, so the count is unchanged by construction, not only by this
one run. See `verification.md` for the exact command and output.

## Open debt

The review (`pipeline/reviews/story1171-pass1.md`) returned this story over
six failing corpus-gated release tests in two clusters, both self-inflicted
regressions from this story's own earlier work. Both are now fixed for real,
not narrowed or skipped; the six tests themselves are unmodified except for
one, `TestReleaseEngagement1163OriginalAndNativeContinuation`, whose own
oracle constant was stale (below).

**Cluster B — `importSavedActorActions1171` phase/countdown decode.**
`TestReleaseEngagement1163OriginalAndNativeContinuation` (loads
`2026-08-14/game0013.sav`) and `TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect`'s
`synthetic-terminal-zero-timer` subtest (loads `2027-09-07/game0031.sav` and
`game0032.sav` — not game0013.sav, correcting an earlier draft of this
document) both load a pinned actor already mid-action. The actual defect was
in `importSavedActorActions1171` itself, not in these tests' oracles:

- It forwarded a raw continuation value with no membership in
  `sim.Entity.ActorState`'s enum, rejected downstream by
  `actorStateDefined`.
- It resolved a legitimate boundary-phase idle actor's self-referential
  `U5C` convention as a live attack target, which
  `sim.ImportOriginalActorActions` correctly treats as invalid — this
  produced the "owner graph resurrects" failure in
  `TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect`'s
  `synthetic-terminal-zero-timer` subtest, fixed by gating the decode on
  `OrdinaryTargetable()`.

With both fixed, `TestReleaseEngagement1163OriginalAndNativeContinuation`'s
own oracle needed correcting too: the pinned actor's raw order byte
(`U158[9]`) already reads 1, so `AI-ORDER-039`'s arm2 (the sole writer of
that byte) does not fire again on resume; what governs continuation is
`HERO-CADENCE-023`'s own phase/countdown pair at actor+0x58/+0x6c, which
`importSavedActorActions1171` now restores correctly. The pinned raw
phase/countdown (7/1, Relaxing with one tick owed) reaches
`HERO-CADENCE-023`'s own recovery-zero boundary one tick after resume
(tick2987), not the twelve-tick fresh charge from Ready (tick2999) the test
asserted before this decode existed. `pkg/game/engagement1163_release_test.go`
now asserts 2987 with the derivation recorded inline; nothing about the test's
target/state assertions changed.

**Cluster A — native TOWN/CITY save vs. a mission20-generated party member's
live pickups.** `TestReleaseDocuments1176ActualGrantsReturnShopColdSave`,
`TestReleaseNativeConsumedCompanionGrant`,
`TestReleaseTownReturnCurrentCampaign1168`'s `fresh-mission20` subtest, and
`cmd/scenariofixture`'s `TestReleaseLegacyMission20EndpointScenarios` all
carry a mission20-generated party member through real gameplay back to town
and then require a lossless native TOWN/CITY SAV.

The root cause was not that native TOWN/CITY export cannot represent
live-acquired inventory/basis state at all — it always could. It was that
`nativeCityHumanState` (`pkg/game/nativecityhuman.go`) and
`nativeCityAttachItems` (`pkg/game/nativecityitems.go`) both used
"`Carry.LiveLoad`/`ActorLoad.Source.Class != 0` is set" as a proxy for
"retains an untranslatable *imported* basis, refuse" — a proxy that was
correct before this story existed, when only a genuinely-imported Human
could ever reach `Source.Class` 2, but is violated by this story's own
`ConstructActorBasis` (`pkg/mapload/actorconstructor.go`), which
unconditionally gives *every* generated Human that same `Source.Class`
whether or not it has a donor. The fix does not weaken either refusal for
the population they were built for (a genuinely retained ORIGINAL basis,
`member.OriginalHuman != nil`, still requires lossless `.ags`, and
`nativeMissionItems1173`'s own, differently-populated `Source.Class`
gate for mid-mission-live exports is untouched):

- `nativeCityHumanState` now computes the member's own fresh re-derive
  first, and — only when `Carry.LiveLoad != nil` — verifies the retained
  `sim.SourceActor` against `mapload.HumanSourceActor` of that same
  re-derive by field equality, refusing only on a genuine mismatch.
  `TypeID` is excluded from the comparison (a legitimate cross-document
  numbering convention nothing reads across that boundary); `Reach`,
  `AttackCharge`, `AttackRelax` and `EquipmentRuntimePresent`, which
  `HumanSourceActor` does not set, are populated on the rebuilt side from
  the same `data.Derived` values `ConstructActorBasis` itself uses before
  comparing.
- The same function's fresh re-derive now inherits `ManaReservePercent`
  from the retained basis when one exists, instead of always writing the
  fresh-actor constant `nativeCityManaReserve`. This field is a per-player,
  not per-character, setting (`sav/actorgraph.go`:
  `a.Character.Basis.Human.ManaReservePercent = a.Owner.PlayerF58`) that
  the sim actively recomputes `ManaFloor` from at runtime as `ManaMax`
  changes — the byte-equality check above caught this as a genuine,
  independent pre-existing bug (a real companion NPC resumed from an
  original SAV differed only in this one field) rather than a fixture
  artifact, and fixing it is what let the check be exact rather than
  fuzzy.
- `nativeCityAttachItems` now routes a member with `Carry.LiveLoad` set
  directly through `nativeCityAttachItemsConstruct` instead of delegating
  to story1173's `nativeMissionItems1173`, whose own `Source.Class != 0`
  refusal is deliberately tested for a different population
  (`pkg/game/missioncity1173_controls_test.go`'s `source_human` case, mid-
  mission-live exports reached through `exportMissionCity1173`) and would
  incorrectly reject every generated actor too. The pre-existing refusal
  for `OrderedStacks` set with no `LiveLoad` (no current-load snapshot to
  verify a container total against) is preserved unchanged; so is
  `nativeCityAttachItemsConstruct`'s own weight-consistency check.

All six tests pass on both EN and RU; see `verification.md` for the exact
gate output and the full release-test count (now larger than 283 after
reconciling `origin/main`, which landed unrelated stories in the interim).

Completed original persistence stays Medium/Unknown per the qualifications
above. No RU original-runtime acceptance is claimed; RU proof here is limited
to the same fixture-only witnesses as EN.

One remaining, narrower debt (from the review's Finding 2, unrelated to
either cluster above): `milestone2_players_controls_test.go` has no
deterministic unit coverage for the `population.excluded` code path; the
existing corpus-driven `TestMilestone2Players` exercises it but does not
isolate it. Left as debt rather than added under this correction pass's own
budget; a future touch to that exclusion path should add a table case rather
than rely on the corpus alone.
