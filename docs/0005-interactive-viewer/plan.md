# Plan — 0005 interactive terrain map viewer

## Shape of the work

0005 adds no format decoding. It puts a camera and a window in front of the 0004 pipeline:

```
cmd/mapview  ──wires──►  pkg/formats/{res,alm}   (archive + map bytes)
     │                   pkg/render/terrain      (tileset + tile-word mapping, 0004)
     │                   pkg/game                (asset-root resolution)
     └──runs───────────►  pkg/ui                 (Ebitengine loop)
                              └── pkg/render/camera  (pure camera math)
```

The split is driven by FR-5: **asset loading and camera math must be unit-testable without opening a
window.** So all camera arithmetic lives in a package that imports neither Ebitengine nor any `formats`
package, and the Ebitengine loop is a thin shell over it.

## Package placement (DAG)

| Package | Imports | Why |
|---|---|---|
| `pkg/render/camera` | stdlib only | Pure camera model. No engine, no formats — so AC-1…AC-4 and P-1/P-2 are plain unit tests. Sits under the render tier because it is render-side view math. |
| `pkg/ui` | `pkg/render` (incl. `render/terrain`, `render/camera`) + Ebitengine | Already in the DAG as `pkg/ui → pkg/render`. Owns the `ebiten.Game` implementation, the input→camera intent mapping, and the sub-cell → `*ebiten.Image` cache. |
| `cmd/mapview` | `pkg/ui`, `pkg/render/terrain`, `pkg/formats/{res,alm}`, `pkg/game` | The standalone viewer executable. cmd tier may wire any `pkg/*`. |

`internal/archtest` is fail-closed, so all three get registered, and `externalAllowed` must be widened:
today it permits `golang.org/x/text` **only** for `pkg/formats/`. Ebitengine is permitted only for
`pkg/ui` and the cmd tier — the formats and sim tiers stay engine-free.

## Camera model (ours — not a claim about the original engine)

State: `Zoom float64` (1.0 ⇒ 32 px tiles), and `X, Y float64` = the **world-pixel** coordinate of the
view's top-left corner. World extent is `cols*32 × rows*32` world px; the view covers
`viewW/Zoom × viewH/Zoom` world px.

- **Clamp** (P-1, AC-1/AC-2): per axis, if the world is smaller than the view (`world*Zoom <= view`) the
  axis is **centered** (`X = (world - view/Zoom)/2`, which goes negative — that is the centering);
  otherwise `X` is clamped to `[0, world - view/Zoom]`. Applied after every pan and zoom, so no sequence
  of operations can escape the bounds.
- **Zoom about cursor** (AC-3): the world point under the cursor is `wx = X + sx/Zoom`. After changing
  zoom, re-solve `X' = wx - sx/Zoom'` so that point stays under the cursor, then clamp. Zoom is clamped
  to `[ZoomMin, ZoomMax]` first, so the invariant holds even at the limits.
- **Visible tile range** (AC-4): `col0 = floor(X/32)`, `col1 = ceil((X + viewW/Zoom)/32)`, clipped to
  `[0, cols]`; same for rows. Half-open `[col0, col1)`. Clipping only ever shrinks the range, so P-2
  (never index a tile out of range) follows.
- **Pan** is a world-space delta so edge-scroll speed is zoom-independent (the spec's stated UX).

## Rendering path (FR-2, FR-4)

Per frame, for each visible cell: `terrain.Resolve(word)` → slot + sub-cell → `Tileset.Slot(i).SubCell(k)`
(0004, unchanged). A `map[key]*ebiten.Image` cache converts each distinct `*image.RGBA` sub-cell to a GPU
image once; cells whose strip slot is absent draw a solid `terrain.PlaceholderColor` fill (FR-4 — never a
crash). Water draws its phase-0 base cell and terrain is unshaded: animation is 0006 and lighting is 0007,
both out of scope here.

## Test strategy

- AC-1…AC-4, P-1/P-2 — table tests on `pkg/render/camera`, no window, plus a randomized pan/zoom sequence
  asserting the clamp invariant never breaks.
- AC-5 — `cmd/mapview -check` over a **synthetic** `res` archive + synthetic `.alm` built in test code,
  asserting the summary line and exit 0 with no window.
- AC-6 — manual, against a lawful install; recorded in verification.md. Not automated.

Ebitengine is only ever constructed inside the run loop, so `go test ./...` never opens a window or needs
a GPU.

## Known limitation (not a spec deviation)

`pkg/formats/alm` currently rejects a zero-length section, which the research documents as legitimate
(`0 ⟺ type8 empty`, ALM-META-025). 8 of the 10 shipped maps hit this, so the viewer can presently open
only the 2 that decode. That is a 0003 defect tracked separately — 0005 neither causes nor works around
it, and the viewer's own error path surfaces it cleanly before the window opens (FR-5).
