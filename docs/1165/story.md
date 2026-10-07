# Wall and fire occlusion

Earth and Fire walls now cover scenery earlier in their cell sweep. Ordinary
actors remain behind trees and structures on later rows. Air actors use their
separate late shadow/body passes.

## Authority and scope

Knowledge snapshot13 is `abdefd56d239c6f203889c0d77521045e3b1012a`.
`ANIM-REGISTER-083`, `ANIM-CATEGORY-084`, `ANIM-CELL-085`,
`ANIM-AIRPASS-086`, `ANIM-DRAWGATE-087` and `ANIM-WALKORDER-088` provide
the registration/category/cell-pass join. The narrowed clauses of `ANIM-047`,
`REG-UNITS-061`, `TERR-STRUCT-104`, `TERR-SPR-065` and
`TERR-SPR-137` through `139` were read with their retraction rows.

The slice changes presentation classification and submission order. It changes
no damage, lifetime, area footprint, native save form or hashed simulation field.
An empty `CellEffect.Cells` produces no wall sprites. The damage scan is not a
render footprint.

## As built

- `game.LoadUnits` retains registry `Z`. `MapEntity.DrawCategory` reads the
  placed class and corpse stage before composed/corpse art substitution.
  Nonzero `Z` selects air at every stage; `Z=96` adds no pixel lift.
- Structure shadows precede flat structure bodies. Alternate units draw in
  their early cell phase. The main sweep orders each cell as non-flat structure,
  ordinary unit shadow/body, static shadow/body, Fire/Earth. Cells run by row
  ascending and column descending. Actor-local marks stay with their body.
- Air shadows, the native projectile list, Freezing/Poison cells and air bodies
  follow as separate phases. Projectiles retain input order. Inspection reads
  the same category-sorted opaque bodies as painting. Command selection keeps
  its existing ground-area policy.
- Existing anchors, palettes, source alpha, shadow transforms, rectangle culls
  and fog admission remain intact. `0068` FR5's equal-row unit-last clause was
  an implementation design; its cross-row intent and AC6 remain tested in the
  actual `drawArt` path. Native corpse/sack/live ties remain explicit policy.

## Proof and debt

The production mission101 crop changes from tree-over-wall to wall-over-tree;
both lawful installs retain ground point `(720,620)` and the same rectangles.
Installed-art tests compare Earth and Fire opaque/transparent overlap, clipping,
earlier/later rows and columns, and all three actor categories. Synthetic
composites preserve actor/tree/building depth and corpse/sack/live ties.
Exact counts and commands are in [verification.md](verification.md).

`DIV-528` now records the narrowed projectile/shower remainder. `DIV-1141`
through `1143` retain native sack/corpse, occupancy/lifetime and client-input
policies. No native original-game pixel equivalence or new lifecycle is claimed.
This candidate includes accepted1164 main
`92e338c3df704130989a2fecec0a91fc04cdab05`. The seat owns the sole independent
review and final gate chain.
