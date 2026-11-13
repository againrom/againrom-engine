# The zoomed mission picture is scaled once from its own pixels

## Result

A mission map zoomed by the wheel is now scaled a single time, from the pixels it was drawn with, straight
to the window. Before, on a window larger than the game's own frame, the zoomed map was filtered twice and
the health bars were cut to uneven rows at a zoom that is not whole. The bars, selection marks and grid now
zoom with the map as one picture.

Smoothing chooses the map's filter: on is smooth, off gives hard pixels.
The interface and its text keep the starter's filter at either setting. While a mission dialogue is open
the map is scaled twice, as before.

## Authority

- Owner: the picture is scaled once, from its own pixels, with a good filter; no stage resamples an already
  resampled picture; Smoothing stays one switch. Nearest neighbour below a factor of 2 with Smoothing off is
  the coordinator's correction order.
- DIV-4876: the zoomed mission map is composed at zoom 1 and scaled; this story restates its row.
- DIV-4919: what Smoothing off means in a mission; open until the owner confirms the meaning.
- `TOWN-SMOOTH-460`: the original's Smoothing has no scaling meaning; `MENU-078`: the original has no zoom.

## Rules

1. In a mission with no dialogue open, each picture reaches the window through at most one scaling.
2. At a zoom other than 100 percent the map, with status bars and marks, is scaled once by zoom times window scale.
3. The interface, paths and numerals are drawn at their own pixels on the canvas, scaled once by the window scale.
4. At a factor of exactly 1 the scaler copies the picture and filters nothing.
5. Smoothing on scales the map with the Catmull-Rom kernel; its weights sum to 1 and a flat colour stays flat.
6. Smoothing off: the map by nearest below 2, from 2 whole pixels and a linear step; the canvas by the starter's filter.
7. The wheel still changes the zoom and posts the zoom line; Home still restores the opening zoom.
8. The map editor's draw is unchanged.
9. The drag selection rectangle is drawn on the canvas in frame pixels at every zoom.
10. While a mission dialogue is open the map is placed on the canvas, so the backdrop and frame shadows reach it.
11. The cell grid hides where a cell is under 16 pixels at the player's zoom.

## Code

| kind | its one builder | change |
|---|---|---|
| final-frame scaling | `pkg/ui/framesmoothing.go` `drawFinalFrame` | takes a filter; the pixel filter is nearest below 2; the zoomed map reaches it once |
| zoomed map composition | `pkg/ui/mapzoom.go` `smoothMapPlacement`, `drawSmoothMap` | answers with Smoothing off too; composes the map-space marks; `blitMapToWindow` places it |
| map marks | `pkg/ui/viewer.go` `drawMapMarks` | one method draws the map-space part into the composition and the frame-space part on the canvas |
| mission filter choice | `pkg/ui/mapzoom.go` `mapFilter`, `canvasFilter` | the map: Catmull-Rom with Smoothing on, the pixel filter with it off; the canvas: the starter's switch at either setting |
| dialogue backdrop | `pkg/ui/dialoguebackdrop.go` `DialogueBackdropPlan`, unchanged | `canvasRemapsMap` keeps the map on the canvas while the plan shows |
| headless frame request | `pkg/ui/headlesspicture.go` `HeadlessPicture` | `scaled` answers the window frame, the mission included, and names its stages |
| CPU mission frame | `pkg/ui/headlessmission.go` `headlessMissionFrame` | the stages' reference in Go; it composes no notice and no dialogue backdrop |

The kernel is Catmull-Rom (Keys cubic, a = -0.5): 4 by 4 source pixels, support 2 pixels each side. It keeps
flat colour and linear ramps exact and its overshoot at a hard edge is at most 7.4 percent of the step, judged on the
400 percent crops of an interface edge and a text line in the lane notes; Lanczos-3 rings wider on the same edges.

## Proof

| test | rules |
|---|---|
| `TestMissionFrameIsResampledOnceOntoTheWindow`, `TestSmoothingOffKeepsTheStartersFilterOnTheInterface` | 1, 2, 3, 5, 6 |
| `TestDrawBlitsTheSmoothMapOnlyWhereTheRuleAnswers`, `TestSmoothMapPlacementRule` | 2 |
| `TestFrameSmoothingDispatchAtEverySite`, `TestTheDimIsComposedAfterEverythingButThePopup` | 3, 4 |
| `TestReferenceFinalFrameCopiesAndKeepsFlatColour` | 4, 5, 6 |
| `TestWeightsSumToOne`, `TestResampleScaleOneIsIdentity`, `TestResampleLinearRampStaysLinear` | 4, 5 |
| `TestPixelFilterIsNearestBelowTwo` | 5, 6 |
| `TestSharpBilinearNonIntegerScaleUpscalesTheIntermediateByAnExactInteger` | 6 |
| `TestWheelNotchPostsTheZoomAndTheNextReplacesIt`, `TestHeadlessHomeKeyResetsTheZoom` | 7 |
| `TestMapEditorKeepsItsZoomedDraw` | 8 |
| `TestDragRectangleStaysOnTheCanvasInFramePixels` | 9 |
| `TestDialogueBackdropReachesTheZoomedMap` | 1, 10 |
| `TestGridFloorReadsThePlayersZoomInsideTheComposition` | 11 |

Release tests at the release: none. EN and RU witnesses owed: an installed mission in a window at zoom 120 and
83 percent with Smoothing on and off, with and without a dialogue, read back from the GPU. Frames: the lane notes.

## Open debt

- GPU frame time is unmeasured; the zoomed frame shades about 1.4 times the pixels of before at 2560 by 1440.
- Catmull-Rom has no prefilter: fine detail aliases when zoom times window scale falls below about 0.5.
- Past the 4096-pixel composition cap (deep zoom out) sprites are drawn at the zoom, then the canvas is scaled.
- With Smoothing off and a dialogue open the map is on the canvas, so the window scale gives it the starter's
  filter, not hard pixels.
- Home's zoom reset posts no zoom line.
