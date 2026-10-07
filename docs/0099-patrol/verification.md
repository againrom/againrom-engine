# Verification — 0099

Base: `fa415af`. Branch `story/0099-patrol`, three commits. Research pin `e6f9ee6`, unmoved from the
story's start; `git submodule status` shows no leading character. Every number below was taken on a
clean tree at `b20135d` unless it says otherwise.

## The gate

    go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')   clean
    go test -trimpath -count=1 ./...                                     EXIT=0, no failures
    bash scripts/check-no-game-assets.sh    clean (tree scan), EXIT=0
    bash scripts/check-doc-budget.sh        EXIT=0
    bash scripts/check-sdd-audit.sh         one FAIL before this file existed:
                                            "0099-patrol: every task in tasks.md has landed and
                                            there is no verification.md". No other FAIL.

No declared overrun. spec 11147/13312 (83%), plan 9930/13312 (74%), tasks T1 1309/1400, T2
1373/1400, analysis 5392/7168, provenance 4171/16384 — all inside the ordinary ceilings.

## What the game does now

`missionrun -mission 10 -census`, both roots, byte-identical output:

| | before | after |
|---|---|---|
| no drive, 4000 ticks | `0 of 36 moved, 0 fell`, undecided | **`2 of 36 moved, 0 fell`**, undecided |
| the milestone drive | `2 of 36 moved, 1 fell`, lost at 272 | `4 of 36 moved, 1 fell`, **lost at 272** |

The two new movers with no drive at all are

    moved  u57    slot 2 group 18  (64,21) -> (60,19)  246 step(s), 45 hp
    moved  u58    slot 2 group 17  (67,12) -> (58,13)  269 step(s), 45 hp

which are exactly the two units the map's two Patrol nodes name, and **exactly two**. Both are slot
2, `Villagers`. Nothing else on the map moved, fell or lost a hit point over 4000 ticks.

`almtool script` on `10.alm`: `instant arm 6 sub-command 14 (group command): 2 node(s) not
implemented` **before**, and the line is **absent after**. Every other row is unchanged, including
sub-commands 11 (1 node) and 15 (2 nodes), and `inert triggers: 1 of 12`.

`10.alm` is byte-identical on the two roots (`md5 2d983ccbf249c5336ebb7ccc41fc405c`), and every
measurement above was run on both.

## Mission 10's outcome — measured, not tuned (AC-6)

**Unchanged, and this story never claimed otherwise.** Before and after, on both roots:

    waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (39,41),
                Chebyshev 20, after 272 ticks
    outcome lost at tick 272

`TestTheTenthMissionIsDrivenToAWin` fails with exactly that message before and after, on both
roots. It was not edited, and neither was the loss predicate.

The census makes the reason legible: the same two units move in the driven run before and after —
`u21` (driven by us) and `u32 slot 4 group 7`, from (48,44), driven by nothing. A tick-by-tick read
of the protected unit's health says `u32` is the **only** entity ever holding her as a victim, and
it takes her from 45 HP to −2 between ticks 145 and 250 while she walks. Her group and `u32`'s
group are named by **no** script node on this map. Left undriven she takes no damage in 4000 ticks.

So the hypothesis the story was opened on — that the two Patrol nodes walk her captors away — is
**refuted**, and was refuted before any code was written. `analysis.md` carries the measurement.

## The witnesses

- **AC-1** the `almtool script` census above.
- **AC-2** `TestPatrolWalksTheMissionTensPatrollers` (`pkg/game/patrol_test.go`): resolves the two
  through `mapload.ScriptUnits` by map ids 57 and 58, steps with no commands, asserts each leaves
  its placement cell and returns. **Run against both lawful roots: passes on both.** It skips
  without an asset root, so it is not part of the default gate.
- **AC-3, AC-4, AC-5** `pkg/sim/actorform_test.go` — round trip to identical bytes and identical
  digest, a version-20 form refused naming both versions, and each of FR-13's four refusals on a
  form built to carry exactly it.
- **AC-7, FR-9** `TestOrderNoneIsNotDecidedOver` (`pkg/sim/patrol_test.go`).
- **SC-1** a `go/ast` read of `actor.go` asserting the pass's switch has exactly one case and no
  default, plus a behavioural test that an unhandled state leaves an entity in every field. Go
  offers no runtime reflection over a switch, so the parse is the honest mechanical check and is
  named as such rather than dressed up as a behavioural one.
- **SC-2, FR-1..FR-8** `pkg/sim/patrol_test.go`.

## Two things that were checked by deletion, not by reading

- **The decide gate.** Reverting `|| order == orderNone` on `engage.go:365` makes
  `TestOrderNoneIsNotDecidedOver` fail on **both** subtests — a member of an order-0 group is given
  an attack target, and another is released on an empty candidate list. Re-run from this seat, not
  taken on report.
- **`formatVersion`.** The record's own length constant and every re-pinned digest are checked by
  the round trip, which fails on a form that does not decode to itself.

## Discrepancies, reconciled

**`spec.md` FR-9 was wrong and has been revised in place before landing.** It said a patroller's
exemption from the group decision "follows from FR-4 and needs no rule of its own". It does not:
FR-4 makes order 0 storable, and the decision's own reachability test is the same predicate the
decoder uses, so widening that predicate silently opened the ordinary scoring loop to a group the
actor layer had just been handed. The decision needs an explicit clause, it has one, and AC-7 is
what makes it checkable. Caught by T2 while making AC-7 hold, not by review.

**A brief-level number was wrong and the executor was right.** The entity record was **100** bytes
on master, not 92; 92 is an offset inside the layout comment. It is now 118.

**One landed architecture pin was widened, not weakened.** `internal/archtest`'s
`TestDestinationWritersMatchFR2` (0097 AC-11) is an allow-list of the functions that may give a
unit a destination. `armPatrol` is a genuine sixth, and 0097's own FR-2 named this class of writer
in advance. The test still fails on any writer not on the list; the two diff tests were updated to
match the new count.

## What was not done

- **No owner-review artifact**, per the standing rule that a story's deliverable is the build.
- **The guard post and the re-anchor latch are not modelled** (D-3). Decoded, cited in
  `provenance.md`, owed to the story that adds actor state `0xb`.
- **`missionrun`'s `aim` is still occupancy-blind.** Named, not fixed; a separate story. No
  conclusion here rests on a single waypoint line — every one is corroborated by the census.
- **No claim is made about `openMission`/`openMapWorld`.** `-census` drives `ms.World` through
  `StartMission` and touches neither front-end path.

## The build

`builds/0099-patrol/` — `againrom.exe`, `missionrun.exe`, `almtool.exe`, `restool.exe` and a README
whose every command was run from that directory before it was written down.
