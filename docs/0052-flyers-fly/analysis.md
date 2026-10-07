# Analysis — two landed halves, and the wire between them

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/mapload` and `pkg/game`
(both change observable behaviour), **greenfield** for the reporting verb added beside them.

## What was not known

The movement domain exists in the simulation and the domain column exists on a loaded unit
definition. Whether anything carried a value from the second to the first was not known, and the
question had never been asked of the whole tree at once.

## What was looked at

**The claim that nothing joins them holds, and it is wider than it looked.** No file under
`pkg/game`, `pkg/mapload` or `cmd/` names a domain at all; `Domain` occurs in `pkg/sim`, in three
render tests that mean something else by the word, and nowhere else in our own code.
`data.UnitDef.MovementType` is written by the slot walk and read by nothing — the field's own
comment says so.

**A second break sits behind the first, and it is the one that decides whether anything shows on
screen.** The map-loading tier has two entry points: one takes a definition table, one does not.
The front-end calls the one that does not. So a domain joined inside the loading tier alone would
be joined on a path the running game never takes, and the story would land green and invisible —
which is the same defect it exists to fix, one tier up. The front-end opens three archives at
startup; the definition table is in none of them, and an installed root carries an eighth archive
that holds it.

**The three-phase context build** (architecture, then module, then detail) came out as: the loading
tier resolves a placement to a definition and consumes it into exactly one statement, which builds
the entity; the front-end builds its world through one call in one function; the archive set is one
fixed list in one function. Each of the three is a single site, so the change is three small writes
and not a diffusion.

**An implicit assumption checked by hand rather than believed.** The block plane the loading tier
derives sets bit 0 and bit 1 and nothing else, and the story in flight beside this one holds it
there — its contract refuses a byte setting any bit above bit 1. So no object bit arrives during
this cycle, and what a domain-2 mover is stopped by here is settled by that, not by this story.

## Measurements taken on the two lawful installs

A developer harness outside the repository read the shipped table and every shipped map through
this tree's own loader and classifier. What it found:

- The table's Units collection holds **16** rows whose domain column is not the ground value —
  four difficulty variants each of four named classes, two classes on each of the two non-ground
  values. No row of either searched collection carries a value outside the three the simulation
  models plus the empty cell, so the encoding distinguishes exactly the cases the simulation has
  and nothing is collapsed by mapping them one to one.
- **Every one of the ten shipped maps carries non-ground placements**, from 6 on the smallest to
  444 on the largest.
- The two installs answer identically, and the archive that holds the table is byte-identical in
  size between them. The English root spells its archives in lower case and the Russian root in
  upper; both resolve, because the container filesystem already folds.

## The question the owner's account leaves open, and where it is answered

The owner reports a ghost and a bee crossing a river. Both are on the same non-ground value, and
the research ledger at this pin settles what that value means on both halves: it passes water and
mountain, and it is stopped by every object and building. The other non-ground value is stopped by
the map border and nothing else. So the two are not two grades of one thing, and the tree question
is **answered rather than open** — a ghost stops at a tree and a dragon does not.

What this tree can reproduce of that answer is the water half only. Its derived plane carries no
object bit, so a domain-2 mover here is stopped by the border alone, exactly as a domain-3 mover
is: the two differ in this build by which occupancy plane they contend on and by nothing else. The
gap is the plane's and not the domain's, it was disclosed when the domains landed, and it is
measured on the shipped corpus at 83 203 cells of 880 704.

## What was rejected while reading

- **Deriving the object bit here to close that gap.** The plane is another story's subject this
  cycle and its contract refuses the bit. Two writers on one plane in one cycle is how a lane loses
  work; the gap stays disclosed and sized.
- **A domain source that is not the table.** The registry key, the sprite layer and the `Z` column
  all correlate with flight on the shipped corpus. Each is a correlate of the decoded column rather
  than the mechanism, and one of them was already tried and withdrawn in an earlier story.
- **Making the new archive optional.** A front-end that silently continues without the table gives
  every unit the ground domain, which is precisely the defect being fixed, and it would come back
  invisibly on any install that had lost the file.
