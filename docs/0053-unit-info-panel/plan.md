# Plan — one value for the look, one filter for the subject

## Shape

Four pieces, in dependency order. Only the third is new code of any size.

```text
pkg/render/terrain   UnitClass gains its class's name                     FR-1
pkg/game             LoadUnits fills it; the snapshot carries it          FR-1
pkg/ui               the layout value, the subject, the picture, the draw FR-2..FR-6, FR-8
pkg/game             the front-end loads the font and hands it over       FR-7
```

`pkg/ui` may already import the whole render tier, so the font model and the class bundle are both
in reach and no allow-map entry moves (FR-8).

## DD-1 — the name rides the class bundle, and is filled by the loader

`terrain.UnitClass` gains `Name string`; `LoadUnits` sets it from the registry class's `DescText`
and nothing else touches it. The bundle is already the render tier's mirror of a registry class,
filled by a loader in the tier that may read a registry, and `Anim` and `Corpse` ride it this way —
a second map keyed by class id would be a parallel structure with the same lifetime and filler.

Verbatim: this is the one tier that could apply an encoding, and it must not, because the font
downstream indexes bytes and a conversion here would be a second, wrong codec before a correct one.

**Rejected: a lookup table at the panel.** It would need the class id at the window tier, which the
seam deliberately withholds — that tier says *which* entity and never *what* it is (0028 FR-9).

## DD-2 — the snapshot carries the name, filled from the LIVE class

`ui.MapEntity` gains `Name string`, filled in the snapshot builder from `classes[e.Class]` — the
entity's own class — **before** and independently of the corpse substitution that replaces `Art`.

This is the whole of AC-3, and a placement decision rather than a computation. The death path
replaces `Art` whole; a name read off it would be right for a live unit, right for the common case
of a class that names itself as its own dying class, and silently wrong exactly where the two differ
— which shipped data would hide. Filling from the live class puts the substitution out of reach.

A nil class leaves the zero value, the empty name, which is what FR-1 asks for — the same "resolved
to nothing" answer `Art` already gives.

## DD-3 — the panel is a picture, composed in `pkg/ui` and pure

The composition takes a layout, a font and a subject and returns an `*image.RGBA`. It is a plain
function, not a method on the viewer, so nothing it does reaches viewer state and every case of FR-4
and P-1 is drivable from a struct literal with no window.

**Why the picture and not a draw list.** The font's own blit paints into an `*image.RGBA` (0050
FR-6); a draw list would re-express glyph placement in the engine's terms and so put a second copy
of the pen rule here — the failure 0050's single placement rule exists to prevent. An image uses
that blit unchanged, is assertable pixel by pixel, and is what a developer run writes to a PNG.

**Why `pkg/ui`.** The panel's placement is a function of the window's size and its subject is the
selection; both live here, and a render-tier package would need both handed to it for no separation
the layout value does not already give.

## DD-4 — the layout is one exported value, and the authored one is a function

One struct carries anchor corner, margin, fixed-or-fit size, minimum width, padding, gap, background
picture, fill/border/label/value colours, and the ordered rows; a row carries its field, its label
and its own offset. `AuthoredLayout()` returns this project's values **by call, not as a package
variable**, and builds its row slice fresh each time: a package variable — or a shared backing array
behind a copied slice header — is writable from anywhere, and P-5 would then be falsifiable by
mutation rather than true by design.

Flow-versus-placed is one bool rather than a per-row optional offset: the per-row option has a
representable middle state — some rows flowing, some placed — that no reader could lay out
unambiguously, and one bool has none.

**This is the substitution point named by the story's verdict.** Replacing the authored panel with
the original's, once the interface layer is read, is supplying a different value of this type — a
background picture, placed rows, its own colours — and touching no other line. The same seam is what
makes the panel customisable.

## DD-5 — the subject is the existing filter, called

The subject is the first element of the package's existing present-and-not-dead filter over the
selection and the current snapshot. **Called, not copied**: the marks, the orders, the blows, the
path overlay and now the panel are the same units, and another reading of "still there to act on"
would be another chance to disagree. It takes the first element rather than searching, because that
filter already emits in the selection's own ascending order.

The subject crosses as a small comparable value — the id and the stated numbers — and not as a
snapshot entry, which carries a route slice and pointers and so could not be a key's field.

**Rejected: a latched subject.** It would keep a panel alive over a unit the world no longer holds,
and need an invalidation rule for every way a unit can leave — which is the filter.

## DD-6 — fit-to-content geometry comes from the font's own measurement

A row's value sits at its label's **pen** plus the label gap — the pen and not the label's box,
which runs wider wherever a glyph's art overhangs its advance and would open a hairline varying with
the last letter. Its **width** is the larger of the label's box and the value's end: overhanging ink
is still ink, and a width off the pen alone lets a fitted box clip it. Both come off one placement
(0050 FR-5), which makes AC-9's containment arithmetic rather than a promise.

The box is the furthest right and lowest any row reaches, plus a padding — **one formula for both
row modes**, so a fitted box over placed rows is not a fourth case nobody wrote. Each axis fits
independently.

The minimum width is a floor and never a ceiling: FR-3 has the panel state a value or omit it, and
a truncated name is a third thing.

## DD-7 — the rebuild key is a value, and comparing it is the whole refresh rule

The viewer holds the last picture and the key it was built from: the subject's id and its stated
values (name, health pair, cell), a serial that changes when the layout or font is replaced, and the
placement area's size. A frame recomputes the key, compares, and rebuilds only on a difference (FR-6, AC-12).

The key holds the **stated values and not the entity**: route, step, art and frame change
constantly and none is on the panel, so keying on the entity would rebuild every frame a selected
unit walked. It holds the area size because a fitted panel anchored to a corner moves on a resize.

**A serial rather than a compared layout.** A layout carries a picture pointer and a row slice and
is not comparable in Go's own sense; a deep comparison would be a second definition of layout
identity. The two setters own the serial.

## DD-8 — one texture, refreshed with the picture and never accumulated

The viewer holds ONE engine-side image. It is dropped in the same statement that replaces the
picture, so a fresh picture is never presented under a stale texture; it is built lazily inside
`Draw` alone, which keeps the viewer's invariant that a built, queried and culled viewer holds no
GPU state; and a rebuild that leaves the box the same size writes into the image already there.

That last clause is not tidiness. The key holds the cell and the health pair, so a selected unit
that is walking or under fire invalidates on EVERY tick, and nothing in this tree disposes a
texture — every other holder is a cache over a bounded domain. A fresh image per rebuild would make
the panel the first unbounded allocator on the draw path.

## DD-9 — the front-end loads the font, hands it over per map, and does NOT die without one

`FrontEnd` gains `Font`, loaded in its constructor from the container filesystem already open, and
`FrontEnd.loadMap` hands it to the viewer it has just built. That site and not the shared map
loader, which also serves the standalone developer viewer — that one holds no selection, so it can
never have a subject and a font would reach it unusable.

**A failed load is reported and carried, not fatal** (FR-7), and that is where this parts company
with the registries beside it. Those gate a map's CONTENT and are worth a refusal before a window
opens; this gates a three-line box to look at. Fatal, it would put a working game on original assets
behind a cosmetic asset and make one missing archive entry equivalent to a missing archive. The
tolerant half is already required — FR-8 and AC-14 have a fontless viewer draw the pre-story screen
— so this only makes the front-end tolerate what the viewer already does.

## DD-10 — the panel draws last, outside the pass slice

`Viewer.Draw` presents the panel after the overlay passes and the path overlay. An `overlayPass` is
a colour, sprite placements and rectangles; an uploaded image is none of those, so folding it in
would widen every pass with a field only one could hold — the same reason the path overlay sits
outside the slice, and NOT a difference of coordinate space: those rectangles are already in screen
pixels and the marquee is built in screen space outright.

A method answers *what to present and where* and `Draw` makes the one engine call, so the origin
arithmetic is assertable with no window (AC-11) — the only automated guard on where the panel
lands — and the texture is still built inside `Draw`, where the viewer's no-GPU-until-drawn
invariant requires it.

## DD-11 — the closed field set is an enumeration with one switch

A row names its field as one of three constants, and one switch turns a subject into that field's
text. **One switch, one place.** FR-3's closed list and P-4's "no value the snapshot does not carry"
are then properties of a single function, and a later story's extension point is a case in it beside
a field that has become real — not a new drawing path.

A field with no value yields no row and consumes no space (FR-3) — one return value from the same
switch, so "omitted rather than filled" cannot go inconsistent across fields.

## Risks

- **R-1 — the panel covers what the owner wants to look at.** It occupies a small corner region and
  the anchor is in the layout value, so moving it is a value change. Accepted.
- **R-2 — the authored look is judged wrong.** It will be, at least once: only the owner can judge
  it, which is why the story ships renders. The layout value makes each iteration a value change.
  One constraint is not taste and bounds the choices: the font's blit REPLACES rather than blends
  and scales a partly-lit pixel toward black, so light text on a dark fill reads cleanly and dark
  text on a light one fringes.
- **R-3 — the health shown is the provisional spawn pair, not the class table's.** Real, disclosed
  and not this story's to fix: the value is the simulation's own and is stated truthfully. Recorded
  in the evidence rather than hidden by dropping the field, which is the number the game simulates.

## Success criteria

- **SC-1** — Every class in a loaded bundle carries its own registry name, byte for byte (FR-1).
- **SC-2** — Every snapshot entry carries its own class's name, corpse substitution included (FR-1).
- **SC-3** — The subject is the filter's first element in all four cases: several selected, some
  dead, none surviving, none selected (FR-2).
- **SC-4** — The picture carries exactly the rows the layout names for fields the subject has, and
  no other quantity (FR-3, FR-5).
- **SC-5** — Composition is byte-identical across repeats and reaches no file, clock or context
  (FR-4).
- **SC-6** — Every painted pixel lies inside the box, over generated names, healths and cells,
  including a font whose ink overhangs its advances (FR-4).
- **SC-7** — The panel's origin is the configured corner at the configured margin, for several area
  sizes (FR-6).
- **SC-8** — The picture is rebuilt on a stated-value change and on no other frame (FR-6).
- **SC-9** — A front-end assembles with a font, and assembles carrying the reason — still opening
  maps — on each broken font container (FR-7).
- **SC-10** — A fontless viewer submits the same passes in the same order as before the story, and
  the existing suites over the touched packages pass unedited (FR-8).
- **SC-11** — The panel renders against a lawful install, at both shipped releases, and the pictures
  are shown to the owner (FR-6, FR-7).

## Traceability

| FR | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-1, DD-2 | SC-1, SC-2 |
| FR-2 | DD-5 | SC-3 |
| FR-3 | DD-11 | SC-4 |
| FR-4 | DD-3, DD-6 | SC-5, SC-6 |
| FR-5 | DD-4, DD-11 | SC-4 |
| FR-6 | DD-7, DD-8, DD-10 | SC-7, SC-8, SC-11 |
| FR-7 | DD-9 | SC-9, SC-11 |
| FR-8 | DD-3, DD-9, DD-10 | SC-10 |
