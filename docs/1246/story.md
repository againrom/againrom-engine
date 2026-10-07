# Town figure input and entry

## Intent and authority

Room popups follow TOWN-480 event consumption and construction lifetime.
Occupied tavern cells select on down and double presses select before acting
under TAVERN-FIGURE-021. TOWN-479 bounds merchant held-item and school overlap
routes. TOWN-481/482 bound entry paint, timer admission and guard endpoints.
Authority is public k105 `2f7f006270dde633634e0ed043aae272b3346c92`, including
amendments. Reconciled released base is
`cb94be7ced6b16cc43951cc532c8c5bee1d92205`.

## As built

The room popup has event-specific input. Body and list down pass below;
checkbox down persists the global option. List up consumes, checkbox and body
up pass below. Close up needs its own registered down and an inside release.
Screen, room, popup revision and content replacement cancel that ownership.
Dismissal ends with the constructed popup. Closing a conversation re-reads
existing content without clearing dismissal or renewing its revision. An
absent popup stays absent until actual room construction. Room entry, square return, menu
resume and LOAD reconstruct it when TipsOff allows it. Chargen keeps its
existing popup input contract. Authored popup geometry remains DIV-162.

Tavern cell down selects a stable candidate; release is inert. Two downs on
the same key and cell within 350 ms, with an intervening inside release, form
one double action. Misses, outside releases, focus/screen departure and popup
reconstruction break pairing. The next down after a double starts another
pair. Selection precedes Talk, hire or dismiss. DIV-1624 states this physical
pairing policy. Merchant release cancels a held App drag without model
mutation; an unheld merchant remains inert. School mask down selects only
the displayed class. Trainer/diamond and button-panel misses remain inert;
enabled buttons act on release. The enabled native merchant prompt has no
identified engine counterpart; its live geometry remains DIV-1557 debt.

App composes a square entry synchronously before another pointer dispatch.
Focus, cutscene, dialogue and SetTownPaintAdmission govern this entry. Nil
client admission allows paint; blocked ordinary square paint also holds its
animation. First admitted paint initializes the timer without a hub. A later
paint admits one hub only after more than 67 ms. Guard step zero remains
still; frames 0/1 are crossed and a closing sweep ends at 0. No synthetic
pointer or endpoint sound is added. The cached FrontEnd townScreen retains
the timestamp across room/menu return, new-game reset and actual same-process
SAV LOAD. Detached SAVE/import drafts do not replace the App screen. A new
FrontEnd has a fresh timer; multiple FrontEnds in one OS process are an engine
construction seam (DIV-1623). Presentation fields enter neither SAV nor the
simulation hash. Ordinary animation under a gate dialogue is separate debt.

## Proof

Transferred behavioral REDs fail on the original base `ab55f096`: checkbox down,
unarmed Close, occupied tender/cauldron down, and synchronous LOAD/return
paint. Empty occupancy, first-paint, strict 67/68 ms, inactive and missing-art
controls discriminate the reached routes. Focused App regressions cover
popup phases and cancellation, stable-key timing, hire/dismiss purse and
party atomicity, unaffordable hire, selected Talk, both school class masks,
figure/button overlap, held merchant cancellation and dialogue admission.
Existing overlap witnesses now test list/body phases independently. Existing
presentation/SAV and simulation-hash controls pass. Ordinary checks caught
redundant neutral surface reads and an old school release assertion. The
dispatcher now reuses its input snapshot; the sound witness keeps one down
request and outside-release retention. The measured comment-byte baseline
falls to 8272143 after reconciliation and release setup alignment. Focused and all four touched packages
passed on the corrected parent.

Registered TestReleaseTownPointerPopupEntryAndColdLoad uses installed EN/RU
art and real FrontEnd/App room controls. It covers shown and TipsOff-absent
popups, checkbox persistence, owned Close, three room Exit/Escape returns,
menu resume, F2 SAV, same-process F3 LOAD and a fresh loader. Cold LOAD reaches
one nonselected occupied cell before release. A supplied zero roster over
installed art is a separate loss control. The synthetic 18-cell
entry/figure fixture is secondary evidence, not native SAV acceptance.

Owner receipts are under review/story1246-town-entry. red.txt preserves the
behavioral RED. focused-fixture-refund-error.txt has an incorrect dismiss
refund expectation. focused-corrected.txt also fails Talk because its
synthetic archive lacked npc22; focused-repaired-controls.txt proves that
reached App route after supplying the payload. Save-dialog and incomplete
interface fixture failures are preserved and classified in proof.md.
The sole adversarial pass returned popup resurrection on conversation close.
TestReleaseTownPopupDismissalSurvivesDialogue reproduces it on the candidate
in installed EN/RU tavern and gate App routes. Both corrected routes keep
dismissal through Enter-close and recreate on actual leave/re-entry. TipsOff
controls and the existing SAV/cold LOAD witness pass. A separate four-room
content test preserves construction, option-gating and content replacement.

Reconciliation preserves immediate shop clicks, stored tavern-record Talk
bodies and the missing-record guard. Both town release witnesses remain in
the union of the gated population. The field and naming ratchets retain
their measured populations. Changed town and overlap focused tests pass on
the reconciled tree. The population and naming guards also pass. Receipts are under
review/story1246-town-entry/reconciled-cb94be7c.

The lane owns changed focused checks. Root owns ordinary precommit checks,
full/final gates and release; a pushed candidate does not establish landing
or SAV completion.

Shared release setup clicks the centre of Close outside its list overlap.
The tavern interior pixel witness physically dismisses the popup recreated
on room re-entry. Its clock, crop, independent art, sounds and SAV comparison
remain intact. The failed merge chain is preserved at
review/verify-story1246-687b36a9-full; it changes no production contract.

## Open debt

Native capture, enabled merchant prompt fields, registry configuration,
physical delivery, presented pixels, audible latency and cadence remain
Unknown. Conditional campaign child precedence on Close is DIV-1625.
DIV-1623/1624 disclose entry admission and double pairing policies.
DIV-1626 is unused. No native run, desktop input or original write occurred.
