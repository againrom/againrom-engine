# Changelog

What changed in each release, in plain words for the people who use it. A
release lists every change since the previous release, including fixes that
reached `main` between releases. New entries go under Unreleased; a release
renames that heading to its version.

Each entry goes under one of three headings, in this order, and an empty
heading is left out: `### For players`, `### For map makers and modders`,
`### For developers`. An entry is one or two short sentences on one line: what
the reader now sees or can do, then, if it helps, what was wrong before. Use
the words a player sees in the game, not internal names: no routines, file
fields, record or row numbers, claim or divergence IDs. An entry starts with
the tag of what it changes: `[ROM1]` the first game, `[ROM2]` the second game,
`[BASE]` both games, `[Starter]` the starter. A change no player, map maker,
modder or developer can notice gets no entry.

## Unreleased

### For players

- [ROM1] [ROM2] An invisible unit stays visible to the spells of its own side and its allies: automatic casting keeps renewing Bless, Haste and the other support spells on it, and a spell cast at it from the book starts from the normal cast range instead of the mage walking up to it and waiting. Enemies still do not see it unless they can detect invisible units nearby.
- [ROM1] A Teleport to a neighbouring cell now shows the same short jump whether or not the caster had just walked; before, a caster that had walked first appeared to snap to the new cell without the jump.

### For map makers and modders

- [ROM1] Guest missions can form a chain with no town between, returning the waiting party to the town the first win opens; a save inside any link continues the chain. Its party can use one to eight characters from npc.reg or the mod, with chosen levels and equipment and human defaults for records with no party template; existing character edits keep their load order, while two mods adding the same new character or mission need load-after.

## 0.133.0

### For players

- [ROM1] A guarding monster group watches a circle as in the original: for a moment after the guard order it is four cells smaller, and after that it is one cell larger or smaller for some groups, its centre can sit one cell nearer the map's top-left corner, and fallen members still count towards it until their bodies are gone.
- [ROM1] Dragons, enemy mages and spellcasting monsters keep casting in a long melee: a spell they choose while a blow is under way is cast as soon as the blow ends, instead of being forgotten so they only bite after their first spells.
- [ROM1] At the start of the next mission, a hero who was still busy late in the last one heals at the slow rate until as much time has passed as in the original game, also after loading a town save; before, every hero switched to the fast rate a few moments into the mission.
- [ROM1] Monsters stop chasing a hero who has left their area, even one who keeps hitting them from outside it, and Darkness no longer sends its victims at the caster; a monster that cannot get back to its post gives up at once and tries again on its next turn, as in the original.

### For map makers and modders

- [ROM1] The object brush has a Forest mode: tick several kinds of trees or objects in the picture grid, pick a density or type a per cent, and one drag scatters a random mix on empty ground, never on water, roads, buildings or existing objects. Dragging with the right button thins the ticked kinds, and Undo takes back a whole drag.

## 0.132.0

### For players

- [ROM1] Dragons and enemy mages no longer get stuck repeating one spell, such as Darkness, that hurts nobody; they pick a new spell or attack each time they choose, so a dragon deals damage again. A Fire Ball or Poison Cloud they cast after walking closer now lands on you instead of the map's corner.
- [ROM1] A shop item no longer shows the same enchantment twice, such as Protection +2 and Protection +1: a repeated enchantment replaces the earlier one, and the price follows what the item holds.
- [ROM1] The starter's shop stock multiplier, 1 to 1000, now fills every shop shelf with that many times as many different items, each rolled on its own, instead of more copies of the same items; the default of 1 keeps the usual stock.
- [ROM1] Enemy mages now cast at an enemy their group sees within spell range, even beyond their own sight, instead of walking closer. Your mages keep their current sight rules.
- [ROM1] Buying a spell weapon straight onto a hero in town no longer blocks entry to missions. Saves made with such a weapon also load and enter missions.
- [ROM1] A save made in Againrom no longer crashes the original game when loaded from its in-game menu.
- [ROM1] Your mages casting an area spell on their own now choose an enemy whose blast spares them and their own or allied units. If no safe enemy is available, they wait or use another spell; casts and attacks you order still work as ordered.
- [ROM1] Monsters in a game saved by Againrom flee at low health again when that save is loaded in the original game.
- [ROM2] A new mage's Heal and Shield appear in the spell bar and spellbook and can be cast, including after loading a save.
- [ROM2] Clicking OK advances or closes town conversations, including Talk in the inn.
- [ROM1] Units at low health and units on Retreat look for enemies within 5 cells and count large units by their size, as in the original; flying enemies are not counted.
- [ROM1] A mage, an enemy one or yours walking into range of a spell you ordered, that can cast while stepping into a cell finishes the step first and then casts standing. It no longer freezes mid-step or shows its walk during the cast.
- [ROM1] The items a dead character still wore or carried when his body decayed, or when he left the map, are now kept in a save and are there again after loading it. They stay with the body and cannot be picked up.

### For developers

- [BASE] a SAV writes a creature's three spell slots and their use thresholds from the current game; an unused slot is written as zero instead of the loaded file's word
- [BASE] Both games build the town talk window through one `ui.TownTalk`; each game gives it its pages, art and a `ui.TownTalkProfile`.
- [ROM1] `World.Hash` digests the passive body contents (form PBC1) with every other World section; a World without passive bodies keeps its hash.

## 0.131.0

### For players

- [ROM1] A controlled creature saved while walking to cast a spell from its book now keeps that cast as a single cast when the original game loads the save.
- [ROM1] A right click after casting a spell from the spell book, or after a cast from F5 to F8, now drops the mage's selection, as in the original; before, the first right click only closed the book and the mage stayed selected.
- [ROM1] Idle mages that cast a spell on their own now aim at enemies on the map, such as a dragon in reach; before, they could keep casting at an empty spot where a unit had been taken off the map.
- [ROM1] A mage can put on a staff from the pack that the game had refused before, such as a staff of Stone Curse found in a city mission; it is worn with its spell, as in the original.

### For map makers and modders

- [BASE] Two mods may now change one companion join condition, one body sheet or one weapon's body when one loads after the other (`load-after`): the later wins and the check names what it overrides; without that order the launch is refused naming both mods, as before. A companion condition or a body sheet can be switched off with `off = true`, and a later companion condition may change only the keys it names. The mod guide lists every data file a mod can carry, its key and whether two mods may change one record.
- [ROM1] The map editor's Units and Structures tabs show a grid of pictures: a click on one opens its settings and each click on the map places a copy, or one per cell while the button is held. A left drag on the map draws a selection box for the open tab, moved objects show their outline until release, and the Groups window now lists each player's groups with their units and the triggers that use them; To group..., Remove from group, Ctrl+1..9 and 1..9 make and choose groups, and emptying a group a trigger uses asks first. A left drag no longer moves the view; use the middle button or the arrow keys.
- [BASE] A mod with weapon bodies can now be packed into one file: name its weapon bodies file `data/wpnbodies.toml`; the old long name still works in a mod folder, and a mod holding both is refused.
- [BASE] When a save will not load because it was made with other mods, the Load page now opens a window listing every difference, such as a missing or added mod, another version or a changed setting, with the differing values in red, and says how to start the game to load it; before, one line named the differences.
- [ROM1] The map editor opens a trigger as When, And if and Do rows chosen from a list in plain words (Hero enters area, Any unit of a player near a point, Give gold, Win the mission and more), with the unit, player, number or area set in the row, a unit picked or an area drawn on the map; the old view stays behind Details. The new Drop zones tool (D) adds, moves and deletes the places the party starts from, showing where the hero and the others land; with several zones the game picks one at random. A zone on a zero coordinate is refused, as the game would start the hero elsewhere, and a number the map cannot hold in 32 bits is refused in the row; Apply without an edit leaves a trigger as it was.
- [BASE] A mod can replace the game's pictures, sounds and texts by putting a file at the same path in its `files` folder, with separate `en` and `ru` folders for text; when two mods replace one file the one that loads later wins and the check says so. A save made with a mod that only replaces pictures and sounds loads without that mod; a mod that replaces a text is kept in the save, since texts feed the game's definitions.
- [BASE] The map editor's terrain brush has a Centre row that chooses how the middle of a large area is filled: Random (different every stroke, the default), the usual pattern, or one named variant of the terrain; the edges still join as before. Every editor window can now be resized by dragging its edge or corner, and its list, rows and fields follow its size.
- [ROM1] The mod check refuses, naming the file and the reason, a town description the game cannot play (it stopped on entering the town or left the party on the world map), a town name the font cannot draw, a picture, track or sound the install does not have, and an unknown key in mod.toml. A refusal of an unknown value now lists the values the game accepts. The mod guide now lists the mod.toml keys, the town description format with a working example, and what print does.
- [BASE] A mod can ship as one file: `againrom -pack-mod <mod folder> -pack-out <folder>` packs the folder into `mod-<id>.res`, which the game reads from the mods folder exactly as the folder, and a save made with one form loads with the other. The pack names any file whose name is longer than 15 characters or holds a character the file format cannot store, and writes over an existing file only with `-pack-replace`; a folder and a packed file of one mod side by side are refused naming both.
- [BASE] `againrom --headless <scenario.json>` is documented for modders in `docs/MODS.md`. From scenario version 10 a `capture` on a town screen names the town's ID, its name as drawn and its pictures, and a new `frame` step writes the current frame as a PNG (never into an install or over a file); an unknown key or argument is refused with the accepted ones listed.

## 0.130.0

### For map makers and modders

- [ROM1] In the map editor, a Show message action shows the whole message, and Cancel in New message no longer leaves an empty message behind.
- [ROM1] The map editor's unit settings are grouped into Health and mana, Attributes, Defence, Weapon skills, Protections and Equipment. Leave a field empty to keep the game's value, shown greyed, or type a value to change it; Inherit takes the change back.
- [ROM1] In the map editor, a unit given Health 0 starts the mission fallen and is drawn fallen on the map.
- [ROM1] Every window of the map editor, the Add window and the map name among them, moves by its title bar.
- [ROM1] In the map editor, painted terrain blends into its neighbours with no hard square edges, and where two terrains have no blend between them the brush puts one in between (Land between Sand and Stones). A stroke changes terrain no more than 4 cells from the brush; where three terrains meet there, one hard corner may remain.
- [ROM1] The map editor's brush list shows the brush size with - and + buttons; Alt with the mouse wheel, or [ and ], change it from 0 to 16. Every brush is a Circle or a Square, and its outline shows exactly the cells a stroke will paint.
- [ROM1] The map editor shades the map's impassable border with a line at the playable edge; View > Map border hides it.
- [ROM1] A mod can add, rename, change or switch off towns in `data/towns.toml`; a new town has its own pictures, place on the world map, opening mission and offers. When two mods change the same town, the later wins only if it loads after the other with `load-after`. A town name may be written in Russian; a town that could never open is refused at launch.
- [BASE] `againrom -dump-base <folder>` writes the installed game's towns as a ready mod folder, every town with every key at the game's own value, to copy and change; it never writes into the game folder, a folder that is not empty, or over a file.

### For developers

- [BASE] Each game now lists its towns in one town catalog, one entry per town naming its town screen, room hooks, offer source and world map point, so a later mod can add a town in the same format.
- [BASE] A town catalog entry can now name its town, the mission that opens it and the missions it offers, and the first game's world map shows every open town, so the party can travel between towns and return from a mission to the town it left.

## 0.129.0

### For players

- [ROM1] A male mage made in the character generator now starts with the Uncommon Wood Staff and holds it in the generator, as in the original. Before, he got the woman's staff with the large crystal ball.
- [BASE] Hovering over an item on the character doll now shows its description in the character generator, the tavern and the school, as it already did in the inventory and the shop.
- [ROM1] Units in a group move that cannot reach their spot now stop and fight only what comes within reach, as in the original. Units held up only by other units keep trying, and stuck units in a swarm or roam order now attack an enemy next to them.
- [BASE] A save made in Againrom from an original save now keeps the party's place on the world map and the next mission, so the original game continues it exactly as it would the original file.
- [ROM1] When the original game saves over an Againrom save, heroes keep their full pack weight after loading it again. Before, some items counted as weightless, so an overloaded hero walked too fast.
- [ROM1] Winning a mission in an old save whose campaign sends the party back to its current main mission now does what the original does: the party returns to that mission's map point and plays it again. Before, the win counted and the campaign moved on. Campaigns started in Againrom are unchanged.

### For developers

- [BASE] The README explains how to build and run the game on Linux.

## 0.128.0

- [BASE] Message lines now say who they are about. A skill-raise line ends with
  the character's name in parentheses, as in `Blade skill improved: 11 (Gord)`,
  and a pickup line names the character who picked the item up:
  `Picked up Wood Short Bow (Gord)`, `Picked up Bronze Long Sword (now 2
  pieces, Gord)`, `Picked up 78 gold (Gord)`. An item a mission gives you or
  `#create` makes now shows `Received Cure for Plague (Gord)`, with the count
  before the name when there is more than one; before, it showed nothing. Gold
  a mission gives or `#create` makes shows `Received 10000 gold`, with no name,
  because it goes to the shared purse. The Russian game uses `Получено:`. The
  name is the one the character's card shows: your hero shows your name.
- [ROM1] A weapon's description now lists what the original lists, in its
  order: the damage, the attack and defence bonuses when they are above zero,
  and the range of every weapon whose range is above 1, bows and crossbows
  included. A weapon that fighters may not wield, such as the Staff, lists
  none of these; the Sonic Beam, Flame Thrower and Boulder Thrower keep their
  range line. The character creation screen shows the damage of the first
  spell a mage's weapon casts.
- [ROM2] A weapon's description follows the same rule as the first game's: the
  damage, the attack and defence bonuses when above zero and the range when
  above 1; the Staff and the Shaman Staff list none of these.
- [ROM1] A mage whose staff lost its spell, as in a game loaded from a save
  that holds the staff without it, now makes its ordinary ranged attack (the
  shot looks like Fire Arrows) instead of casting the staff's spell, as in the
  original. Take the staff off and put it on again and the mage casts its
  spell. A staff taken off, dropped or left by a fallen mage loses its spell
  until it is worn again.
- [ROM1] A unit that holds an enemy as its target but neither walks nor
  strikes, such as a mage without a staff whose way to the enemy is refused,
  now stands as an idle unit does; before, it was drawn making a swing.
- [ROM1] Dragons and Daemons cast spells again, as in the original: each
  Dragon knows the spells of its kind (Fire Ball, Wall of Fire, Lightning and
  more for the stronger ones) and has the matching magic skills. This takes
  effect when a mission starts; a dragon in a save made earlier keeps its
  empty book until the mission is started again.
- [ROM1] Gold a mission script hands you, such as the farmer's 10000 for
  bringing his son home in Dragon Clan, now shows "Picked up 10000 gold" on
  the message line, as gold picked up from the ground does. So does gold from
  the `#create N Gold` cheat.
- [ROM1] A spell weapon's description now reads as in the original: it names
  the spell, its damage, the weapon's own range (5 for a Staff, 4 for a Shaman
  Staff) rather than the spell's, and the spell's duration, and it is the same
  whether the weapon is worn, carried, on a shop shelf or on the ground. A
  mage's card shows one damage row, the damage of the spell its weapon casts,
  and no attack row; other units show their own damage and attack. A staff that
  lost its spell no longer gets it back when a script gives its owner to you,
  when a guest joins in town, when the party starts the next mission or when
  the town is saved; take it off and put it on again.
- [BASE] With a spell picked in the spell book, closing the book (Space, B,
  or the book button) or choosing the Attack order puts the cursor back to
  normal, and a click on an enemy now attacks it instead of casting the spell.
- [BASE] The game now builds for Linux without a C compiler.
- [BASE] The starter's "In a window" option opens the game in a window instead
  of over the whole screen. Alt+Enter switches between the window and the whole
  screen at any time.
- [BASE] Hold the middle mouse button (the wheel) and move the mouse to drag
  the mission map.
- [ROM1] Map editor: a Brushes menu on the top bar turns on the terrain, height,
  objects and passability brushes. The terrain list shows a picture of each terrain.
  View > Passability shades open cells white and blocked cells red, and the status
  line names what blocks the cell under the pointer. The passability brush (Q)
  blocks and unblocks cells; the height brush shows the height under the pointer.
- [ROM1] Map editor: Shift and a drag on the map selects every unit, structure and
  sack in a rectangle; Shift and a click adds or removes one. Dragging one of them
  moves them all, Del deletes them, Ctrl+C copies them and Ctrl+V pastes them at the
  pointer, also into another map; each is one undo step, and the status line names
  each record that cannot stand where it would land. Flying units may land over
  water and mountains; a unit pasted into another map takes a group number no
  unit or script of that map uses. On the Russian install the brush lists, the
  brush help and the F1 key list are in Russian, and the F1 list shows each key
  line.
- [BASE] The game finds its files whatever the case of their names, so an
  install copied to a case-sensitive system works without renaming.
- [ROM1] Arrows and homing spells aim at the centres of large creatures.
  A suppressed automatic attack keeps the unit facing its previous direction.
- [ROM1] The school yard sounds its commands and looping practice fights at timed intervals.
- [ROM1] Spell effects last for their full duration after each hit:
  Freezing Cloud's Slow and Light and Darkness sight changes.
- [ROM1] A spell whose casting has finished is cast even when its target has stepped
  out of range or out of sight meanwhile; creatures no longer break it off to walk
  after the target. Mana is not charged again.
- [ROM1] A mage whose staff casts a spell attacks from the staff's range, 5 cells with a
  Staff and 4 with a Shaman Staff, not from the spell's range; the same holds for any
  weapon that casts. Spells cast from the book keep their own range.

## 0.127.0

- [ROM1] [ROM2] Damage numbers combine hits on the same unit in one game tick;
  hits in different ticks keep separate numbers.
- [ROM1] The world-map flag stands above your current position. Arrival waits
  for the destination cross to finish; clicking outside the scrolls skips the trip.

- [ROM1] Usable wells, switches and graves show the use cursor and their
  readout. Clicking walks the character to them; wells restore health or mana
  while charged, and switches toggle. Charges and switch state survive SAVE/LOAD.
- [ROM1] Saving in town keeps each hero's full experience when the game is loaded again.
- [ROM1] Map editor lists stop at their ends and their scroll bars drag.
  Add starts ready to place, with a translucent preview at the cursor.
  Unit settings explain stored values and show the game's character card.
  Cancel closes the window and discards its pending changes.
- [ROM1] A spell cast at a unit or a cell out of reach is no longer lost: the
  caster walks until the target is in reach and casts there. Any other order
  cancels it. An enemy mage now casts at the unit it is fighting.
- [ROM1] Map script spell and effect parameters pick spells by number and name,
  using the installed game's language. Unknown values keep their number.
- [ROM1] The map editor's G panel opens each arm's script and the drop cell's
  action. Back returns to the selected row; Enter and D keep their target picks.
- [ROM1] In the map editor, Show message picks a page by its first line.
  Open edits its text and speaker; New message creates a page that travels
  with Save As and mod export. Undo restores the page and its node value.
- [ROM2] The global map replaces the list of destinations. Leaving the first
  town shows the road to the first mission and the party walks it by itself;
  leaving any other town, or winning a mission, opens the map with every place
  you can go marked by a flag. Point at a flag to see the place's name, its
  description and the road there; click to set off. A click or a key during
  the walk skips to its end, and the party enters the place it reached. The
  map has no cancel, as in the original.
- [ROM2] After the first mission the party goes straight on to the next one,
  as in the original, because no town is open yet.
- [ROM2] What the hero buys in a town shop is kept in the hero's own bag. Saves
  from earlier versions keep the hero's shop items.
- [ROM2] Armour and shields a mod puts in the shops are on sale in the town
  shops too.
- [ROM2] The town shops use the same shop screen as the first game: item
  pictures and prices on the shelf, a table for the deal and the hero's bag
  below it. Click a category in the room to show it, click or drag an item to
  put one unit on the table, and shift-click to move the whole stack. Point at
  an item to read it. The mouse wheel over the table scrolls it when the deal
  holds more than five places. Books and enchanted items each take their own
  place. A purchase that would overfill the hero's bag is refused. The hero's
  portrait panel is not shown yet.
- [ROM1] Map editor unit, structure, group and player popups list the scripts
  that use them. Open a listed trigger and return with Back; unapplied object
  fields stay intact.
- [ROM1] Map message pages now stay beside a saved map, with all source languages
  and shared pages preserved by Save As and mod export. The editor loads them
  for Pages and Test play, and reports missing pages for the map or mod by name.
- [ROM1] The map editor's Scripts entry and F2 open one list of triggers,
  variables, and conditions and actions not attached to a trigger. Add
  opens the script record wizard. Each used variable opens a read-only list
  of the triggers that read or change it, with links to those triggers.
- [ROM1] In the map editor, script parameters choose units, groups, players,
  structures, variables and items by name. Open follows a parameter to its
  record, and Back returns through the opened records. Item rows show the
  parameter number beside the name. New variable skips every condition slot.
- [ROM1][ROM2] The character card shows experience (XP) only for your
  heroes and for the characters who can join you as one, such as Brian and
  Naira, also before they join. Other people, hired mercenaries and
  creatures, who gain no experience, no longer show an XP line.
- [ROM1] The map editor opens each map on a Map panel, also reachable from
  the Map menu. Its lines state the size, name, description, players and
  sides, mode, start point, victory, defeat, report and test play, and a
  click on a line opens the place that sets it. A map with no start point,
  or with nothing that wins it, is marked on the panel and in the report,
  and Test play and Save into a mod first name the problem in one notice
  you can continue past; neither refuses a save. A new map asks whether it
  is single-player or multiplayer, single-player by default, and starts the
  party at the map's centre. The map's metadata (M) sets the mode by name.
- [ROM1] In the map editor a selected trigger's areas carry handles: drag
  a box's corner or edge, a square's rim or a cell to change the area its
  condition or action names, one Undo step each. When another trigger or
  pair uses the same condition, the editor names them first and you can
  cancel. L draws a box on the map and makes it a condition "the hero
  stands here" for a new trigger or one you pick from a list; on a
  multiplayer map, where scripts cannot name a hero, L says so and draws
  nothing.

## 0.126.0

- [ROM1] Units placed on a map now keep the strengths, skills and
  resistances the map gives them, as in the original. In the map editor a
  unit's popup edits each of these values beside the value its template
  gives, and clearing a value returns the unit to its template.
- [ROM1] The map editor no longer refuses to close while the map has unsaved
  changes. Closing it (the Exit entry, Esc, or the window's close button) or
  opening another map asks one question naming the map: Save As, Close
  without saving (Open without saving when opening), or Cancel. Save As
  closes or opens only once the map is saved; Cancel and Esc return to the
  editor unchanged. Pressing the window's close button again closes at once.
  With no unsaved changes nothing is asked.
- [ROM1] The map editor adds units, structures, sacks, groups, players and
  triggers through one step-by-step Add window. Every list has an Add
  button, and the old keys open the same window. Each step offers only valid
  choices: a unit's class with its picture, its tier when the creature has
  several, its owner by name and its cell, picked on the map or typed. A new
  group is offered the lowest number no unit of the map carries, so a script
  never reaches the wrong group. A refused value says why. Cancel at any
  step adds nothing, one Undo removes the new record, and the new record's
  popup opens when the window finishes.
- [ROM1] A mage of yours without a staff that cannot get where it was going
  keeps the enemy within its reach as its target without striking it, as in
  the original; on its way to a sack it gives the sack up. Defending a unit
  while standing right beside it, with an enemy next to the mage and none
  threatening that unit, it steps back to its defending distance.
- [ROM1] A mage of yours without a staff no longer goes into melee on its
  own, as in the original. Defending your hero it stays beside him, heals him
  and casts an armed spell at his attacker; under Stand Ground, Guard or Swarm
  it does not strike an enemy beside it. Your own attack order still sends it.
- [ROM1] A unit in a group move or attack move that cannot get through no
  longer stands still while enemies hit it, as in the original. When no way
  exists at all it stops and fights what is within reach; when its way is
  only blocked by others it attacks an enemy within reach the second time it
  is blocked. Saves made in between keep this.
- [ROM1] The map editor picks a creature's tier from a drop-down, also when
  placing a new unit, and draws each creature in its tier's colours. A unit's
  popup shows its characteristics as the game builds them at Normal
  difficulty and what its corpse drops; hovering a unit names it with its
  tier and health.
- [ROM1] A mage of yours heals itself and its wounded companions while an
  enemy shoots it from outside the mage's own reach, as in the original; an
  enemy standing within its reach still stops it healing unless you set Heal
  to cast automatically. The automatic casting level (Minimum, Average,
  Maximum) follows the same rule.
- [Starter] On macOS, Browse opens the system folder chooser and adds the
  chosen folder. The map editor's Open and Save As use the same system
  chooser; Open shows every file and has no file-type menu.
- [Starter] The starter's settings (sound, volume, movies, markers, saves
  folder, mission, skill, extra arguments and the starter's own options) moved
  behind a new Settings button, so the main window shows only the base game,
  mods and mod settings. Close or Esc returns to the main window; the values
  and the saved file are unchanged.
- [ROM1] In the map editor on sloped ground, the cross of a selected unit or
  structure, the cross on a linked footprint and the centre cross of a square
  trigger area now sit where the game draws that cell's path line.
- [BASE] Each release offers only the three game ZIPs, without separate
  .sha256 checksum files.
- [ROM1] A guard walking back to its post or a unit closing on the ally it
  escorts now attacks an enemy within reach when its path is blocked, instead
  of standing still.
- [ROM1] A unit in a group move whose path was blocked tries again at the
  group's next turn instead of staying where it stopped.
- [ROM1] A unit sent to pick up a sack that someone else takes first still
  walks to the sack's place and finishes there as after a pickup, ready to
  attack an enemy in reach, as in the original.
- [ROM1] A unit ordered to retreat whose way is blocked now attacks an enemy
  within reach instead of standing still, as in the original.
- [ROM1] In the map editor a trigger or script node named in Cyrillic shows
  its name in the record list, the trigger popup and the map's area labels,
  keeps it after saving, and can be edited again. Drop-down lists show
  Cyrillic player and item names instead of question marks.

## 0.125.0

- [ROM1] The map editor opens through a top bar and popups: pick a map from a
  scrollable list, open units, buildings, groups, sacks, players and triggers
  from the bar, and edit every field of a selected object in one window with
  drop-downs, Apply and Cancel, in English or Russian.
- [ROM1] The map editor's trigger window reads as the rule: its conditions,
  the comparison in words and its actions by name. Edit each line with a
  drop-down, a number or a click on the map, add and remove conditions and
  actions, and create or delete triggers. The places triggers watch are
  outlined on the map in each trigger's colour with its number and name; a
  click on one opens its trigger, and View hides them.
- [ROM1] A unit on Hold Position that is ordered to attack leaves Hold, as in
  the original.
- [ROM1] Original mission saves that would not load now load: a unit
  between two attacks, and bodies that still carry a spell effect.
- [ROM1] A mage no longer uses up a book of a spell he already knows; the book
  stays where it was, whether he tries it from the inventory, the shop
  shelf or the merchant's table.
- [ROM2] A mage no longer uses up a book of a spell he already knows in a
  mission inventory.
- [ROM1] A unit larger than one cell that flees or is ordered to Retreat
  chooses where to run from the same spot as in the original.
- [ROM1] The party reveals the map as far around it as in the original, and
  loading a save shows what the party sees at once.
- [ROM1] The map editor stores a Follow group command in the layout the
  original maps carry, and a parameter left unchosen after an operation
  change is refused in the editor's own language, naming the field.

## 0.124.0

- [ROM1] Heal and Drain marks continue after loading, follow moving targets,
  and fade independently of the projectile that created them.

- [ROM1] Earthquake's cell sound follows its animation and still plays after
  loading a save made before the sound.
- [ROM1] Author mission conditions, ordered actions and triggers, then test
  the saved mod from the editor through victory and return to town.
- [ROM1] Shots and spell effects continue their saved record numbering when
  the same game session enters another mission or starts a new game.
- [ROM1] Place buildings, assign unit groups and starting inventories, create
  enchanted loot, and clone units with independent items.
- [ROM1] Create rectangular maps, edit mission text and metadata, save
  playable mod packages, and read every editor command from F1.
- [ROM1] Shop Buy and Sell buttons show totals on a centered second line.
- [ROM1] Shots and spell flights keep their release timing and current state
  across saving and loading, including a save made before a shot is released.

- [ROM1] Saving keeps a dead character's current maximum health instead of rebuilding it from the map's difficulty. The value also survives decay and repeated SAVE/LOAD.
- [ROM1] Scroll descriptions cover every ordinary scroll spell, including Darkness and Curse, with the spell's fields and effect caption at the scroll's own power. The spellbook keeps its existing descriptions.
- [ROM1] A scroll's description shows the spell's range and other characteristics at the scroll's own power. Newly created scrolls have the correct value, including the Protection from Fire scroll on mission 81, which sells for 2048. Prices already stored in saves and shop stock are kept. Sold enchanted weapons, shields and armour go to the Magic Items shelf.
- [BASE] Dragging onto a matching backpack stack combines the items in missions and the town shop. A normal drag takes one item; Shift takes the whole stack. The same quantities apply to floor drops in missions. Taking a whole stack also lets it trade places with a different item.
- [BASE] Loading and restoring saves does less repeated decoding and copying of the saved world.
- [ROM1] A staff a hero wears is saved with its spell, whether the hero put it on during a mission, wore it from the start or wore it when joining, so in the original game the hero's attacks cast the staff's spell instead of a plain shot that looks like a Fire Arrow. A save that already lost a worn staff's spell gets it back when it is loaded, without putting the staff on again.
- [ROM1] A save now also writes its difficulty in the campaign record, where the original's own saves keep it; the original played an Easy or Hard save from earlier versions at Normal. A save made by an earlier version keeps loading at its saved difficulty here; save it once to correct it for the original.
- [ROM1] Saves made while a hero casts from a spellbook or charges a staff load in the original game without crashing it. A loaded healer finishes recovering before the next cast. After a weapon swap or removal, the save carries the damage of the weapon now worn, and a hero's own weight counts what the hero wears, so removing a shield in the original no longer makes it negative.
- [ROM1] An item a hero picks up, loots, is handed, unequips or is granted joins an identical item already in the pack, whatever form each was stored in. A spear bought in the city now joins spears looted in a mission, and a spear dropped and picked up again joins the spears already carried. A save that already holds two such cells keeps them on loading; dropping one and picking it up joins them.
- [ROM1] Test Play in the map editor starts sooner after F5: the test mod is written with fewer disk checks.
- [ROM1] A potion drunk in the town is used up and changes nothing, as in the original; a regeneration potion drunk there no longer carries into the next mission. A save made with such a potion by an earlier version loads, and the potion ends when the next mission starts.

## 0.123.0

- [ROM1] The town plays the original's sounds in more places. In the shop: a thud when an item goes onto the trade table and another when it is taken off, the purchase, the refused purchase and the sale, Undo and Out on the button press, the scroll arrows, a shelf change, and the shop's greeting and background on entry. In the tavern: a click on a mercenary or speaker, hiring, a hire you cannot pay for, dismissal, opening a talk or a biography, and the Exit press. In the school: entry and the Exit press. The tavern's and the school's buttons no longer click when released, as in the original. Closing a tip window with its Close button clicks, in town and in a mission, and so does the page button of a town dialogue.
- [ROM1] The shopkeeper speaks about an item you move onto the trade table only some of the time, about three placements in ten, as in the original, not on every placement.
- [ROM1] A spell is aimed only at ground your party sees now, as in the original. Over explored ground that is dark now the cursor is the move cursor and a click moves the selected units; the chosen spell stays ready. Teleport is aimed the same way. Mission 80's grave is reached by walking to the river bank, seeing the grave's island and Teleporting onto it.
- [ROM1] Animal and monster cards head their weapon-kind values Resistance, as the original does.
- [ROM1] The map editor plays the map you are editing: press F5 and the game opens it, unsaved edits included, after character generation. The editor stays open, so you can close the game and go on editing. Your saved map and mods are not touched; the map is written to a test mod beside the editor. A map whose report shows an error is not played, and a missing game program is named on the editor's message line.
- [ROM1] [ROM2] Teleport across several cells draws the caster at the destination at once, without sliding or walking through the intervening cells.
- [ROM1] A Teleport from a spellbook shows its two flashes and starts the caster's casting motion the moment it is cast, once the caster has turned to face the destination, and the caster jumps when the cast completes, as in the original, in whatever direction the destination lies. A save made in that moment keeps the cast: the caster jumps on the same tick after loading. This includes a save the original game wrote during a Teleport to a cell. A Teleport from a scroll shows its first flash where the caster stood, not at the destination.
- [Starter] Browse beside Add path opens the system folder chooser on Windows and macOS. If the chooser is unavailable, the path field remains available.
- [ROM1] The map editor's Open and Save As use the system file dialogs on Windows and macOS, starting at the current map's folder or the base game folder. Cancelling keeps the map and edits; if a dialog is unavailable, the editor shows the path prompt.
- [ROM1] Saves made in a fight load in the original game: an attack whose victim has died no longer leaves an attack order without a target.
- [ROM1] [ROM2] While the game is paused, a changed autocast setting (right-click on a spell cell, Ctrl+A) shows its dashed frame at once, and a second change starts from the shown setting. The setting takes effect on the first tick after you resume.

- [ROM1] The map editor no longer crashes when the map report is open while a brush is on and the report lists more rows than the brush's list has.
- [ROM1] The map editor paints objects. Press B, choose a class from the list (trees, stones, fences, bones and the rest the game draws; the first row clears) and drag across the map: every cell of the brush takes that object, shown as the game draws it, and one stroke is one undo step. Save, Save As and save into a mod write the objects, and the game plays the map with them in place, blocking movement as the installed objects do.
- [ROM1] Map authors can add and edit players, their names, colours, human or computer control and directed relations. Deleting an unused player preserves later ownership and script references; each edit has one undo step and survives saving into a mod.
- [ROM1] A creature you own, including one summoned by a cheat, casts from its own spell slots. On a map with player colours in its roster, each unit uses its player's roster colour.
- [ROM1] The fog opens as your units move, as in the original. Ground a walking unit steps onto is lit at once instead of up to two seconds later, so a mage with automatic casting no longer attacks an enemy that is still drawn under black or fog. Ground a unit has left still stays lit until the fog's next full redraw, as before.
- [ROM1] A spell, book or scroll, aimed at a wide creature now reaches it across the gap between the two bodies, as the original measures it, so a mage can cast at a large creature from a cell farther than before. A Shield-type spell of range 0 no longer reaches another unit standing in the caster's own cell.
- [ROM1] A save no longer keeps the ghosts, bees, bats and dragons that have died: the original game no longer replays their deaths when it loads the save. Saves written earlier with such creatures load as before and are written without them from the next save.
- [ROM1] A map the editor saves matches what the original editor writes. A new map starts with the original's two players, Self and Enemy, hostile to each other; a unit saved with an owner the map has no player for becomes player 1's, and a unit's sack follows the unit's position in the saved file. Deleting a sack no longer makes other sacks on the ground vanish when the map is saved.
- [ROM1] A save made in the town now groups the party as the original does. Each companion who joins between missions is written in a group of his or her own, each hired type in a group of its own (the Catapult and the Ballista included), and a group whose members all left without a refund is written empty instead of being dropped. Returning a hired squad for a refund removes that hire's empty group, including after SAVE and LOAD.

## 0.122.0

- [Starter] Every caption of the starter window comes from one table in English and Russian. With a Russian base the buttons, labels, headings, status and error lines read in Russian, using the game's own Russian words where the install has one (Delete, On, Off, Skills, Saved Games and others). Mod setting labels stay the mod's own.
- [ROM1] In the tavern and the town, a companion who joined the party (Naira with a female hero, the companion of mission 100 with a male mage) now answers his or her own lines with the right voice recording and statistics. Before, Brian answered for them.
- [ROM1] Mod authors write a mission's texts in UTF-8: event and tip pages, the world map's briefmap.txt, the title and the briefing. The game encodes them for the base it runs on, so one mod folder shows Russian on the Russian base and English on the English one. A text file written in the base's own bytes still works. `modmission` copies an installed mission's texts out as UTF-8, and the map editor warns before it saves into a mod when the map's script sends a message the mod gives no text.
- [ROM1] The map editor's brush keeps a cell the map's author closed closed: a terrain stroke beside such a cell no longer makes it walkable. The map check also names more cases for a mod author: a drop-cell table that mixes cells with a zero coordinate and valid cells (the party may land on a random cell), and a unit placed in the border ring, which the game removes when the mission loads.
- [ROM1] The map editor gives a map a win and a loss. In the script panel (G) a map that has none gets rows that add a win arm, won when a chosen unit dies, a chosen group is wiped out or a chosen structure is destroyed, and a lose arm, lost when a chosen unit of the party's side (roster slot 1) dies. Enter on a row, then a click on the unit, a unit of the group or the structure adds the arm as one step that Undo takes back. A new map saved into a mod, with a drop cell, a unit of each side and the two arms, is won by killing the enemy and lost when the chosen unit dies.
- [BASE] No unit fires on the move. A mage on automatic casting walks to where it was sent and casts only once it stands there; after any spell a unit stands until the spell's recovery ends, then carries on with its orders. Units sent to attack-move stop, turn and fight what they meet, then walk on.
- [ROM1] Mod authors can type typographic quotes, dashes and the ellipsis in a mission's texts, title and briefing: the game writes their plain forms (`"`, `'`, `-`, `...`) instead of dropping them. The map check now names a text file with a character the running base cannot show, and the editor's report follows a page added to the mod's texts folder while the map is open.
- [ROM1] The game checks the map of every mod mission, guest and replacing, with the map editor's rules when the mods load. A map the editor reports as an error is refused, naming the mod, the file, the line and the rule; a map with only warnings loads and the warnings print on the launcher's standard error, which the starter's check shows. Mod authors: run the starter's check after saving a map into a mod.

## 0.121.0

- [ROM1] A freshly started mission follows the same cell rules as a loaded save. Creatures cross building doorways and bridges at the original's speed for those cells from the start, and no creature at the start of a mission steps onto a cell that would stop it after loading.
- [ROM1] A mod that changes the terrain movement costs (map.reg) now changes how creatures move on every mission.
- [BASE] The `#pickup all` cheat picks up every sack as an ordinary pickup: the message line names the gold and each item picked up, quest documents go to the main hero, and a mission event that waits for a sack to be picked up follows as it does after a normal pickup.
- [BASE] Fire Wall sounds: its fire crackles from the moment the wall rises until it burns out, even when the view stays still. Before, the sound waited for the view to move or for the next bird call.
- [ROM1] In the tavern, the speaker who offers the mission for the repaired catapult speaks both lines aloud. A tavern speaker with no recording for the hero's sex and class plays the conversation's own recording instead of staying silent.
- [ROM1] The shopkeeper says one line about an item you take from his shelf instead of running through all of its lines one after another. A magic item gets its effect line, a book or wand its spell line, and other goods one of their own lines, picked anew each time.
- [ROM1] A game saved after playing on from an original save loads back exactly as it was left: a monster holding an attack keeps it, a monster that moved and died stays where it fell, and heroes sent away from their group keep their new orders.
- [BASE] A step from one cell to the next completes before a new order acts. Giving a spell order to a walking hero, such as Magic Shield on the hero or a spell at a target, no longer throws the figure to the centre of the cell it was entering: the hero finishes the step, then casts.
- [ROM1] The map editor checks the map before it saves. Save As, and Ctrl+S into a mod, first list what is wrong: each line names the record and the rule it breaks, and clicking a line selects the record and scrolls the map to it. A map the game could not build or play, such as a unit outside the map, a script that names a unit or group the map no longer holds, or an owner the map's sides do not list, is not saved and the list stays on screen. Things that may only play differently, such as a unit on ground it cannot enter, two structures over the same cells or two sacks on one cell, are listed and the map is saved. R opens the same list at any time; `mapedit -check` prints it.
- [ROM1] The map editor moves the drop cell and sets what wins and loses a mission. D then a click on the map puts the party's drop cell there (a map without one gets one); G lists the win, lose and protect arms of the map's script, and Enter then a click on a unit or structure points an arm at it, so a different creature must die to win, or adds a protect check that loses the mission when a chosen unit dies. Every change undoes and redoes, the checks of the report apply to it, and a saved map plays as a mod mission with the new drop cell and the new targets.
- [ROM1] The map editor paints terrain and heights with a brush. T paints the chosen terrain (grass, land, sand, cracked ground, flowers, rocky ground, savanna, high rocks, water, road) over a round area of the map, H raises or lowers the ground, and [ and ] set the brush size. Painted cells get the transitions toward their neighbours, a drag is one undo step, and a saved map plays with the painted ground's movement costs and heights: creatures walk around painted mountain and across painted grass.
- [ROM1] The map editor creates a new map. N lists the sizes above the installed maps (32 to 256 cells a side); the new map opens as any other, U then a click and a class number puts a unit on it, and Save As or Ctrl+S into a mod writes it. `mapedit -new 48x64 -out map.alm` writes the blank map. The report names a map with no drop cell whose fallback landing cells leave the map.

## 0.120.0

- [BASE] Once a spell's mana is spent, the cast completes. Clicking the same spell again while it is being cast spends nothing and does not start the cast over. Choosing another spell or target during the casting no longer cancels it: the new cast follows when the first one is released, and each spends its mana once, when its casting begins. A move or attack ordered during the casting also waits for it.
- [BASE] The in-game menu opens at once, however many saved games the folder holds, and the Load list opens quickly the second time. Both games read each saved game once and remember it until the file changes.
- [ROM1] Your companions and your hero speak mission lines in their own sex. Where a mission has a man's and a woman's version of a line, as in the New Catapult mission's argument about which way to go, the version shown now matches the person speaking, with that person's portrait. A companion's line is shown with that companion's own portrait, not with another hero of the same sex.
- [ROM1] The map editor comes in the release archives for Windows and both Macs, beside the game and the starter. The starter's new Map editor button (Редактор карт on a Russian install) opens it on the selected first-game install, with the install's map list. In the editor you open a map, inspect its records, drag units to new cells, clone units (C), delete them (Delete, or fn+Delete on a Mac keyboard) and change their fields (E), undo and redo, and save the map as a copy outside the install.
- [ROM1] A mod can replace the map of a campaign mission: when the campaign reaches that mission, it is played on the mod's map, with the mod's title, its briefing (the text of the objectives panel) and its texts where it gives them; the world-map briefing and every other text file the mod does not give stay the install's, while the rest of the campaign stays as it was. Saves made in that mission load only with the mod. The map editor started with `-mod <mod folder>` saves the edited map straight into the mod with Ctrl+S, at the path the mod's mission entry names.
- [ROM1] In the map editor a structure or a sack can be moved, cloned, deleted and changed, as units already could, and a sack can be added. A sack takes its items and gold, and a unit's stock its items; each change is one Undo step.

## 0.119.0

- [ROM2] Spells look and sound like the second game's own. A missile, the cells an area spell strikes and the sound of each come from the second game's pictures and sounds instead of those of the first-game spell with the same number; a spell the second game draws without a picture shows none, and one without a sound stays silent. Effects last, flicker and light the ground around them by the second game's tables where those are known.
- [ROM2] Blizzard's ice pieces fall from above onto each struck cell and then shatter there, as in the original.
- [ROM2] Item descriptions and spell lists name every spell from the second game's own text, Slow included, and the spellbook names the spells it holds the same way. In the Russian version Prismatic Spray keeps its own name instead of the first game's name for Drain Life.
- [ROM2] In the inns, the companions ready to join you sit first, before the people you can talk to. Click a companion or press Hire: they agree or refuse in their own words, and a hired companion's seat is marked. Click the companion again or press Fire to let them go. Saving and loading keeps who is hired. After a mission a hired companion rests and refuses to be hired for the next one. Hired companions do not yet follow the hero into a mission.
- [ROM2] Clicking a person in an inn starts the conversation at once instead of only choosing the seat.
- [ROM2] A conversation in the druids' inn that changed nothing now does what it does in the original: a companion becomes ready to hire there, the conversation is not offered again, and a later mission in that inn becomes available.

## 0.118.0

- [ROM2] The shops of the second and third towns open from their squares and trade like the first town's shop, with their own pictures and stock. Saving and loading keeps the gold and the items bought; loading a save made in a town fills the shops anew, as in the original. In the Kaarg shop the fire flares up while weapons are chosen and dies down again.
- [ROM2] The druids' shop shows its fourth group from the seventieth mission on, and the conversation in the druids' inn fills it with elven goods; before the conversation the group is empty. A shop's goods are also drawn again after a mission or story step changes what that town sells.
- [ROM2] The shopkeepers move: after a few idle seconds they play a gesture, and they answer a purchase or a sale. In the first town the keeper also nods when you change the group and shakes his head when you cannot afford a purchase; the Kaarg keeper speaks.
- [ROM1] At the start of a mission every creature and character placed on the map faces the way the original turns it, each its own way, instead of all facing the same direction.
- [ROM1] A save made during a mission stores more of each creature's movement data as the original holds it at that moment, so the original game and this game go on from such a save more alike.
- [ROM2] The main menu's buttons do what they do in the second game: New Game, Load Game, Cut Scenes, Credits, Hall of Fame and Exit. Load Game stays dark until any saved game exists in the saves folder (a save of the first game kept in the same folder also lights it), and Cut Scenes opens only after a cutscene has been seen. Multi Player and Server still do nothing: multiplayer comes after the single-player game is complete.
- [ROM2] Load Game opens the second game's own Load Saved Game page: the saves listed by their names, newest first, with the time of the selected save below the list, and Ok, Delete and Cancel. Saves of the first game are no longer listed there.
- [ROM2] Saving opens the second game's own Save Game page. Type a name and press Ok: the first row, the empty slot, writes a new save; choosing a saved game puts its name in the box, and Ok then writes over the chosen save without asking, as the original does, even when another save with the same name is listed above it. Typing the name of a listed save, with no save chosen, writes over the newest one with that name.

## 0.117.0

- [ROM2] Blizzard works: creatures that know it now cast it. The storm strikes 32 times, one cell after another around its target, hurting everyone there (water protection softens it, and the caster's own side is not spared) and damaging buildings. A save made during a Blizzard loads with the storm still going.
- [ROM2] A save made while a Poison Cloud or another lasting cloud hangs over the map is no longer refused.
- [ROM1] A save made during a Meteor Storm now loads with the storm exactly as it was saved.
- [ROM2] In the first town a click on the shop opens the shop. The shelves show the stock in four groups with each item's name in the game's language, its quantity and its price; Buy, Sell and Undo trade at the original's prices, and a sale pays half the price, rounded. Items bought here stay with the campaign and are not yet carried into a mission. Saving and loading keeps the gold, the items bought and the shop's stock. The shops of the second and third towns still do not open.
- [BASE] Game Options has a new AutoCast mode button. It sets how your mages cast support spells by themselves: Minimum heals and shields, Average adds Bless and Haste, Maximum adds the protections and Invisibility. The choice is kept in saves and in your settings.
- [ROM1] AutoCast mode is Off in a new game and in older saves, so nothing changes until you choose a level. A spell armed in the spellbook, autohealing, Ctrl+U and Ctrl+A work as before; with a level chosen, your mages cast its spells after their armed spell.
- [ROM2] Items in a mission's sacks and item hints now show their own names, for example "Treasure", instead of numeric codes.
- [ROM2] AutoCast mode starts at Average and offers Minimum, Average and Maximum, with the game's own captions.
- [BASE] A mage fighting in melee range still heals and buffs by the AutoCast level; a mage attacking from range keeps attacking.
- [BASE] The spellbook shows the damage, range and duration a cast actually uses. A mage whose skill plus Mind is below 30 no longer sees much larger figures than the spell deals, a mod's damage table shows its whole damage, and in the second game the top of a few spells' damage range is no longer one point too low.

## 0.116.0

- [ROM1] Mods can add a mission that another character plays alone between two campaign missions (`data/missions.toml`). The party waits unchanged; when that character wins, the game goes on to the town, where the character joins the party with the experience, skills, spells and items gained. A save made inside such a mission loads only with the same mod. The repository's example mod `reniesta` lets the mage of Plagat play a copy of the first mission's map after mission 20; `modmission` copies that map from an install into the mod.
- [ROM2] The character creation screen looks more like the original: pointing at a difficulty, a portrait, the name, OK or Cancel shows its hint; "Name:" and the typed name are gold and yellow; the attribute numbers and the points left are gold on their boxes; the statistics card stays inside its frame; and weapon hints show in the original's dark box with gold text and their own line breaks.
- [ROM2] The tip on the second page of character creation now sits at the bottom centre, between the statistics card and the portrait.
- [ROM2] The second page of character creation has a fourth button, Restore, which returns the hero's starting attributes and skill, as in the first game.
- [BASE] Mods can set spell damage above 255: a value of 300 stays 300 before power scaling, and the spellbook shows the whole damage range. Saving a spell already in flight still limits its saved base and spread to 255 each.
- [ROM1] Spells deal the damage the original deals: before resistance a hit lands inside the damage range the spellbook shows, a negative resistance now increases spell damage instead of being ignored, Heal and Drain Life always give at least one point more than the lowest number shown, Fire Sacrifice follows the original's figures, and Stone Curse lasts longer on a target with negative earth protection.

## 0.115.0

- [ROM1] Enemy groups away from the party wait at their posts until the party comes near, as in the original; a group the mission's script names acts wherever the party is.
- [ROM1] A new mission keeps the original's time as a loaded save already did: regeneration and the decay of bodies run on the original's full-tick cadence, a fallen body's health drains more slowly while it lies on the field, the script's time counter runs even on maps without a script, and a mission's outcome is announced one moment sooner.
- [ROM1] The mission minimap now shows the original's picture: the real ground colours with the map's light and shadow, the crystal showing through where nothing has been explored and half through where the fog has returned, buildings at their real size and units in their owner's colour where the party can see them, and a white frame around the part of the map on screen. It still fills the whole crystal, and clicking it still only moves the camera.
- [ROM1] Fixed the position of stack counts and gold amounts in the mission
  backpack.
- [ROM1] The school's idle skill shine starts after half a second and cycles
  faster. Hovering a skill pauses it and restarts the idle delay.
- [ROM2] The third town, the druids' town, is a picture instead of a list: the square shows its shop and tavern, the woman and the man who step out when the pointer rests on the shop or the tavern, the lizard and the bug, with hover hints, and the innkeeper and the shopkeeper speak when clicked. The inn shows its water and taverner beside the people to talk to; Exit or Escape returns to the square, and a save made on the square or in the inn loads back into the same place. The shop still opens nothing.

## 0.114.0

- [BASE] The mission minimap marks every ground sack in sight with an amber dot,
  under a new Game Options entry, on by default. A sack hidden by fog has no dot,
  and a unit standing on a sack shows its own colour.
- [BASE] The Game Options window is one row taller so the new entry fits; every
  control keeps its place inside the window.
- [ROM2] The first town is a picture instead of a list: the square shows its
  gate, guards, tavern, sign, stars, weather vane, birds and passers-by, with
  hover hints, and the inn shows its candle, cauldron, taverner, the people to
  talk to on their seats and the Talk and Exit buttons. The gate stays shut,
  and the guard says so, until the first inn talk; Exit or Escape leaves the
  inn for the square.
- [ROM2] Kaarg, the second town, is a picture too: the square shows its gates,
  which open while the pointer rests on them, the guard on his rounds, two
  girls and a sleeping man, with hover hints and the shop and inn highlights.
  Clicking a girl, the guard or the sleeping man opens what that person says
  at this point of the campaign. The inn is the same page as the first town's,
  with Kaarg's own room and taverner; the shop still opens nothing.

## 0.113.1

- [ROM1] The witch brought to the village in the first mission stays there
  instead of running back to where she stood at the start. A unit a mission
  script hands to another side now holds its ground where it stands, as in
  the original.

## 0.113.0

- [BASE] W on the mission map takes back the weapon each selected unit held
  before, from its own backpack; a shield or bow that went with it comes back
  too, and the current weapon goes to the backpack.
- [BASE] Ctrl+A turns autocast on for every selected caster when any of them
  has it off, and off for all otherwise.
- [BASE] Home returns the map zoom to its default and End returns the game
  speed to its default.
- [BASE] The F1 help lists these keys in English and Russian.
- [BASE] Equipment put on while the game is paused now changes the unit's
  combat values also when the selection changes before the game resumes.
- [BASE] Double clicks on the mission pack use the system's
  double-click setting and keep the same timing at different frame rates.
- [BASE] An inventory item dropped onto a cell that cannot hold its sack lands
  at the hero's feet, joining the sack already there.
- [ROM1] A Stone Cursed unit stands completely still until the curse ends.
- [ROM1] A companion who arrives at a mission holding Quest Documents keeps
  them; they no longer move to the main hero when the mission starts.
- [BASE] A unit that turns to strike, shoot or cast at another unit faces it
  along the line between their body centres, as in the original, so a slinger
  no longer shoots at an ogre facing a diagonal beside it. A cast at a ground
  cell, a building use and a wall spell's direction follow the original's
  whole-cell rule.

## 0.112.0

- [ROM1] In the character generator's detailed page the statistics card's text stands one pixel further right, and the Experience row stands level with the Weight row.
- [ROM1] An ally a mission script hands to the player no longer gives the
  player's own side the key of the side the ally left when the game is saved,
  so loading that save gives the same game as playing on.
- [ROM1] A sack on the ground in an original save whose cell does not name it
  keeps its identity and its items' identities when the game is saved again.
- [BASE] Mods can choose the body a weapon is drawn with and supply their own
  body sheets.

## 0.111.0

- [ROM1] In the character generator the four buttons read Accept, Restore,
  Reset and Back from the top and are laid out as in the shop.
- [BASE] A double click follows the system's double-click speed and distance
  on every screen, as in the original; the tavern no longer wants a faster
  double click than the rest of the game, and a hero's double click in the
  first game's generator wants both clicks on one spot. In the Load list a
  double click loads the save the first click selected, at the second click.
  In the tavern a double click on a mercenary or a guest hires, dismisses or
  talks even when the first click was let go off the card. In the character
  generator a button held after a double click on + or - no longer keeps
  stepping.
- [BASE] A save taken while a spell, a burst or a shot is in flight keeps it:
  after loading, the spell or the shot flies on and lands with the same
  damage, as in the original. This includes a shot at a building.
- [BASE] A staff's bolt is drawn from the moment of the release, not across the
  wind-up. A spell's bolt no longer ends in a burst at the target for the
  spells that have none (3, 7, 8, 12, 17 and 19), a siege rider no longer
  shows a spell bolt, and the smoke trail of a bolt draws its frames in the
  original's order.
- [ROM2] A first-town save loads on an install whose town text lacks the inn
  conversation.

- [ROM1] In the character generator, Reset sets every attribute to 25 and the
  points to 100 as in the original, and the new Restore button returns the
  hero's starting attributes and skill.
- [ROM2] The second game plays its music: the menu, character generation, each
  town, the world map and the credits have their own tracks, and a mission
  plays its seventeen tracks with the theme of the area the hero stands in
  following the current track. A movie pauses the music, and the menu, the
  generator and a town pick it up where it stopped.
- [BASE] The `-nomusic` launch option silences all music.
- [ROM2] New Game opens the character generator: pick one of the four heroes,
  the difficulty and a name, then set the four attributes and the skill on the
  second page. Accept starts the campaign in the first town with that hero,
  the chosen difficulty and 1000 gold, and a save of that town keeps them.
- [ROM1] In mission 100 the servant no longer runs off at the win when the
  party lacks only the companion the map itself places there. A script role
  nobody in the party fills now goes to a named person the map places, as in
  the original.
- [ROM2] On the Russian install the character generator's tooltips read in
  Russian instead of wrong letters, and so do the other words the game reads
  from the install's text tables: menu and option captions, unit, building and
  spell names, the default hero's name and the help text.
- [BASE] Every hero and person now gets health, mana, speed, sight and combat
  values from one calculation, whether he was made by the engine or loaded
  from an original save. A skill bonus from worn items now raises a skill above
  100 for heroes loaded from an original save too, and a very strong hero's
  damage no longer exceeds what the original can hold.
- [ROM1] In the character generator each of the four heroes starts on the
  original's attributes and skill: Danath enters with Body 41, Agility 35,
  Mind 20, Spirit 15 and Blade. The card under the attributes is laid out as
  the town's character card, and the four attribute values and the points
  left draw in the original's small gold font.

## 0.110.0

Withdrawn; its changes are listed under 0.111.0.

## 0.109.0

- [ROM1] A mission script node that names a party role nobody in the party
  fills is no longer built, and a trigger naming it runs the map's first
  action instead, as in the original. In mission 100 a party without the
  companions sends the servant to (13,14) at the win, so he stops following
  the hero. Missions 30, 60, 81, 90, 130, 131 and 151 change the same way
  when their roles are missing; in mission 130 a party without the
  companion wins at the start. When that first action is a message, the
  message is shown, as in mission 90's win without its companions.

- [ROM1] The armour shelf in the town shop no longer offers plain rings and
  amulets, as in the original; they are sold only enchanted, on the Magic
  Items shelf. No shelf offers either beard.

- [BASE] On the world map the task's flag now stands above the red cross, as in
  the original: it shows while the pointer is over a task and stays on the chosen
  task through the whole journey, waving all the time. The red cross draws
  itself once and then stays whole instead of starting over.

- [BASE] In the tavern after mission 100, the speaker who offers the new
  catapult shows the siege engine's statistics and picture in the upper left
  panel, as in the original. Before, that panel stayed empty.
- [ROM2] The second game's chat cheats work in campaign missions: type the
  second game's unlock line, then `#create`, `#modify`, `#summon`, the kill
  commands, `#pickup all`, `#show map`, `#hide map`, `#victory` and `#event`,
  with the original's replies. Alt+D, Alt+T and Alt+Q toggle tracing and safe
  mode. The starter's cheat checkbox and `-chicken` unlock every campaign
  mission start and every load into a mission.
- [BASE] An overloaded hero now moves and turns as in the original: the
  overload slows his own speed first, to no less than 6, and a speed bonus or
  penalty is added after it. His character card shows that speed for every
  overloaded hero, and it is kept through a save.

## 0.108.0

- [BASE] An enemy mage with Mind above 59 now holds back a spell exactly 30
  times in 100, as in the original, instead of 30 in 101. With
  `-original-random`, the time spent in town no longer changes the next
  mission's combat rolls; on the second game the switch runs the default
  generator and says so.

- [BASE] Arrows, bolts and rocks turn toward a moving target while they fly
  and leave the shooter at the point its exact facing gives, as in the
  original. A smoking projectile loaded from a save grows its smoke trail
  again as it flies.

## 0.107.0

- [BASE] The original game's random number generator can be switched on with
  `-original-random`; `-seed <n>` replays a session from one seed.
- [ROM1] The character generator's tips work as in the original. The first
  page shows three step tips in the top right corner, advanced by a portrait
  click and then a level click, and highlights the portraits, then the
  levels, then the amulet and OK in turn. The second page shows the class
  tip, then the skill tip after the first skill click, and highlights the
  skills in turn until then.
- [ROM1] The generator's first page shows its selection cursor, a portrait
  is lit only when chosen or under the pointer, and only the original's
  click areas choose. The statistic buttons stand in the original order, plus
  before minus, and they and Accept, Reset and Back use the original pictures
  and pressed states.
- [ROM1] Missions 10 and 20 show their tips in the top left corner after the
  dialogue that names them, as the original does: at the start, at the sack,
  after the rogues, the woman, the bees and the ghosts, and after the
  Squiells. "Show tips next time" turns them off at once.
- [ROM1] The tavern, shop and school tips keep the original's size at every
  visit. The shop tip changes to its second text once you put an item on the
  table, as in the original.
- [ROM1] Windows, panels and lists now look alike everywhere. Every window
  frame is drawn with whole edge tiles as the original draws it, and the
  cutscene library and Sound Options lists sit in the same sunken well as
  Save and Load. Every button activates only on a release over it; losing
  the window's focus drops a held press.
- [ROM2] The windows shared with the first game (Load Game, Save, the in-game
  menu with its options and objectives pages, notices and the cutscene
  library) now draw the same frame and list well as in the first game, and
  their buttons follow the same press rule.

## 0.106.0

- [ROM1] A Ghost raised by Control Spirit now takes its whole Ghost row: it
  regenerates health, sees invisible creatures within two cells and has
  carrying capacity 300. Before, it had none of these.
- [BASE] The town square is built from a town description by one town
  composer; it looks and plays the same.
- [BASE] The tavern, shop and school are built from the town description by
  the same composer; they look and play the same.
- [BASE] The game is chosen once, by one profile; nothing changes in play.

## 0.105.0

- [ROM1] A party member killed in a mission started from the town now keeps
  his body for his dying time, 12 or 8 ticks, as he does after a save and
  load. Before, his body could break down on the tick he fell.
- [BASE] Images and sounds are read by one decoder per file format. Nothing
  on screen or in the sound changes.

## 0.104.0

- [BASE] Every actor is now built one way. Nothing changes in play.

## 0.103.0

- [ROM1] Lightning and Prismatic Spray draw the bolt the original draws: a
  random zigzag of curved segments built anew on every tick, one star per
  point, flickering through the original's thirteen-step brightness ramp.
  A Lightning or Prismatic Spray cast by a map trigger starts at its own
  ramp, and a map-triggered Prismatic Spray now draws its rays. The bolt
  lights the ground cells under the drawn figure, so a caster whose raised
  hand sits over the next row up is no longer lit by his own bolt.

## 0.102.0

- [ROM2] The town inn offers its missions and talk at stages 40 to 110 once
  the campaign reaches those stages. A new ROM2 campaign still stops at
  mission 30, which cannot be won without companions.

## 0.101.0

- [ROM1] Spells leave the caster's staff tip or hand, on the side he faces,
  as the original places them for his weapon and direction.
- [ROM1] Lightning and Prismatic Spray light the ground and the units along
  their path while they last. Fire Arrow, Fire Ball and the Fire Ball
  explosion light the ground and units around them. With Dynamic lighting
  off, a Lightning or Prismatic Spray still lights the units it passes and
  leaves the ground unlit.
- [ROM1] Wall of Fire also lights the ground around its flames.
- [ROM1] Town tips look like the original: ornate frame, solid teal fill,
  justified shadowed text, a panel sized to its text, and the checkbox and
  Close inside the frame.
- [BASE] The black area around the minimap crystal is gone; the map shows
  through.
- [BASE] The red square around the unit under the attack cursor is gone; the
  unit's highlight stays.
- [BASE] The LOAD window no longer shows a note under original saves; the
  selected save shows its own label.
- [BASE] The map no longer shows black strips at the right and bottom while
  scrolling, and zooming out keeps the view centred instead of moving the map
  into a corner.
- [BASE] Creature cards show the creature's proper name from the game's text
  instead of an internal name such as BAT_SONIC.3. The SAVE and LOAD file
  lists sit in a pressed-in frame, the Save dialog's buttons stay inside its
  border, and the options dialog is centred inside its frame.

## 0.100.0

- [BASE] Every save, in a mission or in town, is now written from the game as
  it stands instead of being patched over the save you loaded. Bodies, the
  orders units were following and the gear of fallen units are saved as they
  are now, not as they were when the save was loaded.
- [ROM1] Original mission saves that could not be saved again after loading,
  failing with a message about the current Player, now save.

## 0.99.0

- [BASE] A unit chasing an enemy it cannot yet reach looks for a way to it on
  the original's schedule: a full search now and then, a short one in between,
  and the original's choice of the cell to head for. This replaces the earlier
  stand-in that sent such units on a wide search every time.
- [BASE] A save no longer fails when an arrow is in flight after a loaded
  spell has ended.

## 0.98.0

- [ROM2] The campaign goes on past the first missions. Winning any
  mission continues to the places it opens, and the cutscenes that follow a
  victory play only when their conditions are met.
- [ROM2] The second town's inn lists the people to talk to. Talking to
  them opens maps 30 and 31 from a new game, and map 31 leads on to 32.
- [ROM2] Every ordinary mission can be saved and loaded.

## 0.97.0

- [ROM2] Spells work. The spell table loads on both language versions,
  and mages cast in ordinary play: the mage on map 10 heals the hero and the
  mage on map 21 casts Ice Missile. Bless, Curse, Shield and Invisibility each
  act as their own spell, and a cast in progress survives a save and a load.
- [ROM2] The autosave written when a mission starts loads again.
- [BASE] Ordering an escorting unit to patrol, move as a group or swarm now ends the
  escort completely; a struck escort that later patrols no longer stops the
  game from saving.

## 0.96.0

- [ROM2] No gameplay change. Groundwork for the campaign: every one of
  its 46 maps is surveyed for what the game still lacks there.

## 0.95.0

- [BASE] Escorts behave as in the original: an escorting unit chooses between healing
  and fighting as the original does, turns in place while idle, and keeps
  aiming at a moving target.

## 0.94.0

- [BASE] Buttons, lists, sliders, check boxes and dialog frames are drawn and behave
  as in the original: a gold caption on hover, a pressed look while held, and
  the list thumb following the selected row.
- [BASE] Units turn at the original speed and stand facing any of their 16 directions.
- [ROM1] Skrakan's line at the portal plays on both language versions.

## 0.93.0

- [ROM1] The original's cheat chat commands and the Alt console.
- [BASE] Unit health bars turn yellow and then red at the original thresholds.
- [BASE] The map camera stops at the map edges where the original stops, and the
  edges are drawn as the original draws them.
- [BASE] Identical items picked up stack in the pack.
- [BASE] The structure readout is centred in the mission card.
- [BASE] A melee attacker turns to face its victim before the first blow.
- [BASE] An open pack with no hero shows its message, and the mouse wheel scrolls the
  pack.
- [BASE] Enemy cards show what the party knows about that enemy.
- [ROM1] Weapon skills taught at the school shine in the order they are drawn.
- [BASE] A trained skill loaded below its experience level rises to that level.
