package game

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// structureCatalogLine is the Buildings row a placed structure's kind names,
// or its raw kind and the words that say no row exists. That a kind names a
// Buildings row is Medium in the format's grade; the line gives the row number
// and invents no name.
func (x *MapInspector) structureCatalogLine(o alm.Object) string {
	row, name, ok := mapload.BuildingName(x.table, o.Kind)
	if !ok {
		return fmt.Sprintf(x.inspectionUTF8(x.word("editor.structure.norow")), o.Kind)
	}
	return fmt.Sprintf(x.inspectionUTF8(x.word("editor.structure.row")), x.inspectionUTF8(name), row)
}
