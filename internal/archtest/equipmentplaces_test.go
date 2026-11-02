package archtest

import (
	"os"
	"strings"
	"testing"
)

const equipmentPlacesSource = "package sav\nconst HeldWeaponField = \"HeldWeapon\"\nvar places = [2]P{{HeldWeaponField, 0}, {\"Worn\", 2}}\n"

// TestCheckEquipmentPlacesNamesEachCopy plants each shape a second slot
// routing of a saved actor's equipment takes, and the routes the table keeps.
func TestCheckEquipmentPlacesNamesEachCopy(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the table's own literals", equipmentPlacesFile, equipmentPlacesSource, ""},
		{"a route through the table", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/formats/sav\"\nfunc f(a sav.ActorHoldings) { for slot, place := range sav.EquipmentPlaces() { _, _ = slot, a.HeldPiece(place.Field) }; for at := range a.Worn { _ = sav.WornOverlap(at) } }\n", ""},
		{"a field named by a literal", "pkg/formats/sav/a.go",
			"package sav\nfunc f(r *Record) int { return len(r.Refs[\"HeldShield\"]) }\n", "named by a literal"},
		{"the field list spelled out", "pkg/formats/sav/a.go",
			"package sav\nvar fields = []string{HeldWeaponField, HeldShieldField, \"Worn\"}\n", "named by a literal"},
		{"the held pair by position", "pkg/game/a.go",
			"package game\nfunc f(a sav.ActorHoldings) { for slot, p := range []*sav.Piece{a.HeldWeapon, a.HeldShield} { _, _ = slot, p } }\n", "listed by slot position"},
		{"a Worn position literal", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/formats/sav\"\nfunc f(a sav.ActorHoldings) *sav.Piece { return a.Worn[1] }\n", "slot-index literal"},
		{"a WornSlots position literal in sav", "pkg/formats/sav/a.go",
			"package sav\nfunc f(r *Record) *Record { return r.WornSlots[0] }\n", "slot-index literal"},
		{"an overlap by slot number", "pkg/game/a.go",
			"package game\nimport \"againrom/pkg/formats/sav\"\nfunc f(a sav.ActorHoldings) bool { for slot, p := range a.Worn { if p != nil && slot < 2 { return false } }; return true }\n", "compared with a slot-index literal"},
		{"the Worn width by number", "pkg/formats/sav/a.go",
			"package sav\nfunc f(r *Record) bool { return len(r.WornSlots) != 12 }\n", "Worn width"},
		{"a city held reference by position", "pkg/formats/sav/a.go",
			"package sav\nfunc f(u *cityUnit, slots [12]*cityObject) { u.reference78 = slots[1] }\n", "taken by slot position"},
		{"a city held reference through the table", "pkg/formats/sav/a.go",
			"package sav\nfunc f(u *cityUnit, slots [12]*cityObject) { for slot := range EquipmentPlaces() { u.reference74 = slots[slot] } }\n", ""},
		{"another package's Worn", "pkg/game/a.go",
			"package game\nfunc f(s sheet) item { for slot := range s.Worn { if slot == 0 { return s.Worn[0] } }; return item{} }\n", ""},
		{"a test file", "pkg/formats/sav/a_test.go",
			"package sav\nvar fields = []string{\"HeldWeapon\", \"HeldShield\", \"Worn\"}\n", ""},
		{"outside pkg", "cmd/x/a.go",
			"package main\nvar field = \"Worn\"\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{equipmentPlacesFile: equipmentPlacesSource, tc.file: tc.src}
			vs := CheckEquipmentPlaces(files)
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckEquipmentPlaces = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file+":") {
				t.Fatalf("CheckEquipmentPlaces = %v, want one in %s naming %q", vs, tc.file, tc.want)
			}
		})
	}
	if vs := CheckEquipmentPlaces(map[string]string{"pkg/game/a.go": "package game\n"}); len(vs) != 1 || !strings.Contains(vs[0].Reason, "slot table is gone") {
		t.Fatalf("CheckEquipmentPlaces without the table = %v, want one violation", vs)
	}
	if vs := CheckEquipmentPlaces(nil); len(vs) != 1 {
		t.Fatalf("CheckEquipmentPlaces(nil) = %v, want one violation", vs)
	}
}

// TestEquipmentPlacesHaveOneTable is the live-tree half.
func TestEquipmentPlacesHaveOneTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[equipmentPlacesFile]; !ok {
		t.Fatalf("%s is not among the scanned sources", equipmentPlacesFile)
	}
	for _, v := range CheckEquipmentPlaces(files) {
		t.Errorf("%s", v)
	}
}
