package sim

import "fmt"

// WeaponSpellSource is the persisted provenance tag for Entity's cached
// weapon-spell pair. The numeric values are format 61's wire values.
type WeaponSpellSource uint8

const (
	WeaponSpellNone WeaponSpellSource = iota
	WeaponSpellItem
	WeaponSpellInnate
	WeaponSpellLegacy
)

func (s WeaponSpellSource) valid() bool { return s <= WeaponSpellLegacy }

// SecondaryDamage is the one active item-effect damage triple. Selector is
// the protection school: Fire through Astral are 0 through 4. Effect kinds
// 44..48 replace this whole value in stored effect order; they are not five
// independent components.
type SecondaryDamage struct {
	Base, Spread uint8
	Selector     uint8
}

func secondaryDamageFault(d SecondaryDamage) error {
	if d.Selector >= 5 {
		return fmt.Errorf("secondary-damage selector %d is not defined", d.Selector)
	}
	return nil
}

// ItemEffect is one ordered effect attached to an item instance. The three
// fields reproduce the persisted Effect value established by ITEM-EFFOBJ-072
// and ITEM-EFFSAVE-077. Order and duplicates are significant.
type ItemEffect struct {
	Kind    uint8
	Mode    uint8
	Operand uint32
}

// ItemInstance is one item object. Price is the signed value stored by the
// producer. Kind is the original Item+0x44 byte used by the stackability rule;
// value 3 identifies Potion objects. Effects is always owned by this value's
// container and must be deep-copied at transfer boundaries.
type ItemInstance struct {
	ObjectID SavedObjectID
	Code     uint16
	Kind     uint8
	Effects  []ItemEffect
	Price    int32
	// Weight is the signed per-unit Item+0x4a value (ITEM-STACK-003).
	// WeightPresent distinguishes explicit zero from native code-table fallback.
	Weight          int16
	WeightPresent   bool
	SourceEquipment SourceEquipment
	NativeRecord    *NativeItemRecord
}

// NativeItemRecord travels with an unregistered item after its ordinary
// record is observed. Gameplay fields and child values keep their own owners.
type NativeItemRecord struct {
	Class         uint8
	Token         SavedObjectToken
	F45, F46, F47 uint8
	F48           uint16
}

func cloneNativeItemRecord(v *NativeItemRecord) *NativeItemRecord {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}

func nativeItemRecordEqual(a, b *NativeItemRecord) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// PlainItem constructs the form-60 and code-only producer representation.
func PlainItem(code uint16) ItemInstance { return ItemInstance{Code: code} }

// ValidateWeight checks the optional signed value's canonical representation.
func (i ItemInstance) ValidateWeight() error {
	if err := i.SourceEquipment.Validate(); err != nil {
		return err
	}
	if i.Code == 0 && (i.SourceEquipment.Class != 0 || i.ObjectID != 0) {
		return fmt.Errorf("sim: empty item has source equipment or object identity")
	}
	if i.NativeRecord != nil && (i.Code == 0 || i.ObjectID != 0 || i.NativeRecord.Class > SourceShield || i.NativeRecord.Token.T1C != 0) {
		return fmt.Errorf("sim: native Item record has no unregistered owner: code %04x, id %d, class %d, price word %d", i.Code, i.ObjectID, i.NativeRecord.Class, i.NativeRecord.Token.T1C)
	}
	return itemWeightFault(i.Code, i.WeightPresent, i.Weight)
}

// Clone returns a deep copy. A caller cannot mutate the source through the
// returned Effects slice.
func (i ItemInstance) Clone() ItemInstance {
	i.Effects = append([]ItemEffect(nil), i.Effects...)
	i.NativeRecord = cloneNativeItemRecord(i.NativeRecord)
	return i
}

// Empty reports the fixed-slot sentinel. An empty instance has no residue.
func (i ItemInstance) Empty() bool {
	return i.ObjectID == 0 && i.Code == 0 && i.Kind == 0 && i.Price == 0 && len(i.Effects) == 0 && !i.WeightPresent && i.Weight == 0 && i.SourceEquipment == (SourceEquipment{}) && i.NativeRecord == nil
}

// HasEnchantment is the single read-only seam for presentation consumers.
// It deliberately answers only whether the ordered effect list is non-empty;
// stored value alone does not make an item enchanted.
func (i ItemInstance) HasEnchantment() bool { return len(i.Effects) != 0 }

func (i ItemInstance) innateWeapon() bool {
	return i.SourceEquipment.Class == SourceWeapon && i.SourceEquipment.Definition.Present && i.SourceEquipment.Definition.Suitable == 0
}

// itemEffectsEqual compares the persisted Effect identity in stored order.
func itemEffectsEqual(a, b []ItemEffect) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

// sameRetainedOperands compares two retained operand sets for a merge. Weapon
// W52 bytes 22 and 23, Item +0x68 and +0x69, take no part: the owner decided
// that equipment differing only there is one item to the player (DIV-762). A
// merge keeps the destination's two bytes and deletes the incoming Item with
// its own (ITEM-MERGE-129).
func sameRetainedOperands(a, b SourceEquipment) bool {
	a.Attack[22], a.Attack[23] = 0, 0
	b.Attack[22], b.Attack[23] = 0, 0
	return a == b
}

// ItemEqual implements ITEM-STACK-003 and ITEM-EFFOBJ-072, with the
// owner-retention Potion exception recorded as DIV-369. Price is not part of
// ROM1 item equality. It is compared only for two Potion objects so merging
// cannot discard an imported effect or transaction value.
func ItemEqual(a, b ItemInstance) bool {
	// DIV-754 and DIV-762 retain distinct saved weight/equipment operands
	// instead of merging one source value away.
	if a.Code != b.Code || a.WeightPresent != b.WeightPresent || a.Weight != b.Weight || !sameRetainedOperands(a.SourceEquipment, b.SourceEquipment) {
		return false
	}
	if a.Kind == 3 || b.Kind == 3 {
		return a.Kind == 3 && b.Kind == 3 && a.Price == b.Price && itemEffectsEqual(a.Effects, b.Effects)
	}
	aStackable := !a.HasEnchantment()
	bStackable := !b.HasEnchantment()
	if aStackable || bStackable {
		return aStackable && bStackable
	}
	return itemEffectsEqual(a.Effects, b.Effects)
}

// CanMergeItemValues is the current container's scalar retention predicate.
// Node identity and child lifetime belong to the caller's explicit graph.
func CanMergeItemValues(a, b ItemInstance) bool {
	return ItemEqual(a, b) && a.Price == b.Price && a.Kind == b.Kind && nativeItemRecordEqual(a.NativeRecord, b.NativeRecord)
}

// JoinForm is incoming written in held's stored form, and whether that
// rewrite loses nothing. ITEM-STACK-003 compares neither the per-unit weight
// nor the definition blocks, and ITEM-MERGE-129 keeps the destination with its
// own weight. A value that is plain, or exactly what construct makes of its
// code, holds no saved operand, so it may take the form of a held item of the
// same constructed kind. The caller compares held with the rewritten value.
// construct fills a plain value's blocks and weight.
func JoinForm(held, incoming ItemInstance, construct func(ItemInstance) ItemInstance) (ItemInstance, bool) {
	if incoming.WeightPresent == held.WeightPresent && incoming.Weight == held.Weight && sameRetainedOperands(incoming.SourceEquipment, held.SourceEquipment) {
		return incoming, true
	}
	plain := incoming.Clone()
	plain.Weight, plain.WeightPresent, plain.SourceEquipment = 0, false, SourceEquipment{}
	built := construct(plain.Clone())
	if (incoming.WeightPresent || incoming.SourceEquipment.Class != 0) && (!ItemEqual(built, incoming) || built.Kind != incoming.Kind) {
		return incoming, false
	}
	if built.Kind != held.Kind {
		return incoming, false
	}
	plain.Weight, plain.WeightPresent, plain.SourceEquipment, plain.Kind = held.Weight, held.WeightPresent, held.SourceEquipment, held.Kind
	return plain, true
}

// CastSpell returns the first ordered kind-41 effect as spell id and signed
// power. ITEM-CASTLINK-076 defines the low word as id and the high word as
// signed i16 power.
func (i ItemInstance) CastSpell() (uint16, int32, bool) {
	for _, e := range i.Effects {
		if e.Kind == 41 {
			return uint16(e.Operand), int32(int16(e.Operand >> 16)), true
		}
	}
	return 0, 0, false
}

// BookSpell returns the one spell taught by a readable book. Kind 5 is the
// decoded Book object kind; a book with no teach effect, more than one teach
// effect, or an out-of-mask spell id is not a readable one-spell book.
func (i ItemInstance) BookSpell() (uint16, bool) {
	if i.Kind != 5 {
		return 0, false
	}
	var spell uint16
	found := false
	for _, e := range i.Effects {
		if e.Kind != 42 {
			continue
		}
		if found || e.Operand == 0 || e.Operand >= 32 {
			return 0, false
		}
		spell, found = uint16(e.Operand), true
	}
	return spell, found
}

func weaponPairPresent(e Entity) bool {
	return e.WeaponSpell != 0 || e.WeaponSpellLevel != 0
}

func normaliseWeaponSources(entities []Entity, equipment [][EquipSlots]ItemInstance) error {
	for i := range entities {
		spell, power, itemCast := equipment[i][slotWeapon].CastSpell()
		e := &entities[i]
		switch {
		case itemCast && !weaponPairPresent(*e) && e.WeaponSpellSource == WeaponSpellNone:
			e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource = spell, power, WeaponSpellItem
		case itemCast && e.WeaponSpellSource == WeaponSpellNone:
			if e.WeaponSpell != spell || e.WeaponSpellLevel != power {
				return fmt.Errorf("entity %d weapon spell pair disagrees with its equipped item", e.ID)
			}
			e.WeaponSpellSource = WeaponSpellItem
		case weaponPairPresent(*e) && e.WeaponSpellSource == WeaponSpellNone:
			e.WeaponSpellSource = WeaponSpellLegacy
		}
		if err := validateWeaponSource(*e, equipment[i][slotWeapon]); err != nil {
			return fmt.Errorf("entity %d: %w", e.ID, err)
		}
	}
	return nil
}

func validateWeaponSource(e Entity, weapon ItemInstance) error {
	spell, power, itemCast := weapon.CastSpell()
	if !e.WeaponSpellSource.valid() {
		return fmt.Errorf("weapon-spell source %d is not defined", e.WeaponSpellSource)
	}
	switch e.WeaponSpellSource {
	case WeaponSpellNone:
		if weaponPairPresent(e) || itemCast {
			return fmt.Errorf("weapon-spell source None retains a spell")
		}
	case WeaponSpellItem:
		if !itemCast || e.WeaponSpell != spell || e.WeaponSpellLevel != power {
			return fmt.Errorf("weapon-spell source Item does not match the equipped item")
		}
	case WeaponSpellInnate, WeaponSpellLegacy:
		if !weaponPairPresent(e) {
			return fmt.Errorf("weapon-spell source %d has a zero pair", e.WeaponSpellSource)
		}
		if itemCast {
			return fmt.Errorf("weapon-spell source %d conflicts with an equipped item spell", e.WeaponSpellSource)
		}
	}
	return nil
}

func syncWeaponItem(e *Entity, weapon ItemInstance) {
	if spell, power, ok := weapon.CastSpell(); ok {
		e.WeaponSpell = spell
		e.WeaponSpellLevel = power
		e.WeaponSpellSource = WeaponSpellItem
	} else if e.WeaponSpellSource == WeaponSpellItem {
		e.WeaponSpell = 0
		e.WeaponSpellLevel = 0
		e.WeaponSpellSource = WeaponSpellNone
	}
}

// applyEquipmentItemState performs the state-0 consumers that are not part of
// a derived-sheet recompute. HealthMax and ManaMax still travel through
// pkg/data; their matching current-pool arms move here at the same equipment
// boundary. teachSpell is equip-only: removing the item never unlearns it.
func applyEquipmentItemState(e *Entity, removed, added ItemInstance, table []SpellRule, observation ...*damageObservation) {
	before := *e
	mage := isMage(*e)
	health, mana := itemPoolDelta(added, mage)
	oldHealth, oldMana := itemPoolDelta(removed, mage)
	e.setCurrentHealth(e.HP + health - oldHealth)
	// Only the direct health arm can wound. A removed maximum-health lift that
	// clamps the current health is a smaller pool, not a blow.
	if len(observation) != 0 && itemDirectHealthDelta(added) < itemDirectHealthDelta(removed) {
		observation[0].loss(before, *e)
	}
	e.setCurrentMana(e.Mana + mana - oldMana)
	if mage {
		learnItemSpells(e, added, table)
	}
}

// itemDirectHealthDelta is the item's health-arm total, kind 6 alone.
func itemDirectHealthDelta(item ItemInstance) (health int32) {
	for _, effect := range item.Effects {
		if effect.Kind == 6 {
			health += itemEffectScalar(effect)
		}
	}
	return health
}

func learnItemSpells(e *Entity, item ItemInstance, table []SpellRule) {
	for _, effect := range item.Effects {
		if effect.Kind != 42 {
			continue
		}
		spell := uint16(effect.Operand)
		LearnBookSpell(e, spell, table)
	}
}

func itemPoolDelta(item ItemInstance, mage bool) (health, mana int32) {
	for _, effect := range item.Effects {
		scalar := itemEffectScalar(effect)
		switch effect.Kind {
		case 6, 7:
			health += scalar
		case 9, 10:
			if mage {
				mana += scalar
			}
		}
	}
	return health, mana
}

func itemEffectScalar(effect ItemEffect) int32 {
	if effect.Mode == 0 || effect.Mode == 8 {
		return int32(effect.Operand)
	}
	return int32(int16(effect.Operand))
}

// ItemStack is one container cell. Its instance fields are flattened to keep
// the longstanding Code/Count API usable by display callers; Instance is the
// only conversion back to the canonical object.
type ItemStack struct {
	ObjectID        SavedObjectID
	Code            uint16
	Kind            uint8
	Effects         []ItemEffect
	Price           int32
	Count           uint32
	Weight          int16
	WeightPresent   bool
	SourceEquipment SourceEquipment
	NativeRecord    *NativeItemRecord
}

// StackItem constructs a container cell without exposing the embedding detail
// to callers that already have a complete instance.
func StackItem(item ItemInstance, count uint32) ItemStack {
	item = item.Clone()
	return ItemStack{ObjectID: item.ObjectID, Code: item.Code, Kind: item.Kind, Effects: item.Effects, Price: item.Price, Count: count, Weight: item.Weight, WeightPresent: item.WeightPresent, SourceEquipment: item.SourceEquipment, NativeRecord: item.NativeRecord}
}

// PlainStack is the compatibility constructor for code-only producers.
func PlainStack(code uint16, count uint32) ItemStack {
	return StackItem(PlainItem(code), count)
}

// Clone returns a deep copy of the stack and its item instance.
func (s ItemStack) Clone() ItemStack {
	s.Effects = append([]ItemEffect(nil), s.Effects...)
	s.NativeRecord = cloneNativeItemRecord(s.NativeRecord)
	return s
}

// Instance returns the complete object represented by this cell.
func (s ItemStack) Instance() ItemInstance {
	return ItemInstance{ObjectID: s.ObjectID, Code: s.Code, Kind: s.Kind, Effects: append([]ItemEffect(nil), s.Effects...), Price: s.Price, Weight: s.Weight, WeightPresent: s.WeightPresent, SourceEquipment: s.SourceEquipment, NativeRecord: cloneNativeItemRecord(s.NativeRecord)}
}

// StackStateEqual compares the complete persisted cell, including the stored
// price which ItemEqual deliberately ignores outside Potion stacking.
func StackStateEqual(a, b ItemStack) bool {
	return a.ObjectID == b.ObjectID && a.Code == b.Code && a.Kind == b.Kind && a.Price == b.Price &&
		a.Count == b.Count && a.Weight == b.Weight && a.WeightPresent == b.WeightPresent && a.SourceEquipment == b.SourceEquipment && itemEffectsEqual(a.Effects, b.Effects) && nativeItemRecordEqual(a.NativeRecord, b.NativeRecord)
}

func cloneItems(in []ItemInstance) []ItemInstance {
	if len(in) == 0 {
		return nil
	}
	out := make([]ItemInstance, len(in))
	for k := range in {
		out[k] = in[k].Clone()
	}
	return out
}

func cloneStacks(in []ItemStack) []ItemStack {
	if len(in) == 0 {
		return nil
	}
	out := make([]ItemStack, len(in))
	for k := range in {
		out[k] = in[k].Clone()
	}
	return out
}

func cloneEquipment(in [EquipSlots]ItemInstance) [EquipSlots]ItemInstance {
	var out [EquipSlots]ItemInstance
	for k := range in {
		out[k] = in[k].Clone()
	}
	return out
}

func plainItems(codes []uint16) []ItemInstance {
	if len(codes) == 0 {
		return nil
	}
	out := make([]ItemInstance, 0, len(codes))
	for _, code := range codes {
		if code != 0 {
			out = append(out, PlainItem(code))
		}
	}
	return out
}

func rawPlainItems(codes []uint16) []ItemInstance {
	out := make([]ItemInstance, len(codes))
	for k, code := range codes {
		out[k] = PlainItem(code)
	}
	return out
}

func itemCodes(items []ItemInstance) []uint16 {
	if len(items) == 0 {
		return nil
	}
	out := make([]uint16, len(items))
	for k := range items {
		out[k] = items[k].Code
	}
	return out
}

func plainEquipment(codes [EquipSlots]uint16) [EquipSlots]ItemInstance {
	var out [EquipSlots]ItemInstance
	for k, code := range codes {
		if code != 0 {
			out[k] = PlainItem(code)
		}
	}
	return out
}

func equipmentCodes(items [EquipSlots]ItemInstance) [EquipSlots]uint16 {
	var out [EquipSlots]uint16
	for k := range items {
		out[k] = items[k].Code
	}
	return out
}

func equipmentItemsEmpty(items [EquipSlots]ItemInstance) bool {
	for _, item := range items {
		if !item.Empty() {
			return false
		}
	}
	return true
}

func itemHasMetadata(item ItemInstance) bool {
	return item.ObjectID != 0 || item.Kind != 0 || item.Price != 0 || len(item.Effects) != 0 || item.WeightPresent || item.Weight != 0 || item.SourceEquipment != (SourceEquipment{}) || item.NativeRecord != nil
}

func itemsHaveMetadata(items []ItemInstance) bool {
	for _, item := range items {
		if itemHasMetadata(item) {
			return true
		}
	}
	return false
}

func equipmentHasMetadata(items [EquipSlots]ItemInstance) bool {
	for _, item := range items {
		if itemHasMetadata(item) {
			return true
		}
	}
	return false
}
