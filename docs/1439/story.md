# ROM1 map record authoring

## Intent

Complete the missing record-authoring routes of the first-game editor. A new
map can receive its first structure, uniquely targeted clones, unit groups,
initial stock and independently linked item recipes. Base: `15d4bbb0`.

## Authority

Owner direction supplies the editor operations. `ALM-OBJ-019` and
`ALM-OBJ-034` establish the structure frame and extension. Their retractions
remain binding. `ALM-STRUCTREAD-228` establishes the owner low-word and signed
identity transfers (High), with the structure binder separately Medium.
`ALM-OWN-039` establishes the roster-slot owner. `ALM-128` distinguishes
Building's zero placement override from Shop's signed stock cap.

`ALM-UNIT-048` and `ALM-TRIG-046` establish the group and target domains
(Medium). `ALM-SACK-065` establishes the loot frame and one-based recipe link
(High). `ITEM-OWNED-028` establishes the carrier destination, with its
missing-actor amendment. `ALM-EFFREC-071` establishes the ordered recipe
(Medium); `ALM-EFFLINK-072` grades link arithmetic High and construction
Medium. `ALM-T9CONTROL-175` qualifies the head byte conversion. These are
format and consumer facts, not proof of the original editor's defaults.

`ALM-UNIT-040` and `ALM-ORD-068` establish the unit's one-based loot link
and the ordered loader dependencies used when adding an absent section.

## As built

The section table, common append builder, field table and host transaction
remain the only builders. J, a cell click and a kind place a structure. The
constructor uses centred coordinates, owner1, placement word1, zero extension
for kind33 and a positive signed identity above every existing or script-held
structure target. Clone uses the same identity allocator. Exhaustion refuses
without mutation. Structure E fields expose the low-word owner, identity and
raw signed placement word while preserving the owner high-word. Changing an
unambiguous identity retargets existing action and condition references in the
same Undo entry; duplicate or reserved destinations refuse.

Unit E fields include the unsigned group word. K on a selected unit adds an
empty stock owned by its unique nonzero unit identity and writes the unit's
one-based loot link in the same edit. Existing stocks, existing links and
ambiguous unit identities refuse. E adds or edits its items as for ground loot.
Unit C clones the owned stock and each linked item recipe with independent
records and repaired owner/link words in the same transaction. The source
stock and recipes remain unchanged. A clone without provable owned stock
receives no stock link. Multiple stocks or ambiguous ownership refuse.
Unit Delete requires owned stock removal first. Stock Delete clears or lowers
surviving unit links atomically, after which Unit Delete works. Imported orphan
stocks are named by the existing stock-link warning. An opaque malformed loot
payload is carried; the unit-only byte model does not infer ownership from it.

I on a selected sack or stock asks for its one-based item number. It appends
a zero-coordinate recipe and its item link atomically, then selects the
Enchantment record. E edits raw A/B/C, spell id/power and ordered effect
kind/low/high words. Adding an effect appends six bytes; kind0 removes it.
Item fields include the one-based recipe link, with zero restoring a plain
item. New recipes are independent; deliberately shared existing links retain
their shared editing semantics. A nonzero-coordinate caster entry is carried;
item fields and Delete refuse it. Delete refuses a referenced recipe. An unused
recipe is removed with every higher loot link lowered in the same edit,
preserving all surviving bindings. No simulation or SAV-format change is made.

Structure, unit, item, definition, spell and effect choices show installed
names through the shared numeric field prompt. Up/Down selects and Enter
commits through the existing constructor or field setter. Manual numeric input
retains the same bounds and installed-table checks. Owner and recipe-item
choices come from the current map. Unnamed rows remain explicitly raw; mod-only
registry definitions that the inspector does not load remain refused.

Constructor and validation policy is recorded in DIV-3378 through DIV-3380.
The former structure/stock limitations in docs/1421/story.md are historical.

## Proof

`TestMapEditorRecordAuthoringMissingRoutes` failed all five routes on the base:
duplicate cloned identity, absent structure placement, absent group field,
absent stock creation and absent recipe creation. It passes after the change.

Model tests cover first placement, script-reserved IDs, signed exhaustion,
ambiguous identity and duplicate target refusals, atomic reference retarget,
stock/link and recipe/link Undo/Redo, independent recipes, full-width group
values, bounds and unchanged bytes/history on refusal. Existing clone buffer
expectations now include the one changed identity word. The architecture scan
still requires exactly one ALM decode entry per decoder; its declared table
names the moved append-part builder and the additional type9 counted layout.

`TestMapEditorRecordAuthoringWindowHistoryAndRefusal` uses production keys,
pointer input and field prompts, then checks exact bytes through every Undo
and Redo and refused host edits.

`TestReleaseMapEditorFreshRecordAuthoringSAVAndGoals` authors a new map through
the production window, saves into a mod, reopens ALM, materializes distinct
stock/ground recipes, cold-loads ordinary SAV, and proves the group and structure
goals. It first admits both target members and an unrelated group member.
Unrelated death and first-member death do not win; last-member death wins.
Destroying the wrong cloned structure does not win; destroying the renamed
original target wins. A temporary ring placement must report that mission load
removes it and must undo exactly. Saved/reopened recipe links can be cleared,
changed and cleaned up without changing other bindings.

`TestReleaseMapEditorUnitStockCloneHasIndependentInventory` clones through C,
checks one Undo/Redo for the unit and stock, drops the clone's item in gameplay,
and checks the original inventory and distinct ground destination through cold
SAV LOAD. `TestReleaseMapEditorNamedRecordChoicesUseExistingPrompts` reaches
J/U/S/I and the recipe spell field using Up/Down and Enter without typed IDs.

Prepared workspace dependency `2117aef8` is applied over the base. It includes
version0.123.15, contextual F1 help and table-derived mission-page admission.
The three new release witnesses plus the workspace/text/route witness pass on
EN (3.435s) and RU (3.241s). Their receipts are
`review/rom1-editor-completion/1439/final-release-{en,ru}.log` at the seat.
The production keys, pointer and field routes run headlessly; no native desktop
or original-runtime session was driven.

Focused model/UI/game tests pass (0.503s, 12.135s, 2.058s). Ordinary touched
packages pass: mapedit (0.819s), game (107.030s), UI (43.463s), strict architecture
(26.034s), comment guard (3.919s), and gated-test manifest (1.585s). The manifest
adds exactly the three new release witnesses. The UI matrix retains all42
prior key inputs, all19 gestures and all56 entry paths. It adds J/K/I and two
structure paths: all92 old golden rows remain in the94-row matrix, each with
exactly three added action columns. Its readable behaviour controls and small
host-path selection remain unchanged.
`TestMapEditorNewRecordNamesAreVisibleBeforeCommit` checks visible modal names,
selection wrap and manual numeric input through the same constructor. The
comment guard passes after its measured count falls by120; no bound rises.
`go build ./...`, `gofmt -l` for the touched packages and `git diff --check`
pass. The generic type9 cleanup boundary has a byte/history refusal test.
The seat owns published-main reconciliation, final candidate gates, sole
review and landing. No full final chain is claimed.

The installed dangling-script save witness removes only the selected scripted
unit's unique owned stock through the window before removing that same unit.
Cleanup leaves no report errors, clears the unit link and preserves every
other loot record. The first Undo restores the post-stock bytes and dirty
state; the second restores the exact installed bytes and clean state. A unit
without stock retains the single-Undo route. Save refusal, no output file,
report selection and final saved-byte assertions remain in place.

## Unknowns

Original-editor defaults for newly constructed structures, stocks and recipes
are not established. Tag0 and other zero recipe defaults are explicit authoring
policy. A/B/C stay raw transfer fields, with the established head conversion
shown in their labels. The editor does not assert universal effect-kind
validity or native runtime admission. Nonzero-coordinate caster authoring,
including removal, remains unsupported with explicit refusal; its complete
reference/cleanup domain is not established here. Mod-added registry classes
that the inspector does not load remain outside this slice.
