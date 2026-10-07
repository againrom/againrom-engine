# 1015 — spec

The behavioural contract of the skill school's training column, canonicalized to as-built at the
landing. It is self-contained: research provenance is in `provenance.md`.

## Scope

The school room's own column surface: which column face is shown, where each class's five skill
icons are drawn, how they are keyed, which skill a click selects, and when a pending selection is
discarded. The room's cells, prices, Train command, Talk dialogue and character region are
unchanged.

## Vocabulary

- **class** — 0 fighter, 1 mage. Resolved from the shown party member's own `Mage` field.
- **slot** — 0..4, the shared skill order Blade/Fire, Axe/Water, Bludgen/Air, Pike/Earth,
  Shooting/Astral. It is `data.SkillNames`'s order and it is `DIV-121`'s accepted order.
- **cell index** — `class*5 + slot`, 0..9. One index space serves the draw side, the hit test and
  the game side.
- **state** — 0 `on`, 1 `shine`, 2 `shine_on`. The rest state is not a picture: it is engraved on
  the column face.

## FR-1 — the column shows the shown member's own class face

`TownSchoolArt.Faces[class]` is drawn at `SchoolFaceOrigin` = (168,176) with `draw.Src`, over the
room background, whenever the school surface is composed with a resolved class and statistics mode
is off. The two faces are `graphics/interface/training/column/rt0000.bmp` for the fighter and
`rt0015.bmp` for the mage, each 148x208.

The room background `trnhall.bmp` bakes one of the two faces into itself. Which one is a property
of the install, not of the shown member, so the composition does not inherit it: the class's own
frame is drawn every time. On both preserved roots the baked face is `rt0015`, agreeing with the
shipped frame on 30152 of its 30784 pixels at that origin.

A nil face draws nothing and leaves the background visible. Nothing else about the room changes.

## FR-2 — each class's five icons are drawn at that class's own rectangles

`schoolSkillRects` is indexed by class and then by slot. The two rows share no rectangle.

Fighter: `(200,196,280,228)`, `(200,216,280,252)`, `(200,248,280,276)`, `(200,272,280,288)`,
`(200,288,280,308)`.

Mage: `(264,232,284,260)`, `(192,240,216,260)`, `(224,200,252,224)`, `(228,272,256,298)`,
`(224,236,256,262)`.

Every rectangle is exactly its patch's own size, so the existing centring of the picture inside the
rectangle is an identity. Each class's five lie inside that class's panel rectangle
(`SchoolPanelRect`): fighter `(192,192,284,312)`, mage `(188,188,288,308)`. The mage's five are
pairwise disjoint. The fighter's are overlapping horizontal bands, which is why the room needs a
raster mask rather than rectangles for its hit test.

`SchoolSkillRect(class, slot)` and `SchoolPanelRect(class)` are exported for `cmd/schoolcheck` and
answer the empty rectangle for an out-of-range argument.

## FR-3 — a skill icon is drawn only for a lit state, and pure black is transparent

For each slot of the shown class whose cell is enabled:

- selected and not hovered — state 0;
- hovered and not selected — state 1;
- hovered and selected — state 2;
- otherwise no picture is drawn, because the rest engraving is already on the face.

Every pure-black pixel of all thirty skill patches carries alpha 0, applied at load by `keyBlack`.
The picture is drawn with `draw.Over`, so a keyed pixel leaves the column face showing through.
The two rest faces are not keyed and every pixel of them is opaque.

## FR-4 — a click selects the skill it lands on

`TownSurfaceControlAt` resolves a point inside the shown class's panel rectangle by reading that
class's `mask.bmp` at the point's panel-relative position, mapping the colour code to a slot, and
answering cell index `class*5 + slot`. The mapping is:

| code | fighter | mage |
|---|---|---|
| `0xff` | 0 sword | 2 air |
| `0x9e` | 1 axe | 4 astral |
| `0xd2` | 2 club | 3 earth |
| `0x87` | 3 pike | 0 fire |
| `0x37` | 4 bow | 1 water |

An unpainted mask byte, a disabled cell, statistics mode and a point outside the panel each answer
no control.

The index is a visual cell index at every consumer: `townScreen.schoolCell` becomes it,
`schoolSurfaceCells` labels cell `i` with `data.SkillNames[i%5]` and marks it selected,
`selectedSchoolSlot` resolves hero skill `i%5+1` and its price, and `trainHeroSkill` buys that
skill. So the mask's answer and the drawn rectangle name the same skill.

## FR-5 — a party-picker step discards the pending skill and its quote

A picker step that moves the shown member while the school room is open sets
`townScreen.schoolCell` to `schoolNoSelection` (-1). Then no cell reports `Selected`,
`selectedSchoolSlot` answers false, the Train button quotes 0 and is disabled, and pressing Train
reports `select a skill for this hero` and changes no hero.

A step that is refused because the party has one member changes no member and clears nothing.
The shop's and the tavern's own picker state is not touched, and a step taken in either of those
rooms does not clear the school's pending selection.

## Data

`LoadTownSchoolArt` reads, in addition to what it read before, `column/rt0000.bmp` and
`column/rt0015.bmp`, each asserted 148x208. A missing or mis-sized picture is an address-bearing
error beside a nil result; the front end carries it and the game still starts. No format changes,
no new archive.

## Not in this story

The rotation animation between the two rest faces (`DIV-142`); the original's own visual-to-stored
permutation, which `DIV-121` overrides on both sides now instead of one; the school's price, Train
command and General slot beyond FR-5; detailed character generation's own skill column, which
reads a different art set through a different origins table; the five fields of `TOWN-138`'s reset
that have no counterpart here (`DIV-143`).
