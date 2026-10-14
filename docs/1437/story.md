# The town plays the original's interface sounds

Base: `e6cde80c`. Knowledge pin k311.

## Intent

The owner, playing the original beside the engine, heard sounds in the shop, the tavern and on
tip windows that the engine did not play. This story finds every town interface sound request the
claims name and adds the missing ones through one builder.

## Authority

`SHOP-SOUND-134`..`SHOP-SOUND-139`, `ITEM-SOUND-183`, `VIDEO-SFX-099`, `VIDEO-SFX-100` (the shop);
`TOWN-547`..`TOWN-554`, `TAVERN-027`, `TAVERN-028`, `DIALOGUE-080` (the town outside the shop).
The site-by-site census, with the engine's status before and after, is the owner artifact
`review/story1437-town-sounds/census.md`.

## Builder

Kind: a town room's interface sound request. Builder: the room scene's sound table in the town
description (`pkg/town` `SoundSpec`), read by one request path, `Scene.sound`, through the host's
`PlaySound(SoundRequest)`.

- A slot names its sample, loop, restart (stop and rewind, then request), priority (128 when
  unnamed) and volume class (`"speech"` or effects). A tracked slot skips while it plays; an
  untracked slot requests every time.
- `triggers` bind a named room event to a slot, with an optional key and the slots stopped first.
  The game raises an event with `townScreen.roomSound(room, event)`; the description decides
  the sample. `Description.Validate` refuses a trigger on a missing or keyless slot.
- The first game's rows are in `pkg/game/towns/rom1.json`: 13 shop slots and 10 triggers, 6 new
  tavern slots and 6 triggers, 2 new school slots and 2 triggers. The school's teach sound moved
  from its own request onto the table, on the Rotate slot that the original shares with it.
- Registry sounds (click01, click00, ibook) keep their one builder, `playUISound`, at priority 220
  with no playing test.

## As built

- Shop: entry requests depart, enter, the InShop loop and start.wav at speech volume. A pause
  stops every shop voice; the scene's paint step requests InShop again on the first paint after the
  resume (focus loss, game menu, save, world map) and requests nothing while it plays. A gesture
  from a trade table cell requests Put_Off before the take; a gesture that puts units on the table
  requests Put_On after them; both restart. Button 0 and 3 presses request Undo and Out; the
  shelf and backpack arrows request Scroll on the press; a shelf tab that changes the shelf
  requests depart. Buy requests buy, or nofit with a short purse, nothing when empty; sell
  requests sell only for a nonzero value; the table clear requests Put_Off per element. The
  application no longer plays click01 on these controls.
- Tavern: a press on an occupied roster cell requests Helper (`DIV-3338`, Medium for both cell
  kinds); the Exit press requests Out; hire Add, a hire short of money nofit, dismissal NoAdd,
  talk and biography Talk.
- School: entry requests Enter; the Exit press requests Out.
- Tavern and school buttons: their releases no longer request click01 in the first game. They are
  not click01 buttons (`TOWN-548`); the upper and middle presses request nothing (`TAVERN-028`,
  Medium) and the school's index 0 nothing (`TOWN-551`, Medium). The second game's town has no
  tavern or school button panel on this path and is unchanged.
- Tip window: the Close press requests click01 in the town rooms and, through
  `MissionTipDress.CloseClick`, in a mission. A town dialogue's page button press requests
  click01. Both are first-game only: the second game's town implements no `TownClickSounds`.
- Unchanged: the square's sounds (already right), the shop item speech and its 30 percent draw,
  the tavern ambience and water loop, the school's skill and Rotate sounds, music and every
  mission sound but the tip Close. No simulation state, SAV byte or random draw changes: every
  request is presentation, and no new draw is taken.

## Proof

- `pkg/game/townroomsound_test.go`: the shop through the application's pointer (entry, drop,
  pick, buttons, trades, arrows, tabs) with parameters; a control that empties the shop's trigger
  table and finds no request; the tavern and school events; trigger validation.
- `pkg/game/townroomsound_test.go` also empties the tavern's and the school's trigger rows and
  finds no request from the roster press, a hire and the leave presses.
- `pkg/ui/townpresssound_test.go`: the town tip Close press, the mission tip Close press with
  and without `CloseClick`, the town dialogue page press with and without the first-game gate,
  and a school button release that clicks only without the first-game click model.
- Installed release witnesses, registered in `internal/gatedtests/testdata/population.txt`:
  `TestReleaseTownRoomsRequestTheCensusSounds` drives the installed EN and RU town through the
  App's pointer: the square's tip Close and the shop and school hovers; the shop's entry, tip,
  drop, pick, Undo, clear, short buy, buy, empty buy and sale, sale, arrows, shelf change and Out;
  the tavern's entry, tip, roster press, refused hire by the roster double click, hire,
  dismissal, biography and leave press; the school's entry, tip and leave press. Each request
  carries an installed sample and the census's priority, loop and volume class. Its control
  empties the shop's trigger rows and finds no shop request. 43 requests on EN and on RU.
  `TestReleaseMissionTipClosePressRequestsClick01` closes mission 10's tip.
- The second game: `TestReleaseSecondTownRequestsTheSameSounds` drives `rom2-ru` town 1 (a hover
  grid over the square, the shop with a buy, a sale and shelf changes, the tavern and its first
  talk). The same test run on base `e6cde80c` and on the candidate logs the same 10 requests
  (nine hover `town/shop/enter.wav`, one `inn-interior` `town/inn/helper.wav`), all at priority
  128; none is a first-game room sample or click01.
- Town traces re-recorded on EN and RU (`pkg/game/testdata/townroomtrace`,
  `townsquaretrace`). Every line keeps its line count, `save=` hash, screen, frame hash and
  message; only `snd=` fields moved. Room trace, per install: 13 lines; added the shop's entry
  four, Put_On, Undo, buy, depart three times, Helper, the school's Enter and Out, with their
  stops; removed five click01 releases. Square trace: 4 lines; added the shop's entry four and
  the school's Enter, with their stops.
- World hashes: every change is presentation; no simulation input, SAV field or random draw is
  touched. The traces' per-tick `save=` hashes are unchanged on both installs.

## Open debt

- `DIV-3339`: the shop's refused-release nofit, shelf keys, idle start.wav, the merchant frame
  sounds, Put_Off on the leave and Esc clears, and ibook on leaving.
- `DIV-3340`: the school yard's command and fight sounds.
- Not added for Unknown triggers: shelf slot `+0x50` (SHOP-SOUND-138), the tavern steppers
  (TAVERN-027).
- The statue's town menu buttons (TOWN-548) open the engine's game menu, whose buttons click on
  release.
