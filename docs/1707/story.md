# The mod window's Towns page edits towns and tavern offers

## Result

The mod window has a Towns page. It lists the first game's towns by their shown names, marks a town the mod changes and
shows the game's value beside it. For a town the modder sets its name, when it opens and its map point, or switches it
off. For tavern offers the modder adds, changes (character, mission, chapter) and switches off an offer and types its
talk text in each language. Every edit is one undo step and writes the mod's towns file; a town or offer that a
mission of the mod needs is refused in plain words. Without the page's edits no mod or game byte changes.

## Authority

- Owner task order: the map editor mod workbench; plan row 12, card `n-modwb-12`. No divergence: the editor is
  Againrom's own and no game rule changes.

## Rules

1. When the towns kind writes, `[[town]]` records, tavern rows and inn rows read back through `ParseTowns` as written.
2. When a town edit is made, it is one undo step and the towns file holds it as a `[[town]]` record.
3. When a tavern offer is edited, it is one undo step and each language holds its talk text.
4. When a mission of the mod needs a town or offer, switching it off is refused naming the mission.
5. When the draft holds a talk text, the game's check serves it.

## Code

| kind | its one builder | change |
|---|---|---|
| towns kind writer | `pkg/mod/towns.go` `WriteTownData` through `writeTownRows` | writes `[[town]]` records beside offer rows |
| mod window page | `pkg/game/mapinspectionworkspace.go` `workspaceScopeRows` | one row, `pkg/game/mapeditormodtowns.go` |
| talk text | `pkg/mod/talktext.go` | one maker and reader of the part form |

## Proof

| test | rules |
|---|---|
| `TestTownsWriterRoundTripsATownATavernRowAndAnInnRow`, `TestTalkTextReadsBackAsItsWords` (mod) | 1 |
| `TestModTownsPageWritesATownAnOfferAndItsTalkTexts` | 2, 3, 5 |
| `TestModTownsPageRefusesATownAndOfferAMissionNeeds` | 4 |

Release witness at the release: headless town runs on rom1-en and rom1-ru with an editor-made towns mod.

## Open debt

- Second-game `[[inn]]` rows (talk text in `town.txt`), a per-language town name and new towns are not on the page.
