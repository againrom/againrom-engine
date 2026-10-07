# World SAV writing

## Target player result

The target is for ordinary mission SAVE to write the current mission as SAV. Explicit SAV-to-AGS
and AGS-to-SAV conversion use the same production restore and export state.
A fresh process can load the result and continue the next action without the
source file. Source-backed and generated worlds share one semantic document
API; imported bytes are not a substitute for generated constructor values.

This story is in progress. A schema serializer, an unchanged-source round trip
or a correct unsupported error does not complete the player result.

The codec now reconstructs the complete container from detached fields. One
archive state spans all five root lists, with nullable/repeated Player slots,
Group/dead aliases and nested/top-level SpellEffect aliases. Unit lists, Diary
arrays, Outpost records and sparse Spellbooks retain every serialized slot.
Fresh archive indices follow traversal, not source offsets or tags.

Typed cell records retain duplicate overlay order, all ten object keys and both
trigger forms. The terrain key, complete session regions and400-byte trailer
survive detachment. The state store includes Fog and dynamic Prj sections, not
only an empty projectile manager. Ordered/duplicate IDs and all16 projectile
fields survive. Campaign arrays, markers and unnamed scalars use the existing
complete campaign codec; label debris and source transport addresses do not.

Independent literal tests erase the source, reconstruct the container, mutate
actor/structure/session/cell/app/projectile/campaign values and re-read them.
The existing city semantic reader accepts the shared codec's city output.
Malformed references, incomplete projectile sections, excess counts, compressed
expansion and repeated-pool allocation amplification refuse before unbounded
allocation. Focused tests and `go test -trimpath -count=1 ./...` pass for this
codec checkpoint. Install-gated tests are not implied by that asset-free run.

A separately tagged read-only codec audit passes all23 top-level SAV paths in
the preserved EN root and all4 in RU. It erases each input buffer, encodes and
re-reads the complete container twice, varies the header clock, and verifies
the source file hash stayed unchanged. Run with `-tags savdocumentaudit`,
`-run '^TestDocument1115LawfulCorpusAudit$'` and an explicit
`AGAINROM_DOCUMENT_CORPUS`. These27 paths are not27 distinct digests, an App
continuation witness, a release-gate replacement or original-runtime proof.

The native envelope and both original mission LOAD paths now retain a bounded,
map-free complete document. Import-only archive origins bind native actors to
local document indices even when saved keys are zero or references alias.
Origins are discarded after binding. Old AGS has explicit absent state; partial
legacy inputs retain a named unavailable boundary. Invalid native documents,
cross-mission data and inconsistent actor/clock bindings refuse before adoption.

Snapshots project current independent clocks, latches, diplomacy, WIN/LOSE and
bound actor values from World. Actor blocks, pools, regeneration remainders,
progression, facing and supported equipment-runtime bytes use canonical
SourceNow state. Unknown imported blocks remain owned, not zero-initialized.
Retired native actor bindings remain explicit; that does not yet rewrite dead
roots or repair their graph edges. Native container size/count limits are
unchanged. The exact pre-document synthetic AGS is a frozen decode fixture.

Literal tests pass ordinary menu SAVE from both original LOAD doors, removal of
the synthetic source, a fresh-process App LOAD and the next driver tick. The
tagged read-only native audit passes22 EN world paths and4 RU world paths:
source buffer erasure, complete-document retention, deterministic AGS,
independent frontend reload, equal World hashes and two next steps. The one EN
city is reported separately, not counted as a world. Source hashes are unchanged.
This audit is not a new-process or original-ROM1 witness and does not substitute
for the paired release gate.

Checkpoint7e226298 passes the full Go suite, no-assets check and one host
paired release invocation: EN149/149 and RU149/149, zero missing subjects.
The preserved-install name/size guard remains181 files. The sandbox-only native
cutscene access failure did not reproduce in that host gate. These checks
validate the additive checkpoint, not the unfinished player result below.

Composition0e7d7840 includes master925da055 and reviewed1114effb2e8f, including
1113's single correction. Both original LOAD routes admit dead actors, Groups
and then exact document bindings. Narrow parser retention and complete-graph
retention both keep their required fields. Native form79 carries the current
Group/structure registries beside the document. Prior77/78 absent registries
stay absent; native LOAD does not reconstruct them from retained source data.

Player Defend and Retreat synchronize their native-authored actor order with
the saved-Group dispatcher. Progress ownership, first-member refusal and
OffMap membership survive native continuation. Incoming original state0x16
remains explicitly unsupported: only the native player setter grants this
Retreat dispatch. Seven tactical tests cover actual movement, close/cover,
replacement, loaded attacks, autocast, scrolls and transit.

The combined literal witness holds a complete document, incoming Group and
moved/source-only Buildings with aliased cell links. Both App LOAD doors,
ordinary structure attack, replacement Move, menu SAVE, source removal and a
fresh process retain the current World and document independently. The next20
driver ticks match. The22-EN/4-RU native corpus audit still passes. The current
form79 envelope hash changes; the earlier1115 form77 descriptor hash and all
frozen historical fixtures remain checked separately.

Final composition gates on0e7d7840 pass: full Go, no-assets, gofmt/diff checks,
all49 scenarios on each install, the28-map census and paired EN152/RU152 with
zero missing subjects. The latter uses absolute install paths and the genuine
city-v3 input at `review/story1109/seat-city-v3/en` supplied through
`AGAINROM_WEIGHT_OLD_CITY`. The earlier relative-root refusal and missing-input
skips are not aggregate passes. Preserved-install names/sizes remain181 files.
The divergence guard scans307 live rows/476 claim IDs, with81 rows citing a
partially retracted claim; no new claim or divergence is allocated. The existing
review decisions are not repeated, and full1115 review remains ahead.

Current structure projection now reaches the ordinary Snapshot path. It binds
the complete represented roster by exact source key/class, retaining repeated
root aliases and rejecting cross-object key collisions. Current health, maximum
health, geometry and attachment replace stale document values. Typed retained
Token/base/subclass fields come from the native registry. Final cell overlays
update only their represented structure/cost/static fields; earlier duplicates
and unrelated object, trigger and residue fields remain owned. Adding or
removing a cell or structure is not inferred. Absent old-native registries
leave the retained document unchanged.

Four projector tests cover all four classes, five distinct objects/six roots,
a physical100-to84 hit, changed retained suffixes, detached output and12 atomic
refusals. The composed App witness now re-reads encoded current Building health
after attack, ordinary AGS SAVE and fresh-process LOAD, while an older Snapshot
keeps its original health. This is current structure projection into the
complete document, not an original-runtime acceptance witness.

The explicit `ReindexDocumentData` operation validates a deliberately changed
graph, preserves all reachable objects and returns an old-to-new local-index
permutation. Tests independently reverse indices and change inline Group member
order, preserving nulls, aliases, cycles and scalar wire keys. Malformed and
over-budget input returns neither a partial document nor partial bindings.
Native adoption and ordinary encoding still reject noncanonical input. Group
producers atomically remap their external bindings before using this API.
Dynamic selector initialization and the remaining typed order-target producers
stay open. The supported current escort producer below uses exact bindings,
not guesses from source keys.

Checkpoint code55bc687c passes the full Go suite, focused projector/reindex/App
tests and the no-assets guard. One paired release invocation passes EN152/152
and RU152/152 with zero missing subjects. The read-only native audit passes22
EN and4 RU world paths;0152 and both1005 scenarios pass on each root. No
simulation timing, pathing or script logic changed, so the prior28-map census
is not repeated. These are branch-checkpoint gates, not a story landing or
original-ROM1 acceptance.

The next Group producer has tested field-level helpers for existing source
Groups and Unit-family order records. They retain the transport tails while
writing current AI bytes, both distinct word lists, membership slots, order
state and actor patrol. Caller-supplied local member indices and resolved keys
remain separate from native IDs and selectors. Native-authored records refuse
in these helpers until constructor and typed-target producers exist. Five
focused tests cover full-container readback, three actor classes,27 atomic
refusals and list/reference bounds. Ordinary Snapshot now calls these helpers
through the exact-binding producer below; complete world export remains open.

Exact synthetic native envelopes from15c707b2 are frozen before adding reachable
Group-binding gob metadata: prior form77 remainsa04b2664, prior72 remainsaa15a5a9
and a populated complete-document envelope isfb534cca. Decode/re-encode and
populated-document Restore/OpenMission/Snapshot preserve their state. No
previous fixture, current descriptor, envelope version or World form changes in
that preparation. The following integration adds optional binding metadata
without regenerating those historical bytes.

Preparation17724d88 passes the full Go suite, all seven focused helper/frozen
fixture tests, gofmt, no-assets and seat-tree checks. One paired release gate
passes EN152/RU152 with zero missing subjects. Production routing and simulation
logic are unchanged; no repeated scenario/census run or full story review is
claimed for this helper preparation.

Current source Groups and orders now project through ordinary Snapshot. Both
original LOAD doors bind each native Group to its exact Player/inline record,
members to local objects and resolved references to their exact targets. The
source ordinal is used only during import; selectors and owner slots are not
identity joins. Unmaterialized members use an explicit native unresolved handle,
never an output archive tag. Nil metadata remains absent in older AGS. The
outer envelope version and World form79 remain unchanged; the current gob
descriptor hash changes while all frozen predecessor hashes remain checked.

Reordering members updates Player aggregates and atomically reindexes the
document, actor bindings, member bindings and both Group references. Current
AI/word/path/order/patrol fields replace the retained values. Resolved wire keys
are checked for collisions and encounter order; an unresolved source address
becomes null. The narrow ActorGraph reader now retains positional null members
instead of dropping them through its compact actor list.

Native-authored Groups, unsupported orders, changed reference authority, new membership or
orphan-producing graph edits retain an explicit export coverage gap. Only that
marker changes on a late unsupported projection; native AGS remains saveable.
This is not constructor or lifecycle support. Malformed binding metadata still
refuses atomically at native encode, decode and Restore.

The literal changed-membership witness covers nullable/aliased Player roots,
three actor classes, current lists and orders, all binding permutations and
older Snapshot ownership. A separate process loads the reindexed AGS,
Snapshots, saves again and matches the next20 driver ticks. Both original LOAD
doors also pass Move/menu SAVE/new-frontend LOAD with an explicit coverage gap.
Reference tests independently cover resolved Player/actor targets and a missing
key. Hostile native counts refuse before large allocation; old populated AGS
loads without reconstructing absent bindings.

Final checkpoint code6daad639 passes full Go, gofmt/no-assets,35 malformed
metadata cases and both hostile-count controls. One complete paired release
invocation passes EN152/RU152 with zero missing subjects. The22-EN/4-RU native
audit and0152/both1005 scenarios per root pass with the same production code.
The first full/census attempt found an older test expecting the now-corrected
null-member loss; its explicit retained-null assertion passes in the final
chain. Preserved-install names/sizes remain181. No simulation timing/pathing or
script population changed; the prior28-map census is not repeated. These are
branch-checkpoint gates, not a full story review or original-runtime proof.

Existing source Groups now admit current native-authored patrol, guard/acquire
and Defend/Follow order fields. Patrol keeps the current ring, cell cursor and
re-anchor latch. Escort targets come from the typed Entity and exact persisted
object binding; active closing also writes the separately consumed target slot
and stop range. Missing/ambiguous/zero target keys and nonzero repair stage stay
explicit gaps. Pursuit, casting, pickup and other pointer-bearing inner orders
still require their own producers. This does not construct new Groups or supply
unknown generated actor/order defaults.

Three controlled ordinary script commands run through the production dispatcher,
Snapshot and complete SAV reader. The resulting SAV then goes through production
LOAD and20 decision ticks with its new target/ring bindings. AGS reload preserves
the same current document and20 next World hashes. Thirteen unsupported-target
and state controls publish no partial fields. Raw source target sentinels remain
unchanged in native World while output uses the bound current target key.

Checkpoint code19cbe504 passes the full Go suite, focused current-order and
atomic-gap controls, gofmt and no-assets. One paired release invocation passes
EN152/RU152 with zero missing subjects; the22-EN/4-RU native audit and0152/both1005
scenarios per root pass. Preserved-install names/sizes remain181. No simulation
timing, pathing or script population changed, so the prior28-map census is not
repeated. These are branch gates, not a full story review, direct-SAV SAVE or
original-runtime acceptance.

The research pin advances to reviewed merge2b39efa3. SAV576..579 distinguish
the explicitly empty embedded list from the selector's retained allocation
bytes, direct command stores and sequential next-SAVE reads. First genuine
SAVE values remain Unknown. DIV788 now names the missing first-rejected
old-Group cleanup as well as the native construction policy. DIV558 cites the
unaffected fresh-group/command-family clause of amended AI-CMD-033, not its
overturned every-rejected cleanup clause. This pin and prose update changes
no implementation code or frozen native envelope.

Current native command Groups now have an explicit document producer. Both
original LOAD routes bind every distinct Player root, including empty Players,
to a fresh opaque native identity. Group containment is separate from its saved
owner reference. Ordinary known-container commands remove only the first old
empty Group before detaching members; handover constructs in the unambiguous
destination without that cleanup. New Groups occupy the tail of their Player's
native block. This traversal is coherent native policy, not a claim to recovered
original global scheduling. Group IDs remain monotonic after removal and LOAD.

The producer builds generated Group records from their current canonical
fields, members and typed owner, updates Player totals and atomically remaps
all document bindings. Handover also updates the bound actor's Token owner.
Generated AI's final pointer word is zero transport spelling because LOAD
replaces it under SAV-GRPAI-563; no unknown first-SAVE default is inferred.
Remaining unsupported orders, references, ambiguous containers and orphaning
edits stay explicit gaps. An older Snapshot remains independently owned.

Native form80 carries bounded Player provenance and the Group-ID highwater.
The added gob fields preserve absent predecessor metadata. Exact b2fd10f1
empty/populated AGS are frozen with SHA256 prefixes3a3012d5/46c3b1e7; the
populated raw form79 World remainsc0681541 and its complete SAVbd016561.
Independent removal of only the new absent footer/header change retains the
old World digest17d67ec0a6f42e60 and next20 digestbacb7b5f783fa1a1. No source
Player identities are reconstructed on historical native LOAD. Its new command
still native-saves and continues while explicitly lacking export coverage.

Independent two-Player fixtures cover three ordinary-command generations, each
continued from a fresh AGS restore, plus raw ALM GiveUnit and duplicate-slot
ambiguity. SAV readback and production LOAD retain exact current Group lists,
owners and orders; native restore retains the whole decoded Snapshot and20
next World hashes. Existing nonempty Swing/Phase maps need semantic equality,
not gob's unspecified map-entry byte order. SAV document bytes remain stable.
Hostile Player/authored metadata refuses at clone/encode/decode/Restore; an
Unavailable marker cannot legalize malformed authority. A million-entry Player
count refuses before allocating that slice. Late unsupported orders publish no partial
new Group, while the current native World remains saveable.

Production d46c7850 passes the full Go suite on5069db83, with only historical
test adapters changed afterwards. The final paired gate ona0edd6a9 passes
EN152/RU152 with zero missing subjects. Earlier failing runs exposed old footer
offsets in synthetic and installed dead-state witnesses; those adapters now
peel form80 explicitly without changing frozen bytes or hashes. Gofmt/no-assets,
the22-EN/4-RU native audit,0152/both1005/1087/1089 per root and the28-map census
pass. Preserved-install names/sizes remain181. No implementation master landing,
full story review, current-build replacement or original-ROM1 launch occurred.

Native form81 retains the last accepted rated stride: exact departed and
destination cells, rate, signed axis steps and native direction octant. The
single movement-rate calculation supplies these fields. Facing, later orders,
speed and terrain changes do not rewrite a transition already accepted. The
record survives arrival; successful non-walk relocation, an unrated step and
death invalidate it. Coarse movement, transit payment and occupancy policy are
unchanged. A bounded ID-ordered tail owns this state, not the retained SAV blob.

Older native forms acquire explicit absent stride state. Their outstanding
transit remains payable; no direction or rate is reconstructed from Facing.
The exact8fbd9521 form80 synthetic active-movement envelope is frozen at
SHA2561e1cf10b, with World36b5756c and20 measured successor hashes. Independent
tail removal preserves those old gameplay bytes while subsequent new strides
acquire their own capture. Both original LOAD doors, unchanged/replaced Move,
ordinary menu AGS SAVE, fresh App LOAD and20 next ticks have focused tests.
Nine checksum-valid malformed records refuse without replacing the session.

Form81's stride capture alone does not restore original fine coordinates or
author the whole mover. SAV-UNITPROG-156 persists180 mover bytes and separate route lists;
SAV-HUMRESUME-460/SAV-ACTORINPUT-547 do not close every mover repair or first
consumer. SAV-CELLLOAD-110 restores actor cell slots from independent saved
identity keys, not Position. Full Position/mover/route/cell-plane coherence
therefore remains required before world SAV output is admitted.

Stride code95dfe635 passes the full Go suite, all26 natural world native audits,
0152 and both1005 scenarios on each root, and the28-map census. Both unassisted
drives still lose at304; that outcome is reported, not asserted as playability.
Final paired EN152/RU152 passes on93c5da98 with zero missing subjects. Its only
changes after95dfe635 are two legacy test adapters: the dead-section offset
peels the new tail, and form63/64 witnesses require absent stride provenance
while comparing every older state byte. Current-form equality remains exact.
Gofmt/no-assets and seat-tree checks pass; preserved names/sizes remain181.
No original process, full story review, master landing or current rebuild ran.

Form82 adds incoming saved-crossing state to the live simulation. Literal
Position, mover, action, both route lists, typed ground/air cell links and block
deltas survive native SAVE/LOAD independently of the retained document. A
supported progress3 actor advances its signed fine coordinates to its first
center; its physical cell, reservations and current document change together.
Presentation reads that same position, including the first frame and pause.
Replacement native orders finish the physical crossing before taking over.
Unsupported cells, callbacks and later native producers remain explicit gaps;
no generated payload, reference or original first-SAVE default is invented.

Older native forms keep absent crossing authority. The exact82d28156 form81
envelope and20 successor hashes remain frozen; current-format adapters peel
only the added absent tail. Independent literal crossing witnesses use both
original LOAD doors, menu SAVE and fresh native LOAD at every phase. Hostile
Position, action, mover, list, cell-link and block-plane pairs refuse before
session adoption. These checks do not prove a genuine original next step.

Crossing code d7d70478 passes the26 natural world native audits, with all27
source SAV hashes unchanged including the excluded city. Both roots pass the
28-map census; its unattended drives lose at304, not a campaign verdict.
Final code f6a4f69c additionally corrects only the headless selection helper:
it prefers an interior selectable pixel over an exposed moving overlap edge.
The natural party-binding release witness remains unchanged, and an independent
red/green regression preserves ordinary press/release movement and selection.
The initial paired release failure was that input race, not a passing gate.

On f6a4f69c, full Go, gofmt, no-assets, paired EN152/RU152 with zero missing
subjects and0152/both1005 scenarios on each root pass. The simulation/native
code is unchanged from the audited d7d70478. The forward research pin is
be4fdb1; the exact cache-byte table refines the already reviewed local claims,
not callback outputs. Seat-tree/divergence checks pass and preserved-install
names/sizes remain181. No original process, full story review, master landing
or current-build replacement occurred.

Form83 carries original fixed256-stride cost/static/dynamic/height planes and
the installed Terrain cost table. Both original LOAD doors read map.reg and
construct the ALM baseline before overlaying saved Blocks. Cell baseline cost
does not overwrite current cost at LOAD. Missing registry leaves explicit
absent authority; malformed parameters and unsupported terrain pairs refuse.
Old82 retains absent plane authority and its original continuation.

Admitted crossing entry creates a zero52 record from known current baselines;
ordinary recomputation uses current actor links, source-bound Building masks,
six layers and the preserved static bit4. Empty-cell detach restores baselines
and removes the current record. SAVE projects current nodes rather than archive
overlay history and rebuilds the inclusive0807..eded, dynamic>15 block set.
The same raw state supplies native routing/rate reads and survives native LOAD.
This is executable cell lifecycle, not a second export-only plane copy.

Independent synthetic checks cover full65536-cell planes, registry defaults,
byte narrowing, both border shapes, all initialized terrain pairs and malformed
inputs. A frozen synthetic form82 envelope from5de3d4ee keeps its exact
historical-form82 state and20 successor hashes after peeling only the absent83
suffix and restoring the version tag; current83 hashes are not asserted equal
to82. Twenty-four App cases cover both original LOAD doors, new-cell entry,
old-cell removal, bit4 off/on and SAVE/fresh LOAD at all six crossing phases.
Original callbacks, pre-entry bootstrap and remaining object lifecycles are not
closed by these tests. Nine checksum-valid current document disagreements
refuse through direct Restore and ordinary LOAD without changing the live
session, hash or Snapshot.

Core6b9ea879 and final codeeda8f965 pass full Go, gofmt/no-assets, paired
EN153/RU153 with zero missing subjects and0152/both1005 per root. The new
installed-content witness independently derives the full raw planes from M10,
checks both original LOAD doors and ordinary SAVE/fresh LOAD, then matches20
actual ticks with non-vacuous cell/plane changes. The22-EN/4-RU natural world
native audit passes oneda8f965; all27 source-file SHA256s remain unchanged.
The28-map census passes on unchanged6b9 simulation code; both unattended drives
still lose at304, not a campaign verdict. Finaleda8 only moves registry parsing
to the data layer and corrects two CLI old-form adapters exposed by the first
full run. The sandboxed video-helper Access denied was not counted as a release
pass; the complete paired rerun with required permissions passes without a
ROM1 process. Seat-tree/div guards pass; preserved names/sizes remain181.
No full story review, master landing or current-build replacement occurred.

Current Player Money now has an explicit live owner. Both original LOAD doors
import the full unsigned value for each distinct, unambiguous Player binding;
ordinary gold producers change World.Purse, and SAVE projects that current value
through the exact Player ID. Repeated roots do not multiply owners. Distinct
Players sharing a native slot remain explicitly uncovered, as do slots beyond
the native purse domain; source-reader admission is not widened. These records
retain their source Money without claiming current independent balance fidelity.
Old native documents with no purse metadata keep that absence.

Independent synthetic App witnesses use real compiled GiveMoney actions for two
different slots, including zero, high-bit values and unsigned wrap. They remove
the synthetic source file, use ordinary SAVE/fresh LOAD and execute the next
scheduled action. Fourteen checksum-valid hostile envelopes and a late import
failure check preserve the active session, complete Snapshot and hash. The
current-owner GiveUnit witness forces graph reindexing before ordinary
SAVE/fresh LOAD and20 further ticks without exchanging Player purses.
Two complete pre-owner form83 envelopes emitted by exact988a47ec remain frozen;
the populated original-import fixture retains its exact World/document,
Player/Group bindings and20 historical successor hashes without reconstructing
the absent purse owner. The current encode-side envelope hash changes only for
the additive gob descriptor; historical fixture bytes are not regenerated.
The installed M10 witness compares five distinct Players against their raw decoded
Money, then checks ordinary SAVE/fresh LOAD and20 continuation ticks. This is
Againrom load/save proof, not an original-process witness.

Codeba484435 passes full Go, gofmt/no-assets, paired EN154/RU154 with zero
missing subjects,0152/both1005 per root and the22-EN/4-RU natural world audits.
All27 source SAV hashes remain unchanged. The new slice changes no simulation
timing, pathing or script census population. The first package check exposed
the changed current gob descriptor and two older probes expecting a later
refusal; exact predecessor bytes remain frozen and direct/menu atomic checks
remain enforced. DIV803 stays open. No full story review, master landing,
current-build replacement or original process occurred.

Form84 binds source Items, ordered Effects, owned Spells and all admitted Sacks
to stable native handles. Archive containment determines the exact actor pack,
worn slot or Sack owner. Equal codes do not establish identity; counts, order,
child aliases, complete retained fields and container bookkeeping remain owned.
Metadata version2 records these bindings. Historical version1 keeps its prior
gold-only coverage instead of reconstructing missing ownership on native LOAD.

Actual pickup stamps every bound Item before repeated one-unit extraction.
Whole transfers retain identity; splits create distinct Items and children;
merges retire only the incoming object. Carried transfers, equipment changes,
drop, consumption, scroll reservation, scripts and terminal loot use checked
World transactions. Weapon changes construct known Spell operands from the
installed rule and retain unknown runtime identity explicitly. Shared-child
destruction refuses before publishing its item mutation. Generated Sacks retain
their constructor debt. Unbound native slots stay ID0; their presence discloses
partial container coverage and never mints a source identity by value equality.

Current Item/Effect/Spell records and actor/Sack edges project before checked
graph retirement. Removed actors lose their obsolete ownership edges without
inventing complete original dead-root lifecycle. Active native scrolls and
counts beyond the original word retain full native state but explicitly lack
complete original projection; cancellation or later representable state can
rejoin the same native identity. A partial document is not export permission.
Other unmatched external sessions remain unsupported. DIV804 records these
boundaries and the remaining original lifecycle and constructor requirements.

The explicit completed-world CarryRoster boundary copies all item-value fields
but clears local object handles. The town and newly authored mission have no
copy of the completed World's registry. Same-world CarryRosterIDs, the live
registry and native mission SAVE/LOAD retain their handles. Full cross-world
Token/F45..F48 and child-identity ownership remains unimplemented; this value
handoff is not complete original-object continuity. The ordinary town/shop to
next-mission route must still support equipment commands and native SAVE/LOAD.

Focused witnesses cover aliases, hostile ownership pairs, mixed ID0 ingress,
split/merge, source Weapon Spell replacement, scroll continuation and death
with new/existing loot Sacks or destroyed holdings. The natural M31 witness
starts with a twelve-row nonempty pack, counts10/9/12 and six worn Items. It
picks up two Items, transfers three units and equips the picked Armor, with
ordinary menu SAVE/fresh LOAD at four cuts and40 next driver ticks. EN/RU are
two consumers of this one source, not separate original recordings.

Native envelopes additionally retain the last sampled visible plane beside
exploration. LOAD replaces constructor exploration, avoiding newly revealed
cells between samples. Old AGS retains exact recorded exploration and explicit
absent visibility; its fallback is restricted to that exploration. Original
SAV import still follows its separate exploration-only law. DIV805 records the
native extension, not an invented original field. A one-tick checkpoint crosses
two later visibility samples with exact World and Fog continuation. Three
genuine a5083c6a predecessor envelopes keep their bytes, every older field and
historical successor hashes. The current encode hash changes only for additive
gob descriptors. Older frozen fixtures are not regenerated.

The complete combined checkpoint requires full Go, no-assets, paired EN/RU
release tests including the genuine city-v3 predecessor, relevant mission
scenarios and the read-only natural corpus audit. Exact candidate and gate
results belong to the seat journal. Fresh ALM worlds have no owned registry;
unbound partial transfer retains its bulk path. No full story review, master
landing, current rebuild or original process is implied by this checkpoint.

Ordinary mission SAVE still writes AGS. Full current graph projection,
remaining item/Spell/effect lifecycle, position/order/death lifecycle, generated constructors,
reference-key reminting and both world conversions are unfinished. Group and
structure gameplay state now composes; represented structures and source-backed
or current native-generated Groups project back into the document. Remaining
typed order targets, complete mover/cell lifecycle and full lifecycle stay unfinished.
The sole fresh story review and final
player/original acceptance remain ahead. This is a checkpoint, not completion
of the player result.

Reconciliation codeaa1625ad merges current master42badc46 (moving off
superseded925da055) and carries the research pin forward to682184eb. Master's
QuickSpells feature exposed a pre-existing story fixture defect:
completeDocumentTail1115's SpellBook/Shortcuts literal used index0 as an
unbound sentinel instead of the documented-1, so validateQuickSpells correctly
read four colliding real bindings across every1115 test sharing that fixture;
the fixture, not the new validator, was corrected. D-1 adds a nil-receiver
guard to SavedObjects' item, effect, spell, sack and container lookup
accessors, closing a latent panic reachable from a registry-less World
(mapload.SourceTownEquipment always builds one) through EquipSourceItem; a
focused test reproduces the panic without the guard and the clean refusal
with it. Full Go, gofmt, no-assets and claim-citations pass on the merge
commit. One paired release invocation passes EN167/RU167 with zero missing
subjects, raised from master's162 by this story's own gated tests; all50
scenarios pass on EN; preserved-install names/sizes remain181. The28-map
script-gap census (`pipeline/check-milestone.sh`) is unchanged from baseline
on both roots -- this reconciliation adds no script-node support.
check-div-claims.sh could not run in this environment: EMPTY SELECTION,
reproduced identically against an untouched master checkout at42badc46, a
pre-existing seat-tooling defect unrelated to this branch. No full story
review or master landing occurred.

## Required state

The document owns Players, Groups, living and dead actors, Items/Effects/Spells,
Buildings, incoming SpellEffects, terrain cells and planes, session values,
Sacks, application state, Fog/Projectiles and campaign state. Stable local
identities and explicit edges replace source offsets and archive indices.
Serialization derives fresh archive tags and coherent identity keys.

The live projection consumes the actor, Group and structure registries from
1111,1113 and1114. It does not duplicate them in an export-only graph. A saved
object that was ignored by gameplay cannot be replayed as current state.
Unknown constructor values require evidence; neither zeros nor corpus values
are generated defaults. Loss detection remains visible until each state owner
has a current-state producer.

## Proof

- An independent literal graph checks schema fields, identity, aliasing,
  sparse roots, count bounds and malformed-reference refusal.
- A changed source-backed mission performs an ordinary action, saves SAV,
  loses access to the input, then loads and performs a next action in a fresh
  process. Both conversion directions use that current state.
- A generated mission takes the same routes without importing a SAV first.
- EN and RU exercise ordinary SAVE, explicit conversion and fresh continuation.
  Exact production output must also load in ROM1, survive an action and distinct
  resave, and reload in both games. A verified original write boundary is a
  prerequisite, not implied by a clean-room round trip.

## Integration

The root owns `wt-story-1115-world-sav-writer`. Published1113/1114 and current
master42badc46 are composed (reconciliation codeaa1625ad, moving off
superseded925da055); later changes reconcile any new master before final
gates and the sole fresh review. DIV802 records the remaining mover/cell
lifecycle boundary; DIV803 records independent Player purse ownership gaps;
DIV804 records current item/Sack ownership versus complete original lifecycle;
DIV805 records native sampled visibility versus original exploration-only Fog;
806..813 remain reserved.
Implementation master and the denied1107 correction remain untouched.

The five-path acceptance matrix in the seat's `pipeline/SAV-COMPLETION.md`
remains authoritative. Complete world state and original interoperability are
not reduced to the first supported witness.
