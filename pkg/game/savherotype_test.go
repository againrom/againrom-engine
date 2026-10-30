package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// heroTypeParty is a new campaign's party of the four player-character
// archetypes and one hired man. The literal type each hero must carry is the
// player-character constructor's: the gender plus 0x23 for a mage or 0x21
// otherwise (HERO-CLASS-013, PARTY-M20-031), with R0841's female set {0x22,
// 0x24}. The hired man keeps his own row's type.
func heroTypeParty(f *FrontEnd) ([]mapload.PartyMember, []uint32) {
	base := MissionParty(nil, data.BodyList{}, f.Table)[0]
	party := make([]mapload.PartyMember, 0, 5)
	want := []uint32{0x21, 0x22, 0x23, 0x24, 7}
	for i, hero := range []struct {
		name         string
		mage, female bool
	}{{"Danath", false, false}, {"Naira", false, true}, {"Fergard", true, false}, {"Reniesta", true, true}} {
		p := base
		p.ID, p.Name, p.Mage, p.FigureDir = []string{"hero", "join:1", "join:2", "join:3"}[i], hero.name, hero.mage, string(data.FigureDirFor(hero.mage, hero.female))
		p.StartingHero = i == 0
		party = append(party, p)
	}
	hired := base
	hired.ID, hired.Name, hired.Class, hired.MercenaryType, hired.Temporary, hired.PlayerCharacter = "mercenary:7:1", "Hire", 7, 7, true, false
	return append(party, hired), want
}

func heroTypeOpen(t *testing.T, f *FrontEnd, opener func() error) {
	t.Helper()
	if err := opener(); err != nil {
		t.Fatal(err)
	}
	if f.live == nil || f.live.mission == nil || len(f.live.mission.state.Start.IDs) != 5 {
		t.Fatal("fixture party did not open")
	}
}

// heroTypeSave makes the game SAVE and answers its document and the T0E of
// each party member's record, in party order.
func heroTypeSave(t *testing.T, f *FrontEnd) (sav.DocumentData, []uint32) {
	t.Helper()
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	var types []uint32
	for _, id := range f.live.mission.state.Start.IDs {
		v, err := savedStructureValue(generatedActorRecord(t, &doc, id), "T0E")
		if err != nil {
			t.Fatal(err)
		}
		types = append(types, v)
	}
	return doc, types
}

// heroTypeLoad LOADs doc after setting each party record's T0E to loaded.
func heroTypeLoad(t *testing.T, f *FrontEnd, doc sav.DocumentData, loaded []uint32) *FrontEnd {
	t.Helper()
	for i, id := range f.live.mission.state.Start.IDs {
		if err := savedActorSetValue(generatedActorRecord(t, &doc, id), "T0E", loaded[i]); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	back := currentPoolFixtureFront(t, 91, 92)
	opener, town, err := back.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("LOAD", err, town)
	}
	heroTypeOpen(t, back, func() error { return back.App("hero type LOAD").OpenMission(opener) })
	return back
}

// A new campaign's first SAVE writes each hero's minted type, the hired man's
// row type, and the same value the World holds for each of them.
func TestSaveNewCampaignWritesEachHeroMintedType(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	party, want := heroTypeParty(f)
	heroTypeOpen(t, f, func() error { return f.App("hero type").OpenMission(f.MissionOpenerWith(10, party)) })
	_, got := heroTypeSave(t, f)
	for i, id := range f.live.mission.state.Start.IDs {
		e, _ := f.live.world.Entity(id)
		if got[i] != want[i] || got[i] != uint32(e.TypeID) {
			t.Errorf("member %d %s: T0E %d, World type %d, want %d", i, party[i].Name, got[i], e.TypeID, want[i])
		}
	}
}

// A loaded party whose records carry a type that does not match each hero is
// saved with each hero's own type; the hired man keeps the type he loaded with.
func TestSaveWritesEachHeroTypeFromTheHeroNotTheLoadedRecord(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	party, want := heroTypeParty(f)
	heroTypeOpen(t, f, func() error { return f.App("hero type").OpenMission(f.MissionOpenerWith(10, party)) })
	doc, _ := heroTypeSave(t, f)
	wrong := []uint32{0x21, 0x21, 0x21, 0x21, 9}
	back := heroTypeLoad(t, f, doc, wrong)
	for i, id := range back.live.mission.state.Start.IDs {
		if e, _ := back.live.world.Entity(id); uint32(e.TypeID) != wrong[i] {
			t.Fatalf("fixture: member %d loaded with World type %d, want the record's %d", i, e.TypeID, wrong[i])
		}
	}
	_, got := heroTypeSave(t, back)
	want[4] = wrong[4]
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("member %d %s: T0E %d, want %d", i, party[i].Name, got[i], want[i])
		}
	}
}

// A loaded party whose records carry the right types is saved with them
// unchanged.
func TestSaveKeepsAHeroTypeThatMatchesTheHero(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	party, want := heroTypeParty(f)
	heroTypeOpen(t, f, func() error { return f.App("hero type").OpenMission(f.MissionOpenerWith(10, party)) })
	doc, _ := heroTypeSave(t, f)
	back := heroTypeLoad(t, f, doc, want)
	_, got := heroTypeSave(t, back)
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("member %d %s: T0E %d, want %d", i, party[i].Name, got[i], want[i])
		}
	}
}
