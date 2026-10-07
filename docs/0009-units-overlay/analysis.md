# Analysis — placed-units diagnostic overlay (ROM1)

## Intensity & terrain (declared for the whole work item)

- **Intensity: `spec-first / static`.** Per `SDD/PROFILE.md`, a render/UI-tier diagnostic overlay is
  "other engine work" — a bounded delivery whose behavior settles quickly; upfront clarity helps but the
  contract will not keep evolving. It is not `spec-anchored`: nothing outside this repo consumes the
  marker geometry, no second team is aligned by it, and the format contract it rests on lives upstream in
  `pkg/formats/alm` (0003, which *is* spec-anchored) rather than here. It is not `no-spec` either: the
  glyph geometry, the clip order, the off-map rule and the composed-summary ordering are all choices a
  reader could not recover from the diff, and the story defines a contract two commands and one library
  tier must agree on. No watcher tool exists in this repo, so synchronization is **static** — discipline,
  not tooling — and is labelled that way.
  *Sibling check:* 0008 declared the same pair for the same reasons; 0009 is its twin for type-6 records,
  so the same call is the consistent one. This is close to the playbook's "a sibling already pins the
  pattern" step-*down* trigger, and the reason it does not fire is that 0009 is not a copy: it composes
  with 0008 (independent toggles, a fixed draw order, a fixed count order), and that composition is
  precisely what needs stating before the code exists.
- **Terrain: greenfield, per changed area.** The unit-marker geometry is entirely new code in
  `pkg/render/terrain`, and the viewer-side unit overlay is entirely new code in `pkg/ui`. The existing
  compositors (`Composite`, `CompositeLit`), the 0008 object overlay, and the viewer draw path are
  extended **additively and off-by-default**: with the unit overlay disabled the output is byte-for-byte
  the pre-0009 baseline (spec P-3), so no shipped library contract is silently changed and no brownfield
  characterization burden falls on those areas.
  **One qualification, recorded honestly:** if the plan chooses to *parameterise* 0008's shipped overlay
  primitives (see the composition baseline, item 6) rather than add parallel unit-named ones, that
  refactor touches already-shipped, already-tested behaviour and is **brownfield for
  `pkg/render/terrain/overlay.go` and `pkg/ui/overlay.go`** — the existing 0008 tests are then the
  characterization pin and MUST stay green unchanged. Which of the two it is, is a plan decision; the
  terrain declaration for that area follows from it.

## Source & confidence

All facts below are game-derived by the research team, not by this repo's own reverse-engineering.

- **Submodule pin:** `research/` at `da54e6d` (`againrom-research`, module `rom1research`) — research
  `master` HEAD at this story's boundary, and frozen for the story's duration (S-5).
- **Experiments:** `research/experiments/EXP-0019` (the variable-content sections type4–9, which first
  located the type-6 unit table) and `research/experiments/EXP-0030` (the corpus-wide container-framing
  correction), both promoted into `research/formats/alm/format.md`.

### The EXP-0030 amendment — what moved and what did not

EXP-0030 proved from `rom.exe` and the shipped bytes that the EXP-0007 section split was **8 bytes too
early**: `typeId` and the per-map `f32` are the last two words of a record's 20-byte *header*, not the
head of its payload (`ALM-FRAME-031`). Every payload offset published before EXP-0030 must therefore be
read **−8**. For type-6 that is:

| Fact | Pre-EXP-0030 label | Pinned (`da54e6d`) label |
|---|---|---|
| Unit anchor `X` | `rec+8` | **`rec+0x00`** |
| Unit anchor `Y` | `rec+12` | **`rec+0x04`** |
| Unit count word | `meta+0x2c` | **`meta+0x24`** |

**The file bytes are identical under both labels** — only where the payload is deemed to start moved.
The 70-byte stride, the `70·count == payloadSize` identity (38/38) and the 8094/8094 in-bounds result all
carry over unchanged, because `payloadSize` is the same header field at the same file position in both
splits. `alm.Unit.X`/`Y` as landed in 0003 read the same bytes before and after the reframe, so this story
— which consumes `alm.Map.Units` and parses nothing — has **no behavioural exposure** to the correction.
The spec was drafted at the earlier pin `e243261` and quoted the superseded labels; it now quotes the
pinned ones, byte-preserving, with no requirement, AC or property changed.

**Research-lane consistency: clean here.** Unlike 0007's `TERR-LIGHT-023` (whose own prose still carries
the superseded offsets, recorded there as a research-lane gap), **`ALM-UNIT-018` carries its `−8`
amendment explicitly** in the claim row, and the promoted spec
`research/formats/alm/format.md` § "type6 — placed units" already states `count = meta+0x24`, `X @ +0x00`,
`Y @ +0x04` in its field table. There is nothing to record as unreconciled and nothing for us to edit.

### What the correction changed *for this story*, which is not nothing

The reframe left the unit anchors byte-identical but moved the **terrain grids**: a decoder on the old
base is `+4` cells off in X on type1 (u16) and `+8` cells on type2/type3 (u8) (`ALM-GRID-032`). 0003's
reframe (landed) fixed that. The consequence for 0009 is that **AC-6's "markers align with terrain cells"
is only a meaningful check at the corrected framing** — before it, correctly-decoded unit anchors would
have sat over terrain that was itself systematically shifted, and the manual check would have failed for
a reason that had nothing to do with this overlay. EXP-0030's own corpus evidence is exactly this
alignment argument run backwards: placement anchors' derived-impassability rate falls to 4.86 % at the
corrected `(4,8)` offsets versus 20.69 % at the old `(0,0)`.

## What the overlay actually consumes

This story adds **no format decoding**. Its entire dependence on the game format is two already-decoded
facts about `alm.Map.Units` (story 0003), plus its count:

1. Each unit's anchor coordinate `(X, Y)` as a `u32` fixed-point `/256` value; the integral anchor cell is
   `(X >> 8, Y >> 8)`, an integer shift.
2. The number of decoded units, `len(alm.Map.Units)`.

Nothing else is available, let alone read: `alm.Unit` exposes **only** `X` and `Y`. The remaining 62 bytes
of each 70-byte record are undecoded (0003 R-2) and are not surfaced by the reader, so no unit type,
owner, stat or inventory value can be read here even by accident.

## Claim inventory (what the overlay's inputs are built on)

| Claim | Statement (at the pinned framing) | Confidence | Used by |
|---|---|---|---|
| `ALM-UNIT-018` | type6 = **placed units**: 70-byte fixed records, `count = meta+0x24`; `X @ rec+0x00`, `Y @ rec+0x04` (`u32` fixed-point `/256`, low byte usually `0x80` = cell centre). Verified `70·count == payloadSize` **38/38** and all **8094/8094** anchors inside `[0,W)×[0,H)`. The loader allocates a `0x50`-byte unit struct and reads `X`,`Y` into its `+0/+4`; its reads total exactly 70 B at format version 990 | **High** (stride, count word, coordinate pair) / **Medium** (the "unit" *label* and every non-coordinate field) | The anchor cell `(X>>8, Y>>8)` each marker is drawn at (spec FR-1/FR-6); the Medium half is why the spec claims no unit type/owner/identity |
| `ALM-CNT-017` / `ALM-META-025` | The type-0 metadata is a manifest of the fixed-record counts: `+0x1c` = #type5, `+0x20` = #type4, **`+0x24` = #type6** (38/38 at the corrected offsets) | High | `len(Units)` reported by spec FR-5; the "count the map's own type-6 metadata declares" AC-6 compares against |
| `ALM-PLACE-033` | **Placement anchor → terrain cell is a bare `>>8`** — no origin subtracted, no border inset, no rounding. For type-6 the loader stores the raw `/256` anchor at the unit struct's `+0x00/+0x04` and the routine walking the loaded unit array (`R0151`) converts with the same bare `SAR ,8`. A whole-binary scan of all 82 shift-by-8 sites finds no `.alm` coordinate biased before or after the shift | High (the shift) / Medium (that `R0151` is *the* unit spawn walker vs one of several consumers) | Direct game-side confirmation that spec FR-1's `(X>>8, Y>>8)` is the engine's own conversion, not our convenience. New at this pin — it did not exist when the spec was drafted |
| `ALM-FRAME-031` | Corrected container framing: 20-byte file header + per-record 20-byte header `[tag=7][hdrLen=20][payloadSize][typeId][f32]` + `payloadSize` bytes of pure payload; no per-section `f0`/`f1`, no payload-head identity, no file trailer | High | Why the type-6 labels moved `−8` while the bytes did not (above); consumed by 0003, not by this story |
| `ALM-GRID-032` | All three grid layers are addressed from the record payload's first byte with `index = row·W + col`; a decoder on the old base is `+4` cells off in X on type1 and `+8` on type2/type3 | High | Not consumed here — but it is what makes AC-6's terrain-alignment check meaningful (above) |

Supporting `alm` implementation facts (already shipped in 0003, no change this story):

| Fact | Statement | Used by overlay |
|---|---|---|
| `alm.Unit` | Struct exposes `X, Y uint32` (fixed-point `/256`) **and nothing else** — the id/stats/inventory tail is undecoded (R-2) and deliberately not surfaced | The overlay reads `X`, `Y` and the slice length; there is no other field it *could* read |
| `alm.Map.Units` | `[]Unit`, one entry per decoded type-6 record; `decodeUnits` requires `len(payload) == 70 · Meta.Count6` exactly, so a mismatch is a decode error rather than a silent truncation | The list the cmd tier wires into the marker geometry; `len()` is the FR-5 count |

## Decoded-vs-open reconciliation

- **The overlay's inputs are fully decoded.** The anchor cell `(X>>8, Y>>8)` and the unit count are
  High-confidence, corpus-verified facts (`ALM-UNIT-018`, `ALM-CNT-017`), now with independent
  instruction-level support for the shift itself (`ALM-PLACE-033`). They are already implemented and
  tested in 0003. **There is no blocking research for this story and no format fact is invented.**
- **The "unit" label is Medium, and the spec says so rather than rounding it up.** `ALM-UNIT-018` is High
  for the *layout* and Medium for the claim that these records are units at all. The corroboration is
  circumstantial but consistent — the `0xFF` sentinel runs read as empty inventory slots, the loader
  allocates a `0x50`-byte struct per record and walks the resulting array through a routine that converts
  anchors to cells, and `research/`'s own status line calls them units. It is not a decoded type field.
  The spec's overlay is therefore honestly a **placement** diagnostic: it asserts that *something* the
  research calls a unit is anchored at that cell, which is exactly what a marker glyph claims.
- **Every non-coordinate field is Medium-or-worse and is scoped out, not guessed.** 0003's R-2 records
  that the per-record owner/template/item ids in type4/5/6 are undecoded, and `research/AGENTS.md` lists
  "Content-record fields (type6 70 B …): owner / template / item ids — currently Medium, only X/Y are
  pinned" as an open next question. The spec consequently claims **no unit-type id**: the research pins
  the stride and the coordinate pair and pins no type-id offset. Unit identity, class, owner and faction
  need the class registries (`REG-*`, EXP-0006) plus the static-data formats, which are downstream.
- **Nothing here is a rendering claim.** The glyph shape, its colour, its size, the clip order and the
  draw order are the project's **own** diagnostic-render design. The research decodes *where* units are,
  never how the original engine drew them; unit sprite lighting is an explicitly open research item
  (`research/AGENTS.md`, "Unit/sprite lighting"). The spec makes no fidelity claim for any of it.

When the research team closes the type-6 field ids (or the unit-sprite path), the flow is: repull the
submodule (`git submodule update --remote research`), derive the newly decoded fact here, then revise
`spec.md` from the earliest affected stage per the SDD revision process.

## Composition baseline — the six 0009-on-0008 contracts, reconfirmed against landed code

The spec was written before 0008 landed, so it phrases six things as contracts it *requires* of the
composed viewers rather than as observed facts. All six were re-checked against the code as landed. **All
six hold**; the notes below are what the plan must design against, and two of them constrain it.

1. **Draw order terrain → objects → units (spec FR-4) — holds.** `pkg/ui/viewer.go` `Viewer.Draw` runs the
   terrain tile loop first, then a separate loop over the object overlay's screen rectangles; appending a
   third loop yields the required order with no change to either existing stage. `cmd/terraintool/main.go`
   composites, then conditionally calls `terrain.DrawObjectMarkers` over the finished image, then writes
   the PNG; a unit pass inserted between the object block and the write is the same order. Both paths
   overlay *after* the fact rather than threading a flag through the compositors, which is what makes a
   second overlay additive.
2. **Object count before unit count (spec FR-5) — holds, and the token shape is fixed.**
   `terraintool`'s summary ends `…, placeholder cells %d, <light><objects>`, where the objects token is
   the literal `, objects N` appended only when the flag is set; a units token appended after it is
   object-then-unit by construction. In `mapview` the objects token is appended inside `load()` — after
   `tile slots L/S` and **before** the `, water speed …` clause that `run()` adds afterwards — so a units
   token must be appended in `load()` too, immediately after the objects token, to land in the same
   window. 0008's own summary test reconstructs the enabled line by splicing its token into the disabled
   line, so it stays green as long as the units token is emitted only under the units opt-in.
3. **Disabled-output baseline (spec FR-1/AC-4/P-3) — holds, in its strongest form.** 0008 does not
   compute-then-discard: `terraintool` guards the entire overlay call, `mapview`'s `load()` guards both
   the cell conversion and the summary token, and the viewer's `objectScreenRects()` returns `nil` when
   the toggle is off or the cell list is empty, so `Draw`'s overlay loop body never executes. With the
   overlay off, **no overlay code runs at all**. A second, independently-guarded overlay inherits the same
   property, which is what makes "byte-identical to the same invocation without this story" achievable
   for all four on/off combinations rather than only the both-off one.
4. **Byte-identity is really tested, and the test extends rather than needs duplicating — holds.**
   `cmd/terraintool/main_test.go`'s object-overlay test PNG-encodes the no-flag render and compares it
   byte-for-byte against an **independently built oracle** — a direct `terrain.CompositeLit(…)` call on
   the same inputs plus `png.Encode` — explicitly *not* against a second no-flag run, so it cannot pass by
   comparing the tool to itself. The same oracle construction extends to 0009's "objects on, units off"
   arm by adding a direct `DrawObjectMarkers` call to the oracle. On the viewer side the automatable
   witness is `objectScreenRects()` returning `nil` with the overlay off; full-frame identity needs a
   window and is 0008's manual criterion, as it will be ours.
5. **A unit marker can draw over a coincident object marker (spec FR-4/AC-6) — holds, with one honest
   caveat.** The PNG rasterizer writes `img.SetRGBA(x, y, MarkerColor)` — an unconditional opaque store,
   no blending — so a later unit pass over the same pixels replaces them; the viewer draws with
   `vector.DrawFilledRect` in an opaque colour, so a later call paints over the earlier one in the
   framebuffer. Drawing units second therefore puts them on top. **Caveat:** at native scale the spec's
   unit cross (`r=4, t=1`) is strictly smaller than 0008's object cross (`r=6, t=3`), so a coincident pair
   renders as a cyan cross *inside* a yellow one — the unit marker is unambiguously on top, but it does
   not hide the object marker. That is compatible with "drawn over", and it is arguably the better
   diagnostic, but AC-6's manual observer must be told to expect it rather than a solid cyan cross.
6. **`pkg/ui` landed objects-specific plumbing, not a reusable hook — this is the one that costs work.**
   Nothing in either tier is parameterised by overlay kind:
   - `pkg/ui`: `SetObjects(show bool, cells []image.Point)`, the fields `objectCells` / `showObjects`, and
     `objectScreenRects()` — which hardcodes `terrain.ObjectMarkerRects` and `terrain.CellSize` — plus an
     objects-named loop in `Draw`.
   - `pkg/render/terrain`: `ObjectMarkerRects`, `DrawObjectMarkers`, and a single package-level
     `MarkerColor` (object yellow) with the arm radius and thickness as untyped package constants, not
     arguments.

   What **is** reusable unchanged: `AnchorCell` (the generic `>>8`), `scaleDim` (the generic
   round-to-nearest dimension scaler), the `screenRect` type, and the `image.Rectangle.Intersect` clipping
   discipline. So 0009 is not parameterisation of an existing hook; it is a parallel implementation that
   *may* be refactored into one. Both options are real and the plan must choose deliberately:
   *(a)* add unit-named siblings and leave 0008 untouched — purely additive, greenfield, some duplication;
   *(b)* generalise the primitives (a shared rect generator taking radius/thickness, a colour argument, a
   `SetUnits` beside `SetObjects`) — less duplication, but it edits shipped exported API (`MarkerColor` is
   referenced from `cmd/terraintool`'s tests) and is **brownfield** for those two files, with 0008's tests
   as the characterization pin. Neither is chosen here; `analysis.md` is not the place a design decision
   is made.
