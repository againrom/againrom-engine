# 1212 — one writer for every town

## Intent

A town save made after an original SAV was loaded writes SAV whatever the
party changed: equipment, pack, a companion's equipment, a purchase or a sale.
The loaded document supplies only the bytes the engine does not model. This is
M3 of the SAV track (`pipeline/SAV-ENDGAME.md`), queue item 14.

## As built

### One producer for a loaded town

`ExportOriginalSave` (`pkg/game/originalsave.go`, `marshal`) writes every
member of a loaded original town from the current `Snapshot`. The member loop
has two arms:

1. **Unchanged.** An unreturned member equal to the replay of its document
   baseline and admitted school and sale operations
   (`originalCityBinding.replayedMember`). The existing baseline, training and
   sale updates apply, so its bytes are the base's.
2. **Current.** Every other member, including every returned member.
   `currentUpdate` (`pkg/game/originalcity_current.go`) writes:
   - the Human from the member's current class-2 actor load
     (`mapload.SourceHumanState`) with its runtime words; failing that, from
     `OriginalHumanState`; failing that, the derived Human of DIV-1321
     (`nativeCityHumanState` over the member's own document unit). A member
     none of the three fits is an error, not a fallback to `.ags`;
   - `Returned` when the member came back from a mission;
   - the pack (`Holdings`) and worn set (`Worn`, 12 slots) from the current
     items. A member with an actor load supplies its container bookkeeping; a
     member without one (a companion grafted in town) follows DIV-895: present,
     insert index 0, accumulator the written weight;
   - book instance parameters from the current `Book`. A legacy table-backed
     book writes none, so the document's Spell objects stay (DIV-652).

   Before writing, `currentUpdate` refuses a member with a timed potion
   effect (the native writer's rule, `.ags`), and for a returned member
   re-runs `checkReturn` on the captured return and requires no historical
   placement (`Saved`) and no world item handle. After choosing the Human it
   requires the member's carried skill XP to equal the written Human's.

   A pack and worn set equal by value to the reference ones (the returned
   graphs, else the document's own) are written as the returned graphs or left
   nil, so their records and bytes stay.

`CityUpdate.Difficulty` writes the document head's difficulty word, which the
reader already maps to 1..3.

### Item record sources

Each written item record comes from the first source that has it:

1. a record of the member's returned graph or of its own pack or worn set in
   the document, equal by value, copied with its Effect and Spell children, so
   its token words and F47 stay the document's bytes. Each is used once;
2. the native item constructor (DIV-1214, DIV-893): `sim.ConstructSavedItem`
   plus the installed shape, material and F48 operands, with a fresh identity
   above every identity in the document and the returned graphs. An item with
   no concrete class and no weight takes the constructor weight 1
   (ITEM-CLASS-001).

Every written record is decoded again and compared with the value it was
written from; the graphs pass `sav.ValidateCityItemGraph` and
`sav.ValidateCityEquipmentGraph`. The existing `pruneCitySoldObjects`,
`clearUnresolvedCityOwners` and `remintCityIdentities` then apply in `Marshal`.

### `checkReturn`

The allow-list is gone: `returnShape`, `returnStacks` and `returnUpdate` are
deleted. `checkReturn` keeps the integrity checks of the captured return
(identity, version, mission, a valid class-2 actor load, graph validity, no
world handles, a normalized current Human, a valid book).

`captureMissionReturn` is unchanged in what it admits. When a mission ran and
its return was not captured, the loaded town is now marked unavailable
("has a mission return it could not capture"), so SAVE keeps `.ags` and the
mark survives an AGS SAVE and LOAD. On `8499c9b` the same state refused through
the Human refusal, which a later AGS LOAD could not tell apart from a town
edit.

### Refusal classification

`marshal` (loaded town). "AGS" means the ordinary SAVE still falls back to
`.ags`; an invariant is an engine defect.

| Refusal | Class |
|---|---|
| state or document or front end nil | invariant, AGS |
| `state.unavailable` (lost source lineage) | remains, AGS, M8 |
| difficulty changed | removed: head word 12 is written |
| mission != 0 | invariant, AGS: a mission routes to the world writer first |
| pending world-map return | remains, AGS, M9 |
| quick spells not representable | invariant, error |
| gold outside uint32 | invariant, error |
| offered mission advisory | removed (DIV-1322) |
| marker-selection latch changed | removed: LOAD derives it (DIV-1322) |
| world-selection history changed | removed: LOAD derives it (DIV-1322) |
| mission return not captured | remains, AGS, M8: was the Human refusal; now named, and kept across an AGS LOAD |
| member without ID, duplicate ID | invariant, AGS |
| non-siege roster count differs | remains, AGS, M8 |
| hired or siege member checks | remain, AGS, M8 |
| legacy sale history | removed: arm 2 writes the current pack |
| missing restored member | remains, AGS, M8 |
| spellbook membership differs | remains, AGS, M8 |
| legacy table-backed spells | removed: the document's book stays (DIV-652) |
| return changed after completion | removed: arm 2 |
| Human updates are not proven safe | removed: arm 2 |
| trained member has inconsistent Human | invariant, error |
| member has no Human to write | invariant, error, new: a Hero no Human source agrees with; tests reach it only by editing `Hero` directly |
| captured return fails `checkReturn` at SAVE | invariant, error: was the allow-list refusal |
| returned member with `Saved` or a world item handle | invariant, error: was the allow-list refusal |
| carried skill XP differs from the written Human | invariant, error: was the allow-list refusal |
| timed potion effect | remains, AGS, M8: was the Human refusal; the native writer's rule |
| book updates break the alias contract | invariant, AGS |
| current item graph requires .ags | remains, AGS: a returned member's source attached effects, M8 |

`checkReturn`: the structure, container topology, equipment topology and
book-state comparisons are removed; the integrity checks stay as invariants.

`ExportNativeCitySave` and `exportMissionCity` are unchanged by this story.

## Proof

### Witnesses

`pkg/game/citycurrentstate_release_test.go`, registered in
`internal/gatedtests`. Each runs chargen, mission 10, a mission SAVE in
mission 20, LOAD, the rest of mission 20, finish, a town change, a town SAVE
that must be SAV, a cold LOAD and the next town action. All four pass on EN
and RU.

| Witness | Change | On `8499c9b` |
|---|---|---|
| `TestReleaseTownEquipAfterAReturnSavesAsSAV` | hero wears 0x1106 from the pack; next action wears 0x103 | town SAVE wrote `.ags`: "city return member changed after completion" |
| `TestReleasePackChangeAndTownSaleAfterAReturnSaveAsSAV` | 0xf72d picked up after the LOAD, 0x1106 sold in town; next action buys 0x1102 | town SAVE wrote `.ags`: "city return member changed after completion" |
| `TestReleaseCompanionTownUnequipAfterAReturnSavesAsSAV` | npc:22 takes off 0xf82b; next action wears it | town SAVE wrote `.ags`: member "npc:22" changed; Human updates are not proven safe |
| `TestReleaseTownPurchaseAfterAReturnSavesAsSAV` | hero buys 0x1102; next action wears it | town SAVE wrote `.ags`: "city return member changed after completion" |

### Unchanged state

- `check-milestone2-acceptance.sh`, EN: original corpus discovered 102,
  round-tripped 94, refused 8, mismatched 0, as on `8499c9b`. The byte-identity
  census (attempted 102, identical 12, differing 90) prints the same per-file
  lines as `8499c9b`.
- R1 (`earned-return.ags`, game9011, owner resave game0066), N5 (source,
  game9016, owner resave game0071) and N6 (`ordered10-20.ags`, game9017, owner
  resave game0073): conversion or LOAD then town SAVE is byte-identical on
  `8499c9b` and the tip for all nine.
- The 13 census files `8499c9b` exported are byte-identical at the tip.

Bytes change only for a member the player changed: its Human record, its pack
and worn records (reused records copied, constructed records appended, dropped
records pruned, identities reminted), and head word 12 when difficulty changed.

#### Release tests changed with the refusal they measured

Each asserted a refusal this story removes; each now uses a trigger that still
refuses, or asserts the current write.

| Test | Change |
|---|---|
| `TestReleaseSaveDialog1173TownPairOverwriteAndRefusal` | the pair refusal is triggered by a roster edit (M8) instead of an Offered advisory |
| `cmd/saveconvert` `TestReleaseImportedCityConversionAcrossFreshProcesses` | the explicit-export refusal is triggered by a roster edit; the forged-baseline step follows |
| `TestReleaseTownReturnCurrentCampaign1168/imported-campaign-only` | on a loaded town a custom advisory writes SAV byte-identical to the control (DIV-1322) |
| `TestReleaseNativeTownSaveSecondSaveAfterReloadHiredEquipmentStrippedSavesAsSAV` | renamed from `...FallsBackToAGS`; the second save is SAV and reloads with the same worn and pack codes |
| `TestReleaseImportedReturn1169LossControls` | identity, world handle, historical placement, current XP and timed potion still fail; an edited worn item is written; the "producer" control is now the real failed capture, which refuses SAV and keeps its mark across AGS |

## Owner files

`saveconvert -to sav` exports citygo and gooofdfd on EN and RU; 20.ags and
21.ags (mission 20 saves) still refuse there at the world writer, on "current
speed has no original speed index" and "engagement lacks a distinct typed actor
target"; no SAV-ENDGAME milestone names either reason yet. On
EN, each file was also LOADed (20.ags and 21.ags then finished mission 20),
town-SAVEd, which wrote SAV, and cold-LOADed: the reload shows the same party,
worn and pack codes, gold, Valuable Documents and chapter.

| File | Party | Gold | Documents | Chapter |
|---|---|---:|---|---:|
| citygo | hero, npc:25, npc:23, npc:24, npc:22 | 2772637 | none | 130 |
| gooofdfd | hero, npc:25, npc:23, npc:24, npc:22 | 1344926 | none | 130 |
| 20.ags | hero, npc:22 | 600 | text 1, 2, 3 | 30 |
| 21.ags | hero, npc:22 | 600 | text 1, 2, 3 | 30 |

The finished-mission count falls to 0 on reload in all four: SAV carries no
completed-mission set (owner ruling: not important).

### Census

`saveconvert -to sav -assets gameversions/en` over the 114 `engine/saves/*.ags`:

| Result | `8499c9b` | tip |
|---|---:|---:|
| exported (all cities) | 13 | 15 |
| never-imported world, no source-backed constructors | 79 | 79 |
| world with local UI settings, no original application record | 15 | 15 |
| world Group graph with unreachable objects | 4 | 4 |
| city whose hero changed; Human updates not proven safe | 2 | 0 |
| world camera not on an integer cell at original zoom | 1 | 1 |

`TestSAVRoundTrip1195AGSCorpus`'s floor rises 13 to 15 and its "offered
mission" and "completed missions" disclosure ceilings rise 13 to 15 for the
two new exports; the removed reason leaves its baseline.

`missionrun -trace -ticks 1` UNSUPPORTED lines: mission 10 0, mission 20 0,
on `8499c9b` and the tip.

## Open debt

- **One producer, remaining step.** `ExportNativeCitySave` (fresh campaign)
  and `exportMissionCity` stay separate writers. The next step writes a fresh
  town's members through `currentUpdate`'s item and Human sources, so a loaded
  and a fresh town share one member writer.
- A roster change, a spellbook membership change, lost source lineage and a
  returned member's source attached effects still fall back to `.ags` (M8). A
  pending world-map return is M9.
- A mission return on a loaded town is captured only when each member's
  mission start equals the town's replay of its document and admitted school
  and sale operations, and no effect is on it at mission end. Otherwise the
  town keeps `.ags` from then on (M8). Which town edits break that equality
  was not measured.
- The invariant refusals that still select `.ags` become plain errors when AGS
  is retired (M11).
- A constructed item has F47 zero and constructor token defaults (DIV-1214).
- A changed member without an actor load or a matching retained Human is
  written with DIV-1321's derived Human.
