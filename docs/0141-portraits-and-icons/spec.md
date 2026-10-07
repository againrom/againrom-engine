# 0141 — portraits, dolls and spell icons

**The contract: what picture this build shows for an actor, and where it comes from.** Five
behaviours under it — a creature's flat portrait, a placed person's composed doll, the tier that
selects between a class's four pictures, the face of whoever is speaking in a dialogue, and a
spell's own icon in the book bar. They are one contract because they are one question asked at four
windows, and because the answer to all four passes through the same two composers and the same
one-test-on-the-class split.

## Functional requirements

**FR-1** Whether an actor is drawn by COMPOSING a figure or by LOADING a flat picture is decided by
one test on its unit class id and by nothing else: below `0x1a` compose, at or above load
(`UNIT-PICT-035`). The two are exclusive — for any actor exactly one arm is taken, and the arm not
taken opens no archive node.

**FR-2** A flat picture's untiered address is `<name>.bmp` under the archive's `infowindow` tree,
`<name>` being the unit class's `InfoPicture` or a structure class's `Picture` verbatim
(`UNIT-PICT-036`, `UNIT-PICT-037`). An empty name addresses nothing and yields the empty string
rather than a directory with an extension on it.

**FR-3** A flat picture's TIERED address appends the tier digit to `<name>`, except at tier 1 where
the digit is dropped — the engine's own truncation, and the reason the shipped set reads `<name>`,
`<name>2`, `<name>3`, `<name>4` with no `<name>1` (`UNIT-PICT-036`). Any integer tier formats; a
tier that names no node is an address that resolves to nothing, not an input to refuse.

**FR-4** A format leaf decodes the one Windows BMP shape this game ships: 24 bits per pixel,
uncompressed, `dataOff` 54, no palette (`SPR256-PICT-043`). It refuses every other shape by name,
computes the pixel run's length from the width, the height and the row stride rather than reading
the header's own `imgSize`, accepts a file longer than that run and ignores the tail, and delivers
rows TOP first with each pixel's channels in red-green-blue order. A read outside the image is the
zero pixel, not a panic.

**FR-5** The info column shows a selected non-composing actor's flat picture AT THAT ACTOR'S OWN
TIER. **This is a disclosed divergence.** `UNIT-PICT-036` reads the original's info panel as
formatting the name with three arguments — no tier reaches that leaf, so in the original a red orc's
info panel shows the green picture — and it is the DIALOGUE panel that appends the digit. The owner
ruled on 2026-08-11, having played the build, that the tier is shown in both. What is reproduced is
the ADDRESS; which window uses it is ours. A tier below one is the first tier, and a tier the class
ships no file for falls back to that class's first picture, which cannot show the wrong creature
because the name is the same either way.

**FR-6** The info column shows a selected COMPOSING actor's doll: its figure base sheet with one
layer painted over it per occupied equipment slot, in ascending slot order. No base sheet is no
picture, and a layer is never promoted to the picture on its own.

**FR-7** A placed person's figure directory and face come from the three consecutive columns its
`Humans` row states — type id, face, gender (`DAT-HUMANS-008` slots 16, 17, 18). Gender 1 is female;
a type id in `0x17..0x18` is a mage; the face is the column WHOLE and is not split. A placed person
is drawn on the hero arm of the engine's own composer, because the constructor overwrites the type
id with `gender + 0x21` for a fighter or `+ 0x23` for a mage before the byte reaches the wire
(`DAT-HUMANS-008`, `DAT-ACT-006`, `ALM-CLS-054`), and on that arm the sex is bit 0 of
`typeID - 0x21` (`HERO-APPEAR-041`) with `typeID ∈ {0x22, 0x24}` as the sex test
(`HERO-CLASS-013`). The face column's bit 7 belongs to the OTHER arm and is not read here.

**FR-8** With nothing selected the picture box shows nothing (owner, 2026-08-11). An actor whose
picture this build cannot produce — a class the bundle does not hold, a node the install does not
carry, a map with no archive at all — falls back to that unit's own drawn world frame, which is a
missing FILE's fallback and not a missing decode's.

**FR-9** A dialogue part that names a speaker shows that speaker's picture. The `<npc=N>` number
names the `npc<n>` section of `scenario.res::npc.reg` (`DLG-NPCTAG-018`) and the record decides
among three answers (`REG-NPC-088`):

- a record whose `Flags` lack `Human` and which states `Picture` loads the flat picture of the unit
  class that value names, at the record's own `Face` as the tier digit;
- a record that states `Face` composes a figure, the directory chosen by the `Mage` and `Female`
  flags and the sheet by that `Face`;
- a record naming neither key is the player's own character and shows the party subject's own
  composed figure, worn layers and all.

The flag decides before the key: a record carrying both keys and `!Human` takes the portrait arm. A
negated token (`!X`) counts as the token being absent.

**FR-10** The dialogue pane draws a fixed 72x96 window cut from the speaker's picture at that
record's own `PortraitX1`/`PortraitY1`, in the picture's own top-down pixels, and blits it to a
fixed point inside the pane (`REG-NPC-089`, `REG-NPC-091`). A record stating neither coordinate — or
only one of them — takes the engine's own absent-key default instead, the literal 72x92 window at
picture rows 8..100. `PortraitX2`/`PortraitY2` are not read.

**FR-11** The pane's ground is opaque black and the picture is COMPOSITED over it, source-over
(owner, 2026-08-11). Every pixel of the pane's interior is fully opaque whatever the picture's own
alpha is. A window falling partly outside the picture draws the part that overlaps, in the place it
would have landed; one falling wholly outside draws nothing.

**FR-12** A speaker this build cannot resolve draws an empty pane and is not an error: a number
naming no record, a record whose class the bundle does not hold, a picture the install does not
carry and a mission with no party subject all answer the same way, and everything else about the
notice works.

**FR-13** Each spellbook cell carries that spell's own icon: a 36x36 square cut from the single
pre-composited `interface/SpellBook.bmp` strip at the position the engine's own 24-entry slot table
gives the spell — `x = 6 + 38*(slot%12)`, `y = 6 + 38*(slot/12)` (`MAGIC-ICON-024`). The strip is
read once per mission and only if it decodes at exactly 480x85; a picture of any other extent is
refused rather than cut.

**FR-14** A spell that occupies no slot keeps the three-letter abbreviation the bar drew before
icons existed, and so does every spell on an install whose strip will not read. Four of the
twenty-eight shipped spells — ids 11, 17, 27 and 28 — are in no slot at all, which is the shipped
game and not a gap. A cell draws its icon or its letters, never both.

## Acceptance criteria

**AC-1** The compose predicate is true for every class id below `0x1a` and false at and above it,
and over the shipped roster that partitions ids 1..24 onto the doll side and 26, 27 and 64..80 onto
the flat side.

**AC-2** The untiered formatter answers the empty string for an empty name and `infowindow/<n>.bmp`
otherwise; the tiered formatter equals the untiered one at tier 1, appends the digit at 2, 3 and 4,
and formats a zero or negative tier like any other integer.

**AC-3** The decoder refuses a wrong magic, `planes != 1`, `bpp != 24`, `compression != 0`,
`dataOff != 54`, `nClrs != 0`, a non-positive extent and a short pixel run, each with an error
naming the field; it accepts a file carrying bytes past its pixels; it honours the four-byte row
padding; and it returns the file's bottom row as the image's last row.

**AC-4** A selected creature's pushed picture is its class's picture at the entity's own tier; a
class shipping one file with a placement at a higher tier gets that class's first picture; and one
address is read exactly once per session, a miss included.

**AC-5** A selected person other than the party subject is pushed its composed doll, the party
subject is pushed no flat picture at all (its figure reaches the box by its own seam), and a doll
recomposes when the worn set changes.

**AC-6** The figure resolver maps the three columns onto the four directories with the two axes
independent: gender 1 is female at either class, a type id of `0x17` or `0x18` is a mage at either
sex, the face passes through unchanged, and a man and a woman of one face never share a directory.

**AC-7** Nothing selected pushes no owner and no picture; a selection the world no longer holds does
the same; and a map opened with no mission behind it — hence no archive — produces no picture and
does not panic.

**AC-8** The speaker table resolves a `!Human` section with a `Picture` key to a portrait request
carrying that class and that `Face`, a `Human` section with a `Face` key to a figure request in the
flag-selected directory, a section with neither key to the no-picture kind, and a section carrying
both keys with `!Human` to the portrait arm; a section naming no record at all is absent rather than
present and empty.

**AC-9** A record stating both window coordinates carries a 72x96 window at that origin; a record
stating one of them carries none; the window is cut from the picture's own coordinates and is
offset by the picture's own origin when it arrives as a sub-image; a window off the picture draws
only what overlaps; and after a picture whose canvas is mostly transparent is drawn, every pixel of
the pane's interior is opaque.

**AC-10** The slot lookup answers false for spell ids 11, 17, 27 and 28 and a slot for the other
twenty-four; every cell of the grid lies inside 480x85; a spell with a slot receives a 36x36 fully
opaque cut; a spell without one, and every spell when the strip is refused, keeps its letters.

## Properties

**P-1** No game asset, no shipped string and no install path enters the repo. Every fixture is
synthetic bytes built in test code, and `go test` is green with no game present.

**P-2** `pkg/sim` is untouched. No figure directory, picture, window rectangle or icon reaches a
world, an entity or a byte form, so no digest changes and the byte-form version does not move.

**P-3** The drawing tier receives a PICTURE and nothing else: no class id, no registry key, no
archive address, and no knowledge of which of the two arms produced it.

**P-4** The format leaf imports the standard library only and sits at the leaf tier of the import
DAG, imported by no format and importing none.

**P-5** Every archive read is cached WITH its misses — by picture address, by (figure, worn set)
pair, by spell id, and once for the atlas — so an install missing a node costs one read per session
or per mission rather than one per frame.
