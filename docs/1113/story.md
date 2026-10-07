# Executable saved Groups

## Contract

Both original SAV LOAD doors restore a canonical saved-Group registry. Group
identity, literal selector, reference and owner are separate. Membership is the
current ordered archive-reference list, including empty Groups and unbound
members. No pointer key becomes a native identity. Native authored Groups and
older AGS retain explicitly absent saved-Group mode.

Supported dispatch consumes saved/current Group order, enable, notice base,
rate and command operands. Move uses each actor's order destination. Patrol
uses its own arbitrary ordered ring, cell-valued cursor and re-anchor latch.
Commands, handover and removal mutate current membership. Native SAVE and
fresh LOAD carry that state and its next action, not a retained SAV document.

## Authority and limits

Research pin: `dde13d1d75c192a14250dec948eeb388af4275e0`.
Authority: SAV-GRPORD-058, SAV-GRPLOAD-560, SAV-GRPOWNER-561,
SAV-GRPIDENT-562, SAV-GRPAI-563, SAV-GRPDISPATCH-568,
SAV-GRPMUTATE-569, SAV-GRPPATROL-570, SAV-PATROLCURSOR-571 and
SAV-GRPSAVENEXT-572; canonical SAV and AI format pages.

Unknown: original first-loaded chronology, arbitrary callbacks, Group+20 and
Group+40 meanings, unnamed AI fields, full default authoring and original
runtime acceptance. Store raw numeric operands without claiming their
semantics. Replaced AI/order list pointers are not native identities.
Unsupported actor forms, missing members and selector collisions must remain
explicit operation boundaries; they are not grounds for dropping membership
or rejecting every otherwise successful natural import.

## Proof targets

Independent synthetic inputs cover equal selectors on distinct identities,
empty Groups, B/A/C membership, null/discordant owners, distinct Group/actor
Move cells, and a duplicate-valued patrol ring longer than two nodes. Runtime
commands and mutations must affect ordinary SAVE/fresh LOAD/next action.
Both LOAD doors and the existing natural authored-Group controls are required.
A genuine form77 fixture must predate the form change.

## As built

`savedgroups.go`, `savedgroupsai.go` and `savedgroupcommands.go` own current
state and consumers. `originalgroups.go` is the detached admission projection;
it joins late-dead records only by exact admitted archive identity, never by
first MapUnitID. Form78 appends an explicit optional registry after form77.
The actor order is absent on records that did not serialize it, not a made-up
zero constructor. GroupAI's replaced list pointer and actor-order's replaced
list pointer are excluded; their lists are independent ordered values.

Supported actor states are0 (idle),1 (stored move),a (patrol),b (guard) andc
(standing acquire). Actor State and pending raw order+08 are independent.
Existing script Attack/Defend/Follow commands use the same current membership
and validated Script.Unit bindings; their new order carries a hashed Authored
flag and typed native target/range, separately from retained source pointer
words. Native escort dispatch, next movement, SAVE/fresh LOAD and a raw-key
coincidence negative control are tested.
Incoming actor states8/11h also execute when stage-zero order+10 has a nonzero
Token source key uniquely bound to an admitted actor. RepairStage and the typed
EscortTarget/Bound pair are distinct from retained raw words and native-authored
Entity escort state. Missing/nonzero-stage/expired bindings remain saved and
refuse only their applicable primary operation, before any member changes.
The amended SAV-PTRMAP-035/SAV-HUMRESUME-460/SAV-ACTORINPUT-547 are hit-only
order repair authority; AI-DEFEND-111/AI-FOLLOW-112/AI-FOLLOWRANGE-115 supply the
named consumer. This is not original unchecked-pointer lifetime equivalence.
Other incoming target states still need their specific callback/progress
consumers, not a new identity join. No blanket missing-key-to-null rule remains.
Group primary orders0/1/2/3/4/5/11h/ff and default use current membership;
unknown actor contexts refuse only the affected primary operation. The common
withdrawal tail starts from the then-current head. Its completeness admission
is separate from primary support. Order selection occurs once at entry.
OffMap membership remains structural: the ff callback skips a hidden actor's
decision without removing it, changing its count or stopping the current
successor walk. The sole review correction adds this missing presence gate.

Selector lookup is exact and unique in saved mode, with no ALM fallback.
Structural membership is not a living count. Failed count leaves its register
alone and blocks dependent triggers before actions/latches; unsupported named
Group actions similarly block their containing trigger. `SavedGroupIssues`
provides deterministic non-mutating diagnostics, also emitted once at LOAD.

DIV-786..792 document admission, chronology, native command authoring, Roam RNG,
remaining native movement/engagement differences, old-native absent mode and
incoming escort reference safety. DIV-793 remains unused. Existing DIV-770's provenance-only
Group boundary is superseded for supported Group operations here; its unrelated
actor constructor and presentation defaults remain outside this slice.

## Checkpoints

Base: published1112 `755476d66fb86188868f6f20e1fcc1c045dd20fa`, composed
with published1111 `0bd40bc0cb20660fcb7bc572d9ae1fa688d950fa`.
Final reconciliation and release evidence are recorded in `verification.md`.

Architecture triage: `savedMove` and the Swarm arm of `savedDecision` genuinely
originate native movement at decision time. They are not hidden LOAD repair
ticks. The saved registry bypasses native `aiGroups` partition/exclusion;
current Group order and the common withdrawal tail own later replacement.
