package game

import (
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

type lootTableInput struct {
	Present bool
	Rows    []lootTableRow
}

type lootTableRow struct {
	Index   int
	Name    string
	Params  []int32
	Strings []string
	Doubles []float64
}

func observeLootConstruction(table *mapload.Table) (bool, map[string]lootTableInput) {
	if table == nil {
		return false, nil
	}
	inputs := map[string]lootTableInput{}
	for name, collection := range map[string]data.Collection{"Units": table.Units, "Humans": table.Humans, "Weapons": table.Weapons, "Armors": table.Armors, "Shields": table.Shields, "Magic": table.Magic, "MagicItems": table.MagicItems, "Spells": table.Spells} {
		input := lootTableInput{Present: collection != nil}
		if collection != nil {
			for index := 0; index < collection.Len(); index++ {
				input.Rows = append(input.Rows, lootTableRow{Index: index, Name: collection.EntryName(index), Params: slices.Clone(collection.EntryParams(index)), Strings: slices.Clone(collection.EntryStrings(index))})
			}
		}
		inputs[name] = input
	}
	for name, scale := range map[string]data.ScaleTable{"Shapes": table.Shapes, "Materials": table.Materials} {
		input := lootTableInput{Present: scale != nil}
		if scale != nil {
			for index := 0; index < scale.Len(); index++ {
				input.Rows = append(input.Rows, lootTableRow{Index: index, Name: scale.EntryName(index), Doubles: slices.Clone(scale.EntryDoubles(index))})
			}
		}
		inputs[name] = input
	}
	return true, inputs
}
