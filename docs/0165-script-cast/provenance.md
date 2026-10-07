# 0165-script-cast — provenance

## The claims this story is built on

Every row cites a research claim id at the pin `aae16ee`, read through `research/tools/claim`.

| Claim | Grade | What the spec takes from it |
|---|---|---|
| `TRIG-CAST-033` | High | Instants 21 and 24 build a temporary casting actor; 21 casts at a cell (state `0xe`), 24 at a unit (state `0xd`). Argument order `(fromX, fromY, toX, toY, spell, power)` for 21 and `(fromX, fromY, Target_Unit, spell, power)` for 24. The actor is appended to a list and walked on later ticks; the transient actor is not one of the original save format's serialized lists, so a consumer must hash it while it is live. |
| `TRIG-CASTACTOR-044` | High for the field writes and the two-caller enumeration; High for the corpus | The constructor is one routine with exactly two callers, the helpers of instants 21 and 24. It places the actor at the byte-truncated source cell, writes the destination as two bytes, builds the spell object from `(u8)spell` and stores the power as a **word**. A power of 0 is replaced by the immediate `0x63` = 99. 35 authored instant-21 nodes over 6 maps, and `Power/skill` is 0 on all 70 nodes across both roots. |
| `TRIG-CELLEFFECT-045` | High for the key arithmetic, the six slots, the comparison width and the absence of any send; High for the corpus | Instant 29's cell key is `(u16)(((u16)p1 << 8) + (u16)p0)` — a 16-bit **ADD**. The arm walks **six** consecutive effect pointers on the matched cell record and, for each non-null one whose id byte equals the **full 32-bit** `p2`, writes `(u16)p3` to the effect's lifetime field. It does not stop at the first match and it sends no packet. 23 authored nodes over 3 maps, all trigger-referenced, `Duration` in {1, 30000, 60000}, and every node's cell is an instant-21 destination on the same map. |
| `TRIG-INSTCENSUS-046` | High for the widths and bounds; Medium for the counts and the once-flag universal | Every parameter of these opcodes is a plain int; no string, item or building parameter appears. Every trigger naming one is `once = 1`. The cell key is 16 bits with x in the low byte, so the addressable space is 256 × 256 and the largest shipped campaign map is exactly that. The lifetime field is a word, so a duration above 65535 truncates; the largest shipped duration is 60000. |
| `MAGIC-SHAPE-008` | High for the call sites and the two constructors; the `+0x4c` clause partially retracted and **superseded** | `getParam(row, 8)`, title 9 `Distribution system`, decides the shape at `L05102`: value 1 builds a `PointEffect` and **requires a target unit**, printing an error when there is none; anything else builds an `AreaEffect` sized by slot 9 and timed by slot 11, title 12 `Area Effect Duaration`. |
| `MAGIC-SHAPE-008`'s retraction row, carried forward in `MAGIC-CEIL-013` | High | The corrected lifetime is `effect+0x4c = (AreaEffectDuaration << 4) + (power << 4)/10`, the first term being the column value and not the distribution value. The power term is added only when the power is non-zero. The withdrawn form used `distribution << 4`; **spec FR-4 uses the corrected expression**, and this file names the retraction so the withdrawn one cannot come back through a quotation. |
| `MAGIC-BURST-031` | High for the routing; Medium for the 18/10 split | The `AreaEffect` set is exactly the 10 shipped spells whose `Distribution system` is not 1: 2, 3, 4, 7, 8, 9, 12, 17, 19, 21. The burst picture is that class's own vtable slot, which spec SC-2 cuts. |
| `MAGIC-CEIL-013` | High for the bounds; amended | The power is consumed in thirteen distinct expressions, five of them `pow(1.025, p)` durations on individual spells. That enumeration is why spec SC-1 cuts the per-spell point-effect arms rather than approximating them. |
| `MAGIC-ATTACH-016` | High | The no-stack, annihilate and self-healing-mask rules govern the **actor's** attached-effect list, not a cell's. Spec SC-1 and SC-5 both rest on this being a different list from the one FR-5 builds. |
| `MAGIC-TARGET-017`, through `pkg/data/spell.go` | already in tree | `Spell Target`, slot 4, is a **different column** from `Distribution system`, slot 8, and the two disagree on shipped rows: Shield has slot 4 = 2 and slot 8 = 1. FR-7 therefore adds a field rather than reusing `TargetsUnit`. |

## What the shipped corpus says, measured in this worktree

Measured with a throwaway program over the 28 campaign maps of the EN preserved root, using this
tree's own `mapload.CompileScript`. The program was deleted before the first commit; the numbers are
reproduced by the milestone census, which is committed.

- **Instant 21, 35 nodes.** Spell ids 19 (23 nodes), 21 (5), 3 (3), 8 (3), 17 (1). Every one of the
  five is in `MAGIC-BURST-031`'s AreaEffect set. Every node's `Power/skill` is 0. This matches
  `TRIG-CASTACTOR-044`'s independently measured distribution exactly.
- **Instant 24, 20 nodes.** Spell ids 20 (Stone Curse), 23 (Bless), 5, 10, 16, 22 (the four
  Protections). Every one has `Distribution system` = 1, so every one is a point effect. Powers are
  0, 1, 99 and 100 — unlike instant 21, instant 24 authors non-zero powers.
- **Instant 29, 23 nodes.** Third parameter is 19 or 3 on every node; durations are 1, 30000 and
  60000. Every cell is an instant-21 destination on the same map.

Read against the shipped `Spells` collection: slot 8 is 1 on 18 rows and one of {3, 4, 5} on the ten
`MAGIC-BURST-031` names, agreeing with that claim row for row. Slot 11 is 15 on spells 3, 7, 8 and
19, 20 on spells 12 and 17, 10 on spell 21, and 0 on 2, 4 and 9.

## Inferences, and how far they are held

**The remaining lifetime counts down (FR-5).** No claim in this pin reads the area effect's own tick
routine. What is decoded is that the field is a lifetime **in ticks** and that instant 29 writes it.
The shipped data is what carries the inference: on maps 91, 101 and 131 each cell is written twice by
two different triggers, once with 30000 or 60000 and once with **1**. A field that does not count
down makes the value 1 meaningless; a field that does makes it "take this wall down now", which is
what the surrounding trigger structure reads as. Held as **inferred**, not decoded. A rate other than
one per tick would change how long a wall stands and is not claimed here.

**A pending cast resolves on the next tick (SC-3).** Decoded: the actor is appended to a list and
walked by a later tick, so the latency is at least one tick. Not decoded: the cast-time countdown's
own length. One tick is the machine's minimum and is what this build does.

**A cast that finds no target is dropped (SC-4).** Decoded: a cast returning zero retries. Not
decoded: what makes it return zero. Dropping is the conservative choice — it cannot leave a record
alive forever on state the byte form would then have to carry.

## What the original save format does and does not carry

`TRIG-CAST-033` states that the transient actor is not one of the original format's serialized
top-level lists, while attached effects do serialize. This build's byte form is its own and must
round-trip the live world, so FR-8 carries both. That is a divergence in provenance only: it does not
change how any shipped file is read or written, and nothing in this story reads or writes an original
save.

## Customisation limits this decode implies (G2)

Each is an engine width or an engine bound rather than a stored field, so raising any of them changes
no byte of any shipped file. `TRIG-INSTCENSUS-046` states the first three.

- The cell key is 16 bits with x in the low byte, so an area effect can stand only within a
  **256 × 256** space. The largest shipped campaign map is exactly 256 × 256.
- A remaining lifetime is a **word**: a duration above 65535 truncates. The largest shipped duration
  is 60000.
- A cast's power is a **word**, and the spell id reaches the constructor truncated to a **byte**.
- At most **six** effects stand on one cell. No shipped map reaches two.
- Instant 29's spell comparison is against a full dword, so authored values of 256 or more can never
  match. This is a limit of the arm and not of the data.
