# Story 1144 — restore dying actors still in their owner's list

An original mission SAVE can contain a stage-1 actor in its Player list.
LOAD currently excludes that actor from the living registry while the separate
late-dead manager does not contain it. Its ALM placement remains alive. The
owner's `game0031` turns map unit 32 from HP 0/stage 1 into HP 20/stage 0;
`game0032` turns unit 54 from HP -8/stage 1 into HP 30/stage 0.

## Result required

Restore each supported dying owner-graph actor's source binding, position,
health, stage, current profile, holdings and load through both original LOAD
doors. A fresh native SAVE/LOAD must retain it and continue the same native
simulation. Loading must not award death rewards or replay a loot drop.
Existing late-dead records and MapUnitID-0 hirelings remain controls.

`SAV-DEADLOAD-126` establishes independent stored health, stage and timer;
`SAV-DEADLOAD-128` names the first dying tick's stage 1. These do not make
the Player list interchangeable with the separate manager walked by
`SAV-DEADLOAD-124`. The prior SAV-DEATH-051 corpus observation that all
owner-graph actors were alive does not cover these new files.

## Proof and limits

The seat's read-only census on base f566f49 finds 66 stage-1 owner-graph
records excluded by the Dead predicate over 62 mission files. Exactly two
have an unambiguous living ALM replacement, on both asset roots. The other
64 already matched a nonliving ALM actor, but their source profiles and
holdings were still excluded. All 66 now bind by archive identity. Sources
and initial results are staged outside Git under
`review/milestone2-dying-owner-actors/`.

The independent milestone-2 witness must discover its population and name
exclusions. This story does not build a world SAV exporter. Any gap between
native dying continuation and ROM1 runtime remains explicit divergence debt;
no new ROM1 runtime witness is claimed from engine tests.

The existing actor registry admits stage 1 with nonpositive HP and a nonzero
runtime ID. Source construction restores its actual HP/stage from the start;
the stock, profile, pool, spellbook and facing import seams accept this bounded
dying state. Action cleanup follows each construction and the final
document/group overlay without running either death transition, preserving
the source stage even at terminal health with a zero timer.

The independent `TestMilestone2DyingOwnerActors` reads HP/max HP, mana/max
mana, stage, timer, cell coordinates and own weight/load directly from file
bytes. The decoder supplies actor enumeration and serialized-run start
offsets only. It discovers 66 subjects in 16 files, all compared, zero
refusals and mismatches. The previous weight audit now agrees on all 2330
map-bound actors. These are focused EN results; final EN/RU evidence follows.

`TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect` exercises both
original LOAD doors on the two discriminating files, ordinary App menu
SAVE, a fresh FrontEnd LOAD, and 64 driver ticks with equal world hashes.
It also checks the held weapon and stored weight/load of unit 54.
`TestOriginalDying1144CleanupIsAtomicAndDoesNotReplayDeath` checks batch
atomicity, no first-death replay, native wire persistence and continuation.

No wire version changes. The timer uses the existing native dwell field.
DIV-1003 records the native continuation and negative-timer refusal boundary;
DIV-1004 through DIV-1008 retire unused.

## Single review correction

The sole adversarial pass returned eebb7f9 with two material findings
(`pipeline/reviews/story1144-pass1.md`). F-1 found that HP -10/stage 1/timer 0
could enter the terminal loot transition during LOAD, despite bypassing the
first-death arm. `clearFelledActions` now separates action cleanup from both
death transitions; the existing native death path calls that same helper in
its original order. The release witness adds a temporary two-field mutation
of game0032 and proves both LOAD doors, no initial item loss, native SAVE/fresh
LOAD and 64 subsequent driver ticks.

F-2 found a shallow staged Group mutation surviving a late duplicate-source
identity refusal. Construction now detaches saved Groups, motion and pending
cast collections before cleanup. Two permanent sim regressions preserve the
zero-timer source tuple/holdings and the original world hash after that late
refusal. The focused sim controls and extended EN release witness pass.
