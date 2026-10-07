# Spec — a creature's tier is its own colours

## Problem and current behaviour

A creature class ships in up to four **tiers** — a class and its second, third and fourth versions —
and the game shows each in different colours. This build draws every tier of a class identically,
because a drawn frame carries exactly one colour table: the one embedded in the sprite sheet the
class names. Nothing anywhere in the tree reads a `.pal` entry, and no class knows it has tiers.

Three things stand between a placed creature and its own colours:

- The colour tables are **shipped files** in the graphics container, in the sprite sheet's own
  directory, and no decoder for that file exists here.
- Which file belongs to which tier is a **name nobody stores** — it is built from the sheet's path
  and the tier number.
- Which tier a placement is is a **column of the definition table**, already parsed into a resolved
  definition field and read by nothing.

The unit draw itself needs no new path. A frame is drawn through a sixteen-row ramp built from that
frame's own palette, and a tier changes which colour table goes into that ramp — nothing else.

## Terms

A **tier** is a numbered version of a creature class, counting from 1; tier 1 is the class as its
sheet ships. A class's **tier count** is how many it has. A **colour table** is 256 colours a
frame's pixel indices select in. A **placement** is one unit record of a map; it **resolves** to at
most one **entry** of the definition table, through one of four arms, only one of which reaches a
stat-bearing entry. A **sprite entry** is the container address a class's sheet sits at.

## Functional requirements

### The file

- **FR-1 — a colour table is 256 four-byte entries at a fixed offset.** A decoder MUST take a byte
  stream and answer 256 colours, each read as four bytes `[B, G, R, X]` beginning at offset `0x36`.
  The fourth byte of an entry MUST NOT be read, and no entry's colour may be altered, reordered or
  reserved a meaning — index 0 included.

- **FR-1a — two refusals, and both name their reason.** The decoder MUST refuse a stream shorter
  than `0x436` bytes, and MUST refuse one whose first two bytes are not `B` then `M`. It MUST be
  total on every other input: no input may panic it, and it MUST NOT consult a length, a width or
  any other field of the stream to decide what to read.

### Which file, and how many

- **FR-2 — the name is built, never stored.** For a class that resolves a sprite entry, and a tier
  N, the tier's colour table is the entry at: that sprite entry's **directory**, then `palette`,
  then N in decimal **only where N is above 1**, then `.pal`. So tier 1 is `palette.pal` and tier 3
  is `palette3.pal`, in the sheet's own directory. A class resolving no sprite entry, and any
  N below 1, MUST answer the empty address — the answer the class's sprite address already gives.

- **FR-3 — the tier count is the registry's own count, bounded by one named limit.** A class's
  tier count MUST be its `Palette` registry key, clamped into `[0, L]` where **L is 4** and is a
  single named constant. A key at or below 0 is a class with no tiers. Exceeding L MUST NOT be an
  error and MUST NOT refuse the class. **L is a customisation seam:** it is a compiled constant
  here and lifting it changes no shipped byte of ours, but a class cannot gain a tier without a new
  shipped colour-table entry beside its sheet, a larger `Palette` in the shipped registry, and a
  further definition-table row carrying that tier's own column value.

### The colours

- **FR-4 — a tier's colours come from that tier's file and from nothing else.** No transform of one
  tier's colours may stand in for reading another's — not a per-channel scale and offset, not a
  general matrix with offset, not a hue rotation, not an index remap — and no such transform may
  exist anywhere on this path. The only arithmetic a tier's colours may pass through is the shading
  ramp every sprite in the build already passes through, unchanged, with the same arguments.

- **FR-5 — a class carries one frame slice per tier, over the sheet's own pixels.** For each tier
  its class has, the class MUST carry that sheet's every frame resolved through that tier's colour
  table. Each such frame MUST carry the sheet's own pixels **shared, not copied**, and nothing may
  write through them. Where a tier's colour table equals the sheet's own, that tier's frames MUST
  be the sheet's own frames — **the same objects** — so one picture never has two identities.

- **FR-6 — a tier that cannot be built is a skip, never an error.** An absent entry, a stream the
  decoder refuses and a class whose sheet did not load each leave that tier with no frames of its
  own, and a class in that state MUST draw its sheet's own colours. No load may fail on any of the
  three, and the count of them MUST be reportable.

### Which tier a placement is

- **FR-7 — a placement's tier is its resolved entry's own tier column.** Where a placement resolves
  to a definition-table entry through the arm that carries stats, the tier it is drawn in MUST be
  that entry's tier column. A placement taking any other arm, one reaching no entry, and a map
  opened with no definition table MUST state **no tier**.

- **FR-8 — the selection is total, and out of range is the sheet's own colours.** Drawing a class
  at a tier below 1, above that class's tier count, or at one the class could not build MUST draw
  the sheet's own frames. There MUST be no input at which selecting frames for a tier fails.

- **FR-9 — the tier is presentation and reaches no simulation state.** It MUST NOT be a field of a
  world, MUST NOT enter a world's byte form or its digest, and MUST NOT change any placement's
  cell, health, domain, route or order. It is decided once, when a map opens, from that map and the
  definition table alone.

### On screen

- **FR-10 — the game draws it, and there is no switch.** In the game front-end a placed creature of
  tier N MUST be drawn in tier N's colours, with no flag, option or default anywhere on the path —
  a creature's colours are the map's content, exactly as the structure and object art are.

- **FR-11 — the drawn class is the class whose colours are used.** Where a class's art is
  substituted for drawing, the tier MUST be applied to the substituted class's own tiers under
  FR-8, so no substitution can draw one class's pixels through another class's colour table.

- **FR-12 — a developer instrument reports the corpus and renders the still.** A developer tool MUST,
  against a lawful install whose root comes from a flag or the environment: report per class its
  tier count, how many tiers loaded, and how many fell back under FR-6, with totals; and write a
  PNG showing one named class's tiers side by side, one panel per tier in ascending order, outside
  both repositories. Neither output may be an error on any count.

## Acceptance criteria

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | A synthetic stream of `0x436` bytes with `BM` and a known table | decoded | the 256 colours are the bytes at `0x36` read `[B,G,R,X]`, `X` dropped, entry 0 included |
| AC-2 | Streams of `0x435` bytes, and of `0x436` bytes not starting `BM`, and the empty stream | decoded | each is refused with a reason naming it, and none panics |
| AC-3 | A class resolving a sprite entry, and one resolving none | asked for tiers 1..4 and for 0 and -1 | the four built addresses are `palette.pal`, `palette2.pal`, `palette3.pal`, `palette4.pal` in the sheet's directory; 0, -1 and the entry-less class answer the empty address |
| AC-4 | Classes whose key is -1, 0, 1, 4 and 7 | asked their tier count | 0, 0, 1, 4 and 4 |
| AC-5 | A synthetic container: a sheet, `palette.pal` equal to the sheet's own table, `palette2.pal` differing, `palette3.pal` absent, `palette4.pal` refused | loaded | tier 1's frames are the sheet's own slice by identity; tier 2's frames carry the second table with the sheet's pixel values; tiers 3 and 4 fall back to the sheet's own frames and are counted as fallbacks |
| AC-6 | The bundle of AC-5 | asked for tiers -1, 0, 2, 5 | -1, 0 and 5 answer the sheet's own frames; 2 answers tier 2's |
| AC-7 | A synthetic map and table placing one class twice with different tier columns, and once through a non-stat arm | opened and pushed to the draw seam | the two carry frames with different colour tables and equal pixel values; the third states no tier and carries the sheet's own frames |
| AC-8 | The map of AC-7 | opened, stepped and hashed | the world's entities, byte form and digest are identical to a build with no tier resolution at all |
| AC-9 | Each lawful install root | the instrument's census | every named colour-table entry exists and decodes, tier 1's table equals its sheet's own on every class that has one, the two roots agree row for row, and no class reports a fallback |
| AC-10 | A lawful install and a named tiered class | the instrument's still | a PNG with one panel per tier of that class, in ascending order, written outside both repositories |
| AC-11 | A class with no tiers, and the bundle of AC-5 | drawn | its frames are the ones it had before this story, by identity |

## Derived properties

- **P-1 — the base frames are never written.** A tier's frames share the sheet's pixels; no code on
  this path assigns through them, so drawing a tier cannot change any other tier's picture.
- **P-2 — equal colours are one identity.** Two tiers whose tables are equal are the same frame
  objects, so a consumer keyed on frame identity holds one entry for them.
- **P-3 — entry 0 is carried, not interpreted.** The transparent index is whatever the file holds
  and is written by nothing here; which pixels are holes stays the frame's own per-pixel answer.
- **P-4 — one map open, one tier decision.** Asked twice with no map change between, the tier of
  every placement is the same, and it is a function of the map and the table alone.
- **P-5 — a map with no tiered placement draws byte-identically to this build.**

## I/O examples

| Input | Output |
|---|---|
| sprite entry `units/monsters/goblin/sprites.256`, tier 1 | `units/monsters/goblin/palette.pal` |
| the same, tier 4 | `units/monsters/goblin/palette4.pal` |
| the same, tier 0 | `""` |
| `Palette = 4`, tier column 3 | the class's third frame slice |
| `Palette = 0`, tier column 3 | the sheet's own frames |
| a stream of `0x436` bytes beginning `00 00` | refused: not a colour-table file |

## Constraints

- No game asset, decoded or otherwise, may enter either repository or its history — a colour table
  is game data, and no table, entry or fragment of one may appear as a fixture, a test constant or
  a committed file. Fixtures are synthesised in test code.
- Every test runs green with no game install present, and none reads one.
- Every asset root comes from a flag or the environment; no shipped source names an install path.
- The still is written outside both repositories.
- The simulation tier is not touched.

## Out of scope

- **Team colouring** — the other branch of the same selection, which the 18 classes with no tiers
  take, drawing a unit in its owner's colours from a shared sixteen-table file of a different shape.
  Deferred as a named follow-up: the identification of which of the sixteen an owner takes is
  Medium in the research this build is entitled to use, and this tree's simulation carries no owner
  to subscript with. Those classes draw their sheet's own colours, as they do today.
- The shared sixteen-table file itself, and its greyscale seventeenth table.
- The sprite **shadow** pass, which this build does not have.
- The `palette_.pal` entry shipped beside every `palette.pal`, which matches no built name.
- Any change to the shading ramp, the blit, the anchor, the frame selection or the animation.
- Any tier fact reaching the simulation, a save, or a hash.

## Verification mapping

FR-1, FR-1a → AC-1, AC-2. FR-2 → AC-3. FR-3 → AC-4. FR-4 → AC-9 (the loaded table equals the
file's own bytes) and the absence of any transform on the path. FR-5 → AC-5, AC-11, P-1, P-2, P-3.
FR-6 → AC-5, AC-9. FR-7 → AC-7. FR-8 → AC-6. FR-9 → AC-8, P-4. FR-10 → AC-10 and the deliverable.
FR-11 → AC-7. FR-12 → AC-9, AC-10.

## Gate check

FR↔AC coverage complete; every current-behaviour claim above was read off the working tree; no
requirement names a package, type or function; no requirement contradicts Out of scope.
