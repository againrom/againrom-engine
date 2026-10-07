# Verification — 0122-trigger-opcodes

Every result below was produced in the lane seat on the committed tree, after the last task
landed. Where a task's own report made a claim, the claim was re-run here rather than recorded.

## The gate

```
go build ./...                              clean
go vet ./...                                clean
gofmt -l $(git ls-files '*.go')             prints nothing
go test -trimpath -count=1 ./...            EXIT=0, every package ok
scripts/check-no-game-assets.sh             EXIT=0
scripts/check-doc-budget.sh                 EXIT=0
git diff --diff-filter=D --name-only 67d2e57..HEAD    empty
```

No game is installed on the path these tests ran against and no test opens a map file, which is
**AC-7**. `check-sdd-audit.sh`'s note and warning **count** is meaningless from a worktree —
`builds/` is untracked, so it is absent here — and only the FAIL set is enforced. Two FAILs were
present before this file existed and both were mine: a missing `verification.md`, and `plan.md`
naming no `FR` id, which was corrected by threading the ids through the design decisions.

## Trailers

```
3012004  T5   SDD-Task: 0122-trigger-opcodes/T5   Co-Authored-By: (none)
62d65bb  T4   SDD-Task: 0122-trigger-opcodes/T4   Co-Authored-By: (none)
bbdbd26  T3   SDD-Task: 0122-trigger-opcodes/T3   Co-Authored-By: (none)
1495216  T2   SDD-Task: 0122-trigger-opcodes/T2   Co-Authored-By: (none)
326665e  --   stages 1-3, docs only, no trailer    Co-Authored-By: (none)
2eecc9d  T1   SDD-Task: 0122-trigger-opcodes/T1   Co-Authored-By: (none)
```

Five task entries, five trailered commits, a bijection. The trailer key and the absence of any
`Co-Authored-By` were read out of the log by commit hash, not taken from a report.

## The witness pass — every arm reverted, and the compile guarded

**SC-5** is the only evidence that makes the node counts mean anything, so each arm's body was
neutered in turn and the suite re-run. The first attempt at two of the five reported *no*
failures, which was wrong: the surgery had broken the build, and a package that will not compile
prints no `--- FAIL` line at all. The pass was redone with `go build ./pkg/sim/` as a gate before
each measurement. That is worth recording because a silent "nothing went red" is exactly what a
faked witness looks like.

```
BASELINE                                             (no failures)
check 15 body neutered      5 tests red, nothing else
check 8  body neutered      3 tests red, nothing else
check 10 body neutered      5 tests red, nothing else
instant 10 body neutered    5 tests red, nothing else
opcodes 11+13 out of scriptCheckSupported   3 tests red, nothing else
RESTORED                                             (no failures)
```

Two negative tests around instant 10 — that an out-of-range slot is a no-op, and that the arm is
bare beside its own cell — stayed green under their revert, and legitimately: "nothing changed"
is also what an absent arm produces. They discriminate jointly with the five positive tests and
not alone.

## The arms, against their acceptance criteria

**AC-1**, check 8: a player with three living entities and one felled writes 3; a player owning
nothing living writes 0; a check naming no player writes no register and records the new
missing-player silence, and its reader is inert.

**AC-2**, check 15: the least distance from the authored cell over the player's living entities,
the player's dead ignored, `0xff` when the player has nothing living. The seed and the absence
are two different outcomes and are witnessed separately —
`TestTheNearestDistanceSeedAndTheAbsentPlayerAreDistinguished` is the whole of **R-3**. The arm
goes through `scriptDistance`/`scriptByte`, which is **P-2**: the seam now has five callers and is
still the only place this file measures a distance.

**AC-3**, check 10: the register takes the low two bits and drops every higher bit; a check
naming fewer than two players writes nothing; a slot the matrix does not hold reads zero.

**AC-4**, instant 10: the mirror cell is untouched, bits 2..7 survive, an addend of 5 on a cell
of 4 gives 9 rather than 5, the sum truncates in the byte, and a **locked pair is overwritten**.
That last is the one a careful implementer would get wrong by symmetry with `turnHostile`, which
declines a locked pair; the arm being reconstructed has no test before either store. **P-3** holds:
the relation has exactly two runtime writers and both take their offset from `relationIndex`.

**AC-5**, the dead arms: a trigger whose condition names an opcode-11 or opcode-13 check is not
in `InertTriggers()`, neither opcode is in `Unsupported()`, and the register such a check names
holds whatever it held before the pass. The suite distinguishes them from a genuinely
unimplemented arm by testing check opcode 9 alongside, which stays undecoded.

**AC-6**, the form: a check carrying both player references round-trips byte-identically; a
version-26 form is refused with a message naming both 26 and 31. `pinDigest` moved
`0xc428573dacfd4a7f` → `0xede6d696093ba13e`, and every recomputed digest in this story was taken
from its own test's failure output rather than computed by hand.

**P-1** holds by construction: four check arms entering `scriptCheckSupported` is what stops
their readers being inert, and nothing else in this story touches inertness. **P-4** holds — the
architecture scan over `pkg/sim` is part of the suite that passed.

## What the plan got wrong, and it was the file lists

**T2's blast radius was badly undercounted.** The entry named five files; the version bump forced
six more, because every hand-pinned digest in the tree is taken over a whole byte form and byte 0
moved: `relaxation_test.go`, `release_test.go`, `commanded_test.go`, `routeform_test.go`,
`fromalm_test.go` and `gridform_test.go`. The tree's own prior version-bump comments say exactly
this ("no prefix survives"), so the omission was mine and not a surprise of the code.

**T3 hit a sentinel.** Four pre-existing tests used check opcode **8** as their stand-in for "a
real arm this build does not evaluate". Implementing 8 made three of them vacuous and broke one
outright. They were moved to opcode 9, which `spec.md` records as still undecoded. The tree has
done this before — a commit in its history moves the same sentinel from 14 to 8 — so a story that
implements an arm should expect to move the sentinel off it. T4 and T5 checked for the same
pattern on their own opcodes and found none.

**A `gofmt` trap worth naming.** `gofmt -l $(git ls-files '*.go')` sees **tracked** files only, so
running it before `git add` on a brand-new test file silently skips that file. One task caught a
misformatted new file only by re-running the gate after staging.

## What did not change behaviour

T1 is comments only and its diff contains no non-comment line. Neither correction alters what the
code does: the distance metric was already Chebyshev and the group count already excluded the
dead. What changed is that both blocks had claimed an open question the ledger has since closed,
and `scriptDistance`'s block still records the one clause that genuinely remains open — the
mask-before-subtract ordering, which this build reproduces by masking the result instead.

The dead arms change no shipped map. No map on either root authors check opcode 11 or 13, so the
difference between reproducing a dead arm and failing to implement one is reachable here only by
a customised map — which is the customisation seam it exists for, and is why no corpus figure is
claimed for it (**R-2**).

## What is open

**D-1 is live and it is the one thing this story cannot settle by itself.** Byte-form version
**31** was chosen in the lane because a lane has no channel back to the seat that allocates
version numbers; 27 through 30 were skipped, three of them known to be out to sibling lanes and
one unaccounted for. The number lives in exactly one place, `pkg/sim/binary.go`'s `formatVersion`,
and its doc block says so. If the seat renumbers, the recomputed digests come back from the tests
themselves.

**SC-1, SC-2, SC-3, SC-4 and SC-6** are each answered above. **SC-5** is the witness pass.
