# Spec — a placed unit is born in its own movement domain

## Problem and current behaviour

Every unit a world builds from a map is a ground mover. The simulation carries a mover's **domain**
and honours three of them — deciding which cells terrain closes and which movers contend for a cell
— but the map-loading tier names none, and the domain's zero value is the ground mover, so the
absence reads as a decision. A flying class placed on a map walks; ordered across water it does not
go.

The number that says otherwise is already loaded: a unit definition built from the definition table
carries a **domain column**, written by the walk that reads the row and read by nothing in the tree.

Two facts stand between that column and the screen:

- The map-loading tier has **two** entry points: one takes a definition table and a difficulty and
  gives a resolved placement its own health maximum, one takes neither and gives every placement a
  provisional constant. **The application calls the second**, so no placement in the running game is
  resolved against a table at all.
- The application opens **three** archives at startup and requires all three. The table is in none
  of them; an installed root carries a further archive that holds it.

## Terms

The **domain column** is the definition-table column naming a mover's movement domain; the three
**domains** are the simulation's own — **ground**, **ghost**, **air**. A **placement** resolves
through one of four **arms**, only the units arm reaching a stat-bearing entry. The **derived
plane** is the per-cell byte built from a decoded map.

## Functional requirements

### The join

- **FR-1 — a placement's domain is its entry's domain column.** Where a placement resolves to a
  units entry, the entity built for it MUST take that entry's domain column as its domain, mapped
  **1 to ground, 2 to ghost, 3 to air**. **Any other value MUST give the ground domain**, and MUST
  NOT be an error, refused, or folded into a non-ground domain. An entry whose domain cell is empty
  carries the column at 1 and so takes the first arm rather than this one; the mapping MUST
  nonetheless be **total**.

- **FR-2 — one assignment site, and the unresolved case is ground.** The domain MUST be assigned at
  the single place a placement becomes an entity, out of the same resolution that decides that
  entity's health, and MUST be written nowhere else. A placement that takes the npc, server-id or
  humans arm, or reaches no entry on the units arm, MUST be a **ground** mover — the same case as the
  health pair it keeps.

- **FR-3 — the table-free entry point does not move.** A world built from a map with no table MUST
  hold exactly the entities, the byte form and the digest it holds today: every mover ground, every
  health pair provisional.

### What stops a mover of each domain

- **FR-4 — the three domains' terrain and contention, as this build ships them.** The derived plane
  carries **two** terms: one closing a cell to a ground mover, folding water, mountain, impassable
  terrain, placed scenery and the map border together; one closing a cell to a non-ground mover,
  which the **map border alone** sets. A **ground** mover MUST be stopped by the first and a
  **ghost** and an **air** mover by the second, so both cross water and neither reads the term
  carrying scenery. A ghost MUST contend for a cell with ground movers; an air mover only with other
  air movers. This is what the simulation already does: **no route search and no step resolution
  changes**, and the requirement is met by demonstrating it rather than by writing it.

### Reaching the screen

- **FR-5 — the application resolves its placements against the installed table.** Every map the
  application opens MUST be built as a world **through the resolving entry point**, against the
  definition table read from **the one installed archive that holds it**. Two things therefore reach
  the running game together, and both are intended: a placed unit's **domain**, and the **health
  maximum** its class entry carries in place of the provisional constant.

- **FR-6 — the archive holding the table is required, and so is the table in it.** The application
  MUST open that archive at startup beside the three it already opens, reporting a failure to open
  it with the path named as it does for the others, and MUST equally refuse to start when the
  archive opens but the table cannot be read out of it or parsed, naming what failed. Starting with
  no table — and therefore with every mover ground again — MUST NOT be reachable by any route.

- **FR-7 — the difficulty the application applies is the identity.** Every placement the application
  resolves MUST be built at the difficulty value that leaves a definition unchanged, so no number
  reaching the screen is scaled by a choice no one made. The application MUST expose no setting for
  it. The developer verb keeps the argument it has.

### Seeing it

- **FR-8 — the tool that resolves a map's placements reports their domains.** The developer verb
  resolving each placement of a named map against the table MUST additionally report the domain
  **per placement**, and the count of placements in each of the three **per map**. No test may read
  an install.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a table whose units entries carry the domain column at 1, 2, 3, an empty cell and a value outside the three, and a map placing one of each | the world is built | the domains are ground, ghost, air, ground and ground, in that order, and no build fails |
| **AC-2** | unit | a map with a placement on each of the npc, server-id and humans arms, and one on the units arm reaching no entry | the world is built | all four are ground movers and all four keep the provisional health pair |
| **AC-3** | unit | one map, built through the table-free entry point and through the resolving one with no table | both worlds, each also against the world that map built before this story | all three agree field for field, in byte form and in digest |
| **AC-4** | unit | a map whose plane carries a band of ground-blocking cells across it, a table giving one placement each domain, and an order sending each across the band | stepped for a budget the unobstructed movers arrive well inside | the ghost and the air mover stand on the far side; the ground mover stands on no blocked cell at any tick and is not on the far side at the end |
| **AC-5** | unit | a ghost and a ground mover **ordered onto** one free cell, and two air movers ordered along crossing paths | the worlds are built and stepped | the ghost and the ground mover end at rest on distinct cells; the two air movers occupy one cell at some tick while both still hold orders |
| **AC-6** | unit | three asset roots: one missing the archive that holds the table; one where that archive opens but holds no table; one where it holds bytes that will not parse | the application's startup is run over each | each refuses, naming the archive's path or what failed to read; no map is listed, no map is opened and no world is built |
| **AC-7** | unit | two worlds built from one map and one table, advanced under one schedule | stepped in lockstep | equal digests at every tick and equal byte forms |
| **AC-8** | manual | a lawful install and a shipped map placing both non-ground domains | the developer verb is run over both | every placement reports a domain; the counts sum to the placement count; the non-ground counts fall on the classes the table's own non-ground rows name |
| **AC-9** | manual | the built application, a lawful install, and a shipped map carrying a flyer beside water | a non-ground unit is selected and ordered to a cell across the water, then a ground unit near the same water is ordered to the same cell | the non-ground unit crosses the water and reaches the far side; the ground unit does not cross it |
| **AC-10** | unit | one map placing two units whose class entries carry **different** health maxima, and a table holding both entries | the world is built through the resolving entry point at the difficulty the application uses | each entity's health and maximum are its own entry's maximum, unscaled and different from each other's, and neither is the provisional constant |

**Error cases:** AC-6 is the error case. A domain code outside the three is deliberately **not** an
error (FR-1), and a placement resolving to nothing is deliberately not an error.

## Derived properties

- **P-1 (invariant)** An entity's domain is a pure function of the definition its placement resolved
  to: no clock, file, generator or float takes part, and it does not depend on the order placements
  are walked in.
- **P-2 (completeness)** Every placement yields exactly one domain: each arm has an outcome, no arm
  has two, and only a matched units entry can yield a domain other than ground.
- **P-3 (negative-invariant)** For any asset root from which no table can be loaded, no world is
  built and no map is opened — the startup that would have done so does not complete.
- **P-4 (invariant)** A world built from a map with no table is byte-identical to the world that map
  built before this story, so every digest recorded against that path still names the same world.

## I/O examples

```
domain column  ->  mover domain
  1                ground   the whole derived plane closes cells to it
  2                ghost    crosses water and mountain; contends with GROUND movers
  3                air      crosses everything this plane derives but the border;
                            contends with AIR movers only
  empty cell       ground   the cell leaves the column at 1
  anything else    ground   not an error, not a refusal

archives the application requires at startup, in order
  main.res   graphics.res   scenario.res   world.res
                                           ^ added by this story; it is the one
                                             holding the definition table, whose
                                             entry is addressed below that
                                             archive's own identity segment

developer verb, unchanged in shape
  <tool> -databin <world.res> <map.alm> [<difficulty>]
    per placement : arm, entry index, health maximum, DOMAIN
    per map       : placements counted per domain, summing to the placement count
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The domain is read from the **definition table's own column**. | **(A) a graphics-registry key, the drawn layer, or the height column** — each a correlate of flight rather than its cause. **(B, chosen)** the column, which costs the application a fourth archive. |
| **C-2** | A domain code outside the three yields the **ground** domain. | **(A) refuse the entry by name** — loud, and diverges from the original, whose selector leaves the ground value standing for a code it does not know. **(B, chosen) the ground domain**: a mistyped column is then silent, which FR-8's per-placement report answers. |
| **C-3** | The archive holding the table is **required**, not optional. | **(A) optional** — the application runs without it and every mover is ground: this story's own defect, restored silently on any install that lost the file, and green in every test. **(B, chosen) required**, refused at startup with the path named, as the other three are. |
| **C-4** | The application applies the **identity** difficulty. | **(A) expose a setting** — no screen exists for one, and the value would be a second source for numbers the world already carries. **(B, chosen) the identity**, so every number reaching the screen is the table's own. |

## Out of scope

- **A static-object term of the derived plane that a non-ground mover reads.** Scenery is folded into
  the ground term and there is no second term beside it; the plane keeps exactly the two it has.
- **Ownership**, and any restriction on which units a player may select or order.
- **The domain's other consumers** — everything keyed off the same value besides passability and
  contention.
- **The footprint column beside the domain**, and any mover occupying more than one cell.
- **The remaining definition stats reaching the simulation** — every column but the health pair
  stays loaded and unread.
- **Any change to route search or step resolution.**
- **A scenario or difficulty picker**, and any user-visible setting for either.

**Disclosed limitations**, accepted and owned: a ghost here passes a tree as well as water, so the
two non-ground domains differ in this build by their occupancy plane alone. And because FR-5 routes
the application through the resolving entry point, health maxima on screen change from one constant
to each class's own — intended, and named here so it cannot later be read as a regression.

## Verification mapping

Every AC's Level column above says where it lives. Only AC-8 and AC-9 need a lawful install.

## Gate check

FR-1 → AC-1, P-2 · FR-2 → AC-2, P-1, P-2 · FR-3 → AC-3, P-4 · FR-4 → AC-4, AC-5 ·
FR-5 → AC-7, AC-9, AC-10 · FR-6 → AC-6, P-3 · FR-7 → AC-10, AC-9 · FR-8 → AC-8.
