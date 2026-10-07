# 0141 — plan

## Shape

A new format leaf (`pkg/formats/bmp`), four new `pkg/data` files that format addresses and answer
predicates and read no archive, four new `pkg/game` files that open the archive and hand pictures
across the seam, and three `pkg/ui` changes that receive them. The direction is the existing DAG's
and nothing crosses it: `data` never opens a node, `ui` never learns a class.

## Design decisions

**DD-1 The compose test is written as the engine's test, not as the shipped partition.**
`ComposesFigure(classID) = classID < 0x1a`, with the roster's 1..24 / 26,27,64..80 split recorded as
a consequence. Writing the partition instead would be correct for the shipped registries and wrong
for a class they do not carry.

**DD-2 Two path formatters off one name, not one with a flag.** The info panel and a structure form
`<name>.bmp`; the dialogue panel forms `<name><tier>.bmp` with the digit dropped at tier 1. A
tier-4 monster's two addresses are two different files in the original, so collapsing them into one
formatter with a nullable tier would hide the fact that the engine's own consumers disagree.

**DD-3 The tier divergence is taken where the address is FORMED and disclosed there.** The choice to
use the dialogue panel's formatter for the info column is a single line in `classPortrait`, with the
claim, the owner's ruling and the sentence "what is reproduced is the address; which window uses it
is ours" beside it. Taking it at the call site would leave the divergence invisible to whoever next
reads the formatter.

**DD-4 A tier a class has no file for falls back to that class's FIRST picture.** The original
addresses a node that does not exist; three of the sixteen portrait classes ship one picture and
declare `Palette 1`, so a placement stating a tier on one of them cannot resolve. Showing the
class's own first picture is more forgiving than the original and cannot show the wrong creature,
the name being the same either way. It is one recursive call, and the fallback address is compared
before it is taken so the ordinary case costs one read.

**DD-5 `pkg/formats/bmp` is a seventh leaf, and it refuses everything the corpus does not contain.**
A general BMP reader would carry palettes, run-length compression, bit fields and five header
versions this corpus has none of, and every one of those branches would be untestable here. The
decoder also declines to trust the header's `imgSize` — 0 on 85 of 86 shipped portrait nodes — and
computes the run from the geometry instead. It performs exactly two conversions: BGR to RGB, and the
bottom-up row order to top-down, both once rather than in each of the two consumers.

**DD-6 The sex axis is the row's GENDER column, not bit 7 of its face column** — and `HumanDef`
carries slot 18 verbatim instead of dropping it. This is the story's one reversal. Bit 7 of the face
byte is read on the composer's `typeID < 0x1a` arm; a placed person never reaches that arm, because
the constructor overwrites the type id with `gender + 0x21`/`+ 0x23` and puts the actor in the hero
range. The field states the CELL and not the reading; that 1 is female is `HERO-CLASS-013`'s sex
test and lives where the picture is chosen.

**DD-7 The class axis is a declared proxy.** The engine reads the fighter/mage axis off `+0x4c`
bit 2, which this tree does not model; a range test on the type-id column (`0x17..0x18`) stands in
for it, said so at the constant, and corroborated twice — 57 of 57 rows carrying `0x18` have mana or
a spellbook against 0 of 153 below `0x17`, and the fifteen `npc.reg` person records stating `Mage`
agree 15 of 15.

**DD-8 The `npc=` tag resolves through the REGISTRY, and the placement reading was eliminated before
a line of it shipped.** Binding `npc=N` to the placement whose npc subscript is N has real support
and is the person the player can see standing there; it matched 53 of 337 campaign tags, and seven
missions carry dialogue while placing no npc at all. The registry reading matched 337 of 337.
`DLG-NPCTAG-018` then settled it by subscript.

**DD-9 The flag decides before the key, and `Face` is read on BOTH arms.** Thirty-three shipped
records carry both keys and all thirty-three are `!Human` creatures whose `Face` would otherwise
compose a person out of a tier digit. `Face` is the sheet number on the figure arm and the tier
digit on the portrait arm, so it is read before the arm is chosen.

**DD-10 The speaker table is built once per process and handed into each mission; the mission driver
is its own face source where none is supplied.** A speaker's record is a fact about the CAMPAIGN and
not about a map — half the shipped speakers are not placed on the map that quotes them. The driver
is the one object holding the mission's archive, its unit classes and the party subject a record can
refer to instead of carrying a picture, so it fills the seam; a caller supplying its own source
still wins, which keeps the seam a seam. The registry is parsed a second time rather than widening
the placement table's lookup: two answers, two lifetimes, two consumers.

**DD-11 The window travels WITH the picture, through one seam and under one serial.** The face
source returns both, the dialogue value carries both, and the viewer writes both at every path that
writes either — so a window can never outlive the picture it was read for or reach the composition
without it. The 72x96 extent is a constant of the PANE and lives beside it; the origin is a registry
value and arrives from the tier that reads registries. The zero rectangle is "this record states
none" and leaves the layout's own default standing, resolved by a `WithFaceWindow` helper that
mirrors the existing `WithPortrait`.

**DD-12 The destination point is read as PANE-relative, and that reading is ours.** The claim gives
the bare point `(8, 7)` without saying what it is measured from. `8 + 72 + 8` is 88, the pane's
width exactly; panel-relative would put the picture outside the pane, which is the alternative this
eliminates.

**DD-13 The pane composites source-over onto an opaque black ground.** It used to copy the source
pixel verbatim, alpha and all, and a composed figure is mostly transparent canvas — so the pane was
a hole through the notice box with a person floating in it. The blend is stated a second time here
rather than shared with the inventory's, because that one centres a picture in an area and this one
places a window at a point.

**DD-14 An icon is CUT, not loaded, and the atlas's extent is checked before the grid is applied.**
There is no per-spell node: the strip is one picture the original blits whole, and what is per-spell
is a position in it. Cutting the measured grid out of a differently sized picture would hand the
window somebody else's pixels rather than fail, so an unrecognised atlas draws no icons. The bar's
cell grows 32 to 38 — the icon plus its one-pixel border — because nothing in this tree resamples a
picture, so the cell follows the art.

**DD-15 The picture push follows the SELECTION, once per frame, beside the spellbook push.** What is
selected changes on frames carrying no tick, and it is very often not the inventory subject. The
viewer holds the picture with the id of the entity it belongs to, so a stale push cannot answer for
a newly selected unit, and "this unit has no picture" and "nothing is selected" are one state rather
than two.

**DD-16 `mw.archive()` is one guard for a nil mission.** The mission pointer is nil for every map
the picker opens; every earlier reader sits behind a guard that happens to exclude that case, and
both new readers sit on a per-frame seam that runs for every map. The guard is written once so the
next reader inherits it.

## Success criteria

**SC-1** `go build ./...`, `go vet ./...`, `gofmt -l` and `go test -count=1 -trimpath ./...` are
clean, and `check-no-game-assets.sh`, `check-doc-budget.sh`, `check-hotfix-ledger.sh` and
`check-sdd-audit.sh` pass.

**SC-2** The import-graph test accepts the new leaf at the format tier, and `pkg/sim`'s structural
and behavioural checks are unchanged.

**SC-3** Over both lawful roots: 18 unit classes compose a figure and 16 take a flat picture; all 16
of those read and decode at 160x240 with none absent; thirteen carry tiers 2, 3 and 4 and three
carry only tier 1.

**SC-4** Over six campaign missions on the RU root (10, 20, 30, 40, 71, 130): every placement
resolves to a picture, split between the two arms, with no miss on either, and tiers 2, 3 and 4 all
appear and all read.

**SC-5** Over both roots' fifteen campaign missions: the gender-column reading lands every human
placement on a shipped sheet and draws a substantial minority of them as women, while the face-byte
reading lands every one and draws NONE as a woman; inverting either axis breaks hundreds; and no
shipped `Humans` row sets bit 7 of its face column.

**SC-6** Every `<npc=N>` tag over the campaign, on both roots, names a record and resolves to a
picture that reads — split across the three answers, with zero unresolved.

**SC-7** Every npc record that resolves to a figure moves right under the decoded window, by a
margin consistent with the owner's own report of the defect; the split between records stating
their own window and records taking the default is the registry's.

**SC-8** `interface/SpellBook.bmp` decodes at 480x85 with all 24 cells inside it and none a single
colour, and `interface/SpellBack.bmp` decodes at 36x36 — exactly the cell.

**SC-9** Two reversion witnesses: restoring the bit-7 sex split fails tests, one of them on the
single line that a man and a woman of one face must not share a directory; and lowering one pane
pixel below full alpha fails the black-ground test.

**SC-10** The script-gap census is unchanged, which is the correct outcome for a story that runs no
new script node. The result this story owes is in `builds/current/` instead.

## Traceability

| Requirement | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-1 | SC-3 |
| FR-2, FR-3 | DD-2 | SC-3 |
| FR-4 | DD-5 | SC-2, SC-3, SC-8 |
| FR-5 | DD-3, DD-4, DD-15 | SC-4 |
| FR-6 | DD-15, DD-16 | SC-4 |
| FR-7 | DD-6, DD-7 | SC-5, SC-9 |
| FR-8 | DD-15, DD-16 | SC-4 |
| FR-9 | DD-8, DD-9, DD-10 | SC-6 |
| FR-10 | DD-11, DD-12 | SC-7 |
| FR-11 | DD-13 | SC-7, SC-9 |
| FR-12 | DD-10, DD-11 | SC-6 |
| FR-13 | DD-14 | SC-8 |
| FR-14 | DD-14 | SC-8 |
| every one of them | | SC-1, SC-10 |
