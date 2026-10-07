# Plan — what blocks what, and what stands in front of what

Two independent changes, two tasks, no shared file.

## Design decisions

**DD-1 — the fix is the table load, not the plane derivation.** Every world builder in the tree
already calls the two-argument plane derivation and already passes the front-end's table down. The
only thing missing is the collection itself. So the change is one field in the table the front-end
builds, and no builder, no signature and no call site moves. This was arrived at by reading the call
graph rather than the symptom: the symptom points at the map screen, and the missing collection
breaks the mission path identically.

**DD-2 — the viewer's grid plane is left alone, deliberately.** Its one reader tests the air bit and
the structure stage writes the ground bit. Handing it a table would widen a signature used by the
standalone viewer and by twenty test call sites to change nothing observable. Recorded as FR-6 so a
later reader does not "fix" it and conclude the story was incomplete.

**DD-3 — the entity plane merges in the render tier, beside the merge it joins.** The
structure/object merge exists exactly once, in the render tier, so that neither renderer carries an
ordering opinion. A three-way merge written in the window tier would be a second ordering rule in a
package that is supposed to have none. So the render tier gains one function that takes the prebuilt
two-way order and the three lists and returns the three-way one, and the window tier walks it.

**DD-4 — the entity order is derived, not required.** The two art lists are row-sorted by their
builders; the entity list is in world-entity order and is not. The merge therefore sorts an index
permutation of the entity list itself, stably, rather than demanding a sorted input — otherwise
every caller would carry half of the ordering rule.

**DD-5 — the units move into the content band; the instruments do not move.** The frame is terrain,
then the content band, then a slice of instrument passes. A unit's sprite is content. It moves out of
the instrument slice into the content band, and every instrument — the lattice, the blocked tint, the
three crosses, the health bars, the selection rim — keeps its position and its relative order. The
entity **square**, which is the fallback for a unit whose art does not resolve, stays in the
instrument slice: it is a diagnostic glyph, and leaving it there keeps "a square is drawn over the
sprite band" true by position rather than by argument.

**DD-6 — one painter, one cull.** The entity sprites are painted through the same per-sprite cull and
transform the art plane already uses, extended with the mirror bit that the instrument slice's own
painter carried. Two painters would let the drawn sequence and the tested one disagree about an edge.

**DD-7 — flat structures stay first.** The flat pass is the early band and is not merged with
anything. The three-way merge emits the two-way order's flat prefix untouched, then merges.

## Tasks

T1 — the buildings collection reaches the table, and the tests that prove a crossing.
T2 — units join the cell-anchored plane's row order.

## Success criteria

**SC-1** A synthetic archive whose table carries buildings rows loads into a table whose buildings
collection reports those rows (AC-1).

**SC-2** A synthetic map plus a synthetic table close the cells the blocking set names, measured on
the world the front-end hands out (AC-2, AC-5).

**SC-3** The same, opening cells the arms closed (AC-3, AC-5).

**SC-4** A reachability walk over the built plane connects two regions with the table and leaves them
disconnected without it (AC-4).

**SC-5** A recorded draw target shows a unit's frame submitted before a later-row drawable and after
an earlier-row one (AC-6), and after every flat one (AC-7).

**SC-6** The recorded art sequence on a viewer holding no entities, and the recorded entity sequence
on a viewer holding no bundles, are unchanged from the pre-story values (AC-8, AC-9).

**SC-7** `git diff --name-only <base> HEAD -- pkg/sim` is empty and the full test suite passes with
no pinned digest edited (P-1, P-2).

## Risks

**R-1** The buildings collection is subscripted rather than searched, and a wrong collection id would
resolve plausible-looking rubbish rather than failing. Mitigated by SC-1 asserting an entry's own
parameters, not merely a count.

**R-2** Moving the entity sprites out of the instrument slice changes what a large body of existing
tests reads. Those tests are rewritten against the recorded target rather than deleted; a test that
can no longer be expressed is a signal, not an obstacle.

## Traceability

| requirement | criterion | task |
|---|---|---|
| FR-1 | SC-1 | T1 |
| FR-2 | SC-2 | T1 |
| FR-3 | SC-3 | T1 |
| FR-4 | SC-4 | T1 |
| FR-5 | SC-5 | T2 |
| FR-6 | — (a requirement met by changing nothing; witnessed by T1's scope fence) | T1 |
| FR-7 | SC-7 | T1, T2 |
| AC-1…AC-5 | SC-1…SC-4 | T1 |
| AC-6…AC-9 | SC-5, SC-6 | T2 |
| P-1, P-2 | SC-7 | T1 |
| P-3 | SC-5, SC-6 | T2 |
| DD-1, DD-2 | SC-1…SC-4 | T1 |
| DD-3…DD-7 | SC-5, SC-6 | T2 |
