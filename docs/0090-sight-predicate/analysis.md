# Analysis — what a group can actually see

**Intensity:** spec-anchored / static. **Terrain:** brownfield in `pkg/sim` — the engagement
decision already ships and its candidate population changes — and greenfield for the window tables
and the march, which have no counterpart in the tree.

## What was not known when 0086 shipped, and what closed it

0086 built the group engagement decision with no line-of-sight term. The predicate had been read
end to end, but the two grids it reads had no traced writer, so the shape of the region it admits
could not be reproduced. That story disclosed the consequence — a group sees through walls — rather
than hiding it, and named the seam.

Both grids now have one writer, it runs once at world construction, and it depends on the map for
nothing: geometry and a single registry parameter. The region is therefore reproducible from
arithmetic alone, and this story fills the seam.

## The reading, re-executed before anything was designed

Four published landmarks were reproduced from the claims alone, in a scratch program outside the
repository, before a line of the contract was written. Each is a discriminator: a wrong reading of
the zone tests, of the mirroring order or of the truncation fails at least one of them.

| Landmark | Reproduced |
|---|---|
| Window cells whose predecessor is not strictly closer, without the builder's four trailing stores | 2 |
| The same with them | 0 |
| Step-cost range over the window | 128..181 |
| Flat-ground visible cells, `scanRange` 1..6 | 9 / 21 / 45 / 69 / 105 / 145 |
| Flat-ground visible cells, `scanRange` 19 | 1 253 |

Two things were settled on the way. The step grid's stride-`0x80` index is the **column** and its
stride-2 index the **row** — fixed by the two literal fix-up addresses, which the source names as
the cells `(+1,0)` and `(-1,0)`. And the step cost, published as a float expression ending in a
truncating conversion, agrees with an integer form built on `isqrt` at **all 1 681** window cells,
so the determinism wall does not have to be argued with.

## Where the flat-ground figure is contested, and why nothing here depends on it

The published server figure at `scanRange` 6 is 145 cells and the structurally parallel client
implementation publishes 127 at the same parameter. Research has deliberately not decided whether
that is a real divergence between two implementations or a defect in one of two readings, and grades
"the same algorithm" Medium for that reason.

**Nothing built here rests on the choice.** The contract is derived from the server side's own
closed forms — the seed, the cost expression, the zone tests, the ring bound — each of which is
graded High on cited instructions. 145 is what those forms produce; it is a consequence of the
contract, not an input to it. Had the two figures been used to choose a parameter, this story would
have had to stop.

## What 0086 got wrong, and it is worth naming

0086's disclosure said the law's region is a **subset** of the Chebyshev disk it shipped, so the
build acquired strictly more. That is now false in one direction: the per-cell term is the
observer's altitude minus the cell's, and descending ground **returns** budget, so the region can
reach far past the flat radius. Over the shipped corpus at `scanRange` 6 the visible count runs
29 minimum against a 145-cell flat baseline and 1 248 maximum. Reach scales with where the unit is
standing. So this story does not only narrow what a group acquires; on descending ground it widens
it, and the milestone measurement is not a one-way bet.

## What a unit's sight range is, and why it is still a constant here

The predicate takes a per-unit range. It is not undecoded: it is a `Data.bin` column landing in the
actor byte the stamp reads, and a hero recomputes it from two stats, overwriting the same byte.
`pkg/data` already decodes that column on both bands and defaults it to 5 where the cell is empty.

What is missing is the plumbing, and the plumbing is a byte-form change: a per-entity range is
hashed simulation state, so it needs a field, a form version and a decoder arm. That is a story, not
a clause, and the form version is held elsewhere. The range therefore stays the constant this tree
already carries — which is that column's own default — and the correction to *why* it is a constant
is written where the constant is.

## Two shapes of the original that a consumer meets head-on

A cell that blocks is **not pruned**: the march stores the failing value into that cell's own slot
and walks on, and a later cell whose predecessor is the blocked one continues from the stored
negative. Nothing is ever re-lit behind a blocker on shipped data, but that is a **corpus fact**,
not a property of the code: the cheapest step costs 128, the height plane is read signed, and every
shipped altitude byte is 0..127 — so the largest rise available anywhere is 127, one short.

And the walk's own boundary test is defective in the original: the rectangle is compared as bytes
while the plane is indexed in 32 bits, so on a 256-wide map a true column of `-9` passes the test
and reads the previous row. It needs a ring of radius 17 or more, so it cannot bite until sight
extends far past the range down a slope; 7 of 38 EN maps and 4 of 34 RU maps are 256x256.

Both are settled in the contract rather than left to the implementation.
