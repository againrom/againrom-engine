# Successful pickup completion

## Contract

A successful ordinary map-click pickup or underfoot-key pickup stops the
actor's stale movement order. The following simulation step completes the
pickup into acquire state `0x0c`. Target selection remains on the existing
phase-6 AI cadence. It is reach-limited and has no guard-post leash; an attack
on the completion step is not promised.

`ITEM-PICK-016` and the unchanged arm-7 clause of `AI-PROGRESS-034` establish
transfer followed by next-tick completion. `AI-STATE-011` and
`AI-ACQUIRE-002` establish the destination state and standing picker.

## State and scope

`TakeSack` remains a pure transfer. A separate gameplay producer records only
an already-successful transfer in canonical `ActorState = 2`. This value was
previously invalid; entity offset `+100`, record width and native form version
do not change. A pre-story reader refuses this newly reachable value. Existing
valid native forms keep their meaning and historical fixtures are not rebuilt.

Canonical pending completion is not original SAV `SavedActorOrder.State = 2`.
The latter also covers approach and progress and remains unsupported; no
original state is admitted by this change.

Refused and missing sacks, quest-document routing, replacement commands,
death and off-map suspension retain their existing boundaries. SAVE/LOAD at
transfer and completion must preserve exact canonical state and continuation.

The ordinary Move approach, destination-slot support, refusal/copy semantics,
and full original progress machine remain outside this slice (DIV-337,
remaining DIV-338 and DIV-558). SAV work remains paused.

## Proof

Focused tests cover all 16 transfer phases at hostile distances 1, 2 and 8;
only distance 1 acquires, on the next phase-6 decision after completion.
Move, stop, stand, patrol, Retreat, Defend, direct attack, book and scroll
orders supersede pending completion. Death clears it; off-map suspension and
return preserve it. Actor ID zero is valid. Corrupt state, movement, attack,
group speed, patrol residue, group order and death records refuse atomically.
The committed crossing interval remains separate and ages normally.

The click and underfoot gameplay tests also cover missing/dead/off-map actors,
older queued orders and later replacement Move. Existing missing-sack,
disappeared-sack and quest-document refusal/routing controls remain green.

On both lawful installs, `TestReleasePickup1117ClickKeyAndNativeBoundaries`
opens mission 10 through App, drops one installed item, then takes it through
the ordinary map pointer or F key. Controlled hero placement and revealed
presentation fog isolate the route; this is not a naturally played campaign
or a GUI screenshot. Click transfer occurs at tick 28 and completion at 29;
F transfer at 3 and completion at 4. Ordinary menu SAVE and fresh FrontEnd LOAD
at both boundaries preserve exact World bytes/hash, followed by 33 matching
successor ticks through the production mapWorld tick.

Reconciled with school master `7f0f7939`; research pin `1172d41a` is preserved.
Final Go, paired EN/RU release tests (148 each), ten selected scenarios,
full 28-map census on both roots, asset/claim/tree guards and 181-file install
preservation checks pass. The census is unchanged, including M10/M20 = 0/0
UNSUPPORTED nodes. Exact gate revisions and one corrected test-inventory
failure are recorded in [verification.md](verification.md).
