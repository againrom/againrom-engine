# 1002 — effect marks: closure

## Twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `TileSize` added to `terrain.UnitClass` and filled from `units.reg` in `pkg/game/units.go`; the art resolves through `projectiles.Sheet(kind)` at the record index |
| Runtime state | PASS | `mapWorld.markElements` and `mapWorld.stoneHold` in `pkg/game/effectmark.go`, opened at `0xffff`, decremented per rebuild, dropped on close; `TestAnAttachedEffectOpensAnElementAndACloseRemovesIt` |
| Simulation | PASS | Read-only `World.HasEffectSpell` and `World.InvisibleTo`; no world field, byte form or digest changes; both added to the `worldMethods` pin and the no-mutation sweep |
| Player input | N-A | No command is added. The marks follow effects already castable |
| AI | N-A | No decision reads a mark. `InvisibleTo` reuses the detector radius AI acquisition already applies, and does not change it |
| UI / HUD | PASS | `Viewer.entityMarkRects` and the two passes interleaved in `planeSprites`; `TestTheTwoMarkPassesSplitAroundTheActorSprite` |
| Triggers / scripts | N-A | No script node reads or writes a mark. Census unchanged, below |
| Inventory / equipment | N-A | — |
| Persistence | PASS | No byte-form change; `formatVersion` untouched. Both new `mapWorld` fields are ruled cosmetic in `docs/0143-save-and-load/spec.md` FR-5 and `TestEveryMapWorldFieldIsRuled` holds |
| Campaign / session | PASS | `cmd/effectmarkcheck` runs against campaign mission 91 on the preserved `en` install; transcript below |
| Shipped content | PASS | All ten marking kinds name a defined `projectiles.reg` row on the `en` install; the two non-marking kinds name none. Transcript below |
| Interactions | PASS | The `0154` `SpellFX` school ring and the `0161` Heal burst are untouched and draw beside the marks; `TestAnEntityWithNoMarksDrawsExactlyWhatItDidBefore` pins the unmarked band |

No in-scope GAP. Three UNKNOWNs are carried as divergence rows, below.

## Integration witness

`builds/1002-effect-marks/effectmarkcheck.exe -assets <seat>/gameversions/en -mission 91`:

```
install <seat>/gameversions/en: projectile sheets=24 unit classes=34
  spell  5 protection_from_fire   kind=0x0012 sheet=true  frames=6
  spell  6 heal                   kind=0x0014 sheet=true  frames=8
  spell  8 poison_cloud           kind=0x0018 sheet=true  frames=8
  spell 10 protection_from_water  kind=0x001c sheet=true  frames=6
  spell 11 drain_life             kind=0x001e sheet=true  frames=9
  spell 15 invisibility           kind=0x0026 sheet=false frames=0
  spell 16 protection_from_air    kind=0x0028 sheet=true  frames=6
  spell 18 shield                 kind=0x002c sheet=true  frames=5
  spell 20 stone_curse            kind=0x0030 sheet=false frames=0
  spell 22 protection_from_earth  kind=0x0034 sheet=true  frames=6
  spell 23 bless                  kind=0x0036 sheet=true  frames=5
  spell 27 curse                  kind=0x003e sheet=true  frames=5
mission 91: actors=45 tilesize histogram=[{1 45}]
  tilesize=1 spell  5 protection_from_fire   records=  1 behind=  0 in-front=  1
  tilesize=1 spell  6 heal                   records=  0 behind=  0 in-front=  0
  tilesize=1 spell  8 poison_cloud           records=  1 behind=  0 in-front=  1
  tilesize=1 spell 10 protection_from_water  records=  1 behind=  0 in-front=  1
  tilesize=1 spell 11 drain_life             records=  0 behind=  0 in-front=  0
  tilesize=1 spell 15 invisibility           records=  0 behind=  0 in-front=  0
  tilesize=1 spell 16 protection_from_air    records=  1 behind=  0 in-front=  1
  tilesize=1 spell 18 shield                 records=144 behind= 72 in-front= 72
  tilesize=1 spell 20 stone_curse            records=  0 behind=  0 in-front=  0
  tilesize=1 spell 22 protection_from_earth  records=  1 behind=  0 in-front=  1
  tilesize=1 spell 23 bless                  records= 20 behind= 10 in-front= 10
  tilesize=1 spell 27 curse                  records= 20 behind=  8 in-front= 12
  cast spell  5 kind=0x0012 attached=true  records=1
  cast spell  8 kind=0x0018 attached=false records=1
  cast spell 10 kind=0x001c attached=true  records=1
  cast spell 16 kind=0x0028 attached=true  records=1
  cast spell 18 kind=0x002c attached=true  records=144
  cast spell 22 kind=0x0034 attached=true  records=1
  cast spell 23 kind=0x0036 attached=true  records=20
  cast spell 27 kind=0x003e attached=true  records=20
```

Three results from the install, none of which an asset-free test can see:

1. Exactly the two kinds `MAGIC-ACTOR-066` routes to the by-kind arms, `0x26` and `0x30`, have no
   `projectiles.reg` row. All ten kinds that reach a builder have one. That agrees with
   `MAGIC-MARK-061` and `MAGIC-PIC-026` on installed data.
2. Every builder's phase index stays inside its own sheet: the Protections' `countdown mod 6`
   against 6 frames, the Shield's and Bless/Curse's `4 - q` in 0..4 against 5 frames.
3. Seven of the eight marking spells cast in the mission's own terrain and spell table attach to an
   actor and therefore open an element. Poison Cloud does not, and is DIV-076.

## Script-gap census

`pipeline/milestone-baseline.txt` records script node counts, not `UNSUPPORTED` lines. Both figures
were measured against the preserved `en` install:

```
this branch, missionrun built from 2e318c7:
  -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
  -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
master, builds/current/missionrun.exe:
  -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
  -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
```

The master binary was built shortly before `badfd70`, which is a documentation-only commit. Both
numbers are unchanged by this story, which adds no script node handler.

This story's result is not in that census. It is in `builds/current/`: the four Protections, the
Shield, Bless and Curse draw shipped art on the actor, and Invisibility and Stone Curse change the
actor's own draw.

## Revert witness

The central behaviour is FR-1's two-pass split. Folding the first pass into the second in
`pkg/ui/statics.go` — `front = append(back, front...)` in place of `out = append(out, back...)` —
reddens `TestTheTwoMarkPassesSplitAroundTheActorSprite` on all three assertions:

```
--- FAIL: TestTheTwoMarkPassesSplitAroundTheActorSprite (0.00s)
    effectmark_test.go:41: band[0] = {screenRect:{X:11 Y:49 W:10 H:6} ... }, want the depth>0 mark before the sprite
    effectmark_test.go:44: band[1] = {screenRect:{X:13 Y:51 W:8 H:8} ... }, want the actor's own sprite
    effectmark_test.go:60: the back mark landed at {X:11 Y:49 W:10 H:6}, want {X:13 Y:51 W:8 H:8}
```

`git checkout HEAD -- pkg/ui/statics.go` restores green. The line is witnessed by reverting it, not
by reading the assertion.

## Research reconciliation

| Claim | Built as decoded | Not built |
|---|---|---|
| `MAGIC-MARK-059` | The record, the array, the two passes split on `depth`, the position expression | — |
| `MAGIC-MARK-060` | Re-derivation per rebuild, open at `0xffff`, close by effect removal, decrement per rebuild | The third writer, the projectile driver. DIV-076 |
| `MAGIC-MARK-061` | `kind = 2*spellId + 8`, the dispatch range, the seven builders, the non-marking set | — |
| `MAGIC-PROT-062` | The whole builder and its placement table | — |
| `MAGIC-SHIELD-063` | The rasteriser, the envelope, two records per point, the signed depth, the frame quotient | Rotation per step, point count and horizontal extent are stated as not established; authored. DIV-075 |
| `MAGIC-BLESS-064` | 20 marks, five steps of 18 degrees, opposite senses, `4 - step/18` | The assignment of the four sign combinations; authored. DIV-075 |
| `MAGIC-CLOUD-065` | Poison Cloud's single mark builder | Heal's and Drain Life's carry-forward arms. DIV-076 |
| `MAGIC-ACTOR-066` | Both kinds, both arms: the animation hold and the sprite gate | The gating bit's owning table was not identified; the participant test is authored. DIV-074 |
| `MAGIC-PIC-026` | The record index is the picture id | — |

Divergence rows: DIV-074, DIV-075, DIV-076 in `docs/DIVERGENCES.md`. All three allocated ids are
spent. `formatVersion` 54 is returned unused: the serialized byte form does not change.

## Not witnessed

The on-screen result was not witnessed by driving the GUI. The owner's desktop was in use and this
lane sent no synthetic input to it. What is witnessed is the geometry, the element lifecycle, the
band order and position, and the installed art and spell rows through `effectmarkcheck` against
campaign mission 91. Whether the marks look right on screen is unverified here and is for the owner
to see in `builds/current/`.
