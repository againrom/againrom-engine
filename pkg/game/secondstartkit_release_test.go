package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func secondKitLine(f *FrontEnd, m mapload.PartyMember) string {
	spell := func(id int) string {
		if id > 0 && id < f.Table.Spells.Len() && f.Table.Spells.EntryName(id) != "" {
			return f.Table.Spells.EntryName(id)
		}
		return fmt.Sprintf("spell %d", id)
	}
	item := func(code uint16, inst sim.ItemInstance) string {
		s := itemName(data.ItemCode(code), f.Table)
		for _, e := range inst.Effects {
			if e.Kind == 41 {
				s += fmt.Sprintf(" {castSpell=%s:%d}", spell(int(e.Operand&0xffff)), int16(e.Operand>>16))
			}
		}
		return s
	}
	var worn, carried, book []string
	for i, c := range m.Worn {
		if c != 0 {
			worn = append(worn, fmt.Sprintf("%d:%s", i, item(c, m.WornItems[i])))
		}
	}
	for i, c := range m.Carried {
		var inst sim.ItemInstance
		if i < len(m.CarriedItems) {
			inst = m.CarriedItems[i]
		}
		carried = append(carried, item(c, inst))
	}
	for i, s := range m.Book.Slots {
		if s != (sim.BookSpell{}) {
			book = append(book, spell(i+1))
		}
	}
	return fmt.Sprintf("worn [%s] carried [%s] known %#x book [%s]",
		strings.Join(worn, "; "), strings.Join(carried, "; "), m.KnownSpells, strings.Join(book, "; "))
}

func TestReleaseSecondGameStartKitByRoute(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second start kit")
	app.Layout(1024, 768)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if err := f.startSecondCampaign(); err != nil {
		t.Fatal(err)
	}
	if len(f.Carried) != 1 {
		t.Fatalf("town start: party of %d", len(f.Carried))
	}
	town := secondKitLine(f, f.Carried[0])
	t.Logf("town start (default hero): %s", town)
	class, female := defaultChargenAxes()
	chosen := ui.ChargenResult{Name: "Hero", Choices: []int{int(boolInt32(female)), int(boolInt32(class)), int(PartySkillSlot() - data.SkillBlade)}}
	if err := f.startGeneratedTown(chosen); err != nil {
		t.Fatal(err)
	}
	if got := secondKitLine(f, f.Carried[0]); got != town {
		t.Fatalf("town start kit %s, generator kit of the default row %s", town, got)
	}
	for _, mage := range []int{0, 1} {
		for _, female := range []int{0, 1} {
			for skill := range data.SkillSlots - 1 {
				res := ui.ChargenResult{Name: "Hero", Choices: []int{female, mage, skill}, Stats: []int{31, 27, 24, 29}}
				if err := f.startGeneratedTown(res); err != nil {
					t.Fatal(err)
				}
				if len(f.Carried) != 1 {
					t.Fatalf("generated start: party of %d", len(f.Carried))
				}
				t.Logf("generated mage=%d female=%d skill=%d: %s", mage, female, skill, secondKitLine(f, f.Carried[0]))
			}
		}
	}
}
