# The mod window opens a mod, holds its captions and moves

## Intent

The map editor's mod window (W) opens an existing mod: a mod folder or a packed `mod-<id>.res`, read-only. Every
editor window built by the popup builder holds each button's caption inside its button in EN and RU. W's field
labels are whole, W moves by its title bar and resizes as the editor's other floating windows do, and every string
W shows comes from the word book.

## Authority

Owner report on the Russian build after the mod window's manifest story: W offers no way to open a mod; the button
row paints «Сохранить мод» and «Опубликовать» over their neighbours; the window does not move by its title bar and
resizes wrong. The seat's reading of the same frame added the cut labels («Папка публика…», «Требует (ID, ог…»)
and an English status line in the Russian window. Owner direction in the story brief: one builder per kind; the
editor's one path chooser; the mod opens through the route the editor already has (`SetMod`, `openModContext`,
`admittedFork`, `FrontEnd.SetModData`); no SAV, simulation, mod format or manifest key change. Mods and the editor
are Againrom surfaces: no ROM1 claim applies and no divergence row changes.

## As-built behaviour

Opening a mod. W's button row is the table `modButtons` (`pkg/ui/mapeditormod.go`): Open mod, Add, Remove, Save
mod, Publish, Close. Open mod asks the editor's path chooser (`askChoice` with a `pathChoice`, the chooser File >
Open uses) for one file, filtered to `mod.toml` and `.res`, starting in the folder that holds the mod in W's folder
field when that field holds an absolute path. The chosen path goes the folder field's own route:
`SetWorkspaceField("package", "folder", path)` reaches `openOrCreateWorkspace` (`pkg/game/mapeditorpackage.go`),
which now takes a chosen `mod.toml` as the folder that holds it. With unsaved changes that route returns
`WorkspaceLeave` and W asks the editor's Save / Discard / Cancel question before anything opens. Otherwise
`openWorkspace` calls `SetMod`, which reads the folder or the packed file through `mod.OpenTree`; `openModContext`
builds the mod's data through `admittedFork` and `FrontEnd.SetModData`. A packed mod opens read-only. W then shows
the opened mod's fields, manifest and overview. A host with no path window, or a path window that fails, focuses
the folder field and says to type the mod's folder or packed file there and press Enter.

Buttons. `popupButtonWidths` (`pkg/ui/editorpopup.go`) gives every button of every builder window the larger of
its least width (`ButtonW` or the default) and its caption's advance plus 16 pixels. The `ButtonFit` switch is gone:
this is the rule for all. A button row wider than the window wraps onto more lines in every window, not only in a
`Fit` one, and a window at its own size is at least as wide as its button row. W no longer sets `ButtonW`.

Labels. In a window of several columns each column has its own label column: two fifths of the column, widened to
the column's longest label up to half the column. A label longer than that stands on a line of its own over its
field. W keeps its base of 80 pixels and widens from there. The record window's field rows cut a label at 150
pixels before the builder saw it (`objectFieldRows`, `pkg/ui/mapeditorobject.go`); a window of several columns,
which today is W alone outside the native person window, now passes its labels whole to the builder.

With the word font a longer label of a popup of several columns takes its own line over its field (the character window shows it).

Moving and sizing. W takes its corner from `personCorner(homeMod)` and its width from `popupWide(popObject, …)`,
the placement store the editor's other floating windows read. A title drag moves W; the placement keeps the title
row and at least `popupKeepWide` pixels inside the editor window. A border drag sizes W as `resizeFloating` sizes
the others; a left or top drag keeps the opposite border. The place holds while W stays open.

Words. Every error the W route returns from `pkg/game` reads the word book: 49 `editor.mod.err.*` ids, plus
`editor.mod.openmod`, `editor.mod.openfilter` and `editor.mod.typepath`, each in EN and RU. W's default status line
now reads `plainWord`, so the host encoding runs once. A mod error carries the mod's id through `editor.mod.err.mod`
and keeps its cause as before: wrapped where it was wrapped, as text where it was text. A range error now leads with
the field's label instead of its key.

Other windows. The one visible change outside W, found by a census over every modes path, host, key and gesture at
1024x768 and 1280x960 with the installed EN and RU fonts: the Russian Unit, Structure, Sack and Player record
windows' «Применить» button grows from 96 to 106 pixels. The Scripts list buttons already fit their captions.

Unchanged. No SAV byte, simulation state, mod file format or manifest key changes. Saving an opened mod without an
edit writes every file as it was.

## Proof

- `pkg/game/mapeditormodopenbutton_test.go`: `TestModWindowOpenButtonOpensAModAndASaveWithoutEditKeepsItsBytes`
  writes a mod into `t.TempDir()`, opens it with Open mod, checks the request, the opened folder and W's manifest
  lines, saves without an edit and compares every file's bytes; `TestModWindowOpenButtonOpensAPackedModReadOnly`
  opens a packed `mod-<id>.res` read-only and leaves its bytes as they were.
- `pkg/ui/mapeditormodopen_test.go`: the Open mod button by key and by click sends the chosen path down the folder
  route, and with unsaved changes asks the shared question first; with no path window, or a failing one, the folder
  field takes focus with the worded line; `TestEditorPopupButtonsHoldTheirCaptionsInBothLanguages` walks every
  modes path in EN and RU with a ten-pixel word font (W also at 640x480) and finds no caption past its button, no
  button outside its window and no overlap; it reports 118 faults when the caption widening is switched off; every W
  field label of the game's set is drawn whole in EN and RU at 1024x768 and 640x480, and the test reports 14
  faults with the 150-pixel cut put back; a title drag moves W, the place holds through keys and a
  layout, a drag past the corner keeps it inside, and a left border drag keeps the right edge; every
  `editor.mod.*` word and the Cancel word differ between EN and RU, with no exception.
- The modes table (`pkg/ui/testdata/mapeditor_modes.golden`): the 12 `workspace/mod-*` rows moved, from the Open
  mod button, the caption-wide buttons and the wider label column; no other row moved. A new row, `mod-open`,
  presses Open mod with no path window.
- Owner frames from synthetic content, drawn headless with the installed fonts: W in EN and RU after opening a
  mod, and moved. They stay in the review notes; no frame is committed.
- Release witnesses that draw an editor window, for the release, in EN and RU: `TestReleaseMapEditorOwnerFrames`
  and `TestReleaseEditorInteractionFrames` (both draw W), `TestReleaseMapEditorAddWindowFrames`,
  `TestReleaseMapEditorBrushFrames`, `TestReleaseMapEditorInstalledTextIsCapturedAtScaleOne`, and the record
  window witnesses `TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes`,
  `TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes`, `TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes`
  and `TestReleaseMapEditorGroupPopupFollowsUndoAndRedo`. None ran in this lane.

## Open debt

- The workbench folder and a copy-to-workbench button for a packed mod do not exist in the code; the packed mod
  opens read-only only.
- The mission list in W clips a long mission name with an ellipsis; that is a value, not a field label.
- Detail a W error carries from `pkg/mod`, `modrt` or the operating system stays in English.
- The page-field errors of the mission pages window and the staging errors shared with test play stay in English.
