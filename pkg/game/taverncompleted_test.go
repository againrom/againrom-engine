package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestCompletedExtraTavernSpeakerLeavesSurfaceAndLegacyRows(t *testing.T) {
	for _, source := range []string{"side", "offered_without_section"} {
		for _, route := range []string{"pointer", "double", "legacy"} {
			t.Run(source+"/"+route, func(t *testing.T) {
				f := tavernAvailabilityFrontEnd(t, true)
				s := f.townUI
				if source == "offered_without_section" {
					f.Town.camp.Side = nil
					delete(f.Town.camp.Chapters, 31)
				}
				s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
				if !s.TownSurface().Buttons[tavernButtonTalk].Enabled {
					t.Fatal("unfinished extra quest cannot talk")
				}
				if side, accepted := f.Town.Won(31); !side || !accepted || !f.Town.Done(31) ||
					f.Town.progress.record(31) != nil || f.Town.Chapter() != 30 {
					t.Fatal("fixture did not complete only the extra quest", side, accepted)
				}
				before, gold := f.Town.progress.projection(), f.Town.Gold()
				roster := f.Town.TavernRoster()
				v := s.TownSurface()
				if got, want := tavernLabels(s), []string{"Mercenary 3", "NPC 90"}; !reflect.DeepEqual(got, want) {
					t.Fatalf("completed extra quest still leaves a talker: %v, want %v", got, want)
				}
				if v.Cells[0].Selected || v.Cells[1].Selected || v.RosterUnpainted ||
					v.Buttons[tavernButtonTalk].Enabled || v.Buttons[tavernButtonHire].Enabled || v.CandidatePixels != nil {
					t.Fatal("removed selection silently retargeted another candidate", v.Cells)
				}
				s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
				if again := s.TownSurface(); s.room != roomTavern || again.Cells[0].Selected || again.Cells[1].Selected ||
					again.Buttons[tavernButtonTalk].Enabled || again.Buttons[tavernButtonHire].Enabled {
					t.Fatal("repeated input retargeted a removed selection")
				}
				if rows := s.Rows(); len(rows) != 1 || !strings.Contains(rows[0].Text, "NPC 90") {
					t.Fatal("legacy rows did not use the visible roster", rows)
				}
				switch route {
				case "pointer", "double":
					s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, route == "double")
					if route == "pointer" {
						s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
					}
				case "legacy":
					s.Choose(0)
				}
				if path, ok := s.townTextPath(); !ok || s.room != roomTalk || path != "main/text/inn/npc/npc90m30.txt" {
					t.Fatal("compacted input did not address the sentinel", s.room, path)
				}
				s.AdvanceTownDialogue()
				if s.room != roomTavern || len(s.innQueue) != 0 || f.Town.Gold() != gold ||
					!reflect.DeepEqual(before, f.Town.progress.projection()) || !reflect.DeepEqual(roster, f.Town.TavernRoster()) {
					t.Fatal("filtering or compacted input changed campaign state")
				}
			})
		}
	}
}

func TestCompletedExtraTavernSelectionWithoutMercenariesClears(t *testing.T) {
	f := tavernAvailabilityFrontEnd(t, true)
	f.Town.progress.mercenaries = nil
	s := f.townUI
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, false)
	if !s.TownSurface().Cells[0].Selected {
		t.Fatal("unfinished talker was not selected")
	}
	if _, accepted := f.Town.Won(31); !accepted {
		t.Fatal("extra quest completion was rejected")
	}
	v := s.TownSurface()
	if len(v.Cells) != 1 || v.Cells[0].Semantic != "NPC 90" || v.Cells[0].Selected ||
		!v.RosterUnpainted || v.Buttons[tavernButtonTalk].Enabled || v.CandidatePixels != nil {
		t.Fatal("selection moved from the removed talker to an unrelated one", v.Cells, v.RosterUnpainted)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, true)
	if path, ok := s.townTextPath(); !ok || s.room != roomTalk || path != "main/text/inn/npc/npc90m30.txt" {
		t.Fatal("remaining sentinel cannot be selected after compaction", path)
	}
}

func TestTavernSelectionKeepsLaterTalkerWhenCompletedExtraCompacts(t *testing.T) {
	f := tavernAvailabilityFrontEnd(t, true)
	s := f.townUI
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 2}, false)
	if cells := s.TownSurface().Cells; !cells[2].Selected || cells[2].Semantic != "NPC 90" {
		t.Fatal("later sentinel was not selected", cells)
	}
	if _, accepted := f.Town.Won(31); !accepted {
		t.Fatal("extra quest completion was rejected")
	}
	v := s.TownSurface()
	if len(v.Cells) != 2 || v.Cells[0].Selected || !v.Cells[1].Selected || v.Cells[1].Semantic != "NPC 90" ||
		!v.Buttons[tavernButtonTalk].Enabled {
		t.Fatal("list compaction changed the retained selected identity", v.Cells)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
	if path, ok := s.townTextPath(); !ok || s.room != roomTalk || path != "main/text/inn/npc/npc90m30.txt" {
		t.Fatal("retained selection spoke to a different candidate", path)
	}
}

func TestTavernVisibilityPreservesOtherGreyAndMainTalkers(t *testing.T) {
	for _, control := range []string{"absent_not_done", "completed_main", "main_side_overlap", "live_completed_extra"} {
		t.Run(control, func(t *testing.T) {
			f := tavernAvailabilityFrontEnd(t, control == "main_side_overlap")
			s := f.townUI
			switch control {
			case "completed_main":
				f.Town.camp.Main = []int{20, 30, 40}
				ch := f.Town.camp.Chapters[30]
				ch.Inn[0] = 20
				f.Town.camp.Chapters[30] = ch
			case "main_side_overlap":
				f.Town.camp.Main = append(f.Town.camp.Main, 31)
				if _, accepted := f.Town.Won(31); !accepted {
					t.Fatal("overlap control did not complete its active record")
				}
			case "live_completed_extra":
				f.Town.progress.innNPC, f.Town.progress.innMission = []int{22, 90}, []int{31, 0}
				f.Town.won[31] = true
			}
			offer := f.Town.TavernRoster()[0]
			grey := control != "live_completed_extra"
			if s.tavernOfferCanTalk(offer) == grey || control == "absent_not_done" && f.Town.Done(31) ||
				control == "completed_main" && !f.Town.Done(20) {
				t.Fatal("control has the wrong eligibility or completion", offer)
			}
			before := f.Town.progress.projection()
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
			v := s.TownSurface()
			if got, want := tavernLabels(s), []string{"Mercenary 3", "NPC 22", "NPC 90"}; !reflect.DeepEqual(got, want) ||
				!v.Cells[1].Selected || v.Buttons[tavernButtonTalk].Enabled == grey || len(s.Rows()) != 2 {
				t.Fatal("unrelated eligibility control changed", got, v.Buttons)
			}
			if grey {
				s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
				if s.room != roomTavern || len(s.innQueue) != 0 || !reflect.DeepEqual(before, f.Town.progress.projection()) {
					t.Fatal("grey control opened or changed campaign state")
				}
			}
		})
	}
}
