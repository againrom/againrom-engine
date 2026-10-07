# Spec — an idle guard goes home

## Terms

**Post** — the cell an actor is anchored to. Every actor has one at every moment.

**Stance** — a group order that anchors its members' posts. There are two: the **guard stance**,
group order 1, and the **stand-ground stance**, group order 3. The other four group orders this
build names — none, swarm, move, swarm 2 — are not stances and anchor nothing.

**Stance setter** — the act that puts a group under a stance. This build has two: the world
constructor, which puts every group under one, and the script's group-order command.

**Victim** — the entity a member is attacking: what this package's records call an attack target,
distinct from the destination cell a member may also hold.

**Walk home** — a destination write naming a member's own post.

## Why

A guard member that scores nothing to fight has, in this build, exactly one thing happen to it: if
its group's owner is not the local participant it drops its victim, and then it stands where it
stopped for the rest of the mission. That is not what a shipped hostile does. It walks back to where
it was placed. Because the guard stance is what the overwhelming majority of shipped hostile
placements run, this one missing outcome is most of what an unscripted enemy population looks like
— a map full of creatures that drift once and never return against a map that holds its shape.

The obstacle was never the arm; it was the destination. This tree had no per-actor cell to walk back
to, and the reason recorded for not adding one was that nothing wrote such a cell for a group under
this stance, so the walk had nowhere to go. That reason no longer holds: the act that installs the
stance writes the cell itself, for exactly the members the stance's own arm then reads. So the post
is real state with a real reader, and this story adds it and the one outcome that consumes it.

## Scope

**In.** The post as per-entity state; its two writers; the guard stance's walk home, including on a
map where nothing is in range at all; the stand-ground stance's own anchoring; the byte form and the
digest that carry the new state.

**Out, and to what.**

- **The idle turn** — what the law hands an at-post, idle, AI-owned member instead of a walk. Owed
  to a story of its own: it needs a per-actor flag written where a blow lands, two draws from this
  world's generator on every idle member on every tick, and a facing rule. The draw count is the
  reason (`plan.md`, DD-4).
- **The heal** — what an at-post, idle member of the local participant's own group receives. Owed to
  the story that gives this tree any regeneration at all; it has none.
- **The re-anchor latch** — the patrol arm's own moving post, which `0099` recorded as owed. Still
  owed, now to the story that builds the per-actor guard state, which is its only consumer. A
  patroller's group is under order 0 in this build, so its post is read by nothing.
- **The notice radius's per-flip re-roll** — the jitter the guard arm applies when a group's
  membership latch flips. Owed to the story that revisits the notice base; this one does not touch
  it.
- **The other four group orders' arms, and every per-actor arm but patrol.** Unchanged, and not
  extended.
- **The stance setters' session branch** — which of two mechanisms writes a post depends, in the
  law, on a session test this build has no shape for. Out permanently unless multiplayer arrives.

## The contract

**FR-1 — every entity carries a post, and it is a cell.** Two coordinates on the entity record, of
the same width as the coordinates already there. It is not optional, not nullable, and there is no
value meaning "no post": an entity that has never been under a stance still has one.

**FR-2 — the post is written by a stance setter, and by nothing else.** The two writers are the
world constructor, which writes **every** entity's post, and the script's group-order command when
it installs either stance, which writes the post of **every living member** of every group record
the command's group id names. No other command, no arm, no per-tick pass and no decode-time
derivation writes it. In particular there is no first-tick initialiser: a post is never written
because an actor noticed it was unset.

**FR-3 — a setter anchors an actor at the cell it occupies.** Unconditionally, with no test on
whether the actor is idle, moving, engaged or paying for a crossing. An actor mid-crossing is
anchored at the cell it stepped **into**, and in this tree that is the cell it occupies, so the two
readings coincide and one write serves both.

**FR-4 — a guard member holding no victim and not standing on its post is sent to its post.** Its
destination becomes its post and any destination, route or stall count it held is discarded, exactly
as any other order replacement discards them. This is the outcome that **supersedes** `0098`'s
"nothing is put in the order's place": under the guard stance something now is. What `0098` said
about the release itself is unchanged — a member that scores nothing and whose group's owner is not
the local participant still ends holding no victim — and the walk home follows that release rather
than replacing it.

**FR-5 — a guard member holding a victim takes no post decision.** Whether it acquired the victim on
this pass or was already holding one, and whichever slot its owner sits in. The test is the victim,
never the owner: the owner decides only the release, which FR-4 leaves where it was.

**FR-6 — a guard member holding no victim and standing on its post is left in every field.** No
destination, no facing change, no state change. This is where the law turns such a member on the
spot; this build does not, and that is a disclosed divergence rather than an oversight.

**FR-7 — the walk home runs where nothing is in range at all.** A guard group whose candidate list
is empty — the ordinary state of a quiet map, and the state in which the walk home matters most —
takes FR-4 through FR-6 for every member exactly as a group with candidates does. The decision may
not exit early past this outcome.

**FR-8 — the stand-ground stance gives no member a destination and no member a turn.** Its members'
posts are written by FR-2 and read by nothing in this build. Its arm is otherwise unchanged: no
clip, the reach-refusing scorer, and the behaviour it already has.

**FR-9 — nothing outside the guard stance reads the post.** Not the other five group orders, not the
patrol arm, not the route layer, not the drawing tier, and no read of it derives anything about a
member's radius, its notice base or its candidate list.

**FR-10 — the byte form is version 24, and version 23 is refused as every earlier version is.** The
entity record's tail gains the two coordinates. No section moves and no count is added. The post is
carried **whole**: no coordinate is refused, folded or clamped, on the rule the commanded cell and
the patrol ring already take — every coordinate pair is a state a setter can leave behind, so
refusing one here would make a world this package produces a world it will not read back.

**FR-11 — the digest moves, and is re-pinned rather than adjusted.** The pinned world's bytes and
its digest are both re-taken at the new version, together, from the same world.

## Acceptance

**AC-1** A guard member standing away from its post, holding no victim, with an empty candidate
list, holds its post as its destination after one tick.

**AC-2** The same member, once it has walked back, holds no destination and is unchanged on the
next tick: it does not oscillate and does not re-issue.

**AC-3** A guard member holding a victim and standing away from its post keeps its victim and gains
no destination.

**AC-4** A guard member of a group whose owner is not the local participant, holding a victim it can
no longer score, ends the tick with no victim **and** with its post as its destination — the release
and the walk home in the same tick, in that order.

**AC-5** A guard member holding a destination of its own that is not its post has that destination
replaced by its post.

**AC-6** A stand-ground member standing away from its post, holding no victim and scoring nothing,
gains no destination and does not move.

**AC-7** Every entity a freshly constructed world holds has its own cell as its post, including one
in no group, one not alive, and one under an order that is not a stance.

**AC-8** A group commanded to either stance mid-mission has every living member's post rewritten to
where that member is standing at that moment, and a member that is not alive is not written.

**AC-9** A member mid-crossing when a stance is issued is anchored at the cell it holds, and walking
home afterwards returns it there.

**AC-10** A world carrying a post survives a marshal-and-read round trip unchanged, including a post
outside the map's bounds and a post on a dead entity.

**AC-11** A version-23 form is refused, naming the version.

**AC-12** Deleting the walk home leaves the decision compiling and leaves every guard member in
every field it holds today.

## Properties

**P-1 — invariant: one anchor, one writer set.** A post changes only when a stance setter runs. For
any tick in which no world is constructed and no stance command is issued, every entity's post is
the one it held at the start of the tick.

**P-2 — invariant: the decision reads no clock and draws no random number.** This story adds no draw
from the world's generator, on any path, for any member. The number of draws a tick makes is
identical before and after it for every world.

**P-3 — negative invariant: the post is not a stance's property.** The field is on the actor, every
actor has one, and no code path asks which order a group is under in order to decide whether to
write it.

**P-4 — invariant: the drawing tier and the loaders learn nothing.** No package outside the
simulation gains a field, an import or a call on account of this story.

**P-5 — invariant: nothing outside the guard stance moves.** For any world holding no group under
the guard stance, every entity's every field after a tick is byte-identical to what this build
produces today.

## Decisions and divergences

- **DD-1 — the post is per-actor, not per-group.** Both stances anchor one, and a group's members do
  not share a cell. A group-level anchor would collapse a group into a point and would be a second
  spelling of the centroid, which is a different object with a different job — it clips candidates
  and it is nobody's destination.
- **DD-2 — the constructor writes it unconditionally, so it is never an input.** A caller cannot
  supply a post, and whatever an entity literal carries is overwritten. This is the actor state's
  own rule, and it makes the field in-bounds by construction without a clamp anywhere.
- **DD-3 — the walk home's gate is the cell alone.** The law's gate is the cell **or** the actor not
  standing on a cell centre. This tree has no second half to that test that means the same thing:
  it commits a mover's coordinates before it pays the crossing, so its mid-crossing flag means "owes
  ticks", not "is between two cells". *Divergence, disclosed:* a member at its post that still owes
  crossing ticks is re-ordered to the cell it is on in the law, and left alone here. The two differ
  only in stall counts and route churn.
- **DD-4 — the idle turn is cut, and the reason is the draw count.** Building it would take two draws
  from this world's generator per idle member per tick. Every combat roll after the first idle
  member would then depend on how many hostiles were standing still, which is a coupling of the
  determinism surface that deserves its own falsification rather than arriving inside a story about
  where a guard walks. *Divergence, disclosed:* an at-post idle guard does not turn.
- **DD-5 — the heal is cut with no divergence to disclose beyond its absence.** This build has no
  regeneration at all, so there is nothing for it to be inconsistent with.
- **DD-6 — `0099`'s D-3 is paid in half, and the half that is paid is the post.** That decision
  recorded the post as having no reader outside the per-actor guard. It has one: the guard stance's
  own arm, which this tree has had all along. The re-anchor latch's half stands unchanged and
  unpaid.
- **DD-7 — the version is bumped rather than the record reordered.** Two coordinates go on the
  record's tail where the last story's byte went, so no existing offset moves and no reader of an
  earlier field changes.

## Scope fences

**SC-1** No new draw from the world's generator, on any path (P-2).

**SC-2** No file outside `pkg/sim` changes (P-4).

**SC-3** The walk home is reached from the guard stance only, and that is checkable by deleting it
(AC-12).

**SC-4** No test transcribes a shipped map, and no task asserts a measurement against an install.

**SC-5** The full local gate is green: build, vet, gofmt, tests, and the repository's own asset,
document-budget and audit checks.

## Traceability

Every FR above has a paragraph in `plan.md` and a task in `tasks.md` naming it. Why each fact is
believed is in `provenance.md`; neither it nor `analysis.md` is needed to build from this file.
