# Second-game palettes list the second game's catalog

## Result

A second-game map in the editor now shows the Units, Structures and Sacks palettes. They list the second game's own
catalog by name. A placed unit or structure shows the name of its catalog row. A placement whose kind or definition
reaches no row shows its raw id and the words "no catalog row". The palettes only list and show: placing stays off.

## Authority

- Owner: «Еще задача это редактор rom2»; one editor, game differences as profile data.
- R2-ASSET-124: the Buildings row names; that a type-4 kind indexes Buildings is Medium and the editor follows it.
- R2-ASSET-239, R2-ENGINE-1290: a type-6 record names a Units row by server id, or a Humans row with the person flag.

## Rules

1. On a second-game map the Units palette lists every named Units and Humans row of the install, not only keyed ones.
2. The Structures palette lists every named Buildings row; the Sacks palette lists the install's item names.
3. A placed unit or structure shows its catalog row's name; an unresolved one shows its raw id and "no catalog row".
4. A press on a palette cell of a second-game map arms no placing and the host refuses every add.
5. The first game's palettes, placing and field rows behave as before.

## Code

| kind | its one builder | change |
|---|---|---|
| unit palette | `identityChoices` | profile flag lists every named row |
| structure palette | `recordCatalog` "kind" | profile flag lists Buildings rows |
| palette grid | `placeGridSpec`, `paletteHost` | a map with no edits lists from `Palettes` |
| placed object rows | `inspectModel` | catalog name or "no catalog row" |
| structure row join | `mapload.BuildingName` | one reading of the kind-to-row rule |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondPalettesListTheCatalogAndNameEveryPlacement` | 1, 2, 3, 4 |
| `TestViewOnlyMapListsPalettesAndArmsNothing` | 4 |
| `TestBuildingNameSaysWhenNoRowNamesTheKind` | 3 |
| `TestReleaseFirstPalettesStillListAndPlace` | 5 |

Release tests at the release: both second-game installs and the first-game install.

## Open debt

- Sacks list the install's item-name table; no claim joins it to the data.bin item groups.
- Placing from the palettes awaits a supported second-game writer.
