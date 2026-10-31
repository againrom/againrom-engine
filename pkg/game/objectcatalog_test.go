package game

import (
	"errors"
	"io/fs"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
)

func catalogFixture() memorySource {
	class := func(id int32, desc string) []synth.RegNode {
		nodes := []synth.RegNode{
			{Name: "ID", Kind: 0x02, Int: id}, {Name: "File", Kind: 0x02, Int: 0},
			{Name: "Index", Kind: 0x02, Int: 0}, {Name: "Width", Kind: 0x02, Int: 8}, {Name: "Height", Kind: 0x02, Int: 8},
		}
		if desc != "" {
			nodes = append(nodes, synth.RegNode{Name: "DescText", Kind: 0x00, Str: desc})
		}
		return nodes
	}
	return memorySource{ObjectRegistry: synth.ObjectsReg([]string{`trees\oak`}, class(0, "Oak"), class(1, ""), class(4, "Rock"))}
}

// Statics and the object list read one catalog.
func TestStaticsAndObjectListReadOneCatalog(t *testing.T) {
	src := catalogFixture()
	catalog, err := LoadObjectCatalog(src)
	if err != nil {
		t.Fatal(err)
	}
	set, err := LoadStatics(src)
	if err != nil {
		t.Fatal(err)
	}
	listed := &terrain.StaticSet{}
	for b := 1; b <= 0xff; b++ {
		_, named := catalog.ByCode(byte(b))
		if named != (set.Classes[b] != nil) {
			t.Fatalf("byte %d: catalog %t, statics %t", b, named, set.Classes[b] != nil)
		}
		if named {
			listed.Classes[b] = &terrain.StaticClass{Frame: &terrain.StaticFrame{}}
		}
	}
	rows := objectBrushRows(catalog, listed)
	want := map[int]string{1: "Oak", 2: "", 5: "Rock"}
	if len(rows) != 1+len(want) || rows[0].Code != 0 || rows[0].Name != "" {
		t.Fatalf("rows = %+v", rows)
	}
	for _, r := range rows[1:] {
		if name, ok := want[r.Code]; !ok || r.Name != name {
			t.Errorf("row %+v, want name %q", r, name)
		}
	}
	if got := objectBrushRows(nil, listed); len(got) != len(rows) || got[1].Name != "" {
		t.Errorf("rows without a catalog = %+v", got)
	}
}

func TestObjectCatalogErrors(t *testing.T) {
	if _, err := LoadObjectCatalog(memorySource{}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing registry: %v", err)
	}
	if _, err := LoadObjectCatalog(nil); err == nil {
		t.Error("nil source accepted")
	}
	if _, err := LoadObjectCatalog(memorySource{ObjectRegistry: []byte("not a registry")}); err == nil {
		t.Error("corrupt registry accepted")
	}
}
