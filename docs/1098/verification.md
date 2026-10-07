# Structure-health verification

## State

Reconciled with master `685d748fb994ca96008e8d526073917f382d6cc5` as
`dbb3d8ae531d79b595768f43bfd0013b1647f2db`, including stories 1095 through 1097
and the diagnostic hotfix. Research pin remains
`ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`; reconciliation changed no divergence
row. Final gates below measure that code tree. The following proof commit
changes only this document. This is a review candidate, not a landing; the
seat owns the sole fresh adversarial review and final merge release chain.

## Observable result

The same standalone `review/story1098/health-drive.go` was run against master
`2cdd37688081030a98be8fa48ed0fc2d80800bad` and this worktree, on both lawful
roots. It reads the canonical world returned by `ResumeOriginalSave`, outside
test files. Source is `gameversions/saves/2026-08-15/game0017.sav`, SHA256
`eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0`.

| Authored ID | Sim ID | Master HP/Max | Reconciled HP/Max | Unchanged shape |
|---|---|---|---|---|
| 7 | 5 | 2000/2000 | 0/2000 | col101,row46,3x3,attach511 |
| 8 | 6 | 2000/2000 | 0/2000 | col93,row43,3x3,attach511 |
| 9 | 7 | 2000/2000 | 0/2000 | col98,row41,3x3,attach511 |

All reads are at tick zero. `health-before-{en,ru}.log` records the master
control; `final-health-{en,ru}.log` repeats the changed result on reconciled
code. The built `final-missionrun.exe -sav <source> -trace -census -ticks 1`
completed on each root and printed 11 restored structure-health records, zero
skipped and zero map-only placements. `final-ruins-drive-{en,ru}.log` records
mission40 reaching tick1. No original game process or owner desktop window
was opened.

The actual App LOAD path additionally shows these three structures as ruins
and reports 0/2000 on hover. Native ordinary SAVE/LOAD retains the world hash,
ruin frames and rendered statistics-card pixels. That is witnessed by the
install-gated test below, not claimed as an observed OS window.

## Contract coverage

| Contract | Evidence |
|---|---|
| FR-1, DD-1 | `TestBuildingsExactCountedHealthProjection` walks base and all three decoded subclasses, independent runtime/authored orders, zero and signed-negative words. `TestBuildingsRejectMalformedDocumentWithoutPartialRows` rejects every truncation, count/schema/identity errors and invalid heads. Existing GroundSacks tests and full SAV package pass after sharing the exact walker. |
| FR-2, DD-2 | `TestOriginalStructuresUniqueJoinAndCountedLimits` refuses source and target collisions atomically, distinguishes the full authored dword from its low word, and counts unbound/unmatched/subclass records. `TestOriginalStructuresRejectedAppLoadKeepsLiveSession` retains the previous session on either collision. |
| FR-3, DD-2, DD-3 | Each class/column/row/width/height/blocking/attachment mismatch is tested as a counted skip without health mutation. An absent saved list does not zero or remove map placements. `TestOriginalStructureHealthBatchIsAtomicAndNative` retains both words, shape, slots, order and native hash, including zero maximum and signed-negative current health. |
| FR-4, DD-3 | `TestOriginalStructuresAppLoadAndNativeSaveKeepBothWords` runs the real menu LOAD/SAVE paths, then 32 paired continuation ticks. The EN/RU release test inspects actual installed structures and compares rendered card pixels after native reload. `TestOriginalStructureHealthControlsStrikeAndNativeContinuation` verifies saved zero-max immunity, positive-max damage, ruin stopping and 64 paired attack-continuation hashes. |

## Lawful and independent inputs

The checkpoint's read-only `corpus.go` scan covered exactly
`gameversions/saves/*/*.sav`, deduplicating SHA256 and ignoring deeper
generated-output folders. Both EN and RU loads passed for all 28 distinct
direct saves: 26 world documents, 547 Buildings, 547 restored health pairs, zero
unsupported matches. Nine unequal pairs occurred in three saves: game0017,
game0018 and game0019 under `2026-08-15`, each with authored7/8/9 at 0/2000.
This is not a claim that all other owner saves or every possible SAV topology
are supported. `corpus-{en,ru}.log` holds that census; final reconciliation
reran the game0017 witness, not this whole standalone corpus scan.

Pinned research `tools/savfull` independently places game0017 Building objects
at reference offsets46675/46754/46833, with two-byte tags and 77-byte bodies.
Literal body words at +19, +60, +62 supply authored7/8/9 and 0/2000. The source
walk is exact, with no physical gaps or overlaps; five unresolved pointer-map
references remain in this document. This does not claim complete pointer rebinding.

The first single-input research run is preserved in `reader-result`: exact1/1,
19/20 mutations. Its alignment-byte poison has no byte to poison because this
document has zero decoded pad. A separate unmodified-tool run adds
`000-world-control.sav`, SHA256
`3407c9fce2ac9ce091871a17b0c08013cf2d1e1849396c6142b1a2257aec707d`, as the first
exact document. `reader-controls-result` reports exact2/2 and20/20 mutations.
Those mutations exercise the first world control, not the original ruin save.

## Commands and final gates

Focused Buildings, GroundSacks and original-structure tests pass in
`final-focused.log`. `go test -trimpath -count=1 ./...` passes once on the
reconciled code tree in `final-full-go.log`. `gofmt -l` on changed Go files and
`git diff --check` are clean. `check-no-game-assets.sh` prints
`clean (tree scan)` in `final-no-assets.log`.

On each of the two lawful roots, run:

```
AGAINROM_ASSETS=<root> AGAINROM_SAVE_CORPUS=<seat>/gameversions/saves \
  go test -trimpath -count=1 ./pkg/game -run TestReleaseOriginalStructures1098HealthRuinAndNativeSave -v
```

`final-release-en.log` and `final-release-ru.log` pass all three cases. The original has
0/2000; two explicitly synthetic, in-memory word variants use7/31 and-1/73 for
authored7 while leaving authored8/9 at0/2000. These variants are not evidence
that the original produced those particular states. Before/after native hashes
are equal on each root and match between roots:

| Case | Reconciled world hash |
|---|---|
| Original three ruins | `8c2f25f0ed46eb38` |
| Synthetic wounded | `b6ed30a2e06dba29` |
| Synthetic signed ruin | `be26c61332b88532` |

The checked-in manifest includes this release test. Its static exact-population
guard passes in `final-manifest.log`. One paired `check-release-tests.sh en ru`
invocation selects 8 packages and 126 gated tests in its asset-free runtime
census. It exits1 on both roots solely at
`TestReleaseCutsceneNativeAppCompletionAndSkip`: its 386 helper build reports
`error obtaining VCS status: exit status 128`. `final-paired-release.log` names
no other failed test. This is an instrument failure, not a clean paired gate;
the seat owns the host-context native witness and merge gate. The direct 1098
release tests above are the scoped EN/RU result.

The candidate drive was built with `go build -buildvcs=false -trimpath` because
Go's VCS discovery selected the seat repository above this worktree. The built
mission census `-mission 10/20 -trace -ticks 1` prints zero UNSUPPORTED nodes on
each root, unchanged from `pipeline/milestone-baseline.txt`: mission10 has
16 checks/27 instants/12 triggers, mission20 has14/15/11.
`final-mission-{en,ru}-{10,20}.log` records these results. This CLI's ordinary
idle arm runs64 ticks despite that per-waypoint ceiling; the separate `-census`
ruined-save drive above runs the requested single tick. No script node
population is changed by this slice. The observable result is the saved ruin
health and native/UI continuity, not a lower script-gap count.

`check-div-claims.sh` scans283 live rows and412 claim IDs at this pin. It
reports69 existing retraction-bearing rows; DIV-666 is not among them.
`final-div-claims.log` records the full result, not a zero-retraction claim.

Allocator sweeps bracketing DIV-666:32 namespaces, missing answers0.
`check-preserved-installs.sh`:181 files, both roots as recorded (name/size guard,
not an all-file content hash). No source save or install was written.

## Remaining work

The seat owns the sole review, merge release chain and current-build promotion.
No new command or native byte-form version is introduced. DIV-666 is used;
667..673 are unused reserved IDs.

Full SAV acceptance remains open. Saved structure populations, cell references,
changed shapes, subclasses, dynamic creation/removal and world SAV writing are
outside this slice. DIV-666 records known implementation debt; DIV-547 keeps the
separate Unknown health-to-destructor and retained-obstruction policy.
