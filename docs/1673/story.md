# One spell catalog for both games

## Result

Every spell of both games has one name, and each game's list of spells is written down once: which spells the
game has, their numbers, their places in the book, whether a shop sells their book and scroll, and which first-game
behaviour a second-game spell runs. The book, the shops and casting read that list, so without a mod both games
play, look and save exactly as before. A mod names a spell by the same name in both games, and two mods that edit
different values of one spell are combined into one edit.

## Authority

- Owner: «Spells in mods», accepted with refinements: one spell catalog for both games, one name per spell, each
  game's profile as data; this story is the first of four.
- `MAGIC-SPELL-001`, `R2-ENGINE-019`: the spell numbers of each game; `DAT-NAMES-016`: row names are the same on
  both language roots.
- `AI-SPELLIDENT-286`, `MAGIC-ICON-024`, `R2-ENGINE-444` (`DIV-3107`): the book positions.
- `SHOP-CONSUME-073`: the shop's book and scroll admission; `R2-ENGINE-024`: the second game's arms.

## Rules

1. Each game's book order, book icons, shop books, shop scrolls and second-game arms are the catalog's profile.
2. Without a mod, every value of rule 1 equals the table it replaced, on both games.
3. A spells row names its spell by its catalog name; `_` and a space are one, so an old `target` keeps working.
4. The rows of several mods for one spell merge into one row through the layer merge; one field is still one mod's.
5. The Spell names table of the modding guide lists the catalog: every name once with its number in each game.
6. A later row keeps each field an earlier row set and omits, `self_cast` and `heal_hostile` too (main reset them).

## Code

| kind | its one builder | change |
|---|---|---|
| spell catalog | `pkg/rules/spellcatalog.go` | new: the names and both profiles; `SpellIDs` is the first profile's length |
| book order | `pkg/data/spellpresentation.go` `SpellPresentation` | `BookRows` and `NamedRows` read the profile |
| book icon slot | `pkg/data/spellicon.go` `SpellIconSlot` | reads the first profile's positions |
| shop admission | `pkg/game/shopconsumables.go` `shopBookSpellAdmitted`, `shopScrollPool` | read the table's game profile |
| second-game arms | `pkg/sim/spellarms.go` `SecondGameSpellArm` | reads the second profile |
| spells mod kind | `pkg/mod/kindread.go` `readSpells`, `pkg/mod/spells.go` `MergeSpells` | catalog key; rows through `Merge` |
| spells applier | `pkg/game/modspell.go` `setModSpells` | merges over the table's names, reports overrides |

## Proof

| test | rules |
|---|---|
| `TestSpellCatalogProfilesReproduceTheReplacedTables` | 1, 2 |
| `TestSpellIconSlotIsTheEnginesOwnTable` | 1, 2 |
| `TestShopBookAndCaptionSpellListsKeepTheirAnswers` | 1, 2 |
| `TestSecondGameArmsFollowThePublishedTable` | 1, 2 |
| `TestSpellCatalogNamesEachSpellOnce` | 3, 5 |
| `TestMergeSpellsKeysRowsByTheCatalogName` | 3, 4 |
| `TestModSpellRowsNameASpellByCatalogNameOrOldTarget` | 3, 4 |
| `TestLaterModSpellRowKeepsWhatAnEarlierRowSetAndItDoesNotName` | 6 |
| `TestTwoModsCannotSetTheSameSpellField` | 4 |
| `TestFixtureModsGiveTheRecordedMergedData` | 2, 4 |
| `TestModsDocSpellNamesAreTheCatalog` | 5 |

Release tests at the release, on all four bases: `TestReleaseShopStockCensusWithoutModsIsShipped`,
`TestReleaseSecondShopStockMatchesMain`, `TestReleaseSecondShopBookIcons`, `TestReleaseOwnerShopScrollStoredValues`,
`TestReleaseSpellbookPopupComposesFromSpellsTxtAndInstalledLabels`, `TestReleaseSecondGameSpellsDrawAndSoundTheirOwn`,
`TestReleaseSecondGameAttachedEffectsActAsTheirOwnArms`, `TestReleaseModSpellKeysEditTheInstalledTable`,
`TestReleaseModSpellEditsReachACastAndTheSave`.

## Open debt

- The spell switch, book position, book size and shop fields wait for the second spells-in-mods story.
- The spells kind stays kept: the reader still refuses a field two mods set; `load-after` does not order it yet.
