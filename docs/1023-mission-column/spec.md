# Story `1023` — the mission screen's right column at its decoded geometry (as built)

This document states the behaviour shipped by story `1023`, not the behaviour intended before it.
`contract.md` states the intent and is not rewritten. Where the two differ, the differences are
listed under "Differences from the contract" at the end.

## FR-1 — the column is 160 pixels wide

`sidebarWidth` (`pkg/ui/panel.go`) is `MissionPanelW`, 160 (`pkg/ui/viewer.go`, story `1026`). Every
reader in the mission's own right column follows it: `hudBandX`, `rightColumnBox`, `minimapGeometry`,
`hudToggleBarRect`, `characterPanelBoxRect`, `dollBoxRect`, `wornBoxRect` (`pkg/ui/hud.go`,
`pkg/ui/minimap.go`, `pkg/ui/inventory.go`).

`AuthoredPanelLayout` no longer reads `sidebarWidth`. It has its own constant,
`authoredPanelWidth = 300` (`pkg/ui/panel.go`), the value `sidebarWidth` itself carried from
`0140` through `1022`. This decouples the two: `AuthoredPanelLayout` was the mission's own live panel
before this story, so one constant correctly served both concerns at once; this story replaces the
mission's live panel with `CompactPanelLayout` (FR-3), so the coupling no longer holds. Splitting the
constant means `AuthoredPanelLayout`'s remaining callers — `pkg/game`'s chargen tests and
`cmd/paneldump` — see no change at all.

**The column's own left edge is flush to the frame's own right edge.** Round 1 of adversarial review
found `rightColumnBox` (`pkg/ui/hud.go`) drawing the whole column 12 pixels inside that edge, over the
map viewport: `rightColumnBox` subtracted `hudMargin` from `area.X` before subtracting `sidebarWidth`,
so at the shipped 1024x768 frame the column stood at x 852..1012 instead of `SESS-VIEW-028`'s own
864..1024. Round 2 removed that subtraction: `rightColumnBox` now returns
`image.Rect(area.X-sidebarWidth, y, area.X, y+h)`, with no `hudMargin` term. `hudMargin` remains in use
elsewhere in this file (the pack and spellbook bars, and `AuthoredPanelLayout`'s own town/chargen
callers) but no longer reaches the mission column's own placement.

## FR-2 — the map viewport is the frame minus the column

Unchanged by this story. `MissionViewportSize()` was already `image.Pt(MissionFrameW-MissionPanelW,
MissionFrameH)` = 864x768, built by story `1026`. `MissionPanelRect()` is `(864,0)-(1024,768)`. The
contract's own B2 asked this to be established and matched; it was already true, and this story adds
no viewport code.

## FR-3 — the character panel occupies the decoded id-7 slot, reusing `CompactPanelLayout`

`Viewer.panelLayout` defaults to `CompactPanelLayout(nil)` (was `AuthoredPanelLayout()`).
`Viewer.panelPresent` and `Viewer.panelRect` read `characterPanelBoxRect(image.Pt(v.frameW,
v.frameH))` instead of a fit-to-content box anchored to a panel corner of `v.cam`'s own view.

`characterPanelBoxRect` places a `sidebarWidth x compactPanelH` (160x242) box with its top at
`hudFloorY()`. `hudFloorY()` returns the literal `hudPanelTopY`, 238 — `SHOP-FIGURE-041`'s own id-7
top boundary at 1024x768 (local rect `(0,238,160,480)`, offset onto the column) — rather than a value
derived from the boxes above it. The panel's own height, `compactPanelH` (242), is that same slot's
height, so the panel occupies `(238..480)` exactly: top, height and bottom all match the decode. This
is the one boundary in the column the build reproduces as a literal; every other boundary stays
authored (FR-4, FR-5).

The panel draws at `v.cardFont()`, a new resolver (`SetCardFont`/`cardFont`, `panel.go`) that returns
`v.panelCardFont` if set, else `v.font`. `pkg/game/frontend.go` calls `mv.Viewer.SetCardFont(f.tipFont())`
at both mission-open call sites, immediately after the existing `SetFont(f.Font)` call — the same
resolver shape `pkg/ui/townshell.go`'s `TownCharacterView.cardFont()` already used for the town's own
card (story `1021`).

`layoutLines`' `panelFitValue` pass, which truncates a row too wide for its own layout, no longer runs
on the mission panel at all: `CompactPanelLayout` carries no WEAPON or WORN row (story `1022` B3), so
there is no row of that shape to fit or cut during a mission.

## FR-4 — the minimap and the control panel keep the owner's ordering, at the decoded sizes

`hudLowerStack` places the minimap above the control panel, matching the owner's 2026-08-11 ordering
("the control panel UNDER the minimap"), unchanged from before this story. Round 1 of adversarial
review shipped this at the two slots' own pre-existing authored sizes, `hudMinimapReserve` = 160 and
`hudToggleBarH` = 42, with a 12-pixel gap between them; round 2 replaced both with the decoded
literals, `hudMinimapReserve` = 158 and `hudToggleBarH` = 80 (`SHOP-FIGURE-041`'s own ids 5 and 6,
`(0,0,160,158)` and `(0,158,160,238)`), contiguous, per the owner's 2026-08-21 ruling that a decoded
value collects an authored one where no directive drove the difference. The two now sum to
`hudPanelTopY` (238) exactly, pinned by `TestHudColumnSlotsSumToThePanelsOwnTop`
(`pkg/ui/hud_test.go`).

The minimap's own box is not 160 wide at 158 tall: `minimapGeometry` (`pkg/ui/minimap.go`) clamps the
square to the shorter of the two, 158, and insets it 2 pixels inside the reserved 160-wide slot, so the
square stays square (owner, 2026-08-11: "the map must always be square") without claiming a width the
decode gives to a different id. This is the one place in the column where the drawn box is not flush
with the reservation on all four sides.

`DIV-201` is narrowed at this landing to the one thing it still records: no claim names either id's own
tenant, so there is no decoded assignment (which widget is id 5, which is id 6) to diverge from or
agree with — only the sizes are now decoded and matched; the assignment stays the owner's own
directive.

## FR-5 — the doll follows the panel by authored choice; the worn box does not draw during a mission

`hudLowerStack` places the doll immediately under the character panel: `doll = panel + compactPanelH
+ hudBarGap` = 238+242+8 = 488. `dollBoxRect` returns a `sidebarWidth x dollBoxSize().Y` (160x260) box
there, and `rightColumnBox`'s own bound (`y+h <= area.Y`, 748 <= 768 since round 2 removed the
`hudMargin` term from this bound as well as from the column's own left edge) allows it, with 20 pixels
to spare.

The worn box is offered a slot at `worn = doll + dollBoxSize().Y + hudBarGap` = 756, and
`rightColumnBox` refuses it: `756+224 = 980 > 768`, 212 pixels past the frame's own bottom edge.
`wornBox()` (`pkg/ui/inventory.go`) reflects the refusal directly — it never measures a picture before
consulting the rect, so a mission viewer never composes a worn picture that goes unused. `DIV-200`
carries the divergence: no claim names a doll or worn presentation anywhere in the decoded column, so
the choice (keep the doll, drop the worn box) is this project's own, not a reading of any slot.

`invSlotColumns` is 3, not 4 (`pkg/ui/inventory.go`): four columns wrap to 204 pixels of grid, which
fit the former 300-pixel box but not this one; three wraps to 152, four rows deep, which does. This
changes `wornBoxSize().Y` from 172 to 224, which is the number FR-5's own arithmetic above uses. The
worn box is never drawn during a mission at the shipped frame size regardless, but `wornSlotRects` and
`RenderWorn` are tested directly and have to describe a valid, non-overlapping grid independent of
whether a mission ever draws it.

## FR-6 — every right-column and lower-band reader takes the frame, not the camera view

Before story `1026` the camera's view and the frame were the same size, because the map filled the
whole window and the HUD was an authored overlay on top of it. Since `1026` the camera's view is the
map viewport alone (864x768 at 1024x768), narrower than the frame by exactly `sidebarWidth`. A
function that kept reading the camera's view would carve a second, nested `sidebarWidth` out of an
already-narrowed viewport.

This story found and fixed five such call sites, all using `image.Pt(v.cam.ViewW, v.cam.ViewH)` where
the frame was owed:

- `minimapGeometry` (`pkg/ui/minimap.go`)
- `hudToggleBarRect`'s caller, `hudToggleBar` (`pkg/ui/hudtoggles.go`)
- `spellbookBar`'s caller of `bookBarRect` (`pkg/ui/spellbook.go`)
- `itemPopupPresent`'s clamp bounds (`pkg/ui/itempopup.go`) — the popup draws over the doll, the worn
  box and the pack, all in frame space, and clamping it to the camera's view left it able to draw
  partly inside the reserved column
- `dollBox`, `wornBox`, `wornBoxArea`, `packBar`, `packBarArea` (`pkg/ui/inventory.go`), which also
  dropped their now-unused `panelH` parameter — the fit-to-content unit panel FR-3 replaces never
  passed a height these functions needed to avoid

`panelBoxSize` (`pkg/ui/minimap.go`) is deleted: dead code once `inventory.go` and `panel.go` stopped
calling it.

## Witnesses

- `pkg/ui/missioncolumn_test.go`'s
  `TestMissionColumnDrawsThePanelDollMinimapAndControlPanelAtTheirOwnSlots` records the four
  `blitColumnLayer` calls `Viewer.Draw` makes for the panel, the doll, the minimap and the control
  panel, through the package variable `blitColumnLayer` (new, mirroring `blitFrameCanvas` from story
  `1026` for the same reason: ebitengine refuses a pixel readback with no graphics context). Round 2 of
  adversarial review replaced the test's own literal left edge (`852`, baked in from `area.X -
  hudMargin - sidebarWidth`, internally consistent with the round-1 P finding and blind to it) with
  `MissionViewportSize().X`, story `1026`'s own camera-area function, an independent formula that never
  carried `hudMargin`. Every expected y origin and size is still written out by hand from
  `hudPanelTopY`, `compactPanelH`, `hudBarGap`, `hudMinimapReserve` and `hudToggleBarH`, not read back
  from the rect-returning functions under test. It also asserts the worn box makes no call at all.
- `pkg/ui/hud_test.go`'s `TestHudColumnSlotsSumToThePanelsOwnTop` (new, round 2) pins
  `hudMinimapReserve + hudToggleBarH == hudPanelTopY`, the invariant that keeps the two authored-turned-
  decoded slots meeting id 7's own boundary with no gap.
- `pkg/ui/minimap_test.go`'s `TestMinimapBoxIsSquareAndDoesNotMoveWithTheSelection` was updated at
  round 2: the box's own width assertion now reads `hudMinimapReserve` (158), not `sidebarWidth` (160)
  — the two parted ways when the minimap's own bound stopped being the unit panel's width and became
  id 5's own decoded height.
- `pkg/ui/screenregistry.go`'s `ScreenMap` entry gains this test as a third `GeometryTests` string.
- `pkg/game/release_integration_test.go`'s `TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity`
  rebuilds its oracle on `ui.CompactPanelLayout(nil)` (was `ui.AuthoredPanelLayout()`), asserts all
  lines equal with no carve-out (`CompactPanelLayout` has no CELL row, so no placement-dependent line
  remains to exclude, unlike the layout it replaces), and runs — not skips — against `AGAINROM_ASSETS`
  on both roots. The line count is 17, not 16: story `1025` (merged into this branch ahead of this
  story's own landing) added `CompactPanelLayout`'s own WEIGHT row, one line taller.
- Existing panel, inventory, minimap and hit-test suites were extended in place where this story's
  geometry change moved their own fixtures' expectations (`panel_draw_test.go`, `inventory_test.go`,
  `invclick_test.go`, `minimap_test.go`, `viewportfixture_test.go`), detailed in `closure.md`'s
  mutation table.

## Differences from the contract

- **B4's own citation was corrected.** The contract characterizes ids 5 and 6's undecoded identity as
  "research is Medium on widget identity," citing `SESS-VIEW-028`'s own Medium grade. Reading
  `SESS-VIEW-028` and `SHOP-FIGURE-041` whole at this landing found that imprecise:
  `SESS-VIEW-028`'s Medium clause names a different construction site's child 2/3 (the map view's own
  Inventory/SpellBook toggle recompute, `DIV-214`'s territory), and no claim carries any confidence
  grade for ids 5 and 6. `DIV-201` is recorded as `UNKNOWN`, not as a Medium reading.
- **The column's own placement was not pixel-exact at round 1, on two axes, and round 2 of adversarial
  review corrected both.** Horizontal: `rightColumnBox` subtracted a stray `hudMargin` from the frame's
  own right edge, drawing the whole column 12 pixels inside `SESS-VIEW-028`'s own 864..1024 strip, over
  the map viewport — a P finding (player-visible). Vertical: the minimap's and control panel's own
  reserved heights shipped at their pre-existing authored sizes, 160 and 42, not `SHOP-FIGURE-041`'s own
  decoded 158 and 80, with a 12-pixel gap between them the decode does not carry. Both are now the
  decoded literal (FR-1, FR-4); `DIV-201` is narrowed rather than closed, to the one thing that stays
  undecoded — which of ids 5 and 6 is the minimap and which is the control panel.
- **`DIV-213` and `DIV-218` close, and `DIV-217` is amended, not opened fresh.** The contract named
  `DIV-217` as pre-existing and out of scope; this story's own change to which layout composes the
  mission panel (`CompactPanelLayout`, not `AuthoredPanelLayout`) made one clause of that row's own
  text stale, and it is corrected in place rather than left to describe a composer this build no
  longer calls.

See `closure.md` for the twelve-aspect matrix, the mutation table and the research reconciliation.
