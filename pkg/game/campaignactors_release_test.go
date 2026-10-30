package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func actorWitnessReceipt(t *testing.T, name string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %x", name, sha256.Sum256(b))
	if out := os.Getenv("CAMPAIGN_ACTOR_OUT"); out != "" {
		if err := os.MkdirAll(out, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, name+".json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func actorWitnessFrame(t *testing.T, app *ui.App, name string) {
	t.Helper()
	p, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	actorWitnessReceipt(t, name, sha256.Sum256(p.Pix))
}

func actorWitnessPatron(t *testing.T, s *townScreen, npc int) *secondInnPatron {
	t.Helper()
	d, room, ok := s.composedRoom()
	if !ok {
		t.Fatal("no composed inn")
	}
	return s.innPatron(d.Room(room), s.roomArt(d, room).page, npc)
}

func TestReleaseCampaignActorLossControls(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("set AGAINROM_ASSETS to a lawful install")
	}
	if filepath.Base(os.Getenv("AGAINROM_ASSETS")) == "rom2-en" || filepath.Base(os.Getenv("AGAINROM_ASSETS")) == "rom2-ru" {
		for _, dina := range []bool{false, true} {
			f, app, _ := secondIglezJoins(t)
			name := "controlled"
			if dina {
				f.Town.second.bank[773] = 23
				f.CampaignSession.leftLocation(f.townInstall())
				name += "-dina"
			}
			actorWitnessReceipt(t, name+"-party", f.Carried)
			secondLaterChoose(t, app, "EXIT", "GATES")
			secondLaterEnter(t, f, app, 30)
			actorWitnessMission(t, f, app, name)
		}
		return
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Actor", Choices: []int{0, 0, 3}, Stats: []int{31, 27, 24, 29}})
	for _, chapter := range []int{30, 70, 110, 150} {
		town := NewTown(Campaign{})
		town.main = chapter
		for typ := 1; typ <= 15; typ++ {
			members, ok := buildMercenarySquad(f.Table, town, typ, 2)
			actorWitnessReceipt(t, fmt.Sprintf("squad-%d-%d", chapter, typ), struct {
				Members []mapload.PartyMember
				OK      bool
			}{members, ok})
		}
	}
	for sex := 0; sex < 2; sex++ {
		for class := 0; class < 2; class++ {
			hero := f.ChargenParty(ui.ChargenResult{Name: "Actor", Choices: []int{sex, class, 3}, Stats: []int{31, 27, 24, 29}})
			for _, chapter := range []int{30, 70, 110, 150} {
				for _, npc := range []int{22, 23, 24, 25, 26} {
					member, ok := townCompanion(f.townInstall(), chapter, npc, hero)
					actorWitnessReceipt(t, fmt.Sprintf("companion-%d-%d-%d-%d", sex, class, chapter, npc), struct {
						Member mapload.PartyMember
						OK     bool
					}{member, ok})
				}
			}
		}
	}
	f.Town = NewTown(f.Campaign.Value())
	f.Town.Won(10)
	f.Town.Won(20)
	f.arriveInTown()
	f.addChapterCompanions(f.Town.Chapter())
	for typ := 1; typ <= 15; typ++ {
		members, ok := buildMercenarySquad(f.Table, f.Town, typ, 1)
		if ok {
			f.Carried = mapload.OwnParty(append(f.Carried, members...))
		}
	}
	actorWitnessReceipt(t, "first-party", f.Carried)
	app := f.App("campaign actor control")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(30, f.Carried)); err != nil {
		t.Fatal(err)
	}
	actorWitnessMission(t, f, app, "first")
}

func actorWitnessMission(t *testing.T, f *FrontEnd, app *ui.App, name string) {
	t.Helper()
	actorWitnessReceipt(t, name+"-world", f.live.world.Hash())
	for i, p := range f.live.mission.party {
		actorWitnessReceipt(t, fmt.Sprintf("%s-figure-%d", name, i), sha256.Sum256(memberPicture(t, f, p).Pix))
	}
	for _, draw := range f.live.entityDraws() {
		for i, id := range f.live.mission.ids {
			if uint32(id) == draw.ID {
				if draw.Frame == nil {
					t.Fatalf("party member %d draws no sprite frame", i)
				}
				encoded, err := json.Marshal(draw.Frame)
				if err != nil {
					t.Fatal(err)
				}
				actorWitnessReceipt(t, fmt.Sprintf("%s-sprite-%d", name, i), sha256.Sum256(encoded))
			}
		}
	}
	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := f.playerMissionSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	actorWitnessReceipt(t, name+"-sav", sha256.Sum256(b))
	if out := os.Getenv("CAMPAIGN_ACTOR_OUT"); out != "" {
		if err := os.WriteFile(filepath.Join(out, name+".sav"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseCampaignActorInnIdentity(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	secondLaterChoose(t, app, "TAVERN")
	s.TownSquareActive(true)
	actorWitnessSelect(t, app, s, 22)
	p := actorWitnessPatron(t, s, 22)
	wantName := f.townInstall().localizedNPC(21, "")
	if wantName == "" || p.subject.Name != wantName {
		t.Errorf("Iglez before talk is named %q, want %q", p.subject.Name, wantName)
	}
	if !p.known || p.figure == nil || !p.subject.Char.Known {
		t.Error("Iglez has no inn subject/figure before talk")
	}
	if len(f.Carried) != 1 {
		t.Fatal("Iglez joined before talk")
	}
	secondPageTalk(t, f, app, "TALK 22")
	actorWitnessSelect(t, app, s, 22)
	p = actorWitnessPatron(t, s, 22)
	if p.subject.Name != wantName {
		t.Errorf("Iglez after talk is named %q, want %q", p.subject.Name, wantName)
	}
	if !p.known || p.figure == nil || !p.subject.Char.Known {
		t.Error("Iglez has no inn subject/figure after talk")
	}
	if len(f.Carried) != 2 {
		t.Fatal("Iglez did not join at talk end")
	}
}

func actorWitnessSelect(t *testing.T, app *ui.App, s *townScreen, npc int) {
	t.Helper()
	list, _ := s.state().innSeats()
	d, room, _ := s.composedRoom()
	for i, offer := range list {
		if offer.npc != npc {
			continue
		}
		seat := d.Room(room).Seats.Seat(i)
		at := seat.Min.Add(seat.Size().Div(2))
		for _, edge := range []string{"hover", "press", "release"} {
			if err := app.HeadlessPointer(edge, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
		}
		for range 2 {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		if s.composed.seat != i {
			t.Fatalf("selected seat %d, want %d", s.composed.seat, i)
		}
		return
	}
	t.Fatalf("NPC %d has no seat", npc)
}

func TestReleaseCampaignActorHireRoute(t *testing.T) {
	for _, hire := range []int{1, 2, 3, 4, 5, 6, 7} {
		t.Run(fmt.Sprint(hire), func(t *testing.T) {
			f, app, s, _ := secondTownTwo(t)
			c := f.Town.second
			stage, town := 30, 2
			if hire >= 3 {
				stage, town = []int{0, 0, 0, 60, 70, 60, 60, 110}[hire], 3
			}
			c.bank[768] = int32(stage)
			c.current = partyPlace{2, town}
			c.bank[531+hire], c.bank[551+hire] = 1, int32(town)
			c.available = append(c.available, partyPlace{1, 30})
			secondLaterChoose(t, app, "TAVERN")
			s.TownSquareActive(true)
			actorWitnessSelect(t, app, s, hire)
			p := actorWitnessPatron(t, s, hire)
			wantClass := []int32{0, 15, 5, 100, 102, 101, 23, 26}[hire]
			wantName := []string{"", "2MercenaryXBowman30", "2MercenaryKnight30", "MercenaryDruid60", "MercenaryTroll70", "MercenaryOrc60", "2MercenaryMage60", "MercenaryCatapult"}[hire]
			var expected mapload.PartyMember
			if hire == 1 || hire == 2 || hire == 6 {
				row := data.FindHumanByName(f.Table.Humans, wantName)
				expected, _ = mapload.CharacterMember(f.Table, mapload.CharacterSpec{Row: row}, nil)
				expected.PlayerCharacter = false
			} else {
				for row := 1; row < f.Table.Units.Len(); row++ {
					if f.Table.Units.EntryName(row) == wantName {
						def, err := data.NewUnitDef(wantName, f.Table.Units.EntryParams(row))
						if err != nil {
							t.Fatal(err)
						}
						expected.Class, expected.FigureFace = def.TypeID, int(def.Face)
					}
				}
			}
			if !p.known || p.subject.UnitNameIndex != int(wantClass) || p.figure == nil {
				t.Errorf("hire %d inn class/figure: known=%v class=%d figure=%v", hire, p.known, p.subject.UnitNameIndex, p.figure != nil)
			}
			if p.figure == nil || !sameImage(p.figure, memberPicture(t, f, expected)) {
				t.Error("hire seat figure differs from its claimed template")
			}
			d, room, _ := s.composedRoom()
			r := d.Room(room)
			bound, err := screenArtOf(f.Archives.artSource()).Bind(r.Name, r.Art, r.Seats.Patron, wantClass)
			if err != nil || len(bound) == 0 || len(p.frames) == 0 || p.frames[0] == nil || !sameImage(p.frames[0], bound[0]) {
				t.Errorf("hire seat art does not resolve class %d: %v", wantClass, err)
			}
			secondLaterChoose(t, app, fmt.Sprintf("HIRE %d", hire))
			hireFace := memberPicture(t, f, expected)
			reference := s.townSpeakers()
			reference.campaignActors = nil
			lines := map[int]int{}
			for page := 0; page < 64; page++ {
				v := s.view()
				if !v.Open {
					break
				}
				if speaker, named := tagSpeaker(s.dialogue.tag); named && v.Page.Portrait {
					lines[speaker]++
					want := hireFace
					if speaker != hire {
						want, _, _ = reference.SpeakerFace(speaker)
					}
					if want != nil && (v.Page.Face == nil || !sameImage(v.Page.Face, want)) || speaker != hire && v.Page.Face != nil && sameImage(v.Page.Face, hireFace) {
						t.Errorf("accept page %d (%q): speaker %d does not draw its own face", page+1, s.dialogue.tag, speaker)
					}
				}
				if err := app.HeadlessActivate("notice"); err != nil {
					break
				}
			}
			// The installed npc5accept60 is empty.
			if hire != 5 && lines[hire] == 0 || hire == 2 && lines[22] == 0 {
				t.Errorf("accept dialogue portrait lines by speaker %v", lines)
			}
			secondInnNotice(t, app)
			secondLaterChoose(t, app, "EXIT", "GATES")
			secondLaterEnter(t, f, app, 30)
			if len(f.live.mission.party) != 2 {
				t.Fatalf("hire %d mission party=%d, want hero and one hire", hire, len(f.live.mission.party))
			}
			member := f.live.mission.party[1]
			id, ok := mapload.PartyIdentity(member, f.Table)
			if !ok || id != hire || member.MercenaryType != 0 {
				t.Fatalf("hire identity=%d/%v, MercenaryType=%d", id, ok, member.MercenaryType)
			}
			e, ok := f.live.world.Entity(f.live.mission.ids[1])
			if !ok || e.TypeID != wantClass {
				t.Fatalf("hire actor class=%d/%v, want %d", e.TypeID, ok, wantClass)
			}
			before := memberPicture(t, f, member)
			out := t.TempDir()
			secondMissionNamedSave(t, f, app, out, "Hire")
			cold, _ := secondMissionColdAt(t, out, secondSaveFile(t, out, "Hire"), 30)
			if len(cold.live.mission.party) != 2 {
				t.Fatal("cold LOAD duplicated or lost hire")
			}
			loaded := cold.live.mission.party[1]
			got, ok := mapload.PartyIdentity(loaded, cold.Table)
			if !ok || got != hire || loaded.ID != member.ID || !sameImage(before, memberPicture(t, cold, loaded)) {
				t.Fatalf("cold LOAD identity=%d/%v ID=%q->%q temporary=%v->%v class=%d->%d face=%d->%d figure=%q->%q name=%q->%q", got, ok, member.ID, loaded.ID, member.Temporary, loaded.Temporary, member.Class, loaded.Class, member.FigureFace, loaded.FigureFace, member.FigureDir, loaded.FigureDir, member.Name, loaded.Name)
			}
			if loaded.DefinitionRow != member.DefinitionRow {
				t.Errorf("cold LOAD definition row %d, fresh %d", loaded.DefinitionRow, member.DefinitionRow)
			}
			if cold.live.world.Hash() != f.live.world.Hash() {
				t.Error("cold LOAD changed World hash")
			}
		})
	}
}

func TestReleaseCampaignActorFireAfterTownLoad(t *testing.T) {
	for _, fired := range []int{1, 3} {
		t.Run(fmt.Sprint(fired), func(t *testing.T) {
			f, app, s := secondIglezJoins(t)
			secondLaterChoose(t, app, "EXIT")
			c := f.Town.second
			c.bank[768] = 60
			c.bank[532], c.bank[534], c.bank[552], c.bank[554] = 1, 1, 2, 2
			secondLaterChoose(t, app, "TAVERN")
			s.TownSquareActive(true)
			for _, id := range []int{1, 3} {
				secondLaterChoose(t, app, fmt.Sprintf("HIRE %d", id))
				secondInnNotice(t, app)
			}
			out := t.TempDir()
			secondTownNamedSave(t, f, app, out, "Selected hires")
			cold, coldApp := secondTownCold(t, out, secondSaveFile(t, out, "Selected hires"))
			cs := cold.TownScreen().(*townScreen)
			coldApp.SetTown(cs)
			cs.TownSquareActive(true)
			secondLaterChoose(t, coldApp, fmt.Sprintf("FIRE %d", fired))
			other := 4 - fired
			if cold.Town.second.bank[511+fired] != 0 || cold.Town.second.bank[511+other] != 1 || len(cold.Carried) != 2 || cold.Carried[1].CompanionNPC != 22 {
				t.Fatal("Fire changed another hire or the hero")
			}
			secondLaterChoose(t, coldApp, "EXIT", "GATES")
			secondLaterEnter(t, cold, coldApp, 30)
			if len(cold.live.mission.party) != 3 {
				t.Fatalf("Fire left mission party of %d", len(cold.live.mission.party))
			}
			id, ok := mapload.PartyIdentity(cold.live.mission.party[2], cold.Table)
			if !ok || id != other {
				t.Fatalf("Fire kept hire %d/%v, want %d", id, ok, other)
			}
		})
	}
}

func memberPicture(t *testing.T, f *FrontEnd, p mapload.PartyMember) *image.RGBA {
	t.Helper()
	v := composeMemberPortrait(f.Archives.Containers, f.Units, 0, mapload.EquipmentFromParty(p), p)
	if v.Figure == nil {
		t.Fatal("actor has no picture")
	}
	return v.Figure
}
