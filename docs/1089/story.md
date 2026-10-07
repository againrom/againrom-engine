# 1089 — Explicit player Retreat

R and the Retreat panel cell issue an immediate selected-unit command. The
command installs persistent actor state `0x16`, separately from Ctrl+F's
automatic withdrawal thresholds. A started movement, attack or cast keeps its
progress before the away decision executes. Later admitted orders replace it.
New automatic casts wait while Retreat owns the actor; the autocast preference
is retained. Each UI press carries its complete selection under one fresh tag.

Reconciled with landed master `335672d822bb83fa0cf4e801764c2683c10d0f10`,
including Defend's sole correction `8fae845f`. Research pin
`1172d41a90ef928aa7345f76d0d3dcf0fc987bc9` matches master and is a descendant
of the original Retreat pin `5ef2cdf2ee4fb344d49113d669e84cbf8d223d95`.

## Consumable integration

R and panel Retreat clear unissued scroll targeting. Admitted Retreat cancels
an unstarted walk-to-cast reservation through the existing exact-item refund;
an already-started scroll keeps its wind-up and recovery before Retreat runs.
Refused Retreat preserves the reservation. A later admitted scroll replaces
Retreat through its existing command-group writer. Native checkpoints cover
the next Retreat action, its scroll release or refund, and the next Move.
Book admission and consumable cancellation by other orders are unchanged.
Retreat adds no native-form fields or version bump; inherited format 72 stays.

## Authority and scope

Promoted claims read through the pinned reader: `AI-RETREAT-270` through
`AI-RETREAT-275`, amended `AI-WITHDRAW-028`, `AI-WITHDRAW-026/027`,
`AI-KEY-125`, `AI-PANEL-123/158/053/060`, `AI-CMD-032/033`, `AI-STATE-011`
and `AI-ACQUIRE-002`. No raw research or original executable was read.

The slice covers input admission, queued group membership, persistent execution,
action-progress precedence, replacement and native continuation. The shared
withdrawal geometry follows the corrected whole-list centroid and distinct
positive-HP gates. Dense-list byte wrap, original corpse occupancy, exact visible
final-hit/cast timing and transitive target-death lifecycle remain Unknown.

## Sole review correction

The sole review returned autocast starvation and coalesced disjoint paused
presses. Both are corrected. Retreat suppresses new automatic admission without
clearing AutoSpell or pre-existing wind-up/recovery. Replacing Retreat enables
the retained setting again. The Retreat-only UI/game seam now passes a complete
selection per event; generic group batching is unchanged. Same-press membership
and first-member refusal still apply.

The unchanged review overlay passes all 14 leaf cases; its three SHA256 values
match the sole report. Armed Retreat reaches x=9 instead of remaining at x=18;
the separate downed/healthy R presses receive tags 1/2 and the healthy actor
reaches x=15. Focused sim/UI/game tests also cover idle, wind-up and recovery
with autocast, setting resumption, same-press refusal and native next actions.
No second review or full branch chain is run. Root owns final merge gates.

## Pre-review author proof (`5f3202e6`)

Focused sim and UI packages pass, including action-progress native roundtrips,
missing-first/later-member semantics, player scope, 253-member cap, dense-list
safe divisor, actor ID zero, own death, one loaded blow, book cast, transit,
Stone hold, command replacement and physical panel down/held/release edges.
R and Ctrl+F remain separate. Modal, town, item-drag and focus gates emit no
Retreat. Existing native fixtures pass; inherited sim format 72 is unchanged.
The final full Go run passes. The extracted destination writer is registered
as `withdrawFromAny`, with its FR-2 originator role in the architecture contract.

The 19-step production App scenario passes on EN and RU: paused repeated R,
resume, F2/F3 native save/load, character continuity and another R after restore.
Saves are confined to ignored `.local/final-en` and `.local/final-ru`.
The installed App/art/native witness passes on both roots. Healthy installed
hero 35 retreats from x=20 to x=19 away from hostile 0; thresholds stay 0/0.
Fixture placement and fog are explicit test setup. The App receives repeated R
and real panel down/release events; the 34x34 Retreat region matches the
independently named active BMP and literal source rectangle. Three native
checkpoints (before, retreating, panel reissue) restore exact bytes, enqueue
nothing and match 65-tick continuations. Existing Defend and command-panel art
release tests pass too. No original runtime or desktop-input claim is made.

EN/RU Defend and consumable scenarios also pass 22 and 25 steps respectively.
The complete 28-map paired milestone census and its drive match the baseline.
EN missionrun `-trace -ticks 1` reports M10/M20 unsupported 0/0. Script counts
remain the milestone baseline: 16/27/12 and 14/15/11 checks/instants/triggers.
The player command moves, not this census. Test binaries use `-buildvcs=false`
because sandbox VCS stamping cannot inspect the seat; source and pin SHAs are
recorded independently. No `builds/current/` file is written by this lane.

Allocator sweeps before/after ledger edits: 32 namespaces, missing answers 0.
DIV-230/291 close; DIV-201/347/348 are amended against the corrected claims;
DIV-574/575/576 retain scoped admission/progress, dense-list and lifecycle debt.
Reserved DIV-577 through DIV-581 remain unused. Preserved-install check passes
for 181 files on both roots; no-game-assets tree scan passes. The claim-ledger
check reads 291 live rows and 440 distinct IDs, with no malformed rows and
72 retraction-bearing rows reported for clause judgment. The touched
DIV-348/575 explicitly honor the amended centroid; DIV-201 retains its minimap
retraction boundary and the corrected panel ownership/command labels. Concrete
final evidence is in `verification.md`. Root owns the sole fresh Retreat review,
final merge gates, and rebuild of `builds/current/`.
