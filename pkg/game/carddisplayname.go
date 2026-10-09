package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

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
