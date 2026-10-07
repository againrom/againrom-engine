# spec.md — 1022, the character generator's canonical composition

Canonical at landing. This describes what shipped, not the original intention in `contract.md`.

## Result

The character generator's detailed page composes at the decoded rects `contract.md` names, reusing
`1021`'s pane type (`TownPane`/`drawTownPane`) for every border-column slot. The authored 300-wide
card and the black message rectangle are gone. The tip panel opens on this page. The town's own
DOLL/STATS toggle places the same 160x242 card in the character pane in place of the doll, on every
room that draws it.

## B1 — the centre column at its decoded rect

`chargenColumnDestination = image.Rect(160, 0, 480, 480)` (`pkg/ui/chargen_page.go:153`), whole,
uncropped. `chargenColumnOffset = chargenColumnDestination.Min = (160,0)` (`:189`). The former source
crop (`chargenColumnSourceCrop = (72,0)-(252,480)`) is removed. `detailedSkillOrigin` is unchanged in
source-space and now composes correctly because the offset it is read against changed with it.

## B2 — the four column-adjacent slots draw as shipped panes through `1021`'s pane type

Every border-column slot draws through `drawTownPane`, not a private copy:

- `chargenPlateSeamRegion = (160,0)-(176,238)`, `chrgen/rollstatsr.bmp`, keyed.
- `chargenCardSeamRegion = (160,238)-(176,480)`, `chrgen/fullstatsr.bmp`, keyed.
- `chargenLowerSeamRegion = townCharacterSeamRegion` (shared with `pkg/ui/townshell.go`), the doll
  pane's own seam.
- `townUpperSeamRegion`, the nav pane's own seam (pre-existing, `1021`).

**Draw order, decoded (`TOWN-234`): one background blit, then the four border-column blits on top of
it.** All four bodies (plate, card, both class columns) draw before `chargenColumnDestination`
composes; the plate seam, the card seam and the nav seam all draw AFTER it, matching the decoded
order. This story's own defect and fix are recorded below.

## B3 — the character card at its decoded 160x242 box, on `fullstatsl.bmp`

`chargenCardBox = image.Rect(0, 238, 160, 480)` (`:162`). `CompactPanelLayout` (`pkg/ui/panel.go:850`)
composes at `Size: (160,242)`, `Background` set to `fullstatsl.bmp` when the load succeeds
(`drawPanelBackground`, nil-safe), the owner's field order:

```
                <name>
BODY     nn   HEALTH  nnn/nnn
AGILITY  nn   MANA    nnn/nnn
MIND     nn
SPIRIT   nn
DMG    nn-nn  ABSORB   nn
ATTACK   nn   DEFENSE  nn
SKILLS        RESISTANCE
<five skill rows>   <five element rows>
XP      nnnn
SIGHT   nn
SPEED   nn
```

**WEIGHT is not drawn** (`DIV-209`, opened round 2, OPEN — the owner did not accept dropping this
row; the absence is a capability gap, not a layout choice: no field anywhere in `pkg/sim` or
`pkg/game` carries a party member's carried weight or an encumbrance value). `DIV-191` covers only the
row arrangement and order, not whether WEIGHT is drawn.

The font is the page's own already-loaded font (`p.Font`), the same one drawn elsewhere on the page;
no separate font object is measured or loaded for the card. Round-2 adversarial review (P-1) found the
card's own interior inset used a fixed `Pad: image.Pt(4, 3)` regardless of the shipped background's
own painted border. `compactPanelInset` (`pkg/ui/panel.go`) now measures the actual interior of the
`*image.RGBA` background passed in — the most frequent colour is taken as the fill, the first
column/row where at least a quarter of pixels match it is the settled edge, plus a 2px margin — and
`CompactPanelLayout` reads `Pad` from it instead of a constant.

Measured on both preserved roots (`readChargenBMP`/`LoadTownCharacterPaneArt`, a throwaway dev tool,
byte-identical between EN and RU): `chrgen/fullstatsl.bmp`'s own settled interior starts at column 18,
row 16 (`Pad = (20,18)`), and `interface/textbackr.bmp` (the town statistics card's own background) at
column 7, row 16 (`Pad = (9,18)`) — the fixed `(4,3)` was 16px short of the card's own left margin and
15px short of its top, and 5px/13px short of the statistics card's. The composed frame
(`cmd/plaquescreens`, EN root) reproduces the first pair exactly: cropping the rendered detailed
page's own `chargenCardBox` and running the same settled-edge scan on the composite gives the same
`(18,16)`, confirming `compactPanelInset`'s measurement reaches the actual draw call and is not just a
value computed and discarded. The owner's own canon screenshots
(`pipeline/archive/owner-ruling-2026-08-21-stats-card/canon-card-fergard.png`,
`canon-card-danath-named.png`) settle at column 23 in both; the two screenshots agree with each other
but not pixel-for-pixel with the composed frame's `(18,16)`, because the owner's own crop includes a
few pixels of the card's visible gold border that `chargenCardBox`'s own exact rect does not — the two
readings are structurally consistent, not numerically identical, and no claim gives the screenshot's
own crop origin to close that gap exactly.

The name row is centred (`PanelRow.Center`, read only when the row's own `Size.X` is
fixed) rather than left-aligned like the data rows below it (P-1). Every column's own values now share
one right edge for the whole card (`PanelLayout.AlignValues`, `pkg/ui/panel.go`), matching the canon
screenshot, rather than sitting at each row's own natural width (P-5).

MANA prints `nnn/nnn` unconditionally, including `0/0` for a character with no mana pool
(`PanelFieldManaCardHeading`/`PanelFieldManaCard`, dedicated fields distinct from
`PanelFieldMagicHeading`/`PanelFieldMagic`), matching the owner's 2026-08-21 canonical screenshot.
This does not touch `AuthoredPanelLayout`'s own pre-existing MANA row, which still suppresses on
`MaxMana > 0` (`0109` spec FR-10) and is unaffected because it reads the older, dedicated
`PanelFieldMagicHeading`/`PanelFieldMagic` pair, not the new card-only fields.

SIGHT and SPEED are two rows, not one: `pkg/ui/panel.go`'s field table had them sharing a single
`PanelRow` (Left cell SIGHT, Right cell SPEED), which round-2 adversarial review (D-7) found
disagreed with this document's own diagram and with the owner's canon screenshot, both of which show
separate rows. The table now uses a dedicated `PanelFieldBlank` (an empty, always-resolved Left cell)
paired with SIGHT on one row and SPEED on the next, deliberately forcing the two-cell-row branch of
`panelItems` so the row does not slide SIGHT's value into the Left column.

SIGHT itself prints a whole number (`fmt.Sprintf("%d", s.Char.Sight)`, unchanged by this story). The
owner's canon screenshots show a fractional digit (`6.1`, `6.4`, `6.5`). This story does not implement
that: `Derived.Sight` (`pkg/data`) drops the sub-cell remainder by design, and printing a fraction
means storing that remainder, which changes `Derived.Sight`'s own representation and reaches
`pkg/game`'s `ScanRange` and the actor's own sight — hashed simulation state, a different contract
from this story's card composition. Research bearing on the divisor is in progress at another seat, on
a branch this story's pin does not carry; no claim is cited here for it, and no divergence row is
opened by this story for it.

`CompactPanelLayout` is also the town's own statistics card (B6): one function, two call sites.

## B4 — the black message box is removed, and a message channel is restored

`chargenDetailMessageBox` and the old `drawChargenDetailMessage` are gone, per spec B4's original
instruction to remove "the black rectangle with text" (owner, 2026-08-21). The centre column now
occupies that screen space (B1).

Round-2 adversarial review (P-3) found the replacement this document originally described —
"the copy is dropped, not moved" — left `chargenDetailMessage()` (`pkg/ui/app.go`) with no production
caller at all: a name-empty refusal, a too-long-name refusal and a skill-budget refusal each still set
`a.chargenHoverText`/`a.flow.msg`, but nothing on the detailed page read them, so the message a
refusal is meant to report was invisible. Contract B4 permits removing the old black box but requires
a replacement channel for what it carried; this story now provides one instead of removing the
refusal-setting call sites:

- `Chargen.SetDetailMessage(msg string)` (`pkg/ui/chargen.go`) stores the message on the model.
- `composeChargenScreen` (`pkg/ui/app.go`) calls `c.SetDetailMessage(a.chargenDetailMessage())` before
  composing, carrying the same hover/refusal text the removed box used to show.
- `drawChargenMessage` (`pkg/ui/chargen_page.go`) paints it, centred, at
  `chargenMessageRect = (chargenDollBox.Min.X+4, chargenDollBox.Max.Y-18)-(chargenDollBox.Max.X-4,
  chargenDollBox.Max.Y-4)` — inside the doll box's own lower strip, not a new black rectangle — and
  `composeChargenDetailedPage` calls it once, right before `ComposeTipPanel`.

`DIV-192` (the dropped channel) is closed. The copy still does not move into the tip panel: the
panel's text area shows the shipped `chrgen2.txt` (B5), and swapping shipped copy for transient UI
wording would mean the panel no longer shows what it is named for.

## B5 — the tip panel opens on the detailed page

`ChargenSetup.TipTextDetail` carries `main/text/tips/chrgen2.txt`, loaded in
`pkg/game/chargen.go` and wired at construction. `Chargen.TipPanel()` (`pkg/ui/chargen.go`) selects
`TipTextDetail` when `c.stage != PreCreateStage`, in place of the earlier `PreCreateStage`-only gate.
`stepChargenDetailed` (`pkg/ui/app.go`) swallows a press/release the panel covers, the same discipline
`DIV-162`'s round-3 fix established for the other four panels. `ChargenTipRect = (160,280)-(472,480)`
(`TOWN-314`, hotfix `23bb8e6`) is unchanged by this story and is out of scope
(`TestEveryTipRectSitsAtItsResearchedOrigin`).

The panel opens by default on this page, so it legitimately overlaps part of the doll seam column at
`x:[464,472), y:[280,480)` — the last thing composed on the page. This is not a defect; see the test
note below.

## B6 — the town's DOLL/STATS toggle places the card in the character pane

`townStatisticsCardRegion` (formerly aliasing `chargenCardBox`) is removed.
`DrawTownCharacterRegion` (`pkg/ui/townshell.go`) composes `v.StatsPane` (`DIV-175`) and the
`CompactPanelLayout` card in place of the doll, at `TownCharacterRegion = (480,238)-(640,480)`, when
`v.Statistics` is true — uniformly at the tavern's, the school's and the shop's own call sites
(`DIV-193`):

- Cells, title, buttons and the bottom message no longer suppress during Statistics
  (`ComposeTownSurface`); this reverses `1021`'s own design of hiding the left content.
- `drawTownStatisticsSurface` is now blank-only (`drawTownShellBox`), used by the shop alone for its
  own grid-replacement behaviour; tavern and school no longer call it.
- The shop's own doll drag-and-drop (`shopGridControlAt`, `pkg/ui/shopscreen.go`) refuses a hit while
  `v.Character.Statistics` is true, checked first, before the doll-slot test. `app.go`'s cross-family
  drag-release destination check carries the same guard for a drag that originated before a mid-drag
  mode toggle.
- `ShopHoverLines` (`pkg/ui/shopscreen.go`) now checks `v.Character.Statistics` before testing the
  doll slot, matching `shopGridControlAt`'s own order. Round-2 adversarial review (P-2) found the
  prior order tested the doll slot first: a populated doll slot still produced its item-name tooltip
  over the card, though `shopGridControlAt` already refused the click on the same pixel.

**Round-3 adversarial review (P): the member-name row and the persistent chevron/mode-box/Book
chrome drew unconditionally over the card in Statistics mode.** `DrawTownCharacterRegion`
(`pkg/ui/townshell.go`) drew `townCharacterName` regardless of `v.Statistics`; the shop's own
`shopBookRect` label (`pkg/ui/shopscreen.go`) had the same defect. For a two-or-more-member party
(`v.MemberCount > 1`, the ordinary case) this landed on the card's own populated `SKILLS`/`BLADE`/
`AXE`/`BLUDGEON` rows, at production font metrics, making a real party member's card illegible.
Round-2's own witness did not catch it: `panelSubjectFixture()`'s near-empty values (HP 63/100) left
those rows blank, so the overlay painted onto empty space.

The population of chrome `DrawTownCharacterRegion` draws over `TownCharacterRegion` is closed by
code identity: `DrawTownCharacterRegion` has exactly two call sites in this tree —
`ComposeTownSurface` (`pkg/ui/townshell.go`, shared by both `TownSurfaceKind` values, tavern and
school) and `ComposeShopScreen` (`pkg/ui/shopscreen.go`). No other draw call lands inside
`TownCharacterRegion` in any composer. Fixed:

- At this story's landing the member-name row stopped drawing when `v.Statistics` was true because
  `CompactPanelLayout`'s own first row already stated the name. The later owner-directed name-card
  hotfix removes the separate row in DOLL mode as well: the statistics card is the only name surface,
  and the former row remains doll/equipment interaction space.
- The prev/next chevrons, the mode box and the shop's own Book toggle each get an opaque backing
  plate exactly when `v.Statistics`/`character.Statistics` is true, nothing when it is not
  (`drawShopChevron`'s own fill parameter; a `draw.Src` plate under `townCharacterMode` and
  `shopBookRect`). Round-2 had removed the mode box's own fill and the chevrons never had one, on
  the premise that the pixels under the control were either doll or empty pane; the card fills the
  same rectangles with real text at production font metrics, so an unbacked chevron or box border
  reads as noise mixed into the card's own digits. DOLL mode is byte-for-byte what round-2 left it.
- `TownCharacterPanelControls()` is renamed `TownCharacterPersistentControls()` and now returns three
  rectangles (chevrons, mode box), not four: the name row is no longer chrome drawn in every mode, so
  a caller comparing a Statistics-mode frame against the card must not exclude it. The two release
  witnesses that use it (`pkg/ui/shopscreen_test.go`'s and `pkg/ui/townshell_test.go`'s own
  Statistics-mode pixel comparisons, and `pkg/game/townpanes_release_test.go`, install-gated) are
  updated: the name rectangle is no longer excluded, and both unit tests switch from
  `panelSubjectFixture()`/`panelFont()` to a new realistic wide-valued fixture
  (`panelWideSubjectFixture()`/`panelWideFont()`, `pkg/ui/panel_test.go`) so the collision region is
  actually populated when compared, rather than blank.

Mutation-proved at the use site: reverting the three gates above (both files) makes
`TestShopStatisticsPlacesTheCardInTheCharacterPaneAndDisablesTheDoll` and
`TestTownStatisticsKeepsLeftContentAndPlacesTheCardInTheCharacterPane` fail (exit 1); restoring them
makes both pass (exit 0). See `closure.md`.

## Card value truncation: a card overflow this story found and fixed

**Round-3 adversarial review (P), a regression inside round-2's own P-5 fix.**
`CompactPanelLayout`'s `AlignValues` mechanism (`layoutLines`, `pkg/ui/panel.go`) computes each
column's shared right edge as an unclamped maximum over every row's own natural width, with no clamp
against `l.Size.X` (the fixed 160x242 card). `panelSubjectFixture()`'s near-empty two/three-digit
values never reached the card's own right edge, so round 2's own witness did not show this: a real
leveled character with three-digit HEALTH/MANA or two-digit DEFENSE/RESISTANCE values overflows or
clips past the card's right border for `HEALTH`, `MANA`, `ABSORB`, `DEFENSE`, `RESISTANCE` and the
five element rows.

`DIV-191` already rules on this exact card: the owner's 2026-08-21 ruling, quoted in that row, is
"truncation is acceptable and the main information must fit." No new divergence row is opened for
this fix; it implements a decision `DIV-191` already recorded.

Fixed in `layoutLines`: a new helper, `panelFitValue`, truncates a value string one character at a
time until it fits the space available before the card's own right/bottom pad, and two new passes —
one for left-column values, one for right-column values, both gated on `l.Size.X > 0` — call it before
`AlignValues`' own shared-edge computation runs. `panelFitValue` truncates to the empty string if
nothing fits (no one-character floor, unlike `townShellTextLayout`'s pattern): a floor still let
overflow through when a fixed authored label alone, not just its value, exceeded the available
budget. `l.Size.X > 0` is only true for `CompactPanelLayout`'s callers; `AuthoredPanelLayout` (the
mission column and the older detail panels, `Size.X == 0`, fit-to-content) is unaffected — verified
by running the full `pkg/ui` suite unchanged.

> **Corrected 2026-08-22 by story `1027` (D-1).** The sentence above is false.
> `AuthoredPanelLayout`'s `Size.X` is `sidebarWidth`, 300; only its `Size.Y` is 0, and the
> pinned axis is the one `layoutLines` reads. The truncation passes therefore do run on the
> mission panel. Measured with `cmd/paneldump`, identically on both roots: a shipped
> starting party member's `WORN Short Sword, Soft Mail, Soft Boots` row is 299 pixels
> against a 280-pixel budget and draws as `Short Sword, Soft Mail, ` on missions 10, 20, 30
> and 40. The `pkg/ui` suite passing unchanged is not evidence of the claim: no test in it
> composes `AuthoredPanelLayout` at a real font with a real worn set. Recorded as
> `DIV-218`; the mission panel itself is story `1023`'s.

`panelFitValue`/`layoutLines`'s new passes apply identically to every caller of `CompactPanelLayout`,
by code identity: `RenderCharacterPanel(CompactPanelLayout(...), ...)` is called from
`DrawTownCharacterRegion` (`pkg/ui/townshell.go:670`, town/shop) and from the chargen detailed page
(`pkg/ui/chargen_page.go:908`), both through the identical `layoutLines`, and `CompactPanelLayout`'s
own `Size` is the fixed `(160,242)` regardless of the `background` argument either caller passes. No
separate wiring is needed for chargen.

Mutation-proved at the use site (`layoutLines`'s two new passes): disabling them makes
`TestCompactPanelLayoutRightValuesFitTheBox` (algorithm-level, `panelFont()`) and
`TestCompactPanelLayoutTruncatesAWideValueRatherThanOverflow` (integration-level, through the real
`CompactPanelLayout(nil)` and a realistic wide-valued subject) both fail (exit 1); restoring makes
both pass (exit 0). See `closure.md`.

**Disclosed, not fixed: a related but distinct property of the same column-split algorithm.**
`layoutLines` splits each row into a left column (`widestLeft`) and a right column
(`rightX = widestLeft + l.ColumnGap`) before the truncation passes run; the truncation passes clamp
each column's own VALUE text, not the fixed authored LABEL text ("RESISTANCE", "DEFENSE"). At a
plausible real production font width a wide enough left column could in principle still push a
right-column label past the card's own right edge, independent of the value clamp this fix adds.
This is a pre-existing property of the column-split algorithm, not introduced by round-2's `P-5` or
by this fix, and it is not what `DIV-191`'s ruling or this round's brief scoped Defect B to (VALUES
overflowing, not the fixed label set). Verified empirically at the tuned test font/subject
(`panelWideFont()`/`panelWideSubjectFixture()`): the whole card's own rightmost painted pixel sits at
x=155, inside the 160px box, so no label overflows at that width. Left open rather than redesigning
the column-split algorithm, which is a materially larger change than a round-3 regression-scoped fix.

**Chargen's own card, played through the real character generator against the EN root, does not
reach overflow.**

> **Instrument named 2026-08-22 by story `1027` (W-1).** The measurement below came from a
> program that was deleted with its worktree, so nothing reproduced it. It is reproduced now
> by `TestReleaseChargenMaximumAllocationFitsTheCard`
> (`pkg/game/towncard_release_test.go`), which drives the same production API to budget
> exhaustion, prints the derived subject, and asserts the composed card truncates nothing.
> Its run on the `en` root reports `remaining=0`, `HP=91/91`, `Defence=17`, matching the
> numbers below.

A throwaway diagnostic (not committed) drove `ui.Chargen` through the production
API (`SelectPreChoice`, `Forward`, then `AdjustStat` in a loop until the point-buy budget was
exhausted) and composed the detailed page through `ui.ComposeChargenFrame`, the same call the shell
itself makes. At this install's maximum PreCreate allocation (BODY/AGILITY/MIND/SPIRIT 34 each,
budget exhausted at 0), the derived subject is `HP=91/91`, `Combat.Defence=17` — two digits
throughout, well inside the card's own width at production font. Chargen's own point-buy system does
not, in this install, produce a subject wide enough to exercise the truncation this fix adds; the
town's Statistics card (B6) shows a mid-campaign party member's current stats, which can exceed a
fresh chargen allocation through levelling and equipment, and is where a real overflow was found.
Both callers go through the identical `layoutLines`/`panelFitValue` code path (above), so the fix
carries to chargen with no separate wiring and no regression risk verified against real production
font metrics for chargen's own normal range.

## The production defect this story found and fixed

`check-release-tests.sh` (install-gated, not reachable from `go test ./...`) failed both `plate_seam`
and `doll_seam` subtests of `TestReleaseChargenDetailedSeamColumnsDrawShippedStrips` on both roots
after B1 widened `chargenColumnDestination` to `x:[160,480)`. The plate seam and the card seam, both
at `x:[160,176)`, were drawn BEFORE the column in the function body — the column's own opaque
`draw.Src` copy erased both immediately after. The nav pane's seam already drew after the column, with
a comment citing `TOWN-234`'s decoded order; the plate and card seams did not follow it. Fixed by
moving both draws to after `chargenColumnDestination` composes, immediately before the nav seam's own
draw. Verified: the test now passes on both class subtests, both roots.

A second, legitimate mismatch remained on `doll_seam` after the fix: the tip panel (B5, open by
default) draws last and covers `x:[464,472), y:[280,480)` of the doll seam column. This is correct
per-B5 behaviour, not a defect. The test's `assertSeamRegion` call now excludes
`dollSeam.Intersect(ui.ChargenTipRect)`.

## Divergences

Opened: `DIV-190` (four body-slot archive entries, size-matched, no claim names the reader), `DIV-191`
(the card's field arrangement and row order, and its reused font — WEIGHT's presence or absence is
NOT this row's question, see `DIV-209`), `DIV-193` (the DOLL/STATS toggle's card-replaces-doll
behaviour, uniform across tavern/school/shop). Closed: `DIV-189` (the lower-left border strip now
draws, moved to `DIVERGENCES-CLOSED.md`) and `DIV-192` (the message box's dropped hover/refusal copy —
now carried by `SetDetailMessage`/`drawChargenMessage`, moved to `DIVERGENCES-CLOSED.md`).
`DIV-194`..`DIV-198` are returned unused.

Round-2 adversarial review (D-1) found the WEIGHT row's absence had been folded into `DIV-191` and
recorded as accepted "on the owner's word," which the owner had not given — he ruled the opposite
(«нет, WEIGHT не принимал», 2026-08-21). `DIV-209` (new, OPEN, FIDELITY-DEBT) now carries the WEIGHT
question on its own: the row is not drawn because no field anywhere in `pkg/sim`/`pkg/game` carries a
carried-weight or encumbrance value, building one is outside this story's Client/Assets domains, and
story `1025` is named as the place that closes it. Round-2 review (D-2) also found `DIV-193`'s
Implemented-behaviour cell had not yet recorded the `ShopHoverLines` ordering fix (P-2, B6 above), and
its Reason cell overstated the doll-refusal ordering itself as owner-accepted rather than as this
story's own engineering consequence of the reuse the owner did direct; both are corrected in
`docs/DIVERGENCES.md`.

Round-3 adversarial review (pass 3, final for this story) opens no new divergence row: both P
findings it fixed (the chrome/name overlay and the value-overflow above) are covered by `DIV-191`'s
existing ruling, which already states truncation is acceptable and the arrangement is the owner's;
round 3 amends that row's Implemented-behaviour cell to record that the truncation half of the ruling
is now actually implemented, corrected from round 2 stating the arrangement was implemented without
also stating the fit was not yet enforced.

## Out of scope (unchanged from `contract.md`, plus two round-2 exclusions)

The map screen's own column (`sidebarWidth = 300`, story `1023`); the pre-create stage's own
composition, past its tip rect; splitting `shopmenu.bmp`; `interface/chrgen/centerarea.bmp`
(320x480, unreferenced in this tree, not drawn). Added round 2: the carried-weight/encumbrance value
itself (`DIV-209`, story `1025` — this story's own domains are Client/Assets, not the simulation
fields a WEIGHT row would read); SIGHT's fractional digit (a `Derived.Sight` representation change
reaching hashed simulation state, a different contract from this story's card composition — see B3).
