# Analysis — what a blow shows, and what the tool was saying instead

## What we did not know

Whether the original draws anything at all when a blow lands. 0081 made the swing visible and left
the blow's own consequence out of scope with a note that it had just been decoded; this is that
decode arriving. Three of the four things a naive build would get wrong were unknown to us before
reading the rows: whose colour the figure takes, which clock bounds its life, and what a second blow
inside that window produces.

## What we looked at

`research/claims/anim.md` rows `ANIM-BLOW-019`, `ANIM-NUM-020`, `ANIM-NUM-021`, `ANIM-SND-022`, and
the listings under the experiment's `evidence/` that back them. The rows are the contract; the
listings were read only to settle two things the row prose leaves ambiguous — which way the
horizontal drift goes for whom, and how the initial offset is computed — both unambiguous in the
instructions and both agreeing with the rows.

## What the reading settled that changes the shape of the work

**The damage figure does not have to come out of the simulation.** The engine's own notification
carries the victim's *new health as a level* and never the delta; the client subtracts the health it
was already holding. So the front end computing `old - new` from the health that already crosses the
seam is not an approximation of the original — it is what the original does. `pkg/sim` is untouched
and the byte-form version does not move.

**The merge rule and the tick granularity agree.** A record already standing for a victim takes the
new damage *added into its own number* rather than becoming a second figure. Our seam observes
health once per frame rather than once per notification, so two blows inside one frame arrive as one
subtraction — which lands on the same figure with the same sum the merge rule would have produced.
The granularity difference is therefore invisible in the drawn result for any two blows inside the
window, and the window is the only place the two could differ.

**The severity colour is a trap.** The arm computes a three-band colour on every landed blow and
stores it in a stack local that a text scan of the whole containing function shows written three
times and read zero. It is exactly the shape a "damage flash" implementation would reach for. It is
not implemented, and the spec says so.

**One decoded term has no counterpart here.** The initial offset is the victim's own `vt+0x20()`
scaled — `±16` horizontally, `-48` vertically. That virtual is not decoded, so the scalar it returns
has no name in this tree and nothing to derive it from.

## Two owner observations, used as a check and not as a source

The owner reported seeing the numerals **diagonal** and in **two colours**. The rows were produced
blind to both. The drift has two nonzero axes per tick — `-2` vertical and `±1` horizontal — which is
a diagonal; and the colour is the victim owner's, which over two participants is two colours. Both
observations fall out of the rows rather than being fitted to.

## The instrument that had to be fixed first

`cmd/missionrun` reports the outcome of a drive against a lawful install, and it is this story's only
way to say anything about a running world. Its health reader answered `0` for an entity the world
does not hold, and the fell test reads `<= 0` as fallen — so a drive naming a party slot that does
not exist reported `FELLED it after 1 ticks, victim at 0 hp`. Zero is also a real health value here:
a downed unit stands at exactly zero. The tool could not distinguish absent from downed, and the
answer it gave for absent was the strongest positive result it can produce.

A previous lane found this, discarded the evidence, and left the tool alone. It is fixed here before
anything else, because every measurement below it would otherwise be worth nothing.
