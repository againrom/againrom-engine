# 1334 — recovery holds release and pickup

## Result

A standing no-pick, a group release and a pickup completion written while an
actor is in recovery or either boundary turn of an attack cycle no longer drop
the cycle. The actor keeps its victim, phase and countdown; the blow lands on
that victim; when the cycle returns to ready the pending order is read and no
second cycle loads. Manual casts and scrolls are unchanged.

## Authority

- Owner direction: build the hold only if the original does it; casts and
  scrolls stay interruptible in recovery (DIV-1015, DIV-1544); the immediate
  drop in `TestPlayerDefendCloseCoverAndAcquire1087` was an engine cadence
  choice, not an owner observation (DIV-558).
- Evidence, Medium with a stated basis: `AI-350`, `AI-ORDER-039` and
  `AI-RETREAT-272` (High) place nonzero progress before the pending order;
  `HERO-CADENCE-112` (High) keeps progress through recovery and the boundary;
  `AI-356`, `AI-351`, `AI-381`, `AI-383` give the stores that write no progress
  and no victim. The `AI-354` common tail is a live exception whose native
  reachability is Unknown.

## As built

`Entity.holdsCycleFor(kind)` in `pkg/sim/pendingorder.go`. For `PendingRelease`
and `PendingPickupComplete`, `releaseBetweenCycles` keeps the cycle in every
phase but ready (`cycleLoaded`). For every other kind it keeps the previous
rule, application ahead only. `retainsOldStrike` does not count a held
release marker or pickup completion, so a cast, a scroll or a Teleport approach
in recovery replaces the cycle and the marker as before. Row: DIV-2307. DIV-1578
and DIV-558 name the change. No persisted field is added: the marker is the existing pending order.

## Proof

`pkg/sim/recoveryorders_test.go`: release (standing and group) and pickup
completion in both recovery phases, with cold-reload continuation equality and a
loss control that removes the marker and shows the hash and the victim differ;
a cast, a scroll and a Teleport approach given after a held marker replace it; Hold replaces a held marker
and a pickup request. `TestPlayerDefendCloseCoverAndAcquire1087` now expects
the attack target cleared once the cycle completes, with the position and
`HasTarget` unchanged. The three owner-directed self-cast tests pass unchanged.

## Open debt

Native first dispatch after a release or pickup completion in recovery is
unwitnessed (DIV-2307 revisit condition).
