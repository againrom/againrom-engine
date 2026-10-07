# Story 1027 — spec (as built)

The statistics card is the 160x242 `CompactPanelLayout` composition built by story `1022`. Two
callers compose it: `DrawTownCharacterRegion` (`pkg/ui/townshell.go`), which draws the town's
character pane in the tavern, the school and the shop, and the character generator's detailed page
(`pkg/ui/chargen_page.go`). This story changes the town caller's font, the shop's chrome over the
card, and the shared layout's treatment of a centred row. It changes no ROM1-derived value.

## B1 — the card composes at its own font

`TownCharacterView` carries two fonts.

- `Font` is the town shell's chrome font. It stays `(*FrontEnd).Font`, which resolves
  `DefaultFont = "font1"`, a 16x15 cell. The chevrons (`townCharacterPrev`, `townCharacterNext`),
  the mode box (`townCharacterMode`) and the shop's `Book` label draw with it. The later
  owner-directed name-card hotfix removes DOLL mode's separate member-name row.
- `CardFont` is the statistics card's font. `pkg/game/townshell.go`'s `townCharacterView` sets it
  from `(*FrontEnd).tipFont()`, which returns `ChargenAssets.Presentation.Font` — font2, an 8x10
  cell — and falls back to `Font` when the chargen assets did not load. That is the same seam story
  `1018` built for the tip panel; this story adds no second resolution path.

`DrawTownCharacterRegion` composes the card with `v.cardFont()`, which is `CardFont` where the
caller set one and `Font` otherwise. A caller predating this story composes exactly the card it did.
`shopCharacterView`'s fallback view carries `CardFont` through.

Measured on the `en` root, the shipped starting party member Danath, before and after:

| | font1 (before) | font2 (after) |
|---|---|---|
| left column's widest cell | 120 px | 58 px |
| shared right-column origin | 126 | 64 |
| right-column budget (`Size.X - 2*Pad.X - rightX`) | 16 px | 78 px |
| right-column cells carrying a value | 11 | 11 |
| of those, drawn as stated | 0 | 11 |
| rows whose right edge passes the card's usable width (151) | 12 of 16 | 0 of 16 |
| rows placed past the card's last usable row (y=224) | 4 of 16 | 0 of 16 |
| rows placed past the card's own 242-pixel height | 2 of 16 | 0 of 16 |

At font1 nine of the eleven value-bearing right cells were shortened to the empty string and two to
a single character (`145/145` to `1`, `0/0` to `0`). The twelve right-column labels ran past the
card and the font clipped them at x=160, so the player read `HE`, `MA`, `AB`, `DE`, `RE`, `FI`,
`WA`, `AI`, `EA`, `AS`. The SIGHT and SPEED rows were placed at y=242 and y=258 on a 242-pixel card
and painted nothing.

## B2 — every value is drawn

At the production font no cell of the town card is shortened: for every laid-out row,
`PanelLineReport.Value == FullValue` and `RightValue == FullRightValue`. All sixteen rows are placed
inside the card, the lowest at y=183 with its ink ending at y=192, and the card's whole text ink
spans (9,19)-(129,192) inside the padded box (9,18)-(151,224). Three right cells are empty, and
those are the HEALTH, MANA and RESISTANCE headings, which state no value in any composition.

## B3 — the shop draws nothing over the card

`shopBookRect` is `(548,242)-(632,272)`. `TownCharacterRegion` begins at `(480,238)`, so the control
occupies card-local `(68,4)-(152,34)`, which contains the card's name row and the top of its
BODY/HEALTH row. Story `1022` round 3 filled that rectangle opaquely whenever Statistics was on, to
keep a transparent label off the card's digits, and thereby covered the name row that same round had
stopped drawing separately because the card carries it.

The control moves out of the mode rather than to another rectangle. `ComposeShopScreen` draws
neither the plate, the outline nor the `Book` label while `character.Statistics` is true, and
`shopScreenControlAt` no longer answers `ShopControlBook` there.

Two reasons, and the second is why no other rectangle was chosen:

1. `Book`'s display is the five table cells at `shopTableRegion`. Statistics mode paints
   `drawTownStatisticsSurface` over that whole region before the character pane is drawn, so a press
   on `Book` in that mode already changed nothing the player could see.
2. The card occupies the whole pane. Its sixteen rows run to card-local y=193 and the three
   persistent controls below them start at y=205, so the region has no free 84x30 area.

`ShopScreenView.Book` itself is untouched. A book opened before Statistics was entered is still open
when it is left, and `Book` draws and answers normally in DOLL mode.

## B4 — the name row is centred

`PanelRow.Center` and `PanelLayout.AlignValues` disagreed. `AlignValues` gives every non-empty value
in a column one shared left edge; the name row's own value was moved out to that edge, which made
`panelLine.width` measure from the box's left inset to the edge rather than the name's own run, and
the centring division then centred that wider box. The name sat at its right-hand end.

A centred row is not part of the aligned column. `layoutLines` now skips a centred item in both
places: it does not contribute to `leftEdge`, and it does not take the shift. Its `valueX` stays at
the cell's own origin and its `width` is the name's own measured width, so `(Size.X - width) / 2`
centres the name.

Measured through the production composer, the name row's ink:

| call site | before | after | card centre |
|---|---|---|---|
| town character pane | x=[73,108), centred at 90.5 | x=[62,97), centred at 79.5 | 80 |
| generator detailed page | x=[73,108), centred at 90.5 | x=[62,97), centred at 79.5 | 80 |

The two agree because the centred position depends on the card's width and the name's width, and on
neither layout's own `Pad`, which differ (the town pane's background settles at `Pad=(9,18)`, the
generator's at `(20,18)`).

## B5 — the generator page is otherwise unchanged

The generator already composed at font2 and this story does not touch its font. Its card moves by
B4's centring alone, which is the correction B4 requires at both call sites. `pkg/ui`'s own suite
passes unchanged; the generator's install-gated witnesses pass on both roots.

## Reporting seam

`ui.CharacterPanelReport(l, f, s)` returns the rows `RenderCharacterPanel` would paint, as
`[]PanelLineReport`: each row's origin, its two runs of text, the pen offsets they draw at, the
row's own right edge, and `FullValue`/`FullRightValue`, the two values before `panelFitValue`
shortened them. It calls `panelItems` and `layoutLines`, the pair `composeItems` itself calls and in
that order, so a caller reads the production result rather than a second derivation of it. It exists
because a truncated cell has no pixels to observe, and `PanelStatement` reports what the panel means
to state rather than what it draws.

`cmd/paneldump` prints the cut rows for whatever layout it composes, with each shortened value's own
measured width and the budget it exceeded.

## What is not changed

- The card's row order, its field set and its arrangement. `DIV-191`, accepted on the owner's word.
- The WEIGHT row. `DIV-209`, story `1025`.
- The mission panel's own truncation. Measured and recorded as `DIV-218`; what the mission column
  does with a row wider than its budget is story `1023`'s.
- `AuthoredPanelLayout`, `sidebarWidth` and `panelFitValue`'s own rule.
- Anything in `pkg/sim`. A font is a presentation value; no hashed simulation state is reachable
  from this story.
