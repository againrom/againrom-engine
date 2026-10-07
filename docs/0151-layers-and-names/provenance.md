# 0151 — provenance

Research pin: `research` submodule at `de6a080`. Claim rows were checked at that pin.

## Backing

| Spec anchor | Claim | Confidence | What is taken from it |
|---|---|---|---|
| FR-1 | `HERO-FIGURE-058` | High | A compositor stencil value is the equipment slot number. Primary and secondary drawable index `k` both represent equipment slot `k+1`. |
| FR-1 | `HERO-FIGURE-059` | High | Program order is paint order. The non-mage primary order translates to slots 12, 11, 7, 4, 5, 9, 10, 8 and 6 before the held layers. Slot 3 is absent. The mage order differs and has a head-layer position this build does not reproduce. |
| FR-1 | `HERO-FIGURE-060` | High for the six-name predicate and swap; Medium for calling the two indices weapon and shield | The slot-1 layer paints last for body names `bowman`, `archer`, `xbowman`, `axeman2h`, `swordsman2h` and `mage_st`; otherwise slot 2 paints last. |
| FR-1 | `HERO-APPEAR-050` | High for field B and the secondary-slot set; Unknown for what distinguishes two slot-1 builders | Secondary sheets exist for equipment slots 4, 8, 9 and 10. Primary sheets occur for slots 1, 2, 4–10 and 12. |
| FR-1 | `HERO-APPEAR-051` | High for the compositor routine | All twelve visible-equipment fields are inspected. An occupied field contributes a sheet under one of four figure directories; selected fields may contribute a secondary sheet. |
| FR-1 | `HERO-APPEAR-052` | High for the list load and contents; Unknown past the list end | Slot 1's body name comes from `main/text/heropicture.txt`, indexed by item row minus one. `bowman` is not present in the shipped list. |
| FR-1 | `ITEM-ARMSLOT-031` | High for the armour slot column and indexing; Medium for the corpus absence of slots 1, 2, 3 and 11 | An armour row's `Slot` column selects the worn field. The shipped rows occupy slots 4–10 and 12, excluding 11. |
| FR-1 | `ITEM-APPEAR-024` | High | Item art is addressed by the item's seven-digit code. The inventory icon tree and the figure-sheet tree are distinct. |
| FR-1 | `HERO-APPEAR-051` | High | The base figure path uses the sex/mage directory and face byte; this is the canvas the equipment sheets augment. |
| FR-3 | `ITEM-DISPNAME-036` | High for table construction and lookup; Medium for the onward interface path | The displayed item name is a stored `itemname.txt` line selected by a `u16` key from `itemname.bin`, not a composed data-row name. The original build does not bound keys against a shorter line array. |
| FR-3 | `ITEM-NAMEKEY-037` | High | The lookup key is the packed item code. Its material, class, shape and row cuts are the same cuts used to construct an item. |
| FR-3 | `ITEM-NAMEPOP-038` | High for shipped counts and encoding; Medium for the 5,648 reachable unnamed-key count | Both roots carry 416 keys and lines. Most codes expressible by the encoding have no stored line, and the stored names cannot be reproduced as a field product. |
| FR-3 | `TEXT-ITEMNAME-019` | High | EN and RU carry different 416-line text corpora and the same key file. RU lines contain high bytes and are longer in bytes than EN lines. |
| FR-3 | `TEXT-CONV-001` | High for the converter and byte map; the former injectivity clause was High and is retracted | Selector 1 maps source bytes `0x80..0xAF` to `0xB0..0xDF` and `0xE0..0xEF` to `0xF0..0xFF` immediately before glyph indexing. The map is not injective: 64 output values have two preimages. The implementation relies only on the two moves and the per-byte call, not on the retracted clause. |
| FR-3 | `TEXT-LANG-002` | High | The selector comes from the trailing digit of `main/id`: EN selects 0 and RU selects 1. One executable serves both data sets. |
| FR-3, P-3 | `TEXT-CAP-018` | High | The font subscript is byte-wide. Selector 1 can distinguish 160 glyphs; other selectors can distinguish 224 shipped records. Lifting the selector-1 aliasing changes every shipped Russian byte's meaning and requires rewriting text. |

## Ours by choice

| Spec anchor | Choice |
|---|---|
| FR-1 | Slot 3 has no decoded paint position. The build places it immediately before the two held slots. |
| FR-1 | A secondary sheet is painted immediately after its slot's primary sheet. The decoded compositor interleaves some secondary sheets later; this build preserves the paired part and one shared slot order rather than duplicating the full mage/non-mage programme. |
| FR-3 | A missing or short name table is non-fatal. Parsing stops at the shorter input, drops empty lines, and lookup falls back first to a resolvable weapon name and then to the code's seven digits. The original's out-of-range line read is not copied. |
| FR-4 | The popup uses the inventory frame colours, a one-pixel border, three pixels of interior clearance and a fourteen-pixel cursor offset. It is clamped to the view and drawn over every other layer. |
| FR-4 | Only decoded characteristics are stated. A shield and a class-14 magic item receive no invented numeric line. |
| FR-5 | Unequipping appends to the unbounded container and uses its existing fold rule, so an equal carried code gains one count. |

## Open

| Spec anchor | Open question and shipped answer |
|---|---|
| FR-1 | `HERO-FIGURE-059` decodes a separate mage order: the head is inserted among equipment layers and mage slot 9 is tagged but not colour-blitted. This build uses the common base-first compositor for mages. Equipment that should lie behind a mage's head can cover it, and slot-9 equipment can be visible where the original would leave it invisible. |
| FR-1 | `HERO-APPEAR-052` grades an index past `heropicture.txt` Unknown. This build treats an unresolved body name as not matching the six-name predicate and paints slot 2 last. |
| FR-4 | No published claim supplies shield characteristics or a characteristic mapping for class 14. Their popup has a name line only. |
| FR-1 | `heropicture.txt` is 248 bytes and 26 lines, of which the 23rd is empty and the other 25 carry a name. Measured at the merge from both preserved installs with this repository's own container reader. `HERO-APPEAR-052`'s index map skips the empty line, so its indices 22, 23 and 24 are `Sonic Beam`, `Flame Thrower` and `swordsman`; this build's reader keeps the empty line as an entry, so the same three names sit at 23, 24 and 25 and index 22 is an empty entry the body derivation already refuses. The two readings agree on indices 0 through 21, which is where every name in the six-name predicate sits, so no figure this build composes can differ from the decoded order because of it. Whether the original's list loader keeps or drops an empty line is not established. |

## Removed

| Statement removed | Why |
|---|---|
| Slot 1 always paints last. | It was an authored first answer. `HERO-FIGURE-060` supplies the conditional slot-1/slot-2 swap, and the owner chose the decoded answer. |
| A crossing unit always keeps its origin depth row. | It fixes northward movement but regresses southward movement. The shipped rule uses the later row of the two cells the crossing spans. |
| The selector-1 converter is injective over high bytes. | The clause in `TEXT-CONV-001` is retracted. Unmoved bytes already occupy both destination blocks, creating 64 collision pairs. No implementation statement depends on injectivity. |
