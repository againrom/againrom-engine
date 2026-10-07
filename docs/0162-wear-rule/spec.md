# 0162-wear-rule — a character cannot equip an item his class cannot use

Intensity: **spec-first**. The contract is one predicate and its two consumers; it is not expected
to keep evolving after this story.

## The defect

In this build a fighter hero equips mage items and a mage hero equips fighter items. The owner
reported it from play on 2026-08-15 and ruled it a defect. The original refuses such an equip.

## Terms

**sutableFor column** — parameter index 15 of an item's own `Data.bin` row, in the `Armors`,
`Shields` and `Weapons` collections. It is a runtime column number, that is, the parameter array's
own index. `pkg/mapload/spawn.go` already reads the same column under the name `weaponCarrySlot`.

**fighter bit**, **mage bit** — bit 0 and bit 1 of the sutableFor value.

**mage flag** — `mapload.PartyMember.Mage`, whether this character is a mage. It is already carried
to the map screen's inventory as `invPartyGear.mage` and to the shop as the party member's own
field.

**wear rule** — the predicate `(fighterBit && !mage) || (mageBit && mage)`.

**equip interaction** — the map screen's pack double-click, drained by `mapWorld.enqueueEquip`
(`pkg/game/world.go`), which is where a container index becomes a `sim.KindEquip` command.

**shop cell background** — which of the shop screen's cell background pictures a cell is drawn on
(`ui.ShopCellBack`).

## Requirements

**FR-1 The wear rule is one predicate over two item bits and one character bit.** An item whose
fighter bit is set may be used by a character whose mage flag is clear. An item whose mage bit is
set may be used by a character whose mage flag is set. An item with both bits set may be used by
either. An item with neither bit set may be used by neither. No slot, class or level takes part.

**FR-2 The item's two bits are read off the sutableFor column of the item's own collection row.**
The row is selected by the item code: field B names the collection (`1` weapons, `2` shields,
`3`..`12` armours, which are the equipment slots) and field D is the row index. The value is bit
tested as it stands; no ordering is read into it.

**FR-2a A row this build cannot read answers "unknown", and unknown does not refuse.** Three cases
answer unknown: a code whose field B names none of the three collections, a row index outside the
collection, and a nil collection. A row that exists but is shorter than sixteen cells answers the
zero value, that is, usable by neither. That split is the same shape `spawn.go`'s `carriable` and
`twoHanded` already use: a column that cannot be read does not apply, while a column that is read
applies as read.

**FR-3 The equip interaction refuses an unsuitable item.** `enqueueEquip` asks the wear rule as a
fourth question after its existing three. A refusal appends no command, so nothing moves: the item
stays in the pack at the same index, the equipment array is untouched, and no combat block is
re-derived. This matches the original's observable refusal, where the drop is rejected and the
drag ends with the item still held.

**FR-4 The shop screen greys a cell the shown party member cannot use.** A cell holding an item the
wear rule refuses is drawn on the grey background, which is the same picture an empty cell gets. A
shelf cell is drawn on the affordable background only when the purse covers the price **and** the
shown member can use the item. All three grids — shelf, table and pack — take the test, because the
original paints all three from one routine. The cell is still occupied: its icon, price and count
are drawn, and it can still be bought, sold and moved.

**FR-5 `pkg/sim` gains no rule.** The simulation still equips whatever it is told, at every slot,
for every entity. No entity field, no command field and no serialized byte changes.

## Disclosed divergence, in the player's units

The refusal is a client rule and nothing repairs a mismatch that arrives by another route. A
character loaded from a saved game may be wearing an item of the other class, and he keeps wearing
it: his statistics, his picture and his combat numbers all count it. A mission script that dresses a
character does the same. The original behaves this way too — its own refusal lives only in the
window, and its simulation applies no class rule at all. So the rule this story adds is "the game
will not let you put it on", not "such a combination cannot exist".

The refusal is silent. The item stops moving and nothing else happens: no message, no sound and no
change to the cursor. The original does something at that moment — its refusal path calls a method
on the screen object — but what that method draws or plays is not decoded, so this build refuses
without telling the player why. A player meeting it sees a double-click that does nothing.

Two further limits of this build, both unchanged by this story. The shop still sells an item the
shown member cannot use; buying it is permitted and it lands in the pack. And magic items, item
class 14, take no part: this build resolves no `MagicItems` row from a bare code, so a magic item is
neither refused at the equip interaction (which already declines to equip one at all) nor greyed in
a shop cell.

## Acceptance criteria

**AC-1** The predicate answers the full table: for sutableFor values 0, 1, 2 and 3 against a mage
flag of false and true, the eight answers are false, true, false, false, false, true, true, true.

**AC-2** Over a lawful install, every `Armors`, `Shields` and `Weapons` row's sutableFor value is
read, and the partition matches the shipped census: 30 armour rows split 19 fighter-only, 9
mage-only, 2 both; 9 shield rows all fighter-only; 27 weapon rows split 19 fighter-only, 2
mage-only, 3 both and 3 neither. Named rows are asserted individually: `Plate Cuirass` fighter-only,
`Robe` mage-only, `Ring` both, `Sonic Beam` neither.

**AC-3** A mage subject double-clicking a fighter-only pack item raises no command; the container
and the equipment array are byte for byte what they were.

**AC-4** The same subject double-clicking a mage-only item, and a fighter subject double-clicking a
fighter-only item, both enqueue the equip command they enqueued before this story.

**AC-5** A shop cell holding an item the shown member cannot use is drawn on the grey background and
reports itself occupied; the same cell for a member of the other class is drawn on the item
background, and on the affordable background when the purse covers it. An unusable shelf item is
never drawn affordable however large the purse.

**AC-6** A `sim.KindEquip` command applied directly to a world still equips a fighter-only item onto
a mage entity. The simulation is silent on class.

**P-1** `formatVersion` stays 46. No serialized byte moves and no digest changes.

**P-2** The predicate and both enforcement sites are witnessed without an install, over collections
built in the test. The census criterion AC-2 is the one that needs an install and it skips without
one.
