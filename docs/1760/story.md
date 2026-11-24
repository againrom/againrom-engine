# The unit window sets maximum mana and the unit's spell book

## Result

In the map editor's unit window the author now sets a placed unit's maximum mana as maximum health is set:
an empty or zero value keeps the maximum the unit's kind gives, and the game spawns the unit with the set
maximum and its mana no higher.

On a unit whose card states the "uses magic" caption, a press on the caption opens a small spell book drawn
as the game's book; each spell is a switch, and the spells are the ones the map's mod states. A spell turned
on is saved as the record the original editor writes, and reopening the map shows it on. On a second-game
map both stay refused, with the reason.

## Authority

- Owner: «давать возможность редактировать максимальную ману (как здоровье)».
- Owner: «если есть строка ИСПОЛЬЗУЕТ МАГИЮ, то по клику можно туда нажать и открывается
  небольшой экран настройки книги (что знает персонаж) -- там тогглы в виде самой книги
  заклинаний ... важно переиспользовать все».
- `DIV-5150`: maximum mana is an Againrom word at type-6 +0x26, which the original leaves unread
  (`ALM-TAILU16-082`, `ALM-UNITREAD-227`).
- `DIV-5152`: the book writes the original editor's Unit Spellbook record (`ALM-EDITT9-328`) with power 100.
- `DIV-5151`: the game's spawn does not yet give a unit the map's unit spell records (`ALM-T9ACTOR-176`).
- `R2-ENGINE-1292`, `R2-SESSION-162`: the second game's unit record holds no mana word, and what its mode-8
  record gives a unit is not read; both stay refused there.

## Rules

1. When the unit window sets maximum mana, the map holds it, a reload reads it and the spawn uses it.
2. When maximum mana is empty or zero, the kind's maximum stands, and mana never exceeds the maximum.
3. When the SPELLCASTER caption is pressed, a book window opens with a switch for each spell of the book.
4. When an off spell is pressed, the map gains the original's unit spell record; a second press removes it.
5. When the saved map is reopened, the book shows the same spells on; Undo restores the map byte for byte.
6. When a spell is one the unit's kind knows, it stays on and the press is refused with its reason.
7. When the map's mod changes the spell table, the book's switches follow its spells and order.
8. When the map is a second-game map, maximum mana is absent and a book press is refused in its words.
9. When an installed map of either game is opened and saved untouched, the file is byte-equal.

## Code

| kind | its one builder | change |
|---|---|---|
| unit record field | `pkg/mapedit/field.go:95` unitFields | `manamax` row at +0x26 |
| unit override | `pkg/game/mapinspectionoverrides.go:17` overrideKeys | `manamax` beside `hpmax` |
| spawn tail | `pkg/mapload/placementtail.go:60` withTailWords | maximum mana replaces the derived one |
| human derive | `pkg/data/humanderive.go:82` DeriveHuman | authored maximum mana kept |
| type-9 add | `pkg/game/mapinspectionadd.go:74` addCellSpell | the book's record goes through the add |
| type-9 unit arm | `pkg/formats/alm/cellbinding.go:61` UnitSpellRecords | the book and a unit's delete (`mapinspectioncasts.go` objectSpells) read A&8 records here |
| book cells and painter | `pkg/game/mapeditormodspells.go:165` bookCells, `pkg/ui/mapeditormodbook.go:58` modBook | shared by the mod window and the unit book; a cell may be a switch (`Toggle`) |
| per-game row | `pkg/game/mapeditrow.go:138` editRow.unitBookWord | the second game's refusal word |

## Proof

| test | rules |
|---|---|
| `TestEditorUnitManaMaxReachesTheMap` | 1, 2 |
| `TestEditorUnitBookTogglesTheUnitSpellRecord` | 3, 4, 5, 6 |
| `TestEditorUnitBookFollowsTheModSpellTable` | 7 |
| `TestSecondEditorRefusesTheUnitBookAndManaMax` | 8 |
| `TestEditorUntouchedMapWritesByteEqual` | 9 |

The save census over the four installs is in the review report `story1760-editor-unit-mana-book/RESULT.md`.

Release tests at the release: `TestReleaseMapEditorSavesAnInstalledMapKeepingItsUntouchedBytes`.

## Open debt

- `DIV-5151`: the game's spawn does not give a unit the spells of the map's unit spell records.
- Whether the original loads a map with an edited book is Unknown; no original run was made.
- A spell the mod turns off keeps its record, and only the type-9 records window removes it.
- A spell with no book position (for example Slow) can be given by a record and has no cell.
- Whether an off press removes a record whose A holds 8 with other bits (A=12) is Unknown.
- `manamax` on an NPC-arm placement is applied but the window prints the ignored line it shares with `hpmax`.
- No EN or RU install witness of the book window ran; only fixture tests at 1920x1080.
- `pkg/game/chargentrace_release_test.go` strips a `%+v` string that no longer matches; its helper has no caller.
- The `strings.toml` edit left whitespace churn on the `editor.map.secondscript` line.
- The population and release names of the gated tests settle at the landing.
- The second game's maximum mana and unit book wait for a claim of the mode-8 record's effect.
