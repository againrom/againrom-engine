# Verification — placed-objects diagnostic overlay

Work item `0008-structures-overlay`. Research pin: submodule `8a406d6` (the overlay adds no format
decoding; `claims/alm.md` and `formats/alm/` are unchanged from the `87a72f4` pin `analysis.md` was
written against, and EXP-0031 touches only terrain lighting). Toolchain: the Go release pinned in
`go.mod` (`go 1.26.1`), windows/amd64.

**Real-map evidence re-gathered 2026-07-25, against the EXP-0030-corrected `.alm` framing.** The
headless AC-6 evidence below was first taken with the pre-EXP-0030 reader, which began each grid
layer eight bytes early. That moved the **terrain**, not this story's markers: object anchors come
from type-4 records, which are not a grid layer and whose bytes the correction did not touch. The
re-run confirms exactly that split — the counts, the token positions and the marker pixels are
identical, the rendered terrain underneath is not — and one recorded check had to be restated rather
than repeated, which is set out where it appears.

**Re-run again 2026-07-25 for the 0007 RED revision (EXP-0031).** 0007 corrected the terrain
relief-lighting gradient, so **every shaded render moved** and the image MD5s recorded here are stale.
This story's own code and tests are untouched; only the digests below are refreshed, and the
falsifiable check is that the object count and the marker pixels do **not** move. They do not. Figures
marked *(0007 RED revision)* are from that re-run; everything else is unchanged from the EXP-0030 re-run
and was re-measured, not carried over.

## Gates

All run from the repository root with **no game install present**.

| Gate | Result |
|---|---|
| `go build ./...` | clean (no output) |
| `go vet ./...` | clean (no output) |
| `go test ./...` | ok — every package green |
| `go test ./internal/archtest/` | ok — DAG + determinism-wall checks pass (no new package; `pkg/render/terrain` stays stdlib-only, `pkg/ui` adds only `ebiten/v2/vector`, already inside its tier) |
| `gofmt -l $(git ls-files '*.go')` | no output |
| `scripts/check-no-game-assets.sh` | `check-no-game-assets: clean (tree scan)` |
| `scripts/check-no-game-assets.sh --history` | `check-no-game-assets: clean (history scan)` |

Each of the four implementation commits was additionally checked out into an isolated detached
worktree and verified there — `go build ./...`, `go vet ./...`, `go test ./...` green at its own
snapshot, not only at HEAD — so every commit is independently build/vet/test-passing.

```
$ go test ./pkg/render/terrain/ -run 'Anchor|ObjectMarker|DrawObject' -v
--- PASS: TestAnchorCellAndOffMap
--- PASS: TestObjectMarkerRectsGeometry
--- PASS: TestObjectMarkerRectsExtremes
--- PASS: TestDrawObjectMarkers
    --- PASS: TestDrawObjectMarkers/map_extent
    --- PASS: TestDrawObjectMarkers/off-map_cells_only
    --- PASS: TestDrawObjectMarkers/nil_cells
    --- PASS: TestDrawObjectMarkers/undersized_image_at_the_origin
ok      againrom/pkg/render/terrain

$ go test ./pkg/ui/ -run TestObjectScreenRects -v
--- PASS: TestObjectScreenRects
    --- PASS: TestObjectScreenRects/origin_at_native_zoom
    --- PASS: TestObjectScreenRects/panned_at_native_zoom
    --- PASS: TestObjectScreenRects/panned_and_zoomed_in
    --- PASS: TestObjectScreenRects/panned_and_zoomed_out
    --- PASS: TestObjectScreenRects/panned_at_a_wheel-step_zoom
    --- PASS: TestObjectScreenRects/two_arms_per_object,_horizontal_first,_in_list_order
    --- PASS: TestObjectScreenRects/wholly_outside_is_culled,_straddling_is_kept_whole
    --- PASS: TestObjectScreenRects/off_state_and_toggle
    --- PASS: TestObjectScreenRects/off-map_anchors_contribute_nothing
ok      againrom/pkg/ui

$ go test ./cmd/terraintool/ -run TestRenderObjectsOverlay
--- PASS: TestRenderObjectsOverlay
$ go test ./cmd/mapview/ -run TestObjectsFlag
--- PASS: TestObjectsFlag
```

`pkg/render/terrain/overlay_test.go` and `pkg/ui/overlay_test.go` were authored by separate contexts
that read `spec.md`, the relevant plan sections and the API signatures but **not** the
implementation. The render-tier oracle is the FR-6 formula transcribed into the test; the viewer
oracle is `cam.WorldToScreen(corner)` plus a `Zoom`-scaled size of hand-written native arms, derived
from the camera contract and never by re-running `objectScreenRects`. Both suites passed on their
first run against the implementation, with no expectation adjusted after the fact.

## Acceptance criteria

| AC | Level | Covered by | Status |
|---|---|---|---|
| AC-1 | unit | `TestAnchorCellAndOffMap` — six `/256` coordinates, every one with a non-zero low byte so the `>>8` genuinely truncates (`0x180`→1, `0x27F`→2, `0x7FE`→7, `0xFFFFFFFF`→`0xFFFFFF`); each in-map anchor yields the FR-6 cross at `(X>>8, Y>>8)`, each off-map anchor yields nil, and no result ever exceeds 2 rectangles (no footprint) | PASS |
| AC-2 | unit | `TestObjectMarkerRectsGeometry` — **16 hand-computed cases in total**: 12 non-degenerate (`cellpx` 32/16/17/15 × three cells), each matching the FR-6 rectangles exactly, and 4 degenerate `cellpx ≤ 2` cases (1×1 and 2×2 maps) where the map-rect clip genuinely truncates, asserted edge-by-edge as `max(rawMin,0)`/`min(rawMax,mapMax)` and asserted to *differ* from the unclipped arm on both the low and the high side. `TestDrawObjectMarkers` covers the rasterized half | PASS |
| AC-3 | unit | `TestObjectMarkerRectsExtremes` — `cellpx ∈ {0,-1,-32,-(1<<30)}` → nil; five off-map extremes (widest `AnchorCell` output, huge/negative col and row) → nil with 0 allocations; an in-map extreme (`col=row=2²⁴−1` in a 2²⁴×2²⁴ map at `cellpx=4096`) → exactly 2 arms, no panic; `testing.AllocsPerRun` is 1 for both a small- and a huge-magnitude anchor, so allocation does not scale with magnitude | PASS |
| AC-4 | unit | `TestRenderObjectsOverlay` (the no-flag PNG is byte-identical to the compositor's own output and the summary carries no `objects` token), `TestObjectsFlag` (the no-flag summary is character-for-character its pre-0008 shape), `TestObjectScreenRects/off_state_and_toggle` (the viewer returns nil with the overlay off, non-nil after re-enabling) | PASS |
| AC-5 | unit | `TestRenderObjectsOverlay` and `TestObjectsFlag` — both report `objects 3`, the decoded `len(Map.Objects)`, with the off-map anchor counted | PASS |
| AC-6 (headless half) | manual | a real GOG map through both tools with the overlay on — developer-run on `Islands.alm`; both report `objects 287`, the token appears only with the flag and in the DD7 position, the overlay writes exactly its marker rectangles and nothing else, and the marker pixels are byte-for-byte where they were before the framing correction (see *Developer-run evidence*) | PASS |
| AC-6 (live-window half) | manual | markers visually sitting on their objects' terrain cells and staying aligned through pan/zoom in `mapview` — needs a GUI session and a human eye, **not yet run** (see below) | ⏳ |

## Derived properties

| P | Covered by | Status |
|---|---|---|
| P-1 (≤ 2 rects per object; never indexes outside map/output bounds) | `TestAnchorCellAndOffMap` (the ≤ 2 assertion on every case), `TestObjectMarkerRectsExtremes`, and `TestDrawObjectMarkers` — which walks **every** pixel of the target image asserting marker-vs-background, and includes an undersized origin-anchored image where the arms truncate to the image bounds with no synthetic border. See the honest bound on the output-bounds half below (*the stage-2 clip is not independently witnessed*) | PASS, with a stated bound |
| P-2 (no panic, no magnitude-proportional allocation) | `TestObjectMarkerRectsExtremes` — the `AllocsPerRun` equality between small and huge magnitudes (1 vs 1, logged so the equality is visibly not two zeros), and the extreme in-map/off-map cases completing without panic | PASS |
| P-3 (overlay disabled ⇒ terrain output byte-identical) | `TestRenderObjectsOverlay` — the no-flag PNG bytes equal a **direct `CompositeLit` call** for the same map, tileset, light and scale, encoded the same way. The oracle is the compositor, not a second no-flag run, so the check cannot pass by comparing the tool to itself | PASS |

## Plan success criteria

SC-1…SC-4 are the render-tier tests above; SC-5 is `TestRenderObjectsOverlay`; SC-6 is
`TestObjectScreenRects`; SC-7 is `TestObjectsFlag`; SC-8 is the existing `internal/archtest`
fail-closed DAG check, green with `pkg/render/terrain` still stdlib-only and `pkg/ui` adding only an
ebiten sub-package its tier already permits. SC-9 is the developer-run evidence below.

**SC-7 is covered in part only.** Its first half — `mapview -objects` reports `objects N` on the
`-check` summary, in the position `load()` builds it — is asserted directly, and the assertion is
position-sensitive by construction (the expected string is rebuilt by splicing the token into the
disabled summary, so appending it elsewhere fails; verified by mutation). Its second half, "passes
the cells to the viewer", has **no independent automated witness**: the viewer exposes no accessor
for its overlay state, and adding one would have been an unplanned API beyond the plan's
files-to-touch. What the token does witness is that the conditional block ran, since the
`SetObjects` call and the token are appended in that same block. That the cells are then transformed
and drawn correctly is covered by `TestObjectScreenRects` at the tier below and by AC-6 end-to-end —
which is pending. This is a real gap in the automated chain, recorded rather than papered over.

## Green-but-hollow audit

Each suite was challenged rather than trusted:

- **The render-tier suite was mutation-probed by its author**: decrementing the *returned* arm's `Max.X`
  failed all 16 geometry cases, and painting a single stray marker pixel failed all three complement
  sub-cases of the draw test. The probes were removed afterwards. (Mutating the FR-6 *formula* instead —
  `cx+r+1` → `cx+r` — fails 13 of the 16, the three degenerate `cellpx ≤ 2` cases surviving because the
  map clip truncates the difference away.)
- **The layered audit found one probe the suite does not survive**, and it is recorded rather than
  quietly fixed: deleting the FR-3 stage-2 clip changes nothing observable. See the first bullet under
  *Limitations* — that is a genuine hollow spot in the coverage, not a passing check.
- **The clip is tested where it is actually reachable.** At every realistic `cellpx` (≥ 3) the cross
  fits strictly inside its own cell, so a `cellpx=32` edge cell would *not* exercise the map-rect
  clip — a suite that only tested those scales would be green and hollow. The truncating cases are
  therefore at `cellpx ≤ 2` on 1×1 and 2×2 maps, and they assert the clipped arm **differs** from the
  unclipped one, so a no-op clip fails.
- **The viewer suite was discrimination-checked**: at the origin at native zoom `WorldToScreen` and
  `ScreenToWorld` coincide, which would make a transform bug invisible — so the panned and zoomed
  positions carry the weight, including a non-dyadic `1.2` wheel-step zoom. Probes confirmed that a
  `ScreenToWorld` swap, a missing zoom scaling, and a dropped or over-eager cull each fail the suite.
- **The terraintool wiring was mutation-probed here**: drawing the overlay unconditionally (removing
  the flag guard) failed the P-3 byte-identity assertion *and* the "the unmarked render already holds
  N marker-coloured pixels" guard. The marker-pixel count is an independently derived 138 = 2 × (13·3
  + 3·13 − 3·3), and the test first asserts the unmarked render contains **zero** marker-coloured
  pixels, so the count is evidence of the overlay rather than of the terrain palette.
- **The mapview wiring was mutation-probed here**: moving the token to the front of the summary
  failed the reconstruction assertion.

## Pre-task gate findings

The defect class the pre-task gates were aimed at, and what they actually found:

- **T1 — a synthetic clip edge, and a clip that is never exercised.** No instance of the first was
  found. The second was real and was already anticipated at the plan gate (the DD4 honesty note); the
  suite is built around it, as described in the audit above.
- **T2 — overlay code leaking into the disabled path.** No instance found in the shipped code; the
  mutation probe confirms the assertion would catch one.
- **T3 — a transform that is not the terrain tile's.** No instance found. The float64/float32
  boundary that would have made the SC-6 comparison inexact was found *at the plan gate*, before any
  code existed, and settled in DD6.
- **T4 — a count that is a recomputed record count rather than the decoded total** (plan R-2). No
  instance found: the code reports `len(m.Objects)` directly.

One material defect was found by the gates and is recorded under *Classified revisions* below.

## Developer-run evidence (AC-6 / SC-9, headless half)

Collected by the developer on a lawful GOG install — `C:\Program Files (x86)\GOG Galaxy\Games\Rage of
Mages`, map `Islands.alm` ("Deadly Islands", 256×256 cells, 8192×8192 px at scale 1) — using the
`builds/0008-structures-overlay/` binaries, output to the git-ignored `terraintool-out/`. Observations
only; no game bytes are committed.

**The count, and the flag's effect on the summary.** `terraintool render -objects`:

```
terrain: 256x256 cells (65536), 8192x8192 px at scale 1, tile slots 52/128, placeholder cells 0, shaded (theta=0.7854 ambient=14 range=32), objects 287
```

The same render with the flag absent is identical except that the `, objects 287` token is gone. So on a
real map the token appears only with the flag and only at the end of the line — the DD7 position, here
confirmed outside the synthetic fixture (FR-5, AC-4, AC-5). **That line is character-identical after the
0007 revision** *(0007 RED revision)* — the light descriptor prints `%.4f`, and `0.7854` is what both
the old `math.Pi/4` and 0007's corrected literal round to, so nothing this story reports moved.

**`objects 287` held across the framing correction; `placeholder cells` moved from 2 to 0.** The
count is `len(m.Objects)` from the type-4 record, which the correction left byte-for-byte alone; the
placeholder pair was two cells of the *terrain* grid — the type-1 record header's identity words
resolving to absent tile slots — and they are gone now that the grid starts where it belongs (see
`docs/0004-terrain-viewer/verification.md`, AC-8). Nothing in this story reads or reports either.

`mapview -check -objects` reports the same count, with the token between the tile slots and the cadence:

```
mapview: Deadly Islands 256x256 cells (65536), tile slots 52/128, objects 287, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)
```

and without the flag the `objects 287` clause is absent. **Both tools agree on 287** — the DD7 token
position is now pinned on a real map in both cmds, which is exactly what the adversarial reviewer asked
for and what the synthetic fixtures could only pin at 4×3 scale.

**P-3 on a real map — restated, because the original comparison cannot be repeated.** The first run
compared the flag-off render against the binary shipped before this story
(`builds/0007-terrain-lighting/terraintool.exe`) and found them byte-identical at
`5b33e2f136388124d1da48c04dce1ee0`. That comparison is no longer available and cannot be reconstructed:
the binary users actually had embeds the pre-EXP-0030 reader, so it renders the mis-framed terrain by
construction and *must* differ; and both build directories now hold the same rebuilt tool, so comparing
them would only compare a binary with itself. Recording it as though it still discriminated would be
dressing up a tautology. Two checks that do **not** depend on which binary produced the image replace
it.

*The overlay writes exactly its marker rectangles and nothing else.* On `Islands.alm`, through the same
render-tier calls the tool makes: the lit composite was hashed, the 287 anchors' markers were drawn, and
the pixels inside the FR-6 rectangles were then restored. Every one of those pixels carried
`MarkerColor` after the draw (22 386 rectangle pixels, 19 803 distinct — the two arms overlap in the
3×3 centre), and the restored image hashed **identically to the pre-overlay render**. So on real map
data the overlay's writes are confined to the rectangles FR-6 specifies; no other pixel moves. This is
the disabled-path identity P-3 asks for, witnessed from the other side and without a binary provenance
claim. **Re-run after the 0007 revision (0007 RED revision):** unchanged in substance — the overlay
changes exactly **19 803** pixels and the restored image hashes identically to the pre-overlay render
(`5403ba937395892f0888b9f0279ea523` both sides). The pre/post-overlay hash itself is a *different*
number than before, because the terrain underneath it is, and it is quoted only as an equality.

*The markers did not move when the terrain did.* The `-objects` render of `Islands.alm` taken before the
framing correction and the one taken after carry **exactly 19 803 marker-coloured pixels each** — 287 ×
69, the FR-6 cross at `cellpx = 32` — with an **identical digest over their coordinates**
(`04cb96b57247527fbaa388c6383e92a6`), while the two images' own MD5s differ
(`f14c3309924028054fe56ba5bdde3226` before, `9d24f9c6987e6392b5cbf3db874a3969` after). The flag-off
render moved the same way (`5b33e2f1…` → `ea65f49ca4862cdc29dc9162949eb4d8`). Markers unchanged,
terrain beneath them changed — which is the whole of this story's exposure to the correction.

*And they did not move when the terrain changed again (0007 RED revision).* The same check was repeated
across 0007's arithmetic correction, with the pre-revision binary built at `c63cb25` and the
post-revision one at `09bef64`:

| `Islands.alm` render | Before (`c63cb25`) | After (`09bef64`) | |
|---|---|---|---|
| `-objects`, image MD5 | `9d24f9c6987e6392b5cbf3db874a3969` | `fbeac976cc5e3d9a07472d5e8ccb88b8` | **differs** (expected) |
| flag-off, image MD5 | `ea65f49ca4862cdc29dc9162949eb4d8` | `d945b6a0d1f2d85ff96282a1858ba231` | **differs** (expected) |
| `-objects`, marker pixels | 19 803 | 19 803 | **identical** |
| `-objects`, marker coordinate digest | `0795fb6a817134346f9c996da8b812ae` | `0795fb6a817134346f9c996da8b812ae` | **identical** |
| reported `objects` | 287 | 287 | **identical** |

The coordinate digest is MD5 over the ASCII lines `"x,y\n"` of the marker-coloured pixels in raster
order. That serialization is stated because it is **not** the one behind the EXP-0030 figure
`04cb96b5…` above, whose probe's exact serialization was not recorded — so `0795fb6a…` is a new
measurement with a stated method, not a reproduction of the older digest. What the pair of digests
establishes is the property that matters: identical marker coordinates across a change that moved every
terrain pixel.

**Scope, stated honestly: one map, scale 1, default shading globals** — not a claim across scales, maps
or light overrides.

**Independent count cross-check, and precisely what it does not show.** `almtool info` on the same map
reports a type-4 `payloadSize` of 5740 — the same figure as before the framing correction, which moved
where each record's payload begins but not how large it is — and `5740 = 287 × 20` exactly. So this map carries **no**
extension records, and on it the adaptive walk's count coincides with `type4_size/20`. That corroborates
the reported number, but it **does not exercise the extension path R-2 warns about** — it is not evidence
that the walk beats `size/20` in general, and no such claim is made here. R-2's mitigation remains
argued from the decoder, not demonstrated on this map.

**What this does not cover.** Nothing above involves a window. Whether markers visually sit on the terrain
cells of the structures they mark, and stay aligned through pan and zoom, is unobserved — see below.

## AC-6 / SC-9 — remaining pending developer-run verification

The **live-window half** of AC-6 remains unrun: it needs a GUI session and a human eye, and no test in
this repository reads a game file. A developer runs:

```
go run ./cmd/terraintool render -assets <install-dir> -map <map.alm> -out terraintool-out/objects.png -objects
go run ./cmd/mapview -assets <install-dir> -map <map.alm> -objects
```

(or with `AGAINROM_ASSETS` set; `-scale N` enlarges the PNG). To close the remaining half, record here:

- that the PNG markers sit on cells that hold visible structures, and that no marker lands in open
  terrain where nothing stands;
- that the live viewer shows the same markers and that each stays on its terrain cell through pan and
  through zoom in and out — the check is alignment with the terrain cell, **not** pixel-equality with
  the PNG (R-1 makes them differ by up to a pixel at a boundary, by design);
- whether the count matches the map's object count as the editor or the game shows it. The headless run
  established that both our tools report 287 and that 287 × 20 is the exact payload size; it did **not**
  establish that 287 is what the game itself considers this map's object count.

Only these observations are committed; a rendered PNG is a converted game asset and is git-ignored.
**Status of this half: not yet run.** No result may be recorded here that was not observed on a real
install, and nothing in the conclusion below rests on it.

## Classified revisions taken during this pass

- **A material defect found at the Phase 3 task gate started a revision at `plan.md`.** The plan's
  frozen baseline did not record that `alm`'s type4 walk discriminates the optional 8-byte extension
  by testing the *candidate next record's* anchor with the same off-map predicate this overlay
  applies. The consequence is that the fixture both cmd tasks were told to build — "objects at chosen
  anchor cells including one off-map" — is not constructible as worded: in the plain no-extension
  layout an off-map anchor at any index ≥ 1 is swallowed as the previous record's extension and
  `alm.Open` fails outright. Verified against the real decoder on hand-built maps before the fact was
  written down, including the counter-case that shows this is a property of the *layout*, not a limit
  of the format (a payload with a genuine extension does carry an off-map anchor at a later index and
  decodes cleanly). The revision was re-gated by an adversarial re-read, which caught a residual
  overstatement in two files-to-touch rows before the gate passed. No `spec.md` change was needed:
  the constraint is a fixture-construction fact, not a contract change, and the render tier's own
  off-map cases take plain cells rather than bytes and are unaffected.
- **The same re-gate closed two design decisions the plan had left open**, both of which would
  otherwise have been decided in a task: whether `screenRect` holds `float32` or `float64` (it holds
  `float64`, narrowing only at the drawer call, which is what makes the SC-6 comparison exact), and
  `DrawObjectMarkers`' image-origin precondition (without which SC-4's undersized-image case would
  have passed vacuously by drawing nothing into a mismatched coordinate frame).
- **The layered audit found a material S-6 violation and started a revision at `spec.md`.** FR-3 as
  originally written required every marker rectangle to be intersected with "the output/view rectangle",
  which the interactive viewer does not do — it culls wholly-outside rects and lets the framebuffer clip
  the rest, because rounding screen coordinates to integer view bounds is precisely the independent pixel
  snapping FR-6 forbids. The resolution existed only in the plan's DD6, i.e. a downstream document
  deciding what a spec MUST meant. FR-3 now states the two realizations and why the viewer's is the only
  one compatible with FR-6, and DD6 derives from it instead of arguing with it. FR-2's blanket "no
  floating-point state" was reconciled the same way — it binds the marker *model*, not the viewer's
  float placement, which FR-6 mandates. Both keep their IDs: the observable contract (no marker pixel
  outside the output rectangle, no synthesized edge) is unchanged, so this is a reconciliation of
  under-specified text rather than a different contract. The audit also corrected the spec's own
  gate-check mapping, which had left FR-1's and FR-3's enabled, both-viewers half traced to no criterion
  at all — it is AC-6, the manual one.

## Limitations and honest bounds

- **The FR-3 stage-2 (output-rect) clip is not independently witnessed, and cannot be.** `image.RGBA`'s
  `SetRGBA` bounds-checks and **returns silently** on an out-of-bounds point — it never panics. So
  deleting `arm.Intersect(img.Bounds())` from `DrawObjectMarkers` yields a **bit-identical** image and the
  whole suite stays green; this was confirmed by running that mutant. Consequences, stated plainly:
  the *behaviour* FR-3 requires (arms truncate at the image edge, no synthetic border, nothing written
  outside the image) is real and is witnessed by the every-pixel complement walk and the undersized-image
  case; but our explicit clip is **redundant with `SetRGBA`'s own bounds check**, so no test can
  distinguish its presence, and none claims to. An earlier draft of this file asserted that the tests run
  "under a `recover` guard so an out-of-bounds write would fail loudly" — that was **wrong** and has been
  removed: `SetRGBA` cannot panic, so the guard proves nothing about out-of-bounds writes. The clip stays
  because it avoids iterating a potentially large run of no-op writes, which is a performance property,
  not the correctness guarantee the earlier wording implied.
- **The clip contract is API-level, not reachable through either shipped tool.** `terraintool` enforces
  `scale ≥ 1`, so `cellpx = 32·scale ≥ 32`, at which the arm reach `6·scale` is always well inside the
  half-cell `16·scale`; the viewer is fixed at `cellpx = 32`; and both production callers pass an image
  that is exactly the map extent, so stage 2 never truncates in production either. The truncating cases
  in the test suite are therefore *API* cases at `cellpx ≤ 2` and with a deliberately undersized image —
  genuine tests of the contract, but not of a situation a user can currently reach.
- **FR-4's composability half is asserted, not verified.** What is tested is that the overlay is an
  *independent* toggle (enabled and disabled cases in both tools, and the viewer returning nil when off).
  The draw order `terrain → objects → [units]` cannot be tested yet: the units overlay does not exist
  (story 0009). Composability with a second overlay is therefore a design property here, not an
  evidenced one.
- **No engine-fidelity claim is made about anything this story draws.** The glyph, its colour, its
  size and the draw order are the project's own diagnostic design. The only decoded facts consumed
  are that a type4 record's `X`/`Y` are `/256` fixed-point and that the integral cell is `X>>8`,
  `Y>>8` (`ALM-OBJ-019`); this story parses no ALM bytes and interprets no `kind`/`id`/`value`/
  extension field.
- **No object footprints, and that is a decode limitation, not a simplification.** The type4
  extension is an 8-byte pair of undecoded meaning, not a width/height, so each object is marked by
  its anchor cell alone. Real object extents need the class registries and the static-data formats.
- **R-1 stands as designed:** the viewer's markers are not pixel-identical to the PNG's. The viewer
  routes them through the float camera transform exactly as terrain is routed, so at a given zoom a
  marker can differ from the PNG by up to a pixel at a boundary. That is the requirement, not a
  defect — a marker tracks its terrain cell rather than a fixed pixel grid.
- **R-3 stands as designed:** at an even thickness (e.g. `cellpx=17` ⇒ `t=2`) the centred strip is
  biased one pixel toward the low side of the centre pixel. This is the exact FR-6 contract and is
  pinned by `TestObjectMarkerRectsGeometry`. The native `cellpx=32` render (`t=3`) is symmetric.
- **`objects 0` is not exercisable at the cmd tier.** A synthetic `.alm` cannot represent
  `#type4 = 0`: an empty section may only eliminate to type 8, and a non-empty payload with count 0
  fails the walk's exact-consumption check. Only the *disabled* path witnesses "no count reported".
  A real map with no placed objects would exercise it; that is an observation for AC-6, not a
  covered case.
- **The full-frame viewer identity is not automated.** With the overlay off, `objectScreenRects`
  returns nil — that is the automatable witness, and it is asserted. That the drawn frame is
  therefore identical to the pre-0008 frame follows from the overlay being the only added draw call,
  but proving it pixel-wise needs a window and belongs to AC-6.
- **No research gap is open for what this story ships.** Nothing here was reverse-engineered or
  guessed; the one fact consumed is already at claim level in the pinned research, and every other
  decision is labelled project engineering.

## Commits

| Commit | Subject | `SDD-Task` |
|---|---|---|
| `b83b2bd` | docs(0008-structures-overlay): revise plan — alm type4 fixture constraint, float boundary, clip preconditions | — (workflow) |
| `0f8f668` | docs(0008-structures-overlay): task breakdown | — (workflow) |
| `9178947` | feat(render/terrain): placed-object marker geometry and PNG rasterizer | `0008-structures-overlay/T1` |
| `b592fbb` | feat(terraintool): opt-in placed-objects overlay and object count | `0008-structures-overlay/T2` |
| `2651c4e` | feat(ui): placed-objects overlay in the interactive viewer | `0008-structures-overlay/T3` |
| `ed81598` | feat(mapview): opt-in placed-objects overlay and object count | `0008-structures-overlay/T4` |
| `b6e2487` | docs(0008-structures-overlay): editorial — tag SC-6 with FR-3 and SC-9 with FR-5/FR-6 | — (workflow) |

One `SDD-Task` id per implementation commit, none reused; workflow commits carry none; no
`Co-Authored-By` trailer anywhere. (Two unrelated `docs(0009-units-overlay)` commits are interleaved
in this range — a concurrent work item on the same branch, touching only its own spec file and
outside this story's mapping.)

## Conclusion

The placed-objects diagnostic overlay is implemented in both viewers and **all five automatable
acceptance criteria and all three derived properties pass** under separate-context, spec-derived
tests, with every gate green and no game install present. The clip behaviour FR-3 demands is tested
where it is genuinely reachable rather than where it would be free, and the disabled-path identity is
witnessed against the compositor's own output rather than against the tool itself.

The **headless half of AC-6 is evidenced on a real map**: both tools report `objects 287` on
`Islands.alm`, the token appears only with the flag and in its specified position in each summary, the
overlay's writes are confined to the FR-6 marker rectangles (restoring only those pixels reproduces the
pre-overlay render's hash exactly), and the marker pixels are byte-for-byte where they were before the
EXP-0030 framing correction moved the terrain beneath them. Bounded to one map at scale 1 with default
shading. **The same independence holds across 0007's RED revision**, which moved every terrain pixel
again for a different reason: `objects 287`, 19 803 marker pixels and an identical marker-coordinate
digest, with both image MD5s changed. Two independent terrain changes have now failed to move this
story's output, which is the strongest form the claim can take without a window.

Three things bound this conclusion. **The live-window half of AC-6 has not been run** — whether markers
visually sit on their structures' cells and hold alignment through pan and zoom is unobserved, and
nothing above rests on it. **The "cells reach the viewer" half of SC-7 has no independent automated
witness**, only a structural one. And **the FR-3 stage-2 clip is not witnessed at all**: it is redundant
with `SetRGBA`'s own bounds check, so deleting it changes nothing observable — the required behaviour
holds, but our code is not what is proven to provide it.

Within those bounds the geometry, the map-rect clipping, the camera transform, the two flags, the
disabled-path identity and the reported count are verified. The on-screen alignment against real game
data is not yet evidence, and is the one thing a reader should not assume from this document.
