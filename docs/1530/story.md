# Native NPC setup in the map editor

## Scope

Base: `a88fedd9e9417e2f437ab2ee10dfc4d17e78654b`, merged fast-forward from lane entry `56eeabc86557ee87325e6f5acdae26dfe63196ae`.

The owner chose native storage: type-6 placement fields, owned type-8 stock, and linked type-9 recipes. A named person uses the NPC arm and an existing registry record. No custom name, mod character binding, own ALM section, per-item drop policy, or new ordinary-human map-sprite policy is authorized. Clone retains the existing independent stock/recipe copying route.

DIV-4207, DIV-4209 and DIV-4210 remain unused. DIV-4206 describes the interface. DIV-4208 records stock equipment interpretation and its bounded Unknowns.

## Slices

1. Face and sex with a draft doll preview. Unify doll composition first. Ordinary Human type-key and definition-ID arms offer the native fields. NPC and Units arms keep their secondary-key meanings. A face edit preserves unrelated bits. Changing sex from an inherited face materializes that face: the native placement cannot independently inherit a face and override its sex.
2. Class, Humans definition and NPC registry record, without a custom placement name.
3. Worn-slot rows and nonzero stock dispatch, after checking what each native arm can express.
4. Carried items, recipes and derived read-only corpse loot.
5. The complete window through the existing popup and field builders.

Stop before a slice needs a storage or gameplay decision native records cannot express.

## Correction

The correction starts at reviewed candidate `6aa96ac9d8be24f67cbb38206ea2ec0eb181db64`. Merge `b1b69dec1c0c773362904b3b3729cef880c22bea` brings main `1b222aab53e8c81a51249899e6d6f8985b49d999`, including the F5 package route. Unreleased retains both audience headings and every incoming entry. Storyguard instructed CommentBytes 9877299 -> 9871377 on the merged tree; its repeated test passed.

`writeStockItem` stores a weapon's kind 41 and a book's kind 42 in the native recipe spell/power head. A preceding scalar can use the recipe's first effect head without reordering effects. Inherent base effects are not duplicated. The existing loot decoder is exposed as `ItemInstanceFromLoot` and reads the written element back before accepting it. `ItemRecipeEqual` compares code (including material and tier), kind, price and every ordered effect. Unrepresentable modes, spell words or effect orders fail atomically. The draft shows the localized refusal before Apply. Materialized worn items that match their suppressed row retain that row's complete constructor form through `equip`. Initial rearm receives the placement as data and uses `blockFor`, `withTailWords` and `NewActor`, preserving the row's speed, sight, training, XP, pools and placement overrides. The initialization route is refused after tick zero and for imported actors. Potion pool deltas survive a later equipment element.

Zero stock elements retain the decoder's base representation. Only nonzero elements enter `SourceConstructedItem` and equipment dispatch. ITEM-OWNED-028's carrier clause imposes no source-form conversion. The no-stock clone policy is data on `UnitStock`; duplicate nonzero unit IDs without owned stock clone as on the base. New stock still requires an unambiguous owner.

Fixtures cover a material/tier spell weapon with a nonzero price, a readable book, refusal in the draft and Apply, zero-element representation/hash equality, and exact Undo of a stockless duplicate-ID clone. `TestReleaseEditorEmptyMaterializationPreservesHumanItems` is registered without a root column and must run on EN and RU. It covers every ordinary inherited-gear Human on every listed map, retained-slot Inherit after a non-weapon Empty, same-slot restoration, other units, ordered item values, exact Undo and NPC Empty refusal. The lane invokes its shared body only through the separately authorized temporary overlay; the registered release test remains NOT RUN.

Final install proof: EN 38 maps / 1407 ordinary Humans / 280 spell weapons / 13 NPC Empty refusals; RU 34 / 610 / 96 / 13. All three transitions, every actor field, complete item state and native byte Undo pass. The shared body uses the existing SetField batch context and independently rebuilds each world; the full game-host preview and Undo are covered by fixtures. Against review base 9d40c209, all 216 start worlds were compared: EN 108 equal / 6 changed; RU 96 equal / 6 changed. Only scenario/110.alm and scenario/111.alm differ, at three difficulties per root. The fixture also preserves healing by a potion before a later worn element. CommentBytes falls from 9871377 to the measured 9871151.

Correction gates pass: 38 focused game/UI tests in 33 s; the nine named packages and guards in 37 s; full game package in 109.019 s; build in 22 s; vet in 30 s; fast gate in 40.2 s; DIV claim check in 1 s with 352 inherited partial-retraction rows. The final install overlay takes 214 s including compilation. Registered release tests remain NOT RUN. The correction commit and full receipts are in the lane RESULT.

## Claims

`MISSION-ARM-006` (High) establishes class-first resolution and NPC precedence over definition ID. `ALM-FLAGPATH-109` (High/Unknown) establishes ordinary-Human face/sex stores and the skip-equipment argument, and their absence on the NPC arm. `UNIT-PLACEGEAR-098` (High/Unknown) establishes the constructor gate, not complete native outcomes. `ITEM-OWNED-028` (High/Medium) distinguishes carried and actor-dispatched stock elements. `ITEM-HUMEQ-030` (High/Medium) and `ITEM-203` (High) identify Human equipment dispatch and refused/displaced-item carrier destinations. These claims were read through the pinned claim tool. No original program was run or inspected.

## Stock census

Input: the lawful EN and RU roots. The read-only instrument uses `BuildMapList`, `DirMaps`, `ArchiveMaps`, `alm.Open`, `Map.Loot` and the production Normal-difficulty loader. It was built with `scripts/build-tools.sh` at the merged base. The population is every map the installed map-list builder lists. These are installed-data observations, not original-runtime outcomes.

| Root | Maps | Owned stock records | Elements | Nonzero element +0x04 |
|---|---:|---:|---:|---:|
| EN | 38 | 43 | 25 | 5 |
| RU | 34 | 43 | 25 | 5 |

The five nonzero elements are identical on both roots. Stock and element indices count from zero.

| Map | Stock index | Element | Owner unit ID | Placement index | Field04 | Item code | Recipe |
|---|---:|---:|---:|---:|---:|---|---:|
| scenario/110.alm | 0 | 0 | 46 | 25 | 1 | 0x5648 | 0 |
| scenario/110.alm | 0 | 1 | 46 | 25 | 1 | 0x212f | 0 |
| scenario/110.alm | 0 | 2 | 46 | 25 | 1 | 0x2832 | 0 |
| scenario/111.alm | 2 | 0 | 91 | 42 | 1 | 0x7173 | 0 |
| scenario/111.alm | 2 | 1 | 91 | 42 | 1 | 0x0e08 | 0 |

Both owners have flags and secondary key zero and resolve through the Human definition-ID arm: owner 46 to Humans row 124, owner 91 to row 126. On the base every listed item is carried because `stockFrom` ignores `Field04`. The only observed nonzero value is 1. A native writer should use 1. Values beyond zero/nonzero and original native gear outcomes remain Unknown.

The same read-only census rebuilt against slice 3 observes the same counts and codes on both roots. On 110.alm, 0x212f is worn in slot 1, 0x5648 in slot 6 and 0x2832 in slot 8; the carrier becomes [0xa627, 0x9134], the displaced row head and weapon. On 111.alm, 0x7173 is worn in slot 1, 0x0e08 is consumed through the potion operation, and the carrier becomes [0x2133], the displaced row weapon. All five stock elements stop being carried. Stock gold remains ignored. This is an Againrom loader observation, not an original-runtime witness.

## Builders

| Kind | One builder |
|---|---|
| Doll layers and masks | `composeDoll`; icon loading is data, inventory and unit wrappers keep their backgrounds |
| Native field bytes and flags | `mapedit.Field.put` / `decode`, `Editor.SetFields` |
| Equipment slots | `mod.EquipmentSlots`; existing item and anchor declarations derive their maps from it |
| Equipment targets, hand conflicts and rearm | `mapload.EquipTarget`, `EquipmentDisplacement`, `RearmWithLayers` / `ApplyRearmLoadout`; game wrappers delegate |
| Initial owned items | `stockFrom`, `World.ReceiveStockItem` and the existing `equip` / `usePotion` operations |
| Owned stock lookup and recipe copies | `Editor.UnitStock`, `AddUnitStock`, `AddItemRecipe` and the existing clone writers |
| Carried elements and duplicate bytes | `inventoryFields`, `setCarried`, `setStockRecipe`; the existing `Field.write` copies an element or writes its words |
| Read-only corpse explanation | `corpseDropLines`, shared by the inspector and draft |
| Inline magic fields | Existing `Fields("Enchantment")` and `setField`; `AddItemRecipe` makes shared links private |
| Face/sex semantics | `lookFields`, `PlacedFigure` and the existing ordinary-Human admission |
| Native NPC identity | `npcFields` expands a selected registry record into the native flag and secondary key through `Editor.SetFields` |
| Placement field domain | The existing `UnitKeys` policy declares class-field capability; the editor does not branch on a game ID or compare policy identities |
| NPC choice list | `npcChoices`; message speakers and placement records use the same catalog/label builder with their existing name policies |
| Draft preview | `mapInspectionEdits.UnitSettings`, the production world builder and doll composer |
| Popup and dropdown | Existing object popup, `layoutPopup` / `drawPopup` with section/column data, `Picker.Find`, `drawObjectPreview`, `fitRect` |
| Font text and search | Existing `inspectionUTF8` / `messageUTF8`, passed through the editor's host text seam |
| Apply, history and Save | Existing `change`, `SetFields`, `SetObject`, workspace history and fenced `SaveAs` |

The static guard confines game-tier doll layer-program reads to `composeDoll`. No save format or FrontEnd field is added.

## Native identity

Slice 1 commit: `92a8990c3b93044d92c553d96a55b6b454c52c03`.

Slice 2 adds an existing-NPC dropdown beside class and Human definition. The list admits only catalog records the current Human placement resolves. Selecting a record sets flag bit 0 and its whole secondary word. Other flags, class, definition, position, owner, group and unit ID are retained. The existing definition field states that the NPC arm ignores it. There is no editable placement name.

The None choice is -1 only in the UI. It clears the NPC flag and writes secondary word zero, restoring the class/definition arm and its inherited face. It does not store -1, nor remember the face the same word held before it became an NPC reference. NPC record 0 is distinct from None when the catalog holds it. Unknown, unresolvable and out-of-range records fail atomically. The raw NPC-flag field is not exposed or accepted separately by the game host.

The draft hides fields its resolved arm no longer reads. Selecting a different identity and applying rebuilds the available fields. New appearance and NPC controls require the class-key placement layout; server-key placements retain their existing policy. The shared message speaker choice list retains its prior role/name presentation.

Slice 2 fixtures passed: NPC preview, exact flags/secondary key, refusal of NPC-inapplicable face/sex/gear edits, Save/reopen, exact Undo/Redo, clone identity, record 0 versus None, malformed choices, alternate key layout, and mission SAV/cold LOAD with an equal world hash.

## Native equipment

The admitted native encoding does not establish how to author Empty for an inherited worn slot on the NPC arm. `ALM-FLAGPATH-109` states that this arm passes literal zero rather than flag bit 7 to the constructor. `UNIT-PLACEGEAR-098` therefore cannot supply its row-suppression mechanism here. `ITEM-OWNED-028`, `ITEM-HUMEQ-030` and `ITEM-203` establish item dispatch and replacement/refusal, not an independent clear-slot stock operation.

An ordinary Human could suppress all row cells and materialize its retained equipment as worn stock. Applying that strategy to a named NPC would ignore the arm distinction. No extra format, custom template, unknown Field04 value or weakened gear guard is introduced.

The seat accepted unavailable Empty on the NPC arm. Its dropdown carries the reason. Inherit removes that slot's explicit worn stock; an explicit item runs after the row and displaces through the common equipment operation. The native NPC flag and key are retained.

Ordinary Human Empty sets the skip bit and writes the retained row items as worn stock, plus any row overflow as carried stock. Existing stock elements keep their order behind those materialized items. Effects are written through the existing type-9 recipe builder. Inherit while the skip bit remains set materializes that slot's row item; the native records retain concrete items rather than an independent per-slot inheritance flag. Material and tier choices come from the existing catalog. Choosing a recipe creates a private copy. Clone uses the existing owned stock/recipe copy route.

The stock decoder preserves the zero/nonzero predicate and runs the elements after row construction without advancing the clock. The shared equipment target, hand displacement and rearm builders serve the live UI and initial stock. The potion route uses the existing clockless operation; ITEM-USE-113 establishes that base Item application can consume a potion. Refusals stay carried. The existing second-game profile retains its carry-only policy and ordering. No new persisted field or own map section is added.

Slice 3 fixtures pass equipment replacement, two-hand displacement, nonzero values 1/2/65535, full-health and wounded potions, carry-only profile isolation, the unavailable NPC Empty choice, ordinary suppression/materialization, private recipes, variants, exact Undo/Redo, Save/reopen, clone and cold SAV LOAD.

## Carried items and derived loot

Slice 4 offers added carried elements separately from the resulting carrier. Add, replace, remove and duplicate use the existing native stock writer. A duplicate preserves all ten element bytes and copies its recipe privately. Quantity is represented by repeated elements. New potions retain their inherent effects once, without copying those effects into a second recipe. Compound edits process carried indices before worn-item materialization and discard recipe edits for removed items.

The read-only result includes displaced or refused equipment and the existing corpse-drop explanation. `corpseDropLines` is the one explanation builder used by the inspector and the draft. No new death policy or per-item drop flag is stored. Fixtures compare the explanation's contents against terminal death and template-name suppression, plus exact Undo, private recipe Save/reopen and mission SAV/cold LOAD.

## Complete window

Slice 5 extends the existing object popup. E opens it for an admitted native Human. At width 1100 or above, identity/look, equipment and bag/loot occupy three columns beside the card/doll. Stats replaces the right column when selected. Smaller windows show one of four sections. The card, doll, section buttons, override count and Apply/Cancel stay outside the scrolled rows. All twelve equipment rows remain visible in the wide fixture; unsupported empty slots 3 and 11 are read-only. A selected item exposes its variant and recipe controls below the primary rows. Hover retains full labels.

Face choices carry thumbnails composed by `composeDoll` and include only readable bases in the resolved directory. The live doll remains the detailed appearance witness. The final picker uses thumbnail rows rather than the proposal's thumbnail grid. Recipe effects appear as selected-item child rows in the same popup rather than a separate modal. Both are presentation choices within the existing builders.

`Picker.Find` owns case-insensitive substring search; the editor decodes font bytes before matching typed text. Tab follows the rendered fields and sections. Enter opens or accepts a dropdown, Left/Right cycles a choice, Ctrl+Enter applies, and Escape closes a child dropdown before cancelling the window. Reset overrides restores the existing absent sentinels and requires Apply.

`UnitSettings` reconciles the draft's fields after each valid preview. A sequential `InspectionFieldValue.Step` journal lets a newly added item or recipe be edited before Apply. It is editor-only input data. `SetFields` replays each batch through the same checks and native writers, then squashes the whole operation. A failure changes no map bytes. A byte-identical roundtrip creates no history. Cancel discards the journal. The existing `Fields("Enchantment")`, `setField` and `AddItemRecipe` routes serve inline effects; editing a shared item recipe first makes a private copy. No placement, SAV or map-format field is added.

The whole-window fixture combines face/sex, material/tier, worn and carried magic, group and overrides from all five groups at 1280x800, 800x600 and 640x480. It checks no writes before Apply, Cancel, exact one-entry Undo/Redo, Save/reopen and equal-world-hash cold SAV LOAD. Save/reopen compares the existing save form, whose owned-stock coordinates follow the owner. Separate fixtures cover shared recipe isolation, invalid and out-of-order drafts, no-op history, popup hit/draw geometry, both-language layout, missing choices, search and keyboard selection.

## Verification

Slice 1 is implemented. Focused fixtures cover exact masked bytes, inherited appearance, draft preview, Apply/Undo/Redo, editor Save/reopen, mission start, ordinary SAV and cold LOAD. The UI fixture reads the painted draft doll and keeps creatures on their existing preview route. Synthetic sprites are built in memory. Fixture success is not install validation.

| Gate | Result | Elapsed |
|---|---|---:|
| Changed packages plus architecture, story, gated-test and divergence guards, count=1 | PASS | 143 s |
| Public build, ./... | PASS | 23 s |
| Vet, ./... | PASS | 12 s |
| Full editor key/gesture mode test | PASS | 85 s |
| check-fast.sh | PASS | 41.1 s |
| Divergence claim check | PASS, 352 inherited retraction warnings | less than 1 s |

Release tests, scenarios and original-runtime experiments were not run.

Slice 2 verification:

| Gate | Result | Elapsed |
|---|---|---:|
| Focused native identity and shared choice fixtures | PASS | 33 s |
| Changed packages plus architecture, story, gated-test and divergence guards, count=1 | PASS | 146 s |
| Public build, ./... | PASS | 26 s |
| Vet, ./... | PASS | 7 s |
| Full editor key/gesture mode test | PASS | 86 s |
| check-fast.sh | PASS | 40.5 s |
| Divergence claim check | PASS, 352 inherited retraction warnings | 1 s |

## Slice 3 verification

Slice 3 package receipts pass game, mapload, mapedit, sim, mod, words, synth and all four guards. The second combined run took 146 s and failed only the editor-ledger size ceiling; the final mapedit/divledger rerun passed in 2 s after condensing the own row. The extended architecture guard passed in 17 s. Focused fixtures passed in 31 s. Build passed in 24 s, vet in 14 s, full editor modes in 99 s, and the DIV claim check in less than 1 s with the same 352 inherited retraction warnings. The final fast gate passed in 15.3 s; the prior uncached run passed in 45.7 s.

## Slice 4 verification

Slice 4 verification: focused fixtures PASS in 41 s; all changed packages and four guards PASS in 153 s; build PASS in 28 s; vet PASS in 9 s; full editor modes PASS in 111 s; fast gate PASS in 50.2 s; DIV claim check PASS in less than 1 s with 352 inherited warnings. The extended frame witness adds a carried item and reads the draft loot, then checks exact Undo. It remains compiled only.

## Slice 5 verification

Focused fixtures, including native EN/RU choice encoding, PASS in 29 s. Final changed packages and all four guards PASS in 139 s; the preceding run passed in 140 s and a final UI/guard receipt passed in 77 s. Final build PASS in 24 s, vet PASS in 10 s and full editor modes PASS in 94 s. The DIV claim check passes in less than 1 s with the same 352 inherited warnings. The mode golden adds one expected no-op column for Ctrl+Enter on the generic hosts; all 109 rows preserve every previous column byte-for-byte. Native Ctrl+Enter is tested separately. No observed replacement hashes were accepted automatically.

The final fast gate passes in 38.2 s, within the 60 s bound. All five slices are complete. Release tests are named below and were not run.

The initial broad run exposed the forbidden UI-to-format-codec edge, global dropdown Enter affecting cell spells, positional diagnostic changes and a falling comment counter. The host supplies the existing text decoder. Enter, search and Ctrl+Enter use the person window; Up/Down skips Shut rows in every object dropdown. The test logger retains every field plus nonzero steps. CommentBytes falls from 9869024 to 9868986. No guard or size ceiling was relaxed.

## Release witnesses

`TestReleaseOwnedStockDispatchesOnBothShippedMaps` must run on EN and RU. It reads scenario/110.alm and scenario/111.alm, checks the five elements leave the carrier, and checks the four equipment items are worn. The potion is consumed rather than assigned an invented equipment slot. This witness is compiled, not run in the lane.

Run on EN and RU at the next release, not in this lane: `TestReleasePlacedPersonsDrawTheFaceByteTheirSpawnerWrites`, `TestReleaseZeroModeNPCFaceByteSurvivesSaveAndLoad`, `TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes`, `TestReleaseMapEditorOverrideFrames`, `TestReleaseMapEditorOwnerFrames`, `TestReleaseMapEditorUnitValuesEqualTheGame`, `TestReleaseMapEditorFreshRecordAuthoringSAVAndGoals`, `TestReleaseMapEditorUnitStockCloneHasIndependentInventory`, and `TestReleaseEditorInheritedHintsWithStatOverride`. The extended override frame witness changes face/sex and NPC identity through dropdowns, reads the installed doll and checks exact Undo. It adds carried contents and the 800x600/640x480 setup frames. The SAV witnesses retain native face/identity after cold LOAD. New-selector mission SAV proof remains fixture-only until release validation joins those routes.
