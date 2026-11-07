# Pictured, grouped sack item choices

## Intent

A map maker chooses a sack item by its inventory picture and a readable kind
and material, without opening the entire installed item list.

## Authority

Owner direction: the editor owns its layout, uses the existing popup builder,
and presents item kinds before their materials. Remove and Duplicate remain
first. This is a ROM1 editor feature.
No claim about the original editor's layout is made.

## As-built behaviour

The existing field-choice registry still supplies the same ordered numeric
choices. Its item rows also supply `InspectionChoiceGroup` data and a lazy
picture provider. The record window and Add window use `choiceList` to open
kind, material and item levels. Back returns one level. Ungrouped control
choices stay first at every level. Existing flat choices keep their rows.
With no typed value, Enter on a navigation row returns or opens a group even
after a sack has items. Moving to that row clears the pending item. Enter with
an item value adds it; an empty Enter on an item row finishes the step.
Ungrouped Add lists retain zero row ink, including their non-choosable
identity headings.

Sacks, unit stocks, the unit's carried-item controls and new-sack authoring
use this choice. The unit's plain view retains its existing link to the stock
window; its detailed view also offers the carried-item controls.

`ItemCode.A`, `B` and `D`, `EquipSlotFor`, and the loaded `Table` identify an
equipment row and material. The row's data name identifies the kind; carried
items reuse `ItemInstanceFromCode.Kind` for potions, scrolls and books. Other
carried items have their own group. Category captions use the editor's word
tables. An unfamiliar mod row retains its data name; an unavailable row or
material retains its numeric identity. No item-name catalogue is handwritten.
Every individual label still comes from `ItemDisplayName`.

The inspector's existing `recordPictures` cache also owns item pictures.
`ItemLook` and the group's provider use `itemIcon` and `ItemIconPath`; a missing
picture is cached as missing. Opening a kind or material menu reads no group
pictures. Entering a material group resolves its members once per inspector
session. Painting uses the existing popup, dropdown and picker row painters.
The former separate `itemIcons` map is removed.

## Proof

Focused fixtures check exact kind/material membership, unique coverage,
English and Russian captions, picture equality with `itemIcon`, failed-picture
caching, unchanged flat lists, and navigation without synthetic item values.
Keyboard fixtures add a second sack item from another group and return from
an item highlight without adding it. Identity rows match the former flat Add
rows, including every row field.
Full HD pointer paths through ground sacks, unit stocks and carried-item
controls compare edited ALM bytes with the existing field writers.

`TestEditorInstalledItemGroups` checks installed code coverage and prints the
group tree. `TestEditorInstalledItemChoiceFirstOpenCost` selects the largest
installed map by decoded area and times the first choice and largest group.
These instruments read installs through the engine loaders.

Correction gates are focused tests, archtest and the fast gate; the seat
owns the landing modes walk.
The next release owes these tests on ROM1 EN and RU:
`TestReleaseMapEditorUnitSackBytes`,
`TestReleaseMapEditorUnitStockCloneHasIndependentInventory`,
`TestReleaseMapEditorStructuresAndSacksPlayAsModMission`,
`TestReleaseDeletingAGroundSackKeepsTheOtherSacks`,
`TestReleaseMapEditorFieldChoicesKeepTheirLists`, and
`TestReleaseMapEditorAddWindowWritesTheRouteBytes`.

## Open debt

The seat owns publication, review, reconciliation with main and landing.
Installed results and sandbox refusals are recorded in the lane return.
The ROM2 EN and RU roots cannot open this editor: `NewMapInspector` calls
`LoadTable`, whose `LoadDefinitions` uses the first game's layout. Both stop
at Armors row 4 before a choice opens. The shared ROM2 release exception
therefore does not apply. No ROM2 tree or editor support is claimed.
No window, original runtime or audio witness is claimed. Unfamiliar mod row
names have no automatic translation. No version changes on the branch.
