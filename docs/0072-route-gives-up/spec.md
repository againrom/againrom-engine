# Spec — the route the hero does not take

**Intensity: spec-anchored / static. Terrain: brownfield** for the near search and the tick that drives
it — both already ship the behaviour this changes — and greenfield for the mission driver.

A mover ordered across a map holds a route its far search found over terrain alone, and steps along it
by a second, unit-aware search aimed four cells ahead. When a body stands on that fourth cell the near
search answers that there is no route without searching for one, the mover keeps its cell, and the
sixteenth consecutive refusal drops the order. A one-cell step aside would have taken it past.

The consequence is not local. Every unit on a campaign map is a body a route may cross, so an ordinary
order over an ordinary map ends short of where it was sent, in a place that depends on where the other
units happen to stand — and a mission objective that measures a unit's distance to a point is then
unreachable for a reason no part of the world states.

## Context

Two searches, and only the near one sees units. The far search reads the map alone and sweeps it
whole; its answer cannot depend on where in a tick it was asked for, and it is the search that finds
the way round a lake. The near search reads occupancy as it stands at the mover's own turn, is bounded
to the cells about the mover, and is the search that finds the way round a body. Between them the far
search chooses the shape of the walk and the near search walks it.

A search that cannot reach the cell it was aimed at may **settle**: it takes, from the cells its own
failed wave labelled, the cheapest to reach from the mover in the first expanding ring around the cell
it was aimed at that holds any labelled cell. The far search does this today. The near search does
not, and that asymmetry is what this story removes.

## Functional requirements

- **FR-1 — a near search settles.** A near search that cannot label its waypoint returns a route to a
  substitute chosen by the rule the far search already uses — ring order outward from the waypoint,
  cheapest-to-reach-from-the-mover within a ring, and membership being *this wave labelled it* rather
  than *this cell is passable*. It answers that there is no route only when no ring inside its bound
  holds a labelled cell the mover may rest on.

- **FR-2 — the two searches differ in the ring bound and in nothing else.** The near search's bound is
  **flat: eight**, whatever the waypoint's distance. The far search's stays shaped on the distance
  ordered. A search that may not settle asks for no bound.

- **FR-3 — the far search is untouched.** Its relation, its window, its budget, its bound, and its
  writing of the settled cell back into the order are exactly what they were. No route this build
  produces today over an unobstructed map changes.

- **FR-4 — a near search whose waypoint is free behaves exactly as it does today.** Settling is
  reachable only where the wave ends with the waypoint unlabelled; where the waypoint is enterable and
  reached, the search stops on it as before, and the route returned is the same route.

- **FR-5 — the give-up survives.** The stall count, its limit, and the order it drops are unchanged.
  A mover whose near wave labels nothing it can use still counts the tick, and the sixteenth
  consecutive one still ends the order with no residue. A mover boxed in on every side is such a
  mover: settling for the cell it already stands on is not an advance and does not clear the count.

- **FR-6 — a mission can be driven to its outcome outside the suite.** A developer tool starts a
  campaign mission from a lawful install, issues plain move orders toward named points until each is
  reached within a named radius, advances the world, and reports the outcome, the tick it was decided
  on, and where each ordered unit ended. It reads its asset root from configuration and writes no
  game data anywhere.

## Acceptance criteria

- **AC-1** — a mover whose stored route's waypoint a standing body occupies, with the way past it open
  by one cell, walks past that body and arrives at its ordered cell. Today it holds its cell, and the
  same fixture with the body removed shows the arrival is not the fixture's doing.

- **AC-2** — the near search's bound is eight and not the far search's: on a fixture where the two
  bounds differ, a waypoint whose only labelled neighbourhood lies between the two rings is settled
  for under the near rule and refused under the far one.

- **AC-3** — a mover boxed in by bodies on all eight sides counts a stall every tick, reaches the
  limit, and ends holding no target, no route and a zero count, exactly as it does today.

- **AC-4** — a mover ordered **onto** a cell a downed unit holds walks up to that cell and stops
  adjacent to it, still holding its order; it never stands on it. A one-row corridor sealed by a
  downed unit is still not passable.

- **AC-5** — the near search still runs under the scaled generation budget and not the flat one,
  driven through the tick rather than called directly: on a fixture where the two budgets yield
  different first steps, the mover takes the scaled rule's.

- **AC-6** — a world advanced over a map with no body on any waypoint produces the same routes, the
  same positions and the same digest as it does today, across the whole existing corpus.

- **AC-7** — the tenth campaign mission, started from a lawful install and driven from its own start
  by plain move orders alone, runs end to end: the map loads, an order is issued, a unit carries one
  out, and the reporter reaches a decided outcome rather than hanging. Against the pre-change build
  the same drive leaves it undecided.

  *Amended 2026-08-11 (hotfix `f5b68b9`).* As written this criterion demanded the **won** outcome,
  and the drive did reach it when the story landed — `won at tick 3520`, then 3504, then 2608. `0086`
  made the AI engage and it has lost ever since, which `0087`'s own three-tip drive located at the
  time. That is not a defect: mission 10 is an **escort**, script unit 21 is the subject of a check-18
  (VIP) node, and once enemies engage the map's hostiles reach it before the party can. The win was an
  artefact of an inert AI. An unattended drive of this mission loses by the mission's own design, so
  the outcome **word** is no longer a criterion of anything; the distance to a simulated win is
  measured by `pipeline/check-milestone.sh`'s unsupported-instant census instead. Owner's ruling,
  2026-08-11: the mission has long been passable by hand.

## Properties

- **P-1 — the determinism wall holds.** The substitute is chosen by integer comparison over the wave's
  own labels, in a fixed ring order, with a strictly-less accept — no clock, no float, no map
  iteration. Two identical worlds settle on the same cell.

- **P-2 — no new state.** No entity field, no world field, no byte-form record and no version. The
  settle rule is a search parameter, chosen at the two call sites in the tick and stored nowhere, so
  nothing a digest covers gains a member.

- **P-3 — total.** The near search cannot panic on a waypoint off the map, on a mover standing on a
  blocked cell, on a world with no in-bounds cell, or on a wave that labelled nothing.

## Out of scope, and disclosed

**No wait rule.** The original has a mover turn to face a blocked adjacent waypoint and stand still
instead of re-searching. Its gate is undecoded, this tree has never had it, and it is not added here:
a mover one cell short of a body steps around it. Where the original would wait, this build moves.

**The optimised routing mode takes no substitute, on either search.** Its plane holds costs *to* the
target and says nothing about what the mover can reach, so it has no labels to settle from; it refuses
where the canonical mode settles. That asymmetry already exists for the far search and this story does
not close it — the mode is a route-quality instrument, not the shipped tick.

**The far search stays unit-blind.** A body it cannot see still shapes the walk only through the near
search's detours. Making it unit-aware would make a route depend on where in the tick it was asked
for, which is the property that keeps the search movable off the tick later.

**No cost plane and no rate plane.** Every step still costs the flat straight/diagonal pair and the
rate law still takes its fallback mean. Both change how long a walk takes and which of two equal-cost
routes is chosen; neither is in this contract.

**The milestone drive is not in the synthetic suite.** The tool reads a lawful install, so the check
that mission ten reaches a won outcome runs only where an asset root is configured and is skipped
otherwise; the suite stays green with no game present, and the shipped regression evidence for FR-1
is synthetic.
