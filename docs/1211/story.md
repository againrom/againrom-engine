# 1211 — a town return after a weapon change saves as SAV

## Result

A mission-20 equipment change no longer decides whether the town saves as
SAV. After a mid-mission LOAD, a weapon swap, a picked-up weapon or armour
worn, an added shield, a removed weapon and the corpus-route bow equip each
write `game0000.sav`. The town LOAD of that file shows the changed
equipment and the drawn class it implies. Before this story each of these
cases wrote `.ags`.

## Owner report

In the RU build the owner changed the hero's weapon during mission 20, and the
town could not be saved as SAV. The fix covers a player who LOADs a save in
the mission and then changes equipment, as well as the owner's own file.

## As built

### Captured worn set

`CityEquipmentGraph{Objects, Slots [12]uint16}` (`pkg/formats/sav/city_holdings.go`)
is the worn counterpart of the pack's `CityItemGraph`. `Slots` follows
`mapload.MemberItemEquipment`: 0 is the weapon, 1 the shield, 2 to 11 the
armour slots. Zero is an empty slot. `applyCityEquipment` writes
`reference74`, `reference78` and `equipment[2..11]` of the Unit.

`pkg/game/cityholdings.go` captures it from the mission's saved-object
registry (`captureCityEquipmentGraphs`, `cityEquipmentGraphFromWorld`) and
checks it against the member (`checkCityEquipmentGraph`).
`SnapshotCityReturn` gains `Worn`. Version 1 carries neither graph, Version 2
carries `Holdings`, Version 3 carries `Holdings` and `Worn`.

### Return comparison

`returnDerived` (`pkg/game/originalcity_return.go`) removes the fields that
follow from the worn set from every comparison of two members:

- `Weapon` is dropped, and the starting-weapon latch is set through
  `materializeStartingWeapon`, its one writer, on both sides.
- A composed member's `Body`, `BodyDir` and `Class` are dropped. They are
  `data.HeroAppearance` of the figure equipment and `Mage`, which the mission
  build, the town faces and a SAV restore each recompute. The retained party
  keeps the values computed when the mission opened. A hired member's `Class`
  is his row and stays compared.

`returnShape` applies `returnDerived` for every version. Version 3 checks the
captured pack and worn graphs against the member. Version 1 and 2 returns
have no worn graph, so `checkReturn` still compares their equipment codes
through `MemberItemEquipment`. Version 1 also compares the pack topology. An
equipment change on a Version 1 or 2 return therefore still refuses.

`returnUpdate` compares `returnDerived` of the finished member with the
current one. A town room's faces rewrite `Class` and `Body`, which before
this story refused with `city return member changed after completion`.
`seedImportedCityObjects` uses the same rule.

### Modifier block and OwnWeight

`checkReturn` no longer compares the Human's 64-byte Modifier block or
`OwnWeight` with the entry. The writer projects both from the current Human
(`cityHumanUpdate` over `SourceHumanState`).

The sim applies each equip and unequip event's fold to the block as the event
happens, starting from the entry value (`SAV-HUMFOLD-446`, `SAV-HUMEQUIP-447`).
The written block is therefore the entry value with the published folds
applied in the order the player took. A recomputation from the end state
alone is not established: kinds 11 and 12 assign the to-hit word from
General, and removal subtracts the weapon's own word, which is not an inverse
(`SAV-HUMEQUIP-447`). On the corpus bow case the entry bytes would write to-hit
3 where the live hero carries 10. No Modifier byte is left at its entry value.

### Picked-up worn items

`remintCityIdentities` repairs a stale owner Reference on the base-document
parse through the holder Human's own container edges. It covered
`HeldWeapon`, `HeldShield` and the pack. It now also covers the worn armour
slots (`pkg/formats/sav/city_semantic.go`). A picked-up armour worn in the
mission, whose Reference names the ground Sack it came from, refused with
`sav: city Armor owner references unknown identity 0x010000b5` before.

### Owner Reference on export

The capture keeps every graph item's owner Reference, as base did.
`clearUnresolvedCityOwners` (`pkg/formats/sav/city_holdings.go`) runs after
the prune on both export paths. It writes 0 for an added Item or Effect
whose Reference names no object left in the document. That is the value the
original's Token load gives an absent key (`SAV-PTRMAP-035`).
`remintCityIdentities` then translates every resolving Reference.

## Reachability, base against the new tip

RU root. The restored route is chargen, mission 10, the mission SAVE in
mission 20, LOAD, the change, finish, town SAVE. The corpus route is
`2026-08-15/game0010.sav` to mission 30. Base is `817e648`; the reviewed head
is `7281210`. The base and reviewed-head columns are the review's
measurement. The new-tip column is `zz_review1211_reach_test.go` with
`R1211_VIASAV=sav` and the corpus probes, run on `7f47a88`. Every new-tip row
is Version 3; every restored-route row reloads through the town LOAD.

| Case | Base | Reviewed head | New tip | Hero after the town LOAD |
|---|---|---|---|---|
| no change (control) | SAV | SAV | SAV | `0x103`, class 3 |
| after LOAD, swap `0x103` for `0x1106` | .ags | .ags | SAV | `0x1106`, `0x103` in pack, class 5 |
| same swap before the mid-mission SAVE | SAV | SAV | SAV | `0x1106`, class 5 |
| swap, then swap back | SAV | SAV | SAV | `0x103`, class 3 |
| pick up bow `0x8134`, equip | .ags | .ags | SAV | `0x8134`, class 14 |
| pick up `0x9127`, equip | .ags | .ags | SAV | `0x9127`, class 10 |
| pick up `0x8134`, keep in pack | .ags | SAV | SAV | `0x8134` in pack |
| pick up armour `0xf72d`, keep in pack | .ags | SAV | SAV | `0xf72d` in pack |
| add shield `0x1222` from mission 10 | .ags | .ags | SAV | `0x103` and `0x1222`, class 4 |
| wear `0xa1a` from the pack | SAV | SAV | SAV | `0xa1a` worn |
| unequip armour slot 7 | SAV | SAV | SAV | `0xb72f` in pack |
| pick up `0xf72d`, wear it in slot 7 | .ags | .ags | SAV | `0xf72d` worn |
| pick up `0x812`, wear it | .ags | .ags | SAV | `0x812` worn |
| weapon removed | .ags | .ags | SAV | unarmed, class 1 |
| corpus: move bow `0x8134` to `npc:22` | SAV | SAV | SAV | `0x8134` gone from the hero's pack |
| corpus: equip bow `0x8134` | .ags | .ags | SAV | `0x8134`, `0x106` in pack, class 14 |
| corpus: equip bow, then swap back | SAV | SAV | SAV | `0x106`, class 5 |

The corpus control exports 3265 bytes on the new tip, as on base. The hero's
`HeldWeapon` `0x106` and worn `0x162a`, `0x1833` and `0x916` keep Reference
`0x1000010`, which loads as the Player.

## The owner's file

`review/owner-m20-inputs/21.ags` was read from a byte-identical scratch copy
and is not a fixture. At base (`817e648`) the export refuses with
`hero: city return changed unsupported character structure`. On the
new tip it resumes at mission 20, finishes, and exports 2758 bytes. The town
LOAD shows the hero with `0x1106` held, `0x103` in the pack and class 5, and
the companion `npc:22` with `0x810d`.

- `0x1106` was carried in from mission 10's sack at (20,65). It is not a
  mission-20 pickup.
- `0x8134` is a mission-20 pickup. Its owner Reference `0x51000b50` is the
  mission-20 ground Sack at (15,11). The export writes that Reference as 0.
  Every other hero item's Reference translates to the hero's Human.

## Saved bytes that change

- An added graph item's owner Reference that names no written object is
  written as 0. Base refused the export. A resolving Reference is translated,
  as on base.
- The Human Modifier block and `OwnWeight` are written from the current Human
  when they differ from the entry. Base refused the export.
- A worn item whose parsed owner names no object is repaired to its wearer.
  Base refused the export.

No `World.Hash` or scenario hash moved. `pkg/sim` is untouched, and the
full test run passes, including the envelope and scenario hash tests.

## Proof

### Witnesses

`pkg/game/cityequipmentreturn_test.go`, registered in
`internal/gatedtests/testdata/population.txt`. Each passes on the new tip.
Each fails on `817e648` with the refusal below (RU root, save corpus).

| Test | Failure on `817e648` |
|---|---|
| `TestReleaseLoadedMission20WeaponSwapSavesTheTownAsSAV` | `hero: city return changed unsupported character structure` |
| `TestReleaseLoadedMission20PickedUpArmourWornSavesTheTownAsSAV` | `sav: city Armor owner references unknown identity 0x010000b5` |
| `TestReleaseLoadedMission20ShieldAddedSavesTheTownAsSAV` | `hero: city return changed unsupported character structure` |
| `TestReleaseImportedTownBowEquipSavesTheTownAsSAV` | `original-compatible town save member "npc:22" changed; Human updates are not proven safe and require lossless .ags` |

The three restored-route tests open a town room before the SAVE, so the town
faces' rewrite of `Class` and `Body` is inside the witness. The bow test also
requires the three unchanged hero armours to keep a Reference that loads as
the Player.

The earlier armour test is removed. It passed on a harness name mismatch.

### Changed controls

The owner direction removes two earlier refusals.

- `TestReleaseImportedLootUnclosedOwnerReferenceWritesZero` replaces the
  unclosed-reference control. The town SAVE and the converter both write SAV,
  and the item's Reference is written as 0.
- The `cleanup` case of `TestReleaseImportedReturn1169LossControls` becomes
  `current modifier`: a changed Modifier byte writes SAV, and the written
  Human carries the current block.
- `TestCityGraphOwnerReferenceKeepsAResolvingKeyAndZeroesAMissingOne`
  (`pkg/formats/sav`) marshals a pack whose Reference names first a missing
  key, which is written as 0, and then the hero, which is written as a Human
  identity.

### Census

`missionrun -mission N -trace -ticks 1 | grep -c UNSUPPORTED` on the EN root:
mission 10 = 0 and mission 20 = 0, as on master. The story changes no script
or simulation code.

### Gates

Run once on `1db12fa`. The commit that records them changes only this file.

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...`: exit 0; 54 packages `ok`, 33 with no
  test files, none `FAIL`.
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `pipeline/check-release-tests.sh <en> <ru>` with `AGAINROM_IMPL` set to
  this worktree: `9 package(s), 331 gated test(s), 2 root(s)`; EN `ok (331
  of 331 ran, 0 lacked a subject)`; RU `ok (331 of 331 ran, 0 lacked a
  subject)`; exit 0.
- `scripts/check-milestone2-acceptance.sh <en> <ru>`: exit 0 on both roots.
  `SAV-ROUNDTRIP-AGS-CENSUS discovered=114 round-tripped=0 disclosed=13
  refused=101 mismatched=0` and `SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=102
  round-tripped=94 disclosed=0 refused=8 mismatched=0`, as before this story.
- `pipeline/check-preserved-installs.sh`: `ok — 554 file(s), every root as
  recorded`, before and after each command that read an install.
- `internal/storyguard`: `TestIdentCount` falls from 6068 to 6065 and
  `CommentBytes` rises from 7724118 to 7726353. The note in
  `internal/storyguard/baseline.go` names every file that grew.

## Open debt

- **Version 1 and 2 returns.** When the mission world has no saved-object
  registry, or the pack or worn graph cannot be projected, the return has no
  worn graph. An equipment change then still refuses on
  `city return changed unsupported equipment topology`, and a pack change on a
  Version 1 return on `unsupported container topology`. The writer has no
  current graph to write there. Writing the base document's equipment would
  drop the player's change. No row of the table above reaches this path.
- **Town changes after the return.** A town equip, purchase or sale made after
  the return and before the SAVE was not measured on this route.
  `returnUpdate` still refuses a member whose non-derived fields changed after
  the finish.
- **Where the original puts a displaced weapon.** The swapped-out weapon goes
  to the pack position the live mission container gives it. This was not
  measured against ROM1.
