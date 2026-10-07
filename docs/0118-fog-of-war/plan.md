# Plan — 0118 fog of war

## Shape

Four tasks, in this order, each green on its own.

1. `pkg/sim` — parameterise the march and export the fog reader. No new state.
2. `pkg/ui` — the plane's home on the render side, the shroud, the drawable
   gates, the debug reveal.
3. `pkg/game` — build the plane from the world, accumulate it, push it.
4. `pkg/ui` — the minimap over the plane.

2 precedes 3 because 3 calls the API 2 declares. 4 is last because it reads
both.

## Design decisions

**DD-1 — The parameterisation is a value, not a fork (FR-1, FR-2).** A
`sightReader` value carries the three parameters: the built window tables, the
shift they were built at, and which in-bounds rectangle to test. Two
package-level values, `aiSight` and `fogSight`, are the only two that exist;
`marchSight` and `marchCell` take one. The alternative — a second march function
for the fog — is exactly what the decoded law says a consumer must not do, and
it is what produced two implementations that had to be proved equal after the
fact.

**DD-2 — `k` becomes a builder argument and the package-level tables stay.**
`buildSightTables(shift)`; `sightTables = buildSightTables(sightShift)` is
unchanged as a value. The grids are a property of the build and not of a world,
so nothing becomes per-world and nothing becomes mutable. The constant
`sightShift` remains the one seam a future registry read replaces, and both
readers name it rather than inlining 7.

**DD-3 — The explored plane is NOT in `sim.World`, and the byte-form version is
not bumped (FR-4, FR-11).** Version 28 was allocated to this story and is
returned unused. Three reasons, in order of weight: nothing in `Step` reads the
plane, so it is not simulation state and the field-set pin exists to refuse
exactly that; it is per-participant *view*, so putting it in a shared world
makes "whose view" a question the world cannot answer; and it would put one byte
per cell of player exploration into the digest the determinism readout compares.
The plane lives in `pkg/game`'s `mapWorld`, built from the world through the
exported reader — which is `0112`'s `Stock()`/`Sacks()` idiom, state read *out*
of a world rather than added to it. The cost is D-6: it does not survive a save.

**DD-4 — The seed is one function in 1/256 cell (FR-3).** `sightSeed(sight256,
shift)` returns `1<<(shift-1) + (sight256 >> (8-shift))`. The old whole-cell
form is a call with `r<<8`, and the identity `(r<<8) >> (8-k) == r<<k` is
asserted over the whole parameter square rather than argued. This is the seam
D-2 names: a sub-cell sight radius needs a wider entity field and nothing else.

**DD-5 — The fog rectangle is computed; the AI rectangle stays read off the
grid.** The AI's 8-cell inset is currently read from the grid's air-block bit,
because every map this tree loads marks that region and nothing else. The fog's
7-cell inset has no bit, so it is computed from the world bounds. Keeping the
AI arm on the bit rather than recomputing it to 8 is deliberate: changing it
would change AI answers, which FR-2 forbids, and the two derivations agreeing is
a property worth leaving testable rather than assuming.

**DD-6 — Three states in one byte, and the state is the index.** `0` unseen,
`1` explored, `2` visible — so the shroud table is a three-entry array indexed
by the byte, and an out-of-range byte cannot be produced by the only writer.
The decoded law's `01` unreachable combination has no representative here, which
is the point: a pair of bits admits a state the law says cannot happen.

**DD-7 — The shroud composes onto the existing corner scales (FR-6).**
`withScales` already multiplies each of a tile's four corners by that corner's
own scale, and both terrain paths call it. The fog factor multiplies all four
equally, applied where the scales are produced, so day/night tint, relief
shading, geometry, the texture cache key and the draw order are all untouched.
Black comes out of a factor of 0 rather than out of a second pass drawing black
over the top.

**DD-8 — The reveal is a view flag, not a plane edit (FR-9).**
`Viewer.fogReveal` short-circuits the one accessor every consumer reads the
plane through. The plane is never written by it, so AC-11's "turning it off
restores the previous drawing exactly" is true by construction rather than by a
restore path.

**DD-9 — One accessor, and every fog consumer goes through it.** `fogAt(col,
row)` answers the state for a cell, applies the reveal, and answers unseen for a
cell off the map or when no plane has been pushed. Terrain, the drawable gates
and the minimap all call it. A second reading of the plane is how the shroud and
the cull come to disagree on the frontier.

**DD-10 — With no plane pushed, nothing is fogged.** `fogAt` answers *visible*
when the plane is empty, not unseen. Every existing viewer test and every
non-mission path (the map viewer, the picker) pushes no plane, so they must draw
exactly as they did. Fog is opt-in at the seam and default-on at the mission,
which is where FR-10 actually lives.

**DD-11 — The minimap is composed into an RGBA and cached on what it depends on
(FR-8).** Terrain colour per cell is sampled once from the tileset and kept; the
fog and unit layers are redrawn when the plane's revision or the unit set
changes. The composer is a free function taking (terrain colours, plane, units,
box) so AC-10's purity is the signature, not a discipline.

**DD-12 — Keys: `F4` reveals, `M` toggles the minimap.** `F1`, `F2`, `F3` are
the readout, the cell grid and the light step, and `F3`'s own comment names that
band as the diagnostic register that reproduces nothing. `F4` joins it. The
minimap is a game affordance rather than a diagnostic, so it takes a letter, and
`M` is unbound today.

**DD-13 — The refresh period is 32 world ticks and is one constant (FR-5).** The
decoded clock is the presentation tick at period 32; our presentation clock is
driven from the world tick through `SetLightClock`, so using the world tick is
the same period at the shipped cadence and one clock fewer. `fogPeriod` is
named beside the plane.

**DD-14 — The march is not cheap and the period is what bounds it.** One march
is a 41x41 window walk per living owned entity. At period 32 with a party of a
few, that is a few thousand cell evaluations twice a second, which is nothing;
per tick with a large roster it would not be. The period is therefore load
bearing as well as faithful, and the plane is rebuilt rather than incrementally
patched because a rebuild has no staleness to reason about.

## Risks

**R-1 — Existing tests that count drawn entities go red.** FR-7 removes
drawables from the drawn set. DD-10 confines that to callers that push a plane,
so only the mission path is affected; any test that does go red is a test
asserting the pre-fog behaviour and is retargeted, never deleted.

**R-2 — The 7-cell inset changes AI answers by accident.** Mitigated by DD-5:
the AI arm keeps the bit it already reads and the fog arm gets the computed
rectangle. A test pins that the AI reader's output on a fixed world is
unchanged.

**R-3 — A minimap at one pixel per cell is unreadable on a large map, or
enormous on a small one.** The scale rule is "largest whole pixel scale that
fits the box, at least 1", and the box is a constant. A map larger than the box
at scale 1 is sampled, not averaged; a sampled minimap is still a minimap.

**R-4 — The plane and the world disagree about dimensions.** The plane crosses
the seam with its own width and height and `fogAt` bounds-checks against them,
not against the terrain grid. A mismatch draws unseen rather than panicking.

**R-5 — `pkg/ui` cannot import `pkg/sim`.** The plane crosses as `[]byte` plus
two ints, exactly as the passability plane already does. No sim type appears in
a `pkg/ui` signature.

## Success criteria

**SC-1** `go build ./...`, `go vet ./...`, `gofmt -l` and `go test -count=1
./...` are clean with no game install present.

**SC-2** `internal/archtest` passes unchanged: no float identifier or literal,
no `os`/`time`/`math/rand`, and no new import edge.

**SC-3** The AI reader's stamp on a fixed synthetic world is byte-identical
before and after T1.

**SC-4** `pkg/sim`'s world field set and byte-form version are unchanged; the
version test does not move.

**SC-5** Every fog consumer in `pkg/ui` reaches the plane through one accessor —
greppable, and asserted by a test that flips the reveal and observes all three
consumers change together.

**SC-6** Running a mission opens on a black screen with the party's surroundings
lit, `F4` opens the map, `F4` again closes it back to what was explored, and `M`
hides and shows the minimap.
