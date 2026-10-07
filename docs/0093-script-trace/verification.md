# 0093 — verification

Toolchain `go 1.26.1`, `go test -trimpath` (Windows Defender quarantines an untrimmed test
binary). Both lawful installs were read; `<EN>` and `<RU>` below stand for the two roots, which
are never written into the tree.

## SC-1 / AC-1 / P-1 / P-3 — observing a world moves no byte of it

Two worlds built identically from one script that both fires a trigger and counts a loss; one
advanced with `StepTraced` at every tick, the other with `Step`; the byte form and the digest
compared **at every one of 256 ticks**, not at the end.

```
$ go test -trimpath -count=1 -v -run 'TestObservingAWorldMovesNoByteOfIt' ./pkg/sim/
=== RUN   TestObservingAWorldMovesNoByteOfIt
--- PASS: TestObservingAWorldMovesNoByteOfIt (0.00s)
ok  	againrom/pkg/sim	0.485s
```

The run crosses sixteen full cycles, so it contains several evaluation passes and several outcome
reports; the test fails if the run decides nothing, so a vacuous pass is refused.

**The same equality against the pre-story binary, on a real mission.** The tenth mission driven
with the readout **off**, this story's binary against `fc7ba83`'s, on both roots — identical but
for the FR-8 status word, which is the one output line this story deliberately changes:

```
$ diff prestory-en.txt plain-en.txt
2c2
< waypoint 1  u21 -> (56,21) r3 : reached (43,46), Chebyshev 25, after 272 ticks
---
> waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (43,46), Chebyshev 25, after 272 ticks
```

Identical on RU. The tick the mission is decided on (272) and the cell the escorted unit ends on
(43,46) are unmoved, which is the simulation itself saying nothing changed.

## SC-2 / P-1 — no canonical state, no format version

`formatVersion` stays **18**. This story took none: it adds no persisted field, so there is
nothing for a version to be about.

```
$ git diff fc7ba83..HEAD -- pkg/sim/binary.go pkg/sim/nostate_test.go pkg/sim/hash.go | wc -l
0
$ grep -n '^const formatVersion' pkg/sim/binary.go
176:const formatVersion = 18
```

The version literal is **not** the evidence. The field-set pin is — it compares `World`, `Entity`,
`Bounds` and `rng` against a hand-written table for **exact equality**, so a field added anywhere
in the canonical world fails it whether or not any byte moves:

```
$ go test -trimpath -count=1 -v -run TestTheCanonicalWorldsFieldSetsArePinned ./pkg/sim/
--- PASS: TestTheCanonicalWorldsFieldSetsArePinned (0.00s)
```

Together with the round-trip and digest equality above, and the untouched encoder, that is the
whole of P-1. P-3 follows from the shape rather than from a test: a trace is a local in
`StepTraced`, returned by value, and nothing in `pkg/sim` holds a reference to one.

## SC-3 — the synthetic witnesses

All on worlds built in test code, no install present.

| Id | What it witnesses |
|---|---|
| **AC-2** | `TestATickWithNoScriptPhaseObservesNothing` — a tick outside both script phases reports empty, and carries the tick it ran on. |
| **AC-3** | `TestAFiringNamesItsTriggerItsValuesAndItsInstants` — trigger subscript 0 and latch 12 recorded separately; pair 0 records the `alive` check's 1 against the constant 1; pair 1 records the **measured** Chebyshev 4 against the constant 9; the unused third slot is not marked used; the implemented `LOSE` arm is marked supported and the arm this build lacks is not. |
| **AC-3a** | `TestAPairIsRecordedAtTheValueItCompared` — see the mutations below. |
| **AC-4** | `TestALossCountedByACheckBelongsToNoTrigger` — a script with **no trigger at all** records a loss, naming the check and the entity, with zero firings and zero silences. |
| **AC-5** | `TestTheThreeSilencesAreNamedAndTheTwoByDesignOnesAreNot` — one check of each silent kind is recorded with its own reason and its own register; the constant check and the loss-counting check whose unit is alive are **not** recorded. |
| **AC-6** | `TestARegisterNamesTheCheckThatWritesIt` — an owned register yields its check and subscript, an unowned one yields "none", a nil script claims nothing. |
| **AC-7a** | `TestTheReadoutNamesEachEventInTheMapsOwnTerms` — a pass in which a firing's `LOSE` and a check's own loss both land takes the counter to 2, and the readout prints `REPORT won=0 lost=2 -> undecided`. |

### AC-3a, by mutation rather than by assertion

The pin puts **one register on both sides of one instant inside one pass**: register 5 reads 0
when the first trigger's condition tests it, that trigger's own instant sets it to 1, and the
second trigger's condition reads 1. Two honest values for one register in one pass, which no
reading taken at any single later moment can produce. Both cheaper placements were built and both
died:

```
mutation: record the firing after that trigger's own instants
  --- FAIL: TestAPairIsRecordedAtTheValueItCompared
      the first trigger read register 5 as 1; it held 0 when the condition was tested

mutation: collect the firings and record them all after the pass
  --- FAIL: TestAPairIsRecordedAtTheValueItCompared
      the first trigger read register 5 as 1; it held 0 when the condition was tested

restored: ok  againrom/pkg/sim  0.385s
```

An earlier, weaker version of this pin survived the first mutation. It was replaced rather than
recorded as passing.

## SC-4 / AC-7 — the tenth mission, both roots

Identical output on EN and RU, line for line:

```
$ AGAINROM_ASSETS=<EN> ./missionrun.exe -mission 10 -trace -waypoint u21:56:21:3 -waypoint p0:66:16:3
mission 10  scenario/10.alm  80x80  36 entities
script  16 checks, 27 instants, 12 triggers
script  UNSUPPORTED check 15: check op 14
script  UNSUPPORTED instant 0: instant op 2          (... 21 more, ops 2, 6, 20, 28)
script  INERT triggers [4] — they read a register nothing writes
tick 6  check 15 check op 14(NO UNIT) WROTE NOTHING into r15: this build does not evaluate that arm (its readers are inert)
tick 6  trigger 0 FIRED (map latch 0, once)
          pair 0  r0[check 0 const(0)] == r0[check 0 const(0)]  ->  0 == 0  = true
          slot 0  instant 0 instant op 2  [THIS BUILD DOES NOT RUN IT]
          slot 1  instant 20 instant op 6  [THIS BUILD DOES NOT RUN IT]
          slot 2  instant 21 instant op 6  [THIS BUILD DOES NOT RUN IT]
tick 6  counters won=0 lost=0
tick 262  check 12 vip(u21) COUNTED A LOSS: its unit is not alive (-1 hp)
tick 262  counters won=0 lost=1
tick 271  REPORT won=0 lost=1 -> lost
waypoint 1  u21 -> (56,21) r3 : was STOPPED BY THE WORLD DECIDING, short of (43,46), Chebyshev 25, after 272 ticks
outcome lost at tick 272
```

**The answer. No trigger fires.** The arm that decides the tenth mission is **check 12, a VIP
check on u21** — the map's own "protect this unit" objective on the very unit the drive escorts.
It counts the loss from the check loop, so it belongs to no trigger and would have been invisible
to a record of firings alone. The reporter turns `lost=1` into the outcome nine ticks later at
271, and the world reads 272 because a step ends by advancing the clock.

`-1 hp` is **killed outright**, not downed. The passes bracket the death between ticks 247 and
262.

A second drive, the same but without u21's own waypoint:

```
waypoint 1  p0 -> (66,16) r3 : reached (63,17), Chebyshev 3, after 1287 ticks
outcome undecided at tick 1351
```

u21 never dies and the mission is never decided, so the death follows from **walking** u21 and not
from the mission running.

## SC-5 / SC-6 / AC-8 — the walk stops honestly

SC-5 is the diff under SC-1. AC-8 is pinned on a synthetic world whose script loses on its first
pass while a unit walks 63 cells:

```
$ go test -trimpath -count=1 -run TestAWalkEnded ./cmd/missionrun/
ok  	againrom/cmd/missionrun	0.559s

reverted (reach answering true on the decision):
  --- FAIL: TestAWalkEndedByTheDecisionDoesNotClaimTheRadius
      reach reported the radius met after 16 ticks; the unit is at [16 16] and the point is (63,63)
```

`TestTheReadoutIsSilentWhenItIsOff` pins that the flag off prints nothing at all, and
`TestAnUnnamedUnitIsNotPrintedAsAMapNumber` pins that an entity the map's script never named is
not dressed up as one.

## SC-7 — the gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...
(clean; every package ok)
$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ bash scripts/check-doc-budget.sh ; echo $?
0093-script-trace: plan <= 1.2 x spec            5706 <=   9174 bytes  ok
0093-script-trace: tasks <= 1.2 x plan           2340 <=   6847 bytes  ok
0
$ git diff --diff-filter=D --name-only fc7ba83 HEAD
(empty)
```

`check-sdd-audit.sh`'s FAIL set is empty for this story. Its note/warning **count** is not
reported: the check is guarded on a top-level `builds/` directory, so a lane that has built its
own deliverable sees every other story's gap appear at once, and the number says nothing about
this story.

## P-2 — no counter moves unobserved

Not a test but a reading, stated as a limit rather than as a proof. A grep for `w.won` and
`w.lost` over `pkg/sim`'s non-test files finds exactly **three increment sites**, and all three
are recorded: `runInstant`'s two arms, reached only from a firing the trace already carries
(`script.go:735`, `script.go:737`), and `runCheck`'s VIP arm (`script.go:602`). Every other
mention reads them — the reporter, the accessor, the encoder — except the decoder, which sets
both from stored bytes when a world is built and advances nothing. That is a source reading;
nothing mechanical would catch a fourth writer added later.

## Limits

- **Only firings are recorded.** A trigger evaluated and found false leaves no trace, so the
  readout answers "which arm fired" and not "why this one did not". That is the contract's own
  boundary, and it is the question the milestone asked.
- **The names are ours.** `groupcount`, `LOSE` and the rest are this tree's labels; every line
  also carries the subscript, so a wrong name is checkable against the code and never changes a
  measured value.
- **What killed u21 is not answered here.** The trace ends at "its unit is not alive"; which
  blow, and from whom, is the next story's question.
