# 1001-spell-effects — round 2, area lane

Branch `story/1001-r2-area`, based on `5cee2da22d879246fb38752c95c244b911872cf3`, research pin
`744214fe1f4b461cbd2d6766390b850ab15779ab`.

This lane owns review findings A4, B3, B4, D1, D2, D3, D4, D5, E1 and E4, plus one item added
mid-round from the owner's 2026-08-15 play report: area-effect sprites are drawn at the map's
origin instead of on their cells.

Every fix below has a witness that fails at `5cee2da` and passes on this branch. The failure was
produced by reverting the fixed lines and running the named test, not by reading the assertion.
The revert output is quoted with each item.

## Research authority

Read from the pin before the behaviour each governs was changed: `MAGIC-AREACELL-039`,
`MAGIC-AREATICK-036`, `MAGIC-AREAPULSE-037`, `MAGIC-WALLEARTH-042`, `MAGIC-RING-048`,
`TERR-CELLREC-146`, `TERR-FOOTPRINT-147`, `TRIG-CELLEFFECT-045`, `MOVE-DOM-024`. The two wall
tables' values come from `research/experiments/EXP-0172-area-tick/evidence/wall-tables-and-switches.txt`;
the shipped per-row columns from that experiment's `evidence/area-effect-modes.csv`.

## What was wrong and what changed

### A4 — Wall of Fire painted a 25-cell diamond

`celleffect.go` forked the painted shape on `rule.ID == 19`, so only Wall of Earth got wall
geometry. `MAGIC-AREACELL-039` selects the paint by the row's `Distribution system` column: 3
walks the diamond, 4 goes through `R1066` and the two wall tables. The installed table
carries `Distribution = 4` on exactly two rows, Wall of Fire (3) and Wall of Earth (19), and
`Distribution = 3` on the five diamond rows.

The shape dispatch now reads `rule.Distribution`. The occupied-cell refusal stays on spell id 19
alone, because `R1079` tests the cell's `+0x4` slot for id `0x13` only
(`MAGIC-WALLEARTH-042`), so Wall of Fire paints over an occupied cell and Wall of Earth does not.

Witness: `TestBothShippedDistributionFourRowsPaintAWall` (`pkg/sim/spelleffect1001_test.go`), and
on installed data `TestTheRealMissionWitnessSeparatesTheWallFromTheDiamond`.

Reverted to `if rule.ID == 19`:

```
spelleffect1001_test.go:703: Wall of Fire painted 21 cells, want the table's 10
```

Reproduce: `go test -trimpath -count=1 -run TestBothShippedDistributionFourRowsPaintAWall ./pkg/sim/`

### D1 — the diagonal wall was a drawn line, not table B

`wallCells` built its cells from the bearing's signs: a 5x2 block on an axis, and a 1-thick line
spanning +/-4 on a diagonal. The four axis arms happened to agree with table A cell for cell,
which is what fixed the octant convention; the four diagonal arms did not agree with table B at
all. Table B is 9 cells as an anti-diagonal of 5 plus the same line shifted by (+1,0), spanning
+/-2.

`wallCells` is now the two tables and the 8-way switch at `L05427`, transcribed with the
switch's own arms: table, optional dx/dy array swap, and one or both multipliers negated.

The direction byte changed convention with it. `effect+0x4a` is written once, at `L05428`, as
the caster-to-target bearing truncated by `>> 5` — the same 0..7 index Acid Stream reads as its
orientation. The build had two conventions: a 0..8 three-by-three grid index for walls and the
0..7 bearing for rings. There is one now (`areaDirection`), and the wall switch is indexed by it.
The four axis arms produce the same cells as before under the new index, so this is not a
behaviour change for an axis wall.

Witness: `TestWallGeometryIsTheShippedTablesAndNotADrawnLine`, which builds its expectation from
the two tables and the transform rather than from 80 literal coordinates.

Reverted to the drawn line:

```
spelleffect1001_test.go:666: direction 1 wall = [4112 4369 4626 4883 5140 5397 5654 5911 6168],
                             want [4626 4627 4883 4884 5140 5141 5397 5398 5654]
spelleffect1001_test.go:666: direction 3 wall = [4120 4375 4630 4885 5140 5395 5650 5905 6160],
                             want [4630 4885 4886 5140 5141 5395 5396 5650 5651]
```

Directions 0, 2, 4 and 6 passed under the revert, which is the axis half agreeing.

Reproduce: `go test -trimpath -count=1 -run TestWallGeometryIsTheShippedTablesAndNotADrawnLine ./pkg/sim/`

### D2 — the cloud diamond walked one cell too far

`diamondCells` was called with `r + 1` and filtered on `<= r`, which walks `[-(r+1), r+1]` and
paints the whole Manhattan diamond of radius `r + 1`. `R1067` walks `[-r, +r]` and filters
on `|dx| + |dy| <= r + 1`: the filter reaches one cell further than the loop, so the four axis
tips at distance `r + 1` are never produced. The painted set is that diamond intersected with the
`(2r+1)` square.

The helper now takes the row's Radius and carries both quantities. Counts: radius 1 gives 9
(was 13), radius 2 gives 21 (was 25), radius 4 gives 57 (was 61). The four extra cells were also
outside the `(2r+1)` box the pulse scan walks, so they were painted and never pulsed.

Witness: `TestTheCloudDiamondStopsAtTheLoopBoundNotAtTheFilter`, which asserts the count, the
absence of each axis tip, and cell-by-cell membership against the filter.

Reverted to the `<= r` filter:

```
spelleffect1001_test.go:754: radius 1 painted 5 cells, want 9
```

Reproduce: `go test -trimpath -count=1 -run TestTheCloudDiamondStopsAtTheLoopBoundNotAtTheFilter ./pkg/sim/`

### D3 — the cloud pulse phase came from elapsed ticks

`decayCellEffects` counted elapsed ticks and pulsed on every sixteenth. `MAGIC-AREAPULSE-037`
starts `effect+0x4c` at `V0 = (AreaDuration << 4) + (power << 4)/10`, decrements it once per tick,
and pulses when the **new value** is a positive multiple of 16. The two agree only when
`V0 % 16 == 0`; otherwise the whole pulse train moves by up to 15 ticks.

The pulse now reads the counter. `cellEffect.Remaining` is that counter plus one in this build,
because the decay pass tests the lifetime before it subtracts and `areaLife` returns `V0 + 1`, so
the decoded new value is `Remaining - 1` after the decrement.

`Phase` still counts elapsed ticks for a cloud, and is now presentation only: it is the age the
render seam feeds to the sheet's animation clock (`pkg/game/spellbolt.go`). Dropping the increment
would freeze a standing cloud on one frame, which is half of the owner's play report below.

Both existing fixtures picked a power that hid the defect. The unit test used power 0 with
`AreaDuration = 1`, giving `V0 = 16`; the mission-91 witness uses Mind 100, power 70 and
`AreaDuration = 15`, giving `V0 = 352`. Both are multiples of 16. The rewritten test uses Mind 35,
power 5, `V0 = 24`: one pulse, on tick 8, where elapsed ticks put it on tick 16.

Witness: `TestACloudPulsesOnItsOwnCounterAndNotOnElapsedTicks`.

Reverted to `e.Phase%16 == 0`:

```
spelleffect1001_test.go:229: counter tick 8 = friend 100 enemy 100 air 100;
                             want both slot +0x4 occupants hit and the flyer untouched
```

Reproduce: `go test -trimpath -count=1 -run TestACloudPulsesOnItsOwnCounterAndNotOnElapsedTicks ./pkg/sim/`

### D4 — the occupant walk was not keyed by domain

`applyAreaCells` walked every entity covering the cell in id order and stopped after three,
counting corpses toward the cap. `TERR-CELLREC-146` (High) gives the cell record's payload as
`+0x4` an actor of movement domain 1 or 2, `+0x8` an actor of domain 3, `+0xc` a structure and
`+0x10` a sack, with **both actor slots holding at most one actor** — the add returns without
storing when the slot is taken. This build's `Domain.layer()` is the same split: ground and ghost
on layer 0, the flyer on layer 1, which is the original's `< 3 -> 0x40` / `== 3 -> 0x80`
(`MOVE-DOM-024`).

New `cellSlotOccupants` returns the cell's one layer-0 actor and its one layer-1 actor. A cloud
pulse reads `+0x4` alone (`L05435`); ring and blast read all three (`L05436` / `L05439` and
their two neighbours). The structure slot has no analogue among this world's entities and is
always empty here.

Membership is `cellRecordHolds`: `counted()`'s life ladder (route.go) without `counted()`'s flyer
rule. An off-map actor holds no slot, a body holds its cell while it lies where it fell and none
once it has begun to decay. The flyer rule is deliberately not borrowed — a flyer with an order
contends with no mover but is still over the cell, and borrowing it would let a moving flyer fly
out of a Fire Ball.

The old test cemented the wrong rule with four living entities on one cell, which the occupancy
model forbids. The replacement uses the reachable configuration the review measured: three decayed
bodies and a living actor on one cell.

Witness: `TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots`.

Reverted to the id-order walk with a cap of three:

```
--- FAIL: .../blast_reads_+0x4,_+0x8_and_+0xc
    spelleffect1001_test.go:614: the cell's one ground-layer occupant 4 was not hit
```

Reproduce: `go test -trimpath -count=1 -run TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots ./pkg/sim/`

### D5 — Wall of Earth's refusal read the anchor cell only

`skipOccupiedGround` tested `e.Alive() && e.Domain != DomainAir && e.X == x && e.Y == y`.
`TERR-FOOTPRINT-147` (High) stores an actor's pointer in every cell of its `n x n` footprint and
makes the entry all-or-nothing, so a cell inside a large unit holds that unit. The refusal now
asks `cellSlotOccupants` for the cell's `+0x4` occupant, which is footprint-aware and uses
`cellRecordHolds`, so a fallen body refuses its cell and a flyer overhead does not.

Witness: `TestWallOfEarthRefusesEveryCellOfAnOccupantsFootprint` — a live 3x3 unit, a fallen body,
and a flyer under one 10-cell wall.

Reverted to the anchor test:

```
spelleffect1001_test.go:794: the wall painted (21,19), inside the 3x3 unit's own footprint
spelleffect1001_test.go:794: the wall painted (20,20), inside the 3x3 unit's own footprint
```

Reproduce: `go test -trimpath -count=1 -run TestWallOfEarthRefusesEveryCellOfAnOccupantsFootprint ./pkg/sim/`

### B3 — a save taken while a staged effect stood could not be loaded

The ring generators produce their cells in the decoded visit order, which is significant at
application and is not ascending. That list was stored on the standing record, and
`castbinary.go` refuses a decoded cell list that is not strictly ascending, so `MarshalBinary`
wrote a form `UnmarshalBinary` refused.

The visit order is significant **at application** and nowhere else: the layer overlap test, the
passability restore and the client draw all read the stored list without order. So the application
keeps the generator's own list, and the record keeps `canonicalCells(cells)` — the same cells,
ascending, each once. The decoder's strict-ascending refusal stays, which is what makes a decoded
cell list a set rather than a sequence two worlds could spell two ways.

The byte layout is unchanged; `formatVersion` 53 does not move.

Witness: `TestAStagedEffectSurvivesASaveAtEveryTickOfItsLife` — Fire Sacrifice and Acid Stream, all
eight orientations, a round trip at every tick of each program's life.

Reverted to storing the visit order:

```
spelleffect1001_test.go:837: spell 4 direction 0 tick 0: UnmarshalBinary:
                             sim: area effect 0 cell list is not strictly ascending
```

Reproduce: `go test -trimpath -count=1 -run TestAStagedEffectSurvivesASaveAtEveryTickOfItsLife ./pkg/sim/`

### B4 — a Wall of Earth across a stored route made the world unloadable

`decodeRoutes` (`binary.go`) refuses a route cell the entity's own domain cannot cross, under a
comment naming its five refusals as "the states a tick cannot produce". A wall setting
`blockMagicWall` on a cell of a stored route produced that state, and nothing on the movement path
re-tests a stored route against the grid.

New file `pkg/sim/routeblock.go` holds `invalidateBlockedRoutes`, called from `addAreaEffect`
immediately after the wall's bits are set. It asks the decoder's own test — the same grid, the
same `Domain.blocks()` mask, the same `cellIndexIn` — so a route it leaves standing is a route
`decodeRoutes` accepts. A dropped route costs its mover a fresh search: an empty route is what
`subGoal` already answers false for.

It is called only where a cell closes. Removing a wall opens cells, and an open cell cannot make a
standing route illegal.

Witness: `TestAWallAcrossAStoredRouteLeavesALoadableWorld`.

Reverted by removing the call:

```
spelleffect1001_test.go:874: the mover kept a 29-cell route across the wall
spelleffect1001_test.go:882: a world holding a wall over a stored route did not load:
                             sim: entity record 0: route cell 14 is (20,20), which its domain 0 cannot cross
```

Reproduce: `go test -trimpath -count=1 -run TestAWallAcrossAStoredRouteLeavesALoadableWorld ./pkg/sim/`

### E1 and E4 — the mode came from a spell-id switch

`areaModeFor` hard-coded `2 -> blast`, `{4,9,21} -> ring`, else cloud, and `SpellRule.Distribution`
was parsed, serialized and read by nothing. `MAGIC-AREATICK-036` gives the selector: the builder
writes `effect+0x08` three times, and the `Distribution system == 5` arm runs after the
`Area Effect Duaration` arm and overwrites it. So `Distribution == 5` selects staged, else a
positive duration column selects cloud, else blast — and a staged row's duration column has no
effect, which is why `meteor_storm`'s shipped `Area Effect Duaration = 10` does nothing.

`areaModeFor` now takes the rule and reads those two columns. On the installed table the two
spellings agree exactly (1 blast, 3 staged, 6 cloud), so no shipped behaviour moves; an edited row
now selects its own mode.

`areaLife`'s `rule.ID == 21` branch was **not** unreachable, contrary to the review's reading: the
area path does overwrite `Remaining` for a staged row, but `SpellCharacteristicsFor`
(`pkg/sim/spell.go`) calls `areaLife` for every `rule.Area` row, so the spellbook popup showed
Meteor Storm a duration of `(Radius + 2) * 2 = 12`, a number with no research behind it. The
branch is now a staged arm keyed by the mode, returning `ringLife(id) = (stages - 1) * 3 + 1` from
`MAGIC-RING-048`'s cadence — 4, 16 and 94 ticks for Fire Sacrifice, Acid Stream and Meteor Storm.
That is one formula for both the record's lifetime and the popup.

Witness: `TestTheAreaModeComesFromTheRowsOwnColumns`, seven rows over mode and life.

Reverted to the id switch:

```
spelleffect1001_test.go:913: Fire Ball given a duration column: mode = 1, want 3
spelleffect1001_test.go:913: Fire Sacrifice without the staged column: mode = 2, want 1
```

Reverted to the `rule.ID == 21` life branch:

```
spelleffect1001_test.go:916: Fire Sacrifice's own columns: life = 1, want 4
spelleffect1001_test.go:916: Acid Stream: life = 1, want 16
```

Reproduce: `go test -trimpath -count=1 -run TestTheAreaModeComesFromTheRowsOwnColumns ./pkg/sim/`

### Owner play report — area sprites were drawn at the map's origin

Report: «спрайты должны анимироваться все действие заклинания и отдельный спрайт на каждую клетку,
а не только центральную».

`ui.SpellBolt.Pos` is documented as being in `ui.ShotScale` units of a cell, and the consumer
divides by that scale (`px := b.Pos.X*terrain.CellSize/ShotScale`, `pkg/ui/spellbolt.go:100`).
`spellDraw` is the only constructor of a `SpellBolt`, and it has three callers: `boltDraws` and
`weaponBoltDraws` pass `shotPoint(...)`, which multiplies both axes by `ui.ShotScale`;
`areaEffectDraws` passed a raw cell. Every area sprite was therefore drawn at `cell/256` of its
own world pixel.

Measured on the witness world, a wall cell at (76,109): the sprite was placed at world pixel
`(76*32/256, 109*32/256) = (9,13)` where its cell is at `(2432,3488)`. `placeArm` takes those
numbers straight into `cam.WorldToScreen`, so the whole set collapsed within a few pixels of the
map's origin.

`areaEffectDraws` now passes `image.Pt(at.X*ui.ShotScale, at.Y*ui.ShotScale)`, which is exactly
what `shotPoint` produces for an object standing on that cell. One object per painted cell was
already how the loop was written and is unchanged.

The animation half of the report is served by the `Phase` increment restored under D3. The
animation **law** is not authored here: what the original draws for a standing area effect is
`EXP-0176`'s question, and `Phase` simply feeds the current sheet clock.

Witness: `TestAnAreaEffectSpriteStandsOnItsOwnCell` (`pkg/game/areadraw_test.go`). It asserts the
position in the units the consumer divides by, over a wall at (76,109) — coordinates far enough
from the origin that the two readings cannot coincide.

Reverted to the raw cell:

```
areadraw_test.go:74: the sprite for cell (76,107) stands at (76,107), want (19456,27392)
areadraw_test.go:74: the sprite for cell (77,107) stands at (77,107), want (19712,27392)
```

Reproduce: `go test -trimpath -count=1 -run TestAnAreaEffectSpriteStandsOnItsOwnCell ./pkg/game/`

### Owner play report — the orange projectile is not accounted for

Report: «при касте заклинаний какой-то оранжевый снаряд улетает по диагонали, выясни что это».

**Not closed by this lane, and not attributed to the placement fix.** What was checked:

- `spellDraw` is the only `ui.SpellBolt` constructor. Its three callers are `boltDraws`,
  `weaponBoltDraws` and `areaEffectDraws`. After the fix all three pass a position in
  `ui.ShotScale` units.
- All three run from the caster's cell to the target's cell. `spawnCast` takes `ev.FromX/FromY`
  and `ev.ToX/ToY` from the simulation's own cast observation; `spawnBurst` puts a stationary
  object at the landing cell with `from == to`; `weaponBoltDraws` interpolates the attacker's cell
  toward the victim's by the swing clock. None of the three can aim somewhere neither actor is.
- `data.CastFlight` gives a non-zero flight to seven pictures. `CastPicture(spell) = 2*spell + 8`
  maps them to spells 1, 2, 6, 11, 13, 14 and 26. **No area spell's cast picture flies**: casting
  Wall of Fire, Wall of Earth, Freezing Cloud, Poison Cloud, Light, Darkness, Fire Sacrifice, Acid
  Stream or Meteor Storm puts no travelling object on the map at all.
- The two orange flying pictures are Fire Arrow's (10) and Fire Ball's (12). A diagonal cast of
  either produces a diagonal projectile from caster to target, which is what the picture is for; a
  long cast is slow, because `CastFlight(10, dist) = dist/200` in units of 256 per cell, so a
  40-cell Fire Arrow flies for 51 ticks.

One possibility this lane cannot rule out without seeing the screen: before the placement fix, an
area effect's sprites were drawn diagonally up and left of their cells — 2423 pixels left and 3475
pixels up for the measured cell — which is a stray sprite appearing off the caster's diagonal.
Whether that is what the owner saw is unverified. This lane did not drive the GUI.

Recommendation: re-check this report against a build carrying the placement fix before opening an
investigation. If it survives, the next places to look are the sim-side cast observation's
`ToX/ToY` for an AI or autocast release, and `advanceBolts`' compaction, neither of which this
lane read.

## Proposed `spec.md` sentences

Ready to paste. Line numbers are `5cee2da`'s.

Replace, in the paragraph at `spec.md:168-174`:

> Cloud mode paints its decoded cells, retains the record for
> `(AreaDuration << 4) + (power << 4) / 10 + 1` ticks when the base duration is positive, and applies
> on every sixteenth elapsed tick. Its collector excludes air and applies to friends as well as
> hostiles.

with:

> Cloud mode paints its decoded cells and retains the record for
> `(AreaDuration << 4) + (power << 4) / 10 + 1` ticks when the base duration is positive. Its
> counter starts at `V0 = (AreaDuration << 4) + (power << 4) / 10`, falls by one each tick, and
> pulses when the new value is a positive multiple of 16, so a cloud whose `V0` is not a multiple
> of 16 pulses off the sixteenth tick. Each pulse reads the covered cell's first occupant slot
> alone, so it reaches one ground or spirit actor per cell and no flyer, and it applies to friends
> as well as hostiles.

Replace, in the same paragraph:

> Other retained clouds paint the Manhattan diamond of radius `r + 1`.

with:

> The painted shape is the row's `Distribution system` column and not its spell id: 4 takes the
> wall tables and 3 takes the diamond walk. The diamond walk runs both offsets over `[-r, +r]` and
> keeps a cell when `|dx| + |dy| <= r + 1`, so the four axis cells at distance `r + 1` are outside
> the walk and are not painted: 9 cells at radius 1, 21 at radius 2, 57 at radius 4.

Replace, at `spec.md:172-173`:

> Wall of Fire and Wall of Earth use directional wall geometry: ten cells for an axis-facing 5-by-2
> wall and nine cells for the diagonal.

with:

> Wall of Fire and Wall of Earth are the two `Distribution system = 4` rows and use the same two
> cell tables, selected by the cast's bearing truncated to one of eight. The four axis bearings take
> the 10-cell table, a 5-by-2 block; the four diagonal bearings take the 9-cell table, an
> anti-diagonal of five plus the same line shifted by one cell in x, spanning two cells either side
> of the target. A wall's cell count does not come from its radius column.

Replace, at `spec.md:166`:

> Blast visits ground, spirit and air occupant domains.

with:

> A covered cell holds at most one ground or spirit actor and at most one flyer, keyed by movement
> domain, and a third slot for a structure that this build's entities never occupy. Blast and staged
> modes read all three in that order; a cloud pulse reads the first alone. A body occupies its slot
> while it lies where it fell and none once it has begun to decay.

Replace, at `spec.md:188-189`:

> Each accepted cell walks the three occupant slots in stable entity order.

with:

> Each accepted cell walks its occupant slots in slot order. Cells are applied in the generator's
> own visit order; the record stores the same cells in ascending key order, which is the byte form's
> own order.

Replace, at `spec.md:202`:

> Wall of Earth refuses occupied ground cells, harms no unit and marks every accepted cell in the
> ordinary passability plane.

with:

> Wall of Earth refuses a cell whose first occupant slot is filled — any cell of a ground or spirit
> actor's footprint, including a body lying where it fell — harms no unit, and marks every accepted
> cell in the ordinary passability plane. A stored route crossing a cell the wall has just closed is
> discarded at the landing, and its mover routes again on its next tick.

Add, after `spec.md:161`:

> An area row's per-tick mode comes from its own columns. `Distribution system = 5` selects staged
> mode and leaves the duration counter at zero, so a staged row's `Area Effect Duaration` has no
> effect; otherwise a positive `Area Effect Duaration` selects cloud mode and a zero one selects
> blast. A staged row's whole life is its own cadence, `(stages - 1) * 3 + 1` ticks, and that is
> also the duration the spellbook popup reports for it.

## Proposed `docs/DIVERGENCES.md` rows

Ready to paste, in the ledger's column order. `DIV-028` onward assumes `DIV-027` is the last row at
the landing; renumber if another lane has taken those ids.

| DIV-028 | simulation / area effect lifetime | — | `effect+0x4c` is the cloud's own counter: instant 29 writes it directly (`TRIG-CELLEFFECT-045`) and the pulse reads its new value (`MAGIC-AREAPULSE-037`) | `cellEffect.Remaining` is that counter plus one, because the decay pass tests the lifetime before it subtracts. An instant-29 write of `D` therefore leaves a counter of `D - 1`, shifting a retimed cloud's pulse phase by one tick and its life by one tick | DEVIATION | The field carries a lifetime in this build, which `0165` established and three tests pin; the offset is one tick against shipped durations of 1, 30000 and 60000 | A story that makes `Remaining` the raw counter and moves the decay boundary with it | OPEN |
| DIV-029 | simulation / cell occupant slots | — | The cell record's occupant slots are keyed by movement domain and hold one actor each (`TERR-CELLREC-146`); `+0xc` holds a structure and `+0x10` a sack | Slot membership is this build's own dwell ladder: a body holds its cell while `Decay == DecayFallen && Dwell > 0` and none afterwards. Whether the original clears an actor's slot on death is not decoded. The structure and sack slots have no analogue among this build's entities and are always empty | UNKNOWN | The keying and the one-actor-per-slot rule are decoded and implemented; what the original does with a corpse's slot is not | A claim reading the actor slot's clear-on-death path, or a structures story | OPEN |
| DIV-030 | simulation / area paint for unshipped distributions | — | `MAGIC-AREACELL-039` names the paint for `Distribution system` 3 and 4 only; `MAGIC-AREATICK-036` names staged mode for 5 | A cloud row carrying any distribution but 4 paints the diamond, and a staged row whose id has no stage program lands nothing. Neither case is shipped: the installed table is 18 rows of 1, 5 of 3, 2 of 4 and 3 of 5 on both roots | UNKNOWN | The customisation seam (G2) needs an answer for an edited column that the research does not carry | A claim reading the paint dispatch's default arm | OPEN |
| DIV-031 | simulation / route invalidation | — | No claim establishes what the original does to a stored route when a wall closes one of its cells | A wall landing discards every stored route that now crosses a cell its own mover cannot cross, at the landing site. Without it the world's byte form refuses a state a tick produced | UNKNOWN | Authored to keep the save loadable; the original's own answer is not decoded | A claim reading the wall's effect on a queued path | OPEN |

## Gate

Run in the worktree at the pushed commit.

```
go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')
go test -trimpath -count=1 ./...
bash scripts/check-no-game-assets.sh
```

All clean; `gofmt` printed nothing; `check-no-game-assets: clean (tree scan)`.

## Measurements

`go run ./cmd/spelleffectcheck -mission 91`, both roots, byte-identical output on each:

| | `5cee2da` | this branch |
|---|---|---|
| spell table | rows=28 point=18 area=10 distributions=[{1 18} {3 5} {4 2} {5 3}] | unchanged |
| campaign sweep | maps=28 cast-at-cell=[{3 3} {8 3} {17 1} {19 23} {21 5}] cast-at-unit=[{5 4} {10 4} {16 4} {20 3} {22 4} {23 1}] cell-effect-age=[{3 1} {19 22}] | unchanged |
| mission 91 point | speed=10->15 attached=1 | unchanged |
| mission 91 area | friend=1000/1000/990 enemy=1000/1000/996 | unchanged |
| mission 91 wall route | ground=true/false/true ghost=true/false/true air=true/true/true | unchanged |
| mission 91 outside-wall | not measured | 1000/1000 |

The witness gained a fourth actor two cells west of the target cell. It is inside the
distribution-3 diamond of radius 3 and outside the 10-cell wall table, so its health is the one
figure on installed data that tells the two shapes apart. It reads `1000/1000` here; with the
shape dispatch reverted the same actor takes 20 damage.

Script-gap census, EN root, `cmd/missionrun -trace -ticks 1`:

| mission | `5cee2da` | this branch |
|---|---|---|
| 10 | 0 UNSUPPORTED | 0 UNSUPPORTED |
| 20 | 0 UNSUPPORTED | 0 UNSUPPORTED |

Unchanged, and this story was not expected to move them. The observable result of this lane is in
`builds/current/`: Wall of Fire lays a wall instead of a diamond, a cloud covers the cells its row
asks for, and an area effect's sprites are drawn on the cells they cover.

## Files

- `pkg/sim/celleffect.go` — mode and paint selection, the wall tables, the diamond bound, the
  pulse counter, the occupant slots, the wall refusal, the canonical cell list.
- `pkg/sim/routeblock.go` — new; route invalidation at a wall landing.
- `pkg/sim/effectwitness.go` — a fourth area actor outside the wall table.
- `pkg/sim/spelleffect1001_test.go` — area witnesses.
- `pkg/sim/scriptcast_test.go` — one fixture line: the area row now carries `Distribution: 4`, the
  shipped Wall of Earth column that selects the wall paint.
- `pkg/game/spellbolt.go` — `areaEffectDraws`' position units.
- `pkg/game/areadraw_test.go` — new; the placement witness.
- `cmd/spelleffectcheck/main.go` — one printf field.

---

# Round 2, area lane — follow-up: the cell form's cost contract

Branch `story/1001-r2-area2`, based on the merged story branch `8e73699`, which carries all three
lanes. Handed back to this lane because the file is its own; scope is this item alone.

## What was wrong

The POINT lane closed the pay-then-refuse hole for the unit form of a book cast by adding
`pointEffect` and `pointEffectRefusal`, asked at admission and before the cost. It could not close
the cell form and named the gap under C1: `castBookAt` pays and then returns when `landArea`
refuses.

`landAreaFacing` refuses in three ways, and the seat's reading of them is confirmed by execution:

- a staged row whose spell id has no cell program (`ringStageCount == 0`);
- an area lifetime of zero;
- a cell already holding the six records one anchor takes, refused inside `addAreaEffect`.

The third is the one reachable in play. Measured at `8e73699`, before any change: a mage with 500
mana casting Meteor Storm at a cell already carrying six records ends the tick at **475 mana, six
records, no recovery started and no experience awarded**. The player pays the row every time.

Meteor Storm is the row that reaches it. A staged row registers no map layer and
`resolveLayerConflicts` returns immediately for one, so repeated casts at one cell accumulate;
its life is 94 ticks. Every layer row instead removes its own older record where the two overlap,
which is why six same-spell cloud records never stand together.

Two further sites were found while confirming the premise, and both are the same defect:

- **`castSpell`'s area arm** — the unit-target form of an area row lands at the victim's cell
  (DD-4). That arm **discarded** `landAreaFacing`'s result, so it paid the mana, marked both
  actors and started recovery for a landing that never happened. Measured under the revert: 25
  mana and 4 ticks of recovery for nothing.
- **Fire Sacrifice** — its arm sets the caster's health to 1 and its mana to 0 *before* the record
  is built, and the six-slot cap was tested after that. A refused Fire Sacrifice took everything
  the caster had. Measured under the revert: a caster at 90 health and 60 mana ends at 1 and 0,
  with no record landed. Nothing downstream restores it; the drain is a direct write, not a cost
  the cast arm could unwind.

## What changed

`areaLandingRefusal(rule, power, x, y)` in `pkg/sim/celleffect.go` is the cell form's counterpart
of `pointEffectRefusal`, in the same idiom: read-only, `""` means admissible, one function
answering for admission, for the pre-cost gate and for the instrument. It states the three
refusals above through the same three functions the landing uses — `areaModeFor`, `areaLife` and
`anchorFull` — and answers `""` for a blast, which retains no record and so is not held by a full
cell.

It is placed beside `landAreaFacing` rather than beside `pointEffectRefusal` because the drift it
must not have is with the landing, not with the point arm. If the seat prefers them adjacent, the
function moves without changing.

`anchorRun` and `anchorFull` are new and are now the one spelling of "which records stand on this
cell": `placeCellEffect` (script instant 21), `addAreaEffect` and the refusal all read them, so an
admission that disagreed with the landing is no longer expressible.

`landAreaFacing` asks `areaLandingRefusal` first, before the Fire Sacrifice arm spends the caster
and before any cell is painted. `addAreaEffect` keeps its own two refusals: they are properties of
the slice, reached independently through `placeCellEffect`'s invariant, and a caller that forgot to
ask must not be able to write a state the byte form refuses. They are now unreachable from the
landing path.

Four call sites ask it: `castBookAt` and `castSpell` before the cost, `bookSpellCellRefusal` (new)
for the cell form and `bookSpellRefusal` for the unit form of an area row. `beginBookSpellAt`'s
inline refusal list moved into `bookSpellCellRefusal` and it now calls it, which is
`beginBookSpell`'s own shape for the unit form — that is what makes the instrument report what the
admission path will do.

`BookSpellCellRefusal(casterID, x, y, spellID)` is a second exported entry point rather than a
widened first one: the two forms take different arguments and answer about different admissions,
and one signature would have to accept a meaningless victim for half its callers. It is added to
`worldMethods` and swept as a reader.

### One case the new order decides differently

At a full cell, a row whose own older record the layer-conflict pass would have emptied used to
make room for itself and land; it now refuses and the older record stands. Reaching it needs six
records on one cell and a seventh cast of a row already among them. No shipped script produces it,
and no claim covers either order.

The old order was already inconsistent with itself: the same row cast at the same full cell from a
**different bearing** paints different cells, does not empty the old record, and was then refused
*after* paying. The new order is uniform, and it is the only order under which admission and
release can be made to agree.

## Witnesses

Every changed line was reverted and the named test re-run. All in `pkg/sim/areacost1001_test.go`;
reproduce with `go test -trimpath -count=1 -run <name> ./pkg/sim/`.

| Line reverted | Test that fails | Failure |
|---|---|---|
| both cell-form gates | `TestACastAtAFullAnchorIsRefusedBeforeItsCostIsPaid` | `a refused cell cast spent 25 mana, want none spent` |
| `castBookAt` pre-cost gate | `TestAnAnchorFilledDuringWindupRefusesAtReleaseWithoutPaying` | `a release onto a filled anchor spent 25 mana, want none` |
| `castSpell` 6c pre-cost gate | `TestAUnitTargetedAreaRowFilledDuringWindupRefusesWithoutPaying` | `a unit-target release onto a filled cell spent 25 mana, want none`; `it started 4 ticks of recovery, want none` |
| `bookSpellRefusal`'s landing call | `TestAUnitTargetedAreaRowIsAlsoGatedAtItsLandingCell` | `the instrument reports a unit-target area cast onto a full cell as admissible` |
| `landAreaFacing`'s hoisted refusal | `TestAFireSacrificeAtAFullAnchorDoesNotSpendItsCaster` | `a refused Fire Sacrifice left the caster at 1 health and 0 mana, want 90 and 60` |
| `landAreaFacing`'s hoisted refusal | `TestARefusedLandingLeavesTheStandingRecordsUntouched` | the refused cast shrank a standing record |

The first row needs **both** gates removed at once: with either standing the cast never reaches the
other, which is what asking at admission and before the cost buys. The two wind-up tests are what
separate them — a cast admitted at an empty cell can be released onto a full one, because the cell
fills during `castWindupTicks`.

`TestTheCellFormRefusalIsReportedByTheReadOnlyInstrument` and
`TestTheAdmissionPredicateAndTheLandingAgreeOnEveryAreaRow` carry the instrument and the
admission/landing agreement over all three refusals. Neither is in the table because both name
symbols that do not exist at `8e73699`.

The precondition in every test is built by the production landing path, never by writing
`w.effects`: a precondition assembled by hand could hold a state a tick cannot produce, and the
refusal under test would then be answering about nothing.

## Proposed `spec.md` sentences

Ready to paste. To the paragraph stating the cost contract (`spec.md:214-216` at `5cee2da`: "A
refusal, a release-time cancellation and an effect that cannot apply spend no mana, start no
recovery and award no training"), append:

> That covers the cell form as well as the unit form. An area landing is refused before its cost
> when the row selects a staged program its spell id has none of, when its computed lifetime is
> zero, or when the target cell already holds the six records one cell carries. The same three
> refusals are asked at admission, at release, and by the read-only instrument, so a cast the
> instrument calls admissible is one the release will land. A blast is not held by a full cell,
> because it applies once and retains no record. Fire Sacrifice's payment of the caster's health
> and mana happens after those refusals, not before them.

And, to the area-effect section:

> Where a cast is refused, nothing on the anchor changes: no cell is painted, no overlapping
> record is shortened, and no record is removed. At a full cell this refuses a row whose own older
> record would otherwise have been displaced to make room.

## Proposed `docs/DIVERGENCES.md` row

The lane's four earlier rows were allocated `DIV-032` through `DIV-035`. This is one more;
renumber if the seat has already taken `DIV-036`.

| DIV-036 | simulation / area landing at a full cell | — | `TRIG-CELLEFFECT-045` reads the retiming arm walking six consecutive effect pointers on one cell record, so six is the structure. What the original does with a seventh landing, and whether it runs its layer-conflict rule before or after that test, is not decoded; its conflict rule is per cell inside the per-cell add (`MAGIC-WALLEARTH-042`), not a separate pass | A seventh landing is refused before any cost, paint or conflict resolution. At a full cell a row whose own older record the conflict pass would have emptied is refused rather than displacing it | UNKNOWN | Authored so admission, release and the read-only instrument give one answer; the alternative order paid the caster's mana for a landing that did not happen | A claim reading the seventh-landing path | OPEN |

## Gate

```
go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')
go test -trimpath -count=1 ./...
bash scripts/check-no-game-assets.sh
```

All clean; `gofmt` printed nothing; `check-no-game-assets: clean (tree scan)`.

`go run ./cmd/spelleffectcheck -mission 91`, both roots, identical to the figures recorded above
including `outside-wall=1000/1000`. Script-gap census unchanged: mission 10 and mission 20 both 0
UNSUPPORTED. This item changes no shipped-content behaviour, since no shipped script casts an area
row at a cell already holding six records, so both instruments were expected to be unchanged and
are.

## Files

- `pkg/sim/celleffect.go` — `areaLandingRefusal`, `anchorRun`, `anchorFull`; the refusals hoisted
  to the head of `landAreaFacing`.
- `pkg/sim/spell.go` — the pre-cost gate in `castBookAt` and `castSpell`; `bookSpellCellRefusal`
  and the exported `BookSpellCellRefusal`; `bookSpellRefusal` reports the landing;
  `beginBookSpellAt` reads the refusal instead of restating it.
- `pkg/sim/areacost1001_test.go` — new; the seven witnesses.
- `pkg/sim/world_test.go` — the new exported reader in the method census and the sweep's args.
