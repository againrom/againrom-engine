package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

type itemChoiceKind struct {
	word       string
	class, row int
}

func (x *MapInspector) itemChoiceGroups(choices []ui.InspectionChoice) []ui.InspectionChoiceGroup {
	var groups []ui.InspectionChoiceGroup
	seen := map[itemChoiceKind]int{}
	for _, choice := range choices {
		if choice.Value <= 0 || choice.Value > 65535 {
			continue
		}
		code := data.ItemCode(choice.Value)
		d := mapload.DecodeItem(x.table, code)
		key := itemChoiceKind{word: d.KindKey}
		if key.word == "" {
			key.class, key.row = code.B(), code.D()
		}
		i, ok := seen[key]
		if !ok {
			kind := d.Kind
			if kind == "" {
				kind = fmt.Sprintf(x.word("editor.item.unknownkind"), key.class, key.row)
			}
			i = len(groups)
			seen[key] = i
			groups = append(groups, ui.InspectionChoiceGroup{Block: x.itemGroupWord("kind", kind, key.word)})
		}
		groups[i].Values = append(groups[i].Values, choice.Value)
	}
	return groups
}

// itemGroupWord is the caption of a kind, material or tier row: the word
// its row key names, else the row's own name.
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
