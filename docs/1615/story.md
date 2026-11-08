# One mission message delivery

## Intent

Use one delivery pipeline for both games. Preserve message ticks, order, text,
sound, duration, discards, special-number effects, mission outcomes and SAV bytes.

## Authority

Owner direction: one builder per kind, in the second game's shape, with the
first game represented by profile data. R21 of architecture audit f08f426e
names the two mission delivery paths. Base: d45af35743c0101d0f63bb97b0f8de696d3f9b68.
This is architecture work and adds no player behaviour or release count.

## As-built behaviour

`pkg/game/secondgamenotices.go` owns delivery for both games. The existing file
now contains the common pipeline. The campaign service selected from the profile
returns a `missionMessagePolicies` row as data. The pipeline reads that value
through `missionNotices.messagePolicy()` without indexing by campaign identity.

| Policy | Chapters | Destinations |
|---|---|---|
| Input | Announcer latches | Ordered script report |
| Retain pending messages | false | true |
| Interpret specials at intake | false | true |
| Special table column | 0 | 1 |

One `missionMessageSpecials` table interprets 255 as a loss for both profiles.
250, 253 and 254 are ordinary dialogue numbers in column 0. In column 1,
250 plays the idle completion cue and posts the new-mission words for 30 s;
253 plays the idle completion cue; 254 plays the idle failure cue.

`observeScriptMessages` admits the report through `acceptMissionMessages`.
`settleNotices(messageStep)` preserves the guarded-character and world-outcome
checks, then samples the Announcer on every tick, including while busy. The
profile admits one source. One loop consumes `missionNotices.pendingMessages`.
The reduced profile discards the remainder on return; the retaining profile
keeps it in the existing SAV field.

`settleNotices(messageAcknowledge)` preserves the final-page acknowledgement:
retained delivery closes the window and drains; reduced delivery shows a pending
outcome before closing. `messageDrain` is the existing save witness's drain-only
entry, without acknowledgement or simulation work. Neither entry samples
latches or advances the world.

The campaign service no longer declares `scriptMessages`, `settleNotices` or
`acknowledgeNotice`. Their per-game implementations are removed. Mission entry,
chat victory and loss restore enter the pipeline's `recordMissionOutcome` port.
It forwards to the unchanged `latchOutcome`. The win finisher, codecs, notice
painter, SAV producer and SAV reader keep their builders.

## Proof

The first commit contains only `TestMissionMessageDeliveryRecord`, its frozen
226-line synthetic record, and the comment baseline update. Seven scenarios per
profile cover ordinary and repeated raises, busy discards, missing text,
250/253/254/255, victory, defeat and dialogue on an outcome's tick. Each checkpoint
records ticks, notice text keys, order through source reads, the pending queue,
audio selectors, posted text and lifetime, outcome and counters, scenario-state
digest, world hash and the full SHA256 of the SAV document bytes.

The SAV witness calls the existing mission document producer, current-session
projection and encoder. It also covers states where player SAVE is refused;
it does not alter save admission or claim to test the second game's native
envelope. The record remains byte-for-byte equal after the refactor. A planted
retaining policy for the reduced profile fails at line 6, tick 16: `[3 4 999]`
survives and the SAV digest changes. The planted file is removed.

`CheckMissionMessages` guards one settle, the queue, special dispatch and
campaign hooks. Its self-tests name planted second builders. Named ports admit
the unchanged residue carrier, headless observation and authoring validation;
none admits another player delivery. No new package import needs a DAG edge.
The base census has 18 matching lines in 7 files; the candidate has 5 in 2 files.

Focused ordinary proof uses `Notice|Message|Outcome|Victory|Defeat|Win|Lose|Latch|Settle|Chat|Residue`.
Guard proof selects `MissionMessages|Dag|^TestLiveTreeClean$` in archtest and
`^TestLiveTreeClean$` in storyguard. `go build ./...` passes. The final candidate
also receives the fast gate and four serial release runs: ROM1 EN/RU and ROM2
EN/RU. The release selection uses the brief's message/outcome patterns, excludes
the named editor/codec/debug/window tests, and adds exact live dialogue and
outcome witnesses found by source inspection. Exact selections and run receipts
travel with the candidate handoff.

## Open debt

- A restored non-empty ROM1 queue now opens and clears messages; the base ignored and retained it, but no project-written ROM1 SAV carried such a queue.
- `recordMissionOutcome` only forwards to `latchOutcome`; the guard does not limit calls to the wrapper.
- The guard misses literal `event == 255`, a `specialEvents` map and a renamed campaign-service delivery method, as the review overlays showed.
- `scenario.go` keeps the unchanged PlayWorld mirror: latch sampling and reserved 255, without the shared rows, report input or ROM2 specials 250/253/254.

Two existing focused tests attempted sandbox-denied operations before the
production change: `TestFindAssetRootEarlierSourceWins` and
`TestMapEditorCommandEditsMatchTheWindowRoutes`. They are excluded from subsequent
focused commands. The denial rule forbids repeating them on an archive copy.
The standard fast timing table already excludes both (0.61 s and 4.02 s).
Their unrestricted verification remains with the seat.

The synthetic record is a regression oracle for this change, not original-game
evidence. Native replay ordering while busy remains Unknown under DIV-2379.
There is no admitted second delivery path. DIV-4653 and DIV-4654 are unused.
