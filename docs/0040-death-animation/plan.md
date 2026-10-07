# Plan — one length, one link, one selection, one memory

## Approach

Four additions, each in the tier that already owns the thing it extends, and one dispatch that
joins them.

The descriptor gains the **length of the dying block's per-direction slot** — the only quantity
the sheet arithmetic already computes with and never named (FR-1). The unit bundle gains a
**corpse link**, resolved once at load from the class id the dying key holds, so the draw path
never resolves anything (FR-2). The unit layer gains a **second selection** beside the one it
has, pure and total, owning its own bounds guard exactly as the live one does (FR-3). The map
screen's driver gains a **third per-entity memory** beside the facing and step memories it
already keeps, written in the same walk (FR-4).

The dispatch is one `if` in the push: an entity the simulation reports not alive takes the
death selection, and falling out of it takes the live one (FR-5, FR-6). Nothing downstream of
the seam changes — a corpse crosses as art, frame and mirror bit like any other sprite, and
every placement, anchor, cull and texture path receives what it always received.

Two of the contract's requirements need no code and are pinned rather than built: a body keeps
its facing because the simulation does not advance it and the facing memory is only written
from a step (FR-7), and nothing here touches `pkg/sim` (FR-8).

## Baseline

The push resolves an entity's class, classifies it moving from the step it took, remembers the
octant of that step per id, and calls the layer's one selection at the scene clock offset by
the entity's id. A frame comes back already proved against the sheet's own count; art, frame
and mirror cross the seam; the window tier places it. A class with no art, and an entity whose
class id names none, cross with no frame and draw the square.

The descriptor derives the six blocks' bases from the class's phase scalars, each clamped at
zero so an absent phase contributes no block, and it names the move and idle slot lengths but
not the dying one. The dying base is computed and read by nothing.

`pkg/sim` gives a dead entity no advance, no order and no occupancy, never removes it, never
frees its id, and never changes a health once it is at or below zero. The seam already carries
the life state, and the health bar, the selection filter and the order filter already read it.

## Files to touch

`pkg/data/anim.go` and its test — the slot length. `pkg/render/terrain/unitanim.go` and a new
test — the mirrored field and the death selection. `pkg/render/terrain/units.go` — the corpse
link on the bundle's class type. `pkg/game/units.go` and a new test — the load-time
resolution. `pkg/game/world.go` and a new test — the memory and the dispatch.
`pkg/game/drawn_invariance_test.go` — the death-bearing stream. Nothing under `pkg/sim`,
`pkg/ui`, `pkg/mapload` or `cmd/` is opened.

## Design decisions

### DD-1 — one new length on the descriptor, and no new gate beside it

`DyingSlot` joins `MoveSlot` and `IdleSlot`, derived from the same clamped scalar by the same
arithmetic, in both the data tier's descriptor and the render tier's mirror of it (FR-1).

**No `DyingOK` is added.** `MoveOK` and `IdleOK` exist because each combines two facts — a
positive phase count *and* a non-empty track — and the dying block has no track: the engine
indexes it by the run's own clock with no modulus. A boolean equal to `DyingSlot > 0` would be
a second copy of one comparison, free to disagree with it. The selection tests the length.

The mirror stays a field-for-field copy performed by the loader, so the render tier still
re-derives nothing from a registry.

### DD-2 — the corpse is a resolved pointer on the bundle, not an id resolved at draw time

`UnitClass` gains `Corpse *UnitClass`, filled by the loader in a second pass once every class
is in the map (FR-2). The draw path follows a pointer and performs no lookup.

Carrying the dying **id** across and looking it up per draw was rejected twice over: it puts a
registry key in a tier whose whole rule is that it holds none, and it makes a per-frame map
lookup out of a fact fixed at load.

**One hop, and the miss is nil.** The resolution is a single subscript, which is what the
engine's own corpse arm performs; a class naming itself therefore points at itself, and a class
naming one the bundle does not hold points at nothing. A chain walk with a visited set was
rejected: no evidence asks for one, and its cycle branch is unreachable by construction.

`Corpse` is filled for **every** loaded class, frameless ones included, because it is a fact
about the class and not about any sheet — the same rule the descriptor is filled under.

### DD-3 — a second selection beside the live one, and the dispatch is the caller's

`SelectDeathFrame` sits beside `SelectUnitFrame` and shares its direction rule and its guard
shape (FR-3). It is a separate entry point rather than a fourth arm of the existing switch: the live
selection's inputs are the moving flag and the effective tick, this one's are the ticks since
death, and one function taking both would carry two inputs never both meaningful.

What that costs is the invariant "this layer has one selection path", and the replacement is
stated here: **there is one selection path per life state, and one place that chooses between
them** — the push, the only site holding both the life state and the class. Two call sites would
be two rules to keep in step.

The signature answers three things where the live one answers two: frame, mirror, and whether
there is a frame at all. The live selection answers a refused index as sheet frame 0 unmirrored — the picture drawn
before it existed — the right total answer there and the wrong one here, where a refusal must
reach the fallback (FR-6).

### DD-4 — one expression covers the fall and the frozen frame

    phase = min(elapsed / 2, DyingSlot - 1)

The engine plays a run of `2 x DyingPhases` ticks whose drawn frame is its clock halved, and
then a corpse fork that freezes on the block's last frame. Both are that one clamp: inside the
run the division is the engine's, and past it the clamp is the freeze. Written as two branches
they would be two places for the boundary frame to be off by one.

The divisor `2` is a **named constant with the decoded reading beside it**, not a tuning knob
(C-2). The phase is clamped below as well as above, so a negative tick count — which no caller
can produce — is the first frame rather than an index before the block.

### DD-5 — the death clock is a third memory, written in the walk that already writes one

`mapWorld` gains `died map[sim.EntityID]int`, alongside `facing` and `prev`, born with the map
screen and dropped with it (FR-4). It is **written where the facing memory is written** — in
the snapshot build, from the entity being pushed — and read by lookup only.

Writing it there is safe for the reason writing the facing memory there is safe, and the reason
is not the same as `prev`'s: the step memory is a **delta** memory, so a build that wrote it
would consume its own input and answer differently the second time. This one is **set once**.
A build that finds an entry leaves it, so a snapshot built twice with no advance between gives
the same elapsed count both times, and the push stays the repeatable read it is (P-2).

`prev` is written before the step, in the tick, because it must hold the pre-step cell. This
memory must hold the tick death was **first observed**, which is only knowable after the step,
so the same placement would be wrong for it by exactly one tick.

**Nothing is pruned and nothing is cleared.** The simulation never drops an entity and has no
heal, so an entry can neither be orphaned while the map is open nor be wrong to keep: an entity
that returned to life would read its own live selection and never consult the entry. The
memories beside it keep entries on the same terms.

### DD-6 — the elapsed count carries no per-entity offset

The live selection is asked at `scene + id`, a deliberate cosmetic de-sync so a crowd of one
class does not march in step. The death selection is asked at `scene - died[id]` **alone**: a
fall must begin at its own first frame, and an offset would start each corpse partway through
its own animation and, for a large id, past the end of it (FR-4).

### DD-7 — the fallback is a fall-through, not a second decision

The death path writes art, frame and mirror and returns; if it produces no frame it writes
nothing and control reaches the live path unchanged (FR-6). So "never vanish" is a property of
where one `return` sits rather than of a rule someone maintains, and the three refusals — no
corpse link, a corpse class with no frames, an index the sheet cannot hold — reach the same
place without being enumerated at the call site.

An entity whose own class has no art still draws the square, exactly as before, because that
decision is downstream of both selections and is not touched.

### DD-8 — the seam does not widen

No field is added to what crosses the seam. A corpse crosses as `Art`, `Frame` and `Mirror` —
with `Art` being the **corpse class** — so the window tier learns nothing new and every glyph
path, cull and texture key behaves as it did (FR-5). The life state already crosses and already
has three readers; none of them changes.

## Risks

| # | Risk | Mitigation |
|---|---|---|
| **R-1** | The corpse class's canvas differs from the dying unit's, so substituting the whole class moves the sprite's anchor as well as its sheet. | It is the intended reading and it is disclosed as ours rather than as the game's. The anchor path is untouched, so if the other reading is later shown correct the change is one field of one struct. |
| **R-2** | A class whose corpse resolves to a sheet with too few frames draws nothing of the death, silently. | The selection refuses the index and the fallback keeps the unit drawn (FR-6), and that path is a criterion rather than a hope (SC-6). |
| **R-3** | The death path is only reachable when something dies, so a defect in it is invisible in every test that never kills. | The witnesses drive kills and damage explicitly, and the invariance stream contains both (SC-8). |
| **R-4** | A body lying in one frame forever reads as a bug to a player who expects it to decay. | Disclosed in the contract, and the decoded mechanism it awaits is recorded in the evidence ledger rather than approximated. |

## Success criteria

- **SC-1** — Over hand-written classes, including ones declaring no dying phase, the derived
  slot is the declared count clamped at zero and `DyingBase + D x DyingSlot == TailBase` holds
  in every case; the render tier's mirror carries the same value.
- **SC-2** — Over a hand-built bundle, each class's corpse link is the class its own key names:
  a sibling, itself, nothing where the bundle lacks it, and the class it names rather than the
  one beyond it where two hops exist.
- **SC-3** — For a descriptor with a known base, direction count and slot, the selection at
  ticks `0 .. 2L-1` yields each frame of that direction's slot twice, in order, and every
  answer lies inside that slot.
- **SC-4** — At every tick from `2L` to a very large count the answer is the slot's last frame,
  and never anything else.
- **SC-5** — At eight stored directions no octant mirrors and the slot is the octant; at five
  the upper octants fold and mirror; an absent dying block and a sheet shorter than the index
  both answer that there is no frame.
- **SC-6** — Driven over a hand-built world: a killed entity carries the corpse class's art and
  that class's dying frames from the tick it dies, advancing one frame per two ticks and then
  holding; a downed one does the same; and three failing shapes — no corpse link, a corpse
  class with no frames, an absent dying block — each keep the entity drawn with no death frame.
- **SC-7** — A body's drawn direction is the direction it last walked, at every tick after it
  dies, and the frame it draws is a dying frame at every one of them.
- **SC-8** — For one command stream containing kills and damage, the pinned world fields, the
  byte form and the digest at every tick index are identical whether the snapshot is built
  every tick or never; and two builds with no advance between are equal, death frames included.
- **SC-9** — Two mutants of the phase arithmetic are killed by named tests: the divisor changed
  from 2 to 1 (the fall plays twice as fast), and the upper clamp dropped (the frozen frame
  walks past the dying block into the one after it). Both applied to the line that owns the
  arithmetic, run over the whole tree, and reverted.

## Traceability

| FR | Design | Criteria |
|---|---|---|
| FR-1 | DD-1 | AC-1, SC-1 |
| FR-2 | DD-2 | AC-2, SC-2 |
| FR-3 | DD-3, DD-4 | AC-3, AC-4, AC-5, P-4, SC-3, SC-4, SC-5, SC-9 |
| FR-4 | DD-5, DD-6 | AC-6, P-2, SC-6, SC-8 |
| FR-5 | DD-3, DD-8 | AC-6, AC-7, P-3, SC-6 |
| FR-6 | DD-7 | AC-8, P-3, SC-6 |
| FR-7 | DD-5 | AC-9, SC-7 |
| FR-8 | DD-8 | AC-10, P-1, SC-8 |
