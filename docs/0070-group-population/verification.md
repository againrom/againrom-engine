# Verification — the group a check counts

Run in the lane's worktree, on the toolchain `go.mod` pins. Every number below was
produced by a command in this file; the corpus measurements read two lawful
installs through the developer tools and write nothing into the tree.

## What this story does not do

**A firing win is not reachable after this story.** The campaign's first mission
still cannot be won: its chain is three triggers, and the one this story unblocks
hands the escortee to the player through an instant this build does not run, after
which that unit has to cross thirty cells to the cell the next trigger measures.
Nothing here moves it. The honest headline is the corpus one — **nine more maps
now hold a trigger that can reach the win instant** — and it is not mission 10.

## Gates

Each judged by its exit code, in a script that stops on the first nonzero.

```
go build ./...                        exit 0
go vet ./...                          exit 0
gofmt -l $(git ls-files '*.go')       no output
go test -trimpath -count=1 ./...      exit 0
scripts/check-no-game-assets.sh       exit 0
scripts/check-doc-budget.sh           exit 0
scripts/check-sdd-audit.sh            exit 0
```

Note counts are not comparable from a worktree, which carries no `builds/`; only
the FAIL set is enforced and it is empty. The deletion set of each commit against
its parent is empty (`git diff --diff-filter=D --name-only`).

## SC-1, SC-2 — the identifier and the reference (AC-1, AC-2)

`TestEveryEntityCarriesItsRecordsGroup` over a four-placement map: groups 3, 3, 0
and 4294967295 arrive as written, the zero one included.
`TestAWorldBuiltFromNoMapCarriesTheZeroGroup`: every entity of a map naming no
group is in group zero.

`TestTheGroupACheckNamesReachesTheCompiledCheck`: a node naming group 18 compiles
present/18, one naming group 0 present/0, one naming none absent/0 — so group zero
and no group are two records.
`TestAGroupParameterMovesNoPlainParameter` over the corpus's own slot shape
(const, group, unit, int): the plain values still pack to slots 0 and 1 with
nothing between them.
`TestOnlyTheFirstGroupSlotIsTaken`: a node with two group slots keeps the first.

## SC-3 — the byte form, anchored backwards (AC-5, P-3)

The entity record is 83 → 87 bytes, the check record 58 → 63, the version 10 → 11.
`TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth` re-partitions the whole
form by hand and covers it exactly, group words at 117 and 204.

Every moved digest is established against a literal older than this story rather
than pasted from a run:

| pin | pre-story literal | after |
|---|---|---|
| `pinDigest` (`pkg/sim`) | `0x9cfe358e8b5c1fae` | `0xb56c622a377738cf` |
| `rtfDigest` (`pkg/sim`, routed) | `0x6b56295a3f0a55f8` | `0x5e1a6c6e7ea13679` |
| `rlxTick1Digest` | `0x1877f13583b8654d` | `0x5bcf10d095620c72` |
| `hybTick1Digest` | `0x32a13e90447c7c81` | `0x9528b1cc98c786b8` |
| `gfDigest` (`pkg/mapload`) | `0xb5f72b1253c19040` | `0xa34f237fd5ce2305` |

The two `pkg/sim` byte pins are hand-transcribed; taking the new transcription,
removing the four bytes each record grew by and putting byte 0 back to 10
reproduces the pre-story digest **exactly** — for the plain pin and for the routed
one, whose records are not the last thing in its form.
`TestThePinIsThePreStoryPinPlusTheGroupWord` and
`TestTheRoutedPinIsThePreStoryPinPlusTheGroupWord` run that backwards on every
run. The two relaxation digests were reassembled outside the tree from the
contract's text with an FNV-1a checked against its published vectors first; the
same assembly reproduces both pre-story literals from the pre-story description,
which is what says the new pair differ only where they are meant to. `gfDigest` is
established backwards only, through **two** anchors: lifting the group word gives
`0xb5f72b1253c19040`, and lifting the eight combat fields from that gives
`0x9761ea070c9b922c` — a literal older than the story before this one.

0067's two guards were generalised rather than replaced: `zeroedEightDigest` and
the new `strippedOfTheGroupWord` lift the group word and restore the version, so
`preStoryDigest`, `zeroedPreStoryDigest`, `zeroedPinDigest` and `pinDigest` in
`pkg/mapload` all still hold at their original values.

`TestTwoWorldsDifferingOnlyInAGroupHashDifferently` is the differential that does
not care what any digest is. `TestADeadUnitKeepsItsGroupThroughTheForm` and
`TestTheCheckRecordCarriesItsGroupAndItsPresence` close AC-5. The refusals hold:
version 10 among the 255 refused bytes, a truncated widened record, and a group
presence byte outside `{0,1}`.

## SC-4, SC-5, SC-6, SC-7 — the arm (AC-3, AC-4, AC-6, AC-7)

`TestTheArmCountsTheGroupsLivingMembers` asserts the transition rather than a
value: 3 → 1 → 0 as three members of one group are felled, 2 for the group beside
it, 1 for group zero, 0 for a group no entity carries, and 2 → 1 when a member is
downed at exactly zero health.

`TestACheckNamingNoGroupWritesNoRegister` is **SC-4**'s second half and **AC-4**:
the register keeps a preset 77 that no count of that world could produce, while
its trigger is evaluated and latches — this arm is implemented, so nothing about
it is inert. SC-4's first half was witnessed at T1, where the arm was still
reported and its readers still inert; the same case now stands on opcode 14.

`TestTheSackArmIsStillLoudBesideTheGroupArm`: opcode 14 is still the only arm
reported, only its reader is inert, the mission stays undecided over 64 ticks with
an authored zero that would otherwise win it on the first pass, and the group
check beside it measures and fires.

`TestTheFirstMissionsWinChainReachesAWin` drives the chain's shape end to end: the
mission stays undecided with the guard group standing and with one of two felled,
and reaches `OutcomeWon` with counters 1/0 when the last one dies.

## SC-8 — P-1 and P-2

`go test -trimpath -count=1 ./...` is green, which runs `internal/archtest`'s
import check and its determinism-wall source scan over `pkg/sim` — the arm reads
entities and nothing else, and no clock, float or `math/rand` name entered the
package. **P-2**: the arm answers over an empty world, over a group no entity
carries and over a check holding no reference, in the cases above; no opcode
panics and no script is refused for holding an arm this build lacks.

## The corpus, both roots

```
go run ./cmd/restool cat <root>/scenario.res <n>.alm > <scratch>/<n>.alm
go run ./cmd/almtool script <scratch>/<n>.alm
```

Over the 28 campaign maps, inert triggers summed:

| root | before | after |
|---|---|---|
| en | 233 | **174** |
| ru | 232 | **173** |
| maps holding a live win-carrying trigger, en | 11 / 28 | **20 / 28** |
| maps holding a live win-carrying trigger, ru | 11 / 28 | **20 / 28** |

Both totals are exactly what the story's own counterfactual predicted before a
line was written, which is the strongest thing the corpus can say about it.

`10.alm` is byte-identical across the roots (`2d983ccbf249c5336ebb7ccc41fc405c`)
and both report the same thing: check arm 1 has left the unimplemented list, and
inert triggers fall from **7 of 12 to 1 of 12** — the survivor being map position
5, which reads the sack arm.

## Limitations

- The count excludes the dead on a reading the arm's own body does not fix. It is
  the reading every shipped use of the arm forces; the rival is recorded, and the
  behaviour is pinned through zero so that a correction is one predicate and a
  failing test.
- Membership ignores the owning player, which the original keys its runtime groups
  on. No placed unit's owner is decoded anywhere in this tree, so the distinction
  is not expressible here and no map was measured for it.
- T2 also edited `pkg/sim/groupform_test.go`, which T1 added and `tasks.md` did
  not list under T2: the criterion's first half became its second there rather
  than being deleted and rewritten elsewhere.
