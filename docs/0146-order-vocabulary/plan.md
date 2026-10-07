# 0146 — plan

## Shape

Six packages' worth of seam over one finished mechanism. Bottom up:

- **`pkg/sim`** — three command kinds and one generalised group-order writer. `groupOrder` already
  resolves WHO an order reaches and WHERE it is aimed and then hands the pair to `commandGroup` and
  `issueGroupDestination`; it gains a switch on the first command's kind that chooses the order
  byte and what runs beside the write. `commandGroup` gains one statement: the release drops the
  patrol ring too.
- **`pkg/game`** — two far sides, `stance` and `march`, built on `enqueue`'s own body: convert,
  append to the one pending queue, mark the entity commanded, step no world.
- **`pkg/ui`** — two seams appended to the loader tuple, four bindings, one armed-command byte on
  the viewer, one field on `order` and on `gesture`, one arm in `decide`, one readout row.

## Where each requirement lands

| Requirement | Where |
|---|---|
| FR-1 Guard | `pkg/sim` `commandStance` under `KindGroupStance`; `pkg/game` `mapWorld.stance`; `pkg/ui` the `Guard` binding and the `MapStance` seam |
| FR-2 Stand Ground | the same three, on the other value of the seam's bool |
| FR-3 March | `pkg/sim` `groupOrder`'s `KindGroupSwarmTo` arm; `pkg/game` `mapWorld.march`; `pkg/ui` the `March` binding, `armCommand` and the `MapMarch` seam |
| FR-4 Patrol | `pkg/sim` `commandPatrol`; the same far side and the same seam on the other value of its bool |
| FR-5 a player order ends a patrol | one statement in `pkg/sim` `commandGroup` |
| FR-6 an order names a group | `groupOrder`'s membership scan, now keyed on kind and tag; `pkg/game` `queueGroup`/`joinsGroup` |
| FR-7 the arms are exclusive | `pkg/ui` `armAttack`, `armCommand`, and the one resolving statement in `Viewer.command` |
| FR-8 no simulation type in `pkg/ui` | the two seam signatures and the `commandNone/Patrol/March` byte; the conversion is `mapWorld.stance` and `mapWorld.march` |
| FR-9 Swarm is cut | nothing is built; the sub-command keeps its script arm and gains no key |
| FR-10 the readout row | `pkg/ui` `PanelFieldOrder`, `readoutSubject.Aimed` and `AuthoredReadoutLayout` |

## Decisions

**DD-1 — three kinds, not one carrying an order byte.** `Command` has `Kind`, `Entity`, `X`, `Y`
and `Group`, and `Group` is the tag. A single kind serving all four orders would have to carry the
order somewhere, and the only free field is `X` — which Patrol and March need for the cell. So:
`KindGroupStance` (the order rides in `X`, the same 32 bits under another name `KindAttack`'s
victim already is), `KindGroupPatrolTo` and `KindGroupSwarmTo` (both carry a cell). Three kinds is
also what the file already does for two things that differ only in a flag — `KindKill` and
`KindDamage` are separate kinds, not one with a bool.

**DD-2 — the generalisation lives in `groupOrder`, not in three sibling functions.** Membership
resolution, the tag scan and the consumed marking are identical for all four orders and are already
written once. Three copies of that scan is three places for "an order names a group" to drift. What
the switch chooses is: the order byte, whether the formation distribution runs, whether posts are
anchored, and whether the members are handed to the actor layer.

**DD-3 — the posts are anchored beside the write, not inside `commandGroup`.** `cmdGroupGuard` and
`cmdGroupStandGround` anchor posts beside their own order write; the player's stance arm does the
same, over the same members, in the same walk. Putting it inside `commandGroup` would anchor a post
on every plain move as well, which is a behaviour no arm has.

**DD-4 — `clearPatrol` goes in `commandGroup` and nowhere else.** That function is already "the
whole of what a player order does that a plain walk never did": it takes every member out of
whatever group it stood in. A ring is exactly such a standing arrangement, so it goes out on the
same statement the group membership does — which makes FR-5 true of every player order there is and
of every player order there will be, rather than of a list someone keeps. The player's own Patrol
arm calls `commandGroup` FIRST and installs its ring after, so the clear cannot undo the order that
caused it.

**DD-5 — two seams, each carrying a bool.** `MapStance(entity uint32, guard bool)` and
`MapMarch(entity uint32, patrol bool, x, y int)`. The split is between orders that need a cell and
orders that do not, which is the one real difference in payload; within each, a bool is the whole
choice, on `MapAffect`'s own grounds — this side names the things it can ask for, not the command
stream that answers them. A single seam would give the two cell-free orders an `x, y` they never
mean, which is what every rejected widening in `flow.go` was rejected for.

**DD-6 — one armed byte on the viewer, not two bools.** `commandArmed` holds none, patrol or march.
Two independent bools would have a representable state — both up — with no defined press, and FR-7
would be a rule instead of a shape. The attack arm stays its own flag because it is a different
control with a different gate (`canArmAttack` reads the local participant); the exclusion is two
statements, each lowering the other.

**DD-7 — `decide` gets one more branch inside the right-press arm, beside the armed attack.** Same
placement and same reason: a press is ONE outcome, and putting the command arm beside the attack
arm inside that branch keeps the four outcomes total. Unlike the attack arm it does not fall
through when it misses — a March or a Patrol names a CELL, so the only thing that can make it miss
is the extent test the plain move already performs, and it shares that test.

**DD-8 — the tag scan in `pkg/game` is generalised over kind rather than copied.** `joinsGroup`
already answers "does this order join the group being assembled" by walking the pending queue; it
takes the kind as an argument now, so a stance press and a move press cannot land in one group.

## Risks

**R-1 — the digest moves for a world where a patrolling unit is commanded.** FR-5 is a behaviour
change reaching hashed state (`ActorState`, the ring, the leg). It is not a format change: no field
is added, removed or resized, and the byte-form version does not move. The set of worlds affected
is exactly those where a script's Patrol sub-command has run and the player then orders one of its
members — which before this story produced a unit that ignored him.

**R-2 — the loader tuple grows from nine to eleven.** Every destructure changes. Accepted on the
file's own stated grounds: appending buys that no existing member changes meaning.

**R-3 — Guard's walk home needs posts that mean something.** A fresh command group whose members'
posts were never anchored would walk them home to wherever the map placed them. DD-3 is what stops
that, and AC-1 is what witnesses it.

## Order

1. `pkg/sim`: kinds, the `groupOrder` switch, the stance/patrol/march arms, `clearPatrol` in
   `commandGroup`. Tests for each arm and for the cancel.
2. `pkg/ui`: seams, bindings, the armed byte, `decide`, the readout row, app dispatch.
3. `pkg/game`: `stance`, `march`, the generalised tag scan, the loader tuple.
4. Gate, the census argv, the run.
