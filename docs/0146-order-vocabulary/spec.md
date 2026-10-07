# 0146 — the player's own order vocabulary

**Intensity:** spec-first / breadth-over-polish. **Terrain:** brownfield seam over a finished
mechanism. Every behaviour this story exposes is already built, tested and shipped inside
`pkg/sim`; what is missing is a way for the player to reach it. Nothing about how an order behaves
once issued is decided here.

## Why

The simulation can put a group under six standing orders. A scenario script can ask for all six —
`World.cmdGroupOrder` dispatches Guard, Swarm, Stand Ground, Move, Swarm 2 and Patrol, each to its
own writer. The player can ask for exactly two: walk there, attack that. Everything the map's own
script can tell a group to do, the person playing the game cannot.

This is the cheapest win left in the front-end. The arms are done; the gap is a command kind, a
key and a seam.

## Scope

**In:** three new standing orders the player can issue over his current selection — Guard, Stand
Ground and March — plus Patrol, which needs a cell; the keys they are bound to; the two front-end
seams they cross; the command kinds that carry them into an advance; and the rule that any player
order takes its units **off** patrol.

**Out of scope:** the Swarm sub-command (cut — FR-9). The four published sub-commands this build
does not implement at all (`10 Attack`, `11 Defend`, `15 Follow`, `17 Roam`): they have no arm to
expose and giving them a key would be inventing behaviour. Every existing script arm — none of
their bodies moves. The byte form and its version: this story adds no field to any record. Panel
buttons: the vocabulary is on keys, and a control surface for it is a later story.

## Functional requirements

**FR-1 — the player can put his selection under Guard.** One key, acting on exactly the units the
selection still holds alive-or-downed, in ascending id. Every unit named lands in ONE fresh command
group held at the Guard order, each member's post anchored at the cell it presently stands on, and
the group's notice base frozen from that membership's own geometry. Thereafter the engagement pass
runs Guard's own arm for it: candidates clipped to a circle about the group's centre, and members
that end a fight with nothing to score walk home to the post this order anchored.

**FR-2 — the player can put his selection under Stand Ground.** One key, same membership rule, same
one fresh command group, same post anchoring. Thereafter the engagement pass runs Stand Ground's
own arm: no clip, nothing engaged that the member could not already strike, no walk home.

**FR-3 — the player can send his selection somewhere under March.** A key ARMS the order and the
next secondary press supplies the cell, exactly as the attack key already arms and the next
secondary press supplies the victim. The units land in one fresh command group at the Swarm 2 order
with that cell as the group's commanded cell, and are distributed to it by the same formation walk
a plain move uses. The difference the player sees is in the engagement pass: a March group engages
what it scores whether or not it has arrived, where a Move group's own arm engages only a member
that has.

**FR-4 — the player can set his selection patrolling.** A key arms, the next secondary press
supplies the far cell. Every named unit is released into one fresh command group whose order is
none — the group is not decided for at all — and each member is handed to the actor layer at
patrol, with a two-cell ring from where it presently stands to the pressed cell, clamped into the
map, standing on the far leg. Thereafter it walks between the two forever.

**FR-5 — any player order ends a patrol.** A unit taken into a fresh command group by any order —
the plain move, the group move, Guard, Stand Ground or March — leaves the patrol state, with its
ring and its leg cleared in the same statement. Before this story a patrolling unit could not be
countermanded at all: the actor pass re-issued its ring every tick, so a move order was applied and
then silently overwritten one phase later.

**FR-6 — an order names a group, not a unit.** All four orders carry the group tag the move order
already uses: every command a single key press or a single armed press emits is one member of one
group order, applied once, in one advance. A selection of one takes no other path — a group of one
is its own centroid.

**FR-7 — the arms are exclusive.** Arming a command lowers the attack arm and arming an attack
lowers the command; at most one is up when a secondary press is resolved, so no press is claimed by
two arms or by neither. A secondary press with no arm up is the plain move it has always been.

**FR-8 — the front-end cannot name a simulation type.** The two new seams carry scalars and
booleans only, and neither the order byte nor the sub-command number appears anywhere in `pkg/ui`.
Which of the law's orders a front-end request becomes is decided on the far side.

**FR-9 — Swarm is not exposed, and that is a decision rather than an omission.** The Swarm
sub-command writes a commanded cell onto the group record and NOTHING ELSE: it gives no member a
destination, and the only reader of what it writes is the engagement pass's own swarm arm, which
sends a member there only if that member scores no candidate at all and is not already idle on the
cell. On a quiet map — no hostile in reach of anyone — a player who pressed a Swarm key would watch
his units stand still for a tick and then trickle toward the cell out of formation. Beside March,
which he reaches with the same gesture and the same cell and which actually walks the selection
there in formation while still engaging on the way, Swarm is the same verb minus the walk. It stays
available to scripts, where a mission author knows what he is asking for.

**FR-10 — what the readout states.** The map screen's readout gains one row: which command is
armed, or that none is. Both states are stated and neither is an absence, on the attack row's own
grounds — a row that vanished when nothing was armed would make "the key did nothing" and "the
readout has no such row" the same picture.

## Acceptance

**AC-1** Pressing the Guard key with three units selected leaves all three in one command group,
that group's order at Guard, every member's post at the cell it stood on, and the group's base at
`noticeBase` over that membership.

**AC-2** Pressing the Stand Ground key does the same with the Stand Ground order and writes no
base of its own — the group is fresh, so its base is frozen at construction like every other.

**AC-3** With March armed, a secondary press inside the extent puts the selection in one group at
Swarm 2, that group's commanded cell at the pressed cell, and every member holding a destination
the formation walk computed; a press outside the extent issues nothing.

**AC-4** With Patrol armed, a secondary press puts every named unit in the patrol actor state with
head at its own cell, tail at the pressed cell clamped into the map, and the leg on the tail; the
group they land in is held at order none.

**AC-5** A patrolling unit given a plain move order stops patrolling: the actor pass does not
re-issue its ring, and it walks to the ordered cell. Reverting FR-5's clear alone puts the unit
back on its ring within one advance.

**AC-6** A frame that sees both the attack key and a command key ends with exactly one arm up.

**AC-7** Every key this story binds was unbound before it: the four letters do not appear in
`readAppInput` or in the viewer's own pan keys before this story.

**AC-8** A loader that installs neither new seam still drives the whole map screen: the keys reach
nothing and nothing panics.

## Properties

**P-1** No order reaches a world except through an advance. The seams append to the same pending
queue the move, the attack and the blow already use, and `sim.Step` is reached from one call site.

**P-2** The command vocabulary is closed at the seam. `pkg/ui` names four things it can ask for —
guard, stand ground, patrol, march — as two booleans, and no third value type crosses.

**P-3** Nothing this story adds is state. Commands are correlated inside one `Step` and stored
nowhere; the byte form, its version and the digest layout are untouched.

## Authored, and on what grounds

**A-1 — that the player gets this vocabulary at all.** The research decodes the group command as
the map script's own authoring surface. Nothing we hold says the original's own control panel
issues these same eleven literals, and no claim is made here that it does. What is decoded is the
vocabulary and every arm's behaviour; that the person playing may reach four of them, and by which
gesture, is ours.

**A-2 — the four keys.** One rule, four keys: the order's own initial where it is free, otherwise
the word's next free letter. `G` is the pick-up key, so Guard takes `U`; `S` is the camera's
pan-down, so Stand ground takes `T`; `P` is free, so Patrol takes it; `M` is the minimap toggle and
`A` the camera's pan-left, so March takes `R`. Every one of the four was unbound (AC-7).

**A-3 — that March and Patrol arm rather than open a cursor mode.** The attack key already arms and
spends on the next secondary press, and that shape is reused whole rather than a second one
invented. It costs the player one press either way and costs this package no new gesture kind.

**A-4 — that the player's Patrol builds a command group.** The script's own Patrol sets the placed
group's order to none and leaves membership alone. A player's selection can span several placed
groups, so his patrol takes its members into one fresh command group at order none — the same
release every other player order performs. The consequence is disclosed: a unit set patrolling by
the player has left the group the map placed it in, and a script that later orders that placed
group does not reach it.

**A-5 — that a player order clears the patrol state (FR-5).** The law's own behaviour here is not
decoded and no claim is cited for it. The alternative is an order the player cannot give: the actor
pass re-issues a patroller's ring every tick, so without this an order to a patrolling unit is
applied and undone inside one advance. The script's own five setters are left exactly as decoded —
this clears on the PLAYER'S path, in `commandGroup`, which is the player-owned construct, and no
script arm gains a statement.
