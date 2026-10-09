# ROM2 first town drawn as the first game's town

## Intent

Before this story the second game's first town was a text list: a header,
the rows TAVERN and GATES, and TALK rows in the inn. The owner directed that
the second game's first town is the first game's town, built from the town
screens the engine already has. This story draws town 1's square and tavern
with that machinery over the second game's own art. Towns 2 and 3 (Kaarg and
the druid town) keep the text list. The campaign logic, the departure and its
gate rule do not change. ROM1 code paths, release tests and hashed state do
not change.

Base: `772868a6` (game 0.100.6). Knowledge pin: k205.

## Authority

| Part | Authority |
|---|---|
| town 1 is the first game's town | owner direction (DIV-2672) |
| type-2 ID-1 route reaches town presentation; mission 10 unlocked by TALK | R2-ENGINE-073 |
| town-1 departure removes the town record | R2-ENGINE-144 |
| stage-10 inn speakers | R2-ENGINE-161 |
| stage-30 inn entries | R2-ENGINE-216 |
| TalkTo and AddMission | R2-ENGINE-219 |
| speakers are kinds 0 and 3, first match, npc%dtalk%d | R2-ENGINE-220 |
| square mask codes, labels, exterior motion | the first game's town claims the square already cites (TOWN-163, TOWN-399..404, TOWN-489) |
| school building opens nothing, no highlight | owner direction (DIV-2673) |
| shop closed, no first-game stock | seat direction; no second-game shop decoder (DIV-2675) |
| no hiring | seat direction; no second-game hire authority (DIV-2674) |

Observed on both second-game roots (`graphics.res` listing): `interface/town`
(42 entries) and `interface/townbirds` (41) carry the first game's names;
`interface/inn` (154) lacks the Unit2 and Unit6 sheets and adds others;
`interface/training` has no entry; `interface/shop` carries the first game's
names beside `shop_druid` and `shop_kaarg`. `main.res` has no `text/tips`.

## As built

- `pkg/game/secondtown.go`: `secondCampaignScreen` keeps a private
  `townScreen` used only for presentation, bound through `bindTown`. Its room
  follows the campaign state on every seam call: the square for town 1 at
  `secondTownSquare`, the tavern at `secondTownInn`, and no drawn room
  elsewhere. Room changes run the first game's entry and exit steps
  (`atSquare`, `enterTavernInterior`, `composeShopFaces`).
- The screen implements the square seams (`AtTownSquare`, `TownSquareView`,
  `TownSquarePointer`, `TownSquareActive`, `AdvanceTownSquareAnimation`), the
  tavern seams (`AtTownSurface`, `TownSurface`, `TownSurfaceClick`,
  `TavernInteriorActive`, `AdvanceTownSurfaceAnimation`) and `TownMusic`.
- Square rows are the first game's four doors in its door order: TAVERN,
  SHOP and SCHOOL (both not choosable) and GATES (choosable by the existing
  gate rule). A mask click reaches `Choose` through the existing door index.
- A pointer over the school's mask code is delivered as off the picture, so
  the school shows no label, motion or sound.
- `townScreen.gateRule` replaces the first game's gate answer when set; the
  second game sets it to `secondCampaign.gateOpen`. Nil keeps the first
  game's rule.
- The tavern view lists one talk-only card per speaker in option order. No
  card is painted until one is selected. A click selects; Talk or a double
  click runs the existing TALK (`npc%dtalk%d` dialogue, then TalkTo). Sleep
  and Hire are disabled; Exit returns to the square. The inn's text rows are
  `TALK <npc>` without GATES; they remain the `Choose` seam.
- `LoadSecondTownTavernArt` loads the tavern art with a missing mercenary
  sheet left empty; `loadInstallShare` uses it for a second-game install.
  `LoadTownTavernArt` is unchanged for the first game.
- `reportWordsFor` converts the tavern Talk, Exit and hired captions and main
  text hover slots 233 to 237 on the RU second-game root, on DIV-2378's
  mapping.
- SAV: no new field. The existing `currentSecondCampaign.Room` carries the
  square or the tavern; a cold LOAD lands in that room.
- `ui.TownRow`'s text list stays for the destination list, towns 2 and 3,
  the inn's `Choose` seam and an install whose town art does not load.

## Proof

Focused tests (`pkg/game/secondtown_test.go`, synthetic install with a
synthetic square mask):

- `TestSecondFirstTownOpensTheSquare`: a new game opens the square, its four
  door rows and its picture.
- `TestSecondFirstTownDoorHoverAndClick`: hover selectors (tavern 2, shop 1,
  school none, gate 8), closed doors open nothing, the tavern click opens
  the tavern.
- `TestSecondFirstTownTavernTalksThenGatesLeave`: the speaker cards, select,
  Talk, double click, TalkTo and AddMission, Exit, the gate rule, the
  departure and the destination list.
- `TestSecondFirstTownRowsKeepTheTalkSeam`: the inn's TALK rows run TalkTo.
- `TestSecondFirstTownRoomSurvivesTheSaveDocument`: square and tavern
  survive the current-state document.

Installed witnesses (`pkg/game/secondtown_release_test.go`), run on the EN
and RU second-game roots:

- `TestReleaseSecondFirstTownSquareAndTavern`: a fresh game reaches town 1;
  the square is the installed picture; the hover labels; the closed shop,
  school and gate; the tavern over the installed inn art; TALK 517 makes
  mission 10 available; the gates lead to the destination list.
- `TestReleaseSecondFirstTownColdLoadKeepsTheRoom`: a named SAVE at the
  square and in the tavern, then a cold LOAD in a fresh front end, keeps the
  room and the campaign state.

The existing second-game tests now name the speaker card `NPC 517` and leave
the tavern by Escape before GATES.

## Open debt

- DIV-2673: whether the original highlights the school building or shows its
  hint on hover is Unknown.
- DIV-2674: no hiring, no purse figure on Exit.
- DIV-2675: the shop is closed until second-game stock, prices and the shop
  flow are decoded.
- DIV-2676: speaker cards have no actor picture and no inspection panel.
- DIV-2677: the closed gate shows no gate line.
- DIV-2678: the town dialogue's button word is not converted on the RU root.
- Towns 2 and 3 keep the text list.
