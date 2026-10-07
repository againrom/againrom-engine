# Provenance — height-projected placements

`claims/retracted.md` was read first; re-read at `03a9448`, 2026-07-30. `TERR-SPR-041`'s
identification and, since, `TERR-SPR-048`'s unit-sprite pass and its placement clause are all
retracted at **High** — repaired in the FR-1 unit row and in *Open*, the only two places this
story cites them, and no contract here rests on either. What was withdrawn from `TERR-SPR-039`
is the universal "the two
altitude models disagree by construction on any sloped cell", quantified in *Open* below; this
story's tie-break question is a separate, Medium-graded half of that row, not what was retracted.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| FR-1 — a marker's anchor cell is lifted by the mean of its four corner heights, truncated toward zero, the same subtraction sign the terrain mesh already uses | `TERR-GEOM-031` (amended), `TERR-SPR-039` (amended) | High — the mean-of-four build (`CDQ; AND 3; ADD; SAR 2` over four `MOVSX` corner reads) and the index algebra tying it to the anchor's own cell, read instruction by instruction. Medium only for the cross-check that this grid is what an object sprite is lifted by — not depended on here, since this story lifts a diagnostic cross, not sprite art |
| FR-1 — an out-of-grid corner index clamps to the nearest valid vertex, never reads zero | `TERR-EDGE-024` | High, carried unchanged from 0012 (`Projection.Altitude`'s existing clamp; our own port choice for a region the engine addresses raw and out of bounds) |
| FR-3/FR-4 — object and unit anchors are unsigned 8.8 `/256` fixed-point cell coordinates, `X>>8`/`Y>>8` | `ALM-OBJ-019`, `ALM-UNIT-018` (both amended by EXP-0030's frame correction) | High for the stride/count/coordinate triple (38/38 corpus maps, every anchor in-bounds); unaffected by this story. `ALM-OBJ-019`'s `+0x0c` gloss has since been withdrawn at Medium — a shop's stock cap (`SHOP-CAP-004`), not a field read here |
| FR-2, FR-5 — draw order terrain → objects → units, later marker wins on overlap; `Mode()`'s altitude-validity half; the camera/culling contract markers must respect | 0008, 0009, 0012, 0013 | carried unchanged, backed there |
| FR-3/FR-4 — a marker's anchor is its cell centre; the reference pixel is `(col*32+16, row*32+16 − alt)` | `TERR-SPR-040` | High — six named instructions per axis; alternatives (frame top-left, canvas centre) excluded by which class fields appear, not by a fit. Our own `AnchorCell`/`ObjectMarkerRects`/`UnitMarkerRects` (`pkg/render/terrain/overlay.go`, called from `cmd/terraintool/main.go`/`cmd/mapview/main.go`) already collapse both kinds to that pixel — port evidence, not the claim's basis |
| FR-1 — the cell-mean lift on **unit** markers; the engine's own unit rule is raw-position | `TERR-SPR-041` (retracted), `TERR-SPR-065`, `TERR-SPR-067` | Two backings **retracted at High**: `TERR-SPR-041`'s `vt+0x30` draws the HP/mana bars, and `TERR-SPR-048`'s `vt+0x2c` is the unit's **shadow**. The body draw is `vt+0x28` = `R0552`: it ignores arguments 1–2, reads the third (a light level), and places from its own `+0x60`/`+0x64`, minus the sprite anchor, minus `+0x10`, minus `+0x68` — still the raw-position model the old row said "never" of, filed below as the unresolved rival. The cell mean on unit markers is ours by choice, not decoded |

## Ours by choice

| What the spec fixes | What the evidence actually says |
|---|---|
| The height feeds a translated diagnostic cross, never real sprite art with its own internal vertical anchor | The engine's own lift formula has a further `anchorY` term for actual sprite art (`TERR-SPR-039`); our markers have no art or footprint, so that term is dropped outright — a scope choice, not a further decoded fact |
| `AnchorHeight` uses Go's native truncating integer division for the sum of four corners | The engine's own tie-break for a negative sum (`round4` vs. a bare `SAR 2`) is undecidable from the shipped corpus — no height byte reaches `0x80` in any of the 38 maps (`TERR-LIGHT-028`(d)); Go's plain `/4` reproduces the named instruction sequence for every value either reading could ever differ on, so the open tie-break is not exposed by anything this story ships |
| The offset is applied as a whole-rectangle translation after the existing flat geometry is built, not folded into `ObjectMarkerRects`/`UnitMarkerRects` themselves | Either factoring is behaviourally identical; keeping those two functions unchanged avoids widening an already-tested contract (0008/0009's no-floating-point, no-allocation scope) |
| `Mode()` drops its overlay check outright rather than narrowing it | Once `AnchorHeight` is total over any valid projection, no case remains where one cannot serve a marker, so the check has nothing left to guard |

## Open / undecoded

| What | State |
|---|---|
| The engine's exact tie-break for a negative corner-sum mean (`round4` vs. bare `SAR 2`) | Undecidable from the shipped corpus — no height byte reaches `0x80` in any of the 38 maps (`TERR-LIGHT-028`(d)). Inherited open, not reopened; this story's own rounding is disclosed above and does not depend on resolving it |
| Whether the engine's two altitude models (terrain mesh, sprite mean) reconcile beyond a minority of cells | `TERR-SPR-039`: they agree on 12,513/135,129 sloped cells (9.26%), diverging up to 73 rows on the rest. Carried as a disclosed non-claim of pixel fidelity, not resolved here |
| Whether a unit's sub-cell `/256` low byte reaches its destination pixel | Narrowed: the "second, unresolved model" WAS the unit draw (`TERR-SPR-065`/`067` — see Backing), and it reads neither `+0x08`/`+0x0c`; open is how `+0x60`/`+0x64` are maintained. This story asserts nothing either way — both overlays discard the fraction via `AnchorCell` first |
| Real object/unit art, footprint and per-corner anchor conventions | Still downstream of the class registries and static-data formats (0008/0009's own open scope); unaffected by this story |

## Removed and why

Deleted outright, not annotated, under golden rule 4 and because several were simply untrue of this
tree — a labelled hypothesis still steers the next search:

- the `heightAt=0` external-vertex convention — `Projection.Altitude` clamps instead, a 0012 choice;
- the structure-vs-unit dual anchor rule (cell-centre vs. raw 8.8) — both overlays already share one
  cell-centre anchor (now also decoded, `TERR-SPR-040`); no "structure" concept exists in code, only
  "objects" (0008);
- the structure "footprint outline" rectangle — no such geometry exists; 0008 draws a fixed cross;
- `cmd/almview`'s "-height combined with -structures/-units" rejection — no such command, flag, or
  rejection exists in this tree;
- `pkg/game.MapScene`, `StructureRects`/`UnitRects`, `SampleHeight8p8`, and the `roundDiv(dyNative*
  cellpx,32)` rule — `cellpx` here is always an exact integer multiple of the native cell.

Nothing here left a hole: the mean-of-four-corners formula, the far-edge clamp, and the shared
cell-centre anchor all come from claims or code this repo already ships against.

## Appended 2026-08-02 — pin `01c64e2`: `ALM-OBJ-019` is contested, and the clause FR-3/FR-4 cites is not the contested one

`ALM-OBJ-019` now reads `● active (amended, contested)` under new live contradiction **C-7**
(`claims/registry.md`): the row has the `.alm` type-4 `+0x12` word **sign-extended into
`obj+0x10`**, while `TERR-STRUCT-075` reads `obj+0x10` as a **pointer** to a 12-byte position object
(store `L02083`, dereferenced by both footprint routines). EXP-0081 read that store and both
dereferences but never `+0x12`'s own, so it **picked neither side**, lowered the destination clause
to Medium and marked both rows contested. The field's *role* — a type-7 `Target_Structure` id
(`ALM-TRIG-046`) — is untouched.

The Backing row above cites `ALM-OBJ-019` for the **stride, count and the `X`/`Y` coordinate
triple**, graded High on 38/38 maps with every anchor in bounds. `+0x12` is a different word, read
by nothing this story ships, and the contested fact concerns a live object this story never builds.
FR-3 and FR-4 are unaffected, exactly as that row already says of the `+0x0c` withdrawal beside it.
Recorded because a *contested* row is easy to read as a weakened row: here the weakening lands on a
clause this story does not use.

## Appended 2026-08-02 — pin `acb8fb0`: `TERR-STRUCT-075`'s Medium clause is superseded upward, and the anchor axis is now read from the image

The note above cites `TERR-STRUCT-075` for reading `obj+0x10` as a **pointer**. That half was always
High and is untouched. What moved is the row's **Medium** half — which of `L02090`/`L02091` is
the low axis — now **High as amended by EXP-0091**, carrying a `claims/retracted.md` row classed
**SUPERSEDED**. Both of its old reasons are withdrawn and the answer they reached is confirmed:
`ALM-OBJ-061` reads the chain end to end, from the loader's two `LEA`s through the packed cell key,
and finds no permutation in it; the "0 of 17 057 footprint cells off-plane" statistic never
discriminated at all, because 37 of the 38 shipped maps are square. Re-made as connectivity the
corpus does separate the two — bridge joins 93 against 5, for the published reading.

**Nothing FR-3 or FR-4 rests on moves.** The Backing row cites `ALM-OBJ-019` for the stride, count
and the `X`/`Y` triple, and `ALM-OBJ-061` is that triple's axis assignment re-sourced from the
disassembly rather than from a corpus fit — the same answer, better carried. **C-7 is unchanged and
still live**, and the note above it governs.
