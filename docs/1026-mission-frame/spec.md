# Story `1026` — the mission screen's virtual 1024x768 frame (as built)

This document states the behaviour shipped by story `1026`, not the behaviour intended before it.
`contract.md` states the intent and is not rewritten. Where the two differ, the differences are
listed under "Differences from the contract" at the end.

## FR-1 — a placement carries its own frame size

`pkg/render/frame.Fit` takes the frame size as two parameters:

```go
func Fit(frameW, frameH, winW, winH int) Placement
```

A `Placement` stores `frameW`, `frameH`, `winW`, `winH` and the scale as the exact rational
`num/den`. Every mapping on it reads the frame size the placement was fitted with. `FrameSize()`
reports that size, and the zero `Placement` reports the zero point.

`Fit` yields an unusable placement when any of the four values is not positive. `frame.W` and
`frame.H` remain 640 and 480 and are now the town family's own frame size: nothing inside the
package reads them.

The package's three invariants are unchanged and hold for any frame size:

- the whole frame is inside the window, for any window at least 1x1;
- a window position maps to at most one frame pixel, and a position in the letterbox maps to none;
- `WindowToFrame` after `FrameToWindow` is the identity wherever `FrameToWindow` reports ok.

`WindowToFrameExtended` is new. It maps a window position to a frame position and continues the
frame's own pixel lattice outside the frame instead of reporting a miss, reporting false only for an
unusable placement. It agrees with `WindowToFrame` wherever `WindowToFrame` answers. It exists for a
continuous cursor and not for a hit test: the mission's edge scroll asks whether the cursor is inside
the view and its drag asks how far the cursor moved, and both need a position for a cursor that has
left the frame.

## FR-2 — the mission composes at 1024x768 and that frame is placed in the window

`pkg/ui` declares:

```go
MissionFrameW = 1024
MissionFrameH = 768
MissionPanelW = 160
```

A `Viewer` holds `frameW`, `frameH` and a `frame.Placement`. `Viewer.Layout` computes
`frame.Fit(v.frameW, v.frameH, outsideWidth, outsideHeight)`, sets the camera's view to
`viewportSize(v.frameW, v.frameH)` and returns the window size back to Ebitengine, so the window
still carries one device pixel per window pixel.

`Viewer.Draw` composes the whole mission on a `frameW x frameH` `ebiten.Image`, clears it first, and
then draws that canvas onto the window with `GeoM.Scale(scale, scale)` followed by
`GeoM.Translate(originX, originY)` and `Filter = ebiten.FilterNearest`. `drawFrame` is the previous
body of `Draw` and composes onto that canvas. Nothing is drawn when the placement is unusable.

## FR-3 — the viewport is the frame minus the 160-pixel right strip

`MissionViewportSize()` is `image.Pt(MissionFrameW-MissionPanelW, MissionFrameH)` = 864x768.
`MissionPanelRect()` is `image.Rect(864, 0, 1024, 768)`. The two partition the frame: their union is
the frame rectangle and their intersection is empty.

At native zoom the map view spans `864/32 = 27` columns by `768/32 = 24` rows, which is
`SESS-VIEW-028`'s own pair for a 1024x768 screen. `startColumns` is 27, so `applyStartView` sets the
start zoom to `ViewW / (27 * CellSize)` = 1.

`viewportSize` clamps a width below zero to zero, so a frame narrower than the strip has no viewport
rather than a negative one.

## FR-4 — window positions become frame positions at three doors

Every hit test on the mission screen reads frame pixels. The conversion happens at exactly three
places, and no individual hit test changed:

- `Viewer.step` maps `in.CursorX, in.CursorY` at entry, before the cursor is stored or read;
- `Viewer.command` maps `in.CursorX, in.CursorY` at entry, before any surface is asked;
- `Viewer.noticeButtonAt` maps its arguments before the notice's own placement is applied.

`Viewer.windowToFrame` performs the conversion through `WindowToFrameExtended` and returns its
arguments unchanged for an unusable placement.

The headless pointer oracles run the conversion in the other direction. `Viewer.frameToWindow`
converts a frame point to a window pixel through `Placement.FrameToWindow` and reports an error when
a frame pixel has no window pixel. `HeadlessDollSlotPoint`, `HeadlessPackCellPoint` and
`HeadlessDollBoxPoint` return through it; `HeadlessGroundPoint` skips a frame pixel with no window
pixel and continues scanning; `HeadlessDropCell` maps window to frame first;
`headlessSelectEntityOnce` converts its `entityPickRect` point before building its press and release.

## FR-5 — a press off the map view names no cell

`Viewer.groundCellAt` refuses a point outside the camera's view before it resolves a cell. The frame
is larger than the view, so there are two such places: the 160-pixel right strip, where the map is
not drawn, and the letterbox outside the frame. Both resolve through the camera transform to cells
that are on the grid and were never drawn. The refusal is the same one the method already gave a
point past the edge of the grid: no order is issued, and a tap there clears the selection.

## FR-6 — the town family is unchanged

`App.Layout` and `NewApp` call `frame.Fit(frame.W, frame.H, a.winW, a.winH)`. The menu, the picker,
the chargen pages, the town screens and the in-game menu panel compose at 640x480 as before, and the
menu's integer-multiple window rule is untouched. A window now carries two placements: the app's own
640x480 one and the viewer's 1024x768 one.

## Witnesses

- `pkg/render/frame/frame_test.go` runs the whole package contract at five frame sizes: 640x480,
  1024x768, 800x600, 320x200 and 97x131, each in its own size, a wider window, a taller window and a
  smaller window. It also pins `FrameSize`, the non-positive frame, and the extended mapping's
  agreement with `WindowToFrame` inside the frame and its continuation outside it, with expectations
  built from `Origin()` and `Scale()` rather than from the mapping under test.
- `pkg/ui/missiongeometry_test.go` records the `DrawImage` call `Viewer.Draw` makes, through the
  package variable `blitFrameCanvas`, and asserts the composed canvas is the frame's own size and the
  transform is the scale and origin the window implies, at eight window sizes. Every expected scale
  and origin is written out by hand from the window size and 1024x768. It also asserts the frame's
  partition into viewport and strip and the 27 by 24 spans.
- `pkg/ui/hitscale_test.go` drives the shipped hit tests at 1:1, 2:1, 1:2 and at two windows whose
  aspect is not the frame's, and asserts a tap selects the unit it is aimed at, a right press orders
  the cell it is aimed at, the last drawn column still orders, and a press in the strip or in the
  letterbox orders nothing. Its window positions are computed from the case's own hand-written scale
  and origin. `TestUnmappedWindowPixelsWouldMissTheirSurface` is its control: at 2:1 the same numbers
  taken as window pixels reach a different cell.
- `pkg/ui/screenregistry.go` gives `ScreenMap` a `GeometryTests` list naming the two geometry tests.
  `cmd/screencensus` now prints a refusing screen's own geometry tests, which it dropped before.

## Differences from the contract

- **The contract's B4 asked for a hit-test population and got a boundary instead.** No mission hit
  test was changed. The population that had to be enumerated was therefore the set of places a window
  coordinate enters the mission screen, and that set is the three doors above plus the synthetic
  entry points. `closure.md` states how the population was established and what the instrument cannot
  see.
- **FR-5 is not in the contract.** The contract's B3 gives the strip but says the strip's contents
  are out of scope. Making the frame larger than the map view creates two regions where a press
  previously could not land, and leaving them to the ground path is a player-visible defect.
- **`DIV-213` is not closed by this story alone.** The authored mission HUD's right column, as this
  story shipped it, does not fit a 768-pixel frame; it closes at story `1023`'s combined landing in the
  same push, which replaces the authored column with the decoded one. See `closure.md`.
