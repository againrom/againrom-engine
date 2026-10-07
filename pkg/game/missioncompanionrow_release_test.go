package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// A mission SAVE must write each companion's own PC_ Humans row, the row the
// city SAVE writes for the same member. The first row sharing its TypeID is a
// creature row: the reader then drops the companion as a temporary ally and
// draws it from that creature row.
func TestReleaseMissionSaveWritesCompanionRows(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Row hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	for _, npc := range []int32{22, 23, 24, 25} {
		member, ok := mapload.CampaignNPCMember(f.Table, npc, 0, f.Carried)
		if !ok {
			t.Fatal("missing installed companion", npc)
		}
		f.Carried = append(f.Carried, member)
	}
	hero := nativeCityHeroOf(f.Carried)
	want := map[string]byte{}
	for _, m := range f.Carried {
		if m.CompanionNPC != 0 {
			want[m.Name] = nativeCityDefRow(m, hero, f.Table)
		}
	}
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(10, f.Carried)(); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(3)
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	chars, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, c := range chars {
		row, ok := want[c.Name]
		if !ok {
			continue
		}
		seen++
		if c.DefRow != row || !persistentOriginalCharacter(c, f.Table) {
			t.Fatalf("%s saved row %d (%q), want %d", c.Name, c.DefRow, f.Table.Humans.EntryName(int(c.DefRow)), row)
		}
	}
	if seen != len(want) {
		t.Fatalf("saved %d of %d companions", seen, len(want))
	}
}
