package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestMapMessageHeroAudienceCellsKeepTheirIdentity(t *testing.T) {
	rows := []struct {
		kind         herokind.Kind
		female, mage bool
		bit          ui.HeroAudience
		cell         int
		word, dir    string
		literals     []string
	}{
		{herokind.FemaleMage, true, true, 1, 0, "editor.message.hero.womanmage", "fmage", []string{"iamfemale", "iammage"}},
		{herokind.FemaleFighter, true, false, 2, 1, "editor.message.hero.womanfighter", "ffighter", []string{"iamfemale", "iamfighter"}},
		{herokind.MaleMage, false, true, 4, 2, "editor.message.hero.manmage", "mmage", []string{"iammale", "iammage"}},
		{herokind.MaleFighter, false, false, 8, 3, "editor.message.hero.manfighter", "mfighter", []string{"iammale", "iamfighter"}},
	}
	_, editor := workspaceFixture(t)
	s := editor.Document().Edits.(*mapInspectionEdits)
	audiences := s.heroAudiences()
	cells := ui.HeroCells(true, true)
	if len(cells) != len(rows) {
		t.Fatalf("HeroCells = %v, want four cells", cells)
	}
	sp := s.missionStart()
	s.inspector.table = &mapload.Table{}
	s.inspector.archives = &Archives{}
	sp.table, sp.world = s.inspector.table, &sim.World{}
	sp.records = map[int32]data.NPCFace{22: {}}
	sp.heroFaces = map[heroFaceKey]*image.RGBA{}
	for _, row := range rows {
		t.Run(row.kind.String(), func(t *testing.T) {
			if bit := ui.HeroOf(row.mage, row.female); bit != row.bit || cells[row.cell] != row.bit {
				t.Errorf("bit %d, cell %d = %d; want %d", bit, row.cell, cells[row.cell], row.bit)
			}
			if herokind.AudienceOrder[row.cell] != row.kind || row.kind.In(herokind.AudienceOrder) != row.cell {
				t.Errorf("audience order moved %v from cell %d", row.kind, row.cell)
			}
			if word := ui.HeroWord(row.bit); word != row.word {
				t.Errorf("word = %q, want %q", word, row.word)
			}
			party := plantedHero(row.kind)
			if len(party) != 1 || party[0].Mage != row.mage || party[0].FigureDir != row.dir || party[0].FigureFace != 1 || !party[0].PlayerCharacter || !party[0].StartingHero {
				t.Fatalf("planted hero = %+v", party)
			}
			aud := audiences[row.cell]
			if aud.HeroFemale != row.female || aud.HeroMage != row.mage {
				t.Errorf("cell audience = %+v", aud)
			}
			if helper := heroAudience(row.cell); helper.HeroFemale != row.female || helper.HeroMage != row.mage {
				t.Errorf("test reader's cell audience = %+v", helper)
			}
			if literals := heroLiterals(row.bit, true, true); !slices.Equal(literals, row.literals) {
				t.Errorf("literals = %v, want %v", literals, row.literals)
			}
			pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
			pic.Pix[0] = uint8(row.cell + 1)
			sp.heroFaces[heroFaceKey{22, row.cell}] = pic
			if got, ok := s.heroFace(22, row.bit); !ok || got != pic {
				t.Errorf("cell's cached face = %p, %v; want %p", got, ok, pic)
			}
		})
	}
}
