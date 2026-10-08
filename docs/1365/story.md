# Pursuit search

## Intent and authority

Owner report on a save: orcs on a river bank stand still instead of closing in
on a victim, although cells nearer it are free. Hotfix 0.89.4 answered it with
a stand-in, an eight-ring far search for a victim within eight cells
(DIV-2456). Owner direction: replace the stand-in with the original's
full-versus-near search cadence and picker B.

Authority, knowledge pin k201: `AI-413`..`AI-418`, `MOVE-097`..`MOVE-100`,
`AI-373`, `AI-384`, `MOVE-ALT-018`, `MOVE-ALT-019`, `MOVE-ALT-020` (narrowed by
`MOVE-098`). Medium and Unknown parts take the smallest rule and a row,
DIV-2556 to DIV-2562.

Base: public main `3ddc6ed6`, game 0.92.1.

## As built

A unit attacker outside reach of a unit victim runs one pursuit pass per tick
on its cell centre (`pkg/sim/pursuitsearch.go`). Its search record stands for
mover bytes `+0x7c`, `+0x8a`, `+0x09`, `+0x76` and `+0x8c`:

- A victim other than the recorded one resets the record: count and passes
  0xff, route end and aim at the victim's cell, no static list (`AI-413`).
- Passes above a third of the static list plus one re-search. A count above
  five runs the full search toward the victim's cell, picker A over rings 1 to
  `(D>>2)+3` (`AI-414`, `AI-394`); five or less rebuilds the route end as one
  node (`AI-415`). An empty list refuses the pursuit.
- Standing on the route end while the victim stands where the last full search
  found it forgets the victim, so the next pass searches in full (`AI-415`).
- The near search aims at the route end while five nodes or fewer remain, else
  at the head, or at the node three after it when the head is within three
  cells. A head within three cells is removed after a counted pass (`AI-373`,
  `AI-417`).
- The near search settles by picker B (`pkg/sim/pickerb.go`): eight rings
  around the victim, entered on the edge the 16-way bearing selects where the
  line between the fine centres crosses it, two walkers turning at the side
  ends, at most 100 steps per ring, the first strictly lowest label kept,
  walker 1 first; an occupied cell is skipped and the start cell may be
  returned (`MOVE-097`..`MOVE-100`, `MOVE-ALT-020`). The single-precision slope
  is reproduced exactly in integers.
- An empty near search aimed at the route end refuses an AI-owned attacker;
  aimed at a waypoint it retries next pass. A human participant counts stalled
  passes instead (DIV-2561, DIV-1316).
- A pass whose step only turns is not counted (DIV-2556). A refusal, a lost
  victim and a new order clear the record (DIV-2557). A group reissue at the
  held victim keeps the route (DIV-2558).

The stand-in's eight-ring far search and the nearest-to-victim settle rule are
removed; DIV-2456 and DIV-2225 are closed. The engine save form carries the
record as optional form 115; a world holding no record keeps its earlier bytes.
A SAV carries the record in the native continuation supplement; the SAV mover
bytes are neither read into the record nor written from it
(DIV-2562).

## Proof

Focused tests in `pkg/sim/pursuitsearch_test.go`: the bearing sectors, the
crossing against a double-precision line over a single slope, the wide-mover
entry that leaves the ring, the first strictly lowest label, occupied cells and
a full ring cancelling, full search then rebuild with counts 8, 5 and 1, the
band case of `AI-418` from 7 and from 12, the record round trip, and a reissue
at the held victim. `pkg/sim/pursuitbank_hotfix_test.go` (the bank test of the
stand-in) passes unchanged.

PROOF-RESULTS

## Open debt

- DIV-2556: no stored dynamic list and no `mover+0x78`; the passes per stepped
  cell are Medium.
- DIV-2559: the fine centre ignores a sub-cell position in transit; the FPU
  precision in play is Unknown.
- DIV-2562: SAV mover projection of the record.
- The owner's save: the column of orcs seven cells from the victim is refused
  under `AI-418` (Medium). That state was produced by this build before the
  change; whether the original's crowd ever reaches it is Unknown.
