# The editor's speaker choice shows each speaker's small portrait

## Intent

A map maker chose a dialogue page's speaker from names alone. The speaker list now shows each
speaker's small portrait, the page card shows the chosen speaker's portrait, and the person
window's NPC list shows the same portraits.

## Authority

- The tester's list (`review/tester-editor-wishes/doc.txt`, item 13): the speaker of a dialogue
  page is chosen blind; the tester asks for small portraits.
- Owner: a story of its own after the editor's menu and button layout (hotfix
  editor-tester-layout, 0.141.22).
- The map editor is this project's own program. No claim is cited and no DIV row is added.

## As built

### Kinds and builders

| kind | builder |
|---|---|
| npc choice list | `npcChoices` (`pkg/game/mapinspectionnpc.go`), unchanged; order `speakerRank` unchanged |
| npc face picture | `speakerResolver.SpeakerFace` (`pkg/game/speakers.go`) over `speakerFaces(table)` (`pkg/game/gamefiles.go`), the records `InstallResources.NPCFaces` holds; cut by the dialogue pane's own `blitNoticeWindow` at `AuthoredDialogueLayout().WithFaceWindow(window)` |
| choice picture | `InspectionField.Icons`, choice value to picture, the slot the unit look list already uses |
| editor choice list rows | `choiceRows` (`pkg/ui/mapeditorobject.go`), the object popup's one drop-down row builder, drawn by `drawListBox` (`pkg/ui/widgetscroll.go`) through `PickerRow.Icon` |
| editor field-choice producer | `MessageFields` (`pkg/game/mapmessagepages.go`) and the unit field producer's `unitFieldValue` (`pkg/game/mapinspectionlook.go`) set `Icons` on the field they already return |
| popup card row | `messageSpec` (`pkg/ui/mapeditormessage.go`) through `popupRow.Icons` and `drawRowIcons` |

### Behaviour

- `MapInspector.npcPortraits` maps each choice value to the face a dialogue shows when that NPC
  speaks. In the editor no live actor stands, so SpeakerFace takes its synthesised arms: a
  person record composes its figure bare, a creature record takes its class portrait, a record
  with no picture and a hero record answer none. Each face is composed once per inspector
  (`recordPictures["speaker"]`).
- `ui.SpeakerPortrait` cuts the record's window, or the pane's default window, and centres the
  cut on a transparent 96x96 square, so the list's square cell and the card's fitted cell draw it
  unstretched.
- The speaker field of `MessageFields` and the person window's `npc` field carry `Icons`. The
  person window and the speaker list read one cache, so the two lists carry one picture per
  record.
- `choiceRows` gives each row its picture; when any choice of the field has one, every row keeps
  the picture cell (`PickerRow.IconCell`), so a speaker without a face and "No speaker" show an
  empty cell and the names line up. The unit look list takes the same rule: its first row
  (inherit) now keeps an empty cell.
- The page card's speaker field draws the chosen speaker's portrait after the name.
- The list's order, its labels, the values written to the page and the MCP results are
  unchanged. `plainFields` already drops `Icons` from a record read; the message read lists
  choices only.

### Install measurements

A temporary probe (not committed) opened `gameversions/en` and `gameversions/ru` with
`NewMapInspector`: 105 speaker choices, 87 with a face, none cut blank, first composition 21 ms
(EN) and 24 ms (RU). The 18 without a face are records 1 to 15 and 22 to 24.

## Proof

- `TestMessageSpeakerListCarriesEachSpeakersFace` (`pkg/game`): speakers 51 (person) and 103
  (creature) carry the picture an independent resolver over the same records and pictures
  answers, cut as the pane cuts it; 103's pixels are its class picture at the default window;
  "No speaker", records 3, 21, 22 and the unknown 999 carry none; the order
  `-1, 21, 22, 3, 51, 103, 999`, the labels and the value equal the same list without faces;
  speaker 51 writes `<part=1,npc=51>`. On the base tree (`git archive f547ab60`, the candidate's
  `SpeakerPortrait` copied in so it compiles) the picture assertions fail and the order and
  markup assertions pass.
- `TestPersonWindowNPCListCarriesTheSpeakerListsFaces` (`pkg/game`): the person window's NPC 8
  carries the game's dialogue face, the same picture the speaker list carries; records without a
  face carry none. Fails on the base tree.
- `TestMessageCardShowsTheChosenSpeakersPortrait` (`pkg/ui`): the card of speaker 5 carries and
  draws the portrait in its speaker field; the card of speaker 6 draws none.
- `TestSpeakerListDrawsEachSpeakersPortrait` (`pkg/ui`) and its new golden
  `pkg/ui/testdata/mapeditor_speaker_portraits.golden`: the open list's rows, "No speaker" first,
  the portrait only on speaker 5, every row with its cell, the portrait drawn in the list, and the
  frame's hash.
- `TestSpeakerPortraitIsThePanesCut` (`pkg/ui`): the default and a stated window, the transparent
  margins, nil for no face.
- Existing: `TestMessageSpeakerListStartsWithTheFourLeadingSpeakers`,
  `TestMapEditorModesHoldEveryKeyAndGesture` and the rest of `pkg/ui` pass unchanged; no golden
  is re-recorded.

## Open debt

- Release witnesses for the next release, EN and RU (`pkg/game`): `TestReleaseTriggerMessageFrames`,
  `TestReleaseTriggerMessageEditsAndUndo`, `TestReleaseMessagePickerParts`,
  `TestReleaseMessagePickerPopulation` (message editor and speaker list) and
  `TestReleaseMapEditorOverrideFrames` (person window). No release test was run in the lane.
- The person window's large preview (`mapinspectionunitsettings.go`) reads `catalog.Faces`, not
  `speakerFaces(table)`; on a second-game install the two can differ. Unchanged here.
- `NewMapInspector` fails on `gameversions/rom2-en` (`data.bin` Armors entry 4), so the
  second-game speaker portraits are not observed on an install.
- A list row is the body font's height; the portrait is drawn at that height.
