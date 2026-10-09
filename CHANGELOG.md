# Changelog

What changed for players in each release. A release lists every change since
the previous release, including fixes that reached `main` between releases.
New entries go under Unreleased; a release renames that heading to its version.
Each entry starts with its scope: [BASE] for the engine under both games,
[ROM1] for the first game, [ROM2] for the second game.

## Unreleased

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
