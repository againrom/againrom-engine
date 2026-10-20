# Campaign record tail at a town SAVE and after a LOAD

## Intent and authority

Every SAV the engine writes carries the campaign record's tail from current campaign state,
by the rules research established in knowledge k338 (EXP-0602): `SAV-1329` .. `SAV-1336`,
with `REG-SCN-063`, `FAME-021`, `FAME-022` and `MERC-POOL-012`. A LOAD of an original file
keeps its AutoGetMission (`+0x110`), and a SAVE before any mission end writes it byte for
byte. Mission-end routing does not change (seat brief). The tail is written by the one SAV
producer: `FrontEnd.ExportCurrentSave` through `currentCityDocument` or
`currentMissionDocument`, then `sav.ProjectDocumentCampaign`.

## As built

| Field | Rule | Claim | Engine before |
|---|---|---|---|
| `+0x118` SelectedMission | the campaign's selected mission | `SAV-1332` | unchanged |
| `+0x110` AutoGetMission | native: the current main record's declaration (20 at main 10, else -1); loaded: the file's word until a main advance loads a new record | `REG-SCN-063`, `SAV-1330` (Medium) | loaded word replaced by the record's declaration at LOAD |
| `+0x11c` LastMission | native: the current main record's declaration (1 at main 150, else 0); loaded: the file's word until a main advance | `REG-SCN-063`, `SAV-1336` | native wrote 0; a main advance kept the loaded word |
| `+0x120` FirstMapPoint | 1 when the party stands on MapPoint 0, else 0; any nonzero word reads as MapPoint 0 | `SAV-1331` | town forced 1, native mission 0, a world-map selection cleared a loaded 1; the reader refused a word above 1 |
| `+0x124` MissionTime | the fame clock | `FAME-021` | unchanged |
| `+0x128` counter | the received hostile corpse-transition counter | `FAME-022`, `SAV-1334` | unchanged |
| markers | append order, once per mission | `SAV-1333` | new markers appended in ascending order |
| hire flags `+0x88` | set by hire, cleared by return, reset and mission end; the pool cell is untouched | `SAV-1334`, `MERC-POOL-012` | unchanged |

### FirstMapPoint

The campaign state answers "the party stands on MapPoint 0": a town arrival and a world-map
return to the city set it; opening a mission clears it; a world-map selection no longer
clears it, because selection does not move the party. When the state says no, the SAVE
writes 1 if the selected mission's map object shares MapPoint 0's point
(`missionAtFirstMapPoint`, `pkg/game/campaigntail.go`). In the EN and RU `GlobalMap.reg`
(35 objects, MapPoint 0 at (215,234)) no mapped mission shares that point, so a mission
writes 0. The second game's documents keep their projection's word.

### AutoGetMission and mission-end routing

`campaignProgressFromSAV` carries the loaded word. `advanceMain` latches the announce flag
when the won record's own declaration names the next main mission, so the route after a
win is the one the engine took before. A loaded 0 or a stale 20 at main 20 still returns
the party to the city (`DIV-1497`).

### Markers

The town screen keeps the registration order (`worldSelectedOrder`). The snapshot lists it
first, and `currentCampaignMarkers` appends each mission the loaded list lacks, so a LOAD
followed by a registration appends after the loaded list.

## Proof

Focused tests, each with its control:

- `TestMissionAtFirstMapPointComparesPoints`: an object on MapPoint 0's point, one elsewhere,
  an invalid object and an unmapped mission.
- `TestReleaseNewCampaignMissionTenTail`: the mission-10 SAVE of a new campaign holds
  selected 10, AutoGetMission 20, LastMission 0, FirstMapPoint 0, time 0, counter 0, no
  marker and no hire flag; the control is the first town, FirstMapPoint 1.
- `TestReleaseTownSAVMarkersKeepAppendOrder`: orders {60,50} and {50,60}, each written
  as registered and extended by 70 after a LOAD.
- `TestReleaseTownSAVHireFlagSetAndCleared`: type 6's flag false, true, false; the pool cell
  unchanged.
- `TestReleaseLoadedTownWritesItsTailBack`: a corpus town with AutoGetMission 0, -1 and 40
  and FirstMapPoint 1 and 0 writes each back.
- `TestALoadedSAVKeepsItsAutoGetMissionUntilAMissionEnd`, `TestCampaignFirstMapPointReadsAnyNonzeroWord`
  and the raw-word round trip in `TestCityCampaignOverlayPreservesUnknownScalarAndWritesScoreHistory`.

Corpus: every `.sav` under `gameversions/saves` (146 files, 5 refused at base and head
alike) was restored and saved at base `c1f8dfbf` and at this branch, on EN and on RU. 30
digests move on each install, and in each the only tail difference is AutoGetMission: 27
files hold 0 where the base wrote -1 (mains 30, 110, 130, 140 and 150) and 3 hold 0 where
the base wrote 20 (main 10). They are exactly the 30 corpus files whose stored word
differs from the record's declaration. No FirstMapPoint, LastMission, marker or hire flag
moves.

## Open debt

- `DIV-1497`: mission-end routing ignores a carried AutoGetMission other than the record's
  declaration; by `SAV-1329` the original travels to the current main mission's map object
  for 0. The owner rules on the travel arm.
- `DIV-882`: no engine route reads LastMission; the 0x428 consumer is Unknown.
- `DIV-903`: the fields a new marker carries beside its identity are not named by a claim.
- `DIV-1412`: the map-name rule rests on 7 original towns; a town before any mission is a
  test witness only.
- The original's reading of a town SAV before any mission is Unknown (`SAV-1332`); the
  owner kit carries one per install.
