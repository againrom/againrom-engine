# Verification — the player can order an attack

Run from `wt-0075`, against the Go toolchain `go.mod` names.

**The pin moved under this story and the docs were reconciled to it.** The three
task commits were authored on `53d7679`; master then bumped the submodule to
`20921e2` at the 0074 boundary and this branch was rebased onto that. The gate
below is re-run at the rebased head on `20921e2`, and `analysis.md` records what
the bump discharged — every clause the orchestrator had to hand over as a
seat correction is now a cited row.

## The gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -count=1 -trimpath ./...
EXIT=0
ok packages: 30, 0 failures, gofmt printed nothing

$ bash scripts/check-no-game-assets.sh   ; echo $?
0
$ bash scripts/check-doc-budget.sh       ; echo $?
0
$ bash scripts/check-sdd-audit.sh        ; echo $?
0
```

The exit code is taken from the command and not from a pipe: `cmd | grep …` would
report grep's.

`go test` is run with `-trimpath`: Windows Defender quarantines one untrimmed test
binary on this machine.

The three commits and their trailers, one per task and no more:

```
c3ed2e3  0075: the readout states whether an attack is armed         SDD-Task: 0075-attack-order/T3
e9d3d12  0075: the attack seam, the armed mode and the press ...     SDD-Task: 0075-attack-order/T2
413a0f1  0075: an attack order closes the distance                   SDD-Task: 0075-attack-order/T1
```

Nothing was deleted by the story:

```
$ git diff --diff-filter=D --name-only b86f234 HEAD
(no output)
```

## What witnesses what

39 test cases were added or moved, all passing. The mapping below is per
criterion; each names the test that runs it.

**SC-1 — arming (FR-1; AC-1, AC-1a, AC-15).**
`TestArmingIsGatedOnOwnershipAndNotOnClass` in `pkg/ui/attack_test.go`, six
sub-cases: a selection with a present unit arms with no participant established;
an empty selection and one of a single corpse arm nothing; a second key press
lowers it; **the primary present id's owner decides** — the same two ids arm or
refuse depending only on which of them carries the participant's slot; and
**class, name and art do not enter**, asked over three different names against
one owner. That last case is the one the retracted capability clause would have
broken. `AC-15` is `TestTheArmIsSpentByOnePress`'s last sub-case: a tap while the
mode is up leaves it up and the following press orders the units the selection
then holds.

**SC-2 — one press, two possible orders (FR-2, FR-3; AC-2, AC-3, AC-6).**
`TestAnArmedPressMakesOneOfTwoOrders`: over a unit it emits one attack per
selected present unit in ascending id and **no move at all**; over empty ground it
emits *byte-for-byte the same orders an unarmed press emits*, compared against a
second App driven without the key; an unarmed press over a unit emits a move and
never an attack. `TestTheVictimIsTheOrdinaryHitTest`: two rectangles on one point
yield the lower id (the two units are given in descending id, so slice position
could not agree), and a corpse under the point is not a victim — the press falls
through to the move arm.

**SC-3 — the arm is spent (FR-2; AC-4, AC-5).** `TestTheArmIsSpentByOnePress`,
four sub-cases: a press that attacked, one that moved, one resolving outside the
map extent, and one made while the primary button was still down. The last two
issue nothing at all and still spend it.

**SC-4 — the seam (FR-4, FR-5; AC-7).**
`TestAnAttackOrderBecomesACommandInTheQueueTheOrdersUse` in
`pkg/game/attack_test.go`: four calls become exactly four `KindAttack` commands
on the same pending queue the orders and blows use, the world's digest is
unchanged across all four, and **every attacker is marked commanded** — measured
against the blow seam's own test one file over, which asserts the opposite for
the same field. `TestAnAttackOrderReachesTheWorldOnlyThroughAnAdvance` drains it
and finds the victim on the attacker.

**SC-5 — the approach (FR-6; AC-8, AC-9, AC-10).** `pkg/sim/pursuit_test.go`:
`TestAnAttackerWalksToItsVictim` (sets out on the ordering advance itself, lands a
blow, and is **in reach** when it does, holding no destination afterwards);
`TestAnAdjacentVictimIsNotWalkedTo` (six advances, not one step, and it strikes);
`TestAVictimThatMovesIsFollowed` (the destination equals the victim's cell **as
each advance began**, over twelve advances and more than one distinct cell — which
is what separates a re-aim from a destination written once);
`TestLosingTheVictimEndsTheWalk`; `TestAMoveOrderStillEndsAnApproach`, which is
0064 FR-2's surviving half asked of a state that could not exist before.

**SC-6 — no version bump (FR-6, P-3; AC-11).** `TestAnApproachRoundTrips` catches
a world holding a victim, a cycle, a destination and a stored route at once,
asserts the encoded version byte is **12 by number**, and round-trips it to equal
bytes and an equal hash.

**SC-7 — the readout (FR-7, FR-1; AC-12, AC-13).**
`pkg/ui/readout_attack_test.go`: both positions are values and they differ; the
shipped layout carries the row with a label; and the subject follows the key that
arms and the press that spends it. `TestLeavingTheMapScreenDisarms` leaves the map
screen with Esc and re-enters it through the picker: the fresh viewer is not
armed.

**SC-8 — nothing else moved (FR-8, P-1, P-2, P-4).** The whole suite is green
above. `pkg/ui`'s import set is unchanged and the two seam payloads are builtins;
the flow's own field-count guard in `flow_test.go` was extended by exactly one
name, which is what makes a seventh field a failure rather than a surprise.

**SC-9 — the fan-out (FR-9; AC-14).** `TestTheFanOutIsUnbounded`: 300 selected
units, 300 orders, ascending id, none dropped.

## The runnable build, measured against the lawful install

`builds/0075-attack-order/` holds the binary and a README with the exact
invocation. The binary's own self-check runs against the preserved EN install:

```
$ AGAINROM_ASSETS=…/gameversions/en ./againrom.exe -check
againrom: 38 map rows, 8 of 8 buttons have a mask region
EXIT=0
```

The window itself cannot be driven from here, so the end-to-end claim was
measured **headlessly on mission 10's own map** with a throwaway probe (not
committed) that starts the mission exactly as `cmd/missionrun` does and issues one
`KindAttack` — the very command the seam produces — then advances. Results, EN
root:

| ordered | onto | Chebyshev at order | closed to reach | outcome |
|---|---|---|---|---|
| 6 squirrel | 7 squirrel | 10 | yes | **down at tick 182** |
| 11 bee | 12 bee | ~12 | tick 125 | **down at tick 147** |
| 3 ghost | 11 bee | ~31 | yes | **down at tick 875** |
| 35 hero | 0 party unit | 12 | tick 352 | no wound in 4000 ticks |
| 6 squirrel | 11 bee | 26 | never | stopped 3 cells short |

**So a player-issued attack order does close the distance and does kill**, on the
first-milestone map, against the shipped numbers.

Two of those rows are honest limits and neither is this story's:

- **The hero and the party's map-placed units do no damage at all.** The probe
  read their combat block off the running world: `dmg = 0 + U[0,0]`, `toHit = 0`.
  Research measured the same thing from the other side — a bare-handed hero at the
  default stat line is refused by the clamp against **every** class on this map,
  and mission 1 is not winnable bare-handed. The blow path is correct; the
  arithmetic is the game's, and equipment is what moves it.
- **A ground attacker ordered onto a flyer can stop short.** Bees and ghosts carry
  a movement domain that stands on ground a walking unit cannot cross, so the
  approach closes as far as the terrain allows. It is `plan.md` R-2 exactly: while
  it cannot close, it buys one far search per tick. A flyer ordered onto a flyer
  reaches (row 3).

## What was NOT verified, and why

That the window shows any of it. No frame was composed and no pixel compared;
every front-end assertion above is over `App.step` and the seams, which is where
the behaviour is. The one judgement left is whether the two-click mode reads as
usable, and that is the README's manual criterion.

That the divergence above 253 selected units matters in play. FR-9 states it and
`TestTheFanOutIsUnbounded` witnesses our side of it; nothing here measures the
engine's.

That the arming gate refuses anything **in this build**. It cannot: the far side
pushes no local participant, so the comparison has nothing to compare and the
control is open. What is verified is that the comparison is *there* and *bites* —
`TestArmingIsGatedOnOwnershipAndNotOnClass` pushes a participant and watches the
same selection arm and refuse on the owner alone.
