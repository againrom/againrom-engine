# Spec — the unit info panel

## Problem and current behaviour

The map screen can be told which units are selected: it rims their cells, routes their orders and
draws their health bars. It cannot say **what one of them is**. A unit's name is decoded, loaded and
dropped one tier below the window; the game's own font draws no word on the map screen; the health
pair and the cell cross the per-tick snapshot already and are read only as a bar's ratio and a
placement. Nothing on screen states a number.

This story puts a panel there.

## The subject

The panel describes **exactly one unit at a time**, and the unit it describes is the **first of the
current selection, in the selection's own order**, among the entries the current snapshot still
holds and still reports **alive or downed**. That is the same set the map screen's marks, orders and
blows already take.

Three states follow, and each is decided here:

- **Nothing selected — no panel at all.** Not an empty frame, not a frame of blanks: the panel is
  absent.
- **Several selected — the first is described, and nothing is said about the others.** The panel
  neither summarises them, counts them, nor cycles through them.
- **The subject leaves the set** — it dies, or the world stops holding it — **and the panel goes
  with it**, on the very next frame, without the selection itself changing. A **downed** unit stays
  described.

## What the panel states

Three things, and it is a **closed list**:

| Field | Value |
|---|---|
| **Name** | the subject's actual name when non-empty; otherwise its own class name from the installed registry, drawn unconverted |
| **Health** | the unit's health and the maximum it was built with, as `<hp>/<max>` in decimal |
| **Cell** | the column and row the unit stands on, as `<x>, <y>` in decimal |

A negative or zero health is stated as it stands, unclamped. Every value is
the simulation's own answer as the per-tick snapshot carries it, **carried, never re-derived**: the
panel performs no arithmetic on a unit's state beyond rendering an integer in decimal.

**Only the name can be absent** — a subject with neither an actual name nor a resolvable class name
has no name, and its row is the one that can be skipped. Health and cell always have a value for a
subject that exists.

**The list is closed.** No value the tree cannot source for that unit may appear on the panel in any
form: not as a zero, not as a dash, not as a label with an empty value. **An omitted field is
honest; a placeholder that looks like data is not.** A field joins this list only in a story that
first makes its value real.

**The name belongs to the subject, not to its current drawing.** A non-empty actual name always wins.
The installed class name is only the fallback and is never taken from a class substituted for
drawing.

## The layout, and what it is for

**Everything about the panel's appearance is one value** — its geometry, its colours, its background,
its label text, and which field each row states in which order. The panel is a function of that
value, the font, and the subject, and of nothing else: there is no dimension, colour, string or
ordering anywhere else in the drawing path for that value to disagree with.

The panel's appearance is **authored**. Replacing it wholesale — with the original's, once that is
decoded, or with any other — is a change of that one value and of nothing else.

The value carries:

- **Where the panel sits** — which window corner it is anchored to, and the margin from it.
- **How big it is** — either a fixed size, or *fit to content*, in which case the box is derived from
  the measured rows and a minimum width.
- **What is behind the text** — either a supplied background picture, drawn as given at the panel's
  own origin, or a flat fill in one colour under a one-pixel border in another.
- **The rows** — an ordered list, each naming one field of the closed list above, its label text and
  its own position. Rows either **flow** — stacked from a padding inset, separated by a gap — or are
  **placed**, each at its own offset. A row whose field has no value is skipped, and skipping it
  consumes no space in flow.
- **Where a row's value sits** — the gap left after the label's own pen before the value begins.
- **The colours** the label and the value are drawn in.

**Nothing painted leaves the panel's box.** A row a layout places outside it, a value wider than a
fixed size, and a background larger than the box are all clipped there rather than growing it or
escaping it; a fit-to-content box is derived from the rows and so clips none of them.

The authored value is available under a name.

## The text

Text is drawn with the game's own font by the placement rule already in the tree — bytes select
records, nothing is transcoded — so a name holding bytes at or above `0x80` reaches its own glyphs.
Whether those glyphs then *read* as the name is a question about the atlas's arrangement, which this
story does not settle and does not claim.

## Functional requirements

- **FR-1 (the name reaches the window tier)** — The unit-art bundle carries each class's own name
  text alongside its geometry and frames, taken from the class registry verbatim as bytes with no
  character encoding applied. The per-tick snapshot carries the subject's actual name separately
  from the index of its own class. Composition uses the actual name when non-empty and otherwise
  resolves that class index through the installed name table. A class substituted for drawing never
  changes either source. A subject with neither source carries no displayed name.

- **FR-2 (the subject)** — The panel's subject is the first entry of the selection, in the
  selection's own order, restricted to entries the current snapshot holds and reports alive or
  downed. Nothing selected, and a selection none of whose entries survives that restriction, both
  yield no subject.

- **FR-3 (what is stated)** — The three fields above, their values carried from the snapshot and
  formatted as decimal integers or drawn as bytes, with no arithmetic on unit state. No other value
  is stated, and no field is emitted with a substitute, default or placeholder value.

- **FR-4 (the panel is a picture, and a pure one)** — Composing the panel is a function of the
  layout, the font and the subject: same inputs, same pixels, every time. It opens no file, reads no
  clock, requires no graphics context, and is fully exercisable with no window.

- **FR-5 (the layout is one value)** — Everything the section above lists is carried by a single
  value, and the composition reads its appearance from that value alone. The authored one is
  available under a name, and nothing else in this contract is a function of which value is supplied.

- **FR-6 (on screen)** — The panel is anchored to the configured window corner at the configured
  margin, clipped to the window, drawn after everything else the map screen draws, and rebuilt only
  when the subject's identity or a stated value, the layout, the font or the placement area changes.

- **FR-7 (the font reaches the game)** — The front-end loads the game's own default font once when it
  loads its other art, from the same container, and hands it to every map it opens. **A font that
  will not load is reported and is not fatal**: the front-end assembles without one, carries the
  reason, and every map still opens and plays. A viewer holding no font draws no panel and does not
  fail.

- **FR-8 (what does not move)** — No simulation, digest, save, codec, map-loading or map-drawing
  behaviour changes. No entity gains a field, no world's byte form or digest moves, and nothing
  inside the determinism wall is touched. No package's permitted import set is widened. A
  viewer that was never given a font, or has nothing selected, draws exactly what it drew before this
  story. No game bytes enter the repository.

## Acceptance criteria

Synthetic throughout: every fixture is built in test code, no test reads a game install, and a byte
at or above `0x80` is written as an escape, never as literal text.

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | an installed class-name table whose entries differ, one holding bytes at and above `0x80` | resolving the panel's fallback | each class index resolves its own bytes with no conversion, and an unavailable entry supplies no fallback |
| AC-2 | a named subject, an unnamed subject with a resolvable class index, and one with neither source | composing | the first displays its actual name, the second the installed class name, and the third omits the name row |
| AC-3 | a named unit that is downed and one that is dead, each of a class whose death art resolves to a **differently named** class | building and composing the snapshot | both display their actual names, not either the substituted drawing class or the installed fallback |
| AC-4 | a selection of three units, one of them dead and one absent from the snapshot | asking for the subject | the first surviving entry in the selection's own order is the subject |
| AC-5 | an empty selection; and a selection all of whose entries are dead or absent | asking for the subject | no subject in both cases |
| AC-6 | a subject, a font and the authored layout | composing the panel twice from the same inputs | the two pictures are byte-identical, and no file, clock or graphics context was reached |
| AC-7 | a subject whose name, health pair and cell are known | composing | the picture contains exactly the three rows, each carrying its label and its value; nothing in the picture states any other quantity |
| AC-8 | a name holding bytes at and above `0x80`, against a font whose records differ per byte | composing | each byte selects its own record, so the drawn name is the sequence of those records and not a run of one repeated glyph |
| AC-9 | the authored layout in fit-to-content mode, with rows measuring wider than the minimum width and narrower than it | composing | the box contains every painted pixel of every row, and is never narrower than the minimum |
| AC-10 | a layout with a supplied background picture and placed rows at named offsets, and one with a flat fill, a border and flowing rows | composing each | the background is drawn as given in the first and as fill-plus-border in the second; a placed row's label starts at its own offset, and a flowing row's at the padding plus its index times line height and gap; a row placed outside the box paints nothing |
| AC-11 | a viewer with a font and a subject; the same viewer with no font; the same with nothing selected | asking for the frame to present | a panel in the first, none in the second and none in the third, and its origin in the first is the configured corner at the configured margin for the current area |
| AC-12 | a viewer presenting a panel | frames in which nothing changes; one in which the subject's health changes; one in which the subject is replaced by a different unit stating identical values; one in which only the camera moves | the picture is rebuilt on the second and the third, and on neither of the others |
| AC-13 | a container holding the font's two nodes; one missing a node; one whose nodes disagree in count | assembling the front-end | the first carries a font; the other two assemble with none and carry the reason, naming the failure, and a map opened from either still draws |
| AC-14 | a viewer never given a font, with entities and a selection | drawing a frame | the draw order and every pass it submits are identical to those of the same viewer before this story |

## Properties

- **P-1** — Composition is pure: given a layout, a font and a subject it is a function of its
  arguments, and repeating it yields identical pixels.
- **P-2** — No simulation, determinism, digest or decoded-format behaviour is touched; no entity
  field is added and no world's byte form moves.
- **P-3** — For every layout, font and subject, every pixel the panel paints lies inside the panel's
  own box, and the box lies inside the window when the window can hold it.
- **P-4** — No value appears on the panel that the snapshot does not carry for that subject. Held
  over subjects with empty names, zero and negative health, and a zero maximum.
- **P-5** — The panel's appearance is a function of the layout value alone: two viewers differing
  only in that value differ only in the panel's pixels, and two viewers sharing it produce the same
  panel for the same subject.

## Out of scope

- **Reconstructing the original's panel.** Its layout is not decoded; what ships is authored,
  disclosed as such, and replaceable at one value.
- **A portrait**, and any art taken from the install for the panel's background.
- **The definition table's stats on screen.** They reach no live unit, and the game loads no such
  table; making them real is a story of its own, and this one adds no field it cannot source.
- **Any presentation of more than one unit** — no summary, no count, no cycling.
- **Interaction.** The panel is not clickable, hoverable, dragged, resized or dismissed; it states
  and does nothing.
- **Text layout.** No wrapping, alignment, ellipsis or line breaking: a row is one run of bytes at
  the pen rule already in the tree.
- **Structures and objects.** The subject is a unit; nothing else is selectable today.
