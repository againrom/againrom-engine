# One current SAV producer

## Result and boundary

One captured city or mission produces one SAV Document from current state.
Imported bytes supply unmodelled residue; their presence does not select a
writer or allow them to overrule current values. Ordinary fields remain
canonical. Native continuation carries only missing policy, presence, history,
identity coordinates and anchored width residue. AGS is never a fallback.
Malformed input, contradictory current mirrors and I/O errors stay explicit.

The story is landed and published in engine `main` at
`f5f4626cfcd0ee778d562bc434ca388ca6d594ad`. Its final correction is
`506e91035f133dade1cba9f694acf4526b222d69`, with the sole correction review
passing before the merge. Knowledge is k71,
`48828b8c949ca38687abd646be79a1511a0a3f78`, derived from research775eec59.
DIV-1373..1376 remain the story's recorded divergence range. No new ROM1 claim
or original-runtime acceptance is asserted by this landing.

Story1220 owns the common producer, city mutations, current values, graph
identity and continuation of the discovered saved-state corpus. Removing the
fabricated imported Document at natural entry and transferring current city
object identities are prerequisites of that producer and stay here. Existing
all28-mission release tests and M2 remain required. Story1221 owns additional
campaign sequences, full M8 coverage, remaining M7 proof and the original EN
sample after the common parent lands. Full M8 runs wait for common invariants;
an intermediate landing or a faster test run accepts neither M7 nor M8.

## Current implementation

- `ExportCurrentSave` constructs and encodes both record shapes. Menu SAVE,
  dialog and converter reach that producer. Final ordinary reindex, runtime
  ID allocation and key completion happen once. The architecture guard follows
  calls and callbacks; origin-based dispatch, AGS encoding and legacy version
  migration are forbidden on the producer path.
- The interactive dialog defaults to SAV and publishes one file through that
  same producer. It keeps filename parsing, cancellation and exact overwrite
  targets. Removed AGS/BOTH controls no longer lead to an independent encoder.
  Legacy AGS inputs remain readable. The call-graph guard includes preparation
  callbacks, so an upstream dialog branch cannot hide an alternate writer.
- Live actor records read current World values. Exact source bindings retain
  unowned ordinary fields. Missing representation is constructed. Actor IDs,
  allocator reservations, Player slots, Group root order and archive coordinates
  are distinct. Sparse native coordinates are anchored to ordinary fields;
  changed ordinary fields win instead of replaying old values.
- Current party members are ordinary actor/item values plus missing base,
  presence and presentation policy. New output has no whole PartyMember arm.
  Absent members use detached ordinary fragments without creating World actors.
  Current World owns holdings, books, XP, pools and potion state; member views
  are reconstructed from it. Held weapon display and derivation share the
  current item reader. Retained OriginalHuman cannot veto current training.
- Item graph version2 preserves ordered Pack roots, Worn roots, shared children,
  repeated aliases, Books and allocator state. City operations update the graph
  and its value views atomically. Doll-to-table transfer uses the normal merge
  predicate before checking free space. Worn-only quantity has a sparse carrier;
  a Pack alias remains its sole count source when present.
- Equipment receipt and topology are parameters of the existing three methods
  through `SourceEquipmentOperation`. The six receipt wrapper methods and two
  loader wrappers are removed. World has172 public methods:88 writers and84
  readers. The API pin rejects reintroducing receipt-suffixed equipment methods.
  Receipt output is published only after a successful operation.
- Ordinary Item values retain weight/definition absence by byte anchors. Native
  zero keys for Items, Effects, Spells and Sacks survive allocated transport
  keys; an ordinary key edit removes the matching absence policy. Unregistered
  ground Sacks retain their exact cell and ordered ordinary Contents. The
  independent ItemWeights and Ghost constructor inputs survive cold LOAD.
- Current Book edges are projected before the item graph's sole retirement.
  Book-only and shared Spell roots retain their exact identities. Current Sack
  creation and repeated roots preserve ordinary cells and native absence. A
  retained corpse keeps its ordinary U40 key independently of whether native
  kill credit resolved; changed ordinary keys discard the absence policy.
- Native Order insertion order, stopped death motion and cell Sack-key absence
  remain distinct from the ordinary values they accompany. Death HP writers
  update the existing source health mirror at the same mutation. Dying timers
  are read without being zeroed by a later adapter.
- Frontend attack animation clocks outside CastRuns are captured separately;
  a cast keeps one clock owner. Quick/pressed custom spell IDs use the existing
  session supplement only when no ordinary book index exists. A changed ordinary
  index wins. Malformed policy is rejected before replacing the live game.
- Current campaign main/selected mission, pending first arrival, mission offers,
  Fame history, Taken and consumed hero grants remain separate. Loaded ordinary
  offer arrays are already compacted and are not filtered twice. Stable UI
  indices are anchored to the complete ordered offer array.
- Party-role script endpoints are completed after final Party identity restore.
  Already resolved ordinary references remain authoritative. Wide current
  Effect magnitudes use an ordinary low word and a sparse anchored signed lift;
  ordinary value or owner-edge edits discard that lift.
- Native areas, deliveries and their reservations retain execution order and
  ordinary targets, clocks and payloads. A high native tick is restored before
  delivery admission. Pending Sacrifice constructs its impact from then-current
  pools; SAVE does not fabricate a prepared payload.

## Proof and limits

The current checks are focused invariants, not M8 acceptance. Receipts live in
`review/`; preserved inputs and install bytes do not enter this repository.

| Population | Evidence |
|---|---|
| Common continuation checkpoint | Focused controls cover two cold cycles and actual successors for groups, cursors, retained/removed actors and ordinary edits. Receipts: `story1220-current-group-continuation-third.log`, `story1220-save-failure-intake/current-lane-final-focused.log`, `story1220-staff-current-attachment.jsonl`, `story1220-fresh-process-final.jsonl`. These are bounded checks, not corpus or original acceptance. |
| Mixed Player presence and order | Native and constructed Player carriers, complete escort target tuples, current script Order bytes, two cold cycles, ordinary edits and atomic malformed controls pass. Receipts: `story1220-current-mixed-player-final.jsonl`, `story1220-save-failure-intake/current-incoming-escort-third.log`, `current-order-controls-fixed.log`. |
| Sparse cell planes | All24 existing cell-lifecycle cases now include two cold cycles, exact ordinary Cells/Blocks, five current planes and20 successors. Five ordinary edit cases and six malformed-policy cases pass; omitted per-cell values are anchored to the ordinary row and terrain. Receipts: `story1220-save-failure-intake/current-cell-residue-controls.log`, `current-cell-residue-edit-controls.log`. |
| Interactive SAVE and completion | Synthetic city and mission UI controls cover two SAV cycles, training/purchase, damage/ticks and cancel. Installed final/side missions150/151, ending, Fame and scheduled light pass on EN. The complete migrated dialog/campaign population is still being integrated; these are not release acceptance. Receipts: `story1220-checkpoint-02f6bbc/manual-save-campaign-third.jsonl`, `ending-party-and-marker.jsonl`, `current-item-and-ending-second.jsonl`. |
| Book roots, new Sacks and retained corpses | Book-only/shared Spell roots, actual Drop after LOAD, repeated Sack aliases, corpse U40 and unresolved-credit presence pass two cold cycles, successor ticks, ordinary edits and malformed controls. Installed EN pre-town dialog SAVE/cold continuation passes. Receipts: `story1220-save-failure-intake/current-book-roots-second.log`, `story1220-current-sack-alias.jsonl`, `story1220-corpse-reference-installed-final.jsonl`; scope: `story1220-current-sack-constructor.md`. |
| Script endpoints, effect widths and Sack mutations | Mission130 passes both heroes on EN/RU through two cold cycles, ordinary Position edits, GiveUnit and hostility. Magnitudes20/40000/-40000 pass two cycles, ticks, expiration and ordinary/malformed controls. Plain Pour, ordinary item edits, late error atomicity, retired actor/item edges and scroll reservations pass. Receipts: `story1220-save-failure-intake/current-script-endpoints-ru-second.log`, `story1220-effect-width-result.md`, `story1220-sack-pour-final.jsonl`. |
| Current API and city topology | Existing method/reader pins, atomic failed receipt controls, shared/repeated roots, full-table merge and incompatible-item controls pass. Receipt: `story1220-doll-table-final.jsonl`. |
| Native m20 and actual m10 combat | Strict World equality on m20; combat SAVE/cold cycles at ticks485,525,565 continue through605, including death and ground loot. `story1220-save-failure-intake/current-death-native-final-en.log`. EN only in this short integration run. |
| Death, orders and ground | Two cold cycles with20 successor ticks; ordinary Position/U154/U158/Sack-key edits and malformed controls. `story1220-save-failure-intake/current-death-two-cycle-values-ready.log`. |
| Actual Unequip/SAVE/Equip | Item, Effect and Spell identities and values survive two cold cycles and20 ticks. Native zero child keys remain zero; ordinary key edits win. `story1220-current-child-identity.log`. |
| Group diagnostics | Report reflects final restored Groups; genuine unresolved warnings can coexist with an exact World hash. `story1220-group-selector-audit.md` and `currentgroupreport_test.go`. Selector0 alone is not invalid. |
| Duplicate/wide actor runtime IDs | Exact actor/object selection, unique ordinary transport IDs, two cycles, ordinary ID/selection edits and atomic malformed controls pass. `story1220-current-runtime-first.log` and current runtime controls. |
| Training and custom shortcuts | Valid purchases/trainings use current Human operands. Sparse custom shortcut/pressed IDs, ordinary edit precedence and atomic malformed controls pass. `story1220-training-current-final.jsonl`, `story1220-custom-spell-current-controls-corrected.log`. |

An older sealed binary `2d3f21b60b98ae61391c8f0b882824dbb93bfd178fe0919b8463b1233ab22b5e`
passed228/228 cases: all114 saved inputs on EN/RU,3852 cuts,3624 cold cuts,
two SAV cycles, zero differences or blocks. It predates the current graph and
entry corrections. Receipt: `story1220-corpus-continuation/candidate-city-session-groups-all114/`.
It is not a verdict on the current dirty tree. Its raw frozen V3 comparison
remains FAIL; separate qualified Carry/outer-view dispositions follow below.
Earlier receipts and the previous narrative remain preserved in Git and
`review/story1220-evidence-before-scope-cut.md`.

The final Go, release and M2 chains were run on the merge. Focused SAV tests,
`pkg/sim`, `pkg/formats/sav`, the asset guard and preserved-install checks pass;
the original EN/RU SAV census is exact at 102/102. The broad Go/release chains
remain red on existing continuation and fixture failures, while the M2 gate's
only failing family is the legacy AGS corpus (90 round-tripped, 4 disclosed,
20 mismatched). This is recorded evidence, not a waiver. No new owner kit is
qualified; the rejected original runs remain evidence.

Test acceleration changes scheduling and redundant setup, not population,
cold-process boundaries, assertions or tick windows. On one sealed binary,
the six-case EN/RU subset fell37.273s to25.952s (30.4 percent); SAV bytes and
all measured outputs stayed equal. The full114 runtime has not been remeasured.
Ascending key reservation also stopped inserting keys it will never revisit.
Three synthetic65,536-key benchmark runs measure a median2.824ms to0.388ms,
with2.70MB to0.28MB allocated. Exact sequence/exclusion controls and the SAV
codec package pass; this local7.28x result is not a full-corpus timing.
The durable parallel locale helper and release manifest selection are seat
changes; `review/story1220-test-runtime.md` owns the measurements.

## Continuation dispositions

These decisions concern the ninth V7 comparison, not original-runtime
acceptance. Published SHOP-SAVE-015 establishes that shop stock is not saved.
The shop/school seat report is review/exp0392-shop-school/FINDINGS.md. Neither
that stock finding nor its generation trace describes the Offers field.

| Projection | Actual authority and disposition |
|---|---|
| Offers/shop | Town.Offers(TownShop) returns mission rows (Index, Mission, NPC), from Chapter.Shop after taken-offer filtering. Native construction writes those rows to CampaignProjection.ShopMission; campaignProgressFromSAV reads them back through campaignProgress.chapter. This is not Shop.shelves or generated merchandise. Keep the ordinary mission-array producer and the comparison; no stock-generation waiver applies. |
| Offers/tavern | Town.Offers(TownTavern) pairs Chapter.InnNPC with Chapter.Inn, retaining the zero-mission NPC sentinel and filtering taken rows. Those are ordinary InnNPC/InnMission arrays. Keep their values, pairing and order checked. Mercenaries and PermanentMercenaries are separate campaign fields, not this projection. |
| Generated shop stock and school prices | Neither is captured by Offers. Shop owns generated shelves separately; school prices are service calculations. This continuation comparison proves neither service works. Original shop/school acceptance remains open. |
| Six pre-town mission10 inputs | Corrected in the current Town model: current main mission is independent of the future offered chapter, and Offers/Take remain closed until first arrival. The city/type seal passes all six inputs on EN/RU without an observer exclusion. The prior failure remains preserved. Future chapter30 offers are not inserted into mission10. |
| Loaded offer indices and Taken | Ordinary arrays contain the remaining offers; Taken records history. Reapplying it as a compact-array index removed a second offer after LOAD. Loaded arrays now bypass that filter. Stable UI indices are anchored to the complete ordered Mission/NPC array; valid ordinary edits discard stale labels. Duplicate and zero-mission rows retain separate behavior across two cycles. |
| Roster Worn/Carried code views | Current Carry is authoritative, including empty containers. Production updates both redundant outer views. The city/type frozen comparison remains FAIL; a separate pinned qualification proves all720 full Carry pairs unchanged and all1080 reported leaves are those derived views, with loss controls. It does not qualify Weapon or original-game acceptance. |
| Ending cached weapon label | The migrated ending SAVE witness compares the complete current weapon view from PartyLoadout on both sides, plus every other PartyMember field. Its old cached label was Bronze Pike while the existing town-panel reader already returned Common Bronze Pike from the held Item. The item's code and all numeric weapon operands are unchanged. This is the same owned-item view rule tested by TestCurrentPartyWeaponUsesHeldObjectAndDefinition; no saved actor or item value is omitted. |
| Current city outer views | On seal2d3f21b6, all402 full Carry pairs and90 ordered identity vectors match. Separate strict predicates reconstruct all6280 changed outer leaves:4408 holdings views,696 OriginalHuman and1176 Weapon. The native Weapon subset uses freshly extracted Shapes/Materials doubles, all14 fields and20 negative controls. Source members supply every current operand. Raw FAIL receipts stay unchanged. This does not cover city alias topology or original acceptance. |

The two Offers branches intentionally share ChapterData and native taken-offer
filtering: both represent building mission offers. Loaded arrays are already
compacted. Merchandise generation is
a different path, so their shared implementation is not itself a modeling
defect. The first-arrival witness covers input013949; the city/type corpus
additionally covers all six pre-town inputs. Neither excludes offer comparison.

A complete recount of candidate-ninth-qualified-v7 gives18 differences per
path per pre-town input and locale:108 shop-length and108 tavern-length leaves
per locale (216 each across EN/RU). The supplied72 count was sampled; these
are repeated cuts of six inputs, not216 independent defects. No instrument,
frozen expectation or verdict was changed for this disposition.

## Remaining work

1. Admit a current native/native-retired profile with a zero mana regeneration
   period before the original-profile validation stage. The native state and
   emitted SAV are valid, but cold LOAD currently refuses. Original-current
   unsafe-divisor controls must remain intact. Receipt: `story1220-save-failure-intake/current-profile-before.log`.
2. Finish campaign witness integration: the physical-document loss control
   changes archive indices without preserving current character bindings;
   the imported-loot witness still demands old city provenance even though
   its mission World hash is exact. Preserve independent ordinary loss checks
   and current item/owner semantics. Receipts: `story1220-save-failure-intake/current-campaign-en-next.log`, `current-loot-en-diagnostic.log`.
3. Run a fresh asset-free census and the complete migrated interactive/campaign
   population. Completed Book, corpse, group, script, Effect and Sack controls
   are bounded proof; final integration remains pending.
4. Done for this story: candidate `506e910` is on its exact remote branch, its
   correction review passed, merge `f5f4626` is published, and `builds/current`
   is promoted from that exact commit. The final-chain red results above remain
   open debt rather than being waived.
5. Reconcile story1221 to `f5f4626` and finish its M7/M8 acceptance evidence,
   including the original EN sample. M9-M11 remain later work.

Original retention of the custom continuation leaf, especially through an
external original resave, remains Unknown under DIV-1369. Original shop/school
service behavior is not established by current Offers or engine continuation.
