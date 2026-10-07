package game

import (
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// speakerFrontEnd is the shared-shell tavern of a chapter with one mercenary
// cell and three talk cells: NPC 9 and NPC 22 each hold a mission, NPC 90 has
// nothing to give.
func speakerFrontEnd(t *testing.T) *FrontEnd {
	t.Helper()
	c := shellCampaign()
	c.Chapters[30] = Chapter{Mission: 30, InnNPC: []int{9, 22, 90}, Inn: []int{31, 32, 0}, Mercenaries: []int{3}}
	f := shellFrontEnd()
	town := NewTown(c)
	town.Won(20)
	town.Arrive()
	town.gold = 1000
	f.Campaign, f.Town = resolved(c, nil), town
	f.Archives = &Archives{Containers: townTextFS(t, []synth.File{
		{Path: "text/inn/npc/npc09m31.txt", Data: []byte("<part=1,npc=9>\r\nnine")},
		{Path: "text/inn/npc/npc22m32.txt", Data: []byte("<part=1,npc=22>\r\nfirst<part=2,npc=22>\r\nsecond")},
		{Path: "text/inn/npc/npc90m30.txt", Data: []byte("<part=1,npc=90>\r\nnothing")},
	})}
	f.Font = resolved(missionFont(), nil)
	f.townUI.room = roomTavern
	f.townUI.composeShopFaces()
	return f
}

func tavernLabels(s *townScreen) []string {
	var out []string
	for _, c := range s.tavernSurfaceCells() {
		out = append(out, c.Semantic)
	}
	return out
}

// hearSpeaker presses the talk cell at index, then the Talk button, and reads
// the whole conversation to its end. It returns the action the last page
// produced.
func hearSpeaker(t *testing.T, s *townScreen, cell int) ui.TownAction {
	t.Helper()
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
	act, _ := talkToSelected(t, s)
	return act
}

// talkToSelected presses Talk on whatever the roster has selected and reads
// the conversation to its end. It also returns the text path the conversation
// opened on.
func talkToSelected(t *testing.T, s *townScreen) (ui.TownAction, string) {
	t.Helper()
	s.townSurfaceButton(tavernButtonTalk)
	if s.room != roomTalk {
		t.Fatalf("Talk opened room %v, want the conversation", s.room)
	}
	path, _ := s.townTextPath()
	var act ui.TownAction
	for n := 0; s.room == roomTalk && n < 16; n++ {
		act = s.AdvanceTownDialogue()
	}
	if s.room != roomTavern {
		t.Fatalf("the conversation left room %v, want the tavern", s.room)
	}
	return act, path
}

// A speaker whose conversation queued a mission is still in the roster, still
// selected and can be heard again, and every live hearing queues the identity.
// Nothing is committed inside the tavern: leaving it registers the queue, and
// the next visit still lists each speaker.
func TestAcceptedTavernSpeakerStaysAndRepeats(t *testing.T) {
	f := speakerFrontEnd(t)
	s := f.townUI
	want := []string{"Mercenary 3", "NPC 9", "NPC 22", "NPC 90"}
	if got := tavernLabels(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("roster before any conversation = %v, want %v", got, want)
	}
	gold := f.Town.Gold()

	act := hearSpeaker(t, s, 2)
	if act.Msg != "" || len(f.Town.Available()) != 0 || len(s.innQueue) != 1 {
		t.Fatalf("NPC 22 posted %q, available %v, queued %v; want no line, nothing at the gates and one queued mission", act.Msg, f.Town.Available(), s.innQueue)
	}
	if got := tavernLabels(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("roster after NPC 22's conversation = %v, want %v", got, want)
	}
	if cells := s.tavernSurfaceCells(); !cells[2].Selected {
		t.Fatal("NPC 22 lost the selection after his conversation")
	}

	// The second press: no new click, the same Talk button.
	act, path := talkToSelected(t, s)
	if path != "main/text/inn/npc/npc22m32.txt" {
		t.Fatalf("the second press opened %q, want the conversation it opened before", path)
	}
	if act.Msg != "" || len(f.Town.Available()) != 0 || len(s.innQueue) != 2 || f.Town.Gold() != gold {
		t.Fatalf("hearing NPC 22 again: %q, available %v, queued %v, gold %d; want no line and two queued hearings", act.Msg, f.Town.Available(), s.innQueue, f.Town.Gold())
	}

	// Another speaker behaves the same, and the sentinel speaker queues nothing.
	hearSpeaker(t, s, 1)
	hearSpeaker(t, s, 1)
	hearSpeaker(t, s, 3)
	hearSpeaker(t, s, 3)
	if got := tavernLabels(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("roster after every speaker was heard twice = %v, want %v", got, want)
	}
	if len(f.Town.Available()) != 0 || len(s.innQueue) != 4 || len(f.Town.Offers(TownTavern)) != 3 {
		t.Fatalf("available %v, queued %v, %d live offers; want nothing at the gates, four queued and all three offers still live", f.Town.Available(), s.innQueue, len(f.Town.Offers(TownTavern)))
	}

	// Leaving the tavern commits the queue.
	if !s.Back() {
		t.Fatal("Back in the tavern reported it left no room")
	}
	if got := f.Town.Available(); !reflect.DeepEqual(got, []int{31, 32}) || len(s.innQueue) != 0 || len(f.Town.Offers(TownTavern)) != 1 {
		t.Fatalf("available %v, queued %v, %d live offers; want [31 32], nothing queued and only the sentinel", got, s.innQueue, len(f.Town.Offers(TownTavern)))
	}

	// The next visit keeps every speaker; a kept speaker hands over nothing.
	s.Choose(0)
	if s.room != roomTavern {
		t.Fatalf("the tavern door opened room %v", s.room)
	}
	if got := tavernLabels(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("roster on the next visit = %v, want %v", got, want)
	}
	act = hearSpeaker(t, s, 2)
	if act.Msg != "" || !reflect.DeepEqual(f.Town.Available(), []int{31, 32}) || len(s.innQueue) != 0 || f.Town.Gold() != gold {
		t.Fatalf("hearing kept NPC 22: %q, available %v, queued %v, gold %d; want no line and nothing more handed over", act.Msg, f.Town.Available(), s.innQueue, f.Town.Gold())
	}
}

// The roster keeps the speaker whose pair an accepted mission took out of the
// live arrays, in the native model and in a model restored from a save. The
// live offers stay what they were, a kept speaker cannot be taken again, a
// save still holds the shortened arrays, and the chapter's advance ends it.
func TestTavernRosterKeepsAnAcceptedSpeakerInBothModels(t *testing.T) {
	for _, restored := range []bool{false, true} {
		name := "native"
		if restored {
			name = "ordinary"
		}
		t.Run(name, func(t *testing.T) {
			_, town := preTownCampaign(t, restored)
			town.Won(10)
			town.Won(20)
			town.Arrive()
			before := []TownOffer{{Index: 0, Mission: 30, NPC: 22}, {Index: 1, NPC: 90}}
			if got := town.TavernRoster(); !reflect.DeepEqual(got, before) {
				t.Fatalf("roster before acceptance = %+v, want %+v", got, before)
			}
			if m, ok := town.Take(TownTavern, 0); !ok || m != 30 {
				t.Fatalf("Take = %d/%v, want mission 30", m, ok)
			}
			after := []TownOffer{{Index: -1, Mission: 30, NPC: 22}, {Index: 1, NPC: 90}}
			if got := town.TavernRoster(); !reflect.DeepEqual(got, after) {
				t.Fatalf("roster after acceptance = %+v, want %+v", got, after)
			}
			if got := town.Offers(TownTavern); !reflect.DeepEqual(got, after[1:]) {
				t.Fatalf("live offers = %+v, want only the sentinel %+v", got, after[1:])
			}
			gold := town.Gold()
			if m, ok := town.Take(TownTavern, after[0].Index); ok || m != 0 ||
				!reflect.DeepEqual(town.Available(), []int{30}) || town.Gold() != gold {
				t.Fatalf("Take of a kept speaker = %d/%v, available %v, gold %d", m, ok, town.Available(), town.Gold())
			}
			if restored {
				p := town.progress.projection()
				if !reflect.DeepEqual(p.InnNPC, []uint16{90}) || !reflect.DeepEqual(p.InnMission, []uint16{0}) {
					t.Fatalf("a save would hold InnNPC %v InnMission %v, want the pair gone", p.InnNPC, p.InnMission)
				}
			}
			town.Won(30)
			next := town.TavernRoster()
			if !reflect.DeepEqual(next, town.Offers(TownTavern)) || len(next) == 0 {
				t.Fatalf("roster after the chapter advanced = %+v, want the next chapter's live offers %+v", next, town.Offers(TownTavern))
			}
			for _, o := range next {
				if o.Index < 0 {
					t.Fatalf("the next chapter lists a kept speaker %+v", o)
				}
			}
		})
	}
}

// A save an original game wrote after it accepted a mission already lacks the
// pair. The roster still lists that speaker, in registry order, and reading it
// changes nothing the save holds.
func TestTavernRosterListsTheSpeakerAnOriginalSaveAlreadyAccepted(t *testing.T) {
	c, projection := restoredCampaignFixture()
	projection.InnNPC, projection.InnMission = []uint16{1, 3}, []uint16{0, 51}
	projection.Main.Announced = true
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	town := newTownFromCampaignProgress(c, progress)
	want := []TownOffer{{Index: 0, NPC: 1}, {Index: -2, Mission: 50, NPC: 2}, {Index: 1, Mission: 51, NPC: 3}}
	if got := town.TavernRoster(); !reflect.DeepEqual(got, want) {
		t.Fatalf("roster = %+v, want %+v", got, want)
	}
	if got := town.Offers(TownTavern); !reflect.DeepEqual(got, []TownOffer{want[0], want[2]}) {
		t.Fatalf("live offers = %+v, want the two rows the save still holds", got)
	}
	if got := progress.projection(); !reflect.DeepEqual(got.InnNPC, projection.InnNPC) || !reflect.DeepEqual(got.InnMission, projection.InnMission) {
		t.Fatalf("reading the roster changed the arrays a save holds: %v %v", got.InnNPC, got.InnMission)
	}
}
