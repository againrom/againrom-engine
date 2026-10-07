# Area-layer movement cost

## Intent and authority

A cell under an area layer (Wall of Fire, Wall of Earth, Poison Cloud and the other layer spells) changes its mutable movement-cost byte in the original. The engine now models that byte. Authority, snapshot k117: `MAGIC-AREACOST-046` (High for the instructions; the net effect in a running session was Unknown), `MOVE-084` (High / Medium / Unknown), `MOVE-085` (High / Medium / Unknown), `MOVE-086` (High / Medium / Unknown) and `TERR-STRUCT-071` (High).

## As built

- A recompute of a cell shifts its baseline cost left by two once per occupied layer slot, in eight bits. One layer gives 24..64 for the shipped range 6..16, two layers give 96..240 and 0 for 16, three layers give 128 for cost 6 and 0 for 8, 12 and 16. Slot order is the engine's layer table (3, 7, 8, 19, 12, 17); Wall of Earth takes the shift like any other slot.
- A ground mover's transit start reads the source cell and then the destination cell before any recompute. Each read of a layered cell divides the stored byte by four and stores the quotient back, whatever the layer count. One layer answers the baseline, then a quarter, then 0 (1 at cost 16). Two readers of one cell with no recompute between them divide twice. A cell with no layer answers its byte unchanged.
- The route search prices a step with the stored byte, not the divided one, so a layered cell costs four times its baseline to every route and a decayed or truncated byte is priced as stored.
- A cell's recompute restores the byte from the baseline. It runs when the layer set of the cell changes and at the crossing of a transit, for the cell left and the cell entered.
- Actor updates run before the area pass. A native world applies a layer set to the cost byte once per tick after every actor update, so a layer first present during a tick is read as no layer by that tick's movers.
- A world with a saved cell plane (an original SAV, or one derived from it) keeps the byte in the plane. The read divides and stores there, and a layered cell is a record with static bit 5 set and a non-zero layer count or a native layer. A native world keeps the byte in `World.areaCosts`, one entry per layered cell (cell key, layer mask, byte).
- No cost plane is saved. `World.areaCosts` reaches the world form only as a trailer (form 104) listing the cells whose byte differs from a recompute, and a world without the trailer recomputes. After an original SAV load, `RecomputeLayeredCosts` rewrites the byte of every layered cell from its record.
- The two original readers that divide by the byte with no zero test have no counterpart: the rate law divides by the mean of the two reads and takes the original's own substitute of 8 for a mean of 0, and the route search adds the byte. A stored 0 faults nowhere.

## Divergences

- `DIV-021` is closed.
- `DIV-1841`: zero divisor of the two inline readers, guarded by absence (ACCEPTED).
- `DIV-1842`: original SAV load keeps the ingest byte of a layered cell in the original; the engine recomputes it (OPEN).
- `DIV-1843`: crossing recompute is one derived tick, with no slot-based skips, none for a mover that dies or is redirected first (OPEN).
- `DIV-1844`: layer recompute order against the actors in a saved-plane world and the engine's effect step (OPEN).
- `DIV-1845` and `DIV-1846` are returned unused.

## Proof

- `pkg/sim/areacost_test.go`: the decay table for one and two layers at every cost 6..16, three layers at 6, 8, 12 and 16; recompute restoring the byte; an unchanged layer set keeping its decay across the area pass; a layer present for its first tick read as no layer by that tick's movers; a second actor reading the byte the first left; a crossing recomputing both cells on the derived tick; the rate query never storing; a zero byte through the rate query, the transit start and the route search; the byte form omitting undecayed cells and round-tripping a decayed one; the saved-plane read and the load recompute.
- `TestReleaseAreaLayerMovementCost` (`pkg/game`, EN and RU roots): on mission 20's terrain, cost plane and spell rules, a caster raises the install's layer-painting cloud. One mover leaves a layered cell as another enters it on the same tick. The first transit equals the unlayered baseline, the second equals the transit a cell of a quarter of the byte gives on bare ground, and the same assertion fails over bare ground.
- `TestReleaseAreaLayerCostContinuesAcrossSaveAndLoad` (`pkg/game`, EN and RU roots): a mage raises a layer, the mission is saved and cold-loaded, and the step onto the layered cell has the same rate query, transit and position in both worlds.

## Open debt

- Unknown in the claims: whether two reads of one cell without a recompute happen in play, whether a decayed zero reaches an inline reader, and the tick on which a crossing falls.
- The effect step still runs before the actor updates (`DIV-1844`).
- The route search reads the multiplied byte for every layered cell. Whether this changes a shipped scenario's routes is covered by the scenario gate named in the return.
