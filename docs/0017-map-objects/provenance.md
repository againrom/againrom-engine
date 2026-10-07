# Provenance — the ALM type-3 static-object layer

Pinned at research `9c01af7`, frozen for the story. The interactive half folded in from the 0018
baseline asserts **no game fact** — it re-renders these semantics — so every row below serves both
renderers. `claims/retracted.md` was read first; two rows bear here. `TERR-SPR-040`'s clause
"`frameWidth`/`frameHeight` … **with frame index 0**" was withdrawn at the **High** it held: true of
the block that was read (the shadow pass), false of the pass that draws the object. And
`ALM-CLS-042`'s scope was corrected — seven fire-variant classes ship no art, not four. Neither
touches the anchor **formula**.

Re-read against research `03a9448`, 2026-07-30. One row cited here has fallen since, in a clause
this story does not consume: `TERR-SPR-038`'s naming of the `vt+0x3c` push list's **fifth**
argument as a brightness `level` is retracted at the **High** it held — it is a 16.16 per-row X
slope, the shadow's sun shear, and the *fourth* argument is the shroud level (`TERR-SPR-066`,
High). What this story takes from that row is its **destination** arithmetic on the object path
(cell centre, altitude subtracted), which the retraction leaves standing. The shear itself is now
decoded, which narrows one *Ours by choice* row below rather than any contract.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| *Byte to class* — `0` is no object; `b` names the class whose `ID` is `b - 1` | `ALM-CLS-035`; `REG-KEY-044` (amended) for `ID == section index` on 82/82, which is why that claim's "section index" and our `ID` are one key | High — three consuming sites in `rom.exe` at instruction level, and the row records that corpus alone could not have carried it: `c - 1` scores 99.9100 % in range against the rival `c`'s 99.8256 %, a 60-cell gap |
| *Byte to class* — a code may index past the registry, and a resolved class may have no loadable sprite | `ALM-CLS-035`, `ALM-CLS-042` (amended) | **Unknown** whether the engine reads the 64 out-of-range cells, or what it draws at the 33 placing artless classes / High that the art is absent (every archive re-walked) / Medium that those 33 are the four named |
| *Sprite anchor* — `anchorX = (CenterX - Width/2) + frameW/2` and its Y twin, each division truncating toward zero | `TERR-SPR-040` (amended) | High — six named instructions per axis; the rivals (frame top-left, canvas centre) excluded by **which class fields the instructions read**, not by a fit |
| *Sprite anchor* — the frame whose size enters it is the one being drawn, and a differently-sized frame is a live case | `TERR-SPR-043`, `SPR256-FRAME-023` | High / High — both anchor blocks and both frame arguments are named instructions, the shadow/body split decided by which `dstX` subtracts the sun shear; and 131 of the install's 1384 `.256` sheets mix frame sizes, two of them object sheets |
| *Ground point* — the cell centre, altitude subtracted | `TERR-SPR-038` (fifth-argument clause retracted) | High — the destination arithmetic and the push order are named instructions, and the `SHL 5 ; ADD 0x10` idiom has one definition in the image. What fell in that row is the *name* of one push slot (`TERR-SPR-066`), not a term of this destination |
| The `-staticmarkers` instrument — that the decoded anchor deserves a second, independent derivation to check against | none; this is the gap, not a fact | The formula is High **as decoded** and has never been rendered on real data by us. Research's own render is marked a *display*, not a check — every term in it was `TERR-SPR-040` transcribed. Agreement between art and marker is the first evidence the transcription is right, which is why the two may not share code |
| *Ground point* — the lift is that cell's four corner altitudes, meaned, truncated toward zero | `TERR-SPR-039` (amended), `TERR-GEOM-031` (amended) | High for the build (30 instructions, every read `MOVSX`, the divisor an immediate) / Medium for the cross-check that it stands a sprite on drawn ground. Carried from 0015, whose `AnchorHeight` implements it |
| *Class to sprite frame* — the drawn frame is the class's `Index` | `TERR-SPR-042` (a) | High for the arm — a named `MOV` pair off the class record, in a function read end to end, its writer set enumerated. Its three other writers are gated; see below |
| *Class to sprite frame* — the sheet is `objects/` + `Files[File]` + `.256`, its own palette, structural transparency | `REG-VAL-029`, `REG-OBJ-046`; 0002/0016 for the decode and path | High for the `Files[File]` index; the path construction is ours, Medium in 0016 and unchanged |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| One draw per cell — the **body** pass of `sprites.256`; the shadow's sun shear is now decoded, what `[L04367]` gates is not | `TERR-SPR-043` decodes **four**. The shadow passes subtract a sun shear that `TERR-SPR-038` named and `TERR-SPR-066` has since read end to end — a 16.16 per-row X slope written once from an `__ftol` of a sun-angle expression, High. Drawing one pass stays this story's choice, now a disclosed omission rather than an undecoded one; the second pair is gated by `[L04367]`, whose pairing with `spritesb.256` is Medium. Three of the four are still not drawn |
| Row-major painter's order, a later row owning the overlap | No claim states the order the engine walks the grid in; ours, on back-to-front reasoning alone |
| The interactive layer's engineering — GPU-free bundle, build-once placements, exact-rectangle visibility, zoom scaling, lazy upload, and the canvas/camera clip | No claim bears on any of it and none could: it is our own rendering, not the original's |
| An unresolved code, an artless class, an out-of-range frame index: skipped, not failures | `ALM-CLS-042` states that a type-3 decoder has to tolerate a resolved class with no loadable sprite and grades what the engine draws there Unknown, as `ALM-CLS-035` does the 64 out-of-range cells. Skipping is the only behaviour neither contradicts |

## Open / undecoded

- **Whether a static object ever animates.** `TERR-SPR-042`'s animated arm is gated on the cell's
  four corner tile words ORing to `0xc000`; `TERR-TILE-044` finds 0 of 880 704 shipped cells setting
  either bit and names **no writer anywhere in the image**, publishing reachability Unknown. The
  global that forces frame 0 when animations are off (`TERR-ANIM-009`, amended) has no switch here.
- **The dead-object arm, blocked on us and not on research.** `REG-OBJ-047` measures `DeadObject` as
  -1 on 54 classes; `pkg/data` yields -1 on 21 and `0` on 33, and `0` is the valid id of `Object0`.
  `FireObject` fails the same way. The unset-scalar default is not uniformly zero — `REG-OBJ-046`
  gives literal -1 for `File`, 0 for `InMapEditor` — and 0016 never asked. 184 cells on the loose
  corpus would take the dead form.
- **`File` alone does not inherit** (`REG-OBJ-046`, one named immediate) and `pkg/data` inherits it.
  Unobservable on shipped data — 39 of 39 `Parent`-carrying object classes write their own `File` —
  but the sprite path rests on it.

## Removed from the baseline and why

- **`R-1` and `R-2` as research items**, in both baselines. Answered at High — the anchor by
  `TERR-SPR-040`, the drawn frame by `TERR-SPR-042`, both published after them. This story carries
  no research item at all.
- **The corroboration offered for each.** For `R-1`, index ranges matching sheet frame counts plus
  "a rendered prototype … looks visually correct across 5871 placements" — a coherence check a wrong
  rule passes too, from the prototype that carried the wrong formula. For `R-2`, "`CenterX` equals
  `Width/2` … in every sampled class" — 81 of 82, and never what the anchor rested on.
- **0018's `AC-8` pin to that prototype's `Cross 5871`.** Merged, the figure is ours: 5871 cells,
  all resolving, all drawable.
- **The placement formula `destX = x*32 + 16 - CenterX`,** and 0018's derived property that it is
  reused unchanged. Right only where a frame fills its canvas; wrong by 8 px on eight classes.

## Appended 2026-08-02 — pin `01c64e2`: `TERR-TILE-044`'s "no writer anywhere in the image" is refuted, and this story's own census is not

**The writer of tile bits 15..14 is found.** `TERR-TILE-044` stands at `● active (amended)`; what
`claims/retracted.md` classes **REFUTED** is its dichotomy — "either a writer exists that this
enumeration's shape cannot see (a non-immediate store), or the path is dead". Neither held. EXP-0084
finds an **immediate** store of the wrong *width*: `OR byte ptr [...],0xc0` on the word's high byte,
at `L02581`, `L02582`, `L02583`, `L02584` in `R0554` (`CUnit`/`CAirUnit` `vt+0x48`),
stamped on the gate's own four corners over a 41×41 window around every drawable, with bit 14
cleared map-wide every 32 ticks (`ANIM-TICK-011`). A scan for `0x4000`/`0x8000`/`0xc000` as 16-bit
operands could not see it.

**Nothing this story measured moves.** Both censuses are of shipped **file** bytes and the bits are
**runtime** state, so `TERR-TILE-044`'s 0 of 880 704 and this tree's own 0 of the 40 900 nonzero
cells on the ten loose maps stay exactly true, and the *Backing* row citing the four-corner gate is
cited for the gate's shape rather than for reachability.

**The *Open* item changes.** "Whether a static object ever animates" is recorded above as
`TERR-TILE-044` naming **no writer anywhere in the image**. That half is now false. The other half
survives, narrowed: whether the gate is ever *satisfied in play* is **unestablished** — the stamp's
two guards and its `CMapView+0x17cc` mask are unread — rather than Unknown for want of any mechanism
at all. This story draws one pass of four and gates no requirement on the answer.

`analysis.md` carries the same figure in its merge list and the same "whether anything ever sets
tile bits 15..14" in its closing paragraph. It sits **5 bytes** under its 7168-byte ceiling and
cannot carry a note of its own; this one covers it, and where the two disagree this one is right.
## Appended 2026-08-02 — pin `9ff259c`: the gate's remaining Unknown closes, and bits 15..14 are the fog of war

**`TERR-TILE-044`'s reachability `Unknown` is CLOSED to High.** The append above left half the
question open — whether the gate is ever *satisfied in play*, the stamp's guards and its
`CMapView+0x17cc` mask being unread. All three are read now and the answer is that it fires. The
guards are **three**, not two: a decay-stage test precedes the tail-jump nobody had followed. The
mask is not a shape at all but a **per-drawable line-of-sight field** the map view rebuilds on every
stamp. And the stamp's immediate is `0xc0` — **both** gate bits into one word — so a single stamped
cell already passes the four-corner OR. Bits 15..14 are the **fog of war**, three-state, which is
also why one reader tests `== 0x8000` before `== 0xc000` (`TERR-TILE-079`).

**Nothing this story measured moves, again.** `TERR-TILE-044`'s 0 of 880 704 and this tree's own 0 of
the 40 900 nonzero cells are censuses of shipped **file** bytes; the runtime grid is a different
object and neither census bounds it. The *Backing* row is still cited for the gate's shape.

**The *Open* item is now answered rather than narrowed.** "Whether a static object ever animates" has
an answer: on the cells the local player can currently see. This story draws one pass of four and
gates no requirement on it, so nothing here is wrong — but a reader should no longer take the item as
open. `analysis.md` remains 5 bytes under its ceiling and carries the old wording; the note above
covers it, and this one supersedes both where they disagree.
