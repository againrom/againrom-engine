package game

import (
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// projectSavedActorValues updates only the bound actor's current value fields.
// ObjectIndex is the explicit DTO binding, never a CArchive index or Token key.
// The caller owns an independent complete document. Preparing the whole batch
// before publication also keeps that document unchanged on a late refusal.
//
// SAV-UNITPROG-156 gives the block/scalar widths; SAV-HUMRUN-444,
// SAV-HUMFOLD-446 and SAV-HUMMUT-448 distinguish current blocks, modifiers and
// progression. SourceNow owns those complete blocks, including retained tails
// and slot-zero operands. Reading them neither derives nor initializes bytes.
// This is not a topology, order, position, inventory or original-export gate.
//
// fresh, an optional trailing flag, forwards to savedActorValueRecord: true
// only when this pass runs immediately after currentworldbuild.go's one-time
// fresh construction of the same document. Every other caller passes nothing.
func projectSavedActorValues(document *sav.DocumentData, bindings []SnapshotSAVActor, world *sim.World, fresh ...bool) error {
	if document == nil || world == nil {
		return fmt.Errorf("saved SAV actor values require a document and world")
	}
	if len(bindings) > len(document.Objects) {
		return fmt.Errorf("saved SAV actor values have too many bindings")
	}
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	modifierEffects := currentModifierEffectDeltas(world)
	type update struct {
		index  uint16
		record sav.DocumentRecordData
	}
	var updates []update
	seenObjects := make(map[uint16]bool, len(bindings))
	seenEntities := make(map[sim.EntityID]bool, len(bindings))
	for _, binding := range bindings {
		if binding.Retired {
			continue
		}
		if binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(document.Objects) || seenObjects[binding.ObjectIndex] || seenEntities[binding.EntityID] {
			return fmt.Errorf("saved SAV actor %d has an invalid or duplicate binding", binding.EntityID)
		}
		seenObjects[binding.ObjectIndex], seenEntities[binding.EntityID] = true, true
		e, exists := entities[binding.EntityID]
		record := document.Objects[binding.ObjectIndex-1]
		if !exists || !currentActorRecordMatches(world, e, record) {
			return fmt.Errorf("saved SAV actor %d has no matching current class/entity", binding.EntityID)
		}
		if err := e.SourceBinding.Validate(e); err != nil {
			return fmt.Errorf("saved SAV actor %d: %w", e.ID, err)
		}
		if err := savedActorValueDomain(e); err != nil {
			return err
		}
		next, err := savedActorValueRecord(record, e, len(fresh) != 0 && fresh[0])
		if err != nil {
			return fmt.Errorf("saved SAV actor %d: %w", e.ID, err)
		}
		if e.ActorLoad.Source.Class == 0 && next.Class == "Human" {
			if e.NativeTraining.Present {
				items, ok := world.EquippedItems(e.ID)
				flags, err := savedStructureValue(&next, "U4C")
				if !ok || err != nil {
					return fmt.Errorf("saved native actor %d has no equipment or class", e.ID)
				}
				bonus := mapload.EquippedSkillBonus(items, flags&4 == 0)
				block, err := savedActorRaw(&next, "UD4", 64)
				if err != nil {
					return err
				}
				for j, v := range bonus {
					binary.LittleEndian.PutUint16(block[20+2*j:], uint16(v))
				}
			}
			if err := projectSavedModifierWords(document, &next, modifierEffects[e.ID]); err != nil {
				return fmt.Errorf("saved SAV actor %d: %w", e.ID, err)
			}
		}
		if err := projectCurrentDeadOwner(document, &next, world, e); err != nil {
			return err
		}
		if err := projectCurrentActorOwner(document, &next, e); err != nil {
			return err
		}
		updates = append(updates, update{binding.ObjectIndex, next})
	}
	for _, update := range updates {
		document.Objects[update.index-1] = update.record
	}
	return nil
}

// projectCurrentActorOwner writes the actor's owner Reference from its current
// owner: the key of the one Player whose slot is that owner. A script hand-over
// sets actor+14 to the new Player before the actor joins its new Group
// (PARTY-JOIN-025), and the original writes that field as it stands
// (SAV-PTRMAP-035). A body keeps the owner it had. A reference the loaded
// document carried from an earlier owner is therefore not kept. An owner no
// Player of this document carries leaves the record's own bytes.
func projectCurrentActorOwner(document *sav.DocumentData, record *sav.DocumentRecordData, e sim.Entity) error {
	if _, err := savedStructureValue(record, "Reference"); err != nil || e.Owner == 0 {
		return nil
	}
	var key uint32
	for _, index := range document.Players {
		if index == 0 || int(index) > len(document.Objects) {
			continue
		}
		player := &document.Objects[index-1]
		slot, err := savedStructureValue(player, "Slot")
		if err != nil || uint32(uint16(slot)) != e.Owner {
			continue
		}
		value, err := savedStructureValue(player, "This")
		if err != nil || value == 0 || key != 0 && key != value {
			return nil
		}
		key = value
	}
	if key == 0 {
		return nil
	}
	return savedActorSetValue(record, "Reference", key)
}

// savedModifierEffect is one timed effect kind whose magnitude the original
// adds to a Human's modifier block word and subtracts again at expiry
// (MAGIC-ATTACH-016, MAGIC-CONSUME-144). Offsets are block-relative: speed
// +4, sight +0x10 in 1/256 cell, absorption +0x2c and the protections from
// +0x2e (HERO-FOLD-035, SAV-1116 for the fold that adds each into its live
// word).
type savedModifierEffect struct {
	kind   sim.EffectKind
	key    uint32
	offset int
	scale  int32
}

var savedModifierEffects = [...]savedModifierEffect{
	{sim.EffectSpeed, 17, 4, 1},
	{sim.EffectScanRange, 19, 16, 256},
	{sim.EffectAbsorption, 16, 44, 1},
	{sim.EffectProtectionFire, 21, 48, 1},
	{sim.EffectProtectionWater, 22, 50, 1},
	{sim.EffectProtectionAir, 23, 52, 1},
	{sim.EffectProtectionEarth, 24, 54, 1},
}

// currentModifierEffectDeltas sums each actor's live non-continuous effects
// per modifier-backed kind.
func currentModifierEffectDeltas(world *sim.World) map[sim.EntityID]map[sim.EffectKind]int32 {
	out := map[sim.EntityID]map[sim.EffectKind]int32{}
	for _, e := range world.ActiveEffects() {
		if e.Mode&sim.EffectContinuous != 0 {
			continue
		}
		for _, m := range savedModifierEffects {
			if e.Kind != m.kind {
				continue
			}
			if out[e.Target] == nil {
				out[e.Target] = map[sim.EffectKind]int32{}
			}
			out[e.Target][e.Kind] += e.Magnitude
		}
	}
	return out
}

// projectSavedModifierWords writes each modifier-backed word of a Human as its
// remaining part plus the live effect sum. The previous sum is the record's own
// Effects of that kind; a zero word beside such Effects is an older save that
// never folded them.
func projectSavedModifierWords(document *sav.DocumentData, record *sav.DocumentRecordData, current map[sim.EffectKind]int32) error {
	block, err := savedActorRaw(record, "UD4", 64)
	if err != nil {
		return err
	}
	refs, _ := savedObjectRefs(record, "Effects")
	for _, m := range savedModifierEffects {
		word := int32(int16(binary.LittleEndian.Uint16(block[m.offset:])))
		var previous int32
		for _, index := range refs {
			if index == 0 || int(index) > len(document.Objects) {
				continue
			}
			child := &document.Objects[index-1]
			kind, kindErr := savedStructureValue(child, "E3C")
			mode, modeErr := savedStructureValue(child, "E3D")
			operand, operandErr := savedStructureValue(child, "E40")
			if kindErr != nil || modeErr != nil || operandErr != nil || kind != m.key || mode&uint32(sim.EffectContinuous) != 0 {
				continue
			}
			previous += int32(int16(operand)) * m.scale
		}
		remainder := word - previous
		if word == 0 {
			remainder = 0
		}
		binary.LittleEndian.PutUint16(block[m.offset:], uint16(remainder+current[m.kind]*m.scale))
	}
	return nil
}

func savedActorValueDomain(e sim.Entity) error {
	if err := e.ActorLoad.Validate(); err != nil {
		return fmt.Errorf("saved SAV actor %d: %w", e.ID, err)
	}
	return nil
}

// fresh is true only for currentworldbuild.go's one-time construction of a
// freshly generated mission's own initial document, and false for every
// later ordinary SAVE through projectSavedActorValues. It gates the Human
// own-weight, mover and order repairs below to that one construction. The
// Unit capacity and own-weight rule applies on every SAVE.
func savedActorValueRecord(record sav.DocumentRecordData, e sim.Entity, fresh bool) (sav.DocumentRecordData, error) {
	next := record
	next.Values = slices.Clone(record.Values)
	next.Raw = slices.Clone(record.Raw)
	if e.SourceBinding.Class != 0 {
		if err := savedActorSetValue(&next, "U148", e.SourceBinding.DisplayBacking); err != nil {
			return sav.DocumentRecordData{}, err
		}
	}
	s := e.SourceNow()
	if s.Class == 0 {
		for i, name := range []string{"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen"} {
			v, _ := savedStructureValue(&record, name)
			s.Stats[i] = uint16(v)
		}
		for _, pair := range []struct {
			name string
			dst  []byte
		}{{"UA6", s.Attack[:]}, {"UBE", s.Defence[:]}, {"U114", s.Base[:]}, {"UD4", s.Modifier[:]}} {
			for _, raw := range record.Raw {
				if raw.Name == pair.name {
					copy(pair.dst, raw.Bytes)
				}
			}
		}
		s = currentActorSource(e, s)
	}
	if record.Class == "Unit" {
		s.Stats[sav.StatCapacity], s.Stats[sav.StatOwnWeight] = unitLoadWords(s.Stats[sav.StatCapacity], s.Stats[sav.StatOwnWeight], s.Stats[sav.StatLoad], e.ActorLoad.Present && !fresh)
	} else if fresh && s.Stats[sav.StatOwnWeight] == 0 {
		// SAV-792/ITEM-LOAD-005: a generated Human's own weight is never
		// populated. The kit export mirrors its load.
		s.Stats[sav.StatOwnWeight] = s.Stats[sav.StatLoad]
	}
	if err := s.Validate(); err != nil {
		return sav.DocumentRecordData{}, err
	}
	// SAV-UNITFLD-049 and ITEM-LOAD-005 establish the fourteen word slots;
	// no current load, maximum, period or aggregate is recomputed here.
	for i, name := range [...]string{"Body", "Reaction", "Mind", "Spirit", "Speed", "U8E", "U90", "Capacity", "Health", "HealthMax", "HealthRegen", "Mana", "ManaMax", "ManaRegen"} {
		if err := savedActorSetValue(&next, name, uint32(s.Stats[i])); err != nil {
			return sav.DocumentRecordData{}, err
		}
	}
	for _, v := range []sav.DocumentValueData{
		{Name: "T0E", Value: uint32(s.TypeID)},
		{Name: "U49", Value: uint32(e.TokenSize)},
		{Name: "UA0", Value: uint32(s.ManaFloor)}, {Name: "UA4", Value: uint32(s.Sight)},
		{Name: "U130", Value: s.Experience},
		// SAV-REGENWIRE-532 stores the entire byte, including signed-arithmetic
		// remainders above 99. Do not normalize them on SAVE.
		{Name: "UA2", Value: uint32(e.HealthHundredths)}, {Name: "UA3", Value: uint32(e.ManaHundredths)},
	} {
		if err := savedActorSetValue(&next, v.Name, v.Value); err != nil {
			return sav.DocumentRecordData{}, err
		}
	}
	// SAV-653 and UNIT-STREAM-001: a creature's experience value is the Unit's
	// own +0x1c, which the SAVE writes as the record's T1C. The information
	// card's spellcaster caption tests it for non-zero, so a zero here costs
	// the original that caption on the creature. A Human keeps its own rule.
	if record.Class == "Unit" && e.TypeID >= 0x1a {
		if err := savedActorSetValue(&next, "T1C", uint32(e.XPValue)); err != nil {
			return sav.DocumentRecordData{}, err
		}
	}
	// MOVE-TICK-017/TRIG-OFFMAP-041: ROM1's LOAD re-links every actor to the
	// on-map list except one carrying U4C bit 3, the bit opcode 16 sets and
	// opcode 17 clears. The live presence is current state, so it replaces
	// whatever this bit held in a loaded document.
	flags, err := savedStructureValue(&next, "U4C")
	if err != nil {
		return sav.DocumentRecordData{}, err
	}
	flags &^= sav.ActorOffMapFlag
	if record.Class == "Human" && e.NativeClass.Present {
		flags &^= 4
		if !e.NativeClass.Fighter {
			flags |= 4
		}
	}
	if e.OffMap {
		flags |= sav.ActorOffMapFlag
	}
	if err := savedActorSetValue(&next, "U4C", flags); err != nil {
		return sav.DocumentRecordData{}, err
	}
	if s.EquipmentRuntimePresent {
		// SAV-EQUIPORDER-552: these bytes are independently produced after
		// derive. Presence is explicit; absent support supplies no defaults.
		for _, v := range []sav.DocumentValueData{{Name: "U12C", Value: uint32(s.Reach)}, {Name: "U134", Value: uint32(s.AttackCharge)}, {Name: "U135", Value: uint32(s.AttackRelax)}} {
			if err := savedActorSetValue(&next, v.Name, v.Value); err != nil {
				return sav.DocumentRecordData{}, err
			}
		}
	}
	for _, block := range []sav.DocumentRawData{{Name: "UA6", Bytes: s.Attack[:]}, {Name: "UBE", Bytes: s.Defence[:]}, {Name: "U114", Bytes: s.Base[:]}, {Name: "UD4", Bytes: s.Modifier[:]}} {
		dst, err := savedActorRaw(&next, block.Name, len(block.Bytes))
		if err != nil {
			return sav.DocumentRecordData{}, err
		}
		copy(dst, block.Bytes)
	}
	mover, err := savedActorRaw(&next, "U154", 180)
	if err != nil {
		return sav.DocumentRecordData{}, err
	}
	// MOVE-TURN-031 names these two bytes. Neither desired facing, turn
	// progress, position, transit nor route storage is authored by this pass.
	mover[0], mover[10] = e.Facing, s.MoverSpeed
	domain, err := savedStructureValue(&next, "U4A")
	if err != nil {
		return sav.DocumentRecordData{}, err
	}
	// TERR-PASS-051: the freshly generated mission's own initial document
	// reaches this pass with a zero passability mask that blocks nothing,
	// not even the border. Every actor needs a real mask to move at all, so
	// this one repair applies regardless of owner.
	if fresh && mover[5] == 0 {
		mover[5] = sim.MoverPassabilityMask(domain)
	}
	// SAV-1096/AI-POST-042/AI-GRPGUARD-074: every AI-owned mission-start
	// original carries the idle-turn/guard state 0x0b and its own post
	// (PostX/PostY, the cell the guard/group-order-1 leash anchors to) in
	// the AI-start order block. (The mover's own copy of the post and its
	// 5,255/0x80,0x80 constants are sim.ProjectActorMotion's own write,
	// currentmotion.go: projectMotion's later, unconditional whole-block
	// copy from Mover makes any write to that slice here dead before it
	// ever reaches the document.) SAV-1096's own corpus (game9232/9233/9234,
	// three mission-start saves) is entirely AI actors; DIV-1388 records
	// that against game9237 (the byte-corrected mission-10 reference SAV
	// that plays combat correctly) and three real corpus mission-10
	// originals sharing the same hero row, the player's own hero carries
	// zero at both offsets instead. A party actor's post and state are the
	// player's, not this AI-start default, so both stay excluded for the
	// party's own actor. An ordinary SAVE's record already carries an
	// original's own loaded bytes here whenever one exists, which are
	// current state and outrank this derivation (owner precedence), so the
	// repair stays inside this one-time construction, each site additionally
	// gated on its own untouched zero.
	if fresh && e.Owner != sim.SelfSlot {
		order, err := savedActorRaw(&next, "U158", 148)
		if err != nil {
			return sav.DocumentRecordData{}, err
		}
		if order[0] == 0 && order[1] == 0 {
			binary.LittleEndian.PutUint16(order[0:], uint16(e.PostX)|uint16(e.PostY)<<8)
		}
		state, err := savedActorRaw(&next, "U50", 4)
		if err != nil {
			return sav.DocumentRecordData{}, err
		}
		if binary.LittleEndian.Uint32(state) == 0 {
			binary.LittleEndian.PutUint32(state, 0x0b)
		}
	}
	if record.Class != "Unit" {
		xp, err := savedActorRaw(&next, "H1CC", 24)
		if err != nil {
			return sav.DocumentRecordData{}, err
		}
		// SAV-HEROXP-063: six raw signed-dword bit patterns, not an aggregate
		// derived from their sum or from skill levels.
		for i, value := range s.SkillXP {
			binary.LittleEndian.PutUint32(xp[4*i:], value)
		}
	}
	return next, nil
}

func savedActorSetValue(record *sav.DocumentRecordData, name string, value uint32) error {
	index := -1
	for i, field := range record.Values {
		if field.Name == name {
			if index != -1 {
				return fmt.Errorf("duplicate actor value %s", name)
			}
			index = i
		}
	}
	if index < 0 {
		return fmt.Errorf("missing actor value %s", name)
	}
	record.Values[index].Value = value
	return nil
}

func savedActorRaw(record *sav.DocumentRecordData, name string, width int) ([]byte, error) {
	index := -1
	for i, field := range record.Raw {
		if field.Name == name {
			if index != -1 {
				return nil, fmt.Errorf("duplicate actor block %s", name)
			}
			index = i
		}
	}
	if index < 0 || len(record.Raw[index].Bytes) != width {
		return nil, fmt.Errorf("actor block %s does not have width %d", name, width)
	}
	record.Raw[index].Bytes = slices.Clone(record.Raw[index].Bytes)
	return record.Raw[index].Bytes, nil
}

func currentActorClassMatches(e sim.Entity, class string) bool {
	if e.SourceBinding.Class != 0 {
		return savedActorClass(e.SourceBinding.Class) == class
	}
	if e.Humanoid {
		return class == "Human" || class == "Humanoid"
	}
	return class == "Unit"
}

// unitLoadWords is a Unit's saved capacity and own weight. The original
// divides load by capacity, and SAV-UNITFLD-049 reads data.UnitCapacity on
// every original Unit record, so a zero is replaced. A native or freshly
// constructed Unit holds no own weight; every original Unit record carries
// own weight equal to load.
func unitLoadWords(capacity, own, load uint16, held bool) (uint16, uint16) {
	if capacity == 0 {
		capacity = uint16(data.UnitCapacity())
	}
	if !held && own == 0 {
		own = load
	}
	return capacity, own
}
