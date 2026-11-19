# The mod window's Screens, Settings and Speakers pages edit their rows

## Result

The mod window has three more pages beside Towns, Bodies, Companions and Dialogues: Screens, Settings and Speakers.
Each page lists the kind's rows and a modder adds, changes and removes them; every edit is one undo step.
Save mod writes through the one project writer and refuses a row the game cannot use, naming it. A speaker is renamed,
given another picture through the asset import, or removed; a speaker a message names is kept and the refusal names
where it is used. A mod saved with no edit keeps each file. No game rule, hash or save byte changes.

## Authority

- Owner: the mod workbench epic; the editor is Againrom's own and has no divergence.

## Rules

1. When a row is added, changed or removed on a page, the kind's file reads back as the rows set.
2. When a file is saved with no edit, its comments, `off` rows and unknown keys are kept and the file reads back equal.
3. When an edit is undone, the project returns to what it was before the edit.
4. When an edit would leave a row the game cannot use, it is refused naming the row.
5. When a setting's bounds or choices change, its default moves into the new range.
6. When a speaker is renamed, only its name changes; a picture is changed by the asset import in the same undo step.
7. When a speaker is removed and a message names it with `npc=`, it is not removed and the refusal names the files.
8. Save mod refuses a screen, setting or speaker the game cannot use, naming the row and what to do.

## Code

| kind | its one builder | change |
|---|---|---|
| screens, settings, speakers writers | `pkg/mod/kinds.go` dataKinds, `writeRows` | a write column and the held-draft mark each |
| speaker add | `pkg/mod/speakerwrite.go` | `NewSpeaker` replaces the append function |
| mod window pages | `pkg/game/mapinspectionworkspace.go` workspaceScopeRows, `pkg/ui` modScopePages | one row each |
| page fields | `pkg/game/mapeditormod{screens,settings,speakers}.go` | fields and setter on the kind step |
| list page layout | `pkg/ui/mapeditormod.go` modListPages | three pages join |

## Proof

| test | rules |
|---|---|
| `TestScreensWriterKeepsFileAndRoundTripsEdits`, `TestSettingsWriterKeepsFileAndRoundTripsEdits`, `TestSpeakersWriterKeepsOffRowsAndRenames` | 1, 2 |
| `TestModScreensPageEditsScreens`, `TestModSettingsPageEditsSettings`, `TestModSpeakersPageEditsSpeakers` | 1, 3, 5, 6 |
| `TestModScreensPageKeepsFileSlotsAndGatesSave`, `TestModSettingsPageKeepsFileAndGatesSave` | 2, 4, 8 |
| `TestModSpeakersPageRefusesRemovingAUsedSpeaker` | 7, 8 |

## Open debt

- A speaker named only inside another map's packed pages is not found by the removal check.
- A setting's key is not renamed on its page.
