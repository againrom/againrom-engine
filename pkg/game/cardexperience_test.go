package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// cardExperienceRow reports whether a card states the XP line, by the label
// the card itself draws.
func cardExperienceRow(subject ui.PanelSubject) bool {
	label := subject.Words.PanelCaptions[46]
	if label == "" {
		label = "XP"
	}
	for _, row := range ui.PanelStatement(ui.CompactPanelLayout(nil), subject) {
		if strings.HasPrefix(row, label+" ") {
			return true
		}
	}
	return false
}

// The mission card states XP for a player-character-band actor and omits it
// for a person and a creature outside the band (DIV-3613). The fixture's three
// source actors are a creature (type 35), a person (type 3) and a band person
// (type 33); the band person is the loss control.
func TestTheMissionCardStatesExperienceOnlyForTheBand(t *testing.T) {
	f := actorRegistryFront(t)
	f.SetDeterministicFrames(true)
	raw := actorRegistrySave(f.Table.Units.EntryName(1), f.Table.Humans.EntryName(1))
	mission, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("original LOAD: town=%t err=%v", town, err)
	}
	app := f.App("card experience")
	app.Layout(1024, 768)
	if err := app.OpenMission(mission); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	hash := f.live.world.Hash()
	seen := map[int32]bool{}
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class == 0 {
			continue
		}
		subject := cardCaptionFixtureSubject(t, f, app, e.ID)
		if !subject.Char.Known {
			t.Fatalf("type %d: the card knows no character", e.TypeID)
		}
		band := sim.InPersistBand(e.TypeID)
		if got := cardExperienceRow(subject); got != band {
			t.Errorf("type %d: XP line %v, want %v", e.TypeID, got, band)
		}
		seen[e.TypeID] = true
	}
	if want := map[int32]bool{3: true, 33: true, 35: true}; !reflect.DeepEqual(seen, want) {
		t.Fatalf("fixture types %v, want %v", seen, want)
	}
	if f.live.world.Hash() != hash {
		t.Fatal("the card projection changed the World")
	}
}

// The town card of a hired member omits XP; a generated hero's states it.
func TestTheTownCardOmitsExperienceForAHiredMember(t *testing.T) {
	hero := mapload.PartyMember{ID: "hero", Name: "Hero"}
	if s := partyPanelSubject(hero, nil); s.Char.ExperienceWithheld || !cardExperienceRow(s) {
		t.Fatalf("a generated hero's town card withholds XP: %+v", s.Char)
	}
	hired := mapload.PartyMember{ID: "NPC09_1", Name: "NPC09_1", MercenaryType: 8, Class: 0x08}
	if s := partyPanelSubject(hired, nil); !s.Char.ExperienceWithheld || cardExperienceRow(s) {
		t.Fatalf("a hired member's town card states XP: %+v", s.Char)
	}
}
