# A mod gives a spell its own book icon, cast picture and burst

## Result

A mod can give any spell its own picture in the spell book and its own flying picture and burst, in both games.
A spell the game's book has no picture for shows the mod's icon in the book, the shops and the editor's catalog.
In the map editor's Spells page the author picks the three pictures as files, or copies them from the other game's
install with one button. A missing or broken picture stops the mod check with the file named; publishing a mod
with copied game pictures warns and still writes it.

## Authority

- Owner, card n-modwb-25 and rulings «Spells in mods», «Spell book size and picture source»: a mod gives a spell
  a book icon and a cast effect; install pictures are import only, copied into the mod with their registry rows.
- Owner, answer (a) to card o-spell-import-publish: publishing a mod with imported game pictures warns, not refuses.
- Owner, seat answers: the other game's install comes from the starter's bases; a mod's own rows govern that mod's
  game, flight time included, and its world hash may differ; the game without a mod keeps main's hash and pixels.

## Rules

1. When a mod's spell row names an icon, the book and the editor's catalog draw it over the spell's cell.
2. When a row names a cast picture or burst, the game draws its frames at rows 2n+8 and 2n+9 on both games.
3. When a row names a cast picture or burst, that mod's game takes its flight from the row; the hash may differ.
4. When no mod names a picture, the book, the casts, the flight inputs and the save stay as on main.
5. When a named picture is missing, of the wrong size or unreadable, the check refuses with mod, file and line.
6. When the author picks a picture file, it is checked, copied under assets/spells and named in the row at once.
7. When the starter lists the other game's install, its icon, sheets and rows are copied and the row is imported.
8. When the starter lists no install of the other game, the copy button is shut with a caption saying so.
9. When a mod with an imported row is published, the note warns and names the spells; the mod is written.
10. When a .256 sheet's trailer counts more frames than its bytes hold, the decoder refuses before allocating.

## Code

| kind | its one builder | change |
|---|---|---|
| book icon | `pkg/game/spellicon.go:155` cutBookIcon, `pkg/game/spell.go:48` markMovedSpells | mod icons as data |
| projectile art and flight | `pkg/game/projectiles.go:234` projectileSet, `pkg/game/flightinputs.go:38` | over registry `With` rows |
| mod spell pictures | `pkg/game/modspell.go:64` setModSpellPictures, `:167` spellSheetRow | rows from the sheet |
| spell rows | `pkg/mod/spells.go:377` ParseSpells, `pkg/mod/spellwrite.go:18` writeSpells | picture fields |
| mod asset import | `pkg/game/modassetimport.go:33` modAssets, `:268` importModFiles | spells rows; many files |
| Spells page | `pkg/game/mapeditormodspells.go:269` spellPictureFields | pickers and copy |
| install locator | `pkg/launch/starterini.go:141` StarterBases under StarterBase | every base listed |
| .256 decoder | `pkg/formats/spr256/spr256.go:71` Decode | frame count bounded |

## Proof

| test | rules |
|---|---|
| `TestModSpellPicturesDrawInTheBookAndInFlightOnBothGames` | 1, 2, 3 |
| `TestSpellRowsWithoutPicturesDrawAsTheInstall` | 4 |
| `TestModSpellPicturesRefuseAMissingOrMalformedPicture` | 5 |
| `TestSpellPictureFieldsRefuseAtTheirLine`, `TestTreeCheckNamesSpellPicturesAndWarnsOnImported` | 5, 9 |
| `TestSpellsPagePicksTheChosenSpellsPictures`, `TestImportRefusesWhatTheKindsLoaderRefuses` | 1, 6 |
| `TestSpellsPageCopiesTheOtherGamesPicturesAndPublishWarns` | 7, 8, 9 |
| `TestRejectShortAndOversizedStreams` (spr256) | 10 |

Release tests at the release: none; every fixture is synthetic. Evidence: `review/mod-workbench/spell-pictures/`.

## Open debt

- none
