# Verification

Behavior commit `f68f3f9843cf367c8258a96ff87a83f066448a41` reconciles engine
`87bda0ae0aaf92f935a2694b696a1b5397051917`. Knowledge remains
`84f328362e6c946303da3d15dddfcefbcfec4946`. The following documentation commit
does not change code or tests.

- EN/RU: 28 selected city release families pass per root, including both new
  town-return families, current quick spells, existing first/second native SAV,
  source city sales and generated conversion. Receipts:
  `review/story1168/f68f3f9-{en,ru}.log` at the seat.
- Four focused packages (`pkg/game`, `pkg/formats/sav`, `cmd/saveconvert`,
  `internal/gatedtests`) passed before reconciliation. The seat runs their
  final full chain after the sole review; it has not run in this lane.
- Three deliberate losses are detected: unconditional main announcement, the
  old nonzero-advisory refusal, and dropping the current main announcement in
  the source writer. Each overlay exits with the intended failing assertion.
  Receipts and reproducer: `review/story1168/red/` and `loss_controls.py`.
- Standalone `saveconvert.exe`, `savecheck.exe` and `missionrun.exe` were built
  from that exact clean behavior commit. Their VCS stamp is that SHA with
  `vcs.modified=false`. All outputs are under `review/story1168/f68f3f9/`.
- Four independent converter runs (two origins on two installs) take the saved
  current AGS with its positive advisory and emit bytes identical to the raw-
  checked first SAV. Input hashes remain unchanged. Separate savecheck processes
  report town30/40 with `available []`; another converter process reloads each
  SAV to AGS. Per-case `convert.log`, `load.log` and `back.log` are beside outputs.
- Exact `missionrun -mission N -trace -ticks 1` on EN/RU reports zero
  `UNSUPPORTED` for both mission10 and mission20, unchanged from the seat's
  `pipeline/milestone-baseline.txt` census. Logs are
  `review/story1168/f68f3f9-{en,ru}-m{10,20}.log`.
- Gofmt, asset exclusion and divergence checks pass. The allocator sweep before
  and after the two reserved rows reports `missing answers: 0`; DIV1163..1168
  remain unused. Preserved installs pass: 554 files, each root as recorded.

The observable change is AGS refusal to ordinary SAV after fresh mission20
return, without announcing mission30 before accepting its NPC. The exact
standalone converter writes that same state outside the test executable.
No desktop input, original process, original-runtime SAVE/LOAD acceptance,
merge, full release chain or build-bundle promotion was performed by this lane.
Imported current-actor return remains the explicit DIV-1162 refusal.
