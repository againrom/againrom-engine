# verification — 0128 a person wears his whole row

Every number below was produced by re-running the tool or the gate from the lane seat, on the four
task commits with a clean tree. Nothing here is taken from a task report.

## Gate

Run from the worktree at `907ced6`, tree clean but for the untracked story folder:

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      nothing
go test -count=1 -trimpath ./...     every package ok, no FAIL
bash scripts/check-no-game-assets.sh clean (tree scan)
bash scripts/check-doc-budget.sh     every artifact of this story inside its ceiling
bash scripts/check-sdd-audit.sh      no FAIL naming this story
```

`check-sdd-audit`'s note and warning **count** is not comparable from a worktree — `builds/` is
untracked, so a lane emits none of those until its own build stage runs. Only the FAIL set is
enforced and only it is reported here.

Four commits, four trailers, a bijection with `tasks.md`:

```
907ced6  SDD-Task: 0128-wear-the-whole-row/T4
41d2898  SDD-Task: 0128-wear-the-whole-row/T3
9a58180  SDD-Task: 0128-wear-the-whole-row/T2
c22aa6f  SDD-Task: 0128-wear-the-whole-row/T1
```

No commit carries a `Co-Authored-By` trailer. `git diff --diff-filter=D --name-only master..HEAD` is
**empty**: this story deletes no file.

## The measurement (AC-6, FR-11)

`go run ./cmd/wearcheck -assets <root>`, run against both preserved roots. **The two outputs are
byte-identical**, so every figure below is an agreement across two installs and not one reading.

```
215 Humans rows scanned, 166 name at least one item

class    used  worn  carried  dropped
weapon   156   156   0        0
shield    46    46   0        0
armour   707   706   0        1

slot histogram: 1:156  2:46  3:0  4:36  5:39  6:110  7:148  8:81  9:64  10:105  11:0  12:123
destination collisions (two cells of one row resolving to the same slot): 0

anomalies, in full (1):
  row=143 cell=5 armour name="Leather Gauntless" dropped
    data: armour "Leather Gauntless": no Armors entry named "Soft Gauntless"
```

**908 of 909 equipment cells resolve and are worn — 99.89%.** The single failure is one authored
name, a typo for a shipped row, and it costs the divergence DD-1 declares: this build drops it where
the original keeps a nameless object in the backpack. Nothing else in the whole shipped corpus is
carried rather than worn, so DD-1's cost is exactly one cell in 909 and DD-2's is zero.

Two independent checks fall out of the same run. The **slot histogram** reproduces, from this tree's
own code and without consulting the census, exactly the set of armour slots research reports for the
shipped rows — 4, 5, 6, 7, 8, 9, 10 and 12, with nothing at 3 or 11 — plus slot 1 for the weapon and
slot 2 for the shield. And **zero destination collisions**: no row names two pieces that want one
slot, so no piece is silently overwritten by another and the 908 are 908 distinct places.

## What the measurement caught, and it was not a test

The first run of this tool reported **902**, not 908. Six cells failed, all the same authored string
carrying a double space, whose residue came out with the space still in it. Reading the one
unidentified call in the implied-shape routine as a left trim takes those six and nothing else. That
is **DD-7**, and it is recorded as an inference rather than a decoded fact: the instruction is the
image's and the identification is ours. **It is a question for research** — the routine at the
re-attachment calls one no-argument method on the subject immediately before prepending, and what
that method is is not published. If it is not a left trim, six shipped cells resolve some other way
and this build is wrong about them.

## What a placed person now wears

`go run ./cmd/paneldump -assets <root> -mission 10`, the first map-placed unit of mission 10:

```
placed id=0    box=227x139  rows=7  name="Human ClubMan"
         | Human ClubMan
         | HP 15/15
         | CELL 24, 54  SPEED 16
         | WORN Club, Soft Boots
         | DMG 2-3  HIT 3
         | DEF 6  ABS 0
         | SWING 7/4
```

Seven rows where the story brief found six. Before this story that unit held its club and nothing
else; it now wears the boots its row has always named. **`ABS 0` is the disclosure holding**: the
boots are worn and contribute nothing, because what a worn piece is worth is not decoded and the
additive block is still its structural zero (P-2). A follow-up story folds the numbers.

## Acceptance

| | Witnessed by |
|---|---|
| AC-1, AC-2 | `pkg/mapload/wear_test.go` — a synthetic row filling all ten cells, and one whose cell 2 and cell 8 pieces land in the slots their own columns name rather than their cell indices |
| AC-3 | the same file, both halves: a two-handed weapon displaced into the container by the shield, and a one-handed one worn beside it |
| AC-4 | the same file, a Slot of 0 and a Slot of 13, each carried and no slot written |
| AC-5 | the same file, an armour cell naming no row leaving the rest of the placement untouched |
| AC-6 | the run above, both roots |
| AC-7 | the same file, an `NPC`-templated row with slots 1 and 2 empty and every armour slot filled |
| AC-8 | `pkg/ui/worn_test.go`, both halves — the second by pruning the new row out of the authored layout and comparing the composed box, not against a copied golden string |
| P-1 | `git diff master..HEAD -- pkg/sim` is empty |
| P-3, P-4, P-5 | `pkg/data/itemparse_test.go` and `pkg/data/wear_test.go`; `pkg/mapload/wear_test.go` with each collection absent in turn |

No test in this repository reads a game install; every fixture above is built in the test from names
invented for it.

## The byte form did not move, and 37 is unspent

The story was allocated `formatVersion` **37**. It is **not taken and 36 is still live.** The
twelve-slot worn array, the container, their encoding and their place in the digest all already
existed; what was missing was entirely above the determinism wall, in a loader that resolved one cell
of ten. No field was added to `pkg/sim`, no encoder or decoder changed, and a version bump with no
grammar change behind it is a number spent on nothing. **A world's digest does change** for any map
whose placements now wear more — that is the content moving, which is what the digest is for, and it
is not a format change.

## Three landed tests changed, each adjudicated

Each was asserting a rule this story replaces, and each was caught by an executor refusing to edit it
on its own authority.

* `pkg/data/weapon_test.go` asserted that a table entry which is not a whole word is not a match —
  the authored parse (DD-6). It now asserts what the original does, and is renamed.
* `pkg/game/hero_test.go` carried a fixture whose shape table omitted the very word its weapon name
  leads with, which only the authored parse tolerated. The fixture now models an install.
* `pkg/mapload/human_test.go` asserted the humans-band "first cell that resolves wins" search, down
  to its own name — the workaround FR-1 removes. Rewritten and renamed to the positional rule.
* `pkg/ui/sheet_test.go`'s field census had the count in its NAME. It is renamed to carry no number
  and the count literal beside the map is gone, so the census is stated in one place.

DD-6's safety is a measurement and not an argument: no row name of the shipped weapon, shield or
armour collections contains any entry of either prefix table, so no shipped name can reach the
positional rebuild. That was checked before the change was authorised, by dumping all three
collections and testing every name against both tables.

## Left for later, deliberately

What a worn piece contributes to defence, absorption, protection or resistance. A placed person's
health maximum. What a corpse drops. The running window's own fill of the panel row, which needs a
name source the per-frame entity seam does not carry (DD-3) — the story's instrument fills it, and
the window states nothing rather than something wrong.
