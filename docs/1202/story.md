# A town save the original game can actually open

The owner ran the current original-acceptance kit (`review/owner-current-sav-11c4a33/`)
in the real EN game. Two of three candidates loaded and played; the third, a
native city save produced from the campaign's first town, loaded and then
crashed the original the moment the tavern opened. This story's first pass
(`76328d1`) made the writer refuse that shape and fall back to `.ags`. The
owner then reversed the approach mid-story: the writer must repair the
missing state where the campaign's own data determines it, and refuse only
where that data does not. This is the repair.

## Cause

`review/owner-run-tavern-collections/OUTCOMES.md` measured the original's own
tavern-open truth table across eight resave candidates: it opens only when
both `Mercenaries` and `PermanentMercenaries` are restored (T4, T7); either
alone crashes (T1, T2); `Documents` never changes the outcome. `MERC-SHELF-002`
(High) has the permanent unlock array at `record+0xac` appended only by
`R1797`, draining a completed record's own `[Mission<n>] EnableMercenary`
keys — `[Mission10]` unlocks nine mercenary types, mission 20 unlocks none, and
the three side missions that also unlock one type each are not on ROM1's main
line to a first town. ROM1's first town cannot exist with an empty permanent
list, and a census of the readable original corpus found none that does.

`TestReleaseTownReturnCurrentCampaign1168`'s `fresh-mission20` subtest had
opened mission 20 directly, skipping mission 10, so `Town.mercEnabled` stayed
empty and `nativeCampaignProjectionForChapter` wrote an empty
`PermanentMercenaries` beside a populated `Mercenaries` shelf — the exact T1
shape, and the exact file staged into the kit the owner ran and crashed on.
This lane kept `76328d1`'s fix to that one fixture (it plays mission 10 to
completion before mission 20, the route ROM1 and the owner's own corpus always
take); everything below replaces `76328d1`'s writer-side refusal.

Separately, `Town.Arrive` (`frontend.go`), reached through `arriveInTown`,
only ever sets the open flag — it never itself runs `Town.Won`. A route that
reaches a settled, open town without ever completing mission 10 therefore
never has `Town.applyMercenaryUnlocks` drain mission 10's own
`EnableMercenary` keys: a `-mission 30` or `-picker` NEW GAME start,
`citySnapshot1173`'s forced `n.Town.open = true`, and a test fixture that
calls `arriveInTown()` directly all reach a settled town this way, with no
completed record ever draining `t.mercEnabled`.

CORRECTION (story1202 adversarial review finding 3,
`pipeline/reviews/story1202-adversarial.md`): this paragraph previously
claimed that a real, fully played campaign has the same gap, because "neither
[mission 10 nor mission 20] ever runs a `Town.Won` call". That is false.
`FinishMissionWithRoster` (`frontend.go`) calls `f.Town.Won(n)`
unconditionally for every completion, including mission 10's, before its
first town exists. Measured against the real route (mission 10 win, mission
20 win): the live unlock list is already `[2 3 4 6 7 9 12 13 14]` before
either mission's completion ever reaches this function, so a fully played
campaign never has an empty permanent list to repair at all. The repair's
real population is the three routes named above, not ordinary play.

## What changed

**The repair.** `impliedMercenaryUnlocks` (`pkg/game/nativecity.go`) walks
forward from `Campaign.Main[0]` along `Campaign.AutoAdvance` — the same
successor chain `againrom.exe -check` reports as "winning mission 10 opens
mission 20" — until the chain's own natural end, which on shipped EN/RU data
coincides with `Campaign.TownBegins()` (mission 30): no `AutoGetMission` key
ever names the town's own first offered mission as a target, so the walk's own
natural stop *is* the boundary, not a special case bolted onto it.
`nativeCampaignProjectionForChapter` drains every mission that walk visits
into the settled town's `PermanentMercenaries`, through the exact same
accumulation `Town.applyMercenaryUnlocks` uses on a real `Town.Won` call, for
any chapter `Campaign.Main` places at or after `TownBegins()` — not only the
one chapter that happens to equal it. That is deliberate: a town at chapter
40, 50, or later still implies the same completed auto-prologue as a town at
chapter 30 does, because nothing else can have opened it. On EN/RU data this
means mission 10's own nine `EnableMercenary` types (`[14 6 13 4 7 3 2 9 12]`)
get drained whenever they are still missing; mission 20 declares none.

The function still refuses — through the same `originalCityUnsupportedf`
boundary, with `SaveSeams`' existing D-3 rule falling back to the lossless
`.ags` — when the walk cannot place `chapter` at or after `TownBegins()` in
`Campaign.Main` at all: a chapter that is still one of the automatic-prologue
missions itself (mid-play, not yet the town), or a mission number
`Campaign.Main` does not carry (a side mission, or any other number). This is
the genuinely unreconstructable case: `citySnapshot1173` and the legacy
between-mission path (`resume.go`) can both force `Town.Open()` true for a
live session that is still mid-playing one mission, and nothing in the shipped
data distinguishes that debug/mid-mission route from one that actually
finished the auto-prologue first. No fixture in this story's own re-measured
survivor list reaches this branch (below); it is derived from the function's
own domain, not from an observed failure.

**What is and is not "implied."** Only mission 10 and mission 20 are treated
as implied, and only because `Campaign.Auto` names a single deterministic
edge from each to the next, ending at the town boundary with no further
`AutoGetMission` entry for any of missions 20 through 150 (checked against the
installed EN data). Every mission a town offers afterwards — every main
mission after 30, and every side mission (41, 71, 111, 121) — is a player or
town choice the shipped data does not name a predecessor for, so none of them
are added. This is DIV-1360's own row: the owner's direction ("Side missions
are optional and are NOT implied by anything — do not add them") and the data
agree, not just the direction.

**The re-measured survivor list.** `76328d1`'s predecessor listed roughly
twenty broken fixtures and fixed each with a `Won(10)/Won(20)` injection. This
lane reverted every one of those injections (`git checkout dc7d269 --` on ten
files) except `townreturn1168_release_test.go`'s own kept fix, then re-ran the
full `pkg/game` and `cmd/saveconvert` suites against the repair alone. Four
failures came back, none of them a residual refusal:

- `TestReleaseNativeTownSaveHiredMercenaryAscendingOrderRoundTrips` and its
  Descending twin (`nativecity_release_test.go`) — `hireForOrderTest`'s own
  fixture arrives at the town boundary via `f.arriveInTown()` (chapter 30, no
  `Won` calls at all — the same structural gap the cause section describes),
  then plays `Town.Won(30)` and `Town.Won(40)` for real, reaching chapter 50
  with `PermanentMercenaries` still empty because nothing had ever drained
  mission 10's own keys. The repair covers this directly: `chapter` is 50, at
  or after `TownBegins()`, so `impliedMercenaryUnlocks` still names missions
  10 and 20. Both pass unmodified once the repair is generalized past
  `chapter == TownBegins()`.
- `TestReleaseNativeTownSaveSecondSaveAfterReloadDescendingHireOrderFallsBackToAGS`
  — same `hireForOrderTest` fixture, same chapter-50 shape; its own first
  checkpoint save was failing before the repair ever let the test reach its
  actual subject (a descending hire order after reload forcing `.ags` for an
  unrelated reason). Fixed by the same generalization; the test's own real
  assertion — the unrelated fallback — was never wrong.
- `TestReleaseGeneratedCityConversion1166` (`saveconvert1166_release_test.go`)
  — not a residual refusal at all: the fixture's own `want` snapshot, captured
  from a live session that never called `Town.Won`, is compared against a
  FrontEnd state built by round-tripping a native SAV. Before this story that
  round trip preserved `PermanentMercenaries` exactly (empty in, empty out);
  now it repairs it, so the comparison target changed on purpose. Fixed by
  updating `want.MercenaryEnabled` to include the same implied set
  `impliedMercenaryUnlocks` adds, applied only after the checkpoint that
  compares against the pre-write live state and before the one that compares
  against the reloaded SAV — the repair itself never mutates the live
  session's own `Town.mercEnabled` (a fixed-size array copy, not the field
  itself), so only the second checkpoint's expectation needed to move.

Zero of the four are a case where refusal is still the right answer. The
`.ags`-fallback branch this story keeps (mid-mission chapter short of the
boundary, or a mission `Campaign.Main` does not carry) has no exercising
fixture at all; DIV-1361 records that as an open, evidence-free case rather
than an assumption.

**The test this story's brief asked for.**
`TestReleaseSaveDialog1173NewGameMission30RepairsPermanentMercenaryList`
(`savedialog1173_release_test.go`) runs the unmodified
`scenarios/1173-save-dialog.json` scenario's own debug NEW GAME → "Mission 30:
30.alm" route through its SAV save step (`scenario.Steps[:12]`, verified
against `Steps[10].Name == "Expedition city"` and `.Target == "SAV"`), which is
exactly `citySnapshot1173`'s detached-city path with `chapter == 30 ==
TownBegins()` — the reconstructable case, not the owner's own illustrative
"nothing won" example (which this story's own derivation shows is actually
the one chapter the campaign graph determines without ambiguity). The test
computes the expected `PermanentMercenaries` by calling
`impliedMercenaryUnlocks` itself and draining the same `EnableMercenary` keys,
decodes the written "Expedition city.sav" through `sav.Open`/`.Campaign()`,
and asserts the two lists are equal. Mutation-tested twice: once against the
prior refusal-only code (fails with the old refusal message), once against
`impliedMercenaryUnlocks` returning `nil` unconditionally (fails with this
story's own refusal message, character for character); both revert to a
passing, `gofmt`-clean tree.

`cmd/savtool`'s `campaign` verb, kept unchanged from `76328d1`
(`savtool campaign FILE...` → `<path> head_mission=%d main=%d selected=%d
mercenaries=%d permanent=%d documents=%d`), remains useful for the owner kit's
own independent check below and for general debugging, regardless of
refuse-vs-repair.

**The owner kit.** `review/stage-current-sav-owner-kit.ps1` (seat-level,
untracked by every repository) keeps its independent staging-time check
(refuse to stage a candidate whose decoded `permanent=` count is zero), with
its comment updated to describe the repair rather than a writer-side refusal
— the check itself did not need to change, since it verifies the *output*
shape regardless of which mechanism produced it. N1's own source is
regenerated (see Owner kit section) from the current, fixed `fresh-mission20`
route rather than staying pinned to story1168's own `f68f3f9` baseline, frozen
before that fix landed.

## Authority

`review/owner-run-tavern-collections/OUTCOMES.md` is the measured authority on
which mercenary lists an original tavern needs. `MERC-SHELF-002`'s fill
mechanism (`R1797` draining a completed record's own
`[Mission<n>] EnableMercenary` into `record+0xac`) is the authority the repair
reproduces; the repair's own boundary (`Campaign.AutoAdvance`, ending at
`TownBegins()`) comes from the shipped `Campaign` data itself, never from a
literal mission number. `MERC-SHELF-002` carries a partial retraction
(`claims/retracted.md:422`, EXP-0062): the retracted clause is that the
four side-mission unlocks are "observable in play" a mission early: the fill
mechanism this row relies on (`R1797`, `record+0xac`,
`[Mission10]`'s nine types, mission 20's none) is the row's own **High**-graded
clause and is unaffected. DIV-1359, DIV-1360 and DIV-1361 record what is and
is not established: no claim traces the exact instruction the original
tavern-open faults or refuses at on an empty `+0xac` array; the `TownBegins()`
boundary and the auto-prologue's own scope are read from the shipped data, not
from a traced original instruction.

## Touched surfaces

- `pkg/game/nativecity.go` — `impliedMercenaryUnlocks` and the repair-then-
  conditionally-refuse block in `nativeCampaignProjectionForChapter`,
  replacing `76328d1`'s refusal-only block.
- `pkg/game/savedialog1173_release_test.go` — reverted `76328d1`'s three
  `Won(10)/Won(20)` fixture injections back to their pre-story shape, added
  `TestReleaseSaveDialog1173NewGameMission30RepairsPermanentMercenaryList`.
- `pkg/game/saveconvert1166_release_test.go` — `TestReleaseGeneratedCityConversion1166`'s
  `want.MercenaryEnabled` now expects the repaired set after the native-SAV
  round trip.
- `pkg/game/nativecity_release_test.go`, `nativecityplayer_test.go`,
  `nativegrants_hotfix_test.go`, `cityquickspells1167_release_test.go`,
  `nativecityhuman_release_test.go`, `nativecityitems_hotfix_release_test.go`,
  `nativecitymovement_test.go`, `savedialog1173_correction_test.go`,
  `cmd/saveconvert/generated1166_test.go` — reverted to their pre-story1202
  (`dc7d269`) shape; the repair alone keeps every one of these passing with no
  fixture injection.
- `pkg/game/townreturn1168_release_test.go` — kept unchanged from `76328d1`
  (the `fresh-mission20` route fix and its two moved `docs=` expectations).
- `internal/gatedtests/testdata/population.txt` — added this story's one new
  gated test to the checked-in population list
  (`TestScanMatchesTheCheckedInPopulationList`).
- `docs/DIVERGENCES.md` — DIV-1359 rewritten for the repair; DIV-1360 and
  DIV-1361 added. DIV-1362 was reserved for this story and is returned
  unused: no fourth genuine divergence was found.
- `cmd/savtool/main.go`, `main_test.go` — unchanged from `76328d1`, kept.
- `review/stage-current-sav-owner-kit.ps1` (untracked, seat-level) — comment
  update only; N1's source path/hash (see Owner kit section).

## Proof

`go test ./pkg/game/... -v` (redirected, `grep -c "^--- FAIL"`): 0 failures,
`ok  	againrom/pkg/game	285.1s`. `go test ./cmd/saveconvert/... -run
TestReleaseGeneratedCityConversionAcrossFreshProcesses1166 -v`: PASS.
`gofmt -l` on every changed file: nothing.

`go test -trimpath -count=1 ./...` (no install required, golden rule 2): PASS,
no `FAIL` line, `internal/gatedtests`'s own checked-in-population gate
included (318 gated tests, matching `check-release-tests.sh`'s own count
below).

`scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree scan)`.

`pipeline/check-div-claims.sh` (`AGAINROM_IMPL` = this worktree): exit 0, 137
pre-existing rows cite a claim carrying a retraction row (advisory, not a
failure); DIV-1359 is one of them, checked above — the retracted clause is
not the one this row leans on.

`scripts/check-milestone2-acceptance.sh` (EN then RU, one invocation): both
`ok`, exit 0. Every corpus census number is byte-identical to the `dc7d269`
baseline this lane started from: `SAV-BYTEID-1197-CENSUS attempted=102
byte-identical=12 differing=90 refused=0`; `SAV-ROUNDTRIP-AGS-CENSUS
discovered=105 round-tripped=0 disclosed=13 refused=92 mismatched=0`;
`SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=102 round-tripped=94 disclosed=0
refused=8 mismatched=0`. Unmoved is expected: the round-trip corpus is real
original SAV files, and DIV-1359's own finding is that none of them ever
carries an empty permanent list to begin with, so the repair's own guard
(`len(PermanentMercenaries) == 0`) never fires against this corpus either way.

`pipeline/check-release-tests.sh` (EN then RU, one invocation, `AGAINROM_IMPL`
= this worktree): checkout `a5f6db0`, 9 package(s), 318 gated test(s), 2
root(s). `root .../gameversions/en: ok (318 of 318 ran, 0 lacked a subject)`;
`root .../gameversions/ru: ok (318 of 318 ran, 0 lacked a subject)`. The
gated-test count rose from the `dc7d269` baseline's 317 to 318 for this
story's own new test, checked by `internal/gatedtests`'s own
`TestScanMatchesTheCheckedInPopulationList` above.

`git log --format='%h %(trailers:key=Co-Authored-By)' a5f6db0..HEAD`:
`8087e99` and `f84c760`, each followed by no trailer text — neither commit
carries a `Co-Authored-By` or any other trailer. `f84c760` is the code
landing (`pkg/game`, the tests, `internal/gatedtests`, `docs/DIVERGENCES.md`);
`8087e99` is a docs-only follow-up that records this exact Proof and Owner
kit section, which depends on `f84c760`'s own commit SHA and so could not be
written before it existed. `git diff --stat f84c760 8087e99` touches only
`docs/1202/story.md` — the kit and tools below are built from `f84c760` and
are code-identical to what `8087e99` (this branch's pushed HEAD) ships.

**mission10/mission20 UNSUPPORTED census (this lane's own gate table).**

```
go build -o /tmp/mr1202 ./cmd/missionrun
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr1202 -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr1202 -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
```

Result: mission10 = 0, mission20 = 0 — unchanged from what `dc7d269` (master
before this lane) carries. `pipeline/milestone-baseline.txt` does not record
this exact UNSUPPORTED-node count; it records `check-milestone.sh`'s own
separate script-gap census (for m10: 16 checks, 27 instants, 12 triggers; for
m20: 14 checks, 15 instants, 11 triggers), also unmoved, since this story's
whole diff is in SAV-export/persistence and test code — nothing it touches is
on the path `cmd/missionrun`, `pkg/sim/script.go` or `pkg/sim/step.go` walks.
This story's own observable result is `builds/current/`-visible instead: a
native SAV for a settled town other than the one debug-only unreconstructable
shape now opens its original tavern rather than being refused down to `.ags`.

## Owner kit

N1's frozen source (`review/story1168/f68f3f9/en/...-fresh-mission20-baseline.ags`)
predates story1168's own mission-10-then-mission-20 completion order fix
(`f68f3f9` is an ancestor of `76328d1`); regenerated from the current, fixed
`fresh-mission20` route (chargen, real mission 10 opener and
`LiveCompleteCampaign`, real mission 20 opener and a second
`LiveCompleteCampaign`, `Snapshot`, `EncodeSave` — the same steps
`TestReleaseTownReturnCurrentCampaign1168`'s own subtest runs) into
`review/story1202/f84c760/en/fresh-mission20-repaired-baseline.ags`
(17511 bytes, SHA256
`e97090fe8a73b823e0b9ce848905fb12a1e57761f6555b378e048c2c8ff42ff9`). The
`.ags` bytes do not depend on this story's repair (`ExportNativeCitySave`'s
repair runs only during SAV conversion, never during `EncodeSave`); the
frozen `f68f3f9` file itself is untouched and stays in place, since
`nativecitymovement_test.go` and `nativecityplayer_test.go` still hash-check
it under `AGAINROM_N3_AUDIT_ROOT` (both gated off by default, unaffected by
this story).

`review/stage-current-sav-owner-kit.ps1`'s N1 case now sources that file and
hash instead of `f68f3f9`'s; its comment block above the campaign check
narrates the repair instead of "the current writer already refuses to
produce one" (diff below).

Built `saveconvert.exe`/`savtool.exe` for `f84c760` from an isolated local
clone of this worktree (`git clone --local` into a directory with no
ancestor Git repository), not from the worktree in place — see "Found
outside story1202's scope" below for why. `go version -m` on both:
`vcs.revision=f84c7607135dbf2cffb9148dd0e3951572f8c500`,
`vcs.modified=false`.

Ran `review/stage-current-sav-owner-kit.ps1 -ExpectedSHA f84c760... -TownOnly
-ToolDir <isolated-clone-build>`: `Prepared owner kit
<seat>\review\owner-current-sav-f84c760-town`, exit 0.
`savtool campaign` on the written candidates: N1
(`game9010.sav`) `head_mission=0 main=30 selected=30 mercenaries=1
permanent=9 documents=3`; R1 (`game9011.sav`)
`head_mission=0 main=40 selected=40 mercenaries=3 permanent=9 documents=3`.
Neither hit the kit's own independent `permanent=0` refusal check. W1 is
excluded by `-TownOnly`, per the pivot's part (a)/(c) scope (this story never
touched the world-save family).

`pipeline/check-preserved-installs.sh`, run after the kit: `check-preserved-
installs: ok — 554 file(s), every root as recorded` — unchanged from the
554-file baseline recorded before the kit ran, since the kit only reads
`gameversions/`, never writes into it.

## Found outside story1202's scope

Building `saveconvert.exe`/`savtool.exe` for the kit directly from this
story's worktree (`wt-story-1202-town-save-the-original-opens`, a normal
`git worktree` of `engine`, nested directly under the seat root
`<seat>`) stamped `go version -m`'s `vcs.revision` with the
*seat* repository's own HEAD (`2519f7e...`, `Land the research prose spacing
sweep...`), not the worktree's own HEAD (`f84c760`), even though `git
rev-parse HEAD` run in the same directory correctly reports `f84c760` and
`go env GOWORK` is empty. Reproduced with `-buildvcs=true` and with `go build
-C <worktree>`; both gave the same wrong SHA. Root cause: a worktree's own
`.git` is a plain file (`gitdir: .../engine/.git/worktrees/...`), and Go's
VCS-root walk-up appears to require a real directory named `.git`, skipping
the worktree's file and continuing up past it to the seat's own `.git`
directory (a real directory, one level up) — silently mis-stamping the wrong
commit rather than reporting none. Building `engine/`'s own checkout, or any
plain `git clone` placed outside the seat's tree, does not exhibit this: the
first ancestor `.git` found is a real directory either way. Worked around
here by building from an isolated `git clone --local` (a real `.git`
directory, made outside any ancestor repository) instead of the worktree in
place; used that clone only to build the two kit tools, never to develop in.
This threatens every `go version -m`-based verification run against tools
built straight from a lane's own worktree, not just this kit script — a
result outside this story's own scope to fix.

## Unknowns carried forward

- No claim traces the exact instruction the original tavern-open faults or
  refuses at on an empty `+0xac` array; the repair reproduces `MERC-SHELF-002`'s
  own fill mechanism rather than a traced fault site.
- Whether ROM1's own save format can represent a settled town whose auto-
  prologue unlock is still missing at all is not established either way; no
  readable corpus shows one, and this engine's own debug-only
  `citySnapshot1173`/mid-mission path is the only way this engine reaches an
  analogous state (DIV-1361).

## Open debt

None the repair itself creates. Every side mission (41, 71, 111, 121) stays
outside `impliedMercenaryUnlocks` on purpose (DIV-1360): the shipped
`Campaign.Auto` data does not name a deterministic predecessor for any of
them, and filling one in without that evidence is exactly what the owner's
own direction forbids. A future story that needs one of them repaired too
would need either a claim establishing that side mission's own predecessor is
deterministic after all, or an owner decision to fill one in without it.

## Correction pass

The sole adversarial pass returned this story on one finding (RETURN, finding
1) and separately recorded two non-returning findings (2 and 3); this is the
one correction pass covering all three.

**Finding 1 (RETURN) — the refusal escaped a mission-OPEN path.**
`constructGeneratedWorld1171` (`pkg/game/generatedworld1171.go`) calls
`nativeCampaignProjectionForChapter(f, cs, 20)` on every fresh mission-20
open, and that function reads `f.Town` — the currently installed town, not
the fresh candidate `NewGameOpener` builds. Winning mission 20 without ever
completing mission 10 opens the town with `Town.mercEnabled` all false
(mission 20 grants no `EnableMercenary` keys of its own), so the second NEW
GAME of mission 20 hit this story's own refusal and OpenMission failed
outright, with no `.ags` fallback anywhere on this path — reproduced by the
review through `FrontEnd.NewGameOpener`/`App.OpenMission`/
`FrontEnd.LiveCompleteCampaign`, the production doors `-mission 20` and
`-picker` both reach. Fixed by catching `*originalCityUnsupportedError`
specifically at that one call site and treating it like this constructor's
other own "cannot construct" guards: build nothing and keep the mission open
on its existing native SAVE; a non-refusal error still fails the open.
`TestReleaseGeneratedWorld1171Mission20SecondNewGameOpensAfterEmptyUnlockWin`
(`pkg/game/generatedworld1171_1202_correction_test.go`, new) drives the exact
production route (first NEW GAME of mission 20, `LiveCompleteCampaign`,
assert the town opened with an empty permanent list, second NEW GAME of
mission 20) and fails on the pre-fix code with the escaped refusal's own
message, character for character; passes after the fix; reverted and
re-verified both ways.

**Finding 2 (non-returning) — the repaired list had no literal witness.**
`TestReleaseSaveDialog1173NewGameMission30RepairsPermanentMercenaryList`
gained a second assertion pinning `PermanentMercenaries` against the literal
`[2 3 4 6 7 9 12 13 14]` (mission 10's own nine `EnableMercenary` types,
`[14 6 13 4 7 3 2 9 12]` on installed EN/RU data, sorted and deduplicated),
beside its existing oracle-based check. Mutation-tested: the review's own
`implied = append(implied, camp.Side...)` mutation inside
`impliedMercenaryUnlocks` now fails this test (`PermanentMercenaries = [1 2 3
4 5 6 7 8 9 10 12 13 14], want the literal [...]`); reverted and re-verified
passing.

**Finding 3 (non-returning) — the recorded cause was false.** DIV-1359's
Mechanism and Reason cells, this story's own Cause section, and the
production comment in `pkg/game/nativecity.go` all claimed this engine "never
runs `Town.Won` for a pre-town mission", so a real, fully played campaign has
the same gap. `FinishMissionWithRoster` calls `f.Town.Won(n)`
unconditionally for every completion, including mission 10's, before its
first town exists — a real prologue (win 10, win 20) already drains mission
10's nine types before the town settles (measured: live list `[2 3 4 6 7 9 12
13 14]` before either completion reaches the repair). All three places are
corrected to name the repair's real population instead: routes that reach a
settled, open town without ever completing mission 10 —
`arriveInTown` after a `-mission 30`/`-picker` start, `citySnapshot1173`'s
forced `n.Town.open = true`, and a test fixture that calls `arriveInTown()`
directly.

**Explicitly out of scope.** The review's Finding 4 (the repaired array is
ascending; ROM1 appends `EnableMercenary` in declaration order) touches
`mercEnabledU16s`, shared by every native write, and is queued as its own
scoped item — untouched here.

**Gate results.** `gofmt -l` on every changed file: nothing. `go test
-trimpath -count=1 ./...`: PASS, no `FAIL` line (includes
`internal/gatedtests`'s own `TestScanMatchesTheCheckedInPopulationList` at
319 gated tests). `scripts/check-no-game-assets.sh`: `check-no-game-assets:
clean (tree scan)`. Remaining gate lines are recorded in the lane's own
final report to the seat.
