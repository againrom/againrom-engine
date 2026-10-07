# Analysis — selecting a group and ordering it from the running game

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile names UI work at this tier; no watcher tool exists, so it is discipline |
| Terrain — the cell-range answer, the screen rectangle, the outline pass, the gesture latch | **greenfield**: none of it exists in any form |
| Terrain — the camera inverse, the pick, the tap detector, the selection state, the highlight pass, the shared camera step, the map arm's order call | **brownfield**: every one is shipped and drawn from today, and each is touched here |

## The baseline describes a build this repo does not contain

The old clean-room spec for this story names a pick function, a geometry, an input field, a gate and a
scene that are not here under any spelling. Each was re-derived against the tree:

| The baseline says | What is actually here |
|---|---|
| the marquee reuses the `pickUnitAt` geometry | there is **no `pickUnitAt`**, and no sprite-rectangle pick anywhere. The pick is `decide` (`pkg/ui/command.go:92`), which resolves one cell through `Camera.ScreenToCell` (`pkg/render/camera/camera.go:210`) and hits by **cell equality** (`command.go:101`), lowest id on a tie (`:104`) |
| the sprite rect, or a `32x32` cell rect when undrawable | an entity whose art does not resolve draws a filled square of side **10** at native scale (`pkg/render/terrain/overlay.go:114`, `:239`). 32 is `CellSize` (`camera.go:14`), the cell, not the glyph. The sprite's own rect is `StaticPlacement.Rect()` (`terrain/statics.go:198`), used only to draw and cull (`pkg/ui/statics.go:102`), and it carries `Cell` and **not `ID`** |
| edge-scroll has an existing `!in.Left` guard | true in substance, wrong in name: there is no `Left` field. `panIntent` returns before its edge-scroll term when `in.PrimaryDown` (`pkg/ui/viewer.go:647`) |
| the marquee is gated behind `g.sim != nil` | no such field, and no scene holds a sim. The sim-less path is a **separate binary**, `cmd/mapview`, driving `ui.Viewer` directly |
| `pkg/game.MapScene` | not in this module. Already recorded three times — `docs/0013/analysis.md:34`, `docs/0015/analysis.md:19`, `docs/0020/analysis.md:15` |
| a group order is `N` `sim.MoveTo(id, cell)` calls | no such symbol. A command is `sim.Command{Entity, X, Y}` and `Step` applies a whole slice (`pkg/sim/step.go:16,95`); the front-end seam carries one order per call (`pkg/ui/flow.go:55`) |

## The gate the baseline assumes does not exist, and it is the load-bearing one

`Viewer.step` is `cmd/mapview`'s path as well as the map screen's (`pkg/ui/viewer.go:582`, reached
from `Viewer.Update:568` and from the front-end's map arm), so taking panning off the plain drag
*there* takes it off the standalone viewer too. What keeps that viewer selection-less today is not a
runtime test at all: `Viewer.command` is **unexported** (`command.go:144`) and lives in the same
package as its one caller. So the separation this story needs has to be built, and the mechanism is a
decision rather than a lookup someone forgot.

## AC-5 as the baseline states it is false against this tree

The baseline asserts that after a group order every unit reaches the target or a cell adjacent to it,
the surplus stopping "on the nearest reachable free cells". Measured against the landed code, it does
neither. `enterable` is the occupancy predicate and the wave expands only into enterable neighbours
(`pkg/sim/route.go:230`), so a route exists only while the target cell is **itself free**. Once the
first unit stands there every other search fails; the failing arm raises `e.Stall` and touches nothing
else, and at `stallLimit = 16` (`pkg/sim/world.go:54`) it calls `e.clearTarget()`
(`pkg/sim/step.go:135-142`). The surplus therefore does not approach and does not settle nearby — it
**freezes where the refusal found it** and abandons the order sixteen ticks later. The contract states
that outcome instead of the baseline's, and claims no arrival for those units.

## What we did not know while writing, and now do

Three questions the draft could not answer from itself, each decided rather than left implied: whether
the rectangle lives in screen or world coordinates (screen, resolved at the release — the two differ
visibly because keyboard pan and zoom stay live mid-drag); whether the outline is drawn before the slop
is passed (it is not, so a tap flashes nothing); and whether an id absent from a snapshot leaves the
selection (it does not — it is skipped while absent). Each is now a clause in the contract rather than
an inference a builder would have had to make.

## We searched the pinned claims for a selection or group-order machinery

Read at the submodule as checked out here, pin `ce40c15`, with `claims/retracted.md` and the registry's
standing corrections first. The ledgers hold **no** selection, marquee, drag or group-command machinery
of any kind; they do hold the original's tick-loop ordering, its cell reservation and its wait rule, and
every one of those differs from what this story fixes. What follows from that for what we are entitled
to assert is the ledger's business, not this file's.

## What we looked at

`pkg/ui/{command,viewer,overlay,statics,app,flow}.go`, `pkg/game/world.go`,
`pkg/sim/{step,route,world}.go`, `pkg/render/camera/camera.go`,
`pkg/render/terrain/{overlay,statics,units}.go`, `cmd/mapview/main.go`, `pkg/ui/command_test.go`,
`pkg/ui/selection_overlay_test.go`, `pkg/game/order_invariance_test.go`,
`docs/0028-app-unit-command/`, `docs/0029-unit-pathfinding/{spec,plan,tasks}.md`, `AGENTS.md`, both
check scripts, and in research `claims/retracted.md` and `claims/registry.md` first, then
`claims/move.md`, `claims/menu.md` and `claims/alm.md`.
