# Plan — dim the non-playable map border

## Where the change has to live, and why that is not a choice

The tier that draws is granted the render packages and nothing else. That grant is a pinned entry with
a test of its own asserting it is *exactly* those two prefixes, so widening it to reach the derivation
is not an option that quietly exists — it fails a landed criterion by name. The simulation publishes no
reader for the plane it holds either: its world exposes a tick, bounds, entities and a digest, and the
grid is unexported with no accessor. So neither the width nor the world's copy of the answer is
reachable from the draw path, and the design follows from that rather than from taste.

- **DD-1 — the answer travels as data, on the map's own layer set.** The render tier's cell-layer type
  already carries the tile words plus two optional layers — altitudes and the object grid — each riding
  because it is a fact about the *map* rather than about a viewer's configuration, each set through a
  keyed literal so a caller that omits it is unaffected, and each validated by its own reader rather
  than by the type. The block plane joins them under exactly that convention. It differs from its two
  neighbours in one way worth writing down: it is **derived**, not decoded, and that is precisely why it
  has to travel — the rule that derives it lives where the drawing tier may not look.

  The rejected alternative is a second depth constant at the render tier. It would work, it would be
  three lines, and it would be a second source of truth for one decoded literal. Those drift; that is
  the whole reason the plane exists.

- **DD-2 — the load path is where the derivation is called.** One function turns map bytes into a
  running viewer, and both front-ends come through it. It may import the derivation, it already holds
  the decoded map, and it already fills the other two optional layers there. So the margin the viewer
  dims and the margin the simulation refuses come from one function called over one decoded map — not
  two rules kept in agreement. The plane is set unconditionally, as the object grid is, so what a viewer
  sees depends on the map rather than on what the caller remembered to ask for.

  What this does **not** buy is a read of the world's own bytes: the derivation runs twice over the same
  input. That is a weaker property than "the same slice", and the contract states the weaker one.

- **DD-3 — per tile, and the step is the deliverable.** The dim is decided once per cell and applied to
  all four of its corners, so the boundary is a hard brightness step between adjacent quads. A
  per-*vertex* rule would have removed the step for nothing — adjacent quads evaluate a shared vertex
  identically, so the seam would close by construction — and it was still rejected: it ramps the dim
  across the innermost margin cell, which blurs the one line the picture exists to draw. The quads carry
  their own four vertices and are not shared, so a step renders exactly as written, with no gap and no
  interpolation across the boundary to fight.

- **DD-4 — multiply, never replace.** The dim scales the corner multipliers the lighting produced. A
  margin cell therefore keeps its own relief, one shade darker, rather than flattening to a single
  value — which matters because the margin is where a map's edge cliffs usually are, and a flat fill
  there reads as a missing layer rather than as a boundary. It also makes FR-4's bound trivial instead
  of conditional: a positive constant times a positive multiplier is smaller than the multiplier and
  still positive, with no case analysis over what the light did first.

- **DD-5 — the placeholder is exempt, and it is not this story's call.** A cell resolving to an absent
  tileset slot draws a solid diagnostic fill, and a landed criterion of the lighting story pins that
  fill at full brightness with a live test asserting it. Dimming it would revise another story's
  contract from inside this one. It is also the right answer on its own terms: the fill is a loud
  magenta whose job is to be noticed, and this story has no interest in making it quieter.

  The cost is one real asymmetry — the corner-scale source has three answers and the dim is applied at
  two of them, so a future arm added to that function could miss it. The alternative was to restructure
  the function so the dim has a single application point, which would have meant moving a check whose
  present position carries its own stated correctness reason. Three criteria cover the three arms
  instead, which is what makes the asymmetry checkable rather than merely documented.

- **DD-6 — the bit is respelled at the reading tier; the width is not.** The reading tier names bit 1
  for itself, because the DAG puts a shared constant out of reach and a bare `2` at a read site would
  leave the bit's meaning written down nowhere on that side of the seam — the same reason, and the same
  shape, as the derivation tier already respells it against the simulation. What is duplicated is a bit
  position in a byte layout, which is what a tier boundary costs. The **width** is duplicated nowhere,
  and P-3 is the assertion that this stayed true.

- **DD-7 — the census asks the reader, it does not read the plane.** The margin count walks the cells
  and calls the FR-1 predicate, rather than popcounting the plane. A census with its own reading of the
  bytes would agree with a broken predicate; this one cannot disagree with the draw path because it is
  the draw path's own question. It costs a `W*H` walk on a call nothing makes per frame.

## Task shape

Three entries, ascending, each green on its own and each landing in one tier:

1. the layer and the predicate, at the render tier — nothing consumes it yet;
2. the derivation call in the load path, plus the count accessor its criterion needs to observe
   anything at all;
3. the dim itself.

The count accessor lands with (2) rather than (3) because (2) is the entry whose criteria need to see a
plane arrive; splitting it out would leave (2) with a wiring change and no way to witness it.

## Success criteria

- **SC-1** — The full gate is green from a clean tree: build, vet, the whole test suite with
  `-trimpath -count=1`, the asset guard, the doc budget, the SDD audit, and `gofmt` over our own files.
  The package count is recorded so the suite's extent is a number rather than a claim.
- **SC-2** — Mutation kills, one per design decision that a passing test could otherwise be blind to:
  the bit the predicate masks (DD-6), the predicate's totality arms (DD-1), the dim's multiply turned
  into a replace (DD-4), the dim applied to the placeholder arm (DD-5), the derivation call dropped from
  the load path (DD-2), and the census reading the plane directly (DD-7). Each mutant is applied to the
  line the entry that owns it actually wrote, run, and reverted.
- **SC-3** — Every criterion the tree held before this story still holds: the lighting story's corner
  scales, its placeholder rule and the front-end summary composition are unmoved, which the suite
  asserts as it stood rather than as this story rewrote it. Nothing in the pre-existing suite is edited.
- **SC-4** — The import-graph check passes with the load path's new edge to the derivation and with **no
  new edge out of the drawing tier**, so the grant that shaped this design is still exactly what it was.
- **SC-5** — The diff touches no file under the simulation, the decoded formats, the digest or the byte
  form, and adds no float, clock or generator anywhere near them.

## Traceability

| FR | Decided by | Witnessed by |
|---|---|---|
| FR-1 | DD-1, DD-6 | AC-1, AC-2 · SC-2 |
| FR-2 | DD-2 | AC-8 · SC-2, SC-4 |
| FR-3 | DD-3, DD-5 | AC-3, AC-4, AC-5, AC-6, AC-7 · SC-2 |
| FR-4 | DD-4 | AC-3 · SC-2 |
| FR-5 | DD-7 | AC-8, AC-9 · SC-2 |
| FR-6 | — | SC-3, SC-4, SC-5 |
| P-1 | DD-3, DD-4 | AC-3, AC-5, AC-7 |
| P-2 | — | SC-5 |
| P-3 | DD-1, DD-6 | AC-8 · SC-2 |
