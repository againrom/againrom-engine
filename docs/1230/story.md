# 1230 — tavern roster presentation

## Intent

The owner compared the original EN tavern with Againrom's on `cityafter.sav`
(mission 130). The original draws the talk-only candidate Brian with his own
sheet, orders the mercenary cells differently, shows Brian's statistics when
his cell is selected, and groups large numbers with commas. Research answered
in knowledge k78 (`870202dc`).

## As built

| Surface | Behaviour | Claims |
|---|---|---|
| Grid | Position `i` draws in rect `i`: column `i%6`, row `⌊i/6⌋` from the bottom, 48x64 cells from x 176 | TOWN-467 |
| Halves | Mercenary cells first, then one talk cell per `InnNPC` element in array order, over `manbacktalk.bmp` with no text | TOWN-468, SAV-1112 |
| Selection | Tavern entry selects position 0, the bottom-left cell, when a mercenary cell exists (owner ruling; the original's activation store). A roster with no mercenary keeps the -1 store: nothing is selected and no cell is painted until a click (`TownSurfaceView.RosterUnpainted`); the hit test covers every position | TOWN-468 |
| Mercenary order | `tavernStockWalk`: types 1..15 as the actor-map walk visits stock units built in type order, newest first inside a 16-id block; `tavernStockIDPhase = 6` puts types 1..10 in one block and 11..15 in the next. Medium for the phase (DIV-1410) | TAVERN-ORDER-015, SAV-1111 |
| Talk sheet | `resolveTavernTalk` resolves the npc record over the carried party, then the stage's Human stock mercenary units in the stock walk (`tavernTalkCast`; party first is this build's order). The `Human` term excludes heroes. A Hero object draws `HeroFighter` or `HeroMage` by its mage bit, any other `Unit<+0x15b>`: npc90 and npc59 resolve to the stock units `NPC10_1` and `NPC08_2` and draw `Unit10` and `Unit8` (Medium in the claim). An unshipped sheet on the synthesised arm draws only the ground (DIV-1408). Replaces `29 + index%2` | TAVERN-TALKPIC-016, TOWN-470, TAVERN-ORDER-015 |
| Talk statistics | A live party object shows its statistics card and figure, a live stock unit its mercenary card; a synthesised object shows none, and below the panel its composed face (Hero or Human) or the class `infowindow` picture under a `Picture` token. The panel renders once per selection and party | TAVERN-TALKSTATS-017 |
| Sight | The tavern candidate card prints sight as the whole sight word: whole cells, then one decimal truncated toward zero; the same truncation now applies wherever a card carries the word | HERO-SIGHT-007, HERO-104 |
| Grouping | `ui.GroupDigits`/`ui.GroupSigned` at these sites: tavern cell prices, Hire/Fire price and Exit money; school button values; shop button values; shop stock grid quantity and price; shop pack money cell; mission pack purse cell (item-stack counts stay ungrouped); generator remaining points and signed step text; hall of fame score. The tavern count, hall rank and character card stay ungrouped. Plaque prices draw in the card font so that every price fits its plaque at every digit count; no claim names the font (DIV-1405) | TOWN-469, SHOP-SCREEN-037 |
| Pin | knowledge k75 `0ea8f842` → k78 `870202dc`; DIV-426 and DIV-485 restated against the new claims | TOWN-065 retracted; TOWN-010, TOWN-012, TOWN-066, TOWN-282, REG-118, REG-SCN-064 partly retracted; MERC-TYPE-001 amended |

The owner-directed four-button layout (DIV-483) and the statistics card and
tooltip are unchanged. The owner ruled that tavern entry selects the
bottom-left cell, superseding story 1048's empty entry (DIV-426).

## Proof

- `pkg/game/tavernroster_test.go`: owner roster order and rect hits,
  sheet choice per arm, statistics gate, school grouping.
- `pkg/ui/grouping_test.go`: sign rule, shop grid and money cell, mission
  purse cell against an item stack, hall score, generator step text, sight
  decimal.
- `pkg/game/tavernroster_test.go`: stock-unit resolution, the Human term,
  the talk panel cache and its invalidation.
- `TestReleaseShopPricesFitTheirPlaques` (EN and RU): grouped figures of 4 to
  7 digits against the real plaque ink on both sides and all three grids.
- `TestReleaseEveryMercenaryUsesInstalledTalkPortraitAndInspectionArt` (EN and
  RU): every `InnNPC` element at its own stage draws its sheet; npc90 and npc59
  resolve to the stock units of types 10 and 8.
- `pkg/game/townshell_test.go`: activation and the paint guard.
- `TestReleaseTavernRosterOnAReloadedTownSave` (EN and RU): mission-130 town
  written as SAV, reloaded, tavern opened.
- `cityafter.sav` witness outside the committed tests, EN and RU: the 14
  cells match the owner's original screenshot cell by cell (price, count,
  position); Brian resolves to the live party member, draws `HeroFighter`
  and shows his statistics. The three 420,000 mage cells sit at positions 6
  (middle row, first), 7 (middle row, second) and 5 (bottom row, last) and
  show the statistics OBSERVATIONS.md transcribes for its first, second and
  third panels, sight included (6.5, 6.3, 6.3).

## Open debt

- DIV-1405: the plaque font is measured, not decoded.
- DIV-1408: an unshipped talk sheet draws the ground; the original aborts.
- DIV-1409: the generator step text has no researched surrounding prose.
- DIV-1410: the stock id phase is fitted to one screenshot.
- The mage panel match rests on the seat's Medium reading of "left to
  right".
- `Platoon` and the `+0x15a` state gate are not evaluated in town; npc2
  takes the synthesised arm.
- The tavern dialogue portrait still resolves over an empty cast, so a talk
  cell resolved live can differ from its dialogue picture.
