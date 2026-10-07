# 0132-corpse-drops-worn — verification

## Environment

Go toolchain as pinned in `go.mod`. Unit evidence needs no game install. The drive evidence was
taken against both lawful roots, `gameversions/en` and `gameversions/ru`, through
`cmd/missionrun -mission 10`, which reads the archives under the root it is given and writes
nothing.

## The gate

```
go build ./...                     EXIT=0, no output
go vet ./...                       EXIT=0, no output
gofmt -l $(git ls-files '*.go')    EXIT=0, no output
go test -count=1 -trimpath ./...   EXIT=0, 33 packages ok, 0 failures
bash scripts/check-no-game-assets.sh   EXIT=0   check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh       EXIT=0   every row ok, chain ok
bash scripts/check-hotfix-ledger.sh    EXIT=0   19 commits since 9729459 examined, ok
bash scripts/check-sdd-audit.sh        FAIL set empty for this story
```

`check-sdd-audit` was run from the lane, where `builds/` does not exist, so its note and warning
**count** is not comparable and is not quoted. Only its FAIL set is, and the one FAIL it reported
before this file existed was this file's absence:

```
FAIL 0132-corpse-drops-worn: every task in tasks.md has landed and there is no verification.md
```

That is SC-1.

## The drive, on both roots — AC-10, SC-3

The tenth mission's two `M10_Brigands` placements are the clubmen. Their class row names two items:
a **Wood Club** in the weapon cell and **Leather Boots** in an armour cell. The loader wears them as
code `0801007` in slot 1 and code `1012028` in slot 12.

**Before the ten-slot loop landed** — the tool from T1 against the code as `0123` left it. Both
roots, identical output:

```
mission 10  scenario/10.alm  80x80  36 entities
attack 1  p0 -> u19 : FELLED it after 193 ticks, victim at -6 hp, attacker facing N
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12)
  before, carried: none
  ground (24,56): 0801007(slot 1) "Common Wood Club"
outcome undecided at tick 257
```

The boots are on the body before the blow and **not** on the ground after it. That is the defect as
reported.

**After** — the same command, same roots, felling both clubmen:

```
===== ROOT en =====
mission 10  scenario/10.alm  80x80  36 entities
attack 1  p0 -> u19 : FELLED it after 193 ticks, victim at -6 hp, attacker facing N
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12)
  before, carried: none
  ground (24,56): 0801007(slot 1) "Common Wood Club" 1012028(slot 12)
attack 2  p0 -> u20 : FELLED it after 21 ticks, victim at -1 hp, attacker facing SE
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12)
  before, carried: none
  ground (27,60): 0801007(slot 1) "Common Wood Club" 1012028(slot 12)
outcome undecided at tick 278

===== ROOT ru =====
mission 10  scenario/10.alm  80x80  36 entities
attack 1  p0 -> u19 : FELLED it after 193 ticks, victim at -6 hp, attacker facing N
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12)
  before, carried: none
  ground (24,56): 0801007(slot 1) "Common Wood Club" 1012028(slot 12)
attack 2  p0 -> u20 : FELLED it after 21 ticks, victim at -1 hp, attacker facing SE
  before, worn: 1=0801007(slot 1) "Common Wood Club" 12=1012028(slot 12)
  before, carried: none
  ground (27,60): 0801007(slot 1) "Common Wood Club" 1012028(slot 12)
outcome undecided at tick 278
```

The criterion is the **difference** between the two runs, not a reading of one: the same code
`1012028` is reported worn before the blow in both, and appears on the ground only in the second.
The two roots print the same bytes, and the cell differs from the placement cell because the
brigand walked before he was reached.

## Unit evidence — the acceptance criteria

Every row was run by `go test -count=1 -trimpath ./pkg/sim/`, green.

| AC | Test |
|---|---|
| AC-1 | `TestAKillDropsCarriedThenSlotTwoThenSlotOneThenArmourAscendingAndEmptiesTheBody` |
| AC-2 | the same test's second half — `Equipped` all zero and `Carried` empty after the kill |
| AC-3 | `TestTheTenArmourSlotsAreDroppedAscending` |
| AC-4 | `TestAnEmptyContainerLeavesTheSackListUnchanged` |
| AC-5 | `TestAKillDropsOnlySlotsOneFiveAndTwelveAsThreeCodesWithNoZero` |
| AC-6 | `TestASecondKillOnAnAlreadyDeadEntityChangesNothing` |
| AC-7 | `TestADeathOffTheBoundsLeavesAllTwelveSlotsAndTheContainerOnTheCorpse` |
| AC-8 | `TestADropOntoAStandingSackAppendsAtItsTailAndLeavesItsGoldAlone` |
| AC-9 | `TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion` |
| AC-10 | the drive above, both roots |
| AC-11 | `TestTwoFullLoadoutsFellingOnOneCellConcatenateEachInFR2Order`, both command orders |

That is SC-2.

## The properties

- **P-1** — witnessed by example, not exhaustively: AC-1's fixture holds a container and three
  slots and every code arrives once; AC-5's holds three scattered slots and exactly three codes
  arrive. No test quantifies over all reachable holdings, and none is claimed to.
- **P-2** — AC-2's half of AC-1's test: after the drop the body's container and all twelve slots
  read empty, so no code is in both places.
- **P-3** — **read, not test-witnessed.** The changed block writes `w.carried`, `w.equipment` and
  `w.sacks` and nothing else; this tier holds a combat block as state and computes it nowhere. No
  combat test changed behaviour, which is consistent with it but does not witness it.
- **P-4** — `TestASecondKillOnAnAlreadyDeadEntityChangesNothing`, whose fixture now wears armour as
  well as carrying.
- **P-5** — `TestADeathAndDropDrawsNothingFromTheGenerator` compares the generator after an advance
  with a death against the same advance without one. **Limitation:** its fixture carries a code and
  wears nothing, so the ten-slot loop is not on the path it measures. The loop appends to a slice
  and reads no generator, which is read-verified rather than measured.

## SC-4 — which lines are witnessed, by removing them

Removing the ten-slot loop from `clearFelled` and running `go test ./pkg/sim/`:

```
--- FAIL: TestTwoFullLoadoutsFellingOnOneCellConcatenateEachInFR2Order
--- FAIL: TestASecondKillOnAnAlreadyDeadEntityChangesNothing
--- FAIL: TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion
--- FAIL: TestAKillDropsCarriedThenSlotTwoThenSlotOneThenArmourAscendingAndEmptiesTheBody
--- FAIL: TestTheTenArmourSlotsAreDroppedAscending
--- FAIL: TestAKillDropsOnlySlotsOneFiveAndTwelveAsThreeCodesWithNoZero
```

Six, all of them ones that claim the widened drop, and nothing else in the package. Narrowing the
sack-worthiness guard back to the container and the two hand slots fails exactly one:

```
--- FAIL: TestTheTenArmourSlotsAreDroppedAscending
```

Both mutations were reverted with `git checkout --` and the tree was clean before the gate above was
run.

## What was found while verifying, and what it cost

`FR-6`'s same-tick tie-break, as first written, said the earlier entity in the world's own entity
order appends first. Probed directly — two kill commands naming entity 2 then entity 1, both on one
cell — the sack came back in **command** order, `[3 1 2]`. The clause was authored rather than
decoded and it was wrong, so the revision started at `spec.md` and cascaded to `plan.md`,
`provenance.md` and a third task. The contract now fixes the property that is true and reproducible
— the drops appear in the order the tick resolved the crossings — and names it as ours.

## The suppression gate, measured rather than assumed — spec divergence 2, DD-8

The layered audit found the loader's NPC gate arguing from the premise this story removes. Its
consequence was checked against the en root rather than reasoned about. Driving an attack on one of
the tenth mission's `NPC14_1` guards:

```
attack 1  p0 -> u48 : did NOT fell it after 40000 ticks, victim at 20 hp, attacker facing S
  before, worn: 6=1106107(slot 6) 7=1107115(slot 7) 8=0108018(slot 8) 9=0109020(slot 9) 10=1010024(slot 10) 12=1012028(slot 12)
  before, carried: none
  ground (65,14): none
```

Slots 1 and 2 are empty — the gate did its work at load — and six armour codes are still on him.
Under the widened drop those six reach the ground, where the original destroys the whole inventory.

**That drop was not observed.** The hero could not fell this guard in 40 000 ticks, nor in three
runs of that length, and no other unit was driven onto him. What is recorded above is his loadout
and the empty ground beside him; the consequence for his corpse follows from the rule the unit
tests witness, and is reported as derived, not seen.

## Conclusion

The contract is met. A body drops everything it was wearing, in the order the original's own
sequence puts the codes into the container, and a clubman in the tenth mission leaves his club and
his boots on both lawful roots. Two things are recorded as weaker than the rest and neither is
claimed otherwise: P-3 is read rather than measured, and P-5's fixture does not exercise the new
loop. Of the disclosed divergences in `spec.md`, the unconditional slot-1 drop and the two
unrepresented fields are unchanged by anything measured here; the suppression gate's narrowness is
now measured rather than assumed, and its consequence is derived rather than observed.
