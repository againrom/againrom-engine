# Story 1623: one headless picture request, one window size parser, one control finder

## Intent

A headless caller asks the App for a picture with one request value and names the part as data. A tool reads a window size through one parser. A tool or test finds a map editor control by role or caption through one finder. Kind R31, row 23 of the architecture audit `pipeline/reviews/arch-audit-5d474c65.md`.

## Authority

Owner, on the architecture audit: "Берем те 7 видов, пытаемся достигнуть 13". Instrument code; no player-visible behaviour changes.

## As-built behaviour

- `ui.App.HeadlessPicture(HeadlessRequest) (*image.RGBA, HeadlessExtras, error)` is the one headless picture entry point. Each part is one row of `headlessPartRows` (`pkg/ui/headlesspicture.go`); extras (`Note`, `Statistics`, `At`, `Lines`, `Cells`, `Tooltip`) come back in one `HeadlessExtras`. The 16 picture methods are gone and every call moved.
- `routeWitness.frame` (`pkg/game/routewitness.go`) is rows of request parts, `routeFrameRows`; `routeWitness` keeps its own walk.
- `ui.ParseWindowSize` reads `WxH`, each side 1..8192, for the route witness test and `cmd/screenshot`. The screenshot tool now refuses a side over 8192 and refuses text after the height; those are the only behaviour changes, both in a development tool.
- `MapEditor.FindControl(ControlQuery)` is the one lookup of an editor control by role or caption, over `editorControlRows` (`pkg/ui/controlfinder.go`). Thirty named accessors moved: `PopupButtonRect`, `FieldRect`, `RowButtonRect`, `PromptButtonRect`, `BarRect`, `DropdownRowRect`, `ToolbarRect`, `PopupRowRect`, `PlaceGridCellRect`, `ListButtonRect`, `TitleRect`, `ListButtonKeyRect`, `GroupsRowRect`, `ListRowRect`, `ObjectSectionRect`, `ViewSwitchRect`, `PanelButtonRect`, `MenuRowRect`, `GoalOpenRect`, `AreaLabelRect`, `SelectionButtonRect`, `TreeRowRect`, `GroupsButtonRect`, `GroupMemberRect`, `PromptListRowRect`, `PanelRowRect`, `AreaTriggerRect`, `GroupDeleteRect`, `PlaceGridSwitchRect`, `FieldChoiceRect`.
- A caption query resolves the button number in the prompt or the object popup's button row; other roles have no caption rows.
- `internal/archtest` states one picture entry point, one parser, and a falling count of exported `MapEditor` rectangle accessors (5).

## Proof

`pkg/ui/testdata/headless_picture.golden` holds, as numbers only, every picture part in 40 states on a synthetic App: hash, bounds, extras and the error text hash. It was recorded at base `d94d61412` with the old accessors and passes unchanged after the move (`TestHeadlessPictureWitness`). `TestHeadlessPictureWitnessFailsOnAPlantedFault` shows a changed pixel, note, position, line, tooltip state or error fails it. Census at the base and after: `review/story1623-headless-request/census-base.md`, RESULT.

## Open debt

- `Viewer.HeadlessMapFrame` stays on the Viewer: it needs no App and its release callers hold a viewer. The route's last row calls it.
- `App.GameMenuPanel` stays a method: production calls it (`missionmenu.go`, `questobjectives.go`, `headless.go`). The route witness asks for it through `HeadlessPartGameMenuPanel`; 31 other test and witness call sites still call it directly.
- `MapEditor.HeadlessFrame` is the editor's own frame and stays.
- `ui.TownSurfaceControlAt` is a hit test by point. The town surface has no rows of control rectangles by role, so a finder over it is out of scope.
- `controlLayout` ids are private per screen and no tool finds them by role today.
- `MapEditor.CardRects` returns two rectangles (a card and its grip); `PopupRect` and `PromptRect` are surfaces; `PopupTipRects` is a list of hover boxes; `MapRect` is the map area. None is one control; each is counted by the falling ratchet.
- `CellPoint`, `AreaCornerPoint`, `HeightLabels` are map queries and stay.
- `pkg/game/mapinspectionnew.go` reads a new-map size key, not a window size.
- The witness's HeadlessMapFrame row is the error path only: the fixture viewer has no tileset.
