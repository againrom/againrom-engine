# Independent Unit mover and route acceptance

Original LOAD movement retention now has a raw-Body acceptance instrument.
Both lawful roots compare 2,837 tagged actors and 2,435 live motion/order
carriers without deriving expected movement values or populations from the
decoder under test. No movement rule or current-state producer changes.

The branch began at engine 6cbe75c, checkpoint 80f7a088, and reconciled serialized
main c6183c9 through merge 62bd200d. Its knowledge pin is k6,
84f328362e6c946303da3d15dddfcefbcfec4946.

## Contract and authority

SAV-TOKENPOS-074 supplies Position12. SAV-UNITPROG-156 supplies the six raw
writes, including Mover180 and Order148, and the separate order path.
SAV-WLIST-040 supplies short/extended counts and two-byte elements. SAV-630
distinguishes static and dynamic packed-cell lists; SAV-631 and SAV-633 bound
their known consumers and the order-path evidence. SAV-GRPPATROL-570 keeps
the actor ring separate from the Group path.

SAV-STREAM-013, SAV-TOKEN-034 and SAV-PTRMAP-035 distinguish archive identity,
Token bytes and pointer keys. Their amended/retracted clauses were read.
SAV-UNITFLD-049 and SAV-DEADLOAD-126/128 provide independently stored HP/stage
diagnostics; SAV-ACTORBIND-544 keeps exact Humanoid distinct. All claims were
read through this worktree's public knowledge reader. No private research,
original executable, install write, current motion producer or world SAV
writer was used.

## As built

`DocumentActorLocation.RoutesOff` exposes only the first embedded count's
structural start. The test reader owns the 24+22+24+64,180,148 arithmetic,
each list count and element read, and exact endpoint checks. The third list
begins after Order148. Counts are bounded by the remaining source span before
allocation; extended zero,65535,65536 and truncated forms have controls.

The reader reuses 1158's raw tagged-object/actor population and bijective
archive-to-DTO framing proof, including independently read HP/stage. Its
combat values supply no movement expectation. Decompression, structural
starts and the identity-only permutation are the only production inputs.
Record.Raw, Document values, ActorGraph and import manifests supply no
expected values or counts.

Every Unit/Human/Humanoid record in the complete Document is checked,
including city records and records without live carriers. Resumed world
Documents are compared before Snapshot. Each raw living or supported stage 1
actor requires an entity, motion, order and exact DTO binding, even when
another carrier survives. Position12, Mover180, two motion lists, Order144,
the separate order path and the existing entity cell/facing/byte10 carrier
compare independently. Order's final four bytes remain Document-only pointer
residue. Unsupported continuation and late-dead exclusions are named.

`TestMilestone2MoverRoutes` joins the discovered-corpus gate. Its summaries
and exclusions are selected by the script's output filter.
`TestReleaseMilestone2MoverRoutes1160` is one new registered release witness.
The checked gated-test population is 231; the milestone family grows 14 to 15.

## Proof

Focused logs are outside Git under `review/story1160/` in the seat.
`focused-en.log` and `focused-ru.log` each report:

| measurement | EN and RU |
|---|---:|
| discovered files | 102 |
| world / city / unreadable | 62 /39 /1 |
| raw tagged actors, world /city | 2,837, 2,746 /91 |
| Unit /Human /exact Humanoid | 1,792 /1,045 /0 |
| live motion/order carriers | 2,435 |
| current /active /already handed to native route | 2,333 /97 /102 |
| supported dying carriers | 66 |
| world raw-only records | 311 |
| named unsupported continuation issues | 104 |
| static /dynamic /order nonempty lists | 417 /162 /36 |
| static /dynamic /order elements | 1,293 /508 /72 |
| mismatches /resume refusals | 0 /0 |

The registered App witness uses preserved `2026-08-02/game0009.sav`, hash
60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6.
It checks 57 raw actors after title LOAD and map-menu LOAD, before Snapshot.
One advancing body update changes six supported carriers. Ordinary menu
SAVE, removal only of the private original copy and a fresh FrontEnd native
LOAD preserve World and the complete selected movement Document. Each of 20
further steps advances exactly once and compares World hash, current motion
and orders against their Document, and the paired retained Documents. A following actual
App pointer Move installs target 114,128 on both sessions with equal World
hashes and another independently checked Snapshot. The synthetic App witness
also carries three nonempty separate lists.

The six `TestMover1160*` tests pass in `controls-final.log`. Literal controls
exercise every Position/Mover/Order byte, all three lists and their counts,
Unit/Human/Humanoid classes, nulls/aliases, equal-valued objects, zero/duplicate
Token identities, malformed structural starts, missing living/dying carriers
with a survivor, live list loss, and native World loss. A modified Order tail
survives a valid native encode/decode with identical World bytes but fails
the separate Document check. That checkpoint's current-World comparison covered
motion only; the sole review exposed its missing current-order comparison.

An executed decoder control detached `Record.Raw["U158"]` from Body and flipped
byte 147 after raw decoding. `decoder-order147-mutation-en.log` records 5,583
new instrument mismatches; `decoder-restored-en.log` records 0 after removal.
The legacy `TestMoverRouteCorpusAudit1134` has 16 population-count failures on
the unmutated corpus and the same 16 failed files and exact errors under this
mutation. `mutation-comparison.json` records that comparison. This is not a
claim that the legacy audit was green. Its living-only population differs
from the current live carriers by 66; it remains an opt-in legacy audit.

## Sole correction

Review F-1 is corrected at every witnessed Snapshot, including the next actual
App Move. The oracle reads current `SavedGroups()` orders and independently
compares all144 bytes and the separate `U158_90` list through exact actor/DTO
bindings. Static `U15C` and dynamic `U178` lists have separate current-World
assertions. Equal corruption in both Documents cannot establish correctness.

Supported authored Move, Patrol and Acquire preserve current raw bytes. Follow
and Defend explicitly transform their target and range operands from typed
World state; the target's nonzero retained World identity supplies the expected
key. AI-FOLLOWSET-116, AI-DEFEND-111, AI-FOLLOW-112 and the amended
SAV-HUMRESUME-460 bound these cases. Active attack/other inner orders, skipped
repair stages, unsupported authored states and targets without retained World
keys fail with named boundaries. No Document value or production projector
supplies the expected order bytes. Order's final four bytes remain under the
separate retained-Document check.

Five new permanent control tests cover 144 retained order-byte mutations,
144 mutations for each of Patrol/Defend/Follow, five losses for each of the
three distinct paths, four transport-tail bytes, missing/aliased bindings,
absent registry and named unsupported typed arms. The review's two production
overlays now fail the existing App continuation: byte143 reports `80 != 00`
for two carriers; dropping `U158_90` reports `[] != [5140 6682 5140]`. Both
overlays passed that App scenario at the paused checkpoint. No production
change was needed for the correction.

Focused real App continuation passes EN and RU with the six changed carriers,
20 advancement pairs and next Move. Logs and executable controls are in the
seat's untracked `review/story1160-correction/`; see [verification.md](verification.md).
The root owns the final paired release/milestone2 chain, serialized landing
and build. The candidate's Go/gofmt/assets evidence is returned by exact SHA;
the old checkpoint's final chain is not evidence for this correction.

## Open debt

DIV-1092 remains explicit. After Current becomes false, equal Worlds can
retain different historic Position/Mover/embedded-route bytes. The real App
probe names DTO 2,29,47,89,91 on both roots. Initial full Document equality,
all orders and order paths, all current motion bytes and independently current
facing/rotation bytes remain strict comparisons. Historic blocks are never
forced current to conceal this boundary. Current motion, each of its two lists,
current order bytes and the third order list are compared directly with their
projected Document, so equal paired corruption cannot pass.

Initial route handoff's 102 records remain byte-compared retained carriers;
they are not counted as current runtime motion. The 104 unsupported
continuations, 311 raw-only records, city App continuation, exact Humanoid
runtime acceptance and unclaimed field consumers remain outside a fidelity
claim. No new mismatch requiring a divergence was found. Reserved
DIV-1098..1105 remain unused; no allocation floor or ledger changed.

The separate city-to-mission retained Human turn-rate issue reported by the
seat is outside this original-LOAD acceptance slice and was not edited.
