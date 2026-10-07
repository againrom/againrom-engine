# 0164 — plan

## Shape

One new file, `pkg/sim/presence.go`, holds the whole of map presence: the two arms, the placement
search and the fit test. Everything else is a gate added at a reader that already exists.

The gate is deliberately not one filter applied in one place. There is no single list in this
package corresponding to the original's global on-map actor list — `w.entities` is the only slice
and it is also the identity, the group membership and the save. Removing an entity from it would
change ids, break `indexOfEntity`'s binary search and rewrite the byte form's record order. So the
bit is read at each reader that asks about presence, and the readers are enumerated in FR-3 so
that the set is a stated one rather than whatever a search finds.

## FR accounting

| FR | Where it lands |
|---|---|
| FR-1 | `Entity.OffMap` in `pkg/sim/world.go`, beside the other per-entity state |
| FR-2 | `takeOffMap` in `pkg/sim/presence.go`; dispatched from `runInstant`'s opcode-16 arm |
| FR-3.1 | `counted` in `pkg/sim/route.go` — one line at the top |
| FR-3.2 | `candidates` in `pkg/sim/engage.go` |
| FR-3.3 | `aiGroups` and `groupLivingMembers` in `pkg/sim/engage.go` — **two** readers, see DD-10 |
| FR-3.4 | the move loop in `pkg/sim/step.go` |
| FR-3.5 | the attacker loop in `pkg/sim/step.go` and the victim resolution in `advanceAttack`, `pkg/sim/combat.go` |
| FR-3.6 | `entityDraws` in `pkg/game/world.go` |
| FR-4 | no change: the group-count check and the group hand-over already read `Group` and nothing else. Witnessed rather than written |
| FR-5 | `returnToMap` and `placeNear` in `pkg/sim/presence.go`; opcode-17 arm |
| FR-6 | opcode-18 arm in `runInstant`, calling `takeOffMap` then `placeNear` |
| FR-7 | `groupMembers` + the opcode-32 and opcode-33 arms in `runInstant` |
| FR-8 | `formatVersion`, `entityLen` and the record's encode/decode in `pkg/sim/binary.go` |
| FR-9 | `scriptInstantSupported` in `pkg/sim/script.go` |

## Design decisions

**DD-1 — The bit is a field on the entity, not a set on the world.** Every other piece of
per-entity state in this package is a field, the byte form is one record per entity, and the
digest is the byte form. A side set would be state beside the digest that `UnmarshalBinary` would
have to rebuild, which is the reason the route scratch is not one either.

**DD-2 — `counted` is where the footprint clears.** It is the one predicate the occupancy seed,
the self-presence term in `enterable` and the flyer re-seed all read, so a single line there
removes an off-map entity from every occupancy answer at once. Adding the test to the seed loop
instead would leave `enterable`'s subtraction reading a presence the seed never wrote.

**DD-3 — The fit test is a linear scan, not a `routeScratch`.** The script pass runs before the
tick's scratch plane is built (`engagementPass`'s own note), and a scratch allocates one slot per
map cell. A placement search runs at most 17 attempts, once per fired node, and every referencing
trigger fires at most once per mission, so a scan over the entity slice is the cheaper of the two
by a wide margin. It reads `counted` and `terrainOpen`, so it asks the same two relations the
movement predicate asks and is not a second copy of a movement law.

**DD-4 — Every attempt draws two values from `w.rng`, including the two `r = 0` attempts.** The
original's helper calls its random routine at every attempt whatever the radius, and this
package's `uniform` always draws for the same reason. So the number of draws a return makes
depends on how many attempts ran and on nothing a draw produced. The `r = 0` attempts draw
`uniform(0)`, which is always 0 — the draws are consumed, not used.

**DD-5 — The search commits nothing until an attempt succeeds.** SC-4 states the divergence and
why it is unobservable. Written this way, a failed return is a true no-op and AC-9 can be checked
against the world digest rather than field by field.

**DD-6 — Instant 18 reuses `placeNear` with the exact-cell stage skipped**, rather than a second
search function. `TRIG-MAPGROUP-043` reads `R0122` as `R0123`'s placement half with
the cell supplied by the caller, which is exactly this factoring: `returnToMap` is "read the
retained cell, try it twice, then `placeNear`", and instant 18 is "`placeNear` at the other unit's
cell".

**DD-7 — The group arms read `Group`, not `effectiveGroup`.** Every other script arm over a group
— the count check, the hand-over, the group order — reads the map's own group word. Reading the
command group here would make a script's removal of a group depend on whether the player had
moved it, which no claim supports.

**DD-8 — An off-map entity's attack target is cleared by the ATTACKER, not by the removal.**
`TRIG-OFFMAP-041`'s untouched list does not include the order, and instant 16 writes one bit.
So `takeOffMap` leaves the removed unit's own attack target standing, and it is `advanceAttack`
that drops a victim that has left the map — the same place and the same line that already drops a
dead one. A removal that reached into every other entity's order would be a write the arm does
not make.

**DD-10 — The group-decision gate is in TWO places, and the second was found by
reverting the first.** `groupLivingMembers` is not the set a decision walks; `aiGroups` is, and it
is a separate loop. With the gate only in `groupLivingMembers`, a removed unit was still put into a
group and handed a victim. Both carry it: `aiGroups` because it builds the set, `groupLivingMembers`
because it is a second, narrower reader (the stance's own walk home) that must not disagree with it.

**DD-9 — The version test is renamed rather than renumbered.** `pkg/sim/binary_test.go` carries a
test whose name spelled the live version number; it has gone stale three times. The name loses the
number in this story.

## Where each disclosed divergence lands

`spec.md`'s SC entries are the places this build parts company with what EXP-0169 decoded, or
authors what it left open. Each is carried by code and named in a doc comment, so a reader meeting
the code meets the disclosure:

| SC | Carried by |
|---|---|
| SC-1 | `Entity.OffMap`'s doc (`world.go`) and the `formatVersion` block (`binary.go`) |
| SC-2 | `presence.go`'s file header |
| SC-3 | `returnToMap`'s doc |
| SC-4 | `returnToMap`'s doc and `placeAt`'s two-writes-are-one-act rule (DD-5) |
| SC-5 | `tryAttempts`' doc, which quotes both readings and says which is implemented |
| SC-6 | `trySquare`'s doc |
| SC-7 | the opcode-18 arm's doc (`script.go`) |
| SC-8 | `verification.md` alone: it is a statement about what was done, not about the code |

## Order of work

1. The field, the constructor's silence about it, and the byte form at version 48.
2. `presence.go`: the two arms, the search, the fit test.
3. The six gates of FR-3.
4. The support table and the five dispatch arms.
5. Tests, including the revert-check of AC-18.
6. The census, measured before and after on all 28 campaign maps of both roots.

## Risks

- **The digest moves for every world.** Version 48 changes the entity record width, so every
  pinned digest in this package's tests changes. That is expected and the pins are recomputed;
  what must not happen is a pin being recomputed while a field it covers is silently dropped, so
  AC-15's round-trip is checked on a world that actually has the bit set.
- **`counted` has three readers.** A test that only exercises the occupancy seed would not witness
  `enterable`'s subtraction. AC-3 drives a real route, which reaches both.
