# A second-game body decays at the original's rate

## Result

In the second game a dead unit's body now moves through its corpse stages and leaves the world at the rate the
original does. Before, it did so one and a half times slower, so it stayed raisable longer. The first game is
unchanged.

## Authority

- Owner: «Починить» on decision 7 of the second-game triage.
- R2-ENGINE-1207: the refill takes 1 health below 0 on one pass in four.
- R2-ENGINE-776: corpse ageing takes 1 health on each even pass.
- DIV-4916 states the gap; it keeps only the list membership, which the claim grades Medium.

## Rules

1. A second-game unit below 0 health loses 1 health on one pass in four, and 1 on every second pass.
2. A unit at 0 health loses nothing to the refill.
3. A body reaches each corpse stage at the health the loss rate names.
4. The first game's bodies lose health as before.

## Code

| kind | its one builder | change |
|---|---|---|
| world pass refill | `World.regenPass` | one profile flag calls the below-zero loss |
| decay walk | `World.decayPass` | none |

## Proof

| test | rules |
|---|---|
| `TestSecondGameBodyLosesHealthAtTheClaimsSum` | 1, 2, 3 |
| `TestFirstGameBodyLosesNoPoolPassHealth` | 4 |

Release tests at the release: none.

## Open debt

- Which classes the pass's list holds is Medium; every on-map unit below 0 is taken.
- The state send and the `+0x54` = 0x10 return are not modelled.
