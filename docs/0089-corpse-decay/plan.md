# Plan — a body decays

## Approach

Three new canonical numbers on the entity, one new pass at the end of a tick, one new selection in
the render tier, and one column carried through the two definition loaders. Nothing existing is
rewritten: the death transition goes where "leaving the living" is already stated once, the
occupancy change goes into the one predicate that already answers it, and the drawn fall is
untouched.

Files: `pkg/sim/{world,step,route,binary,hash}.go`, `pkg/data/{unitdef,humandef,anim,classes}.go`,
`pkg/mapload/{spawn,fromalm}.go`, `pkg/render/terrain/unitanim.go`, `pkg/game/{units,world}.go`.
**`pkg/sim/engage.go` is not touched**, and neither is the acquisition path: another lane holds it.

## Design decisions

### DD-1 — Three fields, and why not two and not one (FR-1, FR-2, FR-4)

`Decay uint8` is the stage. `Dwell uint16` is what the stage-1 body still owes before it is torn
down. `DyingTime int32` is the class's own dwell length, an inert carried stat beside `Defence` and
the other seven.

Two would not do. The stage cannot be derived from health (spec C-1), and the dwell cannot be
derived from anything: it is a countdown whose length is per class. One could not do either — a
single field would have to encode "counting down" and "which rung" in one number, and the two ranges
would then have to be kept apart by arithmetic instead of by types.

`DyingTime` is carried on the entity rather than looked up at the moment of death because `pkg/sim`
holds no class table and may not gain one: it is stdlib-only and knows a class id as an opaque key
it never interprets. The number therefore has to arrive with the entity, exactly as the seven combat
numbers do.

The widths are the values' own. The stage has five stored values; the dwell is a tick count that a
`uint16` bounds at 65535, which at this package's tick is longer than any mission; the dying time is
an `int32` because that is what the definition column is, negative included.

### DD-2 — The death transition goes in `clearFelled` (FR-2)

`clearFelled` is already the one place that says what leaving the living costs — the order, the
crossing, the group term, the attack — and it already has the three callers a blow can arrive
through. The stage, the defence halving and the dwell go there, guarded on the stage being zero so
that a second blow on a body changes nothing.

The alternative was the new pass. It was rejected because the pass runs at the end of a tick while
`clearFelled` runs in phase 1 as well as phase 3: a unit felled by a command in phase 1 would then
spend that tick not alive and at stage zero, and the occupancy seed at the top of the move loop
reads exactly that state. There would be one tick in which a body is neither living nor dwelling.

The halving is `>>= 1` on the signed field, which is the shift the decode names rather than a
division: the two agree on every non-negative defence and this is the one that is right at the
boundary. It fires once, on the transition, so a body's defence cannot be halved twice.

### DD-3 — One pass, at the end of Step, over the whole slice (FR-4, FR-5, FR-6)

It is also the whole of FR-11: the pass reads the world and the tick index and nothing else — no
clock, no map, no float — so determinism is a property of this shape rather than a mechanism beside
it. `decayPass` runs after phase 3 and before the tick is incremented, so it sees post-move, post-blow
positions and health, and so a unit felled anywhere in the tick is already at stage 1 when it runs.
It is its own loop for the reason phase 3 is: nothing in it routes and nothing reads the occupancy
scratch, which by then is no longer read at all.

Per entity, in ascending id: a living one is skipped. A dwelling one has its dwell decremented, and
if that reaches zero and the entity is not a ground mover its health is pinned. Once the dwell is
zero, the walk applies on its own tick phase and the ladder is then evaluated from health. Ascending
id is the priority everywhere else in this package and it costs nothing here — no entity's decay
reads another's — but it is kept so that the one walk order in this file is the one walk order in
every other.

### DD-4 — The walk's period is `2 x scriptCycle` and its phase is the engine's own (FR-4)

`decayCycle = 2 * scriptCycle` and `decayPhase = 12`, both spelled once beside the script's two, so
the three cannot come to disagree about how long a full tick is. The engine filters its full-tick
counter's low bit and dispatches the pass on sub-tick phase 12; the low bit selects one of the two
full ticks in a period and which one is not published, so `12` rather than `28` is ours — and the
choice can only move the tick index a rung is reached on, never the ladder.

The health step saturates: `HP > minHP` is tested before the decrement, so the representation floor
holds and nothing wraps a body back to life.

### DD-5 — Occupancy is one clause in `counted` (FR-3)

`counted` already answers "does this entity stand in its layer's plane" for its three readers. The
not-alive arm becomes `Decay == decayFallen && Dwell > 0` — the body dwells, so it blocks — and
every other not-alive entity answers false, which is what the dead arm already did.

The downed rule goes with it, and that is the point rather than a casualty: a unit at exactly zero
health is the engine's death trigger too, so it now dwells and then decays like everything else
instead of blocking a doorway for the rest of the mission. Nothing else in the package tests
`Downed()`; the predicate stays, because the three life states remain pairwise exclusive and jointly
total and the script's own reading of "dead" is written over `Alive()`.

### DD-6 — Removal is one compaction at the end of the pass (FR-5)

An entity that reaches stage 5 is not deleted where it is found: the pass marks it, and one
compaction afterwards rebuilds `entities` and `routes` in lockstep, keeping ascending id. Deleting
in place would invalidate the index the loop is walking, and it is the kind of defect that survives
every test that removes one entity and fails only when two go at once.

Two things go with the record. Its route slot, because the two slices are parallel by construction.
And every **attack order naming it**, on the entities that remain — which is the constructor's
second pass done again for the same reason it exists there: an order pointing at nothing is a shape
the byte form refuses, so a tick that produced one would build a world this package cannot marshal
and read back. It is a second walk of the slice and it runs only when something was removed.

Nothing else is swept. The group word and the owner slot are membership, not references, and a
script reference resolves by id every time it is read.

### DD-7 — The constructor normalises, the decoder refuses (FR-1, FR-10)

The relation the package is already in. `newWorld` folds a positive stage on a living entity to zero
and a zero stage on a not-alive one to stage 1 with its own dwell; `UnmarshalBinary` refuses both,
and both refuse a stage above 4 outright, on the trade the movement domain and the routing mode
already make — a value normalised on the way in would map two byte forms onto one world.

The constructor's normalisation is a **pairing** fix and not a death: it writes the stage and the
dwell and touches no combat number, so building a world holding a corpse cannot halve a defence that
was already halved when that world was cut.

`Dwell` is cleared on every entity not at stage 1, so residue cannot survive a construction — the
same rule the target coordinates and the transit pair already take.

### DD-8 — Version 17 puts the three at the record's tail (FR-10)

The entity record grows from 92 bytes to 99: `Decay` at +92, `Dwell` at +93 as a little-endian
`uint16`, `DyingTime` at +95 as a little-endian `int32`. At the tail as every block since the health
pair has gone, because putting them anywhere earlier moves every offset after them and buys nothing.

The version moves on all three grounds this package has ever used: the record changes **width**, so
a version-16 buffer read against these offsets misparses every record after the first; the stage is
canonical state that enters the **digest**, so a form carrying none decodes to a world whose digest
is not the digest of the world it was cut from; and there is a **reader** — a version-16 form says
every body in it is freshly fallen, which is a claim, not a gap, and a save cut twenty seconds after
a battle contradicts it.

The digest follows the encoding, as it always has: `hash.go` digests the marshalled bytes, so the
three fields enter it by being encoded and there is no second list to keep level.

### DD-9 — The dying-time column, on both definition loaders (FR-2)

The unit loader already reaches slot 33 and drops it with a note saying the corpse's dwell is
fetched from that same slot elsewhere. It now stores it, and the note goes with the dropping.

The human loader stops at slot 22 and the column is at 23, so `lastHumanSlot` moves to 23. That is a
widening of what a row must carry, and it is the one change in this story that a shipped table could
refuse: a row of exactly 23 parameters would now be rejected by name. It is checked rather than
assumed — the mission drive loads both roots' real tables, so a short row fails loudly there, and if
one does the column is dropped for that collection and the default stands, disclosed.

The default is the constructor's, 8, and it is written **once** in `unitCtorDefaults`, which the
human loader already seeds itself from. An absent cell leaves it standing by the slot cursor's own
rule, so "defaults to 8 when the column is absent" needs no clause of its own.

### DD-10 — `BoneSlot` and `SelectBoneFrame` (FR-7, FR-8)

`pkg/data`'s block arithmetic already computes the bone base; `BonePhases` entered only the
predicted total. It becomes `BoneSlot` beside `DyingSlot`, clamped by the same `phaseCount`, and the
render tier's mirror carries it value for value like every other field of that descriptor.

`SelectBoneFrame(a, frameCount, oct, stage)` is `SelectDeathFrame`'s shape over the bone block:
`TailBase + slot*BoneSlot + (stage - 2)`, with `unitSlot` supplying the slot and the mirror so the
direction rule is shared rather than copied, and with the same three-return refusal so a caller can
fall through. It gates on `BoneSlot > 0` and on `stage >= 2`, and it guards the index against the
sheet's own count last, as its two siblings do.

It is a **fourth** selection rather than an argument on the third. The two answer different blocks
off different clocks — one a run clock, one a stage — and a merged function would take an argument
that means nothing on one of its two arms, which is what the one-selection-per-drawn-state rule in
that file exists to prevent.

### DD-11 — The seam asks the stage first (FR-9)

In `pkg/game`'s snapshot the not-alive arm tries the bone selection when the entity's stage is at or
above 2, then the fall selection, then falls through to what it drew before. Three attempts down one
chain, each refusing rather than answering a frame the caller cannot tell from a real one, so the
completeness property is a property of the chain and not of any one link.

The stage crosses the seam as the simulation's own byte; the render tier is handed a number and
never asks what a corpse is. That is what keeps `ANIM-DEATH-007`'s finding true of this tree too:
nothing on the drawing side advances anything.

## Risks

- **R-1 — the drive moves.** A body now blocks for its dwell where it blocked for nothing, and a
  downed one stops blocking for ever. Both reach routing and therefore the mission run. It is
  measured on both roots before and after rather than predicted.
- **R-2 — a shipped human row is too short for slot 23.** DD-9 names the check and the fallback.
- **R-3 — removal reaching a script.** A finished body is no longer resolvable by id, so a check
  naming it measures nothing. Twenty minutes of game time away at the shipped cadence, disclosed in
  the spec, and the outcome counters are latched.

## Success criteria

- SC-1 — Every AC of `spec.md` has evidence, and P-1 to P-5 hold over the schedules that drive them.
- SC-2 — `go build`, `go vet`, `gofmt`, `go test ./...`, the asset gate, the doc budget and the SDD
  audit are green by exit code, and the deletion set against `e1a8df5` is empty.
- SC-3 — The mission drive is measured on both roots at `e1a8df5` and at this branch's head, and any
  movement is explained by a rule this story states.
