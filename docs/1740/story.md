# The second game aims an area spell cast at a unit from the caster's cell to the target's cell

## Result

In the second game an Acid Stream, a wall or another area spell cast at a unit now points from the
caster's cell to the cell it lands on, as the original does. Before, it took the caster's facing, so
a target that moved during the wind-up or a caster larger than one cell could turn the cone aside.
A creature keeps outliving its dead caster, and an Acid Stream at the map edge keeps dropping the cells
past the border whole. The first game is unchanged.

## Authority

- Owner: «продожаем по списку» (epic: the second game in full): adopt the Summon and target research.
- R2-ENGINE-792: every second-game area build stores the heading from the caster to the target position.
- R2-ENGINE-1545: that heading is the octant of the cell offset, ties to the diagonal, a zero offset 7.
- R2-ENGINE-1546: per heading the cone takes the builder table's cells; a cell past 8 or size minus 9 is skipped.
- R2-ENGINE-1549 (Medium): no Summon key site tests the caster.
- R2-ENGINE-1547, 1548, 1550: a cast at a gone target and Stone Curse on a building stay Unknown.

## Rules

1. In the second game an area built at a cell takes the octant from the caster's cell to that cell.
2. In the first game an area cast at a unit keeps the caster's facing; a cast at a cell takes the octant.
3. The octant has slope thresholds one half and two, a tie goes to the diagonal and a zero offset gives 7.
4. A cone cell whose x or y lies below 8 or above the map size minus 9 is skipped whole, in both games.
5. A summoned creature whose caster dies keeps its key and lives on; the cap never takes a key's last actor.
6. A queued area delivery saves the heading its game's rule gives.

## Code

| kind | its one builder | change |
|---|---|---|
| area heading | `pkg/sim/celleffect.go:507` areaHeading | new rule row over `areaDirection`; landing and save call it |
| per-game data | `pkg/sim/gameprofile.go:97` areaHeading | second game `areaHeadingCells` |
| heading octant | `pkg/sim/heading.go:11` headingOf | unchanged, matches |
| Acid Stream cone | `pkg/sim/areaarms.go:187` acidStreamStage | unchanged, matches |
| cone edge | `pkg/sim/celleffect.go:614` ringCellsAtOffsets | unchanged, matches |
| summon cap, caster link | `pkg/sim/secondspell.go:209` summonCapPass | unchanged, matches |
| cast at a gone target | `pkg/sim/spell.go:2097` releaseBookRefusal | unchanged, first game's rule |

## Proof

| test | rules |
|---|---|
| `TestSecondAreaHeadingIsTheCellOctantNotTheFacing` | 1, 2, 6 |
| `TestSecondAreaHeadingSectorsFollowTheSlopeThresholds` | 3 |
| `TestSecondAcidStreamSkipsCellsPastTheInsetWhole` | 4 |
| `TestSecondSummonedCreatureOutlivesItsCaster` | 5 |

Release tests at the release: none.

Evidence: `review/story1740-rom2-summon-dead/RESULT.md`.

## Open debt

- DIV-2618, DIV-4388, DIV-4431, DIV-4432, DIV-4385, DIV-4386, DIV-4387, DIV-2620 and DIV-4907 stay Unknown.
