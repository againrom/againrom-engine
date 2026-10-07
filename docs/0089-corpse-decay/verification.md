# Verification — a body decays

Environment: Windows 11, Go as pinned in `go.mod`, `-trimpath` throughout (Windows Defender
quarantines a test binary otherwise). Branch `impl/0089-corpse-decay`, forked from master `e1a8df5`,
research submodule at pin `96f0b15` with no leading character in `git submodule status`.

## Gates

Each was run on its own and branched on its own exit code, never on a grep of its output.

| Gate | Exit |
|---|---|
| `go build ./...` | 0 |
| `go vet ./...` | 0 |
| `gofmt -l $(git ls-files '*.go')` | 0, and it printed nothing |
| `go test -count=1 -trimpath ./...` | 0 |
| `bash scripts/check-no-game-assets.sh` | 0 — `clean (tree scan)` |
| `bash scripts/check-no-game-assets.sh --history` | 0 — `clean (history scan)` |
| `bash scripts/check-doc-budget.sh` | 0 |
| `bash scripts/check-sdd-audit.sh` | 0 |

```
$ git diff --diff-filter=D --name-only e1a8df5 HEAD
$
```

The deletion set is **empty**.

```
$ git log --format='%h %(trailers:key=SDD-Task,valueonly)' e1a8df5..HEAD
7fe3b96 0089-corpse-decay/T5
8ddddf1 0089-corpse-decay/T4
bd51ea1 0089-corpse-decay/T3
beaf81d 0089-corpse-decay/T2
a0c11f9 0089-corpse-decay/T1
```

Five implementation commits, five ids, none twice.

## The milestone

`cmd/missionrun`'s `TestTheTenthMissionIsDrivenToAWin` was measured on both roots at the base commit
and at this branch's head. It skips silently without an asset root, and every run below named one, so
none of the four is a skip.

```
e1a8df5, ru:  waypoint 1  u21 -> (56,21) r3 : reached (44,46), Chebyshev 25, after 272 ticks
              outcome lost at tick 272
e1a8df5, en:  waypoint 1  u21 -> (56,21) r3 : reached (44,46), Chebyshev 25, after 272 ticks
              outcome lost at tick 272
HEAD,    ru:  waypoint 1  u21 -> (56,21) r3 : reached (44,46), Chebyshev 25, after 272 ticks
              outcome lost at tick 272
HEAD,    en:  waypoint 1  u21 -> (56,21) r3 : reached (44,46), Chebyshev 25, after 272 ticks
              outcome lost at tick 272
```

**It did not move**, on either root, and the four reports are identical line for line. The test was
already red at the base commit and is red at the same place for the same reason, which is not this
story's.

It is not unmoved because the ladder is dormant. A throwaway probe, run from this worktree against
both installs and **removed before any commit**, opened the tenth mission and read what its
placements carry:

```
start 36 entities, dying times map[0:1 8:26 10:5 12:4]
after 272 ticks: held 36, alive 36, dwelling 0, torn 0
```

Thirty-five of the thirty-six placements carry a real dwell off their own definition rows, so the
column reaches the entity on both collections' arms; and **nothing is felled inside the window the
mission is decided in**. The occupancy change is therefore live and unexercised on this drive: no
body stands on anybody's route before the outcome flips, which is why two runs differing in what a
body does to the ground under it come out identical.

## Acceptance criteria

| AC | Evidence |
|---|---|
| AC-1 | `pkg/sim/decay_test.go` `TestTheConstructorPairsTheStageWithBeingNotAlive` — a stage and a dwell on a living unit are dropped; a body handed none is put at the first stage owing its own dying time with its defence untouched; a dwell at a later stage is dropped; stages 5, 6 and 255 are refused. |
| AC-2 | `TestFallingSetsTheStageHalvesTheDefenceAndStartsTheDwell` — a kill, a damage past zero and a damage to exactly zero each leave stage 1, a defence of 41 shifted to 20, and the dwell the dying time names; over dying times 8, 1, 0 and absent the dwell reads 7, 0, 0 and 0 after the tick of the fall; a second blow on a body at zero does not halve the defence twice. |
| AC-4 | `pkg/sim/downed_test.go` `TestABodyHoldsItsCellForItsDwellAndNoLonger` — eight runs: each way of leaving the living, asked whether a mover may stand on the body's cell and whether it may walk through it, inside the dwell and past it. Inside, the mover is at (1,0) and still holding its target; past it, it reaches (2,0) and (4,0) and holds none. |
| AC-6 | `TestTheWalkTakesOneHealthPerPeriodAndSaturates` — health falls by exactly one on every tick congruent to 12 modulo 32 across three periods and on no other tick, the period and the phase asserted against literals rather than the constants the pass reads; and a body at the representation floor does not move and does not come back alive. |
| AC-7 | `TestTheStageIsTheLadderOfHealthAndNeverFalls` — eight cases, at and one short of each threshold, plus a body driven from the first stage to the last, which reaches stages 1, 2, 3, 4 in that order and never falls. |
| AC-8 | `TestTheBottomOfTheLadderTakesTheBodyOutOfTheWorld` — a body at -600 stays and is at stage 4; one at -601 is gone, the route slice is level with the entity slice, and a survivor's attack order naming it has been cleared. |
| AC-9 | `TestANonGroundMoverLeavesNoBody` — a ghost, a flyer and a ground mover felled together are three bodies while the dwell is owed, and on the teardown tick the world holds the ground mover alone, at the first stage. |
| AC-11 | `pkg/data/anim_test.go` `TestAnimBoneSlotIsTheBoneBlocksOwnLength` and `TestAnimBoneSlotMovesNoBaseAndNoTotal` — the slot is the declared count and zero for both the absent sentinel and a resolved zero; the block ends inside the space the predicted total already reserved; every base and the total are unchanged. |
| AC-12 | `pkg/render/terrain/boneanim_test.go` `TestTheBoneFrameIsTheDirectionsOwnSlot` — the three stages at all eight octants at both layouts, each answer inside its own direction's slot; the eight octants at three stages give **24 distinct frames**, which is the assertion an index with no direction term fails. `TestTheBoneSelectionRefusesWhatItCannotDraw` covers the same criterion's refusals — stages 0, 1 and negative, no bone block, the absent sentinel, a sheet too short, a sheet with no frames, and an octant past the eight all answer no frame, at frame 0 unmirrored. |
| AC-14 | `pkg/game/bones_test.go` `TestABodyPastItsFallIsDrawnFromTheBoneBlock`, `TestABodyStillFallingIsDrawnFromTheDyingBlock`, `TestACorpseClassWithNoBoneBlockKeepsTheDrawingItHad` — three bodies reach stages 2, 3 and 4 through the simulation's own ladder and are pushed carrying the corpse class's art at that class's bone frames 0, 1 and 2 for the direction they died facing; a body at the first stage is pushed from the dying block; one whose corpse class has no bone block is still drawn, from the dying block it had before. |
| AC-16 | `pkg/sim/decayform_test.go` `TestADecodedWorldKeepsItsDecayState`, `TestUnmarshalRefusesTheDecayShapesNoTickCanLeave`, `TestTwoWorldsDifferingOnlyInADecayFieldHashDifferently`; `pkg/sim/binary_test.go` `TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth`, `TestMarshalledBytesArePinned`, `TestUnmarshalRefusesEveryVersionButSeventeen`, `TestThePinIsThePreStoryPinPlusTheDecayBlock`. |
| AC-17 | `pkg/sim/downed_test.go` `TestTheCorpusCrossesADecodeAtEveryTick` — the corpus is walked to quiescence with the ladder live, marshalled and decoded at every tick, and the decoded world stepped beside the original form for form and digest for digest at every later tick. |
| AC-18 | `pkg/mapload/dwell_test.go` `TestAPlacementCarriesItsOwnRowsDyingTime` and `TestAnEmptyDyingCellLeavesTheConstructorsOwn` — a units row, a humans row and an unresolved placement give three different numbers, none of them the others'; an empty cell on either collection leaves the constructor's own. |
| AC-19 | **Not run here.** It needs a lawful install and a window, and it is what `builds/0089-corpse-decay/README.md` asks the owner to watch. Recorded as an open check, not as a pass. |

P-1 is witnessed by AC-1 and AC-2, P-2 by AC-7 and AC-8, P-3 by AC-12's slot assertion made on every
answer, P-4 by AC-14's three cases read over one snapshot, P-5 by AC-17.

## The pins that moved, and what separates them from pins that moved wrongly

Every pinned digest and byte transcription in `pkg/sim`, `pkg/mapload` and `pkg/game` moved with the
version, and a digest that moved for the right reason looks exactly like one that moved for the wrong
one. Two things separate them, and neither can be satisfied by changing the code under measurement.

The **peel chain**: `strippedOfTheDecayBlock` removes the seven bytes each record grew by, puts the
version byte back, and must hash to the literal the tree carried before this story —
`0x047c5d3c0b15a2d0` for the pinned world and `0x8ac12c66c9a2fd56` for the routed one, both pasted
from the pre-story tree. It is the newest peel, so the relation, facing, plane, owner and group peels
behind it now run on its output, and every one of their literals is still a fixed point.
`pkg/mapload`'s two fixtures carry the same derivation.

The **differential**: `TestTwoWorldsDifferingOnlyInADecayFieldHashDifferently` asks nothing about
what any digest is. Two worlds differing only in a dying time — a field no tick reads until its unit
falls — must not hash alike, which is exactly what a field lost between the constructor and the
encoder fails.

## Mutation audit

Thirteen mutations of **production** code, each run against `./...` rather than the package under
change. A mutation the compiler rejected is not recorded as a kill.

| # | Mutation | Result |
|---|---|---|
| M1 | `decayBonesHP` -10 becomes -11 | **survived at first**; killed after the fix below by `TestTheStageIsTheLadderOfHealthAndNeverFalls` (`pkg/sim`) |
| M2 | `decayPhase` 12 becomes 13 | **survived at first**; killed after the fix below by `TestTheWalkTakesOneHealthPerPeriodAndSaturates` (`pkg/sim`) |
| M3 | `decayCycle` loses its doubling | killed by `TestTheWalkTakesOneHealthPerPeriodAndSaturates` (`pkg/sim`) |
| M4 | the defence halving is dropped | killed by `TestFallingSetsTheStageHalvesTheDefenceAndStartsTheDwell` (`pkg/sim`) |
| M5 | the bone frame loses its direction term — `slot*BoneSlot` becomes `slot*0`, reformulated from a form the compiler rejected | killed by `TestTheBoneFrameIsTheDirectionsOwnSlot` (`pkg/render/terrain`) **and** `TestABodyPastItsFallIsDrawnFromTheBoneBlock` (`pkg/game`) |
| M6 | a body never holds its cell | killed by `TestABodyHoldsItsCellForItsDwellAndNoLonger` and `TestTheCorpusCrossesADecodeAtEveryTick` (`pkg/sim`) |
| M7 | removal at -600 rather than below it | killed by `TestTheBottomOfTheLadderTakesTheBodyOutOfTheWorld` (`pkg/sim`) |
| M8 | `formatVersion` back to 16 | killed in `pkg/sim` **and** `pkg/mapload` — `TestAWorldHoldingAnArmedPartyRoundTrips`, `TestATableBuiltWorldRoundTripsAtTheVersionTheTreeCarries`, `TestTheDerivedPlaneIsCarriedHashedAndReadBack` |
| M9 | the non-ground pin fires on the ground domain instead | killed in `pkg/sim` **and** `pkg/game` — `TestABodyHoldsItsCellForItsDwellAndNoLonger`, `TestTheCorpusCrossesADecodeAtEveryTick`, `TestOneCommandStreamReachesOneDigestWhateverIsDrawnBetweenItsTicks` |
| M10 | the dwell never falls | killed by `TestABodyHoldsItsCellForItsDwellAndNoLonger`, `TestTheCorpusCrossesADecodeAtEveryTick`, `TestANonGroundMoverLeavesNoBody` (`pkg/sim`) |
| M11 | the spawn drops the dying time | killed by `TestAPlacementCarriesItsOwnRowsDyingTime`, `TestAnEmptyDyingCellLeavesTheConstructorsOwn`, `TestFromALMBuildsOneEntityPerUnit` (`pkg/mapload`) |
| M12 | the decoder stops refusing a stage on a living unit | killed by `TestUnmarshalRefusesTheDecayShapesNoTickCanLeave` (`pkg/sim`) |
| M13 | the drawer never reaches the bone selection | killed by `TestABodyPastItsFallIsDrawnFromTheBoneBlock` (`pkg/game`) |

**M1 and M2 survived, and both were real gaps.** Every ladder case sat well past its threshold, so a
threshold moved by one changed no answer; and the walk test computed what it expected from
`decayCycle` and `decayPhase` themselves, which is a test that cannot notice either of them moving.
The fix ships in the same commit as this file, untrailered: eight ladder cases at and one short of
each threshold, and the period and the phase asserted against literals with the constants checked
against them once. Both mutations then died.

Three of the thirteen were killed only by a package other than the one changed, which is why every
run is `./...`.

## Success criteria

- **SC-1** — every acceptance criterion above but AC-19 has evidence, and P-1 to P-5 are witnessed by
  the schedules named beside them.
- **SC-2** — the gate table above: eight gates, each green by its own exit code, and an empty
  deletion set against `e1a8df5`.
- **SC-3** — the milestone was measured on both roots before and after and did not move; the probe
  above says why, and the reason is a rule this story states rather than an unexplained agreement.

## Limitations

- AC-19 is unrun. The build is what carries it.
- The dwell is measured in this package's own tick, the shorter of the two readings the source
  allows, and the teardown fires at the dwell's end unconditionally. Both are recorded in
  `provenance.md`, the second as an open question for research rather than as a decided fact.
- A corpse class whose bone phase count is the absent sentinel has no bone block here, where the
  engine indexes outside its own. One shipped class is affected, and it is disclosed in the spec.
