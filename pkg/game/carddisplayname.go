package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// drawnName is the name the map draws for e with class c (its composed body
// when it has one): the caption and the selection card show it, the card
// falling back to unitNames[nameIndex] when it is empty (ui.PanelSubjectName).
// definitionCharName reports that the character's own name is only its
// definition row's, which an ordinary actor does not show.
func (mw *mapWorld) drawnName(e *sim.Entity, c *terrain.UnitClass, unitNames []string) (name string, nameIndex int, definitionCharName bool) {
	if c != nil {
		name = c.Name
	}
	saved, hasSaved := mw.actorNames[e.ID]
	if hasSaved {
		name = saved
	}
	ordinary := (e.SourceBinding.ActorClass() == 1 || !sim.InPersistBand(e.TypeID)) &&
		mw.missionPartyMember(e.ID) == nil
	definitionName := ""
	if ordinary && mw.mission != nil {
		definitionName = sourceActorDefinitionName(mw.mission.table, e.SourceBinding)
	}
	sourceClassName := definitionName != "" && saved == definitionName
	char := mw.chars[e.ID]
	if ordinary && definitionName != "" && char.Name == definitionName {
		char.Name, definitionCharName = "", true
	}
	if char.Name != "" {
		name = char.Name
	}
	nameIndex = int(e.TypeID)
	nativeClassName := e.SourceBinding.Class == 0 && e.ActorLoad.Source.Class == 0 &&
		e.NativeBasis.HasValues() && c != nil && saved == c.Name
	if !ordinary && char.UnitNameIndex != 0 {
		nameIndex = char.UnitNameIndex
	}
	if (ordinary || char.Known) && char.Name == "" && (saved == "" || nativeClassName || sourceClassName) {
		if ordinary {
			name = ""
		}
		if nameIndex >= 0 && nameIndex < len(unitNames) && unitNames[nameIndex] != "" {
			name = unitNames[nameIndex]
		}
	}
	return name, nameIndex, definitionCharName
}

// shownName is the name the selection card shows for entity id, the name a
// message line gives the character it is about; "" when the card shows none.
func (mw *mapWorld) shownName(id sim.EntityID) string {
	e, ok := mw.world.Entity(id)
	if !ok {
		return ""
	}
	var classes map[int32]*terrain.UnitClass
	if mw.units != nil {
		classes = mw.units.Classes
	}
	c := classes[e.Class]
	if body := mw.art[e.ID]; body != nil {
		c = body
	}
	var unitNames []string
	if mw.view != nil {
		names := mw.view.Words().UnitNames
		unitNames = names[:]
	}
	name, nameIndex, _ := mw.drawnName(&e, c, unitNames)
	if name == "" && nameIndex >= 0 && nameIndex < len(unitNames) {
		name = unitNames[nameIndex]
	}
	return name
}

func sourceActorDefinitionName(table *mapload.Table, binding sim.SourceBinding) string {
	if table == nil {
		return ""
	}
	var collection data.Collection
	switch binding.ActorClass() {
	case 1:
		collection = table.Units
	case 2:
		collection = table.Humans
	default:
		return ""
	}
	row := int(binding.DefinitionRow())
	if collection == nil || row >= collection.Len() {
		return ""
	}
	if binding.ActorClass() == 1 && row == 0 {
		if _, err := data.NewUnitDef(collection.EntryName(row), collection.EntryParams(row)); err != nil {
			row = data.FindUnit(collection, int32(uint8(binding.TypeID)), int32(binding.Face))
			if row == data.NotFound {
				for candidate := 1; candidate < collection.Len(); candidate++ {
					if collection.EntryName(candidate) != "Ghost" {
						continue
					}
					definition, err := data.NewUnitDef(collection.EntryName(candidate), collection.EntryParams(candidate))
					if err == nil && definition.TypeID == int32(uint8(binding.TypeID)) {
						row = candidate
					}
					break
				}
			}
			if row == data.NotFound {
				return ""
			}
			if _, err := data.NewUnitDef(collection.EntryName(row), collection.EntryParams(row)); err != nil {
				return ""
			}
		}
	}
	return collection.EntryName(row)
}
