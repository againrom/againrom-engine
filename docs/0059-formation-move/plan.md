# Plan — how the group order is built

## Shape

Five commits. The first gives an entity the term and wires the rate read to it, so the
persistence rules exist before anything can set the term. The second gives the package a
group order whose every member takes the ordered cell — the behaviour before formation
existed, and a complete one. The third forks it on the formation flag. The fourth raises
the byte form. The fifth makes the map screen issue one.

## Design decisions

- **DD-1 The term is a byte on `Entity`, not a group object** (FR-5, FR-6). `GroupSpeed
  uint8`, zero meaning "no group term". One reader — `moverSpeed(e Entity) int32`,
  returning `int32(e.GroupSpeed)` when nonzero and `e.Speed` otherwise — and `rated`
  becomes `moverSpeed(e) > 0`, so the question *has this mover a rate* keeps having one
  answer in one place. `rateOf` is not touched: it already takes a speed as a parameter,
  so the term enters at its single call site and the law's two arms both see it, which is
  what FR-5's "every movement domain" needs. **This is the seam.** Lifting the defect FR-6
  reproduces means calling the existing `clearGroupSpeed` at a fourth site; re-homing the
  term onto a real group means changing `moverSpeed` and the three writers and nothing
  else, because nothing else reads the byte.
- **DD-2 A group order is a tag, not a container** (FR-1). `Command` gains `Group uint32`
  and the kinds gain `KindGroupMoveTo`. Phase 1 keeps its single pass in slice order and
  gains a small consumed-set: at the first command of a tag it collects that tag's whole
  membership from the slice and applies the order there. The tag is a **correlation key
  for one advance** and is stored nowhere — no world field, no byte-form field, nothing in
  the digest. `KindMoveTo` stays the zero value and stays a single-unit order.
- **DD-3 The centroid is computed in sub-cell units** (FR-2). Per axis,
  `sum += cell*subCell + subCell/2` over the members, then a **floor** division by the
  member count, then `>> log2(subCell)` — spelled as a division by `subCell` so no shift
  hides the units. `subCell` is `rate.go`'s existing constant; the half is written as
  `subCell/2` rather than as 128, so the centred fraction and the grid cannot drift apart.
  The arithmetic is `int64` throughout: a member count times a coordinate cannot wrap into
  a plausible small mean.
- **DD-4 One flag, computed once, forking one loop** (FR-3, FR-8). `formationSpread = 2`.
  The flag is a single `bool` local of the order function; it is computed by one pass over
  the members and read at exactly two places — the destination arm and the term store.
  There is **no member-count branch anywhere**: a group of one runs the same pass, its
  centroid is its own cell, its Chebyshev distance is zero, its offsets are zero. That is
  how FR-8 is satisfied, and the test for it reads the source for the absence.
- **DD-5 The minimum is the original's own loop, widths included** (FR-5). A running
  `uint8` at `groupSpeedInit = 250`; per member `if int16(e.Speed) < int16(min) { min =
  uint8(e.Speed) }`. Both narrowings are deliberate and both are customisation limits: the
  16-bit compare is the class field's own width, the byte store is the group field's, and
  together they mean a group term can never exceed 249 and a class speed above 255 folds
  onto its low byte. Written out rather than replaced by a plain `min`, because the plain
  one is right for every speed the shipped data carries and wrong outside it.
- **DD-6 The byte form goes to version 8** (FR-7). One byte at the record tail, offset
  `+43`, `entityLen` 43 to 44; the header is untouched. The decoder refuses a nonzero term
  on an entity that is not alive, beside the target and transit refusals it already makes;
  the constructor normalises it in the same statement that clears a dead unit's order. The
  hand-transcribed pin in `binary_test.go` grows by one byte per record and its digest is
  recomputed **outside this tree**, by a third FNV implementation, never captured from the
  encoder — which is what that pin is for.
- **DD-7 The front-end names the click, not the tick** (FR-9). `mapWorld.enqueue` emits
  `KindGroupMoveTo`, and the tag is the **click**. *Revised during implementation:* one tag
  per advance is wrong — two right presses between one pair of advances are two orders, and
  the later must win, which one tag would take away. The boundary is **derived from the
  pending queue**: an order joins the group being assembled when that group has a member,
  every member names the same cell, and none names this entity; anything else opens a new
  tag. That is exact for the stream the front-end can emit — a press emits distinct
  entities all naming one cell — and it needs no state but a counter, because a drain
  empties the queue and so ends a group for free. `ui.MapOrder` keeps its signature.
  A selection of one is a group of one, which DD-4 makes the identity.
- **DD-8 The formation mode is not modelled** (FR-3). The per-player mode byte has no
  object to live on here; every order this tree can issue runs at the constructor's own
  default, which is the conditional-on-spread arm and the only value the shipped corpus
  authors. The other two behaviours are written into the flag's own doc comment as the
  branch that would go there, so the narrowing is visible at the site rather than only in
  `provenance.md`.

## Traceability

| FR | Decisions | Witnessed by |
|---|---|---|
| FR-1 | DD-2 | AC-1, AC-3, SC-1 |
| FR-2 | DD-3 | AC-2, SC-2 |
| FR-3 | DD-4, DD-8 | AC-3, SC-2 |
| FR-4 | DD-2, DD-3 | AC-1, AC-4, SC-2 |
| FR-5 | DD-1, DD-5 | AC-5, AC-9, SC-2, SC-3 |
| FR-6 | DD-1 | AC-6, AC-7, SC-4 |
| FR-7 | DD-6 | AC-7, AC-8, SC-5 |
| FR-8 | DD-4 | AC-9 |
| FR-9 | DD-7 | AC-10 |
| P-1 | DD-3, DD-5 | SC-6 |
| P-2 | DD-1, DD-2 | SC-7 |
| P-3 | all | SC-7, SC-8 |

## Success criteria

- **SC-1** A boxed row of three ordered to a far cell takes three distinct destinations,
  measured as world state after one advance, and the row's shape is intact at arrival.
- **SC-2** Mutation testing on the arithmetic that reaches the digest: the centroid's
  half-cell term, its floor division, the spread comparison's boundary, the offset's sign
  narrowing, the minimum's initial value and its strict comparison. Every mutant is killed
  by a named test, and the kill list is recorded.
- **SC-3** The two-unit rate leg: a slow and a fast unit ordered in formation cross at the
  slow one's transit length, both of them, measured in ticks; ordered out of formation
  each crosses at its own.
- **SC-4** The persistence leg end to end: order, arrive, kill a member, marshal, decode,
  advance again — the term is the same byte at every step, and a plain order clears it.
- **SC-5** The form's version is refused at 7, the record width is checked against a
  computed span, the pin's bytes and digest agree with a decode.
- **SC-6** `internal/archtest`'s source scan over `pkg/sim` stays green: no float type,
  literal or import arrives with this story.
- **SC-7** `go test -trimpath -count=1 ./...` green across the module, and the local gate's
  **FAIL set** is byte-identical to the baseline taken on this branch before the first
  edit.
- **SC-8** Every existing test that changes is listed with the reason it changed, and no
  assertion is removed to make one pass.

## Risks

- **A term in hashed state is not revisable.** Every digest and every replay changes
  meaning with it, which is why the threshold here is High and why SC-2 exists.
- **The tag could be mistaken for state.** It is a per-advance key; nothing stores it, and
  DD-2 is where that is written down. A future wire format that carried it would be
  carrying a correlation key into a world, which is what the byte form must never hold.
- **The front-end change is one statement with a wide blast radius.** Every order the map
  screen issues becomes a group order. P-2 and SC-7 are what bound it: a group of one is
  the identity, so nothing about single-unit play moves.
