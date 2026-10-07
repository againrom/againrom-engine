# Autosave without a frame hitch

## Intent and authority

Result: the five-minute timed autosave and the mission-start autosave cause no
visible hitch. The frame thread takes a capture and returns; a worker exports
the SAV and writes the file.

Authority: owner direction, and the tester's report that the stutter is the
autosave. No ROM1 evidence is involved: the original has no autosave and its
timing is not the authority. Measured by the A21 frame-cost instrument on the EN
install, mission 20 (`review/a21-perf/FINDINGS.md`): `Snapshot` 0.5 ms and
`ExportCurrentSave` 71 to 97 ms inside `App.step`, for both autosaves, so the
window froze 4 to 6 frames each time. No DIV row: the autosave is an Againrom
feature and the written bytes are unchanged. DIV-2254 to DIV-2256 are returned
unused.

## As built

- `pkg/game/autosavequeue.go`: `autosaveQueue` runs jobs one at a time, in
  submission order, on a worker goroutine that exists only while jobs are
  pending. `wait` blocks until idle. A panic in a job becomes a failure result.
  Deterministic headless runs (`SetDeterministicFrames`) run the job inline, so
  a save is on disk before the next frame as before.
- `detachedExporter` builds the front end the export runs on. The export reads
  the front end only through `Archives`, `Table`, `Campaign`, the world-map
  registry, the font selector and `live.mission.state` (map address, decoded map,
  dead-body art). The detached front end carries a copy of `InstallResources`, the
  world-map registry only if already loaded (else the worker loads its own), and
  a minimal live mission holding the decoded map and a copy of the dead-body art.
  Every other input is in the `Snapshot`, which `Snapshot` builds from clones.
  No `FrontEnd` field was added.
- `autosaveMissionStart` and the timed poll capture on the frame thread
  (`Snapshot`, 0.5 ms) and submit the export and write. The timed job also
  selects its slot and sequence on the worker, because the label names the slot.
  The timed poll skips while a job is queued, sets the one-minute retry before
  the capture as before, and restarts the interval on the first frame after a
  success. A failure reaches the player on the first frame after it: the timed
  failure through the existing poll error, the mission-start failure through a
  new `TimedAutosaveControls.Notices`.
- Ordering: the SAVE seam, the SAVE dialog's prepare and commit, quick save,
  quick load, the LOAD seam and file delete call `queue.wait()` first, so no
  automatic save finishes after a save the player made later. `App.Run` and the
  headless runner call `App.FlushBackground` on exit, which waits for the queue.
  The file layer already claims names exclusively (`WriteOriginal` takes the
  lowest free `game####`, the timed slots check their sequence), so a late
  automatic save cannot replace a player's file.
- Bytes: the worker calls the same `playerMissionSave` and so the single SAV
  producer (`ExportCurrentSave`); the archtest producer check still holds.

## Proof

Byte equality (same snapshot, frame-thread export against worker export):
`TestAutosaveWorkerBytesEqualFrameThreadBytesInMission` (ticks 0, 16, 37),
`...InTown`; installed EN and RU:
`TestReleaseAutosaveWorkerWritesTheFrameThreadBytesInMission` (the production
mission-start file and the production timed file, both against a frame-thread
export of the same tick) and `TestReleaseAutosaveWorkerBytesEqualFrameThreadBytesInTown`.

Isolation, with no race detector (the host has no C compiler, so `go test
-race` is unavailable; see Open debt):
- `TestAutosaveExportIgnoresLaterFrameThreadChanges`: 300 ticks, a purse change
  and a replaced live mission while the worker holds the export; the bytes are
  those of the capture.
- `TestAutosaveExportWhileTheFrameThreadKeepsRunning`: 12 rounds of export on
  the worker against stepping and fresh captures on the frame thread; every
  output equals the frame-thread export of its capture.
- `TestCapturedMissionSharesNoWritableMemoryWithTheLiveGame`,
  `TestCapturedTownSharesNoWritableMemoryWithTheLiveGame` and the installed
  versions: a reflection walk over every pointer target, slice backing array
  and map reachable from the capture and from the detached front end, against
  everything reachable from the live front end except `InstallResources`. The
  only shared regions are the three that nothing writes after they are built
  (the mission's decoded map, the loaded world-map registry, the town's group
  membership). A loss control requires the walk to find at least 10 regions
  shared with a deliberately live value. The walk found one real alias on first
  use (the town group membership), which `town.go` documents as replaced, never
  edited.

Ordering: `TestAutosaveInFlightOrdersLaterSavesLoadAndExit` (quick save, quick
load and exit each block while a gated write is in flight and complete after),
`TestAutosaveFinishingLateKeepsEarlierPlayerFiles`,
`TestAutosaveInFlightSurvivesLeavingTheMission`,
`TestAutosaveQueueRunsJobsInOrderAndRecoversPanics`. The existing timed,
mission-start and quick-save tests run through the inline path.

Frame cost: `review/story1325-autosave/measure/autosavecost_test.go`, the A21
frame-cost method (QueryPerformanceCounter around `App.HeadlessStep`), mission
20 with a chargen party, non-deterministic frames, base `ecfe9d73` against this
branch, interleaved runs on a machine with no other Go process. One timed
autosave tick, EN (31 samples after, 28 before) and RU (5 and 12):

| step | before | after |
|---|---|---|
| timed autosave step, EN | min 102.1, median 128.3, max 157.8 ms | min 0.30, median 0.41, max 0.64 ms |
| timed autosave step, RU | min 102.0, median 131.3, max 163.3 ms | min 0.30, median 0.34, max 0.73 ms |
| `OpenMission` incl. mission-start autosave, EN | 99.8 to 116.3 ms | 10.8 to 12.1 ms |
| largest of the 5 steps after the autosave, EN | 4.6 ms | 0.12 ms |
| ordinary step p50 / p99, EN | 0.035 / 0.109 ms | 0.035 / 0.099 ms |

## Open debt

- `go test -race` could not run on this host (CGO off, no C compiler, no
  usable WSL). Race freedom rests on the static read audit of
  `ExportCurrentSave` (front-end fields reached: `Archives`, `Table`,
  `Campaign`, `live`, `worldMapCache`, font), the reflection walk and the
  stress tests above.
- The window draw (GPU) cost of the frame that follows an autosave is not
  measured; no window may be launched.
- Memory pressure: the export still allocates on the worker, and GC work it causes
  can fall on the frame thread. The after-measurement records the
  frames following an autosave.
- Timed interval restarts on the first frame after the write completes, not at
  completion.
