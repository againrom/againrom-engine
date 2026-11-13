# The picture is drawn in the screen's own pixels at any Windows display scale

## Result

At a Windows display scale of 125, 150 or 200 percent the game now draws its picture straight into the
screen's real pixels. Before, it drew a smaller picture and Windows-side scaling stretched it once more, so
the game's own scaling was followed by a second blur. Menus, towns, missions and films now pass through one
scaling only, and a click still lands on what is drawn under the pointer. The map editor keeps its panel
and text sizes and is scaled once, with the game's smooth filter, to the screen's pixels.

## Authority

- Owner: the picture is resampled once by a good filter; no stage resamples a picture another stage has
  already resampled. Owner on the board card: «Делать следующей».
- No claim applies: the original games ran full screen at their own resolution and know no display scale.

## Rules

1. At any device scale the front end's frame is the window's device pixels, and the library copies it at 1.
2. Every picture of a menu, town, mission or film frame is scaled once, by the placement in device pixels.
3. A pointer on a device pixel lands on the same frame pixel, menu button and map cell as at scale 1.
4. The map editor lays out in points, draws in points, and its frame is scaled once to the device pixels.
5. The map editor reads a device pixel as the point drawn over it.
6. At a device scale of 1 every frame is drawn as before; Smoothing and the starter's switch keep their meaning.
7. Window sizes, full screen and the monitor size stay in points, as the player and the library state them.

## Code

| kind | its one builder | change |
|---|---|---|
| layout size | `pkg/ui/devicescale.go` `windowPixels` | new; `LayoutF` of the App, a lone Viewer and the editor ask it |
| final-frame scaling | `pkg/ui/framesmoothing.go` `drawFinalFrame`, unchanged | the editor's points frame reaches it once |
| zoomed map composition | `pkg/ui/mapzoom.go` `blitMapToWindow`, unchanged | scales to device pixels because the placement is in them |
| pointer to frame mapping | `frame.Placement` through `windowToFrame` and `windowToNativeFrame`, unchanged | the library hands device pixels |
| editor pointer | `pkg/ui/devicescale.go` `readNative` | the editor's one input door; device pixel to point |
| frame guard | `pkg/ui/mapeditor.go` `frameGuard.LayoutF` | passes the editor's `LayoutF` through |
| headless frame request | `pkg/ui/headlesspicture.go` `HeadlessPicture`, unchanged | answers the device-size frame and its stage |

## Proof

| test | rules |
|---|---|
| `TestDisplayScaleMeasurement` | 1, 4 |
| `TestWindowPixelsRoundsAsTheLibrary` | 1, 7 |
| `TestFrontEndFrameAndPointerAtEveryDeviceScale` | 2, 3 |
| `TestMissionFrameAndPointerAtEveryDeviceScale` | 2, 3 |
| `TestMapEditorFrameAndPointerAtEveryDeviceScale` | 4, 5 |
| `TestMissionFrameIsResampledOnceOntoTheWindow`, `TestFrameSmoothingDispatchAtEverySite` | 6 |

Release tests at the release: none. EN and RU witnesses owed: at 125 and 150 percent display scale, the main
menu, a town, a mission zoomed and at 100 percent, a film, and the map editor, read from the screen. Measurement
table, cost and witness steps: `review/story1653-display-scale/RESULT.md`.

## Open debt

- The map editor's zoomed map is drawn at the zoom and then scaled by the device scale: two stages.
- The starter is its own program with a fixed frame the library stretches; it does not use this path.
- Moving the window to a monitor of another scale is followed on the next frame; it is not witnessed.
- GPU frame time at 2560 by 1440 and scale 1.5 is unmeasured; the CPU draw cost is unchanged.
