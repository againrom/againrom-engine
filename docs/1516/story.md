# Current world-effect Tokens in SAV

## Intent and authority

SAVE writes the Tokens of imported world spell effects and their payloads from World state.
Changing the retained Document after LOAD no longer changes those Tokens. The existing graph
writer still owns record classes, references, payloads, timers, retirement and archive order.
This narrows DIV-2502. It does not establish constructors for new original effect graphs.

The owner requires current state first, then an evidence rule, loaded bytes, and an established
constructor. Missing producers remain debt and never refuse an otherwise supported SAVE.
SAV-TOKEN-034 establishes the common 37-byte Token STORE/LOAD, SAV-TOKENPOS-074 its position
object, SAV-678 its publication mask, and SAV-1010 the separate PointEffect target. These are
serialization and bounded lifecycle claims, not a complete effect-construction model.

## Field census

Locations below name the production builder. Exact line numbers and run receipts are in the
external lane census and result.

| Audit row | Field and builder | Source and verdict |
| --- | --- | --- |
| DIV-2502 | World effect and payload Token; `currentWorldEffects` | Before: loaded record. After: `SavedWorldEffects.Tokens`, with existing spell/area overlays and runtime collision repair. Movable and implemented. The graph template and historical Worlds without Tokens remain debt. |
| DIV-1492 | U50; `projectSavedCurrentOrder`, `projectSavedActorActions` | Existing World order state and supported action producers. The imported state already lives in World. Complete party AI transition/import equivalence remains debt under AI-CMD-054, AI-RETREAT-275 and the row's other bounded claims. |
| DIV-1414 | Town Human cell; `cityBaseDocument` | Existing current `Saved.Cell`, including zero, where supplied. No claim establishes the post-mission town placement producer. Keep the fallback debt. |
| DIV-1412 | Head map name; `currentCityDocument` | Existing current `LastMap`. SAV-HEAD-025 and SAV-CITY-030 establish transport and an example, not the town writer rule. Keep the rule debt. |
| DIV-1741 | Before-first-tick action words; `projectSavedActorActions` | Existing `Entity.HeldOrder` supplies phase, countdown, completion and typed body target. This is current state. Original first-tick semantics remain Unknown under HERO-FINISH-066, AI-374, AI-636, AI-640 and AI-642. |
| DIV-1814 | Projectile raw domains and spelling; `projectSavedWorldEffects` | Active leaves already follow World. SAV-1141..1147 and ANIM-101..103 do not establish all inactive-domain producers or the remaining client consumers. Keep that debt. |
| DIV-1419 | Item runtime IDs; `savedObjectRecord`, `nativeCityItem` | Existing World registry Token or `NativeRecord.Token` when bound. SAV-ID-015/MOVE-ID-016 concern actors/Buildings; SAV-776 does not establish universal Item/Effect allocation or LOAD membership. Keep that allocation debt. |
| DIV-878 | Options, panels and view; `projectCityApplication` | Existing current application projection, else retained town leaves or typed defaults. AI-QUICKSAVE-281/AI-SPELLIDENT-286 establish shortcuts; SAV-914 does not establish the other fresh-session defaults. Keep that debt. |
| DIV-3358 | U8E; `savedActorValueRecord`, `Entity.SourceNow` | Existing current own-weight accumulator plus equipment deltas. SAV-792/HERO-170/HERO-171 do not establish map-placed Human construction. Keep that constructor debt. |
| DIV-1092 | Superseded motion; `projectSavedActorMotions` | Existing bounded World motion/stride producer. SAV-GRPSAVENEXT-572 is a current-store law, not complete scheduling authority. Keep the remaining scheduling debt. |
| DIV-1729 | Six creature slot words; `savedActorValueRecord` | Already current, including all zero words. SAV-1066/AI-377 support STORE; the legacy zero-block LOAD distinction remains. |
| DIV-2503 | Actor T18; `materializeCurrentWorld`, `projectNativeScalars` | Existing native scalar where represented; otherwise 2/on-map or 0/off-map. SAV-678 describes publication, reset and clearing helpers; SAV-1093 is a bounded drawing witness. A complete actor publication lifecycle remains debt. |
| G7 item flag | Worn-item Token +08; `nativeCityItemToken`, `savedObjectRecord` | The current constructor already writes zero, not the audit's former one. Current imported flags already live in item Tokens. SAV-1114 establishes constructor zero, pickup set, merge OR and publication clear; SAV-1115 does not establish a universal worn-item normalization. No new zeroing rule is added. |
| G7 five words | Group AI +4c and four Order +90 words; typed Group/Order writer | SAV-1113 identifies raw list addresses, not object references. LOAD replaces them before reading them. The deterministic zero is load-equivalent; actual heap addresses cannot be persisted as deterministic state. DIV-4039 records that difference. |
| DIV-1719 | Uninitialised Unit bytes; `nativeCityUnitData`, `nativeCitySiegeUnit` | Constructor remainder still zero. The row supplies differing witnesses without a complete producer/consumer claim. Keep the debt. |
| DIV-2504 | Unbound actor runtime ID; `materializeCurrentWorld`, `projectCurrentRuntimeIDs` | Source-bound IDs already follow World. The unbound actor still has no insertion-time allocation carrier. SAV-ID-015/MOVE-ID-016 establish the original shared allocator, but this implementation does not reproduce that namespace for every native actor. This remains implementation debt. |
| DIV-3953 | Passive item identity/sharing; `currentTerminalWorn`, `currentTerminalCarried` | Item values already follow `PassiveBody`. SAV-DEADLOAD-130 establishes container restoration, not passive item allocation/alias consumers. Fresh per-reference records and unreadable-item limitations remain debt. |
| DIV-2501 | Departed actor with a map unit; `currentTerminalBodies` | Existing `PassiveBody` overrides placement items for placed actors too; carried stacks already use the current terminal writer. No additional missing-items route was found there. Item-free unplaced bodies and own effects remain debt. |

The allocator and complete publication/AI/graph lifecycle gaps above are not claimed complete.
No second-game original evidence is added. Shared Token transport uses the same producer; native
second-game Worlds without original Tokens keep the absent carrier.

## As built

`SavedWorldEffects.Tokens` is a bounded array of common `SavedObjectToken` values, one per unique
record in the existing world-effect graph traversal. LOAD reuses `savedItemToken`; SAVE reuses
`savedObjectRecord`. Snapshot metadata binds each array entry to its ordinary archive object.
The clone/import builders detach the array. The existing spell-continuation JSON section carries
it through World byte forms; older forms omit the array and retain their previous source.

The writer overlays these Tokens before its existing payload, spell, area, retirement and runtime
projection. Runtime IDs are retained unless an already emitted object holds the same ID. Merely
reserving a current Token's own ID does not count as an emitted collision.

The existing action supplement carries Token object bindings. An ordinary written Token normally
supplies the value on cold LOAD. A differing original/current coordinate is anchored to the final
ordinary Token, as existing persistence carriers do. An edit to that ordinary Token wins. A retired
record carries its unbound value without resurrecting the effect. The ordinary SAV graph remains
the authority for object bindings; unrelated object classes are refused as Token bindings.

The common SAV relocation table handles those bindings. No save point, format, locale writer,
actor constructor or effect executor is added. The World hash includes the new carrier, as it
includes every persisted field. A newly imported effect graph therefore has a new hash; native
play without this imported carrier has unchanged state producers. Rendering does not read Tokens.

## Proof

- `TestCurrentWorldEffectTokensIgnoreRetainedDocument`: conflicting retained T18 after LOAD;
  the former writer fails by emitting that retained value.
- `TestCurrentWorldEffectTokenWordsFollowWorld`: independent root/payload scalar and position-byte
  mutations, excluding root fields already owned by the area/spell producer.
- `TestWorldEffectTokensSurviveBinaryAndDetachedCopies`: byte form, digest and detached array.
- `TestNativeWorldEffectTokenBindingsRelocateAndRetainAbsentValues`: relocation and retirement.
- EN/RU `TestReleaseCurrentWorldEffectTokenFieldsSaveColdLoad`: PointEffect and payload words,
  ordinary menu SAVE, cold LOAD, second SAVE/LOAD, and following ticks.
- EN/RU `TestReleaseCurrentWorldEffectTokensFollowWorldSAVEAndColdLOAD`: independent area and
  payload masks/T1C, second SAVE and following ticks.
- EN/RU `TestReleaseCurrentWorldEffectTokenCorpusResave`: discovered owner corpus, field counts,
  normalization counts and retained-source SAVE comparison, excluding new Token metadata.
- Existing EN/RU `TestReleaseWorldEffectsContinue1162` and
  `TestReleaseRetiredWorldEffectsColdLoad`: expiry and retired-record continuation.

The external result records actual gate status, controls, corpus counts, failures and limitations.
Fixture success alone does not establish install or original-program acceptance. No original
program or GPU window runs in this lane. There is no new ordinary player-visible operation, so
there is no CHANGELOG entry.

## Research questions and open debt

1. Which routines create, copy, publish and retire each world-effect Token and graph edge in the
   original, including transport and shared payload records?
2. Which per-actor AI states and transitions does the original retain for a party attacker or caster
   across LOAD and the following actor/AI calls?
3. What position does a kept Human hold in the town after mission return and town LOAD?
4. Which producer writes a town SAVE's head map name, including a sub-map return?
5. What happens to a loaded attack on a body over the first actor and AI calls?
6. Which producers and consumers use inactive projectile domains and record naming/order?
7. How are Item/Effect runtime IDs allocated and restored into the original's namespaces?
8. Which values do the original's option, panel and view controllers hold before a first STORE?
9. Which calls populate a map-placed Human's own-weight accumulator?
10. How do the remaining original motion arms advance and supersede their saved Position/Mover?
11. Which actor publication events set, clear or replace each recipient bit over the actor lifecycle?
12. Which native Unit constructor bytes are read before they are assigned?
13. How does the original's shared runtime-ID allocation order interleave native actors, Buildings,
    Sacks and item/effect objects, including insertions and removals after LOAD?
14. Which original consumers observe a terminal body's item identities, sharing or definition data?
15. Which fields and owned effects remain on departed actors with and without items?

DIV-4039 is used. DIV-4040 through DIV-4044 are unused.

The storyguard comment-byte baseline falls by 147 because the world-effect writer comment now
describes the current Token source in two lines. Its test identifier count falls by nine: the two
shared effect-fixture helpers now have names without task numbers. No FrontEnd field or command
literal is added.
