# Shared effects, speech and ambient delivery

## Intent and authority

All current effects, speech and ambient producers use one process delivery owner.
Music and movie streams retain their separate devices. The owner requires shared
sample-first admission, retained backend buffers, caller-owned retry and explicit
room, view, candidate and process cleanup.

Pinned public knowledge k158 is `885467c23f816294aa24bf1c8d11a086fde60474`.
VIDEO-SFX-085/086/087 supply the finite request tuples and service boundaries;
VIDEO-SFX-088/089/090/091 supply positional and ambient inputs and arithmetic.
ANIM-SND-022's old log10 label is partially retracted. Each claim was read
individually; its own confidence and Unknowns apply. The canonical companion is
`knowledge/formats/video/sfx-requests.md`, blob
`10afa7bd265cbd5ca616efd79b022504654c91e2`.

Private preparation remains in `review/b3-delivery/PREPARED.md` and
`review/b3-delivery-plan/PLAN.md`. Engine base is
`831c42fafda4b70f89cb7e56c6d4f28f5c1d1867`.

## Touched surfaces and behavior

`pkg/audio` owns typed semantic recipes, sample objects, 16 duplicate slots per
object and 16 shared channels. Capacity is an explicit G2 seam with those defaults.
An idle duplicate is selected before global competition. The first idle channel
wins; saturation evicts the first minimum only when its priority is strictly
lower. Refusal retains no queue. Stop preserves phase; StopReset rewinds; eviction
retains the rewound buffer; destruction closes all owned buffers once.

`pkg/ui` owns one lowest buffer factory for effects, speech and ambient, detached
receipts, phase-preserving loop movement/settings and App teardown. Positional
terms use fine world coordinates and associated view geometry. Ambient scans the
published inclusive bounds and reduces each source's weight and pan. Current
camera origin/zoom and initial bird deadline remain engine presentation policy.
Native attenuation-to-device conversion and registry attenuation preferences are
not reproduced by the portable percentage controls.

`pkg/game` joins the existing fixed-interface, chargen, school, town, dialogue,
reply, combat and spell events to those recipes. Water, river, fire and crowd use
true repeat. Horse retries its existing paint event; crowd retries entry only.
Registry sample identity uses its published registry owner and selector. Voice
identity uses the bank-source selector; named samples use their owning surface
scope and selector. Equal paths or PCM do not merge independent named owners.
Native aliases and receiver lifetimes remain Unknown.

Temporary town focus loss or menu inactivity pauses the current voice and reserves
its duplicate. The sample, handle, generation, phase and pending response sequence
survive. Active return resumes that handle through shared priority admission;
refusal preserves it for the town controller's next explicit attempt. Natural
completion, displayed-line replacement, room disposal and process shutdown still
destroy their buffers once. This portable pause policy adds no native claim.

Successful view/load commit disposes the outgoing request scope. A late failed
candidate disposes only its new scope. Shared process-owned registry/voice sample
objects survive request-scope changes; that scope destroys only its own buffers.
Shutdown closes all retained buffers once. Presentation delivery state is absent
from World and SAV. `cmd/againrom` is version 0.54.0; Starter remains 0.3.1.

## Proof

The bounded version-only production RED admits 17 alternating installed effects
and speech voices on both installs. Focused core/backend controls discriminate
sample-first refusal, mixed competition, strict priority/ties, stop/reset,
retention/destruction, stale handles, movement/settings and no automatic retry.

Installed families are `TestReleaseSharedAudioDelivery`,
`TestReleaseSharedAudioCombat` and `TestReleaseSharedAudioDeliveryScopeLifecycle`.
They use actual FrontEnd/App/shared service/backend construction on EN/RU, with
controlled lowest players and real physically muted construction. Ordinary App
routes reach menu, chargen, town, school, tavern, river/crow/fire, book direct,
Fire Ball deferred effect, Storm phase, selection/command and combat cues.
Separate production observer controls reach delayed book/weapon animation and
script/weapon messages with installed selectors and PCM. Their authored charge
16 and observation inputs are disclosed. Mission41's three ordinary mages have
charge8 equal to delay8, so that fixture cannot establish their delayed book hook.
No original trigger coverage or native audibility is claimed by those controls.
The town voice control selects installed tavern topic100/npc25, then steps the
actual dialogue controller. Mission20 contains 1341 water and 47 crow cells,
with no eligible nonnegative FireObject bird source. Registry60's installed bird
sample is exercised at the ambient recipe boundary with an explicit source;
the asset-free ambient controller test covers the otherwise unreached selector.

Frozen full current SAV bytes and World hash stay equal across delivery changes;
an independent purse change alters both. Ordinary F2 SAVE, cold ordinary LOAD,
next tick, cancelled LOAD, late draft failure and successful outgoing cleanup are
separate controls. DIV-1675 qualifies the cold Ghost capacity projection. The
private manifest names exact test sources, commands, populations and hashes under
`review/story1319-shared-sfx`; output is selected by absolute precreated
AGAINROM_SFX_DELIVERY_WITNESS_DIR outside either install.

The sole review returns a temporary-focus failure on installed npc25/topic100.
Its frozen EN/RU red and original author artifacts remain unchanged. The correction
adds `town-pause` to the existing delivery scope lifecycle family. Actual App focus
and ordinary F3 LOAD cancellation/menu return preserve full current SAV bytes,
the voice sample/duplicate, lowest handle and nonzero phase. The menu control uses
an explicit two-installed-response sequence in the school; it does not prove a
native queue trigger. Ordered lowest operations distinguish pause/resume from
replay, rewind and destruction. Natural completion, App paging, final dialogue
closure, paused room exit and double shutdown prove the separate destruction
boundaries. New receipts and the corrected candidate freeze are under
`review/story1319-shared-sfx/correction/`.

Author gates are focused/ordinary touched packages, installed EN/RU, architecture
and population ratchets, public build and vet. Existing simulation timing,
pathing and script population are unchanged; App mission-input controls cover
the touched presentation routes. The seat owns sole review, merge, once-on-merge
full suite/release/M2/final chain, promotion and cleanup.
Default vet also requires names on 84 pre-existing struct literals in 11 game
files. The mechanical key insertions preserve their field-expression mapping.

## Open debt

This joins all current engine admissions, not all 112 recovered original sites.
DIV-1943's shared allocator/delivery clause is implemented and proven; its row
remains OPEN for native trigger, alias, lifetime/cleanup and volume/device debt
referenced by DIV-2195/2196.
Missing InShop/Fight1/Fight2/Command1..3/shop-start sources and exact original
triggers, owner aliases, lifetime/retry and cleanup boundaries remain DIV-2195.
Portable native float/device conversion remains DIV-2196. Initial ambient clock
and canonical firewall lookup association remain DIV-506; preference attenuation
remains DIV-1494/2154. Cast-hook class/timing remains DIV-1654. DIV-1944 remains
independent. No new source trigger is restored from a leaf name or waveform.
