# 1021 — column panes, compact tips, character panel

Owner request, 2026-08-21, with ten screenshots comparing the original against `builds/current/` on
the town square, the shop, the school, the tavern and the character generator. Five items: find out
why these screens still do not look like the original; introduce `LeftPane` and `RightPane` and use
them everywhere, as one common interface; complete the left and right panes, which are cut off or
incomplete today; make tips more compact; make the character panel look like the original, by any
means, with truncation acceptable where the information does not fit.

## Result

Every screen of the town family composes its 160-wide columns through one pane type. No column slot
is filled with an authored colour where a shipped body exists for it, and no slot is missing its
seam. Tips draw at a size that wraps the shipped text into the original's line count. The character
panel draws on a shipped pane body.

Concrete result someone can point at: `builds/current/` on both roots, on the five screens above,
with no near-black rectangle in either column.

## The composition is already researched

The rectangles are decoded, at grade High, and this build's own constants already match them. The
gap is not geometry.

| region | rect | claim |
|---|---|---|
| tavern left panel, id `0x44d` | `(0,0)-(160,480)` | `TOWN-282` |
| centre / roster panel, id `0x450` | `(160,0)-(480,480)` | `TOWN-282` |
| button panel, id `0x44e` | `(480,0)-(640,238)` | `TOWN-282`, `TOWN-214` |
| character panel, id `7` | `(480,238)-(640,480)` | `SHOP-FIGURE-041`, `TOWN-087`, `TOWN-282` |

`pkg/ui/townshell.go` already carries `TownUpperRegion = (480,0)-(640,238)` and
`TownCharacterRegion = (480,238)-(640,480)`, and `ComposeTownSurface` already draws the tavern's
centre at `(160,0)-(480,480)`. Those three are right.

**The original reuses one panel object across screens, which is the owner's proposal.**
`SHOP-FIGURE-041`: the shop's sixth region is not built by the shop. `campaign+0xe0` is constructed
once, at `L09790`, as `ctor(id 7, 0, 238, 160, 480)` — natively the mission screen's own left
column — and each visiting screen borrows it by `RemoveChild` / `OffsetRect(±(640-width))` /
`SetRect` / `AddChild`, offsetting `(0,238,160,480)` by 480 to `(480,238,640,480)`. `TOWN-087` reads
the identical sequence in the tavern at instruction level and discriminates it against the
alternative that the tavern owns a separate panel. So one panel, transferred between parents, is
how the original does it.

**The mission screen's own column is four slots**, ids 5, 6, 7, 8 at `(0,0,160,158)`,
`(0,158,160,238)`, `(0,238,160,480)`, `(0,480,160,screenH)` (`SHOP-FIGURE-041`). The 238 boundary is
where id 6 ends and id 7 begins, which is where `TownUpperRegion` and `TownCharacterRegion` already
meet.

**The 16-column overlap is researched.** `SHOP-FIGURE-041`: the shop's six regions tile 640x480
**with one overlap, the button panel over the merchant panel's right 16 columns**. That is this
build's `TownWideUpperRegion = (464,0)-(640,238)`. Note that `TOWN-282` gives the *tavern's* button
panel as `(480,0)-(640,238)`, 160 wide with no overlap stated. The two rows describe different
screens and do not contradict each other, but the lane must not carry the shop's overlap onto the
tavern without checking.

## The shipped pane art

Measured from `graphics.res` on the EN root by byte size rather than by name, because a name-based
selector over this set returns a subset that looks complete. Every entry size is exactly
`3*w*h + 56`, so each size names one `w x h`.

| bytes | dimensions | entries |
|---|---|---|
| 114296 | 160 x 238 | `chrgen/buttonsarea`, `chrgen/rollstatsl`, `chrgen/loader/leftup`, `inn/buttonsarea`, `inn/leftstats`, `training/buttonsarea` |
| 116216 | 160 x 242 | `humanbackr`, `textbackr`, `chrgen/fullstatsl`, `inn/leftpicture` |
| 11480 | 16 x 238 | `chrgen/rollstatsr`, `inn/luover`, `inn/ruover` |
| 11672 | 16 x 242 | `humanbackl`, `textbackl`, `chrgen/fullstatsr`, `inn/ldover`, `inn/tav_09` |
| 115256 | 160 x 240 | `interface/t_back` and 84 `infowindow/*.bmp` |

Reproduce with:

```sh
go run ./cmd/restool list "$AGAINROM_ASSETS/graphics.res" \
  | awk '$1==114296||$1==116216||$1==11480||$1==11672||$1==115256 {print $1, $2}' | sort
```

A pane is a 160-wide **body** plus a 16-wide **seam**, 238 rows in the upper slot and 242 in the
lower, which is the same 238/242 split the decoded rects give. The `inn/` seams name their position:
`luover` (16 x 238), `ruover` (16 x 238), `ldover` (16 x 242). Read as left/right, upper/down, this
is consistent with the slot heights. **That reading is an inference from the file names and the
dimensions, not a decoded fact.**

**Two entries are ruled out as pane art.** `TOWN-265` places `rollstatsl.bmp` and `tav_09.bmp` in the
77 unreferenced `interface/` entries, and its unfiltered raw-ASCII scan over the whole executable
finds **zero** occurrences of `rollstatsl` or `tav_0` at any string length, which rules out a
two-fragment concatenation. Neither is drawn by the original. Do not use either as a body or a seam.

`humanbackr`, `humanbackl`, `textbackr` and `textbackl` are **not** in that unreferenced list, so
each is referenced somewhere in the executable. `TOWN-284` states the 67 entries directly under
`interface/` were not traced in `EXP-0205` beyond four already-published destinations, so their
reference sites are untraced rather than absent. At 160 x 242 they are the size of the id-7 rect.

## Why the screens do not match today

Measured at master `744b2fc`.

1. **The character panel is an authored fill on every screen.** `DrawTownCharacterRegion`
   (`pkg/ui/townshell.go:487`) paints `townShellPanel`, `RGBA{0x18,0x16,0x15}`, over
   `TownCharacterRegion`, draws a one-pixel outline, and then adds only a figure, two chevrons, a
   mode box and a name. No shipped body is drawn there on any screen. **That rectangle is the
   near-black box the owner reports on the shop, the school, the tavern and the generator.**
2. **Four bodies are loaded by nothing.** `humanbackr.bmp`, `textbackr.bmp`, `chrgen/fullstatsl.bmp`
   and `chrgen/rollstatsl.bmp` have no reference in any non-test Go file. `humanbackl`, `textbackl`
   and `fullstatsr` appear only in `pkg/game/chargenassets.go`, two of the three in a comment. The
   seams are loaded and their bodies are not.
3. **The tavern's left column may be inverted, and has no seam.** `ComposeTownSurface` draws
   `LeftPicture` (160 x 242) at y in [0,242) and `LeftStats` (160 x 238) at y in [242,480). Both fit,
   so nothing fails. Against it: the right column's slots are 238 upper and 242 lower; the `inn/`
   seam names put stats upper and picture lower; and the owner's screenshot of the original tavern
   shows stats at the top and the portrait at the bottom. `TOWN-282` reads the paint routine
   `R0892` and states both bitmaps are blitted unconditionally through `vt+0x18` (opaque) **at
   coordinates it did not resolve to a pixel destination**, so research does not settle the order.
   `luover.bmp` and `ldover.bmp` are not loaded, so x in [160,176) is covered by `Center`.
4. **The seam is a per-screen repair, not a property of the pane.** `ruover.bmp` is referenced from
   six files and is read out of `inn/` by the school and the generator, neither of which has one.
   `DIV-166`, `DIV-168` and hotfix `de72458` each closed one seam at one call site. There is no pane
   type, so each screen composes its own column and each is incomplete in its own way. **This is the
   answer to item 1 of the request:** the project has been repairing instances of a missing
   abstraction one at a time.
5. **The seams are blitted opaque.** Every seam draw is `draw.Src` (`townshell.go:587`, `:595`).
   `TOWN-282` distinguishes the original's two blit calls by vtable slot: `vt+0x18` opaque and
   `vt+0x38` keyed. An overlay named `over` blitted opaque paints its own key colour. This is the
   candidate cause of the black at the column edge that the owner reports on the school and the
   tavern, and it is cheap to test.
6. **Tips draw in the largest shipped font.** `pkg/game/tips.go` passes `f.Font`, which is
   `DefaultFont = "font1"`, a 16 x 15 cell. `font2` is 224 records at 8 x 10 and `font3` is 64 at
   8 x 6; both load through the same `LoadFont` call. At `TownTipRect`'s researched width of 312,
   font1 wraps the shipped town text into enough lines that `cmd/tippanelcheck` measures a minimum
   fit of 295 rows on EN and 363 on RU, and the shipped rect is 373 rows of a 480-row screen.
7. **One tip height serves both roots.** The five rects in `pkg/ui/tippanel.go` are single constants
   sized to the worse root. `TownTipRect` is 373 because RU needs 363; EN needs 295. The EN player is
   shown a panel sized for RU text.

## Behaviours

Three behaviours, one contract.

**B1 — one pane type, used everywhere.** A pane value carries a body, an optional seam, a side and a
slot, and draws itself into the rectangle the side and slot name. `LeftPane` and `RightPane` are the
two column positions. Every screen of the town family composes its columns from panes: the tavern's
two left slots, the tavern's, school's and generator's upper right slot, and the lower right slot on
all four. No screen computes a seam rectangle of its own, and no column slot is drawn with an
authored colour where a shipped body exists for it. Seams blit keyed unless drawing them keyed is
shown to be wrong.

**B2 — compact tips.** The tip body draws at a size whose wrap of the shipped text reaches the
original's line count on both roots, and each rect's height is the minimum that draws that root's own
text whole plus a margin, measured **per root** rather than shared across both. The researched widths
and top-left corners of all five rects are unchanged, and the uniform swallow semantics
`pkg/ui/tippanel.go` documents at length are unchanged. `cmd/tippanelcheck` prints the per-screen,
per-root minimum and is the instrument for the new heights.

**B3 — the character panel on a shipped pane.** `TownCharacterRegion` draws a shipped 160 x 242 body
with its 16 x 242 seam, and the character information composes on top of it. Where the information
does not fit, it is truncated rather than allowed to overflow or to force an authored enlargement of
the pane. **The arrangement of the information stays authored and is not re-derived:**
`UNIT-PANEL-011` establishes positively that the original's own character-sheet layout cannot be
recovered from the executable, and the order and pairing are the owner's ruling of 2026-08-10
recorded in `AuthoredPanelLayout`. What changes is the surface it sits on, not the sheet.
`SHOP-FIGURE-041` records that the id-7 constructor `R1763` allocates **two 160x240 surfaces**
into `+0x74` and `+0x78`; `t_back.bmp` and the 84 `infowindow/*.bmp` entries are the archive's only
160 x 240 bitmaps. That is a lead for what the panel draws, not a decoded destination.

## Domains

**Client (8)** owns all three behaviours: `pkg/ui`, and the view parts of `pkg/game`.
**Assets (1)** is read through the existing `readChargenBMP` and `chargenSize` calls; no format,
decoder or archive path changes. No other domain is touched.

## Ceiling

**Three adversarial passes.** The contract names three behaviours, under the five-behaviour limit. It
reaches no hashed simulation state and touches one domain, so the four- and five-pass tiers do not
apply. It is not split: B1 is one mechanism applied to a population of screens rather than one
behaviour per screen, and B3 is an instance of B1 that cannot be built before it. B2 is independent
of B1 and shares the contract because both are the same owner request against the same screens;
splitting it buys a second doc stack, lane and landing for one result.

Reaching the ceiling is a scoping diagnosis: land what works, open the remainder as its own story,
and record the cut in `closure.md`.

## What is authored and what is not

Researched, cite the row: every rectangle in the table above; that one panel object is transferred
between screens; that `rollstatsl` and `tav_09` are not drawn; that the left panel's two bitmaps are
blitted unconditionally and opaquely.

Not researched, and therefore authored under the owner's 2026-08-15 ruling that an owner-directed
feature proceeds on his intent where research is missing: which body fills the id-7 rect; the
destination of the tavern left panel's two bitmaps, and therefore their order; whether each seam is
keyed; the tip font and the tip heights; the character sheet's arrangement (already owner-ruled).

Every authored placement owes a typed row in `docs/DIVERGENCES.md`. **`againrom` code is never
evidence of ROM1 behaviour, and neither is a screenshot of the original**: what the screenshots
establish is the owner's intent for what these screens must look like. Where a placement is decided
by correlating a shipped bitmap against a shipped background, `cmd/buttonframecheck` is the existing
instrument and the offset it prints is the evidence.

**Divergence ids DIV-175..DIV-182 are reserved for this story** (`pipeline/next-div-id.sh`,
2026-08-21: next free was DIV-175). Ids not spent are returned at the landing and recorded as
returned, because a returned range has no row and is still spoken for.

## Out of scope

The shop's inventory art (`backinv*.bmp`, `shopinv.bmp`) is not a column pane. The school's
`trnhall.bmp` is a single 480 x 480 background that already carries its own left column and is not
decomposed. `chrgen/loader/leftup.bmp` is the loader screen's. The mission HUD's own 300-wide
`AuthoredPanelLayout` sidebar is not resized. No simulation, save format or town rule changes.

## Gates

The repository chain on both repos, `implementation/scripts/check-*.sh` by glob, and — because this
story changes what the screen draws — the two install-gated seat gates by name, on both roots:
`pipeline/check-release-tests.sh` and `pipeline/check-scenarios.sh`. Both exit 2 with no asset root,
so a bare run is never a pass, and in the repository chain a skip and a pass both print `ok`. Compare
against the population count each script prints, not against a number written here.
