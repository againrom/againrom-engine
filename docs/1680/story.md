# Mission frames fit common monitor heights

## Result

A mission uses a 720-row picture in a short window and at heights divisible by 360.
A 1280x720 window shows that picture at its own size, without rescaling text.
The map has 22 complete cell rows; the lower card fits whole above the bottom and the other panels keep their positions.
Other window heights retain the 768-row picture, and town screens keep their original size.

## Authority

- Owner: use 720 rows below 768 or at heights divisible by 360; other heights keep 768. Both games.
- `SESS-VIEW-028`: the original has 480, 600 and 768 height arms; the map span divides height by 32.
- `DIV-210`: the 720-row height rule is owner direction beyond those original arms.

## Rules

1. A positive device-pixel window height below 768 or divisible by 360 selects 720 rows; every other height selects 768.
2. The mission keeps its width expansion rule and fits uniformly: 720, 1080 and 1440 give scales 1, 1.5 and 2.
3. With bottom bars closed, both games show 22 complete map rows at height 720 and 24 at height 768.
4. The lower card fits whole at the frame bottom, above overlapping art; other column panels keep their positions.
5. The 1024x768 and 1920x1200 pictures and game state remain unchanged; town screens retain their native frame.
6. Headless control activation uses the nearest representable window pixel inside the control.
7. The whole-card fit keeps the hero panel's rect C hidden and inert.

## Code

| kind | its one builder | change |
|---|---|---|
| mission frame size | `Viewer.Layout`, `frame.ExpandedWidth`, `frame.Fit` | select height from the window |
| right column boxes and mode gate | `missionCardBoxRect`, `rightColumnBox`, `hudLowerStack` | one whole-card fit gates rect C |
| map viewport | `syncMapViewport` | consume the selected frame height |
| mission layers | `missionLayers`, `Viewer.Draw`, `headlessMissionFrame` | retain their shared frame readers |
| final frame dispatch | `drawFinalFrame` | unchanged |
| current SAV | `captureSave`, `ExportCurrentSave` | unchanged; compare captured bytes |
| headless control pixel | `headlessControlPoint` | resolve through the pointer mapping when the centre has no pixel |

## Proof

| test | rules |
|---|---|
| `TestMissionFrameUsesLowWindowHeight` | 1, 2 |
| `TestMissionFrameAndPointerAtEveryDeviceScale` | 1, 2 |
| `TestMissionFrameMapRowsFollowHeight` | 3 |
| `TestMissionFrameColumnKeepsSlots` | 4 |
| `TestMissionFrameKeepsModeCornerHidden` | 4, 5, 7 |
| `TestMissionCardFitsShortFrameAndKeepsBasePlacement` | 4 |
| `TestMissionDrawComposesOnTheFrameAndPlacesIt`, `TestMissionCanvasFollowsLogicalBounds` | 1, 2 |
| `TestMissionFrameInstalledHeights` | 1, 2, 5 |
| `TestNativeTownCanvasIsReusedAcrossWindowResizes` | 5 |
| `TestHeadlessOutcomeControlUsesRepresentablePixel`, `TestHeadlessControlCentresUseOneFinder` | 6 |

Release tests at the release: `TestReleaseCommandPanelComposesFromRealArtInFourStates` on EN/RU of both games;
`TestReleaseMissionMinimapPaintsTheInstalledPicture` on ROM1 EN/RU;
`TestReleaseSecondGameMinimapMarks` on ROM2 EN/RU.

## Open debt

- The seat runs the fast gate and the review outside the lane sandbox.
- The CPU mission capture omits the layers named by `missionLayers`; it is not a full window capture.
- The direct second-game mission fixture has no saveable campaign; its SAV control remains unavailable.
