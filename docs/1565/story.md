# The map editor's plain and technical property views

## Intent

The object window of a unit, a structure and a sack opens in a plain view: the main fields first
under plain words, items by name and icon, the rest in blocks. A switch in the window's title row
turns it to the technical view, the window as it was, and back. The editor profile keeps the
choice. The switch changes no data: both views edit one draft and Apply writes it through
`applyObject`, as before.

## Authority

- Owner, forwarding the proposal `pipeline/BACKLOG.md` "Editor: plain and technical property
  views": units with the main parameters at once, equipment as slots, stats as a readable table and
  the map-file fields under Raw; structures with type, owner, position and script target first,
  the script actions that name them below, and the placement word with its known bits in an
  expandable block; sacks with each item as its name and icon and the ids, recipe and enchantment
  in details. Owner, «Отдельно после панелей»: a story of its own after story 1556.
- The map editor is Againrom's own program: no research, no DIV row. Map editor stories do not
  count toward a release.
- The placement word's meaning is `ALM-128` (a building that is not a shop starts with no health
  when the word is 0, with its type's health otherwise; types 34 and 35 read it otherwise) and
  `ALM-127` (the word keeps 16 raw bits).

## As built

| kind | builder |
|---|---|
| object window | `objectSpecBase` and `objectFieldRows` over `popupSpec`, laid out by `layoutPopup` and drawn by `drawPopup` |
| field row | the `fieldRows` closure inside `objectFieldRows`; the technical loop and the plain builder `plainFieldRows` both call it |
| plain table | `plainViews` in `pkg/ui/mapeditorplain.go`, one `plainView` per `RecordKind`: blocks `{id, word, fold, groups}` and rows `{key, word, order, block, show}` |
| expandable block | a `popupHeading` row with the tree's fold state: `popupRow.Fold` is a `RowFold` (`FoldOpen`, `FoldClosed`), built only by `blockRow` in `pkg/ui/editorpopup.go`; the painter draws `rowFoldMarks[Fold]` and steps the label by `listIndentStep`, as the list box draws a tree row, and `PopupLines` writes the mark as `TreeLines` does; the label holds no mark |
| view switch | `popupSpec.TitleButton`, laid out by `layoutPopup` left of the close button and drawn by `drawPushButton`, the close button's builder; `hit` answers `popupTitleButton` |
| row scroll | the popup's `Top`/`MaxRows`/`RowsFit` and the editor's `objectTop`, moved by the wheel through `scrollSpec`; `hostedObject` sets the hosted window's rows from the fit below the card |
| record kind | `RecordKind` constants as the table's keys; no kind word literal |
| writes | `applyObject` and `commandSetObject`, unchanged |
| item name | `mapload.ItemDisplayName` over `ResolveItemName`, editor list form, through the host's new `ItemLook` (`ui.InspectionItemLook`) |
| item icon | the inventory's icon reader: `itemIcon` in `pkg/game/inventory.go`, factored out of `buildInventoryPack`, which now calls it |
| profile | `panelStateBytes` and `SetPanelProfile` in `pkg/ui/mapeditorsidepanels.go`: the key `"view"` inside `"right"` of `mapedit-panels.json`, written by the same writer |
| words | `pkg/words/text/{en,ru}/strings.toml`, `editor.view.*` and `editor.plain.*` |

The switch is a button in the title row, left of the close button: "View: Plain" or "View:
Technical" (RU «Вид: Обычный» or «Вид: Технический»), as wide as its caption. The button row is
the base's: the same buttons, numbers, widths and lines in both views. The title text is clipped
before the switch. It shows on a unit's, a
structure's and a sack's own window, not on a script, message or mod window. An editor that never
loads a profile (pkg/game hosts and tests) shows the technical view; `cmd/mapedit` loads the
profile, and a missing, broken or older profile, or an unknown `"view"` word, opens the plain view.

A field no table row names stands in the kind's rest block, so every field stays editable in the
plain view too. The plain view's labels are not cut to the technical view's 150-pixel column: the
window takes the `Fit` flow a window in the right panel takes, the label column widened to its
words up to half the row or a label set over its field, and a floating window lowers its rows
until it stays inside the editor.

The window in the right panel scrolls to its last row in both views. Its rows are counted below
the card, from the shown top; the top is not clamped to the floating window's row count, and the
last scroll step is the least top that shows the last row. A row with key `#` matches a number; the number sorts the rows and fills a `%d` in
the row's word.

### Unit

The window shows no tabs in the plain view; the card stays above the rows.

| block | rows |
|---|---|
| main (no heading) | Position (the cell row), Name (the card's name), Owner (`owner`), Kind (`class`), Tier (`tier`), Experience (the card's experience, when the card knows it), Life (`hp`), Mana (`mana`), Group (`group`) |
| Look | `face`, `sex`, `npc`, `npcflag` |
| Equipment | `gear`; `wear#` as the item's name and icon, then its `wearmaterial#`, `weartier#`, `wearrecipe#`, `wearrecipe#.*` in the slot's detail, as in the technical view |
| Bag | `carry#` as the item's name and icon with `carryrecipe#` and `carryrecipe#.*`, then `carryadd` and `holdings` |
| Stats, with each field group's heading | `body`, `mind`, `spirit`, `reaction`, `defence`, `absorption`, `skill.*`, `protection.*`, then `resolved` and `overrides` |
| Map file fields (closed at first) | `definition` and any field no row names |

### Structure

| block | rows |
|---|---|
| main | Position, Type (`kind`), Owner (`owner`), Script target (`identity`), Width (`width`), Height (`height`) |
| uses | the script triggers and actions that name the structure ("Used by"), here instead of after the fields |
| Placement word (closed at first) | Word (`placement`), then "Hex 0x0005; bits set: 0, 2" and the meaning: 0 starts with no health, not 0 keeps the type's health, or for types 34 and 35 that a shop does not read the word as health |
| Map file fields (closed at first) | any field no row names |

### Sack

| block | rows |
|---|---|
| main | Position, Gold (`gold`), Item N (`item#`) as the item's name and icon, Add an item (`additem`) |
| Details (closed at first) | per item: Item N id (the code), `use#`, Item N recipe and enchantment (`recipe#`); and any field no row names |

The record's Data lines, which every object window shows after its fields, form a closed "Data"
block in the plain view.

## Proof

| result | test |
|---|---|
| 1 switch, default, profile | `TestMapEditorViewChoiceRoundTrip` (pkg/ui): missing, broken, older, technical, plain and unknown profiles; the switch saves and reloads; no profile shows technical |
| 2 technical view as the base | `TestMapEditorTechnicalViewKeepsTheWindow` with `testdata/mapeditor_technical_views.golden`, recorded by the same test on the base `749963004` and passing unchanged on the candidate: rows, tabs, columns, buttons and words of the fixture unit (four sections), structure and sack in EN and RU |
| 3, 4, 5 plain rows | `TestMapEditorPlainViewRowsOfEachKind` (pkg/ui), a table written from the fixture, blocks closed and open |
| 3, 4, 5, 6 every field placed and reachable | `TestMapEditorEveryFieldHasAPlace` (pkg/ui fixture) and `TestEditorViewsPlaceEveryHostField` (pkg/game, the host's real fields): each field shape has its block, shows in the plain view once its block opens and shows in the technical view; a shape the host adds fails until placed |
| 6 same bytes | `TestEditorViewsWriteTheSameBytes` (pkg/game): a unit's `hp`, a structure's `placement` and a sack's `gold` typed in the plain view, in the technical view and after the switch, each equal to the host's field writer; `TestMapEditorBothViewsWriteTheSameEdit` (pkg/ui): the draft survives the switch and Apply writes the one value |
| 6 unknown bytes | `TestEditorViewsKeepUnknownBytes` (pkg/game): an unknown section and an uncounted trailer survive three switches and an edit of `group`, and no byte outside that field changes |
| 3, 5 item name and icon | `TestEditorViewsNameAndPictureSackItems` (pkg/game: the name from the item-name builder, the icon from the inventory reader, read once) and `TestMapEditorPlainSackShowsItemNamesAndIcons` (pkg/ui: the row shows the host's name and icon) |
| pictures | `TestMapEditorPlainViewGolden` with `testdata/mapeditor_plain_views.golden`: unit, structure, structure with its placement block open and sack at 1280x800, EN and RU, in the right panel and floating; `AGAINROM_VIEWS_PNG` writes the pictures |
| every field in reach | `TestMapEditorWindowReachesEveryField` (pkg/ui, wide fixture): a human with twelve slots, a bag, every stat and protection group and the read-only sheets, a structure and a sack of three items, at 1280x800 with the right panel 360, 520 and 800 wide and floating, EN and RU, both views; after each tab, block and slot detail opens, the popup's scroll brings every field's control or line on screen |
| technical rows as the base | `TestMapEditorTechnicalViewKeepsTheRows` with `testdata/mapeditor_technical_reach.golden`, recorded by the same test on the base `06538aaab`: the button row's line count and the fields shown before a scroll, per record, tab, width and language |
| plain words whole | `TestMapEditorPlainWordsShowWhole` (pkg/ui, wide fixture, a font of 10 pixels a glyph, wider than the installed EN and RU body fonts' words): every `editor.plain.*` word the window shows, the placement word's three meanings and the view switch's caption fit, at each width and floating, EN and RU; the switch stands clear of the button row and the close button |
| one fold | `TestMapEditorPlainBlocksTakeTheTreeFold` (pkg/ui): every block row is a heading whose `Fold` follows the block, its label holds no mark, `PopupLines` and the painter follow `rowFoldMarks` |

`TestMapEditorTechnicalViewKeepsTheRows`, `TestMapEditorWindowReachesEveryField` and
`TestMapEditorPlainWordsShowWhole` fail on the first candidate `22713be91`: the hosted unit window
leaves its last fields (definition, protections, the sheets) out of reach, the switch puts the
button row on a second line at 360 and on the structure and the sack, and plain words such as
"Item 1 recipe and enchantment" are cut. `TestMapEditorPlainBlocksTakeTheTreeFold` does not
compile there.

Re-recorded:

- `testdata/mapeditor_modes.golden`: the switch draws in the title row of every unit, structure and
  sack window; the same 100 records move as on the first candidate, in pixel hashes only.
- `testdata/mapeditor_side_panels.golden`: the pixel hash of the "search" case, whose hosted unit
  window draws the switch in its title; its rows are unchanged.
- `testdata/mapeditor_plain_views.golden`: block rows are headings with a fold (kind 1, the mark
  from `rowFoldMarks`), the switch is in the title, and the labels take the `Fit` flow.

`popupTranscriptHash` and the person window's three buttons are the base's again.

Changed tests: the profile bytes carry `"view"` (`TestMapEditorPanelStateRoundTrip`, `cmd/mapedit`
`TestPanelStateLivesInTheEditorProfile`); the minimized-panel rectangle test reads the technical
view, the window it recorded.

## Open debt

- A field the plain table gives no word keeps the host's own label, as in the technical view
  (for example "Gear set", the slot labels and the stat labels).
- The sack's item rows edit the code through the item list; the name replaces the "code: name"
  label only in the shown value.
- No release test ran in this lane. The installed editor tests that drive the object window run
  it without a profile, so in the technical view with the switch in the title row; the next release
  runs, in EN and RU: `TestReleaseMapEditorOwnerFrames`, `TestReleaseEditorInteractionFrames`,
  `TestReleaseMapEditorOverrideFrames`, `TestReleaseEditorCancelPreservesInstalledMapBytes`,
  `TestReleaseEditorCharacterCardsMatchGameSample`, `TestReleaseEditorInheritedHintsWithStatOverride`,
  `TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes`,
  `TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes`, `TestReleaseMapEditorUnitValuesEqualTheGame`
  and `TestReleaseMapEditorNewUnitTakesTheChosenTier`. The plain view on installed maps is the
  owner's test build.
