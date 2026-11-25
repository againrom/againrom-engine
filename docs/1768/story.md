# The second-game map editor shows and edits the mission text

## Result

The map editor's pages window for a second-game map now lists the sections of the mission text the game reads for
that map, and shows and edits each one; a new section can be added at the end. Each section names the actions that
send its event, or says that no node names it, and the number of a send-event action opens its section. The ten text
fields of a script node stay editable and say that no game reader was found for them, at Medium confidence. Saved
into a mod, an edited text becomes the mod's own copy of that file, and the game shows it when the event is raised.

## Authority

- R2-ENGINE-049 (Medium): the campaign reads the mission text file numbered by the scenario location; the other mode
  reads the quest text.
- R2-ENGINE-047 (High): action 2 sends its event number to the interface, which tells ordinary events from four others.
- R2-ENGINE-050 (Medium): an ordinary event selects its section key from the event number, or is deferred under bit 8.
- R2-ENGINE-1682 (High / Medium / Unknown): mission entry loads its text into the one string the dialogue lookup
  searches; which file that string holds at an inn talk is Unknown.
- R2-ENGINE-1681 (Medium): no reader of the node's ten 64-byte text fields was found.
- R2-ASSET-364 (High): every campaign node holds nonempty text in all ten fields.
- `knowledge/formats/rom2-text`: the section layout of the text files.
- Owner (brief): an edited text goes where the mod layout puts an override; never into an install or a build.

## Rules

1. The pages window of a campaign map holds the sections of its mission's text; any other map's, the quest text's.
2. Each section line names the actions that send its event, or says that no node names it.
3. A section's edit changes that section only; a new key adds a section at the end of the text.
4. A key the game's lookup would find elsewhere, a key that is not a name, a hash mark or a NUL in a text is refused.
5. The number of a send-event action is the text field whose targets are the event sections, and it opens its section.
6. The ten node text fields stay editable, labelled that no game reader was found at Medium confidence.
7. A map saved into a mod writes its edited text there as the language folder's override of the install file.
8. Saved outside a mod, the map is written, the text is not, a note says so, and the edit stays unsaved.
9. A mod that already holds another file at that path keeps it, and the save is refused.
10. When the mission raises the event, the game shows the edited text.
11. An untouched map saves byte-equal and writes no text; the first-game editor texts are unchanged.
12. A text differing from what the target mod plus install serve is written into every mod the map is saved into.
13. A character the mod text file's serve and read-back shows as another letter is refused by name.

## Code

| kind | its one builder | change |
|---|---|---|
| map format edit row | `pkg/game/mapeditrow.go` editRows | second row: mission or quest text address, pages scope only |
| workspace pages fields | `pkg/game/mapinspectionworkspace.go` WorkspaceFields, SetWorkspaceField | a held text gives its sections as pages |
| script parameter kind | `pkg/game/mapinspectionscriptauthor.go` kindRuleBodies | event kinds: send event's number is the text field |
| message targets | `pkg/game/mapmessagepages.go` messageTargets | a held text gives one target per event section |
| install text reader | `pkg/game/installtext.go` installText, textSource rows | quest text row; the section writer shares the game's lookup |
| mod map save | `pkg/game/mapedittestplay_files.go` writeMapInMod | carries the edited text into the mod's files folder |
| node parameter open | `pkg/ui/mapeditorobjectnav.go` openParameter | a target naming a page opens the pages prompt there |

## Proof

| test | rules |
|---|---|
| `TestSecondMapEditorShowsAndEditsTheMissionTextSections` | 1, 2, 3, 4, 5 |
| `TestPageTargetOpensThePagesPromptAtItsText` | 5 |
| `TestSecondNodeTextFieldsNameTheirConfidence` | 6 |
| `TestSecondMapEditedMissionTextSavesIntoTheMod` | 7, 8, 9 |
| `TestSecondMapEventTextCodecNamesWhatTheGameShowsAsOtherLetters`, `TestReleaseSecondMapEditorRefusesTextTheGameShowsAsOtherLetters` | 13 |
| `TestSecondMapEditedMissionTextFollowsTheTargetMod` | 12 |
| `TestReleaseSecondMapEditedEventTextShowsInTheGame` | 7, 10 |
| `TestReleaseSecondMapEditorOpensEveryMissionText` | 1, 11 |
| `TestMessageChoicesUseWorkspaceFirstLine`, `TestMapEditorWorkspaceMetadataAndPagesShareHistory` | 11 |

Release tests at the release:
`TestReleaseSecondMapEditorRefusesTextTheGameShowsAsOtherLetters`, `TestReleaseSecondMapEditorOpensEveryMissionText`,
`TestReleaseSecondMapEditedEventTextShowsInTheGame`, `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.

Census and the headless proof on both second-game roots: review/story1768-rom2-mission-texts/RESULT.md.

## Open debt

- Againrom's game reads no quest text for a map outside the campaign, so an edited quest text is not yet shown.
- Test play from the editor does not carry an edited text into its mission.
- A map of a mod is matched to its mission number only through the mod's mission list.
- A header with a trailing space (`#briefing `, `#failure2 `, `#failure3 `; EN 137, RU 138 sections)
  is refused as finding
  another key's section first. Which section the game's lookup takes for it is not settled,
  and the message is wrong for it.
- Committing a section body unchanged adds a closing line end to a final section that has none.
