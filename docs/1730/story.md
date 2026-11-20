# The second game's inn and shop follow the claimed values

## Result

In the second game the Kaarg inn taverner now plays its long animation through the original's 95 steps, with the
pauses where the original holds one picture. A shop shows its first tip at the entry and a second tip once an item
lies on the deal table, once per visit. The druid inn's lower-right edge starts with the original's picture. The first
game's town screens are unchanged.

## Authority

- Owner: «Чинить всё по очереди» on decision 8 of the second-game triage.
- R2-ASSET-269: the 95 step values of the Kaarg taverner's long series.
- R2-ENGINE-1352, R2-ENGINE-1353: the second shop tip and the list it counts.
- R2-ENGINE-1359, R2-ENGINE-1360: the druid inn panel field starts at 1 and picks the edge picture.
- DIV-2974 closed; DIV-2968 and DIV-2986 narrowed to the Medium and Unknown parts.

## Rules

1. The Kaarg taverner's long series walks the 95 claimed entries, one per step, and ends after the last.
2. A shop popup shows the first tip at entry and the second once the deal table holds an item.
3. The second tip shows once per entry and the next entry starts with the first.
4. The druid inn's lower-right edge is HumanBackL at entry.
5. The first game's shop tips and rooms are unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| taverner scene | `Alternating` in `pkg/town/roomactors.go` | table data of the town 2 description |
| shop room tip | `townScreen.advanceShopSecondTip` | reads the page's own deal; second text is row data |
| inn page art | the town 3 description's art list | one key changed |

## Proof

| test | rules |
|---|---|
| `TestKaargInnTavernerStepsEachSeriesOnce` | 1 |
| `TestReleaseSecondTownKaargInnTaverner` | 1 |
| `TestSecondTipRowDecidesTheRetext` | 2, 3, 5 |
| `TestReleaseSecondTownShopSecondTip` | 2, 3 |
| `TestReleaseSecondTownDruidInnPage` | 4 |
| `TestReleaseRoomTipsAtEveryEnterAndShopSecondText` | 5 |

Release tests at the release: the four named Release tests.

## Open debt

- DIV-2968, DIV-2986, DIV-2972, DIV-2967, DIV-3063 keep their Medium and Unknown parts; the other 12 rows wait for a
  second story and research.
