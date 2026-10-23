package data

import (
	"slices"

	"againrom/pkg/formats/reg"
)

// NPCCatalog is the scenario NPC registry read once. Every reader of npc.reg
// asks it: a placement and a campaign grant read Defs, a dialogue portrait
// reads Faces, and the map editor's speaker list and a mod's character
// catalog read Entries. One parse answers all three, so a speaker the editor
// lists is a record the placement table knows (REG-NPC-058).
type NPCCatalog struct {
	// Defs is the placement half: the definition id, flags, hero templates
	// and mercenary terms of each record.
	Defs *NPCDefs
	// Faces is the portrait half: the dialogue picture of each record.
	Faces map[int32]NPCFace
	// Entries are the npc<n> records in the registry's own order.
	Entries []NPCEntry
}

// NPCEntry is one npc<n> record of the registry.
type NPCEntry struct {
	// ID is the subscript n; Section the section name as the registry spells it.
	ID      int32
	Section string
	// Name is the record's own Name string; empty when it states none, as
	// every shipped record does.
	Name string
}

// npcNameKey is the key a record may name its speaker with.
const npcNameKey = "Name"

// LoadNPCCatalog reads every npc<n> section of r once. A nil registry is a
// catalog with no records, whose Defs answer no entry and whose Faces are empty.
func LoadNPCCatalog(r *reg.Reg) *NPCCatalog {
	c := &NPCCatalog{Defs: loadNPCDefs(r), Faces: loadNPCFaces(r)}
	if r == nil || r.Root == nil {
		return c
	}
	for _, sec := range r.Root.Children {
		id, ok := npcSubscript(sec.Name)
		if !ok {
			continue
		}
		name, _ := r.GetString(sec.Name, npcNameKey)
		c.Entries = append(c.Entries, NPCEntry{ID: id, Section: sec.Name, Name: name})
	}
	return c
}

// Entry is the record of subscript id.
func (c *NPCCatalog) Entry(id int32) (NPCEntry, bool) {
	if c == nil {
		return NPCEntry{}, false
	}
	i := slices.IndexFunc(c.Entries, func(e NPCEntry) bool { return e.ID == id })
	if i < 0 {
		return NPCEntry{}, false
	}
	return c.Entries[i], true
}

// Records are the catalog's records; none for a nil catalog.
func (c *NPCCatalog) Records() []NPCEntry {
	if c == nil {
		return nil
	}
	return c.Entries
}

// Placement is the catalog's placement half; nil for a nil catalog.
func (c *NPCCatalog) Placement() *NPCDefs {
	if c == nil {
		return nil
	}
	return c.Defs
}

// Portraits is the catalog's portrait half; nil for a nil catalog.
func (c *NPCCatalog) Portraits() map[int32]NPCFace {
	if c == nil {
		return nil
	}
	return c.Faces
}

// LoadNPCDefs is the placement half of the catalog of r.
func LoadNPCDefs(r *reg.Reg) *NPCDefs { return LoadNPCCatalog(r).Defs }

// LoadNPCFaces is the portrait half of the catalog of r.
func LoadNPCFaces(r *reg.Reg) map[int32]NPCFace { return LoadNPCCatalog(r).Faces }
