# Verification — placed-units diagnostic overlay

Evidence about the implementation as landed. Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` →
`plan.md` §Success criteria; `R-x` → `plan.md` §Risks; `Tn` → `tasks.md`.

Environment: Go 1.26.1 (the toolchain pinned in `go.mod`), Windows 11. Automated evidence was produced
with **no game install visible to the test suite** — every fixture is bytes built in test code. The
developer-run evidence used a lawful GOG install at an asset root supplied through `AGAINROM_ASSETS`;
**no game byte, map file or rendered image is committed**.

**Developer-run evidence re-gathered 2026-07-25 for the 0007 RED revision (EXP-0031).** 0007 corrected
the terrain relief-lighting gradient, so **every shaded render moved** and the image MD5s recorded here
were stale. This story's code and tests are untouched; only the digests are refreshed, and the
falsifiable check is that the unit/object counts and the marker pixels do **not** move. They do not.
Figures marked *(0007 RED revision)* are from that re-run; every other figure below was re-measured and
found unchanged rather than carried over.

## Gate results

Run from the repo root at `3a7d36e`:

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `go test -count=1 ./...` | **all packages ok** — `cmd/mapview`, `cmd/terraintool`, `internal/archtest`, `internal/notices`, `pkg/formats/alm`, `pkg/formats/res`, `pkg/formats/spr256`, `pkg/game`, `pkg/render/camera`, `pkg/render/terrain`, `pkg/ui` |
| `gofmt -l $(git ls-files '*.go')` | empty |
| `internal/archtest` import DAG | green (runs inside `go test`) |
| `bash scripts/check-no-game-assets.sh` | `clean (tree scan)` |
| `bash scripts/check-no-game-assets.sh --history` | `clean (history scan)` |
| `SDD-Task` bijection over `78a2699..HEAD` | 4 IDs (`T1`…`T4`), 0 duplicates, 0 `Co-Authored-By` trailers |

## Automated criteria

| SC | Method | Result |
|---|---|---|
| SC-1 | `TestUnitAnchorCellAndOffMap` | **pass** — `AnchorCell` is exactly `>>8` on both axes over fractional `/256` inputs; a 5×5 sweep of both low bytes over one cell proves the fraction changes no output; two anchors reaching one cell by different low bytes give identical geometry; three off-map anchors (incl. `0xFFFFFF` from the `uint32` maximum) give `nil`; never more than 2 rectangles |
| SC-2 | `TestUnitMarkerRectsGeometry` | **pass** — 21 hand-computed rows over `cellpx` 32, 15, 16, 17, **48, 64, 96**; an explicit centring block at 48/64/96 (where `⌊t/2⌋ = 1`) asserting the strip is `[c-⌊t/2⌋, c-⌊t/2⌋+t)` and naming the `[c, c+t)` a dropped centring term would produce; a 1×1 map at `cellpx` 1 and 2 where the clip genuinely truncates, with a guard that the arm really was truncated; and all four corners at `cellpx = 32` asserted (not assumed) to truncate **nothing** |
| SC-3 | `TestUnitMarkerRectsExtremes` | **pass** — `cellpx ∈ {0, −1, −32, −2³⁰}` → `nil`; five off-map extremes → `nil`, no panic, **0 allocations**; an in-map extreme (`col = row = 2²⁴−1` at `cellpx = 4096`) → the exact FR-6 pair against a hand-derived literal, no wrap; allocation count identical for small and huge anchors |
| SC-4 | `TestDrawUnitMarkers` | **pass** — exact clipped pixel sets with the complement asserted pixel-by-pixel; off-map cells draw nothing; coincident units give a byte-identical image to one unit; nil image, nil/empty cell list, zero-size target and `cellpx = 0` are no-ops without panic; an undersized origin-anchored image truncates to the bounds with no inset and no smear; and the **stacking witness** — objects→units leaves cyan on every unit-cross pixel and yellow on the 52 object-only pixels, while units→objects leaves **zero** unit-marker pixels anywhere |
| SC-5 | `TestRenderUnitsOverlay` | **pass** — every in-map anchor cyan at `-scale 1` and `-scale 2`, off-map anchor nowhere; marker totals equal the FR-6 formula computed in the test; `objects N` precedes `units N`; the summary is identical under `-objects -units` and `-units -objects`; a unit-free fixture prints `units 0`; and with `-units` absent the PNG is byte-identical to an oracle built directly from the render tier in **both** object states |
| SC-6 | `TestUnitScreenRects` | **pass** — the transform equals an oracle derived from the camera contract at five pan/zoom positions with exact `float64` equality; a straddling arm is kept **whole** (the framebuffer performs that clip; rounding it would be the pixel snapping FR-6 forbids) while a fully-outside arm is culled; off-map anchors contribute nothing; `nil` with the overlay off, with nil cells and with an empty slice; and **all four** toggle combinations assert *both* overlays, so neither toggle can be shown to disturb the other |
| SC-7 | `TestUnitsFlag` | **pass** — each enabled summary reconstructed by splicing its token into the disabled one, pinning position as well as content; each flag emits only its own token; `objects` precedes `units`; identical under either CLI flag order; a unit-free fixture prints `units 0` |
| SC-8 | `internal/archtest` + the unmodified 0008 tests | **pass** — the fail-closed DAG check is green (`pkg/render/terrain` stdlib-only, `pkg/ui` gained no dependency), and `pkg/render/terrain/overlay_test.go` and `pkg/ui/overlay_test.go` are **byte-identical to their 0008 versions** (`git diff` against the pre-story commit is empty) and pass |
| SC-10 | `TestOverlayPassOrder` | **pass** — both overlays on gives exactly two passes, object yellow at `[0]` and unit cyan at `[1]`; one overlay gives one pass; neither gives none; an enabled-but-empty object pass is omitted rather than left to shift the index; guarded by asserting the two colours differ, so the order check cannot be vacuous |

**Non-vacuity check (SC-6/SC-10).** The test author mutated its own FR-6 transcription (horizontal arm
`cx-4` → `cx-5`) and confirmed every relevant subtest then failed, before restoring and re-running green.
The tests can fail.

**Separate-context discipline.** `pkg/render/terrain/unit_overlay_test.go` and
`pkg/ui/unit_overlay_test.go` were authored by two contexts that read `spec.md` and the relevant plan
sections but **not** the implementation files under test. Both reported **no discrepancy** between the
spec-derived expectations and the implementation, on the first run, with no test weakened to get there.

## Developer-run evidence (T5 / AC-6)

Install: a lawful GOG *Rage of Mages* install, asset root supplied via `AGAINROM_ASSETS`. Binaries:
`builds/0009-units-overlay/{terraintool,mapview}` built from `3a7d36e`. All output written outside the
repo.

### Reported counts (headless, both tools)

| Map | Size | Objects | Units |
|---|---|---|---|
| `Kids.alm` ("Kids Paradise") | 80×80 (6 400) | 16 | 51 |
| `LuMoir.alm` | 144×144 (20 736) | 107 | 228 |
| `Horror.alm` | 256×256 (65 536) | 415 | 1 815 |

`terraintool` and `mapview -check` report the same counts for each map, in the order
`…, objects N, units M`, and the line is character-identical under `-objects -units` and
`-units -objects`.

### Placement — SC-9, the half that carries the diagnostic value

Every decoded unit's anchor cell was checked for a marker at its centre pixel, by an independent probe
run outside the repo that re-derives `(X>>8, Y>>8)` itself:

| Map | Units | Centre-pixel hits | Misses |
|---|---|---|---|
| `Kids.alm` | 51 | 51 | **0** |
| `LuMoir.alm` | 228 | 228 | **0** |
| `Horror.alm` | 1 815 | 1 815 | **0** |

**2 094 / 2 094 anchors marked, zero misses.** Total marker pixels also match the FR-6 formula computed
independently of the render tier — `2·(2r+1)·t − t²` per marker:

| Map / scale | `r`,`t` | Expected | Observed |
|---|---|---|---|
| `Kids` units, `cellpx` 32 | 4, 1 | 51 × 17 = 867 | 867 |
| `Kids` units, `cellpx` 64 (`-scale 2`) | 8, 2 | 51 × 64 = 3 264 | 3 264 |
| `LuMoir` units, `cellpx` 32 | 4, 1 | 228 × 17 = 3 876 | 3 876 |
| `Horror` units, `cellpx` 32 | 4, 1 | 1 815 × 17 = 30 855 | 30 855 |
| `Kids` objects, `cellpx` 32 | 6, 3 | 16 × 69 = 1 104 | 1 104 |
| `Horror` objects, `cellpx` 32 | 6, 3 | 415 × 69 = 28 635 | 28 635 |

The `cellpx = 64` row is R-6's production-reachable even-thickness case, exercised end to end.

**All of the above re-measured after 0007's RED revision** *(0007 RED revision)*: `Kids` 51 units /
16 objects, `LuMoir` 228 / 107, `Horror` 1 815 / 415; centre-pixel hits 51 / 228 / 1 815 with **0
misses** at `-scale 1`, and the same at `-scale 2`; marker pixel totals 867 / 3 264 / 3 876 / 30 855 /
1 104 / 28 635 exactly as tabulated. Not one figure in this subsection moved.

### P-3 on real maps

The pre-story binary (built from `315e1a3`, the commit before T1) and the post-story binary were run on
the same maps and their PNGs compared by MD5:

| Map | Flags | Pre-story | Post-story | |
|---|---|---|---|---|
| `Kids.alm` | none | `a76b022d9c89e3aef2e567f061d8c7cf` | `a76b022d9c89e3aef2e567f061d8c7cf` | **identical** |
| `Kids.alm` | `-objects` | `efc81c791fa56e1bc877e3753e44fa6c` | `efc81c791fa56e1bc877e3753e44fa6c` | **identical** |
| `LuMoir.alm` | none | `2b326f9195a5e38d0d182988a59e5fa5` | `2b326f9195a5e38d0d182988a59e5fa5` | **identical** |
| `LuMoir.alm` | `-objects` | `f84da5980484c42f55aff443a1f282eb` | `f84da5980484c42f55aff443a1f282eb` | **identical** |

This was P-3/AC-4 on real data, and simultaneously the strongest evidence for **R-3**: the DD2 refactor
of the shipped object geometry reproduced the previous binary's object renders byte-for-byte on real
maps, not merely on synthetic fixtures.

**This comparison is no longer repeatable, and is not restated as though it were** *(0007 RED
revision)*. The `315e1a3` binary embeds 0007's superseded relief gradient, so it now renders different
terrain **by construction** and must differ from any current build — re-running the table would compare
two different lighting models and prove nothing about this story. The four digests above are left as the
historical record of a check that did discriminate when it was run, at the arithmetic of its day; they
are **not** current expected values, and no current binary reproduces them.

What replaces it are two checks that do not depend on which binary produced the image:

*The disabled overlay's output still comes from the compositor alone.* With `-units` absent, the PNG is
byte-identical to an oracle built directly from the render tier, in **both** object states — that is
`TestRenderUnitsOverlay` (SC-5), which is unaffected by 0007 and stays green.

*The markers did not move when the terrain moved.* The same maps were rendered with the pre-revision
binary (`c63cb25`) and the post-revision one (`09bef64`):

| Render | Before (`c63cb25`) | After (`09bef64`) | |
|---|---|---|---|
| `Kids.alm` flag-off, image MD5 | `a76b022d9c89e3aef2e567f061d8c7cf` | `cec6de282e37644d72bd093e40921435` | **differs** (expected) |
| `Kids.alm` `-objects`, image MD5 | `efc81c791fa56e1bc877e3753e44fa6c` | `26fc5caf493ed7d95e97f1ae66e866c5` | **differs** (expected) |
| `LuMoir.alm` flag-off, image MD5 | `2b326f9195a5e38d0d182988a59e5fa5` | `e19c6b1b5bfb5fcb2b926eaee9f30b9f` | **differs** (expected) |
| `LuMoir.alm` `-objects`, image MD5 | `f84da5980484c42f55aff443a1f282eb` | `5c0b14fee5ebb81ee2dd1c4501b2c4c0` | **differs** (expected) |
| `Kids.alm` `-units`, image MD5 | `b31239012df5b87ed55ce87060ec9106` | `9a6282ed8de4d00f481fb21ce65be016` | **differs** (expected) |
| `LuMoir.alm` `-units`, image MD5 | `8024e822bea2cf0cade274b289eb2aaa` | `ac07de7e4ccf768b0814f28af5401c77` | **differs** (expected) |
| `Horror.alm` `-units`, image MD5 | `5e0a08a2ce76d6e0e6a252b9c5d04c85` | `da5fb519e73a31357c1dd9679e123b7a` | **differs** (expected) |
| `Kids.alm` unit markers | 867 px, digest `fbd49719217d2f2d712ca205f6299d50` | 867 px, `fbd49719217d2f2d712ca205f6299d50` | **identical** |
| `LuMoir.alm` unit markers | 3 876 px, `7b08d1d6061ddd65f7271ce11cf26800` | 3 876 px, `7b08d1d6061ddd65f7271ce11cf26800` | **identical** |
| `Horror.alm` unit markers | 30 855 px, `4c417a6cba9a31433f540447ea55a137` | 30 855 px, `4c417a6cba9a31433f540447ea55a137` | **identical** |
| `Kids.alm` object markers | 1 104 px, `bf38bdffb164fcc90f34ad74a0877b36` | 1 104 px, `bf38bdffb164fcc90f34ad74a0877b36` | **identical** |
| `LuMoir.alm` object markers | 7 383 px, `b341a7d35c11f66550d1b9eb98e5b259` | 7 383 px, `b341a7d35c11f66550d1b9eb98e5b259` | **identical** |
| reported counts (all three maps) | 51 / 228 / 1 815 units, 16 / 107 / 415 objects | same | **identical** |

The digest is MD5 over the ASCII lines `"x,y\n"` of the marker-coloured pixels in raster order; the
serialization is stated because these are new measurements with a stated method, not reproductions of an
earlier probe's numbers. Terrain moved under every marker; not one marker pixel moved with it. That is
the falsifiable form of "markers are independent of terrain", and it is the property P-3 and R-3 were
really about.

### Corpus placement census

An independent probe (run outside the repo, using the landed reader) over **all 38 shipped maps** — the
10 standalone plus the 28 embedded in `scenario.res`:

- **8 094 units** total, matching `ALM-UNIT-018`'s corpus figure exactly.
- **0 off-map unit anchors.**
- **0 cells holding both a unit and an object.**
- **0 cells holding two or more units.**

## Limitations and criteria not claimed

Three things are **not** claimed as passed. Each is recorded here rather than folded into a green
summary.

- **SC-11 / AC-6's live-window half — PENDING, not run.** No human opened `mapview -objects -units` at a
  window and panned/zoomed. Automated evidence covers the transform (SC-6, against the camera contract at
  five positions) and the draw order (SC-10), and the PNG path is verified on real maps above — but
  **whether markers visually hold their terrain cells through live pan and zoom has not been observed**,
  and no conclusion here rests on it. This mirrors 0007's AC-8 fidelity half and 0008's AC-6 live half,
  both of which are likewise still pending.
- **AC-6's stacking half — NOT EXERCISABLE on real data.** No map in the 38-map corpus places a unit and
  an object on the same cell, so no real render can show a unit marker over a coincident object marker.
  The property is carried entirely by synthetic evidence: SC-4 (both draw orders at the pixel level) and
  SC-10 (the pass order). It is **not** marked passed on the strength of a real render, and the
  corpus-wide zero above is why.
- **AC-6's count half evidences a successful decode, not a correct count.** `alm.Open` rejects any file
  whose type-6 payload is not exactly `70 × Meta.Count6`, so `len(Map.Units)`, `Meta.Count6` and
  `payloadSize / 70` are the same number by construction for every file that opens at all — their
  agreement carries no information about correctness. **No external oracle (the Map Editor's or the
  game's own unit count) was consulted.** The placement result above is what gives the counts their
  credibility, not their internal agreement. *(An earlier draft of the plan proposed `almtool`'s
  `payloadSize / 70` as an independent cross-check; it is not independent, and that mitigation was
  withdrawn at the plan gate.)*

Two smaller notes, neither a defect:

- An off-map anchor allocates **0**; an in-map anchor allocates exactly **1** regardless of coordinate
  magnitude. P-2 requires magnitude-independence, not zero allocation, so this satisfies it.
- At `-scale 2` the unit marker's arm thickness is even and the strip therefore sits one pixel toward the
  low side of the centre pixel (R-6). This is the exact FR-6 contract, pinned by SC-2, not a rounding
  defect.

## Conclusion

Every automated criterion (SC-1…SC-8, SC-10) passes, with the acceptance tests authored from the spec in
contexts that had not seen the implementation, and with a mutation check confirming they can fail. The
developer-run placement half (SC-9) passes on three real maps covering 2 094 units with zero misses, at
both the native scale and the even-thickness `-scale 2` — re-measured after 0007's RED revision with
every figure unchanged.

The disabled-overlay identity was originally evidenced by a byte-identical comparison against the
pre-story binary. **That comparison is no longer repeatable** — the pre-story binary embeds 0007's
superseded relief gradient — and it is recorded as such rather than reproduced. In its place: the
`-units`-absent PNG is byte-identical to a render-tier oracle (SC-5, automated and unaffected), and
across 0007's arithmetic change every terrain digest moved while every marker pixel count and marker
coordinate digest stayed identical on all three maps. That is the independence claim in falsifiable
form.

**The story is complete except for SC-11**, the live-window pan/zoom observation, which is declared
pending above and is the only criterion whose evidence is absent. AC-6 is therefore **partially
satisfied**: its placement and count halves are evidenced, its stacking half is carried synthetically
because the corpus cannot exercise it, and its live half is outstanding.
