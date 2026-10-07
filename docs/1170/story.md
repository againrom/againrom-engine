# Current imported mission SAVE

An imported mission can continue, take damage, move through a cell boundary,
drop a carried item into a new ground Sack and save its current World as SAV.
Ordinary SAVE and both `saveconvert` directions use `ExportCurrentWorldSave`:
current producers, complete graph admission, qualified key remint, then
`sav.EncodeDocumentData`. The output is `Asg&/.sav`. Source Body bytes are never
an output operand. Unsupported current state keeps lossless AGS on ordinary
SAVE; the explicit converter names the refusal and publishes nothing.

## Current state and authority

The producer extends the existing Snapshot owners for actor values, Groups,
orders, formation, Player purses, item/container graphs, terrain, effects and
campaign progress. It creates no separate writer Group graph. Results come from
the live script registers after the existing builder/load ordering; a current
win/loss updates the local Player outcome. Historical raw/trailer fields with
no native producer remain explicit retained state.

Native movement supersedes imported motion authority. Its accepted destination
is different from its current near cell. The writer projects retained stride,
elapsed crossing, fine coordinates, route and current occupancy together.
The reader qualifies every retained native stride operand before continuing
through its boundary and remaining route. This is a native continuation policy,
not provenance or execution of an original boundary callback. Unqualified
original inputs retain their callback guard. A current native turn has no exact
SAV facing/countdown continuation and therefore keeps lossless AGS.
Source bindings select actor records and retain newly dead roots, child edges
and terminal tuples after native compaction. Source-free generated actors and
unsupported orders still require their own complete constructors.

The common SAV producer writes current `ActionClock.End` to actor+138; original
LOAD restores the literal deadline without a tick. ANIM-CLOCK-001,
SAV-UNITPROG-156 and SAV-REGENORDER-531 establish that stored dword and its signed
wrapping age against the independent session subtick. This preserves the idle
regeneration boundary. Old native World forms retain their established bootstrap
policy and frozen Document projections. Complete original deadline producers
and callback chronology remain Unknown.

Pending native book, scroll or script cast actions refuse SAV until their whole
lifecycle has a producer. The current native random stream also needs an exact
carrier: any state different from the seed used by current SAV LOAD keeps AGS.
No unrelated original field is repurposed as a random-state carrier.

SAV-PTRMAP-035 supplies qualified named reference remapping, not a general scan
for pointers. Document indices, archive indices, native object/entity IDs,
runtime IDs and script selectors stay separate. Every emitted Token, Player,
Spell and terrain receives a distinct deterministic key; missing named order
references remain unresolved, and allocated keys exclude retained raw values.
SAV-TOKENPOS-074, SAV-UNITPROG-156, SAV-GRPSAVENEXT-572 and SAV-CELLENTRY-582 supply
the existing field programs. SAV-SACKENTRY-590 preserves existing-node planes;
a new Sack cell captures the baselines before TERR-STRUCT-078 recomputation.

New bound caster-free attachments have complete Effect fields and owner edges.
SAV-EFFCHAIN-046 defines their serializer, including its amended scope.
Unused Token fields for these attachments and generated Sacks use an explicit
zero native constructor policy. A source item split retains current Token/F47
values and mints identity. ITEM-EFFSPLIT-074 callbacks and partial Spell creation
remain refused. SAV-PROJSTORE-428/SAV-PROJLOAD-429 support construction and
retirement of complete registered projectile sections, allocator and ordered
IDs. Presentation cast events are not projectile records.

The additive application checkpoint retains live selection, view, options,
panels and shortcuts, plus raw source integer domains and a native baseline.
Unchanged raw values survive UI boolean/clamped interpretations. Changed
selection joins exact actor bindings to runtime IDs; this is the disclosed
native selection policy, not a claim about an unlocated original consumer.
SAV-FOG-061/TERR-FOG-145 persist explored bit15 only. AGS retains the exact visible
sample; cold SAV derives current sight masked by exploration. A fractional pan,
zoom or native-only control without a SAV leaf retains AGS. Conversion commits
the prepared mission before capture and preserves source option domains.

## Proof

Registered release witnesses run against both lawful installs. They compare
values sampled from the live World before SAVE with wire fields, relational
references and cold native gameplay. Two equal projected Documents or hashes
are not the current-state oracle.

| Input / instrument | Observed result on EN and RU |
|---|---|
| Unchanged `2026-08-15/game0016.sav`, `TestReleaseCurrentWorldSAV1170` | After 16 driver ticks, the hero drops carried slot0 and a nonparty bound Unit takes 3 damage. At tick511, runtime1 has 12/13 crossing ticks left, runtime30 has HP17/20 and deadline323, and Sacks grew 4->5. Ordinary SAV/cold LOAD preserves all 161 full-route/current-damage samples, item fields and aliases, next order, second ordinary SAV and cold LOAD |
| Same source and test, `uninterrupted_route` | Actual MapOrder runtime1 (14,13)->(17,13), first crossing12/13. Ordinary SAV, a second SAV at successor7 and both cold continuations have 0/161 position/facing differences; arrival40/40. A genuine native AGS cold control preserves the complete World hash throughout |
| Same source and test, `current_native_turn` | Halfway native turn Facing128, Desired64, Remaining2/Total4, Drawn96 refuses SAV precisely. Ordinary AGS and a separate AGS cold control preserve the complete World for 161 samples and arrival42/42 |
| Same source and test, `pending_native_heal` | The prior hero-damage cut retains the real pending cast. Ordinary AGS preserves native caster31 healing native hero30 by 3 HP at successor7, the complete World for 161 samples and arrival37/37. No actor or action is removed to manufacture a SAV positive |
| Same source, `TestReleaseCurrentWorldDeadRandomRefusal1170` | Actual lethal damage advances native RNG to `0b98ea18ebb9dd79`. SAV export and AGS->SAV refuse the unrepresentable stream; ordinary AGS/cold LOAD preserves the current corpse, identity and complete World for 161 samples and a second SAVE. Separate synthetic terminal-removal controls retain exact identity and terminal tuples |
| Unchanged `2027-09-07/game0125.sav`, `TestReleaseCurrentWorldEffectSAV1170` | The Light root's final zero pulse creates current attachments. First SAV/cold LOAD retains them; later root and Light attachment expiry survive second SAV/cold LOAD |
| Same game0016 source, `TestReleaseCurrentWorldConversionProcesses1170` | Actual nonparty damage and hero Move, ordinary SAV -> AGS -> SAV in separate command-entry processes, with earlier files removed and original corpus unavailable. Current HP/deadline, fine position, owner, Results and explored Fog agree; all 161 route/damage samples, next order and second SAV/cold LOAD preserve the successor |
| `TestReleaseApplication1170CurrentFogSelectionOptionsAndColdSave` | Controlled application leaves on game0016 distinguish raw integer domains from UI defaults. Real controls change two selected actors, 154/6400 explored cells, panels, spell and speed; native cold LOAD, AGS -> SAV and two ordinary SAV generations preserve their represented current values |
| `TestReleaseApplication1170PointerPanKeepsFractionalViewInNativeSave` | Real pointer pan produces a fractional origin; explicit SAV export refuses and ordinary AGS/cold LOAD keeps the exact view |
| Unchanged `2026-08-15/game0018.sav`, projectile and CLI refusal witnesses | Its valid unsupported transport graph is retained. A current projectile at X19465 with two segments survives AGS/cold LOAD and expires after four ticks. Neither CLI direction emits partial output or modifies its input |

The three immutable source SHA256 values are:

- game0016: `5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345`.
- game0125: `3a055c8dcef6f053721e1c1478c82552c199b92fb178e9f034fe7500077b4efd`.
- game0018: `1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b`.

`TestCurrentWorldSAVAdmission1170`, behind `sessioncorpusaudit`, separately tries
every DISCOVERED world input unchanged. Each EN/RU run imports 62, writes 60
actual ordinary SAVs and cold-loads all 60. It names two writer refusals:
`2026-08-15/game0018.sav` and
`2026-08-30/EXP-0278-human-runtime-en/game0018.sav`, both the game0018 hash above,
for SpellTransport/PointEffect scheduling, PE44 and shared lifetimes. This is an
unchanged-input writer census; it does not claim every gameplay mutation on all
60 subjects. The only discovered nonempty projectile source has that complete
unsupported graph. Its AGS expiry proof and synthetic complete projectile
constructor controls do not establish an authentic full-SAV projectile positive.

`TestCurrentWorldSAVStrideQualification1170` separately inspects unchanged
original motion on both installs: 62 worlds, 2435 retained motions, 97 active
motions with no prior Issue, zero structurally qualified native strides. This
measures the discovered population; it does not make shape a provenance test.
Seventeen single-operand near-match controls retain refusal or the original
boundary guard. A separate northeast diagonal control preserves native fine
coordinates through the two adjacent near-cell boundary updates.

Single-cause loss controls cover stale Body/Document, new-root omission,
reference permutation, stale Position with current cells, Results omission,
action-deadline omission, Fog omission and expired-root resurrection. Valid graph mutants still encode;
the current-state oracle rejects them. Invalid bindings/selection/Fog reject
atomically. Pure remint controls retain null/repeated edges and independent
runtime IDs. Temporary constructor keys share the final allocator's raw-value
exclusion set; missing order references cannot become bound to a new Sack.
The changed-world witness compares every carried/Sack item value and order and
their injective alias relation after both cold SAV generations. Historical native
blobs remain frozen. The correction adds no World or gob fields and changes no
predecessor or current-descriptor constants.

Native-continuation instruments now select an explicitly logged AGS callback
where they require the complete native byte form and source keys. Affected
helpers are `holdingsNativeFresh`, `spell1152MenuFresh`, `producer1162Fresh`,
`worldEffects1162App`, `mover1160App`, `unit1156App`, `unit1158App`,
`trailerAcceptanceApp`, `projectile1157App`, `cellAppSaveFresh1106` and the
formation1159 App harness. Direct native release families for engagement1163,
item weight1109, actor books1105, dead actors1100, ground1076, pools1094 and
structures1098 use the same explicit callback. These remain native-codec
continuation proofs, not default SAVE-format proofs. Their full World comparisons
are unchanged. The correction also names the explicit native codec in the
two mover1134 refusal witnesses, natural-profile1107 and original-turn1147.
Story1170's ordinary SAVE witnesses remain independent.

## Remaining boundaries

DIV-1182 through DIV-1192 record the policies and remaining debt. Current native
turns, pending cast actions and advanced random state retain AGS. New native
area graph/layer construction, caster-bearing attachments, source-free new
actors, unbound item slots and unsupported action targets remain specifically
refused. MAGIC-AREAAPPLY-038 records innerEffect+0x44=caster, but the complete
Effect serializer in SAV-EFFCHAIN-046 has no explicit +0x44. No promoted consumer
establishes generic Token.Reference as the caster; reconstruction and lifetime
remain Unknown. Retained projectile direction policy and
raw source residue are not original-runtime equivalence. A never-imported
generated world is a separate dependent story. No original executable is run
and no original-runtime interoperability is claimed.

The seat owns the sole fresh-context review, serialized reconciliation, final
Go/release/milestone-2 gates and exact-main rebuild. Focused receipts and the
named writer census remain under the seat's ignored `review/` directory.
