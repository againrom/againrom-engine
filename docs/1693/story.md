# Trigger windows that fit their contents

## Result

The trigger's wizard and details size their windows to the fields and buttons. Long triggers scroll
between a fixed title and footer, with warnings under the scrolling area. Help marks explain the
trigger, its choices and its controls. A clipped choice shows its full text on hover.

## Authority

- Owner: «T3 и T5 делаем». The editor window migration and the three T5 answers govern the layout.

## Rules

1. Both trigger views size their rows and scroll a long body between the title and the footer in EN and RU.
2. Adding a condition or an action uses an unlabelled text row with buttons and a help mark.
3. Missing-condition and multiplayer warnings appear below the scrolling body.
4. Row buttons with text are at least 96 pixels wide.
5. A clipped trigger choice ends in a cut mark and its shared hover tip shows the full text.
6. The wizard explains preserved records behind help marks.
7. Applying the same trigger edit writes the same map bytes.

## Code

| kind | its one builder | change |
|---|---|---|
| trigger details | `scriptSpec` and `screens/triggerdetails.json` | bind the existing pairs and actions to measured rows |
| trigger wizard | `wizardSpec` and `screens/triggerwizard.json` | bind help and warnings to the current wizard state |
| window layout | `layoutPopup`, `naturalWidth`, `pageObjectRows` | measure and page trigger windows through the shared framework |
| row and footer button widths | `rowButtonWidths`, `popupCaptionWidth` | apply the common minimum to text buttons |
| hover tip | `popupTipAt` and `layoutTip` | wrap a clipped choice's full text in the shared tip |
| words | `pkg/words/text` | English and Russian trigger help |

## Proof

| test | rules |
|---|---|
| `TestTriggerWindowsHoldTheLayoutProperties` | 1, 2 |
| `TestTriggerAddRowsKeepOnlyButtonsAndHelp` | 2 |
| `TestTriggerWarningsStayBelowThePagedBody` | 3 |
| `TestTriggerTextRowButtonsUseFooterMinimum` | 4 |
| `TestTriggerWizardClippedChoiceShowsItsWholeValue` | 5 |
| `TestTriggerWizardExplainsKeptRecordsOnTheirRow` | 6 |
| `TestTriggerPopupApplyWritesTheFieldCallBytes`, `TestTriggerWritersKeepTheirBytes` | 7 |

Release tests at the release in EN and RU: `TestReleaseMapEditorTriggerFrames` and `TestReleaseTriggerWizardFrames`.

## Open debt

- none
