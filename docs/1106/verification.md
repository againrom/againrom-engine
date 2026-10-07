# 1106 verification

## Contract witnesses

- FR-1 / DD-1: independent literal 54-byte records check exact `+2c..+31`
  projection, poisoned adjacent bytes, duplicate-key archive order, fresh
  short/extended count offsets after editing `File.Body`, and detached values.
  Simulation checks overwrite, insert, construction-only retention, zero clear,
  byte-coordinate limits, out-of-map keys, and all six bytes through native form.
- FR-2: a malformed last record and oversized count fail the fresh exact walk.
  A late invalid simulation record leaves the old hash intact. Replacing a
  listed synthetic SAV with a truncated final cell before actual App activation
  keeps the old Snapshot, live driver, Town, hash and LOAD error screen.
  `ResumeOriginalSave` also refuses the same malformed envelope.
- FR-3: both original LOAD APIs expose the overlay at tick0 with no pending
  cast. Saved tails on a blocked cell survive while normal terrain refusal
  remains. Next-entry tests cover ground/ghost/air, two-cell footprints, literal
  zero power, occupied attempts and repeated attempts through the existing owner.
- FR-4: ordinary App SAVE produces AGS; a fresh FrontEnd/App LOAD preserves
  canonical bytes and hash. Installed pointer entry continues identically, and
  another ordinary SAVE while a cast is pending preserves the next event/hash.
  No native form version or layout changed.
- DD-2: zero, operation26, unknown operations and nonzero trailing coordinates
  persist. No operation26 relocation consumer was added; `DIV-730` and the
  diagnostic resume report name that limit and the other unrestored fields.

Focused compile/tests pass for `pkg/formats/sav`, `pkg/sim`, `pkg/game`, selecting
`1106|CellEntry1084`. The initial candidate is published as `82846441` under
the owner's SAV publication authorization. No independent review is claimed.

## Installed observable result

Input: owner SAV `2026-08-24/game0021.sav`, SHA256
`7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c`.
The two natural M10 triggers are `(21,63)={13,1,19,61,21,63}` and
`(22,64)={13,1,23,65,22,64}`. The independent expected-state walker reads 184
literal records at body offset43033, stride54, trigger offset46 within each
record. It starts with the two literal ALM tails and overlays every saved row,
including zero rows; it does not call the production projection or importer.

The root's independent published-master witness at `d521686f` is recorded in
`review/story1106/baseline-d521686f-{en,ru}.log`. An explicitly diagnostic edit
sets body byte44159 from13 to0 and leaves the other five bytes intact. On both
roots the old LOAD incorrectly retained operation13; normal pointer entry at
`(21,63)` requested Lightning from `(19,61)` at tick5 despite the saved clear.
The lane's EN and RU release witness now preserves the clear, enters the cell
through the App pointer path, and requests no cell cast. No original-runtime
clearing event is claimed.

The same witness also exercises the unmodified source, a diagnostic source/power
change to `{13,2,17,59,231,232}`, and an out-of-map operation26 insertion with
construction-only retention. Natural and changed-source union sizes are184;
the insertion union is185. Positive pointer casts use exactly the saved
source/power, and before-entry/pending AGS SAVE/fresh LOAD preserve bytes, hash
and next events. Actor teleport to adjacent `(21,64)` is setup only; entry is
ordinary pointer input. No desktop input or GPU/on-screen witness is claimed.

The root census (`review/story1106/corpus-tails-d521686f.log`) covers58 paths,
29 unique files and26 world saves:4423 cell-record occurrences,45 nonzero tails,
zero duplicate keys. No natural operation26 or zero-operation/nonzero-rest tail
was found. Those controls are explicitly authored fixtures, not natural corpus
observations. The corpus and preserved installs were not modified.

## Final local gates

Code checkpoint: `443945c6f3d2d897feb392adce018b44216304f2`. The later checkpoint
changes only this verification record. Code, tests and runners stayed frozen
through these measurements; the research gitlink remains the assigned pin.

- `go test -trimpath -count=1 ./...`: PASS once, all packages;
  `review/story1106/go-full.log`.
- `check-release-tests.sh en ru`: the successful paired invocation reports
  8 packages,134 gated tests; EN134/134 and RU134/134, zero subjects missing;
  `review/story1106/release-paired-fixed.log`. This includes the new four-arm
  saved-trigger App witness and the existing1084 pointer-entry witness.
- The first paired invocation failed only the native386 cutscene helper's VCS
  ownership check on both roots. No code changed in response. Process-local
  protected `safe.directory` entries for seat, worktree, worktree/research,
  standalone research and implementation corrected the host-user mismatch.
  The failed invocation remains in `review/story1106/release-paired.log`.
- Go VCS discovery also selected the parent seat for a worktree build. Narrow
  build-local `GIT_DIR` resolved from the worktree and `GIT_WORK_TREE` naming
  that same worktree produced `missionrun-exact.exe` and
  `cutscenehelper-exact.exe`, both stamped `443945c6` with `vcs.modified=false`.
  No `-buildvcs` bypass was used. The native386 completion/skip release test
  passed again with those explicit Git paths; `native386-exact.log`.
- The exact mission runner, preserved EN, `-trace -ticks 1`: M10/M20
  `UNSUPPORTED=0/0`, unchanged from `pipeline/milestone-baseline.txt`.
  Script populations remain16 checks/27 instants/12 triggers and14/15/11.
  Logs: `mission10-exact.log`, `mission20-exact.log`. This story moves the
  player-visible cleared-trigger outcome above, not the script-gap census.
- Gofmt and `git diff --check`: clean. `check-no-game-assets.sh`: clean tree.
- Pre/post allocation sweeps:32 namespaces, `missing answers: 0`, floor738.
  Only DIV-730 is spent; DIV-731..737 remain unused. `check-div-claims.sh --ids`
  selects287/287 live rows,419 distinct claims,70 retraction-bearing rows and
  no malformed rows. The new row's claims are active. Edited DIV-026 retains
  the unretracted document order and corrected world-only session statements;
  no result-register lifecycle is inferred from `TRIG-SAVE-008`.
- `check-preserved-installs.sh`: PASS,181 files across both roots.
- Commit trailer audit: no `Co-Authored-By`. No original-game asset, install byte
  or generated binary enters the repository. Evidence logs and binaries remain in the
  untracked seat `review/story1106/` directory.

## Reconciled candidate

Code checkpoint: `630d9cc2c932ee5e4f2575c925a790bfbad0272e`, containing master
`038f4b0bc3523cdc6991ab0edeac147bfec632de` and its1104/1105 changes. The pin is
still `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`. The only conflict resolutions
combine the release-manifest subjects, the two explicit World restore writers,
and the LOAD report for restored books and cell tails. Both LOAD doors retain
the cell overlay before actor restore and before live-session publication.
There are no deleted files relative to the reconciled master.

- Focused Go selection across `pkg/formats/sav`, `pkg/sim`, `pkg/game`:32
  top-level tests PASS. It covers1106,1105,1084, second-physical/native form
  compatibility, the bounded malformed-form71 refusal, and the World reader
  and method-set sweeps. Five install-gated tests skip in that no-assets run.
  `review/story1106/reconciled-630d9cc2-focused.log`.
- Separate focused EN/RU runs execute those five selected installed subjects:
  5 PASS and0 skips per root. The four1106 arms retain the natural/changed
  pointer cast, the saved clear, the out-of-map insertion and ordinary AGS
  SAVE/fresh LOAD before entry and with a pending cast. The1105 saved books
  and1104 city SAVE/mission/native continuation also pass in this union.
  `review/story1106/reconciled-630d9cc2-{en,ru}.log`. These are focused
  reconciliation checks, not another full paired release chain.
- `go test -trimpath -count=1 ./internal/gatedtests`:PASS. Gofmt,
  `git diff --check`, no-game-assets and commit-trailer audit are clean.
- `missionrun-reconciled-630d9cc2.exe` carries exact revision `630d9cc2` and
  `vcs.modified=false`. Preserved EN, `-trace -ticks 1`:M10/M20 each exit0,
  `UNSUPPORTED=0/0`; script populations remain16/27/12 and14/15/11, matching
  `pipeline/milestone-baseline.txt`. The visible result remains the cleared
  trigger, not a script-gap reduction. `reconciled-630d9cc2-mission{10,20}.log`.
- Pre/post allocation sweeps report32 namespaces and `missing answers: 0`.
  No further ID is spent. `check-div-claims.sh --ids`:287/287 live rows,
  420 distinct claims,71 retraction-bearing rows, exit0. DIV-730 has no
  retracted claim. DIV-026 retains the amended document-order and world-half
  claims, not a result-register overwrite claim. Full output is retained as
  `review/story1106/reconciled-630d9cc2-div-claims.log`.
- `check-preserved-installs.sh`:PASS,181 files across both preserved roots.

The original full branch-chain evidence above remains attributed to443945c6.
The seat owns the single fresh review and final full chain on the landing
commit; this reconciliation does not claim they have run.

## Open boundary

Original runtime trigger completion after restore remains Unknown. This slice
does not provide original in-flight cast restore, terrain/occupancy identity
rebind, result registers, orders, general effects or source-free WORLD export.
The all-five-SAV-path owner goal remains broader than this LOAD correction.
