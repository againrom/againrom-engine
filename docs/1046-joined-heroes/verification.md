# Story `1046` — verification

This record accounts for every requirement and design decision in `spec.md`. Commands run from the
story worktree. The outside-test result is the EN/RU production-route join witness; the mission
script-support census is not intended to move.

## Contract-to-evidence map

| Requirement | Evidence | Verdict |
|---|---|---|
| FR1, complete producer population | `TestNPCDefsResolvesEveryPersistentCampaignHeroTier` enumerates the ten selector results. `TestReleasePersistentJoinProducerPopulationSelectsTenExactRows` runs all ten on each lawful root. | PASS |
| FR2, one pre-mint selector and constructor | `TestCampaignMissionSelectsTheTieredComposedRow`, `TestTownCampaignMemberUsesTheSameExactRowConstructor` and the release population witness cover campaign map, town and fixed-row arms. | PASS |
| FR3, atomic exact payload | `TestReleaseCampaignJoinIdentitiesAndMission20ClubmenUseCanonicalRows` compares literal Brian/Naira identity, statistics, skills, XP, all carried/worn slots, effects and prices. `TestEquipmentInstanceStoresAggregateOrdinaryAndCastEffectPrices` checks ordinary and cast stored-price formulas. | PASS |
| FR4, same-tick live transfer | `TestScriptHandoverJoinsEveryOrdinaryHeroSurfaceInTheSameTick` observes ownership and party membership in the same tick. Both production-route subtests reach the shipped instants. | PASS |
| FR5, ordinary UI and inventory | `TestReleaseJoinedHeroHandoverTickComposesExactPanePopulation` covers fixed Brian, both `npc23` branches and all four `npc24` variants on each lawful root. It selects each foreign map actor before GiveUnit, then compares the ownership-changing tick's complete composed pane pixel-for-pixel with the exact joined figure and full live equipment. Both Naira arms are also compared against Danath and naked-Naira panes. The two release routes retain the prior inventory subject on the handover tick, then bind inventory to the retained selection before ordinary queued unequip/equip on the next production frame. | PASS |
| FR6, save/load once | The same-tick test checks the snapshot entry prefix and two world survivors. Both release routes encode, decode and restore, then find one joined actor with exact live XP and equipment. | PASS |
| FR7, campaign and terminal state | Both release routes mint the actor exactly once in the next mission. `TestJoinedRosterDeduplicatesTheRuntimeActorAndCullsDeadCandidates` covers same-id skip and dead candidate exclusion; existing join-boundary tests cover win, loss and ordinary second-boundary carry. | PASS |
| FR8, independent shipped witnesses | The gated population, exact-payload and route tests are selected by the release manifest and run independently on EN and RU. | PASS |
| DD1, data-owned selector | `NPCDefs` tests cover fixed, composed, tiered and absent records without a game-layer copy of the formula. | PASS |
| DD2, shared row constructor | Town and map tests reach `rosterTemplate` through the same definition-id input and compare complete row outputs. | PASS |
| DD3, post-step observation | The synthetic GiveUnit test would fail if membership appeared before ownership or one tick later. | PASS |
| DD4, world as live state | Reversible equipment, saved-world decode and restored entity comparisons read `World` state, not a presentation copy. | PASS |
| DD5, structural save dedup | Snapshot contains only the entry hero while saved `BoundarySurvivors` contains both; restore returns two party members once. | PASS |
| DD6, parallel ids | `CarryRosterIDs` and the same-tick/release assertions compare party length, id length and joined position at every boundary. | PASS |
| DD7, price at construction | Exact source and post-save item comparisons retain one `ItemInstance.Price`; no consumer recomputes it. | PASS |
| DD8, literal research expectations | Release tables spell EXP-0242 values directly and run against each selected lawful root. | PASS |

## Repository and release gates

The production candidate, joined-figure correction and viewer-witness correction were measured serially:

| Gate | Result |
|---|---|
| `go test -p 1 -trimpath -count=1 ./...` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test -p 1 -trimpath -count=1 -tags ebitenstub ./...` | PASS |
| `go test -race ./pkg/sim` | NOT RUN: the host has no `gcc`; the first attempt without CGO also reported that `-race` requires CGO. This is a toolchain limit, not a passing race claim. |
| `scripts/check-no-game-assets.sh` | PASS |
| `scripts/check-claim-citations.sh` | PASS: 1,362 distinct citations resolved against 1,615 claims and 238 experiments under 835 prefixes. |
| pinned-research `go test`, `go build`, `go vet` | PASS at `be95a8b482cfed678625b5a7e4c8fa29c17264e6`. |
| pinned-research claim, regeneration and retraction gates | PASS: 1,615 distinct ids in 31 ledgers; 29 regeneration scripts honour `OUT`; all 255 overturned ids are marked. |
| `pipeline/check-pin-forward.sh story/1046-joined-heroes` | PASS: the candidate pin is 25 research commits ahead of master's pin. |
| `pipeline/check-div-claims.sh` | PASS structurally: it read this worktree and `be95a8b`; 248 live rows of 248, zero closed, nine cells per row. It listed 66 pre-existing rows citing partially retracted claims; none concerns joined-hero production or consumption. |
| `pipeline/check-scenarios.sh`, EN and RU | PASS: 15 of 15 on each root, including `0159-mission40-join.json`. |
| `pipeline/check-release-tests.sh`, EN and RU | PASS: 63 of 63 install-gated tests ran on each root, zero skipped. |

The first EN release run exposed a witness defect: the older Reniesta defence test cleared only
legacy worn codes after the exact constructor began carrying canonical worn instances. Commit
`d420b0be` clears both representations; the focused witness and then all 63 release tests pass on
both roots.

The W-1 correction replaces the previous cache and non-nil-pane assertions with an exact viewer
frame comparison. A mutation that cleared `SetUnitPortrait` for composed Humans failed the Brian
pane comparison. A mutation that changed the installed joined face failed the `npc24` Naira arm.
Production was restored byte-identical after both mutation runs.

Two owner-tree checks are red outside this branch. `pipeline/check-preserved-installs.sh` reports
the already populated EN save slots and changed `game0000.sav`, `game0001.sav` and `game9999.sav`
against its recorded manifest; this story only read both installs. `pipeline/check-seat-tree.sh`
builds and vets the seat checkout successfully with the isolated cache, but reports its unrelated
untracked `.gocache-hotfix/`. Neither path is in this worktree and neither was changed.

## Milestone and outside-test result

The required one-tick trace reports zero `UNSUPPORTED` lines for missions 10 and 20 on both EN and
RU. The complete milestone instrument also passes: all 28 script-census rows and the 240-tick drive
match `pipeline/milestone-baseline.txt` on each root. In particular, the recorded mission 10/20
baselines remain `16 checks, 27 instants, 12 triggers` and `14 checks, 15 instants, 11 triggers`;
this story was not intended to move the script-gap counter.

The outside-test result is the release route, not a test-only fixture. On EN and RU it opens shipped
mission 40, selects Brian while he is foreign, drives the handover at `(76,108)`, and checks his
localized hero state, body art, full live equipment and composed selected figure in the
ownership-changing tick while the inventory subject still names the primary. One ordinary frame
binds inventory to Brian; the witness then unequips and re-equips his exact item, saves/restores
once, and opens the next mission with him once. The second arm does the same for Naira in shipped
mission 70 at `(38,108)`. The independent exact population witness selects all ten persistent
joined-hero variants from the installed data on each root. The seven-variant viewer witness uses
the same installed mission maps, roster rows, equipment and pane art, performs GiveUnit through an
ordinary simulation step, and checks the exact pane produced by the retained map selection.

## Remaining surface

The full producer and consumer census is in `closure.md`. No in-scope GAP remains. Combat-dependent
Brian progress is preserved and tested as live state rather than asserted as a constant; forced
replay beyond normal campaign state is excluded and receives no invented identity guard.
