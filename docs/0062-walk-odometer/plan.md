# Plan — the walk odometer

## Decisions

- **DD-1 — the odometer is a memory of `pkg/game`'s `mapWorld`, not a field of `pkg/sim`.** It joins
  `prev`, `facing` and `died` on the struct born with a map and dropped with it: a
  one map keyed by id, holding the count and DD-3's crossing tick, looked up and never ranged.
  *Why not `pkg/sim`:* an odometer there is hashed state, reaches `MarshalBinary` and the digest,
  and buys nothing — nothing in the simulation reads a walk frame. It is the priced choice too: the
  threshold rule wants High-confidence research behind anything reaching hashed state, and the
  research says the opposite, that the engine's own count is on the client drawable and outside the
  serialized set. *Prevents:* a cosmetic quantity acquiring a determinism obligation it cannot
  meet. *Covers:* FR-1, P-3.

- **DD-2 — it is written by the advance and read by the push.** `prev`'s own rule and reason: a
  memory written from `entityDraws` consumes its own input, and the second build of one tick's
  picture differs from the first with no digest to catch it. The write goes in `tick()`, after
  `sim.Step` and before `push()`. *Prevents:* P-4 failing silently. *Covers:* FR-1, P-4.

- **DD-3 — the crossing is read from `Transit` and a counted tick, and never from `TransitTotal`.**
  The delta is the entity's cell minus `prev[id]`, under `prev`'s own presence test, and it is
  nonzero for the whole crossing because 0056 freezes `prev` for its duration. The tick index is
  **counted, not derived**: `recordCells` already tests `Transit > 0` before the advance, which is
  exactly "this tick continues a crossing", so the same test zeroes or increments a per-entity tick
  and the span is then `tick + 1 + Transit`. *Why not the obvious `TransitTotal - 1 - Transit`:*
  that total **outlives the transit it measured** — `pkg/sim` writes the pair only for a rated
  mover, so an entity that loses its speed keeps the old total while crossing a cell per tick, and
  the derived index would read the last tick of a 22-tick crossing and credit a whole cell with
  twelve sub-cell units. `Transit` is zero whenever nothing is owed and cannot go stale that way.
  `ui.MapEntity` is untouched: the frame is already chosen in `pkg/game`, so the odometer never
  reaches `pkg/ui`. *Covers:* FR-1, FR-3, FR-5.

- **DD-4 — the per-tick share is the engine's own recurrence, in closed form.** The engine takes
  `remaining / ticksRemaining` per tick, truncating, and subtracts it; over `n` ticks that splits a
  cell's 256 sub-cell units into `n - (256 mod n)` shares of `256/n` followed by `256 mod n` shares
  of `256/n + 1`. The closed form is used because it is O(1) and carries no remaining-delta state —
  but it is **proved rather than believed**: SC-1 re-executes the recurrence itself for every `n`
  in `1..256` and requires the two to agree share for share. The split is taken on the delta's
  **magnitude**: written for a signed delta the formula is not sign-safe under truncation toward
  zero, while the engine's negative arm is the exact mirror of its positive one. *Prevents:* an
  off-by-one hiding inside an optimisation. *Covers:* FR-3, FR-4.

- **DD-5 — one tick's contribution is the integer square root of the sum of the two axes' squared
  shares.** `isqrt(sx² + sy²)`, integer throughout — the engine's `FSQRT` and its truncating
  float-to-int of the sum with an integer count, which for a whole count and a non-negative length
  adds the length's whole part. Integers rather than `float64`, which this tier may hold, because
  an integer root cannot land one below at a perfect square, and that is the case FR-3 turns on:
  a straight cell has `sy = 0`, so the length is `|sx|` exactly and the sixteen steps close by
  identity rather than by rounding luck. *Covers:* FR-3, FR-4.

- **DD-6 — the arithmetic lives in `pkg/render/terrain`, beside the timeline it drives.** A new
  file, pure: no clock, no map, no allocation, total for every input. *Why there and not in
  `pkg/sim` or `pkg/game`:* the two decoded numbers — 256 sub-cell units to a cell, 16 to a
  timeline step — are only ever used together and only ever to pick a walk frame, and `unitanim.go`
  already keeps a decoded animation constant of that kind beside the selection it serves. In
  `pkg/game` half of one arithmetic sits a tier from the other half; exported from `pkg/sim` it
  widens the determinism package's API for a cosmetic. *Covers:* FR-2, FR-3, FR-4.

- **DD-7 — the live selection takes the odometer as a sixth parameter; there is still one selection
  path per life state.** The moving arm reads the count, the idle arm and the standing fallback the
  tick, and the fallback chain is unchanged, so a mover that fails its gate still drops to the idle
  selection at the tick. *Why not a second function:* the caller would then hold the moving flag
  *and* choose between two selections, which is the one thing that file's contract forbids.
  *Covers:* FR-1, FR-2, FR-7.

- **DD-8 — the timeline step is `odo >> 4`, an arithmetic shift, taken in the render tier.** The
  shift and not a divide: the engine's is `SAR`, which floors, where Go's `/` truncates toward
  zero, and the two disagree exactly on the negatives P-1's totality answers for. The existing
  euclidean reduction modulo the track length is left alone and now receives the shifted count.
  *Covers:* FR-2, AC-7.

- **DD-9 — the reset forks on the gate that already decides whether an idle cycle draws.** A tick on
  which an entity's derived step is zero sets its odometer to zero when its class's idle gate is
  closed, and leaves it when the gate is open. Three things the fork must be explicit about, since
  the tick and the push resolve classes at different moments and not always the same class: it is a
  **second lookup site**, in `tick()`, of the same bundle by the same class id — DD-2 makes that
  unavoidable; it reads the entity's **own** class, never the corpse class the death path
  substitutes, that substitution being a fact about drawing a body; and a class that **does not
  resolve** has no idle cycle and resets — the "resolved to nothing" answer the seam gives a nil
  class everywhere else. *Covers:* FR-5.

- **DD-10 — no alignment is introduced anywhere, and the modulus stays the track's own length.**
  Nothing rounds the count at a cell boundary, nothing snaps the step to a cycle boundary, and the
  reduction is the plain modulus over the run-length expansion the loader already builds. The 44
  shipped pairs whose cycle does not divide sixteen are *drawn as they fall*. *Covers:* FR-6.

- **DD-11 — the corpus audit sweeps one index into both clocks.** `UnitAnimAudit`'s domain stays
  rectangular and the same size: index `i` over the longer of the two tracks, passed as `tick = i`
  and `odo = 16*i`, so every step of the move track and every step of the idle track is still
  reached and the in-range/guarded split still partitions the whole domain. It is a **regression
  guard and not an instrument**: sixteen times the index puts the moving arm on exactly the step
  the tick used to, so the figures are expected bit-identical. That is what SC-8 asks; the moving
  arm's totality on counts the sweep cannot reach is SC-6's. *Prevents:* a sweep that silently
  stops covering the moving arm once that arm stops reading the tick. *Covers:* FR-2, FR-7.

## Risks

- **R-1 — the odometer and the drawn position are two readings of one movement, and inside a long
  crossing they genuinely disagree.** The drawn position interpolates linearly over the crossing
  with sub-tick resolution; the odometer accumulates the engine's truncating per-tick split. They
  agree exactly at the start and end of every crossing; inside one the gap is `r(n-r)/n` sub-cell
  units at worst, for `r = 256 mod n`, which is a fifth of a timeline step at the crossing lengths
  shipped speeds produce and several steps at lengths only a customised speed reaches. In the
  engine there is no gap: its drawn position is moved by the same share the clock accumulates.
  *Not held by SC-4*, which measures the boundaries, where the gap is zero by construction. The
  risk is **accepted and disclosed at its measured size**; what closes it is a later story moving
  the drawn position onto the same split, which is 0047/0056's contract and not this one's.
- **R-2 — a class whose idle gate is open but whose walk gate is closed.** It walks while drawing
  the idle cycle at the tick, and its odometer advances unread. *Held by:* the fallback chain being
  unchanged, and AC-7.

- **R-3 — an entity whose derived step spans more than one cell.** A hand-built world hands one
  over: a snapshot's transit pair is only bounds-checked, so an entity first seen mid-crossing has
  no step memory and the next tick's delta is whatever it walked. *Held by:* the odometer taking
  `prev`'s own presence test, the share being taken per axis from that axis's own whole delta, the
  64-bit product inside the root, and DD-6's totality, exercised at an absurd delta.

- **R-4 — the sub-cell grid now has two spellings, one each side of the determinism wall.**
  `pkg/sim` derives its longest transit and its tick count from a 256 this tier cannot import, and
  the odometer's correctness is the claim that the two are one number — the duplication
  `AGENTS.md` names as the failure that goes stale without failing anything. *Held by:* SC-9,
  which measures `pkg/sim`'s grid through the only window onto it there is, the transit a mover at
  the rate floor is given.

## Success criteria

- **SC-1** — the closed-form share agrees with the engine's own recurrence, share for share, for
  every span in `1..256` and both a straight and a diagonal delta.
- **SC-2** — over every span in `1..256`, a straight cell's shares sum to exactly 256 and the
  odometer's advance over the whole crossing is exactly 16 timeline steps.
- **SC-3** — over every span in `1..256`, a diagonal cell's advance is at least a straight cell's
  and no greater than the exact euclidean length of a cell diagonal, and the two are **equal** at
  the longest spans, where every share is one unit and the root truncates the surplus away.
- **SC-4** — a unit driven across several cells in a running world shows an odometer equal to the
  running total of its crossings, and its drawn frame at each cell boundary is the one the total
  predicts under the class's own cycle length — checked on a class whose cycle divides sixteen and
  on one whose cycle does not.
- **SC-5** — a stop resets the count for a class with no idle cycle and leaves it for a class with
  one.
- **SC-6** — the frame selection answers for a negative odometer, a zero-length track and a
  zero-frame sheet without panicking, and equal inputs give equal answers.
- **SC-7** — the canonical byte form and the digest of a driven world are unchanged at every tick
  index, and a snapshot built twice with no tick between selects identical frames.
- **SC-8** — the idle, standing, death and ambient selections are unchanged, and the corpus audit's
  in-range/guarded split still partitions a domain of the same size.
- **SC-9** — the transit `pkg/sim` gives a mover at the rate floor equals the distance the odometer
  pays for one straight cell, so the two spellings of the sub-cell grid cannot drift apart
  unobserved.

## Traceability

| FR | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-1, DD-2, DD-3, DD-7 | SC-4, SC-7 |
| FR-2 | DD-6, DD-7, DD-8, DD-11 | SC-2, SC-6 |
| FR-3 | DD-3, DD-4, DD-5, DD-6 | SC-1, SC-2, SC-9 |
| FR-4 | DD-4, DD-5, DD-6 | SC-1, SC-3 |
| FR-5 | DD-3, DD-9 | SC-4, SC-5 |
| FR-6 | DD-10 | SC-4 |
| FR-7 | DD-7, DD-11 | SC-7, SC-8 |
