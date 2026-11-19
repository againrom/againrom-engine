# View second-game maps in the editor

## Result

The editor opens the second game's scenario and root maps for viewing.
The existing map canvas shows terrain, objects, units and structures.
Edit tools and saving stay off, with a plain explanation in either language.
First-game maps still open for editing.

## Authority

- Owner: «Еще задача это редактор rom2»; one editor, with game differences supplied as profile data.
- R2-ASSET-020: the second game's map acceptance ceiling is 1600.

## Rules

1. A second-game install opens its scenario and root maps through the shared reader and canvas.
2. A second-game map exposes no edit capability and saving refuses without writing.
3. The view preserves raw values and section bytes where their meaning is Unknown.
4. First-game maps keep their existing editing and save route.

## Code

| kind | its one builder | change |
|---|---|---|
| install map profile | `gameFiles` through `installGame` | supplies the editor's map format |
| map catalogue | `BuildMapList` and `alm.OpenInfo` | reads metadata with the install format |
| map reader and physical records | `alm.open` and `alm.OpenDocument` | shares validated physical extents |
| editor map session | `mapedit.New`, `Editor.Map`, `Editor.Document` | retains the format across reads and clones |
| map canvas and records | `MapInspector.inspectModel` | reuses the existing view and raw section listing |
| editor capabilities and save | `mapInspectionEdits.project`, editor command table | omits edits and explains refusal |

## Proof

| test | rules |
|---|---|
| `TestSecondGameDocumentKeepsPhysicalRecordsAndBytes` | 1, 3 |
| `TestSecondMapFormatSurvivesCloneAndRefusesSave`, `TestViewOnlyMapRefusesEveryEditCommand` | 2 |
| `TestReleaseSecondMapEditorViewsEveryInstalledMap` | 1, 2, 3 |
| `TestReleaseFirstMapEditorStillOpensAndEdits` | 4 |

Release witnesses: `TestReleaseSecondMapEditorViewsEveryInstalledMap` on both second-game installs;
`TestReleaseFirstMapEditorStillOpensAndEdits` on the English first-game install.

## Open debt

- ROM2 editor epic: map authoring awaits a supported second-game writer; this result remains view-only.
