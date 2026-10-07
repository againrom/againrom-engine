# 0142 — verification

## Gates

Run from the lane worktree, on a clean tree, at `8afe632` + the `verification` commit's tree.

```
go build ./...                     clean
go vet ./...                       clean
gofmt -l $(git ls-files '*.go')    prints nothing
go test -trimpath -count=1 ./...   all packages ok
bash scripts/check-no-game-assets.sh
bash scripts/check-doc-budget.sh
bash scripts/check-sdd-audit.sh
bash scripts/check-hotfix-ledger.sh
git log --format='%h %(trailers:key=Co-Authored-By)' master..HEAD
```

`check-sdd-audit.sh`'s **FAIL set for 0142 is empty**; only the FAIL set is reported, because a
worktree has no `builds/` and the note/warning count is meaningless from here in both directions.
The audit's one note against this story is `no tasks.md (fanned out to no executor, or in flight)`,
which is the intended state: one lane implemented its own slice, so every `FR` and `DD` is
accounted for below instead.

The `Co-Authored-By` query prints a bare hash and an empty field for every commit on the branch.

## The result someone can point at

**The script-gap census did not move, and that is the true claim.** This story adds no script node
and touches nothing `pkg/sim` reads.

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 17
AGAINROM_ASSETS=<en> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 13
```

`pipeline/milestone-baseline.txt` carries m10 = 1 + 2 + 13 + 1 = **17** and m20 = 1 + 11 + 1 =
**13**. Both unchanged.

What moved is in `builds/`: a walk the owner can do, and a headless line that used to be wrong.

## The walk, as it was run

Driven over the lawful `en` install through the **real** `FrontEnd`, the **real** `Town` and the
exact five `ui.TownScreen` methods `pkg/ui` calls, by a temporary `cmd/townwalk` deleted before the
story landed. What is elided below is repetition of the footer.

```
[win mission 20] againrom: winning mission 20 leads to the town, chapter 30;
                 tavern holds npc 22 -> 30; shop holds 31; shop prices 0-1000; gold 0; party 1

--- the square
the town square — chapter 30    [Esc: back]
  * 0  tavern    1 with something to say
  * 1  shop      1 on the shelf
  * 2  school    0 on the board
  * 3  gates     0 mission(s) available
    | gold 0    party 1    carried 0 items, 3 worn
    | available []    done 1

--- the gates, before anyone has spoken
    0  the road is empty — nobody in the town has given you work yet

--- the tavern
  * 0  NPC 22   wants to talk

--- NPC 22 ... three presses ...
    0  NPC 22 looks up as you sit down.
    1  "There is work, if you have the stomach for it."
    2  "Mission 30. Take it or leave it."
  * 3  accept mission 30
  * 4  leave him to it
  >> mission 30 is now waiting at the gates

--- the square, after accepting
  * 0  tavern    0 with something to say
  * 3  gates     1 mission(s) available
    | available [30]    done 1

--- the gates, after accepting
  * 0  walk out to mission 30
  >> you walk out of the gates towards mission 30
  >> an opener crossed: this row walks out to a mission
```

`done 1` at the square is mission 20, marked won by `AdvanceLine` — plan R-2's disclosed
consequence of that method calling `FinishMission` on a mission it merely started.

### The headless line, both roots

```
AGAINROM_ASSETS=<en> builds/0142-the-town/againrom.exe -check -mission 20
  againrom: winning mission 20 leads to the town, chapter 30; tavern holds npc 22 -> 30; shop holds 31; shop prices 0-1000; gold 0; party 1
AGAINROM_ASSETS=<ru> builds/0142-the-town/againrom.exe -check -mission 20
  againrom: winning mission 20 leads to the town, chapter 30; tavern holds npc 22 -> 30; shop holds 31; shop prices 0-1000; gold 0; party 1
AGAINROM_ASSETS=<en> builds/0142-the-town/againrom.exe -check -mission 30
  againrom: winning mission 30 leads to the town, chapter 40; tavern holds npc 22 -> 40, npc 90 -> 41; shop prices 0-3000; gold 1000; party 1
```

**G1's two roots agree.** The third line is FR-2 and FR-9 together on real data: the chapter moved
30 → 40 because 30 was won, and the gold is `[Mission30] Payment = 1000`.

### The windowed build

`builds/0142-the-town/againrom.exe` was built by the README's own command and launched by the README's own
invocation against `<en>`; it ran for 12 s and printed nothing to either stream before it was
killed. **The click-through of the walk in the window was not performed here** — what is witnessed
above is every press of it through the same seam the window drives, plus the application's own
input dispatch in `TestTheApplicationDrivesTheTown`.

## Every id, and what witnesses it

There is no `tasks.md`, so this is where the ids are accounted for.

| id | witness |
|---|---|
| FR-1 | `FinishMission`'s `offer.Town` arm calls `Town.Arrive`; `TestAWinAtTheTownsBoundaryOpensTheTown`, `TestATownOfferOpensTheTownAndTheChapterIsDerived`, `TestANoticeCanSendTheFrontEndToTheTown`; the walk above |
| FR-2 | `Town.Chapter` recomputes; `TestTheChapterIsDerivedAndASideMissionDoesNotMoveIt` (the side-mission case is the discriminator), and the `-check -mission 30` line |
| FR-3 | `continuity`'s two arms; `TestAWinAtTheTownsBoundaryOpensTheTown`, `TestALostMissionReturnsToTheTownOnceItIsOpen`, `TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen` |
| FR-4 | `Town` + `FrontEnd.Carried`; the walk's footer (`3 worn` survives the round trip); `TestALostMissionReturnsToTheTownOnceItIsOpen` asserts the carry does not move on a loss |
| FR-5 | `TestEveryRoomSaysWhatItIsAndCanBeLeft` — header, rows, footer and a way out, for all four doors |
| FR-6 | one `Town.available` map, three callers of `Town.Take`; `TestThreeBuildingsWriteOneListAndAWinLeavesIt` |
| FR-7 | `shelfRows` + `takeShelf`; `TestTheShopStatesItsPricesAndGivesTheHeadOnlyOnce` |
| FR-8 | `tavernRows`, `talkRows`, `talk`; `TestTheTavernIsPerNPCAndTheZeroEntryIsNotConsumed`, `TestTheTavernWalkPutsAMissionAtTheGates` |
| FR-8a | `townScreen.Choose`'s `roomGates` arm through `MissionOpener`; `TestWalkingOutOfTheGatesEntersAMapScreen` |
| FR-9 | `Town.Won` reads `Chapter.Payment`; `TestWinningPaysOnceAndOnlyWhatTheSectionDeclares`; the `gold 1000` above |
| FR-10 | no marshaller, no path, no format anywhere in `town.go`; its header names the whole state |
| FR-11 | `drawTown` through the picker's own constants; `TestTheTownIsHitTestedByTheMapListsOwnModel` |
| AC-1…AC-6 | AC-1 `TestAWinAtTheTownsBoundaryOpensTheTown`; AC-2 `TestTheTavernWalkPutsAMissionAtTheGates`; AC-3 the walk's `3 worn` and 0131's `carry_test.go`, unchanged; AC-4 `TestALostMissionReturnsToTheTownOnceItIsOpen`; AC-5 `TestEveryRoomSaysWhatItIsAndCanBeLeft`; AC-6 `TestTheShopStatesItsPricesAndGivesTheHeadOnlyOnce` |
| P-1 | `git diff master..HEAD -- pkg/sim pkg/mapload` is empty; `CarryParty` and `MissionOpener` are called, not changed |
| P-2 | `ui.TownScreen` is five methods over `string`, `[]TownRow`, `[]string`, `int` and `MapOpener`; `TestTheTownCrossesTheSeam`; the import-graph test in `internal/archtest` is unchanged and green |
| P-3 | `TestTheTownAppendedToBothEnums` asserts every pre-0142 value's number, not just the new one |
| P-4 | `TestEscapeUnwindsARoomAndThenTheTown`, `TestEveryRoomSaysWhatItIsAndCanBeLeft` |
| DD-1 | `Town.Chapter` has no field to write; the side-mission case above |
| DD-2 | `TestAChapterKeepsTheThreeListsApartAndTheSentinelInPlace` asserts `Offered` did not move |
| DD-3 | one `ScreenTown`; `townRoom` is unexported and lives in `pkg/game` |
| DD-4 | `TestTheTownCrossesTheSeam`, and the `fakeTown` in `pkg/ui/town_test.go` — a town that knows nothing about missions still drives every arm |
| DD-5 | `TestTheTownListIsRebuiltOnEveryMutation` |
| DD-6 | `NoticeToTown`; `TestANoticeCanSendTheFrontEndToTheTown`, `TestNoticeToTownWithNoTownLandsOnTheMapList` |
| DD-7 | `TestALostMissionStillReachesTheMenuBeforeTheTownIsOpen` is the "nothing changed before it opens" half |
| DD-8 | one `Town.Take`; `TestThreeBuildingsWriteOneListAndAWinLeavesIt` |
| DD-9 | `TestTheShopStatesItsPricesAndGivesTheHeadOnlyOnce` asserts row 0 unchoosable and row 1 choosable |
| DD-10 | `Town` has no conversation field; `TestTheTavernWalkPutsAMissionAtTheGates` drives the conversation through `Choose` alone |
| DD-11 | every string is composed in `pkg/game`; `pkg/ui/town_test.go` supplies its own and `pkg/ui` composes none |
| DD-12 | `town.go`'s header; nothing in the package imports `os`, `encoding/*` or names a file |
| R-1 | `Town.Offers` pairs to the shorter array; the fixture's chapter 40 has `InnNPC` and `InnMission` both single-valued and chapter 30 both two-valued |
| R-2 | disclosed and observed: `done 1` in the walk above |

## What was checked and did not hold up

`TownNotBuiltMessage` said "the campaign leads to a town, and the town is not built". That sentence
is now false wherever the town is reached, so its **text** changed to say what actually happens on
the path that still reaches it — a win with nothing leading on from it. The **name** did not: what
it names, the destination that is not a town, is the same one, and three previous stories have been
repaired after re-spelling a fact into a name.

## Not done

- No save. `Town` and `FrontEnd.Carried` are the whole state and neither is written anywhere.
- No mercenary hire. Decoded (`MERC-*`, `research/docs/status/tavern.md`) and built as nothing; the
  seam is named in `town.go`'s header.
- No trade and no training: the shop and the school hand out missions only.
- No world map behind the gates.
- No graphics anywhere in the town.
