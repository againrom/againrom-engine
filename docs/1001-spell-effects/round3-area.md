# 1001-spell-effects — round 3, R3-A2 fix (occupant slots)

Branch `story/1001-r3-area`, based on the second review's pushed sha `70a7b74`, research pin
`744214fe1f4b461cbd2d6766390b850ab15779ab`. Owns one finding, R3-A2 (`round3-review.md`). Scope is
`pkg/sim/celleffect.go`, `pkg/sim/route.go` and test files created here; three other lanes own the
remaining findings and fold into `spec.md`, `closure.md` and `docs/DIVERGENCES.md` at the landing.

## What was wrong

Fix round 2 (`round2-area.md`, item D4) made `applyAreaCells` read at most one ground-or-ghost
actor and one flyer per cell, keyed by movement domain, on `TERR-CELLREC-146`'s decoded record: the
cell's `+0x4` slot holds one actor of domain 1 or 2, `+0x8` one of domain 3, and both adds return
without storing when the slot is already taken. Round 1 had walked up to three covering entities in
id order, counting corpses toward the cap, which let three decayed bodies shield a living actor from
a Fire Ball; round 2 closed that.

Round 2 did not maintain the invariant the cap depends on. `TERR-FOOTPRINT-147` (High) establishes
that ROM1 registers an actor's cell-record slot once per covered cell of its own `n x n` footprint,
through `R0050`, called from the sub-cell step and the arrival routine
(`MOVE-STEP-010`/`MOVE-REFRESH-012`), and that **a refusal from any covered cell aborts the whole
registration** — so ROM1's own movement never lets a second actor's footprint register over a cell
another actor's already covers. `MOVE-AREA-038` (High) reads the movement search's own six
plane-reading routines exhaustively and confirms none of them reads a cell-record occupant slot: the
exclusivity is enforced once, at registration, and nothing downstream re-checks it.

This build's occupancy plane does not enforce that. `pkg/sim/route.go`'s `routeScratch.occ` is
anchor-only: `occupy` seeds one cell per mover at its own `(X, Y)` and `moved` adds and removes one
cell per step. `TokenSize` appears nowhere in `route.go` or `step.go`. `entityCoversCell`
(`celleffect.go`), by contrast, is footprint-aware. So an ordinary 1x1 mover can path onto a cell
inside a larger actor's footprint — reachable on shipped data: 15 installed `Units` rows carry
`TokenSize > 1` and 147 such actors stand across the 28 EN campaign maps, and mission 140 reaches it
through the ordinary mover. A second route needs no large unit at all: `castBookAt`'s Teleport arm
(`spell.go`, rule id 26) tests only `w.terrainOpen`, with no occupancy test, so a caster can land
directly on a cell a different 1x1 actor already stands on.

Reading the decoded per-layer cap in either state made the actor that lost the collision immune to
every blast, ring and cloud on the shared cell. Measured by the review at `70a7b74`:

```
2x2 actor 1 at (6,6); 1x1 actor 2 at (7,7) — inside its footprint
cellSlotOccupants(7,7) = ground index 0, air index -1
Fire Ball at (7,7): 2x2 HP 100 -> -4, 1x1 HP 100 -> 100
```

## Which option, and why the other was rejected

Two fixes were named in the brief and are not equivalent.

**Option 1 — widen the read.** Make `applyAreaCells` walk every covering actor of a layer, not the
first one found, and disclose that this build does not maintain the footprint invariant the decoded
cap depends on. Cheap, restores the correct player-visible outcome, stays inside `celleffect.go`.

**Option 2 — maintain the invariant.** Make occupancy footprint-aware, so a 1x1 actor cannot stand
inside a 2x2 actor's footprint and Teleport refuses an occupied cell, and the one-slot read is then
correct as decoded.

Option 2 was evaluated and rejected for this lane, on file scope rather than on preference. Making
`route.go`'s `occupy`/`moved` footprint-aware only fixes the query side; the plane is also
maintained incrementally as movers advance one cell per tick
(`scratch.moved(w, e.Domain.layer(), from, cell{x: e.X, y: e.Y})`, `pkg/sim/step.go:890`), and that
call site would need to add or remove every cell of a mover's own footprint, not one — which needs
`step.go`'s move-execution loop to pass more than a `from`/`to` cell pair. Left unfixed, a large
mover's stale footprint cells would linger occupied or under-counted across ticks, which is a worse,
silent defect than the one being fixed. Separately, `castBookAt`'s Teleport arm
(`pkg/sim/spell.go:975`) would need an occupancy test added beside its terrain test. Both are named
"not yours, at all" in the brief (`step.go` — orders lane; `spell.go` — weapon lane), and the brief's
own instruction on reaching one of those files is to stop and report rather than edit it.

Option 1 was built. It does not require touching `step.go` or `spell.go`: it makes the one function
that produced the player-visible symptom — `applyAreaCells`, entirely inside `celleffect.go` —
correct regardless of how an overlap arose, without needing to enumerate and close every path that
can produce one. `DIV-054` and `DIV-055` (below) record the resulting divergence and name Option 2
as the closing story.

## What changed

`pkg/sim/celleffect.go`:

- `cellSlotOccupants(x, y int32) (ground, air int)` — returned the first covering actor index per
  layer, `-1` for none — is replaced by `cellLayerOccupants(x, y int32) (ground, air []int)`, every
  covering actor per layer, in the world's stable entity-id order. Membership is unchanged:
  `cellRecordHolds` still requires the actor be on-map and either alive or a body within its dwell
  (`Decay == DecayFallen && Dwell > 0`).
- `skipOccupiedGround` (Wall of Earth's occupied-cell refusal) reads `len(ground) == 0` instead of
  `ground < 0`. The two are equivalent for every input under both the old and the new occupant
  function — "is there at least one" — so this is a rename with no behaviour change.
- `applyAreaCells` walks every entity in each of the two groups (`[...][]int{ground, air}`) instead
  of testing a single index per layer. The per-mode read count (cloud reads the ground group alone;
  blast and ring read both) is unchanged. The Fire Ball footprint-division arm, the `seen` dedup for
  every other rule, `markSpellEffect`, skill training and the Control Spirit follow-attack are all
  unchanged in what they do to one entity — the change is which entities the loop reaches.

`pkg/sim/route.go` is untouched: Option 1 needed no change there.

## The one existing test this necessarily changed

`TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots` (`spelleffect1001_test.go`) places two
living 1x1 ground actors, entity 4 and entity 5, at the identical cell `(20, 20)` and asserted, under
round 2's code, that only entity 4 is hit — "a second ground-layer actor took N damage; slot +0x4
holds one". That assertion pins exactly the R3-A2 defect: two ground actors sharing a cell, one
immune. The brief's own framing of this test as narrow-but-correct does not survive fixing the
Teleport-reachable route, which the finding states explicitly needs no large unit — a Teleport onto
another 1x1 actor's own cell is the identical configuration this fixture hand-builds. Confirmed by
running the suite with the fix applied and nothing else changed: it is the only failure in the
repository, at exactly that assertion —

```
--- FAIL: TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots/blast_reads_+0x4,_+0x8_and_+0xc
    spelleffect1001_test.go:653: a second ground-layer actor 5 took 20 damage; slot +0x4 holds one
--- FAIL: TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots/a_cloud_pulse_reads_+0x4_alone
    spelleffect1001_test.go:653: a second ground-layer actor 5 took 20 damage; slot +0x4 holds one
```

The fix was not narrowed to avoid this — a narrowing that special-cased "distinct anchors" against
"identical anchors" would leave the Teleport-onto-a-1x1 route exactly as broken as before, which the
finding names as reachable with no large unit. The single assertion was corrected instead: entity 5
is now expected hit, exactly like entity 4, with a comment recording why and a revert instruction.
This is a deviation from "leave it" in the brief, made deliberately and disclosed here rather than
silently; the doc comment above the test was also corrected, since it stated the old cap as fact.
Nothing else in the test — the three decayed bodies read as no-slot-at-all, the flyer's own
ground/air split — changed or needed to.

## Witnesses

Every line quoted below was produced by reverting `cellLayerOccupants` to the old single-int
`cellSlotOccupants` and `applyAreaCells`'s loop to `slots[:read]` over that pair (`git stash push
--keep-index -- pkg/sim/celleffect.go`), then rerunning — not by reading the assertions. Reproduce
with `go test -trimpath -count=1 -run <name> ./pkg/sim/`.

| Test | Reverted | Failure |
|---|---|---|
| `TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots` (existing, entity-5 assertion corrected) | `cellLayerOccupants` / `applyAreaCells` group walk | `spelleffect1001_test.go:670: a second ground-layer actor 5 on entity 4's cell was not hit` |
| `TestAreaEffectHitsEveryGroundActorOnAFootprintContestedCell` (new; blast, ring, cloud) | same | `areaoverlap1001_test.go:55: the 1x1 actor inside the footprint (entity 2, at the same cell (7,7)) took no damage: HP=100` |
| `TestFireballDamagesATargetStandingInsideAnotherActorsFootprint` (new; reproduces the review's own measured numbers) | same | `areaoverlap1001_test.go:84: the 1x1 target inside the footprint (entity 2) took no Fire Ball damage: HP=100` |
| `TestAreaEffectHitsBothActorsAfterATeleportLandsOnAnOccupiedCell` (new; drives the real, unedited `castBookAt` Teleport arm) | same | `areaoverlap1001_test.go:119: the actor Teleport landed on (entity 2) took no damage: HP=100` |

All four were run together against the reverted file in one pass; all four redden, all four pass
restored. The Teleport test calls `castBookAt` through the ordinary `KindCastAt` command path —
nothing in `spell.go` is edited or stubbed — so it is also a witness that the Teleport route the
finding names ("no large unit needed") is real on the unedited production code, not merely asserted.

## Gate

```
go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')
go test -trimpath -count=1 ./...
bash scripts/check-no-game-assets.sh
```

All clean; `gofmt` printed nothing; `check-no-game-assets: clean (tree scan)`.

## Shipped-content instruments, before and after

Both roots, `AGAINROM_ASSETS` pointed at the read-only preserved installs; nothing was written into
either. "Before" is `pkg/sim/celleffect.go` reverted to `70a7b74`'s round-2 state (`git stash`), all
other files (including this fix's new/changed tests, which do not affect the built binary) held
constant; "after" is the fix applied.

**`bash scripts/campaign-sweep.sh --census`, 28 EN maps, 2000-tick budget:** byte-identical before
and after — `diff` over the full census (mission, tick, outcome, unsupported/reached counts, alive,
fallen, and the digest `hash` column) produced no output. Same result on the 28 RU maps. The sweep
issues no orders — it is an unattended AI/script-only drive — and evidently does not put two ground
actors on one cell within its 2000-tick budget on either root, so the fix has no effect on it. This
is consistent with the finding's own reachability evidence, which needed a player order (mission 140)
or a driven Teleport, neither of which an unattended sweep produces.

**`go run ./cmd/spelleffectcheck -mission 91`, both roots:** identical to `round2-area.md`'s recorded
figures — `area-hp friend=1000/1000/990 enemy=1000/1000/996`, `outside-wall=1000/1000`,
`wall-route ground=true/false/true ghost=true/false/true air=true/true/true` — and identical to each
other. Mission 91's controlled witness places one actor per position and does not exercise a
footprint or Teleport overlap, so it was expected unchanged and is.

Both instruments' unchanged output is expected, not a check the fix did nothing: R3-A2 changes
outcome only in the specific state — two ground actors covering one cell when an area effect lands
there — that the finding's own reachability evidence (mission 140 under a player order; Teleport
under a driven cast) already shows the unattended, orderless instruments above do not produce.

## Proposed `spec.md` sentences

Ready to paste. Line numbers are the pushed branch's current `spec.md` (`efd89b8`), section "Area
effects".

Replace, at `spec.md:232-235`:

> A covered cell holds at most one ground or spirit actor and at most one flyer, keyed by movement
> domain, and a third slot for a structure that this build's entities never occupy. Blast and staged
> modes read all three in that order; a cloud pulse reads the first alone. A body occupies its slot
> while it lies where it fell and none once it has begun to decay.

with:

> A covered cell's occupants are keyed by movement domain into a ground-and-spirit group and a flyer
> group, plus a third, structure group that this build's entities never occupy. The decoded record
> caps each group at one actor, but that cap depends on an invariant (`TERR-FOOTPRINT-147`) this
> build's occupancy plane does not maintain: movement occupancy is anchor-only and a Teleport landing
> tests only terrain, so two ground actors can cover one cell. Application reads every actor of a
> group and not the first one found (`DIV-054`, `DIV-055`). Blast and staged modes read all three
> groups in that order; a cloud pulse reads the ground-and-spirit group alone. A body occupies its
> group while it lies where it fell and none once it has begun to decay.

Replace, at `spec.md:241-242` (inside the cloud paragraph):

> Each pulse reads the covered cell's first occupant slot alone, so it reaches one ground or spirit
> actor per cell and no flyer, and it applies to friends as well as hostiles.

with:

> Each pulse reads the covered cell's ground-and-spirit group alone, so it reaches every ground or
> spirit actor covering the cell and no flyer, and it applies to friends as well as hostiles.

Replace, at `spec.md:270` (Ring mode paragraph):

> Each accepted cell walks its occupant slots in slot order.

with:

> Each accepted cell walks its occupant groups in group order, applying every actor a group holds.

Replace, at `spec.md:284`:

> Wall of Earth refuses a cell whose first occupant slot is filled — any cell of a ground or spirit
> actor's footprint, including a body lying where it fell — harms no unit, and marks every accepted
> cell in the ordinary passability plane.

with:

> Wall of Earth refuses a cell whose ground-and-spirit group is non-empty — any cell of a ground or
> spirit actor's footprint, including a body lying where it fell — harms no unit, and marks every
> accepted cell in the ordinary passability plane.

## Proposed `docs/DIVERGENCES.md` rows

Allocated ids `DIV-054` and `DIV-055`. Ready to paste at the ledger's end, in column order (`ID |
Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason |
Revisit condition | Status`).

| DIV-054 | simulation / movement occupancy is anchor-only | — | `TERR-FOOTPRINT-147` (High): an actor's cell-record slot is registered once per covered cell of its own `n x n` footprint on arrival (`R0050`, called from the sub-cell step and the arrival routine, `MOVE-STEP-010`/`MOVE-REFRESH-012`), and a refusal from any covered cell aborts the whole registration, so ROM1's own movement never lets one actor's footprint register over a cell another actor's already covers. `MOVE-AREA-038` (High): none of the movement search's six plane-reading routines reads a cell-record occupant slot, so footprint exclusivity is enforced once, at registration, and nothing downstream re-checks it | `pkg/sim/route.go`'s occupancy plane (`routeScratch.occ`) is anchor-only: `occupy` seeds one cell per mover at its own `(X, Y)` and `moved` adds and removes one cell per step; `TokenSize` appears in neither `route.go` nor `step.go`. `entityCoversCell` (`celleffect.go`) is footprint-aware, so an ordinary mover can path onto a cell inside a larger actor's footprint, and `castBookAt`'s Teleport arm (`spell.go`, rule id 26) tests only terrain, with no occupancy test at all | DEVIATION | Making occupancy footprint-aware needs `route.go`'s `occupy`/`moved` to track a mover's whole footprint, `step.go`'s move-execution call site to supply enough to compute it, and `castBookAt`'s Teleport arm to test occupancy beside terrain. The first is `celleffect.go`/`route.go`'s own lane; the second and third are `step.go` (orders) and `spell.go` (weapon), out of this story's file scope | A story reaching `step.go`'s move execution and `spell.go`'s Teleport landing to make occupancy footprint-aware end to end | OPEN |
| DIV-055 | simulation / area effect application on a cell-slot collision | — | `TERR-CELLREC-146` (High): a cell record's `+0x4` (domain 1 or 2) and `+0x8` (domain 3) each hold at most one actor; a second registration returns without storing (`L02040`/`L02041`). `MAGIC-AREACELL-039` (High): a cloud pulse reads `+0x4` alone; blast and ring read `+0x4`, `+0x8` and `+0xc` | `applyAreaCells` (`celleffect.go`) reads every actor covering a queried cell within a layer (`cellLayerOccupants`), not the first one found — superseding fix round 2's single-index `cellSlotOccupants`. Domain keying and dwell-based membership (`cellRecordHolds`) are unchanged from round 2 | DEVIATION | `DIV-054`'s occupancy gap lets this build reach a state — two ground actors covering one cell — that ROM1's own movement never produces. Reading the decoded per-layer cap in that state (round 2) made the second actor immune to every area effect on the cell: reachable on 15 installed `Units` rows carrying `TokenSize > 1` (147 such actors across the 28 EN campaign maps, mission 140 through ordinary movement) and through Teleport with no large unit at all (round 3 review, R3-A2). Restoring the cap before closing `DIV-054` would reproduce that regression | Closing `DIV-054` removes the state this row depends on, after which the per-layer cap can be restored to match `TERR-CELLREC-146` exactly | OPEN |

`DIV-033`'s reason cell states the keying and "the one-actor-per-slot rule" as both "decoded and
implemented", which is no longer accurate — the cap is decoded and deliberately not implemented.
Replace `DIV-033`'s Reason column:

> The keying and the one-actor-per-slot rule are decoded and implemented; what the original does with
> a corpse's slot is not

with:

> The keying and the dwell-based membership are decoded and implemented. The per-layer one-actor cap
> is also decoded but is deliberately not implemented, because this build cannot maintain the
> footprint invariant it depends on (`DIV-054`, `DIV-055`). What the original does with a corpse's
> slot remains separately undecoded

and its Revisit condition column:

> A claim reading the actor slot's clear-on-death path, or a structures story

with:

> A claim reading the actor slot's clear-on-death path, or a structures story, for the corpse
> question; `DIV-054`'s occupancy story for the cap

`DIV-033`'s Implemented behaviour column is accurate as written (it describes dwell-based membership
only, never claims the cap) and needs no change.

## Files

- `pkg/sim/celleffect.go` — `cellSlotOccupants` replaced by `cellLayerOccupants`; `applyAreaCells`
  and `skipOccupiedGround` updated to the widened return; doc comments on both plus `cellRecordHolds`
  correct the decoded-cap claim and cite `TERR-FOOTPRINT-147` and `MOVE-AREA-038` for why this build
  does not maintain it.
- `pkg/sim/spelleffect1001_test.go` — one assertion and its surrounding doc comment corrected in
  `TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots` (entity 5 is now expected hit); nothing
  else in the file touched.
- `pkg/sim/areaoverlap1001_test.go` — new: the footprint-overlap witness (blast/ring/cloud), the
  Fire-Ball-specific footprint witness reproducing the review's own numbers, and the Teleport witness
  driven through the real, unedited `castBookAt`.
- `pkg/sim/route.go` — untouched.
