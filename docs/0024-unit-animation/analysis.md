# Analysis — unit animation

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — render-side frame selection over the shipped snapshot; no watcher tool, so static + discipline |
| Threshold | **Medium** — nothing reaches hashed sim state; the one sim-adjacent seam is read-only and `pkg/sim` is untouched by contract |
| Terrain — `pkg/data` | **brownfield revision**: the 0016 defaults debt goes live (FR-1), plus an additive descriptor |
| Terrain — `pkg/render/terrain`, `pkg/ui` | **brownfield, additive**: the bundle grows frames and descriptor data; the sprite pass learns the mirror |
| Terrain — `pkg/game` | **brownfield**: loader, resolution seam, scene clock, facing memory |

Research pin `3d95f2a`, frozen for the story; `claims/retracted.md` read first.

## The baseline aged in three ways

**The seams.** `MapScene`, `openrom`, `sprite256`, `LoadUnitSprites` and `ObjectAnchor` occur
nowhere (grep over the module). What is here: `terrain.UnitClass`/`UnitSet`/`UnitPlace` over
`StaticAnchor`, `game.LoadUnits` and the unexported map-world holder pushing `ui.MapEntity`
values through `SetEntities`, `mapload` owning ids (the unit-slice index) and cells,
`cmd/againrom -check` already exercising the unit load headlessly. Two baseline doubts dissolved
on reading: the snapshot DOES carry `TargetX/TargetY/HasTarget` (0019) — the facing derivation
needs no render-side position-delta reconstruction — and the mirrored-hotspot question needs no
anchor signature at all, because the decoded mirror never reaches the destination arithmetic.

**Both research blocks are closed at this pin.** R-1: the sheet layout is decoded at
instruction level and differs from the baseline's ROM2 transcription in one load-bearing place —
the idle block is not "the last `ID·D` frames from the end"; it sits at the bone base
`S + D·(MB+MV+AT+DY)`, the two blocks sharing it disjointly on shipped data. The from-the-end
formula coincides exactly where `BN = 0`, which held on every shipped idle class — why the
transcription kept passing its own corpus check. R-2: `Flip` is a sheet-layout switch (9/5
stored, reflection by `16−g`/`8−d`, the mirror a blit argument). What survives of either item is
one cosmetic sliver: which compass direction stored dir 0 depicts.

**The phase model was confirmed and sharpened, not contradicted.** The baseline's corpus-proven
cycle semantics — `len(AnimFrame)` steps, step k held `AnimTime[k]` ticks, looping — is the
observable shape of what the loader actually builds: a run-length track (`Frame[i]` repeated
`Time[i]` times) that the move arm indexes modulo its own length. The Ghost ping-pong is the
expansion working. One refinement matters to the contract: the idle arm takes NO modulus in the
engine, so looping is our choice, disclosed.

## The defaults debt goes live

0016 declared `units.reg` per-key defaults undecoded and kept Go zeros; 0022 filed the note
("a possible 0016 revision"). This story's descriptor consumes `MB`, `MV`, `AT`, `DY`, `ID` and
`Flip` — four of the six default to −1, not 0, where a chain never sets them — so the debt is
now load-bearing and FR-1 adopts the decoded table wholesale: adopting only the anim keys would
leave one record resolved under two conventions. Checked against the tree first: `pkg/data`'s
inheritance mechanics already match the decoded loader — scalars chain through the parent's
resolved row by presence, arrays one hop by resolved length, the empty-string sentinel
inheriting — so only the absent-everywhere values move, and `defaults_test.go`'s pins move with
them.

## What the story is not

The engine's corpse arms substitute `classes[Dying]`'s sheet and phase counts for the unit's
own — out of scope with the whole death chain, and the exclusion is grounded rather than
assumed: the idle base needs no corpse fact, because the block identity is over the class's own
sheet and its own scalars. `Z`-based air layering is likewise out — decoded as a draw-layer
switch, nothing this story renders differently.

## What we looked at

`pkg/sim` (world, step — the snapshot's actual fields), `pkg/game` (units, statics, world,
census, frontend), `pkg/render/terrain` (units, statics), `pkg/ui` (overlay, statics, viewer's
draw and entity layer), `pkg/data` (keys, classes, load, sprite), `pkg/mapload`,
`cmd/againrom`, `cmd/classdump`; `docs/0016`, `0022`, `0023` as landed; in research at
`3d95f2a`: `claims/retracted.md` first, then `spr256.md` (`SPR256-UNIT-024`), `terrain.md`
(`TERR-SPR-038`/`041`/`047`/`048`, `TERR-MOVE-056`), `reg.md` (`REG-UNITS-018`/`049`/`050`/
`051`, `REG-KEY-044`/`045`); and the staged baseline, read as a hypothesis set.

## Open rather than guessed

- The compass anchor of stored direction 0 — cosmetic; the visual check arbitrates, and the
  step tables `TERR-MOVE-056` names would bear a definitive answer if wanted.
- What advances the engine's phase counter, state word and corpse stage — no writer located; our
  clock and classifier are choices, not transcriptions.
- States 2 and 4 of the selection switch; the original's animation cadence; sprite lighting —
  each disclosed in provenance with what would settle it.
- `Unit33`: the engine runs off the end of its sheet unguarded; here the bounds guard makes the
  same data draw its static frame.
