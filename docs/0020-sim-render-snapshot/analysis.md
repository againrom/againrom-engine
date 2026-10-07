# Analysis — a running world under the map screen

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — this fixes the determinism wall's read side, the contract every later entity-render story reuses; no watcher tool exists, so it is discipline |
| Terrain — `pkg/game`, `pkg/ui`, `pkg/render/terrain` | **brownfield**: a shipped load path, a shipped tick loop and a shipped marker family, all with behaviour to preserve |
| Terrain — the world's lifecycle and the entity layer | **greenfield**: nothing in the program has ever owned a world or drawn one |

## The scene the baseline describes is not in this tree

`cleandocs/0020-sim-render-snapshot/spec.md` names a surface that does not exist here, and the
mismatch is not cosmetic — its FR-2…FR-5 are written against it throughout. Checked by `grep` over
the module: **`MapScene`, `PickerScene`, `openrom`, `sim.Schedule`, `sim.Unit`, `AttachSim`,
`SampleHeight8p8` and `buildObjectPlacements` occur nowhere.** What is here instead:

- the game front-end is `pkg/game`'s `FrontEnd`, and a map becomes a picture through the single
  `LoadMapViewer`, which returns a `MapView` — the `*ui.Viewer` that draws it and the decoded
  `*alm.Map` behind it;
- the tick loop is `ui.App`'s, dispatching per screen, the map screen forwarding to the viewer's
  own camera step;
- the entity type is `sim.Entity`, `Run` takes `[][]Command`, and `Step` is the package-level
  `sim.Step(w, cmds)` that 0019 chose deliberately;
- height comes from `terrain.Projection` — `AnchorHeight(col,row)` and `MinV` — and the marker
  transform already consuming it is `Viewer.overlayScreenRects`.

`World` already has `Tick()` and `Bounds()`. The baseline's FR-1 is therefore two thirds landed
before the story opens, and its third part is the question below.

## The wall the baseline would have walked through

`internal/archtest`'s allow map gives `"pkg/ui": {"pkg/render", "pkg/render/"}`. **The package that
opens a window may not import `pkg/sim`**, and the render sub-packages it *can* reach (`terrain`,
`camera`, `frame`, `menu`) carry empty allow-lists of their own, so there is no route to a simulation
type through them either. A viewer holding a `*sim.World` costs a widened allow map — the same kind
of edit 0019 refused when the baseline asked `pkg/sim` to import `pkg/formats/alm`.

The tier that may see both is `pkg/game`, and it already does this job for the format tier:
`MarkerCells(m *alm.Map)` converts placement records to integer cells and pushes them through
`SetUnits`, because "the viewer never sees a map type, which is what keeps the format tier out of the
UI tier" (`pkg/ui/overlay.go`). Applying the identical rule to the simulation tier costs no DAG edit,
keeps Ebitengine out of every package that can touch a world, and leaves the conversion in the one
place allowed to know both halves.

## Does the world need a second read path?

`Entities()` returns a fresh slice per call — an allocation the renderer would make once a tick. An
appending twin beside it (`AppendEntities(dst)`) would remove that, and costs more than it looks:

- `pkg/sim/world_test.go` pins `*World`'s exported method set *and sweeps every exported method for
  mutation*, and the sweep is **fail-closed on arity** — `if m.Type.NumIn() != 1 { … "takes
  arguments and is not in worldWriters" }`. A reader taking a destination slice can only pass it by
  being declared a **writer**, which it is not. The instrument guarding FR-2's negative half would
  have to be weakened to admit a method needing no exemption.
- Two read paths with identical semantics must then agree forever, in the one package whose exported
  surface is deliberately small enough to pin.

Against an allocation that is not measured and is not out of line with what this draw path already
does: `staticMarkerScreenRects` builds one `[]image.Point` **per frame** over a map that may hold
thousands of placements, and `overlayScreenRects` appends to a nil slice per frame beside it. The
entity conversion runs once per *tick*, not per frame. So: no new call, and the pin stays as 0019
left it — it turns red the day someone adds one, which is what it is for.

## Where the entity layer goes in the draw order

The baseline draws the sim layer last, topmost. This tree has a rule that forbids it as written: the
three diagnostic glyphs are a nested family (object r6/t3 ⊃ unit r4/t1 ⊃ static r3/t1), and the draw
order is load-bearing because a later pass that is *not* a strict subset does not degrade a
coincident marker, it **erases** it (0009 R-4, 0017 DD-4). A filled square drawn last would erase the
cyan unit cross at exactly the cell where the two most need comparing — tick 0, where they coincide.

The tree already answers this for the other layer that is content rather than instrument: static
object art is drawn **before** the overlays, so "no sprite can hide the instrument measuring it".
Entities are content by the same test, so they belong in the same place, and the nested-cross
invariant is left untouched rather than extended.

## The two things 0019 left open

- **No real map has ever been through `FromALM`.** This story is the first that would put one
  through — every map the owner opens builds a world. Taken up rather than deferred, as the
  agreement between an entity's cell and its unit marker's cell: both are a `>>8` of the same
  record, derived independently, sharing only the cell-to-screen transform, so a disagreement shows
  on a real map instead of being argued about.
- **`mapload.Seed`'s literal is unpinned.** Considered and left alone. 0019's reason — a pin in
  `pkg/mapload` would fail a legitimate `pkg/sim` encoding change in a package that did not change —
  is untouched here, and the generator is still never consumed, so the seed reaches nothing on
  screen. Making every played world come from that constant does not make the constant more
  observable.

## What we looked at

`internal/archtest/{dag.go,determinism.go}`, `pkg/sim/{world.go,step.go,run.go,world_test.go}`,
`pkg/mapload/fromalm.go`, `pkg/game/{mapload.go,frontend.go}`,
`pkg/ui/{viewer.go,overlay.go,app.go,flow.go}`, `pkg/render/terrain/overlay.go`,
`cmd/againrom/main.go`, `docs/0019-walking-skeleton/` (spec and verification), and in research
`claims/retracted.md` before `claims/alm.md`.

Open and disclosed rather than guessed: how fast the original moves a unit and how long its tick is
(nothing here fixes either — one step per front-end tick is ours, and it is a placeholder); how the
original draws a unit at all; and the original's own marker rounding, which no research covers and
about which nothing here asserts anything in either direction.
