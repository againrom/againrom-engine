# Lightning figure as the original builds and draws it

## Intent

A Lightning or Prismatic Spray bolt is generated, phased and drawn on both
ROM1 roots as MAGIC-275..282 publish it. The owner directed that the random
source is the engine's own; everything that consumes the values matches.

Base: `ba0b7ef3` (game 0.101.0). Knowledge pin: k207, moved to k208.

## Authority

| Part | Claims |
|---|---|
| producer endpoints, rotation about the first sample, truncation, 16-bit store | MAGIC-275 |
| walk: sign draw, two draws per attempt, `abs(v)>2`, `d>0.15`, alternating sign, `[-3,3]` clamp, 0.7 stop, (1,0) end | MAGIC-276 |
| -0.5 midpoint knots, 43-operation quadratic, half-open sampling every 6 | MAGIC-277 |
| inclusive band `0.15*L`, whole re-run on the continued stream | MAGIC-278 |
| CRT recurrence; the stream between calls | MAGIC-279 |
| one stamp per stored point in list order; sheets 34/36; frame `phase` and `phase+5*tag`; ramp | MAGIC-280, ANIM-BOLTDRAW-034, ANIM-BOLTRAMP-035 |
| route initial phases and counters; geometry on every live call | MAGIC-281, MAGIC-272 |
| scaled hypot, constant bits, control word | MAGIC-282 |
| picture gate, stationary object, end | MAGIC-BOLTGATE-069, MAGIC-BOLTSTILL-072, MAGIC-BOLTEND-074 |

Retired with the partial retractions: MAGIC-BOLTSHAPE-070's no-interpolation
and -0.5 smoothing clauses (the authored corner cutting and stamp
densification are removed), MAGIC-272's direct sequence `4,3,2,1,0` (now
`0,4,3,2,1`), and the universal 13-call clause of MAGIC-BOLTSTILL-072 and
ANIM-BOLTRAMP-035 (the phase is per route).

## As built

- `boltFigure` (`pkg/game/boltfigure.go`) is the producer: float64 with every
  product rounded by an explicit conversion, so no target fuses a multiply-add
  (DIV-2680). Constants are declared by their published bits. Re-runs and walk
  attempts are bounded far above any reachable count (DIV-2682).
- The rand recurrence is MSVC's (`boltRNG`). Each driver call seeds it from the
  observation seed and the object's age; a re-run continues that stream
  (DIV-2681, owner).
- Endpoints are native display pixels: the launch point and the target cell
  centre as ground pixels less the cell's terrain height
  (`Viewer.DisplayHeight`, zero in the flat view) (DIV-2683).
- `pathDraws` hands one `ui.SpellBolt` per stored point with `Display` set. The
  viewer places it at `(x-8, y-8)` and moves it into the camera world by
  `-MinV`; no cell lift is interpolated. Sheet 34 draws frame `phase`, sheet
  36 frame `phase+5*(victimIndex%7)` (DIV-2684 for pixels).
- `boltDriverPhase` runs the ramp `4,3,2,1,0,1,2,1,0,1,2,3,4` over
  `age+1` calls from actionphase 0 (caster) or -1 (cell source), keeping the
  phase outside 1..13. Normal casts live 13 calls, direct 0x8b Lightning 5,
  and a source-cell 0x8c Prismatic Spray 13: a trigger-cast Prismatic Spray
  now draws one link per resolved victim (`ScriptCastEvent.Victims`,
  observation only).
- LOAD: the visual record keeps age, life and the cell-source flag, so a
  restored bolt continues its counter and phase. The SAV visual record gains
  no field. The simulation's loaded picture 34/36 Prj now keeps its phase
  outside actionphase 1..13 (MAGIC-281); an original SAV's live bolt is not
  drawn (DIV-2685).
- The path light stamps the stored display points: column `x>>5`, row from the
  ground picker edge model (`Viewer.DisplayRow`) (DIV-2659 amended).
- Hashed state: only `SavedProjectile.Phase` of a loaded picture 34/36 record
  whose actionphase leaves 1..13 changes. The figure, its seed and the
  observation victims are presentation.

## Divergences

Closed: DIV-080. Narrowed: DIV-081 (ramp and phase block removed). Amended:
DIV-2659. Added: DIV-2680..2686. DIV-2687 is unused.

## Proof

- `pkg/game/boltfigure_test.go`: constant bits; the walk on a scripted stream
  (admission, draw count, clamp, sign, removal past 1); midpoint knots;
  half-open sampling; the coefficient program bit-equal to a math/big oracle
  of the published listing; the hypot; truncation and word wrap; rotation; the
  re-run on the continued stream; 125 whole figures equal to the oracle; one
  pinned figure.
- `pkg/game/boltroute_test.go`: phase sequences of the normal, direct and
  source-cell routes; picture 36 frame for victim index 9; one stamp per stored
  point in order; restore at calls 2..5 continues the sequence.
- `pkg/ui/spellbolt_test.go`: display stamp placement; display row.
- `TestReleaseLightningFigureOverItsLife` (EN and RU): 13 calls of an
  installed Lightning cast on mission 41, every stamp on the installed
  5-frame `lightnin` sheet at the ramp frame and the stored point.

## Open debt

DIV-2680 (control word), DIV-2683 (display fields), DIV-2684 (native
pixels), DIV-2685 (original SAV live bolt), DIV-2686 (visible-frame order).
