# The editor's object windows on the one window framework

## Result

The structure, sack, record, group and used-by windows of the map editor, and the Back step between them, are now
laid out by the editor's one window framework in English and Russian. The record's picture is the window's first row.
A structure the scripts name shows that name in its title. The group window lists its members with their pictures.
A long used-by line wraps and stays one link, and a long used-by list starts folded. Fields, Apply, Cancel and every
link target stay as they were.

## Authority

- Owner: the nine T1 answers on the object-window mockup, binding; answer 8 took the fold closed over five uses.
- Owner: «one builder per kind»; the object windows keep the right-panel host they have today.

## Rules

1. The record picture is the window's first row, three rows high, in the label column, in both views.
2. A structure with a script number is titled "Structure N: name" ("Постройка N: name").
3. The sack's Details fold names an item's dispatch and the plain recipe in Russian words in a Russian game.
4. A record without editable fields says so behind the title's (?), not in a note.
5. The group window's members row is "Members N" ("Состав N") with the members' pictures, then the member links.
6. A used-by line wider than the window wraps under its link, and a press on any of its lines opens the use.
7. Navigation between object windows is Back only, the last button of the button row.
8. The used-by block is a fold, closed when there are more than five uses and open otherwise.
9. The view switch is a button of the button row: "View: Basic" / "View: Detailed" ("Вид: Обычный" / "Вид: Подробный").
10. Each window lies inside the screen at 640x480, 800x600 and 1920x1080 and its rectangles hit as laid out.

## Code

| kind | its one builder | change |
|---|---|---|
| object window | `pkg/ui/mapeditorwindow.go` objectSpecRows | picture row first; script-name title |
| object fields | `pkg/ui/mapeditorobject.go` objectFieldRows | read-only note as the title help |
| group window | `pkg/ui/mapeditorobject.go` groupSpec | members row with pictures, member links |
| Back | `pkg/ui/mapeditorobjectnav.go` objectBackButton | unchanged; the only navigation button |
| used-by list | `pkg/ui/mapeditorusedby.go` scriptLinkSection | fold; wrapped lines are one link |
| row picture | `pkg/ui/editorpopup.go` rowPictureBoxes | one box builder for drawing and the page |
| words | `pkg/words/text/{en,ru}/strings.toml` | dispatch and plain-recipe words; "Подробный" |

## Proof

| test | rules |
|---|---|
| `TestObjectWindowsHoldTheLayoutProperties` | 10 |
| `TestObjectWindowPictureIsTheFirstRow` | 1 |
| `TestObjectWindowStructureTitleIsTheScriptName` | 2 |
| `TestObjectWindowSackDetailsAreRussian`, `TestEditorNativeChoiceWordsUseFontBytesInBothLanguages` | 3 |
| `TestObjectWindowReadOnlyNoteIsTheTitleHelp` | 4 |
| `TestObjectWindowGroupMembersRow` | 5 |
| `TestObjectWindowWrappedUseIsOneLink` | 6 |
| `TestObjectWindowNavigationIsBackOnly` | 7 |
| `TestObjectWindowUsedByFold` | 8 |
| `TestObjectWindowViewSwitchIsARowButton`, `TestMapEditorViewSwitchNamesBasicAndDetailed` | 9 |

Release tests at the release: none.

## Open debt

- The used-by fold returns to its default when an Apply reloads the record; none waits in BACKLOG.
