package sim

import (
	"fmt"
)

// SourceBinding is canonical file-local construction identity, not EntityID or
// MapUnitID. GroupIndex identifies an inline Group; GroupSelector is its saved
// literal +1c. Group owner is distinct from the actor's effective Owner. Raw AI
// is not represented here and no incoming AI continuation is implied.
type SourceBinding struct {
	Class                                    uint8 // 1..3 imported; 4 generated Unit; 5 generated Human
	ArchiveIndex                             uint16
	Identity, RuntimeID                      uint32
	TokenRow                                 uint8
	TypeID                                   uint16
	Face, ClassFlags                         uint8
	DisplayBacking                           uint32
	GroupIndex, GroupSelector, GroupOwnerKey uint32
	GroupOwnerSlot                           uint16
	GroupOwnerResolved                       bool
}

const sourceBindingLen = 35

const (
	GeneratedUnitBinding  uint8 = 4
	GeneratedHumanBinding uint8 = 5
)

// Generated is explicit provenance in the existing class byte. Zeroing an
// imported archive index never converts an imported binding into this arm.
func (s SourceBinding) Generated() bool {
	return s.Class == GeneratedUnitBinding || s.Class == GeneratedHumanBinding
}

func (s SourceBinding) ActorClass() uint8 {
	if s.Generated() {
		return s.Class - 3
	}
	return s.Class
}

func (s SourceBinding) DefinitionRow() uint8 {
	if s.ActorClass() == 2 && s.TypeID >= 33 {
		return 5
	}
	return s.TokenRow
}

func (s SourceBinding) Validate(e Entity) error {
	if s.Class == 0 {
		if s != (SourceBinding{}) {
			return fmt.Errorf("sim: absent source binding has residue")
		}
		return nil
	}
	if s.Class > 5 || (!s.Generated() && s.ArchiveIndex == 0) || (s.Generated() && (s.ArchiveIndex != 0 || s.Identity == 0 || s.RuntimeID == 0)) || e.Humanoid != (s.ActorClass() != 1) {
		return fmt.Errorf("sim: source binding class/index disagrees with entity %d", e.ID)
	}
	basisClass := s.ActorClass()
	if basisClass == 3 {
		basisClass = 2
	} // bounded native Humanoid numeric policy, not a SAV Human alias
	if !e.ActorLoad.Present || e.ActorLoad.Source.Class != basisClass {
		return fmt.Errorf("sim: source binding lacks its canonical actor basis for entity %d", e.ID)
	}
	if s.GroupIndex == 0 && (s.GroupSelector != 0 || s.GroupOwnerKey != 0 || s.GroupOwnerSlot != 0 || s.GroupOwnerResolved) {
		return fmt.Errorf("sim: absent source Group has residue")
	}
	if !s.GroupOwnerResolved && s.GroupOwnerSlot != 0 {
		return fmt.Errorf("sim: unresolved source Group owner has a slot")
	}
	return nil
}

func sourceBindingsFault(entities []Entity) error {
	indices, keys := map[uint16]bool{}, map[uint32]bool{}
	for _, e := range entities {
		s := e.SourceBinding
		if err := s.Validate(e); err != nil {
			return err
		}
		if s.Class == 0 {
			continue
		}
		if s.ArchiveIndex != 0 && indices[s.ArchiveIndex] || s.Identity != 0 && keys[s.Identity] {
			return fmt.Errorf("sim: duplicate source actor identity")
		}
		if s.ArchiveIndex != 0 {
			indices[s.ArchiveIndex] = true
		}
		keys[s.Identity] = true
	}
	return nil
}
