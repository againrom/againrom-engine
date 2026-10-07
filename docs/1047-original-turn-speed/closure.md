# Closure — story 1047 original turn speed

## Twelve-aspect matrix

| Aspect | Status | Evidence |
|---|---|---|
| Data | PASS (round 3; was a mis-rated PASS in round 2, actually GAP — P1/P2) | Existing `RotationSpeed` columns (`pkg/data/unitdef.go`, `humandef.go`), constructor defaults and equipment modifiers (`pkg/mapload/effectapply.go`) reach the live actor: `mapload.blockFor`'s resolved, creature and unresolved-loadout arms, `PartyLoadout`, `PartySpawn`, `recomputeRaisedSkills` and every `game.Rearm` call site thread a resolved rotation base (`RotationSpeedBase`, `pkg/mapload/loadout.go:136`) through to `sim.SetDerived`. Measured live, through the production mission door, not the `.alm` file (`cmd/rotationcensus`, both lawful roots): 2361 of 2361 entities across the 28-mission campaign carry a positive `RotationSpeed`, 16 distinct values in 8..23, none at 0. One construction path is excluded and disclosed: a newly generated, non-hired hero takes the flat constructor default regardless of the chosen archetype's own row (`DIV-437`). **Round 2's own witness for this row never opened a hired party or exercised a raise**, so it missed two defects on exactly the population it claimed to cover: a raised Control Spirit ghost dropped `RotationSpeed` to 0 regardless of its Units row (P1, `pkg/mapload/ghost.go`), and a hired mercenary resolved the wrong Humans row for 36 of 52 shipped `NPC%02d_%d` TypeIDs (P2, `pkg/mapload/loadout.go`'s `RotationSpeedBase`). Both are fixed this round and both now have a dedicated production-code witness that fails against the pre-fix tree: `TestBuildMercenarySquadCarriesTheHiredRowsOwnRotationSpeed` (`pkg/game/mercenaryunit_test.go`, synthetic, mutation-verified) and `TestReleaseEveryReachableMercenaryCarriesItsOwnRotationSpeed` / `TestReleaseRaisedGhostCarriesTheInstalledUnitsRowRotationSpeed` (`pkg/game/rotationcensus_release_test.go`, real shipped data on both lawful roots, mutation-verified against the pre-fix code — see Round 3 below). |
| Runtime state | PASS | `Entity.DesiredFacing`, `Entity.TurnRemaining` and request-time `Entity.TurnTotal` sit beside `Facing`; the inactive/active invariant is enforced by `turnFault` (`pkg/sim/facing.go`) on both `MarshalBinary` and `UnmarshalBinary` (`pkg/sim/binary.go`). Positive rearm changes leave the stored pair intact; completion, replacement, death and the non-positive compatibility arm clear both progress values. |
| Simulation | PASS | Arc arithmetic, countdown, movement/attack/cast gates, route destruction and interruption; `pkg/sim/turn1047_test.go`, including this round's death-during-an-active-turn fixture (`TestDeathDuringAnActiveTurnClearsItWithoutCompletingIt`). |
| Player input | PASS | Move, attack and unit/cell cast commands reach the same `requestFacing` gate as AI decisions; no source-specific facing arithmetic remains outside `pkg/sim/facing.go`. |
| AI | PASS (with disclosed exclusion) | Approach, withdrawal and reacquisition pay turn intervals through the same gate. `AI-COST-071`'s `turnCost` scorer term stays explicitly undecoded and unchanged, per contract exclusion. |
| UI/HUD | PASS (pass-3 P1 fixed) | No HUD control added. `pkg/game/world.go` excludes a turning actor from swing selection and selects the standing octant through `Entity.DrawnFacing()`. The interpolation now divides by hashed request-time `TurnTotal`, not mutable live `RotationSpeed`, so an equip-frame rearm cannot make the drawn facing jump backwards, freeze or jump ahead. `TestTheDrawnOctantAdvancesAcrossAMultiTickTurn` covers the render call at a constant rate; `TestEquipAndUnequipKeepTheRequestTimeTurnDuration` covers that same call after production equip/unequip, with independently fixed first/middle/final octants and monotone arc progress. Each render read leaves its contemporaneous `World.Hash()` unchanged. Output resolution stays eight-way, not the original's sixteen-way (`ANIM-DIR-006`); that disclosed gap remains `DIV-438`. |
| Triggers/scripts | PASS | Every script-issued movement order (`cmdGroupCommandedMove`, `armPatrol`, `armDefend`, `armFollow`) writes `TargetX`/`TargetY`/`HasTarget`, the one field set `stepWorld`'s per-tick loop reads to reach `turnToward` (`step.go:949`); every script-issued attack order (`cmdGroupAttack`) calls `orderAttack`, this package's sole writer of `AttackTarget`, read by the same loop's `HasAttackTarget` branch into `approach` (`combat.go:345`). `requestFacing` (the sole writer of an active turn) has exactly one caller, `turnToward`, which has exactly four callers in the package: `step.go:949`, `combat.go:345`, `spell.go:1626`, `spell.go:1797` — the four producers this row already named. One script mechanism does not converge: instants 21/24 (`castAtCell`/`castAtUnit`, `pkg/sim/scriptcast.go`) build a temporary, non-Entity caster object (decode `DD-1`, story 0165) with no `Facing`/`RotationSpeed` field, resolved by `stepScriptCasts` outside the Entity turn model entirely — this is the decode's own shape, not a fifth path around the gate. No opcode changed. Mission 10 and mission 20 UNSUPPORTED script-node counts are unchanged (0 on both, EN root, see Milestone below). |
| Inventory/equipment | PASS (pass-3 P1 fixed) | Rotation-speed effects reach `sim.SetDerived` through the existing equip/unequip and rearm chain. A positive change affects the next request but preserves an active turn's `TurnRemaining` and `TurnTotal`; a non-positive change keeps the earlier compatibility guard, snapping to `DesiredFacing` and clearing the turn. `TestEquipAndUnequipKeepTheRequestTimeTurnDuration` applies real kind-18 item instances through `enqueueEquip` and `enqueueUnequip`, then runs `mapWorld.tick` in production order. Its six cases cover positive increase and decrease at first, middle and final remaining ticks. |
| Persistence/save-load | PASS | Byte form appends `TurnTotal` after the existing turn pair (current version 64, `entityLen` 294); `turnFault` refuses malformed total/remainder relations on encode and decode. Version 63 migrates losslessly by recovering the denominator that reproduces its renderer's saved-facing sample without changing `TurnRemaining`; earlier readable forms still migrate to an inactive turn with `lostTurnProgress`. `TestVersion63MigrationFreezesTheFacingItsRendererCouldDraw` covers unchanged, increased and decreased live rates plus byte-exact round-trip. The separate additive `mapload.PartyMember.HiredRotationSpeed` field is repaired at the resume ownership boundary from the saved member's exact named `Units` or `Humans` row; compatibility hotfix `51f3051e03f6a02e9d8e45f98af8a5a9ea9fa4b6` closes `DIV-436`. |
| Campaign/session | PASS | Turn duration is counted in actor ticks (`MOVE-CLOCK-032`); no session clock added. |
| Shipped content | PASS (partial witness, see Open items) | The constant-rate lawful-root witnesses remain: mission 100's positive-rate retreat on both roots and mission 10's 130 multi-tick patrol intervals across 20000 ticks, longest 6, with intermediate `DrawnFacing` octants. Pass-3 P1's shipped reachability is the positive kind-18 magic-effect population in both generated lawful-root tables; production consumes those effects through the same item/rearm seam the six-case synthetic witness exercises. The exact withdrawal outward-and-return drive remains W2, as recorded below. |
| Interactions with existing mechanics | PASS (pass-3 P1 fixed) | Action cadence (1045), route state, transit, withdrawal, visibility, death, Stone Curse, off-map return, effects and rendering remain covered by `pkg/sim/turn1047_test.go` and the updated pin suite. The pass-3 rearm/render counterexample is closed by storing request-time duration in canonical state: positive rearm changes cannot alter presentation progress, while the countdown and its hash remain canonical. The earlier raised-ghost, missing base-rate and non-positive rearm interactions remain fixed. |

No aspect is a known in-scope GAP. `pkg/sim/turn1047_test.go` and this round's death-during-turn
fixture cover the case list; the open items below are **W**-class findings (the production
behaviour the contract specifies is implemented and tested, and what is incomplete is a
verification instrument), not **P**-class, and do not block this landing.

## Integration witness

`missionrun -withdrawal` (extended in this pass to print `RotationSpeed`, `Facing`,
`DesiredFacing` and `TurnRemaining` alongside its existing tick-6/tick-7 report) was run
against the real, lawful EN and RU installs:

```
AGAINROM_ASSETS=<en>/gameversions/en  missionrun -mission 100 -withdrawal
AGAINROM_ASSETS=<en>/gameversions/ru  missionrun -mission 100 -withdrawal
```

Both roots print, byte-identically apart from the root:

```
withdrawal mission=100 map="scenario/100.alm" entity=109 class="Bat_Sonic.2" mode=wimpy threshold=4 radius=7 hostiles=1 rotationspeed=21
  before-tail tick=6 cell=(129,62) target=none attack=108 facing=0 desired=0 remaining=0
  after  tick=7 cell=(129,62) target=(131,59) facing=32 desired=32 remaining=1
```

This is round 2's re-run, after the P1 fix. Round 1's own run of this same witness printed
`rotationspeed=0` and `remaining=0` at tick 7 — the pre-fix state, where this actor's shipped
placement took `mapload.blockFor`'s unresolved arm and never received a rotation base. With the
fix, the same actor now carries `RotationSpeed=21` and its retreat order pays a real turn: `Facing`
0 to 32 (a one-direction arc, the shortest kind), `TurnRemaining` 1, and the cell held for the tick
the turn owns before the crossing starts. This is a one-direction, one-tick turn, not the
multi-tick reversal the arc/rate matrix in `pkg/sim/turn1047_test.go` covers synthetically; a
shipped scenario reaching a wider arc on a positive-rate actor, and a moving pursuer across
multiple legs, are not built (Open items, D8). This is real shipped data, not a test artifact.

`World.Hash()` was compared directly across both lawful roots for the first time this round
(remaining-surface item 3): `TestReleaseAMissionTenTurnHashMatchesAcrossBothLawfulRoots`
(`pkg/game/turn1047_release_test.go`) runs mission 10's own no-command patrol drive, with real
turns active, for 500 ticks on the EN root and again on the RU root and compares the resulting
digest. Both roots produce `0x472a211d931b5de0`.

The mission 10 / mission 20 unrunnable-script-node counts (this project's milestone
census; `pipeline/check-milestone.sh` is the seat's own instrument) are unchanged by this
story: both are 0 on the EN root, measured with a `missionrun` built from this branch
(`AGAINROM_ASSETS=<en>/gameversions/en missionrun -mission {10,20} -trace -ticks 1 | grep
-c UNSUPPORTED`), matching master. This story adds no script opcode and changes no
trigger dispatch, so no movement in this count is expected and none is observed.

**`pipeline/check-milestone.sh` exits red on this branch, and it is expected (D5).** Run with a
`missionrun` built from this branch on both roots, every script-node line is unchanged (the diff
above shows only mission 141/150/151 unchanged and no other mission moves), and only the drive
lines move, identically on both roots: `outcome lost at tick 256` becomes `tick 288`, and `4 of 36
unit(s) moved, 1 fell` is unchanged over the longer run. This is the story's own prediction coming
true — a turn interval now costs real ticks, so an unattended mission-150 escort's own unassisted
NPCs take longer to reach the same outcome — and it is a baseline update owed at the landing, which
is the seat's own call and the seat's own file (`pipeline/milestone-baseline.txt`). Neither
`--update` nor that file was touched by this round.

`cmd/rotationcensus`'s live-actor distribution independently corroborates `MOVE-LIMIT-033`'s
column-level census. That claim states the shipped `RotationSpeed` column carries 16 distinct
values in 8..23 with mode 19, over 333 combined Units/Humans rows, identical on both roots.
`cmd/rotationcensus` measures 16 distinct values in the same 8..23 range with the same mode (472
of 2361 entities at 19, the largest single bucket) over live spawned entities, not CSV rows.

**This agreement has no discriminating power over whether every construction arm reaches a real
row (round 3, correcting round 2's own reading of it).** The unresolved-actor constructor default
is 16 (`data.UnitDefaults().RotationSpeed`), which is already inside the shipped alphabet 8..23. A
construction arm that silently substituted the default for every actor would still land the whole
distribution inside 8..23 and would still very plausibly place a large bucket at or near the mode,
so the two-population alphabet agreement is consistent with correct wiring and is equally
consistent with the default substituting silently. This was demonstrated directly, not only
argued: mutating `unitCtorDefaults.RotationSpeed` from 16 to 8 and re-running the census still
produces a distribution inside the shipped alphabet (8 replaces 16 in exactly one entity per
mission, 28 of 2361 — the never-hired party hero; see `pkg/data/unitdef.go`'s own comment). What
the census DOES establish, and is the evidence this row should rest on instead, is the direct
count: 2361 of 2361 live entities carry a positive value and none carries 0, which is what P1 and
P2 each broke on specific actors (a raised ghost, 36 of 52 hired-mercenary TypeIDs) before this
round's fix.

## Research reconciliation

All seven claims cited in `contract.md`'s Research baseline were re-read whole from the
bumped pin (`be95a8b482cfed678625b5a7e4c8fa29c17264e6`) via `go run ./tools/claim <ID>`
in the research submodule. None carries a headline/body disagreement material to this
story, and none of the seven is retracted (`check-div-claims.sh` did not flag `DIV-429`
among the 67 rows it lists as citing a claim with a retraction row, re-run after this
round's edits to `docs/DIVERGENCES.md`):

- `MOVE-TURN-031`, `AI-FACE-066`, `AI-FACE-067`, `MOVE-CLOCK-032`, `MOVE-DIR-034`,
  `AI-COST-071` read as summarized in `contract.md`; no amendment since the contract was
  written moves their conclusions.
- `TERR-SPR-047` carries an amendment (`[EXP-0053]`) noting the routine this row
  originally read is a duplicate shadow pass; the arm-by-arm arithmetic it describes is
  true of both the shadow and the body it shadows, so the row's use here (state 5
  selects the standing frame) is unaffected. Noted for completeness, not because it
  changes this story's presentation rule.
- `DIV-437` cites `SESS-HERO-014`, one active row and one retracted row under the same id.
  The retracted row's overturn is scoped to the default-hero-name BSS arrays; the active
  row's derive/`Data.bin`-lookup content `DIV-437` cites is untouched by that retraction, per
  the retraction's own scope clause quoted in `claims/retracted.md`.
- `MOVE-LIMIT-033` (read for `cmd/rotationcensus`'s corroboration, see below) is unaffected:
  no amendment or retraction since it was published.

**The two-entry difference is completely accounted for by the collections' reserved index-zero
slots (pass-3 D1).** Data.bin's one-based Units collection allocates 119 entries and writes 118;
Humans allocates 216 and writes 215. The slice lengths therefore total 335 while the complete
written-row population in `MOVE-TURN-031` is 118 + 215 = 333. The two reserved slots are not actor
definitions and carry no `RotationSpeed` row, so no shipped actor, rate or construction branch is
unaccounted for. This distinction is also recorded at `pkg/data/unitdef.go`'s constructor default.

## Runtime entity introduction (adversarial return, section 7 item 1, closed)

Whether a script instant can introduce an actor into a live `sim.World` after mission open — a
reinforcement instant, `AddHero`, or a town joiner crossing into a mission mid-play — through a
path other than the one P1 fixed.

Searched: every append onto `w.entities` (the world's live entity slice) or onto any variable
named `Entities`/`entities`, across `pkg/sim`, `pkg/mapload` and `pkg/game`, by `grep` for
`append(.*entities` and `append(.*Entities\b` (case-sensitive, both forms) over every non-test
`.go` file in the three packages. Three matches: `pkg/sim/celleffect.go:1168` builds a local
`victims []CellPoint` slice of existing entities' cell coordinates, not an entity; `pkg/sim/step.go:1767`
builds a local `keep []Entity` slice by filtering the existing list (the world's own removal path),
appending no new entity; `pkg/sim/spell.go:340` is `w.entities = append(w.entities, ghost)`, the
raised-ghost append P1 fixes.

`AddHero` itself names no runtime mechanism in `pkg/sim`: `pkg/mapload/joined.go`'s
`CampaignNPCMember` and `pkg/mapload/start.go` reference it only in prose, describing a
town/campaign-level roster addition. `CampaignNPCMember`'s own doc states it "uses the same Humans
row, item-instance resolver and roster constructor as an NPC placement" — the same
`PartyMember`/`PartyLoadout`/`RotationSpeedBase` path P2 fixes, whether the member is added at
town or through a scripted mid-mission handover, because either way the member reaches a live
world only through world construction (`StartMission`/`MissionOpenerWith`), not through a live
append.

Conclusion: `pkg/sim/spell.go:340` (the raised ghost) is the entire population of code that adds a
new `sim.Entity` to an already-running world. P1's fix is this class's whole closure, not one
instance of it; there is no separate reinforcement, `AddHero` or mid-mission-joiner append site to
examine.

## Hired-`PartyMember` construction (adversarial return round 3, item 4, closed)

Every construction of a `mapload.PartyMember` that can satisfy `Hired()` (`MercenaryType != 0`) is
a producer of `RotationSpeedBase`'s hired input, and round 3's own P2 fix addressed one instance
(`buildMercenarySquad`'s human arm) while a second, `buildSiegeSquad`, still built a member with no
`HiredRotationSpeed` at all — reachable via `Rearm` at original-save resume
(`pkg/game/originalsave.go`'s `restoreOriginalActorStock`), where it produced `RotationSpeed = 16`
against the row's own 8 for a hired Catapult or Ballista, on both lawful roots (fixed this round,
`pkg/game/tavern.go`).

Searched: every assignment to `MercenaryType` in a `PartyMember{...}` literal, by `grep` for
`MercenaryType\s*[:=]` over every non-test `.go` file in the repository. Two matches:
`pkg/game/tavern.go:163` (`buildMercenarySquad`) and `pkg/game/tavern.go:243` (`buildSiegeSquad`).
Every other reference reads the field (`p.MercenaryType`), never writes it — `RestoreParty`
(`pkg/game/originalparty.go`) takes an already-built `[]mapload.PartyMember` and reconciles it
against a save's own records; it constructs no new hired member and cannot introduce a third
producer.

Conclusion: `buildMercenarySquad` and `buildSiegeSquad` are the entire population of code that can
mint a `PartyMember` satisfying `Hired()`. Both now resolve `HiredRotationSpeed` from the row in
hand at construction time. This is this class's whole closure; there is no third producer to find
by the same route a fourth review pass could take.

**What this fix does and does not reach, verified live.** A fresh mission start with a newly hired
Catapult or Ballista in the party was measured, before this round's `buildSiegeSquad` fix, at the
correct live `sim.Entity.RotationSpeed` (8) either way: `mapload.startMission`'s own
`p.MercenaryType == 1 || p.MercenaryType == 2` branch resolves the entity through `blockFor`
against the Units table directly (`definitionFor`, keyed by `ClassID`), not through
`RotationSpeedBase`, so the missing field was never read on that path. `RotationSpeedBase` and
`HiredRotationSpeed` are read by three other production consumers: `mapload.PartyLoadout` /
`PartySpawn` (used for every non-siege party member's own mission-start spawn, and by a live
equip/unequip recompute), `pkg/game`'s `Rearm` at `switchInventorySubject`
(`pkg/game/world.go:3218`, independently refused for any hired member since hotfix `403bf5af`,
2026-08-25, an ancestor of this branch — `member.Hired()` returns before `Rearm` is called, so this
call site was already dead for a hired Catapult regardless of this fix), and `Rearm` at original-save
resume (`pkg/game/originalsave.go`'s `restoreOriginalActorStock`, not gated on `Hired()`, and the
path this fix protects). Confirmed live with a throwaway probe against mission 10 on the EN root,
not committed: with the fix reverted, a freshly hired Catapult/Ballista's live entity read
`RotationSpeed = 8` at `StartMission` (unaffected, as above) whether or not `HiredRotationSpeed` was
set; `mapload.PartyLoadout(member, f.Table).RotationSpeed`, which `Rearm`'s three other callers all
resolve through, read `16` reverted and `8` fixed, matching the coordinator's own probe of
`RotationSpeedBase`'s inputs. The player-visible consequence is narrower than "any hired siege
engine's live turn rate": it is confined to a save resumed from an original ROM1 `.sav` file while a
hired Catapult or Ballista from the current session is present in the roster being reconciled.

## Divergence ledger

`DIV-429` is closed by this story. Round 1's adversarial review found the closure premature: the
turn interval it describes was unreachable on 100% of shipped content, because the actors it names
never carried a positive `RotationSpeed`. Round 2's `cmd/rotationcensus` (committed, both lawful
roots, 28-mission campaign) measures 2361 of 2361 live PLACED entities at a positive value, none
at 0. Round 2's own wording overstated that instrument as covering "every live entity":
`cmd/rotationcensus` opens each mission once with a one-member, non-hired starting party, so it
contains no hired actor and cannot see a Control Spirit raise. Both gaps were live defects, found
by adversarial pass 2 and fixed this round (P1, P2 above), and round 3 adds the population
`cmd/rotationcensus` could not see: `TestReleaseEveryReachableMercenaryCarriesItsOwnRotationSpeed`
(33 Humans templates, extended to also check both siege templates against their own Units row —
see the second P2 instance below) and `TestReleaseRaisedGhostCarriesTheInstalledUnitsRowRotationSpeed`
(the raise path), both release-gated on both lawful roots. The row's own Resolution and Revisit cells
are corrected in `implementation/docs/DIVERGENCES-CLOSED.md` to name the corrected scope; it
stays `CLOSED`.

`DIV-022`'s implementation text is updated to describe the desired-facing request and turn gate
rather than the pre-story instantaneous write, per contract.md's Research baseline.

`DIV-437` is opened by round 2, unchanged by round 3: a newly generated, non-hired hero's own
construction (`data.ChargenBase`, `pkg/game/hero.go`) does not read the chosen archetype's own
shipped `RotationSpeed` and takes the flat constructor default instead. See
`implementation/docs/DIVERGENCES.md`.

`DIV-438` is opened by this round (P3): `ANIM-DIR-006` (High) establishes ROM1's standing facing
as sixteen-way; this build's `sheetOctant` resolves eight-way, and `DrawnFacing()`'s linear
interpolation across a turn advances over those same eight octants rather than sixteen. P3's own
scope was the frozen-facing defect (drawn facing not advancing at all during a turn), not the
render resolution; sixteen-way quantization is a larger, separate change with no shipped-art
population yet measured, and is carried as `FIDELITY-DEBT`, `OPEN`. See
`implementation/docs/DIVERGENCES.md`.

`DIV-436` was opened by this round: `PartyMember.HiredRotationSpeed`, added this round (P2), is an
additive gob field, and a save written before it existed decodes it at zero. The later compatibility
hotfix `51f3051e03f6a02e9d8e45f98af8a5a9ea9fa4b6` closes that residue. At the resume ownership
boundary, before either a town adopts the party or a mission derives entities from it,
`mapload.RepairLegacyParty` resolves the saved member's exact named `Units` or `Humans` row and
restores its positive `RotationSpeed`; current-form nonzero values remain authoritative and a
second pass is inert. `TestRepairLegacyPartyCoversTownItemsAndExactHiredRows` covers both row
families, an unresolved name and idempotence. The corrected row is recorded in
`implementation/docs/DIVERGENCES-CLOSED.md`; no valid-save rotation-speed residue remains.

## Gated-test population census reconciliation (adversarial return round 3, item 3 of 3, closed)

`internal/gatedtests/testdata/population.txt` is 79 lines total: 10 comment lines and 69 data
lines. `gatedtests.Scan()`, run against this tree, finds 69 gated test functions, and
`TestScanMatchesTheCheckedInPopulationList` passes. The two agree; "79" was this round's own
earlier read of the file's total line count, not its data-line count, and is corrected here.

`pipeline/check-release-tests.sh`'s own runtime census, computed by its documented formula
(`grep -B1 -- '--- SKIP' | grep -oE 'AGAINROM_[A-Z_0-9]+'`, then counted), reads 70 on this same
tree with all four `AGAINROM_` variables unset. 70 and 69 are not the same population measured
twice. Each instrument has its own blind spot, in opposite directions, and the near-agreement is
coincidental:

1. **Scan() misses three tests reached through a two-level helper chain.** Scan()'s own header
   discloses the limit: it resolves a gated `t.Skip` one call level deep (the test's own body, or
   one same-package helper it calls directly by name) and no further.
   `TestReleaseGroundPickerQueuesTheCornerMeshCellOnRealMissionContent` calls
   `openReleaseGroundMission`, which calls `releaseFront(t)`, the actual `t.Skip` site: two levels.
   `TestReleaseMissionMapLeftTapSelectsThenOrdersOnRealMissionContent` and
   `TestReleaseMissionMapRightUpCancelsOnRealMissionContent` reach `releaseFront(t)` the same way,
   through `releaseMissionInputApp`. All three are genuinely install-gated, are present in the
   runtime census, and are undercounted by Scan() alone.

2. **The runtime census misses two gated tests entirely, from how `go test -v` renders subtests.**
   `TestReleaseJoinedHeroHandoverTickComposesExactPanePopulation` (7 subtests) and
   `TestReleaseCampaignJoinRoutesAreImmediateInteractiveAndPersistent` (2 subtests) each skip
   inside a `t.Run` closure, gated on the same `AGAINROM_ASSETS` check as every other release test
   (`pkg/game/release_integration_test.go:35`). `go test -v` prints every subtest's own `=== RUN`
   and log line as it runs, then prints the parent's `--- PASS:` line followed by all of that
   parent's `--- SKIP:` lines together, as one block, once every subtest has finished. No
   `--- SKIP:` line in that block is immediately preceded by the `AGAINROM_`-naming log line; each
   reason was logged earlier, during the `=== RUN` phase. Read directly from a captured
   `go test -trimpath -count=1 -v ./...` run on this tree with `AGAINROM_ASSETS` unset:
   ```
   === RUN   TestReleaseJoinedHeroHandoverTickComposesExactPanePopulation/npc24_Naira
       release_integration_test.go:607: no AGAINROM_ASSETS: release integration needs a lawful install
   --- PASS: TestReleaseJoinedHeroHandoverTickComposesExactPanePopulation (0.00s)
       --- SKIP: TestReleaseJoinedHeroHandoverTickComposesExactPanePopulation/fixed_Brian (0.00s)
   ```
   The parent itself reports PASS, never SKIP, and none of its subtests' `--- SKIP:` lines carries
   an adjacent `AGAINROM_` token. Both tests are genuinely install-gated (Scan() finds each by its
   own direct `t.Skip` call inside the `t.Run` closure) and both are absent from the runtime
   census's population, its `remaining` count, and its `nosubject` count alike: the script's own
   accounting cannot see them skip. When `AGAINROM_ASSETS` is supplied, both run their subtests
   normally and are not blocked; the blind spot is in the script's own count of how many gated
   tests exist, not in its pass/fail verdict for an ordinary invocation.

The true population of top-level, `AGAINROM_`-gated test functions in this tree is the union of
what each instrument gets right: Scan()'s 69, plus the 3 two-level-helper tests the runtime census
finds and Scan() misses, for **72**. Neither 69 nor 70 is the right count on its own; each is short
by what the other instrument's blind spot removes (Scan() short by 3, the runtime census short by
2), and the two shortfalls happen to leave a residual of 1 between the raw totals that reads as
near-agreement and is not.

This changes nothing in `internal/gatedtests` or `population.txt`: the one-call-level scope is
disclosed in `Scan()`'s own header, and this finding is consistent with that disclosure rather than
a defect in it. Fixing `pipeline/check-release-tests.sh`'s subtest-adjacency grep is a seat-gate
change, out of this lane's scope, and was not made. Authoritative for "is the gate clean" on an
ordinary invocation: the script's own printed `remaining`/`nosubject` counts, understanding now
that a gated test built on `t.Run` subtests is invisible to that specific accounting and needs a
direct read of the `go test -v` log, as done here, rather than the script's summary line, if its
correctness is ever in question. Authoritative for "how many gated tests exist in this tree": the
union count above, 72, not either raw census figure.

**A third mechanism, found while tracing this: the census run's own population count is
contingent on the calling shell's ambient `AGAINROM_ASSETS_RU`, which the script never controls.**
`pipeline/reviews/1047-lane-return-round3.md`'s own earlier gate log recorded a single-root
invocation selecting 70 install-gated tests and a paired invocation selecting 69, with a guess at
the cause that this round corrects. The census run (the part of the script that computes
`population`) unsets `AGAINROM_ASSETS`, `AGAINROM_SAVE_666`, `AGAINROM_ORIGINAL_SAVES` and
`AGAINROM_SAVE_CORPUS` but not `AGAINROM_ASSETS_RU`, so that one variable's ambient value at
invocation time carries into the census. `TestOriginalConditionalPanelCaptionsRenderOverBothLawfulInstalls`
(`pkg/game/installtext_test.go:591`) calls `t.Skip` only when neither `AGAINROM_ASSETS` nor
`AGAINROM_ASSETS_RU` yields a root; a paired invocation exports `AGAINROM_ASSETS_RU` before calling
the script, so during the census (with `AGAINROM_ASSETS` unset but `AGAINROM_ASSETS_RU` still set
from the caller's own environment) this test runs its "second" subtest instead of skipping, and
the population count is one lower than a single-root invocation's. Verified directly: a census
run with `AGAINROM_ASSETS_RU` unset finds this test's `--- SKIP:` line; the same run with
`AGAINROM_ASSETS_RU` exported does not, and no other test's presence changes. This is a real
defect in the script's own census computation (its population is supposed to represent "every
variable unset," and one of the five relevant variables is not part of that unset list), separate
from the two mechanisms above; not fixed here for the same reason as those.

## Open items

1. **D7 (plan.md) — the AST-based facing-writer census under `internal/archtest` was not
   built.** In its place, this pass manually enumerated every write of `Facing`,
   `DesiredFacing`, `TurnRemaining` and `TurnTotal` across `pkg/sim`'s production sources (`grep` for
   every assignment, increment and composite-literal key against the four field names,
   cross-checked against every `Entity{...}` composite literal in the package). The
   population found matches the contract's four gated runtime producers
   (`step.go`, `combat.go`, `spell.go` x2, all through `facing.go`'s `requestFacing`)
   plus three construction/restoration sites (`world.go`'s `newWorld`,
   `binary.go`'s `UnmarshalBinary`, `spell.go`'s `raisedGhost`). The manual census found
   and this pass fixed one defect: `raisedGhost` left `DesiredFacing` at its zero value
   (commit `331d5193`). Unlike the destination-flag census this story's own D7 would have
   built, this manual audit is a one-time check, not a standing regression guard: a future
   writer added outside the four gated call sites will not fail a build. Recommend a
   follow-up pass building D7 as a bounded, single-purpose addition to
   `internal/archtest`, modelled on `internal/archtest/destination.go`.
2. **D9 (plan.md) — the lawful-root two-phase (outward and return) turn witness named in
   contract.md's Witnesses section is partially built.** Round 2's re-run of `missionrun
   -withdrawal`, after the P1 wiring fix, now exercises a real positive-`RotationSpeed` actor
   (`Bat_Sonic.2`, mission 100, `RotationSpeed=21`) instead of the 0-rate compatibility arm: the
   printed tick-6/tick-7 pair shows a real one-direction, one-tick turn on the outward leg. It does
   not step the drive loop far enough to observe the return turn after reacquisition, and the arc
   this shipped case reaches (one direction, one tick) is the smallest kind the arc/rate matrix
   covers, not a multi-tick reversal. `pkg/sim/turn1047_test.go`'s literal arc/rate matrix and
   movement/attack/cast gate tests cover the arithmetic and the gates independently of this
   witness. What remains open is a headless demonstration of a multi-tick turn and a return turn
   against a real placed positive-`RotationSpeed` ranged actor on both roots, as contract.md's
   Outcome paragraph names. Recommend a follow-up pass extending `driveWithdrawal` to continue
   stepping past reacquisition and to select a shipped candidate whose retreat direction reaches a
   wider arc.
   Round 3 adds a different, general-purpose witness for the multi-tick-reversal half of this
   gap on different shipped content: mission 10's patrol drive (see Shipped content above), not
   the `driveWithdrawal` scenario this item names. The `driveWithdrawal`-specific extension
   (return turn after reacquisition, on this specific withdrawal scenario) remains not built.

D8 (return-brief round 2, item 8, "withdrawal under a moving pursuer is not built") is closed by
adversarial pass 2's own check rather than carried forward as an open item. Pass 2 ran three
probe geometries in `package sim` through `-overlay`, all against a real positive-rate actor
with the `Wimpy`/`Withdraw` arms live: (A) two pursuers closing on their own AI at rate 19 and at
rate 0, no stall; (B) five sub-cases over the `Wimpy` radius (rates 19, 8 and 0, three facings,
one and two pursuers), no stall in any; (C) the hostile's cell rewritten by hand every tick to
force `withdrawFrom`'s destination to change and starve the countdown -- the countdown fell
monotonically from 12 to 0 and the actor moved at tick 18. The withdrawal destination held at
`(17,21)` throughout because `withdrawFrom` computes `tx` as `self.X ± 3`, a function of
`sign(dx)` alone, and `ty` from a coarse truncated ratio, so a pursuer moving one or two cells
does not change the destination and `cancelTurnForTargetChange` does not fire. Scoped to what was
run (three geometries, rates 0, 8 and 19, one and two pursuers), the hypothesis that a moving
pursuer can starve or extend a withdrawing actor's turn countdown is refuted. No further build is
owed against this item.

Item 9 (return-brief round 2) — death and interruption during an active turn — is now closed, not
open: `TestDeathDuringAnActiveTurnClearsItWithoutCompletingIt` (`pkg/sim/turn1047_test.go`) kills an
actor mid-turn through the production `KindKill` command path (not the reviewer's invalid
already-dead-at-construction probe) and asserts `clearFelled`/`clearTurn` leave `TurnRemaining` 0,
`TurnTotal` 0, `DesiredFacing == Facing`, and `Facing` unchanged from its pre-death value — a corpse
keeps the facing it held at death, not the direction it never finished turning to. The assertion was
mutation-checked against `clearTurn` completing the turn instead of cancelling it, and fails under
that mutation.

Both remaining open items (D7, D9) are **W**-class findings by this project's finding taxonomy:
the production behaviour the contract specifies is implemented and tested
(`pkg/sim/turn1047_test.go`'s arc/rate matrix, movement, attack, cast, interruption and
death-during-turn cases; the persistence and migration suite), and what is incomplete is a
verification instrument, not the behaviour itself. Recorded here rather than treated as a blocking
gap. D8 (round 2's third open item) is closed above by adversarial pass 2's own check rather than
carried forward.
