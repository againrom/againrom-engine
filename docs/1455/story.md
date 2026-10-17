# The map editor picks a creature's tier and shows a unit's characteristics

## Intent

In the first game's map editor the author sees and sets what makes a unit
strong. A creature whose class has several tiers gets a Tier drop-down in its
popup and in the new-unit flow, and is drawn in its tier's colours. Every
unit's popup shows its values as the game builds them at Normal difficulty and
what its corpse drops. The story started at engine main `4f7b6d70` (game
0.125.4, pin k325); the pin does not move.

## Authority

Owner direction: a tier picker (each creature has four variants, each stronger)
that draws the chosen tier's sprite, and a popup with the enemy's
characteristics, at the middle difficulty where difficulty matters. The owner
split the work: this story picks the tier and shows values; a later story makes
the game apply the map's per-unit override bytes and then edits them.

Claims: `ALM-CLS-038`, `ALM-CLS-052` and `UNIT-PROTO-008` (the secondary key
`+0x0a` selects a Units row by type id and face), `PAL-TIER-004` (a tier is the
class sheet through `palette.pal`, `palette2.pal`..`palette4.pal`),
`ALM-EDITUNIT-278` (Medium: the original editor writes the key from a spin
control clamped to 1..4 for a class `>= 0x40`), `UNIT-GATE-012`, `-013` and
`-033` (Normal is the identity; Easy and Hard scale creatures only),
`ITEM-DEATH-012`, `ITEM-CORPSE-034` and `HERO-KILL-027` (the corpse drops its
own inventory, gold only for a type id above `0x40`), `MISSION-ARM-006` and
`ALM-FLAGPATH-109` (flags bit 0 resolves a person through `npc.reg`). Row:
`DIV-3209` is restated; `DIV-3533` to `DIV-3540` are returned unused.

## As built

### The tier

- `mapedit` gains the unit field `tier`: the u16 at `+0x0a`, 0..65535, check
  `CheckUnitTier`. A set writes those two bytes as one edit.
- `MapInspector.tiersOf` lists the keys 0..255 that `mapload.Resolve` takes
  through the Units arm for a placement's class, one per distinct row, in key
  order; the answer is cached per resolution key. The host offers the field
  only when that list has more than one entry, labelled with the word
  `editor.unit.tier`; its choices read `<key>: <row name>`, the catalogue
  form. A key outside the list is refused with the keys that resolve
  (`tierError`). Human placements, single-tier classes and placements another
  arm takes get no field.
- The new-unit flow (U, a click, the class key) asks the tier when the class
  has more than one: a second prompt of the same builder, its text the key
  `AddUnit` takes by default (the first installed row's), Up and Down stepping
  the choices. `ui.InspectionUnitTiers` is the host seam;
  `mapInspectionEdits.AddUnitTier` places the record, and `AddUnit` is
  `AddUnitTier` with key 0.
- Drawing: the inspector takes each placement's tier from `entityTiers`, the
  function the game's map draw uses, and its frames from
  `UnitClass.TierFrames`. The map, the popup preview and the record label
  (`<class name>  Tier N`) follow it. Owner tint for human classes is
  unchanged.

### Characteristics and loot

- `unitSheets` builds the world with `mapload.FromALMWith(map, table,
  DifficultyNormal)` and reads placement i's actor (entity i), its worn items
  and its carried items. No value is computed in the editor; `valuesOf`
  reads the actor's fields. Two ranges are read as the simulation rolls them:
  damage base to base plus spread, gold minimum to minimum plus maximum.
- The popup gains two headed sections after the record data: "At Normal
  difficulty" (the scaling note, the Units row or Humans template row, the
  named-character line for flags bit 0, health and mana, Body, Mind, Spirit,
  Reaction, to-hit, damage, defence, absorption, speed, sight, experience, the
  five protections, the five resistances, the six skills, the spells, and for
  a person the equipment by item name) and "Corpse drops" (carried, worn, the
  weapon or "innate, not dropped", gold chance and range, or "drops nothing"
  for an NPC-named template).
- `ui.InspectionLine` role `heading` draws as the popup builder's heading row.
- Hover: `ui.InspectionRecord.Hover` is shown in the existing status strip
  while the pointer is on the record's cell: label, tier and health.
- `sim.ItemInstance.InnateWeapon` exports the death drop's own test.
- Words: 28 `editor.unit.*` ids in English and Russian. Words are encoded for
  the font; installed names (rows, items, spells) keep the install's bytes.

## Proof

Focused: `pkg/mapedit` `TestSetUnitTierWritesTheSecondaryKeyWord` (the two
bytes, Undo, Redo) and the field-table tests; `pkg/ui`
`TestNewUnitAsksTheTierOfAMultiTierClass` and `TestHoverLineAndHeadingLine`.

Release, EN and RU (`pkg/game/mapeditunittiers_release_test.go`):

- `TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes`: the first
  multi-tier creature of the installed maps, a tier chosen through the
  drop-down and Apply: the bytes equal `SetField(KindUnit, i, "tier", v)` on
  the installed bytes, 1 or 2 bytes change, Undo restores the installed bytes,
  Redo the edit, the preview's palette is the chosen tier's, and a key that
  selects no row is refused and writes nothing.
- `TestReleaseMapEditorNewUnitTakesTheChosenTier`: U, a click, the class and
  the tier on a new map write `NewUnitRecord(class, tier)`; the default tier
  equals `AddUnit`'s key.
- `TestReleaseMapEditorUnitValuesEqualTheGame`: every installed map opened in
  the editor; each placement's popup values, worn and carried item codes and
  corpse rule equal the actor `StartMissionFrom` spawns at Normal difficulty,
  read field by field by the test; the popup's lines carry the compared
  numbers. EN: 38 maps, 8094 placements compared, none skipped. RU: 34 maps, 3991 placements compared, none skipped.
  Loss control in the test: a perturbed to-hit and a perturbed weapon are
  reported. A run with `valuesOf` reporting to-hit plus one failed on all 8094 EN
  placements.
- `TestReleaseMapEditorUnitTierCensus` (counts below).
- `TestReleaseMapEditorUnitTierFrames` writes the owner frames.

### Tier census

EN: 6672 Units-arm placements on 38 maps; multi-tier placements per key
1: 1081, 2: 1168, 3: 1061, 4: 3336. Thirteen classes have four tiers, keys 1
to 4, each with four palettes: 64 Goblin_Pike, 65 Orc_Bow, 66 Ogre, 68 Troll,
69 Ghost, 70 Bat_Sonic, 71 Dragon, 73 Bee, 74 Squirrel, 75 Foot_Animated, 76
Turtle, 79 Goblin_Sling, 80 Orc_Sword. No shipped (class, key) of the Units
arm resolves to no row. RU: 3366 Units-arm placements on 34 maps; per key
1: 783, 2: 950, 3: 816, 4: 810; the same thirteen classes with keys 1 to 4; no
unresolved (class, key).

## Open debt

- The game ignores most per-unit override bytes of a placement: stats, skills,
  resistances, mana, defence and absorption, and flags bit 7
  (known defect B19). The popup shows what the game builds, so it shows the
  row's values for such a placement. Applying and then editing the overrides is
  the next story.
- The original editor clamps the key to 1..4 (`ALM-EDITUNIT-278`, Medium); the
  editor offers the keys whose rows the install holds, which are 1..4 on both
  roots.
- The gold chance is printed as the death roll compares it, `chance in 101`;
  a claim on the original's displayed wording is not sought.
