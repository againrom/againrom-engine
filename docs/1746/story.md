# Second-game mana debit before application refusal and area-hit hostility

## Result

In the second game, casting a spell whose target or landing cell refuses the effect now spends the mana before the
refusal; casting without enough mana still spends nothing. A neutral unit caught in an area spell's cell hit turns the
caster hostile toward it at once but turns hostile back only after more than ten area hits, below half health or dead;
its hit count survives saving and loading. A poison tick is not an area hit here and still turns the victim both ways
at once. Direct blows and the first game keep their behaviour.

## Authority

- Owner: remarks 2 and 5 (second-game cast debit before application refusal; area-hit reverse hostility gated by
  counter, half health or dead state, reading closest to the claims).
- `R2-ENGINE-544`: mana is debited at cast-routine entry before delivery, once per call reaching delivery.
- `R2-ENGINE-026`: Heal refuses a target at -10 health or below or with zero maximum health after the debit.
- `R2-ENGINE-791`: a failed Summon placement keeps the debited mana.
- `R2-ENGINE-018`: a caster without enough mana is refused and pays nothing.
- `R2-ENGINE-981`: hit hook `(A, B, F)` ORs bit 0 into `M[A][B]` if low two bits are zero; returns if `M[B][A]&3` is
  nonzero; else ORs bit 0 into `M[B][A]` when `F=0`, command `+0xac` signed > 10, health < trunc(maxHealth / 2), or
  state `+0x54` = `0x10`; with `F` nonzero then increments `+0xac`.
- `R2-ENGINE-985`: an area cell occupant calls the hit hook with `F=1` without an owner or relation test.
- `R2-ENGINE-1547`: immediate stores of `0x10` into `+0x54` lie on the death path.
- `R2-ENGINE-845`: the hit hook writes the victim's command block, the 184-byte raw Unit block the SAV stores, so
  `+0xac` of `R2-ENGINE-981` is that block's byte `+0xac`.
- `DIV-5095`: cast refusals not placed by the claims (range, busy, non-restorative target state) stay before debit.
- `DIV-5096`: `+0xac` is read before it grows (hits 1..11 defer from 0; hit 12 flips) and state `0x10` is `!Alive()`.

## Rules

1. In the second game, Heal on a target at -10 health or below or with zero maximum health charges mana and refuses.
2. In the second game, a point-effect refusal or failed Summon placement charges mana and applies nothing.
3. In the second game, an area-landing refusal on a cell or unit cast charges mana and lands nothing.
4. In the second game, insufficient mana and out-of-range refusals still charge nothing.
5. In the second game, an area spell's cell hit sets forward hostility, defers reverse if no gate holds, grows hits.
6. In the second game, reverse hostility flips if count (read first) > 10, health < trunc(max/2), or dead (`!Alive()`).
7. A direct blow and a poison tick (both ways at once) in the second game, and the first game, keep as was.
8. Light and Wall of Earth cells never call the hook (`R2-ENGINE-985` returns before step 8), so they turn nothing.
9. The victim's area-hit counter persists in the engine byte form, current actor values and SAV unit command `+0xac`.

## Code

| kind | its one builder | change |
|---|---|---|
| cast payment order | `pkg/sim/spell.go` `castSpell`, `castBookAt`, `bookSpellAdmissionRange`, `bookSpellCellAdmissionRange` | `castPayRules` row in `gameProfile` orders debit before application refusals for second game |
| blow relations | `pkg/sim/world.go` `flipOnBlow`; `pkg/sim/celleffect.go` `answerAreaBlow` | `blowRelationRules` row in `gameProfile` gates reverse hostility on `blowArea` and grows `AreaHitCount` |
| area-hit persistence | `pkg/sim/areahitbinary.go`, `pkg/sim/currentvalues.go`, `pkg/sim/guardstrike.go` | form wrapper 141 (`AHC1`), `ActorValues.AreaHitCount`, `ImportLoadedStrikes` restore |
| second-game SAV command | `pkg/formats/sav/engine_form.go`, `pkg/game/originalgroups.go`, `pkg/game/savform.go` | read and write `CommandStrike.AreaHits` at `URaw184` offset `+0xac` |

## Proof

| test | rules |
|---|---|
| `TestSecondGameHealRefusalChargesMana` | 1 |
| `TestSecondGamePointEffectRefusalChargesMana`, `TestSecondGameSummonFailedPlacementChargesMana` | 2 |
| `TestSecondGameAreaLandingRefusalChargesMana` | 3 |
| `TestSecondGameInsufficientManaChargesNothing` | 4 |
| `TestSecondGameAreaHitDefersReverseHostilityWhenNoConditionHolds` | 5 |
| `TestSecondGameAreaHitFlipsReverseWhenCounterExceedsTen` | 5, 6 |
| `TestSecondGameAreaHitFlipsReverseBelowHalfHealth`, `TestSecondGameAreaHitFlipsReverseInDeadState` | 6 |
| `TestSecondGameDirectBlowStillTurnsBothDirections`, `TestSecondGamePoisonTickTurnsNeutralVictimAtOnce` | 7 |
| `TestFirstGameApplicationRefusalsKeepManaUnchanged`, `TestFirstGameAreaHitStillTurnsBothDirectionsImmediately` | 7 |
| `TestSecondGameLightAndWallOfEarthCellsRunNoHitHook` | 8 |
| `TestSecondGameAreaHitCountSurvivesBinaryAndValuesRoundTrip`, `TestSecondGameUnitCommandPreservesAreaHitCounter` | 9 |

Release tests at the release: none.

## Open debt

- `DIV-5095`: cast refusals not placed by the claims (such as range, caster busy, and non-restorative target state)
  still refuse before the mana debit; awaits claims placing those checks relative to the cast debit.
- `DIV-5096`: whether `+0xac` has other writers/resets and whether state `0x10` covers any state beyond death stay open
  (chosen reading: read `+0xac` before incrementing; treat state `0x10` as `!Alive()`); awaits complete writer claims.
  Also open: whether the original's poison tick runs the area-hit hook is Unknown; the tick (`poison.go` to
  `awardPoisonTick` to `flipOnBlow`, no area flag) still turns a neutral victim both ways at once, as before the story.
