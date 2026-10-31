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

// previousRosterName is the former mission party naming, the oracle.
func previousRosterName(in *InstallResources, p mapload.PartyMember) string {
	nameIndex := p.CompanionNPC - 1
	if p.CompanionNPC >= 21 && p.CompanionNPC <= 24 {
		selector := -1
		for i, base := range data.ChargenBaseNames() {
			if p.Name == base || strings.HasPrefix(p.Name, base+"_") {
				selector = i
				break
			}
		}
		if selector < 0 {
			selector = 0
			if data.FigureDir(p.FigureDir).Female() {
				selector++
			}
			if p.Mage {
				selector += 2
			}
		}
		nameIndex = 20 + selector
	}
	if p.CompanionNPC != 0 {
		return in.localizedNPCName(nameIndex, p.Name)
	} else if name, ok := in.modCharacterName(p.Name); ok {
		return name
	} else if index := int(p.Class); index >= 0 && index < len(in.Words.UnitNames) && in.Words.UnitNames[index] != "" {
		return in.Words.UnitNames[index]
	}
	return p.Name
}

// previousTownName is the former town candidate naming, the oracle.
func previousTownName(in *InstallResources, p mapload.PartyMember, nameIndex int, words ui.Words) string {
	name, _ := ui.PanelSubjectName(ui.PanelSubject{UnitNameIndex: nameIndex, Words: words})
	if modName, ok := in.Table.Mods.CharacterName(p.Name); ok {
		name = modName
	}
	return name
}

func definitionNameFixture(t *testing.T, game string) *FrontEnd {
	t.Helper()
	npc := map[int]string{}
	for i := 0; i < 30; i++ {
		if i%5 != 3 {
			npc[i] = fmt.Sprintf("%s companion %d", game, i)
		}
	}
	f := localizedJoinFront(t, npc)
	f.Table = &mapload.Table{Mods: mapload.ModContext{Characters: []mapload.ModCharacter{
		{Mod: "m", Rows: []string{"M_Brigand3", "PC_Naira"}, Name: game + " mod brigand"},
	}}}
	for i := range f.Words.UnitNames {
		if i%6 != 4 {
			f.Words.UnitNames[i] = fmt.Sprintf("%s class %d", game, i)
		}
	}
	return f
}

func definitionNameMembers() []mapload.PartyMember {
	var out []mapload.PartyMember
	names := append([]string{"", "M_Brigand3", "PC_Naira", "NPC06_1", "Someone"}, data.ChargenBaseNames()...)
	names = append(names, data.ChargenBaseNames()[0]+"_2")
	for _, name := range names {
		for _, class := range []int32{-1, 0, 3, 4, 42, 80, 81} {
			out = append(out, mapload.PartyMember{Name: name, Class: class})
		}
		for _, companion := range []int{-1, 1, 4, 20, 21, 22, 23, 24, 25, 40} {
			for look := 0; look < 4; look++ {
				p := mapload.PartyMember{CompanionNPC: companion, Name: name, Class: 3, Mage: look&2 != 0}
				if look&1 != 0 {
					p.FigureDir = string(data.FigureDirWomanFighter)
				}
				out = append(out, p)
			}
		}
	}
	return out
}

func TestDefinitionNameGivesTodaysRosterAndTownNames(t *testing.T) {
	for _, game := range []string{"first", "second"} {
		f := definitionNameFixture(t, game)
		in := &f.InstallResources
		for _, p := range definitionNameMembers() {
			if got, want := in.definitionName(p, int(p.Class), &in.Words, p.Name), previousRosterName(in, p); got != want {
				t.Fatalf("%s roster %+v: definitionName = %q, want %q", game, p, got, want)
			}
			if p.CompanionNPC != 0 {
				continue
			}
			for _, words := range []ui.Words{in.Words, ui.AuthoredWords()} {
				if got, want := in.definitionName(p, int(p.Class), &words, ""), previousTownName(in, p, int(p.Class), words); got != want {
					t.Fatalf("%s town %+v: definitionName = %q, want %q", game, p, got, want)
				}
			}
		}
		roster := map[sim.EntityID]mapload.PartyMember{}
		want := map[sim.EntityID]string{}
		for i, p := range definitionNameMembers() {
			id := sim.EntityID(i + 1)
			roster[id] = p
			want[id] = previousRosterName(in, p)
		}
		ms := &Mission{Start: mapload.Start{Roster: roster}}
		f.canonicalizeJoinedRoster(ms)
		for id, name := range want {
			if ms.Start.Roster[id].Name != name {
				t.Fatalf("%s roster member %d named %q, want %q", game, id, ms.Start.Roster[id].Name, name)
			}
		}
	}
}
