# Provenance — a running world under the map screen

Pinned at research `e53c779`, frozen for the story; `claims/retracted.md` was read before
`claims/alm.md` and nothing in it bears here. Two rows below rest on a decoded game fact, both of
them already consumed in this tree before this story; everything else is recorded as ours rather
than given a citation it does not have.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| FR-8 — an entity's cell and its unit marker's cell are both the record's `/256` anchor shifted right by 8 | `ALM-UNIT-018` (amended), `ALM-PLACE-033` | High for the shift itself — the loader stores the raw `/256` anchor and the walker converts with a bare `SAR ,8`, no origin subtracted and no inset added, over all 82 shift-by-8 sites in the binary; Medium only for the identification of that walker as *the* unit spawn routine. Already consumed twice in the tree: `terrain.AnchorCell` and `mapload.FromALM` each perform it |
| FR-1 — a world's bounds are the decoded map's width and height | 0003's container contract, carried unchanged through 0019 | High as decoded there; this story re-derives nothing about the container |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **The world lives in the tier that may see both halves, and the tier that opens a window names no simulation type** (FR-9) | Our own architecture, enforced by `internal/archtest`: `pkg/ui`'s allow-list is `pkg/render` and its sub-packages, so a viewer holding a world needs a widened allow map. The same rule already keeps the format tier out of the UI tier — the viewer takes integer cells, never an `alm` type |
| **`pkg/sim` gains no exported call** (FR-9) | Nothing decoded bears on it. `World` already answers tick, bounds and entities, and the last of those hands out copies; an appending twin would have to be declared a writer to pass a mutation sweep that is fail-closed on arity. The trade is recorded in `analysis.md` and is engineering, not fidelity |
| **One step per front-end tick** (FR-2) — *disclosed* | The original's tick rate and unit speed are both undecoded. Ground is therefore crossed at the front-end's tick rate, which is a rate nothing claims the original has. It is a placeholder and the contract says so |
| **The schedule is scripted, map-derived and finite** (FR-4) | No claim describes what the original orders a unit to do at map start; nothing decoded is being reproduced. Determinism is what the contract needs and is what it asks for |
| **A 10-pixel filled square, in a colour none of the three diagnostics uses, drawn under all three** (FR-6, FR-7) | The project's own diagnostic design, as the three marker glyphs already are (0008/0009/0017 FR-6). Placing it with the content rather than the instruments follows this tree's own rule that no content may hide the instrument measuring it |
| **An entity outside the map's extent is simply not drawn** (FR-5) | 0019 permits an entity to walk off the grid and clamps nothing; the marker family already drops an off-map cell before building geometry. Neither is a statement about the original |
| **The world is discarded when the map screen is left, and rebuilt at tick 0** (FR-1) | Ours. Nothing decoded says whether the original preserves a map's state between visits, and this story has no save path in the front-end to preserve it with |
| **`mapload.Seed` stays unpinned** | 0019 left it so deliberately and the reason is unchanged here: a pin in `pkg/mapload` would fail a legitimate `pkg/sim` encoding change in a package that did not change. The generator is still never consumed, so the seed reaches nothing on screen |

## Open / undecoded

- **How long a tick is, and how fast a unit moves.** Unchanged from 0019: nothing decoded gives
  either, and this story fixes neither — one step per front-end tick is a placeholder rate, not a
  claim.
- **How the original draws a unit.** Sprite selection, anchoring and facing are all undecoded here.
  A square is a placeholder that asserts nothing about them.
- **The original's own marker rounding.** No research covers it. Nothing in the contract asserts
  anything about it in either direction, and the absence is deliberate rather than an omission.
- **Whether the original's unit identity matches ours.** Entity ids are still the map's unit-slice
  index (0019); the record carries a unique id of its own at `+0x40` (`ALM-UNIT-048`) that
  `alm.Unit` does not expose. FR-8 compares *cells*, not identities, so nothing here rests on the
  two agreeing.

## Removed from the baseline and why

- **`MapScene`, `PickerScene.launch`, `AttachSim`, and the whole optional-sim scene lifecycle
  (FR-2, FR-3, FR-6).** No such type or seam exists in this tree; the front-end is `FrontEnd` plus
  `ui.App`, and a map becomes a picture through one loader. Re-derived against what is here.
- **`AppendUnits` on `World` (FR-1, AC-2).** `Tick` and `Bounds` already exist; the appending
  entity reader is refused, with the cost of both directions recorded in `analysis.md`.
- **`sim.Unit`, `sim.Schedule`, `world.Step(cmds)`, `sim.FromALM`, `openrom`.** The types are
  `sim.Entity` and `[][]Command`, the step is the package-level `sim.Step(w, cmds)` that 0019 chose
  so that `*World`'s method set stayed pinnable, the loader is `mapload.FromALM`, and the game is
  `cmd/againrom`.
- **`buildSimPlacements` mirroring `buildObjectPlacements` (FR-5).** Neither exists. One marker
  transform already serves three glyphs, carries the displaced lift and the exact-rect cull, and is
  unit-testable without a GPU; a fourth derivation of the same geometry would be a second thing to
  keep in agreement.
- **AC-8's "`pkg/sim` still imports `formats/alm` + stdlib".** Refused mechanically: the allow-list
  is empty, tests included. This restates the arrangement 0019 already removed.
- **The sim layer drawn last, topmost (FR-4).** Refused on this tree's own evidence: a last pass
  that is not a strict subset of the nested cross family erases a coincident marker rather than
  degrading it, and tick 0 is exactly where the entity and its unit cross coincide.
- **The provenance-basis preamble and the closing section reporting the absence of a research
  item** — both refused by the doc-budget content bans wherever they appear.
