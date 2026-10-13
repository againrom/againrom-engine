# Casts start as the first game starts them

## Intent

Every cast the engine starts in first-game play is compared with what the
claims of k297 and k299 state for its source. One difference with High-grade
support and a player-visible effect is changed; the others are recorded.
Started at public main `78a92259`; the pin moves k289 to k299.

## Authority

`AI-609`..`AI-611`, `AI-617`..`AI-620`, `MAGIC-340`, `MAGIC-341`, `MAGIC-311`,
`MAGIC-348`, `MOVE-171`, `TRIG-CASTACTOR-044`, `AI-554`, `AI-561`,
`MAGIC-239`, `MAGIC-REACH-179`, `MAGIC-REACH-180`, `MAGIC-CADENCE-127`.
Owner rules that stand: DIV-3191 (automatic cast moment), DIV-474 (a click is
one cast), DIV-1015 (a new order interrupts a recovery). No claim contradicts
them.

## Comparison

Grades are High (H) or Medium (M) for the clause. Verdicts: agrees, differs,
not modelled (NM), not observable (NO).

| # | Source / behaviour | Claim, grade | Engine site | Verdict |
|---|---|---|---|---|
| 1 | Player cast: setters park the Spell, test nothing else | AI-487, MAGIC-340 H | `manualcast.go` `beginManualCast` tests at the click | differs; click dropped (DIV-3266) |
| 2 | Unit gate: self bypass, footprint-gap distance <= reach | MAGIC-REACH-179 H | `spell.go` `withinCastReach` | differs on main (Chebyshev of anchors); changed in the first game, the second keeps main |
| 3 | Cell gate: whole-cell max axis <= reach | MAGIC-REACH-180 H | `bookSpellCellAdmissionRange` | agrees |
| 4 | Gate failure walks toward target, order stays | MAGIC-239 H/M | refused, except Teleport | differs; DIV-1310 (existing, updated) |
| 5 | Admission refuses on mana only, mage mask and no scroll | MAGIC-340, MAGIC-348 H/M | `bookAffords`, `bookCaster` | agrees |
| 6 | Refused cast retries to the progress-2 exit | MAGIC-340 M, MAGIC-CADENCE-127 H | no retry (DIV-474) | differs by owner rule; NO in play |
| 7 | Manual cast runs one cycle (ord+0x60 = 0) | AI-489 H/M | one-shot record | agrees |
| 8 | New order in step or cast waits for progress 0 | MAGIC-341, MAGIC-311 M | `PendingOrder`, DIV-1015, DIV-3161 | agrees / existing rows |
| 9 | Stand (0x17) with mover+0x98 ends a cast | MAGIC-341, MOVE-171 M | none | NM (DIV-3267) |
| 10 | Scroll arm: item and type 0x29 tests; 0x26 undoes for a non-cell Spell | AI-617 H/M | `usescroll.go` `beginScroll` | agrees; unit form extra target test (DIV-3266) |
| 11 | Scroll reach gate, then approach | MAGIC-239 H/M | `scrollInReach` | differs on main (Chebyshev); changed in the first game, the second keeps main |
| 12 | Scroll costs no mana | MAGIC-340 H | `beginScroll` | agrees |
| 13 | Automatic cast | DIV-3191 owner | `castOrder` at order phase none | owner rule stands |
| 14 | Heal arms R0015 / R0113 | AI-610 M | idle heal, DIV-126, DIV-3009 | owner rules stand |
| 15 | Computer mage: hand-back, affordable non-defensive ids, uniform draw | AI-554 H/M | `engage.go` `aiCast` | agrees |
| 16 | Computer mage: cast order at the engage victim | AI-554 H/M | own target pick | differs; caller unknown (DIV-3268) |
| 17 | Computer mage: empty list with id 28 still casts | AI-554 H | not built | NO (no shipped book holds id 28) |
| 18 | Retained flag ord+0x60 = 1 for selector casts | AI-554 H | `Retained` records | agrees |
| 19 | Creature slot draw and approach | AI-609, AI-610 H/M | `creatureEngageCast` | agrees; extra refusals DIV-1725 |
| 20 | Cell-entry and script casts: no range, sight or allegiance test | AI-618 H/M | `scriptcast.go`, `cellentry.go` | agrees (extra target test, DIV-541) |
| 21 | Wrapper power: word, default 99, spell byte | TRIG-CASTACTOR-044 H | `scriptCastPower` | agrees |
| 22 | Wrapper starts: mana of the new actor | AI-618 M | temporary caster is not an actor (DIV-541) | NM |
| 23 | Admission callees test no range or sight | MAGIC-348 H/M | release re-test | differs (DIV-3269) |
| 24 | Release stores actor+0x138 | AI-561 H | `CastWait` | NO |
| 25 | Mana spent once at admission | MAGIC-311 M | `admitBookPayment` | agrees |

Counts (25 rows): 12 agrees (3, 5, 7, 8, 10, 12, 15, 18 to 21, 25); 6 differs
(2 and 11 changed in the first game; 1, 4, 16 and 23 recorded as rows); 1 differs
by owner rule (6); 2 not modelled (9, 22); 2 not observable (17, 24); 2 owner
rules stand (13, 14).

## As built

`World.withinCastReach` (`pkg/sim/spell.go`) is the one unit-target cast gate,
keyed by `w.SecondGame()` as the package keys its other per-game rules. In the
first game the caster itself is always in reach; another target is in reach
when `edgeDistance` (the footprint gap of `MAGIC-REACH-179`, already used for
Prismatic Spray) is at most the reach. In the second game the gate keeps the
whole-cell maximum axis between the two anchor cells, which is the compare
`R2-ENGINE-348` (High / Medium) states; no second-game claim states a footprint
gap, so a wide body there is measured as on main. Book admission, the release,
the automatic target pick and the scroll gate (`scrollInReach`) read it, so
all of them follow the game. A cast at a cell keeps the whole-cell maximum axis
in both games. In the first game, for one-cell bodies the two metrics agree, so
only a target or caster wider than one cell changes; the gap is measured from
the anchor cell, so it differs by side.

A range-0 cast (Shield) at another unit standing in the caster's own cell was
admitted on main and is refused in the first game now: the gap distance
between two bodies is at least 1, and only the caster itself bypasses the
gate (`MAGIC-REACH-179`). Two bodies share a cell rarely; the automatic pick
of a range-0 buff likewise leaves out a unit sharing the caster's cell.

## Proof

`TestAUnitCastReachIsTheFootprintGap`, `TestACasterAlwaysReachesItself`,
`TestAScrollCastAtAWideTargetStartsAcrossTheFootprintGap` fail on main
(`withinCastReach` replaced by the Chebyshev test reproduces main) and pass.
`TestTheSecondGameCastReachStaysTheAnchorCells` holds the second game at main's
measure: a two-wide target at anchor distance 8 (gap distance 7) is refused,
not released and not passed by the gate, and at anchor distance 7 is admitted
and released. It fails with the gate unkeyed (the state at `8ddbff07`).

## Open debt

DIV-1310 (approach), DIV-3266 to DIV-3269.
