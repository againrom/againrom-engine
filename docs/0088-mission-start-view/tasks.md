# Tasks — the mission opens where the party lands

Legend: **Kind** is `impl` (one coherent product change, one trailered commit). `Done when:` is the
entry's own exit condition. `SC-n` is `plan.md` §Success criteria; the risks are its §Risks.

## T1 a world point at the centre of the view

Kind: impl. Carries FR-8, P-1, DD-1, DD-9.
Criteria SC-4, SC-6.
Files: `pkg/render/camera/camera.go` MODIFY, `pkg/render/camera/camera_test.go` MODIFY.

Boundary: from a world point to a view position. Nothing here learns that a cell, a tile or a party
exists — the point arrives already in world pixels.

Scope fence: no existing method's behaviour, signature or doc contract changes; `Clamp`,
`clampAxis`, `zoom` and the zoom limits are read, never edited; no field is added to `Camera`; the
package gains no import.

Done when: the new mutator puts a given world point at the view's centre on both axes when the world
allows it; it ends inside the same bounds every other mutator ends inside, for a point beyond either
world edge, for a non-finite point, and for a world smaller than the view on either axis; and the
package's existing tests are unedited and pass.

## T2 the authored extent, armed once and applied when a size is adopted

Kind: impl. Carries FR-3, FR-4, FR-5, FR-6, FR-7, P-2, P-3, DD-2, DD-3, DD-4, DD-5, DD-7, DD-8.
Criteria SC-2, SC-3, SC-6.
Files: `pkg/ui/viewer.go` MODIFY, `pkg/ui/path.go` MODIFY, `pkg/ui/startview_test.go` ADD.

Boundary: from a cell handed in by a caller to a positioned, scaled view, once. The cell's meaning is
the caller's; this tier knows only that it is a cell of this map.

Scope fence: `NewViewerWithStatics` gains no parameter; `syncWorld`, `Mode`, `SetFlat` and both
overlay setters are untouched; `placeArm` keeps its own arithmetic and its cull; `pathCellCentre`'s
callers see no behaviour change; `WheelZoomStep`, the zoom limits and the pan and drag intents are
not edited; no drawing path is touched.

Done when: one exported function states the count, its doc naming it as ours with its derivation and
its upper-bound status; a viewer armed and then given a view size spans that many whole map columns
centred on the cell, the rows following the view's proportions and differing between two widths;
arming before any positive view size exists takes effect at the first one; a pan then further layouts
leaves the view where the pan put it; a viewer never armed is unmoved by any number of them; a cell
on a displaced grid is centred where it is drawn; and the tests read the count from the function.

## T3 the mission arms it from its own start report

Kind: impl. Carries FR-1, FR-2, FR-9, DD-6.
Criteria SC-1, SC-4, SC-6.
Files: `pkg/game/frontend.go` MODIFY, `pkg/game/startview_test.go` ADD.

Boundary: from a started mission's start report to the one cell the view opens on.

Scope fence: `LoadMapViewer` is not edited — the picker's path and the standalone viewer must keep
opening as they do; `MissionLine` and the headless check mode are untouched; the mission is not
started twice and the map is not decoded twice; nothing consults the map's start table; no
simulation type, byte form or digest is reached.

Done when: the anchor is the first reported member's cell for a party of one and for a party of
several, and the reported decided cell for a party of none; the rule is a pure function of the start
report, exercised without opening a window; the door hands that cell to the view and nothing else
about the door changes; and a headless drive of the tenth mission reaches the same outcome on the
same tick as at the fork point.

## Traceability

| Requirement | Tasks |
|---|---|
| FR-1 | T3 |
| FR-2 | T3 |
| FR-3 | T2 |
| FR-4 | T2 |
| FR-5 | T2 |
| FR-6 | T2 |
| FR-7 | T2 |
| FR-8 | T1 |
| FR-9 | T3 |
| P-1 | T1 |
| P-2 | T2 |
| P-3 | T2 |
