# Verification — the mission trigger runtime

## The gate

```
go build ./...                        clean
go vet ./...                          clean
gofmt -l $(git ls-files '*.go')       nothing
go test -count=1 -trimpath ./...      ok, 25 packages, 0 failures
sh scripts/check-no-game-assets.sh    check-no-game-assets: clean (tree scan)   EXIT=0
sh scripts/check-doc-budget.sh        EXIT=0
sh scripts/check-sdd-audit.sh         FAIL set EMPTY
```

`builds/` is untracked and absent from a worktree, so `check-sdd-audit`'s note/warning **count** is
meaningless from this seat — one warning per story is emitted for a build directory a worktree never
had. Only the FAIL set is comparable, and it is empty.

## The witnesses

| id | witness |
|---|---|
| AC-1 | `alm.TestScriptDecodesThreeCountedArrays` — the three arrays field for field, over a node whose parameters sit in slots 0, 1, 2 and 9, so a packed reading fails; `TestScriptWalkIsExact`, five refusals (a trailing byte, a node count one too high, a missing count word, a truncated trigger record, a count near the top of its width); `TestScriptAbsentAndEmptyRecords`, both arms told apart by `Present(7)`; `TestScriptLeavesTheRawBodyAlone`. And on shipped bytes: the corpus run below. |
| AC-2 | `alm.TestUnitIdentifierWords` — `+0x40` as a `u16` and `+0x42` as a `u32`, both distinguishable from their neighbours; `mapload.TestScriptUnitsPairsEachRecordsIdWithTheEntityItBecomes`, including the duplicate rule and that the ids really are the indices `FromALM` assigns. |
| AC-3 | `mapload.TestTheThreePasses` — registers in list order over constants and runtime checks alike, ids to subscripts, the drop table carried not run, two build-time actions discarded, the dropped trigger, the latch by MAP position (built trigger at compiled index 0, latch 1); `TestPlainParametersArePackedInEncounterOrder`; `TestTheThreeTargetUnitBands`, seven cases; `TestATriggerSlotNamingAnIdThatWasNeverBuiltResolvesToZero`, both arms. The check arms: `sim.TestTheCheckArmsThisBuildEvaluates`, fourteen cases including the byte truncation, the two saturating arms and the three that write no register. |
| AC-4 | `sim.TestThePassRunsOncePerFullTickOnItsOwnPhase` — forty ticks, the pass on 6, 22 and 38 and on no other, asserted tick by tick; `TestThePassPrecedesTheMovement`. |
| AC-5 | `sim.TestTheFireOnceFlagIsTheWholeDifference` — one firing against three over three passes, and the latch at the trigger's own map position rather than at its compiled index; `TestTheScriptAndItsStateCrossTheByteForm` — a cut after the one-shot fired, the program crossing record for record, the spent latch crossing, forty ticks stepped in lock-step with identical digests at every one, and the trigger not firing again. |
| AC-6 | `sim.TestTheSixComparisonCodes` — every code in both directions, three codes outside the alphabet, and two signed cases a byte-wise or unsigned reading fails; `TestThePairsAreANDedAndTheANDOfNothingIsTrue`, six cases including an unused slot beside a holding one over a register that is not zero. |
| AC-7 | `sim.TestTheOutcomeIsTwoCountersAndAReporter` — six cases: a win, a loss, both in one pass (lost), two losses in one pass (**undecided** — the reporter's own test is for exactly one), two wins likewise, and a win reached past a doubled loss; `TestTheOutcomeLatches`. **The chain:** `TestTheCampaignsFirstMissionWinChainFires` — the escortee's arrival sets the mission variable on the first pass, the hero's win fires on the SECOND, because every condition is evaluated before any trigger runs; the outcome reported nine ticks later; the one-shot not re-firing; the outcome not moving over forty further ticks. |
| AC-8 | `sim.TestAnUnimplementedInstantIsSkippedAndReported` — the report names the arm and the trigger's other arms still run; `mapload.TestTheCompiledScriptReportsTheArmsThisBuildCannotEvaluate` — the same rule through the binder, reported and behavioural. |
| AC-9 | `sim.TestAnUnimplementedCheckMakesItsReadersInert` — the shipped shape (a group-count condition against an authored zero), reported before a tick runs and, over forty ticks, not firing, not latching, outcome undecided; `TestARegisterOwnedByNoCheckIsNotPoisoned` is the negative half — an ordinary mission variable still fires. |
| P-1 | `internal/archtest`'s import check and source scan over `pkg/sim`, both green: nothing added imports outside the standard library and no float, clock or IO reaches the pass. The digest half: the last block of `TestTheScriptAndItsStateCrossTheByteForm` — two worlds alike but for their script hash differently. |
| P-2 | `sim.TestNewScriptRefusesWhatABinderCanOnlyReachByBeingWrong`, eight refusals, and its last clause: an unimplemented opcode is **accepted**. Every unknown arm is a no-op, witnessed by AC-8 and AC-9 running to completion rather than panicking. |
| P-3 | `sim.TestTheScriptSectionRefusesTheBytesNoTickCanLeave` — a latch byte outside {0,1}, an undefined outcome, a presence byte, a pair-use byte, a fire-once byte, one byte too many, one too few, and a decided outcome with no counter that could have decided it. |
| SC-1 | five tests in `pkg/formats/alm`, all pass. |
| SC-2 | `pkg/sim`'s script suite: the pass, the alphabet, the latch, the register file, the constant preset, the bounded subscript, the outcome table, the arms. 124 test/subtest results, all pass. |
| SC-3 | AC-8 and AC-9's four tests, on a hand-built script and on a compiled one. |
| SC-4 | AC-5's second test and P-3's. |
| SC-5 | AC-7's chain test. |
| SC-6 | eight tests in `pkg/mapload`, all pass, the last of them the whole road from map bytes through `alm.Open`. |
| SC-7 | the gate above. |

## Against the shipped corpus

`almtool script`, built `-trimpath` outside the repo, over the loose maps of the preserved lawful
installs. No asset was copied and nothing was written.

| map | W×H | type-7 body | decoded A/C/T | compiled | binder |
|---|---|---|---|---|---|
| en `Forester.alm` | 256×256 | 804 | 1 / 0 / 0 | 0 / 0 / 0 | drop (132,124) |
| en `Horror.alm` | 256×256 | 988 | 1 / 0 / 1 | 0 / 0 / 0 | drop (18,15); trigger 0 dropped |
| en `Islands.alm` | 256×256 | 988 | 1 / 0 / 1 | 0 / 0 / 0 | drop (15,238); trigger 0 dropped |
| en `Kids.alm` | 80×80 | 804 | 1 / 0 / 0 | 0 / 0 / 0 | drop (41,36) |
| en `LuMoir.alm` | 144×144 | 988 | 1 / 0 / 1 | 0 / 0 / 0 | drop (84,77); trigger 0 dropped |
| en `Waters.alm` | 144×144 | 988 | 1 / 0 / 1 | 0 / 0 / 0 | drop (86,121); trigger 0 dropped |
| ru `Horror.alm` | — | absent | — | — | `present=false`, no script, no error |

What that measures, and each of these could have failed:

- **The walk is exact on every one.** 804 = 796 + 4 + 4 and 988 = 796 + 4 + 4 + 184; a body the model
  did not tile would have been an error, not a short read.
- **Every map carries exactly one action node and it is the drop table**, which is what the corpus
  claim predicts for all 38 shipped maps.
- **No loose map compiles a single instant or trigger**, so none of them can be won — which is the
  shipped side of "nothing evaluates a victory condition" and would have been refuted by any
  compiled winning arm here.
- **Four of the six carry a trigger and all four are dropped whole**, by the first-pair-left-zero
  rule and by no other.
- **Zero unresolved references and zero unimplemented arms** across the set.
- The RU `Horror.alm` — the shipped file that is the EN one's first 262 876 bytes with its record
  count cut to 4 — exercises the **absent-record** arm on real bytes: no type-7 record, no script,
  no error.

The campaign maps live inside `scenario.res` and this verb reads a raw file, so the win chain is
**not** witnessed against shipped bytes. What witnesses it is AC-7's replay, built from the decoded
chain rather than from the map. That is a stated gap, not a claim.

## What is not witnessed

- **The distance metric.** No test here discriminates it and none can: it is the story's one
  unevidenced choice, and every acceptance distance is driven to zero, where all candidates agree.
  It is a one-function seam and a research request.
- **The arms not implemented.** Nothing witnesses what they would do, because they do nothing. What
  is witnessed is that each is named before it matters and that no trigger silently reads one.
- **No runnable build.** This story ships a library and a developer verb; the verb's own run over the
  shipped corpus is above, and `builds/` was not backfilled.
