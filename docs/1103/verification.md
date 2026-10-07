# Exact SAV index verification

Base: `d53afa394675942d7461e772f1646efaf0fd2acc`. Reconciled before
implementation with published master `6423fd6c2b8b874e36aab266155c5b45f6ee5e54`.
Research pin remains `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.

## Player result and bounded proof

`TestOriginalExact1103SparseAppLoadSaveFreshLoad` opens complete literal SAVs
through the actual App LOAD list. One- and two-NPC cases move map IDs 91/92
from fresh cells (5,6)/(6,6) to saved cells (7,9)/(8,9). The empty terrain lists
retain latch 17 and asymmetric diplomacy 3→4. Leading nulls and a repeated
human Player produce one human object. Ordinary App SAVE writes `.ags`; a new
frontend LOAD retains the world hash and all 32 subsequent tick hashes.
Diagnostic `ResumeOriginalSave` independently reports the same 1/2 joins and
moves with session restoration. Missing NPC creation is not part of this result.

`TestExactDocument1103SparseAndEmptyIgnoreRawDecoys` covers 0/1/2 actors in
world/no-world envelopes, four Player reference slots naming one Player,
Player/class/head/world decoys in raw Diary bytes, and a real Building with a
colliding map ID. None changes the exact owner-graph index or world selector.
`TestExactDocument1103ExtendedCountEditOffsets` verifies six-byte WriteCount
payload starts and literal-byte edits, including reopen. Malformed count,
reference, trailer/suffix and unsupported CString-prefix cases refuse Open
without returning a partial File.

`TestOriginalExact1103PositionJoinIsAtomicAndClassFiltered` verifies duplicate
saved IDs and duplicate eligible map targets leave map bytes and counters
unchanged. Existing `TestOriginalPools1094RefusedAppLoadRetainsOldSession`
exercises both collisions through App LOAD and verifies the prior session and
subsequent save. A truncated archive now fails in Open and is not offered in
the LOAD list. `TestExactDocument1103CityNullAndRepeatedPlayerProvenance` covers
Open → CityProvenance → native DTO → SourceParty → rebuilt city SAV → Open.
The original city-only converter tests remain unchanged.

## Observable command result

Read-only `savtool info gameversions/saves/2026-08-14/game0013.sav` against the
current clean build stamped `6423fd6c` reports 87 scanned heads: 56 living,
31 dead, 71 carrying a map ID. The story command reports 30 exact owner-graph
actors: 30 living, 0 excluded, 29 carrying a map ID. Both report body length
57998 and terrain/session offsets `0x89bd`/`0xa8b3`/`0xce45`.
This removes unrelated heads from a shipped diagnostic and the production
position source. It does not claim 57 missing actors should be materialized.

The story's missionrun executable, from this worktree with the EN asset root,
reports UNSUPPORTED counts 0 for mission 10 and 0 for mission 20 at one tick.
Script populations match `pipeline/milestone-baseline.txt`: mission 10 has
16 checks, 27 instants, 12 triggers; mission 20 has 14/15/11. The baseline file
does not store UNSUPPORTED totals. The seat supplied prior 0/0 measurements at
master `d46ac7f2` and an EN 0/0 measurement on the 1101 branch; those are prior
evidence, not a new measurement of `6423fd6c`. No script opcode changed.

## Gates

`gofmt`, `git diff --check`, and `go test -trimpath -count=1 ./...`: PASS.
The first full run found the final incomplete legacy fixture in cmd/savtool;
after replacing it with a complete literal envelope, the full run passed.

Paired EN/RU release: PASS, 133/133 ran on each root, zero lacked a subject.
The first attempt failed only when
the 386 cutscene helper could not obtain Git VCS status. Explicit protected
`safe.directory` entries for the seat, worktree and research fixed that
instrument configuration; VCS stamping was not disabled.

`check-no-game-assets.sh`: clean (tracked tree scan).
`check-preserved-installs.sh`: PASS, 181 files, both roots as recorded.
All commands used `GOCACHE=<seat>/.cache/go-build`.
Release and scenario subprocesses required approved host execution. No test
was skipped to work around the instrumentation failures.

Headless production drives: EN and RU each passed `1013-world-map-one-click`,
`0163-mission-to-town`, and the additional `0163-chargen-mission10` selected by
the 0163 filter. No GUI window or synthetic desktop input was used.

Allocator sweeps bracketing DIV-026/DIV-634 amendments: 32 answers, zero missing
before and after. No new divergence ID was consumed. The claim-overturn
instrument reports 70 rows; changed rows use the unaffected/amended
SAV-DOC-053 top-level order and SAV-DEATH-051 owner-graph clauses, not their
retracted populations.

## Exclusions

Fine position bytes remain exact source/edit/map-transfer data. The simulation
and AGS proof cover cell positions, not subcell motion or saved orders.
General population reconstruction, absent NPCs, active casts/areas, trigger
result registers and other uninterpreted runtime state remain unsupported.
Short ANSI CStrings remain the general-reader boundary. City authoring still
requires its supported no-world grammar and marker; no world writer was added.
The published 31/31 structural corpus result is research authority, not a claim
that 31 gameplay states were restored here. No original process or install was
written. No unpublished experiment was consumed. Root owns review, merge,
master gates and build promotion.
