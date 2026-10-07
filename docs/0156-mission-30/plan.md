# 0156 — plan

## Steps

**Step 1 — the compiled reference.** `pkg/mapload/script.go` binds the first
`Target_Item` slot; `pkg/sim.ScriptInstant` grows `Item uint16` and `HasItem
bool`; `pkg/sim/scriptbinary.go` carries both in the record's tail.

**Step 2 — the two arms.** `pkg/sim/script.go` gains
`ScriptInstantAddItem = 12` and `ScriptInstantTakeItem = 13`, both in
`scriptInstantSupported` and both dispatched in `runInstant`.

**Step 3 — the scenario expectation.** `pkg/game/scenario.go` accepts
`expect.carries`.

**Step 4 — the witnesses.** Unit tests in `pkg/sim` and `pkg/mapload`, one
mission-stage scenario in `scenarios/`, and the milestone census.

## Design decisions

**DD-1 — the `0x0e18` base is applied in the BINDER, not in the arm.** The
original resolves a `Target_Item` value into a packed word at **build** time and
stores that word on the compiled record; the arm loads the record's field and
hands it straight to the item factory. Putting the arithmetic in the arm would
make `pkg/sim` hold a fact about the map file's own reference encoding, which is
the map loader's business and not the simulation's. It also keeps the compiled
record readable on its own terms: the field is an item code, the same kind of
value a container element carries, and not a map-file subscript.

**DD-2 — `Item` is a `uint16`.** The original's field is a word and an authored
item is a packed `u16`; `ItemStack.Code` is already that type, so the arms
compare a code against a code with no mask at the comparison. The binder does the
one masking there is, where the addition happens.

**DD-3 — `HasItem` is a byte of its own** rather than folding absence onto a code
of zero. A zero code does name nothing, so the flag could have been folded away —
and it is a byte like its four neighbours for the reason the player flag is:
folding it would make one reference slot on this record read differently from
every other one, a cost paid at every reading to save one byte per node.

**DD-4 — the record's tail grows and the form's version rises.** `Item` at +64,
`HasItem` at +66, `scriptInstantLen` 64 → 67. Every offset before +64 is unmoved,
which is what the group pair, the player pair and the second unit reference each
did inside their own records. `formatVersion` goes to **46**, not 45: 45 is
allocated to a story running in parallel off the same master this one branched
from, and this tree carries none of its change, so there is no version-45 form
this build ever writes. A buffer declaring 45 is refused for that reason and not
because its bytes are malformed on this tree's terms — the same treatment version
40 got for the same reason.

A version-44 buffer is this one's sharp case: the header, the state block, the
counts and every check record are identical, so the **second** instant and every
one after it would be read starting three bytes into its neighbour while the
first survived intact. Nothing but the version byte separates a correct decode
from that.

**DD-5 — instant 12 reuses `appendUnits` + `foldContainer`** rather than a new
merging add. Those two are already the path every other act that puts an item in
a container takes — an authored loadout, a decode, a sack pickup, a give-all —
and the fold is the one place a container is normalised. A private add on this
arm would be the one path that could leave two elements naming one code.

**DD-6 — instant 13 walks to the first element holding the code** and does not
consult the fold. A folded container holds at most one element per code, so the
first match is the only match; walking is what the original does and it stays
correct if the invariant is ever weakened. The element is removed at a count of
1 and decremented above it, which is the original's own split: the unlink at
`count <= 1` and the one-unit detach above it.

**DD-7 — check opcode 17 is not built.** It reads the same reference on the
**check** record, which would grow that record too and arm triggers on seven
other maps. `30.alm` authors none, so it buys this story's contract nothing and
changes behaviour well outside it.

**DD-8 — the byte-form version test is renamed to carry no version number.** A
test whose name spells the live version has gone stale and been repaired after
the fact three times in this package. Re-spelling the new number is what failed;
the name is what stops carrying it.

**DD-9 — the scenario's `carries` expectation states codes, not names.** The
scenario vocabulary reaches `pkg/sim`, which holds packed codes and no item
names. A name would need the item-name table, which is a different tier and a
different story.

## Traceability

| FR | DD | Witnessed by |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-3 | AC-1, P-3, SC-1 |
| FR-2 | DD-5 | AC-1, AC-2, P-1, SC-1 |
| FR-3 | DD-6 | AC-3, AC-4, P-1, SC-1 |
| FR-4 | DD-3 | AC-6, SC-1 |
| FR-5 | DD-5, DD-6 | AC-5, SC-1 |
| FR-6 | DD-4, DD-8 | AC-7, SC-1 |
| FR-7 | DD-9 | AC-9, SC-3 |

## Success criteria

**SC-1** The Go gate is green in both repos: build, vet, gofmt, `go test
-trimpath -count=1 ./...`.

**SC-2** `pipeline/check-milestone.sh`'s census loses every `instant op 12` and
`instant op 13` row, on both roots — 7 op-12 nodes and 43 op-13 nodes per root.

**SC-3** The mission-30 scenario runs green over the lawful EN root.

**SC-4** `cmd/missionrun`'s mission 10 and mission 20 UNSUPPORTED counts are
recorded before and after, against `pipeline/milestone-baseline.txt`.
