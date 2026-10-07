# Story `1036` — spec

The behaviour this story shipped. Canonical at the landing: where this differs from `contract.md`,
this is what was built.

Claim provenance is `contract.md`'s own corpus table. Every mismatch with researched ROM1 behaviour
is a row in `docs/DIVERGENCES.md`; this file states what the build does and cites the row.

## 1. One widget, two parents

The mission character pane and the town character pane are one implementation.

`pkg/ui/characterpane.go` holds the whole geometry and every gate, as functions of a pane rectangle
and a session mask. It draws nothing and reads no viewer:

- `characterPaneRects(pane)` — the six rectangles, panel-relative, in `TOWN-346`'s own test order.
- `CharacterPaneCornerLive(pane, session, screenH, corner)` — the hit gate.
- `characterPaneCornerDrawn(pane, session, screenH, corner, pickerSuppressed)` — the art gate, a
  separate function because the two disagree for four of the six (`DIV-309`).
- `CharacterPaneCornersAt(pane, session, screenH, p)` — every rectangle a point reaches, in test
  order, because no arm returns between the six (`DIV-310`).
- `characterPaneArtRect(pane, corner, alt)` — each bitmap's own destination, which is not the hit
  rectangle for four of the six.
- `characterPaneFigureOrigin`, `characterPaneFigureRect` — the 160x200 painted crop from `(L, T+2)`.
  The shared pane paints no separate name row; Statistics owns the subject name.

`pkg/ui/townshell.go`'s `DrawTownCharacterRegion` composes the pane from a `TownCharacterView` and is
the only composer. The town screens call it as before. The mission reaches it through
`pkg/ui/panel.go`'s `panelPresent`, which composes the pane into its own 160x242 image and returns
it for the frame's own id-7 slot; `pkg/ui/missionpane.go` builds the view.

Every constant the town screen used to declare is now derived:

```
townCharacterFigure = characterPaneFigureRect(TownCharacterRegion)
townCharacterPrev   = CharacterPaneCornerRect(TownCharacterRegion, CharacterPanePrev)
townCharacterNext   = CharacterPaneCornerRect(TownCharacterRegion, CharacterPaneNext)
townCharacterMode   = CharacterPaneCornerRect(TownCharacterRegion, CharacterPaneMode)
```

The first four are byte-identical to the literals they replace. The fifth moved: see 4 below.

## 2. The mission has a doll

Before this story the mission drew the statistics card in id 7's slot and a separate authored doll
box under it, at `y` 488 with height 260. `rightColumnBox` refuses that box at 640x480 and at
800x600, so at both resolutions the owner plays at, **the mission had no doll at all**.

The separate box is removed. `dollBoxRect`, `dollBoxSize`, `dollFigureRect`, `RenderDoll`,
`dollPresent` and the viewer's second presentation cache are gone; `hudLowerStack` returns two
positions where it returned three. The doll is the pane's own figure presentation mode, inside the
242 the panel already occupies, which is where `TOWN-349` and `TOWN-350` put the equipment
presentation. `DIV-200` records what the emptied fourth slot now owes.

The worn box moved up with it, from `y` 756 to `y` 488, and draws at 1024x768 for the first time.
640x480 and 800x600 still refuse it. `DIV-316`.

## 3. The two presentation modes

`TOWN-351` reads `this+0x70` as the pane's presentation mode: the constructor forces it to 1, rect C
inverts it, and `L11936` skips the statistics text when it is non-zero.

This build stores it as `hudPanelDoll`, which is one of the four HUD display switches and is stored
inverted so SHOWN is the zero value. A fresh viewer therefore starts on the doll exactly as a freshly
constructed widget does, and `hudShown(hudPanelDoll)` **is** `this+0x70 != 0`. There is no second
field.

Three routes flip it: rect C, the `D` key this project already had, and `Tab`. Tab is `TOWN-355`'s
own keyboard route — `R1249` posts `0x412` when its `wParam` is 9, and `wParam` of
`WM_KEYDOWN` is a virtual-key code — and `AI-KEY-125` names the same message as the focused character
panel's Tab. Tab was unbound everywhere in `pkg/` before this story.

The body art follows the mode: `HumanBackR.bmp` behind the figure, `TextBackR.bmp` behind the
statistics card. `TOWN-354` states `this+0x70` chooses between `[L11975]` and `[L11976]` and
`TOWN-356` names those two globals, which confirms the pairing `DIV-175` had inferred from frame
shape.

## 4. The six rectangles

Panel-relative to the pane rect, so they hold at every resolution. `L`, `T`, `R`, `B` are the pane's
own edges.

| | Rectangle | Hit gate | Posts | This build |
|---|---|---|---|---|
| A | `(L, B-0x28)-(L+0x1c, B)` | `sess & 1` | `0x40e` | toggles the pack bar (`DIV-307`) |
| B | `(L, T)-(L+0x1c, T+0x24)` | `sess & 3` | `0x40f` | toggles the spellbook (`DIV-307`) |
| C | `(L+0x80, T)-(R, T+0x24)` | `!(sess & 0x600)` and `!flag` | `0x412` | inverts the presentation mode |
| D | `(L+1, T+0xcd)-(L+0x21, T+0xed)` | `sess & 0x226` | `0x414` | previous party member |
| E | `(L+0x77, T+0xcd)-(L+0x97, T+0xed)` | `sess & 0x226` | `0x415` | next party member |
| F | `(L+0x7e, T+0xce)-(L+0x9e, T+0xee)` | `sess & 1` | `0x416` | raises the in-mission menu |

`flag` is `screenH - B > B - T`. `TOWN-346` states that its own last step assumed the containers
above `campaign+0xd4` contribute no vertical origin and that this was not read; this build computes
the comparison directly from the pane rect it places. At 640x480 the pane's bottom is the frame's
bottom, so rect C is live; at 1024x768 the space below is 288 against a height of 242, so rect C is
neither live nor drawn. `DIV-312`.

Rect F raises exactly what Esc raises: `MENU-ESC-010` reads `0x416` on the frame window's own
`VK_ESCAPE` arm under `campaign+0x3dc == 1`. `pkg/ui/missionpane.go` holds the request and
`pkg/ui/app.go` drains it with `TakeCharacterPaneMenu` in the same frame the command path runs, so
the menu opens on the release rather than one frame later.

**Every rectangle a point reaches fires, not the first.** Rect A overlaps rect D over 27x32 pixels
and rect E overlaps rect F over 25x31, and `R0332` has no early return between its six tests,
so one click in either overlap posts two messages. `DIV-310`.

## 5. The session mask

`campaign+0x3dc` is a bitmask, not an enumeration (`TOWN-352`). This build passes:

- `CharacterPaneMission` (1) on the mission screen. Rects A, B, C and F are live; the party picker's
  D and E are not, because `1 & 0x226 == 0`.
- `CharacterPaneShop` (2) on every town-side screen that composes `TownCharacterRegion` — tavern,
  school, shop, generator. Rects B, C, D and E are live; A and F are not.
- `CharacterPaneShopInMission` (3) exists and no screen passes it yet.

Which value the tavern, school, square and generator carry is not decoded; 2 is authored.
`DIV-311`.

## 6. The held item pre-empts all six

`TOWN-348`: `R0332` inspects `[sess+0x3cc]`, the cursor-held item, before any point-in-rect
call, and returns early on six of its `+0x18` values.

`v.dragActive` — a drag that has crossed `TapSlop` and is carrying an icon — is this build's cursor-
held item, and it skips the whole corner arm in `pkg/ui/command.go`. A release over a corner with an
item in flight falls through to the inventory's own drop resolution, as the original falls through to
`vt+0x7c`. The drop-slot **choice** on values 1 and 2 is not reproduced: `DIV-315`.

## 7. A corner and the figure share pixels, and the gesture decides

Rect B and rect C are the only two of the six inside the 160x200 painted crop; A, D, E and F are
entirely below it.

The original never contends: the six rectangles are `WM_LBUTTONUP` and the slot map is sampled by
`WM_LBUTTONDBLCLK` and `WM_MOUSEMOVE`, so a single left click on a shared pixel is the corner and
nothing else. This build reaches both from one button, so it splits them by gesture:

- a press and release on a shared pixel runs the **corner**, which is the original's answer for the
  single click;
- a press that travels past `TapSlop` arms the **figure's** own drag, and once `dragActive` is set
  the corner arm is skipped entirely.

Nothing subtracts pixels from the slot map. `shopDollSlotAt` and `dollFigureSlotAt` answer over the
whole crop exactly as before this story, which is what
`TestShopDollHitAreaMatchesEveryVisibleSyntheticFigurePixel` holds. `DIV-308`.

## 8. The art

`pkg/game/characterpane.go` resolves nine bitmaps from `graphics\interface\` — `BackPackOp`,
`BackPackCl`, `BookOpened`, `BookClosed`, `HumanMode`, `TextMode`, `diskette`, `ar1`, `ar2`
(`TOWN-356`) — through `readChargenBMP` with `keyBlack`, and caches them on the front end on
`characterPanes`' own precedent. `pkg/ui` opens no archive.

**Both parents get the same nine.** The mission takes them through `SetCharacterPaneArt`, beside the
command panel's four; the town screens take them on `TownCharacterView.CornerArt`, set in
`pkg/game/townshell.go`. `TOWN-353` makes the art a property of the widget and not of the screen it
is re-parented into. On the town side `TOWN-354`'s gates then draw rect B's spellbook, rect C's mode
medallion and the picker's `ar1`/`ar2`, and leave rects A and F unpainted. The authored chevrons and
the authored `DOLL`/`STATS` box the town drew before this story are gone.

**No size is asserted.** The blit size is an operand of the paint routine, not a property of the
file, and `characterPaneArtRect` carries it.

Rect A and rect B each have two bitmaps at two different destinations, chosen by the state of the
surface that corner opens: `BackPackOp` at `(L, T+0xd0)` 32x31 when the pack bar is up and
`BackPackCl` at `(L+1, T+0xc9)` 28x30 when it is down; `BookOpened` at `(L, T)` 28x38 when the
spellbook is up and `BookClosed` at `(L, T+4)` 28x37 when it is down. Rect C takes `HumanMode` or
`TextMode` at `(L+0x80, T+4)` 28x32 by the presentation mode.

A pane with no art still composes. The authored fill and outline `DrawTownCharacterRegion` drew
before story `1021` remain, and the corners fall back to this project's own chevrons and outline —
**except for rect B and rect C**, whose fallback chrome would stamp over the painted figure. Those
two draw nothing when their bitmap did not read, and stay live.

**The statistics card's padding is re-measured against whatever body it is composed on.**
`CompactPanelLayout`'s `Pad` is `compactPanelInset` of its own background, and a `Viewer`'s default
layout is built with none, so it carries the `(4,3)` fallback. `DrawTownCharacterRegion` substitutes
the pane body for a nil `Background` and now re-measures `Pad` with it, giving `(9,18)` against
`TextBackR.bmp` on both screens. Without that the mission card sat 15 pixels high: the name row
landed on the frame's top ornament and two left-column labels went under rect B's bitmap.

The corners are drawn **last**, after the body and the figure or card. The separate member-name row
was removed by the owner-directed name-card hotfix; its former pixels remain part of the doll and
equipment interaction surface.

## 9. What is not built

- The eighth interactive behaviour of `TOWN-344`: the coordinate-free `WM_RBUTTONUP` posting `0x405`
  to the map view. `DIV-217`.
- Sound `0xdc` on rects C, D, E and F, and rect C's `[mapview+0xe0] = 1`. `DIV-314`.
- The `sess+0x6bc == 2` half of the picker art suppression. The parameter exists and every caller
  passes false. `DIV-313`.
- `TOWN-348`'s drop-slot choice. `DIV-315`.
- A second mouse route to the slot map, which would end the gesture split of 7. `DIV-308`.

## 10. Two collisions between decoded art and authored chrome

Both are recorded as ledger amendments rather than new rows: this story's reserved divergence range
`DIV-307`..`DIV-316` is spent.

- **`shopBookRect` overlaps rect C.** The shop's authored `Book` plaque is `(548,242)-(632,272)` and
  rect C is `(608,238)-(640,274)`. `shopControlAt` tests the six corners first, so the mode control
  answers over the whole overlap and the Book control keeps `x:[548,608)`, which carries its whole
  label. Drawn, the plaque's outline crosses the medallion, which stays visible under it. Amends
  `DIV-217`, which already names `shopBookRect` as having no decoded counterpart.
- **`BODY` is clipped by rect B's bitmap** on both panes, at every shipped resolution. The card is
  authored and the bitmap is decoded, and the card cannot move: its padded box is `(9,18)-(151,224)`
  inside 160x242, the two columns leave three pixels of horizontal slack, and lowering the rows by
  the eleven pixels the top corners need pushes the last right-column row under rect F's bitmap.
  Amends `DIV-175`.
