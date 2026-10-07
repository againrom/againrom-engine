# Plan — the deterministic walking skeleton

## Baseline

`pkg/sim` and `pkg/mapload` hold a `doc.go` each and no test file, so `pkg/mapload` has no test
surface to inherit and both docs still promise a future package. `internal/archtest` parses with
`parser.ImportsOnly`, skips `internal/` and nested modules, holds `pkg/sim` to an empty
intra-module allow-list in sources and tests alike, and permits any stdlib import. `alm.Open`
returns `*alm.Map`, whose `Units` carry `X`/`Y` as `uint32` and `Width`/`Height` as `int`;
`internal/synth` builds byte streams from stdlib alone and holds no expected value. `AGENTS.md`,
`docs/ARCHITECTURE.md`, `pkg/sim/doc.go` and one `dag_test.go` case name all defer the wall's
behavioural half to a later story.

**Frames and units.** Positions are whole map cells, `X` the column and `Y` the row on `alm`'s
row-major lattice from `(0,0)`; a tick is an index, not a duration; the byte form is little-endian.
Only an `alm.Unit`'s `X`/`Y` is in 1/256 cell, converted once by the loader.

## Design decisions

### DD-1 — a sorted slice of value entities behind readers that cannot write

`EntityID` is a `uint32`; `Entity` is `{ID, X, Y, TargetX, TargetY, HasTarget}`, plain integers and
one bool; `Bounds` is `{Width, Height int32}`. `World` keeps `tick`, an `rng`, `bounds` and
`entities`, all unexported, so the type has no writer to find.
`NewWorld(seed uint64, b Bounds, ents []Entity) (*World, error)` **copies** the slice given and
sorts it by ascending id; duplicate ids are its one error, so ids are unique and ascending in every
world that exists. Readers are methods — `Tick`, `Bounds`, `Entities`, `Hash`, `MarshalBinary` —
and `Entities` returns a fresh slice of a pointer-free value type, so a caller cannot reach back. A step finds an
entity by binary search over that slice: **no map appears in a world or on any path touching one**,
Go randomising map iteration order per process being a hazard the import ban misses.

A cleared target leaves no residue — `NewWorld` and the step both zero `TargetX`/`TargetY` wherever
`HasTarget` is false — so P-4 holds of the byte form and the digest.

`UnmarshalBinary` replaces a whole world from its canonical form and is `MarshalBinary`'s inverse,
not a field writer; what FR-2 buys is that no exported call sets a position, tick, bounds or RNG
*individually*. Rejected: an exported `AddEntity`/`SetTarget`, a second write path FR-2 forbids; a
`map[EntityID]Entity` store, nondeterministic to iterate and a second ordering to keep true.

### DD-2 — the command, and the step's three phases

`Command` is `{Entity EntityID; X, Y int32}` — the move-to order and the only one. No kind
discriminator: nothing would switch on one, and a second kind is a field added then.
`Step(w *World, cmds []Command)` is package-level, spelling the wall's own `Step(state, commands)`
form as the two documents and `Run`/`Replay` do; readers stay methods, so the only package-level functions
taking a `*World` are the three that advance one. It reaches a named entity by binary search over
the sorted slice, a miss being ignored, walks that same slice for FR-3's move phase, and moves an
axis by the three-valued integer sign of `target - pos`. No bounds value is read there.

Rejected: an interface-typed command — not a serializable value without a registry, and it puts a
pointer in the frame log; interleaving apply and move per entity — "the later command wins" would
then depend on entity order instead of slice order.

### DD-3 — the canonical form: 29-byte header, 21-byte entity record

| off | width | field |
|---|---|---|
| 0 | 1 | format version, `1` |
| 1 | 8 | `tick` `uint64` |
| 9 | 8 | RNG state `uint64` |
| 17 | 4+4 | `bounds.Width`, `bounds.Height` `int32` |
| 25 | 4 | entity count `uint32` |
| 29+21·i | 4+4+4+4+4+1 | `ID` `uint32`; `X`, `Y`, `TargetX`, `TargetY` `int32`; `HasTarget` `0`/`1` |

Positions and bounds share one width and signedness so a later clamp compares like with like, and
32 bits hold any cell a shifted `uint32` position names; the tick is 64-bit, an index that never
wraps. Presence is a byte of its own because FR-4 lets a target name any cell, so no coordinate
value is free to stand for "none".

Length is checked by division — `len(b) >= 29`, `(len(b)-29) % 21 == 0`, `count == (len(b)-29)/21`
— so no declared count can overflow a multiplication, and truncated and over-long are one
comparison, hence `==`. FR-7's refusals are joined by one more, a presence byte outside `{0,1}`,
which keeps the encoding injective so that pinned bytes mean exactly one world. `UnmarshalBinary`
decodes into a local world and assigns it over the receiver only after every check, so no refusal
leaves a partial decode; `MarshalBinary` and `Hash` both call one unexported `encode`, which cannot
fail. Rejected: `gob` or reflection, whose shape moves whenever a field does — what a version byte
exists to prevent; a sentinel target for "none", which makes one cell unnameable, against FR-4.

### DD-4 — the digest is FNV-1a 64 over those bytes

`Hash()` is `hash/fnv`'s 64-bit FNV-1a over exactly the bytes `encode` produces, so FR-6 holds by
construction rather than by a second traversal that could drift from the first. FNV-1a is a fixed
published function, so a pinned digest is recomputable from the pinned bytes by anyone holding no
code of ours — the two pins check each other instead of both falling out of one run. Rejected:
`hash/maphash`, whose seed is randomised per process and so is not reproducible across runs; a
hand-written field mixer, a second traversal where a field can enter one and not the other.

### DD-5 — the RNG is SplitMix64, and it is ours

The world owns one `rng{state uint64}` with `next() uint64`: `state += 0x9E3779B97F4A7C15`, then
the standard xor-shift-multiply finalizer over that value. `NewWorld` sets `state = seed`, so the
state follows from the seed alone and eight bytes carry it whole.

**This generator is our own engineering choice and is not claimed to be the original's.** Nothing
here recovers the game's RNG, no claim describes it, and the value pinned for it is ours — a later
reader must not read a pinned golden value as game-derived. Nothing consumes it (FR-5), so `next`
is exercised by its own test alone. Rejected: `math/rand`, banned by FR-10 and process-global
besides; xorshift64\*, where state zero is a fixed point and a zero seed a silent trap.

### DD-6 — the log records the pre-step tick

`Frame` is `{Tick uint64; Commands []Command}` and `Log` is `[]Frame`, so a log's length *is* its
frame count. `Run(w *World, schedule [][]Command, ticks int) Log` steps `ticks` times, step `i`
taking `schedule[i]` where `i < len(schedule)` and no commands otherwise. `Frame.Tick` is the tick
**before** that step — the tick those commands were applied at — so frame tick and world tick are
one quantity when `Replay` compares them. `schedule` is indexed from the run's first step, not by
absolute tick; for a world at tick 0 — every world the loader builds — the two coincide, and
continuing an interrupted run is expressed by slicing. `Run` copies each tick's commands into its
frame, so mutating the schedule afterwards cannot change what the log says happened.
`Replay(w *World, log Log) error` returns an error *before* stepping whenever
`frame.Tick != w.Tick()`; frames already applied stay applied. Rejected: recording the post-step
tick, which would compare a frame against a tick the world no longer has.

### DD-7 — the source scan lives in `internal/archtest`

A pure evaluator `CheckSimDeterminism(files map[string]string) []Violation` parses source **text**;
a loader `LoadSimSources(root string) (map[string]string, error)` supplies `pkg/sim`'s non-test
`.go` files. It reports an import of `os`, `time` or `math/rand`, matched on the path and anything
under it so `math/rand/v2` is no escape, and any `float32`/`float64`/`complex64`/`complex128`
identifier or `token.FLOAT`/`token.IMAG` literal — complex being float storage under another name,
the scan is stricter than FR-10's letter and never weaker. Its stated limit: a float reached
through another package's untyped constant (`math.Pi`) is invisible to a syntactic scan. The
live-tree test fails when the loader returns no file, so a scan that stopped finding sources reads
as a failure.

Rejected: the same check as a test inside `pkg/sim`. `go/parser` there breaches nothing, but such a
test could only run against the real `pkg/sim`, which is clean, so it could never show it can fail:
a violating file cannot be added without breaking the gate. The split lets synthetic sources drive
the negative case, as the DAG check's table already does.

`dag_test.go`'s case *"stdlib in sim is not a DAG violation (behavioural wall deferred)"* is
renamed to drop a parenthetical this story makes false, keeping its imports and empty `wantEdge`
since the DAG evaluator does permit stdlib: a test name contradicting the shipped contract is how
the next reader is misled. `AGENTS.md`, `docs/ARCHITECTURE.md` and `pkg/sim/doc.go` stop deferring
the behavioural half in the commit that creates the check, so the tree never documents a wall it
does not have.

### DD-8 — the transform and its fixture

`pkg/mapload` gains `Seed`, a fixed exported `uint64` constant, and
`FromALM(m *alm.Map) *sim.World`, which builds FR-11's entities as `ID = uint32(i)`,
`X = int32(u.X >> 8)`, `Y = int32(u.Y >> 8)`, targetless. Ids are slice indices and so unique,
making `NewWorld`'s one error unreachable here — `FromALM` discards it and returns a world alone.

The fixture is an `alm.Map` **literal built in the test**: `internal/synth` imports the standard
library alone, so a decoded-map builder there would make it import a tier package for one literal.
Expected cells are hand-written literals beside the fixture, never `u.X >> 8` recomputed — that
asserts the loader's arithmetic against itself. It is built **twice**: `alm.Map` holds slices, so a
post-step comparison against a struct copy would compare shared backing arrays with themselves and
could not fail. Rejected: `synth.ALM` bytes through `alm.Open` — it routes a known position through
a decoder to recover the position it was written from, a dependency whose failure would read as a
loader defect.

### DD-9 — internal tests, each built so it can fail

`pkg/sim`'s tests are in `package sim`: FR-2 leaves a world no exported writer while AC-6 needs one
field changed at a time, so an external test could express it only through a setter FR-2 forbids.
Against FR-2's negative half going stale, the test pins `*World`'s exported method set through
`reflect`: a method added later fails it until the sweep covers that method too, where a hand-kept
list stops being complete in silence. The byte form is checked by two derivations: field by field
at offsets written out from DD-3's table, and whole against its pin — the pin alone being a change
detector, honest as one, while the offsets and the per-field digest sensitivity say the encoding is
right.

## Success criteria

- **SC-1** `NewWorld` copies the slice it was given and refuses duplicate ids; mutating what
`Entities` returned leaves the world equal; no reader moves tick, entities, bounds or RNG state;
`*World`'s exported method set equals the pinned list (FR-1, FR-2, AC-2, P-5).
- **SC-2** AC-1 and AC-2 hold as written, the arrival tick `max(|dx|,|dy|)` being computed in the
test and nowhere in the step (FR-3, FR-4, P-1).
- **SC-3** One seed gives one sequence and a different seed a different one; the state after k draws
depends on seed and k alone (FR-5).
- **SC-4** Every field sits at DD-3's offset and width; AC-3, AC-5, AC-6, AC-9 and AC-12 hold as
written, with AC-5's schedule placing at least one command after `k` so its continuation is not
vacuous, and a reached target encoding as one never set (FR-6, FR-7, P-2, P-4, P-6).
- **SC-5** AC-4, AC-10 and AC-11 hold as written, replay running from a second world built with the
run's own arguments rather than from a struct copy (FR-8, FR-9, P-1, P-3, P-7).
- **SC-6** The evaluator names a violation for `os`, `time`, `math/rand`, a path nested
under one of them, a float type and a float literal, and none for a clean source; AC-7 holds on the
live tree and the check fails on an empty file set; the renamed DAG case still expects none; no
document still defers the behavioural half (FR-10).
- **SC-7** AC-8 holds over a fixture whose expected cells are literals, the post-step comparison
running against an independently built second map, and the DAG check still shows `pkg/sim`
importing no `againrom` package (FR-11, P-5).
- **SC-8** The suite runs green with no game install, no window and no wall-clock read, and
`pkg/sim`'s test imports stay within the standard library (FR-12).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1, FR-2 | DD-1, DD-9 | SC-1 |
| FR-3, FR-4 | DD-2 | SC-2 |
| FR-5 | DD-5 | SC-3 |
| FR-6 | DD-4 | SC-4 |
| FR-7 | DD-3 | SC-4 |
| FR-8, FR-9 | DD-6 | SC-5 |
| FR-10 | DD-7 | SC-6 |
| FR-11 | DD-8 | SC-7 |
| FR-12 | DD-9 | SC-8 |
