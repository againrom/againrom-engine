# Spec — the player can order an attack

**Intensity: spec-anchored / static. Terrain: brownfield** for the map screen's press path, the
loader's seam tuple and the simulation's attack arm — all three ship today — and **greenfield** for
the armed mode and for closing the distance.

The simulation already resolves a blow: a charge/relax cycle with a jitter, a to-hit roll, damage
base plus spread, flat absorption and a killing blow, applied by an attack command it already
defines. **Nothing issues one.** No key, no press and no seam in this tree produces an attack
command, so no unit has ever attacked in a running game and the whole of the combat code is reached
only by tests. This story is the player's half of that: an order that names a victim, reaches the
world through a seam of its own, and closes the distance to what it named.

It is the PLAYER'S half alone. Nothing here decides what an unordered unit does, what a struck unit
does, or what a unit does when its order runs out — those are a later story's whole subject, and the
one thing this story asserts about them is that it adds none of them.

## Functional requirements

- **FR-1 — an attack is ARMED before it is aimed, and the gate is OWNERSHIP.**
  The map screen holds one flag: attack armed, or not. A map-screen key raises it, and raising it
  succeeds only when **both** hold:
  - the selection holds at least one unit the current snapshot still holds alive or downed — the
    same set an order is emitted for; and
  - the **primary** of those — the lowest such id — is owned by the local participant. The
    front-end is TOLD which owner that is, as one value beside the selection; **zero is no local
    participant established**, and a zero compares against nothing, so the control is open. Roster
    slots are 1-based, so no slot is shadowed by that zero.

  **The gate reads an OWNER and never a unit class.** No class, kind, art, name or capability of the
  selected units is consulted, in either direction.
  A press that fails the gate arms nothing and leaves the flag exactly as it was. Pressing the key
  while armed lowers it, so a press made by accident is undone by the key that made it.
  **The gate is asked at the key and nowhere else.** A selection replaced while the flag is up —
  by a tap or by a box — neither lowers the flag nor re-asks the gate, so the units a consuming
  press orders need not be the units the gate was asked about. That is where the decoded routine
  asks it, and stating it here is what stops a second, invented asking from appearing at the press.
  The flag is the FRONT-END'S OWN: nothing pushes it anywhere, no seam carries it, and it is
  dropped whenever the map screen is left — so a map opened after one is never armed.

- **FR-2 — one armed press makes ONE of two orders, and what is under the cursor decides which.**
  While the flag is up, the next secondary press is the **consuming press**:
  - it names a unit (FR-3) then the frame issues an **attack** order for every selected unit the
    snapshot still holds, each naming that unit as the victim, in ascending id, and it issues no
    move order at all;
  - it names none, and the frame issues exactly the move orders that press issues today, unchanged.

  The flag is lowered by that press **whatever it produced**, including a press that produced
  nothing — a press over an empty selection, one resolving outside the map extent, one every
  selected id has been dropped from. One press, one arm, spent.
  While the flag is DOWN the secondary press is exactly what it is today: move orders, never an
  attack, whatever is under the cursor.

- **FR-3 — which unit a press names is the hit test this front-end already has, and no ownership is
  read.** The victim is the unit whose DRAWN rectangle holds the pressed point — the same test, the
  same relief lift and the same lowest-id tie rule a selecting tap uses — and a dead unit is not a
  candidate. Nothing about who owns the attacker or the victim is consulted at the press: an attack
  order may name any unit the press catches, the ordering player's own included.

- **FR-4 — an attack order leaves the front-end through a SEAM OF ITS OWN.** Two entity ids — the
  attacker and the victim — and nothing else crosses. It is a THIRD seam beside the move seam and
  the blow seam and not a widening of either; it is **appended last** to the loader's tuple, so no
  existing position moves; and it may be nil, a front-end holding none issuing no attack and that
  not being an error. It names no simulation type: two `uint32` name none.

- **FR-5 — the far side turns one attack order into the simulation's own attack command, and does
  nothing else.** It appends to the SAME pending queue the move orders use, and it marks the
  attacker **commanded** — an attack is an order, unlike a blow, and a unit that has taken one must
  stop taking scripted ones. It steps no world: an attack order with no advance behind it leaves
  every world field, the byte form and the digest exactly where they were.

- **FR-6 — an attack order CLOSES THE DISTANCE.** *This supersedes the shipped clause that an
  attacker holds a victim or a destination and never both (0064 FR-2).* An alive attacker holding a
  victim the world still holds and has not killed is aimed at that victim at every one of its turns,
  after any crossing it owes and before it moves:
  - **out of reach** — its destination becomes the victim's own current cell, so a victim that moves
    is followed;
  - **in reach** — its order ends there and it does not walk. The stop distance IS the reach, and
    it is the reach the blow is refused outside of, read in one place.
  - **victim gone or dead** — its order ends the same way. A unit walks only while it has a living
    victim it cannot yet strike.

  The other half of 0064 FR-2 is unchanged and still holds: a move order, alone or in a group order,
  ends whatever fight its mover was on, whole and with no residue.
  **No state is added.** The destination an attacker walks to is the destination field every mover
  already carries, so the world's field set, the byte form and its version are untouched.

- **FR-7 — the armed flag is visible.** The debug readout states whether an attack is armed, on a
  row of its own. It is the only feedback this story ships; no cursor changes and nothing is drawn
  over the map.

- **FR-9 — the fan-out is UNBOUNDED, deliberately and with the bound stated.** A press orders every
  selected unit the snapshot still holds, however many that is. The engine's own command record
  stops appending selected ids at **253** and drops the rest silently, in an order that is a hash
  walk rather than an id order, so above that it orders an unpredictable subset. This build orders
  all of them, deterministically. The divergence is above 253 selected units and nowhere below it.

- **FR-8 — nothing else changes.** The unarmed secondary press, the selecting tap, the box, the two
  blow keys, the cadence keys, the notice and every diagnostic answer exactly as they do today, and
  `pkg/ui` still names no simulation type.

## Acceptance criteria

- **AC-1** With no local participant established, arming over a selection holding one live unit
  raises the flag; arming over an empty selection, and over one every id of which the snapshot has
  dropped or killed, leaves it down; arming twice leaves it down.
- **AC-1a** With a local participant established, arming raises the flag when the primary present
  id's owner is that participant and leaves it down when it is not — and the same two selections
  answer identically whatever the units' classes, art or names are.
- **AC-2** An armed secondary press over a drawn unit yields one attack per selected present unit,
  each naming that unit, in ascending id, and no move order.
- **AC-3** An armed secondary press over empty ground yields exactly the move orders the same press
  yields unarmed — same ids, same cell, same order.
- **AC-4** The flag is down after any secondary press made while it was up, including a press that
  yielded nothing at all.
- **AC-5** An unarmed secondary press over a drawn unit yields move orders and no attack.
- **AC-6** Where two drawn rectangles hold the pressed point the victim is the lower id; a dead unit
  under the point is not a victim and the press falls through to the move arm.
- **AC-7** The seam receives the attacker and the victim; the far side appends exactly one attack
  command per call to the pending queue, marks the attacker commanded, and advances no world — the
  digest before and after is the same.
- **AC-8** An attacker ordered onto a victim several cells away walks to it and lands a blow that
  moves the victim's health. One ordered onto an adjacent victim never moves at all.
- **AC-9** An attacker whose victim walks away follows it, and one whose victim is killed by
  somebody else stops.
- **AC-10** A move order given to an attacker mid-approach ends the fight and leaves the mover on
  the move order's own destination, with no attack residue.
- **AC-11** The byte form's version is unchanged, and a world holding an attacker mid-approach —
  victim, cycle, destination and route together — encodes and decodes back to an identical world.
- **AC-12** The readout states the armed flag and changes when it changes.
- **AC-13** Leaving the map screen and re-entering it leaves the flag down.
- **AC-14** A press over a selection of more than 253 present units yields one order for every one
  of them, in ascending id, with none dropped.
- **AC-15** A tap or a box made while the flag is up leaves it up, and the following press orders
  the units the selection then holds.

## Properties

- **P-1** `pkg/ui` imports the render tier and no other, and names no simulation type; the attack
  seam carries two builtins.
- **P-2** A frame issues attacks or moves, never both, and the four outcomes of a press remain
  total.
- **P-3** The world's field set, its byte form and `formatVersion` are unchanged by this story.
- **P-4** The full local gate is clean: build, vet, gofmt, the whole test suite, and the three repo
  scripts.
