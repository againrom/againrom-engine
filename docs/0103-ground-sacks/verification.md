# Verification — 0103, the map's authored loot reaches the simulation

Run on the branch's last task commit, clean tree, worktree `wt-0103`, research submodule at the
story's pin `53f8bb7`. Measurements against the two lawful installs were taken with binaries built
from that tree and run from outside the repo; nothing converted was written into it.

## SC-1 — the repo gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')
BUILD+VET OK
GOFMT SILENT

$ go test -trimpath -count=1 ./...
ok  againrom/cmd/againrom       ok  againrom/cmd/almtool      ok  againrom/cmd/classdump
ok  againrom/cmd/dlgtool        ok  againrom/cmd/mapview      ok  againrom/cmd/missionrun
ok  againrom/cmd/regtool        ok  againrom/cmd/terraintool  ok  againrom/cmd/texttool
ok  againrom/internal/archtest  ok  againrom/internal/notices ok  againrom/internal/synth
ok  againrom/pkg/data           ok  againrom/pkg/formats/alm  ok  againrom/pkg/formats/databin
ok  againrom/pkg/formats/pal    ok  againrom/pkg/formats/reg  ok  againrom/pkg/formats/res
ok  againrom/pkg/formats/spr16  ok  againrom/pkg/formats/spr256
ok  againrom/pkg/game           ok  againrom/pkg/mapedit      ok  againrom/pkg/mapload
ok  againrom/pkg/render/camera  ok  againrom/pkg/render/frame ok  againrom/pkg/render/menu
ok  againrom/pkg/render/terrain ok  againrom/pkg/render/text
ok  againrom/pkg/sim            ok  againrom/pkg/ui           ok  againrom/pkg/vfs
(3 packages have no test files: cmd/restool, cmd/sprtool, pkg/render)

$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)

$ bash scripts/check-doc-budget.sh
... 0103-ground-sacks: plan <= 1.2 x spec  10586 <= 15502 bytes  ok
... 0103-ground-sacks: tasks <= 1.2 x plan  4525 <= 12703 bytes  ok
EXIT=0
```

`check-sdd-audit.sh`'s FAIL set for this story is empty; its note and warning counts are not
comparable from a worktree, which has no `builds/`, so only the FAIL set is reported.

## SC-2 — no test reads an install

Every fixture added by this story is bytes built in test code: the leaf's type-8 payloads are
assembled word by word in `pkg/formats/alm/loot_test.go`, the sim's worlds are constructed directly
in `pkg/sim/sackform_test.go` and `sackcheck_test.go`, and `pkg/mapload/loot_test.go` builds a whole
`.alm` document in memory. `go test` was run with no `AGAINROM_ASSETS` set and every package is
green. `TestTheTenthMissionIsDrivenToAWin` skips without an install exactly as before — see below,
because a skip is not the guard the plan asked for.

## AC-1 — the payload closes exactly on every shipped map, both roots

`almtool loot` was run over every `.alm` on each root: the standalone maps at the root, plus the 28
maps inside `SCENARIO.RES` extracted with `restool cat` into a scratch directory outside the repo.

| | en root | ru root |
|---|---|---|
| maps read | 38 | 34 |
| maps whose section carries at least one record | 30 | 28 |
| decode errors (overrun or bytes left over) | **0** | **0** |
| ground records | 137 | 133 |
| stock records | 43 | 43 |
| elements | 181 | 177 |
| total gold | 646074 | 135074 |
| item codes whose class is 0 or 15 | 0 | 0 |

The en root's 38 maps are the corpus `spec.md` counts; its 137 ground and 43 stock records are the
numbers the contract was written from, reproduced here from a run. The ru root ships four fewer
standalone maps (`Beast`, `Cross`, `Kids2`, `Tomb` are absent), which is the whole of the difference
in maps, ground records and gold; the 28 maps inside `SCENARIO.RES` are the same set on both roots.
No map on either root failed to consume its payload exactly, so FR-3 holds over the whole corpus.

## AC-2, AC-3 — the leaf

- **AC-2**, the short head: `TestLootHeadWidths` builds one record twice, at format version 988 and
  990, and asserts the same records with `Gold == 0` on the short head; `TestLootHeadOverrunsPayload`
  and `TestLootPayloadHasATrailingByte` are the one-byte-fewer and one-byte-more cases, each an error
  with no partial result (FR-3). `TestLootRecordOverrunsPayload` is the third failure shape.
- **AC-3**, the round trip: `TestLootRoundTripThroughDocument` opens a document, calls `Loot()`, writes
  it back and compares bytes (FR-4).
- FR-5 is `TestLootAbsentSection` and `TestLootEmptySection`; FR-6 is `TestLootItemClassAndIndex`,
  which pairs a class-14 code against a class-2 one so the index widening has somewhere to fail;
  FR-7 is asserted on every decoded record; FR-8 is the `almtool loot` output pasted above and below.

## AC-4, AC-5, AC-6 — the simulation

- **AC-4**, the merge: `TestTwoEntriesOnOneCellMergeIntoOneSack` (one sack, gold summed) and
  `TestThreeEntriesOnOneCellJoinInArgumentOrder` (item lists join in argument order), FR-10.
  `TestThePurseWrapsAtThirtyTwoBits` is the wrapping addition.
- **AC-5**, order-independence: `TestWorldsWithSacksInAnyOrderAreOneWorld` builds one world from a
  scrambled list and one from the ascending list and asserts both the contents and the digest agree,
  then that two worlds differing only in one sack's gold do not. `TestSacksAreOrderedAscendingByYThenX`
  is FR-11's order and `TestAnOutOfBoundsSackIsRefusedNotClampedNotDropped` its refusal.
- **AC-6**, the byte form: `TestAWorldWithSacksMarshalsAndUnmarshalsToAnEqualWorld` is the round trip
  and `TestUnmarshalRefusesTheFourSackShapesFR15` gives each of FR-15's four refusals its own case
  returning an error and no world. FR-14's layout is pinned by `TestAnItemCodeCrossesTheFormUnaltered`
  and by the re-pinned digests below. FR-12 is `TestTheFourOlderConstructorsBuildWorldsWithNoSacks`
  and FR-13 is `TestSacksHandsBackACopy`.

The byte form's version moved 21 -> 22 and the version byte is at offset 0, so every literal digest
in the tree moved whether or not a world holds a sack. Every one was re-derived from a run, not
computed by hand; the old -> new list is in commit `b2df18d`'s body. Fourteen test files carry a
moved pin, across `pkg/sim` and `pkg/mapload`.

## AC-7 — the sack query

`TestTheSackQueryEvaluatesTheNamedCell` reads 1 over a cell with a sack and 0 over one without, and
asserts nothing of the sack reaches the register (FR-16). `TestASackQueryOutsideBoundsDoesNotStopThePass`
is FR-18. `TestATriggerConditionedOnTheSackQueryFiresExactlyWhenOneIsThere` drives a compiled trigger
over both worlds. `TestTheSackQueryIsNoLongerReportedUnsupported` is FR-19, and it holds because the
report and the dispatch read one table — no second list was written.

Over the corpus: `almtool script` on all 38 maps of the en root names check arm 14 **nowhere**. The
before picture was measured rather than assumed — the one table entry was removed, a throwaway
`almtool` built from that tree, and the same sweep run: **11 maps, 28 nodes**, which is the reach
`spec.md` states, reproduced from a run. The maps are `SCENARIO.RES` 10, 20, 40, 60, 70, 71, 80, 81,
90, 131 and 151.

**Witnessed by reverting, not by reading the assertion:**

| Reverted | What went red |
|---|---|
| the arm moved out of the first switch of `runCheck` into the second (FR-16) | `TestTheSackQueryEvaluatesTheNamedCell`, `TestASackQueryOutsideBoundsDoesNotStopThePass`, `TestATriggerConditionedOnTheSackQueryFiresExactlyWhenOneIsThere` |
| `&0xff` dropped from both check arguments (FR-17) | `TestTheSackQueryEvaluatesTheNamedCell` |
| the `Ground()` filter removed from `sacksFrom` (FR-21) | `TestFromALMPlacesExactlyTheGroundRecordsOfAMapWithBothKinds` |
| the ascending sort removed from the constructor (FR-11) | `TestSacksAreOrderedAscendingByYThenX`, `TestSackAtFindsExactlyTheOccupiedCells`, `TestWorldsWithSacksInAnyOrderAreOneWorld` |

The first is the one the plan warned about: an arm in the second switch is reached only after unit
resolution, and a check-14 node names no unit, so it becomes correct code that never runs. Moving it
there does not fail to compile and does not fail loudly — three tests are what catch it.

## AC-8 — the placed set beside the section dump

`SCENARIO.RES:120.alm` on the en root, 144x144, format version 990. It is the map `spec.md` names —
three ground records against twenty stock ones — so it discriminates FR-20 from FR-21 on real bytes.

```
type8: present=true  #records (meta +0x2c)=23  body=510 bytes
  [0]  stock  cell=(71,112)  0 elements
  [1]  ground cell=(22,66)   code=0x6173 class=1 index=19
  [2]  ground cell=(132,78)  code=0x7976 class=9 index=22
  [3]  stock  cell=(125,108) code=0x7461 class=4 index=1
  [4..19] stock, 0 elements each
  [20] stock  cell=(75,107)  code=0x0e13 class=14 index=19
  [21] stock  cell=(74,115)  0 elements
  [22] ground cell=(31,109)  code=0x7461 class=4 index=1
census: 23 record(s) (3 ground, 20 stock), 5 element(s), total gold=0, payload closed exactly
```

Loaded through `mapload.FromALM`, the world carries:

```
  sack[0] cell=(22,66)   gold=0 items=[24947]   (0x6173)
  sack[1] cell=(132,78)  gold=0 items=[31094]   (0x7976)
  sack[2] cell=(31,109)  gold=0 items=[29793]   (0x7461)
placed: 3 sack(s)
digest: 0xe66164a0d153471f
```

Exactly the three ground records' cells, none of the twenty stock ones, item codes carried whole and
unaltered, and the list ascending by Y (66, 78, 109) rather than in file order. The dump above was
produced by `almtool loot`; the placed set by a throwaway `cmd` built in the worktree, run, and
deleted before the tree was committed — no test reads an install.

FR-22 and FR-23 are witnessed synthetically instead, because no shipped map exercises either:
`TestARecordOutsideTheMapIsDroppedAndTheLoadSucceeds`,
`TestALootSectionThisBuildCannotDecodeYieldsNoSacksAndNoLoadFailure` and
`TestAMapWithNoLootSectionLoadsWithNoSacks`.

## What was changed in the inherited artifacts, and why

**`plan.md` and `tasks.md` failed the audit's witness rule as landed.** `check-sdd-audit.sh` reads
ids, not ranges, and reported `plan.md accounts for no: FR-1 FR-2 FR-5 ... FR-22` and
`no task carries: FR-2 FR-3 FR-4 ...`. Fifteen requirements were covered by both documents in prose
without ever being named, and `tasks.md`'s traceability table wrote its three rows as ranges, which
account for their endpoints alone. Fixed by naming the ids where each document already handled the
requirement — the `tasks.md` rows expanded to explicit lists, and one clause per affected plan
paragraph gaining its citation. **No contract clause and no plan decision was changed**, and the
edits are pure traceability; `plan.md` moved from 75% to 79% of its ceiling. `plan.md` also did not
name SC-1, so nothing would have demanded the gate be run; it now says the gate is run here.

**`plan.md`'s "What goes red" list named two files and there are four.** `pkg/sim/groupform_test.go`
and `pkg/sim/scriptowner_test.go` both use opcode 14 as a stand-in for "a check this build does not
evaluate" — neither test is about sacks; the first is about the inert-trigger rule holding beside a
live group arm, the second is a differential over instant arms needing a nonzero inert set. Both
`t.Fatal`ed on `scriptCheckSupported(14)` the moment the arm landed. The sentinel was moved 14 -> 8
(absent from the opcode table and from `scriptCheckSupported`), each test keeping its guard so it
fails the same way when opcode 8 lands too. **Neither function was renamed**:
`docs/0070-group-population/verification.md` and `docs/0071-unit-owner/verification.md` cite both by
name, and a rename would strand two landed citations. `TestTheSackArmIsStillLoudBesideTheGroupArm`
therefore keeps a name that no longer describes what it tests, and its comment now says so.

**The mission-10 guard could not be met by the gate, and the mission is already lost on master.**
`plan.md` requires `TestTheTenthMissionIsDrivenToAWin` not to move, on the ground that mission 10
carries no check-14 node. Both halves needed a run. The test **skips** unless `AGAINROM_ASSETS` is
set, so the gate never exercised it; and `scenario/10.alm` does carry one check-14 node — it is the
first of the eleven maps listed above — so the premise as written is false, though nothing turns on
it. Run with an install, the outcome is identical on every combination tried:

| | en root | ru root |
|---|---|---|
| master `dfb5519` | `outcome lost at tick 272` | `outcome lost at tick 272` |
| this branch | `outcome lost at tick 272` | `outcome lost at tick 272` |
| this branch, the check-14 table entry removed | — | `outcome lost at tick 272` |

So the guard holds in the sense that matters: **this story changes nothing about mission 10.** The
loss itself is not new and is not this story's — 0093 recorded the same mission lost with no trigger
firing at all, attributed to a VIP check on u21, then at tick 262. What is worth saying here is that
the tick has since moved to 272, that the failure reproduces on master and on both roots, and that
the gate cannot see any of it: with no install the test skips, so **the milestone test is green in
every gate this pipeline runs and red whenever it is actually exercised**.

**FR-6's "says so rather than inventing one" is satisfied only in the weak reading.** The build
returns the class bits as written and offers no predicate a caller can ask whether a code names an
item, so a caller must compare against 0 and 15 itself. Nothing hinges on it here — no code on
either root carries such a class — but a later story that resolves an item code will want the
predicate, and it is not there.

Nothing else in `spec.md` was found false. The deletion set against master is empty.
