# Independent Group acceptance

Original LOAD retains a stage-1 dying actor in its serialized Group. Restoring
the dying tuple clears native active actions without replaying a new death
detach. Ordinary native death still detaches. Independent Group acceptance
passes on EN and RU without excluding the 16 affected source files. The
candidate incorporates the landed Player, SpellEffect and trailer acceptance.

## Comparison

`TestMilestone2Groups` discovers the corpus and reads both counted word lists,
80 AI bytes, member count and raw tags, and G1C/G40/G44 directly from Body.
The initial Document is compared before Snapshot. A separate live comparison
checks the normalized ordered members, exact Player/inline/ContainerID binding,
independent Group reference and owner, AI76, Words and Path. Last-occurrence
filtering supplies an independent detach-before-append oracle. Null and true
alias slots retain separate meanings; distinct source identities must map
injectively, even when values agree.

The structural-only `DocumentActorLocations` API adds no decoded values. Its
raw-block start lets the instrument independently read the 148-byte actor
order, active counted Patrol list and scalar state. Document order148 and live
order144 are separate comparisons. Unmaterialized members, unsupported
continuation and unavailable current-Document projections are named and counted.
A refused Snapshot or unavailable projection fails the corpus test explicitly.

`SAV-WLIST-040`, amended `SAV-STREAM-013`, `SAV-GRPLOAD-560`,
`SAV-GRPOWNER-561` and `SAV-GRPSAVENEXT-572` supply the layout and local
membership rules. `SAV-GRPAI-563` replaces AI+4c; `SAV-PATROLCURSOR-571`
replaces order+90. Their original pointer bytes remain transport retention,
not restored list identities. `SAV-UNITPROG-156` supplies the raw-block widths;
`SAV-TOKEN-034` and `SAV-SPELL-044` supply the identity-key layouts.

## Focused evidence

The EN and RU runs each discover 102 files: 62 world-half, 39 between-mission,
one named unreadable file. The world inputs contain 1041 inline Groups and
2435 distinct member actors/slots; no original member repeats or nulls appear.
Both Group lists are empty throughout this population. The separate actor
Patrol lists contain 72 words. All 62 files resume without a raw-reader or
import refusal. Initial Document fields, AI and order retention agree.

The unchanged raw/live comparator reports zero differences and zero current
projection gaps. The dying instrument independently checks all 66 stage-1
actors in 16 files with zero pool/stage/timer differences or refusals. The
Group corpus still prints 30 continuation issues, including repeated selectors
and unsupported saved state 3. Synthetic controls exercise nonzero Group lists.

The red input checkpoint `2d4afb1e0be38cc73874edeb2b3e914e4e13b284` produces
84 differences: 66 missing memberships and 18 Group-count differences in 16
named files. Those same files have unreachable current Document objects.
Its full EN output remains outside Git in
`review/story1155/red-en-groups-2d4afb1.txt`, SHA256
`e7356f4c49d28eaa2186afd8ed65193327569a996163002732f1de2fd5a03ef5`.
Green corpus and installed release logs are
`review/story1155/fixed-{en,ru}-focused.txt`.

The asset-free test families pass omission, member-order, normalization,
same-Slot container, equal-valued identity, alias and extended-count controls.
Six corrupted live-state cases retain a correct Document and equal native
round-trip hashes; the independent source comparator rejects each.

Registered `TestReleaseMilestone2Groups1155` passes on both EN and RU. Its
source is `2026-08-15/game0016.sav`, SHA256
`5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345`.
Each App original LOAD door restores 17 Groups and 32 member identities,
then a real Move command detaches a member and creates a current Group.
The witness compares projected Document fields against that current World,
uses ordinary menu SAVE, removes its temporary original source, loads into a
fresh FrontEnd and checks 20 advancing equal hashes plus another real Move.
Actual cell movement is required. This source has no dying member.

## Dying LOAD correction

Both original LOAD paths call `applyOriginalDying` after Group and Document
installation (`pkg/game/originalsave.go:945`, `:1611`). That helper calls
`ImportOriginalDyingActors` (`pkg/game/originaldying.go:16`), which invokes
`clearFelledActions` (`pkg/sim/originaldying.go:43`). At the red checkpoint,
that cleanup unconditionally called `detachSavedMember` in `pkg/sim/step.go`.
Both source admission and dying-tuple restoration now pass `detachGroup=false`;
`clearFelled` passes true for combat, commands, equipment and other native
death producers. No Slot lookup or membership reinsertion repairs the loss.

Pinned `SAV-GRPLOAD-560` appends the serialized Unit references during LOAD;
`SAV-GRPOWNER-561` says the subsequent Player suffix does not change actor+70.
`SAV-DEADLOAD-126` restores stage, health and timer independently; stage zero
only gates unrelated post-load repairs. `SAV-DEADLOAD-128` distinguishes the
archive's stored-address writer from the first dying tick's stage-1 writer.
The separate top-level dead manager in `SAV-DEADLOAD-124` is not the Player
Group list. These bounded clauses do not authorize replaying a fresh death
detach when loading an actor still present in the source Group. Complete
callback chronology remains Unknown.

`TestOriginalDying1155LoadRetainsGroupAndSavedOrder` was red on both source
entry points with unchanged AI/list/order bytes but an empty member list.
It now checks retained identities and independent owner, no defence/RNG/loot
transition, inactive stored order and native round trip plus 20 equal ticks.
The ordinary-death detach and prior dying atomicity/zero-timer tests pass.

`TestGroups1155DyingBothDoorsRetainSourceAndCurrentGraph` uses a synthetic
stage-1 Human in Group 5, Player container 2, independent Group owner 1 and
actor owner 2. Both App original LOAD doors compare the raw initial Document
separately from normalized live membership, then require a reachable current
Document. Ordinary menu SAVE, source removal, fresh native LOAD and 20 advancing
equal hashes retain Group/order state without moving the dying actor.

Final acceptance requires full Go, assets, the complete paired release and
milestone-2 gates, preserved-install verification and one adversarial review
on the reconciled candidate. DIV-1003 records the remaining native continuation
debt; no new divergence ID is consumed.

## Boundaries

Group+20 meaning remains Unknown. AI+4c is the known owned patrol path,
separate from the active actor Patrol ring. `SAV-GRPPATROL-570` establishes
the local path-copy reversal and own-cell prepend, but not its invocation
after LOAD. This story adds no dispatcher, patrol setter, original runtime
observation or world writer. Existing unsupported Group/order consumers remain
named limits. Authored escort projection needs its own current-target oracle;
the registered changed-state witness uses Move.
