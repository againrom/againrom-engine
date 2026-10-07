# Analysis — 0119-chargen

## What we did not know, and what we looked at

### The owner asked for two things that turned out to be one

He asked for a generation window behind a flag, and separately reported that his hero always has
100 health whatever his statistics are. Reading the tree makes them one story: the generated
character is what supplies the three inputs the health graph needs and nobody in this tree could
state, so the screen is not a decoration over the defect — it is its fix.

### Where the 100 comes from

Not from a broken derive. `Hero.Recompute` (`pkg/data/recompute.go`) is complete and its **Combat**
block already reaches the entity, at `pkg/mapload/start.go:313-321` — to-hit, defence, absorption,
the damage pair, the cadence pair, reach and `AlwaysHits` are all the derive's. What stops is the
two **pools**, and the file says so in its own words at line 108: *"HealthMax AND ManaMax ARE
PRODUCED AND WIRED TO NOTHING"*. The 100 is `SpawnHP` (`pkg/mapload/fromalm.go:39`), the constant
every unit a map places is born at, written onto a party member at `start.go:295`.

The reason given for not wiring it is exact and is not an oversight: the pool graph reads a class
flag, a health column and a mana column, *"three inputs nobody here can state"*, and a hero with no
health column derives **three** health at the shipped spread — inventing a column to avoid that
would have been a fit. So the gap was correct while the tree had no character generation.

`pkg/sim` already carries the door: `SetCombat` (`pkg/sim/rearm.go:73`) writes the nine combat
numbers onto a live entity and deliberately writes neither pool. `sim.Entity` already has
`Mana`/`MaxMana` and both regeneration periods, so no field is added anywhere.

### Where the three inputs actually live

`SESS-HERO-014` closes it. The original's generation does not invent a character: the appearance
byte picks one of **four shipped `Humans` rows** by name, and everything the pool graph needs is a
column of that row. `pkg/data` already decodes that collection — `HumanDef` carries `HealthMax`,
`ManaMax` and `TypeID` — and `pkg/data/defsearch.go` already searches it. So the story is a lookup,
not a decode.

`HERO-CLASS-013` gives the type id its arithmetic: `typeID = gender + 0x23` when the class bit is
set and `gender + 0x21` when it is clear. A shipped row therefore **states its own class flag and
its own sex**, and the flag the pool graph multiplies by can be read off the row rather than chosen.

### What was already waiting

Three markers in the tree name this story. `pkg/game/hero.go:44` calls `PartySpread` *"THE
SUBSTITUTION POINT"*. `pkg/game/inventory.go:31-40` says its two figure constants are *"ONE EDIT
WIDE: a later story that adds character generation replaces these two lines"* — verified true, they
are read at exactly two call sites in one function. And `pkg/data/chargen.go` is the whole
arithmetic of the screen, written by 0113 with a header saying a screen *"is a consumer of this file
and does not exist yet"*: `PointCost`, `StepCost`, `Spread.Legal`, the 140, the [15,45] pair.

### What the screen had to be built against

`pkg/ui/flow.go:12` has three screens and no notion of a form: no stepper, no radio group, no text
entry. The picker is the only precedent — a pure model in one file, drawn with the engine's debug
font through `ebitenutil.DebugPrintAt`, hit-tested through the virtual frame. That font renders no
Cyrillic byte, so every label the program chooses is English, as the rest of this tree's chrome
already is.

### What sex actually reaches, which is not what we were told

The brief said the sex bit is the axis that reaches the drawing. `HERO-APPEAR-045` says the
opposite: the sex bit is written onto the drawable and **neither** routine that turns a drawable
into pixels reads it, so *"the body name, the directory, the sheet pair and the drawn class id are
all independent of the character's sex"*. The map sprite will not change with sex.

It reaches a different picture. `pkg/data/itemcode.go:95` `FigureDirFor(mage, female)` selects one
of four shipped figure directories — `mfighter`, `mmage`, `ffighter`, `fmage` — and that is the
**inventory window's figure**. Today `pkg/game/inventory.go:38` jams `mfighter` into it. So sex is
visible, in the inventory window rather than on the map, and the parameter it needs already exists
one call away with a constant in it.

### The one obligation that was owed before wiring

`recompute.go:159` states that `logBase11` carries no measured margin against the original's own
logarithm, unlike `pow11`'s 2e-4, and that *"whoever wires that seam owes this measurement first"*.
It is not a formality: the pools truncate three times and a truncation amplifies an ULP to a whole
unit of health. The measurement is cheap once you notice that the integer part of the second step is
added, not multiplied, so the fractional part depends on the experience term alone.

### What we could not settle

Which of the four shipped names goes with which (class, sex) pair is not published — the claim names
the four arms and the two discriminating bits but not the mapping. And which value of the gender
addend is female is stated by research as **not established and not needed**. Both are handled by
reading the shipped rows rather than by a table of ours; only the second survives as an authored
constant, one edit wide.
