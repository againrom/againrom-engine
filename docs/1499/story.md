# Items of a decayed body and a departed actor in SAV

## Intent

A SAV the engine writes, and the World after a cold LOAD of it, hold the worn and carried items of
two kinds of terminal actor: an original dead actor whose body has fully decayed (read from a loaded
file or decayed in play), and a departed actor with no map unit and no party entry. The items stay
passive: no loot appears, the body does not return to the live entities, and the digest of play
does not move.

## Authority

- `SAV-DEADLOAD-130` (High / Medium): a dead actor's record stores a container that a load
  restores in place; the record references the item objects the body still holds.
- `SAV-DEADLOAD-125` (High): a dead actor is rebound after load without joining a map unit.
- `SAV-DEADLOAD-126` (High): stage, signed health and timer are independent scalars.
- Owner rule in DIV-2501 and DIV-2505: every SAVE builds the document from the World; only
  unknown-meaning spans come from the loaded file.

## Before

- DIV-2505: an original terminal dead actor was written with its tuple, references, Diary and a
  Unit's loaded held weapon only. LOAD accepted its worn and carried records and dropped them.
- DIV-2501: a departed actor was written from its map placement or party entry with the
  placement's items in the slots its terminal row marked worn. One with neither got no record.
- `World.remove` disposed every item of the removed actor.

## As built

### Carrier (`pkg/sim/passivebody.go`, `passivebodybinary.go`)

- `sim.PassiveBody{ID, Body *Entity, Worn [12]ItemInstance, Carried []ItemStack}`, one row per
  terminal actor, sorted by ID, at most one per terminal row, carried at most 65535 stacks.
- `World.remove` calls `retainPassiveBody` after the terminal row is written and before the
  entity leaves the slices. A row is kept only for an owner that keeps a body (a current terminal
  row that is not `LeavesNoBody`, or an original dead row at a terminal stage) and only when it
  holds an item. `Body` is kept for a native actor with `MapUnitID` 0, without its target, attack,
  escort, pursuit and cast references.
- Items take the form their SAV record gives back: no registry or record identity, weight stated
  (`itemWeightOf` when absent), no weapon definition, no effects marker, a stack over 65535 split
  into adjacent stacks. LOAD applies the same form, so a SAVE then LOAD keeps the World bytes.
- A dead Unit whose one item at decay is its held weapon keeps nothing: its record would load that
  weapon as the dead row's hashed `HeldWeapon`, which the play capture does not set.
- Byte form: outermost form wrapper 127, tag `PBC1`, a JSON section written only when a row
  exists; old byte forms decode with no rows. `World.Hash` digests the bytes without this section
  (`encodePlayInto`), so no hash moves.
- `RestorePassiveBodies` replaces rows by ID, removes a row given no items, and refuses (changing
  nothing) a live ID, a missing owner, a body on a placed or source-bound row, or an item not in
  record form. `RestoreActorIdentities` remaps row and body IDs.

### SAVE (`ExportCurrentSave` route)

- Original dead records (`appendAbsentDeadRecords`, `pkg/game/currentdeadconstruct.go`):
  `passiveDeadContents` writes the kept worn items to `HeldWeapon`, `HeldShield` and `Worn`, and
  the carried stacks to `Inventory`, each as a fresh item record with its Effect and Spell records.
  A held weapon the record keeps from the loaded file stays that record's.
- Departed records (`currentTerminalBodies`, `pkg/game/currentterminalbody.go`): a kept row
  replaces the placement's masked items; `currentTerminalCarried` writes the carried stacks. A
  departed actor with neither a placement nor a party entry is built from the kept `Body`.
- The native action leaf carries `DepartedBodies`, the kept bodies in ID order; old files have
  none.

### LOAD

- `applyOriginalDeadWithOrigins` (`pkg/game/originaldead.go`) reads the items each terminal dead
  record references (`passiveRecordContents`) and calls `RestorePassiveBodies`. A Unit whose held
  weapon becomes the row's `HeldWeapon` contributes its other items only.
- `restoreCurrentActorIdentities` (`pkg/game/savactoridentity.go`) calls
  `restoreDepartedContents`: each departed row with a body takes the items its record references
  and, without a map unit, the body from `DepartedBodies`.

## Proof

See the lane result for the run record. Witnesses:

- `TestReleaseTerminalHumanOriginalResaveContents` (owner file `game0005-original-quest-done.sav`,
  4 terminal Humans with worn items): LOAD keeps the items, two menu SAVE and cold LOAD cycles
  write the same items and keep the World equal.
- `TestReleaseDepartedUnplacedActorContentsSurviveSAV` (mission 20): a constructed witness. No
  shipped ground actor departs with items: death drops every item except an innate weapon, and in
  the installed missions only flyers hold one, which leave no body. A ground native actor without
  its map unit is given a flyer's innate weapon and, after its loot dropped, one stack from a hero's
  pack; SAVE, cold LOAD, SAVE, cold LOAD keep the body, the items and the records.
- `TestTerminalHumanContentsSurviveSAVColdLoadAndTicks` (fixture): worn, held and carried items
  with item Effects survive two cycles and 65 ticks without loot, rewards or effects.
- `pkg/sim/passivebody_test.go`: capture, digest exclusion, byte form, restore boundaries,
  identity remap, record form.

## Open debt

- DIV-3953: each kept item is a fresh record on every SAVE; item identity and sharing between
  slots or records are not kept; a weapon's definition values are not kept.
- DIV-2505: a dead actor's own Effects are written only on the existing retained-effects route; a
  dead Unit captured in play whose one item is its held weapon is written without it.
- DIV-2501: a departed actor's own Effects are not written; one with no map unit, no party entry
  and no items still gets no record.
