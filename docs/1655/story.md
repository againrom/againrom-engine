# One smoothing switch for the game

## Result

Game Options and Ctrl+O use one saved smoothing value across missions, towns and menus.
With smoothing off the mission map and films keep hard pixels.
Interface text and combined town and menu pictures keep even scaling, and the editor frame stays smooth
at raised Windows display scale.

## Authority

- Owner: «Один переключатель»: Ctrl+O is the one switch, with one stored value followed by every screen.
- Owner correction: the starter has no smoothing control and receives none; Game Options keeps its row.
- Owner: the mission map stays pixel-hard with smoothing off, including zoom 100 percent; interface text
  stays evenly scaled; the editor frame at raised device scale stays smooth.
- `TOWN-SMOOTH-460`, read at knowledge k461: High for the boundary draw gate and synthetic pixel execution,
  Medium for the static loader/dispatch join. Full family coverage and native screen equivalence are Unknown.
- Existing mission Smoothing wins over obsolete FrameSmoothing, including the default on when the mission
  key is absent. This retains the player's old mission-map choice. The next options write drops the old key.
- Starting-state table with base file/line receipts: the first commit, `16fd74332`, and lane notes
  `review/story1655-one-smoothing-switch/STATE.md`. The starter had no smoothing control in that base.

## Rules

1. Ctrl+O in a focused game screen changes the same preference the Game Options row shows and sets.
2. The existing profile stores one Smoothing value; restart keeps it and other preferences survive writes.
3. With smoothing on, the existing Catmull-Rom and exact-copy paths remain unchanged at device scale 1.
4. With smoothing off the mission map, including zoom 100 percent, and film frames use the pixel filter.
5. Interface text, town/menu composites and mission dialogue composites keep the text-safe filter when off.
6. The editor frame stays smooth; O and Ctrl+O retain Open and the editor's zoomed draw stays unchanged.
7. The zoom line, Home, wheel, device-pixel placement and pointer mapping retain their existing meanings.
8. The setting names one smoothing switch in English and Russian; no simulation or SAV value is added.

## Code

| kind | its one builder | change |
|---|---|---|
| stored preferences | `OptionsStore.gameOptions`, `setGameOption`, `writeAll` | one Smoothing row, old frame key retired |
| game option | `gameOptionTable`, `gameOptionRows`, `setGraphicsOption` | shared captions and live presentation flag |
| game keys | `missionBindings`, `App.step` | existing Ctrl+O row dispatches before screen input |
| filter choice and final scaling | `filterOf`, `mapFilter`, `canvasFilter`, `drawFinalFrame` | content policy selects text-safe or pixel off filtering |
| mission composition | `smoothMapPlacement`, `Viewer.Draw` | off at unit zoom also separates the map from the interface |
| device scale and editor frame | `windowPixels`, `MapEditor.frameFilter` | fixed smooth editor policy; geometry unchanged |
| headless frame | `HeadlessPicture`, `headlessMissionFrame`, `referenceFinalFrame` | follows the same filter choices |

## Proof

| test | rules |
|---|---|
| `TestSmoothingKeyChangesEveryScreen`, `TestGameOptionTableBuildsThePageKeysAndCommit` | 1, 2, 8 |
| `TestOneStoredSmoothingKeepsTheMissionPreference`, `TestFrameSmoothingPreference` | 2 |
| `TestSmoothingOnFramesKeepTheirPixels`, `TestFrameSmoothingDispatchAtEverySite` | 3, 4, 5 |
| `TestFilmSmoothingOffUsesHardPixels`, `TestMissionFrameIsResampledOnceOntoTheWindow` | 3, 4 |
| `TestTownAndMenuFramesFollowTheSharedSmoothing`, `TestSmoothingOffKeepsTextSafeOnTheInterface` | 1, 5 |
| `TestDialogueReferenceKeepsTheMapOnTheTextSafeCanvas` | 5 |
| `TestEditorFrameStaysSmoothWithSmoothingOff`, `TestMapEditorKeepsItsZoomedDraw` | 6 |
| `TestWheelNotchPostsTheZoomAndTheNextReplacesIt`, `TestHeadlessHomeKeyResetsTheZoom` | 7 |
| `TestMissionFrameAndPointerAtEveryDeviceScale`, `TestFrontEndFrameAndPointerAtEveryDeviceScale` | 3, 7 |
| `TestSmoothingUsesOneStoreKeyAndFilterSelector`, `TestGraphics1190PreferencesAndFailureKeepWorld` | 2, 8 |

Release tests at the release: none. EN and RU witnesses owed: mission map and HUD at zoom 100, 120 and
83 percent, with/without dialogue; town, menus and film on/off; device scales 1, 1.25 and 1.5; editor frame;
Game Options, Ctrl+O and restart. Headless filter dispatch is not GPU readback or install validation.

## Open debt

- Towns, menus and mission dialogues combine art, baked labels and interface in one picture. Their off
  filter remains text-safe, so those composites do not promise hard pixels for every picture region.
- The editor keeps its per-sprite map zoom and smooth combined points frame, as owner-directed.
- The starter still presents its own fixed text frame through the library, as story 1653 records.
- The whole game package and fast gate remain seat-owned outside the sandbox; installed EN/RU witnesses
  and review remain owed.
