# The map editor reads its first-game tables from the open install

## Result

The map editor's lists of Objects kinds, hand-out buildings, hero words, item suits and addable record kinds now come
from the open install. On a first-game install every list shows and does what it did before. On a second-game install
the editor offers units, structures and sacks to its palettes, and a list the research does not name for that game
shows one line saying so.

## Authority

- Owner: «Еще задача это редактор rom2» and «Редактор ром2 давай доделаем»; one builder per kind, the first game as
  profile data.

## Rules

1. On a first-game install the five tables hold the values the editor held before.
2. On a second-game install the addable kinds are units, structures and sacks; the other four tables are empty.
3. A route that reads an empty table shows one line saying the research names no entries, and writes nothing.
4. No package-level variable of the editor carries a first-game table again.

## Code

| kind | its one builder | change |
|---|---|---|
| editor tables | `editorTables` held by `gameFiles` | one value per game; read through `MapInspector.editorTables` |
| Objects kinds | `structurePaletteInput` | takes the install's tables |
| hand-out buildings, hero words, item suits | `editHandout`, `heroField`, `itemsFields` | read the install's tables |
| addable record kinds | `MapEditor.offersKind` | the host lists its kinds; the forms stay in `addKindForms` |

## Proof

| test | rules |
|---|---|
| `TestFirstEditorTablesKeepTheirValues` | 1 |
| `TestSecondEditorTablesHoldOnlyWhatIsNamed` | 2 |
| `TestReleaseEditorTablesFollowTheInstall` | 1, 2, 3 |
| `TestEditorTablesAreNotPackageLiterals` | 4 |

Rule 3 is proven on the hero field; the hand-out and item routes read their table the same way.

Release tests at the release: `TestReleaseEditorTablesFollowTheInstall` on both second-game installs and the first.

## Open debt

- The research names no second-game Objects kinds, offer tables, hero words or item suits: those stay empty.
- Placing on a second-game map awaits a supported second-game writer.
