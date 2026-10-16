# Process projectile counter

## Intent and authority

Fresh ROM1 missions in one FrontEnd instance continue the projectile allocator.
A successful mission SAV LOAD replaces it with the saved low-u16 FreeIndex,
defaulting to zero when absent. Failed or deferred preparation changes neither
the running World nor process state. A fresh candidate takes the latest running
counter when it activates.

Base: `f35bfc41baad66c295fc5e0b7ab8724c39d06a2a`. Knowledge:
`d37db9632a8e0447c2404a5b8847bb707aa8757e` (k319).
Reconciled main: `95f843860737d54523a797974d0cca7d20d14a69`.

`SAV-1191` is High for the reached mission LOAD/default-zero and SAVE arms.
Its former unqualified LOAD scope is partially retracted. `SAV-1299..1301`
establish the selected original town control, the actual town/mission LOAD
routes and the SAVE producer's InBattle guard.
`SAV-1200` and `SAV-1201` retain Medium counter-writer and process/mission-reset
completeness. The native watchpoint was not run. No original acquisition is
part of this result.

## Result

The existing mission builder takes the process counter on fresh preparation
and again at activation. A deferred candidate therefore consumes the latest
outgoing value. Mission LOAD uses the complete accepted SAV store instead.
Seeding changes only FreeIndex on a copy of the candidate's current store.

RuntimeServices owns one typed counter per FrontEnd. CampaignSession tracks
the accepted ROM1 World and captures its scalar when that driver ends, is
replaced or is dropped by NEW GAME or town LOAD. A candidate holds the port
without changing active tracking. No simulation or save field was added.

The existing speech player and acknowledgment policy form one named speech
service consumed by runtimeAudio. All device, volume and option callers keep
their explicit ownership. Composition remains 79 fields / 15 coordinators.

## Proof

Before the fix, installed EN App mission10→20 reset the counter from 30 to 0.
The synthetic RED also reset the wrapping and deferred-entry counters.
Focused GREEN verifies wrapping IDs, separate FrontEnd instances, failed
NewGame/LOAD, deferred activation, late shots and ended-World detachment.
An independently parsed raw FreeIndex checks ordinary SAV output. Mission
LOAD preserves all 32 controlled records and IDs with either a high-word
FreeIndex or an absent FreeIndex; the next insertion uses low-u16/default0.

The installed witness uses authored spellbook credit, actual book/pointer
casts, the production driver, installed objective scripts and App campaign
routes. EN direct successor20 preserves 30→30 and town→mission30 preserves
39→39; RU has the same result. A controlled ordinary SAV sets raw FreeIndex
to 65535. Actual Fire Arrow book/pointer input takes 65535 and wraps to0.
Ordinary SAVE and cold App LOAD preserve the complete current World,
Documents and purse; the next pointer cast takes0. Deferred NEW GAME takes
the old running App's late value2 and its next pointer cast takes2. These
release parents use a recording factory for effects/speech delivery.

Mod RESTART reopens the entry World while continuing the outgoing process
counter. EN/RU each observe entry0, outgoing1 and restarted1 after a live shot.
An owning entry clone with only that observed counter substituted matches
the restarted World's complete binary bytes and hash. The witness retains
the mission, tick, party, purse, records and unrelated state checks.

The two exact release parents pass on EN (3.613s) and RU (3.902s). Focused
counter and FR-5 field classification tests pass (1.741s). Strict architecture,
story and manifest guards pass; ordinary game-package tests pass (89.298s).
FR-5 classifies the driver port as derivable from existing runtime services;
the counter itself remains in the current World. Private receipts, including
failed setup attempts, belong in
`review/c4-c5-completion/1443/`.

## Lifecycle authority adoption

The published snapshot additionally supplies ANIM-175..177 and SAV-1291,
with amendments to ANIM-149/157 and SAV-1202. DIV-944 now names their local
window-getter, rectangle, destructor, geometry and ordinary reconstruction
results. No opaque callback, new World state or invented child lifetime is
implemented from them. Native activation, lower helper effects, dynamic-child
behavior and frame ordering remain bounded by those cards.

## Open debt

The native InBattle-zero town route bypasses the named mission FreeIndex
consumer; absence therefore does not invoke that consumer's default zero.
The located zero-bit SAVE branch also bypasses its counter and IDs writes.
The counter after the whole town LOAD remains Unknown because transitive and
indirect writers were not excluded. The engine accepts an ordinary city SAV
and preserves/detaches its process counter as a bounded policy. Its strict
city decoder refuses an extra Projectiles directory with either explicit
FreeIndex or IDs alone. Another leaf does not change the native route guard.
Same-tick producer ordering and alias lifetime remain separate DIV-1784
obligations. No universal writer census or complete C4/C5 closure is claimed.
