# One worn-equipment transaction

## Intent and authority

Owner decision on the architecture audit at `1210d5e6`, rank 2: the Spell
attachment of a classed weapon gets one builder. This is a unification. No
player-visible behaviour, World hash, SAV byte or frame changes, so there is no
CHANGELOG line and no divergence row.

The rule's authority stays the claims cited by hotfix `b8f976ef`
(`docs/HOTFIXES.md`): `ITEM-187` (removal deletes the Spell; the next equip
builds one), `ITEM-DEATH-012` (the death route), `ITEM-212` (a hand-over, a
join and a SAVE build nothing). The second game keeps a Weapon's Spell through
take-off and builds none on equip; that stays one profile row, now asked of the
holder (`KeepsWeaponSpell`).

## The transaction as built

`pkg/sim/wornchange.go`:

- `WornHolder` is the holder of a change: a World actor or a town city object.
  It answers the profile row and writes an item's Spell where it keeps it:
  `KeepsWeaponSpell`, `DropWeaponSpell(slot, *item)`,
  `BuildWeaponSpell(slot, *item)`.
- `TakeOffWorn(h, slot, item)` applies the take-off rule once: outside the
  second game's row, a classed Weapon holding a Spell has it dropped by the
  holder. It returns the item to place.
- `WearWorn(h, slot, item)` applies the wear rule once: outside the second
  game's row, a non-empty item holding no Spell has the holder build one. It
  returns the item to place and whether a Spell was built.
- Both refuse a slot outside 1 to 12. The slot and container writes stay with
  the caller.
- `WornItemEqual(a, b)` is the held-item identity across worn changes: equal
  but for the Spell.

Holders:

| Holder | File | Spell store | Build |
|---|---|---|---|
| `worldWornHolder{w, i}` | `pkg/sim/wornchange.go` | `replaceObjectSpell` (the saved-object registry when the item has an identity) | the Spells rule of the castSpell Effect, the constructor's class for a classless item (the former `buildWornWeaponSpell` body) |
| `townWornHolder{table, g, id}` | `pkg/game/savweaponspell.go` | the item value; with the city topology, also the Spell edge of node `id` | `nativeCityWeaponSpell` over the installed Spells rows (the former `equippedWeapon` body) |

`World.takeOffWorn(i, slot, *item)` and `World.wearWorn(i, slot)` are the
World's calls; `wearWorn` places the built item in the slot and undoes it when
the registry check fails, as the former `buildWornWeaponSpell` did.
`takeOffTown` and `wearTown` are the town's calls.

Removed: `World.takeOffWeaponSpell`, `World.equipWornSpell`,
`World.buildWornWeaponSpell`, `takenOffWeapon`, `equippedWeapon`,
`dropCityItemSpell`, and the swap's `sameHeld`.

### Former call sites

| Route | Before | Now |
|---|---|---|
| `pkg/sim/equip.go` saved-object equip, plain equip (wear) | `equipWornSpell(i)` | `wearWorn(i, slot)` |
| `pkg/sim/equip.go` plain equip: held item, displaced item; plain unequip | `takeOffWeaponSpell` | `takeOffWorn(i, slot or displace, ...)` |
| `pkg/sim/drop.go` saved-object and plain drop from a worn slot | `takeOffWeaponSpell` | `takeOffWorn(i, slot, ...)` |
| `pkg/sim/savedobjectground.go` saved-object death loot | `takeOffWeaponSpell` | `takeOffWorn(i, slot, ...)` |
| `pkg/sim/savedobjectsmutation.go` saved-object equip (held, displaced), unequip | `takeOffWeaponSpell` | `takeOffWorn(i, slot or displace, ...)` |
| `pkg/sim/step.go` plain death loot, weapon slot | `takeOffWeaponSpell` | `takeOffWorn(i, 1, ...)` |
| `pkg/game/shoproom.go` town wear | `equippedWeapon` | `wearTown` |
| `pkg/game/shoproom.go` town wear's old and other items, value route | `takenOffWeapon` | `takeOffTown` after the city branch |
| `pkg/game/shoproom.go` town unequip to the pack and to the table | `takenOffWeapon` | `takeOffTown` |
| `pkg/game/cityshopmutation.go` city wear's displaced items | `takenOffWeapon` (in the value route) then `dropCityItemSpell` | `takeOffTown` with the node's holder after the node moves; a node minted for an item without one takes the value holder before the mint |
| `pkg/game/cityshopmutation.go` city unequip | `takenOffWeapon` then `dropCityItemSpell` | the same shape as city wear |
| `pkg/game/weaponswap.go` swap comparison | `sameHeld` | `sim.WornItemEqual` |

### The weapon swap comparison

The swap still compares held items without the Spell, in both games. It finds
the weapon it took off in the pack and checks that the worn weapon is the one
it put on. Between those two looks the item went through `TakeOffWorn` and
`WearWorn`, which drop and build the Spell, so a Spell-aware comparison would
lose the weapon in the first game. Making it profile-aware for the second
game would change which item the second game's swap matches, a behaviour
change. The comparison now lives beside the transaction (`sim.WornItemEqual`),
so the rule and the identity that tolerates it are defined in one file.

### Town edge write

`takeOffTown` clears node `id`'s Spell edge whenever the placed value holds no
Spell, in the first game and for any class, as `dropCityItemSpell` did. The
clear sits in the town holder (`townWornHolder.dropNodeSpell`), which
`DropWeaponSpell` also uses; the transaction's rule for the value is
unchanged. Every town take-off and the displaced items of the wear route pass
through it. A worn fighter weapon with a castSpell Effect, loaded from a town
SAV, has a node edge and a value without a Spell; taken off onto the shop
table it now lands there, as at base, with the same SAV.

## Second implementation guard

`internal/archtest/wornspell.go` counts each production `pkg/sim` and
`pkg/game` file's Spell deletions: an assignment of the empty
`SourceItemSpell` or of 0 to a `.Spell` selector, and a `replaceObjectSpell`
call with the empty `SourceItemSpell`. `pkg/sim/wornchange.go` and
`pkg/game/savweaponspell.go` are exempt. `CommittedSpellDeletionSites` holds the
remaining files (8 sites in 5 files): the SAV projections that split a Spell
into its own row, the registry's own row write, the native basis comparison
copy and the source actor's removal virtual. A new file or a rise fails;
a fall lowers its row.

## Proof

Focused tests, each with a loss control:

- `TestWornChangeAsksTheHolderOnlyUnderTheRule` (`pkg/sim`): a recording holder
  is asked to drop only for a classed Weapon holding a Spell, outside the
  second game's row and inside slots 1 to 12, and to build only for an item
  holding none.
- `TestWornItemEqualLeavesOutOnlyTheSpell` (`pkg/sim`).
- `TestWorldAndTownTakeOffLeaveTheSameStaff` (`pkg/game`): the unequip command
  on a World and the town holder give one item state without a Spell; the
  second game's town row keeps it.
- `TestTownTakeOffDropsOnlyItsCityNodeSpellEdge` (`pkg/game`).
- `TestTownTakeOffClearsTheEdgeOfAValueWithoutASpell` (`pkg/game`): a value
  without a Spell loses its node edge for a class the transaction leaves
  alone; the second game keeps it.
- `TestReleaseLoadedCastWeaponUnequipsToTable` (EN and RU): a hero buys and
  wears a fighter castSpell weapon, town SAVE and LOAD, unequip onto the
  table. Without the clear it fails with "bound city stock has stale child
  edges"; with it the weapon is on the table with no edge. The wear route's
  displaced item uses the same function; its edge is cleared again by the
  pack projection, so no test reads it.
- `TestReleaseWorldAndTownWornChangesGiveTheSameStaff` (EN and RU): the first
  town companion's installed staff, taken off and worn again through the World
  (unequip and equip over the installed Spells rules) and through the town
  holder over the installed table, is the same item state at each step, and
  worn again holds the Spell it joined with.
- `TestLiveSpellDeletionsMatchTheirBaseline`, `TestSpellDeletionSitesAreFound`
  (`internal/archtest`).

Loss controls, run by editing the transaction and restoring it
(`review/story1479-worn-items/loss-*.txt`): a take-off that never drops fails
`TestEveryRemovalDeletesTheWeaponSpell`, the recording-holder test and both
`pkg/game` holder tests; a wear that never builds fails
`TestNativeEquipBuildsTheWornWeaponSpell`,
`TestASpelllessSpellWeaponWornAgainCasts`,
`TestAnyOwnerEquipBuildsTheWeaponSpell`, the recording-holder test and the EN
release test; one more empty-Spell write in `pkg/game/currentobjects.go` fails
the live archtest.

The existing witnesses pass on EN and RU: `TestReleaseTavernHireWearsTheBuiltStaffSpell`,
`TestReleaseWornGapStaffTextAndCardFollowTheWeapon` (`wornspellroute_release_test.go`),
`TestReleaseALoadedSpelllessStaffAttacksWithoutACast`,
`TestReleaseTownStaffTakenOffHoldsNoSpell` (`spelllessweapon_release_test.go`).

## Gates

Run on `9499c91a` (the unification on base `86ee5494`) and, where named, again
on `0ec91b86` (the same with engine main `52593558` merged). Receipts are under
`review/story1479-worn-items`.

| Gate | Result |
|---|---|
| Release set EN, RU (`check-release-tests.sh`, headless only) | 1341 gated tests per root, 0 failures, on both commits. The only analyzer refusal is `TestReleaseActivePauseRenderedWorld`, skipped by `AGAINROM_HEADLESS_ONLY` because it needs a window. |
| Milestone-2 acceptance EN and RU | exit 0 on `9499c91a`. Output is identical to the `86ee5494` chain's after normalizing timings and the root spelling, with 0 `ResumeOriginalSave` refusals and the same unit-combat exclusions. |
| Mission-stage scenarios EN and RU | 76 of 76 on both commits. All 76 logs are byte-identical to the `86ee5494` chain's (on `9499c91a`) and to the `52593558` chain's (on `0ec91b86`). No scenario writes a SAV. |
| Second-game SAVE selection, rom2-en and rom2-ru, by hand | `TestReleaseSecond*` in `pkg/game`, `pkg/secondshop` and `pkg/video/smacker`: 112 pass and 0 fail per locale. One skip per locale, `TestReleaseSecondPhysicalSyntheticCityAppRoute`, which falls outside the chain's subject groups and needs `AGAINROM_SAVE_CORPUS`. |
| `go test -trimpath -count=1 ./...` | pass on both commits |
| `gofmt -l`, `go vet` (sim, game, archtest), `git diff --check` | clean |
| `check-no-game-assets.sh`, `check-div-claims.sh` | clean, exit 0 |

No World hash, SAV digest or frame that the release set pins moved, and no
scenario log moved. The second game runs the changed code with
`KeepsWeaponSpell` true. Both transactions then return the item unchanged, as
the removed functions did. Its swap comparison is the same Spell-blind
comparison, and nothing in its selection moved.

## Open debt

- The source actor's removal virtual (`sourceRemoveEquipmentTraced`,
  `pkg/sim/sourceequipmove.go`) drops a removed Weapon's Spell unconditionally,
  and its attach builds one from the first kind-41 Effect
  (`sourcePrepareWeaponTraced`). It is the original's own item virtual with its
  own trace and has no profile test. It is outside the eleven World sites the
  audit named and stays a counted site in the archtest baseline.
- The join and hand-over builds (`joinedWornSpells`, DIV-3362) are not wear
  or take-off routes and keep their own code.
