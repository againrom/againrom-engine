# Plan — arming a start view at load, applying it at layout

## Approach

Three layers, each owning exactly the part of the answer it can see.

`pkg/render/camera` gains one mutator — put a world point at the centre of the view — because that is
camera arithmetic and the package exists to hold it away from a window. `pkg/ui` gains the authored
extent, a one-shot arming call, and the application of both when a view size is adopted, because it
is the tier that knows the map's geometry and the view's size. `pkg/game` arms it from the mission's
own start report, because it is the tier that holds both the mission and the viewer.

The application is deferred rather than done at arming time, and that is the plan's load-bearing
decision (DD-3).

## Facts verified during planning

1. `camera.New` sets `Zoom: 1`, leaves `X`/`Y` zero and calls `Clamp`, which holds a position at 0 on
   any axis whose world exceeds the view. A fresh camera is the map's top-left corner at native
   scale.
2. The constructor has exactly **one** non-test call site, `pkg/ui/viewer.go`'s viewer construction.
   No composite literal of the type exists anywhere in the tree, so construction is enumerated rather
   than sampled.
3. Outside `pkg/render/camera` the only non-test mutations are `SetWorldHeight` (viewer construction
   and the flat/displaced flip), `Pan` and `ZoomAbout` — the last two on the input path alone.
4. `NewViewerWithStatics` takes terrain, a tileset and two class bundles. No world and no entities
   reach it; the camera is built with `DefaultWindowW`/`DefaultWindowH`.
5. `Viewer.Layout` is the sole writer of `cam.ViewW`/`ViewH`, and it re-clamps but does not rescale.
   `App.Layout` forwards to it; `App.syncViewerLayout` exists because the engine calls `Layout` only
   when the window changes.
6. `App.OpenMission` runs **before** `ebiten.RunGame`, so `a.winW`/`a.winH` are still zero when it
   calls `syncViewerLayout`, and `Viewer.Layout` ignores a non-positive size. The first real size
   therefore arrives from the engine's own first `Layout` call.
7. `game.MissionOpener` holds `ms.Start` and `mv.Viewer` in one scope, and is reached from exactly one
   place, `cmd/againrom`'s startup.
8. `mapload.Start` carries `Cells` (per member, `Cells[0]` documented as always `Drop`), `IDs`, and
   `Drop` — decided whether or not anyone stands there.
9. `pkg/ui` already computes a cell's displaced world centre in `pathCellCentre`; the same lift
   expression appears in `placeArm` and in the overlay's shift, and the package's own comments treat a
   fourth copy as the defect to avoid.
10. Zoom is continuous, clamped to `[ZoomMin, ZoomMax]` = `[0.125, 8]`; the wheel applies a
    multiplicative step. There are no discrete levels to disturb.
11. `pkg/render/menu` ships `FrameW`/`FrameH` = 640/480 as the original's fixed frame, and
    `camera.CellSize` = `terrain.CellSize` = 32.

## Design decisions

**DD-1 — the centring is a camera mutator, not a caller's arithmetic.** `Camera.CenterOn(wx, wy)`
sets the position so the given world point is at the view's centre, then `Clamp`s. It goes in the
camera package because the view extent in world units is derived from the view size and the zoom, and
the method that does that (`viewWorld`) is unexported — a caller outside would have to re-derive it,
which is a second expression of the same rule. Ending in `Clamp` is not a nicety: every mutator in
that package ends in `Clamp`, and that is what P-1 rests on. *(spec FR-1, FR-8; P-1)*

**DD-2 — the extent is a column count, not a zoom.** `AuthoredStartColumns() int` returns the number
of map columns the view spans at mission start. A zoom would be meaningless without a view size and
therefore not a property of the original at all, while a tile count is exactly the form the open
research question is asked in — so a measurement replaces the value without reshaping the seam. The
zoom is derived at application time as `ViewW / (columns * CellSize)`. *(spec FR-3, FR-4; P-3)*

**DD-3 — the start view is armed at load and applied at layout.** `Viewer.SetStartView(cell)` records
a pending cell; `Viewer.Layout` applies and clears it after adopting the new view size. This is forced
by fact 6: the mission is opened before any real window size exists, so a zoom computed at arming time
would be computed against `DefaultWindowW` and would be wrong on every other window. Applying inside
`Layout` also makes it work identically for a mission opened at startup and for a viewer handed to a
running application. *(spec FR-5, FR-8; AC-8)*

**DD-4 — it is a one-shot, cleared on application.** The pending cell is an `image.Point` behind a
`bool`, cleared the first time it is applied. `Layout` runs every frame, so anything not cleared would
be a follow camera — the behaviour FR-6 exists to forbid and the out-of-scope list names. *(spec FR-6;
P-2; AC-9)*

**DD-5 — the world point comes from the existing cell-centre arithmetic, extracted.** `pathCellCentre`
is split into `cellWorldCentre(cell) (float64, float64)` — the flat centre plus the displaced lift —
and the camera transform that already followed it. The new path calls the world half. Nothing about
either caller's behaviour changes, and the tree does not gain a fourth copy of
`-AnchorHeight(cell) - MinV`. *(spec AC-5)*

**DD-6 — the anchor is chosen in `pkg/game`, from the start report alone.** `MissionOpener` computes
`Start.Cells[0]` when the party is non-empty and `Start.Drop` otherwise, and hands the result to
`SetStartView`. The rule lives at the door rather than in `pkg/ui` because `pkg/ui` must not learn
what a party is — FR-7 requires a viewer with no start cell to be unchanged, and the cleanest way to
guarantee that is for the viewer to know only a cell. *(spec FR-2, FR-7; AC-3, AC-4)*

**DD-7 — the authored number is 20, and its derivation is written at the site.** 640 (the original's
frame width, which this tree already ships as `menu.FrameW` and which the menu ledger establishes from
two independent asset dimensions and a placement-table consumer) divided by 32 (`camera.CellSize`, the
tileset's own cell). It is disclosed at the site as an **upper bound**: the original's battle screen
also carried a panel of undecoded geometry, so its map area showed at most this and certainly less. A
narrower value would be a guess wearing the same clothes; the bound is the largest claim the evidence
supports, and it is the one being made. *(spec FR-3, and the Divergence section)*

**DD-8 — the tests derive their expectations from `AuthoredStartColumns()`, never from `20`.** A test
that hard-codes the literal would have to be edited when research overrules the value, which would
make the seam cost two edits instead of one and would quietly punish replacing it. *(spec P-3)*

**DD-9 — the zoom is set through `SetZoom`, so the existing range wins.** A demanded zoom outside
`[ZoomMin, ZoomMax]` is clamped by the setter and the centring then runs at the clamped value, which
is exactly FR-8's last clause. No new range and no new rule. *(spec FR-8)*

## Files to touch

| File | Change |
|---|---|
| `pkg/render/camera/camera.go` | `CenterOn` |
| `pkg/render/camera/camera_test.go` | its unit tests |
| `pkg/ui/viewer.go` | `AuthoredStartColumns`, `SetStartView`, the pending fields, the apply in `Layout` |
| `pkg/ui/path.go` | `cellWorldCentre` extracted out of `pathCellCentre` |
| `pkg/ui/startview_test.go` | new: arming, application, one-shot, extent, displaced anchor, untouched viewer |
| `pkg/game/frontend.go` | the anchor rule and the `SetStartView` call in `MissionOpener` |
| `pkg/game/startview_test.go` | new: the anchor rule over a start report |

No file outside these is edited, and `pkg/sim` is not among them.

## Risks

- **The apply site runs every frame.** If the one-shot flag were not cleared, the view would fight the
  player on every frame — the failure would be immediate and total rather than subtle. DD-4 clears it,
  and a test drives `Layout` repeatedly after a pan to pin it.
- **`Layout` is also the resize path.** Re-arming on resize would satisfy a naive reading of FR-5 and
  break FR-6. The flag is set only by `SetStartView`, never by `Layout`.
- **A displaced map's centre is not the flat lattice's.** Getting this wrong would centre the view a
  whole relief's worth above or below the party on a mountainous map, and would look almost right on a
  flat one — so it is tested on a grid with a real altitude spread, not only on a flat fixture.
- **The extent depends on the view's width alone.** On a very wide, short window the authored column
  count could put the party's row off screen at the top or bottom of a short map. The existing clamp
  centres such an axis (FR-8, AC-11), so the outcome is defined; it is a consequence of DD-2 and is
  accepted rather than special-cased.

## Success criteria

- **SC-1** — a mission opened on a large campaign map shows the party at the centre of the view rather
  than a corner of empty ground, and the party is visible in the first frame without panning. (FR-1,
  FR-2)
- **SC-2** — the authored column count is the only value stating the extent: changing it in its one
  place changes what every opened mission shows, and no test needs editing to follow it. (FR-3, FR-4,
  P-3)
- **SC-3** — a map opened through the picker and a map opened by the standalone viewer are unchanged
  in position and scale. (FR-7, P-2)
- **SC-4** — the headless mission drive reaches the same outcome on the same tick as at the fork
  point, and the byte form's version, bytes and digest are unmoved. (FR-9)
- **SC-5 — the owner's, and no test can close it.** Whether *that much* of the map is the right amount
  to see at mission start is a judgement about a look. The number is authored, disclosed and behind
  one seam precisely so that his answer — or a later measurement of the original's viewport — is one
  edit. No criterion above judges it, and none can: a test can only confirm the view spans the value
  we chose, which is a tautology about our own choice. (FR-3, and the spec's Divergence section)
- **SC-6** — `go build`, `go vet`, `gofmt`, `go test -count=1 -trimpath ./...`, the asset scan and its
  history sweep, the doc budget and the SDD audit all pass, and the deletion set against the fork
  point is empty.
