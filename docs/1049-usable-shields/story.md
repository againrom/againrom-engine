# Story 1049 — usable shields

## Player result

Shields can be worn in the mission inventory and the shop. A shield may be
worn only beside a resolved one-handed weapon: not ranged and not in the
decoded authored `Hands == 2` population. A bare hero or a hero
holding an incompatible weapon keeps an attempted shield at its source.
Equipping a weapon that does not satisfy that rule moves the shield to the
same character's pack. Taking off or dropping the supporting weapon moves the
shield there in the same command. Mission construction, save restoration and
the shop normalize an invalid shield-only pair by the same rule. Every move
preserves the complete item instance, including price and ordered magic
effects.

A worn shield contributes its installed defence and absorption, scaled and
rounded by the same rules as armour. Its magic effects continue through the
ordinary all-slot item-effect fold. Inventory and shop tooltips show the base
protection, value and effects, and the character sheet updates immediately.
Changing between two instances of the same shield code also refreshes combat,
the enchantment marker and the worn tooltip; code, kind, value and ordered
effects are all part of the presentation and rearm change trackers.

The existing figure order remains authoritative: a one-handed weapon paints
before the shield, while a two-handed or ranged body paints the weapon last.
The new compatibility gate normally prevents the latter invalid pair from
persisting, without changing the compositor's total behaviour on restored or
synthetic state.

## Authority and as-built behaviour

- `HERO-EQUIP-017` supplies shield slot 2, the `Hands == 2` shield-equip path
  and the lossless move to inventory. The owner replaces that partial rule
  with the positive one-handed-melee invariant and specifies the direction of
  every coupled move (`DIV-445`).
- `ITEM-ARMFILL-032` and `ITEM-ARMFOLD-033` supply shield base defence and
  absorption from runtime columns 9 and 10, including the factor ladders,
  16-bit stores and distinct rounding rules.
- `ITEM-EFFDISP-075` supplies the existing equipped-effect traversal. No
  shield-specific magic path was added: the complete `ItemInstance` in slot 2
  reaches the same fold as every other slot.
- `HERO-FIGURE-060` supplies the held-layer ordering. `ITEM-ARMFILL-032`,
  `ITEM-HUMEQ-030` and the installed census cover authored shield rows and
  human equipment cells.
- `sim.KindEquip` carries an optional displacement slot. Weapon unequip and
  weapon-to-ground also move slot 2 to the unbounded pack in the same
  deterministic mutation. All indices are validated before state is written.
- Mission and shop equip use the same compatibility predicate. A compatible
  display-only starting-weapon fallback first materializes into slot 1, then
  the shield enters slot 2 before either screen can observe an invalid pair.
- Equipment, containers and item effects were already canonical save/hash
  state. The story changes no save form and adds no parallel projection.

Touched surfaces: shield definition resolution, worn-stat folding, starting
equipment, live inventory commands, shop mutations, character-sheet and item
tooltip projections, save/hash witnesses, and held-layer regression tests.

## Proof

- Focused packages: `go test ./pkg/data ./pkg/mapload ./pkg/sim ./pkg/game
  -count=1`.
- Compatibility and atomicity:
  `TestLiveShieldEquipRefusesBareAndRangedWearers`,
  `TestLiveShieldEquipMaterializesCompatibleStartingWeaponFallback`,
  `TestUnequippingWeaponAtomicallyMovesShieldToPack`, and
  `TestDroppingWeaponMovesShieldToPackInTheSameCommand`.
- Shop and restoration:
  `TestShopShieldRefusesRangedWeaponAndRangedEquipDisplacesShield`,
  `TestShopShieldMaterializesCompatibleStartingWeaponFallback`,
  `TestShopWeaponRemovalMovesShieldToPack`, and
  `TestOpeningRestoredShieldOnlyWorldMovesCompleteShieldToPack`.
- Equal-code instances:
  `TestSameCodeShieldSwapRefreshesCombatPopupAndSave` covers plain-to-magic
  and magic-to-different-magic swaps, the live combat block, tooltip value and
  effect, binary round-trip and hash.
- UI and ordering: `TestItemInfoLinesStatesShieldBaseProtectionAndInstanceMagic`,
  `TestFigureDrawOrderIsAPermutationWhicheverHeldSlotIsLast`,
  `TestComposeUnitFigurePaintsWhicheverHeldSlotIsLast`, and
  `TestComposeUnitFigurePaintsTheWeaponLastForATwoHandedBody`.
- EN and RU `TestReleaseShieldDefinitionAndHumanCellPopulation`: 9 shield
  definitions, 9 positive-defence rows, 5 positive-absorption rows, 46/46
  resolved human shield cells, 5 enchanted cells, 0 rejected cells and 0
  incompatible authored weapon/shield pairs. The first live shield is
  `0x8207` at defence 10 / absorption 1. Both installs also contain the one
  ranged definition `Flame Thrower`, authored with `Hands = -1`; owner-directed
  incompatibility is therefore a shipped-data rule, not only a synthetic edge.

The sole independent review ran against the first pushed candidate and returned
the equal-code refresh and shield-invariant defects. The final correction closed
both classes; the review ceiling permits no second pass. Candidate and merge
gates are recorded at the seat boundary, and the landing journal is written only
after the merge is pushed.

## Open debt

- `DIV-445` records the owner's positive one-handed-melee invariant and its
  coupled mission, shop and restore behaviour beyond the decoded `Hands == 2`
  shield-equip path.
- A bounded/full pack does not exist. If container capacity is introduced,
  the atomic command needs an explicit capacity-refusal witness before use.
- Reserved `DIV-446` through `DIV-450` are returned unused and remain retired.
