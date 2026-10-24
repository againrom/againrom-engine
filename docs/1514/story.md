# Cell spells in the map editor

## Intent

List type-9 traps, unit spells and casting structures. Edit only fields whose original-editor writes are claimed. Keep unknown fields and kinds unchanged.

## Authority

ALM-EFFREC-071 frames each record. ALM-EDITCAST-294 through ALM-EDITCAST-297 name the three dialogs and their writes. ALM-EDITLINK-298 names deletion renumbering. ALM-T9CONTROL-175, ALM-T9ACTOR-176, ALM-T9TAG-312 and UNIT-M10CELL-054 distinguish the consumer arms. TRIG-CELLTAIL-035 includes its retraction. ALM-SAVE-253, ALM-SAVE-254 with its amendment, ALM-SAVE-255, ALM-SAVE-260, ALM-SAVE-268 and ALM-CORP-315 bound installed preservation and object identities; UNIT-T9CTOR-110 and UNIT-T9LIFE-111 do not prove autonomous caster lifetime.

## As built

The existing type-9 field codec supplies the existing inspection popup. The existing Add window accepts a known spatial template, an object id, a cell, spell and power. The codec preserves opaque tag and tail bytes from that template. Trap Add clears its object id before later field application. Unit and structure ids and coordinates remain read-only after placement; their new entries require a unique matching object at the anchor cell through the existing reference validator. For a casting structure already recorded at that cell with that id, Add updates its spell and power, keeps its other bytes and opens its popup; otherwise it appends. Unit spell entries keep appending. F3 opens this Add window; E edits and Delete removes a known record. Unknown coordinate-bearing kinds are listed read-only. Cell-spell deletion follows the last loaded back-reference, including duplicate-link retention. Existing item recipes retain DIV-3380's separate all-link preservation policy in the same codec.

The record list's Add button on filter 5 opens the same cell-spell Add window.

Setting any known cell spell to None (spell 0) removes that record immediately; undo restores all bytes. Add refuses None with a reason. Owner direction also removes linked entries and entries at x=0, y>0: these do not meet ALM-SAVE-254's save-removal predicate (nonzero x, no loot backlink, spell 0). This policy applies on edit; unchanged maps still retain their bytes. No divergence row is requested.

## Proof

Focused byte tests cover every known kind, template placement, opaque words, invalid powers, history, later item-link renumbering and last-loaded duplicate back-references. Editor tests cover Key and pointer Apply; native Update reaches F3, AddRecord and Cancel. A static guard pins the inspection row's one builder.

Correction witnesses require Add to update a non-last casting-structure record without copying the template's opaque bytes, preserve the count, open the existing popup and undo in one step. Trap, unit and structure None witnesses cover pointer Apply, linked records, exact undo and refusal through both the Add host and Add popup.

The installed field witness preserves all 38 EN maps with 792 type-9 records and all 34 RU maps with 334 records. Nine selected release tests per root cover byte preservation, owner and F1 frames, the rail width, new-map F5 and an editor-placed trap. The trap witness exports with F5, opens that exported mod through the game's normal mission loader and observes Lightning at cell (24,63), event tick 54, on both roots. No external game process starts. All 22 frames per root match the reviewed candidate.

Moving the power write from offset 20 to 18 fails the exact-record witness. Disabling link renumbering fails both deletion witnesses. Restoring the codec from the local implementation commit restores the ordinary mapedit pass. The correction's final fast gate takes 65 seconds. Ordinary game tests pass with fixed shuffle seed 1; two normal-order runs fail only a message-wizard temporary-file rename with Windows Access is denied, while that named test alone passes. Complete gate receipts and frame hashes are in the lane notes.

## Open debt

This is a bounded implementation. Creation without a spatial template was stopped: no claim defined fresh editor allocation bytes for opaque fields. Scan-range editing was stopped: the spinner range was known but the complete data-exchange bounds were not. The anchor-cell rule refused most shipped casting structures. Story 1517 settles all three on k353 (`docs/1517/story.md`). No original-runtime activation or lifetime claim is added. No reserved divergence is used.
