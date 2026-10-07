# 0068 — verification

Two tasks, two commits, one per task, on `impl/0068-what-blocks-what` off master `7d5ee12`.

This document states what was measured. **Nothing below describes a screen that was not looked at.**
The section "What was run against the lawful install" says exactly how far the windowed run got, and
it did not get to a map.

## The gate

Run from `wt-0068` at the story's tip. `check-*.sh` were run through **bash**; `sh` is absent from
PowerShell here and a gate that cannot run looks exactly like one that passed.

```
go build ./...                       clean
go vet ./...                         clean
gofmt -l $(git ls-files '*.go')      prints nothing
go test -count=1 -trimpath ./...     every package ok
bash scripts/check-no-game-assets.sh check-no-game-assets: clean (tree scan)
bash scripts/check-doc-budget.sh     every 0068 artifact under its ceiling; spec >= plan >= tasks
bash scripts/check-sdd-audit.sh      FAIL set: this document's own absence, and nothing else
```

`check-sdd-audit` was run before this file existed and its only FAIL was
`0068-what-blocks-what: every task in tasks.md has landed and there is no verification.md`. Its
note/warning **counts are not comparable from a worktree** — the builds check is guarded by
`[ -d builds ]` and a worktree checks out no `builds/` — so only the FAIL set is reported.

`check-doc-budget` failed once, on T1's entry at 106% of its ceiling, and was fixed by cutting a
paragraph that restated FR-4 rather than by rewrapping it.

Deletion set over the whole story:

```
git diff --diff-filter=D --name-only 7d5ee12 HEAD    (empty)
```

`pkg/sim` is untouched, and no serialized form moved — SC-7, and with it P-1:

```
git diff --name-only 7d5ee12 HEAD -- pkg/sim         (empty)
```

No pinned digest anywhere in the tree was edited. The research pin was **not** moved:
`git submodule status` reads `f56bb38…` with no leading character, the value it had at the start.

## SC-1 … SC-4 — what blocks what

`pkg/game/blocking_test.go`, internal to the package because FR-2 and FR-3 are asserted on the world
`openMapWorld` hands the front-end and that is unexported. The plane is read out of the world's own
**canonical byte form** — the single traversal the digest is taken over — so what is asserted is the
grid a player's units route on and not a second reading of the rule that built it.

| criterion | what was measured |
|---|---|
| SC-1 | the loaded table's buildings collection reports 3 slots for 2 written rows, the named entry, and the four footprint parameters of one entry (AC-1) |
| SC-2 | a 2×2 placement with every blocking bit set closes exactly its four cells on open interior ground, and the cell one column beyond the rectangle stays open (AC-2) |
| SC-3 | a 1×8 placement with every blocking bit clear opens exactly the eight cells of a blocked strip, and the strip's ninth cell — which no footprint attaches — stays closed (AC-3) |
| SC-4 | 8-connected reachability between two regions the ingest plane leaves disjoint: **absent** without the table, **present** with it (AC-4) |
| — | the same map through `mapload.StartMission`: deck open and hut cell closed with the table, the reverse without (AC-5's mission half) |

**P-2** rides on the plane derivation being reused rather than reproduced, and it is witnessed by
what did **not** move: every `pkg/game`, `pkg/mapload` and `pkg/sim` test in the tree passes
unedited, and every one of them builds its world from a table whose buildings collection resolves
nothing — a fixture table with no buildings rows, or no table at all. Had the second stage written a
single byte on that path, the pinned digests those suites carry would have moved.

**Every one of the five cases was run against the pre-fix table load and fails there** — the
collection was commented out, the suite re-run, and all five reported. That is what makes them
observations of the fix rather than of a fixture that would pass either way. Each case also asserts
its own fixture discriminates: the cell it expects closed is open without the table, and the cell it
expects opened is closed.

**FR-6 is met by changing nothing**, and this is the clause a later reader would otherwise
"fix": `pkg/game/mapload.go` still derives the viewer's `terrain.Grid.Block` with **no** table. The
brief that opened this story located the defect there. It is not there — that plane's one reader,
`terrain.BorderCell`, tests **bit 1**, and the structure stage writes **bit 0** alone, so handing it
a table would move no pixel. The cause was `LoadTable` never reading the collection, which broke the
map screen and the mission path identically.

## SC-5, SC-6 — what stands in front of what

| criterion | what was measured |
|---|---|
| SC-5 | `pkg/ui`: a recorded draw target shows a unit's frame submitted **after** the object on an earlier row, **before** the structure on a later one, and **after** the flat structure whatever the rows (AC-6, AC-7). The two units are handed over in descending row, so the ordering cannot come from the list they arrive in |
| SC-5 | `pkg/render/terrain`: the three-way merge over a fixture where flat, object, structure and entity rows all disagree; every drawable appears exactly once; the caller's entity slice is not reordered (P-3) |
| SC-6 | with no entities the merged order is `PlaneOrder`'s own, ref for ref (AC-8); with no bundle the content band is the entity sprites alone, and an art-less push produces the band a viewer with no entity at all produces (AC-9) |
| — | neither art switch reaches a unit: with both off, the band is the unit alone |
| — | two entities on one row keep their id order (DD-4's stable-sort clause) |

**AC-9 was amended during T2 and the amendment is the honest half.** As first written it said the
entity sequence was unchanged. It is not: units were drawn in ascending entity id, and they are now
drawn by row, so of two overlapping units the one in front is no longer whichever the world happened
to hold first. Same rectangles, same cull, same mirror bits — reordered. The clause now says that.

Six `pkg/ui` test files were rewritten rather than deleted, because the entity sprite pass they read
no longer exists: `overlayPass` carries rectangles only and every pass in that slice is an instrument
again. Where a test pinned a pass index, the index moved by one; where it pinned the sprite pass's
contents, it now reads the content band.

## What was cut, named

**The decoded per-class draw-layer scheme.** `REG-UNITS-061` gives it: `CUnit` registers draw layer
**2** while its corpse stage is below 2 and layer **4** otherwise, and `CAirUnit` registers layer
**3** unconditionally. None of that is implemented. This story gives every drawable **one**
row-ordered plane; the layer scheme is a second, coarser key **above** that order, and a story that
carried both would have had to carry the corpse stage into the render tier as well.

The visible consequence, stated rather than hidden: **a flier standing behind a building is drawn
behind it here**, and the original would draw it in front. On the shipped corpus that is two classes
of 34 — `Sonic Bat` and `Dragon`, the only ones carrying a non-zero `Z` — over 715 placements.

Also cut: `analysis.md` and `provenance.md`, per the speed calibration this story was briefed under.
Nothing surprising was found that `spec.md` and this file do not carry.

## What was run against the lawful install

`gameversions/en`, from this worktree.

```
go run ./cmd/againrom -assets <en> -check
  againrom: 38 map rows, 8 of 8 buttons have a mask region

go run ./cmd/classdump -databin <en>/world.res
  Buildings   66 written   9 titles   66 with parameters

go run ./cmd/terraintool structures -assets <en> -map <en>/Islands.alm
  structures: 287 drawn, 0 no class, 0 undrawable, 0 variable-size
  cells: 1599 strips, 2030 frames (431 overhang), 0 outside the map
  observations: variable-size ids [33 37], extensions 0, rectangle mismatches 2 of 66 [14 57]

go run ./cmd/mapview -assets <en> -map <en>/Islands.alm -blocked -check
  Deadly Islands 256x256 cells (65536), tile slots 52/128, blocked 30219
```

The **66-entry buildings collection is the point of the first fix**: it is what the front-end's table
now carries and what the footprint pass had nothing to resolve against before. The two developer
tools already built their own table, so their numbers are a magnitude reference and are **not**
evidence of this change; the corpus deltas the fix delivers are `TERR-STRUCT-074`'s — 391 of 1299
bridge-deck cells free before the pass and 1299 of 1299 after, and 92 of 104 bridge placements
joining two ground components the ingest leaves disjoint. **Those figures were not re-measured here.**

**The windowed run: how far it got, exactly.** The build was launched against `gameversions/en` and
its main menu opened and responded — that screen was captured and looked at. Driving it further
needs synthetic mouse input on a live desktop, one attempt at that captured an unrelated
application's window instead of the game's, and it was abandoned as unsafe and the captures deleted.
**So no map screen was reached, and nothing in this document describes terrain, a bridge, a building
or a unit as drawn.** Both defects are witnessed by the tests above and by the owner's own look,
which is what closes them.

## What this story does not claim

- That a unit **routes** across a bridge in a running mission. What is measured is the plane the
  world routes on, not a walk over it.
- That the shipped registry's rectangles agree with the definition table's. `terraintool` reports 2
  mismatches of 66 and reconciles nothing; that predates this story and is untouched by it.
- Anything about air movement, or about which of two units on one cell the original draws in front.
