package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (x *MapInspector) itemChoiceGroups(choices []ui.InspectionChoice) []ui.InspectionChoiceGroup {
	var groups []ui.InspectionChoiceGroup
	seen := map[[2]string]int{}
	for _, choice := range choices {
		if choice.Value <= 0 || choice.Value > 65535 {
			continue
		}
		code := data.ItemCode(choice.Value)
		d := mapload.DecodeItem(x.table, code)
		kind, kindKey := d.Kind, d.KindKey
		if kind == "" {
			kind, kindKey = fmt.Sprintf(x.word("editor.item.unknownkind"), code.B(), code.D()), ""
		}
		material := d.Material
		key := [2]string{kind, material}
		i, ok := seen[key]
		if !ok {
			i = len(groups)
			seen[key] = i
			groups = append(groups, ui.InspectionChoiceGroup{Block: x.itemGroupWord("kind", kind, kindKey), Head: x.itemGroupWord("material", material, d.MaterialKey)})
		}
		groups[i].Values = append(groups[i].Values, choice.Value)
	}
	return groups
}

// itemGroupWord is the caption of a group's kind or material: the word the
// decoder's key names, else the row's own name.
func (x *MapInspector) itemGroupWord(axis, name, key string) string {
	if key != "" {
		id := "editor.item." + axis + "." + key
		if caption := x.word(id); caption != id {
			return x.inspectionUTF8(caption)
		}
	}
	return x.inspectionUTF8(name)
}

func (s *mapInspectionEdits) itemPictures(choices []ui.InspectionChoice) map[int64]*image.RGBA {
	for _, choice := range choices {
		_, _ = s.ItemLook(choice.Value)
	}
	return s.inspector.recordPictures["item"]
}
