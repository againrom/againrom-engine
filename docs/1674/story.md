# The mod window's Terrain, Texts and Items pages are picked, not typed

## Result

The Terrain page shows each ground as one picture in a grid; resting the pointer on one shows all
its variations at the preview's size. The Texts page says what it holds and lists each text under
the place it shows; a new text is made for a place, and the editor names its key. On the Items page
a new armour or weapon starts from a game item chosen by name and picture; every item, tier,
material and picture is a choice, only names and numbers are typed, and every row has help.

## Authority

- Owner: «Земли, каждый тип земли надо показывать друг за другом, а не строками.
  при наведении на землю показывается попап со всеми вариациями
  такого же размера как предпросмотр»
- Owner: «Тексты экран который непонятен что он делает»
- Owner: «Экран предметы стал лучше но тоже неюзабелен.
  Вручную никто ничего писать не будет»
- Owner: «ВРУЧНУЮ НИКАКИЕ ЗНАЧЕНИЯ НИКТО НИГДЕ ПИСАТЬ НЕ БУДЕТ,
  ЕСЛИ ЭТО ЗНАЧЕНИЕ ПОДРАЗУМЕВАЕТ КАКОЙ-ТО ПУТЬ
  или ДЛИННУЮ СТРОКУ, КОТОРУЮ НИКТО НЕ ЗНАЕТ»

## Rules

1. When the Terrain page opens, each ground is one picture cell in a grid, the game's own first.
2. When the pointer rests on a ground cell, a popup shows all its variations at the preview's size.
3. When the Texts page opens, a line says what it holds and each text stands under its place.
4. When a modder makes a text for a place, the editor makes its key; the modder types only the texts.
5. When a modder adds armour or a weapon, it starts from a game item chosen by name and picture.
6. An added row's key and name key come from the editor; its name is typed per language.
7. A stand-in, base, changed item, tier, material and picture are choices; no typed key, row or path.
8. Each editable row on the three pages has a label and a help line in both languages.
9. An armour and a weapon added on the page save, publish and load with their names.

## Code

| kind | its one builder | change |
|---|---|---|
| mod page grid | `pkg/ui` popupGrid with TipPicture | grounds as cells with variation tips |
| editor popup | `pkg/ui/editorpopup.go` popup layout | the variation popup is its data |
| install item choice | `itemChoice` over `installItems` | one list for new, stand-in, base, change, material |
| mod record key | `mod.RecordKey` | mission, text and item keys take it |
| install item name | `installItems` | the game's name in UTF-8; a weapon with none is not offered |
| mod window page rows | `modScopePages` | rows unchanged in shape |
| editor words | `pkg/words/text/{en,ru}/strings.toml` | labels and help in modder words |

## Proof

| test | rules |
|---|---|
| `TestModTerrainGridWrapsGroundsWithVariationTips` | 1, 2 |
| `TestModTerrainGridStaysInsideTheEditorWindow` | 1, 2 |
| `TestTerrainPageAddsAGroundFromOnePicture` | 1 |
| `TestTextsPageShowsAMissionNameMadeFromMissions` | 3, 4 |
| `TestTextsPageMarksATextMissingInOneLanguage` | 3, 4 |
| `TestItemsPageTypesOnlyNamesAndNumbers` | 5, 7, 8 |
| `TestItemsPageAddsChangesAndSwitchesOffRows` | 5, 6, 7 |
| `TestItemsPageRefusesAFieldByTheReadersRule` | 7 |
| `TestItemsPageAddedRowsRoundTrip` | 6, 9 |
| `TestItemsPageOffersEveryInstallItemByTheGamesName` (install) | 5, 7 |
| `TestEveryUsedIDIsInTheEnglishTable`, `TestShippedTablesLackNoID` | 8 |

Release tests at the release: `TestReleaseMapEditorWorkspaceRoutesAndTexts`.

## Open debt

- New rows start with zero numbers; no reader turns a game item's scaled numbers back into row values.
