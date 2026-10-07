# Plan — the step cost, relative to the mover

## Design decisions

**DD-1 — the pair the law is evaluated on: the mover's own cell to the cell under the cursor**
(FR-1, FR-4). Three readings of «относительно персонажа» were available and only this one is both
answerable and mover-relative.

Reading it as *the cell's own cost byte* fails the requirement outright: that number is identical
for every unit in the world. Reading it as *the cost of the route to that cell* would need a search
per frame and would answer a different question — a path length, not a step. What is left is the
law's own unit of work: it is a **per-step** law over an ordered pair, and every mover-dependent
term enters through that pair. Half the answer comes from where the mover is standing, which is
exactly what makes it relative to him.

**DD-2 — a pair that is no single step is stated and marked; a pair of one cell is not stated at
all** (FR-5, FR-6). The law reads exactly two cells and is total on them, so a distant pair has a
well-defined answer and refusing it would empty the row for almost every cursor position the reader
actually uses. Stating it unmarked, though, would let a distant reading be taken for the mover's
next step. So it is stated with a marker word — `FAR` — in the readout's existing idiom, where
`EXT` already prefixes a value that is off the shipped set rather than being encoded as a sign.

The degenerate pair is different in kind and is refused rather than marked: a transit from a cell to
itself is not a step, no advance can produce one, and `FAR` would be the wrong word for it.

**DD-3 — one composition site, and the advance and the query share it** (FR-3, P-2). The block in
the advance that composes `rateOf`'s six arguments — the bounds test, the two plane reads at the two
cells, the effective speed, the domain, and the diagonal test handed to `transitOf` — moves into an
unexported method on the world, and the advance is re-expressed through it. The advance's behaviour
must not move: the extraction is textual, and its evidence is that the existing rate, transit and
digest tests stay green untouched.

The alternative — a second function computing the same law for the query — was rejected on the
readout's own founding rule. A number kept beside an authority agrees with it until the first clamp,
truncation or refusal, and is silently wrong afterwards.

**DD-4 — the query is a read on the world, keyed by entity id** (FR-3, FR-8). It follows `Route`'s
shape exactly: an exported method, a binary search through `indexOfEntity`, plain integers out, and
no allocation of anything a caller could write back into a world. It adds no field to any state
type, so the byte form, its version and the digest are untouched by construction.

Its signature returns the rate, the transit, whether the pair is one step apart, and whether there
is an answer at all. **Adjacency is reported by the simulation and not computed by the caller**:
only the world knows which cell it actually evaluated from, so a caller deriving it from its own
copy of the mover's position could mark a number that was computed about a different pair.

**DD-5 — the refusals live in the query, not in the caller** (FR-5, P-1). It answers "no value" for
an id the world does not hold, for a mover that is not alive, for a mover of zero effective speed,
and for a destination equal to the source. The first is `Route`'s own rule; the middle two are the
advance's own gates, restated nowhere — the move loop skips a unit that is not alive, and `rated`
gates the whole rating block. Behind the seam rather than in the drawing tier is what makes them
unbypassable: a second caller inherits them.

**DD-6 — the seam is a function of builtins, installed once** (P-3). The drawing tier may import the
render tier and nothing else, and both of the question's inputs exist only inside it. A value pushed
per frame cannot work, because the pusher holds neither input; recomputing the law in the drawing
tier is the duplication `MapEntity.Speed` already refuses in writing. So the **question** crosses
instead of the answer: a func field on the viewer, set by the tier that owns the world, taking an
entity id and a cell as plain integers and returning a small struct of plain integers.

It is installed at construction rather than pushed per frame, because it is a capability and not a
value: a closure allocated every frame would be garbage for a fact that never changes. A viewer that
was never handed one answers no value everywhere, which is the same shape as the readout's zero
value meaning "not told" — every existing viewer and every hand-built test front-end keeps its
behaviour with no call added.

**DD-7 — the resolved values enter the readout subject; the function does not** (P-4). The subject
is compared with `==` to decide whether the box is recomposed, and a struct holding a func is not
comparable — comparing one panics at run time rather than failing to build. So the func lives on the
viewer beside the layout, and only its resolved output — two integers and two bools — joins the
subject. That also satisfies P-4 without a second rule: what is drawn is in the key because it is in
the subject.

**DD-8 — two rows, at the end of the unit block** (FR-1, FR-2, FR-9, AC-14). Field numbers **28** and
**29**, continuing the readout's own block (16–27) and clear of the unit panel's set, which now
reaches 32. Labels `MOVE` for the rate and `STEP` for the transit, placed after `CROSS`, so the box
reads as the unit's own numbers followed by what a step would cost it.

The transit is stated in the readout's existing tick idiom, and the rate as a bare integer. No unit
suffix for the rate: it is a displacement in sub-cell units per tick, and the denominator is a
simulation constant that the drawing tier would have to hard-code to print.

Both rows are unit rows and are omitted with the other three when nothing is selected, because a box
stating some of a unit's rows would be describing part of a unit.

**DD-9 — the divergence is disclosed at the site that computes the figure** (FR-7), in the doc
comment of DD-3's composition site, where anyone reading how the number is made must pass it.
Neither term is put on a row: the multiplier's absence changes the figure on no shipped map, and
deciding whether a cell is one whose cost the original would rewrite needs a block bit and a table
this tree does not read — a marker for it would be a guess wearing a measurement's clothes.

**DD-10 — the wiring is a constructor argument's twin, not a new load path.** The tier that owns the
world installs the query on the viewer at the point it already performs the readout's tick-0 push,
so a mission opened and left stopped states the value from its first frame rather than after its
world runs.

## Risks

**R-1 — the extraction changes the advance.** The rate site is on the hot path of every moving unit
and its behaviour is covered by the digest. Mitigated by keeping the extraction textual and by
requiring the existing rate, transit, replay and hash suites to pass **unmodified**; a change to any
of them during this story is the signal, not the fix.

**R-2 — an incomparable readout key.** A func reaching the subject builds cleanly and panics only
when a second frame is composed. DD-7 keeps it out; a test that composes two consecutive frames with
the query installed is what makes the panic reachable if it ever returns.

**R-3 — a per-frame binary search.** Bounded by design: the query runs at most once per composed
frame, is a search over the entity slice rather than a wave, and allocates nothing.

**R-4 — the marker misread as a path.** A distant pair's figure is a single hypothetical step, not
the cost of walking there. DD-2's marker is the mitigation; no search sits behind the number.

**R-5 — a concurrently landing story moving the simulation.** Another story is in the same package
in the same period. Mitigated by rebasing before the push and re-running the whole gate afterwards,
rather than by trusting a clean textual merge.

## Success criteria

| | Criterion | Serves |
|---|---|---|
| **SC-1** | The advance's rate and transit are produced by the same function the query calls, and the existing rate/transit/replay/hash suites pass unmodified | FR-3, P-2, R-1 |
| **SC-2** | The query's value equals what the advance writes for the same mover over the same ordered pair, at differing cost, differing height, both step orientations, and with a group term set | FR-1, FR-2, FR-3, FR-4 |
| **SC-3** | The query answers no value for an unknown id, a unit that is not alive, a zero effective speed, and a destination equal to the source | FR-5, P-1 |
| **SC-4** | The query reports the pair adjacent for the eight neighbours and not adjacent beyond them | FR-6 |
| **SC-5** | The readout states both values for a selected rated unit and an on-map cursor, states the absence marker for each cause, omits both rows with the other unit rows when nothing is selected, and marks a distant pair | FR-1, FR-2, FR-5, FR-6 |
| **SC-6** | A viewer with no query installed draws the readout with both rows marked absent and everything else unchanged | FR-5 |
| **SC-7** | Both values participate in the rebuild key: changing either alone recomposes the box | P-4 |
| **SC-8** | The byte form's version literal, a world's encoded bytes and its digest are unchanged, and no simulation state type's field set moves | FR-8 |
| **SC-9** | The drawing tier's permitted imports are unchanged and the import-graph check passes | P-3 |
| **SC-10** | The shipped layout carries both rows and no existing row's number, label or order has moved; the unit panel is untouched; the readout's default and its hide key are unchanged | FR-9, AC-14 |
