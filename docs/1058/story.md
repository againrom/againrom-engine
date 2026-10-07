# Story 1058: ranged weapon damage and General accuracy

## Player result

Ranged weapons now use the wielder's General level for accuracy. Flame Thrower
and the type-12 ranged arm also deliver their authored elemental damage through
the same canonical third component already used by magic item effects. Dragons
therefore retain their ordinary physical attack and add Flame Thrower's Fire
component instead of carrying a harmless ranged weapon.

## As built

`FoldWeapon` receives the live General source explicitly. Every ranged attack
type adds that value to live ToHit, ignores the weapon's own ToHit and Defence,
keeps active skill slot zero, and leaves the physical pair unchanged. Exact
attack types 11 and 12 add the weapon's byte-width base/spread, with modulo-256
stores, to the existing `SecondaryDamage` triple and select Fire or Earth.
The weapon base write precedes its ordered effect walk. A later elemental item
effect therefore replaces the complete triple, including a zero-valued effect;
a transient presence bit preserves that distinction without entering a save.

Human generation and live `Rearm` supply `Hero.Skill[General]`. The Units route
snapshots the authored ToHit copied into General before placement difficulty,
then carries the folded triple through `UnitDef` and the existing spawn block.
Hard difficulty therefore adds its later `+50` once, not through General a
second time. Simulation state, damage resolution and persistence are unchanged;
there is no save-format change.

## Authority

- `ITEM-WEAPCOL-021`: exact ranged types 11/12 add the weapon byte pair and
  assign encoded kinds 1/2.
- `HERO-GENERAL-090`, `HERO-GENERAL-092` and `HERO-FOLD-035`: ranged equip
  copies live General into the to-hit modifier, clears the active slot and
  folds the modifier into live combat; equipment's General-bonus slot is inert
  on human recompute.
- `HERO-DMG2-029`: the third-component selector permutation maps those encoded
  kinds to Fire and Earth.
- `ITEM-EFFDISP-075`: Weapon equip walks effects after its base writes, and
  elemental effects copy/replace the third-component fields in list order.
- `UNIT-DIFF-002` and `UNIT-GATE-013`: a Units row's authored ToHit is copied
  into General before the placement difficulty adjustment, which reaches live
  ToHit alone.

## Proof and debt

Focused data, map-load, simulation and live-Rearm tests cover General sourcing,
type 10 neutrality, type 11 Fire, type 12 Earth, byte wrapping and preservation
of the base-before-effect replacement order, including a zero-valued effect.
The lawful-install witness passes on EN and RU over all 28 maps: 4 Dragon
definitions and 21 placed Dragons carry Flame Thrower type 11 as `1+U[0,7]`
Fire damage with General folded into ToHit.

`DIV-363` is closed. `DIV-364` remains separate: unsupported melee attack types
still have no decoded resistance selector. Ranged projectile flight time is
also outside this story.
