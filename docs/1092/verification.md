# Story 1092 verification

## Focused evidence

`go test -trimpath -count=1 ./pkg/ui ./pkg/game ./cmd/mapedit -run
'MapEditor1092|MapInspection1092'` passes. The shared Pointer/Key doors cover
camera, catalogue, selection, filters, failed opens and all 30 synthetic item
lines at 640x480. The last item is compared against the actual uploaded rail
pixels. Four distinct synthetic textures reach the production art draw.

`TestReleaseMapInspection1092EveryAuthoredPlacement` passes on both read-only
installs. Its independent raw ALM walk supplies every expected placement and
loot field, not the renderer or inspection adapter. Every source is byte-identical
after selecting all its records and reopening it.

| Root | Maps | Units | Structures | Ground sacks | Actor stocks | Nonzero scenery | Resolved scenery art |
|---|---:|---:|---:|---:|---:|---:|---:|
| EN | 38 | 8094 | 3141 | 137 | 43 | 71099 | 71002 |
| RU | 34 | 3991 | 1681 | 133 | 43 | 54954 | 54857 |

Every unit, structure and ground sack resolves art on both roots. Both M10 maps
contain 35/18/4 units/structures/sacks, first unit at (24,54); M20 has 56/30/4,
first unit at (50,46). Both M71 Sack#3 records contain five items; raw code,
raw04 and enchantment-link checks cover every item in every installed map.

## Observable result

The new `mapedit` command is the owner-visible result, increasing native commands
from 40 to 41 after the seat rebuild. Headless full-window renders under the
untracked seat `review/story1092/` were inspected: EN M10 sack at 1280x800,
RU M10 sack at 800x600 (`*-v3.png`), and RU M71 Sack#3 at 640x480
(`ru-m71-sack3-640-top.png`, `ru-m71-sack3-640-end.png`). The final pair shows
the first and last of five items with the same selected catalogue row. These
are CPU diagnostics, not live-window or GPU-readback evidence; shadows omitted.

The required EN `missionrun -mission N -trace -ticks 1` check reports 0
UNSUPPORTED nodes for M10 and M20. Registration remains 16 checks / 27 instants /
12 triggers and 14 / 15 / 11, matching `pipeline/milestone-baseline.txt` from
master. This editor story does not change the script-gap census.

`check-preserved-installs.sh` after the snapshot commands passes: 181 files,
both roots unchanged. No map, asset, owner save or screenshot enters the commit.

## Initial candidate gates

Reconciled implementation master b0dd0e9703f7180daf1e85c910f417f3ced2cffd
(1090 and 1091 landed); research stays ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f.
Code validation at 9a0e0f1a passed; the following commit records only these results.

- `gofmt` and `git diff --check`: clean.
- `go test -trimpath -count=1 ./...`: PASS, no asset environment.
- Focused architecture, UI, game and command tests: PASS after reconciliation.
- The installed placement/loot test above: PASS on EN and RU after reconciliation.
  The seat runs the paired release script once on the final merge; the branch
  did not repeat its just-passed broad 1090 release chain.
- `scripts/check-no-game-assets.sh`: clean tree scan.
- `check-div-claims.sh --ids`: parsed all 276 live rows, 389 claim IDs; 68
  existing retraction-bearing rows reported, none is DIV-618.
- Allocation sweep brackets: 32 ledgers plus DIV, missing answers 0, floor626.
- Native `mapedit -list`: 38 EN / 34 RU keys; `-map scenario/10.alm -check`:
  80x80, 463 records on each root. Final CPU snapshot
  `review/story1092/ru-m71-sack3-640-candidate.png` inspected. No GUI launch.
- Preserved-install check after final command witnesses: 181 files unchanged.
- No file deletions, pin rollback or Co-Authored-By trailer. The sole removed
  source line is replaced by the aligned Viewer field beside `editorView`.

The native editor Pointer/Key regressions are its headless input scenario. No
mission routing or simulation timing changed, so broad mission scenarios were
not repeated on the branch. The seat owns the fresh review and final merge gates.

## Single review correction

The sole report, `pipeline/reviews/1092-adversarial-review.md`, returned F1
(large-map Fit cropped by the shared 0.125 minimum) and F2 (fractional wheel
events discarded). This correction changes only their camera/input paths.

F1: an optional per-camera zoom floor is set by the editor from its full world
extent and current viewport. Game/mapview cameras keep 0.125..8. Independent
synthetic 128x128/256x256 checks cover opening, Fit, first zoom, zoom-out floor
and ordinary Viewer defaults. The EN/RU test now derives terrain extents from
raw type-0 dimensions and signed type-2 altitudes for every installed map and
checks every extent corner at 640x480 and 1280x800. The 256-column witnesses
fit at 0.0390625 with their right edge at x=320.

F2: the list and detail viewports retain separate fractional-row remainders.
Twenty quarter-notch events now move 15 rows; reverse travel returns to the
start. Minimum-window multi-item traversal, viewport isolation and ordinary
integer-wheel pixel coverage pass. Selection/view changes reset their own
pending fraction.

Correction proof: complete camera package; focused editor/game/mapview/command
tests; architecture check; unchanged reviewer UI overlay; and unchanged reviewer
game overlay plus the authored full-population test on both EN and RU all PASS.
`gofmt`, `git diff --check` and no-assets guard pass. Reviewer inputs were not
modified. No second review, broad full-Go/release chain, native GUI launch,
simulation change or install write was performed in the correction pass; the
seat reruns the unchanged overlay on the pushed correction and owns final gates.
