# Verification — 0005 interactive terrain map viewer

## Gates

All run from a clean tree with **no game install required**:

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test ./...` | all packages ok |
| `go test ./internal/archtest/` | ok — DAG green with `pkg/render/camera`, `pkg/ui`, `cmd/mapview` registered |
| `go test ./internal/notices/` | ok — `THIRD_PARTY_NOTICES.md` matches `go.mod` exactly |
| `scripts/check-no-game-assets.sh` | clean (tree scan) |
| `scripts/check-no-game-assets.sh --history` | clean (history scan) |

## AC coverage

| ID | Level | Evidence | |
|---|---|---|---|
| AC-1 | unit | `TestPanClampsToWorldEdges` — a 100×100-tile world in an 800×600 window: panning past the left/top clamps to `(0,0)`, past the right/bottom clamps to `(2400,2600)` = `worldPx-viewPx`, and an interior pan is untouched | ✔ |
| AC-2 | unit | `TestSmallerAxisIsCentered` — a 10×10-tile world (320 px) in an 800×600 window centers both axes at half the negative slack, and a subsequent pan does **not** move a centered axis | ✔ |
| AC-3 | unit | `TestZoomAboutCursorPreservesPoint` — for factors 2, 0.5, 1.25, 0.8 the world point under the cursor is preserved to 1e-9 and the scale stays in `[0.125, 8]`; `TestZoomClampsToLimits` — 64 successive zooms saturate exactly at `ZoomMax`/`ZoomMin` instead of running away | ✔ |
| AC-4 | unit | `TestVisibleTiles` — aligned origin covers `ceil(view/cell)` tiles; an unaligned offset includes the partially covered tiles; the far edge clips to the map and never past it; zooming in covers proportionally fewer tiles | ✔ |
| AC-5 | unit | `TestCheckLoadsHeadlessly` — `-check` over a synthetic `res` archive + synthetic `.alm` prints exactly one summary line naming the map, `4x3`, the cell count and the loaded slot count, and exits 0 with no window. `TestCheckAssetRootSources` covers `AGAINROM_ASSETS` and explicit `-graphics`; `TestLoadFailuresAreReported` covers the six failure paths (each errors before a window could open and writes nothing to stdout); `TestCheckWithEmptyArchive` and `TestUnnamedMapUsesFileName` cover the degenerate cases | ✔ |
| AC-6 | **manual** | **Pending manual verification** — see below | ⏳ |

## Derived properties

| ID | Evidence | |
|---|---|---|
| P-1 | `TestRandomizedPanZoomKeepsInvariants` — 200 random worlds × 40 random pan/zoom/set-zoom steps; after **every** step each axis is asserted either clamped into `[0, world-view]` or exactly centered, and the zoom is asserted inside its limits | ✔ |
| P-2 | The same test asserts the visible range never escapes the grid; `TestVisibleRangeStaysInsideGrid` additionally forms every index the draw loop would compute and asserts it lands inside the tile slice. `TestDegenerateInputsAreSafe` covers NaN pan/position, zero and negative zoom, NaN/zero zoom factors, and a zero-size view without panic or a non-finite state | ✔ |

## Headless run against a lawful install

`mapview -check` was run against a GOG install. The load path — `graphics.res` → tileset, `.alm` → grid —
succeeds and reads the maps' real stored names:

```
Beast.ALM     mapview: Beast  Land 256x256 cells (65536), tile slots 52/128
Cross.ALM     mapview: Crossroads of Mystery 256x256 cells (65536), tile slots 52/128
Forester.alm  mapview: Forester 256x256 cells (65536), tile slots 52/128
Horror.alm    mapview: Horror 256x256 cells (65536), tile slots 52/128
Islands.alm   mapview: Deadly Islands 256x256 cells (65536), tile slots 52/128
Kids.alm      mapview: Kids Paradise 80x80 cells (6400), tile slots 52/128
Kids2.ALM     mapview: Kids Paradise II 80x80 cells (6400), tile slots 52/128
LuMoir.alm    mapview: LuMoir 144x144 cells (20736), tile slots 52/128
Tomb.ALM      mapview: Heroes' Tomb 256x256 cells (65536), tile slots 52/128
Waters.alm    mapview: Waters 144x144 cells (20736), tile slots 52/128
```

All ten shipped root maps load (`failures: 0 / 10`), at three distinct grid sizes. 52/128 slots is the
full shipped set (`tile1`×16 + `tile2`×16 + `tile3`×16 + `tile4`×4). No game bytes were committed; only
these counts and names are recorded.

## AC-6 — pending manual verification

AC-6 needs a display and a lawful install, so it is **not automated**: no test in this repository opens a
window or reads a game file. To close it, run:

```
mapview -assets <install> -map <install>/Cross.ALM
```

and record here: that the window opens titled with the map name and shows terrain; that arrow keys/WASD
pan; that moving the cursor to a window edge edge-scrolls; that the wheel zooms about the cursor; that
scrolling stops at all four map edges; and that Esc closes the window.

## Defect found downstream, since fixed (0003 T5)

Running the load path above over the full install initially opened only `Cross.ALM` and `Tomb.ALM`; the
other **8 of 10** shipped maps were rejected by `pkg/formats/alm` for a zero-length section, which the
research documents as legitimate (`0 ⟺ type8 empty`, ALM-META-025):

```
Islands.alm  alm: section 7 payload too small to type: 0 bytes
```

That was a 0003 defect, not a 0005 one — 0005 neither caused it nor worked around it, and the viewer's
own error path surfaced it cleanly before the window opened, which is FR-5 behaving as specified. It is
fixed in **0003 T5** (empty sections are typed by elimination, gated on the metadata's `+0x34` count);
`mapview -check` now opens all ten maps. See `docs/0003-alm-container/verification.md` §T5 for the
per-map evidence.
