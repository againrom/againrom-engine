package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// heroChoiceMods are second-game mods over the inn rows chosen by the
// starting hero: none, records that name zero alone, nonzero alone, both
// and neither.
var heroChoiceMods = []struct{ id, towns string }{
	{"none", ""},
	{"zero", "[[inn]]\ntopic = 75\nnpc = 2022\nzero = [970, 971, 972, 973]\n\n[[inn]]\ntopic = 82\nnpc = 2009\nzero = [978]\n\n[[inn]]\ntopic = 86\nnpc = 2022\nzero = [980, 981]\n"},
	{"nonzero", "[[inn]]\ntopic = 76\nnpc = 2022\nnonzero = [959]\n\n[[inn]]\ntopic = 94\nnpc = 2022\nnonzero = [980]\n\n[[inn]]\ntopic = 81\nnpc = 2010\nnonzero = [959]\n"},
	{"both", "[[inn]]\ntopic = 74\nnpc = 2022\nzero = [970, 971]\nnonzero = [959]\n\n[[inn]]\ntopic = 87\nnpc = 2022\nzero = [980, 981, 982, 983]\nnonzero = [970]\n"},
	{"neither", "[[inn]]\ntopic = 76\nnpc = 2022\ntown = 2\n\n[[inn]]\ntopic = 93\nnpc = 2022\ntown = 2\n"},
}

// A mod record over an inn row chosen by the starting hero leaves the row's
// hero condition in class and sex: the part a list the record sets held
// goes, the rest stays, and no gate of a layered row reads a hero slot.
func TestModRecordOnAHeroInnRowKeepsItsHeroAsClassAndSex(t *testing.T) {
	type hero struct{ class, sex string }
	want := map[string]map[int]hero{
		"zero":    {75: {"", "female"}, 82: {"", ""}, 86: {"mage", "female"}, 74: {"fighter", "male"}},
		"nonzero": {76: {"", ""}, 94: {"fighter", ""}, 81: {"", ""}, 77: {"mage", "male"}},
		"both":    {74: {"", ""}, 87: {"", ""}, 85: {"fighter", "female"}},
		"neither": {76: {"mage", "female"}, 93: {"fighter", "male"}},
	}
	slots := townBooks["rom2"].catalog.HeroSlots
	for _, m := range heroChoiceMods[1:] {
		dir := t.TempDir()
		writeInnMod(t, dir, m.id, m.towns)
		book, _, err := layerTownBook(townBooks["rom2"], runTownModsFor(t, "rom2-en", dir, m.id).Towns, Campaign{}, nil, nil, modTownInstall{})
		if err != nil {
			t.Fatalf("%s: layer: %v", m.id, err)
		}
		for _, r := range book.offerRows() {
			for _, gate := range r.Gates() {
				if slices.Contains(gate, slots.Mage) || slices.Contains(gate, slots.Female) {
					t.Errorf("%s: row %d gates on a hero slot: %+v", m.id, r.Topic, r)
				}
			}
			if w, ok := want[m.id][r.Topic]; ok && (r.Class != w.class || r.Sex != w.sex) {
				t.Errorf("%s: row %d reads class %q sex %q, want %q %q", m.id, r.Topic, r.Class, r.Sex, w.class, w.sex)
			}
		}
	}
}

// Every choice the game makes by the starting hero: the second game's inn
// rows chosen by the hero for every value of slots 776 and 781 the inn
// tests read, under each mod of heroChoiceMods, at each family's gate and
// at stage 80; and the mod mission after a win of mission 20 for each of
// the four heroes, marked behind a companion of the other class and sex,
// unmarked at the party's head, and saved by a chain's waiting party.
func TestEveryGameChoiceByTheStartingHeroReadsTheSameHero(t *testing.T) {
	var got []string
	for _, m := range heroChoiceMods {
		rows := townBooks["rom2"].offerRows()
		if m.towns != "" {
			dir := t.TempDir()
			writeInnMod(t, dir, m.id, m.towns)
			book, overrides, err := layerTownBook(townBooks["rom2"], runTownModsFor(t, "rom2-en", dir, m.id).Towns, Campaign{}, nil, nil, modTownInstall{})
			if err != nil || len(overrides) != 0 {
				t.Fatalf("%s: layer: %v, overrides %v", m.id, err, overrides)
			}
			rows = book.offerRows()
		}
		for _, v := range heroValues {
			line := fmt.Sprintf("%s 776=%d 781=%d:", m.id, v[0], v[1])
			for _, at := range []struct {
				stage int32
				gate  int
			}{{20, 959}, {20, 970}, {20, 980}, {80, 0}, {80, 959}} {
				bank := map[int]int32{776: v[0], 781: v[1]}
				if at.gate > 0 {
					bank[at.gate] = 1
				}
				c := innAt(2, at.stage, bank)
				c.offers = rows
				var topics []string
				for _, o := range c.innOptions() {
					if o.npc == 2022 || o.topic == 81 || o.topic == 82 {
						topics = append(topics, fmt.Sprint(o.topic))
					}
				}
				line += fmt.Sprintf(" %d/%d=[%s]", at.stage, at.gate, strings.Join(topics, " "))
			}
			got = append(got, line)
		}
	}
	missions, err := heroMissionMod(t, func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	for _, mage := range []bool{false, true} {
		for _, female := range []bool{false, true} {
			dir := data.FigureDirFor(mage, female)
			lead := mapload.PartyMember{ID: "hero", StartingHero: true, PlayerCharacter: true, Mage: mage, FigureDir: string(dir)}
			guest := mapload.PartyMember{ID: "npc:22", PlayerCharacter: true, Mage: !mage, FigureDir: string(data.FigureDirFor(!mage, !female))}
			unmarked := lead
			unmarked.StartingHero = false
			saved := &guestMission{hero: &guestHero{Mage: mage, FigureDir: string(dir)}}
			line := fmt.Sprintf("%s:", dir)
			for _, party := range [][]mapload.PartyMember{{guest, lead}, {unmarked, guest}, saved.heroParty([]mapload.PartyMember{guest}), (*guestMission)(nil).heroParty([]mapload.PartyMember{guest, lead})} {
				m, ok := missions.AfterWin(20, mapload.StartingHeroKind(mapload.HeroHold{Party: party}))
				line += fmt.Sprintf(" %s/%v", m.Key, ok)
			}
			got = append(got, line)
		}
	}
	want := []string{
		"none 776=0 781=0: 20/959=[74] 20/970=[84] 20/980=[93] 80/0=[82] 80/959=[82 74]",
		"none 776=1 781=0: 20/959=[77] 20/970=[87] 20/980=[96] 80/0=[81] 80/959=[81 77]",
		"none 776=0 781=1: 20/959=[75] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[82 75]",
		"none 776=1 781=1: 20/959=[76] 20/970=[86] 20/980=[95] 80/0=[81] 80/959=[81 76]",
		"none 776=-1 781=7: 20/959=[76] 20/970=[86] 20/980=[95] 80/0=[81] 80/959=[81 76]",
		"none 776=2 781=0: 20/959=[77] 20/970=[87] 20/980=[96] 80/0=[81] 80/959=[81 77]",
		"none 776=0 781=-2: 20/959=[75] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[82 75]",
		"zero 776=0 781=0: 20/959=[74] 20/970=[84] 20/980=[93] 80/0=[82] 80/959=[82 74]",
		"zero 776=1 781=0: 20/959=[77] 20/970=[87] 20/980=[96] 80/0=[81 82] 80/959=[81 82 77]",
		"zero 776=0 781=1: 20/959=[75] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[82 75]",
		"zero 776=1 781=1: 20/959=[75 76] 20/970=[86] 20/980=[95] 80/0=[81 82] 80/959=[81 82 75 76]",
		"zero 776=-1 781=7: 20/959=[75 76] 20/970=[86] 20/980=[95] 80/0=[81 82] 80/959=[81 82 75 76]",
		"zero 776=2 781=0: 20/959=[77] 20/970=[87] 20/980=[96] 80/0=[81 82] 80/959=[81 82 77]",
		"zero 776=0 781=-2: 20/959=[75] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[82 75]",
		"nonzero 776=0 781=0: 20/959=[74 76] 20/970=[84] 20/980=[93 94] 80/0=[82] 80/959=[81 82 74 76]",
		"nonzero 776=1 781=0: 20/959=[76 77] 20/970=[87] 20/980=[96] 80/0=[] 80/959=[81 76 77]",
		"nonzero 776=0 781=1: 20/959=[75 76] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[81 82 75 76]",
		"nonzero 776=1 781=1: 20/959=[76] 20/970=[86] 20/980=[95] 80/0=[] 80/959=[81 76]",
		"nonzero 776=-1 781=7: 20/959=[76] 20/970=[86] 20/980=[95] 80/0=[] 80/959=[81 76]",
		"nonzero 776=2 781=0: 20/959=[76 77] 20/970=[87] 20/980=[96] 80/0=[] 80/959=[81 76 77]",
		"nonzero 776=0 781=-2: 20/959=[75 76] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[81 82 75 76]",
		"both 776=0 781=0: 20/959=[74] 20/970=[84 87] 20/980=[93] 80/0=[82] 80/959=[82 74]",
		"both 776=1 781=0: 20/959=[74 77] 20/970=[87] 20/980=[96] 80/0=[81] 80/959=[81 74 77]",
		"both 776=0 781=1: 20/959=[74 75] 20/970=[85 87] 20/980=[94] 80/0=[82] 80/959=[82 74 75]",
		"both 776=1 781=1: 20/959=[74 76] 20/970=[86 87] 20/980=[95] 80/0=[81] 80/959=[81 74 76]",
		"both 776=-1 781=7: 20/959=[74 76] 20/970=[86 87] 20/980=[95] 80/0=[81] 80/959=[81 74 76]",
		"both 776=2 781=0: 20/959=[74 77] 20/970=[87] 20/980=[96] 80/0=[81] 80/959=[81 74 77]",
		"both 776=0 781=-2: 20/959=[74 75] 20/970=[85 87] 20/980=[94] 80/0=[82] 80/959=[82 74 75]",
		"neither 776=0 781=0: 20/959=[74] 20/970=[84] 20/980=[93] 80/0=[82] 80/959=[82 74]",
		"neither 776=1 781=0: 20/959=[77] 20/970=[87] 20/980=[96] 80/0=[81] 80/959=[81 77]",
		"neither 776=0 781=1: 20/959=[75] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[82 75]",
		"neither 776=1 781=1: 20/959=[76] 20/970=[86] 20/980=[95] 80/0=[81] 80/959=[81 76]",
		"neither 776=-1 781=7: 20/959=[76] 20/970=[86] 20/980=[95] 80/0=[81] 80/959=[81 76]",
		"neither 776=2 781=0: 20/959=[77] 20/970=[87] 20/980=[96] 80/0=[81] 80/959=[81 77]",
		"neither 776=0 781=-2: 20/959=[75] 20/970=[85] 20/980=[94] 80/0=[82] 80/959=[82 75]",
		"mfighter: fighter/true fighter/true fighter/true fighter/true",
		"ffighter: fighter/true fighter/true fighter/true fighter/true",
		"mmage: mage_male/true mage_male/true mage_male/true mage_male/true",
		"fmage: mage_female/true mage_female/true mage_female/true mage_female/true",
	}
	if !slices.Equal(got, want) {
		for _, l := range got {
			t.Log(l)
		}
		t.Fatalf("%d choices differ from the recorded %d", len(got), len(want))
	}
}
