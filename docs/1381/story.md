# Widget kit round 2: one frame builder, one press latch, one list

## Intent

Every frame, every push-button press and every list on the ROM1 screens has
one builder. Before this story five painters drew windows, tips, panels,
outlines and the portrait border, eleven screens kept their own press latch,
and only Save and Load drew the list well. Base: `dfb84f0f`.

Owner direction: one builder per kind. Frames drawn where MENU-127 shows the
original frame are drawn as the original draws them. The pixel changes are
these and no others.

## Authority

| Part | Claims |
|---|---|
| window frame: lm.256 frames 0..8, corners, whole edge tiles, whole fill tiles | MENU-127, DLG-PANEL-035, MENU-ART-014 |
| mask shadows of pieces 3, 5, 6, 7, 8 at (+8,+8), level 6, before the body | DLG-PANEL-035, MENU-ART-014 |
| snap rule of the dialog base | MENU-077 |
| tip frame: lm.256 frames 9..17 | MENU-127 |
| press latch: set only when none is held, release inside activates | MENU-116, DIALOGUE-045 |
| Save/Load, Sound Options and the cutscene chooser share one list class | MENU-121 |

## As built

### Frame builder

`pkg/ui/widgetframe.go` holds the frame table. A frame is data: a kind, a
rectangle, the snap rule, and its art or colours. `drawFrame` draws the mask
shadows, then the body.

| kind | row | draws |
|---|---|---|
| `frameWindow` | tiled, cover, shadow 8 | lm.256 frames 0..8; a remainder gets one clipped edge and fill tile (DIV-2724) |
| `frameTip` | tiled, shadow 8 | lm.256 frames 9..17 |
| `framePanel` | fill, border | the engine's flat panel (DIV-2722) |
| `frameOutline` | border | the panel's border alone |
| `framePortrait` | corner 22x27 | the notice portrait border (DIV-2723) |
| `frameTint` | fill over, border | the choice box drawn without sprite art (DIV-2722) |

Sites drawn through the builder:

- Window, snapped: Game Options, Sound Options.
- Window, unsnapped: Load, Save, the cutscene library, quest objectives, the
  in-game menu, mod screens, notices, the dialogue window. Of these, the
  pictures that change are the census list below plus the outcome notice and
  the mod screens, which the census does not hold; the dialogue window and
  dialogue-style notices are whole tiles and do not change.
- Tip: the room tip panel.
- Panel: notices and dialogue fallbacks, inventory bars, the Drop Gold
  editor, the minimap box, the spellbook, item and unit panels, the
  character generation navigation box, the Save dialog boxes, the ending
  pages.
- Outline: the spellbook cell.
- Portrait: the notice portrait.
- Tint: the choice box without sprite art.

Retired painters: `DialogFrame.Draw`, `drawMenuPanel`, `drawSnappedDialog`,
`drawNinePatchBorder`, `fillPanelFrame`, `drawBorder` (spellbook and
character generation), `drawChargenFrame`, `savePaint.box`, `drawMovieBox`,
and the dialogue tile and shadow helpers. DIV-2590 is closed.

### Press latch

`pkg/render/latch` is the one latch; `buttonLatch` in `pkg/ui` is that type.
A press latches only when no latch is held, a release inside the latched
button activates it, and a release elsewhere clears it. A lost or closed
window clears every latch, and so does leaving the screen or in-game menu
page the press was made on (`dropWidgetLatches`, called from
`validateWidgetLatches` and after each step that changes the screen or menu
page). In the in-game menu a release in the same tick as a key activates
nothing and drops the latch.

Former screen-local latches now on the kit latch: the main menu selection
(`pkg/render/menu.Selection`), quest objectives close, the mod main-menu
entries, the mod screen Back, the Save dialog controls and rows, the Drop
Gold editor, the documents panel, the ending page buttons, Sound Options,
character generation (pre-create and detailed pages) and the dialogue
advance button. The notice, in-game menu, Game Options, Load and cutscene
library buttons were already on the kit latch.

The buttons drawn with `buttonInk` (character generation navigation, shop,
town surface) are installed bitmap plaques with a caption; the original draws
its own art there, not the kit push button, so they keep their art.

### Lists

`drawListBox` draws the list well, the edit field's sunken bevel one pixel
around list and bar, for every list (DIV-2645). Before, Save and Load drew
the well themselves before calling `drawListBox`, and the cutscene library
and Sound Options did not. The lists now differ in nothing but their data.

## Proof

- Release census `TestReleaseFrameAndButtonCensus`
  (`pkg/game/framekit_census_release_test.go`): 31 pictures per root, hashed
  with SHA-256, before (`dfb84f0f`) and after on EN and RU. 12 pictures
  change per root, each inside its frame or list rectangle: the cutscene
  library (2), the in-game menu (2), Game Options, Load (3), the success
  notice, quest objectives, Save, Sound Options. 19 hash identically:
  rooms, room tips, the shop dialogue, character generation, mission panels
  and the main menu. The latch and ratchet commits change no hash.
- `TestEveryFormerLocalLatchRunsTheKitCases` runs 13 latch sites through
  press and release inside, a release elsewhere, a lost focus and a double
  press. Mutating the latch to ignore the release point, to keep a press over
  a lost focus, or to let a second press replace the first fails 4, 9 and 5
  sites respectively.
- The witness's fifth case presses, leaves the screen by a key, releases
  elsewhere, returns and clicks once; the click activates on 10 sites. The
  dialogue (keys advance it and cancel the press) and the ending (no key
  returns to the page) skip it. `TestMainMenuPressDoesNotOutliveTheMenu`,
  `TestGameMenuPressDoesNotOutliveTheMenu` and
  `TestGameMenuReleaseWithAKeyDropsTheLatch` fail without the drop.
- `TestFramesAndLatchesHaveOneBuilder` (`internal/archtest`) refuses a frame
  painter or press latch outside the kit that its lists do not name, and a
  list entry nothing matches.
- `TestWindowFrameShadowFallsOnATransparentPanel`, the frame tiling tests and
  `pkg/render/latch` unit tests.

## Open debt

- Frame painters outside the kit, in files the town pages own:
  `drawTownShellBox`, `drawTownShellOutline`, `drawCharacterPaneBody`,
  `ComposeTownSurface` (control outlines), `drawShopHover`,
  `drawShopChevron`, `drawWorldMapBorder` and the `outline` primitive they
  share. Listed in `internal/archtest/widgetkit.go`; the list only falls.
- Press latches outside the kit: `townSurfacePress`, `shopPress`,
  `townTipPress` and its town-room copy `tipPress`. Same list.
- DIV-2724 and DIV-2725 (window cover and shadow tone) wait for a claim or an
  owner ruling.
