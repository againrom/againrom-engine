# Changelog

What changed for players in each release. A release lists every change since
the previous release, including fixes that reached `main` between releases.
New entries go under Unreleased; a release renames that heading to its version.
Each entry starts with its scope: [BASE] for the engine under both games,
[ROM1] for the first game, [ROM2] for the second game.

## Unreleased

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
