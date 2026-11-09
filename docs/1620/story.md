# Story 1620: one description for the modal dialogs

## Intent

Every modal dialog the engine shows should be one value of one description type, laid out by one layout and painted by one painter, with no frame, hit area, text, key, press behaviour, SAV byte, RNG step or World hash changing. This is a cleanup kind (R29): a player sees nothing.

The dialog population named in the story's briefs is done: the description type, the one layout and the one painter exist, and the notice windows, the gold editor, the game menu, the save page's frames and buttons, the four authored notices, the help panel and town talk are values of it. Seven further windows that draw a frame and a button themselves were never in that population; see Open debt.

## Authority

Owner direction recorded in the story's lane brief (`review/story1620-dialog-description/BRIEF.md`, section Item, and `BRIEF-2.md`). The briefs keep these builders: `lineRule.lines`, `scrollSpec`, the popup placement guarded by `internal/archtest/popupplacement.go`, `pushButton`, `buttonLatch`. ROM1 claims stay cited where they were; no claim changed.

## As-built behaviour

`popupSpec` (`pkg/ui/editorpopup.go`) is the description, chosen as the base because it has the most call sites and the 45 editor files stay as they are. `Dialog` (`pkg/ui/dialog.go`) is an exported alias of it for `pkg/game`. `NoticeLayout`, `noticeWindow` and `NoticeShape` are gone; the substitution point `Viewer.SetNoticeLayouts` and `SetSuccessNoticeLayout` takes values of `Dialog`.

`layoutPopup` is the layout. A value's geometry comes from its rows, as before, or from given rectangles: `Box` is a composed picture at the origin with children relative to the box, `Panel` is a notice window over a page rectangle (`dialogGivenForm`, `layoutDialogGiven`). For a `Box` value with no given `Frames`, `layoutNoticeBox` builds the frame (art when it is valid, the value's colours when it is not), the pushes from `Button` and `SecondaryButton`, the body lines from `Body` and the scroll bar from `Scrollbar`.

`drawPopup` is the painter. It reads the frames by value (`Frames`: nil is the rows form's ink panel, empty is no frame), then the portrait pane, then the edit field, the push buttons, the scroll bar, the lines and the answers, the same way for every value (`drawDialogParts`). The title band, grip and close control are drawn only for a layout that has a close rectangle. `RenderNotice` sets `Body` and `Face`, and calls layout and painter into a box-size image. `ComposeDialogueNotice` places that picture over a scene with the frame's shadows.

A notice style is a row of data: `noticeStyles`, indexed by `Style`, gives the line rule, its trim, whether body lines justify by paragraph, the indent, the text shadow, the button ramp, the band kept for the frame's shadows, the default ink, whether the frame's shadows fall on the scene, whether the second button is drawn, whether a disabled first button refuses a press, and the portrait pane's procedure. The layout, the painter, `noticeBodyLines`, `ComposeDialogueNotice`, `paintNotice` and `noticeButtonIDAt` read the row; `grep -n 'Style == NoticeStyle' pkg/ui/*.go` finds no production branch.

The portrait pane keeps two pixel procedures, `drawPlainPortrait` and `drawDialoguePortrait`, selected by the style row: the second composes a keyed 16-bit surface with the dialogue backdrop's pixel format, the first composites over RGBA, and the golden holds both.

Each instance and how it became data:

| Instance | Before | Now |
|---|---|---|
| Notice windows (list, question, scrolled list, save question, mod mismatch) | `noticeWindow`, `NoticeShape`, `noticeWindowLayout` and their compose | `Panel` value. Run state is `dialogRun` (`pkg/ui/widgetnotice.go`). Authored rectangles are `ShapeSize`, `AnswerSize`, `TextTop`, `TextWidth`, `LinePitch`, `AnswerGap`, `AnswerBottom`; the wrapped text is `Runs` through `lineRule.lines`; the answers are `Answers` and the press latch is `Latch`; the word mapping is `Display` |
| Gold editor (Drop Gold) | Private frame, title, edit field and buttons in `goldModalPresent` | `Box` value: `Frames`, `Edit`, `Lines`, `Pushes` |
| Game menu | `composeGameMenuPanel` drew frame, buttons and text rows | `Box` value: `Frames`, `Pushes`, `Lines` with `Clip` and `Dim` for a disabled row. The function keeps only the row composition |
| Save page frames and push buttons | `drawFrame` and `drawPushButton` calls | `Box` values through `drawDialog`. The page's captions, edit fields, list and deferred labels stay on the page, which is not a dialog |
| Four authored notices | `NoticeLayout`, `RenderNotice`, `renderDialogueNotice`, `drawNoticeWindow`, `dialogueButton` | `Box` values (`AuthoredDialogueLayout`, `AuthoredOutcomeLayout`, `AuthoredSuccessLayout`, `AuthoredFailureLayout`); fields moved under their old names; `Style` picks the row |
| Help panel | `HelpPanel`, `helpApply`, `drawHelpScrollbar` | `Box` value resolved by `helpApply`; `Scrollbar`, `FirstLine`, `ScrollbarTopHot`, `ScrollbarBottomHot` become the layout's bar. `drawHelpScrollbar` is gone |
| Town talk | `TownTalk.Layout` | The same `Box` value from `AuthoredDialogueLayout` |

The painter's order moved the push buttons before the lines and the scroll bar before the lines, for every value. The help panel's text can touch the scroll bar's first column, so the bar must stand under it, and the notice painters drew their buttons before the words; the witness holds it.

## Proof

`pkg/ui/dialogwitness_test.go` with `pkg/ui/testdata/dialog_witness.golden` was written and committed on the base commit, before any move. It holds 138 entries at 640x480, 1024x768 and 1280x800, in English and Russian text: rectangles, line count and line hash, and sha256 prefixes of the pixels, with no game byte. Its cases are every notice kind, placed dialogue, outcome and help notices, the town talk portrait, the notice windows (list, scrolled and pressed list, question, clamped question, mismatch), the gold editor, the game menu, the save page and an editor popup.

`TestDialogWitnessFailsAMovedPixelAndAMovedRectangle` plants a moved pixel and a moved rectangle and requires the witness to fail. The golden file has not changed since that first commit, and the witness passed after each move. The fonts in the witness are synthetic, so it says nothing about an installed game; it compares before and after.

Existing tests changed only in mechanics: `reflect.DeepEqual` replaces `==` on two descriptions in `pkg/ui/notice_test.go`, `pkg/ui/words_test.go` and `pkg/game/savepages_test.go`, because the description holds slices; the assertions are the same. No test, assertion or input left.

Census of `grep -E '^type (NoticeLayout|noticeWindow|NoticeShape|popupSpec) ' pkg/ui/*.go`: 4 types in 3 files before, 1 type in 1 file after (`popupSpec` in `editorpopup.go`).

The guard is `internal/archtest/dialogdescription.go`: no retired type or painter, no other struct holding both `Box` and `Frame *DialogFrame`, no comparison against a `NoticeStyle` constant, no function outside `drawPopup` that draws a frame and pushes a button unless named in its falling-only list. `internal/archtest/popupplacement.go` reads the new fields without a hit (`TestEditorPopupPlacementIsOneStore`).

## Open debt

The guard's list names what still draws a frame and pushes a button itself. Two are parts of the one painter or a widget: `drawPopupGrid` and `drawDropdown`. Seven windows were outside the dialog population of the briefs and were not tried: `chromeCompose` (map editor chrome), `composePage` (the load page), `gameOptionsPicture`, `soundOptionsPicture`, `composeCutsceneLibrary`, `questPicture` and `ComposeTipPanel`. Whether each is a dialog of this kind is an owner decision; none failed a witness.

No claim or divergence changed. No CHANGELOG line: no player sees a change.
