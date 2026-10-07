# 0143 — verification

## The gate

```
go build ./...                        clean
go vet ./...                          clean
gofmt -l $(git ls-files '*.go')       prints nothing
go test -trimpath -count=1 ./...      all packages ok
bash scripts/check-no-game-assets.sh  clean (tree scan)
bash scripts/check-doc-budget.sh      exit 0; 0143 spec 10709 / 20480, plan 8165 / 20480,
                                      plan <= 1.2 x spec
bash scripts/check-hotfix-ledger.sh   ok
bash scripts/check-sdd-audit.sh       FAIL set empty for 0143
```

A worktree has no `builds/`, so `check-sdd-audit`'s note and warning **counts** are meaningless from
here in both directions. Only the FAIL set is comparable and it is empty for this story.

## The script-gap census — unchanged, and that is the claim

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
```

| | master before (`pipeline/milestone-baseline.txt`) | this branch |
|---|---|---|
| mission 10 | 17 (1 + 2 + 13 + 1) | **17** |
| mission 20 | 13 (1 + 11 + 1) | **13** |

Unchanged, and that is what this story should do to them: it runs no script node and adds none. Its
result is the other kind — something the owner can see in `builds/current/`, below.

## What was run against a real install, and what it showed

`cmd/savecheck` is the developer tool golden rule 2 names: `go test ./...` is green with no install,
so no test in this tree can answer whether a save taken in a **shipped** mission comes back. It runs
the loop in **two processes** — `save` writes a file and exits; `load` starts from nothing, reads the
directory and restores — so nothing survives in memory across the trip.

**A mission save (FR-3, FR-5):**

```
savecheck -assets <en> -saves <tmp> -mission 10 -ticks 300 save
  savecheck: mission 10 opened at tick 0, 36 entities
  savecheck: saved save-20260812-114454.ags at tick 300, hash f822e7beb95e9384, gold 0

savecheck -assets <en> -saves <tmp> load                     # a NEW process
  savecheck: on disk save-20260812-114454.ags  "mission 10 — tick 300 — gold 0"
  savecheck: loaded ...: town=false gold=0 party=1
  savecheck: resumed at tick 300, hash f822e7beb95e9384, 36 entities, 1 commanded
```

The **world hash is identical** across the two processes, at the same tick, with the same entity
count — and `1 commanded` is FR-5's own point: the unit that had been ordered before the save is
still out of the placeholder script after it.

**A town save (FR-3, FR-4):**

```
savecheck -assets <en> -saves <tmp> -town save
  savecheck: town save ...: gold 4700, chapter 70, took shop offer 0, available []
savecheck -assets <en> -saves <tmp> load                     # a NEW process
  savecheck: on disk ...  "town — gold 4700"
  savecheck: loaded ...: town=true gold=4700 party=1
  savecheck: town came back: chapter 70, available [], done(10)=true
```

**The refusals, on real files (FR-2):**

```
# one byte of the payload flipped
savecheck: load: save is corrupt: the payload checksum does not match
# forty bytes cut off the end
savecheck: load: save is truncated: 44836 payload bytes declared, 44796 present
```

Both files still LISTED — their headers are intact and the label reads — and were refused when
chosen, with the row left choosable. That is FR-7's rule seen rather than asserted.

## What was seen on screen, and what was not

The game was built and run windowed against `gameversions/en`, `-mission 10`, and driven with
synthetic keyboard input. Screenshots were taken at each step.

**Seen:** the generation screen; the mission opening over its own notice; the mission running; and
**the mini-menu drawn over the running mission with exactly four rows in order — RETURN, SAVE, LOAD,
EXIT** (`shots2/r1-04-minimenu.png`, `shots3/r1-03-minimenu.png`).

**Not seen, and this is a gap in the evidence rather than in the code:** the SAVE row being taken and
the LOAD GAME window listing a file, both in the live window. The machine's desktop was in active use
throughout and would not yield foreground to the game; synthetic input intended for it landed in
another application twice before that was noticed, after which the driver was changed to send a key
**only** when the game window is verifiably foreground at that instant — and from then on it sent
nothing, because the window never got the foreground again. The attempt was stopped rather than
continued.

What stands in for it is `TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow`, which drives
`App.step` and `App.Draw` — the same dispatch a keypress reaches — through the whole sequence:
Escape on the map, Down onto SAVE, Enter, EXIT to the map list, Escape to the main menu, `L`, and
Enter on the one row. It is not a substitute for seeing it; it is what could be run.

One thing the live run did find that no test would have: **Escape is taken by an open notice first**
(0066 FR-8), so the mini-menu needs a second Escape when a mission opens over its opening line. That
is correct and unchanged behaviour, and it is worth knowing before anyone reports it as a defect.

## Requirements

| Id | Where it is witnessed |
|---|---|
| FR-1 | `TestATownSaveRoundTripsThroughDisk`; the envelope's fields are read back by `DecodeSave` and the label by `SaveLabel` |
| FR-2 | `TestEveryRefusalHasItsOwnSentence` — five files, five distinct sentences, `Snapshot{}` returned in every case; plus the two real-file refusals above |
| FR-3 | both shapes above, in two processes each; `TestSavingAPlainMapIsRefused`; `TestASaveNamingAMissionWithNoWorldIsRefused`; `TestLoadingPutsTheTownScreenBackAtTheSquare` |
| FR-4 | `TestTheCampaignIsNotInTheSave` — a save restored against a DIFFERENT campaign reads that install's own shop row and keeps the player's gold |
| FR-5 | the ruling table; `TestEveryMapWorldFieldIsRuled` (reflection, fails on an unruled field); `TestTheResidueRidesAndComesBack`; `TestAFogPlaneOfAnotherSizeIsRefused`; `1 commanded` above |
| FR-6 | `TestTheMiniMenuIsFourEntriesInOrder`, `TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom` (both doors), `TestTheMiniMenuWithNoStoreSaysSoAndNeverCallsNil`, `TestSaveTellsTheFarSideWhichScreenItWasTakenOn` (**all four rows driven from BOTH screens**); the screenshot |
| FR-7 | `TestTheLoadWindowListsSavesAndReportsARefusal`, `TestTheTownSaveArmIsTheTownScreen`, `TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow` |
| FR-8 | `TestTheStoreWritesListsAndReadsBack`, `TestTheStoreRefusesANameThatIsNotABareFileName`, `TestDefaultSaveDirIsBesideTheBinaryAndOverridable`, `TestAFileThatWillNotReadIsSkippedAndTheListIsStillShown` |
| FR-9 | `Snapshot` is the one thing `Restore` reads and no test of the restore path names the envelope; `cmd/savecheck` reaches `Restore` through `SaveSeams` alone |
| FR-10 | `TestTheSaveSeamsCrossOnlyStringsAndBools`; `pkg/ui`'s import set is unchanged and `internal/archtest` still passes |
| AC-1 | the two-process mission run above — same tick, same hash |
| AC-2 | the two-process town run above — gold, won set and party intact |
| AC-3 | `TestEveryRefusalHasItsOwnSentence` and the two real-file refusals |
| AC-4 | `TestTheMiniMenuIsFourEntriesInOrder` and `TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow`; the town's own EXIT in `TestEscapeUnwindsARoomAndThenTheTown` |
| AC-5 | `TestEveryMapWorldFieldIsRuled` — it counts the fields and refuses a field ruled twice or not at all |
| AC-6 | `TestDefaultSaveDirIsBesideTheBinaryAndOverridable`; no statement under `pkg/game` derives a save path from an asset root |
| D-1 | the payload decodes back to an equal `[]mapload.PartyMember` including `Weapon` and `Carry` (`TestATownSaveRoundTripsThroughDisk`) |
| D-2 | the resumed world above; `TestAStaleSimulationVersionIsRefusedByTheSimulation` shows the envelope accepting what the simulation then refuses |
| D-3 | `TestTheResidueRidesAndComesBack`, `TestAFogPlaneOfAnotherSizeIsRefused` |
| D-4 | `TestSavingAPlainMapIsRefused`, `TestASaveTakenInTheTownAfterAMissionIsATownSave` |
| D-5 | `TestTheMiniMenuReturnsToTheScreenItWasOpenedFrom` |
| D-6 | the load window's rows are hit-tested through `Picker.RowAt` in `TestTheApplicationDrivesTheMiniMenuAndTheLoadWindow` |
| D-7 | twelve test files updated; every one still asserts what it asserted, with `EXIT` where Escape used to be |
| D-8 | `SaveStore.head` reads a fixed prefix; `TestAFileThatWillNotReadIsSkippedAndTheListIsStillShown` |
| D-9 | `TestTheStoreWritesListsAndReadsBack` (the directory does not exist until the first write) |
| D-10 | every test in `pkg/game/save_test.go` writes to `t.TempDir()` and none reads an install |

## What changed that was not planned

**The save seam carries one bool.** `f.live` is set when a map opens and nothing on the game side
runs when a map screen is left, so a save taken in the town **after** a mission would have been
written as that mission's. The front-end is the only tier that knows whether a mission is still
showing, so `SaveGame` takes `onMap`. `TestASaveTakenInTheTownAfterAMissionIsATownSave` is the hole
it closes on the game side, and `TestSaveTellsTheFarSideWhichScreenItWasTakenOn` is what closes it on
the front-end's: it asserts the bool that crosses is `true` from the map and `false` from the town,
and drives all four rows from both screens while it is there. This was found by asking what `f.live`
holds on the town screen, not by a test failing.

**The town screen is put back at the square on load**, and the front end therefore remembers the one
`townScreen` it built. Without it, loading a game while standing in the tavern showed the loaded
game's tavern because the old game's room was open.

**`applyResidue` pushes only over a real world and viewer.** Every hand-built `mapWorld` in this
package's own tests has neither, and the method has to be drivable over one.

## Left deliberately

`pkg/formats/sav` and everything about the original game's file: this story creates no such package,
reads no `game####.sav` and writes none. `Snapshot` is FR-9's named seam and it ships with **one**
producer — our own envelope — which is what 0145 wires the other side of. That is the only thing in
this story left with one implementation, and it is named rather than silent.

Also not built, and named in the spec's non-goals: typed save names, overwrite, delete, autosave,
restoring a notice that was on screen, and carrying `fog.visible`, the odometer, facing or the death
clock.
