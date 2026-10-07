# 0131 — the campaign advances: verification

Toolchain: Go 1.26.1, `go.mod`'s own pin. Windows 11. Both lawful roots read as
`<seat>/gameversions/{en,ru}`; no asset is compiled in and none
left the machine.

## The gate

```
go build ./...                            clean
go vet ./...                              clean
gofmt -l $(git ls-files '*.go')           empty
go test -count=1 -trimpath ./...          ok, every package
bash scripts/check-no-game-assets.sh      EXIT=0  clean (tree scan)
bash scripts/check-doc-budget.sh          EXIT=0
bash scripts/check-hotfix-ledger.sh       EXIT=0  19 commits since 9729459 examined
bash scripts/check-sdd-audit.sh           EXIT=0
```

`check-sdd-audit`'s note count is not comparable from a lane — a worktree has no
`builds/`, so only its FAIL set is. That set is **empty** for this story. Before
this file existed it held one row, `FAIL 0131-campaign-advance: every task in
tasks.md has landed and there is no verification.md`, which is what the file
closes.

Trailers, `2094e67..HEAD`: four task ids, each appearing once and none twice —
T1 `5e844e8`, T2 `5865af7`, T3 `8f31c6c`, T4 `ee21cf6`. The three untrailered
commits are the artifacts, one correction and one witness, and each names the
story in its subject. No `Co-Authored-By` and no `Claude-Session` anywhere.

## What the campaign actually says, on both roots

`regtool dump <root>/SCENARIO.RES scenario.reg`, run against each root. The
containers differ — `SCENARIO.RES` is md5 `52ac617089c1317487767783057bd14f` on
`en` and `ba67cc75d9b862bcbff6254a9a3821b7` on `ru` — and the parsed trees are
**identical**, `diff` silent, 123 lines each. Twenty-four `[Mission<n>]`
sections. `AutoGetMission` appears in exactly one of them:

```
Mission10:
  Mercenaries = 1
  AutoGetMission = 20
  AddTextDocument = [1 2 3]
  EnableMercenary = [14 6 13 4 7 3 2 9 ... (9 total)]
...
Mission20:
  Mercenaries = 1
```

So mission 10 declares 20, mission 20 declares nothing, and no other section
declares anything. That is SC-1's second half.

## AC-10, AC-11, SC-9 — the advance against both installs

`againrom -assets <root> -check -mission N`, last line of each of four runs:

```
en, 10:  againrom: winning mission 10 opens mission 20 at scenario/20.alm, 144x144, 57 entities; hero health 145/145, mana 0/0, skill xp [0 1593 0 0 0 0]
en, 20:  againrom: winning mission 20 opens nothing — mission 20 declares no successor
ru, 10:  againrom: winning mission 10 opens mission 20 at scenario/20.alm, 144x144, 57 entities; hero health 145/145, mana 0/0, skill xp [0 1593 0 0 0 0]
ru, 20:  againrom: winning mission 20 opens nothing — mission 20 declares no successor
```

Mission 20 really is opened there: its address, its size and its entity count are
read off the mission that run started, and the hero's three numbers are read back
off his own entity inside it rather than recomputed. The 1593 is blade experience
taken off mission 10's entity by the carry and delivered to mission 20's.

**What was forced, and why.** Nothing was forced, and that is the limit rather
than a boast. The run does not observe a **win**: it starts mission 10, runs the
same decision a win runs over that mission's own world and party, and opens what
comes back. The step it skips is the recognition — the wrapper reading the
world's outcome — and it skips it because nothing above `pkg/sim` can decide an
outcome, deliberately. Driving mission 10 to a win instead was attempted and
cannot be done unattended:

```
missionrun -assets <root> -mission 10 -ticks 4000
mission 10  scenario/10.alm  80x80  36 entities
outcome undecided at tick 64          (identical on both roots)
```

The recognition is witnessed instead by
`TestDismissingAWonMissionOpensItsDeclaredSuccessor`, over a world driven to
`OutcomeWon` by the script itself — `pkg/sim` latches the outcome and nothing
above it can write one, so the win there is a real win.

**Not claimed:** that a player has won mission 10 in this build and seen mission
20 open. Nobody has. What is established is that every step after the win is the
shipped path against real installs, and that the win's recognition is the shipped
path in a test.

## Where each criterion was answered

| | Evidence |
|---|---|
| AC-1, SC-1 | `TestOnlyTheSectionThatDeclaresASuccessorReportsOne` — mission 10 gives (20, true), missions 20/30/31/40/41 give (0, false); and the two dumps above |
| AC-2, SC-2 | `TestAutoAdvanceReadsFR1sFourRows` — a `-1` section and a keyless section give the same answer |
| AC-3 | `TestAutoAdvanceAgreesWithTheLaddersFirstStep` — the file's answer and the ascending order are asked separately and compared |
| AC-4, SC-3 | `TestDismissingAWonMissionOpensItsDeclaredSuccessor`; `TestAdvanceToMissionEntersTheOpenersMapThroughTheSharedEntry` |
| AC-5, SC-4 | `TestTheSuccessorsPartyIsTheOneThatFinished` — the hero's entity in the started successor carries the experience the finished entity ended on |
| AC-6 | `TestBothSentencesNameTheMissionTheyOffer`, `TestTheSentencesAtTheEndsOfTheLadder` |
| AC-7, SC-6 | `TestARefusedSuccessorNamesBothMissionsAndCarriesTheParty` — declared `0` and `-2` |
| AC-8, SC-6 | `TestAnUnopenableDeclaredSuccessorFailsAtTheOpener`; `TestAdvanceToAFailingOpenerReportsTheFailuresOwnWords` |
| AC-9, SC-7 | `TestAdvanceToMissionEntersTheOpenersMapThroughTheSharedEntry`, over RETURN, ESCAPE and the notice button |
| AC-10, AC-11, SC-9 | the four runs above, both roots |
| AC-12 | `TestEveryEndingCarriesTheParty`, its no-campaign row; `TestTheSentencesAtTheEndsOfTheLadder` |
| AC-13, SC-11 | `TestAdvanceToMissionEntersTheOpenersMapThroughTheSharedEntry` — the successor's view and its start view |
| P-1, SC-5 | `TestEveryEndingCarriesTheParty` — all five endings |
| P-2 | `TestAdvanceToAFailingOpenerReportsTheFailuresOwnWords`, `TestAdvanceWithANilOpenerReachesTheMapListWithTheSeamsSentence` |
| P-3 | as AC-9 |
| P-4 | `TestAutoAdvanceReadsFR1sFourRows` — main and side sections alike |
| P-5, SC-8 | `TestASecondDismissalOpensNoFurtherMission` |
| P-6 | as AC-13 |
| SC-10 | the gate above |

The six sentences of the contract's I/O examples are each produced by a case
that reaches it. The first is reachable outside the check mode only through
`TestDismissingAWonMissionOpensItsDeclaredSuccessor`, since the mission it names
opens instead of showing it.

## Two lines checked by reverting them, not by reading their assertions

The adversarial plan read found that the advance would have been the one route
into a map screen that never sized its camera. Both halves of the fix were
reverted in place and the suite re-run:

```
guard on the screen instead of the viewer:
  --- FAIL .../RETURN            --- FAIL .../the_button
  (ESCAPE passes either way: its own arm returns before reaching the map arm)

the layout call removed from the guard:
  advance_test.go:167: the successor's view is 1024x768, want the window's own 1600x900
  advance_test.go:170: cell (100,90) draws at (3216,2896), want the view centre (512,384)
```

Both restored, suite green. The same method applied to `Offered`'s clearing
found it **unwitnessed** — every assertion on that field started from zero, so a
clear and a no-op read alike — and
`TestAnAdvanceClearsWhatTheMapListWasLastTold` was added, which fails on the
revert.

## Limitations, stated rather than absorbed

- **No win has been observed against a lawful install.** See above.
- **The advance's use of `MissionOpener` rather than `MissionOpenerWith` is not
  test-witnessed.** The two agree in every state the program can reach and part
  company only where the carry came back empty, where the difference is one
  entity inside a world the opener does not hand back. A test asserting
  `NextParty` instead was written, found to pass under both spellings, and
  removed rather than kept as a hollow pass.
- **`LastMission` is read by nothing**, here or anywhere. It ships once and its
  consumer is undecoded.
- **No second hop is exercised on real data**, because the shipped campaign
  declares exactly one. FR-10 rests on there being no state kept between
  advances, a property of the code rather than an observation of a second hop.
- **The windowed path was not driven by a script.** The runnable build under
  `builds/0131-campaign-advance/` is where that is checked by hand, and its
  README's commands were run before they were written down.

## Conclusion

The campaign's own key decides what follows a mission; a win that declares a
successor opens it with the party that finished; a win that declares none
behaves exactly as before. Every acceptance criterion has evidence, and the two
manual ones ran against both lawful roots and agree. The one thing not shown is
a mission won by a player, and it is claimed nowhere in this file.
