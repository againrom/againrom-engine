package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func innRecordFixture(t *testing.T, second bool) (*FrontEnd, *townScreen) {
	t.Helper()
	if !second {
		f := shellFrontEnd()
		c := shellCampaign()
		ch := c.Chapters[30]
		ch.Inn, ch.InnNPC, ch.AddHero = []int{30, 0}, []int{9, 9}, []int{22}
		c.Chapters[30] = ch
		f.Campaign = resolved(c, nil)
		f.Town = NewTown(c)
		f.Town.Won(20)
		f.Town.Arrive()
		f.Town.gold = 1000
		f.Table.Humans = append(f.Table.Humans.(dbCollection), companionTable(t).Humans.(dbCollection)[1:]...)
		r, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
			{Name: "npc3", Kind: kindDir, Children: []synth.RegNode{{Name: "DataBinID", Kind: kindInt, Int: 3}, {Name: "PriceA", Kind: kindInt, Int: 1}, {Name: "PriceB", Kind: kindInt, Int: 2}}},
			{Name: "npc22", Kind: kindDir, Children: []synth.RegNode{{Name: "Flags", Kind: 0, Str: "Hero,Mage,!MySex,Start"}, {Name: "DataBinID", Kind: kindInt, Int: 26}}},
		}))
		if err != nil {
			t.Fatal(err)
		}
		f.Table.NPC = data.LoadNPCDefs(r)
		if err := f.SetModCompanions(mod.CompanionData{Joins: []mod.CompanionJoin{joinOf(30, 22, 9)}}); err != nil {
			t.Fatal(err)
		}
		f.Archives = tavernTalkArchives(t)
		f.Font = resolved(missionFont(), nil)
		f.Carried[0].StartingHero = true
		f.Carried = mapload.OwnParty(f.Carried)
		return f, f.townUI
	}
	text := "#npc1about\r\n<NPC=1,PART=1>\r\nAbout hire.\r\n#npc1accept30\r\n<NPC=1,PART=1>\r\nAccepted.\r\n#npc22talk30\r\n<NPC=22,PART=1>\r\nA quest.\r\n#npc2108talk31\r\n<NPC=2108,PART=1>\r\nAnother quest.\r\n#npc2110talk39\r\n<NPC=2110,PART=1>\r\nWords.\r\n#end\r\n"
	f, _, s := secondCampaignFixtureFiles(t, false, synth.File{Path: "text/town.txt", Data: []byte(text)}, identityNames())
	f.Table = keyedHireTable(t)
	f.Table.Edition = identityTable().Edition
	f.Table.Humans = append(f.Table.Humans.(dbCollection), identityTable().Humans.(dbCollection)[1:]...)
	f.Carried = []mapload.PartyMember{identityHero()}
	f.Town.second = innAt(2, 30, map[int]int32{532: 1, 552: 2})
	f.Town.second.room = secondTownInn
	f.Town.gold = 500
	f.Town.settleCityGroups(f.Carried)
	s.enterSeats()
	return f, s
}

func innRecordClose(s *townScreen) {
	for i := 0; i < 8 && s.campaignTown().talking(); i++ {
		s.AdvanceTownDialogue()
	}
}

func innRecordPhase(t *testing.T, second bool, phase string) (*FrontEnd, *townScreen) {
	t.Helper()
	f, s := innRecordFixture(t, second)
	if phase == "talk" {
		if second {
			s.Choose(2)
		} else {
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, true)
		}
		innRecordClose(s)
	}
	if phase == "hired" || phase == "dismissed" || phase == "loaded" {
		if second {
			s.Choose(1)
			innRecordClose(s)
		} else {
			s.pressMercenary(3)
		}
	}
	if phase == "dismissed" {
		if second {
			s.Choose(1)
		} else {
			s.pressMercenary(3)
		}
	}
	if phase == "loaded" {
		snap, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		f.Town, err = restoreTown(f.Campaign.Value(), snap)
		if err != nil {
			t.Fatal(err)
		}
		if second {
			s.enterSeats()
		} else {
			s.tavernSelection = tavernCandidateKey{}
		}
	}
	return f, s
}

func innRecord(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&out, format+"\n", args...) }
	state := func(key string, f *FrontEnd, s *townScreen) {
		t.Helper()
		members := make([]string, len(f.Carried))
		for i, p := range f.Carried {
			members[i] = p.ID
		}
		var available []string
		for _, n := range []int{30, 31} {
			available = append(available, fmt.Sprintf("%d:%v", n, slices.Contains(f.Town.Available(), n)))
		}
		if c := s.state(); c != nil {
			available = nil
			for _, p := range c.available {
				available = append(available, fmt.Sprintf("%d:%d", p.kind, p.id))
			}
			line("%s bank 512=%d 769=%d", key, c.bank[512], c.bank[769])
		}
		line("%s party %v available %v queue %v", key, members, available, s.innQueue)
	}
	save := func(key string, f *FrontEnd) {
		t.Helper()
		raw, err := exportTownRecord(f)
		if err != nil {
			t.Fatalf("%s SAV: %v", key, err)
		}
		line("%s SAV %d %x", key, len(raw), sha256.Sum256(raw))
	}
	for _, second := range []bool{false, true} {
		game := "rom1"
		if second {
			game = "rom2"
		}
		for _, phase := range []string{"entry", "talk", "hired", "dismissed", "loaded"} {
			key := game + " " + phase
			f, s := innRecordPhase(t, second, phase)
			state(key, f, s)
			save(key, f)
			count := len(s.tavernCandidates())
			if second {
				seats, hires := s.state().innSeats()
				count = len(seats)
				line("%s hires %d", key, hires)
				for i, seat := range seats {
					line("%s seat %d word %d/%d/%d/%v", key, i, seat.kind, seat.topic, seat.npc, seat.high)
				}
			} else {
				seats := s.tavernCandidates()
				if os.Getenv("AGAINROM_PLANT_INN_RECORD") == "1" && phase == "entry" {
					seats[0], seats[1] = seats[1], seats[0]
				}
				for i, seat := range seats {
					line("%s seat %d key %d/%d offer %v merc %v cell %q/%v/%v", key, i, seat.key.kind, seat.key.id, seat.offer, seat.merc, seat.cell.Semantic, seat.cell.Enabled, seat.cell.TalkOnly)
				}
			}
			for seat := 0; seat < count; seat++ {
				for button, name := range []string{"hire", "talk", "exit", "sleep"} {
					f, s = innRecordPhase(t, second, phase)
					probe := fmt.Sprintf("%s seat %d %s", key, seat, name)
					if second {
						d, _ := s.composedTown()
						view := roomPageView{t: s, d: d, room: "inn"}
						s.composed.seat = seat
						spec := d.Room("inn")
						var point image.Point
						shown := false
						for _, b := range spec.Buttons {
							if b.Name == name {
								point, shown = b.Rect.Rectangle().Min.Add(b.Rect.Rectangle().Max).Div(2), true
							}
						}
						ctl, hit := view.ControlAt(point)
						line("%s shown %v enabled %v caption %v", probe, shown, shown && hit, view.hireCaption() != nil)
						if shown && hit {
							var a ui.TownAction
							if ctl.Kind == ui.TownSquareControlButton {
								a = s.TownSquareButton(ctl.Door)
							} else {
								a = s.Choose(ctl.Door)
							}
							line("%s result %q", probe, a.Msg)
						}
					} else {
						s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: seat}, false)
						i := []int{tavernButtonHire, tavernButtonTalk, tavernButtonExit, tavernButtonSleep}[button]
						b := s.TownSurface().Buttons[i]
						line("%s shown true enabled %v label %q value %q", probe, b.Enabled, b.Label, b.Value)
						if b.Enabled {
							a := s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: i}, false)
							line("%s result %q", probe, a.Msg)
						}
					}
					cursor := s.talkPage()
					line("%s dialogue %q npc %d candidate %d text %q", probe, cursor.key, s.npc, s.mercenaryTalk, s.dialogue.text)
					state(probe+" open", f, s)
					innRecordClose(s)
					state(probe+" closed", f, s)
					if second {
						s.state().room = secondTownInn
					} else {
						s.room = roomTavern
					}
					save(probe, f)
				}
			}
		}
	}
	return out.String()
}

func TestInnRecord(t *testing.T) {
	got := innRecord(t)
	path := filepath.Join("testdata", "inn_record.txt")
	if os.Getenv("AGAINROM_WRITE_INN_RECORD") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := 0; i < max(len(wl), len(gl)); i++ {
		w, g := "", ""
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			t.Fatalf("inn record line %d changed:\n got %s\nwant %s", i+1, g, w)
		}
	}
}

// TestTavernRowChooseIsTheSeatTalk: the first game's row-list Choose on the
// tavern page is innAction's Talk on the same seat that a cell selection and
// the Talk button reach. Both open the same conversation with the same queue,
// party and SAV, and both raise the tavern's Talk room event.
func TestTavernRowChooseIsTheSeatTalk(t *testing.T) {
	state := func(f *FrontEnd, s *townScreen, rec *roomSoundRecorder, from int) string {
		raw, err := exportTownRecord(f)
		if err != nil {
			t.Fatal(err)
		}
		var party []string
		for _, p := range f.Carried {
			party = append(party, p.ID)
		}
		return fmt.Sprintf("room %d npc %d offer %v text %q queue %v party %v SAV %x sounds %v",
			s.room, s.npc, s.offer, s.dialogue.text, s.innQueue, party, sha256.Sum256(raw), rec.keys(from))
	}
	_, probe := innRecordFixture(t, false)
	rows, hires := len(probe.Rows()), 0
	for _, c := range probe.tavernCandidates() {
		if c.seat.hire() {
			hires++
		}
	}
	if rows != 2 || hires == 0 {
		t.Fatalf("fixture has %d tavern rows and %d hire seats, want 2 rows and a hire seat", rows, hires)
	}
	for i := 0; i < rows; i++ {
		f, s := innRecordFixture(t, false)
		rec := addRoomSounds(f)
		if a := s.Choose(i); a.Msg != "" {
			t.Fatalf("Choose(%d) posted %q", i, a.Msg)
		}
		got := state(f, s, rec, 0)
		g, c := innRecordFixture(t, false)
		crec := addRoomSounds(g)
		c.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: hires + i}, false)
		from := len(crec.requests)
		c.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
		if want := state(g, c, crec, from); got != want {
			t.Fatalf("row %d:\nChoose %s\nTalk   %s", i, got, want)
		}
		if keys := rec.keys(0); len(keys) != 1 || !strings.HasSuffix(keys[0], ":talk.wav") {
			t.Fatalf("Choose(%d) requested %q, want the Talk room event", i, keys)
		}
	}
	f, s := innRecordFixture(t, false)
	rec := addRoomSounds(f)
	for _, i := range []int{-1, rows} {
		if s.Choose(i); s.room != roomTavern || len(rec.requests) != 0 {
			t.Fatalf("Choose(%d) left room %d and requested %q", i, s.room, rec.keys(0))
		}
	}
}
