# The second game draws magic effects on sloped ground and held-mark casts as the original does

## Result

In the second game a spell effect on a slope now stands at the height of the ground under its own point,
not at the middle height of its cell. Teleport's two images, flying spell objects and both ends of a
Lightning bolt follow the slope across the cell. A hero or creature casting Poison Cloud or Curse no
longer shows a separate spell picture; only the mark on the unit is drawn. The first game is unchanged.

## Authority

- Owner: «in the second game a spell shows and sounds as the original's instead of borrowing the first-game row».
- R2-ENGINE-1518: an effect's ground term is the bilinear blend of its cell's four corner bytes, 5-bit weights.
- R2-ENGINE-1516: a Lightning path's start and end subtract that ground term.
- R2-ENGINE-691, R2-ENGINE-1520: a unit caster's object of 20 and 64 has life 0; only the held mark stays.
- R2-ENGINE-1515, 1516, 1519: path figure, point type, Teleport raise and first frame, which already match.

## Rules

1. In the second game an effect record stands at the ground blended at its fine point.
2. In the second game both ends of a Lightning or chain path subtract the ground blended at their point.
3. In the second game a unit caster's Poison Cloud and Curse object ends on its first update.
4. In the first game records and path ends keep their cell's mean corner height, and identifier 20 keeps its record.
5. A path point stores type 28 for Lightning and the victim index mod 7 for the chain.

## Code

| kind | its one builder | change |
|---|---|---|
| height of an effect on sloped ground | `pkg/ui/cellprojection.go:55` groundHeight | point blend by `PointHeight` |
| point ground sample | `pkg/render/terrain/project.go:195` PointHeight | new method beside AnchorHeight |
| spell drawing, record | `pkg/ui/spellbolt.go:217` spellBoltLift | `PointLift` from the spell set |
| Lightning path ends | `pkg/game/spellpath.go:114` boltDisplayPoint | height by the set's rule |
| Lightning figure, point type | `pkg/game/boltfigure.go:54` boltFigure, `spellpath.go:72` boltTag | unchanged, matches |
| cast object life | `pkg/sim/flightproducer.go:170` unitSegments | rows 20 and 64 take `UnitLifeZero` |
| per-game data | `pkg/data/spellpresentation.go:279` second set | `PointLift: true` |
| Teleport frames, held mark | `pkg/rules/spelleffect.go:126`, `pkg/game/effectmark.go:78` | unchanged, match |

## Proof

| test | rules |
|---|---|
| `TestSecondGameEffectStandsAtTheGroundBlendedAtItsPoint` | 1, 4 |
| `TestSecondGamePathEndSubtractsTheGroundBlendedAtItsPoint` | 2, 4 |
| `TestSecondUnitCasterOfAHeldMarkSpellEndsItsCastRecordAtOnce` | 3, 4 |
| `TestThePathPointTypeIsTheIdentifierOrTheVictimIndexModSeven` | 5 |

Release tests at the release: none.

Evidence: `review/story1738-rom2-magic-drawn/RESULT.md`.

## Open debt

- DIV-3101, DIV-3102, DIV-3103, DIV-3104, DIV-3106, DIV-3108, DIV-4255, DIV-4289 and DIV-2622 stay Unknown.
