# Verification — 0112-container

Branch `0112-container`, cut from `b23915d` and merged up to master `84aa355`
after 0113 landed. Six task commits, one per entry, each carrying its own
`SDD-Task:` trailer and no other trailer, plus one UNTRAILERED correction —
see the defect section below.

The merge conflicted in `pkg/sim/world_test.go` alone, where 0113 added
`SetCombat` and this story added `TakeSack` to the same two pinned lists.
Resolved as the union with both rationales kept, and `Stock` added to
`worldMethods` by the correction.

## The gate

Run from a clean tree at the branch head, with **no game install visible to the
tests**.

```
$ go build ./...                              (silent)
$ go vet ./...                                (silent)
$ gofmt -l $(git ls-files '*.go')             (silent)
$ go test -trimpath -count=1 ./...            every package ok
$ bash scripts/check-no-game-assets.sh        check-no-game-assets: clean (tree scan)
$ bash scripts/check-doc-budget.sh            every artifact ok; both chain ratios ok
$ bash scripts/check-sdd-audit.sh             no FAIL for 0112-container
$ git diff --diff-filter=D --name-only b23915d..HEAD
                                              (empty — nothing was deleted)
$ git log --format='%h %(trailers:key=Co-Authored-By)' b23915d..HEAD
                                              (no co-author trailer on any commit)
```

39 files changed, 30 new test functions.

## Success criteria

- **SC-1** `go test -trimpath -count=1 ./...` is green tree-wide with no game
  install present, above.
- **SC-2** A well-formed version-25 stream is refused and the message names the
  version: `pkg/sim/binary_test.go`'s `TestThePreviousVersionFormIsRefused`, now
  built at version 25 rather than 24.
- **SC-3** The corpus walk is below.
- **SC-4** Witnessed by reverting, from this seat rather than on report. Changing
  `pkg/ui/inventory.go:299` back to `drawInventoryCell(img, box, nil)`:

```
--- FAIL: TestRenderInventory/a_pack_picture_composes_differently_from_the_same_subject_without_one_(0112_AC-9)
--- FAIL: TestRenderInventory/a_nil_pack_entry_is_a_drawn_absence,_every_other_pack_cell_unchanged_(0112_AC-9)
FAIL    againrom/pkg/ui
```

  The line was then restored and `git status --porcelain` is empty.
- **SC-5** The trailer set is a bijection onto T1..T6, checked by commit hash:

```
1bea9c1  SDD-Task: 0112-container/T6
45483ad  SDD-Task: 0112-container/T5
cb2177c  SDD-Task: 0112-container/T4
f23e4c8  SDD-Task: 0112-container/T3
c28b437  SDD-Task: 0112-container/T2
98a106a  SDD-Task: 0112-container/T1
```

## The corpus, measured here (SC-3)

A throwaway tool under `cmd/` walked both lawful roots' campaign archives
through `pkg/vfs` and `pkg/formats/alm`, and was deleted afterwards — no game
data and no tool reading one is committed.

```
root=<en> maps=28 type8=176 ground=133 stock=43 stockElements=25 elemF04nonzero=5 maxStock=20(scenario/120.alm)
  stock.Owner -> Unit.UnitID: exactly-one=43 none=0 ambiguous=0
root=<ru> maps=28 type8=176 ground=133 stock=43 stockElements=25 elemF04nonzero=5 maxStock=20(scenario/120.alm)
  stock.Owner -> Unit.UnitID: exactly-one=43 none=0 ambiguous=0
```

Identical on both roots, and identical to the figures `ITEM-OWNED-028` publishes
from the other side. The join is a bijection over 43 cases with nothing
unmatched and nothing ambiguous — evidence, not a decode; the claim's own
Medium on the id space stands, and FR-12 is what keeps a wrong join cheap. The
eleven loose maps beside the archive carry four type-8 records between them, all
ground, no stock.

## Acceptance

| Id | Witness |
|---|---|
| AC-1 | `pkg/sim/carry_test.go` — a world built with codes answers them in the order given, zeroes dropped. |
| AC-2 | `pkg/sim/carry_test.go` (the constructor's drop) and `pkg/sim/binary_test.go`'s `TestUnmarshalRefusesACarriedCodeOfZeroNamingTheEntity` (the decoder's refusal). |
| AC-3 | `pkg/sim/binary_test.go`'s `TestCarriedCodesAndPursesRoundTripByteIdentically`. |
| AC-4 | `pkg/sim/binary_test.go`'s two hash tests, one per new field. |
| AC-5 | `TestThePreviousVersionFormIsRefused`, and the 256-value sweep in `TestUnmarshalRefusesEveryVersionButTheCurrentOne`. |
| AC-6 | `pkg/sim/carry_test.go` — the transfer moves every code in sack order and all the gold, and the sack is gone from `Sacks()`. |
| AC-7 | `pkg/sim/carry_test.go` — unknown entity, no sack at the cell, and an owner past the roster, each refused with the world unchanged. |
| AC-8 | `pkg/mapload/loot_test.go` — one stock record reaches its unit's entity in element order and places no sack at that record's cell. |
| AC-9 | `pkg/ui/inventory_test.go`, and the revert above. |
| AC-10 | `pkg/game/inventory_test.go` — one icon per code in carried order, the remainder past the array undrawn, unread addresses reported in carried order. |
| AC-11 | `pkg/game/world_test.go` — the key takes the sack under the subject entity, is inert on a cell with no sack, and is inert when the mission assembled no party. |

## Derived properties

| Id | Witness |
|---|---|
| P-1 | The container is indexed by entity, not by group or owner: `World.carried` is parallel to `entities` and `Carried` takes an entity id. |
| P-2 | `carryFault` tests a code against zero and reads no field of it. `pkg/sim` imports nothing, so no class or index accessor is reachable there. |
| P-3 | `pkg/sim/carry_test.go` — mutating what `Carried` returns, and mutating the slice a caller passed the constructor, both leave the world alone. `Entities()` still documents a pointer-free value type, and `Entity` gained no field. |
| P-4 | Every constructor still funnels through `newWorld`; `NewStockedWorld` is a sixth name, not a second body. |
| P-5 | `pkg/ui/inventory_test.go` — the zero subject and a subject with a nil pack entry both compose. |
| P-6 | `pkg/ui` imports no archive reader and no `pkg/data`; `internal/archtest` is what holds that, and it is green. |

## What the owner can see

Mission **10** is the cheapest demonstration: its map authors four ground sacks
and the nearest one sits **four cells** from an authored start cell — sack
(20,65), start (17,66), on both roots. Mission 20 is next at five.

Select the party member, walk him onto the sack's cell, press **G**, then press
**I**. The item's icon is in the pack area under the twelve equipment cells.
`G` is new and was unbound before this story; `I` is 0110's.

## The defect found at the seat, and the class it belongs to

**FR-11 was met on `FromALMWith` and not on the path the game uses.** Both
mission-start rebuilds in `pkg/mapload/start.go` called `sim.NewLootWorld`,
which names no containers, so every mission opened with the map's stock
dropped. Measured through `mapload` on the shipped mission 10, before the fix,
identical on both roots:

```
FromALM                entities=35 carriers=1 codes=3 sacks=4  entity 2 at (36,51) owner=2 carries [3590 3590 3590]
StartMission           entities=36 carriers=0 codes=0 sacks=4
StartMissionScripted   entities=36 carriers=0 codes=0 sacks=4
```

and after:

```
FromALM                entities=35 carriers=1 codes=3 sacks=4  entity 2 at (36,51) owner=2 carries [3590 3590 3590]
StartMission           entities=36 carriers=1 codes=3 sacks=4  entity 2 at (36,51) owner=2 carries [3590 3590 3590]
StartMissionScripted   entities=36 carriers=1 codes=3 sacks=4  entity 2 at (36,51) owner=2 carries [3590 3590 3590]
```

`3590` is `0x0e06`, three of them — the three elements mission 10's one stock
record authors for the person at (36,51). The sack list was already four on all
three paths, because 0111 had already fixed the sack half of this.

**IT IS THE SECOND INSTANCE OF ONE CLASS, IN THE SAME TWO FUNCTIONS.** 0103 put
the map's ground sacks in a world; both rebuilds dropped them and it shipped for
eight stories, silent because nothing DREW a sack, so "no sack anywhere" was
indistinguishable from "this map authors no loot". This one was silent for
exactly the same reason: nothing shows a stocked actor, so "nobody carries
anything" reads like "this map stocks nobody". A rebuild carries only what it
names, and the warning saying so, in capitals, sits directly above one of the
two lines.

The fix is `sim.World.Stock()`, the analogue of `Sacks()`: one call that reads
every non-empty container back, and is the exact inverse of the `stock` argument
`NewStockedWorld` takes, so a rebuild cannot hold a subset of what it was
rebuilt from. `pkg/mapload/start_test.go`'s
`TestAStartCarriesTheMapsStockThroughEveryRebuild` is the fence, beside 0111's
own, over five start shapes with the plain load as the oracle rather than a
literal.

**The purses are deliberately not named in either rebuild**, and that is an
argument rather than an oversight: their only writer is `TakeSack`, nothing
calls it between the load and the rebuild, so base's purses are all zero by
construction. The test compares them anyway — that assertion passes trivially
today and is the one thing here that would catch a FOURTH instance, the day a
story authors gold at load.

Witnessed by reverting, one call site at a time, and the two reverts fail
DIFFERENT case sets, which is what shows the test discriminates the two
rebuilds rather than one of them twice:

```
StartMission -> NewLootWorld:
  a start with a party ...................................... FAIL
  a scripted start with a party and no script ............... FAIL
  a scripted start with a party and a script ................ FAIL

StartMissionScripted -> NewLootWorld:
  a scripted start with no party and no script .............. FAIL
  a scripted start with a party and no script ............... FAIL
  a scripted start with a party and a script ................ FAIL
```

"A start with no party" fails under neither, and is asked anyway: it returns the
loaded world untouched, so what the other four measure is that they agree with
it.

The correction commit carries NO trailer. It repairs T4's own requirement rather
than adding planned work, so a task entry for it could only restate `spec.md`
and `plan.md` — 0113's precedent.

## Premises that expired here

`pkg/data/hero.go`'s `Speed()` doc block omits the overload penalty and gives
the reason: *"This tree has no inventory, so nothing carries anything and the
gate is never satisfied."* **That reason is now false** — an actor can carry
things, and the map's own stock puts items on 43 placed actors across the
campaign. The term still cannot be written, because it needs a load and the load
needs a per-item weight no claim in the pin establishes for anything but a
weapon. The story that decodes that column owns both: the load field and the
penalty. Nothing was changed in `hero.go` here; this paragraph is the record.

`pkg/ui/inventory.go`'s geometry block said the pack area was "a fixed cell
count this project chose rather than derived from a container this build does
not have". Repaired in T5: the build has a container now, and the cell count is
the window's limit rather than the container's.

## Not shipped

No owner-review artifact. A story's deliverable is the build; the reasoning is
in this folder and in the merge commit.
