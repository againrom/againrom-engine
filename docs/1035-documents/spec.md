# Story `1035` — Valuable Documents: specification

**As built, 2026-08-23.** Branch `1035-valuable-documents`, base `1fcf7e0f`, research pin `d7ee0c6`.
This document states the behaviour that shipped. Claim provenance is in `contract.md`; the divergence
rows are `DIV-297` through `DIV-306` in `docs/DIVERGENCES.md` and `docs/DIVERGENCES-CLOSED.md`.

## B1 — the collection is campaign state and a save carries it

**The record.** One element is a `(value, kind)` pair. `kind` is 0 for a picture and 1 for a text
(`game.DocumentPicture`, `game.DocumentText`). The collection is an ordered list of such pairs held on
`game.Town`, which is this build's campaign-lifetime state.

**The source.** The campaign registry's `[Mission<n>]` sections carry two keys. `AddTextDocument`
grants text elements and `AddPictureDocum` grants picture elements. The second key is spelled with
fifteen characters because that is what the shipped registry contains: registry key names are
truncated to fifteen characters, so `AddPictureDocument` does not appear and a reader looking for it
finds nothing. Both are read into `game.Chapter.TextDocuments` and `game.Chapter.PictureDocuments`
through the existing `sectionIntSlice` reader.

The shipped corpus is identical on both roots: `[Mission10] AddTextDocument 1,2,3`, `[Mission50]
AddTextDocument 4`, `[Mission60] AddPictureDocum 1`. Four text elements over two sections and one
picture element over one, out of 24 `[Mission<n>]` sections.

**The grant.** `Town.CollectDocuments(n)` appends mission `n`'s own two lists. It is called once, from
`missionOpenerMode` (`pkg/game/frontend.go`), on the statement that opens a mission.

It is idempotent twice over. A monotonic guard refuses any `n` at or below the highest mission
already collected, and the append itself is deduplicated on the `(value, kind)` pair. Re-entering
mission 10 from a save grants nothing, and neither does replaying it. A section carrying neither key
still moves the guard, so the guard records how far the campaign has come rather than which section
last granted something.

Text elements are appended before picture elements within one section. No shipped section carries
both keys, so the order is unobservable on stock content and is fixed here rather than left to
whichever loop runs first.

**The save.** `game.Snapshot` gains `Documents []SnapshotDocument` and `DocumentMission int`.
`snapshotTown` writes the collection in grant order, unsorted, because the order is the collection's
own and sorting it would change what the panel shows first. `restoreTown` replays each element through
`Town.addDocument`, so a save carrying a duplicate pair restores one element.

This is a `gob` field addition inside the existing `AGRMSAVE` envelope. An older payload decodes with
both fields zero, which is an empty collection and a campaign that has collected nothing. The
encode-side envelope fixture hash moves because the gob type descriptor grows; the world hash does
not move, because no simulation state changed.

## B2 — the panel

**The frame.** The panel composes a fresh 640x480 image. It is a `ui.Screen` in its own right,
`ui.ScreenDocuments`, registered in `pkg/ui/screenregistry.go` with its selection test and two
geometry tests, and composed by `composeDocumentsPanel` through `composeScreen`. It is not an overlay
over the running mission (`DIV-299`).

**The art.** Eleven bitmaps, all from `graphics.res`:

| Piece | Address | Size |
|---|---|---|
| sheet | `graphics/interface/docs/sheet.bmp` | 640x480 |
| left arrow, three states | `graphics/interface/docs/arrows/00_l.bmp`, `01_l.bmp`, `11_l.bmp` | 56x40 |
| right arrow, three states | `graphics/interface/docs/arrows/00_r.bmp`, `01_r.bmp`, `11_r.bmp` | 60x40 |
| OK, four states | `graphics/interface/docs/ok/ok_off.bmp`, `ok_on.bmp`, `ok_l_off.bmp`, `ok_l_on.bmp` | 44x32 |

The array order is the selection index and is load-bearing: sorting the arrow files by name would swap
`01` and `11`. Every bitmap's size is checked at load, because each control's hit rectangle equals its
own bitmap, so a wrong-sized file would move a control's picture off its hit test with nothing else
failing. A missing or mis-sized node makes `LoadDocumentPanelArt` return an error naming the address,
and the front end then installs no panel art at all rather than a partial one.

**The rectangles.** Left arrow `(0,200)-(56,240)`, right arrow `(576,200)-(636,240)`, OK
`(560,416)-(604,448)`. They do not overlap. The document content origin is `(92,72)` and the content
width is 456.

**Control states.** An arrow draws bitmap 2 while it is the pressed control, 1 while it is hovered and
nothing is pressed, and 0 otherwise. OK draws 3 pressed, 1 hovered, 0 otherwise; `ok_l_off.bmp` is
loaded and never drawn (`DIV-301`).

**Draw order.** Sheet, left arrow, right arrow, OK, then the current element. The order matters where
the picture overhangs its rectangle: the document is drawn last, so it lies over the sheet.

**Paging.** One arrow pair walks two axes and the order is the whole behaviour. An arrow steps the
line offset by 21 lines, bounded by the current element's line count, and refuses rather than clamps.
The element changes **only when that page step refuses**. A picture element has no lines, so its page
step always refuses and an arrow over it always moves to the next element.

A new element is entered at its first line in both directions (`DIV-300`).

Both ends of the collection refuse rather than wrap: the right arrow on the last element's last page
and the left arrow on the first element's first page do nothing.

**Text drawing.** A text element is wrapped once, when the panel is built, to the 456-pixel content
width. The line pitch is the font's own height plus 2. The panel draws at most 21 lines starting at
the current line offset, the first at `(92,72)` and each next one pitch lower.

**Picture drawing.** A picture element is blitted at `(92,72)` at its natural size and is not resized.
The one shipped picture is 464x344, eight pixels wider than the 456-pixel content rectangle, and that
overhang is kept.

**Closing.** OK closes the panel and returns to the screen it was opened over. Escape does the same
(`DIV-304`).

**Rebuilt at every open.** `flow.openDocuments` builds a new panel from the collection as it stands
and drops it on close, so a panel opened after a later mission granted more elements shows them, and
no page position survives a close.

**Refusal.** `openDocuments` returns false with no document source installed and false with an empty
collection. A refusal leaves the screen where it was, so an unusable item does nothing rather than
opening an empty sheet.

## B3 — the two element kinds load from their own paths

Kind 0 resolves to `graphics/interface/docs/<value>.bmp` and kind 1 to
`main/text/docs/<value>.txt`, the first path component naming the archive. Addresses in this tree are
lower case with forward slashes, which is the container filesystem's own normalisation and not a
change of address.

A text element is read whole through `ReadShopTip`, the same reader every other install text in this
tree goes through, and wrapped in `pkg/ui` where the font is. A picture is decoded to RGBA and handed
over at natural size.

**An element that does not resolve is dropped, not drawn blank.** A value naming no shipped file is
not a page the player can be shown, and a blank sheet in the middle of the collection is worse than
one fewer element. The five values the shipped registry carries resolve on both roots, so this arm is
reached by authored content alone.

**The text is install text.** The panel draws the bytes the install carries, in the install's own
language, through the font's language selector. `pkg/ui/words.go`'s membership rule does not apply
here, because the words are the document's and not the program's.

**The font is font4.** Its atlas is `graphics/font4/font4.16a` and its advance sidecar is
`graphics/font4/font4.dat`. font4 is the only shipped font whose atlas is in the `.16a` class; both
roots carry `font4.16a` and `font4.dat` and no `font4.16`. `LoadFontA` decodes the atlas with a
declared 1024-byte leading palette, refuses an atlas with no records, and refuses an advance count
that does not equal the record count. Both roots decode to 224 records. The palette index of each
pixel is dropped and its 4-bit level is kept (`DIV-303`). The font carries the install's own language
selector, so a Russian install's CP866 bytes draw as the right glyphs.

## B4 — the item raises the panel

**The gate is a mask, not one code.** `data.RaisesDocuments(c)` answers true when the code's class
field is 14, the carried class, and its low five bits equal 28. The class field spends the whole low
byte on the row for this class, so rows 28, 60, 92, 124, 156, 188, 220 and 252 all pass for any
material nibble. A build testing `code == 0x0e1c` would be a narrower gate than the original's, and
the difference is invisible until content uses another row. Only row 28 has shipped content.

**The gesture is the equip gesture.** `enqueueEquip` (`pkg/game/world.go`) tests the gate before
anything else and raises a one-shot request on the viewer instead of enqueueing an equip. Nothing is
worn and nothing leaves the container (`DIV-297`). Every route that reaches `enqueueEquip` therefore
reaches the panel: a double-click on the pack cell, and a drag from that cell released over the doll
box, which raises the same request.

**The drain is one statement.** `App.step`'s `ScreenMap` arm takes the request immediately below the
popup gate, opens the panel and returns, because every statement below it on that arm reads a map
screen that is no longer showing. A refusal falls through and the rest of the arm runs unchanged. This
is this build's reading of the original's `campaign+0x3dc == 1` (`DIV-298`).

## B5 — the player starts with it

**The item.** `data.QuestDocumentCode` is `0x0e1c`, 3612. Its English name is `Valuable Documents` and
its Russian name is the Russian install's own, both read from the install's name table through the
existing key path. No name string for it appears in this repository.

**The grant.** `partyInputs.Documents` appends the code to the primary member's carried list.
`ChargenParty` sets it, so every newly generated character starts with it in his pack. `MissionPartyAs`
sets it as well, so a party assembled for a mission outside generation carries it too. It is carried
and never worn: `DAT-DOC-021` establishes that the `Humans` equipment grammar has no MagicItems arm,
and this build's `EquipSlotFor` refuses class 14.

This is a deliberate deviation from a positive research finding, recorded in `DIV-305` with the
**Owner directive** cell filled.

**The shop cannot take it.** `shopItemPrice` answers zero for class 14, which is `SHOP-SELL-010`'s own
`elem+0x1c` test. `Shop.Sell` now keeps a zero-price place on the table and credits nothing for it,
where before this story it handed every staged place of the customer's to the merchant once the
basket's total was non-zero. That defect predates this story and was reachable only with a mission
script's own handout; putting this item in every new character's pack made it reachable with an item
the player cannot get back. The row is `DIV-306`, closed at this landing.

## Witnesses

**Unit.** `pkg/ui/documents_test.go` renders the whole panel through the production composer against
independently built expected frames, at the pixel level, and covers the control state selection, the
page and element walk, and both ends' refusal. The two registered `GeometryTests` entries were
mutation-proved at the composer's own draw call sites, seven mutations, all killed, reverted
byte-identical.

**Integration, on shipped content, both roots.**
`scenarios/1035-documents-mission10.json` generates a character, enters mission 10, asserts the
collection is three elements and the pack carries the access item, double-clicks it, and then walks
the panel: forward to the next element, back to the previous element's first page, refused at the
collection's first end, one page within an element, twenty steps forward to the last element, refused
at the far end, closed with OK, re-opened at the first element, and closed with Escape.

The scenario asserts the panel's own element index, element count, page index and kind, not its page
count: the page count is language-dependent, because the panel wraps the install's own text with the
install's own font, and element 1 fills four pages on the English root and five on the Russian one.

**Visible.** `cmd/screenshot -screens documents` reaches the panel through the production
double-click and writes the composed frame. It captures on both roots.
