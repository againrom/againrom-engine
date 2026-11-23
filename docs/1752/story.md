# Second game: the pick-up key skips sacks a hero cannot stand on, and a blocked patrol turns back

## Result

In the second game, the pick-up key no longer sends a hero after a sack lying on ground he cannot stand on.
A sack under his own feet is still taken. A patrol whose point no route reaches (behind a wall or water) now
turns back when it stands within two cells of that point; farther away it waits and tries the same point again. A point
held by another unit is not covered. Guard, look-around and idle mages already behaved as the research reads them and
are unchanged. The first game is unchanged.

## Authority

- Owner: «Да, запускай» (epic «ROM2: полная игра»): adopt the orders research.
- `R2-ENGINE-1625`: a ring cell of the collect search needs the unit's footprint passable; the centre does not.
- `R2-ENGINE-1624`: on an empty route the patrol moves on only below Chebyshev distance 3 of its point.
- `R2-ENGINE-1621`, `R2-ENGINE-1622`: the player's guard radius is the members' spread plus sight, at least 8.
- `R2-ENGINE-1630`: look-around turns on a draw below 205 by 33 and up to 190 more.
- DIV-4904 open for the occupied point; DIV-4882 updated at Medium.
  Evidence: `review/story1752-rom2-orders-adopt/RESULT.md`.

## Rules

1. When the pick-up key searches the rings around a hero, a sack on a cell his footprint cannot enter is passed over.
2. When the sack lies on the hero's own cell, it is taken whatever the ground there.
3. When a second-game patrol's route is empty (point behind a wall or water) within 2 cells, it moves on.
4. When the patrol stands three or more cells from its point, it keeps the point and walks to it again.
5. When a player orders guard, the radius is the widest member distance to the mean cell plus sight, at least 8.
6. When an idle unit's facing equals its heading, a draw below 205 turns it by 33 to 222 steps; any other stands.
7. The first game's collect, patrol and guard results do not change.

## Code

| kind | its one builder | change |
|---|---|---|
| collect search | `pkg/sim/sackpickup.go:75` OrderSackCollect | ring cell takes `terrainOpenFootprint` |
| empty-route arm | `pkg/sim/combat.go:518` repickRefusedWalkArm (second game's row) | order 0xa answers `walkArmPatrol` |
| patrol executor tail | `pkg/sim/savedgroups.go:458` patrolRefusedTail beside savedPatrol | new tail of the one patrol owner |
| guard issue | `pkg/sim/groupwords.go:84` rowCommandGuard, `pkg/sim/engage.go:420` noticeBase | none; pinned |
| look-around turn | `pkg/sim/facing.go:215` stepIdleTurn | none; pinned |

## Proof

| test | rules |
|---|---|
| `TestCollectRingCellNeedsTheTakersFootprintOpen` | 1, 2, 7 |
| `TestPatrolEmptyRouteMovesOnOnlyNearItsPoint` | 3, 4, 7 |
| `TestLoadedPatrolEmptyRouteTailWritesItsOrderBlock` | 3, 4 |
| `TestPlayerGuardRadiusFollowsSpreadNotCount` | 5, 7 |
| `TestLookAroundTurnsOnADrawBelow205By33AndMore` | 6 |

Release tests at the release: none.

## Open debt

- A patrol whose point another unit holds walks up to it and waits; the original is Unknown (DIV-4904).
- Which path governs a player's guard (order 0xb or the group programme) stays Unknown (DIV-4905).
- The look-around divisor and call rate, the item take of a pickup, and hit, save and load under L stay Unknown.
