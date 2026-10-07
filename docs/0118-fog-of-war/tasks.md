# Tasks — 0118-fog-of-war

**Reading key.** `FR-n`, `D-n` → `spec.md` §The contract, §Divergences. `AC-n` →
`spec.md` §Acceptance. `DD-n`, `R-n`, `SC-n` → `plan.md`.

Tasks are ordered, and each leaves the standing gate green on its own, with no game
install present.

---

## T1 — one predicate, two readers *(implementation)*

**Boundary.** The march becomes parameterised and gains an exported fog reader.
Nothing draws anything and no state is added.

**Files**

- `pkg/sim/sight.go` — `MODIFY`: `buildSightTables(shift int32)`; `sightSeed(sight256,
  shift int32) int32` replacing `sightBudget`; a `sightReader` value carrying tables,
  shift and inset; `aiSight`/`fogSight`; `marchSight`/`marchCell`/`groupSight` take a
  reader; exported `func (w *World) Sight(owner uint32) []byte`.
- `pkg/sim/sight_test.go` — `MODIFY`/`ADD`.

**Covers** FR-1, FR-2, FR-3 · DD-1, DD-2, DD-4, DD-5 · R-2 (AC-2, AC-3, AC-4, AC-5,
SC-3, SC-4)

**Notes**

- The fog inset is computed from the world bounds: a ring cell is admitted when its
  absolute column is in `7 .. W-8` and its row in `7 .. H-8`. The AI inset keeps the
  existing `blockAir` bit test **unchanged** — do not recompute it.
- `Sight(owner)` unions one march per **living** entity whose `Owner` equals `owner`,
  each at that entity's own `ScanRange`, through `fogSight`.
- Assert the seed identity over radius 0..20 and shift 1..8; assert the AI stamp on a
  fixed world is byte-identical to a golden built before the change.
- No float identifier or literal, no `os`/`time`/`math/rand` — `internal/archtest`
  scans this package's source.

---

## T2 — the plane, the shroud, the gates, the reveal *(implementation)*

**Boundary.** `pkg/ui` holds a fog plane and draws by it. Nothing builds one yet; with
none pushed, every frame is byte-identical to today.

**Files**

- `pkg/ui/fog.go` — `ADD`: `FogUnseen/FogExplored/FogVisible` byte consts;
  `SetFog(plane []byte, cols, rows int)`; `fogAt(col, row int) uint8`;
  `fogScale(state uint8) float32` = 0, 0.5, 1; `SetFogReveal`, `ToggleFogReveal`,
  `FogRevealed`.
- `pkg/ui/viewer.go` — `MODIFY`: fields; compose `fogScale` onto all four values
  `cornerScales` returns, where they are produced.
- `pkg/ui/statics.go`, `pkg/ui/overlay.go` — `MODIFY`: the gates.
- `pkg/ui/app.go` — `MODIFY`: `appInput.Reveal` from `KeyF4`, beside `in.Readout`.
- `pkg/ui/fog_test.go` — `ADD`.

**Covers** FR-4, FR-6, FR-7, FR-9 · D-5 · DD-6, DD-7, DD-8, DD-9, DD-10 · R-1, R-4,
R-5 (AC-8, AC-9, AC-11, SC-5)

**Notes**

- `fogAt` is the ONLY read of the plane: it applies the reveal, answers `FogVisible`
  when no plane is pushed (so existing callers are unaffected), and `FogUnseen` off
  the map. Bounds-check against the plane's own cols/rows.
- Gates: an entity is drawn when `Owner == localOwner`, else only at `FogVisible`; a
  sack only at `FogVisible`; structures and statics at anything but `FogUnseen` (D-5).
- Fix, do not delete, any test that asserted the pre-fog drawn set.

---

## T3 — a mission opens closed *(implementation)*

**Boundary.** The plane is built from the world, accumulates, and reaches the viewer.
This is the task the owner can see.

**Files**

- `pkg/game/fog.go` — `ADD`: `fogPeriod = 32`; a `fogPlane` type holding the explored
  and visible layers with the map's dimensions; `refresh(w *sim.World, owner uint32)`.
- `pkg/game/world.go` — `MODIFY`: a `fog` field on `mapWorld`; build it where the
  world and viewer are joined; refresh it in `tick()` every `fogPeriod` ticks and once
  at construction; `mw.view.SetFog(...)` inside `push()`.
- `pkg/game/fog_test.go` — `ADD`.

**Covers** FR-5, FR-10, FR-11 · D-6 · DD-3, DD-13, DD-14 (AC-6, AC-7)

**Notes**

- `refresh` calls `w.Sight(owner)`, REPLACES the visible layer with the result, and
  ORs it into the explored layer. Explored is never cleared. The owner is
  `sim.SelfSlot`, the same slot already pushed via `SetLocalOwner`.
- Every cell starts unseen; the first refresh happens before the first push, so tick 0
  already shows the party's surroundings and nothing else.
- Push the plane every `push()`, not only on a refresh — the viewer keeps no copy
  policy of its own.
- Nothing is added to `sim.World`. The byte-form version must not change; if a version
  test fails, that is a defect in this task, not a reason to bump.

---

## T4 — the minimap *(implementation)*

**Boundary.** A panel that repeats the terrain and shows what has been scouted. It is
drawn and inert.

**Files**

- `pkg/ui/minimap.go` — `ADD`: `minimapBox` and the colours; a pure
  `composeMinimap(terrain []color.RGBA, cols, rows int, fog func(col, row int) uint8,
  units []minimapMark, box image.Point) *image.RGBA`; `SetMinimap`, `ToggleMinimap`,
  `MinimapShown`, and the cached terrain-colour build.
- `pkg/ui/viewer.go` — `MODIFY`: draw it in `Draw`, after the readout.
- `pkg/ui/app.go` — `MODIFY`: `appInput.Minimap` from `ebiten.KeyM`.
- `pkg/ui/minimap_test.go` — `ADD`.

**Covers** FR-8 · D-4 · DD-11, DD-12 (AC-10, AC-12, SC-6)

**Notes**

- Terrain colour per cell is sampled ONCE from the tileset through the viewer's
  existing cell resolution and kept; the map does not change. Scale is the largest
  whole-pixel scale that fits `minimapBox`, at least 1; a larger map is sampled.
- Fog: unseen draws black, explored the terrain colour halved, visible it whole.
  Units: `localOwner`'s in one colour, others in another, and only those the gates
  in T2 would draw.
- Frame and place it with the existing panel helpers, in a corner the readout and
  the panel do not use.
- **No click surface.** Nothing in `app.go` or `command.go` may route a cursor
  position into it (AC-12).
