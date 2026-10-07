# 0123-corpse-loot — verification

Branch `0123-corpse-loot`, off master `67d2e57`. Two trailered commits, `79b7ce4` (T1) and
`0054cf3` (T2), plus this stage. All evidence below is from the committed tree with a clean
working directory, and every gate was re-run in the lane seat rather than taken on an
executor's report.

## Gates

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l` over our own Go files | prints nothing |
| `go test -trimpath -count=1 ./...` | every package ok, exit 0 |
| `scripts/check-no-game-assets.sh` | `clean (tree scan)`, exit 0 |
| `scripts/check-doc-budget.sh` | every 0123 artifact under its ceiling; `plan <= 1.2 x spec` and `tasks <= 1.2 x plan` both ok |
| `scripts/check-sdd-audit.sh` | FAIL set empty for 0123 after this stage; see the note below |
| `git diff --diff-filter=D --name-only 67d2e57..HEAD` | empty — nothing deleted |
| a `Co-Authored-By` trailer scan over `67d2e57..HEAD` | none on any commit |

Trailer bijection: `746af7a` (docs stage) carries none, `79b7ce4` carries
`0123-corpse-loot/T1`, `0054cf3` carries `0123-corpse-loot/T2`. Two tasks, two trailers, no
id twice, and both trailered commits touch files outside `docs/`.

Note on the audit: it failed twice on the way here and both are recorded rather than hidden.
It refused a range in T2's entry — a range is a summary, not a witness — so the five FR ids
are now spelled out; and it refused the story for having every task landed with no
`verification.md`, which this file answers. **The note/warning count is meaningless from a
worktree** (`builds/` is untracked, so a lane emits none); only the FAIL set is read here.

## SC-1 — the local gate

Run on the committed tree, working directory clean. All rows in the table above.

## SC-2 — reverting the drop turns AC-1 red

The drop's body was replaced with a discard, leaving the call site, the empty-container test
and the bounds guard in place:

    --- FAIL: TestAKillDropsTheContainerInOrderAndEmptiesIt (0.00s)
        corpseloot_test.go:37: got 0 sack(s), want 1: []

Restored, and the working tree confirmed clean afterwards.

## SC-3 — reverting the compaction turns AC-6 red, panic included

The container slice was taken back out of `remove`'s compaction loop:

    --- FAIL: TestRemoveKeepsTheContainerListAlignedWithTheEntityList (0.00s)
        removecontainer_test.go:28: 1 entit(y/ies) but 2 container(s), want the same length

That check is a fatal one, so it stops before the panic it exists to prevent. With the
length check alone suppressed and the revert still in place:

    removecontainer_test.go:37: Carried(2) = [] after removing entity 1, want [0x101]
    panic: runtime error: index out of range [1] with length 1
    againrom/pkg/sim.(*World).Stock(...)

So master answers another entity's container for a survivor and then crashes on the
whole-world stock read. That read has two callers outside tests — `pkg/mapload/start.go`
at the two mission-start rebuilds — so the crash was reachable from the game and not only
from a test. Restored; tree clean.

## SC-4 — the form and the digest did not move

The form version constant is 26, the value it held before this story, asserted as a literal
by `TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion`. Every existing form, digest and
replay test passes unchanged, and this story adds no field to any record (P-1). AC-9 is that
test: a world in which a death has occurred encodes, decodes to an equal world by its
digest, and re-marshals byte-identically.

## SC-5 — a death off the map

`TestADeathOffTheWorldsBoundsDropsNothingAndLeavesTheContainer`: an entity outside the
world's bounds is killed; the sack list stays empty and its container still reads `[3 4]`.
Refused, not clamped and not folded onto a nearby cell (FR-5).

## SC-6 — the random-source doc block

The block on `pkg/sim`'s generator no longer asserts that no research claim describes the
original's; it names the original's generator and both of its published constants by claim
id, and states our generator as a disclosed divergence. No code in that file changed — the
generator, its step constant and its bounded draw are byte-identical, which is what makes
this a documentation correction and not a behaviour change. `internal/archtest`'s source
scan over `pkg/sim` still passes, so nothing banned entered with it (P-2).

## Acceptance criteria

| id | Witness |
|---|---|
| AC-1 | `TestAKillDropsTheContainerInOrderAndEmptiesIt` — one sack at (4,6) holding two codes in order, the entity carrying nothing. Reverted red under SC-2. |
| AC-2 | `TestAnEmptyContainerLeavesTheSackListUnchanged` |
| AC-3 | `TestTwoEntitiesFellingOnOneCellInOneAdvanceLeaveOneSackInDeathOrder` — one sack at (5,5) holding three codes in death order |
| AC-4 | `TestADropOntoAStandingSackAppendsAtItsTailAndLeavesItsGoldAlone` — gold 25 untouched, items appended at the tail, list still ascending by (Y, X) |
| AC-5 | `TestASecondKillOnAnAlreadyDeadEntityChangesNothing` — compared by world digest against a quiet tick |
| AC-6 | `TestRemoveKeepsTheContainerListAlignedWithTheEntityList`. Reverted red under SC-3. |
| AC-7 | `TestADeathAndDropDrawsNothingFromTheGenerator` — the generator state after a dying advance equals the state after an identical quiet one |
| AC-8 | `TestAUnitStandingOnTheCorpseCellPicksUpTheDropWithTakeSack` — the existing transfer primitive, from a second entity on the corpse's cell |
| AC-9 | `TestAWorldWithADeathRoundTripsAndKeepsTheFormVersion` |
| P-1 | AC-9's version assertion, plus: the diff adds no field to any struct the byte form carries |
| P-2 | AC-7, plus `internal/archtest`'s source scan over `pkg/sim` |
| P-3 | AC-1 and AC-4 drop whole containers with no size, owner or distance test anywhere on the path; neither the drop nor the merge takes such an argument |

## What this story did not build, and what it costs

Three of the death routine's five clauses are declined, each because it needs per-entity
state the byte form does not carry — and therefore a form version, which was not allocated
to this lane:

- **the gold roll** — three treasure columns and a type gate per entity. `pkg/data`'s
  `UnitDef` stops at slot 37 and would need slots 38 to 40 added; `pkg/sim`'s `Entity` would
  need the three columns and the gate. The draw itself is already available: the world owns
  a generator whose state is serialised and hashed, and its bounded draw is inclusive at
  both ends, which is the shape the roll wants.
- **the two equipment slots**, and the weapon column that gates the second — there is no
  equipment model in this tier at all.
- **the suppression flag** on a template name — no entity here carries a name, or a mark
  standing in for one.

Nothing was hardcoded in their place: no literal treasure number, no literal chance and no
name list appears anywhere in this story's diff.

## Where the two halves meet

An ordinary unit's starting weapon comes from its `Data.bin` Units row's trailing
`EquipItem` strings, which this tree reads and discards. Folding them into what a placement
carries is another lane's work, in `pkg/mapload`, and this story deliberately touched no
file there. **Until that lands, a killed clubman drops nothing, because its container is
empty** — this story's half is complete and unobservable in the game on its own. It needs
no change when the other half lands: the drop moves whatever the container holds, whether
that is nothing today or three things afterwards.

## What the owner does

Open a mission with a party. Select an enemy unit and press **K**, the debug kill key; a
sack appears on the cell it fell on and stays there — a sack in this engine does not tick
and does not expire. Move a party unit onto that cell and press **G**; the sack is gone and
its codes are in that unit's inventory. Today that shows nothing for an ordinary unit, for
the reason above; it is visible immediately on any unit given a container.

## Divergences on record

- The drop site is the felling routine and not one death routine, because a death is split
  across two places here. Behaviourally identical: the guard fires once, on the tick of
  death.
- The merge is a binary search into an ordered list rather than a per-cell registry. Not
  observable through anything this build exposes.
- A corpse outside the world's bounds drops nothing. The original has no such state, so
  this refusal is ours; it is chosen over a clamp because a world this package could
  marshal and then not read back is the worse outcome.
- Every sack this story creates draws the sack sheet's first frame, unchanged from 0111.
  Which frame a sack should draw is an open research question, and nothing here invents a
  selector.
