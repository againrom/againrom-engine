# Mid-strike orders and persistence

## Intent and authority

Standing acquisition without a victim, group release, sack pickup and completion,
manual actor/cell casts and scroll approach keep requested orders separate from
logical progress and the physical body. Hold Position is outside this result (story 1253).

The pinned authority is k106, `57ad4d61a8620f48ff86d69e72f6a2daf8627a16`.
AI-354 separates active operands from pending stores. AI-355 supplies the explicit
Attack control. AI-356 is High for direct pickup/completion/manual-cast stores and
progress-zero row 7/8/9 installation. Its retained-strike application/recovery
composition is Medium and conditioned on AI-354. AI-357 establishes the early
current-scroll-pointer store and conditional cleanup writers. Native input timing,
transfer callees, successful effects, common-tail reachability and inventory loss
remain Unknown. Unit-cast ordered-victim resume is owner direction, DIV-1544.

## As-built behaviour

`Entity.PendingOrder` carries release, pickup cell, completion, actor/cell cast
operands or a waiting scroll marker beside the active endpoint and phase. Known
Retreat progress/completion and imported order progress remain independent of
`AttackReady`. An imported crossing keeps its existing arrival producer.

`RowAdmitted` records row installation at logical zero. It does not require a
ready physical phase. Requested row 7/8/9 and its ordinary action/endpoint/Spell
stores can coexist with an older physical carrier. The public order-progress
reader reports the admitted row; the retained progress carrier remains separate.
An actor with logical one and a ready physical phase waits for its completion
writer. At known logical zero an older charging carrier cannot apply again.
Following-step wind-up/approach and physical fallback completion are engine
scheduling policy, DIV-1636, rather than native timing evidence.

The group release and standing no-pick writers retain an explicit release marker.
`attachAttack` rejects only an automatic attachment of that same held endpoint
while this marker stands. Accepted explicit replacement uses the existing pending
attack path. A distinct automatic endpoint writer remains accepted. This bounds
the repeated attachment defect without excluding all active writers.

Pickup input stores its requested cell canonically. Both map click and underfoot
key wait for logical row admission and arrival before the existing transfer.
The driver reads canonical pickup state after LOAD and after each step. Missing
sacks and refused transfers cancel the request. Successful transfer calls the
separate completion setter: state `0xc`, pending zero and completion word one.
Completion and the later acquisition decision are separate observations.

A request waits behind an old strike only for an active victim whose application is still ahead, a retained pending order, or independently carried logical progress. A strike already in recovery, and movement or a turn without a victim, keep the immediate replacement of DIV-1015 and DIV-1544. A manual cast stores endpoint/Spell operands while the old logical action runs.
At row admission the requested action becomes ordinary SAV action d/e, while
Actions retains the prior physical endpoint/phase/countdown. Wind-up is a later
execution step. A unit cast preserves a human participant's ordered attack endpoint
for later resume, including self and other-unit casts. Cell casts and stance-selected
victims return to acquire-in-place. Invalid requests preserve accepted pending
operands; admitted replacement updates them. Other manual replacement priority
remains DIV-1015; a later Move cancels the pending cast under DIV-1563. An explicit Retreat and a later Move replace any pending order. Defend, Patrol, Guard, structure use, potion use and equip do not clear one: a queued cast or pickup survives them and runs afterwards. Two actors ordered to pick up both keep their requests and are served in order, the first actor's before the second's. A scroll used while a strike is in recovery or boundary phase is now accepted and replaces that strike at once, where it was previously refused.

A scroll reserves its current item before row admission. Its requested Spell is
derived from that item, independently of the cached active weapon Spell. Logical
progress gates row installation; approach and wind-up execute later. Weapon-cast
cleanup can replace the current pointer. This engine refunds that displaced
reservation and retains a release marker, DIV-1635. Detached consumable graphs
remain in Actions, DIV-1369; an unavailable ordinary requested Spell key remains
zero. Unconditional scroll survival and native inventory equivalence are not claimed.

## Persistence and compatibility

Optional World form 102 appends `ORD1` over the unchanged predecessor form,
including TAC1, SBK1, weapon, effect and reservation state. Its row-stage flag and
retained logical-state carrier have deterministic old defaults. Worlds without
pending orders retain their former byte form. Decode bounds count, span, actor
order, kind, flags, unused operands and admitted-cast/completion combinations.
Malformed input leaves the receiver unchanged.

Current SAV projects pending operands, admitted row stores and separate completion
stores from current state. Actions retains independent logical and physical state,
including the existing ordered-victim resume endpoint. A native scroll reservation is detached from the archive
graph (DIV-1369): its actor SAV pointer slot holds the explicit zero and the
current supplement carries the reservation, before and after row admission. Current motion action and
requested order operands are synchronized before SAVE. The existing absent Spell
key seam also covers requested book operands without a source Spell identity.
Snapshot and independent binary peels preserve ORD1/TAC1/SBK1/weapon/effect tails;
structure compatibility helpers count the full outer suffix.

## Proof and remaining debt

`TestMidStrike*` covers first release dispatch with pending-only loss; same-endpoint
attachment and explicit/distinct endpoint controls; pickup/transfer/completion/next
decision; pending replacement/refusal; requested versus active scroll Spell and
cleanup; malformed form 102 and old defaults; nested compatibility tails; Snapshot;
and current SAV, cold LOAD and the next action. Requested-only, active-only and
clock-only controls retain separate failure meanings. Self/other-unit casts use
both pending and admitted SAVE cuts before ordered-victim resume.

Known logical one plus physical ready, and known logical zero plus physical
charging, have row-stage SAVE/cold LOAD/first-effect controls. Completion-only loss
changes the first admission. Imported Raw9=3 crossing tests retain movement until
its arrival writer, then observe row admission and first actor/cell cast separately.
Synthesized independent logical/physical states do not establish native timing or
corpus prevalence.

`TestReleaseMidStrikePickupCurrentSAVAppColdLoad` uses the installed mission-20
warrior arena and App F2/main-menu LOAD. `TestReleaseMidStrikeManualCastCurrentSAVAppColdLoad`
uses a generated installed mage, that terrain/hostile arena, installed spell rules
and a fixture Teleport grant. Both require EN and RU roots. They compare entities, Actions, sacks, purse and tick rather than the whole-world hash, because CellTails differ between the arena and cold LOAD even without a pending order (existing, unexamined). Placement, health and
non-retaliating relations are fixtures, not original-runtime observations.

The lane's first focused RED and the root's eight failing checkpoint tests remain
preserved externally. Fixture clock/scroll/regeneration repairs are distinct from
the production release-marker, canonical-pickup and logical-progress defects.
This staged checkpoint awaits root focused, ordinary and installed receipts.
Native interleavings, common-tail prerequisites, first native dispatch timing,
per-spell equivalence and displaced-scroll inventory loss remain Unknown.
