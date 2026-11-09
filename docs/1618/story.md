# The starter no longer links the game

## Intent

The starter is a small file. `cmd/starter` links neither `againrom/pkg/game`, `againrom/pkg/ui` nor
`againrom/pkg/sim`. Every starter screen, window, text and setting is unchanged.

## Authority

Owner, on the 94 MB release archive: «Делай эти шаги»; `pipeline/BACKLOG.md`, section Release archive
size, step 2. No DIV row: this story changes nothing a player sees apart from the file size.

## As-built behaviour

New package `pkg/ui/widgetkit` holds the widgets the starter shares with the game. New package
`pkg/installinfo` holds install inspection. Neither imports `pkg/game`, `pkg/ui` or `pkg/sim`.

| Kind | Builder (after) | How the game reaches it |
| --- | --- | --- |
| Flat panel frame painter | `widgetkit.DrawFlatFrame`, `widgetkit.DrawPanel` | `pkg/ui` `drawFrameBody` calls `DrawFlatFrame` for the flat kinds (panel, outline, tint, fill) |
| Check box and radio text marks | `widgetkit.CheckSpec.Mark` | type alias and constants in `pkg/ui/widgetkitnames.go` |
| Text input state and edit | `widgetkit.TextInputState`, `widgetkit.EditText` | type aliases; `pkg/ui` calls `widgetkit.EditText` |
| List wheel step | `widgetkit.WheelStep`, `widgetkit.ListWheelLines` | `scrollSpec.wheelStep` forwards to `WheelStep` |
| Window placement row | `widgetkit.WindowRow`, `widgetkit.ApplyWindow`, `widgetkit.ApplyWindowRow` | type aliases; `pkg/ui` calls the kit |
| Window icon | `widgetkit.SetWindowIcon` | `ui.SetWindowIcon` is a one-line forwarding function |
| Text clipboard | `widgetkit.TextClipboard`, `ReadTextClipboard`, `UseMemoryTextClipboard` | `pkg/ui` calls the kit |
| Install inspection | `installinfo.Inspect`, `installinfo.Detect`, `installinfo.Info`, `LanguageSelector`, `LanguagePath`, `MainPrefix` | `game.InstallInfo` is a type alias; `game.InspectInstall`, `game.DetectBase` and `game.LanguageSelector` are one-line forwarding functions; `game.LanguagePath` and `game`'s `mainPrefix` read the moved values |

Moved bodies are unchanged except for these seams.

- `archiveLanguage` calls `vfs.Open([]string{path}, nil)`, the one line `OpenContainers` was, because `pkg/installinfo` cannot import `pkg/game`. `LanguageSelector` takes `installinfo.EntrySource`, a `ReadFile` interface, in place of `terrain.EntrySource`.
- Clipboard members are exported for `pkg/ui`: `Clipboard`, `Read`, `Write`, `MemoryClipboard.Value`, `NewTextClipboard`.
- `TextInputState.Repeat` takes `reached func(deadline, now time.Time) bool`. The repeat clock's deadline
  test was `gateAtLeast.reached`, a `pkg/ui` cadence. `pkg/ui` passes `gateAtLeast.reached`.
- `TextInputState` members `pkg/ui` uses are exported: `Rules`, `Repeating`, `Reset`, `Selection`, `SelectAll`,
  `TypeText`, `Move`, `MoveTo`, `Replace`, `DeleteWordEdge`, `Erase`, `Selected`, `Drag`, `Repeat`.
- The frame builder stays in `pkg/ui`: tiled, art and scroll frames, the shadow plan and the widget record
  hook need `pkg/ui` types. Only its flat branch moved, as `DrawFlatFrame`; `frameSides` and its bits are
  aliases of `widgetkit.FrameSides`. `ui.DrawPanel` is gone; the starter called it alone. The record hook never
  fired in the starter, so `DrawPanel` carries none.
- The list model stays in `pkg/ui`. Only the notch arithmetic moved, as `WheelStep`. The launcher's
  list row, one line a notch and not whole, moved with `ListWheelLines` as `WheelStep(wheel, 1, false)`.
- `editorSingleLine` and `textInputKeys` stay in `pkg/ui/editortextinput.go`: they read `appInput` and the
  editor's paste, not the model.

## Proof

- `go list -deps ./cmd/starter` names none of `againrom/pkg/game`, `againrom/pkg/ui`, `againrom/pkg/sim`.
  `internal/archtest` `TestStarterLinksNoGamePackage` (`CheckStarterClosure`) walks the same closure
  statically over every platform's production files.
- `internal/archtest` guards move with the code: `textModelFile`, `checkSpecFile`, `windowApplyFile` and the
  editor word-class home name the new paths; the frame guard names `widgetpanel.go` as the flat painter's home
  beside `widgetframe.go`. `TestEditorWordClassHasOneBuilder` reads `pkg/ui` and `pkg/ui/widgetkit`.
- `pkg/ui` `TestKitPanelIsTheFrameBuilderPanel` compares `drawFrame(panelFrame(...))` with `widgetkit.DrawPanel`
  pixel for pixel over six rectangles, clipped and empty ones included.
- `pkg/ui/widgetkit` tests cover the marks, the panel, the wheel step, the text model and the clipboard;
  `pkg/installinfo` tests cover the language selector and the inspection of an empty directory.
- `starter.exe`, Windows amd64, built with `scripts/build-tools.sh`: 44 620 288 bytes at `39d6987a8`;
  14 062 592 bytes on this story.

## Open debt

- The flat frame painter is one function in `widgetkit` and the rest of the frame builder is in `pkg/ui`; the
  frame guard names two homes.
- The list model (`scrollSpec`, its rows and its key and bar steps) is in `pkg/ui`; only the wheel arithmetic
  and the launcher's row are in `widgetkit`.
- `pipeline/BACKLOG.md` step 3 (the map editor) is out of this story.
