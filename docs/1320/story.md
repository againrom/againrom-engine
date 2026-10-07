# Ordered causal hurt messages

## Intent and authority

Carry each causal health message through sim, game and Viewer in application
order. Owner result B5, DIV-2153. Public k159 is
`769071b656873de6a24a4cdaa312da54a9fdab29`, descended from k158
`885467c23f816294aa24bf1c8d11a086fde60474`.

`ANIM-095` requires signed 16-bit client-stored HP comparison with each message's
new HP, equality before the -10 floor, and the new HP store after the hook.
Its retractions place stage hooks at message handling and resolve three
intervening helpers only for native bindings. `ANIM-094` requires the shared
1500 ms stamp before source selection or admission; zero and fall do not stamp.
`ANIM-125` bounds actor strike and direct-damage senders. `ANIM-074` sends
Token8 for any nonzero computed result, including signed-resistance restoration.
`ANIM-129..133` narrow projector masks and local predicates; they do not prove
global native stage 1 reachability.

## As built

Simulation emits each physical strike before its rider, each spell application,
repeated area reference and nonzero Token8 application. The physical sender
retains alive-before/dead-above-floor admission. Positive ordinary spell damage
emits only a decrease. Existing engine observation of Drain Life victim loss,
health effects, Fire Sacrifice expenditure, debug/script writes and equipment
pool reductions uses explicit producer seams (DIV-2205). Generic health/source
publication, ordinary Heal, potion restoration, regeneration, reconstruction
and corpse decay emit no causal message.

The transient World sink is bound before session-clock scripts and cleared on
every reported-step return. It is absent from binary/hash encoding. Source
candidates clone its storage; only successful outer commits publish tentative
messages. The tick-head HP map remains at its original kill-credit boundary.

Game retains ordered messages across ticks and drawable context for a removed
target. Viewer appends both causal messages and state snapshots, drains them
once, compares AfterHP with its retained signed 16-bit HP, then stores it after the
hook. Throttle, missing source, fog and admission refusal still advance HP;
eligible wounds publish the shared stamp before source/admission. Repeated
snapshots replay no messages. Falls and owner bleed remain snapshot hooks;
no additional native projector sender is implemented. Numeral memory stays
separate. World/SAV rules, random draws, attenuation and admission policy stay
unchanged. Game version is 0.55.0; starter remains 0.3.1.

## Proof

- Simulation: literal 100→93→82 from two real physical strikes, mixed
  100→100→89 and insertion permutations; 100→60→48 physical/rider;
  100→90→80 repeated area references; source replacement refusal versus
  successful publication; health attachment, expiry, Poison initial/pulse,
  nonzero restoration 100→103, equipment commands, Drain victim,
  Sacrifice expenditure, debug kills and session script 100→70→40.
- `TestDamageMessagesKeepPhysicalApplicationOrder` compares observed/plain
  World bytes and hash. Existing zero-strike, corpse-decay and observed-step
  controls remain. `TestDamageObservationClearsEveryStepAndEarlyRefusal`
  checks nil restoration and no next-tick replay.
- Viewer: client 40/server 90/new 90 selects hard; -10 equality selects zero,
  -10→-11 stores silently; signed 16-bit narrowing; three zeros across queued
  catch-up snapshots, independent falls, missing-source/fog/throttle storage,
  and stamp/old-HP probes inside sample resolution and refused admission.
- `TestOrderedHurtAppRouteKeepsEveryZeroAcrossCatchUp` delivers all zeros from
  13 simulation ticks through App in one Viewer frame, then checks no replay.
- EN/RU `TestReleaseOrderedHurtMessagesKeepWorldAndSAV` uses installed mission 20,
  App, sim reports, game projection, Viewer, voice archive and shared audio.
  Explicit engine controls subtract 7 then 13. It compares observed/plain World
  bytes/hash and repeated-snapshot SAV bytes. Optional private output is
  `AGAINROM_ORDERED_HURT_WITNESS_DIR`, fenced outside the install.
- Controlled report drop, reversal and coalescing leave corrected physical
  World bytes/finalHP unchanged and fail ordered expectations. Viewer drop,
  reversal, coalescing, replay, zero stamping, server-before comparison and
  delayed stamp publication fail focused behavioral witnesses.

## Open debt

DIV-1489 retains Medium/Unknown global corpse-stage trigger/reachability debt.
DIV-2154 retains owner choice for native attenuation words. DIV-2205 discloses
bounded engine causal observation without asserting an original damage sender.
Native client/server disagreement on zero strikes remains Unknown. This result
does not close all B5. The sole adversarial review and merge final chain belong
to coordination; candidate evidence does not claim a landing.
