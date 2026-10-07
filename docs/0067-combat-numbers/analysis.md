# Analysis — a placed unit's combat numbers

What was not known when this story opened, what was read to settle it, and what is still open.

## The question, and why the phrase had to be replaced

The story was opened as *fill a placed unit's combat numbers from its class*. "Combat numbers" is
not a field list, and taking it as one is how a story invents a field the game does not have. The
ledgers were read to derive the list instead, and it came out at **exactly eight**: the two cadence
numbers, to-hit, defence, absorption, the damage pair, and the always-hits mark. Eight, not seven,
because the mark is a flag rather than a number and is set by the same column group as the pair; and
not nine, because **reach is not among them** (below).

The list has a second, independent confirmation that costs nothing: the simulation's own entity
record already declares those eight and no others, and the definition tier already decodes those
eight and no others. Two tiers written a story apart, each from the ledgers, arrived at the same
eight. This story is the wire between them.

## What was read

The four ledgers named in the brief, plus the promoted `formats/unit/format.md`, which turned out to
carry the whole creation order and the whole slot map as a Level-3 spec rather than as scattered
rows. On the tree's side: the entity record, the blow resolver, the definition decoder, the
placement resolver and the two sites that build entities.

Every row cited in `provenance.md` was read whole in the ledger, not through a summary — including
this story's brief, which described the eight as "seven" on the strength of the previous story's
own disclosure. That disclosure said seven numbers **and a mark**; the count was right and the
sentence that carried it forward was not.

## Three things a first reading gets wrong

**Reach looks like a ninth number and is not one.** It is the obvious candidate — a per-actor value
the blow resolution reads, currently a package constant. But the Units table has **no reach column**:
the constructor sets it to one, and only an equipped weapon's own range moves it. So a reach
"filled from the class" would be the constant one for every class on the map, and turning the
constant into a field would add four bytes of hashed state with exactly one reachable value. The
previous story guessed this and said so; this story measured it. The divergence that remains is
equipment's, not the class table's, and it is real: a substantial minority of shipped classes end
up reaching further than one cell once armed.

**The difficulty setting looks like it scales a unit and it adjusts two numbers.** It moves the
health maximum, the to-hit and the defence, and nothing else — not the damage, not the absorption,
not the cadence. That is already what the tree implements, but nothing could fail if it were wrong,
because to-hit and defence never reached an entity. This story is the first point at which the rule
is observable, so it gets a criterion that fails if any of the other six ever moves.

**The byte form does not move.** The natural assumption for a story that fills state is that the
state is new. It is not: the previous story added all eight fields to the record and left them at
zero on every loaded map. Version 11 was allocated for this story and **is not used**; version 10
stands, no offset moves, and no world becomes unreadable. What does move is the two pinned load
digests, because their records' content changes.

## The unresolved population, and an inconsistency it exposed

Roughly one placement in six on the shipped corpus resolves to something this tree does not model —
a humans-collection entry, an npc, or nothing at all — and those entities need eight values too.
Reading the existing loop to decide what they should get turned up a split that predates this story:
the same statement hands the movement-column lookup a **zero** definition and the rate a
**constructor-defaults** definition. Both happen to answer correctly, so nothing fails. The design
collapses the two into one value rather than adding a ninth default beside the eighth.

Two related things were found and deliberately **not** fixed here, because each is a different
concern and fixing them inside this story would hide them:

- the provisional spawn health an unresolved placement carries is not the constructor's own health,
  while its rate and now its eight combat numbers are. The health is flagged as ours and
  provisional at its own site; this story leaves it exactly where it is.
- the party-placement site spells its defaults by name in a second literal. This story makes the two
  sites share a source because the contract requires the two populations to be identical, but it
  does not restructure either site.

## What could not be settled, and why

- **Whether a blow whose damage falls to zero or below after absorption removes nothing.** The
  resolver's arithmetic order is published and read; the clamp on the *physical* component is not.
  The previous story authored "removes nothing" and disclosed it. Until now that branch was
  unreachable on any loaded map, because every loaded map's absorption was zero. Filling absorption
  makes it reachable. Nothing about the resolution changes here and no test asserts what the
  original does — it is a standing question for research, recorded as such.
- **Whether it is reachable on the first mission in particular.** That needs the shipped Units
  table's per-class column values, which exist only inside an experiment's evidence and are
  therefore not citable. A claim publishing them is requested.
- **What a humans-arm placement's own combat columns are.** The humans streamer's slot map is
  described by difference from the units one rather than published in full, so a placement below the
  class-key floor keeps the constructor's eight rather than gaining a second stat source. Requested.

## What this story does not make observable

Nothing in this tree can order an attack on a loaded map: no front-end issues the order and nothing
walks an attacker into reach. So the entire product of this story is, today, visible in the byte
form, in the digest, and in a printed stat dump — and in nothing that moves. The mission the pipeline
is aiming at needs an order path as well, and that is a different story. Saying so here is cheaper
than a verification stage that reads as if a fight had been watched.
