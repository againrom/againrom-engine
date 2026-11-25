# The second-game map editor reads and edits a node's label and its ten texts

## Result

In the second game's trigger window an action or a check now shows its ten text fields under its label. Each
is marked as text no game reader was found for. The map maker edits the label and every text, also on a node
whose operation is not decoded, and a save writes each into its own bytes. Untouched maps save byte for byte.
Pages, messages, dialogues and variable names of a second-game map are still not offered.

## Authority

- Owner: «Редактор ром2 давай доделаем»; group G6 of the editor gap census.
- R2-ENGINE-1681: the loader puts the 64-byte name at +0x00 and ten 64-byte text fields at +0x9c; no reader
  of the text fields is found (Medium); the name reaches two compile-failure messages (Medium).
- R2-ASSET-364: every installed node has a nonempty name and ten nonempty texts (at most 45 and 23 bytes);
  most carry bytes after the first terminator.
- R2-ENGINE-1682, R2-ENGINE-037, R2-ENGINE-344: the mission text is one file of section tags in the shared text
  buffer, and which file that buffer holds at a dialogue is Unknown.

## Rules

1. When a second-game node is shown, its ten text fields follow its parameters, numbered 1 to 10, as text.
2. When a node's operation is not decoded, its label and texts stay editable and its other fields shown only.
3. When a text or the label is applied, it is written into its own 64-byte field, filled with zeros.
4. When a text is applied, every other byte of the map stays; a parameter edit of an undecoded node is refused.
5. When a first-game node is shown, it lists no text field.

## Code

| kind | its one builder | change |
|---|---|---|
| node record field table | `pkg/mapedit/script.go` scriptSection, nodeTextFields | a format's text fields as data |
| second-format sections | `pkg/mapedit/second.go` init | node sections take the ten texts |
| text field | `pkg/formats/alm/encode.go` TextField | ScriptNodeText, 64 bytes |
| node text key | `pkg/scriptprofile/profile.go` TextKey, NodeTextKey | one spelling for both sides |
| script model node | `pkg/game/mapinspectionscriptwords.go` inspectNodeTexts | texts read from the field table |
| node record writer (second game) | `pkg/game/mapinspectionscriptwords.go` setParameterWords | label and texts on any node |
| trigger window node | `pkg/ui/mapeditortrigger.go` nodeFields, secondNodeFields, scriptValues | texts listed and sent |

## Proof

| test | rules |
|---|---|
| `TestSecondMapNodeTextFieldsRead` | 1 |
| `TestSecondGameNodeWindowShowsAndSendsItsTextFields` | 1, 3 |
| `TestSecondGameUndecodedNodeKeepsItsTextsEditable` | 2 |
| `TestSecondGameNodeFieldsEditOnlyDecodedParameters` | 2 |
| `TestSecondMapNodeTextEditSavesIntoItsField` | 3, 4 |
| `TestFirstFormatNodeListsNoTextField` | 5 |

Release witness, EN and RU: `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.
Evidence: the lane's review report.

## Open debt

- Pages, messages and dialogues: no claim names which script action shows a mission text section, and the game
  reads a second-game mission's text from the install only; the panel and the windows stay off.
- Variable names and the map panel: the labels file and the workspace are off for the second game.
- The trigger prefix's bytes +0x40..+0x7f are kept and not shown (R2-ASSET-364: no installed text passes 56 bytes).
- What the ten texts mean is Unknown; settling it needs the watched load run R2-ENGINE-1681 names.
