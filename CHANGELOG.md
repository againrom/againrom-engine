# Changelog

What changed for players in each release. A release lists every change since
the previous release, including fixes that reached `main` between releases.
New entries go under Unreleased; a release renames that heading to its version.
Each entry starts with its scope: [BASE] for the engine under both games,
[ROM1] for the first game, [ROM2] for the second game.

## Unreleased

- [BASE] The original game's random number generator can be switched on with
  `-original-random`; `-seed <n>` replays a session from one seed.
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
