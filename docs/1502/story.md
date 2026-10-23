# One codec for equipment slots in SAV records

## Intent

An actor's SAV record keeps its equipment in three fields: `HeldWeapon`, `HeldShield` and the
twelve-reference `Worn` run. Slot 0 is the held weapon, slot 1 the held shield, slots 2 to 11 are
worn items, and a `Unit` record has no `Worn` field. Four writers and seven readers each spelled
that layout. One codec in `pkg/game/savequipmentslots.go` now states it, and every writer and
reader of the layout uses it. Where the sites disagreed, each keeps its present result through a
policy passed as data. Behaviour is unchanged: SAV bytes and World hashes equal main on the whole
corpus (Proof).

## Authority

- Owner decision on audit `pipeline/reviews/arch-audit-29d32509.md`, rank 6 (actor equipment
  slots in SAV records): one record-slot codec {class to fields, slot to field and ordinal,
  overlap policy}, used by every writer and reader; item record construction stays in
  `currentItemRecordValue` and `savedItemRecord`.
- Seat rule "One builder per kind". Refactor only; no ROM1 fact moves and no claim is cited.

## As built

### The codec

- `savEquipmentSlots`: one place {field, ordinal} per slot. Slot 0 is `HeldWeapon` ordinal 0,
  slot 1 `HeldShield` ordinal 0, slot n (2 to 11) `Worn` ordinal n. `savEquipmentSlotRef` reads
  it; `savEquipmentFieldSlot` is the inverse (a held field's slot, or a `Worn` position's slot).
  `savEquipmentFields` lists the three fields in write order.
- Overlap positions: `Worn` positions whose slot lives in a held field (0 and 1). Writers leave
  them empty; readers treat them under their `Low` policy.
- Record class: `savClassCarriesWorn` (every class but `Unit`). `savWornRule` says whether a site
  decides by class (`savWornByClass`) or by the record's present `Worn` field (`savWornByField`).
- `writeSavEquipment(refs, carriesWorn, set)`: sets each held field to its one ref and, when the
  record carries `Worn`, the twelve-position run with the overlap positions empty, in that order,
  through the site's own setter (`mustSetRefs` or `savedObjectSetRefs`).
- `readSavEquipment(record, policy, get)`: one ref per slot plus `Located` (the record has a
  position for the slot), or a `savEquipmentError` naming the fault (`Worn` shape, held shape,
  overlap). Each site turns the fault into its former error text. Policy fields: `Worn` (rule),
  `Held` (first ref of a nonempty field; only a one-ref field; one-ref field or fault), `Low`
  (ignored; fills a held slot without a nonzero ref; fills a held slot whose field is one zero ref;
  fills a zero held slot and faults beside a held ref; any nonzero is a fault), `High` (positions
  past slot 11 ignored; nonzero is a fault; a present run must have exactly twelve positions).

### Sites

Line numbers are at `18c85b63` (before) and at the candidate (after).

| Site | Before | After | Policy |
|---|---|---|---|
| City party writer `(*cityObjectProjection).holdings` | cityobjectprojection.go:496,530 | cityobjectprojection.go:496,530 | `cityProjectionWorn` = by field |
| Terminal body writer `currentTerminalWorn` | currentterminalbody.go:130 | currentterminalbody.go:130 | `terminalBodyWorn` = by class |
| Current item writer `projectCurrentItemGraphMode` | savcurrentitems.go:275 | savcurrentitems.go:275 | `currentItemsWorn` = by class |
| Saved item writer `savedCurrentActorItems` | savitemobjects_project.go:254 | savitemobjects_project.go:254 | `savedItemsWorn` = by class |
| City topology reader `captureCityObjectTopologyWithRecords` | cityobjecttopology.go:450 | cityobjecttopology.go:450 | `cityTopologyEquipment` |
| Slot mask `savedRecordWornMask` | currentterminal.go:254 | currentterminal.go:256 | `wornMaskEquipment` |
| Passive body `passiveRecordContents` | passivebody.go:30 | passivebody.go:30 | `passiveBodyEquipment` |
| Saved item graph `attachSavedItemObjects` | savitemobjects.go:470 | savitemobjects.go:470 | `savedItemEquipment` |
| City party reader `currentOrdinaryPartyState` | currentcityparty.go:243 | currentcityparty.go:243 | `cityPartyEquipment` |
| Current object capture `captureCurrentObjects` | currentobjects.go:248 | currentobjects.go:248 | slot place only |
| Retired actor item clearing `clearCurrentRetiredActorItems` | savcurrentitems.go:699 | savcurrentitems.go:694 | field-to-slot inverse |
| Item binding check in `validateSavedItemBindingWorldDecoded` | savitemobjects_project.go:470 | savitemobjects_project.go:463 | slot place; `itemBindingCheckWorn` = by class |
| Equipment strip in the same check, ownership field set | savitemobjects_project.go:488, savitemobjects.go:252 | savitemobjects_project.go:477, savitemobjects.go:252 | field list |
| Retained dead weapon `HeldWeapon` writes | currentdeadconstruct.go:70, currentterminalbody.go:178,188 | currentdeadconstruct.go:70, currentterminalbody.go:173,183 | field name |

### Sites outside the codec

- `pkg/formats/sav`: `program.go:362-376` declares the fields; `holdings.go:110-128` (typed held
  pieces and the twelve-slot check), `party.go:418` (all three fields as one list) and
  `dead.go:67,75` (equipment count and dead weapon) read the format's parsed `Record.Refs` view.
  The codec reads `sav.DocumentRecordData` in `pkg/game`, which `pkg/formats/sav` cannot import;
  moving these readers is a format-package change outside this story.
- `pkg/game/originalholdings.go:167-188` reads the typed `sav.Actor` pieces, checks each piece's
  class and refuses a piece at `Worn` position 0 or 1. It does not read record refs.
- `pkg/game/originaldead.go:151` reads `DeadActor.Worn`, a count, not slots.
- The city party writer's packing of slots 2 and up into `Inventory` for a record without `Worn`,
  and the city party reader's location list that reads them back (`currentcityparty.go`), are a
  city route of their own; the codec decides only whether the record carries `Worn`.

### Disagreements kept as data

Each is a present difference between sites, kept so every site keeps its result. Which behaviour
the original has is open; each question below is for research.

1. `Worn` on write. The city party writer writes `Worn` when the record already has the field; the
   other three writers write it for every class but `Unit`. Question: for an actor record whose
   class is `Unit`, does the original archive ever write the twelve-reference `Worn` run, and for a
   `Humanoid` record does it always write it?
2. `Worn` on read for a `Unit`. The passive body reader and the item binding check skip `Worn` on a
   `Unit` record; the other readers read a present `Worn` field whatever the class. Question: does
   the original's loader read worn items from a `Unit`-class record?
3. Overlap positions (`Worn` 0 and 1). City topology fills an empty held slot from them and refuses
   a held ref beside one; the slot mask counts either; the city party reader uses them only when the
   held field is one zero ref; the passive body reader, current object capture and item binding
   check ignore them; the saved item graph refuses any nonzero one. Question: when an original SAV
   record has a nonzero reference at `Worn` position 0 or 1, does the original load it as the held
   weapon or shield, ignore it, or refuse the file, and does a nonzero `HeldWeapon` beside it
   change that?
4. Held field shape. City topology and the saved item graph refuse a held field that is absent or
   has other than one ref; the passive body reader skips it; the slot mask and the city party reader
   take its first ref. Question: does every original actor record carry exactly one reference,
   possibly null, in each of `HeldWeapon` and `HeldShield`?
5. `Worn` length. City topology refuses a present run that is not twelve long; the saved item graph
   refuses a nonzero position past slot 11; the others ignore such positions. Question: does the
   original always write exactly twelve `Worn` references for a `Humanoid` record?

### Changes that no decoded or written record reaches

- `clearCurrentRetiredActorItems` took a `HeldWeapon` position k as slot k; the codec takes every
  `HeldWeapon` position as slot 0. The format decodes `HeldWeapon` as one object reference
  (`stpObjRef`) and every writer sets one ref, so no record has a second position.
- `attachSavedItemObjects` now checks both held fields' shape before it adds the held items. A
  record whose weapon add fails and whose shield field is malformed reports the shape error instead
  of the add error; both refuse the load.

## Proof

- Table tests, `pkg/game/savequipmentslots_test.go`: the slot place of every slot for `Unit`,
  `Humanoid` and `Human` and its inverse (`TestSavEquipmentSlotLayoutTable`); every site's
  policy (`TestSavEquipmentSitePolicyTable`); seven record shapes under the five reader policies,
  including each overlap, held-shape and length case (`TestReadSavEquipmentPolicies`);
  `Located` (`TestReadSavEquipmentLocated`); writer fields and order (`TestWriteSavEquipment`).
- SAV bytes and World hashes. Instrument: a temporary test (kept in the lane notes, not committed)
  loads each SAV through `RestoreOriginal`, writes it back through `ExportCurrentSave`, loads the
  written bytes again, and records the SHA-256 of the written bytes and the World hash after both
  loads. It ran on the candidate and on a `git archive` of main `18c85b63`, with the EN and the RU
  install, over `gameversions/saves` (146 files) and the converted milestone-2 corpus
  `review/story1257-ags-migration/out-en` and `out-ru` (114 files each). Per root: 97 map saves and
  44 town saves from the original corpus, 99 map and 15 town saves from the converted corpus, 5
  refused at load on both trees (4 written with a mod set, 1 not a SAV). The candidate's records
  equal main's byte for byte in all four populations: 0 differences in written bytes or World
  hash.
- Loss control. Slot 2's place changed to `HeldShield` ordinal 0 in `savEquipmentSlots`. Former
  sites fail: the current object capture (`current equipment binding is incomplete`:
  `TestInstanceWeightNonpartyBothLoadDoorsAndNewTransfer`, `TestActorRoots*`,
  `TestNativeItem*`, EN `TestReleaseDepartedUnplacedActorContentsSurviveSAV`), the saved item
  writer (`TestItemObjects1115HumanoidArmorSlotsExcludeHeldRefs`: held refs in the armour run), the
  city party writer and topology (`TestCityShopShieldTakesAWeaponOffAndBothNodesSurviveSAV`:
  unreachable objects; EN `TestReleaseShopStagedGoodsF2CurrentSAV/current_city`), and the three
  table tests. Restored from the commit, every one of those tests passes.
- Release sets through `pipeline/check-release-tests.sh` at `8b7fa648`, headless, EN and RU:
  1410 gated tests each, no failure; the one skip is `TestReleaseActivePauseRenderedWorld`. They
  include `passivebody_release_test.go`, `wornchange_release_test.go` and
  `shopsave_release_test.go`, unchanged.
- The ROM2 save groups (`pipeline/check-rom2-save-tests.sh`) did not run: a safety rule refused the
  lane copy of the script without its chain-ownership check. ROM2 shares these sites; that gate is
  owed.

## Open debt

- The five disagreements above stay until research answers them; the owner then picks one policy
  per question.
- The `pkg/formats/sav` readers keep their own spelling of the layout.
