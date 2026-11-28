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

- [BASE] Resizing the game window no longer makes the game use more and more memory.
- [ROM1] After loading a saved game, Control Spirit raises a Ghost from bones again.
- [ROM1] A unit ordered to attack a building keeps attacking it when an enemy comes near.
- [ROM2] If hero creation does not open, a new game starts with the default hero carrying the weapon and armour hero creation gives that class, not a short sword and a teleport scroll.

### For map makers and modders

- [BASE] A mod saved from the editor with a line break or a control character in a town, mission, item or other text now opens again instead of failing to load.
- [BASE] In both games the `map.reg` of the install or of a mod now sets how fast ground units step (`SpeedMultiplier`, 8 by default) and the precision of the computer units' sight (`ScanShift`, 7 by default); a value outside 1 to 255 or 1 to 8 is refused when the file loads.
- [BASE] A spell picture a mod changes now looks the same in flight after a save is loaded as before it, and a broken projectile list in a mod now stops the load with an error instead of loading silently.

## 0.162.0

### For players

- [ROM1] Multiplayer: a hero who uses a Shop leaves the map while the Shop page shows and comes back on his own cell, or on a free cell within three; each deal saves the character, at most once every 15 seconds.
- [ROM2] Multiplayer: a hero who uses a Shop or an Inn opens that building's town page with the single-player stock or offers, trades with his own pack and purse, and comes back on his own cell.
- [ROM1] [ROM2] Multiplayer: a hero who dies no longer ends the map; press Space to return him near the last shop used or the drop point, with a tenth of his skill experience lost and full health and mana
- [BASE] On a multiplayer map, a monster group whose members are all dead comes back 60 seconds later on its own cells, with full health and mana.
- [BASE] Map editor: the trigger condition is now "Unit changes its cell" and asks only for the unit; it holds each time that unit stands on a different cell than a moment before, so a trigger can fire on the unit's first step.

### For map makers and modders

- [BASE] A mod can now carry its own `map.reg` (`files/<layer>/world/data/map.reg`) to set how far computer units search routes in both games; a route refresh value above 255 means never and below 0 means always.

## 0.161.0

### For players

- [BASE] A quick load that the active mods refuse now opens the same mod mismatch window as the Load page, and the quick save and quick load messages are now shown in the game's language instead of English.
- [BASE] Changing one option no longer erases your other saved options when the options file cannot be read at that moment; the change is refused and the file stays as it was.
- [BASE] Leaving a multiplayer map through the game menu (or winning it) now returns to the character list on that character's row and saves its statistics, gold and skill experience to its file (the first game's file keeps statistics only); the next map entered with that character in the same run starts with everything it carried, items included.
- [BASE] The multiplayer map choice page now exists in both games but is not yet reached from the menu: it lists the multiplayer maps of the map folder (with each map's level shown in the second game) and leaves out unreadable files without crashing; choosing a map and pressing OK returns it to the caller, and Cancel returns with nothing chosen.
- [BASE] The Multiplayer character list now shows each game's character files from the player profile by name, newest first, ends with a New Character row, says how many unreadable files it left out, and in the first game deletes the chosen character after the Load page's question.
- [ROM1] A first-game map trigger can now run a Teleport action that moves any living unit standing on its source cell to its target cell, or to the nearest free cell when the target is occupied, and repeats whenever the trigger fires again; maps that use it save as extended maps, and the map editor sets both cells by typing or clicking on the map.
- [BASE] The main menu's Multiplayer button now opens an empty character list page with Accept and Cancel (and Delete in the first game); Cancel returns to the menu. Before, the button did nothing.
- [ROM1] A walking unit that reaches the middle of a map's Teleport trap cell is now moved to the trap's target cell and stands there; a unit that was following a move order then walks on to that order's destination; before, the trap did nothing. A flying unit is not moved.
- [ROM1] Fire Sacrifice, Acid Stream and Meteor Storm now strike their first cells one tick after the cast, as in the original; before, the first cells were hit on the cast's own tick.
- [ROM2] On a map that sets up Outposts, a computer group that is wiped out now comes back from its Outpost when the countdown runs out, and a map script can set that countdown; a loaded game's Outposts bring no group back.
- [ROM2] A mage's automatic Heal set to allies now also heals the player's own units, and one set to neutral units alone does not, as in the original.
- [ROM2] Computer units search long routes, aim their walking steps near a route end, pick targets to cover a defended unit and decide to withdraw from invisible enemies by the rules read from the original: a defender of a short-range unit ignores flyers, and a group member that detects an invisible enemy keeps its comrades from dropping it.
- [ROM2] Hired and placed humans no longer start a mission ready to back off or flee by their class; they fight on until a Retreat setting tells them otherwise, as in the original.
- [ROM1] A unit placed on a map now knows the spells the map's unit spell records give it, so a spell given to a unit in the map editor's spell book is in its spell bar in the game; before, it knew only the spells of its kind.
- [ROM2] Shop prices of armour, shields and weapons with magic effects now follow the original: the effects' surcharge is 8750 gold for a size of 70 rather than 7000, a spell on an armour or a shield adds to that size, and an item with a price effect costs that price.
- [ROM2] A negative protection now raises the damage of a spell or of a weapon's magic part, and a protection above 100 stops it whole, as in the original; before, protection counted only from 0 to 100.
- [ROM2] The experience a mage gets for killing a unit with a spell now goes to the skill the original picks, which the spell's sphere number chooses, rather than always to the spell's own sphere.
- [ROM2] A corpse now turns to bones only below -10 health, and reaches its later stages below -20 and -40, as in the original; a dead hero's body stops decaying at -61 health and, in the Russian version, is never removed by decay.
- [ROM2] In the Russian version, a unit that dies on a cell whose sack already holds 1024 or more items adds only its gold to that sack and its items are lost, as in the original.
- [ROM2] A unit that kills with its weapon's spell now gets the experience for the kill, as in the original.
- [ROM2] A multiplayer character file that the character list shows now also opens; a file with a damaged first section is left out of the list instead of failing on open.
- [BASE] Bodies reach the next stage of decay only below -10, -20 and -40 health. Reloading at -10 keeps the loot already dropped, and older saves still load.
- [BASE] New Character in the multiplayer character list now opens the character generator, and Accept saves the new character in the original's file format and shows it in the list.
- [BASE] Accepting a character in the multiplayer character list now opens the multiplayer map choice, and Ok starts the chosen map with that character, its statistics, money and experience.
- [ROM1] A map can trigger an event when a chosen unit leaves its starting cell.
- [ROM1] A unit moved by a teleport stays where it lands instead of walking back.
- [ROM1] An armour or a shield with a spell is priced as the original prices it, by the spell's power, and no longer at a weapon's spell price.
- [ROM1] Experience from a blow, a kill or a cast is scaled by Mind as the original computes it, so some awards are one point lower.
- [ROM1] A weapon with Fire Ball now throws its ball after every blow, even one that fells the target, as in the original.
- [ROM2] A second-game map can carry the engine's extended script, so its triggers can fire when a chosen unit leaves its starting cell.
- [ROM2] A weapon with Fire Ball now throws its ball after every blow, even one that fells the target, as in the original.
- [ROM1] A walking unit with a route of exactly five cells now aims its next steps at the end of the route instead of the fourth cell, and a unit pushed more than three cells off a longer route aims at the route's nearest cell, as in the original.
- [ROM1] A guard post and a defender's cover now look as far as the unit's own search radius byte, which a loaded save can change, instead of always five cells.
- [ROM1] A defender now picks the enemy nearest edge to edge, so a wide enemy counts from its edge, and ignores flying enemies while the unit it defends can reach only one cell.
- [ROM1] A wounded unit that is retreating now keeps an invisible enemy in mind when any unit of its group can see invisible at that distance, measured edge to edge, even if no unit sees it by sight.
- [ROM2] A monster felled to exactly -10 health now drops its gold and item once, instead of rolling again on every tick until its body decays further.

### For map makers and modders

- [BASE] Both games now read the [Path Finding] route values (route refresh rates, scan-ahead, and the route-end distances) from the data/map.reg of the install, and use the old built-in value for a key the file lacks; an unchanged install plays as before; a StaticIsntNeeded below 3 is used as written and never reaches past the end of a route.
- [ROM2] In the map editor a second-game map's message pages (the event sections of its mission text) now show their text and speaker and take edits to the message and its parts; before, they were reported as unavailable.
- [ROM1] In the map editor a first-game trap of the Teleport spell now reads as a teleport from its cell to its target cell, and its cell, target and power are set and saved as before.
- [ROM1] In the map editor a placed or changed Teleport trap now shows its Source and Target as two cells, each with an On map button whose click sets the cell and Esc keeps the old one; a new Source moves the trap to that cell.
- [ROM2] In the map editor a second-game map's pages window now shows the sections of the mission text the game reads for the map, names the actions that send each event, edits a section or adds one, and saves the edited text into the map's mod, where the game shows it.
- [ROM2] In the map editor a second-game unit, structure and sack now name every field and mark the ones no game reader was found for, a sack item's word says whether the owner takes the item in hand or into the bag, and the passability view and brush show and set which move kinds each cell closes.
- [ROM2] In the map editor the trigger wizard now offers the teleport as its own ready action under the cells heading, asking for the entry cell and the exit cell, and the action line reads as the teleport with those two cells; the event number, the default power and a source equal to the entry cell are no longer shown.
- [BASE] In the map editor the unit window's spellcaster caption now looks and reacts like a button, spells that are off in the unit's spell book are drawn dimmed, and a spell given to a unit no longer shows as a label beside it on the map.
- [BASE] In the map editor the unit window's spellcaster button no longer has the yellow caption showing under its label.
- [ROM2] In the map editor's trigger window, a second-game node now shows its ten text fields beside its label, each marked as read by nothing the game was found to do, and both the label and the texts are edited and saved into their own bytes, also on a node whose operation is not decoded; pages, messages, dialogues and variable names of a second-game map are still not offered.
- [ROM2] In the map editor, a second-game map's traps now show and change their spell, power and cells, and a building's spell caster its spell, power and radius; the palette adds both.
- [ROM2] In the map editor a second-game map now draws only what is read of it: the passability view stays off and says why, the border view marks the edge ring, trigger areas are drawn as the cells a node names, height numbers show the stored height, and a new map on a second-game install says it is a first-game map; search and the player legend work on a second-game map as before.
- [BASE] In the map editor a trap can no longer be placed on, or have its Source moved to, a cell that already holds a trap: the editor says "This cell already holds a trap." and leaves the map as it was.
- [ROM1] In the map editor a first-game map's message text now goes through the second game's letter check too, so both games refuse a letter the game would show as another, and a NUL, with the same words; the pages window lists the map's pages with the actions that show each.
- [ROM2] In the map editor a second-game send-event action now offers New dialog: it adds the next free event section to the mission text, points the action at it and opens the section's text in the pages window.
- [BASE] In the map editor a node only an extended map runs (an interactive dialogue, a mod global, Unit left its cell, Teleport) now asks to extend the map at the edit that adds it, and again when a map in the original format that already holds one is saved; a refused fold names the reason in the editor's language.

## 0.160.0

### For players

- [BASE] A mage under the Retreat order now casts its automatic spells, healing and strengthening itself and its friends, whenever no living enemy stands near; before, it never cast while retreating.
- [BASE] A swing that misses, or whose damage armour or resistance cancels, now still counts as an attack on its victim: the victim's side turns hostile and the attacker is remembered, as in the original; before, only a blow that hurt did.
- [ROM2] A unit that flees by Teleport and whose cast is refused now stays where it stands instead of walking away, as in the original.
- [ROM2] Prismatic Spray now skips a side target farther from the caster than the spell's range, as in the original; before, it took the nearest enemies in sight at any distance.
- [ROM2] A fallen body now fades more slowly, as in the original: one at -10 health or lower loses a point every 32 ticks and one above -10 a point every 64; before, the two kinds of loss were added together.
- [ROM2] A unit killed after a Drain Life hit now credits the caster with the kill, as in the original; before, the kill earned the caster nothing.
- [ROM2] A unit whose weapon charge is 0 now waits 256 turns before its blow, as in the original; before, it struck at once.
- [ROM2] An enchanted armour, shield or weapon now costs its base plus one surcharge on the sum of its effect sizes, as in the original; before, each effect was priced alone, so two effects of 70 cost 14000 instead of 26798.
- [ROM2] A fallen body above -10 health now loses health half as fast as one below -10, as in the original; before, every fallen body lost it at the faster rate.
- [BASE] A walking unit now stays on the cell it leaves until it is halfway across and keeps both cells while it steps, so no other unit walks into either; a unit blocked by a walker waits for it to move on, one blocked by a standing unit goes round it, and a blocked unit searches again only after a set number of passes, as in the original.
- [ROM2] An enemy's card now opens its groups by your kills of that kind for every creature, humanoid ones and kinds such as the catapult included, as in the original; before, many showed blank stats however many you killed.
- [ROM2] The pick-up key no longer sends a hero after a sack lying where he cannot stand; a sack under his feet is still taken, as in the original.
- [ROM2] The Retreat order now makes a unit look for enemies within 10 cells, not its own sight range, and a mage with Teleport and mana teleports away by it, as in the original.
- [ROM2] A patrol whose point no route reaches (behind a wall or water) now turns back when it stands within two cells of that point, as in the original; farther away it waits and tries again. A point held by another unit is not covered: the patrol walks up to it and waits.
- [ROM2] Poison Cloud's poison now stays on a unit for the same short time whatever the caster's power, and each new pulse or cast sets it back to that time, as in the original; before, a stronger caster's poison lasted much longer after the unit left the cloud.
- [ROM1] A Heal ordered at a body at -10 health or below, or at a unit with no health, now costs its mana and does nothing, as in the original; before, the order was refused without cost.
- [BASE] Poison now keeps lowering a fallen unit's health until it runs out or the body is gone, and the body's effects end with it, as in the original; a decaying corpse shows no poison picture. Before, poison stopped on a unit it had killed.
- [ROM2] Acid Stream now paints its first stage one tick after the cast lands, like Blizzard, as in the original; before, it painted at once.
- [ROM2] On the world map, a place's name and description lines are now centred over the map's text area and the name is drawn in the lettering of the hero's name on the character screen, as in the original; before, each line began at the middle of the area and ran off to the right.

### For map makers and modders

- [BASE] A mod.toml with a malformed value or a malformed line after its first `[` table header is now refused with its line number, and the editor no longer overwrites a malformed value when it saves a manifest; a triple-quoted description is no longer skipped silently.
- [BASE] In the map editor's unit window, a placed unit's maximum mana is now set as its maximum health is, empty or 0 keeping the kind's maximum, and a press on a mage's SPELLCASTER caption opens its spell book, drawn as the game's book over the spells the map's mod states, where each spell turns on or off and is saved as the unit spell record the original editor writes; the game does not yet give the unit those spells, and a second-game map refuses both with the reason.
- [ROM2] In the map editor's trigger window, an action 25 node of cell event 23 now reads as a teleport with its entry and exit cells, which a map click changes, and a new teleport is chosen from the operation list; on a second-game map the editor now adds, changes and removes a type-9 teleport record from its cell to its target, and a unit on a teleport now moves when its step ends at the centre of the teleport's cell, as in the original, not when it first crosses into any cell under it.
- [ROM2] In the map editor, a second-game map now takes the height brush, the passability brush, random heights and the object brush to clear objects, each inside the map's 8-cell edge ring; object classes, bridges and random terrain are refused with their reason; before, every brush but terrain was off.
- [ROM2] A mod's `duration_factor` for Poison Cloud is now refused with its reason, since the second game's poison lasts a fixed time.
- [ROM1] In the map editor's Dialog window, a dialogue reply now takes as many conditions and actions as a trigger: every condition must hold, the actions run in their order, and each row can be added, removed or moved with Alt and the arrow keys; before, a reply showed one condition and one action, and a dialogue file with longer lists was refused.
- [ROM1] A mod can declare its own global variables, each with a number, a name and a starting value, and the triggers and dialogue replies of its maps can read, set and add to them during a mission, and a save taken in the mission keeps them; a map that names a variable its mod does not declare is refused.
- [ROM1] A mod's global variables now last for its whole campaign: a won mission passes its values to the next mission, a lost, left or restarted mission throws away what it wrote, town and mission saves keep the values, and an original or older save starts at the declared values.
- [ROM1] In the map editor's mod window, a Globals page declares, renames and removes the mod's global variables, and the trigger wizard and the Dialog window's replies can compare, set and add to them by name, with a choice that makes a new one; a global a map or reply uses cannot be removed, and the page names who uses it.
- [ROM2] In the map editor, every check and action of a second-game map now shows by name and its parameters edit, a trigger's name, once flag, comparisons, slots and a node's label edit, and a trigger opens in the trigger wizard, where its rows move and go, and copies with its checks and actions; before, most operations showed "not decoded" with read-only parameters and a trigger took only a move.
- [ROM2] In the map editor, a second-game map now takes new triggers, checks and actions; check 21 takes the id of an object on the map, action 6 shows its arms by name, actions 21 and 24 show as a spell cast at a cell or a unit, 16 other actions show what they do, and a trigger holding action 65538 now copies; before, all of these were refused or showed raw numbers.
- [BASE] The map editor's Globals page now counts only the dialogue files the game loads.
- [BASE] In the map editor's trigger window, Alt+Up and Alt+Down move the focused condition or action row, and an added row takes the focus, as in the dialogue editor.
- [BASE] The map editor now refuses a command or edit that gives a script parameter the new-global marker, saying a new global is added on the Globals page; before, it failed with an out-of-range number.
- [ROM2] In the map editor, a check or action of a second-game map now changes to any other check or action whose parameters are known, taking that operation's parameters, the trigger wizard offers ready rows for the second game's checks and actions, and a map already holding 1000 checks is refused in words naming 1000, not 100.
- [ROM2] On a second-game map the trigger editor gives box areas and script variables the second game's register limit, 1000, not the first game's 100.

## 0.159.0

### For players

- [ROM2] Altars now switch on and off when used, units knocked unconscious still count for mission goals until they die, script triggers that name a not-yet-known object now run instead of being dropped, a trap without a source strikes from its own cell, and mission scripts can hand an item from one unit to another or move a unit, after which a computer unit stands guard.
- [BASE] A unit at full health now backs away or flees from a nearby enemy when its health is at or below its withdrawal point, as in the original; bats and archers no longer stand still at full health.
- [ROM2] A spell rolled on a mage's item in a shop now gets a budget of a hundred times the item's price, as in the first game; before, it got ten times, leaving those spells weaker or absent.
- [ROM1] Computer units now pick a unit under Stone Curse as their target last, as in the original; before, Stone Curse did not change their choice.
- [ROM2] Units now climb slopes at the original's speed, a unit with nowhere to flee stands still, and your units' retreat point now depends on their class, as in the original.
- [BASE] A creature with several weapons now reaches as far as all its weapons together, as in the original; before, only the first weapon counted.
- [BASE] An invisible mage who casts at another unit or at a cell now stays invisible one step longer, as in the original; a spell cast on itself does not change its invisibility.
- [ROM2] In the English version, the second pulse of a straight Acid Stream now strikes one cell twice and skips the cell on the far side, as the original does; the Russian version keeps its own cone.
- [ROM2] Units now step as in the original: diagonal steps keep the first game's pace and pauses, a struck guard steps ten cells away along the longer direction, and computer units rank a target under Stone Curse lower.
- [ROM2] A spell still flying when you save now lands after loading without a caster: the target loses health but does not learn who attacked it, does not turn hostile, and a kill gives no experience.
- [ROM2] A town save now keeps each party member's experience and the player's kill count; what a loaded save from the original holds about the player stays as it was.
- [ROM2] The Kaarg inn taverner now runs its whole long animation, a shop's second tip shows once the deal table holds an item, and the druid inn's lower-right corner shows its original picture.
- [ROM2] A mission's step costs on layered cells now follow the original: after a second walker shares a cell, the cell keeps its decayed cost, and a mission save keeps it.
- [ROM2] A dead unit's body now sinks through its corpse stages and leaves the world as fast as in the original; before, it did so one and a half times slower, which also kept it raisable longer.
- [ROM2] A new hero who picks Shooting or Astral now starts at level 10 in it, as in the original, and the hero's experience survives a town save.
- [ROM2] The mission report after a win is now a framed popup at the original's size and place with its two buttons where the original has them, the failure page has the original's width and button places, and movies such as the teleport play at their original place centred on the screen.
- [BASE] "Abandon mission" is now a row of the End Quest page instead of the game menu's own row, and it abandons the mission at once without asking again.
- [BASE] "Abandon mission" and the rows mods add to the game menu are now drawn as buttons, like the other rows.
- [BASE] A mage's Teleport now plays its sound also when he was still finishing his turn toward the target as the spell began; before, the second game cast it in silence, and the first game lost its second sound.
- [ROM2] The Teleport sound from a spell or a scroll now plays where the caster stands, once per cast, and is no longer lost after the jump.
- [BASE] The in-game chat and command line now remembers the last 50 lines you sent during the session; Up and Down step through them, and Down past the newest brings back what you were typing.
- [BASE] A scroll used while a spell is being cast or recovering now waits and works right after it; two Teleport scrolls in a row both work.
- [BASE] A reply row in a dialogue now looks pressed only while the mouse button is held, not when the pointer is over it.
- [ROM2] Computer Humans with no spell book no longer retreat at any wound; mages and creatures with Teleport now teleport away when they flee instead of only walking.
- [ROM2] A computer mage now heals itself and its allies even while an enemy strikes it, as long as it attacks no one itself.
- [ROM2] An area spell no longer sends monsters of another side that it hits after the caster, and a spell aimed at a large creature lands at the centre of its body.
- [ROM2] Computer units now flee from an enemy within 10 cells however far they see, a mage with Teleport does not flee while the enemy is more than 5 cells away, and a wounded mage no longer starts healing while it retreats.
- [ROM2] A wounded guard with no target that was just struck now steps aside, away from its attacker, as in the original.
- [ROM1] An area spell cast at a large creature (ogre, troll, dragon) now lands in the middle of its body, as in the original.
- [ROM1] Water and earth damage from a hero's items in a save from the original is now reduced by protection from its own element, as in the original; before, water and earth were swapped.
- [ROM1] An item's elemental damage now adds to a blow only when the blow hits through accuracy or the weapon has no damage of its own, as in the original; a blow that lands only through an automatic hit deals plain damage.
- [ROM1] A spell rolled at random on a mage's item in a shop now gets a budget of a hundred times the item's price, as in the original; before, it got ten times, so those spells were weaker or absent.
- [ROM2] Spell effects on a slope (Teleport pictures, lightning and flying bolts) now sit at the ground's height where they are, as in the original, not at the cell's average height.
- [ROM2] Poison Cloud and Curse cast by a hero or a creature no longer draw their own spell picture; only the mark on the unit shows, as in the original.
- [ROM2] An area spell cast on a unit, such as Acid Stream, now heads from the caster's cell toward the target's cell, as in the original; before, it followed the caster's facing and could miss the target.
- [ROM2] Your Humans now take their class retreat and flee points at mission start, as the computer's Humans do; with the retreat option unset, a wounded mage steps back from a near enemy and flees at a quarter of health.
- [ROM2] Leaving a town through its gate no longer adds Dina to the party; as in the original, only the win of mission 40 brings her in.
- [ROM2] Shortly after starting a book or scroll spell, a mage again regains mana three times faster, as in the original; flying spells such as Fire Arrow now travel the straight distance to the target, so they arrive later on a diagonal, while Lightning and Prismatic Spray still strike almost at once.
- [ROM2] Casting a spell whose target or cell cannot take it now spends mana first, as in the original (casting without enough mana still costs nothing), and a neutral unit caught in an area spell only turns hostile after more than ten such hits, below half health, or on death (a poison tick still turns it at once).
- [ROM2] A second-game mission now keeps its trigger cells working after a save and a load from the main menu; before, they were lost on loading a saved mission.
- [ROM2] The second game's 29th spell now stays in a hero's spell book after a save and a load from the main menu; before, the saved book held only 28 spells and the 29th was lost.

### For map makers and modders

- [ROM2] Mission scripts now run with all 1,000 registers (starting with register 901 at 1, counting each script tick in register 999, and keeping registers 100..999 across save and load) and support the player-buildings check, the occupied-cell relocate check, and the counter action for register 902.
- [ROM2] In the map editor, opening or editing a mod that replaces install files now reloads the second game's definition table under the second game's layout; before, it read the table under the first game's layout.
- [ROM1] In the map editor, the record card of a unit, structure or sack lists its fields in the second game's layout, with labels in English and Russian.
- [ROM2] The map editor now shows and changes the fields of a unit, structure or sack on a second-game map: owner, definition by name, health, flags, type and items; a field whose meaning is not known yet is named by its place, such as "Field 0x0a".
- [ROM2] The map editor's Players window now lists the players of a second-game map, renames them, sets their colour and relations and adds a player; two unread roster words stay read-only and a player cannot be deleted there.
- [ROM2] The map editor now checks a second-game map, shows the report and a problem mark, asks before saving a map with errors and starts a test game of it; checks that rest on first-game facts are skipped, and the report names them.
- [ROM2] The map editor now selects with a frame, copies, pastes and groups units, structures and ground sacks on a second-game map, and a pasted unit keeps its own sack; the store and recipe tools and area copy stay off there.
- [BASE] The mod window has a "Move old paths" button that carries every file a row names on an old path into the mod's `assets` and `maps` folders and rewrites the rows; it refuses, naming both files, when a target holds a different file.
- [BASE] A mod whose numbered map files replace no mission of the install marks no save again, and the override line of a mission replaced by a map file names the file without a line number.
- [ROM1] On the mod window's Towns page, "Switch on" gives a town back the table its towns file holds, with every key written by hand, and emptying only one number of a map point leaves the point as written and says why.
- [BASE] A mod town's tavern page now takes its buttons and seat appearance from that town's own tavern room, each button acting by its name (hire, talk, exit, sleep); a button with another name does nothing, and what the room does not name stays the game's.
- [BASE] The map editor now edits plain and interactive dialogues in one Dialog window, opened from the Start dialog's Configure button and from the mod window's Dialogues page: a short list of lines on the left, the selected line on the right with its speaker, text, hero variants by sex or class and what follows the line, and up to four replies per line; message pages are written while no line has a reply choice and a dialogue file when one has, and the earlier separate dialogue window with its Legacy and Interactive switch is gone.
- [BASE] The mod window has Screens, Settings and Speakers pages: list, add, change and remove each kind's rows, one Undo step per edit, with a speaker's picture changed through the asset import and a speaker a message still names kept until its uses are gone; Save mod refuses a row the game cannot use, naming it, and a mod saved with no edit keeps its files as they were.
- [ROM2] The map editor opens scenario and root maps for viewing, with terrain, objects, units and structures; editing and saving stay off.
- [ROM2] A second-game map in the editor now takes terrain brush strokes and saves: a map saved unchanged is byte for byte the file it was read from, a stroke changes only the tile plane and its group mask, and every other tool is refused with one line.
- [ROM2] On a second-game map in the editor, the trigger window lists each check and action by the second game's own operation name, or as "Operation N, not decoded" with its raw parameters where the operation's meaning is unread, shows compiler parameters (operation 65538) as such, and states the second game's 1000 registers; editing stays off.
- [ROM2] On a second-game map in the editor, the trigger window now edits the parameters of every action and check whose operation is decoded, and the map saves with the edit; a not-decoded operation stays shown as stored, and triggers cannot yet be renamed, added or removed there.
- [ROM2] The mod window opens on a second-game install with no map open: its pages for texts, items, characters, spells, grounds, bodies, companions, screens, settings and speakers edit and save a mod, and a map the mod holds offers to replace one of the second game's missions, named by its map's name where it has one.
- [ROM2] On a second-game map in the editor, the Units, Structures and Sacks palettes list the second game's own catalog by name, and a placed unit or structure shows its catalog name or says it has no catalog row; placing stays off.
- [ROM2] On a second-game map in the editor, the place, move, copy and delete tools now work for units, structures and ground sacks: a new unit or structure is a copy of one the map holds, takes the lowest free id and stands at its cell's centre, a structure within 8 cells of the edge is refused, a deleted unit takes its sacks and their bound elements and a deleted structure its elements and list entries, and the map saves and reloads; objects, scenery and the dialog fields stay off, and whether the original program loads an edited map was not tested.
- [BASE] In the map editor, a window opened from another window, such as a trigger opened from the Scenario list, now stays open when its parent is closed by its close button or Cancel; the top bar entry and a change of mode still close them all.
- [BASE] The mod window has a Companions page: add, change and remove the rows that make a town's companion join only when the hero opens a chosen tavern conversation; Save mod refuses a row the campaign cannot run, naming it.
- [BASE] The mod window's Bodies page now offers the weapon body choice even while the mod holds no body, so a weapon can take a body from the game's bodies.
- [BASE] The map editor now edits interactive dialogues: the Start dialog action chooses between a message page and a dialogue with replies, a window edits the dialogue's nodes, their texts in two languages and up to four replies each with its next node, once mark, hidden or disabled look, condition and outcome, and the mod window has a Dialogues page; Apply and Save mod refuse a broken dialogue in plain words, and a dialogue is saved and published with the mod.
- [BASE] The map editor now draws its notes, refusals and default node names in Russian instead of question marks, and the default node name falls back to English where the editor's font cannot draw the letters.
- [BASE] Saving a map from the map editor now writes its interactive dialogue files into the .dialogs folder beside the map, in the map's own folder and in a mod folder alike; before, a trigger that started a dialogue with replies ran its other actions and showed no dialogue.
- [BASE] A mod whose records did not load now refuses edits on every mod page in the map editor, not only on Speakers.
- [BASE] In the Dialog window a reply's text no longer turns into unreadable letters after you change its When unavailable, Only once or Transition choice or reopen the dialog, and the Reply field shows Russian text readably.
- [BASE] The Dialog window no longer counts the sex or class word a tag holds as an extra condition on a cell already sorted by that sex or class, so an original message split by the hero's sex opens and applies; a tag attribute such as tips that a dialogue file cannot hold is now named in the refusal, with an x on its Also in the tag row that drops it from the file, and the never-shown, ending and extra-condition refusals each say which button fixes them.
- [BASE] In the Dialog window, a click anywhere on a line's row selects that line, and the list no longer jumps back to the top after the choice.
- [BASE] The Dialog window now opens and edits a dialogue with many lines at once; before, a 21-line dialogue took about five seconds on every open and edit.

## 0.158.0

### For players

- [ROM2] Slow is now a spell you can cast: a hero loaded from an original save keeps it in his book, the two Necromancer creatures that know it cast it, a map can place it, and a book of Slow teaches it to a mage; before, Slow was dropped from every book.
- [ROM2] Slow is not shown on the spell book page and no shop sells a book of Slow; a book of Slow that you find still teaches it, and creatures that know Slow still cast it.
- [ROM2] Kills of creatures of one kind now count together toward learning their stats, also after a save and a load, and the hero keeps the skill chosen for him as his main skill, so its training is no longer cut; a fighter credited with a spell kill, or already dying when his victim falls, earns no skill for it, as in the original.
- [ROM2] The second game's world map now plays the route sounds while the road to a place grows, and its flags and your party marker move at the original's pace; before, the map was silent and the flags cycled at their own pace.
- [ROM2] The credits of the second game now rise at the original's speed; before, they rose about two thirds as fast.
- [ROM2] Iglez and Dina from a save the original game wrote now show their names in the party and in the shop, as when they join in this game; before, their names were blank.
- [BASE] A unit killed in the middle of a step now finishes that step as a body, without a health bar, and falls on the next cell, as in the original; before, it jumped at once to the cell it was walking to.
- [BASE] An enemy killed by Poison Cloud no longer shows damage numbers rising from its body; the cloud still hurts the living units inside it.
- [BASE] A spell that a mage began on an enemy in range, from his book or with a spell weapon, now hits that enemy even if it ran out of range while he cast, as in the original; before, the mage only waved his hands (and in the second game the mana was lost).
- [BASE] With Smoothing on, the soft edge drawn over a unit, a hero or an object now sits centred on it, instead of a few pixels to one side where it looked like a pale second copy.
- [ROM2] Killing enough creatures of one kind now reveals their stats in the info window for every creature, as for squirrels and bees, and creatures that look alike show the same stats; before, zombies, skeletons, wolves, spiders and the rest stayed blank however many you killed, and look-alikes counted kills apart.
- [ROM2] A Teleport or Drain Life scroll in the pack now casts when you double-click it or press its bound key, instead of doing nothing; before, only the spell book worked.
- [ROM2] The spell book now shows Stone Curse's duration as the second game states it, 480 at power 100, instead of the first game's figure.
- [ROM2] After loading a save made inside a mission, the menu still offers to abandon the mission and return to the town, as it did before the save.
- [ROM2] Creature info cards follow the original's kill rule: knowledge stops at 14 kills, #show map opens every card, and saves keep your kill counts in the original save format.
- [ROM2] Summon and Animate Dead now bring back a unit that has finished dying, as in the original, and Animate Dead raises the corpse that fell first on the cell, not the one with the lowest number.
- [BASE] The mod window has a Towns page: rename a town, change when it opens, and set tavern offers with their talk texts.
- [BASE] A warrior who reads a Teleport scroll now shows the start picture where he stands while he casts, as a mage does, and it no longer appears only at the moment he jumps.

### For map makers and modders

- [BASE] A trigger's dialogue can now be interactive: a dialogue file beside the map holds nodes with up to four replies, each with its condition, outcome and once flag, and the player picks one by key, arrows with Enter, or click; a pick can change gold or a variable and lead to another node.
- [BASE] A mod can give any spell its own picture in the spell book and its own flying picture and burst, in both games. The map editor's Spells page picks the three picture files, or copies them from the other game's install the starter knows; publishing a mod with pictures copied from a game install warns.
- [BASE] In the map editor, every refusal of an edit, a save, a paste, a script change or a mod file read now reads in the editor's language, English or Russian, and a mod file that does not read names its file and line; before, many refusals came out in English.
- [BASE] In the map editor, the test play notes and refusals, the mod folder and file refusals, the spell and terrain page refusals, and an edit the map library refuses now read in the editor's language, English or Russian; a reason the library gives itself still shows in English after the editor's words.
- [BASE] In the map editor, a refusal when adding a speaker or importing its picture now reads in the editor's language, and the question to extend the script always says why, naming the limit and its number; before, some refusals were in English or empty.
- [BASE] In the map editor every scrolled area now scrolls the same way: the wheel works over the content and the bar, a click on the bar's track moves one page, and dragging the thumb moves the content. The item grids, such as the bag window's, answered only the wheel over their cells before.
- [BASE] The map editor has no right panel. A selected object's window always opens as a floating window over the map, and the map gets the freed width.
- [BASE] In the map editor, each item type has one group in the Sacks palette and item lists. Plate boots of every material are together, with separate groups for chain boots and soft boots.
- [BASE] In the map editor, the (?) tips of a unit's equipment rows name the worn and carried items in readable Russian and English.
- [BASE] Close any map from its own tab, scroll long menus and search results, and point at a cut-off tab name or status line to read it in full.
- [BASE] Map editor search opens below the tabs and toolbar. Hover over a cut-off query to see it whole. The toolbar fits smaller buttons or moves a whole group to a second row when it needs more space.
- [BASE] Drag the inner edge to resize a docked map editor panel. The edge and the floating window's corner grip light up when you point at them. With no object selected, Properties shows one status line and puts its help behind the (?) button.
- [BASE] Map editor lists size themselves to their contents and show the full text of a cut line when you hover. The Report window shows the check or save result below the list.
- [BASE] The Map panel fits between the side docks and scrolls beneath its title. Long descriptions and object names show in full on hover, and the docked report keeps save instructions visible and room for up to three findings.
- [BASE] Place units, structures, objects, sacks and spells directly from their palettes. Tick items for the next sack and choose a trigger in the condition window.
- [BASE] Trigger windows fit their contents and keep warnings visible while you scroll. Help marks explain the fields, and hovering over a cut-off choice shows its full text.
- [BASE] In the map editor, question windows such as leaving with unsaved changes or deleting a player are as wide as their text, name the window in the title, and keep what the choice means behind the (?) mark.
- [BASE] In the map editor, the dialog window shows every message's text whole at the width it needs, and its x and + say what they do; the variable window names the variable and shows its shared condition as a link.
- [BASE] In the map editor, sack recipe choices now use the editor's language and show spell names instead of spell numbers.
- [BASE] In the map editor, the Used by links in the unit window's detailed view now wrap inside their column instead of running past it.
- [BASE] In the map editor, a name typed into a unit's window now takes effect when you press Enter or save the map, instead of only when you press Apply.
- [BASE] In the map editor, the variable window is as wide as its longest link text needs, within the screen, and the Used by section title is a heading.
- [BASE] In the map editor, the variables table is as wide as its columns need, within the screen; a trigger name too long to fit ends in "..." and shows whole in a tip, and the title's (?) says that Value is the number in the trigger's condition or action, not the variable's current value.
- [BASE] In the map editor, the placing palette chooses its owner from one drop-down list, the Players window has palette-sized cells with a named legend, and a player's window shows every label whole, its control and relations as words, and its Data section folded.
- [BASE] In the map editor's Groups window, the "no groups yet" hint is now behind the (?) mark in the title.
- [BASE] In the map editor, a structure, sack or other object window shows its picture as the first row, a structure that scripts name is titled by that name, a group window shows its members' pictures, a long Used by line wraps and stays one link, and a Used by list of more than five uses starts folded.
- [BASE] In the map editor, the windows that ask for a value are titled by what they ask for, with what to type and the keys behind the (?) mark; a field with choices shows a drop-down, or a grid of pictures for a recipe's item, a record field is titled by its name in the plain view, a long message shows whole, and every text in these windows is in the game's language.
- [BASE] In the map editor, every page of the mod window is one width with its tabs on one line; pages of things list them left and show the chosen one right with its actions, Remove and Add map sit on the Missions page, the overview counts what each page changes with a link to it and folds the game's own check, folders are chosen in the system folder window, and a long text stays on one line with its whole text in its tip.
- [BASE] In the map editor, the mod window has a Spells page: pick a spell to change its numbers, drag spells between the catalog of both games and the mod's book to place, swap or take them out, and the game run from the mod shows the same book.
- [BASE] In the map editor, the Structures, Objects, Trees, Height, Terrain and Bridge windows are titled by their names with the how-to behind the (?) mark and are as wide as their words need; Forest moved from Objects to Trees, where kinds are ticked in a grid of large cells, Terrain shows the chosen ground beside its list, and the Sacks button opens the window that adds a sack.
- [BASE] In the mod window's Spells page, a catalog spell dropped on a book cell that already holds a spell now takes that cell, and the spell that was there leaves the book and shows in the catalog again; before, the drop did nothing.
- [ROM2] A mod can now change the second game's spell numbers, such as power, damage, range, duration and effect strength, with the same tables as the first game's; a table a spell cannot use is refused by name.
- [ROM2] A mod's `rays`, `heal_hostile` and `self_cast` on the second game now follow each spell's own behaviour, so Prismatic Spray takes `rays` and Heal takes `heal_hostile`, as in the first game.
- [BASE] In the Russian map editor and in mod texts, ё and Ё show as е and Е, and the quote marks « » “ ” as straight quotes, instead of ?, Л or other letters; saved names and labels are unchanged.
- [BASE] The mod window has a Bodies page: add a body from a picture, set its frames, directions and timing, and choose which body a weapon uses.

## 0.157.1

### For players

- [BASE] Whether hitting another side's unit gives experience now depends on whether that side counts you as its ally, as in the original, not on whether you count it as yours.
- [BASE] The `+god` cheat now raises a new hero's elemental protections only as far as his Spirit allows, as in the original and as for a hero from a loaded save, and his card shows them at once; before, they went straight to 100 and the card kept the old values.
- [BASE] The SPELLCASTER and ARMOR PIERCING captions on a character card now stand in the middle of the card instead of at its left edge.
- [BASE] In a mission, a key pressed with Alt held now does nothing unless the original acts on it: F1, F2, F3, Pause, A, Home, End, `=`, `-`, numpad + and -, and 0 on the digit row no longer act with Alt.
- [BASE] Missions now fit 1280x720 without shrinking the text, and use cleaner scaling at 1920x1080 and 2560x1440.
- [BASE] Game and editor text is smooth again, as in the original, instead of stepped.
- [BASE] Two flying units no longer stand in one cell or fly through each other, as in the original; they go round or wait.
- [BASE] A fighter-mage who equips a staff mid-swing casts its spell as the swing ends instead of winding up again.
- [BASE] A new starter option keeps the mouse inside the game window on a computer with two monitors.
- [BASE] A window switched from full screen now fits the screen, so its edges can be dragged.
- [BASE] Game text, such as the name on the new-character screen of the second game, no longer shows a dark blurred rim when the window is larger than the original screen.
- [BASE] Darkness darkens and Light brightens the ground and units under them again, also with the Lighting option off.
- [BASE] A double-click on a spell in the book casts it on the selected heroes themselves when the spell can target its caster; other spells still select and clear.
- [ROM2] Potions raise a hero's Body, Agility, Mind and Spirit only up to the limit for his class and sex (46 to 52), not 50 for all; items still lift a statistic above the limit.
- [BASE] A group a map's script sets to guard keeps the guard distance the map gives it, as in the original, instead of always at least eight cells.
- [ROM2] In a shop, an item put from the hero on the shelf no longer turns into a copy of the previous one.
- [ROM2] A Darkness area in a saved game is still there after loading the game: the covered ground stays dark and sight under it stays reduced.
- [BASE] Hero names and labels in the map editor's windows no longer get a dark rim when the window is scaled; the starter offers a softer picture filter for the window.
- [ROM1] A hired mercenary in a saved city is now stored the way the original stores one, so the original reads the file as its own.

### For map makers and modders

- [BASE] The map editor opens maximized, its mod window opens wide enough to show every page tab in one row, and F5 test play starts the game with the Chicken flag.
- [BASE] The unit window keeps its first column steady when switching views and shows the dialogue portrait under the figure, including speaking animals.
- [BASE] Named units show their own name in the editor card, and F5 applies the card's edits before preparing the test map.
- [BASE] The map editor and its mod window draw a frame faster, so they no longer lag at a large window.
- [BASE] A map now draws only the terrain groups its terrain group mask enables, as in the original; a group missing from the mask is no longer drawn.
- [BASE] The map editor's mod window no longer shows a stray tree picture.
- [BASE] Map editor: a new condition that would clash with a script variable now offers to switch the map to the extended script instead of refusing, and the question shows the reason.
- The map editor's mod window no longer shows the Files page, a click on any tab switches pages from every page, and labels such as "New key" show whole in English and Russian.
- The map editor's message box no longer shows question marks instead of Russian text, for example when a trigger copy is refused.
- [BASE] Mod window: Terrain shows each ground as a picture with its variations on hover, Texts lists each text under where it shows, and Items starts new armour and weapons from a game item picked by name and picture; only names and numbers are typed.
- [BASE] Map editor: the unit window's normal mode is its detailed mode shortened, the name is typed on the character card, and the bag lists gold, then loot, then items. An animal or monster opens the same window as a person, and the card's text has no dark rim.
- [BASE] Map editor: the Units tab places units for a chosen player with no window, the unit window shows corpse drops with the bag and the character card under the identity, and the Players window alone manages players.
- [BASE] Map editor: every question, notice and list window first opens in the middle of the editor, and a window taller than the editor scrolls its rows, then shortens its list, then folds its notes behind the title's (?), so it fits at 640x480; tips show after 300 ms and a (?) at once, a choice list opens inside its window, and each button is as wide as its caption.
- [BASE] A mod can turn a spell off, move it in the book and set the book's size; shops follow, and a spell a hero already knew comes back when the mod is removed.
- [BASE] Map editor: a window dragged below the editor's bottom edge is shortened by pressing its top border (now as deep as the other borders) and dragging down, its rows scrolling inside, and the detailed unit window opens as wide as its columns need instead of the whole editor width.
- [BASE] Mod window: a Characters page adds a guest character with its name in each language and a face, renames or re-faces a game character, and the mission's guest party offers the new characters.
- [BASE] Map editor: the unit card's hit points and mana inputs are one character wider, so the lowest value (-32768) fits.
- [BASE] A person a map places from a character a mod renames now shows the mod's name in its panel, and a speaker built from that character is labelled with it; a person with its own name keeps it.
- A shield or armour picked for a guest character in the map editor is kept instead of dropped.

### For developers

- scripts/check-touched.sh runs the ordinary and release tests a change can reach, selected from the type-checked call graph.
- The release post in Discord is short and links the full notes, and a repeated changelog entry fails the check.

## 0.156.0

### For players

- [ROM1] A withdrawing or retreating unit now measures a large enemy from its corner cell when it picks where to flee, a unit stepping north-east or south-west enters the next cell on both axes at once, and a spell cast given while a loaded game's unit is between cells waits for its step to end, all as the original does.
- [BASE] With -smooth-regen, a negative health or mana regeneration now takes its whole step at the end of each window for every unit, as it already did for some.
- [ROM2] Computer mages no longer cast Slow when they attack.
- [BASE] Computer mages now heal the first hurt unit near them without keeping a mana reserve, and at full mana cast a protective spell from their book on a unit of their own side.
- [BASE] Light no longer widens anyone's sight: creatures in a Light cloud see less far while humans and heroes are unaffected, and Darkness now shortens only the sight of humans and heroes, leaving creatures as they are.
- [BASE] A scroll picked in an open spellbook now stays picked after each cast, like a spell, until the scrolls run out.
- [BASE] A unit sent to a cell that an enemy then steps onto now stops next to the enemy and attacks it, instead of standing and waiting.
- [BASE] A scroll cast on an enemy no longer sends your character walking to the enemy's spot: it steps only as close as the spell needs, casts, and stands, even when the enemy is right next to it.
- [ROM2] Poison Cloud no longer hurts a unit the moment it first hits; the damage now starts with the first poison tick, as in the original.
- [ROM2] In the Kaarg inn, a selected hire seat now shows each new animation frame at the same moment as in the original game.
- [BASE] Item descriptions now write elemental damage without a colon, for example "Fire 5-8".
- [BASE] Map editor: the basic view of the unit window now lets you set a unit's maximum health on the card.
- [BASE] Map editor: the health and maximum health boxes on the unit window's card now hold five digits.
- [BASE] A mod names a spell by one name in both games, and two mods' rows for one spell combine field by field.
- [BASE] A unit sent to a cell covered by any part of a large enemy now stops beside it and attacks, as it already did when sent to the enemy's own cell.
- [BASE] After a cast the caster waits as long as in the original: in the first game reading a scroll no longer adds the spell's Complication to the wait, and in the second game the wait keeps its low byte instead of stopping at 255.
- [ROM1] A walking unit now holds its cell until it crosses into the next one, so a lasting spell area slows units walking through it as in the original.
- [ROM1] A hero who has fallen keeps the same look after you save and load the game.

### For map makers and modders

- [BASE] Map editor: a map's draft pages and labels now count and move with the map whatever their letter case or folder depth.
- [BASE] a data file in a mod loads by itself, with no script line; main.star is only for rules and settings, and the docs and example mods no longer call game.data.add.
- [BASE] The map editor's unit window opens small with the Kind row or wide with every field and no tabs; it shows the unit's figure at game size, a facing pad with eight directions and random, maximum health, gold dropped and a drops-loot switch.
- [ROM1] The mod window gains a Terrain page: add, edit and remove tile groups 13 to 31, import their four strips checked as the game reads them, paint a new group before saving, and publish it with no main.star line.
- [ROM1] The mod window gains a Texts page: edit each text key in English and Russian, see a key one language lacks, and pick a mission's title and briefing from a key.
- [ROM1] the mod window has an Items page: add armour and weapon rows, change or switch off shipped rows, set material masks and import item sprites, with undo, redo and save
- [ROM1] the editor's mod window has a Files page: replace install pictures, sounds and texts per language, checked by the game's rules, with the overrides of the enabled mods
- [ROM1] Map editor: the Files page no longer lists override lines for a mod that is not enabled in the starter; it says so in one line instead.
- The trigger window no longer names a variable "Constant / variable preset"; it shows "Variable N" unless the map's author gave it a name.
- [BASE] Every page of the mod window and every editor window explains itself behind a (?) mark, editor windows first open centred, and the Terrain page lists every ground and adds a new one from a single picture.

## 0.155.0

### For players

- [BASE] Change one saved Smoothing setting in Game Options or with Ctrl+O across game screens.
- [ROM2] P now sends the first selected hero who sees a sack to collect sacks one after another, and he holds his ground when none is left in reach; heroes sent to the same sack no longer wait for each other.
- [BASE] Mouse wheel zoom in missions moves 5 percent per notch up to 200 percent and 25 percent per notch above it; the map editor keeps its old step.
- [BASE] F2 now quick saves and F3 quick loads in both games; Ctrl+F2 and Ctrl+F3 open the save and load windows, and Shift+F4 and Shift+F9 no longer save or load.
- [BASE] Both games: the mission menu has a Keys row; pick an action, press its new key (Shift, Ctrl and Alt allowed, Esc cancels); a key another action held moves to the new one, Defaults gives the game its own keys back, and each game keeps its keys in your profile, never in a saved game. The mission menu's frame now fits its rows with even room above and below.
- [BASE] Both games: F4 to F12 are nine shortcut slots, each a spell or an item: Shift with the pointer over a carried item binds it and the item shows its key, Ctrl binds a book spell (one you hold only as scrolls too); the key puts on, drinks or readies what it holds, as the inventory would. Your shortcuts are kept in your saves. In the first game F5 to F8 keep your spells and your own keys; the grid, readout and frame rate keys move to Ctrl+Shift+F10 to F12.
- [BASE] On the Keys page the mouse pointer no longer freezes while the game waits for a new key.
- [BASE] F2 quick saves and the timed autosave are kept per game (rom1-/rom2- files), so one game no longer rotates away the other's saves, F3 loads the running game's newest quick save, and each game's Load list shows its own; an older unprefixed quick or timed save stays loadable and is never overwritten.
- [BASE] New launch option -smooth-regen: health and mana come back in many small steps instead of rarer larger ones, at the same overall speed, the faster recovery after a rest included; off by default.
- [BASE] The starter has a "Smooth regeneration" check box, off by default, that starts the game with -smooth-regen.

### For map makers and modders

- [BASE] The starter's Missions list now shows the missions a mod replaces with a map named by its number (10.alm shows as "10. replaced by the map 10.alm"), where before it showed none. A mod's applies-to now takes rom2 for every second-game install, and the map editor offers it.
- [BASE] The map editor's campaign graph stands missions in rows by tens, colours each kind of node with a legend, and shows the start film and the victory films the game will play.
- [BASE] Opening a mod now uses the system's file window everywhere: in the map editor's mod window a click on Working folder, Enter in it, Ctrl+O and Open mod all open it, and it no longer fails before the first mod is saved; the starter's mods folder has a Browse button.
- [BASE] In data/towns.toml a mod sets which mission a town's shop and school hand out ([[shop]], [[school]]), and a talk or hand-out can wait for wins (won) and stand from a chapter on (from); the editor's campaign graph edits these mod hand-outs (add, change, switch off, remove), and Ctrl+S writes data/towns.toml keeping its comments.
- [BASE] In the map editor's mod window the Overview page scrolls to its end, the Campaign graph buttons show their words instead of question marks, and the Missions page lists every mission of the mod.
- [BASE] The map editor's help lists Ctrl+Tab and Ctrl+Shift+Tab (next and previous open map) only while two or more maps are open; with one map or none the two lines are left out, and the keys work as before.
- [ROM1] F5 test play from the map editor now starts the map with the hero and party chosen in the party window, with no character generation; the window remembers the choice between editor runs, and a mission with its own party gets a Party: Standard / Custom switch.
- [BASE] Campaign graph: films now stand in their own first column, a little below the row of the mission they follow, with no arrow and no extra row.
- [BASE] On the campaign graph, where two arrows cross, the upper arrow now steps over the other in a small square bump.
- [BASE] Map editor Units window: the tabs are now "Humans and Creatures" and "Named characters", a new "Humans" row lists every non-hero human sprite type, and pointing at a human shows their portrait or full figure.
- [BASE] Map editor mod window: the Maps tab is gone (the Open window still lists the mod's maps), the Files list on the Overview tab starts folded and opens with a click or Enter, and pressing a tab while a confirmation waits cancels it and shows that tab.
- [BASE] Map editor F5 "who plays the map" window: on Standard it no longer shows the character boxes that could not be changed, and every character box stays pressable after the window was made narrower or switched from Standard to Custom.
- [BASE] Map editor campaign graph: cards stand further apart, each town lists its hand-outs side mission first, then main mission, then none, and the lines no longer cross or run close together.
- [BASE] The map editor's mod window lists each map no mission uses on its Missions page, with Make mission and Replace, and each map that replaces a mission by its file name, with Make row; "Add map without mission" copies the open map into the mod, "Add folder of maps" adds every map of a folder, and F5 on an unused map plays it as a guest mission with the mod active.
- [BASE] Map editor: F5 now plays a mod's unsaved maps, so a map added from a folder or made a mission needs no Ctrl+S first, and the mod window's bottom buttons stay on screen at 640 by 480 when its page is scrolled.
- Add folder of maps now carries a map's pages folder whatever the case of its name.
- The map editor's test play of a mod on a second-game install now names the second game.
- On the campaign graph each town is an area with its name in the top-left corner; its tavern, shop and school cards stand in columns with their missions under them, the main mission's card first, and a main mission's arrow runs straight down to the next town's area.
- A map file a mod's draft deletes under another letter case now leaves the mod's file list too.
- An open map carried into a mod draft keeps the pages stored inside the map.
- [ROM1] The map editor and its test play now treat a replaced mission whose AutoGetMission is -1 as opening the town, as the campaign graph and the game do, so a guest mission can follow it.

## 0.154.0

### For players

- [ROM2] Enemy people with a spellbook back away when hurt with a foe beside them and flee at a quarter of their health, and enemy archers back away below half; a small enemy guard group that sees little chases from farther away, and a summoned creature guards the place where it appeared.
- [ROM2] Wounded characters and creatures no longer all regain health at the same moment; each heals on its own turn among four, as in the original.
- [ROM2] A creature with a strong damage row now hits as hard as in the original, and a mage who has to turn toward his target pays the mana for his spell once he faces it, not at the moment he is ordered to cast.
- [ROM2] A game saved in Againrom and then loaded in the original keeps characters and creatures blocking one another right after the load.
- [ROM2] In the Druids' inn the patron who becomes a hired orc shows as the orc from the start, not as a two-handed swordsman.
- [ROM2] A game saved in Againrom and then loaded in the original marks the ground under buildings the way the original's own saves do.
- [ROM2] A unit at zero health or below no longer restores health or mana, even one a map made undying.
- [BASE] The starter, the game and the map editor share one mods folder: the editor publishes a mod there with no path typed, and the starter lists a ticked mod's missions and starts the chosen one.
- [BASE] Map editor: opening the mod window leaves the keyboard with the map, and mission rows show their text and Open action without an empty drop-down.
- [BASE] A mission map zoomed with the wheel is scaled once, with no double smoothing; health bars and selection frames grow with the map and are even. With Smoothing (Ctrl+O) off the mission picture is drawn in hard pixels.
- [BASE] With Smoothing off in a mission only the map keeps hard pixels; the interface and its text are scaled by the starter's filter again, so letters no longer get uneven strokes.
- [BASE] At a Windows display scale of 125, 150 or 200 percent the game draws its picture directly in screen pixels and scales it once: no second blur, a click lands where it is drawn; the map editor keeps the size of its panels.
- [BASE] At a Windows display scale above 100 percent, the strip at the screen edge that scrolls the map is as wide as it was before.

### For map makers and modders

- [ROM1] W saves a mod mission's map to `maps/<name>.alm`, or `.almx` for an extended map, with its pages and labels beside it, moves a mod in an older layout there on its next save, and edits the starting hero's class and sex; before, an extended map saved into a mod was named `.alm`.
- [ROM1] The mod window (W) has a Campaign graph page: the start, every mission, every town and what its tavern, shop and school hand out, with arrows for what follows what. With a mod open it marks the missions and arrows the mod changes, and a click on a mission opens its map. The page only shows the campaign and changes nothing.
- [BASE] In the map editor, the trigger window's Copy (also Tools, Copy trigger) makes a copy of the selected trigger right after it, named with "(copy)", which you then change like any other trigger; Undo removes it.
- [ROM1] A save made in town loads after you edit or update one of its mods or change a mod setting. Items, skill levels, clothing layers, guests and campaign missions the changed mods no longer hold are taken away, and the Load page lists them; items the mods still define stay yours. A save made inside a mission, or under other mods, is still refused.
- [BASE] Map editor: the mod window (W) has a Maps page that lists every map of the mod with what the mod does with it (replaces a mission, guest mission, unused, missing) and opens it; File > Open leads with the same list while a mod is open, and Ctrl+Tab / Ctrl+Shift+Tab switch between the open maps. The mod check now warns about a map no mission uses.
- [BASE] Map editor: missions that a mod replaces with a map file named by the mission number (for example 10.alm) now appear in the mod's mission list and open from the campaign graph with a click.
- [ROM1] The map editor's File menu and its save window have the switch "Texts inside the map", off by default: on, the briefing and message pages are saved inside the map, which becomes an `.almx`, and the game, the mod check and a packed mod read them from there. A save that would leave such a map's pages in neither the map nor its folder, or write a map too large for the game to read its pages from, is refused and writes nothing.
- [BASE] In the map editor, a copied trigger that shows a message now gets its own text page with the same text, so changing the copy's text no longer changes the original's. A trigger with a drop zone is not copied; the editor says why.
- [BASE] In the map editor a picture or a town description can be added to a mod by picking a file: the editor puts it into the mod's folder, checks it at once and writes it on Ctrl+S; the mod check now warns about files the game does not read and about files kept outside their place.
- [BASE] Map editor: with a mod open, several of its maps stay open at once as tabs under the menu bar; each keeps its own view, selection, unsaved changes and undo history.
- [BASE] Map editor: the mod window's mission rows now show their text in Russian instead of question marks, and while the mod window is open you can still zoom, scroll, select and paint on the map beside it; click the map to give it the keyboard.
- [ROM1] Map editor: F5 asks who plays the map: the hero and the five companions of the campaign, with "Skills 100" and "All spells"; the window names the hero's kind, and the cards show the companions as the game builds them beside that hero.
- [ROM1] Mods: a party member from an npc.reg record with a one-row definition (npc25 Brian, npc26 Rood Glaen and others) plays under the record's own name, not under the name of its class and sex archetype.
- [ROM1] Mods: a mission that names class or sex, started by its number, refuses a hero of another kind.
- [BASE] Map editor: with a mod open, the Campaign graph page of the mod window now edits the campaign. Pick a mission to set which mission follows it at once, choose its party, or insert a new mission that uses a map already in the mod. Each change shows in the graph right away, is one Undo step, and is written when you press Ctrl+S.
- [BASE] With the programs inside the game folder, the map editor saves a mod into Againrom\workbench and publishes it into the mods folder beside the programs.
- [BASE] Map editor: winning a map started with F5 brings the hero to the campaign town again instead of the map list, and the key help shows Ctrl+Tab and Ctrl+Shift+Tab on two lines that fit.

## 0.153.0

### For players

- [BASE] Abandoning a mission restores the backpack and town bag exactly as they were before entering it, with every item in its original place.
- [ROM2] Use Ctrl+F4 through Ctrl+F12 to bind nine spell shortcuts, and F4 through F12 to recall them. The first game keeps its existing keys.
- [ROM2] Quick save is Shift+F4 and quick load is Shift+F9, because F4 through F12 now recall spell shortcuts.
- [ROM2] Holding Ctrl with G, D, C, S, T, R or M no longer also issues that letter's ordinary order.
- [ROM2] A defender standing too close steps away along the line between the two characters, with the destination kept inside the map.
- [ROM2] After unlocking cheats, use Ctrl+Shift+F10 for the grid and Ctrl+Shift+F11 for the debug readout.
- [ROM2] Press P to send your selected heroes to pick up a nearby sack, even when they are not standing on it.
- [ROM2] Press L in a mission to make your selected units stand still: fighters do not attack or strike back, and mages heal and protect your side from where they stand. Any other order ends it. The first game keeps its L key as before.
- [BASE] With Smoothing on (Ctrl+O), the mission map is drawn smooth when you zoom it with the wheel, without blocky pixels or flicker while it scrolls, and each turn of the wheel shows the zoom in percent; before, a zoomed map looked blocky and nothing showed the zoom.
- [BASE] The debug keys K, L and Shift+F4 no longer act in a mission by accident. Kill, chip and map reveal moved to Ctrl+Shift+K, Ctrl+Shift+; and Ctrl+Shift+F4, and work only after the cheats are unlocked.
- [ROM2] The shop of a town has new goods each time your party comes back to the town from the world map, as in the original; before, the goods stayed the same until you slept at the inn or loaded a save.
- [ROM2] A mage casts at a unit from as far as the spell reaches (7 cells for Fire Arrow or Drain Life, 6 for Heal, 5 for Curse), not only as far as he sees himself; before, an order onto a target 6 or 7 cells away was ignored.
- [BASE] The cell grid (F10) and the debug readout (F11) now work in a mission only after the cheats are unlocked; before, any press toggled them.
- [BASE] The mission menu now lets you abandon an unfinished mission and return to its town without keeping the mission's gains or losses.
- [ROM2] A mage casting by himself picks a target any of your units sees, not only one he sees himself, and casts at it from as far as the spell reaches; a target none of your units sees is not picked. A cast you order is not changed.
- [BASE] A shop sells every scroll in a stack of 51 to 100 at the same price per scroll; before, a shop held only one or a few of a scroll.
- [ROM2] A mage casting by himself also picks a target that a unit of an allied player sees, not only one your own units see.
- [ROM2] Town sound and speech no longer go silent after you have visited another town in the same run: the square, the inn and its talk, and the shop keep their sound in the later town, and in a town you load after another.
- [ROM2] The mission menu shows the whole "Abandon, return to town" line in English and Russian, and the window widens for it; before, the line was cut to "Abandon, return to to...".
- [Starter] The mod settings panel shows every setting of a mod: turn the mouse wheel over it to reach the sixth and later ones. Numbers get - and + buttons that stay within the allowed range, and a value the game would refuse is marked with the reason; before, only the first five settings could be set.
- [BASE] At the fastest game speeds large maps stay smooth instead of dropping to a few frames a second.

### For map makers and modders

- [ROM1] The mod window (W) has an Open mod button that opens a mod folder, or a packed mod read-only, after asking about unsaved changes. Its buttons and labels show their whole words in English and Russian, its messages are in the game's language, and it moves by its title bar and resizes like the editor's other windows. Buttons in the other editor windows also widen to their words.
- [ROM1] W edits a mod’s version, author, description, target base and dependencies. Adding a speaker can be undone and is saved with the project; new packages and base dumps need no script. Saving keeps files the editor did not change and leaves a handwritten mod handwritten.
- [ROM1] Mods can choose the next mission, insert missions for the main party, and give a campaign mission to a guest hero or party. Saving and loading keeps the route and the waiting main party.
- [BASE] A mod loads its held data files without a script. Check warns about files outside the fixed tree and old paths; publishing leaves unread files out of the pack and save mark while keeping every file its rows read. Old row paths and scripts still work. Speaker data now enters the save mark even without a script call.
- [ROM1] A mod mission whose next mission is an earlier campaign mission no longer sends the campaign back to that chapter, so the town offers the same missions before and after you save and load it.
- [ROM1] In the map editor, after Add open map, saving the mod into a folder that does not exist yet creates the folder and saves; before, the save failed with "cannot find the file".
- [ROM2] The base dump of the second game loads back as a mod that changes nothing: its three towns are listed as comments for reading. Before, loading the dump as a mod was refused.
- [ROM1] In the Russian map editor at 640x480, the mod window's Close button is on the screen again, and the person window's buttons no longer cover the last lines of the character's stats.

## 0.152.0

### For players

- [ROM2] Loading a save keeps every skill of your heroes at the level it had when saved; before, with the new skill growth, a skill whose experience had not yet reached the next level rose by one level when the save was loaded.
- [BASE] A room tip, the character generator tip and the mission tip now close on a click anywhere on the painted Close button; before, a click on the top few rows of the button did nothing, however often it was repeated.
- [ROM2] Experience falls off for a kind of enemy your hero keeps killing, as in the original: from his third kill of the same kind, each kill and each blow against that kind pays less (about a tenth less at the third kill, a little less with each kill after), a kind that is not among the last five he killed pays in full again, the count starts over in each new mission, and a saved game keeps it.
- [BASE] Automatic casting puts Bless (Luck in Battle) on a creature with mana again, such as a dragon or an orc shaman, and such a computer creature that draws Bless for itself casts it on itself; human mages, necromancers and druids are still skipped.
- [ROM2] Skills grow as in the original: a hero's main skill and his fifth skill (shooting, or astral magic for a mage) take the whole experience of a blow or a spell, every other skill takes an eighth of it, and a skill rises only when its experience reaches the next level's mark; before, every skill took the whole amount and rose one level at the first experience it earned. A warrior no longer gains weapon skill from a weapon's spell damage, and a blow on a unit already at zero health pays nothing.
- [ROM2] Summoned and animated creatures show their own portrait, including after loading an older save, instead of an empty picture or the first picture of their kind.
- [ROM2] A weapon's magic damage (fire, water, air, earth or astral) is now dealt only by a blow that hits on the attacker's skill and the roll; a lucky blow that lands only because the roll was very high, or a blow by a creature that always hits, deals its ordinary damage without the magic part, as in the original.
- [ROM2] Casting an effect spell again on a unit that already holds it keeps the larger strength and the longer remaining time of the two casts, so a weaker Haste, Shield, Protection, Bless, Curse or Light no longer replaces a stronger one. For Slow and Darkness the effect nearer zero is kept: a weaker repeat eases a stronger one and a stronger repeat does not deepen a weaker one.
- [BASE] A unit standing in a Darkness cloud sees less far for as long as it stands there: the map shows a smaller visible area around your own darkened unit and fewer of the units near it, and a computer unit in the cloud notices your units only from closer. Light makes the sight larger the same way. Before, Darkness and Light changed nothing in the second game, and in the first game a cast at an unlucky moment changed nothing on the map.
- [ROM2] Lifting a worn item off the shop's doll now redraws the doll without it at once, and it returns if the lift is cancelled; the first game already did.
- [BASE] Automatic casting no longer casts Bless (Luck in Battle) on a mage, the caster included: your mages bless a warrior instead or cast their next spell, and a computer mage that drew Bless for itself casts another spell or attacks. Your own order still casts Bless on any mage.
- [ROM2] A staff's card states the staff's own range (6 for a Staff, 5 for a Shaman Staff) instead of its spell's, and a mage attacking with a spell staff starts casting at that range, counted from the nearest cell of a large target.
- [BASE] A mage attacking with a spell staff casts as soon as the target is within the staff's range, even when the mage does not see it, instead of walking closer until it does.
- [BASE] A quick click on the Close button of a room's tip panel (the shop's, the tavern's, the school's and the town square's) closes it at the first click; before, a click fast enough to press and release within one frame left the tip open until the next click.
- [ROM2] A hired druid or troll shows its own portrait and colours in a mission, and the druids and necromancers of a mission keep theirs after a save is loaded, instead of the first picture (for a druid, the young woman in a dress of leaves).

### For map makers and modders

- [BASE] The map editor's test play and a mod load no longer refuse a map that carries the extended script record under an .alm name; they start and show a warning that the original game does not run its triggers under that name. Name the map .almx to remove the warning.
- [BASE] Publishing a mod keeps long map, folder and data file names, including Cyrillic names, up to 255 UTF-8 bytes per name. A packed mod keeps the same files and save compatibility as its working folder.
- [ROM1] The map editor's mod window (W) now opens any mod, also one with no missions or a packed mod file, which opens read-only, also with no map open; an Overview page lists what the mod holds, what the game reports for it, its files and its warnings, and the mod's terrain, characters and items are in the editor's lists.

## 0.151.1

### For players

- [BASE] The starter lists the enabled mods in the order the game loads them, and a mod's row says why the game would refuse it, such as a mod made for the other game, a missing required mod or two mods that conflict, before you press Play.
- [ROM2] Winning a campaign started with New Game now adds your hero to the Hall of Fame with a score, and loading a save made in such a campaign keeps the time the score counts.
- [BASE] In a city shop, double-clicking a wearable or usable item that came from an original save now uses it; before, the second click was refused.
- [ROM2] A unit loaded from an original save with an unfinished leftover action no longer refuses queued spells and pickups for the rest of the mission.
- [ROM2] Standing units keep the normal target restrictions when only fallen enemies remain nearby.
- [ROM2] Computer units look around while waiting at their posts and finish turning before choosing another direction.
- [ROM2] Units whose route ends without a step can turn to a nearby enemy; computer units with no enemy look around instead of keeping a stale movement order.
- [ROM2] Standing units choose nearby enemies by distance and facing. Units finish their last step before switching from a move to an attack.
- [ROM2] A save from an earlier version that the English game refused to open with an error about a unit definition now opens and plays on.
- [ROM2] A save no longer loses the unit type of a monster group: its units keep their kind and name when the game is saved and loaded.
- [ROM2] Winning the campaign's last mission now ends the game: after the closing movie come the credits and the Hall of Fame, then the main menu, instead of the world map and more play.
- [BASE] A unit that dies no longer keeps the glowing rings of a spell such as Protection, Bless, Shield, Poison Cloud or Curse drawn over its body.
- [BASE] A unit that picks among several targets by itself now favours or avoids one kind of target only when the originals do: in the first game that depends on the target's size and Mind, in the second game on the size and Mind of the unit choosing.
- [ROM2] A weaker Poison Cloud no longer shortens the poison already on a unit: the unit keeps the longer remaining time. Recasting the same spell still does not make the poison last longer.
- [ROM2] A hero caught in an enemy Darkness keeps following your orders and no longer runs at the enemy mage by himself.
- [ROM2] A scroll in the pack shows in the spell book at its spell's cell: pick it there and cast it as from the pack, on the hero himself too. A hero with no spells of his own reads scrolls from his book.
- [BASE] A mod can hold its own speakers: in the map editor, clicking a unit no speaker shows on a map in a mod offers to add one with your own name and picture.
- [BASE] The download is smaller: the game, the starter and the map editor no longer carry debugging data.
- [Starter] The starter is a much smaller file.
- [Editor] The map editor now opens from the game program; the separate mapedit program is gone.
- [Editor] To open maps from Windows, associate .alm files with againrom.exe again; opening a map starts the map editor.
- [ROM2] Winning the campaign after loading a save now adds your hero to the Hall of Fame with a score.
- [ROM2] A computer-controlled mage no longer heals neutral monsters, such as an ogre attacking your party. It heals itself, its own side and its allies.

### For map makers and modders

- [ROM1] A mod can replace a campaign mission's map just by holding the map under the mission's number, such as 10.alm, in the mod's folder or in its maps folder; no mission list and no script are needed. The mission's title and briefing stay the game's own.

## 0.150.0

### For players

- [ROM2] The starter no longer says that original saves are refused, that only the first town and mission 10 work, or that the hero's save record is not written; its note for a Rage of Mages II base names the base alone.
- [ROM1] A creature at full health no longer backs away from a nearby hero. It may back away after it has been wounded.
- [ROM2] In the mission Lost Caravans («Исчезнувший караван») and every other map that has them, spell towers now fire at your party when it comes near, and a destroyed tower stops firing. Your saves keep which towers still stand.
- [ROM2] A creature at full health no longer backs away from a hero who comes near: it stands and fights, and may back away only once it has been wounded.
- [ROM2] A weapon with an attack enchantment states its own damage again on its description, above the spell's lines.
- [ROM2] At the lowest difficulty creatures on the map now have half their health instead of two thirds, as the original's rule gives. A mission save now stores each creature's withdrawal and flee points; in a save made earlier by this game they are empty, so creatures in it stay and attack, as the original game plays that same file.
- [ROM2] A save made on the Empty Slot of the save page is now a file named after the name you type, such as `Camp fire.sav`, instead of `game0003.sav`. A name already in use gets ` 2`, ` 3` and so on, and a name with letters a file cannot hold is adjusted.
- [ROM2] The teleport in the mission with the traveller now works: a unit that steps on it appears on the far bank, so the traveller can be brought across and the mission finished, also from a save made after the teleport opened.
- [BASE] A grave no longer shows a use cursor or a beehive picture when the pointer rests on it: the cursor is the ordinary one and a click just walks there. Wells, switches, shops and the saving place keep theirs.
- [ROM2] A town save no longer hangs while the shop fills when the shop stock multiplier is set high.
- [ROM2] At the Druids' inn the fourth guest now shows his own picture, his Druid card and his figure, as in the original.
- [ROM2] The mission Forest Madness («Бешенство») no longer ends a moment after it starts: the dying Chief Druid now waits for you to heal him.
- [BASE] W now remembers the previous weapon after you load one of your own saves, so the swap works straight away. A save from the original game still starts with nothing to swap back to, and a weapon you no longer carry stays gone.
- [ROM2] A mission save resaved by the original game now opens when one of its units has more health than its maximum.
- [ROM2] Bats and other creatures that backed away from the hero no longer stand idle for the rest of the mission after a load: they pick a target again and attack.
- [ROM2] A staff of Stone Curse now shows, as the original does, that the spell lasts for a varying time up to a longest value, for example "Duration 0.0 - 6.8", instead of one fixed time.
- [BASE] On macOS, Command now works where Control does in a mission: Command+1 assigns the selected units to squad 1, Command+2 to squad 2, and so on. Control still works.
- [ROM2] An Elven Scroll now works like an ordinary scroll at its own greater power: its description names its spell, and any hero can read it to cast that spell. Before, its description showed only its name and no hero could read it.
- [ROM2] Drain Life's picture on its target now starts on the same frame as in the original game.
- [ROM2] A creature hit by a Darkness its own side casts now keeps fighting its enemy: in mission 10 the troll under its bats' Darkness keeps hitting the hero instead of turning on the bat.
- [BASE] Enemy dragons' and mages' Fire Ball, Wall of Fire and Poison Cloud now land where the target stands when the spell goes off, a fleeing caster stops casting, and in the first game a caster out of mana attacks instead of standing still.
- [ROM2] The Talk button on a hire seat now opens that person's description.
- [ROM2] Our mission saves now also give each creature and person the map places on your own side the number the original game knows it by, as they already did for the other sides.
- [ROM2] The starter's shop stock multiplier now also fills the second game's shops with more items.
- [ROM2] Mission 42 mage hoods and other shipped units now take the colour their map gives their side, so the hoods are brown as in the original.
- [BASE] Cheat replies now use the install's letters and put spaces around the player's name.
- [ROM2] Kaarg's town square now plays background voices, birds and the guard's footsteps.
- [ROM2] Visiting the inn can now bring back a missing Iglez or Dina at the appropriate point in the campaign, including after loading an older save.
- [ROM2] A hired companion who dies in a mission no longer comes back for hire at the inn after his rest; a companion who survives still rests one mission and can be hired again.
- [ROM2] The message line now names a picked-up or received book, magic item or piece of equipment by its name instead of by a number, and the item card shows the same name.
- [ROM2] In the mission "The Astrologer" the granddaughter's lines now show Dina's portrait, not a man mage's.
- [ROM2] Drain Life's sparks now start moving at once and last as long as Heal's, and the Drain picture no longer stays on the target.
- [ROM2] The Crown of Yazes from missions 70 and 80 no longer disappears after the victory when it is not worn: in town it is in the bag, or on the strip of the companion who received it in the shop, and a town SAVE and LOAD keeps it.
- [ROM2] A mage with no staff worn no longer shows a damage row on the character card; before, it showed the fighter damage such as 8-16 or 0-0. A mage with a staff keeps its row.

## 0.149.0

### For players

- [ROM2] Heal and Drain Life sparkles now surround a large creature's body instead of gathering at its upper-left corner.
- [BASE] A creature that has only one set of colours now shows those colours at every level instead of the picture's default colours, and a unit turned to stone by Stone Curse now has the original's shades of grey.
- [ROM2] Units on the map no longer all face north when a mission starts: each one, and each creature a Summon or Animate Dead spell brings, starts facing its own direction, as in the original, and keeps it through saving and loading.
- [ROM2] Saves of mission 42 made before 0.148.20 now load with their bridges in place.
- [ROM2] On the English install the map's wooden bridges now appear and can be crossed, and a building that cannot be damaged, such as the crypt on the Dragon Killer map, is drawn whole instead of as ruins; a save made inside such a mission before this fix keeps the missing bridges, so enter the mission again from a save made outside it.
- [ROM2] Computer-controlled mages with Heal and Shield now protect themselves with Shield while idle when they have enough mana.
- [ROM2] Slain monsters now leave items in the sack beside their gold, as in the original: dragons nearly always a staff, book, scroll or potion; ogres, orcs, the undead and necromancers now and then an enchanted weapon, shield, armour or magic item. Goblin spearmen now drop gold too.
- [ROM2] Some town shops now offer the correct selection of magic goods and weapons.
- [ROM2] In the shop, items you lay on the table stay there when you switch to another party member, so you can pass items between members through the table; leaving the shop or saving puts every item back in a pack or on its shelf, where before a save made with items on the table lost them.
- [ROM2] In the objectives window, the objective rows now start right below the last line of the briefing instead of leaving a gap of empty lines.
- [ROM2] In the shop, an item you take from the shelf now leaves the shelf at once instead of staying behind as an item that cannot be taken or bought; putting it back returns it, and after a purchase the shelf shows only what the shop still holds.
- [ROM2] In the shop, switching to another party member now shows that member's items, and buying, selling and the table act on that member.
- [ROM2] In the shop, two equal staves, two equally enchanted items or two copies of one book in a party member's bag show as one stack again, for the hero and for a companion.
- [ROM2] In the shop, putting an item on a character plays the put-on sound and taking it off plays the take-off sound.
- [ROM2] In a mission, putting an item on a character in the inventory plays the put-on sound, and taking it off or dropping it plays the take-off sound.
- [ROM2] Our saves load in the original game without it stopping on a hero who was attacking; before, the original stopped a moment after loading such a save.
- [ROM2] In our saves loaded in the original game, a hero you had sent somewhere goes there and stays; before, he walked off towards a corner of the map.
- [ROM2] A hero who changed his weapon before you saved keeps his real damage when you change weapons again after loading that save in the original game; before, the original could show him with damage near 243-485.
- [ROM2] Our mission saves now give each creature and person placed on the map the number the original game knows it by. In such a save loaded in the original, a wounded dragon is expected to show its real health at once and to stay visible when it comes back into sight; before, it showed full health until its next hit and could turn invisible.

### For map makers and modders

- [ROM1] Choose sack items by their inventory pictures, grouped by item type and material, in both the sack and carried-item windows.
- [ROM1] The map editor's unit window now shows the face the dialogue would draw if this unit were the speaker, under its "Dialogue speaker" line; a unit no speaker shows gets the reason in words instead of a face.
- [ROM1] In the map editor, choosing a dialogue speaker "on the map" now picks a unit that stands under another unit's cell: a click on a small unit inside a larger unit's outline takes the larger unit as the speaker when the small one shows no speaker, instead of refusing the click.
- [BASE] The map editor draws faster, and typing a name in the unit window no longer stalls on every letter; the unit picture refreshes when you pause typing or press Enter.
- [BASE] In the map editor's unit list, people are sorted into story heroes, player characters, mercenaries, and other fighters, mages and riders. Each picture is the figure the game draws on the map. Each level of a mercenary is its own entry, a type's levels one after another, and his unit window also has a level from 1 to 4.
- [ROM1] The map editor now draws a placed person on the map with the figure the game draws for him; for example the peasant in mission 30 at cell 62,18 is a man with a club, not an axeman.
- [BASE] The map editor's status line shows "Saving..." while a map is written and "Saved: file name" afterwards for a few seconds, or the reason the save failed (EN and RU).
- [BASE] In the map editor the Traps and spells tab now stands before Buildings, in the top bar and in the tree.
- [BASE] A unit's own name can now hold a space between words ("Old Bob"); before, the space typed after a word was lost when typing paused.
- [ROM1] A map can now be an `.almx` file whose triggers fire when any of several groups of conditions holds, with up to 64 conditions and 64 actions per trigger, 4096 variables and 10000 triggers; the game lists, plays, saves and loads it, and a mod mission may name one. The map editor does not open such a map yet.
- [ROM1] The map editor now opens `.almx` maps and edits their triggers: groups of conditions joined by OR, up to 64 conditions and 64 actions, variables past 99. On an `.alm` map the first edit the original trigger cannot hold asks once whether to make the map an `.almx`, which the original game does not open; such a map saves as `.almx`, and one that fits the original triggers again can be saved back as `.alm` after a question. Saving back keeps Cyrillic trigger names and lets a trigger with no conditions run its actions in the original game.

## 0.148.0

### For players

- [ROM2] A town shop shows its room and its goods again and buys and sells; after New Game the shop page showed no shop picture and nothing could be bought.
- [ROM1] A save of the original game made while a mage recovers after casting from a spellbook, saved again at once, keeps that cast: the mage still finishes the recovery before the next spell.
- [ROM2] Enemy casters out of mana for their spellbook strike with their staff instead of standing and waiting for mana, and every caster turns to face its target before it casts.
- [ROM2] The original game now loads our saves with necromancers that still cast spells from their staff and their spellbook.
- [ROM2] Quest Objectives lists each opened goal with the original's box, tick or cross, and a finished goal plays a sound instead of showing a caption. A save made in a mission now keeps the goals opened there.
- [ROM2] Escape on the world map skips the current journey, or leaves the map open when no journey is running.
- [ROM2] The village elder shows «Староста» (Elder in English) instead of Peasant, and hired mercenaries show «Наемник» (Mercenary) as in the original.
- [ROM2] Necromancers and hired troops show their translated unit names instead of blank names or internal labels.
- [ROM2] Loading and immediately saving a mission no longer changes an idle hero's orders.
- [ROM2] A monster that cannot reach you steps back for a moment and then attacks again.
- [ROM2] A purchase is refused in full when there is not enough gold for everything on the table.
- [ROM2] The shop's buy, sell and exit totals now match the prices shown on the table.
- [BASE] With smoothing on, units, objects, buildings and some shots get the soft edge the original draws.
- [BASE] Health and mana bars sit higher above tall units such as horsemen. Shorter units keep their usual bar positions.
- [ROM2] Monsters fight back again when attacked instead of running away or standing still.
- [ROM1] Improved save compatibility with the original game for protections, Bless, haste and invisibility, including saves made by earlier versions.
- [BASE] A cell trap now fires when a character steps onto it, not before.
- [BASE] A popup now darkens the whole screen, with no undimmed edge: the right column and the bottom boxes darken with the map.
- [ROM2] A merchant in the second game sometimes speaks about an item you move from the shelf to the table.
- [ROM1] Enemy casters and spellcasting creatures turn to face their target before they start a spell.

### For map makers and modders

- [BASE] In the map editor, a selection made by Shift-clicking units on a freshly opened map shows its window again, with the group buttons and Delete.
- [BASE] In the map editor the Window menu opens a variables table: each variable the scripts use, with its value, its comparison or action and its trigger; a click on a row selects that trigger.
- [BASE] In the map editor Ctrl+Left/Right and Ctrl+Shift+Left/Right use the same word and punctuation stops as Ctrl+Backspace/Delete, including around underscores.
- [BASE] In the map editor, remembered windows stay on the same spell or item recipe after an earlier one is deleted.
- [BASE] In the map editor the Scenario palette no longer has a New variable button; choose New variable in a trigger's variable field instead, and the variable appears in the list.
- [BASE] In the map editor the Units grid has a Creatures / People switch at its top: it shows one set of units at a time, starts on Creatures and keeps the last choice while the editor is open.
- [BASE] In the map editor a click on empty ground inside a trigger's area on the map no longer selects that trigger and clears the selection, as anywhere else; a click on the trigger's label above its area selects it.
- [BASE] In the map editor the unit window is shorter: Kind and Player head its first column, the bag is an Add item pick over the bag's numbered items, and the figure stands at the game's own size. The Basic / Detailed switch now sits at the left of the bottom button row; the cell, facing, dialogue speaker, map keys and hint lines are gone, and their values are kept.
- [BASE] Ctrl+Backspace and Ctrl+Delete remove a whole word before or after the caret in the dialogue editor and other map editor text fields.
- [BASE] Switching tools on the map editor's top bar puts away the windows of the tool you leave and brings them back when you return.
- [BASE] In the map editor the Units grid shows creatures and people in two blocks, a row per creature type with each variant in its own colours, and people grouped as NPC, others, heroes and PC.
- [BASE] In the map editor the Scenario palette's New trigger and New variable buttons add the record in one click; the old Add window for script records, and the palette's Add button, are gone.
- [BASE] In the map editor each top bar window button opens its windows and stays pressed; a second click closes them, and choosing another button closes the previous windows.
- [BASE] A unit's stat inputs leave space around the card's labels, neighbouring inputs and edges in English and Russian.
- [BASE] Scrolling to a unit's direction buttons keeps their rows and hover tips clear of the character card.
- [BASE] Choose a unit's facing on nine compass buttons in the Add window and unit panel; the middle dot keeps the original's random direction.
- [BASE] Closing the map editor's Map window after resizing the editor no longer crashes it, and closing it keeps the start-point tool it opened.
- [BASE] Every picture in the map editor's Objects grid now shows the object's map sprites, including decorations whose information picture is missing in the Russian install.

## 0.147.0

### For players

- [BASE] Hovering a unit or building now keeps its cursor and information card together across the whole target area. The hovered unit lights up and the pointer stays in place when the cursor changes.
- [ROM2] A monster walking home after you lead it away no longer turns on you at once when you hit it from beyond its guard circle; in a loaded original save its guard circle now widens once, as in the original.
- [ROM2] A guarding monster you hit and then step away from now backs off for a while before it comes for you again, as in the original. A mission save keeps who hit it and when, so a loaded save, the original game's own included, carries on the same way.
- [ROM2] When the original game loads a mission saved in Againrom, the player now sees around their own units again, and the monsters no longer walk off towards a corner of the map at the start.
- [ROM2] Trolls, ogres and dragons speaking in the second game now show their whole head in the portrait, as in the original.
- [ROM2] Saves now keep the character and creature identities the original game needs when loading a mission.
- [ROM2] A mission save now writes each enemy group's guard order and each unit's post, as the original's own saves hold them; before, they were written empty.
- [ROM2] P now picks up the item under the selected unit, and the unit answers with its pick-up reply, as in the original; the first game keeps P for Patrol.
- [ROM2] A road sign you have seen and walked away from shows the use cursor again and opens its message on a click.
- [ROM2] Turning music off fades the melody out over up to two seconds instead of cutting it off; a dialogue page with a tune tag fades to the melody it names.
- [ROM2] The dragon's spell book now lists its spells, and a mission save keeps each creature's spells, such as the bats' Darkness; before, the book was empty and the save wrote no creature spells.
- [ROM2] A mission save now keeps each road sign's message and the number the original game finds a clicked building by, so the original, loading the save, opens the sign's text on a click; a save made before by Againrom gets both on its next save.

### For map makers and modders

- [BASE] The map editor's Bridge tool paints wooden bridges with a left drag and erases them with a right drag, joining the bridges the stroke meets into one shape of bridges; each stroke undoes in one step, and pieces smaller than 3 by 2 cells are removed.
- [BASE] The Magic Symbol (Магический символ in the Russian editor) is listed in the map editor's Objects palette, not in Structures; its picture, placement and map data are unchanged.
- [BASE] A wooden bridge placed in the map editor now spans a 3 by 3 area that stays open to walkers, as the bridges of the installed maps do; before, it was one closed cell. Its width and height fields resize it.
- [BASE] The map editor no longer lists the Horizontal Wooden Bridge among the structures you can place; no installed map uses it and the map format cannot size it. A map that already holds one still opens and saves unchanged.
- [BASE] A click on the Scenario branch of the map editor tree, on its Scenario button or on its key now leaves the trigger or other record that was open and opens no trigger window; a click on a trigger row still opens that trigger.
- [BASE] The map editor of the Russian game shows the trap and spell texts in Russian: the spell name beside a trap on the map, the field names of a trap's window and its range message were drawn as question marks.
- [BASE] The map editor's Units palette shows a picture on every creature and person again; most entries were empty after the choice of who a unit is replaced the class list.
- [BASE] The map editor's top bar has a File menu with Undo, Redo and Search, a Map button for the map panel, and Tools, View and Window at its right end. Every button shows when it is pressed.
- [ROM1] The map editor's dialogue editor shows each dialogue part as one card with a cell for each hero who sees a different text: by sex, by class, or both. A part can be split by sex or by class and merged back, each cell edits its own text and speaker, a hero-dependent speaker shows the face that hero's dialogue gets, a tag no hero reaches shows as never shown, and an untouched page saves byte for byte.
- [ROM1] The map editor sets who a unit is with one choice, a creature, a person or a named character, in the unit window and in the Add window, and shows only the settings the game reads for it; the raw map keys stay visible read-only under Map keys. The unit window says whether a dialogue shows the unit as a speaker, and a trigger's unit field names each unit by who it is, its number and its cell, and a click on the map takes the unit standing on that cell.
- [ROM1] Traps, unit spells and casting structures have named marks on the editor map and a shared palette; traps show their source and target cells and can be moved and edited on the map.
- [ROM1] Trap area fills a dragged rectangle with one spell and power. Delete removes the whole stroke, with one Undo; sets last while the map stays open.
- [ROM1] Saving an untouched map or changing only its cell spells preserves every unrelated byte.
- [ROM1] The map editor's unit window, in both views, gives a placed unit its own name, speed and sight, and chooses the direction it faces when the map starts; the Add window asks the direction too. The game shows the name and uses the speed, sight and direction, and a saved game keeps them; after a load, a person keeps his authored speed and sight only until his next equip, training or potion.
- [ROM1] Choosing Scenario in the map editor, from the top bar, with F2 or in the record tree, shows the scenario's records and no longer opens the first trigger.

## 0.146.1

### For players

- [ROM2] Monsters' Teleport checks occupied cells, Acid Stream keeps its original edge aim, and Blizzard starts its pulses after the spell lands. Stone Curse refreshes a creature's mana reserve when its hold starts and ends.
- [ROM2] Trolls and other non-human speakers in the second game's missions now show their portraits.
- [ROM2] A mage made in the generator now starts with a staff that casts the chosen school's spell; before, the staff was empty.
- [ROM2] Townsfolk speak their lines aloud in the towns' talks, and characters cry out in their own voices when wounded or killed; before, the town talks and these cries were silent.
- [ROM2] The shops of the second game list their goods in the original's order.
- [ROM2] The inn has a Sleep button, the first of its four buttons as in the first game's tavern, that restocks the town's shop; the four buttons sit on the same panel as the shop's.
- [ROM2] In the English edition, Sagita, Hildarius, Galinel, Iglez and Dina now answer when selected or ordered.
- [ROM2] A character escorting you now heals you with Heal when you are wounded, and an unarmed mage with auto-healing on heals with Heal; before, they never did. The spellbook and item popups of the second game state the caption of their own spell, not another spell's.
- [ROM2] A spell in flight in a loaded mission save now finishes: it lands and does its damage, and a save made before it lands keeps it; before, it hung in the air, never landed and was dropped from the next save.
- [ROM2] Animate Dead raises a zombie, a skeleton or a ghost from the corpse as in the original.
- [ROM2] A creature you summon gives no experience and drops nothing when it dies, as in the original, and so does a map unit the original marks the same way. Protection from Fire, Water, Air and Earth, Bless and Shield cast by a creature or a hero show only the held mark, as in the original; before, a cast picture flew first.
- [BASE] Saving after loading a save of the original game keeps each character's current target, spell and aim point as that save held them, and each building's data as the original wrote it; before, some characters lost the target or spell and each building's data was rewritten.

### For map makers and modders

- [BASE] The Structures grid has a Ruined switch to place ruined buildings; bridges, graves, the teleport pad and the campfire show their own picture instead of the hive.
- [BASE] A mod's sound may be a 4-bit Microsoft ADPCM WAV, mono or stereo.
- [BASE] The map editor no longer quits when you drag a character window in its detailed mode down or to the side past the editor's edge; the window stops with its title row inside the editor, as the other editor windows do.
- [BASE] In the map editor each placeable thing lists in one palette: Structures holds what can be selected, destroyed or switched and the bridges, Objects the decoration (fences and the like included), Trees the trees and bushes; before, every object listed in both Objects and Trees. A palette with nothing in a part says so, and what you place writes the same map as before, trees included: the Trees window plants a forest on open cells or paints one tree kind on every cell you cover.
- [BASE] In the map editor a new trigger takes one click: the Scenario palette (RU «Сценарий») lists New trigger (RU «Новый триггер») first and then the scripts' records as the tree lists them, a record row opening the record; New trigger, F4 or Tools, New trigger adds a trigger named Trigger that fires once and opens it at once with its name selected, so you type the name and press Apply; one undo takes the new trigger out, with its name, until you save or close its window, and Cancel keeps it.
- [BASE] In the map editor's Scripts list, Add then Condition (RU «Условие») offers the trigger wizard's conditions with their fields and puts the condition into the first free pair of a trigger you pick or of a new trigger (RU «Новый триггер»), as the wizard writes it; a trigger with three pairs taken is refused. A condition no trigger uses has "Add to trigger" (RU «Добавить в триггер») in its window. Add then New variable (RU «Новая переменная») asks a label and lists the variable at once; the variable window edits and clears the label, and the label shows wherever the variable is chosen or named. The labels are kept in a file beside the map named like it with the ending .variables, which the game never reads; a map with no label writes none.

## 0.145.0

### For players

- [ROM1] A patrolling unit keeps its patrol when you save and load: after loading, it walks its route as before, and the next save keeps the route, so the original game no longer crashes loading that save.
- [ROM2] After the Druids' mission the party arrives in the Druids' town with Iglez and Dina, each keeping what he carries, and the map also offers the mission the original adds there; loading a save made in that town no longer empties Iglez's pack.
- [ROM2] Summoned creatures no longer stay forever, as in the original: while a mage has two or more, about once a second one of them may die and vanish, with a chance of about one in a hundred for each creature beyond the first, and the last one stays; a summoned creature killed in a fight now vanishes after its fall instead of decaying to bones. Creatures summoned before a save stay after loading it until they die.

### For map makers and modders

- [BASE] The map editor draws large maps faster: a 256 by 256 map with 5483 records at full zoom now takes about 17 ms a frame on an idle machine (a frame every 95 ms before), and the same map at the fitted zoom about 80 ms (150 ms before). The map check no longer re-reads the install's text pages on every frame; a page file changed on disk outside the editor shows in the report within a second. Nothing the editor draws, saves or reports changes.
- [BASE] A mod can add new terrain: up to 19 new ground types, each with its own pictures, its land kind (such as grass, water or mountain) and a name the map editor's terrain list shows; the editor paints them and the game draws them, walks on them and shows them on the minimap. See "New terrain tiles" in docs/MODS.md.
- [BASE] Esc no longer closes the map editor, with or without unsaved changes; close it with the window's close button or Exit in the map menu.
- [BASE] While you drag a unit, structure, sack or other map object with the mouse, the windows and panels floating over the map turn almost see-through and ignore the mouse, so you can drop the object on a cell under them; they return when you let go. The top bar, the status line and the side panels docked beside the map stay as they are.
- [BASE] If the map editor stops because of a program error, it now says why: the message and the place it happened are written to the console, or, when it has none, to mapedit-error.log in the Againrom folder of your user configuration directory, and the program exits with an error code; before, such a stop was silent.
- [BASE] The map editor has two side panels. On the left, "Objects and scenarios" (RU «Объекты и сценарии») lists the map's scripts, players, groups, units, sacks and structures as a tree with a search, in the toolbar's order and words; a row selects as a row of the records list does and brings the unit, structure or trigger area into view. A category entry on the bar, such as Units (RU «Отряды»), opens that category's palette, which places or acts and no longer lists records, and leaves the panels and the tree as they are; selecting a unit or another object opens its place in the tree; resting the pointer on a palette cell for a moment names it. On the right, "Properties" (RU «Свойства») shows the selected object's window, a trigger's «Когда/Сделать» and a group's Delete group included, fitted to the panel's width in both languages; a double click on its title opens it as a separate window. Each panel's < or > button, F6, F7 and the panel switches on the right of the toolbar minimize a panel to a narrow strip and restore it, the inner edges drag to resize with no widest size (a panel grows until the map keeps its least width), dragging a panel's title row out of the panel undocks it into a floating window that moves and resizes like any editor window, its close button sends the panel back to its strip, and the editor remembers both panels and whether each floats; a narrow window narrows and then minimizes the right panel first, so the tree stays in reach down to a 640-pixel window.
- [BASE] The map editor's Units and Structures palettes open at a normal width and scroll the rest; a size you drag on either one is kept for that palette when you close and open it again and the next time the editor starts.
- [BASE] The map editor has a toolbar over the map: select, the four brushes and draw trigger area on the left; a switch for each map layer (map border, trigger areas, drop cells, footprints, issues, height numbers, passability view, player colours), grid, fit map, report and the two panel switches on the right, each with a tip naming it and its key. R or the Report button checks the map and shows the report in a panel below the map, also in place of the key list; F8 shows or hides that panel. The panel resizes and stays as you left it, a long finding scrolls with the wheel inside it, and the Report button shows pressed only while the report is shown; random landscape and random heights are under Generate in the Map menu, and a new trigger is made from the Scripts palette's Add.
- [BASE] The map editor shows a unit, a building or a sack in plain words first: a unit as its character card with its stats as boxes (below); a building's type, owner, position and script target with the scripts that name it, and its placement word with its bits in a block that opens; a sack's items by name and picture, with codes, recipes and enchantments in a Details block. The window's "View: Basic" button (RU «Вид: Обычный») switches to the detailed view of every map-file field, "View: Detailed" (RU «Вид: Детальный»), and back, changes no value, and the editor remembers the choice.
- [ROM2] The map editor's unit window shows the same speaker face as the game, for human speakers and stand-in faces too.
- [ROM1] In the map editor's dialogue page each line shows its speaker's portrait on the left, under the line number, and a speaker is named as the game names a person of that record: speaker 57 on the map with the Scroll dialogue is Veglud, where the editor said Archer; before, every speaker carried the name one line below.
- [ROM1] In the map editor's dialogue page the speakers whose face the game takes from the player's hero ("Main hero", "Traitor mage", "Monk", "Prisoner") now show the face the game gives when the first hero is a man, with the tip "The face depends on the player's hero"; before, "Traitor mage" and the others had no portrait there.
- [BASE] In the map editor, with the player colours on, selecting a player makes that player's units and structures stand out in a brighter colour and draws every other one plain; the legend marks the selected player.
- [BASE] In the map editor, deleting a unit that owns a stock deletes the stock with it instead of refusing; one undo brings both back.
- [BASE] The map editor's toolbar icons are twice as large where the window has room for them; in a small window they keep their old size.
- [BASE] The map editor's toolbar can float as its own window: drag it out by the dotted grip at its left end, move it by its top strip, and close the window to put the toolbar back over the map. The editor remembers a floating toolbar and its place.
- [BASE] The map editor's bar has Tools, View and Window menus (RU «Инструменты», «Вид», «Окно») after Map: every toolbar button is also a menu row in words with its key, such as "Terrain brush (T)", marked while it is on; Window lists the toolbar and the three panels, and its Toolbar row hides the toolbar and shows it again where it was, which the editor remembers.
- [BASE] The map editor's bar has a search field, "Search (Ctrl+F)" (RU «Поиск (Ctrl+F)»): Ctrl+F puts the cursor in it and Esc leaves it. Typing lists the matching units, structures, sacks, groups, players and scripts in one list, each with its category, found as the left panel's search finds them; choosing one selects it, shows it on the map and in the Properties panel, and leaves the side panels as they are.
- [BASE] In the map editor a unit's sack opens from the unit's bag in the same window as a sack you select, basic or detailed, with Back to the unit; K opens the stock it adds the same way. The older separate bag window is gone, and undo and redo of an edit there work as for any sack. In a sack's window an item can be duplicated with a copy of its recipe, its recipe can be a new one, a copy of an existing recipe for that item alone, or a recipe shared as it is, and its recipe's effects are edited beside it; an effect edit on a recipe other items share copies it for that item first, so the other items keep their effects.
- [BASE] In the map editor the speaker list of a dialogue page and the person window's NPC list show the speakers' names without the small portraits; resting the pointer on a name shows that speaker's face at full size.
- [BASE] The map editor's view switch reads "View: Basic" and "View: Detailed" (RU «Вид: Обычный» and «Вид: Детальный»); the detailed view is the former technical view. A unit's basic view is its character card drawn twice as large under its name, player, group, kind, tier and NPC record: each stat the map stores, such as health, mana, body, the skills and the protections, is a box on the card that replaces the card's number there, and a value you type and apply redraws the card. Where the value the game uses differs from the stored one (a unit placed with an NPC record ignores most stored stats; a mage's body, mind and spirit are derived), a line under the rows names the stat and the game's value, and a health or mana box leaves the card's "/max" beside it. A value the map cannot store stays as card text, and its tip says so. The scripts that use a unit, its equipment, the bag, the look and the map file fields are in the detailed view.
- [BASE] In the map editor each speaker row of a dialogue page has an "On map" button (RU «На карте»): press it and click a unit, and the speaker becomes the record that shows that unit when the mission starts, the unit's own NPC record first; a creature takes the record whose picture is the one its info window shows. A unit no record shows leaves the speaker as it was and the status line says so; Esc or a second press cancels. The faces in the speaker list and the person window's NPC list are now the faces the dialogue shows at mission start on that map. As in the game, a unit placed dead or on the map's border ring speaks for no record, and a live unit of the same look speaks for it. While the pick is armed the card may be moved and follows its pick; Apply, Cancel or removing the card ends the pick.
- [BASE] When the "On map" pick finds no speaker for a unit, the dialogue popup now says why beside that line (the unit starts down, stands on the withdrawn border, or the game shows another unit for that speaker first, naming the cell), and the pick stays armed for the next click, with the "On map" button drawn pressed in while it is armed.
- [BASE] A mod can give one win several follow-up missions, each with its own map, and the starting hero picks one: a mission may name a class (`class = "fighter"` or `"mage"`) and a sex (`sex = "male"` or `"female"`); two missions that would both play for one hero are refused when the mod loads, naming both.
- [BASE] In the map editor, a row of the Map menu that would do nothing now shows greyed out and cannot be chosen: Save, Save as and Test play with no editable map, the Map panel with no panel, All records with no map, and the random terrain and height commands with no map or in the built-in map list.
- [BASE] The map editor's trigger wizard reads nearly every condition and action of the original scenarios as an ordinary row you can edit, where it used to keep them as "Other" (RU «Другое»): variables, items in a pack and near a point, sacks on a cell, a unit's health, a switch, the attack target, losing when a unit dies, spells cast at a point or a unit, traps and effect times, giving and taking items, putting a unit in another's place, a unit's property, action 7, and group orders Stand Ground, Follow, Attack and Defend. Conditions with several comparisons let you choose the comparison. The item list is grouped under headings (mission, units, players, groups, cells, items, structures, spells, variables) and opens whole at 1920x1080. A variable field lists the map's variables and "New variable" (RU «Новая переменная»); when the map's 100 slots for conditions and variables cannot hold both, the wizard says so in plain words. Rows the wizard still keeps as "Other" are the pairs that name no condition at all and the one distance check written over two units. A row you do not edit saves exactly as it was.
- [BASE] Map editor windows no longer cover the right strip or the Report strip when they open: an object, group, trigger or Add window opens inside the map area, the Forest window ends at the map's bottom edge, area labels no longer overlap, and on a small screen a unit's face and doll leave the window's buttons free.

## 0.144.0

### For players

- [ROM2] Monsters cast their spells as in the original: a monster with Summon now summons, and a monster's Teleport lands next to its target only from more than two cells away; closer in, it fights instead.
- [ROM2] The shop and the inn in each town play their sounds: entering, buying, selling, a refused buy, putting items on the table, scrolling the shelves, undo and leaving in the shop; hiring, dismissing, a refused hire, talking and leaving in the inn.
- [ROM2] A mage learns a spell by reading its book, as in the first game: use the book from the pack in a mission, or drop it on the mage in the shop. The book is used up; a fighter, a mage who knows the spell and a book of Slow are refused.
- [ROM2] Lifting an item out of the pack in the shop no longer leaves its name in the empty slot.

### For map makers and modders

- [BASE] The map editor's View menu has Player colours (key Y): every unit and building is tinted in its player's colour, each player of the map has its own, and a box in the lower left corner of the map lists the players with their colours.

## 0.143.0

### For players

- [ROM2] A game you save during a mission, from the save page, with quick save or by the autosaves, is now written in the original game's own save format too, so the original game can try to load it; loading it in Againrom returns you to the same moment.
- [ROM2] A mission you save after loading a mission save the original game wrote now keeps each building's type and each of the original's pointer records as the file held them. Before, a building whose type number was above 255 was saved with a wrong type, and a pointer record was saved as a plain building.
- [ROM2] A mission save written in the original game's format now lists the map's changed cells, orders the ground entries and opens the diplomacy and script registers as the original does.
- [ROM2] A game saved in a town whose party holds a man whose type number is above 255 is now saved, and loading it keeps his type. Before, such a town save failed.
- [ROM2] In a town, the in-game menu shows the original's five buttons: Save Game, Load Game, Sound Options, Abort Game and Return to Game, with the keys S, L, N, E and R in both languages. The mission's menu is unchanged.

### For map makers and modders

- [BASE] In the map editor, the trigger wizard names a relation as the Players page does: Neutral, Hostile, Ally (RU Нейтралитет, Вражда, Союз) instead of peace, hostile, alliance, and its Change relation action reads as the trigger list does (RU "Изменить отношение").
- [BASE] In the map editor, the speaker list of a dialogue page shows each speaker's small portrait, the face the game shows when that speaker talks, and the page card shows the chosen speaker's portrait beside the name. The person window's NPC list shows the same portraits. A speaker with no face keeps an empty place.
- [BASE] The map editor no longer places a new sack on a cell of the map's border ring; moving or pasting one there was already refused, and adding one now gives the same message.

## 0.142.0

### For players

- [ROM2] Your hero, Iglez, Dina and your hired men now answer when you select them and when you order them to move, attack, swarm, defend, retreat or pick something up, each in their own voice, as in the original; a spell order stays silent. In the English edition only a male warrior hero speaks for now: the other voices are recorded in a form Againrom cannot play yet, and the hired men have no recordings there.
- [ROM2] Iglez, who joins you after his talk in Kaarg's tavern, now stays in your party when you load a Kaarg save the original game wrote, and a town save written after that keeps him. When you load a mission 30 or 31 save the original game wrote with Iglez in it, he is again your party member, not an ally the map lends you: if he falls, the mission is lost with its own notice of his death, and after the mission he comes back to town with the party. In both cases he has no name, because the original's save holds none.
- [ROM2] The town shop now sorts each shelf, gives enchanted armour, shields and clothing one to three attribute bonuses and never a spell, names a book's and a scroll's spell in its hover text, and draws prices in the original's small font so they fit their plates. Bought scrolls and the hero's starting scroll now cast their spell in a mission. Scrolls in saves made before this change still come home to the bag after the next won mission and cast from then on; until then they cannot be cast. Before, a book's hover showed a number, a cloak could cast Lightning and a scroll could not be cast.
- [ROM2] A game you save in a town is now written in the original game's own save format, so the original game can try to load it. A game saved during a mission is still written only in Againrom's own format.
- [ROM1] A companion handed to another side during a mission (Reniesta in mission 130) leaves your party selection, and her pack, worn items, weapon swap and shop are closed to you; the spell book of a unit that is not yours shows "no hero selected" unless the map is revealed; quick save, save and the timed autosave work after the handover, and loading puts her back with the other side.
- [ROM2] After you press Continue on a mission's victory panel you can now save the game; loading that save returns you to the mission map, where the victory panel opens again. Before, the save failed with "saving is not available for this game yet".
- [ROM2] Where the game cannot be saved, a short line in the game's language now says why: finish the conversation, close the message, save in a town or on a mission instead of the world map, or the mission is lost. Before, F2 did nothing there, or the save page showed the same English line every time.
- [ROM1] Towers, wells and the campfire now light the ground and the units at their own corners as the original does, pulsing in step with the game's clock, and a destroyed tower gives no light.
- [ROM2] Teleport now shows its picture where the caster stands and where he lands at once, plays its sound once in the middle of the cast, and the caster plays his whole casting animation.
- [ROM2] A unit under Curse or Poison Cloud now shows that spell's own sparks while the effect lasts.
- [ROM2] After you load a mission save the original game wrote, the enemies guard their places and attack your hero again.
- [ROM1] A game saved in Againrom after a Control Spirit cast, then saved again in the original, now loads, and the dead body the spell used stays gone. Before, the game refused to load it.
- [ROM1] Loading and saving again a game from an older Againrom, whose heroes all carried the male fighter's type, now saves each hero with their own class and sex. The original game stopped loading such a file with a missing-picture error for a mage.

### For map makers and modders

- [BASE] In the map editor, a trigger parameter that names a unit, a structure, a group or an area now has an "On map" button: press it, then click the unit, the structure, a unit of the group, or the area on the map, and the parameter is filled. The node window offers it for unit, structure and group parameters; the trigger wizard offers it for units, groups and areas ("Area on map"). A click that fits nothing says so on the status line and the pick stays armed; Esc or a right click cancels.
- [BASE] In the map editor, with a menu open (Map, View, Brushes, Tools), a press on another menu's arrow now closes it and opens that one; before, View left Brushes dead.
- [BASE] Saving a map in the editor now shows "Saving the map, please wait..." on the status line and then the saved file or the error; Open File starts in your `maps` folder; the pointer over the border or corner of a resizable editor window shows the resize cursor; the group headings of the Scripts list are drawn heavier.
- [BASE] In the map editor, a terrain stroke now joins every terrain it meets, also where three terrains meet, such as Grass, Cracked and Stones; painting over a hard edge with one of its terrains joins it. Before, a stroke could leave a hard edge or corner at such a junction.
- [BASE] An agent connected to the map editor can now read the heights of an area, and the help says what each height brush mode does; saves and key presses answer a short result, a save that wrote its file no longer reports an error, and a missing folder is named. Before, heights could only be changed, not read, and each answer carried the whole window.
- [BASE] In the map editor, Ctrl+C copies the selected row of a report panel to the clipboard. In the MCP connection panel that copies the bearer token, the URL or the token file name without its label, and the panel says so. Before, the token was shown but could not be selected or copied.
- [BASE] The map editor keeps its MCP bearer token and address between runs, in one file, `mapedit-mcp.json`, in the player profile; a client set up once keeps working, and the new `-mcp-new-token` option replaces the token. Before, every run made a new token and a new file, and the address changed too.
- [BASE] The map editor's Map menu now reads Create new map, Open, Save, Save as, then Built-in maps; the built-in list shows each map under its title, single-player maps apart from multiplayer maps. The bar lists Scripts, Players, Groups, Units, Sacks, Structures, Objects, Trees, View and Tools; Objects and Trees open the object brush.
- [BASE] In the map editor, the brushes moved into the Tools menu, above a Whole map group with Random terrain and Random heights; "No brush" is gone, Esc or a right click puts the brush down. The Drop zone switch sits at the bottom of the Scripts list, left of Add. The trigger area tool left Tools and stays in View and on L. Russian "Местность" now reads "Ландшафты".
- [ROM1] In the map editor, a hero of the party reads "Hero 1: Main hero" and so on, with the role the message window gives that speaker, in the trigger wizard, the script choices and the goals; the map keeps the same number.
- [BASE] The trigger wizard offers the condition "Unit near unit": two units, picked on the map or from the list, and a radius; it holds while they stand within the radius, and a dead unit is never near.

### For map makers and modders

- [BASE] An agent connected to the map editor over MCP can now also move, delete and copy a selection or an area, put units in a group, add a shop stock and an item recipe, write message pages, add a box area to a trigger, point a goal at a unit, paint and thin forest, set mod package fields, and read the goals, the message pages and a unit's settings preview, each with the same change and Undo step as the window.
- [BASE] The map editor no longer places a unit, a building or a drop zone in the 8-cell border round the map, which the game does not show, and says where the playable area is. A new unit gets a group of its own unless you choose one. For an agent over MCP, adding a unit or building answers the number a trigger names it by, a trigger that names a unit or group the map does not hold is refused, and each edit lists the map check's warnings on what it changed.

## 0.141.0

### For players

- [ROM2] Bless, the four protections from the elements and the magic shield show their own pictures on the unit while they last.
- [ROM2] Iglez and the available mercenaries show their own figures in the inn. Hired mercenaries join the next mission and keep their appearance when you save and load.
- [ROM2] Mission villagers, guards and brigands no longer all look like one man when inspected: each shows the face and body of his own kind, and the mage in the Astrologer mission walks with the hero mage's figure. Both stay after you save and load.
- [ROM1] A game the original saved while a building was selected now loads. The building stays selected, and your next save keeps it.
- [ROM2] The Kaarg innkeeper now moves through his five animations. The inn plays its background voices and clattering dishes.

### For map makers and modders

- [BASE] A unit, structure or sack placed in the map editor with a click on a cell is now the same record as one added with Enter or with the add_record command. Before, a click wrote a value for a step the Add window leaves out (a unit of a class with no tiers got tier 0).
- [BASE] An agent connected to the map editor over MCP can now build a map with direct commands instead of keys and clicks: make a new map of a size, paint terrain, heights, objects and passability, add units, structures, loot, players and triggers, set fields and map properties, place drop zones, and read records, Add steps, brushes and the map check. Each command makes the same change as the window and takes one step of the same Undo history.

## 0.140.2

### For players

- [ROM2] The mission map now shows through the left edge of the right column, removing the black stripe beside its ornaments.
- [ROM2] In the tavern, the people at the tables are headed with their own names, not the name of the next person in the list; before, a man could be headed as a woman.
- [ROM2] In the Kaarg and druid shops the hover over a shelf now names what is on it; before, the names of the weapons, enchanted equipment and potions shelves were mixed up.
- [BASE] Under Darkness, a hero whose sight drops to nothing still uncovers the cells right around him, as in the original game; before, he uncovered none.
- [ROM2] The spell books in the shop show their picture, on the shelf, on the table and in your pack: each book wears the picture of its school of magic. Before, the cell was empty.
- [ROM2] The shop table holds five kinds of items at most, as in the first game, and always shows all of them; before, it took more and hid the extra ones, which only the mouse wheel over the table brought into view.

### For map makers and modders

- [BASE] The map editor's MCP connection works with current Claude Code clients.
- [BASE] The map editor can let an agent work in its visible window over MCP: point, type, use its keys, inspect the picture and save maps, with the same Undo history. Enable the connection with a launch flag.
- [BASE] The map editor opens a map file you double-click or drop on it in Windows once `.alm` files are associated with it: `mapedit.exe "map.alm"` opens that map, and it finds the game files from its own folder, from the game folder that holds the map, or from the base selected in the starter. A map in a mod folder opens as a loose map. A start that fails writes its reason to `mapedit-error.log` in the Againrom folder of your user configuration folder when there is no console to show it.
- [BASE] A town file's square now takes an optional `pane` entry, as a room's: with it the square shows the character pane. In a first-game town the square's pane needs `rect` `[480, 238, 640, 480]` and has no spellbook. A square without `pane` shows none, as before.

## 0.139.0

### For players

- [ROM2] A lost mission's long reason now shows whole, on several lines, with the window's buttons below it; before, it stopped after the first line.
- [BASE] The map editor no longer crashes when you add a condition or an action to a trigger and open its list: a new row lists only the ready conditions or actions, without the last choice "Other: Choose..." that closed the editor. A row the editor shows as "Other" still offers it, and choosing it again keeps that condition or action as it was.
- [BASE] A map whose text pages exist only in the other language (for example only in the Russian folder on an English game) now shows those pages in the map editor and in the game; a page that exists in both languages shows the game's language. Before, such a map showed every page empty.
- [ROM2] A lost mission now tells you why it was lost, in that mission's own words: when the hero dies, when Iglez or Dina dies, or when the mission's task fails. Before, the hero's or Dina's death showed only the general "mission failed" text.
- [ROM2] Dina now shows her portrait when she speaks, in town talks and in the missions: once she has joined, her lines show her, in what she is wearing. Before, her lines showed a man mage.
- [ROM2] What the hero buys in the shop now goes with him into the mission: his bag's items are in his pack there, to wear, drop and pick up, and a bought potion can be drunk there; they stay after SAVE and LOAD, and after a won mission they are back in the bag at the next town, together with the items and potions he found, where the shop can sell them. Before, the bag stayed behind in town.
- [ROM2] In the shop you can now put items on and take them off: drag an item from the bag onto the hero in the panel to wear it, and drag a worn item back to the bag to take it off. An item the hero cannot use is refused, as in the first game; with NEXT the same works for Iglez, and what each one wears stays after SAVE and LOAD. Before, the hero in the shop took no item.
- [ROM2] A double click on an item in the shop's bag now puts it on the hero shown in the panel, as in the first game; an item he cannot wear stays on the counter, and Undo puts it back in the bag. Before, the double click made the item vanish from the bag without any gold.

- [ROM1] Missions 110 and 111 now put on their starting equipment and use their starting potion instead of leaving those items in the bag. Replaced equipment stays in the bag.
- [ROM1] A shop on a mission map now opens the town's shop page when your hero uses it: he can buy and sell with the gold and the bag he has in the mission, and Exit takes you back to the mission where you left it. The mission waits while the shop page is open, and what you bought stays after SAVE and LOAD.

### For map makers and modders

- [ROM1] The map editor's W window manages mission lists, explicit win links and named parties with equipment. Page edits keep each language's pages, including pages beside the map; removing a replacement keeps its campaign links. Publish creates a packed RES and asks before saving changes or replacing an existing file; Test play keeps the complete mod project.
- [ROM1] Opening a handwritten mod and saving it to a new folder now copies the whole mod with the same ID, including its other missions and language files. Reopen a mod before saving if its files have changed outside the editor.
- [ROM1] The tavern, shop and school of a first-game town file now take their character pane from the room's optional `pane` entry, as the second game's inn and shop do: `book` says whether the shop pane has the spellbook (with `"book": false` the shop has no spellbook corner, the B key does nothing and the spell strip stays closed), and a new `font` field, `"install"`, draws the pane's arrows and mode box in the game's own font instead of the character generator's. The `rect` must be `[480, 238, 640, 480]`, the tavern and school may not have the book and no pane may have a `book_offset`; the town is refused at start otherwise, naming the room. A room without `pane` keeps the first game's own pane.
- [ROM1] The map editor's person window is smaller and shows the dialogue portrait beside the doll as you change the person. Tab also reaches the fields after you select a section.
- [BASE] The map editor's Russian words now spell «е» instead of «ё», which the game's font drew as «И»: «Остается», no longer «ОстаИтся».
- [BASE] The map editor's trigger window now shows a change of diplomacy and its check in words: the two players by name and peace, hostile or alliance (RU «мир», «вражда», «союз»), with a hint that a change works one way only. Before, the row read "Other: #38 Change relation" with bare numbers.
- [BASE] The map editor names a group's orders as the game's command panel does: Attack, Move, Guard, Defend, Swarm and Stand Ground (RU «Идти в боевой готовности» and «Держать позицию»), read from the installed game, and the script-only crowd order is called "Swarm as a crowd" (RU «Идти толпой в боевой готовности»). Before, they read "Swarm 2", "Рой" and "Рой 2", and "Стоять на месте". The stored orders do not change.
- [BASE] In the map editor's trigger window a blank line now stands between one action and the next, so the actions no longer run together; every button, scroll and key works as before. Before, the second action's row followed the first one's rows at once.
- [BASE] In the map editor the pictures on the Structures tab are now twice as large, so a structure is easier to find; the list below them and the placing work as before. Before, they were small.
- [BASE] In the map editor the speaker list of a dialog page now starts with the four leading speakers, Main hero, Traitor mage, Monk and Prisoner, so they need no scrolling or searching; the other speakers keep their order after them. Before, they stood among the others by number.
- [BASE] In the map editor a single tree, stone or other map object now moves the way a building does: drag it to another cell, or select it and press Delete to remove it; Undo puts it back. Before, a drag from a tree drew a selection box and Delete refused it. A unit, building or sack on the same cell still takes the drag first.
- [ROM1] The map editor lets you choose an ordinary person's face and sex and preview the doll before applying the change. The appearance is stored in the map and stays after saving the mission.
- [ROM1] A human placement can select a named NPC from the installed registry in the map editor. The choice keeps the original map storage and offers no custom placement name.
- [ROM1] Worn equipment has a row per slot, item dropdowns, available material and tier choices, and private magic recipes. The preview shows displaced items in the bag. Empty keeps native storage: it copies retained row items for an ordinary Human and is unavailable for a named NPC.
- [ROM1] A person's setup lets you add, replace, remove and duplicate carried items with private magic recipes. It shows the resulting bag and corpse loot before Apply, including displaced equipment and NPC loot suppression.
- [ROM1] E opens one person setup window with a pinned doll, equipment, bag, loot and stat overrides. Wide windows show three columns; smaller windows use sections. Item lists support typing to find a name, and the whole draft applies or cancels together.
- [BASE] F5 in the map editor now plays the map under the route of the mod package open in the W window: the guest companion, the mission it plays after, the town chapter, the gold and the mission number, or the replaced campaign mission. Before, F5 always played it as the first companion the campaign's first town grants. A route that Ctrl+S would refuse now refuses F5 with the same words.
- [ROM2] A mod can now add, change or switch off an inn talk: `[[inn]]` tables in the mod's `data/towns.toml` name a talk by its topic and speaker and say in which town, at which stage of the campaign and under which campaign conditions it shows, the same way a first-game mod edits a tavern talk with `[[tavern]]` tables. A talk the mod adds or changes must name a speaker, a talk text and, for a quest, a mission the game holds; otherwise the game names the mod, the file and the line at start and does not open. Before, the second game's inn talks could not be changed by a mod.
- [BASE] In the map editor a value longer than its field now ends in "..." inside the field, and a field you type in shows its text up to the caret; the cell row of the person window showed "88,167  1" cut through a character. At small window sizes the person window and the W window end above the status line, which they covered at 640x480.
- [BASE] The map editor's person window lists the triggers that use its unit again, under its fields on every section, with a link to each; since the compact person window it showed none, so a trigger naming a person could not be reached from the person.
- [BASE] In a map editor window narrower than its top bar, as at 640x480 and 800x600, the bar puts the entries that do not fit on a second row and the map starts below it; before, Brushes, Tools, Undo, Redo and Save lay past the window's right edge. At the default 1280x800 the bar is unchanged.

## 0.138.0

### For players

- [ROM2] Iglez now shows his portrait when he speaks: in the Kaarg inn talk, in later town talks and in the missions. Before, his lines showed an empty portrait pane. Once he is in the party, the portrait shows what he is wearing.
- [ROM2] Iglez now joins your party after you talk to him in the Kaarg inn, the talk where he asks you to come to Lairisha. He goes into the next missions beside the hero, comes back to town with the party, shows in the hero panel's PREVIOUS and NEXT, and stays after SAVE and LOAD. If he falls, the mission is lost with the mission's own notice of his death. The mission scripts that name him now work, so the second game's later missions no longer lose the parts written for him. Talking to him again adds no second Iglez. Dina, the mage, joins when the party leaves after mission 40, without a talk. Before, the hero always went alone.
- [ROM2] The hero now uncovers the map one cell further east and south, as in the original: on the first mission, at the start and after walking east, the cleared area ends on those sides where the original's does. Before, the dark edge stood one cell closer to the hero.
- [ROM1] The README now says how to take the game files out of the GOG installer on macOS and Linux: one innoextract command unpacks them into a game folder, without Windows.
- [ROM2] A click on a road signpost walks the hero to it and opens its message with Ok, such as "To Great Mage Skrakan's Tower" in the first mission; the game waits until Ok, Enter or Esc. Before, the hero walked next to the signpost and nothing opened.
- [ROM2] Delete on the Save and Load pages now asks in a small box over the page, as in the original: "Are you sure you want to delete game" with the save's name, and the buttons Yes and No. No or Esc closes the box and keeps the same save chosen; Yes deletes it. Before, the question replaced the page's list and Ok answered it.

### For map makers and modders

- [BASE] The map editor's key help now names the selection box in two short lines, "Drag on the map: box select" and "Drag in the box: move it"; before, one long line ran past the help column in both languages.
- [BASE] The map editor's Height tool has two new settings for Raise and Lower. Mode, three picture buttons: Step (the default) lifts or lowers every cell the brush passes over by the same amount once, however often the drag crosses it, so the ground keeps its shape; Level makes every cell the drag passes over one height, the pressed cell's height plus the strength (minus it for Lower), and shows that target on the status line while the button is held, so stroke after stroke builds even terraces; Smooth is the old rounded hill. Strength, with - and + buttons, is how much one stroke raises or lowers, from 1 to 31, 8 by default. Each stroke stays one undo step; Set height and Sample height work as before.
- [BASE] The trigger wizard has a new action, Order a group to a point: choose the group, the command (Swarm, Move, Swarm 2 or Patrol) and click the point on the map, and a shipped order of that kind opens as this row. A row of an editor window whose text ends in "..." shows its whole text under it while the pointer is over the row.
- [BASE] Dragging inside a selection box in the map editor now moves the trees and other objects under the box together with its units, structures and sacks, as one undo step; the ground and its heights stay where they are. Before, the drag only drew a new box and the objects stayed behind.
- [BASE] The map editor's Height tool has a new row, Sample height: press on a cell to take its height, and every cell the brush passes over while the button is held is set to that height, as one undo step.
- [BASE] The map editor's Players page shows how every player stands toward every other as a grid of small pictures: a grey ring for neutral, a red cross for hostile, a green disc for ally. Click a square to switch it to the next relation; each click is one undo step.
- [BASE] The relation grid on the map editor's Players page now names the player over each column, written upwards, the same as the row's name ("1: Self"); before, the columns showed only 1, 2, 3. With the pointer over a square, a line under it says who stands how toward whom, such as "1: Self -> 2: Enemy: Hostile"; over a column's name, the whole name.
- [BASE] The map editor's Forest tool has a Clear marks button left of the density buttons: it unticks every marked kind in the grid and the list at once. It is greyed out while no kind is marked.
- [BASE] In the map editor, Ctrl+C of a selection box now copies every unit, structure and sack in the box, whichever list tab is open; before, with the Units tab open the structures and sacks were left out. Ctrl+V no longer pastes at once: it shows the copy see-through under the pointer, ground included, and a left click places it there as one undo step. Esc, a right click or another tool cancels it; the copy stays for the next Ctrl+V.
- [BASE] The map editor no longer lets a structure stand on another structure, or a unit on another unit: adding, moving, dragging and pasting onto taken cells is refused with a note that names the object in the way. While you place, move, paste or drag, every cell the object will take is outlined, gold where it fits and red where it is refused. Maps that already hold such an overlap still open, and their other objects still move.

## 0.137.0

### For players

- [ROM2] A mission can be saved after a Summon cast, and the creature loads back as it was. The hero's last cast spell also stays selected after a load when the spell was learned during the mission. Before, the save failed after a Summon, and the load forgot the spell.
- [ROM2] Summon can be cast again after loading a save made during a mission. Before, the load left the spell no creature to call, and every later Summon was refused.
- [ROM2] A save the original game wrote during the first mission now loads, with every unit and building, the hero's items and full book, the saved view and the saved game speed. A spell in flight in such a save is not shown, does not land, and is dropped when you save again.
- [ROM2] The Load page lists the saves the original game wrote and loads a town save with the hero, his spells, his bag and the gold.
- [ROM1] When the purse is shorter than the school's price, Exit shows the shortfall as a negative number and Train stays pressable but teaches nothing, as in the original.
- [ROM1] In the school a second click on the selected skill row unselects it, and Train then shows 0 and Exit the whole purse, as in the original.
- [ROM1] In the school a skill row lights only when you select it; the idle shine that runs over the rows plays only while the school's tip is open.
- [ROM1] The shop sorts each shelf in the original's order: items a fighter can use first, then by kind, weapons as swords, axes, maces, spears and then ranged weapons, and cheapest first inside each group.
- [ROM1] The horse, the woman and the dervish on the town square no longer return to the same places after you save and load; their places are new each time you start the game.
- [ROM1] Loot dots on the minimap start switched off, so the minimap looks as in the original; the Game Options entry still switches them on.
- [ROM2] The figures on the town squares no longer return to the same places after you save and load, and loot dots on the minimap start switched off.
- [ROM2] Stone Curse now holds its target still, unable to move, attack or cast, for a random time that its Earth resistance shortens, and Summon calls a squirrel, a turtle or an animated foot to your side beside the caster, stronger as the caster's Astral skill grows; both last through a save and load. Before, Stone Curse did nothing and Summon could not be cast.
- [ROM2] A mage that starts a spell while stepping into a cell, an enemy or one of yours, now finishes the step and casts standing, as in the first game. Before, it froze mid-step for the whole cast.
- [ROM1] An enemy mage that does not see you itself, while its comrades do, now walks only until you are within its spell's reach and casts from there. Before, it walked on towards you as if to strike with its weapon.

## 0.136.0

### For players

- [ROM1] The inn's mercenary card says SPELLCASTER only for a mercenary who knows a spell, such as the mage, and no longer for club, sword, axe or bow fighters.
- [ROM1] A hero the map places, such as Danath with his sword and shield on the seventh mission, now fights with the class his weapon and shield give, as a hero in your party does. A hero's arrows and spells always leave from the place his current weapon gives, also when the map's data names a fighter a mage.
- [ROM2] While the mission menu or one of its windows is open, the whole mission screen behind it is darker, the right-hand column, the open pack bar and the messages included; the menu stays at full brightness.
- [ROM1] The shop, inn and school buttons use the original's gold lettering, turn yellow under the pointer and stay lit even when they cannot be used; the shop shows Buy and Sell with their sums on one line. Hover help text is gold-orange, and the Load list shows each save by its own name in ten framed rows.
- [ROM2] The mission menu and its windows open at normal size, centred over the darkened mission.
- [ROM1] After loading a town save, the inn lists its mercenaries in the original's order and selects the same first mercenary.
- [ROM2] The mission menu's Save and Load pages look as in the original: a centred title, ruled rows, a gold heading while the list is in use, and a save's date shown when the pointer rests on its row. After a delete the name field is empty.
- [ROM1] After you load a save the original wrote, a dead body that is put back on the map keeps its own number, so a missile flying at it still aims at it when you save and load again.
- [BASE] In the shop the Armor shelf opens on the shields, as in the original.
- [ROM1] The shop and the school look and count as in the original: the gold is the pack's last item, empty cells show the bronze plate, and the school's Exit shows the purse left after the price.
- [ROM2] Quest Objectives shows the objective's title on its own line with the text below it, instead of both on one line with four question marks between them, and the gold in the pack reads MONEY on one line, as in the original.
- [ROM2] In the mission menu the last button of each window (Save Ok, Save Cancel, Load Cancel, Game Options Cancel, Sound Options Esc, Quest Objectives Ok, End Quest Return to Game) closes the window and the menu, as in the original. End Quest opens with Exit to Main Menu selected, and Abort Game in the town with Return to Game selected.
- [ROM2] The minimap shows the marks the original shows: a dark blue dot at the signpost, red dots on enemy creatures while your hero is near them, and red marks on the spiky mounds; the explored ground is drawn in the original's colours over the crystal.
- [BASE] The Hall of Fame shows a zero score as 0 instead of leaving it empty, and View Cutscenes draws its list in ruled rows like the original, with eight rows in each game. In the first game Ok and Cancel there stand where the original puts them, the weapon pictures on the hero creation detail page open no tooltip, and its Accept, Reset and Back captions take the original's tan and yellow.
- [ROM2] The main menu behaves as in the original: the intro movie plays at launch, Esc on the menu or the Hall of Fame does nothing instead of asking to quit or closing the hall, and Cut Scenes always opens its list of watched movies. The credits show the Nival logo instead of the text "nival.bmp", and back on the menu no caption shows until the mouse moves.
- [ROM2] Hero creation matches the original in four details: the statistics card's words and numbers have a dark shadow, as do the inn patron's card and the hero panel's card; a command under the pointer (Accept, Reset, Back, Restore) turns yellow without changing its plaque; the last weapon the pointer touched stays lit until another weapon is touched or one is chosen; and the letters of the name field have no dark edges.
- [ROM1] The first mission opens as in the original: the view starts on the hero in the same place, the minimap stays empty under the first message, the inventory strip stays closed until the bag is clicked (its tooltip reads OPEN INVENTORY <~>,<I>), the gold in the inventory reads MONEY, and a fighter's Cast cell is empty with no tooltip.
- [ROM2] The mission screen matches the original at 1024x768: the right column is wider and the map ends where the original's does, the opening message stands in the middle of the screen, the hero is selected from the start with his portrait and a statistics card in whole words (AGILITY, ABSORB), the inventory row stays closed until the bag is clicked, the minimap stays empty under the opening message, and a fighter's Cast cell is empty with no tooltip.
- [ROM1] The first mission's opening message stands in the middle of the screen as in the original, not shifted left over the map, and the minimap's view frame is as tall as the original's (24 rows) while the empty corners of the right column are black.
- [ROM2] The minimap's view frame is white and 24 rows tall and your hero's dot is blue, as in the original, and the empty corners of the right column are black instead of dark blue.
- [ROM2] A hero loaded from a save keeps the right health and mana maximum after a potion, an item change, training or a sale in town; before, the maximum could come out a few points too high.
- [BASE] Catapults, ogres, fat trolls, the Death Star and the dragon stand where the original draws them, in the middle of the ground they cover, and their shadow, selection mark, health and mana bars and clicks go with them; before, they were drawn half a cell up and to the left (a whole cell for the dragon).
- [ROM1] A hero who swaps a two-handed sword for a one-handed one, or a mage whose staff comes off, now takes the class the new equipment shows at once and is saved with it, and a save that kept the old class opens with the class the equipment selects; before, the hero kept the old class in the game state until the mission was reopened, and the next save wrote it.

### For map makers and modders

- [BASE] The map editor's Save As window starts in the map's own folder, or in a `maps` folder beside the saves for a new, archive or installed map, and saving there works even when Againrom is unpacked inside the game folder. A refusal to save into the game folder now says so in your language and names the `maps` folder. Before, the window started in the game folder and every save there failed.
- [BASE] In the map editor's Scripts list a message action now shows as "Message" with the first line of its text, not as a plain "Action" with an English operation name, in English and Russian.
- [ROM1] F5 in the map editor plays a map that lies in the game folder, and Save mod package saves a mod into a folder inside the opened map's folder; a refusal names whether the folder is, contains or lies inside the opened map's or the selected mod folder. Before, F5 refused every map in the game folder because its test folder lies there.
- [BASE] In the trigger wizard the message action is now Start dialog, one per trigger, and its Configure opens a dialog editor: one card per page with its speaker and text, moved by dragging, removed with a cross until Apply, added with +, and a choice of a new or an existing dialog.

### For developers

- [BASE] Both games now build their town through one town screen; what differs between the games is town description data, and players see no change.
- [BASE] Units, buildings and dropped sacks take their numbers from one counter the way the original does, a save restores that counter on load, and saving writes the numbers the game holds; players see no change.

## 0.135.0

### For players

- [ROM2] The inn patrons' cards match the original: the mage now reads HEALTH 158/158 and MANA 604/604, only the mage's card says SPELLCASTER, and in Russian the mercenaries' heading reads НАЕМНЫЙ above their class. The town gate's guard now shows his face, a knight in a great helm, beside the refusal text.
- [ROM2] The shop looks as in the original: the chosen shelf lights its corner of the room (the armour stand with its torch for armour), the hero row is the ornate strip with a gold pile, items for sale sit on dark red cells and the hero's items on green ones, a bought item comes ahead of the gold, and the Undo, Buy, Sell and Exit words sit lower on their plaques.
- [ROM2] The inn and the shop show the hero panel at the lower right as the original does: the hero in full figure or, after the eye or Tab, his statistics, with Previous hero and Next hero below and, in the shop, the spellbook icon that opens the spellbook grid over the table. Hovering or clicking a worn item on the hero or on the inn patron in the left oval shows its item text, and the patron figure now stands where the original draws it.
- [BASE] The game, the map editor and the starter now show their own icons in Explorer, on shortcuts, in the taskbar and in the window title bar: the game's icon, the same with a grid and pencil for the map editor, and the same with a play sign for the starter. Before, the game took the icon from the installed `rom.exe`, and the starter and the editor had none.
- [BASE] Linux x86_64 downloads now include an AppImage and a tar.gz with the game, starter and map editor. The starter can browse for the game folder with zenity or kdialog.
- [ROM2] With no saved game, the pointer on the main menu's Load Game book now shows the LOAD GAME caption like every other icon; a click on it still does nothing.
- [ROM2] The inn shows its patrons as the original does: the first patron is selected on entry, the left card gives the selected patron's name and statistics, the oval shows the patron in full figure, and each seat shows its own patron; a click on a seat now selects that patron instead of starting the talk, and Talk talks to the selected patron.
- [ROM2] The map travel ends one step sooner, as in the original: after START MISSION in the first town the map shows for about 2.5 seconds instead of 2.6, and the first mission shows before its opening dialogue.
- [ROM2] The inn's Talk and Exit and the shop's Undo, Buy, Sell and Exit now show in the original's gold serif letters that turn light yellow under the pointer; Undo is no longer greyed when there is nothing to undo, and a pending purchase shows on Buy as the gold it takes, such as -3. Before, they used the large town font in cream and the purchase showed as 3.
- [ROM2] The hero creation page shows Accept, Reset, Back and Restore on the second game's own four-plaque brown panel, as the shops show four commands, so a click where the original has Reset or Back now gives Reset or Back; Restore is the fourth plaque, below Back.
- [ROM2] The town square, the inn and the shop now show their tip box as in the original: on arrival in a town, on every return to the square and on each inn and shop entry, until SHOW TIPS NEXT TIME is cleared. CLOSE hides the box until the next entry.
- [ROM2] The inn talk box and the first mission's dialogue show the speaker's portrait beside the text, and the text wraps as in the original.
- [ROM2] Hover captions in the town, the shop, the mission and every other screen now look as in the original: small gold capitals in a small box, as on the hero creation screen. Before, they used the large town font in cream, in a box about twice the size.
- [ROM1] A save made in our game no longer sends a stray spell to the map corner when the original game loads it: a mage in the middle of a cast keeps its target, and the bodies on the map and the units that fought them keep their attack timers and kill credit in the saved file.
- [BASE] After loading a saved game, a mage's spells reach as far as the mage's current skill allows from the first cast; before, Teleport could reach only the distance stored in the save, 1 cell instead of 3, until the mage had cast it once.
- [ROM2] Two mission talks that have no words for some heroes now open on `Nothing to say`: one for a hero who is not a mage, and, on the English edition, one for a female mage, who now also sees the rest of that talk. Before, neither talk opened.
- [ROM1] Enemies chasing a hero who teleports or is moved far away now stop where they stand until the hero comes near them again; before, they kept walking after the hero across the map.

### For map makers and modders

- [BASE] Save in the map editor now writes the map over its own file without asking for a name, when the map came from a file of yours or was already saved once; a new map, a map from the game's archive and Save as... still ask. The write goes through a temporary file, so a crash cannot leave a half-written map, and the map's own texts folder beside it is rewritten with it.
- [ROM1] The map editor's height brush has a Set height row: press on a cell, drag up or down to choose a height from 0 to 127, shown beside the pointer and on the status line as the same numbers the height view shows (from -64 to 63), and let go to set that cell to it and blend the cells around it, within the brush size, as the original editor's height tool does; Esc cancels. The status line also shows the height of the cell under the pointer, and the height numbers (V) now show at every zoom, on every second, fourth or eighth vertex when the cells are too small for all of them; before, they vanished when you zoomed out.
- [BASE] The map editor can always save: a map that holds errors in its report now asks once, Enter or S saves it with the errors and Esc goes back, and a texts folder the editor cannot read no longer stops the map file from being written, the status line names what was left out. Before, such a save was refused and the work was lost. W and F5 still refuse a map with errors and now say that Save works.
- [ROM1] The Objects brush in the map editor paints each object with its own picture: big stone 1, 2 and 3, rock 1, 2 and 3 and the two statues no longer look the same on the map. The Objects list also shows the highlighted object's picture beside it, larger than the Add window's. Before, the editor drew the first picture of the sheet for every object.
- [BASE] Map editor windows are easier to handle: Z turns the passability view on and off (shown as Passability (Z) in the View menu); the headings inside a window stand out on a band; a window can be dragged partly off the screen and back, keeping its title and 48 pixels inside; the window's border is 8 pixels wide and its bottom-right corner is a 16 by 16 square that resizes both ways; and in a text field Ctrl+Z undoes and Ctrl+Y or Ctrl+Shift+Z redoes typing.
- [BASE] The map editor's report now shows the text of the selected finding, and Enter on a finding that names no record shows it in full. A map whose texts folder cannot be read, for example a file named Event01.txt, or a map the editor cannot decode, is now reported as texts-folder or map-decode with the cause and file name, not as mission-build. The Russian message after a refused save now says the map and its history were not touched.
- [ROM1] The map editor adds a new trap, unit spell or casting structure on any map, also one without a cell spell to copy; it edits a casting structure's power and scan range (0 to 255), and it edits and copies the many shipped casting structures that sit on another cell of their building. Deleting a structure or a unit also deletes its casting structures or unit spells, and a cell spell left without its object can still be deleted.
- [BASE] A mission message with no page 1 now opens its dialogue on `Nothing to say`, and the next press shows page 2 or closes it; before, it opened nothing. A `tips=` on a page whose text does not read now raises its tip when the dialogue closes there.
- [ROM1] The Forest brush of the map editor no longer plants on the drop cell or on a cell a unit stands on, and its picture grid shows only the object classes that have art; before, a forest could close those cells and the grid had blank tiles.
- [ROM1] The map editor lets you edit every page of a dialogue, choose each speaker, and add, remove or reorder pages in one message; the Map panel warns when a trigger sends a second message the player cannot see while the first is open.

### For developers

- [ROM1] A save made after loading an original save now writes the Token fields of the spell effects on the map from the current game instead of copying the loaded file; a save without such effects is unchanged.

## 0.134.0

### For players

- [ROM1] [ROM2] A spell that was selected when the game was saved stays selected after loading, but with the spellbook closed a click on the map no longer casts it; it casts again once the book is open, or with C, as before the save.
- [ROM1] [ROM2] An invisible unit stays visible to the spells of its own side and its allies: automatic casting keeps renewing Bless, Haste and the other support spells on it, and a spell cast at it from the book starts from the normal cast range instead of the mage walking up to it and waiting. Enemies still do not see it unless they can detect invisible units nearby.
- [ROM1] A Teleport to a neighbouring cell now shows the same short jump whether or not the caster had just walked; before, a caster that had walked first appeared to snap to the new cell without the jump.
- [ROM1] [ROM2] A new mission now starts with the spellbook closed (as in the original first game; the second game is assumed to match). A save made at the start of a mission no longer opens the spellbook when the original loads it; a save made after you open or close the book keeps your choice.

### For map makers and modders

- [ROM1] The map editor lists traps, unit spells and casting structures. F3 adds one from an existing map entry, E edits its spell and power, and Delete removes it; unknown kinds stay read-only. Scan range is shown but cannot yet be edited.
- [ROM1] F5 in the map editor starts a map without a victory condition at once; the Map panel still says the map is endless, and any other warning still asks first. On a second-game map the refusal comes before any notice.
- [ROM1] A mod can now let an armour, shield or weapon come in a material the game never sells it in, with a picture the mod supplies, or take a material away, and can add a new weapon that copies an existing one with its own name, damage, reach and price. Two mods that change the same number or material of one item, or add the same item, now need `load-after` in one of them or the game refuses to start; before, both applied and the later one silently won.
- [ROM1] Every map editor input supports held editing keys, word movement, selection and text copy/paste with Ctrl+C/X/V. New messages are ready for typing, choosing a page name continues directly into its text, and the trigger wizard can create or open a message directly; choosing an existing message applies at once when no other field is pending.
- [ROM1] Guest missions can form a chain with no town between, returning the waiting party to the town the first win opens; a save inside any link continues the chain. Its party can use one to eight characters from npc.reg or the mod, with chosen levels and equipment and human defaults for records with no party template; existing character edits keep their load order, while two mods adding the same new character or mission need load-after.
- [ROM1] The map editor can redraw the whole map's tiles (Ctrl+E) or lay random hills over its heights (Ctrl+Shift+R) from a typed seed, copy the tiles, heights and trees under a selection box with Ctrl+C and paste them with Ctrl+V, start a new map at a chosen height from 0 to 126, and show each cell's height minus 64 with V.
- [ROM1] The map editor's message popup now lists speakers 21 to 24 by role (Main hero, Traitor mage, Monk, Prisoner) instead of a name, because who speaks as each depends on the hero the player chose first; the stored speaker number and the map are unchanged.
- [ROM1] A mod's `data/towns.toml` can now add a talk or a quest to a town's tavern, give a tavern quest another mission, or remove a talk, chapter by chapter, and a town a mod adds can have a tavern of its own; the mod carries the new talk texts as install files. A save made with such a mod keeps the talks and quests and needs the mod to load. A towns mod that removes a talk a companion mod waits for is refused at launch, naming both mods. Without a mod every tavern is unchanged.
- [ROM1] Map scripts can keep a unit from dying with the new "Set undying" action and react when its health falls below a chosen value, even a negative one, with the new "Health below" condition; the editor marks a map that uses them as Againrom-only, because the original game ignores both.

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
