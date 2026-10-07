# Spec — a player order builds a group

## Terms

**Placed group** — the group id the map gave an actor. It is a datum of the map, it is what the
mission script's own group predicates and commands name, and no act in this build changes it.

**Command group** — a group an order built, named by an id that is not any placed group's. An actor
holds at most one; an actor holding none is in its placed group.

**Effective group** — the group an actor is in for the purpose of a decision: its command group when
it holds one, its placed group otherwise.

**Player order** — a command in the tick's own command slice. This build has four kinds; two are
moves — the single move and the group move — and those two are what this spec calls a **player move
order**.

**Group record** — the per-group state a decision reads: the group's stored order, the notice base
its guard clip uses, and the cell a command last named it. One record per (owner, group id).

**Walk home** — the guard stance's own tail, which sends an idle member with no victim to its post.

## Why

A commanded unit obeys and then undoes the order. It walks where it was sent, arrives, and walks
back to the cell it was placed on. The owner reported it of every monster he tries to control, and
the tenth mission reproduces it exactly: a hostile ordered five cells away arrives at tick 60 and
holds its own post as a destination at tick 61.

The walk home itself is not the defect — an idle guard returning to its post is what an unscripted
hostile population does. The defect is that a commanded actor is **still in the guard group** when
its order finishes. While it walks it is skipped, because the skip is keyed on holding a
destination; the moment it arrives the skip stops applying, the guard group takes it back, and the
guard tail sends it home. A test for "under command" in the walk home would therefore change
nothing anyone can see: at the moment the walk home fires there is no command left to detect.

What a player order does in the thing being reconstructed is not to write on the group the actor is
in. It **builds a new one** and puts the commanded actors in it, and the order routine leaves its
own byte on that new group — a move leaves the move order. A commanded actor is not a guard with an
exception carved out of the guard tail. It is not a guard.

So this story adds the one thing the build has never had: a group whose author is a player.

## Functional requirements

**FR-1 — every actor carries a command group.** It is a group id, zero means the actor holds none,
and every actor has one at every moment: an actor in no slot, one that is not alive and one that was
never commanded all carry it exactly as a commanded one does. Zero is not a chosen sentinel — FR-5
makes it an id no command group can ever be given.

**FR-2 — the effective group is what a decision partitions, freezes and reads by.** Every place the
engagement layer asks which group an actor is in reads the effective group: the partition a decision
is built from, the key set a construction freezes records for, a group's living-member scan, and the
lookup that decides whether a walking actor is skipped. Nothing else in the build reads a command
group.

**FR-3 — the placed group is not written by anything.** The mission script's alive count, its owner
hand-over and its group-order command read the placed group and are unaffected by every requirement
here. A player's click may not move a group's alive count, because a trigger comparing that count
against zero is how a campaign says a place is cleared.

**FR-4 — a player move order builds a command group.** Both player move arms take every actor they
command out of whatever group it was in, put all of them into ONE new command group, and leave that
group at the **move order**. The actors' own destinations are written exactly as they are written
today; this requirement adds the group and changes no destination.

**FR-5 — the id.** The command floor is one above every group id the mission script names, and at
least one; a world with no script has a floor of one. The command group's id is the lowest id at or
above the floor that no actor's effective group names, taken AFTER the commanded actors have been
released from the group they were in — so an id a command group has been vacated from is taken
again by the next order rather than left behind. An id at or above the floor is one no script node
can name and one no placed group carries, so a command group is reachable by nothing but this
build's own player orders.

**FR-6 — the record.** One record per distinct owner among the commanded actors, keyed by that
owner and the one new id: the move order, the cell the order named carried whole, and a notice base
frozen from that owner's share of the commanded actors by the same computation construction uses. A
record already standing at that key is overwritten in every one of those fields. The record list
stays ascending by (owner, group id), which is what its own decoder requires of it.

**FR-7 — an actor in no slot takes none of this.** It gains no command group and no record, on the
same ground it belongs to no group today: it has nobody to decide for it.

**FR-8 — a command group is never left.** Nothing in this build returns an actor to its placed
group, and this story adds no such act. A commanded actor whose order has completed is decided for
by the move order's own arm every tick: it engages a candidate that arm's scorer admits, and
otherwise it stands. It does not walk home, because the walk home belongs to the guard stance and it
is not under one.

**FR-9 — the byte form carries it.** The form's version moves to **32** — the number the widening
landed on after two merges forward, see `verification.md`; the requirement is the widening and the
number is the seam it lands on — and the actor record grows by four bytes at its tail for the
command group. Every value is carried whole and none is refused: zero is the ordinary state and
every other id is one a player order can leave. A form at any other version is refused, unchanged.

**FR-10 — no other order builds a group in this build.** The attack order leaves the actor in the
group it was in, and so does every mission-script act. The thing being reconstructed builds a group
for those too and leaves each at its own byte; this story builds the seam and one arm.

## Acceptance criteria

**AC-1** — On the tenth mission, against a lawful install, a hostile commanded away from its
placement never afterwards holds its own post as a destination. The same drive before this story
gives it that destination one tick after it arrives.

**AC-2** — A synthetic world with no install: a guard-group actor off its post, commanded to a cell,
is not walked home by the decision that follows its arrival — and an actor of the same world that
was NOT commanded still is.

**AC-3** — A commanded actor's placed group id is the same before and after the order, and a script
alive-count check over that group counts it exactly as it did before.

**AC-4** — Two actors commanded by one group move end in ONE command group, at the move order, with
one record per distinct owner among them.

**AC-5** — An actor commanded twice takes the same id both times: the id is released before it is
taken, so nothing accumulates.

**AC-6** — No command group id equals any placed group id in the world, and none equals a group id
the mission script names.

**AC-7** — An actor in no slot gains no command group and adds no record.

**AC-8** — The record list is ascending after any sequence of orders, and a world that has taken
orders survives a round trip through its own byte form unchanged.

**AC-9** — A form at the version this one replaced is refused by version alone, and the refusal
names both numbers.

**AC-10** — Deleting the call that builds a command group from the two move arms leaves the package
compiling and every other behaviour intact — which is what makes FR-4 a seam rather than a
condition spread through the decision.

## Design decisions

**DD-1 — a second id, not a moved one.** The actor could have had its placed word rewritten, which
costs no new state at all. It is refused: three script arms key on that word, one of them the alive
count whose triggers decide missions, and a player's click may not reach them.

**DD-2 — zero means none, and no flag byte.** Because FR-5's floor is at least one, no command group
can be given the id zero, so zero carries the whole distinction. This is the one place in the record
where a coordinate-style "there is no unset value" rule does not apply, and it is the allocator that
makes it not apply.

**DD-3 — the floor is above the script's ids, not above the map's.** Above the map's alone would
still let a script node name a command group; above the script's is what makes FR-5's last sentence
true. Holes below the map's own ids are then reusable, which is what keeps the record list bounded.

**DD-4 — the id is taken after the release, not before.** Otherwise an actor commanded twice takes a
fresh id each time and the record list grows with the number of clicks.

**DD-5 — the commanded cell stored is the cell the order named**, not the offset cell any one member
was sent to. The order names one cell; where each member goes from it is the distribution's business.

**DD-6 — the move order, and no other.** The build has an arm for it already, and that arm neither
clips to a notice radius nor walks anybody home. The other arms a player order could leave are named
in FR-10 and not built.

## Properties

**P-1 — determinism.** Nothing added here reads a clock, a map iteration order or a random draw. The
id is a function of the world's own actors and script; the record set is a function of the order.

**P-2 — no actor is decided for under an order this build has no arm for.** The only order a player
order installs is one the decision already dispatches.

**P-3 — the record list stays sorted and its keys stay unique**, at every moment a form could be
taken.

## Scope claims

**SC-1 — the guard stance is untouched.** No requirement here changes when a guard walks home, only
who is a guard.

**SC-2 — nothing outside this package reads a command group**, so the front end, the loader and the
mission tool are unchanged by it.

**SC-3 — the placed group's own record is left standing** when its last actor is commanded away.
Nothing reads a record no actor names, and the decoder already declines to police that.
