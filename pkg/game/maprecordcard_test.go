package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func TestFirstEditorRecordCardsShowAllValuesInBothLanguages(t *testing.T) {
	raw := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12),
		Units: []synth.ALMUnit{
			{X: 3*256 + 128, Y: 4*256 + 64, ClassID: 5, ClassSubID: 2, Flags: 0x04, DefID: 1234, CurrentHP: 75, HasCurrentHP: true, UnitID: 11},
			{X: 5*256 + 10, Y: 6*256 + 20, ClassID: 300, ClassSubID: 1, UnitID: 12},
		},
		Objects: []synth.ALMObject{
			{X: 7*256 + 128, Y: 8*256 + 128, Kind: 5, Field0C: 12, Field0E: 1, Field12: 21},
			{X: 9*256 + 128, Y: 10*256 + 128, Kind: 0x21, Field0C: 1, Field0E: 1, Field12: 22, Ext: []byte{3, 0, 0, 0, 4, 0, 0, 0}},
		},
	})
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct {
		unit int
		key  string
		val  int64
	}{{0, "owner", 1}, {0, "group", 9}, {1, "owner", 1}} {
		if err := m.SetUnitField(s.unit, s.key, s.val); err != nil {
			t.Fatal(err)
		}
	}
	sack := m.GroundSack(11*256+32, 12*256+48, 0x0101)
	binary.LittleEndian.PutUint32(sack[16:], 250)
	if _, err := m.Place(mapedit.KindLoot, sack); err != nil {
		t.Fatal(err)
	}
	b := m.Bytes()

	table := authoringTable()
	table.Names = data.ItemNames{0x0101: "Iron Sword"}
	structs := &terrain.StructureSet{}
	structs.Classes[5] = &terrain.StructureClass{ID: 5, Name: "House", TileWidth: 2, TileHeight: 2}
	structs.Classes[0x21] = &terrain.StructureClass{ID: 0x21, Name: "Bridge", TileWidth: 1, TileHeight: 1, VariableSize: true}

	path := filepath.Join(t.TempDir(), GraphicsArchive)
	if err := os.WriteFile(path, synth.Archive(nil), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"english", "russian"} {
		t.Run(lang, func(t *testing.T) {
			x := &MapInspector{
				archives:   &Archives{Base: base.Match{Profile: base.Profile{Game: base.GameROM1, Language: lang}}, Containers: fsys},
				tiles:      &terrain.Tileset{},
				structures: structs,
				table:      table,
				font:       &text.Font{Selector: map[string]int{"english": 0, "russian": 1}[lang]},
			}
			d, err := x.inspect(b, "first.alm")
			if err != nil {
				t.Fatal(err)
			}
			w := func(id string) string { return x.inspectionUTF8(x.word(id)) }
			check := func(kind ui.RecordKind, idx int, wants ...string) ui.InspectionRecord {
				t.Helper()
				r := recordByKind(t, d, kind, idx)
				joined := strings.Join(r.Details, "\n")
				for _, want := range wants {
					if !strings.Contains(joined, want) {
						t.Fatalf("%s %d details lack %q:\n%s", kind, idx, want, joined)
					}
				}
				return r
			}
			check(ui.KindUnit, 0,
				w("editor.card.x")+": 896",
				w("editor.card.y")+": 1088",
				w("editor.card.class")+": 5",
				w("editor.card.sub")+": 2",
				w("editor.plain.owner")+": 1:",
				w("editor.card.unitid")+": 11",
				w("editor.unit.field.group")+": 9",
				w("editor.second.unit.flags")+": 4",
				w("editor.second.unit.definition")+": 1234: row",
				w("editor.second.unit.hp")+": 75",
			)
			unresolved := check(ui.KindUnit, 1, w("editor.card.x")+": 1290", w("editor.card.y")+": 1556")
			if !strings.Contains(unresolved.Warning, "300") || strings.Contains(unresolved.Warning, "Definition unresolved") {
				t.Fatalf("unresolved unit warning = %q", unresolved.Warning)
			}
			check(ui.KindStructure, 0,
				w("editor.card.x")+": 1920",
				w("editor.card.y")+": 2176",
				w("editor.plain.type")+": 5: House",
				w("editor.plain.owner")+": 1:",
				w("editor.plain.target")+": 21",
				w("editor.plain.placement")+": 12",
			)
			check(ui.KindStructure, 1,
				w("editor.plain.type")+": 33: Bridge",
				w("editor.plain.width")+": 3",
				w("editor.plain.height")+": 4",
			)
			check(ui.KindSack, 0,
				w("editor.plain.owner")+": 0",
				w("editor.card.x")+": 2848",
				w("editor.card.y")+": 3120",
				w("editor.plain.gold")+": 250",
				w("editor.card.items")+": 1",
				": 257: Iron Sword",
				": 0",
				w("editor.inventory.norecipe"),
			)
		})
	}
}

func TestChoiceCatalogsKeepBothGamesContents(t *testing.T) {
	firstMap := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12),
		Units:   []synth.ALMUnit{{X: 3 * 256, Y: 4 * 256, ClassID: 5}},
		Objects: []synth.ALMObject{{X: 5 * 256, Y: 6 * 256, Kind: 5}},
	})
	structs := &terrain.StructureSet{}
	structs.Classes[5] = &terrain.StructureClass{ID: 5, Name: "House"}
	structs.Classes[9] = &terrain.StructureClass{ID: 9, Name: "Tower"}
	x1 := &MapInspector{tiles: &terrain.Tileset{}, structures: structs, table: authoringTable()}
	d1, err := x1.inspect(firstMap, "first.alm")
	if err != nil {
		t.Fatal(err)
	}
	s1 := d1.Edits.(*mapInspectionEdits)
	wantFirstDefs := []ui.InspectionChoice{{Value: 0, Label: "0: Use class key"}, {Value: 1234, Label: "1234: row"}}
	if got := s1.RecordChoices(ui.KindUnit, 0, "definition"); !slices.Equal(got, wantFirstDefs) {
		t.Fatalf("first unit definitions = %+v, want %+v", got, wantFirstDefs)
	}
	wantFirstKinds := []ui.InspectionChoice{{Value: 5, Label: "5: House"}, {Value: 9, Label: "9: Tower"}}
	if got := s1.RecordChoices(ui.KindStructure, 0, "kind"); !slices.Equal(got, wantFirstKinds) {
		t.Fatalf("first structure kinds = %+v, want %+v", got, wantFirstKinds)
	}

	x1.recordOptions["identity"] = []ui.InspectionChoice{{Value: 7, Label: "7: Person"}}
	if got := s1.RecordChoices(ui.KindStructure, 0, "identity"); len(got) != 0 {
		t.Fatalf("a structure's script target offers the unit identities: %+v", got)
	}

	x2 := secondTestInspector(t)
	d2, err := x2.inspect(secondFixtureMap(), "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	s2 := d2.Edits.(*mapInspectionEdits)
	wantSecondUnits := []ui.InspectionChoice{{Value: 600, Label: "600: Goblin_Pike"}, {Value: 601, Label: "601: Goblin_Pike.2"}}
	if got := s2.RecordChoices(ui.KindUnit, 0, "definition"); !slices.Equal(got, wantSecondUnits) {
		t.Fatalf("second creature definitions = %+v, want %+v", got, wantSecondUnits)
	}
	wantSecondHumans := []ui.InspectionChoice{{Value: 2113, Label: "2113: 2F_Knight1"}, {Value: 2114, Label: "2114: 2A_Knight1"}}
	if got := s2.RecordChoices(ui.KindUnit, 1, "definition"); !slices.Equal(got, wantSecondHumans) {
		t.Fatalf("second human definitions = %+v, want %+v", got, wantSecondHumans)
	}
	wantSecondKinds := []ui.InspectionChoice{
		{Value: 5, Label: "5: Mill"},
		{Value: 33, Label: "33: Vertical Wooden Bridge"},
		{Value: 34, Label: "34: Shop"},
		{Value: 0x01000021, Label: "33: Vertical Wooden Bridge"},
	}
	if got := s2.RecordChoices(ui.KindStructure, 1, "kind"); !slices.Equal(got, wantSecondKinds) {
		t.Fatalf("second extended structure kinds = %+v, want %+v", got, wantSecondKinds)
	}
}
