# Provenance — 0118 fog of war

Facts come from the `research/` submodule at this story's pin. Claims are cited;
experiments are not.

## Load-bearing, and each one is High

| Claim | What it decides here |
|---|---|
| `AI-SIGHT-093` | The drawn fog and the AI's vision are ONE algorithm written twice, agreeing in 0 of 1681 window cells over 43 640 measured observers. Its closing sentence — *"A consumer needs one implementation, parameterised"* — is why this story adds a reader and not an algorithm. |
| `AI-LOS-081` | The march: the four-term recurrence `pred − cost − h(cell) + h(observer)`, stored **before** the test, visible iff `> 0`. Already in the tree; re-cited because the exported reader is a second consumer of it. |
| `AI-LOS-088` | Both grids in closed form — the step grid's signed byte pair in `{-1,0,+1}`, the cost grid's `ftol((1<<k)·sqrt(i²+j²)/max(i,j))` running 128..181 at `k = 7`, and the seed `(1<<(k−1)) + (scanRange<<k)` as a budget in 1/128 cell. Already built at `pkg/sim/sight.go`; this story makes `k` and the seed arguments. |
| `AI-LOS-089` | The region is a disc of radius `scanRange`, not a square, and the term that dominates its reach is the observer's own altitude. Why the fog's frontier is ragged and why a minimap cannot be drawn from a radius. |
| `AI-LOS-090` | A blocking cell is not pruned. Unchanged by this story; named because the fog reader inherits it. |
| `AI-LOS-087` | The grids are built once, at world construction, before terrain exists — so they are a property of the build. Why the parameterised builder still yields a package-level value. |
| `TERR-FOG-118` | **The two readers walk different rectangles.** Client: columns `7 .. W−8`, 32-bit compares. Server: `8 .. W−9`, off four bytes written at map load. Measured: of 79 160 observers, 13 362 disagree and every one stands within 27 cells of a map edge; of the 43 640 that do not, 0 disagree. This is FR-2's second parameter. |
| `TERR-FOG-082` | The shroud level is a branch on the state **pair**: `11 → 0`, `10 → 8`, `00 → 16`, and `01` is unreachable. This is exactly the three-state plane and exactly the three multipliers FR-6 draws with; we did not choose them. |
| `TERR-TILE-079` | The three states and their lifetimes: `00` never seen, `10` explored and not currently visible, `11` currently in sight; bit 14 is cleared map-wide on a period, bit 15 by nothing. Why the explored layer only ever grows. |
| `TERR-FOG-085` | The map-wide clear runs on the **presentation** tick at period 32, not on either simulation clock — so newly-seen ground brightens promptly while ground a unit has left stays lit until the next clear. FR-5's period. Its `[L01661]` switch, whose nonzero value skips the clear entirely, is the original's own inverse of our debug reveal. |
| `TERR-FOG-086` | A drawable on a cell with no currently-visible corner is not drawn: `R0378` ORs the four corner tile words, `AND 0xc000`, `CMP 0xc000`, `SETNZ`. FR-7's gate. |
| `TERR-FOG-088` | The two implementations differ in storage, clock, consumer and lifetime, and `k` has two sources — the view's compiled constant against `[Scanning] ScanShift`. FR-2's third parameter and the customisation trap. |
| `AI-SIGHT-092` | `actor+0xa4` is a `u16` sight radius in 1/256 cell whose high byte is `actor+0xa5`, with six writers, four of them data streamers. This is where a unit's sight radius comes from, and it is why FR-3 forbids a constant. |
| `AI-GROUPSEE-068` | A group sees as one animal — one shared array, every member stamped into it. The union in `Sight` is that same set, taken over an owner's roster rather than over a group's members. |

## Cited with its confidence read, and deliberately not built on

`AI-MINIMAP-062` (High for the arm map) establishes a second click surface
covering exactly the six cursors whose art paths begin `s`. **Its own confidence
cell says the window's identity is inferred from those art names and not
established.** So this story does not claim to reproduce the original's minimap:
it authors one, says so in `spec.md`, and builds no click surface at all (D-4).

## Medium, and read as Medium

`AI-SIGHT-094` — the client keeps 1/128-cell precision where the server
truncates to whole cells, a 40-cell gap for a hero at the worked example. High
for the arithmetic; **Medium for the reach**, because whether a player-controlled
hero is ever the subject of a server sight stamp is not established. Not built
(D-2). `AI-LOS-089`'s and `TERR-FOG-118`'s corpus figures are Medium and are used
only as expectations, never as assertions.

## Ours by choice, not decoded

- The minimap's existence, placement, size, colours and its `M` binding (D-4).
- The debug reveal and its `F4` binding — the original's switch is the inverse
  (`TERR-FOG-085`) and reveals nothing (D-3).
- Holding the explored plane outside `sim.World` (DD-3). Nothing decoded says
  where a consumer must keep it.
- Drawing structures and statics on explored ground (D-5), against
  `TERR-FOG-086`.

## Open

- Whether the original's per-vertex fog projection (`TERR-FOG-083`) is needed
  before a player calls the frontier wrong. The claim states a per-cell consumer
  *cannot* reproduce any frontier cell; we are a per-cell consumer (D-1).
- Whether the explored plane must survive a save. It does not today (D-6).
