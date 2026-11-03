# One described save screen

## Intent

Unify audit-20 R23 (save screen) and R48 (ScreenSave dispatch). Preserve each
game's layout, input, save request and destination. Preserve the SAV producer.

## Authority

Owner direction requires one builder per kind, with the first game expressed
as profile data in the second game's page shape. This is architecture work
with no player change or release count. Base: `63ac65e4a` (engine main the branch merged),
knowledge k406. No new behaviour claim or divergence is introduced.

## As-built behaviour

`SavePage` describes both save screens. `firstSavePage` supplies the existing
file browser: directory and name fields, seven rows, navigation, focus order,
engine words, exact-target overwrite question and existing button rectangles.
The second game's `secondSavePages` supplies its existing description: labels,
Empty Slot, hidden files, slot allocation, name limit and delete question.
It still overwrites a chosen slot without asking.

`flow.saveScreen` holds the sole `savePageState`. `openSaveDialog` builds it
once from the selected description. `stepSaveScreen` drives it;
`savePagePaint` paints it. Composition, drawing, Escape, headless operations,
cheat routing, pointer clearing, menu hosting and tooltips use that holder.
No caller chooses between save screen types. Game identity does not select
an input or paint implementation; the first game's differences are the
unexported flags of its `SavePage` description (`firstSavePage`), which is
the first game's profile data. LOAD and SAVE share one painter for the page's
in-page question, message line, captions and list, and one input helper for
the list keys, scroll bar and row press.

`pageView.confirmingInPage` includes the page's pending overwrite or deletion.
The same page draws the question, with its described contents and controls;
windowed questions retain `PageConfirm` and the existing notice window.
Editing uses `EditText`. Lists, fields, buttons and hit rectangles retain the
widget kit and `controlLayout`. LOAD keeps its handler and composition, with
save-role adapters pointing to the shared save handler. SAVE still calls
the same `Prepare` and `Commit` seams; no SAV writer changes.

## Proof

Record commit `ac465cba04503a43abd0cd372afd321bdc319367` precedes every production
change. `TestSaveScreenRecords` pins 53 records: synthetic-font pixel hashes,
glyph calls, rows, focus, headless state, requests, overwrite flags and the
following screen. It covers empty and populated lists, selection, editing,
controls, overwrite Yes/No, deletion and Escape. Both games retain their
records without regenerating the fixture. A temporary one-pixel NameBox move
failed 25 records; reverting it restored equality.

Production census, excluding tests:

| Pattern | Base lines/files | Result lines/files |
|---|---:|---:|
| `^type saveDialog \|^func \(d \*saveDialog\) ` | 7/1 | 0/0 |
| `f\.saveDialog != nil\|pages\.save != nil` | 11/7 | 0/0 |
| `saveDialog\b` | 36/8 | 0/0 |

`CheckSaveScreen` rejects the retired identifier and a second state holder in
flow or its pages, including aliases, nested records, slices, map keys and
values, channels and renamed state shapes. Twelve planted-shape cases name
their violations. A positive description-only case and empty-source rejection
complete its self-tests. There are no remaining guard entries.

The focused UI expression `Save|SavePage|Load|Dialog|Cheat|Tooltip|MissionMenu|ControlLayout|Headless|Flow`
passes with release tests excluded. Additional focused tests cover the three
`TestSecondGameDelete*` window cases, `TestGameFieldsUseSharedTextModel`,
`TestWindowLastButtonLeavesByTheGamesProfile`,
`TestActivePauseTextAndOutsideMissionZero` and `TestScrollListAnswers`.
Existing assertions and recorded layout hashes remain unchanged; references
to removed names use the shared state. The added tests
`TestSavePageDeleteWithoutCommitNeverSaves` and
`TestSaveScreenDescriptionCanAskBeforeASlotOverwrite` check a refused delete
and an independently combined description.

Focused game tests matching `Save|SavePoint|Snapshot` pass with release tests
and the sandbox-denied cases below excluded. The save-screen guard, DAG,
control-layout guard, architecture `TestLiveTreeClean`, storyguard
`TestLiveTreeClean`, `go build ./...` and `go vet ./...` pass. CommentBytes is
10481859: 10484142 on the base before the story, 10481308 after the merge, and
+551 for the shared helpers' doc comments in the correction pass. Fast gate is deferred to the seat outside the sandbox;
its standard selection would repeat denied tests. No release test ran here.

The next release must run the full release set on both games' EN/RU roots.
Pixel and route witnesses include `TestReleaseSaveDialogUsesInstalledFontWithoutClipping`,
`TestReleaseSaveCaretStaysWhiteAtHome`, `TestReleaseSaveDialog1173MissionCity`,
`TestReleaseSaveDialog1173TownPairOverwriteAndRefusal`,
`TestReleaseSecondGameSavePageWritesSlotsAndLoadPageLoadsThem`,
`TestReleaseSecondGameSavePageOverwritesTheChosenOfTwoEqualLabels`,
`TestReleaseWidgetKitScreens`, `TestReleaseSharedDialogFrames1185`,
`TestReleaseFrameAndButtonCensus` and both profile witness tests.

## Open debt

No residual second save-screen builder or guard exception remains. Installed
pixels, original-runtime reload and installed SAV byte streams were not tested
in this headless lane. Ordinary records prove the fixture inputs and requests.

The seat must rerun the fast gate and these sandbox-denied tests:
`TestEditorPartyRowsPreserveTemplateAndEmptyOnSaveAndReopen`,
`TestEditorPublishUsesThePackerAndRequiresSaveAndReplaceSeparately`,
`TestSaveModPackageInsideTheOpenedMapFolderAndTheSelectedModFolder` (both subtests),
`TestMapEditorWorkspaceCreatesSavesAndReopensBothRoutes`,
`TestSaveIntoModOfAMapWithAnErrorAsksThenWrites`,
`TestSaveIntoAnOwnedPackageWithAnErrorAsksThenWritesTheMapFile` and
`TestEditorSavesIntoTheModMissionPath`. Their base run failed only with
`Access is denied`; none was retried or bypassed here. The seat reconciles,
pushes the candidate and obtains the single adversarial review.
