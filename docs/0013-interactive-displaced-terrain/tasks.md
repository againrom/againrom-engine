# Tasks — interactive displaced terrain

**Kinds.** `impl` — one commit. `evidence` — a developer run against a lawful install. `build` — an
untracked artifact under `builds/`. Only `impl` produces a commit.

---

### T1 — the projection answers in world space  · `impl`

**Files:** `pkg/render/terrain/project.go`, `project_test.go`.

**Covers:** FR-5 (the range itself), FR-7, SC-4 (partly), SC-5.

**Done when:** `Projection` records the grid's signed altitude extremes in the pass `Project`
already makes, and exposes `WorldCorner(c, r int) (x, y int)` and `RowRange(top, bottom float64)
(r0, r1 int)` per DD-3. Tests cover a flat grid, a single-axis slope, an asymmetric grid, a
negative-altitude grid, and a grid whose span is smaller than `Height*CellSize`; the far-edge column
and row are asserted against the clamp rule. `RowRange` is asserted to contain every row whose
corner box meets the window, over grids with `MinV` zero **and** non-zero — raise-only and push-only
families both — at windows against the top and bottom edges, and to stay inside the spec's
two-sided bound.

**Not in this task:** any caller. Nothing outside `pkg/render/terrain` changes.

---

### T2 — the camera takes a world height it is given  · `impl`

**Files:** `pkg/render/camera/camera.go`, `camera_test.go`.

**Covers:** FR-3, SC-1.

**Done when:** the camera stores a world height initialised by `New` to `Rows*CellSize`, `WorldH`
returns it, and `SetWorldHeight` re-clamps — per DD-1. `New`'s signature does **not** change and no
existing call site or assertion moves. New tests pin `WorldH() == Rows*CellSize` after construction,
clamping and centring over a world taller and shorter than the flat one, and that every shipped
camera invariant survives a `SetWorldHeight`.

**Not in this task:** altitudes, modes, culling, drawing. The camera still has no idea a projection
exists, and `VisibleTiles` is untouched.

---

### T3 — altitudes reach the viewer, and the mode follows them  · `impl`

**Files:** `pkg/render/terrain/composite.go` (`Grid`), `pkg/game/mapload.go`, `pkg/ui/viewer.go`,
`pkg/ui/overlay.go`, plus tests in `pkg/ui` and `pkg/game`.

**Covers:** FR-2, P-1, P-2, P-5, SC-2, SC-3, SC-7.

**Done when:** `Grid` carries the altitude grid, the one production load path passes it, and the
viewer builds the projection once when it is valid — per DD-4 and DD-5. `Mode()` is exposed and
computed, not stored. `syncWorld()` is called by `NewViewer` **and** by each overlay setter, so a
front end that never touches an overlay still gets the displaced extent. Tests cover an absent grid,
a wrong-length grid, a valid grid with each overlay and with both, the enable-then-disable round
trip, that a viewer built with no overlay call at all reports the displaced extent, and that the
borrowed slice is byte-identical after construction.

**Not in this task:** the displaced draw path. In displaced mode the viewer still draws flat this
commit, and a test says so, so the mode and the drawing are separable failures.

---

### T4 — the displaced draw path  · `impl`

**Files:** `pkg/ui/viewer.go`, `pkg/ui/viewer_test.go`.

**Covers:** FR-4, FR-5 (the use), FR-6, P-3, P-4, SC-4, SC-5, SC-6, SC-9.

**Done when:** displaced mode takes its rows from `RowRange` over the window
`cam.ScreenToWorld(0,0)`..`cam.ScreenToWorld(ViewW, ViewH)` and its columns from the unmodified
`VisibleTiles`, then draws each tile through the pure `tileVertices` helper and one `DrawTriangles`
per tile, row-major — per DD-2 and DD-6. Flat mode's draw loop is unchanged. Tests are windowless
and check the four vertices per tile at interior, right-edge, bottom-edge and corner positions
against `WorldToScreen(WorldCorner(..))`; the index list, source span and colour; that a tile whose
altitude step exceeds 32 is still drawn; the visit order; and that the resolved cell and water phase
match flat mode's for the same tile and tick.

**Not in this task:** batching, shading, marker projection.

---

### T5 — evidence against a lawful install  · `evidence`

**Covers:** AC-7, AC-8, AC-9, SC-10.

**Done when:** `-check` output is captured before and after the story for every overlay combination
of every command that prints one, and compared byte-for-byte; the 38 shipped maps are opened in the
standalone viewer without overlays and the outcome of each recorded; `Kids` and `61` are compared
against the renders the PNG story left behind on the criterion AC-8 states; the game front-end is
driven through the picker to a map for AC-9. Every fidelity limitation seen is recorded. Anything
that could not be run is recorded as not run, with the reason.

---

### T6 — the runnable build  · `build`

**Done when:** `builds/0013-interactive-displaced-terrain/` holds the built `mapview` and
`againrom`, and a `README.md` giving the exact invocation for each against a lawful install, always
sourcing the asset root from `-assets` or `AGAINROM_ASSETS`, with a Provenance section naming the
commit built from. `builds/README.md`'s index gains the row. Nothing under `builds/` is committed.

---

## Traceability

| Requirement | Task |
|---|---|
| FR-1 | T2, T3, T4 (each preserves it), T5 |
| FR-2 | T3 |
| FR-3 | T2 (the mechanism), T3 (the value) |
| FR-4 | T4 |
| FR-5 | T1 (the range), T4 (the use) |
| FR-6 | T4 |
| FR-7 | T1, T3 |
| AC-1, AC-2, AC-2a | T3 |
| AC-3 | T1, T2, T3 |
| AC-4 | T1, T4 |
| AC-5 | T2 |
| AC-6, AC-6a | T1, T4 |
| AC-7 | T5 |
| AC-8, AC-9 | T5 |
| P-1, P-2, P-5 | T3 |
| P-3, P-4 | T4 |
| SC-1 | T2 |
| SC-2, SC-7 | T3 |
| SC-3 | T3 |
| SC-4 | T1, T4 |
| SC-5 | T1, T4 |
| SC-6, SC-9 | T4 |
| SC-8, SC-10 | T5 |
