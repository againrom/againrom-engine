# Plan — the ground carries a cost, and the mover pays it

## Shape

Five slices in dependency order: the world gains the planes and the form carries them (FR-1, FR-1a,
FR-9); the search spends them (FR-2, FR-3); the rate reads them (FR-4); the map derives them (FR-5,
FR-6, FR-8); the loaders hand them over (FR-7). Only the last moves mission 10, and that is the
story's own tripwire — see R-1.

## Design decisions

**DD-1 — the three planes arrive as one named bundle, and the two existing constructors keep their
signatures.** A world is built through a new root constructor taking a three-field plane value; the
two that exist today become that constructor with only the block field named. Every plane is COPIED
in, as the block plane already is: they are canonical hashed state, and a caller or a second world
sharing the bundler's slice could otherwise mutate a world after it was built.

The alternative — three more positional byte-slice parameters — is rejected on the argument the mode
parameter already rests on. Three adjacent parameters of one type are three a caller can transpose
with no compiler complaint and no failure except a wrong route much later; named fields cannot be
transposed. It also keeps every existing construction site, of which there are about fifty, saying
what it said, which is what makes FR-1a a property of the code rather than of fifty correct edits.
The cost of that is real and is paid in FR-7: the compiler will not find the sites that SHOULD have
gained planes, so AC-14 has to enumerate them instead, and DD-8 is what keeps that enumeration to
one function.

**DD-2 — absence is materialised at construction, never recorded; a wrong length is refused.** An
unnamed cost plane becomes `cells` bytes of the default and an unnamed height plane `cells` zeros,
exactly as an unnamed block plane already becomes `cells` zeros; a named plane of another length is
refused in the words the block plane already uses. No `hasCost` flag exists, so "no cost plane" and
"a uniform one" are one world in the fields, the bytes and the digest.

**The default is the rate law's own zero-mean substitute rather than a second 8, AND IT MUST BE
EVEN.** They are the same number for the same reason — the value most of the original's cells carry
— and spelling it twice is how a later story moves one and not the other. But the two answer to
different constraints, and only one is obvious: the substitute answers to *the mean most cells
carry*, while the default also answers to *(c, c + c>>1) must be proportional to (2, 3)*, which holds
exactly when `c` is even. At 8 the scale is 4× on both arms. That second constraint is what FR-1a
rests on, so it is written at the constant rather than left to the derivation to imply.

**DD-3 — one declared cell count serves all three planes.** The form keeps its single count where it
is; the two new planes follow the block plane, each of that length. No header field is added, so
nothing before the entity records moves and the record does not change by a byte. The pinned offset
table needs only its version byte changed — not because record offsets are relative, but because that
fixture's bounds carry a non-positive height and therefore zero cells, so all three planes are empty
in it. Three counts would be three numbers free to disagree, with the decoder left to pick a winner.

**DD-4 — the step cost stays a pure function and gains an accessor beside it.** It takes the mover's
domain and the ENTERED cell's cost byte; the plane is read at each call site. The cell is named at the
site deliberately rather than hidden inside a method that sees only the step's delta, because the two
modes flood in opposite directions: the canonical relaxation floods FORWARD from the mover and the
entered cell is the neighbour, while the optimised flood runs BACKWARD from the target and the entered
cell is the one being expanded FROM. The same inversion runs through the extractions — the canonical
walk runs backwards over forward labels, the optimised walk forwards over backward labels — so there
are FOUR sites and each names its own cell. One method taking `(dx, dy)` cannot tell them apart.

**The out-of-bounds answer belongs to the rate's call site, not to the accessor.** Zero is the plane's
honest answer where it describes no cell, but zero is neutral for a height DIFFERENCE and is not
neutral for a cost LEVEL: on a world with no cost plane an off-map endpoint would give a mean of 4
where every in-bounds pair gives 8, and FR-1a would break in exactly that corner. So the rate site
tests both cells and hands the law four zeros when either is off the map, which is what it received
before the planes existed. The search never reaches the arm at all — every step is into an open cell,
every open cell is in bounds, and neither extraction ever charges the mover's own starting cell.

**DD-5 — the mover's domain reaches the cost arm the way it already reaches the terrain test**, from
the asking entity's index, so which mover reads the ground and which pays flat cannot come to differ
between the two questions.

**DD-6 — one classifier, two consumers, and the block plane's mountain arm is rewritten to read it.**
The cost derivation needs the whole class-and-cost answer; the block derivation needs one bit of it,
and a second reading of the same tile-word split is the failure the block derivation's own note
already names.

**The water arm stays separate and that is fidelity, not leftover duplication.** The block plane's
water test is taken on the RAW tile index, ahead of any classification, and the classifier's water
arm is a different test on a different value that happens to select the same set. Folding one into
the other would assert an identity the decode does not.

The hazard here is a passability regression inside a cost story, and it is answered by proof plus an
exhaustive check rather than by care. Mountain is the primary of strip group 7 and the secondary of
none; the water range is exactly strip groups 8-11, so it is disjoint from group 7; both readings
reject a sub-cell of 14 or more; and every arm the new classifier rejects — the water nibble of 8 or
more, and groups 13-15 — returns a class that is not Mountain, where the old predicate's group test
was already false. Four lemmas, and AC-11a pins all four at once over all 65 536 words against the
pre-story predicate transcribed into the test. The corpus run over both roots is corroboration.

**DD-7 — the derivations are total and take the map alone.** Both walk the extent the block
derivation already computes and read their source plane through the same short-plane-reads-zero
accessor, so a hand-built map cannot make either fail and neither can return another length.

**DD-8 — the loaders take all three planes from one bundler.** Every world-building path calls one
function returning the bundle for a map and an optional table. Two of those paths REBUILD a world
from an earlier one, and re-deriving is right for them rather than preserving: they rebuild from the
same map that built the first world and already re-derive the block plane that way. A world that
arrived by being DECODED is never rebuilt — its planes are not a function of any map, and nothing on
this path touches it.

**DD-9 — the rate call site changes and the rate law does not.** The law's file already predicts this
story. What does change is its comment claiming the substitute is always taken: after this story it is
taken on a zero or wrapped mean and on an off-map endpoint, and a comment asserting otherwise would be
a false statement sitting beside the code that refutes it.

**DD-10 — FR-1a's premise is checked, not assumed, and in three shapes rather than one.** Uniform
scaling preserves a decision only if the label is never weighed against something that does not scale
with it. So before the search slice lands, every use of the label plane is read and classified against
all three: a label compared to an absolute constant; a label ADDED to a non-label term, which is what
a heuristic or an admissible bound would be; and a label compared against one from a different search,
which is what a "which unit is nearest" rule would be. Ground labels are now four times non-ground
ones, so the third shape is the one that would bite silently. The result is recorded as read.

**DD-11 — the extraction is bounded, because this story makes an unbounded walk reachable.** The walk
terminates today because a positive step cost makes the label fall strictly at every step. A cost byte
of zero breaks that: every candidate ties, the asymmetric accept picks by scan order alone, and a
labelled region of three cells can cycle forever. No derivation produces a zero, but a caller may pass
one and a decoder must accept one, so the hazard is reachable in a package whose worst failure should
be a wrong answer and never a hang.

The bound is the world's own in-bounds cell count, and it is chosen over the decoded 1000 for one
reason: a walk over positive costs visits a strictly falling sequence of labels and so can never take
more steps than there are cells, which makes this bound provably incapable of changing any behaviour
FR-1a is about. The decoded 1000 would not be provable that way, and reproducing it is left out and
disclosed.

## Risks

- **R-1 — the story's own evidence is nearly invisible.** The corpus's dominant cost byte is 8 and the
  rate law's substitute is 8, so an implementation that carried both planes and read neither produces
  the same routes, rates and mission tick as one that works. Answered by requiring a *difference*: AC-2
  two worlds that diverge, AC-14 the planes reaching each construction path, AC-16 a real route over a
  real derived plane differing from its route over a uniform one, and AC-15 the mission number
  reported whichever way it comes out. AC-15 alone cannot fail, which is why AC-16 exists.
- **R-2 — a passability regression hiding inside a cost story.** DD-6, AC-11a and the corpus run.
- **R-3 — a zero cost byte.** Two distinct consequences, and only one was obvious: every ground step
  becomes free (a wrong route, accepted — refusing the byte would refuse a state the constructor can
  build), and the extraction can cycle (a hang, refused — DD-11).
- **R-4 — the four cost sites disagree about which cell the step enters.** DD-4 names all four; the
  two that run backwards are the ones to read twice.
- **R-5 — the reject value meets the byte-wide cost sum.** Two adjacent rejects sum to 254 and read as
  a mean of 127. This is the decoded law's arithmetic on a byte the decoded ingest also produces, so it
  is reproduced rather than guarded, and FR-4 says so.
- **R-6 — mis-stating what cost cannot do.** It is a label-correcting wave, so within the cells it
  reached it DOES find and take a cheaper route, including one longer in cells; what it cannot do is
  reach further because a route is cheap. The wave advances one ring per generation whatever the
  ground costs, so the labelled SET is a property of the map and the order alone. A criterion written
  to the stronger claim fails against correct code and aims the diagnosis at the cost arm; written to
  the weaker one it also answers the budget question, since the budget counts generations and cost
  does not change how many rings away the goal is.

## Success criteria

| # | Criterion | Serves |
|---|---|---|
| SC-1 | A world naming neither plane marshals to an all-8 cost plane and an all-zero height plane of the block plane's length; a plane of any other length is refused on construction and on decode. | AC-1, AC-1a |
| SC-2 | Two worlds alike but for one byte of either plane have different forms and digests; a previous-version form is refused naming both versions; a current one round-trips to an equal world and digest. | AC-12, AC-13 |
| SC-3 | Over two equal-length corridors a ground mover follows the cheap one whichever it is, while a ghost takes the uniform-plane route either way; and two cost planes over one world label the same set of cells with different numbers. | AC-2, AC-2a, AC-4 |
| SC-4 | The arms give 1/1, 0/0 and 255/382; both extractions charge the cell the step enters; an all-zero plane terminates the walk instead of cycling. | AC-5, AC-6, AC-6a |
| SC-5 | A transit over cost-6 ground takes fewer ticks than over cost-16; the tilt saturates at 32 both ways; an off-map endpoint gives the pre-story length. | AC-7, AC-8, AC-8a |
| SC-6 | The classifier answers 8, 16, 13, 9, 255, 255, 255 to AC-9's seven words; short altitude and tile planes derive to zeros and to the word zero. | AC-9, AC-10 |
| SC-7 | Over all 65 536 tile words the derived block byte equals the pre-story build's, and every shipped map of both roots derives the block plane and census it derives today. | AC-11, AC-11a |
| SC-8 | A non-uniform plane survives to the world through every path; a shipped map's route moves; mission 10's decided tick is reported for both roots; the no-plane world is unmoved from the pre-story build. | AC-3, AC-14, AC-15, AC-16 |
