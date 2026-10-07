# Spec — 1021-panes

Canonical at landing: what this build actually does, not the contract's own plan for it.

## Result

Three behaviours under one contract (`docs/1021-panes/contract.md`), all against the town family
of screens (town square, tavern, school, shop, character generator):

- **B1.** One shared column-pane type, `ui.TownPane` (`pkg/ui/townpane.go`), composes every 160-wide
  column slot of the tavern, school, shop and generator through `drawTownPane`: a shipped body drawn
  opaque, and, where the neighbouring content does not already cover it, a shipped 16-wide seam
  strip drawn opaque beside it. No caller of `drawTownPane` computes a seam rectangle of its own;
  each hands the shared function two already-named rectangles. Round-1 adversarial review P-1 found
  two shipped body/seam families among the candidate bitmaps, discriminated by a direct join-score
  render rather than by file name, and corrected which family each of the tavern's left column, the
  shared character panel and the generator's own two seams use (see B3 below); the same review's D-2
  found and fixed a second, undocumented `drawTownPane` exemption in the shop's own upper-slot draw.
- **B2.** `TownTipRect`, `TavernTipRect` and `SchoolTipRect` (`pkg/ui/tippanel.go`) keep their
  worst-of-both-roots value as package constants, but the room a player actually sees now draws its
  OWN root's minimum height that fits its own shipped text, plus a fixed 10-row margin
  (`ui.TipPanelShrinkRect`), resolved once per room construction from that room's own loaded tip
  text and font. Round-1 adversarial review P-3: the search itself shipped correctly but searched
  against the wrong font (`font1`, 16x15) for every tip panel; every town/school/tavern/shop tip now
  draws with `font2` (8x10), the same atlas the generator's own tip panel already used.
- **B3.** `TownCharacterRegion` (`pkg/ui/townshell.go`, `(480,238)-(640,480)`) draws one of two
  mode-switched shipped 160x242 bodies and their own 16x242 seams through the same `ui.TownPane`,
  replacing the authored near-black fill `DrawTownCharacterRegion` drew unconditionally on every
  screen before this story — the box the owner reported on the shop, the school, the tavern and the
  generator. Round-1 adversarial review P-1 found the story's original single-presentation choice
  (`chrgen/fullstatsl.bmp`/`fullstatsr.bmp`) belonged to the WRONG bitmap family for this rect (join
  score 42.49, a visibly broken seam) and replaced it with two mode-switched presentations from the
  correct family: `humanbackr.bmp`/`humanbackl.bmp` (Figure mode, join score 9.52) and
  `textbackr.bmp`/`textbackl.bmp` (Stats mode, join score 6.30), selected by
  `TownCharacterView.Statistics`. The same review's P-2 found the generator's own doll box never
  reached this composition at all — it composited the doll bitmap with `draw.Src`, which erased
  whatever pane body a fix drew underneath it wherever the doll sprite itself has alpha under 255
  (measured 77.2% of its own pixels) — and fixed it to `draw.Over`, so the generator is now the
  fourth screen this behaviour reaches, not the third.

This story touches only town/tavern/school/shop/chargen presentation, no simulation or script node,
so neither of the two milestone instruments moves. Round-1 adversarial review D-1: this section
previously called a two-mission smoke check "the milestone census"; it is not — `pipeline/
check-milestone.sh` is the census, and it walks all 28 shipped campaign maps per root, not two.
The two are reported separately. The SMOKE CHECK (`cmd/missionrun -trace`, missions 10 and 20 only,
run directly against this worktree's own build, not `check-milestone.sh`): both missions report 0
`UNSUPPORTED` lines on `en` and `ru`. THE CENSUS (`pipeline/check-milestone.sh`, `pipeline/
milestone-baseline.txt`): 59 `UNSUPPORTED` nodes on each of `en` and `ru`, summed across all 28
maps' own `cannot run` lines — unchanged from before this story, since `check-milestone.sh` reads a
prebuilt `implementation/builds/current/missionrun.exe` this story's own landing did not rebuild.

## Behaviour B1 — the shared column pane

`ui.TownPane` (`pkg/ui/townpane.go`) is a two-field struct, `Body` and `Seam`, both `image.Image`.
`drawTownPane(dst, p, bodyRect, seamRect)` blits `Body` opaque (`draw.Src`) at `bodyRect` if it is
non-nil, and `Seam` opaque at `seamRect` if it is non-nil. A nil `Body` leaves `bodyRect` for the
caller's own fallback; a nil `Seam` leaves `seamRect` untouched, which is correct where a
neighbouring bitmap already covers those columns.

The owner's own proposal (contract.md, quoting him: "нужно ввести LeftPane, RightPane и везде их
использовать") is met by one shared type composing every column slot, not by two named side
constants: every call site's own body/seam draw is identical regardless of which of a room's two
160-wide columns it fills, so a side discriminator would carry no branch and no reader.

Call sites, all through `drawTownPane`:

- `TownUpperRegion` (`(480,0)-(640,238)`), the school's and tavern's button-area picture plus the
  `inn/ruover.bmp` seam at `townUpperSeamRegion` (`(464,0)-(480,238)`) — pre-existing before this
  story (`DIV-166`, `DIV-168`), now composed through the shared type instead of each room's own
  inline pair of draws.
- The tavern's two left-column slots, `(0,0)-(160,238)` (`LeftStats`) and `(0,238)-(160,480)`
  (`LeftPicture`). **Amended round-1 adversarial review P-1:** each now draws its own seam too,
  `inn/luover.bmp` and `inn/ldover.bmp` respectively, at `x:[160,176)` — the story's original
  landing left that column's own `Seam` nil on the reading that `Center` already covered it, but
  `Center` is drawn AFTER the left column (`ComposeTownSurface`'s own draw order), so the column was
  in fact bare until `Center` painted over it, leaving a visible gap for one frame's worth of no
  meaning and, more importantly, no seam art at all once the draw order is read correctly (`DIV-177`).
- `TownCharacterRegion` (`(480,238)-(640,480)`), one of two mode-switched bodies plus its own seam
  at `townCharacterSeamRegion` (`(464,238)-(480,480)`) — see B3.
- The shop's own upper-slot menu plaque (`shopButtonRegion`). **Amended round-1 adversarial review
  D-2:** this call site drew through `blit(dst, art.Menu, ...)` directly, a second undocumented
  exemption from the shared primitive beside `composeChargenDetailedPage`'s own documented one. It
  now composes through `drawTownPane(dst, TownPane{Body: art.Menu}, shopButtonRegion,
  image.Rectangle{})` at both of its two call sites (`pkg/ui/shopscreen.go`), guarded by
  `art.Menu != nil` — boxing a nil `*image.RGBA` into `TownPane.Body`'s `image.Image` field produces
  a non-nil interface with a nil underlying pointer, and `drawTownPane` calling `.Bounds()` on it
  panics; `blit`'s own `*image.RGBA`-typed parameter had made this check automatic, and the new call
  site restores it explicitly.

`chargen_page.go`'s own `composeChargenDetailedPage` composes its NAV plaque and its stat PLATE
through separate `copyNative` calls, not `drawTownPane`, by a real ordering constraint rather than
by oversight: `NavArt` (the same button-area picture the school and tavern share) is drawn before
the class column art, and `NavSeam` must be drawn AFTER it, because the column's own opaque copy
reaches `x=480` and would erase a seam drawn before it (`DIV-168`'s own amendment, and the comment at
the call site). `drawTownPane` draws body then seam as one call; splitting that call would gain
nothing over the existing `copyNative` calls this page already makes for `NavArt`/`NavSeam` and
`Plate`/`PlateSeam`, so those four stay as separate draws. **`PlateSeam` is new this round** (round-1
adversarial review P-1, production slot row 11): the stat plate's own left-hand column,
`main/graphics/chrgen/leftup.bmp` at `(0,0)-(160,240)`, shipped with no strip at its own
`x:[160,176)` before this fix; `chrgen/rollstatsr.bmp` (16x238) now draws there
(`chargenPlateSeamRegion`), chosen on the same left-column-family join-score reading `DIV-176`
already uses for the tavern (`leftup.bmp`/`rollstatsr.bmp` scores 11.22, matching
`chrgen/rollstatsl.bmp`/`rollstatsr.bmp`'s own 11.22 and `inn/leftstats.bmp`/`inn/luover.bmp`'s
10.31 — all three Family L). The DOLL box, `chargenDollBox` (aliased to `TownCharacterRegion`), is
DIFFERENT: it now composes through `drawTownPane(dst, p.DollPane, chargenDollBox,
chargenLowerSeamRegion)`, the same shared primitive and the same mode-switched `TownCharacterPaneArt`
B3 wires everywhere else (round-1 adversarial review P-1/P-2) — this page is no longer the one
`drawTownPane` exception in the town family, only its NAV/PLATE seams remain separate draws for the
ordering reason above. `chargenLowerSeamRegion` is an alias of `townshell.go`'s
`townCharacterSeamRegion` (`var chargenLowerSeamRegion = townCharacterSeamRegion`) rather than a
second declaration of the same rectangle.

The doll itself (`preview.Doll`) draws over `DollPane` with `copyNativeOver` (`draw.Over`), not
`copyNative` (`draw.Src`, round-1 adversarial review P-2): the story's original landing kept
`copyNative`, which copies the source's own alpha byte for byte onto the destination and, since
77.2% of the doll sprite's own pixels carry alpha under 255, erased most of the newly-drawn
`DollPane` body underneath it wherever the doll sprite itself was not fully opaque — the fix this
same round made to the doll box was invisible in the composed page for exactly this reason until
`copyNativeOver` replaced it.

Seam blit mode is opaque (`draw.Src`), not colour-keyed. `pkg/ui/townpane.go`'s own doc comment
records the test this story ran before keeping it: `fullstatsl.bmp`, the body the story's original
landing chose (since replaced, `DIV-175`), carries pure black on 0.2% of its pixels, so a keyed draw
would be visually indistinguishable from an opaque one; its seam `fullstatsr.bmp` carries 13.4%
black, consistent with the ornamental-divider-on-black shape `DIV-166` already measured for the
upper strips (12.6-14.8%) and not with a mistakenly-opaque key colour. **Amended round-1 adversarial
review (P-1):** the B3 body is now `humanbackr.bmp`/`textbackr.bmp`, measured directly from a
decoded install this round (`en`, not committed): `humanbackr.bmp` carries 8.9% pure black
(3428/38720), `textbackr.bmp` 0.2% (75/38720); their own seams `humanbackl.bmp`/`textbackl.bmp`
both carry 12.6% (488/3872), the same ornamental-divider-on-black shape the retired
`fullstatsl`/`fullstatsr` pair and the upper strips both already showed. No rendering performed this
story shows a keyed draw fixes anything opaque does not, for either family.

## Behaviour B2 — per-root tip height

`ui.TipPanelFitHeight(font, text, width)` (`pkg/ui/tippanel.go`) is a linear search, 20 to 480 rows,
for the smallest height at which `tipPanelTextFits` (the search `TipPanelFits` already performed,
extracted so it can be called with no `TipPanelView` yet) answers true. `ui.TipPanelShrinkRect(max,
font, text)` reduces `max`'s own height to that minimum plus a fixed 10-row margin
(`tipPanelHeightMargin`), never below `max`'s own chrome footprint plus one row of text, and never
larger than `max` itself; a nil font or empty text returns `max` unchanged.

`pkg/game/tips.go`'s `tipView` still takes a `rect image.Rectangle` parameter, unchanged. **Round-2
adversarial review (owner item, "shop tip too tall")** found that only three of the town family's
five tip panels were wired to the shrink at the story's own first landing; the shop and the
character generator both still passed their raw package constant straight through. All five callers
now pass a shrunk rect:

- `pkg/game/townscreen.go`: `t.tipView(roomSquare, t.townTip, ui.TipPanelShrinkRect(ui.TownTipRect,
  t.f.tipFont(), t.townTip))`
- `pkg/game/townshell.go` (school): `ui.TipPanelShrinkRect(ui.SchoolTipRect, t.f.tipFont(), t.schoolTip)`
- `pkg/game/townshell.go` (tavern): `ui.TipPanelShrinkRect(ui.TavernTipRect, t.f.tipFont(), t.tavernTip)`
- `pkg/game/shopview.go:73` (round-2): `t.tipView(roomShop, t.shopTip,
  ui.TipPanelShrinkRect(ui.ShopTipRect(), t.f.tipFont(), t.shopTip))`, replacing a bare
  `ui.ShopTipRect()` argument
- `pkg/ui/chargen.go`'s `Chargen.TipPanel` (round-2): `Rect: TipPanelShrinkRect(ChargenTipRect, font,
  tipText)`, replacing a bare `ChargenTipRect` field value

**Amended round-1 adversarial review (P-3).** All three lines above read `t.f.Font` at this
story's original landing — `font1`/`DefaultFont`, 16x15 — both for the search above and for the
`ui.TipPanelView.Font` `tipView` itself sets (`pkg/game/tips.go`). Neither `TipPanelFitHeight` nor
`TipPanelShrinkRect` hardcodes a cell size; both take `font` as a parameter and read
`font.Height()`, so the per-root search already adapted to whichever font was passed — the font
passed was simply the wrong one. `pkg/game/tips.go` gained `(f *FrontEnd) tipFont() *text.Font`,
which returns `f.ChargenAssets.Presentation.Font` (`font2`, 8x10, 224 records) when set, falling
back to `f.Font` on a nil `ChargenAssets`/`Presentation`/`Font` (a carried `LoadChargenAssets`
error). This reuses the SAME `*text.Font` instance `pkg/game/chargenassets.go` already loads via
`LoadFont(src, "font2")` for the generator's own detailed-page labels, rather than opening the
archive a second time — the same object `pkg/ui/chargen.go`'s own `Chargen.TipPanel` already draws
its own tip text with (`c.setup.PreCreate.Art.Font`), which is this build's only other tip panel and
was never affected by the FONT defect (P-3): the generator's own tip text always drew in font2. Its
own rect was still unshrunk until round-2 (see below), which is a separate defect from the font.
All five `TipPanelView{Font: ...}` construction sites in `pkg/game`/`pkg/ui` now agree on font2 for
tip text.

`ui.TownTipRect`, `ui.TavernTipRect` and `ui.SchoolTipRect` themselves are unchanged package
constants (`pkg/ui/tippanel.go`), still the worst-of-both-roots value: `pkg/ui`'s own synthetic
crossing/overlap tests (golden rule 2, no install) exercise them as a reachable state, whichever
root needs the full extent, and changing their own value would change what those tests measure.
The shrink happens where the root's own shipped text is actually known — `pkg/game`, not `pkg/ui`.

`ui.ShopTipRect()` and `ui.ChargenTipRect` themselves are unchanged: both are sourced from a decoded
rectangle (`SHOP-TIP-045`, `TOWN-187`) that already fits both roots inside its own size. **This
story's own first landing read that as license to skip the shrink for these two rects entirely,
which round-2 adversarial review found wrong**: a decoded rectangle is a decoded MAXIMUM extent, not
a decoded minimum, and B2's own shrink logic is about the gap between the two — orthogonal to
whether the maximum itself is decoded or authored, which is exactly the same reasoning the other
three rooms' own rects (authored maxima) were already shrunk under. Both are now run through
`TipPanelShrinkRect` the same as the other three; neither package constant's own value changed.

`cmd/tippanelcheck` (the committed measuring instrument) reads each room's own resolved
`v.Tip.Rect` / `sqv.Tip.Rect` rather than the package constant. **Round-2 fix**: it read
`ui.ShopTipRect()` and `ui.ChargenTipRect` directly for those two rooms, which is the same
raw-constant blind spot the production code itself carried; it now reads `shopView.TipPanel.Rect`
and `Chargen.TipPanel().Rect` (once per pre-choice, fighter and mage separately, since
`Chargen.TipPanel` shrinks against whichever text the current pre-choice selects and the two can
differ — observed on `ru`), so its own report reflects what every room actually draws and
hit-tests. Chosen-height / min-height, both roots, MEASURED AFTER THE P-3 FONT FIX, THE ROUND-2
SHOP/CHARGEN SHRINK AND THE INSTRUMENT FIX ABOVE (re-run this round; the story's original landing
reported larger figures against `font1` for the first three rooms, quoted in `DIV-179`, and reported
no shrink at all for shop/chargen, both from `cmd/tippanelcheck`'s own unfixed report at `c5c4f00`):

| room | en | ru |
|---|---|---|
| town square | 182 / 172 | 206 / 196 |
| tavern | 182 / 172 | 194 / 184 |
| school | 170 / 160 | 134 / 124 |
| shop | 122 / 112 | 110 / 100 |
| chargen, fighter tip | 170 / 160 | 158 / 148 |
| chargen, mage tip | 170 / 160 | 170 / 160 |

Shop and chargen's own spare, before round-2's fix (`cmd/tippanelcheck` at `c5c4f00`, unshrunk rect
minus chosen height, identical on `en` and `ru` since neither raw package constant is per-root):
66/78 rows for the shop (`ui.ShopTipRect()` is 178 rows tall; the table's own cell 2 sat under 46.2%
of that spare on a shipped shop tip string, `en`/`ru` respectively), 40 rows for the chargen fighter
tip on `en` (`ChargenTipRect` is 200 rows tall; the shipped fighter tip's own "choice 0" line sat
under 97.9% of that spare) — the more severe of the two by coverage fraction even though its own row
count is smaller.

Each root still carries only its own 10-row margin; the absolute heights above are smaller than the
story's original landing reported for the same rooms (town square 305/373, tavern 305/322, school
271/220, `en`/`ru`) because font2's own 8x10 cell wraps the same shipped text into fewer, shorter
rows than font1's 16x15 cell did. The per-root split itself (each root no longer carrying the other
root's own unused spare) is unchanged by the font fix.

## Behaviour B3 — the character panel body

**Rewritten in full at round-1 adversarial review (P-1, P-2).** The story's original landing loaded
one body/seam pair, `fullstatsl.bmp`/`fullstatsr.bmp`, for every mode and every screen. A direct
join-score render (`cmd/plaqueseams`, extended this round: the mean per-channel pixel difference
across each candidate body's own edge column against each candidate seam's own edge column, both
join directions, identical on `en` and `ru` since both roots ship byte-identical `interface/` art)
found that pairing belongs to the WRONG bitmap family for this rect: `fullstatsr.bmp` against
`fullstatsl.bmp` as a right-column join (strip left of body, the shape `TownCharacterRegion` needs)
scores 42.49, a visibly broken seam; `humanbackl.bmp` against `humanbackr.bmp` scores 9.52 and
`textbackl.bmp` against `textbackr.bmp` scores 6.30, both low, both the correct family.

`game.LoadTownCharacterPaneArt(src)` (`pkg/game/characterpane.go`) now loads TWO mode-switched
160x242 body / 16x242 seam pairs: `humanbackr.bmp`/`humanbackl.bmp` (Figure mode) and
`textbackr.bmp`/`textbackl.bmp` (Stats mode), through `loadCharacterPanePair`, returning
`TownCharacterPaneArt{Figure, Stats ui.TownPane}`. `(f *FrontEnd) characterPanes()` (renamed from
the original singular `characterPane`) is the cached accessor, on `tipArt`'s own precedent: loaded
once, not once a frame, cached behind a `characterPaneLoaded bool` flag beside
`characterPaneCache TownCharacterPaneArt` (`pkg/game/frontend.go`). Either half failing fails the
whole call, so a caller never receives one mode wired and the other not.

`ui.TownCharacterView`'s original single `Pane TownPane` field is replaced with `FigurePane` and
`StatsPane TownPane`. `DrawTownCharacterRegion` (`pkg/ui/townshell.go`) selects `StatsPane` when
`v.Statistics` is true and `FigurePane` otherwise, and draws the chosen pane first, through
`drawTownPane`, when its `Body != nil`; a nil `Body` on the selected pane (a caller predating this
story, or a load failure) falls back to the same authored fill and outline as before, so the region
is never left blank.

One production call site sets both fields for every consumer: `townScreen.townCharacterView()`
(`pkg/game/townshell.go`) now returns `ui.TownCharacterView{..., FigurePane: panes.Figure, StatsPane:
panes.Stats}` from one `panes := t.f.characterPanes()` call. `TownSurface()` (tavern, school) and
`shopview.go`'s shop path call `townCharacterView()`, so both reach the fix from this one change.
`cmd/plaquescreens` builds its school/tavern views by hand (outside `townScreen`) and gained its own
`characterPanes(f)` helper (renamed the same way) to populate both fields identically.

MODE ASSIGNMENT is a second, separate inference (`DIV-175`): `humanbackr.bmp` renders as an oval
portrait frame and `textbackr.bmp` a rectangular text frame — the same shape distinction as
`inn/leftpicture.bmp` (oval) against `inn/leftstats.bmp` (rectangular), the tavern's own
figure/stats pair. This build pairs the oval frame to Figure mode (the doll) and the rectangular
frame to Stats mode, on that shape reading; `SHOP-FIGURE-041` names the id-7 object's own two
allocated 160x240 surfaces (two rows short of `TownCharacterRegion`'s 242, a pre-existing gap this
row does not close) and no archive entry for either. No claim states this pairing.

THE GENERATOR is now the fourth screen this behaviour reaches (round-1 adversarial review P-2): its
own doll box, `chargenDollBox` (aliased to `TownCharacterRegion`), now draws `p.DollPane` (the
Figure-mode pane) through the same `drawTownPane` primitive, and the doll sprite itself
(`preview.Doll`) composites over it with `copyNativeOver` (`draw.Over`) instead of the story's
original `copyNative` (`draw.Src`), which erased most of the newly-drawn body underneath it — see
the B1 chargen paragraph above for the mechanism.

The tavern's own left-column order is reversed from this build's prior order (`DIV-176`): `LeftStats`
(238 rows) now draws upper, `LeftPicture` (242 rows) lower, matching the right column's own
238-upper/242-lower convention, the `inn/` seam file names (`luover`/`ruover` 238-tall, `ldover`
242-tall), and the owner's own screenshot of the original tavern. `TOWN-282` reads both blits as
unconditional and opaque but does not resolve their pixel destination, so the order is authored, not
decoded. **Amended round-1 adversarial review (P-1, `DIV-177`):** each of the two left-column bodies
now also draws its own seam at `x:[160,176)` — `LeftStats` with `inn/luover.bmp`, `LeftPicture` with
`inn/ldover.bmp` — closing a gap the story's original landing left bare at that column (see B1).

## Behaviour P-4 — tavern roster cells and room title clear the left column

Round-1 adversarial review found `townSurfaceCellRect`'s tavern case and the room title draw
(`ComposeTownSurface`, `pkg/ui/townshell.go`) both crossing into `x<160` (the left column,
`LeftPicture`/`LeftStats` furniture) and into the new `x:[160,176)` seam column above, overwriting
shipped art with an authored near-black cell box. The owner's own stated placement for mercenaries
is «внизу экрана между left and right panels» — at the bottom of the screen, between the left and
right panels. Round-1's own fix moved the roster in `x` only, to `image.Rect(176+col*78,
280+row*92, 246+col*78, 364+row*92)` (four columns, width 70, gap 8, stride 78, spanning exactly to
`x=480`), clearing both the left column and its own seam; the title moved to `image.Rect(176, 12,
470, 44)`. `y` was unchanged from `master`'s own `280+row*92`: a fixed top-anchored origin, not "at
the bottom" — round-1's own fix addressed "between the left and right panels" and left "at the
bottom" unaddressed, which round-2 adversarial review (P-5, this round's regression finding below)
is what found the roster's own cells narrowed to 70px wide in the same commit and only then noticed
the `y` half of the owner's own sentence had never been built.

**Round-2 fix, together with P-5's price-truncation regression below (single commit, one root
cause):** `townSurfaceCellRect`'s tavern case is rewritten bottom-anchored and roster-size-aware —
`townSurfaceCellRect(kind ui.TownSurfaceKind, i, total int) image.Rectangle`, `total` being
`len(v.Cells)` at both of its two production call sites (the hit-test loop and the draw loop,
`pkg/ui/townshell.go`). Two columns, cell width 144, stride 152, height 84, row stride 92, grid
bottom `y=472`; row count is `ceil(total/2)`, and a cell's own `y` is computed from the BOTTOM
upward by its own distance-in-rows from the last row, so the last row's own bottom edge sits at
`y=472` regardless of how many rows the current roster needs — one roster member's own cell sits at
the same `y` a hundred members' own LAST row would, matching "at the bottom" for any roster size
rather than only for the two-entry case round-1's own fix happened to leave correct by coincidence
of a fixed-`y` layout. A floor clamp (`gridTopFloor = 56`) stops a pathologically large roster's own
top row from climbing over the room title; this is an accepted display limit for an unbounded
roster, not a new defect (`master`'s own prior layout carried the same unbounded-growth property
with no clamp at all). `176` is still chosen only to clear the left column and its own seam; no
claim gives the roster's own coordinates in the original to check this placement against
(`DIV-178`). The school's own cell rect is unaffected — it does not overlap a left-column
presentation and was not part of either finding.

## Behaviour P-5 — round-1's own roster fix regressed cell text width

Round-1's own P-4 fix (above) narrowed the tavern roster cell from `master`'s own 108px
(`image.Rect(10+col*116, ..., 118+col*116, ...)`) to 70px (`image.Rect(176+col*78, ..., 246+col*78,
...)`), moving it clear of the left column without re-checking whether the shipped label/detail
strings this build's own font still fit the narrower budget. They did not: the owner, playing
`c5c4f00`, found roster entries reading `NPC 1` for the shipped `NPC 12`, `5 for 2` for `5 for 250`,
and equivalent truncations across every production format string
(`pkg/game/townshell.go`'s own `tavernSurfaceCells`: `"NPC %d"`, `"mission %d"`, `"Squad %d"`,
`"%d for %d"`, `"return %d"`) — each is a DIFFERENT roster entry's own identity or a wrong price, not
a mere truncation of secondary flavour text. `townShellTextLayout` (`pkg/ui/townshell.go`) truncates
from a string's own end to fit its box; nothing in the round-1 landing measured the new 70px box
against the format strings it would actually carry, and the story's own release-gated tests never
compose a tavern with more than the two-entry roster fixture both preexisting scenario tests happen
to use, so no install-gated witness caught it either.

**Fixed together with P-4's own `y` regression** (`townSurfaceCellRect`'s rewrite above): the cell
width widens from 70 to 144, comfortably wider than the worst production string measured against
this build's own font2 (the roster cell's own font — `10 for 1200` measures under 100px; the
144px cell leaves a budget of 140px after its own 4px inset). The width was not chosen from that one
measurement alone; it follows from the layout becoming two columns of 144px each (`176 + 2*144 +
gap = 480`, the same right edge the 70px four-column layout already used) rather than from a
width picked to fit strings and left unexplained.

**Witness (`pkg/ui/townshell_test.go`,
`TestTownShellRosterCellTextBudgetFitsEveryProductionString`, synthetic, no install)**: asserts
`townShellTextLayout` returns every one of `tavernSurfaceCells`'s own production format strings,
at plausible shipped digit counts, UNMODIFIED at the shipped cell width. `pkg/ui` may not import
`pkg/game` (the tier direction, `internal/archtest`), so the eleven format strings are reproduced as
literal fixtures rather than called through the production formatter; this comment and
`pkg/game/townshell.go`'s own `tavernSurfaceCells` are the one place the two must be kept in step if
a format string ever changes. The cell width itself is read from `townSurfaceCellRect(TownSurfaceTavern,
0, 1).Dx()` rather than duplicated as a literal, so a future change to the production constant moves
what this test asserts against without the test file being touched — the same shape of duplication
that let round-1's own 70px shrink land with no test failing anywhere. A synthetic uniform-advance
font (9px/glyph, calibrated to the worst observed per-character ratio of this build's own real font,
rounded up) stands in for font2, since golden rule 2 keeps a real font out of an install-free test.

MUTATION TESTING: reverting the cell width constant from 144 to 70 (round-1's own shipped value)
fails 11 of the 12 test cases, truncating `"NPC 999"` to `"NPC 99"`, `"10 for 1200"` to `"10 for"`,
and so on, at the same 60px budget the owner's own screenshots showed truncated in the shipped
build — confirming the witness catches the regression it exists to catch, not only a hypothetical
one.

## Round-2 owner items — chargen page

Four further owner-reported defects, all in `pkg/ui/chargen_page.go`/`pkg/ui/townshell.go`, none
reaching hashed simulation state:

- **Brown rectangle around the doll**: `composeChargenDetailedPage` drew `drawBorder(dst,
  chargenDollBox, color.RGBA{138,116,70,255})` unconditionally after every other draw, and a dead
  `drawChargenFrame(dst, chargenDollBox)` call earlier in the same function whose own result was
  always immediately overpainted. Both removed; `TestDetailedControlBoundsAndDollReplacement` and
  `TestChargenSourcePlacementsMasksAndFrames` (`pkg/ui/chargen_page_test.go`) updated from the border
  colour to the page background colour at `chargenDollBox`'s own corner.
- **Fire icon overlapping the right panel on the mage page**: the skill-icon draw loop
  (`composeChargenDetailedPage`) clipped to `chargenColumnDestination`, which reaches `x=480`, four
  pixels past `TownWideUpperRegion`'s own left edge at `x=464` where the seam column (B1 above) now
  draws. A new clip rect, `chargenColumnSkillClip` (`chargenColumnDestination` with its own right
  edge pulled to `x=464`), replaces it at the one call site. Measured (class 1, mage): **349 of
  3808** seam-column pixels differed before the clip, bbox `x:[464,472) y:[150,202)`. The count
  is reproduced by pointing that call's own clip argument back at `chargenColumnDestination` and
  running the `mage/nav_seam` subtest against either install; it reads 349 on both `en` and `ru`,
  and only that subtest reddens. An earlier revision of this line and of the constant's own
  comment said 416, which no run reproduces (pass-3 adversarial review, 2026-08-21). Witnessed by
  `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips`'s new mage-class subtests (see Testing).
- **Near-black control boxes over shipped art**: `shopBookRect`, `townCharacterPrev`,
  `townCharacterMode` and `townCharacterNext` drew through `drawTownShellBox`/`drawShopChevron`'s own
  opaque interior fill, covering shipped ornament under an authored flat colour (6,104 of 38,720
  pane pixels on the shop, 3,584 on the tavern/school). `drawShopChevron` gained a `fill bool`
  parameter (false at both of its call sites in `pkg/ui/shopscreen.go`, skipping only the interior
  `draw.Src` fill and keeping the outline); a new `drawTownShellOutline` (outline only, no interior
  fill) replaces `drawTownShellBox` at `shopBookRect` and `townCharacterMode`'s own draw calls. Every
  box stays hit-testable exactly as before — only the paint changed, not the rectangle. Placement is
  unchanged (out of scope this round).
  `TestShopDollHitAreaMatchesEveryVisibleSyntheticFigurePixel` (`pkg/ui/shopscreen_test.go`) updated:
  `shopBookRect` no longer paints differently from the doll's own area, so the test's own "want"
  ground truth adds an explicit `!p.In(shopBookRect)` exclusion (a documented hit-test precedence
  rule; the former member-name glyph exclusion was later removed with that overlay) rather
  than re-deriving Book's own footprint from paint visibility, which the fill removal broke.
- **`townCharacterMode`'s own label budget**: `font1` measures `DOLL` at 52px and `STATS` at 57px
  against the box's own prior 42px interior width, rendering `DOL`. `townCharacterMode` widens from
  `image.Rect(536, 443, 584, 475)` (48px) to `image.Rect(516, 443, 590, 475)` (74px), clearing both
  labels; the box stays inside `TownCharacterRegion` and does not touch the prev/next chevrons on
  either side.

## Testing

`go build ./... && go vet ./... && go test -trimpath -count=1 ./...` — all packages pass, gofmt
clean. `pkg/game/frontend_session_test.go`'s
`TestResetSessionForNewGameDropsExactlyTheSessionPopulation` gained `characterPaneCache`/
`characterPaneLoaded` in both `nonZeroFrontEnd` and `keepUnchanged`, on `tipArtCache`'s own
precedent (an install-scoped cache, not game progress); the fixture's cache value updated at
round-1 review to `TownCharacterPaneArt{Figure, Stats: ui.TownPane{Body: &image.RGBA{}}}` matching
the renamed field pair. B1's and B3's own package-level rectangles (`TownTipRect` etc.) are
unchanged values.

`pkg/ui/townpane_test.go` is unchanged since the story's original landing: four synthetic tests (no
install, golden rule 2) on `drawTownPane` and on `DrawTownCharacterRegion`'s own nil-pane fallback.

Round-1 adversarial review's required witness added or extended four install-gated instruments
(golden rule 2 keeps all of these out of `go test`; every one was run and read on both `en` and `ru`
this round):

- `pkg/game/chargen_release_test.go`'s `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips` is
  extended from the upper seam alone to all four of the generator's own seam/body slots: `NavSeam`,
  `PlateSeam` (excluding its own overlap with `chargenCardBox`'s bottom 31 rows), `DollPane.Seam`,
  and `DollPane.Body` (excluding the doll's own footprint; the border-ring exclusion this bullet
  named at round 1 no longer applies — see below). **Round-2 rebuild (item 2)**: `NavSeam`,
  `PlateSeam` and `DollPane.Seam` now compare against a suppressed-seam "before" frame at their own
  key pixels, the same shape as `townpanes_release_test.go`'s own `assertSeamRegion`; `DollPane.Body`
  drops its own border-ring exclusion since owner item 4 removed the drawn border entirely. **Round-2
  parameterization (item 5, "if cheap")**: the whole test now runs once per pre-choice class,
  `t.Run("fighter", ...)`/`t.Run("mage", ...)` (`ui.Chargen.SelectPreChoice(0)`/`(1)`), each with its
  own four segment subtests. This is the one instrument that can witness owner item 4's skill-icon
  clip fix at all: the mage page is the only one whose skill icons reach the nav-seam column
  (measured 416/3808 mismatched pixels at `x:[464,472) y:[150,202)` before the clip; the fighter page
  carries no icon there and cannot redden on a regression of it). No test in this repository composed
  the mage detailed page before this round (`cmd/plaquescreens`' own prior comment on the same
  blindness).
- `pkg/game/townpanes_release_test.go` is new: `TestReleaseTownColumnPanesDrawShippedBodyAndSeam`,
  eight subtests covering the tavern's two left-column slots (body and seam each), the tavern's and
  the school's own character panel in both Figure and Stats mode, and the shop's own upper plaque
  and character panel — the same pixel-for-pixel comparison, excluding the three controls
  `DrawTownCharacterRegion` always draws over the pane (the chevrons and mode box) and, for
  the shop only, its own Book toggle. **Round-2 adversarial review, item 6**: this sentence named
  eight subtests at the story's original landing while the committed test carried seven, missing
  "school character panel stats mode" — the school's own Statistics-mode composition had no pixel
  witness at all. Added rather than cut from the sentence: it mirrors the tavern's own stats-mode
  subtest already present, against `SchoolArt` instead of `TavernArt`, and closes a real coverage
  gap rather than only correcting a count. **Rebuilt keyed-aware, all eight** (round-2 adversarial
  review, item 2): the six subtests naming a seam (all but the two shop cases, of which one has no
  separate seam field and one — shop character panel — does) now build a second, independently
  composed frame with that one seam field suppressed and compare a key pixel against IT rather than
  against the raw source's own zero-alpha value, which the keying fix (B1 above) would otherwise
  fail pixel-for-pixel on every one of them; see `assertSeamRegion` in the same file. The three
  hand-copied character-panel control rectangles this file's own `characterPanelControls` slice
  carried (round-1 adversarial review, item 2) are replaced with `ui.TownCharacterPanelControls()`,
  exported from `pkg/ui/townshell.go` for this purpose, so a future change to those four rectangles
  cannot drift silently against a duplicated literal.
- `pkg/game/townbuttons_release_test.go`'s
  `TestReleaseSchoolAndTavernUpperRegionMatchesShippedArtOutsideButtonWells` (**round-2 rebuild, item
  2**): its `x:[464,480)` sub-range comparison, against `UpperSeam`, now builds a second frame with
  `UpperSeam` suppressed (`SchoolArt`/`TavernArt` copy with that one field nil'd) and compares a key
  pixel against it rather than against the raw source's own zero-alpha value; `x:[480,640)` against
  `Upper` (Body, opaque) is unchanged.
- `cmd/plaqueseams` is extended with the 242-row column-pane population (both join directions, every
  candidate body against every candidate strip), the 238-row left-column population
  (`leftstats`/`luover`, `rollstatsl`/`rollstatsr`, `leftup`/`rollstatsr`), and a `shopmenu.bmp`
  self-check (its own left 16 columns against `inn/ruover.bmp`, its own right 160 columns against
  `chrgen/buttonsarea.bmp`, over the whole sub-image rather than one border column) — the reference
  decomposition the family reading is checked against. On `en`: `humanbackl|humanbackr` 9.52,
  `textbackl|textbackr` 6.30, `fullstatsr|fullstatsl` right-column 42.49 / left-column 9.75,
  `ldover|leftpicture` left-column 5.76, `leftstats|luover` 10.31, `rollstatsl|rollstatsr` 11.22.
  `shopmenu.bmp` self-check: left half 0.16, right half 0.27 mean-abs-diff. Byte-identical on `ru`.
- `cmd/tippanelcheck` re-run after the P-3 font fix and, this round, after the round-2 shop/chargen
  shrink; the instrument itself was corrected first (B2 above) since it read the raw
  `ui.ShopTipRect()`/`ui.ChargenTipRect` package constants rather than each room's own resolved rect,
  the identical blind spot the production code carried before this round's fix.

MUTATION TESTING against the new witnesses: mutating `characterPaneFigureBodyPath` to the wrong file
(`fullstatsl.bmp`) did NOT fail `TestReleaseTownColumnPanesDrawShippedBodyAndSeam` — the test reads
its own "expected" pixels from the same `characterPanes()` call the production path uses, so it
proves wiring correctness, not "is this the right shipped file"; that claim is carried by the
join-score measurement above, not by pixel identity. Reverting `ComposeTownSurface`'s tavern draw
order (Center-first back to Center-last) DID fail the test, on both the tavern-left-upper and
tavern-left-lower subtests, confirming the witness catches the class of regression it was built for.

`check-release-tests.sh`'s own selected count moved from 34 (before round-1 review) to 35 (the new
`townpanes_release_test.go` file, one `Test...` function, `chargen_release_test.go`'s own extended
function counting once): 35 of 35 passed on `en`, 35 of 35 on `ru`. **Re-run round-2** (the new
subtests and the class-parameterized chargen loop are all inside these same 35 `Test...` functions,
so the selected count is unchanged by this round; only the subtest count inside three of them grew):
35 of 35 passed on `en`, 35 of 35 on `ru`. `check-scenarios.sh`: 13 of 13 on `en`, 13 of 13 on `ru`,
unchanged both rounds. See `closure.md` for the integration witness.
