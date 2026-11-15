# Light and Darkness change sight by class, as in the original

## Result

Light no longer widens anyone's sight, the player's heroes included. A Light cloud makes creatures standing
in it see less far and leaves humans and heroes as they are. A Darkness cloud makes humans and heroes see
less far and leaves creatures as they are; it still credits the caster with a kill and still provokes the
unit it reaches. Flying units and buildings take neither. Both games follow the same rule.

## Authority

- Owner, board card `o-light-player-sight`: «Как в оригинале»: Light widens nobody's sight, heroes included.
- Owner, DIV-4834: «Тьма обязана снижать обзор» (kept); DIV-4834 and DIV-4835 keep the hold between pulses.
- `MAGIC-443`, `MAGIC-446`: Light takes p/30+1 cells off a plain Unit's sight, Darkness off a Humanoid's.
- `MAGIC-442`: a cloud pulse lands on ground actors only; `MAGIC-445`: only Darkness credits and hooks.
- `R2-ENGINE-1237`, `R2-ENGINE-1238`, `R2-ENGINE-1242`: the same split by the Humanoid predicate.
- `R2-ENGINE-989`: a Unit-class creature under Darkness keeps its sight.

## Rules

1. When a Light cloud reaches a ground unit that is not Humanoid, its sight falls by power/30 + 1 cells.
2. When a Light cloud reaches a Humanoid, its sight and its owner's fog stay as they are.
3. When a Darkness cloud reaches a Humanoid, its sight falls by power/30 + 1 cells.
4. When a Darkness cloud reaches a unit that is not Humanoid, its sight stays as it is.
5. A cut sight stops at 0 and comes back whole when the effect ends (DIV-4949).
6. A unit standing in either cloud keeps its cut sight between pulses (DIV-4834, DIV-4835).
7. Both games follow rules 1 to 6 with no game test in the arm.

## Code

| kind | its one builder | change |
|---|---|---|
| spell arm | `pkg/sim/spellarms.go` `spellArms` | rows 12 and 17 gain `sightClass`; `sightCut` reads it |
| effect attach | `pkg/sim/effect.go` `attachEffect` | the magnitude passes `sightCut` for the victim's class |
| untimed effect apply | `ordinaryEffectPayload`, `applySavedAreaPayload`, the saved delivery | the same `sightCut` |
| divergence area file | `docs/divergences/rom2-magic-light-and-darkness.md` | split from `rom2-magic.md` at its ceiling |

## Proof

| test | rules |
|---|---|
| `TestLightCutsACreaturesSightAndDarknessAHeros` | 1, 2, 3, 4, 5, 7 |
| `TestTheFogStampFollowsTheSightClassUnderLightAndDarkness` | 1, 2, 3, 4, 5, 7 |
| `TestAComputerUnitInDarknessSweepsAShorterRange` | 3, 7 |
| `TestALightOrDarknessCloudHoldsTheSightBetweenPulses` | 6, 7 |
| `TestShieldLightAndDarknessLandTheirDecodedMagnitudes` | 1, 3 |
| `pkg/mapload` `TestADarknessCloudShrinksASourceHumansSight` | 2, 3, 7 |
| `pkg/game` `TestTheVisibleFogFollowsADarknessOrLightCloudWithoutAMove` | 2, 3, 6, 7 |

Release tests at the release: `TestReleaseCloudDeliverySAVContinuation`, `TestReleaseNativeAreaSAVContinuation`,
`TestReleaseFogStampAfterColdLoad` on rom1-en, rom1-ru, rom2-en and rom2-ru.

## Open debt

- DIV-4949: a second-game renewal by a stronger Light cloud keeps the weaker cut.
- DIV-4950: second-game objects that are not units stay unreached; research leaves their reach Unknown.
