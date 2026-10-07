# Spec — a group's decision releases what it does not renew

**Intensity:** spec-anchored / static. **Terrain:** brownfield — the decision, the approach and the
attack cycle all ship, and a defect reported from play is why this story exists. **Threshold: High**
— this changes what every group decision does to every member.

## Terms

- **A victim** is the unit an attacker has been given, with the flag that says it holds one. **A
  destination** is the cell a unit is walking to, with its own flag. Both already exist here.
- **The engagement decision** is the pass that runs once per full tick, partitions the world's
  living owned entities into the groups the map placed them in, and for each group builds a
  candidate list from its members' sight, clips it under one stance, scores every candidate for
  every member, and gives each member the cheapest one it will take.
- **A member scores nothing** when no candidate in the list beats the seed the selection starts at.
  A candidate the preference table vetoes, and — under the second stance — one standing past reach,
  both score exactly the seed and so beat nothing.
- **Stance** is which of the two group behaviours a group is under. This build derives it from the
  group's owner slot and this story does not change that derivation. The two are named here
  **guard** and **stand ground**.
- **Release** is the state change this contract introduces: a member ends up holding no victim and
  no destination. It writes no new field.
- **The approach** is the move loop's arm that aims an attacker at its victim's current cell every
  tick, giving it a destination. It ships and does not change here.
- **Both roots** are the two lawful installs. A claim made on one is not made.

## Why

A member is given a victim by a decision and nothing ever takes one away. The approach then walks it
toward that victim every tick for the rest of the mission, so a single acquisition made in the first
hundred ticks is a standing instruction to cross the map. Over 28 campaign missions stepped 2000
ticks with **no commands at all**, 79 of 2357 entities move 501 cells between them and 35 die — a
brawl nobody ordered, on maps whose authored population is meant to stand where it was placed.

The build believes this is correct: a landed clause states that a member holding no candidate is
left exactly as it was and that no decision at either group order breaks off an attack. That clause
is wrong, and this story reverses it. **0086 FR-20 is superseded by FR-3 below.** It was written
as a decoded rule rather than as a divergence, so its reversal is a correction to a landed contract
and is named as one here.

## Scope

**In scope.** What a decision does to a member it does not give a candidate to; what it does to a
group whose candidate list is empty; and what a released member is left holding.

**Out of scope.** The walk home and the post it needs. The idle turn. The per-actor state machine
and its break-off rule. Roam, patrol, follow. The notice radius. The mission script's group command.
Whether a group should attack a corpse — it should. The remembered-attacker memory that keeps a
struck member's assailant in sight for a bounded number of decisions.

## The contract

### The release

**FR-1** A member that scores nothing is **released**: it holds no victim, no attack phase and no
attack countdown, and it holds no destination, no stall count and no stored route. The four attack
fields and the three order fields go together — a state with only some of them cleared is one the
byte form refuses.

**FR-2** The release is a **no-op on a member that held no victim**. A member holding a destination
and no victim keeps that destination. This is what makes the release safe to apply inside a loop
that also visits members walking under a player's order, and it is the whole of the coupling
between this story and any story about commanded units.

**FR-3** A member that scores nothing is therefore **not** left as it was. Both halves of
0086 FR-20 fall: a decision does break off an attack, and the member it breaks off is not left
untouched.

**FR-4** A released member is left **standing where it stopped, facing where it faced**. Nothing is
put in the order's place. **Disclosed divergence:** the original replaces the ended pursuit with a
walk back to the member's own post, or — at the post and idle — with a turn on the spot. This build
has neither the post nor the turn; both are out of scope and this is the half of the pair it has.

### Which members it applies to

**FR-5** The release applies under the **guard** stance and not under **stand ground**. Under stand
ground a member that scores nothing keeps whatever it held, exactly as today.

**FR-6** FR-5 is a rule about the member's **owner**, expressed through the one stance this build
derives from it. The original spares a human participant's unit and rewrites every other member's
order; this build has exactly one human participant, seated at the slot whose groups are the only
ones under stand ground, so the two tests select the same members here. **Named seam:** a build with
a second human participant, or one where stance stops being a function of the owner slot, must split
them.

**FR-7** A member of a group whose candidate list is empty is released on the same terms, by the
same rule: an empty list gives every member nothing to score, which is the FR-1 case. No separate
statement of it exists in this contract or in the code that implements it.

### The count edge

**FR-8** A candidate list is treated as empty when its length **narrowed to a byte** is zero, and a
group is treated as having no members when its member count narrowed to a byte is zero. So a list of
exactly 256, 512, … candidates releases every member instead of engaging any.

**FR-9** FR-8 is a **code-carried customisation limit**, not a behaviour: 256 simultaneously visible
hostile candidates is unreachable on any map this build can load, and no member count can be zero
because a group exists here only when a living member was found for it. Lifting it changes this
build's code and no shipped file's bytes. Recorded because raising an actor cap reaches it.

### The tie rule, restated because it decides the release

**FR-10** A candidate is taken only when its cost is **strictly less** than the best so far, and the
best starts at the same value a vetoed candidate scores. So a member all of whose candidates are
vetoed takes none and is released, and among equal costs the **first in list order** wins — list
order being ascending entity id, for the parked corpses too. This restates a rule the build already
implements, and is here because FR-1 is defined by its failing.

### What does not move

**FR-11** A member that scores something is engaged exactly as today: same partition, same candidate
list, same clip, same scorer, same order, same rule that an engagement naming the victim already
held leaves the attack cycle alone.

**FR-12** The canonical byte form is **unchanged** — same version byte, same field set, same
encoding, same refusals. Any given world's digest is the digest it had before this story. A world
stepped under this story may differ from the same world stepped before it, and that is the story.

**FR-13** No entity field, no world field and no constructor input is added, removed or widened, and
no writer of a destination either: the release only ever **clears** one, so the pinned writer set of
the destination flag is untouched and is not edited.

### What the release cannot tell apart, and the two landed criteria that go with it

**FR-14** A victim written by a player's **attack command** and one written by a decision are the
same fields set by the same one function. The release cannot distinguish them and does not try:
under guard, a commanded attack ends at the first decision whose member scores nothing. That is the
original's own shape — its arm rewrites the member's order whoever wrote it — and what protects a
player's units is FR-5's stance, not a mark on the command. It is observable only on a world this
build's content cannot produce: every unit a player can command stands at the slot FR-5 exempts.

**FR-15** Two acceptance criteria of `0086` are **superseded by FR-3**, not found faulty. AC-13 —
*making a relation friendly does not end an attack already issued* — now holds under stand ground
and fails under guard, which is the reversal this story is; its test moves to the stance where the
claim survives and the guard case is added with the opposite expectation. AC-12's cadence comparison
uses a peaceful-relation world as its **control**, which must stand at the exempt stance to remain
one. No assertion about cadence, striking or re-issue is weakened: what changes is the owner each
world is built at, and `verification.md` records both before and after.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | a guarding member holding a victim, and a world in which that victim has become unscoreable — vetoed by class, or gone from the list | a decision is taken | it holds no victim, no attack phase, no countdown, no destination, no stall count and no stored route |
| **AC-2** | the same member and world | the tick after the release, and every tick after that | it does not move, and it strikes nothing |
| **AC-3** | a guarding member holding a victim it can still score | a decision is taken | it keeps that victim and its attack cycle is untouched |
| **AC-4** | a guarding member holding a destination and **no** victim | a decision is taken and it scores nothing | it keeps its destination, its stall count and its route |
| **AC-5** | a guarding group whose candidate list is empty | a decision is taken | every member holding a victim is released, and every member holding none is unchanged |
| **AC-6** | a stand-ground member holding a victim that has stepped past reach | a decision is taken | it keeps that victim, and keeps advancing on it |
| **AC-7** | a guarding member whose only candidate is vetoed by the preference table | a decision is taken | it is released, and the veto is what caused it — shown by the same world with the veto absent, where it is not |
| **AC-8** | a guarding group of one member and a candidate list of exactly 256 scoreable candidates | a decision is taken | the member is released rather than engaged |
| **AC-9** | the same group with 255 candidates, and again with 257 | a decision is taken | the member is engaged in both |
| **AC-10** | a world in which every decided member scores something | it is stepped for many ticks | every entity's every field matches the same world stepped on the tree before this story |
| **AC-11** | any world | it is encoded, decoded and hashed | the round trip is byte-identical, the version byte is the one that already shipped, and the digest equals the digest the same world had before this story |
| **AC-12** | a released member | its world is encoded and decoded | it round-trips, holding neither order — the release produces no state the byte form refuses |
| **AC-13** | the simulation package | it is searched for writers and clearers of a victim | the setter is one function, the release is one function, and the release is the only clearer reached from a decision |
| **AC-15** | a guarding unit given a victim by an attack **command**, in a world where nothing is hostile to it | a decision is taken | it is released — the command is not privileged |
| **AC-16** | the same unit and command at the exempt stance | a decision is taken | it keeps the victim and goes on striking, which is 0086 AC-13 where that claim survives |
| **AC-17** | the pinned set of writers of the destination flag | the package is rescanned | it is the set that shipped before this story, unedited |

## Error cases

**AC-14** No new failure and no changed refusal message. The release writes only values the byte
form already accepts on an entity holding no order, and is reachable only from a pass that runs
before the tick's occupancy scratch exists, so it cannot leave that scratch disagreeing.

## Success conditions

**SC-1** The no-command census, on **both roots**: 28 campaign missions, 2000 ticks, no commands.
Entities that moved, cells travelled and deaths, before and after. The before figures are 79 / 501 /
35 of 2357. **If they do not fall, this is not the fix either, and that is what gets reported.**

**SC-2** The clubman on mission 10, before and after, on both roots: the cells it is dragged through
and what it acquires. Recorded whatever it becomes, including unchanged.

**SC-3** Mission 10's outcome and the tick and arm that decide it, before and after, on both roots.
**No test expectation, threshold or predicate is edited to move it** — falsifiable from this story's
own diff.

**SC-4** A release is witnessed by reverting it: with the release removed the world is measurably
different and a named test fails. Every FR that claims a behaviour has a test that fails when the
line implementing it is taken out.

**SC-5** The population that keeps fighting is named exactly and is not empty: by FR-5 and FR-11 a
guarding member beside a hostile it can still score does not release. Measured on a real mission.
