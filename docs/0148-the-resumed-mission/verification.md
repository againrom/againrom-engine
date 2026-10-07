# 0148 — verification

Branch `0148-the-resumed-mission`, base `d10e1cc`. Five commits, `T1` to `T5`.

Every number below comes from a command run in this worktree. Where a figure comes from a command
that is not committed, the command is written out beside it.

## Instruments

| Instrument | What it is |
|---|---|
| `go test -trimpath -count=1 ./...` | 2915 synthetic tests, no game install read |
| `cmd/missionrun -sav <file>` | the resume path, headless, over a lawful install |
| `cmd/savtool party <file>` | the save's own actor list, in file order |
| `pipeline/check-milestone.sh` | the script-gap census and the drive, both installed roots |

Asset root for every run below: `<seat>/gameversions/ru`, except the `UNSUPPORTED`
counts, which use the `en` root as `pipeline/milestone-baseline.txt` does. `ROM.EXE` is the same
bytes on both roots; the runs here read map and registry data, so the `ru` runs are data facts about
the `ru` root and the milestone script covers both.

## The corpus

18 distinct save files, 23 listings, which is `SAV-OWNER-048`'s own population as it stands at this
landing. Files under `gameversions/{ru,en}/`, `gameversions/saves/2026-08-02/` and
`gameversions/saves/2026-08-12/`. Nothing was written into any of them.

`saves/2026-08-02/game0010.sav` is refused by `missionrun` on both binaries with `this save was
taken BETWEEN missions: mission number 0`. That is an existing guarded arm and not a result of this
story.

## FR-1 — the participant's own character leads the restored party

**Measured.** `savtool party` over all 23 listings. Exactly one character carries `RuntimeID == 1`
in all 23. That character is at list position 0 in 15 of them and elsewhere in 8:

| Listing | Position of the id-1 character |
|---|---|
| `ru/game0002`, `en/game0002`, `saves/2026-08-02/game000{2,3,4}` | 1 (behind "Witch") |
| `saves/2026-08-02/game0001` | 4 |
| `saves/2026-08-02/game000{7,9}` | 3 |
| the remaining 15 | 0 |

This refutes the sentence `pkg/formats/sav/party.go` carried: *"The first is the hero"*. T1 replaces
it with what was measured. `SAV-ID-015` is High that the hero's runtime id is 1 and reads the
allocator behind it.

**Witnessed.** `TestTheParticipantsOwnCharacterLeadsTheRestoredParty`,
`TestALeaderAlreadyFirstIsNotMoved`, `TestTheOrderingRuleFallsBackToTheFilesOwnOrder` in
`pkg/game/originalparty_test.go`. The three arms are: the leader moves and the rest keep file order;
zero is a real answer and is told apart from the rule not applying; no id-1 character and two id-1
characters both keep the file's order with `LeadFrom = -1`.

**Confirmed on the install.** The resume report over the corpus prints `led by the character the
file wrote at position N`, and N matches the table above on every file.

## FR-2 — a withdrawn record's unit id resolves to the restored character

**The defect, reproduced on master** (`/tmp/mr0148_before.exe`, built from `d10e1cc`):

```
missionrun -sav saves/2026-08-02/game0000.sav -trace -census -ticks 400
  tick 6  check 6 dist(NO UNIT to 111,132) WROTE NOTHING into r6
  tick 6  trigger 6 FIRED (map latch 7, once)
            pair 0  r6[...] <= r10[check 10 const(3)]  ->  0 <= 3  = true
            slot 3  instant 10 WIN
  outcome won at tick 16
```

The control, the same mission with no save: `check 6` resolves, `trigger 6` never fires, `outcome
undecided at tick 400`.

**After.** No `WROTE NOTHING` line appears on any of the 18 files. Every unit reference in every
resumed mission resolves. The full before/after over the corpus, at `-trace -census -ticks 400`:

| File | Mission | Before | After |
|---|---|---|---|
| `ru/game0000`, `ru/game0001`, `ru/game9999` | 10 | undecided@400 | undecided@400 |
| `ru/game0002`, `08-02/game000{2,3,4}` | 10 | undecided, `trigger 2` fires `instant 26 giveunit` | undecided@400, `trigger 2` does not fire |
| `08-02/game0006` | 10 | won@32 | won@32 |
| `08-02/game000{0,1,5}` | 20 | **won@16** | undecided@400 |
| `08-02/game0007` | 20 | **won@16** | lost@144 |
| `08-02/game0008` | 20 | **won@16** | lost@368 |
| `08-02/game0009` | 20 | won@16 | won@16 |
| `08-02/game9999` | 20 | undecided@400 | undecided@400 |
| `08-12/game001{1,2}`, `08-12/game9999` | 10 | undecided@400 | undecided@400 |

Three results in that table need their reason stated rather than their number.

**`08-02/game0009` still wins at tick 16, and now does so correctly.** The check that fires now
reads `check 6 dist(u136 to 111,132) -> 3 <= 3`. Unit 136 is Sarindar, whom `savtool party` places
at cell (108,130); the Chebyshev distance to (111,132) is 3. The save's label is `666` and its
recorded outcome is 1, so it was taken after the mission was won and the party is standing on the
objective. This is spec L-4: the mission is re-won because its win condition is satisfied, and every
reference resolves. Before, the same trigger fired on a register nobody wrote.

**`08-02/game000{7,8}` now lose.** The trace names the cause:
`tick 134 check 8 vip(u136) COUNTED A LOSS: its unit is not alive (-1 hp)`. Sarindar is mission 20's
escorted character and he was killed in combat. On master that check read `vip(NO UNIT)` and wrote
nothing, so the mission's loss condition could not fire at all for a resumed party. An unattended
drive losing an escort is by design, the same reading `pipeline/check-milestone.sh` records for its
own drive.

**`08-02/game0006` won at tick 32 on both binaries.** Its report says `withdrew 0` and every unit
reference resolved on master too. Its label is `finished` with recorded outcome 1, and the winning
check is `dist(entity 35 to 66,16) -> 1 <= 3`. This is not a case FR-2 reaches, and it was not
changed.

**Witnessed.** `TestARestoredMembersUnitIdBindsToHisOwnEntity` and
`TestAMembersBindingWinsOverASurvivingRecord` in `pkg/mapload/script_test.go` cover the table.
`TestAWithdrawnUnitsProximityCheckMeasuresTheRestoredCharacter` is the end-to-end pair over a
started, stepped world: the same map, script and party, differing only in whether `ScriptUnits` is
given the party. Bound, the outcome is undecided; unbound, which is what 0147 built, it is won.

## FR-3 — a `Saved` does not cross a mission boundary

**Established by reading, then by test.** `pkg/mapload/carry.go:91` `CarryParty` does `copy(out,
party)` and `Saved` is a pointer field, so it was copied verbatim into `FrontEnd.Carried`;
`NextParty()` hands that to the next mission and `pkg/mapload/start.go:404,494` place the member at
`Saved.Cell` with `Saved`'s pools. The reading was not taken on trust: reverting the one-line clear
fails both tests below, which is what makes them witnesses.

**Witnessed.** `TestASavedDoesNotCrossTheMissionBoundary` in `pkg/mapload/carry_test.go`: the
carried party holds no `Saved`, a member whose entity did not survive is cleared too, the finished
mission's own party keeps its `Saved`, and started into the next map the member stands at that map's
drop cell. `TestTheSuccessorDoesNotInheritTheSavesCell` in `pkg/game/continuity_test.go` is the same
claim through `FinishMission` and `StartMission` on the declared successor, and it also asserts the
hero opens the successor at a full fold rather than at the save's health.

**Not driven on an install.** Advancing mission 10 to mission 20 through a save requires the GUI
front end. The two tests above drive the same functions the front end calls, in the same order, and
that is what is claimed here. The owner's own report is what named the defect; this story does not
claim to have watched it disappear on screen.

## FR-4 — the report

The resume report over `saves/2026-08-02/game0000.sav` now reads:

```
        led by the character the file wrote at position 0
        4 claimed a map placement and 4 of those records were withdrawn,
        4 rebound to the script so an arm naming one measures the character
```

Witnessed by `TestARestoredPartyReportsEveryAxisItDidNotApply`, extended with both lines and with
the `LeadFrom = -1` wording, which is a different statement from `led from 0`.

## Acceptance criteria

| AC | Where |
|---|---|
| AC-1 | `TestAWithdrawnUnitsProximityCheckMeasuresTheRestoredCharacter`, both arms |
| AC-2 | the FR-2 table: three of the six mission-20 saves reach undecided@400, two lose to the escort's death, one wins correctly on a measured distance of 3 |
| AC-3 | `TestTheParticipantsOwnCharacterLeadsTheRestoredParty` and the 23-listing measurement |
| AC-4 | `TestASavedDoesNotCrossTheMissionBoundary`, `TestTheSuccessorDoesNotInheritTheSavesCell` |
| AC-5 | the census below, byte-identical |

AC-2 was first written as "all six reach undecided@400". Three do. The other three are accounted for
above and neither is the defect: one wins on a resolved distance, two lose on a resolved VIP check
that could not fire at all before. The criterion as first stated was too strong for the corpus, and
it was corrected at the landing to state what holds; the refuted wording is kept beside it in
`spec.md`.

## Design decisions

| DD | Where |
|---|---|
| DD-1 | `heroRuntimeID` and `leadFirst` in `pkg/game/originalparty.go`; the fallback arm is `TestTheOrderingRuleFallsBackToTheFilesOwnOrder` |
| DD-2 | `mapload.Saved.MapUnitID`; `ScriptUnits(m, party)` in `pkg/mapload/script.go` |
| DD-3 | `TestAMembersBindingWinsOverASurvivingRecord` |
| DD-4 | the clear is in `CarryParty`, `pkg/mapload/carry.go` |
| DD-5 | not changed. `pkg/game/resume.go` reopens the same mission number with the party it was started with, so no boundary is crossed |

The plan's four steps map to commits T1 to T4. T5 adds the front-end witness for the third
step, which the plan named as a `CarryParty` test alone.

## The script-gap census (AC-5)

`bash pipeline/check-milestone.sh`, run from `<seat>` before and after. The output
is identical, line for line, and the script's own verdict is `ok the script gap and the drive are
where they were recorded, both roots`:

```
en m10  cannot run 1 x groupcmd sub-command 11, 2 x groupcmd sub-command 15,
        13 x instant op 2, 1 x instant op 20
en m20  cannot run 1 x instant op 12, 11 x instant op 2, 1 x instant op 23
en drive outcome lost at tick 224; census: 4 of 36 unit(s) moved, 1 fell, over 224 tick(s)
ru      identical to en on all five lines
```

The brief's two counts, `-mission N -trace -ticks 1 | grep -c UNSUPPORTED` against the `en` root:

| | master `d10e1cc` | this branch |
|---|---|---|
| mission 10 | 17 | 17 |
| mission 20 | 13 | 13 |

**Both unchanged, which is the intended result.** This story moves no census number. Its result is
in the resume path, which the census does not measure: the census runs a fresh start, and a fresh
party carries no `Saved`, so `ScriptUnits` adds no entry and the compiled script is the one this
build already produced. The story's observable result is the FR-2 table above — the same binary,
the same install, the same save files, and mission 20 no longer declaring victory at tick 16.

## Gate

Run in the lane worktree at `cdc6959` and re-run at the landing on a clean tree at the branch head
`15e7df9`, which is the commit the gate result belongs to.

| Command | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | empty |
| `go test -trimpath -count=1 ./...` | all packages ok |
| `bash scripts/check-no-game-assets.sh` | PASS |
| `bash scripts/check-doc-budget.sh` | PASS |
| `bash scripts/check-hotfix-ledger.sh` | PASS |
| `bash scripts/check-sdd-audit.sh` | FAIL set empty |

The note and warning COUNT from `check-sdd-audit` is not comparable from a worktree, which has no
`builds/`. Only the FAIL set is reported here.

`git log --format='%h %(trailers:key=Co-Authored-By)' d10e1cc..HEAD` prints no trailer on any of the
five commits. The five `SDD-Task` trailers are `T1` to `T5`, one each.
`git diff --diff-filter=D --name-only d10e1cc..HEAD` is empty.

## Re-measured at the landing

Every figure above was re-run at this seat against the corpus as it stands, on binaries built from
`d10e1cc` and from the branch head `15e7df9`. Each one reproduces.

- The 23-listing measurement: exactly one character carries runtime id 1 in all 23, at list position
  0 in 15 and elsewhere in 8. No listing carries none, and none carries two.
- The before/after table over all 23 listings, at `-trace -census -ticks 400`: identical outcomes,
  and `WROTE NOTHING` appears 2 times per affected file before and 0 times after, on every file.
- `08-02/game0009`'s win: `check 6 dist(u136 to 111,132) -> 3 <= 3`, unit 136 resolved.
- `08-02/game000{7,8}`'s losses: `check 8 vip(u136) COUNTED A LOSS: its unit is not alive (-1 hp)`.
- `08-02/game0006` wins at tick 32 on both binaries, and `08-02/game0010` is refused on both.
- The gate: build, vet, `gofmt -l`, `go test -trimpath -count=1 ./...` over 37 packages,
  `check-no-game-assets`, `check-doc-budget`, `check-hotfix-ledger` all exit 0;
  `check-sdd-audit` exit 0 with an empty FAIL set. The deletion set against `d10e1cc` is empty and
  the submodule pin is unmoved at `d1e38ad`.

The GUI path was checked by reading rather than by running: `FrontEnd.missionOpener` calls
`prepare` before `StartMissionFrom`, so the withdrawal and the party's `Saved.MapUnitID` are both in
place when the script is compiled. That is the path the owner uses and the headless tool does not.

## What this story did not do

- **The explored map is not restored.** A resumed mission still shows only what the party reveals
  after the load. The plane is not decoded in `pkg/formats/sav` and a research lane is opening the
  question. Nothing here was guessed at.
- **What the file's actor-list order means is not answered.** FR-1 authors an ordering and says so.
  No claim was published and nothing was written in `research/`.
- **The restored character is still withdrawn as a map unit** (spec L-2). FR-2 fixes what a check
  MEASURES; an arm that commands the group the record belonged to still does not reach him. Nothing
  in the corpus was observed to need that.
