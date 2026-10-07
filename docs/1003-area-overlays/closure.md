# 1003 — closure

## Integration witness — mission 91, en root

`cmd/areaoverlaycheck` starts a real campaign mission, loads the installed spell table and the
installed `projectiles.reg` with its sheets, and drives every area row through the same simulation
the game uses.

```
mission 91  bounds=80x80  installed rows=28
overlay spell  3  picture 15  sheet frames=11 phases=11  drawn frames=[3 4 5 6 7]
overlay spell  7  picture 23  sheet frames=15 phases=15  drawn frames=[0 1 2 3 4 5 6 7 8 9 10 11 12 13 14]
overlay spell  8  picture 25  sheet frames=15 phases=15  drawn frames=[0 1 2 3 4 5 6 7 8 9 10 11 12 13 14]
overlay spell 19  picture 47  sheet frames=1 phases=1  drawn frames=[0]
row  2 mode=blast  stages= 0 painted-cells=  0 burst-picture=13 life=22 overlay-art=false
row  3 mode=cloud  stages= 0 painted-cells=  0 burst-picture=15 life=16 overlay-art=true
row  4 mode=staged stages= 2 painted-cells= 20 burst-picture=17 life=16 overlay-art=false
row  7 mode=cloud  stages= 0 painted-cells=  0 burst-picture=23 life=16 overlay-art=true
row  8 mode=cloud  stages= 0 painted-cells=  0 burst-picture=25 life=16 overlay-art=true
row  9 mode=staged stages= 5 painted-cells= 25 burst-picture=27 life=18 overlay-art=false
row 12 mode=cloud  stages= 0 painted-cells=  0 burst-picture=33 life= 0 overlay-art=false
row 17 mode=cloud  stages= 0 painted-cells=  0 burst-picture=43 life= 0 overlay-art=false
row 19 mode=cloud  stages= 0 painted-cells=  0 burst-picture=47 life=16 overlay-art=true
row 21 mode=staged stages=32 painted-cells= 32 burst-picture=51 life=16 overlay-art=false
```

Read against research, on the installed table:

- **The mode census matches `MAGIC-AREADRAW-049`** exactly: one blast row, six cloud rows, three
  staged rows.
- **Wall of Fire draws frames 3, 4, 5, 6 and 7 of an eleven-frame sheet**, which is
  `ANIM-WALLFIREFRAME-033`'s five-of-eleven measured against the shipped `firewall` sheet rather
  than against a synthetic one. The other three overlay arms reach every frame of their own sheets.
- **The staged cell counts match `MAGIC-RING-048`.** Fire Sacrifice: 2 stages, 8 + 12 = 20 cells.
  Acid Stream: 5 non-empty stages of 1, 3, 5, 7 and 9 = 25 cells, the sixth intentionally empty on
  an even orientation. Meteor Storm: 32 stages of one cell.
- **Acid Stream's burst life is 18 and every other staged row's is 16**, `MAGIC-AREADRAW-049`'s two
  lifetimes.
- **A cloud reports no paint at all**, so its cells are drawn once, from its retained record.

One honest qualification. The two cloud rows without overlay art, 12 and 17, have burst pictures 33
and 43 that name no loaded sheet either, so they already drew nothing before this story. The
overlay-art restriction is confirmed against the installed data but changes nothing visible on
shipped content; it is a correctness rule for an edited row, not a fix.

## Script-gap census

`pipeline/check-milestone.sh`'s two figures, run from this worktree against the en root:

```
go build -o /tmp/mr1003.exe ./cmd/missionrun
AGAINROM_ASSETS=<en> /tmp/mr1003.exe -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   ->  0
AGAINROM_ASSETS=<en> /tmp/mr1003.exe -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   ->  0
```

Both are **0 and unchanged**. Zero is the floor, so the count cannot have fallen; this story touches
no script node and did not move it. The story's result is in `builds/1003-area-overlays/` instead.

## Revert witness

Each central line was removed, the test was run, and the line was restored with
`git checkout HEAD -- <path>`.

| Line removed | Test | Failure |
|---|---|---|
| the `+ terrain.CellSize/2` term in `effectGroundPoint` | `TestASpellSpriteStandsOnItsCellsCentre` | `blitted at world pixel (160,224); the static layer puts the same art on the same cell at (176,240)` |
| the wall-of-fire phase override and bias in `terrain.OverlayFrame` | `TestABurningCellNeverShowsItsBirthOrFadeFrames` | `a burning cell drew frame 0 at counter 0, cell (0,0)` |
| `obs.recordPaint` in `decayCellEffects`' ring arm | `TestAStagedCastReportsEveryStageItPaints` | `Fire Sacrifice reported 1 stages, want its two` |
| the producer's `+ui.ShotScale/2` restored in `healSpriteDraws` | `TestAHealStarStandsOnItsCellsCentre` | `blitted from base world pixel (152,136), want the static layer's own (136,136)` |

All four passed again after restoration.

## The ground-point producer sweep

The review found a second producer with a hand-rolled half cell, so every producer feeding a point
into `EffectGroundPoint` or into `placeArm` was enumerated and read. A **point** producer is meant
here, not a cell footprint: the marker and footprint builders in `pkg/render/terrain` take a cell
and derive `col*cellpx + cellpx/2` themselves, and are a separate, deliberately independent
derivation.

| Producer | Feeds | Pre-bias | |
|---|---|---|---|
| `healSpriteDraws` (`pkg/game/spellbolt.go`) | `HealSprite.Pos` | **`+ShotScale/2` on X — removed** | the review's counterexample |
| `shotPoint` (`pkg/game/world.go`) | `SpellBolt.Pos` for every travelling bolt and weapon wind-up, and `MapEntity.Shot` for the archer's mark | none | pure interpolation between two cell corners |
| `areaEffectDraws` (`pkg/game/spellbolt.go`) | `SpellBolt.Pos` for the retained overlay | none | added by this story |

`shotPoint` is the only producer of `MapEntity.Shot` (one call site, `world.go`), and
`SetSpellBolts`/`SetHealSprites` have one caller each. So the three rows above are the whole
population. **No third instance was found.**

`EffectGroundPoint` is exported for this reason: a producer on the far side of the seam can now
assert against the function that owns the term instead of restating its arithmetic.

## Aspect matrix

| Aspect | Verdict | |
|---|---|---|
| Data | PASS | `data.OverlaySpells` and `OverlayPicture` carry the four baked ids; `BurstPicture` and `BurstLife` unchanged. |
| Runtime state | PASS | The transient bursts live in `mw.bolts`, the client list that already ages and compacts. |
| Simulation | PASS | One observation channel on `castObs`' own pattern. `TestObservationChangesNoWorld` compares the marshalled bytes of an observed and an unobserved advance. |
| Player input | N/A | No input path touched. |
| AI | N/A | Autocast reaches the same landing arm and is unchanged by it. |
| UI / HUD | PASS | The ground point, the overlay frame law and the mode fork, each with its own test. |
| Triggers / scripts | PASS | `stepScriptCasts` threads the sink, so a script-authored staged cast reports its stages exactly as a player cast does. |
| Inventory / equipment | N/A | |
| Persistence / save-load | PASS | `formatVersion` stays 53. No canonical field added; `nostate_test.go`'s pinned field set is untouched and green. |
| Campaign / session | PASS | The witness above runs in mission 91's own world. |
| Shipped content | PASS | Every installed area row is walked in the witness; all ten behave as their claims state. |
| Interactions with existing mechanics | PASS | Three producers feed a point into the corrected ground point. The heal shower carried a hand-rolled `+ShotScale/2` on X against the old uncentred consumer, so the new term doubled on that axis; the producer's compensation is removed and the term is supplied once, by `ui.EffectGroundPoint`. `shotPoint` (the archer's mark and every travelling bolt) and the area overlay's own position carry no pre-bias and were verified by reading them. The point-cast burst at the landing cell is unchanged and out of scope. |

No in-scope GAP stands.

## Divergences

`DIV-077` (Meteor Storm's descent is undecoded), `DIV-078` (the frame law's spatial term is read as
the cell's own coordinates), `DIV-079` (which of the two overlay draw passes is which). All three
UNKNOWN and OPEN. `formatVersion` 55 was allocated and is returned unused.

## Gate

`go build ./... && go vet ./... && gofmt -l … && go test -trimpath -count=1 ./...` all clean on a
clean tree, plus `bash scripts/check-no-game-assets.sh`.
