# A second-game spell in flight lands without a caster after LOAD

## Result

A second-game mission save that holds a point spell or an area spell in
flight now lands it after LOAD as the original's records allow. The spell
takes the target's health, and since the file stores no caster it records no
attacker memory, turns no side hostile and gives no kill credit. The same
holds for the save the game writes itself and loads again.

## Authority

- R2-ENGINE-1398 and R2-ENGINE-1399: no LOAD reader stores the caster of a
  spell effect or of a direct damage record; a loaded effect still reaches
  its target's apply path, which skips attacker memory, hostility and credit
  while the caster is 0 (Medium).
- R2-ENGINE-1400: the timed area pulses when its word falls to a multiple of
  16 and ends at 0 or below.
- R2-ENGINE-1401: a caster saved in recovery resumes its own count.
- DIV-4175, DIV-4423 to DIV-4427: rows narrowed here.

## Rules

1. A loaded point effect takes its target's health, with no credit or hostility.
2. A loaded area effect takes the health of the units in its cells the same way.
3. A save of such a flight, loaded again, lands as the live game does.
4. The first game's load and spell landing are unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| spell record read | `pkg/game/savworldeffects.go` bindOriginalWorldEffects | unchanged; the claims need no new field |
| spell landing | `pkg/sim/savedspellgraph.go` applyFrozenPoint | unchanged; a caster-less node takes the path |

## Proof

| test | rules |
|---|---|
| `TestSecondMissionFlightLandsWithoutACaster` | 1, 2 |
| `TestSecondMissionFlightSavesAnAreaAndColdLoads` | 3 |
| `TestSavedPointWithoutCasterLeavesNoMemoryHostilityOrCredit` | 1 |
| `TestSecondMissionFlightContinuesAfterLoad` | 1, 4 |

Release tests at the release: none new; the second game's mission-file LOAD
set and the first game's spell-in-flight set run unchanged.

## Open debt

- DIV-4175: a LOAD whose document did not decode binds no effect.
- DIV-4423, DIV-4424: the queued point's tick and the amount without a
  caster are Unknown (R2-ENGINE-1399, 1400).
- DIV-4425, DIV-4427: no claim of Prj16's link or an area's cell planes.
- DIV-4426: the hero's saved recovery count is Unknown (R2-ENGINE-1401).
