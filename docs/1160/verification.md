# Story 1160 correction verification

The changed observable result is that the ordinary App continuation rejects
projected current-order loss. Its original title/map-menu LOAD, ordinary SAVE,
source-free native LOAD and next Move remain green on EN and RU. The two exact
reviewer mutations now stop the same App scenario at Snapshot, before SAVE.
The [story](story.md#sole-correction) records the field contract and controls.

| instrument | result | private evidence under seat review/story1160-correction |
|---|---|---|
| Reviewer baseline oracle | PASS, two current orders | review-baseline.log |
| Paused candidate App with byte143 corruption | PASS; independent reviewer oracle FAIL | review-order143.log |
| Paused candidate App with dropped order path | PASS; independent reviewer oracle FAIL | review-orderpath.log |
| Corrected App with byte143 corruption | FAIL at current World comparison, two carriers | corrected-order143.log |
| Corrected App with dropped order path | FAIL at current World comparison, nonempty World path | corrected-orderpath.log |
| Permanent current-order/path controls | PASS | current-controls.log |
| Corrected installed App witness | PASS on EN and RU; six changed carriers, 20 pairs, next Move 114,128 | app-en.log, app-ru.log |

The lane built `cmd/missionrun` from this worktree and ran `-mission 10/20
-trace -ticks 1 -assets <EN root>`. The main binary used for comparison carries
exact c6183c9 and `vcs.modified=false`. The correction changes acceptance only;
the script census does not fall.

| mission | main unsupported nodes | correction unsupported nodes | script counts, checks /instants /triggers |
|---|---:|---:|---|
| 10 | 0 | 0 | 16 /27 /12 |
| 20 | 0 | 0 | 14 /15 /11 |

Counts match `pipeline/milestone-baseline.txt`. Private evidence is
`missionrun-main-build.txt` and `mission-{main,correction}-{10,20}.log`.
The correction executable was built during editing, not promoted as a release.

Final candidate Go/gofmt/assets receipts are stored outside Git with its exact
SHA and returned to the seat. The seat retains the final EN/RU release and
milestone2 chains and publication of a rebuilt `builds/current/`. No additional
review, original runtime launch, install write, mover producer, DIV-1092 fix or
world SAV writer is part of this correction.
