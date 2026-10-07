# Analysis — what we did not know, and what we looked at

## The question this story arrived with

Two pending items that read as separable: *a group move distributes destinations*, and
*a per-group rate term is never cleared*. The reason they are one story is a single
local flag in the two group-move setters, which gates the distribution and the rate
store and nothing else. Implementing either alone invents a behaviour: a distribution
with no rate term, or a rate term that fires when the group is not in formation.

## What the ledger had said before, and what replaced it

The area's older headline was a clean negative — *no formation, no offset table, no
spread; a group move writes one cell into every member's order block*. That row is now
**partially retracted**, and it fell through the blind spot its own confidence cell
named: it enumerated one instruction encoding for the move opcode, and the per-member
issue routine writes that opcode **from a register**. So the earlier reading was right
about the three routines it read and wrong about the population it thought it had
enumerated.

We took the current rows as the source and did not build on the retracted headline. The
practical consequence is that our own earlier reflex — *units spread only because each
substitutes for itself* — is a true statement about the substitute picker and a false
statement about order time.

## What we had to decide rather than look up

**How a group reaches this package.** The thing being reconstructed carries a group
object with an AI record hanging off the owning player, and every player order allocates
a fresh one. This tree has no player, no AI manager and no group object, and the only
way anything changes a world is a command naming one entity. So a group order had to be
expressed as a *set of commands*, and the group's one durable field had to find a home.

We looked at three shapes:

1. **A group table on the world**, with an id on each entity. Closest to the original.
   It costs a second canonical collection, an allocation counter that is itself hashed
   state, and a lifetime question — when a group with no members goes away — that
   nothing in this tree can answer, because nothing here removes a member from a group
   except a new order.
2. **A group id on the entity with the rate looked up.** Same costs, without the table
   being the thing that carries the rate.
3. **The term on the entity.** One byte, no table, no counter, no lifetime.

We took (3) after asking what could tell them apart. The group's field is written at
exactly one moment — the order — and read at every transit; the only routine that would
change it afterwards is unreachable; and nothing in this tree adds a member to a group
or moves one between groups. So within this tree's command surface, a term held per
member and a term held per group are the same observable behaviour, and (3) is the one
that adds no state nothing can exercise. What (3) cannot express is a group whose
membership outlives an order, which is what an ownership or scripting story brings, and
that is where the seam is named.

## What we went looking for and did not find

**The signedness of the offset read-back.** The offsets are stored as two 16-bit fields
and read back *as bytes*; the row does not say whether the read is sign- or
zero-extending. It cannot be settled from the rows we hold, and it is invisible on the
default formation mode, because the spread gate admits only offsets in `[-2, +2]`.
Ruled, disclosed and pinned.

**The playable rectangle's own values.** The per-member issue routine clamps into a
four-byte rectangle on the world. The rows name the field and not its contents. Our
bounds are the analogue and the clamp is the behaviour; the numbers are not ours to
claim.

**A rounding convention nobody had to invent.** The centroid looked like the story's
biggest exposure — an integer mean whose rounding shifts every member by up to a cell.
It turned out to be decoded, and not as a rounding rule at all: the centroid is computed
over **sub-cell** positions, and a resting mover sits at the centre of its cell, so the
half-cell offset that makes the mean round to nearest is already in the summands. The
independent check is the ledger's own worked example of the spread threshold — two units
four cells apart pass and five apart fail — which only comes out that way under this
reading.

## What we deliberately did not model

The **per-player formation mode**. It is one byte with three behaviours, it is authored
by a trigger and by a player command, and it is in a save. We have no player object to
put it on, and the whole 38-map corpus authors it once, at the constructor's own default.
So every order this tree can issue runs at that default, which is a narrowing of the
input domain and not a divergence in the law: the two values we do not model are named,
their behaviour is written down, and the arm that would read them is one branch.
