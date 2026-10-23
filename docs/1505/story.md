# First-game chase and post rules follow the original

## Intent

In the first game four AI rules now match the original:

1. A Darkness hit gives its victims no order. It runs the blow hook only.
2. The post block counts only living enemies.
3. An AI-owned walk with no victim is refused at once when its near search aims at its own
   destination and finds nothing.
4. A group drops a foe that strikes it from outside the group's circle.

The second game keeps every earlier answer. Each change is gated on `!w.SecondGame()`.

## Authority

- Owner direction, in the owner's words: «Делай как рекомендуешь: 1 — да, три расхождения без
  моего решения привести к оригиналу; 2 — вся группа по-прежнему вступает в бой, но бросает
  погоню, когда обидчик вышел из круга, как в оригинале.» Part 1 covers the Darkness, post-block
  and walk rules. Part 2 narrows DIV-1523.
- Claims at knowledge k348:
  - `AI-707` (High / Medium): a Darkness hit writes no pursuit. It writes only the hook's turn
    flag, remembered cell and diplomacy flip. `MAGIC-LIGHTDARK-044` reads Darkness calling the
    blow hook.
  - `AI-704`: the post block skips a dying body (health 0 through -9).
  - `AI-709` (Medium on this path) and `AI-373`: a walk home whose near search finds no labelled
    cell is refused, and the next guard pass sends the walk again.
  - `AI-GRPGUARD-074`, `AI-RADFREEZE-075`, `AI-SWARM-022`, `AI-ROAM-025`, `AI-RETAL-056`: a group
    scores only the candidates inside its clipped list. Guard clips to the notice circle; Swarm,
    Swarm 2 and Roam clip to the group's sight.

## As built

| Rule | Site at the head (before) |
| --- | --- |
| Darkness, live hit | `pkg/sim/celleffect.go:893-903` (893-895): arm 17 calls `flipOnBlow(ci, i)` in the first game and `orderAttack` in the second |
| Darkness, pulse resumed from a SAV payload | `pkg/sim/savedcontinuation.go:367-372` (367): `flipOnBlow` on the loaded caster in the first game |
| Post block | `pkg/sim/guardarm.go:62-75` (62-71): `postCandidates` drops a candidate with HP < 1 in the first game |
| Walk refused at once | `pkg/sim/combat.go:418-426` adds `refusesWalkAtOnce`; `pkg/sim/step.go:1131-1141` (1125-1139) refuses through `answerRefusedPursuit(i, searchNear)` |
| Engaged foes inside the circle | `pkg/sim/groupengaged.go:65-85, 129-160, 196-198` (71-92): `withEngagedFoes` and `joinEngagedGroup` keep only foes in the clipped list (`clipCircle`, `engagedCircle`, `inCircle`); `pkg/sim/engage.go:537` and `pkg/sim/savedgroupsai.go:257` build that list through `clipCircle` |

The post block's callers are `postEngage` and `postRefused`. They are reached from:

- `armGuard`: the guard state (`actor.go:70`) and `armPatrol` (`actor.go:205`);
- `savedGuard`: `savedPatrol` (`savedgroups.go:440`) and state 0xb (`savedgroups.go:472`).

All four change in the first game.

A walk that keeps the stall count of 16:

- a human participant's walk (slot 0 counts as human);
- an AI walk whose near search aims at a waypoint;
- every second-game walk.

A SAV that was loaded while a Darkness cloud was being resumed still loads. The cloud's next pulse
runs the blow hook on each victim: it remembers the caster's cell, raises the turn flag on a loaded
order and flips diplomacy. The pulse writes no order. A victim pursues the caster only when its
group's next pass finds the caster inside the circle.

## Proof

Focused tests. Each has a loss control: the old line was restored, the test failed, and the file was
restored from the commit.

| Test | Loss control result |
| --- | --- |
| `TestADarknessHitSendsNoVictimAtTheCaster` | "tick 15: Darkness victim 2 holds victim 1" |
| `TestALoadedDarknessPulseSendsNoVictimAtTheCaster` | the loaded pulse gave its victim victim 1 |
| `TestADarknessCasterInsideTheCircleIsTakenByTheNextGroupPass` | no victim remembered the caster's cell |
| `TestThePostBlockEngagesNoDyingBody` (both arms) | both arms engaged the dying body |
| `TestAnAIWalkHomeAimedAtItsPostIsRefusedAtOnce` | 15 stalled passes |
| `TestAGroupStruckFromBeyondItsNoticeCircleLeavesTheShooter` | decide and join pass each: "tick 37: member 1 took the shooter" |

- The three Darkness tests are the headless Darkness witness. The live-hit control shows the earlier
  rule: each victim attacked the caster.
- `TestAGroupStruckFromBeyondItsNoticeCircleLeavesTheShooter` replaces
  `TestAGroupStruckFromBeyondItsNoticeCircleAnswersWithEveryMember`.
- The two refused-retreat tests in `refusedwithdrawal_test.go` now retreat a human participant, so
  they keep the stall-16 path they test.

`kite.sav` witness (`review/kite-chase/engine-kite.sav`, RU, mission 20): the `pocket` plan and the
bandit plan, main against the head. The traces are byte-identical.

- Goblins (group 13):
  - goblin 21 died at tick 77707, 36 cells from its post;
  - goblins 22-25 held the hero until the hero died at tick 77730, 40, 30, 41 and 45 cells from
    their posts.
- Bandits (group 12):
  - bandits 12, 14-20, 29 and 30 released the hero at tick 77814, 5 to 8 cells from their posts;
  - bandit 13 released the hero at tick 77835, 5 cells from its post.

In that save the hero only moves and never strikes from outside the circle. The circle follows the
group (`AI-RADFREEZE-075`).

Release sets with `AGAINROM_HEADLESS_ONLY=1`:

| Root | Head | Main | Fails |
| --- | --- | --- | --- |
| EN | 1290 of 1414 ran | 1291 of 1414 | 0 |
| RU | 1288 of 1414 ran | 1289 of 1414 | 0 |

On each root the skip lists differ only by `TestReleaseActivePauseRenderedWorld`, the expected
headless skip. Mission-stage scenarios passed 76 of 76 across EN and RU. No release test or
scenario failed, so no world hash moved.

## Open debt

- DIV-1600 stays open for the second game. No claim reads that game's Darkness arm, so a hit there
  still orders each victim at the caster.
- DIV-1523 stays a DEVIATION. Two parts are this build's own: the join pass for members that a
  decision leaves out, and the engaged set.
- `AI-709` is Medium on the walk-home path and on one retry per group pass.
