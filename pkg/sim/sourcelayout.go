package sim

import "encoding/binary"

// The source records are written and read on every world encode, hash and
// decode. These field-by-field forms produce exactly the little-endian bytes
// encoding/binary produces for the same struct (a bool is one byte, read back
// as nonzero), without its per-field reflection. sourcelayout_test.go holds
// them to encoding/binary over every field.

func layoutBool(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func putSourceEquipment(b []byte, s *SourceEquipment) {
	_ = b[sourceEquipmentLen-1]
	b[0], b[1], b[2] = s.Class, s.DefinitionRow, s.OwnKind
	copy(b[3:27], s.Attack[:])
	copy(b[27:49], s.Defence[:])
	d := &s.Definition
	b[49] = layoutBool(d.Present)
	le := binary.LittleEndian
	le.PutUint32(b[50:], uint32(d.AttackType))
	le.PutUint32(b[54:], uint32(d.Hands))
	le.PutUint32(b[58:], uint32(d.Charge))
	le.PutUint32(b[62:], uint32(d.Relax))
	le.PutUint32(b[66:], uint32(d.Suitable))
	sp := &s.Spell
	b[70], b[71], b[72], b[73] = layoutBool(sp.Present), sp.ID, sp.Range, sp.Defensive
	le.PutUint16(b[74:], sp.ManaCost)
	b[76] = layoutBool(s.EffectsUnsupported)
}

func getSourceEquipment(b []byte, s *SourceEquipment) {
	_ = b[sourceEquipmentLen-1]
	le := binary.LittleEndian
	*s = SourceEquipment{Class: b[0], DefinitionRow: b[1], OwnKind: b[2]}
	copy(s.Attack[:], b[3:27])
	copy(s.Defence[:], b[27:49])
	s.Definition = SourceWeaponDefinition{Present: b[49] != 0,
		AttackType: int32(le.Uint32(b[50:])), Hands: int32(le.Uint32(b[54:])),
		Charge: int32(le.Uint32(b[58:])), Relax: int32(le.Uint32(b[62:])), Suitable: int32(le.Uint32(b[66:]))}
	s.Spell = SourceItemSpell{Present: b[70] != 0, ID: b[71], Range: b[72], Defensive: b[73], ManaCost: le.Uint16(b[74:])}
	s.EffectsUnsupported = b[76] != 0
}

func putSourceActor(b []byte, s *SourceActor) {
	_ = b[sourceActorLen-1]
	le := binary.LittleEndian
	b[0] = s.Class
	for i, v := range s.Stats {
		le.PutUint16(b[1+2*i:], v)
	}
	copy(b[29:53], s.Attack[:])
	copy(b[53:77], s.Base[:])
	copy(b[77:99], s.Defence[:])
	copy(b[99:163], s.Modifier[:])
	for i, v := range s.SkillXP {
		le.PutUint32(b[163+4*i:], v)
	}
	le.PutUint32(b[187:], s.Experience)
	le.PutUint16(b[191:], s.ManaFloor)
	le.PutUint16(b[193:], s.Sight)
	le.PutUint16(b[195:], s.TypeID)
	b[197], b[198], b[199], b[200] = s.MoverSpeed, layoutBool(s.Fighter), layoutBool(s.HasSpellbook), layoutBool(s.HasOwner)
	le.PutUint32(b[201:], s.ManaReservePercent)
	b[205], b[206], b[207], b[208] = s.Reach, s.AttackCharge, s.AttackRelax, layoutBool(s.EquipmentRuntimePresent)
}

func getSourceActor(b []byte, s *SourceActor) {
	_ = b[sourceActorLen-1]
	le := binary.LittleEndian
	*s = SourceActor{Class: b[0]}
	for i := range s.Stats {
		s.Stats[i] = le.Uint16(b[1+2*i:])
	}
	copy(s.Attack[:], b[29:53])
	copy(s.Base[:], b[53:77])
	copy(s.Defence[:], b[77:99])
	copy(s.Modifier[:], b[99:163])
	for i := range s.SkillXP {
		s.SkillXP[i] = le.Uint32(b[163+4*i:])
	}
	s.Experience = le.Uint32(b[187:])
	s.ManaFloor, s.Sight, s.TypeID = le.Uint16(b[191:]), le.Uint16(b[193:]), le.Uint16(b[195:])
	s.MoverSpeed, s.Fighter, s.HasSpellbook, s.HasOwner = b[197], b[198] != 0, b[199] != 0, b[200] != 0
	s.ManaReservePercent = le.Uint32(b[201:])
	s.Reach, s.AttackCharge, s.AttackRelax, s.EquipmentRuntimePresent = b[205], b[206], b[207], b[208] != 0
}

func putSourceBinding(b []byte, s *SourceBinding) {
	_ = b[sourceBindingLen-1]
	le := binary.LittleEndian
	b[0] = s.Class
	le.PutUint16(b[1:], s.ArchiveIndex)
	le.PutUint32(b[3:], s.Identity)
	le.PutUint32(b[7:], s.RuntimeID)
	b[11] = s.TokenRow
	le.PutUint16(b[12:], s.TypeID)
	b[14], b[15] = s.Face, s.ClassFlags
	le.PutUint32(b[16:], s.DisplayBacking)
	le.PutUint32(b[20:], s.GroupIndex)
	le.PutUint32(b[24:], s.GroupSelector)
	le.PutUint32(b[28:], s.GroupOwnerKey)
	le.PutUint16(b[32:], s.GroupOwnerSlot)
	b[34] = layoutBool(s.GroupOwnerResolved)
}

func getSourceBinding(b []byte, s *SourceBinding) {
	_ = b[sourceBindingLen-1]
	le := binary.LittleEndian
	*s = SourceBinding{Class: b[0], ArchiveIndex: le.Uint16(b[1:]), Identity: le.Uint32(b[3:]), RuntimeID: le.Uint32(b[7:]),
		TokenRow: b[11], TypeID: le.Uint16(b[12:]), Face: b[14], ClassFlags: b[15], DisplayBacking: le.Uint32(b[16:]),
		GroupIndex: le.Uint32(b[20:]), GroupSelector: le.Uint32(b[24:]), GroupOwnerKey: le.Uint32(b[28:]),
		GroupOwnerSlot: le.Uint16(b[32:]), GroupOwnerResolved: b[34] != 0}
}
