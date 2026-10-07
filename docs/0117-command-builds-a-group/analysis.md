# Analysis — what a player order does to a group

## The report

A unit is commanded somewhere. It goes. It arrives. Then it turns round and walks back to the cell
it was placed on, and it does that whatever the player does next.

## The measurement, before anything was changed

The tenth mission, EN root, script unit 21 — a hostile at (36,51), owner slot 2, placed group 2,
post (36,51). One ordinary move order to (42,51); the mover routed to (41,52) and arrived at tick
60 holding no destination. At tick 61 it held a new one: **(36,51), its own post**. It stood on the
post again at tick 127, having been ordered nowhere by anybody.

## Where it comes from

Not from the walk home itself, which is right: an idle guard going back to its post is what an
unscripted hostile population does. It comes from the commanded unit still being **in the guard
group** when the order finishes.

The chain, as the code stands:

- The engagement decision partitions the world by `(owner, placed group id)` and reads each group's
  stored order off a record frozen at construction — Stand Ground for the local participant's
  groups, Guard for every other. Nothing on a tick path rewrites that byte but the mission script.
- A player order writes a destination on an entity and nothing else. It does not touch the group.
- While the unit walks, `aiGroups` skips it, so nothing decides for it. That skip is keyed on
  *holding a destination*.
- On arrival the destination is gone, the skip stops applying, the unit is back in its guard group,
  and the guard stance's own tail sends it to its post.

So the defect is not in the guard tail and cannot be fixed there. A test for "under command" in the
walk home would only cover the walk, which is already covered; on arrival there is no command left
to see.

## What was checked and found false

Two readings were handed to this lane and both had to be corrected before anything was built.

- *"A group's order is derived from its owner, recomputed every tick, so nothing a player does can
  change it."* Half false. The derivation from the owner is real but it runs **once**, in the world
  constructor; since 0096 the order is stored per group, carried by the byte form and by the digest.
  The reason a player cannot change it is different and simpler: **a player order does not name a
  group at all.**
- *"There is no stored per-group order on this path."* False, same finding.

The rest held: the walk home has no test for a unit under command, and adding one would not be
visible to the owner, because on arrival the unit is not under command by that predicate.

## What the law says instead

A player order in the original does not write on the actor's existing group. It **allocates a new
one**, moves the commanded actors into it, and each order routine leaves its own byte in it — a
move order leaves 4. So a commanded actor is not a guard with an exception; it is not a guard.

That is the shape this story builds, and it is why the answer is a membership change rather than a
condition in the guard tail.

## What that costs, and the one thing it must not disturb

The mission script has its own uses for a group id: an alive-count check whose triggers are how a
campaign says "the camp is cleared", an owner hand-over that names a group, and the group-order
command. All three read the entity's group word directly, and the count check's own note in the
source states the invariant they rest on — that nothing in the package ever moves an actor between
groups.

Moving the placed word would put a player's click inside a mission's win chain. So the placed word
does not move: an actor gains a **second** group id, read by the engagement decision alone.
