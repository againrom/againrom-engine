# Analysis — the ALM type-3 static-object layer

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile's default for render work |
| Terrain — the object layer itself | **greenfield**: nothing reads `Map.Overlay` today |
| Terrain — `cmd/terraintool`'s compose path and `pkg/ui.Viewer`'s draw path | **brownfield**: a layer lands between the terrain raster and the 0008/0009 markers in each renderer, and a summary line three stories pin gains a token |

## The research pin, which moved

Opened at `e61153d`, bumped to **`9c01af7`** before a contract sentence was written. EXP-0038
published `TERR-SPR-042` (what reaches the draw call's frame argument), `TERR-SPR-043` (the anchor's
frame size differs between the shadow and the body pass), `TERR-TILE-044`, `SPR256-FRAME-023` and
`REG-OBJ-046`/`047` — all about this exact draw path. Against the old pin this spec would have
shipped two open research items that are now answered, and an anchor measured from the wrong frame.

## The merge

The owner folded `0018-interactive-objects` into this story, so the spec derives from both
baselines. 0018 says of itself that it adds **no new game facts**: it re-renders 0017's semantics
through Ebitengine, and the rest of it is render engineering. Split, the two shared one anchor
function — and 0018 made reusing it *unchanged* a derived property (its P-2) while its parameter
list cannot express the decoded formula, so the defect would have shipped in two renderers. Its
AC-8 likewise pinned the drawable count to the baseline prototype's `Cross 5871`, the same
prototype as the wrong formula. Merged, both become things this story owns and measures.

## What the baseline entries got wrong

`cleandocs/0017-map-objects/spec.md` was written against a tree this repo is not, and against
research that has since moved twice; 0018 inherits every one of those faults by reference.

1. **Almost no identifier exists here, and a flag name is taken in two tools.** `cmd/almview` is
   `cmd/terraintool render`, `cmd/openrom` is `cmd/againrom`, `pkg/game.MapScene` is
   `pkg/ui.Viewer`; `-cellpx 32` is `-scale 1`, `SampleHeight8p8` is `Projection.AnchorHeight`,
   `nativeMinY` is `Render.OriginY`. `-height` is inverted — displaced geometry is this repo's
   default, `-flat` the diagnostic (0012). And `-objects`, which 0018 wants for object art in
   `mapview`, is **taken in both front-ends**: 0008 gave it to the type-4 marker overlay, before
   EXP-0037 established that a type-4 record names a `structures.reg` class.
2. **The placement formula is wrong, in both.** It computes `destX = x*32 + 16 - CenterX`.
   `TERR-SPR-040` decodes `anchorX = (CenterX - Width/2) + frameWidth/2`: the frame is centred in
   the class's `Width x Height` canvas first, and only a frame that fills its canvas makes the two
   agree.
3. **Both research items are answered, and this story has none left.** `TERR-SPR-040` settles
   `CenterX`/`CenterY` as the ground-touching pixel at High — the alternatives excluded by which
   class fields the instructions read, not by a fit — and `TERR-SPR-042` settles the frame argument,
   richer than `Frames[Index]`: three arms and two gates, only the plain arm `Index` verbatim.
   Neither may be imported as an open `[R]` item, here or by reference.
4. **The provenance basis is weaker than what we hold.** It rests byte-to-class on a corpus
   histogram plus a wooded-map cross-check; `ALM-CLS-035` has it at High from three consuming sites
   in `rom.exe`, and records that corpus alone could *not* have carried it — `c - 1` scores
   99.9100 % in range against the rival `c`'s 99.8256 %, a 60-cell gap.

## What we measured ourselves

`cmd/classdump` (0016) and a throwaway probe run from outside the repo, both against the owner's
install with the asset root on the command line; output to screen only.

- **`CenterX == Width/2` on 81 of the 82 object classes** — so the anchor *nearly* reduces to
  `frameWidth/2`, and does not: `Object37` has `Width = 64`, `CenterX = 28`, and is placed on
  shipped maps. One counterexample is enough.
- **`frame[Index]` exactly fills its class canvas on 67 of the 75 classes whose art ships.** On the
  other eight the corrected anchor differs from the baseline's by up to **8 px** in one axis
  (`Object30`…`Object37`) — observable, not notional. Objects are kinder here than units, which
  `REG-VAL-043` measures at 1 exact fit in 18.
- **Cross has 5871 nonzero type-3 cells; all resolve, all are drawable.** 0018's pinned figure
  survives as our own measurement, and no artless class is placed there.
- **0 of the 40 900 nonzero cells on the ten loose maps set either of tile bits 15..14** — the gate
  on `TERR-SPR-042`'s animated arm, agreeing with `TERR-TILE-044`'s 0 of 880 704.
- **185 of those cells set tile-word bit 13, and 184 sit on a class naming a dead form** — the
  dead arm's whole reach on the loose corpus, 0.45 %.

## The defect this story stopped at

The dead-object arm cannot be implemented on `pkg/data` as it stands, and that is a defect of ours,
not a research gap. `REG-OBJ-047` measures `DeadObject` as **-1 on 54 classes** and a valid class
index on 28. Our loader resolves it to **-1 on 21 and `0` on 33**: 35 sections spell the key, 39
spell `Parent`, and where neither an own key nor an ancestor supplies one the field takes Go's zero.
`0` is a *valid* object id — `Object0`, `Pine 1` — so every stone, pointer, fence and bone class
currently claims a pine as its dead form, and a renderer gating on `DeadObject != -1` would put a
128x128 pine on 184 shipped cells. `FireObject` fails identically: `{-2 x21, -1 x7, 0 x54}` against
the claim's `{-2 x21, -1 x61}`. The engine's unset-scalar default is therefore **not uniformly
zero**, which `REG-OBJ-046` states for `File` (literal -1) and `InMapEditor` (0) but not key by key.
Correcting it is a `pkg/data` change, so the dead form is out of scope and the gap is counted rather
than half-built.

## What we looked at, and what stays unknown

`claims/retracted.md` first, then `claims/terrain.md` (`TERR-SPR-038`…`043`, `TERR-TILE-044`,
`TERR-GEOM-031`, `TERR-ANIM-009`), `claims/alm.md` (`ALM-CLS-035`, `ALM-CLS-042`), `claims/reg.md`
(`REG-OBJ-039`/`046`/`047`, `REG-VAL-043`), `claims/spr256.md` (`SPR256-FRAME-023`); in the tree,
`Projection.AnchorHeight`, `Render.OriginY`, `terrain.IsImpassable`, the stdlib-only
`pkg/render/terrain` DAG entry, `pkg/ui.Viewer`'s lazy tile GPU cache and marker transform, and
`spr256.Frame`'s structural transparency.

Unknown and disclosed rather than guessed: the sun shear the shadow pass subtracts from `dstX`
(named, never decoded — `TERR-SPR-038`) and what `[L04367]` gates, together why this story draws
one pass of four; whether anything ever sets tile bits 15..14, so whether a static object animates
at all (`TERR-TILE-044`); what the engine draws at the 33 cells placing a fire variant whose art
does not ship (`ALM-CLS-042`) and at the 64 indexing past the registry (`ALM-CLS-035`); and the
order the engine walks the grid in, which our painter's order only assumes.
