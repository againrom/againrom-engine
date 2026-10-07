# 1016 — spec

As-built. Canonicalized to the behaviour this story shipped.

## FR-1 — the town square draws the shipped picture

`ComposeTownSquare` (`pkg/ui/townsquare.go`) paints, in order:

1. `graphics/interface/town/townmain.bmp` (640x480, 24-bit), opaque, filling the canvas.
2. `graphics/interface/town/town_add.bmp` (552x92, 24-bit), keyed (round-3 review, P-2; see below),
   at `TownSquareAddOrigin`, drawn with `draw.Over`.
3. The three door labels (FR-3), opaque, each at its own measured origin, drawn with `draw.Src`.

`TownSquareAddOrigin` is `(0, 0)`. It is measured, not authored: `cmd/townsquarecheck` correlates
`town_add.bmp` over the whole base picture and reports the winning offset against the next-best
fraction anywhere else in the search window. On both preserved installs the winner is `(0,0)` at
fraction `1.0000` — every one of its 30140 opaque pixels is byte-identical to the base picture
there — against a next-best fraction of `0.1903`. The base picture already carries this exact
rooftop strip. The story still implements the literal compositing the install ships, both because
the contract calls for it and because a single sampled game state cannot rule out the overlay
differing from the base in some other state this story does not reach (`DIV-149`).

`town_add.bmp` is 40.7% pure black: 20644 of its 552x92 = 50784 pixels, the sky above the rooftops
its own 552x92 footprint does not cover. `townmain.bmp`, the base picture, carries none at all (0 of
307200 pixels). `LoadTownSquareArt` (`pkg/game/townsquareart.go`) keys `town_add.bmp` with
`keyBlack` — the shop's own rule (`pkg/game/shopart.go`) for this shape of art: pure black is the
transparent colour — before `ComposeTownSquare` ever sees it, and the compositor draws it with
`draw.Over`. Of its 50784 pixels, the 30140 that are not pure black are the ones the correlation
above measures as byte-identical to the base at this offset; the 20644 that are pure black are its
own transparent surround, and keying them out is what makes the compositing a genuine no-op on the
currently sampled art: every opaque pixel duplicates the base, and every pixel that would have
differed (the sky) is transparent and paints nothing. Before this fix, `ComposeTownSquare` drew the
overlay with `draw.Src`, opaque and unconditional, so those 20644 pixels landed as a black band over
the base picture's own sky (round-3 review, P-2;
`pkg/game/townsquare_release_test.go` is the install-gated witness).

The three door labels carry no pure-black pixel at all (`cmd/townsquarecheck` counts zero across all
three), so they are drawn opaque and unkeyed, the way the school's own rest faces are
(`LoadTownSchoolArt`): standing in for whatever the base picture already shows there. They are not
keyed as a precaution against a defect they do not exhibit; keying a correct path would be an
unmeasured change to it.

Any of the six art nodes failing to load leaves `ComposeTownSquare` undrawn: `app.go` calls it only once
`townSquareView` confirms `Art.Background` is non-nil (readiness test, FR below), and the previous
row-button layout draws instead.

## FR-2 — a click enters the building under the cursor by raster mask

`TownSquareControlAt(mask *image.Paletted, p image.Point) (TownSquareControl, bool)` reads one pixel
of `graphics/interface/town/townmask.bmp` (640x480, 8-bit indexed) and answers which of five
interactive regions, if any, that pixel belongs to. It reuses the school's own exact-code,
no-tolerance rule (`schoolMaskSlot`, `pkg/ui/townshell.go`; `TOWN-003`, `TOWN-017`) rather than a
second mechanism: a code this table does not carry answers no hit at all.

The five significant colour codes (more than 1000 pixels each on both preserved installs) and their
resolution:

| Code | Region | Resolves to |
|---|---|---|
| 128 | tavern doorway | `TownSquareControlDoor`, `Door: 0` |
| 144 | shop doorway | `TownSquareControlDoor`, `Door: 1` |
| 192 | school archway | `TownSquareControlDoor`, `Door: 2` |
| 160 | gate arch | `TownSquareControlDoor`, `Door: 3` |
| 176 | Gilded Statue | `TownSquareControlMenu` |

`Door` is the row index `pkg/game`'s `townDoors` array and `Choose(i)` already accept (0 tavern, 1
shop, 2 school, 3 gates) — the same seam the row-button grid always fed. A door hit calls
`townList.Select(c.Door)` then `chooseTown()`; a menu hit calls `openGameMenu(ScreenTown)`, the same
call the pre-existing Escape key already makes at the square (FR-5).

This mapping was not published by any research claim when the story was built. It is measured
here two independent ways, and the pin bump at the landing added a third, from the executable
(`DIV-148`):

1. **Correlation.** Each door label (FR-3) is correlated onto the base picture at its winning
   offset; the mask code with the most pixels under the label's own footprint is read as that
   label's door. `shop_l` → code 144 (3883 of 3952 opaque pixels, 98.3%); `tavern_l` → code 128
   (1783 of 1792, 99.5%); `trener_l` → code 192 (13210 of 16240, 81.3%).
2. **Shipped text.** `main/text/tips/town.txt` names the shop centre, the inn (tavern) left, the
   school right, the gate, and the Gilded Statue: five names for the five codes with non-trivial
   area. A colour overlay of the mask on the base picture (`cmd/townsquarecheck -png`) shows five
   regions matching that text: the tavern's own archway (left, dark), a stone gate arch in the
   background between the tavern and the shop, the shop's own doorway (centre), the golden statue
   (centre foreground), and the school's own archway with its training column (right).

A third reading, from `EXP-0196` at the pin this landing bumps to, agrees with both. `TOWN-163`
decodes the sampler and gives the five admitted mask bytes `0x80`, `0x90`, `0xa0`, `0xb0`,
`0xc0` — the same five — returning `2`, `1`, `8`, `16`, `4`. `TOWN-161` gives the three
door labels as gated on `+0xb4` tested against `1` (shop), `2` (tavern) and `4` (school): the
same three constants, for the three codes this table calls shop, tavern and school, while the
two returns absent from that comparison, `8` and `16`, are the two regions with no label.
`TOWN-164` gates only the `0xa0` case on a mission-validity check, which is the gate arch, and
leaves `0xb0` as the one case that sends no internal `0x445`, which is the statue rather than a
room. Neither claim states a code-to-room mapping on its own; the three read together corroborate
every row of the table above.

Both of the story's own readings agree, and both were confirmed against production rather than
assumed:
`ui.TownSquareControlAt`, called with the region's own anchor pixel over the install's own mask
loaded through `game.LoadTownSquareArt`, answers the same door or menu kind on both installs
(`cmd/townsquarecheck`'s `hit-test` lines). A region's own bounding-box centre is not always inside
the region — the gate and tavern regions are thin, arch-shaped outlines whose bounding box mostly
covers other codes and background — so the anchor used is the region's own pixel nearest its
bounding box's centre, not the raw midpoint.

## FR-3 — three door labels draw at their own measured positions

| Label | Address | Size | Origin | Best fraction | Next-best |
|---|---|---|---|---|---|
| `TownSquareLabelShop` | `graphics/interface/town/shop_l.bmp` | 52x76 | `(264, 264)` | `0.7715` | `0.1546` at `(264,269)` |
| `TownSquareLabelTavern` | `graphics/interface/town/tavern_l.bmp` | 28x64 | `(144, 332)` | `0.5893` | `0.1618` at `(144,338)` |
| `TownSquareLabelSchool` | `graphics/interface/town/trener_l.bmp` | 140x116 | `(436, 300)` | `0.6884` | `0.0951` at `(434,295)` |

`townSquareLabelOrigin` (private) carries these three points; `TownSquareLabelOrigin(i)` exports
them for `cmd/townsquarecheck`. None of the three fractions reaches 1.0, unlike `town_add.bmp`'s
exact match: each label's own best offset clears its next-best candidate by a factor of 3.6 to 7.2
(shop 5.0x, tavern 3.6x, school 7.2x), which is decisive placement even though the match is not
pixel-exact. **The pin bump at the landing confirms all three exactly.** `TOWN-161` gives the
three labels' blit destinations as `[+0xc]`/`[+0x8]` operand pairs — shop
`([+0xc]+0x108,[+0x8]+0x108)`, school `([+0xc]+0x12c,[+0x8]+0x1b4)`, tavern
`([+0xc]+0x14c,[+0x8]+0x90)` — without stating which operand is x. Reading `[+0x8]` as x and
`[+0xc]` as y yields `(264,264)`, `(436,300)` and `(144,332)`, identical to all three origins in
the table above, which were measured from art alone with no knowledge of these operands. The
three agreements fix the operand roles and confirm the placements at once (`DIV-150`). The residual is anti-aliasing at the label's own edge against whatever the base picture
shows there, confirmed by inspecting the generated diff images (`cmd/townsquarecheck -png`): the
mismatch is a thin border around each label's silhouette, not a scattered or offset pattern.

None of the three labels carries a pure-black pixel (`shop_l` 0, `tavern_l` 0, `trener_l` 0, over
their own canvases). This is the same signal the school's own rest faces carry (`LoadTownSchoolArt`)
for "always opaque, standing in for the base rather than keyed onto it," as against the skill
patches' own pure-black-keyed convention (`DIV-013`). All three labels are drawn unconditionally at
every visit to the square, with `draw.Src`, not gated on any game state: no state this build tracks
correlates with their presence, and the art itself carries no transparency convention that would
suggest a keyed, sometimes-absent overlay. This is a decision made from the art, not from a claim
(`DIV-150`).

## FR-4 — the gate region opens the world map

`Door: 3` (mask code 160) feeds `Choose(3)` at the square, which is unchanged production code:
`townDoors[3].room == roomGates`, and `Choose`'s own `roomSquare` arm already calls
`t.enterWorldMap()` when the resulting room is `roomGates` (story 1013). This story adds no new
call; it adds the path that reaches the existing one from a mask hit instead of only from a row
button. Confirmed by driving a real campaign town: `cmd/townsquarecheck`'s `drive code 160 door 3
Choose -> header "the gates ..." contains "gates" ok`.

## FR-5 — the statue region opens save/load

`TownSquareControlMenu` (mask code 176) calls `a.flow.openGameMenu(ScreenTown)` — the same call the
pre-existing Escape key already makes while at the square (`pkg/ui/flow.go`'s `escape()`,
`ScreenTown` arm). The in-game mini-menu it opens already carries SAVE and LOAD (story 0143); this
story adds no new save/load plumbing, only a second way to reach the existing menu. Confirmed by
`cmd/townsquarecheck`'s production hit test (`drive statue ... -> kind 2, want menu ok`) and by
`pkg/ui/townsquare_test.go`'s `TestTownSquareStatueClickOpensTheMiniMenu`, which asserts the
resulting screen is `ScreenGameMenu`.

## Cut list

`squareRows()` (`pkg/game/townscreen.go`) builds one hint per door, all four cut identically from
the composed picture. `Rows()` still returns every one of them for any caller not drawing this
art (the row-button fallback, and every headless scenario), so no behaviour driven by `Choose(i)`
changes. Each is reachable elsewhere at the cost of entering that room or the gate, which lists the
same count in full:

- **SC-1a.** The gate's `"%d mission(s) available"`, counting `t.f.Town.Available()`. Entering the
  world map builds its own mission cards from that identical call
  (`(*townScreen).worldMapView`, `pkg/game/worldmap.go:390`), so the count is not only reachable —
  it is read from the same call this hint counted.
- **SC-1b.** The tavern's `"%d with something to say"`, counting `t.f.Town.Offers(TownTavern)`
  entries whose `Mission > 0` (`countOffering`). Entering the tavern lists every NPC via
  `tavernRows()`, each row stating "wants to talk" or "nothing to say" for the same `Offers`
  result the hint counted.
- **SC-1c.** The shop's `"%d on the shelf"`, counting `len(t.f.Town.Offers(TownShop))`. Entering
  the shop lists every offer via `shelfRows(TownShop)`, one row per mission on offer, from the same
  `Offers` call.
- **SC-1d.** The school's `"%d on the board"`, counting `len(t.f.Town.Offers(TownSchool))`.
  Entering the school lists every offer via `shelfRows(TownSchool)`, the same shape as SC-1c over
  `TownSchool`.
- **SC-2.** The room header (`Header()`'s "the town square - chapter N [Esc: back]") is not painted
  over the composed picture. `Header()` is unchanged and still answers it.
- **SC-3.** The gold/party footer line (`Footer()`) is not painted over the composed picture, for
  the same reason as SC-2.

None of the six was ever part of the shipped picture; all were this build's own placeholder text,
added when the square had no art of its own. Cutting them from the drawn surface is not a
divergence from ROM1 — it removes text ROM1 never overlaid there — so it is recorded here as a cut,
not in `docs/DIVERGENCES.md`.

## Readiness and fallback

`townSquareView(t TownScreen) (TownSquareView, bool)` (`pkg/ui/town.go`) is the single readiness
test both the draw path and the click path in `app.go` use: the screen must be at the square,
implement `TownSquareArtScreen`, and its `TownSquareView().Art.Background` must be non-nil.
`LoadTownSquareArt` (`pkg/game/townsquareart.go`) fails atomically — any one of the six shipped
nodes (background, overlay, mask, three labels) missing or mis-sized is a returned error, and every
field of `TownSquareArt` stays unset — so a non-nil `Background` is sufficient evidence the whole
set loaded. An install that fails to ship the picture keeps the pre-existing row-button layout,
unmodified.
