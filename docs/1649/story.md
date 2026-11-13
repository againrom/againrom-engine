# F5 in the map editor asks which party plays the map

## Result

On the first game, F5 in the map editor opens one window with six character cards: the main hero and the
campaign's five companions, each under the name and portrait the game gives him beside the kind of hero the
window names. A click puts a character in the party or takes him out, two boxes under a card give him every
skill at 100 and every spell he can learn, and Start plays the map with exactly the chosen party. The window
remembers the last choice until the editor closes.

A mod maker can write the same party by hand for any mission. A party member taken from a record that is one
person now plays under that record's own name, Brian or Rood Glaen, where earlier builds named him by the
archetype of his class and sex; mods already written change with it.

## Authority

- Owner: «on F5 a window offers to start the game with settings: the main hero (on); the traitor mage; the
  monk; the prisoner; Brian; Rud Glaen. These are character cards; a click marks a card as included. A card
  may hold the portrait. Each card has two checkboxes: "skills 100" and "all spells" (where it applies). The
  idea: start the mission with that party.»
- The map editor and the mod format are Againrom's own; no original behaviour is involved and no DIV row.

## Rules

1. On the first game F5 opens the party window: the hero's card, then npc22 to npc26, the hero alone on.
2. A companion's card shows the name, portrait and spells box of the member the game builds beside the hero.
3. A click on a card's box turns the character on or off; a click on Skills 100 or All spells turns that box.
4. A card whose character casts nothing has no All spells box.
5. Start or Enter writes the test mod whose mission declares exactly the cards that are on, and starts the game.
6. With no card on, Start takes no press and Enter does nothing.
7. Cancel or Esc starts nothing and writes no file; the open window writes no file.
8. The next window in the same editor session opens with the last choice.
9. The warnings the notice before a test play listed stand in the party window; no second notice follows.
10. A refused test play, the second game and a host that offers no cards behave as before, with no window.
11. A mission that replaces a campaign mission or declares its own party plays as declared, with no window.
12. A project's guest mission keeps its number, after and gold in the test copy; the working mod is unchanged.
13. A party member written as hero = true is the player's own hero; he takes no level or equipment keys.
14. skills = 100 gives every skill at the rules' cap; spells = "all" gives every spell the member can learn.
15. Another value of either key, a hero naming a character and a second hero are refused with file and line.
16. A mod's own mission with a party array starts by its number, and SAVE and LOAD keep that party.
17. The button between Start and Cancel names the kind of hero the cards stand beside; a press takes the next.
18. The test mission is written for that kind of hero, and a start by number refuses a hero of another kind.
19. A mission whose after names a mission of the project plays in the test copy as a first link.
20. A name wider than its card stands whole on two lines, at the least window too.
21. A party member from a record that is one row plays under the record's own name.

## Code

| kind | its one builder | change |
|---|---|---|
| editor prompt window | `pkg/ui/editorpopup.go` popupSpec, popupGrid | row heights, flat column heads on one or two lines |
| editor prompt kind | `pkg/ui/mapeditormode.go` promptKind | the party prompt, pressed and keyed through the prompt routes |
| test play route | `pkg/ui/editortestplay.go` keyTestPlay | asks the host for cards before the notice |
| test mod | `pkg/game/mapedittestplay.go` testPlayWith | the chosen party, the hero's kind and the first link |
| per-game data | `pkg/game/campaignservice.go` rules objects | the first game names its five companions, the second none |
| mission party | `pkg/mod/missions.go` party parser and writer | hero form, skills and spells keys |
| guest party | `pkg/game/modmission.go` guest member builder | hero member, grants, start by number; builds the cards too |
| editor words | `pkg/words/text/*/strings.toml` | six words in EN and RU |

## Proof

| test | rules |
|---|---|
| `TestTestPlayAsksWhoPlaysTheMap` (en, ru) | 1, 3, 4, 5, 6, 7, 8 |
| `TestThePartyWindowHoldsTheNoticeLines` | 9, 10 |
| `TestTestPlayWritesTheChosenParty` | 1, 4, 5, 7, 8 |
| `TestTestPlayPartyAndThePackageRoute` | 10, 11, 12 |
| `TestPartyMemberHeroSkillsAndSpells` | 13, 14 |
| `TestPartyMemberKeysAreChecked`, `TestPartyHeroEntersAChainAtItsFirstLink` | 13, 15 |
| `TestMissionStartedByNumberPlaysItsDeclaredParty`, `TestDeclaredPartyWithoutTheHero` | 13, 14, 16 |
| `TestMapEditorModesHoldEveryKeyAndGesture` | 3, 6, 7, 17 |
| `TestThePartyCardsAreTheMembersTheGameBuilds` | 2, 4, 21 |
| `TestThePartyWindowNamesTheHeroKind`, `TestThePartyWindowShowsALongNameWhole` (en, ru) | 17, 20 |
| `TestTestPlayWritesTheHeroTheCardsStandBeside`, `TestMissionStartedByNumberRefusesAHeroItDoesNotPlayFor` | 18 |
| `TestTestPlayStartsALaterLinkWithTheHeroAlone` | 19 |

Release tests at the release: `TestReleaseTestPlayPartyWindowFrames`, `TestReleaseTestPlayOfANewMapStartsAtOnce`,
`TestReleaseTestPlayOfAMapWithAnotherWarningStillAsks`. Frames and the card matches are in the review report.

## Open debt

- The owner decides whether a mission that declares its own party should also get the window.
- The mod window was not adapted to a hand-written hero member: it lists him under a blank character name.
- The second and third companions' levels (2 and 3) follow the towns that grant them; the owner may ask others.
- The one-row names were read on the EN install by the review; the RU install is Unknown until the release.
