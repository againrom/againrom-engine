# 1210 — the actor tick while dying and while crossing

## Result

An attack cycle now follows the original's order latch. An attacker faces
and walks only between cycles. A crossing tick still cancels the cycle, live
or restored. A turn at `AttackReady` withholds the charge load and cancels
nothing. A restored dying pursuer keeps its attack order through its dying
ticks, and the SAV writer, the loader and the byte form carry that order.
Melee against a moving target no longer drops to zero blows or doubles its
rate (see "Pacing"). `TestReleaseGeneratedWorldSAV1171` passes with its
first SAV after the blow.

## Authority

| claim | what it settles | engine |
|---|---|---|
| `AI-ORDER-039` | the order switch, which holds the pursuit arms and the walk arms, runs only at progress 0 | `approach` faces and walks only at `AttackReady` |
| `AI-PURSUE-040` | the pursuit arms hold the facing-plus-reach test | `approach`: `closedOn`, then `turnToward` or `walkTo` |
| `AI-RETREAT-272`, `HERO-CADENCE-112` | progress 1 latches until the cycle completes and restores the act-state every tick | the latch: from the charge load to the next `AttackReady`, `approach` neither turns nor walks |
| `AI-FACE-066`, `AI-FACE-067` | the facing test is the act-state entry and runs every tick at progress 0; the swing and strike test reach only | a turn at `AttackReady` withholds the charge load; a blow whose victim left reach misses |
| `HERO-CROSSHOLD-146`, `MOVE-STEP-040` | a state-1 tick zeroes the attack sub-phase; progress 3 is the only stepping arm above the switch | `advanceAttack` cancels on `Transit != 0` or a restored crossing or queued centred turn |
| `HERO-DYINGTICK-145` | the dying branch clears only its own two words and never reaches the order machine | the attack loop spares a dying unit's order; the decay pass ends it when the dying window closes |
| `AI-332`, `formats/sav/actors.md` | the dying actor's order words are in the SAV | writer, loader and byte form accept a dying attack holder |

The walk is settled by the claims, not left open. The walk arms are inside
the switch (`AI-ORDER-039`, `AI-PURSUE-040`). Progress arm 1 restores the
act-state without reaching the switch (`AI-RETREAT-272`). Only progress 3
steps (`MOVE-STEP-040`). So a latched attacker does not chase a victim that
leaves reach mid-cycle; its strike misses on reach, and pursuit resumes at
`AttackReady`.

## As built

- `pkg/sim/combat.go`, `approach`: `latched := AttackPhase != AttackReady`.
  In reach, it turns only when not latched, then rests. Out of reach, a
  latched attacker returns; otherwise it walks.
- `pkg/sim/combat.go`, `advanceAttack`: the cancel is
  `e.Transit != 0 || w.motionActive(e.ID)`. At `AttackReady`, a turning
  attacker does not load its charge.
- `pkg/sim/step.go`: the attack loop clears a non-alive unit's order only
  when it is not dying. The decay pass clears it on the tick the dwell
  reaches zero, so the byte form never holds an order on a unit past its
  dying window. A live fall still clears the order in `clearFelledActions`.
  Only a restored dying pursuer is affected.
- `pkg/game/savactoractions.go`, `pkg/sim/originalactions.go`,
  `pkg/sim/binary.go`: the SAV writer writes, the loader restores, and the
  byte form accepts a dying pursuer's target, phase and countdown. The
  loader normalises a stale reference from a source past its dying window.
- Tests: `TestATurnWithholdsTheChargeLoadLiveAndRestoredAlike`,
  `TestACirclingVictimNeitherStallsNorHastensTheCycle` (fails on `8e2ac63`
  with 0, 0, 40 and 30 blows and gaps of 6 and 8 ticks; passes on the tip
  with 16, 16, 18 and 19 against a stationary 21),
  `TestADyingPursuersOrderStaysFrozenUntilTeardown` (fails without the
  `step.go` change at dying tick 1),
  `TestACrossingCancelsTheAttackPhaseLiveAndRestoredAlike`, and the two
  `pkg/game` dying-pursuer SAV tests.

## Pacing

Reviewer probes `zz_review1210_pacing_test.go` and
`zz_review1210_archer_test.go` on `gameversions/en`. Pacing windows are 400
ticks and archer windows 600; both units are healed every tick. Mission 30
unless named. Rows are swings (mean interval in ticks). Base is `9469bb6`,
head is `8e2ac63`.

| row | base | head | tip |
|---|---|---|---|
| hero, stationary enemy | 26 (15.44) | 26 (15.44) | 26 (15.44) |
| enemy, stationary hero | 25 (15.71) | 25 (15.71) | 25 (15.71) |
| hero, enemy moved one ring cell every 2 ticks | 13 (31.33) | 0 | 25 (16.29) |
| every 4 ticks | 19 (20.83) | 0 | 15 (28.14) |
| every 6 ticks | 22 (18.76) | 1 | 19 (21.83) |
| every 8 ticks | 23 (17.77) | 50 (8.02) | 20 (20.16) |
| every 12 ticks | 24 (16.87) | 33 (12.03) | 23 (17.73) |
| every 16 ticks | 24 (16.57) | 25 (16.04) | 24 (16.52) |
| every 24 ticks | 25 (15.96) | 33 (12.03) | 25 (15.92) |
| hero, enemy walking the ring | 23 (17.23) | 39 (10.13) | 21 (19.30) |
| enemy, hero moved every 4 ticks | 19 (20.78) | 0 | 14 (28.38) |
| enemy, hero moved every 8 ticks | 23 (17.77) | 1 | 20 (20.16) |
| enemy, hero moved every 16 ticks | 24 (16.65) | 25 (16.04) | 24 (16.61) |
| m40 archer, stationary hero | 18 (32.59) | 18 (32.47) | 18 (32.47) |
| m40 archer, hero walking a square | 18 (33.00) | 13 (43.08) | 18 (33.00) |
| m40 archer, hero walking a line | 17 (33.38) | 15 (35.50) | 17 (33.75) |
| m40 archer, hero walking radially | 19 (31.17) | 16 (37.67) | 19 (31.72) |

The tip has no 0-swing row, no row faster than the stationary pair, and 0
cancels on every row. Mission 20 at 2 and 4 ticks: 24 and 15 swings on the
tip, 13 and 19 on base, 0 and 0 on head.

Where a tip interval exceeds the stationary pair's, the cause is the re-face
at `AttackReady`. The latched cycle does not turn, so at `AttackReady` the
attacker turns through the arc the victim covered during the cycle, at its
rotation speed, and the turn withholds the charge (`AI-FACE-066`). At 4 ticks
per ring cell the victim covers about 180 degrees per 16-tick cycle, which
costs about 13 ticks of turning. At 2 ticks it covers about 360 degrees, so
the turn is short. At 16 ticks it costs a one-tick snap. Whether the
original re-aims a turn already in progress is Unknown; see "Open debt".

Hill orc, `2026-09-09/game0076.sav` via `zz_review1210_hillorc_test.go`:
first blow at step 113 on base, 126 on head, 124 on the tip, with 97
crossing ticks. The goblin camp's first blow is at 37 on all three.
`TestReleaseMission111DormantEnemiesWakeAfterOriginalAndNativeLoad` fails
with a 124-tick loop and passes with 125. Its budget is 140: the measured
125 plus 15 ticks of margin.

## Why the witness's fall moved

Instrumented runs of `TestReleaseGeneratedWorldSAV1171`:

| | first health loss | hero blow | fall |
|---|---|---|---|
| base `9469bb6` | 274 | 456, 20 → 10, while still crossing | 470 |
| head `8e2ac63` | 281 | 462, 20 → 13; 478 misses; 494, 9 → 1 | 507 |
| tip | 273 | 462, 20 → 12; 478, 12 → 3 | 483, by another unit |

Base swung while its hero was still crossing. Head and the tip wait for
arrival, so the first blow is 6 ticks later. The damage stream differs from
the first health loss onward, so later draws differ. On head and the tip
the hero's swings are 16 ticks apart with no cancel. The target's own
movement does not cancel the hero's cycle: the cancel tests the attacker's
own crossing.

## The witness

The scenario has no tick with a dying attack holder, because a live fall
clears the order. The dying path is covered by
`TestProjectSavedActorActionsWritesADyingPursuersFrozenOrder`,
`TestImportSavedActorActionsRestoresADyingPursuersFrozenOrder`, and the owner
corpus file `2027-09-07/game0031.sav` (entity 3) through the acceptance
corpus. The witness comment now says so.

On the tip it logs: blow 462, first SAV 473 (11 ticks skipped), fall 483,
second SAV 653 (140 skipped), CLI SAV 693, window end 733. Loss controls at
273.

Two `DIV-790` gaps decide where a checkpoint can land. Both were measured on
the tip:

- A checkpoint on the blow (462) splits at 470. The live saved-group pass
  decides member `runtime553648207` while it is crossing and retargets it.
  The restored pass skips a member whose crossing was restored
  (`walkSavedMembers`). A checkpoint at 599 splits at 600: guard member
  `runtime553648131` has arrived at a cell on its way to its post, and the
  restored world does not hold that path. So a checkpoint is skipped while a
  dispatched saved-group member walks.
- With 80-tick windows the first window ran to 813 and split at 791 on
  `runtime553648206`'s `ActionClock`. The live group is a native command
  group, which the activity refresh (`AI-ACTIVITY-324`) skips. After LOAD it
  is an original group, which runs the refresh. The restored group's activity byte
  went 1 → 2 at 487 and 2 → 0 at 775, when the hero left its coverage. The
  native group stayed at 1, so the live world walked the member home at 791
  and the restored world did not. The windows are now 40 ticks and close at
  733.

The earlier tick-967 attribution was wrong. The review measured it at
`6ebb8c2`. At 967, in the live world only, `runtime553648130` acquires
target 56, whose health is -8. That is an acquisition split. The witness
passed at `c6e424d` because the fall moved to 507 and the loop ended at 527.
With the loop run to 1100 it fails at absolute tick 919 on `ActionClock`,
the family `DIV-790` names.

## Hashed state

Scenarios: all 53 run with `cmd/againrom -headless` on `gameversions/en`,
base `9469bb6` against the tip. Exit codes are identical: 48 exit 0, and 5
exit 1 on both (`0163-mission-to-town`, `1013-world-map-one-click`,
`1087-player-defend`, `1089-player-retreat`, `1186-game-options`). Of the 35
scenarios that print `World.Hash`, four move. The other scenarios differ
only in the saves path and the timestamped save names.

| scenario | tip against base | cause |
|---|---|---|
| `0155-mission10-escort` | loss decided at 480 (base 304, head 416); outcome `lost` | the crossing cancel, and the latch on top of it |
| `0159-mission40-join` | hash at 2079; `u6` health 135 → 136; the join completes | the crossing cancel and the latch |
| `1060-campaign-070-reachable` | final hash at 48; outcome `won`; tip equals head | the crossing cancel |
| `1193-mission121-bridge-guard-leash` | diverges from tick 25. Base and head reconverge at 154; the tip's `u61 within` is met at 156 with its own hash. Every assertion passes | the latch and the crossing cancel |

`2027-09-07/game0031.sav` through `zz_review1210_dyingsav_test.go`:

| | base | head | tip |
|---|---|---|---|
| after load | `9f479e8cb58bfc57` | `74ebc0059eb703b1` | `74ebc0059eb703b1` |
| 1 tick | `135efbf37bf05ab8` | `ee103bf4ade59e8c` | `e9679dfc9e5ae7f6` |
| 5 ticks | `4e3ec414889a06eb` | `c1305c5ab2fb3b6f` | `fbde4ace2173db55` |

On the tip, entity 3 keeps target 42, phase 1 and countdown 6 after 1 and 5
ticks. The writer alone (`zz_review1210_dyingwriter_test.go`) writes
`U58 5`, `U5C 312420880`, `U6C 6` after 0 and 1 ticks. Head wrote 0, 0, 0
after 1 tick. A full export after 1 or 5 ticks is refused on every build
for actors 39 (turn) and 40 (cast), which are unrelated.

## Proof

Final gates on `28f9da5`; the commit after it changes only this section.

- `gofmt -l cmd internal pkg scripts`: empty.
- `go test -trimpath -count=1 ./...`: exit 0; 54 packages `ok`, 32 with no
  test files.
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`.
- `pipeline/check-release-tests.sh` on EN and RU: `ok (326 of 326 ran, 0
  lacked a subject)` on each root.
- `scripts/check-milestone2-acceptance.sh` on EN and RU: exit 0. On each
  root: AGS discovered 114, round-tripped 0, disclosed 13, refused 101;
  original discovered 102, round-tripped 94, refused 8, mismatched 0. These
  equal main's counts.
- `pipeline/check-div-claims.sh`: exit 0; 138 rows cite a claim with a
  retraction row, as on main.
- Headless scenarios: see "Hashed state".
- Milestone census, `missionrun -mission N -trace -ticks 1 | grep -c
  UNSUPPORTED` on `gameversions/en`: mission 10 = 0 and mission 20 = 0, on
  base `9469bb6` and on the tip. Unchanged; this story does not touch
  scripts.
- `pipeline/check-preserved-installs.sh`: `ok — 554 file(s)` before and
  after every command that read an install.

## Open debt

Research questions, written without an expected answer:

1. While an actor's turn toward its attack victim is in progress, does a
   change of the victim's 8-way direction from the actor change the turn's
   target facing or its remaining ticks before the turn ends? This decides
   the tip's intervals above the stationary pair against a victim that
   circles every 4 to 8 ticks.
2. While `ord+0x09 == 1`, which routines, if any, write `mover+0x00` or
   `actor+0x58` on a tick where the victim's 8-way direction from the
   attacker changes? The latch rests on High claims; this question would
   confirm it directly.
3. When a dying actor's attack victim itself leaves the living, does the
   dying actor's order still name it? The engine clears the order then, as
   it does for every holder.

`DIV-790` consumer gaps the witness exposed, now avoided by name in the
witness: the saved-group pass for a restored crossing member, a restored
guard member's path to its post, and the activity refresh a restored
native group runs.

Comment-length cleanup debt from `docs/1205/story.md` is untouched.
