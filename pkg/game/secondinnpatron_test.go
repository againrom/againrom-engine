package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func TestRoomPersonCardUsesTheGeneratorRow(t *testing.T) {
	for _, tc := range []struct {
		id         string
		statistics bool
	}{
		{base.ROM1EN, false},
		{base.ROM2EN, true},
	} {
		t.Run(tc.id, func(t *testing.T) {
			profile, ok := base.Find(tc.id)
			if !ok {
				t.Fatal("missing profile")
			}
			f := shellFrontEnd()
			f.Archives = &Archives{Base: base.Match{Profile: profile}}
			font, generatorFont := missionFont(), missionFont()
			f.Font = resolved(font, nil)
			f.ChargenAssets = &ChargenAssets{Presentation: &ui.ChargenPresentation{Font: generatorFont}}
			figure := image.NewRGBA(image.Rect(0, 0, 2, 3))
			subject := ui.PanelSubject{Name: "Seated person"}
			card := f.townUI.roomPersonCard(subject, figure, true)
			if !card.HasSubject || card.Subject.Name != subject.Name || card.Statistics != tc.statistics {
				t.Fatalf("room card subject/presentation: %+v", card)
			}
			wantFont, wantFigure := font, figure
			if tc.statistics {
				wantFont, wantFigure = generatorFont, nil
			}
			if card.Font != wantFont || card.Figure != wantFigure {
				t.Fatal("room card changed its font or figure placement")
			}
		})
	}
}

func TestInnActorFindsATalkPatronInTheUnits(t *testing.T) {
	input := innTalkPatronInput()
	got := innActor(secondInnProfile(), 681, input)
	if !got.known || got.member.Name != "Druid_Preacher" || got.member.Class != 100 || got.member.FigureFace != 7 {
		t.Fatalf("NPC 681: known %v, %q class %d face %d", got.known, got.member.Name, got.member.Class, got.member.FigureFace)
	}
	if m := got.member; !m.UnitMember() || m.DefinitionRow != 1 {
		t.Fatalf("NPC 681 reads no Units row for its card: units %v row %d", m.UnitMember(), m.DefinitionRow)
	}
	if got := innActor(secondInnProfile(), 682, input); got.known {
		t.Fatalf("a key no row holds resolved to %q", got.member.Name)
	}
}

func innTalkPatronInput() innActorInput {
	row := make([]int32, 56)
	for i := range row {
		row[i] = -1
	}
	row[0], row[4], row[29], row[30], row[31], row[32], row[55] = 40, 200, 100, 7, 1, 1, 681
	tbl := &mapload.Table{Humans: dbCollection{{}}, Units: dbCollection{{}, {name: "Druid_Preacher", params: row}}}
	return innActorInput{campaign: innAt(3, 100, nil), install: &InstallResources{Table: tbl}}
}

func TestTalkPatronCardReadsAsACreature(t *testing.T) {
	input := innTalkPatronInput()
	got := innActor(secondInnProfile(), 681, input)
	if !got.known {
		t.Fatal("NPC 681 did not resolve")
	}
	f := shellFrontEnd()
	f.Table = input.install.Table
	c := f.townUI.buildRoomPatron(got.member, 681, roomPersonProfiles["campaign"], nil).subject.Char
	if c.Sight256 != 0 || c.Band != ui.CharacterBandCreature {
		t.Fatalf("talk patron card: sight word %d, band %v", c.Sight256, c.Band)
	}
}

// An inn patron's own name is the npcnames entry of a key of 1000 or more;
// a smaller key has none and the sheet heads the card with Mercenary over
// the type (R2-ENGINE-581, DIV-4087).
func TestSecondInnPersonName(t *testing.T) {
	lines := make([]string, 2200)
	lines[6], lines[16] = "Mercenary", "Mercenary"
	lines[2107], lines[2108], lines[2109] = "Peasant", "Neighbour", "Guard"
	fsys, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fsys.Overlay(npcNamesPath, func() ([]byte, error) { return []byte(strings.Join(lines, "\r\n")), nil })
	in := &InstallResources{Archives: &Archives{Containers: fsys}}
	for npc, want := range map[int]string{
		207: "", 517: "", 7: "", 17: "", 2108: "Peasant", 2109: "Neighbour", 2110: "Guard", 3000: "", 5000: "",
	} {
		if got := secondInnPersonName(in, npc); got != want {
			t.Errorf("key %d: %q, want %q", npc, got, want)
		}
	}
}

func TestSecondInnPatronNamesInstalled(t *testing.T) {
	archives := secondGameRoot(t)
	in := &InstallResources{Archives: archives}
	payload, err := archives.Containers.ReadFile(npcNamesPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(in.textCode().Bytes(payload)), "\r\n", "\n"), "\n")
	for _, npc := range []int{2108, 2110} {
		want := lines[npc-1]
		if want == "" {
			t.Fatalf("NPC %d has an empty installed line", npc)
		}
		got := secondInnPersonName(in, npc)
		t.Logf("NPC %d reads %q (line %d of %d; neighbours %q, %q)", npc, got, npc, len(lines), lines[npc-2], lines[npc])
		if got != want {
			t.Errorf("NPC %d reads %q, want its own line %q", npc, got, want)
		}
	}
}

// The SPELLCASTER caption's actor value: the first game's person's spell mask
// behind a positive mana maximum (TAVERN-035) and a creature's projected value
// (DIV-1809); the second game's projected value held at zero without a mana
// pool, whatever the mask (R2-ENGINE-580, DIV-4086).
func TestSpellcasterValue(t *testing.T) {
	for _, c := range []struct {
		files       *gameFiles
		value, mana int32
		spells      uint32
		person      bool
		want        int32
		what        string
	}{
		{&firstGameFiles, 1079, 0, 0, false, 1079, "first game creature, no mana"},
		{&firstGameFiles, 1079, 30, 0, false, 1079, "first game creature, mana"},
		{&firstGameFiles, 1079, 30, 0, true, 0, "first game person, no spell"},
		{&firstGameFiles, 1079, 0, 1 << 3, true, 0, "first game person, spell, no mana"},
		{&firstGameFiles, 1079, 120, 1<<3 | 1, true, 1 << 3, "first game person, spells"},
		{&firstGameFiles, 1079, 120, 1, true, 0, "first game person, id 0 only"},
		{&secondGameFiles, 1079, 0, 1 << 3, true, 0, "second game, no mana"},
		{&secondGameFiles, 250536, 604, 0, true, 250536, "second game, mana, no spell"},
		{&secondGameFiles, 250536, 604, 1 << 3, false, 250536, "second game creature, mana"},
	} {
		if got := c.files.spellcasterValue(c.value, c.mana, c.spells, c.person); got != c.want {
			t.Errorf("%s: %d, want %d", c.what, got, c.want)
		}
	}
}
