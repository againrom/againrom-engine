# Provenance — a cell for an anchor, a grid for a sheet, and a block addressed backwards

Pinned at research `acb8fb0` (`acb8fb085757f5436ebb5f737255fe1630662829`) and frozen there for the
story. Every row below is read at that pin; nothing ahead of it is cited.

`claims/retracted.md` and the standing-corrections table were read **first**. Four rows cited here
appear in one or the other, and each is cited as amended rather than as first published:

- **`REG-STR-040`** — partially retracted: its parenthetical about the *other two* registries' key
  bases was wrong. The structure figures cited below — 66 classes, the `1..66` domain, and the three
  tile-extent ranges — are the corrected ones and stand.
- **`ALM-OBJ-019`** — the "objects/structures" label is refuted: a type-4 record resolves against the
  structure roster alone. Cited only for the record's own field layout and the forward walk.
- **`TERR-SPR-040`** — a clause was corrected (which frame's size the anchor is built from). Cited
  here only as the path this story does **not** take.
- **`RES-CASE-036`** — the row that overturned an earlier claim that no whole-path case fold exists.
  Cited for the address rule, at its corrected reading.

**Threshold: Medium.** This story reaches no hashed simulation state — it produces pixels — so the
standing default applies. In the event every clause relied on is published **High** except the three
marked below, each of which is either out of scope or does not decide a byte here.

## Backing

| `spec.md` anchor | Claim | Confidence as published |
|---|---|---|
| *The placement* — a structure has **no anchor pixel and no canvas**; its anchor is the footprint's min-col/min-row **cell**; the destination of image cell `(k,c)` is that tile's top-left screen pixel less the altitude lift | `TERR-STRUCT-100` | High for the formula (every term a named instruction, the destination's meaning fixed by the blitter's own argument use) and for the cell anchor (one create instruction plus 3141/3141; the live rivals — an anchor pixel, or a selection/shadow key as origin — refuted by the *absence* of any such key read on a routine read end to end) |
| *The placement* — the anchor cell is the stored coordinates shifted right by 8, with no origin, inset or rounding | `ALM-PLACE-033` | High for the objects and the shift — a whole-binary scan of all 82 shift-by-8 sites finds no coordinate biased before or after |
| *The placement* — the key's domain is the structure registry's `ID`, and the **low byte** is what selects the class | `ALM-CLS-036`, `REG-KEY-044` | High — the low-byte store is a named instruction, and the registry's class array is subscripted by `ID` rather than by section index |
| *The placement*, *The art rectangle* — the record layout the key and the coordinates are read from, walked forward | `ALM-OBJ-019` | High for the layout and the walk at the amended offsets; the row's label is the retracted part and is not used |
| *The art rectangle* — the drawable covers its whole `TileWidth x TileHeight` rectangle | `TERR-STRUCT-101` | High — the registration and its extent are named instructions in a routine read end to end |
| *The art rectangle* — the eight-byte extension exists only for the key `0x21` | `ALM-OBJ-034` | High — one named compare immediate, and the rule turns a partial record walk into a complete one |
| *The art rectangle* — the extension's two extents reach the **block** footprint bytes, not the class rectangle | `ALM-OBJ-062` | High for the six named stores and the two push orders connecting them / **Medium** for which extent is which, the corpus separating the readings only three times — and neither grade decides anything here, since the one key that carries an extension is a class this contract excludes |
| *The sheet and its grid* — `File` is a **path**, not an index; the registry has no file table and no inheritance; the sheet is `<path>.256` under the structures directory | `REG-STR-081`, `REG-STR-080` | High — two named suffix pushes in one routine, 66/66 resolving, and the absent `Parent` established over a routine read end to end rather than by a sweep |
| *The sheet and its grid* — the address is folded lower-case as a **whole path** | `RES-CASE-036` | High — the folding routine is named, gated on a flag with one image-wide reference, and 12 of the 66 classes spell `File` with a capital no archive node carries |
| *The sheet and its grid* — a row-major grid of `TileWidth` columns by `FullHeight` rows; index `k*TileWidth + c`; every frame `32x32` on 130 of 130 shipped sheets | `SPR256-STR-040` | High — the stride is a named multiply-and-add, and 130/130 uniform `32x32` is a census with no model interposed |
| *The sheet and its grid* — that uniformity is what makes an anchor pixel unnecessary, and is **not** the rule for sheets generally | `SPR256-FRAME-023` | High — a measurement over every `.256` in the install: 1253 of 1384 carry one frame size and 131 do not |
| *The strip* — one draw per rectangle cell, each a vertical strip; `rowTop`, the `ROW0 == 0` arm, the `-32` step, and the overhang | `TERR-STRUCT-101` | High — the loop bounds, the stride and the special case are named instructions; the rival "one image" dies on the uniform frame size and the rival "one frame per footprint cell" on the `limit = 0` arm and on 28 classes with an overhang |
| *The three blocks* — base, animation and a ruin grid addressed **from the end of the file**; the identity `frames = k*TW*FH + P*L`, `k` in `{1,2}`, on 64 of 66 classes with both misses predicted | `SPR256-STR-041` | High — all three bases and both gates are named instructions, and the identity re-executes over the shipped registry and sheets with the two exceptions named in advance |
| *The three blocks* — the three arms and their order, the destroyed test conjoined with `Indestructible`, and the base fallback for a retired cell | `TERR-STRUCT-102` | High — both gates and all three frame expressions are named instructions |
| *Animation* — the preconditions: `Phases > 1` gating the three animation keys, and the run-length expansion rule | `REG-STR-080`, `REG-OBJ-046` | High — the gate is one named compare-and-branch, and the expansion is the same loop the object registry publishes, read instruction by instruction |
| *Animation* — `AnimMask` is a picture of the **sheet grid**, its length exactly `FullHeight * TileWidth`, `-` meaning *never animates*, and the live count and rank | `REG-STR-082` | High — the buffer size is one multiply-and-increment pair, the live count one compare, and the corpus separates the only live rival 14/14 against 8/14 |
| *Animation* — a phase of 0 draws base, and an empty timeline never dereferences the mask | `SPR256-STR-041`, `REG-STR-082` | High — ten classes spell `Phases > 1` and then spell no timeline at all, so the phase scalar alone does not mean animated |
| *The lift* — a **bilinear** sample of the cell's four corner heights at the object's sub-cell position, the position being the footprint centre | `TERR-STRUCT-106` | High for the arithmetic — four sign-extended loads and three interpolations, read instruction by instruction / **Medium** for the consequence that a structure always samples a cell centre, which follows from the corpus and the centre term rather than from a test in the code |
| *The lift* — the lift is subtracted, and the displacement is vertical only | `TERR-SPR-038` | High — the destination is a named shift-add-subtract pair, and no sprite blitter indexes the geometry step table |
| FR-6 — draw order is the cell walk, rows ascending and columns descending, split by `Flat` into an early pass and a main pass | `TERR-STRUCT-104` | High — both gates are the same displacement tested with opposite senses in two passes of one routine |
| FR-1 — `VariableSize` means a separate class with its own selector rather than a data flag | `TERR-STRUCT-105` | High — two allocation sizes, two constructor call sites, two vtable installs and two selectors read whole, with the 9 and 14 frame counts matching the selectors' index ranges exactly |
| *Out of scope* — a structure's world sprite reads no owner: there is no team tint, no per-owner palette and no owner-selected shade | `TERR-STRUCT-107` | High — an enumeration over the class's complete method surface, with the one per-owner slot identified as the minimap blip |

## Ours by choice

| Decision | Why it is ours |
|---|---|
| The phase is `timeline[counter mod len(timeline)]`, advancing every animating structure together | The original advances a per-drawable phase gated on the cell being in the local player's sight (`TERR-STRUCT-102`, `TERR-TILE-079`). This tree has no fog model, so the gate has no input; ungated, every structure advances on one clock, which is what the original does for every structure that *is* in sight. No stagger is invented — unlike the object plane, the structure path carries no positional term |
| The non-flat pass is merged with the static-object plane in rectangle-row order | The original's cell walk reads the structure plane and the unit plane; nothing at this pin places the type-3 object plane in that walk. Both of our lists are cell-anchored and row-sorted, so a row merge is the arrangement that preserves back-to-front order between them; any fixed order puts one plane permanently in front regardless of row |
| Structure art draws before every unit and entity | The original interleaves the two planes cell by cell. This tree draws entities in a later pass, and that is already true of object art, so reproducing the interleave is a change to three other stories' contracts rather than to this one |
| Destruction is a draw-time diagnostic | The ruin arm's live input is a health value filled from a create message and a definition-table row (`TERR-STRUCT-102`); nothing in this tree gives a structure health. A switch makes the decoded addressing reviewable without inventing the state |
| The lift's three interpolations truncate toward zero, and `originY` joins the subtraction | The rounding direction of the intermediates is not stated by the claim, and `originY` is this tree's own vertical origin with no counterpart in the original. Both are chosen to match the object path already in the tree, so one map's two sprite families cannot round apart |
| A frame that is not `CellSize` square excludes its class | The original's arithmetic assumes a frame is a tile and has no defined behaviour otherwise. Refusing is the answer that adds no picture nobody asked for; 130/130 shipped sheets make it unreachable on lawful data |
| The sprite is lit by this tree's own one-row-per-frame ladder | The original passes a per-cell light byte into a lit blit family (`TERR-LIGHT-059`). The tree's sprite lighting contract is already one row from the sun's ambient, and this story inherits it rather than opening it |

## Open

- **The nine-patch frame mapping** of the two `VariableSize` bridge classes. The selector, its two
  frame counts and its inputs are established (`TERR-STRUCT-105`); which frame is corner, which edge
  and which interior is not published. `spec.md` R-1; the two classes draw nothing until it is.
- **Whether the registry rectangle and the definition table's `sizeX`/`sizeY` agree.**
  `DAT-BLD-005` establishes that the two tables are one roster re-keyed and that their *names* agree
  66 for 66, at High for the index law; it asserts nothing about the extents, and its footprint-mask
  clause is itself **Medium**. This contract reads only the registry, so the question decides nothing
  here; AC-10 records the disagreement count as an observation rather than reconciling it.
- **The `b` overlay's blend rule.** `SPR256-OVL-014` is High for the measurements and **Medium** for
  the composite, with the consuming instructions located and not read. Out of scope; a guessed blend
  would put an invented composite on shipped art.
- **The registry's unset-scalar defaults are decoded and this tree does not apply them.**
  `REG-STR-080` publishes them at High — `-1` through the selection band, `0` for `ShadowY` and the
  flags — while `pkg/data` carries a nil default table under a comment saying they are not decoded.
  The repair belongs to another story. FR-9 and P-8 make this contract indifferent: every scalar read
  here decides the same at both values. The row's flag count is stated as five where it lists six
  fields; the ambiguity is not resolved here and cannot reach this story, both candidate values for
  every one of them being `0`.

## Removed

| Statement dropped | Why |
|---|---|
| The shadow pass, and `ShadowY` as the height at which its shear vanishes (`TERR-STRUCT-103`, High) | Decoded and believed, but nothing in this tree casts a sprite shadow, so drawing one is a story about shadows. Recorded here so the key's meaning is not lost with the scope |
| The animation phase's visibility gate and the main pass's `+0x78` skip (`TERR-STRUCT-104`, `TERR-TILE-079`, both High) | Both need a fog model this tree does not have. Named as divergences in the contract rather than silently unimplemented |
| That a unit's animation is tick-driven where a structure's is not (`ANIM-CLOCK-001`) | True and relevant to why no stagger is invented, but it decides nothing a builder builds; it survives as the *Ours by choice* row on the phase |
| The object and unit anchor formula (`TERR-SPR-040`, High, one clause amended) | Cited only to establish that the structure path is not it. Carried in `analysis.md`, where the question of reuse belongs |
