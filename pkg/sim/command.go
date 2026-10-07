package sim

// The typed way to build a Command.
//
// Command stays exactly the value it was: a kind and two numbers whose meaning
// that kind decides, which is what makes it a wire and a replay form. What
// changes is that nothing outside this file writes one. A literal lets X hold a
// cell, an entity id, a container element, an equipment slot, a spell id, a
// damage amount, a structure handle or a parameter selector with no way to tell
// which was meant; a constructor names it once, in the one place the reader on
// the far side is written down.
//
// EVERY CONSTRUCTOR HERE IS PURE FIELD ASSIGNMENT. None validates, clamps,
// defaults or reorders, and none may start: what a kind refuses is its arm's,
// decided inside the advance where an absent entity, an out-of-range slot and an
// unknown spell are already answers rather than errors. A constructor that
// changed a byte would move that decision to the caller's side of the queue and
// make a recorded stream mean something different from the one that produced it.
//
// The conversions are the arms' own, undone exactly where they are read. An
// entity id crossing in X is the same 32 bits under another name, so it is
// written as the arm reads it back and not widened, narrowed or checked on the
// way through.

// SpellID is a spell row id as a command carries one.
//
// It is int32 and not the uint16 a spell row is, because the field a spell
// rides in depends on the kind: KindCastAt carries it in Spell, which is
// uint16, while KindCast carries it in Y and KindAutocast in X, which are both
// int32. The autocast arm decides what an id too wide for a row MEANS - it
// clears the slot - so narrowing here would answer that question in the wrong
// place, and the narrowing that does happen happens exactly where the field is
// uint16 already.
type SpellID int32

// ItemSlot is an element of an entity's own container: the index a carried
// item is at. Books, potions and scrolls are named by one, and so is the item
// an equip order wears.
type ItemSlot int32

// EquipSlot is one of an entity's equipment slots, in the original's own 1..12
// numbering. It is NOT an ItemSlot: the two index different arrays, and telling
// them apart is most of what this file exists for.
type EquipSlot int32

// PlayerParameter is the sub-code of a player-parameter command: which setting
// the command addresses, not what it sets it to.
type PlayerParameter int32

// MoveTo orders the entity to walk to a cell.
func MoveTo(entity EntityID, at CellPoint) Command {
	return Command{Kind: KindMoveTo, Entity: entity, X: at.X, Y: at.Y}
}

func PickUp(entity EntityID, at CellPoint) Command {
	return Command{Kind: KindPickUp, Entity: entity, X: at.X, Y: at.Y}
}

// Kill is the debug kill: it leaves the historic shallow dead value behind.
func Kill(entity EntityID) Command {
	return Command{Kind: KindKill, Entity: entity}
}

// TerminalKill places the target at the first non-targetable health value,
// where Kill leaves a restorable body.
func TerminalKill(entity EntityID) Command {
	return Command{Kind: KindTerminalKill, Entity: entity}
}

// Damage subtracts an amount from the entity's health. The amount is a plain
// number and not an id: it is the one thing X carries that names nothing.
func Damage(entity EntityID, amount int32) Command {
	return Command{Kind: KindDamage, Entity: entity, X: amount}
}

// Attack sets the attacker's victim. The victim rides in X as the same 32 bits
// under another name, which is what the arm converts back.
func Attack(attacker, victim EntityID) Command {
	return Command{Kind: KindAttack, Entity: attacker, X: int32(uint32(victim))}
}

// AttackStructure is Attack over a building handle. It takes a StructureID and
// not an EntityID, so the two handles cannot be exchanged by accident - the one
// confusion a literal could not report, since both are 32 bits in X.
func AttackStructure(attacker EntityID, structure StructureID) Command {
	return Command{Kind: KindAttackStructure, Entity: attacker, X: int32(uint32(structure))}
}

// Cast orders a unit-target spell: the victim in X, the spell in Y.
func Cast(caster, victim EntityID, spell SpellID) Command {
	return Command{Kind: KindCast, Entity: caster, X: int32(uint32(victim)), Y: int32(spell)}
}

// CastAt orders a point-target spell: the cell in X and Y, the spell in Spell.
func CastAt(caster EntityID, spell SpellID, at CellPoint) Command {
	return Command{Kind: KindCastAt, Entity: caster, X: at.X, Y: at.Y, Spell: uint16(spell)}
}

// Autocast arms the entity's own repeating spell. Zero clears it, and so does
// any id the arm finds too wide for a row.
func Autocast(entity EntityID, spell SpellID) Command {
	return Command{Kind: KindAutocast, Entity: entity, X: int32(spell)}
}

// Equip wears the container item at item in equipment slot slot.
func Equip(actor EntityID, item ItemSlot, slot EquipSlot) Command {
	return Command{Kind: KindEquip, Entity: actor, X: int32(item), Y: int32(slot)}
}

// EquipDisplacing is Equip with a second equipment slot emptied into the
// container in the same atomic command. Slot zero names no second slot, which
// is what Equip passes.
func EquipDisplacing(actor EntityID, item ItemSlot, slot, displaced EquipSlot) Command {
	return Command{Kind: KindEquip, Entity: actor, X: int32(item), Y: int32(slot), Spell: uint16(displaced)}
}

// Unequip moves what an equipment slot holds back into the container.
func Unequip(actor EntityID, slot EquipSlot) Command {
	return Command{Kind: KindUnequip, Entity: actor, X: int32(slot)}
}

// DropCarried drops one unit of a container element on the ground.
func DropCarried(actor EntityID, item ItemSlot, at CellPoint) Command {
	return Command{Kind: KindDropCarried, Entity: actor, X: at.X, Y: at.Y, Spell: uint16(item)}
}

// DropWorn is DropCarried over an equipment slot instead of a container
// element. Both carry the index in Spell, which is why they take different
// types for it.
func DropWorn(actor EntityID, slot EquipSlot, at CellPoint) Command {
	return Command{Kind: KindDropWorn, Entity: actor, X: at.X, Y: at.Y, Spell: uint16(slot)}
}

// ReadBook consumes a carried book and teaches its spell to a living mage.
func ReadBook(actor EntityID, book ItemSlot) Command {
	return Command{Kind: KindReadBook, Entity: actor, X: int32(book)}
}

// UsePotion applies one carried potion to its owner.
func UsePotion(actor EntityID, potion ItemSlot) Command {
	return Command{Kind: KindUsePotion, Entity: actor, X: int32(potion)}
}

// UseScroll reads a carried scroll at a unit: the target in X, the carried
// index in Spell.
func UseScroll(actor EntityID, scroll ItemSlot, target EntityID) Command {
	return Command{Kind: KindUseScroll, Entity: actor, X: int32(uint32(target)), Spell: uint16(scroll)}
}

// UseScrollAt reads a carried scroll at a cell.
func UseScrollAt(actor EntityID, scroll ItemSlot, at CellPoint) Command {
	return Command{Kind: KindUseScrollAt, Entity: actor, X: at.X, Y: at.Y, Spell: uint16(scroll)}
}

// UseStructure sends the actor to use a building.
func UseStructure(actor EntityID, structure StructureID) Command {
	return Command{Kind: KindUseStructure, Entity: actor, X: int32(uint32(structure))}
}

// The group orders. Every one of them names ONE member and carries the tag
// that correlates it with the rest inside a single advance: a group order is a
// set of commands sharing a tag, so a caller builds one per member and gives
// them all the same tag. The tag is the queue's, not the world's - nothing
// stores it.

// GroupMoveTo walks the tagged group to a cell as a formation.
func GroupMoveTo(entity EntityID, at CellPoint, tag uint32) Command {
	return Command{Kind: KindGroupMoveTo, Entity: entity, X: at.X, Y: at.Y, Group: tag}
}

// GroupSwarmTo is GroupMoveTo under the Swarm order byte.
func GroupSwarmTo(entity EntityID, at CellPoint, tag uint32) Command {
	return Command{Kind: KindGroupSwarmTo, Entity: entity, X: at.X, Y: at.Y, Group: tag}
}

// GroupPatrolTo sets the tagged group patrolling between where each member
// stands and a cell.
func GroupPatrolTo(entity EntityID, at CellPoint, tag uint32) Command {
	return Command{Kind: KindGroupPatrolTo, Entity: entity, X: at.X, Y: at.Y, Group: tag}
}

// GroupStance puts the tagged group under a standing order. The order is
// OrderGuard or OrderStandGround; the arm reaches no other value, and this is
// the one group kind whose X is not a cell.
func GroupStance(entity EntityID, order int32, tag uint32) Command {
	return Command{Kind: KindGroupStance, Entity: entity, X: order, Group: tag}
}

// GroupDefend puts the tagged group on a protected actor, whose id rides in X
// where the other group kinds carry a cell.
func GroupDefend(entity EntityID, protect EntityID, tag uint32) Command {
	return Command{Kind: KindGroupDefend, Entity: entity, X: int32(uint32(protect)), Group: tag}
}

// GroupRetreat installs explicit Retreat on one member of the addressed
// player's selection. Player is a roster slot and not an entity id, which is
// why it is the only group order that reads that field.
func GroupRetreat(entity EntityID, player uint32, tag uint32) Command {
	return Command{Kind: KindGroupRetreat, Entity: entity, Player: player, Group: tag}
}

// SetPlayerParameter addresses a ROSTER SLOT rather than an entity: it is
// applied before entity lookup, so the player it names need not own an actor.
// The selector and the value are both int32-wide and adjacent, which is exactly
// the pair a literal could swap without a word from the compiler.
func SetPlayerParameter(player uint32, parameter PlayerParameter, value int32) Command {
	return Command{Kind: KindPlayerParameter, Player: player, X: int32(parameter), Y: value}
}

// DropGold asks the addressed player to put amount gold on the ground at a
// cell. The constructor assigns fields only; the application arm decides what
// is admitted.
func DropGold(player uint32, amount uint32, at CellPoint) Command {
	return Command{Kind: KindPlayerDropGold, Player: player, X: at.X, Y: at.Y, Group: amount}
}
