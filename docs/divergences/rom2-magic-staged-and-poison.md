# Divergences -- Second-game magic -- staged area start and poison on a fallen unit

2 row(s). Part of the split described in `docs/DIVERGENCES.md`: read that file first for what a row means, what each column holds, and how a row is found. These rows sit outside `docs/divergences/rom2-magic.md`, which is at the size ceiling.

## Divergences

Implementation and researched ROM1 behaviour differ, or the owner ruled against a claim.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-5142 | ROM2 magic / Acid Stream first stage | Owner: make it as in the original | `R2-ENGINE-795` (High / Medium / Unknown): Acid Stream runs the stage body Blizzard runs, on area updates 1, 4, 7, 10, 13, 16; `R2-ENGINE-392`, `R2-ENGINE-812`: the area pass runs before ordinary actors, so an ordinary insertion first updates on the next tick. The tick from request to acceptance is Unknown | Acid Stream takes Blizzard's first-stage delay 1 in the second game: insertion paints nothing and the next area update paints stage zero, then every third update; life grows by one tick as Blizzard's does. The first game keeps stage zero at insertion | UNKNOWN | The same unread insertion phase as DIV-3049; the claim composition is Medium | The bounded original trace DIV-3049 asks for, taken for Acid Stream | OPEN |
| DIV-5145 | ROM2 magic / Poison Cloud on a fallen unit | Owner: make it as in the original; the second game takes the first game's rule | `R2-ENGINE-1575`, `R2-ENGINE-1576` give the second game's renewal and a duration of 128 only; no second-game claim reads the pulse on a fallen target or the teardown expiry | The second game runs the first game's rule of DIV-5144: the pulse tests no health of its target and the body's teardown expires its attachments | UNKNOWN | The rule is read for the first game only | A second-game observation as DIV-5144 asks | OPEN |
