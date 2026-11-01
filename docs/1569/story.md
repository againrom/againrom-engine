# The map editor's bar search, one property view per object, the speaker's name

## Intent

The bar holds a search field after the record categories: "Search (Ctrl+F)" (RU «Поиск (Ctrl+F)»).
Ctrl+F puts the typing focus in it and selects its kept text; Esc leaves it and empties it. Typing
lists the matching units, structures, sacks, groups, players and scripts in one drop-down under the
field, each hit led by its category's word. A click on a hit, or Enter on the highlighted one,
selects that record as a press on its tree row does: the map scrolls to a record with a place, the
right panel shows it, and no side panel opens, closes or changes width.

A unit's sack opens in the window a selected stock takes, plain or technical as the profile keeps
it. The unit's bag shows one row, "> Open the unit's sack (N items)" (RU «Открыть мешок отряда
(предметов: N)»), which opens it with Back to the unit; a unit with no stock reads "The unit has no
sack yet: K adds one and opens it." K opens the stock it adds the same way. The person window's
technical Bag tab, which kept its own larger floating window over the inventory fields, is gone:
the tab holds the same sack row.

The sack's window offers the carried-item actions the person window had. An item's choice list
holds Duplicate, which appends a copy of the item whose recipe is its own copy of the original's
(not on a unit's worn item, whose slot holds one). An item's recipe choice holds no recipe, a new
recipe of its own, "Copy for this item: Recipe N" (RU «Копия для этого предмета: Recipe N») for
each recipe, and each recipe shared as it stands. The recipe's fields stand after the item's recipe
row, in the plain view's Details and in the technical view; an edit to a recipe more than one item
links copies it for that item first, so the other items keep their effects. One Apply orders an
item's writes as the person window did: items from the last, a removed item's other fields dropped.

The speaker choice of a dialogue page and the person window's NPC list show the speakers' names
only. Resting the pointer on a choice with a face shows the name and the face at full size in the
editor's tip.

## Authority

- Tester, through the owner, on the left panel: only its search is of use. Seat proposal, in the
  owner's queue: a search field in the bar, and Ctrl+F, over every category.
- Tester: a unit's sack opens a second, older window. Seat proposal: the sack opened from its unit
  shows the same view as a selected sack, with a way back to the unit.
- Tester: the speaker choice's small portraits cannot be read. Seat proposal: names in the choice,
  the face at full size in the tip.
- Owner: the remainder of story 1568's remarks is priority; screens are at least 1920x1080; the map
  editor is our own program, so no claim and no DIV row.

## As built

| kind | builder |
|---|---|
| bar entry | `barKeyRows` / `barRows` / `barEntries` (`pkg/ui/mapeditorwindow.go`): one more `barKey` row, `field: true`; `barRows` gives the field `searchFieldW` and narrows it to the room the other entries leave in the row, down to `searchFieldMinW` |
| text field | the one `editorTextView` through `inputTarget` (key `search`) and `inputAt`; its painter `drawEditRow` (`pkg/ui/editorinputview.go`) is the object popup's text row painter moved out of `drawPopup`, which now calls it for both of its text rows |
| drop-down | `menuDropdown` and `drawDropdown`: `dropdown.Edit` draws the closed box as the text field; the open list is the bar menu's own `menuModel` list, its rows from `menuItems` (`searchItems`), at most `searchRows` shown and scrolled |
| record search | `treeMatches` (`pkg/ui/mapeditorsidepanels.go`), factored out of `treeRows`: the tree and `searchHits` both call it over `treeMembers` |
| selection | `chooseHit` calls `chooseGroup` for a group and `chooseListed` for any other member, the two calls a tree press makes; the tree follows through `treeFollow` |
| key binding | the key table row `search` (`Ctrl+F`, `keySearch`); Esc, Enter and the arrows reach the field through `keyEscape`, `keyCatalog` and `keyStep`; plain F (fit) ignores Ctrl |
| object window link row | `objectFieldRows` (`pkg/ui/mapeditorobject.go`): a read-only field with `InspectionField.Link` adds a `popupLink` row whose record the existing link press opens through `openObjectRecord`; Back is `objectBackButton` / `backObject`; `objectPlace` keeps the person window's tab |
| sack fields of the unit window | `sackField` (`pkg/ui/mapeditorobjectnav.go`): the person window's editable bag fields have no row, plain or technical; the host still produces them (see Open debt) |
| sack item writers | the writers the unit's `carry*` fields use, factored, one per action: `duplicateStockItem` (Duplicate, out of `setCarried`), `copyStockRecipe` (a blank or copied recipe, out of `setStockRecipe`), `setItemRecipeField` (copy-on-write recipe field, out of `setUnitRecipeField`), with `recipeRefs`; `setStockItem` routes the sack's `item#`, `recipe#` and `recipe#.<field>` to them from `setField`; `orderItemFields` (was `orderUnitItemFields`) orders both key families by a key parser |
| sack item choices and rows | the field-choice registry: `stockItemChoices` and `stockRecipeChoices` rows for `KindSack` and `KindStock` (`pkg/game/mapinspectionchoices.go`); the recipe fields from `itemRecipeFields` through `stockRecipeFields`; the plain row `recipe#.*` in `sackPlainView` |
| plain view | `plainViews`: `KindSack` and `KindStock` take one table, `sackPlainView`; the unit's bag block holds the `sack` row |
| person window Tab order | `sectionField` (`pkg/ui/mapeditorrecords.go`), the Tab path `recordKey` took, passing over a tab with no field Tab can focus |
| K route | `keyRecords` (`pkg/ui/mapeditorauthoring.go`) keeps the unit's place on the Back stack after `selectNewest` |
| host sack field | `inventoryFields` (`pkg/game/mapinspectioninventory.go`): a first read-only `sack` field with `Link` to the unit's stock, or a line naming K |
| choice rows | `choiceRows`: a field with `InspectionField.IconTip` puts no picture or picture cell in its rows |
| tip | `editorTip` gains `picture`; `observeTip` asks `choiceTipAt` first, then the grid and toolbar tips; `editorTipStyle` passes the picture to `hoverStyleFor(hoverStyleEditor, ...)`; `composeHoverBox` (`pkg/ui/widgethover.go`) draws a style's picture at its own size under the lines |
| host speaker fields | `MessageFields` (`pkg/game/mapmessagepages.go`) and `unitFieldValue` for `npc` (`pkg/game/mapinspectionlook.go`) set `IconTip` beside the `Icons` they already set |
| editor words | EN and RU tables: `editor.bar.search`, `editor.inventory.sack`, `editor.inventory.nosack`, `editor.inventory.recipecopy` |

No second search rule, text field, drop-down drawer, property view or tip drawer was added; the
search lists the members of every category through the tree's one member producer.

The case-fold allow list in `internal/archtest/casefold.go` names `treeMatches` in place of
`treeRows`, which no longer folds case itself.

`InspectionField` gains `Link` and `IconTip`. The MCP tool output marshals `InspectionField`
without JSON tags, so both fields appear in its JSON.

The modal notice K leaves ("Stock created...") takes the first press or Esc, as every editor
notice does; Back works after it.

## Proof

Focused tests at 1920x1080, EN and RU where words show:

- `pkg/ui/mapeditorsearch_test.go`: `TestMapEditorBarSearchTakesCtrlF`,
  `TestMapEditorBarSearchListsEveryCategory` (at least three categories, players, units and
  scripts among them),
  `TestMapEditorBarSearchMatchesTheTreeSearch` (table test against the tree's rows for the same
  text), `TestMapEditorBarSearchChoiceIsATreePress` (click and Enter; selection, camera, panels).
- `pkg/ui/mapeditorunitsack_test.go`: `TestMapEditorUnitSackOpensInTheSackView` (plain and
  technical, the rows equal to a selected stock's, Back to the tab it left, no floating second
  window) and
  `TestMapEditorUnitSackEditUndoes`.
- `pkg/game/mapeditorunitsack_test.go`, over the game's host: `TestEditorUnitSackEditsInTheStockWindow`
  (both views; the bytes equal the host field writer's; undo and redo exact with the stock's window
  kept; Back), `TestEditorStockKeyOpensTheNewStockWithBack`, and
  `TestEditorUnitSackBytesMatchTheWriters`, the release helper on a fixture map.
- `pkg/game/mapeditorsackrecipe_test.go`, over the game's host in the unit's sack's window, EN
  and RU at 1920x1080, plain and technical, each failing at `43f03faa9`:
  `TestEditorSackViewDuplicatesAnItemWithItsRecipe` (the bytes of the unit's `carry1 = -1`),
  `TestEditorSackViewCopiesARecipeForOneItem` (the template and the items linking it unchanged by
  an edit of the copy), `TestEditorSackViewEffectEditCopiesASharedRecipe` (the bytes of
  `carryrecipe1.power`; exact undo and redo) and `TestEditorSackViewListsTheItemRows`.
  `TestEditorSackViewTypedRecipeNumberLinksTheSharedRecipe` holds the typed route a release test
  uses: a typed recipe number still links that recipe shared, 0 unlinks; it passes at
  `43f03faa9` too.
- `pkg/ui/mapeditorsackrecipe_test.go`: `TestMapEditorSackViewListsTheRecipeRows` (the recipe's
  fields in Details after the item's recipe row; fails at `43f03faa9` in the plain view).
- `pkg/ui/mapeditorspeakerportrait_test.go`: `TestSpeakerListNamesEachSpeaker` (EN and RU names,
  no picture in the list, no tip before the delay, the tip's text and its 96x96 face pixels).
- `pkg/game/mapinspectionspeakerportrait_test.go` asserts `IconTip` on the speaker and NPC fields.

At the base each new test file fails to build: `searchRow`, `searchLeave`, `InspectionField.Link`,
`InspectionField.IconTip` and `editorTip.picture` do not exist there.
`TestEditorUnitSackBytesMatchTheWriters` proves equality with the writers, which the base also
meets; its file does not build at the base.

Map bytes on installs: `TestReleaseMapEditorUnitSackBytes` (`pkg/game/mapeditorunitsack_release_test.go`)
opens the first three installed maps with a unit, in both views: Save with no edit writes the bytes
`mapedit` writes for the source; in the sack's window (sack row, or K for a unit without a stock)
an item added from the additem list, a new recipe for it, its Duplicate and its recipe made a copy
of the duplicate's, each chosen from its list and applied, write the bytes the host's `AddStock`
and `SetFields` write. `pkg/mapedit` is
unchanged from the base. The test did not run in the lane.

Goldens re-recorded, with before and after dumps in the lane notes:

- `mapeditor_speaker_portraits.golden`: the list rows lose their picture and cell; EN and RU tip
  lines with the tip box and 9216 face pixels.
- `mapeditor_plain_views.golden`: 16 frame hashes (the bar's search field); the unit's "Carried
  item 1" and "Add carried item" rows leave the bag (4 views), and the floating unit window ends at
  592 in place of 620.
- `mapeditor_technical_views.golden`: the same carry rows leave the technical unit window.
- `mapeditor_side_panels.golden`: 9 frame hashes (the bar's search field); no rectangle moved.
- `mapeditor_technical_reach.golden`: the Bag tab holds the sack row and no fields; the floating
  window shows rows 0..29 in place of 0..33. The EN identity, equipment and stats tabs at 520 and
  800 show one row fewer: the test's 1280x800 window wraps the EN bar with the fixture font. The
  writer no longer ends a line with an empty field list in a space, so 6 more lines differ by that
  space alone.
- The three `chrome/*` pins of `TestFrameInstancesKeepTheirPixels`: the chrome image holds the bar.
  With the search field's `barKey` row taken out of the candidate for one run, the three hashes
  equal the base pins.
- `mapeditor_modes.golden`, once with `MAPEDITOR_MODES_WRITE=1`: every frame hash of the 112
  states, as the bar's search field shows in every frame; the 86 columns per state are unchanged.

## Open debt

- The unit's `carry#`, `carryrecipe#`, `carryrecipe#.<field>` and `carryadd` host fields have no
  reader in the editor's windows. They stay for MCP clients, which address a unit's carried items
  by the unit without resolving its stock, and for the host tests that pin those writes. They
  call the same writers as the sack's rows, so each action keeps one writer.
- The recipe choice labels ("No recipe (plain item)", "Recipe N: spellS, E effects") and the item
  list's "Remove this item" are English in both languages, as before this story.
- The fixture font wraps the EN bar at 1280 pixels; the owner's least screen is 1920x1080, which
  the new tests use.
- Release witnesses for the next release, EN and RU (`gameversions/en`, `gameversions/ru`; the
  ROM2 roots where the test does not skip): `TestReleaseMapEditorUnitSackBytes` (all four roots),
  `TestReleaseMapEditorOverrideFrames` (adds the carried item through the sack's window),
  `TestReleaseTriggerMessageFrames`, `TestReleaseTriggerMessageEditsAndUndo`,
  `TestReleaseMessagePickerParts`, `TestReleaseMessagePickerPopulation`,
  `TestReleaseMapEditorUnitTierFrames`, `TestReleaseMapEditorOwnerFrames` and
  `TestReleaseEditorInteractionFrames`. None ran in the lane.
