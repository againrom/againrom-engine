# Provenance — whom a group fights

Pin: research `404966d7`.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 the matrix, its 50 slots, its slot index | `AI-DIPLO-004` | High |
| FR-1 canonical, survives a save verbatim | `AI-DIPLO-085` | High |
| FR-2 directional; 102 of 866 ordered pairs disagree with their mirror | `AI-DIPLO-005` | High / Medium on the value space |
| FR-3 bit 0 is the only bit acquisition tests | `AI-DIPLO-004`, `AI-FILTER-001` | High |
| FR-4 zeroed at construction; with nothing written nobody is hostile | `AI-DIPLO-004` | High |
| FR-5 the index is `Player+0x04`, 1-based over the type-5 slot; column 0 unwritten | `AI-DIPLO-005` | High |
| FR-6 the sixteen `u16` at `+0x2c`, low byte, forced diagonal 2 | `AI-DIPLO-005`, `ALM-GRP-041` | High |
| FR-7 the AI is one slot of the full tick, `% 16 == 6`, once per full tick | `AI-TICK-008`, `AI-CLOCK-080` | High / Unknown whether a second, unmodulated dispatch also runs |
| FR-8 the group is the unit of AI, formed by equality of the type-6 group id on one player | `AI-GROUP-009`, `AI-TICK-008` | High |
| FR-9 the load walk: type-5 slot 0 aggressive, every other player guard | `AI-AUTHOR-015`, `AI-CENSUS-046` | High for the walk, Medium for the naming |
| FR-9 order 3 is Stand Ground, order 1 is guard | `AI-STAND-076`, `AI-GRPGUARD-074` | High |
| FR-10 one sight map, every member stamps into it, cleared once per build | `AI-GROUPSEE-068` | High |
| FR-10 the sight radius is `actor+0xa5`, constructor default 5, second writer unreachable | `AI-SIGHT-006`, `AI-GATE-079` | High |
| FR-10 the window is 41x41, the expansion stops at radius 20 | `AI-SIGHT-006` | High |
| FR-11 the filter runs with the group's first member as decider | `AI-GROUPSEE-068`, `AI-FILTER-001` | High |
| FR-12 corpses to a second list, moved back when the first ends empty | `AI-GROUPSEE-068` | High |
| FR-13 the clip is around the cell centroid, guard arm only | `AI-GRPGUARD-074`, `AI-RADIUS-014` | High |
| FR-13 order 3 has no radius clip and no walk | `AI-STAND-076` | High |
| FR-14 `max(dist + sight)`, floored by `MinimalGuardRange`, plus 4 | `AI-RADIUS-014`, `AI-RADFREEZE-075` | High for the arithmetic |
| FR-14 the shipped `ai.reg` sets `MinimalGuardRange` to 8, code default 10 | `AI-RADIUS-014` | High |
| FR-15 strict `<`, seed `0xffffff`, first candidate wins a tie, nothing sticky | `AI-SCORE-069` | High |
| FR-16 `(d << 8) + turnCost`, `d` Chebyshev in cells | `AI-COST-071` | High |
| FR-17 the 4x4 matrix, its sixteen values, its index law, the 0 veto | `AI-PREF-070` | High / Medium that nothing else writes the block |
| FR-17 the two modifiers and their two gates | `AI-COST-071` | High |
| FR-17 a ground or ghost melee member never selects a flier | `AI-FLIER-073` | High for the code half |
| FR-17 the domain byte is 0..3 with 3 the flier, defaulting to 1 | `MOVE-DOM-024`, `AI-FLIER-073` | High |
| FR-18 the second scorer's reach veto and its two coarser modifiers | `AI-REACH-072` | High |
| FR-19 the engage writes target and stop distance in one order block | `AI-GUARD-007`, `AI-PURSUE-040` | High |
| FR-19 the group arm re-issues unconditionally every decision | `AI-REISSUE-077` | High |
| FR-20 no break-off exists under group orders 1/2/3/5 | `AI-REISSUE-077`, `AI-DIPLO-086` | High / Medium for the negative's reach |
| D-1 the LOS predicate and its two untraced input grids | `AI-LOS-081` | High for the routine, **Unknown** for the grids' writers |
| D-3 the radius is frozen at the last guard issue | `AI-RADFREEZE-075` | High |
| D-4 the roll is gated on the has-members latch | `AI-RADIUS-014`, `AI-GUARD-021` | Medium |
| D-5 the remembered attacker cell, forced in for 20 decisions | `AI-GROUPSEE-068` | High |
| D-6 the invisibility exception and the spell-20 term | `AI-FILTER-001`, `AI-COST-071` | High / Unknown which spell carries id 20 |

## Ours by choice

| Choice | Why no source decides it |
|---|---|
| A fixed 50x50 matrix carried whole in the byte form rather than only the slots a world uses | The law's own object is exactly 2500 bytes and is serialised verbatim; a sparse form would be a second representation to keep in agreement with it. |
| The decision runs **after** the script pass on the shared phase | Both are placed on phase 6 of the sixteen by their own sources; nothing read this round orders the two against each other. |
| Sight is a package constant rather than an entity field | Its complete writer set is two instructions, one of which is in an arm nothing reaches — the argument `combat.go`'s `reach` already makes about `actor+0x12c`. A field would be four bytes of hashed state with one reachable value. |
| `minimalGuardRange` and `noticeMargin` are package constants | The first is carried by a shipped file and the second is a compile-time immediate; this tree reads no registry, so both are written where the arithmetic is and named as the two limits they are. |
| Groups are built by a linear scan in ascending entity id | A map would put Go's randomised iteration order on a path that touches a world. First-appearance order also makes "the first member" the lowest id without a second rule. |
| The turn cost is 0 rather than omitted from the cost expression | The expression is the law's; only its second term is unavailable, and writing the sum keeps the shape a later story fills in. |

## Open

- **The line-of-sight region.** `AI-LOS-081` reads the predicate whole and states, in its own
  Unknown clause, that neither the step grid at `fog+0x22000` nor the cost grid at `fog+0x28000`
  was traced to a writer. Until they are, only the region's *bound* is reproducible.
- **The turn cost.** `R0115` appears in the ledger at its call sites only. No claim
  publishes its body.
- **Which roster slot a human participant holds.** Made by the session-join path, not by the map.
  The tree already records the gap at `pkg/sim`'s owner field and at `SetLocalOwner(0)`.
- **The post.** `AI-GRPGUARD-074` states that for a group under order 1 from load nothing has ever
  written `ord+0x00`, so the walk half's destination is an unwritten packed cell. Reproducing that
  needs its own falsification, and the walk half is out of scope for it.
- **Whether the AI decision rate really is once per full tick.** `AI-CLOCK-080` narrows
  `AI-TICK-008`: the second dispatch of the AI pass has no modulo in front of it, and whether it
  runs during ordinary play is Unknown.
- **The third bit.** The engine reads the cell with `AND …,0x7` and no shipped map uses a bit
  above 1. Carried, never interpreted.

## Removed

| Dropped | Why |
|---|---|
| A per-group record in the world carrying the frozen radius, the latch and the order byte | It is the right shape and it is a story: it needs membership changes, the guard setter, the player's order vocabulary and the script's group command to have anything to freeze against. D-3 and D-4 name what its absence costs. |
| Deriving hostility from owner inequality | It would contradict `AI-DIPLO-005` directly: 102 ordered pairs disagree with their mirror, and 2362 of 3056 authored row cells are 0. |
| The `Wimpy` / `Withdraw` retreat | Both are `Data.bin` Units columns this tree does not carry, and the Humans table has neither, so nothing the party fields could withdraw anyway. |
