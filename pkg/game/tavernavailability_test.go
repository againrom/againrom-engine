package game

import (
	"image"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func tavernAvailabilityCampaign() Campaign {
	return Campaign{
		Main: []int{30, 40}, Side: []int{31}, Offered: []int{30, 31, 40},
		MercenaryCount: []int{0, 0, 2},
		Chapters: map[int]Chapter{
			30: {Mission: 30, InnNPC: []int{22, 90}, Inn: []int{31, 0}, Mercenaries: []int{3}},
			40: {Mission: 40, InnNPC: []int{90}, Inn: []int{0}, Mercenaries: []int{3}},
			31: {Mission: 31},
		},
	}
}

func tavernAvailabilityFrontEnd(t *testing.T, retained bool) *FrontEnd {
	t.Helper()
	c := tavernAvailabilityCampaign()
	p := campaignProjectionAt(c, 30)
	p.InnNPC, p.InnMission = []uint16{90}, []uint16{0}
	p.Mercenaries, p.PermanentMercenaries = []uint16{3}, []uint16{3}
	p.MercenaryWorking[2], p.MercenaryPristine[2] = 2, 2
	if retained {
		p.Children = []sav.CampaignRecord{{Mission: 31, Announced: true}}
	}
	progress, err := campaignProgressFromSAV(c, p)
	if err != nil {
		t.Fatal(err)
	}
	f := shellFrontEnd()
	f.Campaign, f.Town = resolved(c, nil), newTownFromCampaignProgress(c, progress)
	f.Archives = &Archives{Containers: townTextFS(t, []synth.File{
		{Path: "text/inn/npc/npc22m31.txt", Data: []byte("<part=1,npc=22>\r\nQuest request.")},
		{Path: "text/inn/npc/npc90m30.txt", Data: []byte("<part=1,npc=90>\r\nTown greeting.")},
	})}
	f.Font = resolved(missionFont(), nil)
	f.TownTavernArt = resolved(&ui.TownTavernArt{
		LeftStats:   image.NewRGBA(image.Rect(0, 0, 160, 238)),
		LeftPicture: image.NewRGBA(image.Rect(0, 0, 160, 242)),
	}, nil)
	f.NPCFaces = map[int32]data.NPCFace{22: {
		Kind:   data.NPCNoPicture,
		Tokens: data.NPCTokens(data.NPCTokenNotHuman) | data.NPCTokens(data.NPCTokenHuman),
	}}
	s := f.townUI
	s.clearTavernDetail()
	s.composeShopFaces()
	s.resolver.npcFaces = f.NPCFaces
	s.resolver.playerFigure = image.NewRGBA(image.Rect(0, 0, 160, 240))
	s.tavernSelection = tavernCandidateKey{}
	s.room = roomTavern
	return f
}

func tavernAvailabilityActivate(s *townScreen, route string, offer TownOffer) bool {
	switch route {
	case "button":
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
	case "double":
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, true)
	case "legacy":
		s.Choose(0)
	case "direct":
		return s.openOfferDialogue(TownTavern, offer, offer.NPC)
	}
	return s.room == roomTalk
}

func TestKeptTavernSpeakerWithoutCurrentRecordRetainsSelectionAndBody(t *testing.T) {
	f := tavernAvailabilityFrontEnd(t, false)
	s := f.townUI
	before := f.Town.progress.projection()
	if offer := f.Town.TavernRoster()[0]; offer.Index >= 0 || offer.Mission != 31 || f.Town.Done(31) {
		t.Fatalf("kept offer/current completion = %+v/%v", offer, f.Town.Done(31))
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	v := s.TownSurface()
	if len(v.Cells) != 3 || v.Cells[1].Semantic != "NPC 22" || !v.Cells[1].Enabled ||
		!v.Cells[1].Selected || v.RosterUnpainted {
		t.Fatalf("kept speaker is not selectable: cells %+v unpainted %v", v.Cells, v.RosterUnpainted)
	}
	if v.Candidate.Figure != s.resolver.playerFigure || v.CandidatePixels == nil {
		t.Fatal("kept speaker lost its selected body")
	}
	if v.Buttons[tavernButtonTalk].Enabled {
		t.Error("Talk is enabled for a kept mission with no current record")
	}
	if !reflect.DeepEqual(f.Town.progress.projection(), before) {
		t.Fatal("selecting the kept speaker changed current campaign state")
	}
}

func TestKeptTavernSpeakerWithoutCurrentRecordCannotOpenDialogue(t *testing.T) {
	for _, route := range []string{"button", "double", "legacy", "direct"} {
		t.Run(route, func(t *testing.T) {
			f := tavernAvailabilityFrontEnd(t, false)
			s := f.townUI
			offer := f.Town.TavernRoster()[0]
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
			before := f.Town.progress.projection()
			selection, revision, gold := s.tavernSelection, s.TownDialogueRevision(), f.Town.Gold()
			body := s.TownSurface().Candidate.Figure
			if opened := tavernAvailabilityActivate(s, route, offer); opened || s.room != roomTavern ||
				s.TownDialogueRevision() != revision {
				t.Errorf("stale request opened through %s: opened %v room %v revision %d, want tavern revision %d",
					route, opened, s.room, s.TownDialogueRevision(), revision)
			}
			if s.tavernSelection != selection || len(s.innQueue) != 0 || f.Town.Gold() != gold ||
				len(f.Town.Available()) != 0 || f.Town.Done(31) ||
				!reflect.DeepEqual(f.Town.progress.projection(), before) {
				t.Fatal("blocked replay changed selection, queue, money or campaign state")
			}
			v := s.TownSurface()
			if len(v.Cells) != 3 || !v.Cells[1].Selected || v.Candidate.Figure != body || v.CandidatePixels == nil {
				t.Fatal("blocked replay removed the selected cell or body")
			}
		})
	}
}

func TestKeptTavernSpeakerWithActiveRecordReplaysDialogue(t *testing.T) {
	for _, route := range []string{"button", "double", "legacy", "direct"} {
		t.Run(route, func(t *testing.T) {
			f := tavernAvailabilityFrontEnd(t, true)
			s := f.townUI
			offer := f.Town.TavernRoster()[0]
			if offer.Index >= 0 || f.Town.progress.record(offer.Mission) == nil {
				t.Fatalf("active kept offer = %+v", offer)
			}
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
			before, gold := f.Town.progress.projection(), f.Town.Gold()
			selection := s.tavernSelection
			for hearing := 0; hearing < 2; hearing++ {
				if !s.TownSurface().Buttons[tavernButtonTalk].Enabled {
					t.Fatal("Talk is disabled for an active accepted mission")
				}
				if !tavernAvailabilityActivate(s, route, offer) || s.room != roomTalk {
					t.Fatalf("active mission replay through %s left room %v", route, s.room)
				}
				if path, ok := s.townTextPath(); !ok || path != "main/text/inn/npc/npc22m31.txt" ||
					!strings.Contains(s.dialogue.text, "Quest request.") {
					t.Fatalf("active replay opened %q with %q", path, s.dialogue.text)
				}
				s.AdvanceTownDialogue()
				if s.room != roomTavern || s.tavernSelection != selection || len(s.innQueue) != 0 ||
					f.Town.Gold() != gold || !reflect.DeepEqual(f.Town.Available(), []int{31}) ||
					!reflect.DeepEqual(f.Town.progress.projection(), before) {
					t.Fatal("active kept replay changed selection, queue, money or campaign state")
				}
			}
		})
	}
}

func TestExpiredTavernMissionCannotReplayOrBecomeDone(t *testing.T) {
	f := tavernAvailabilityFrontEnd(t, true)
	f.Town.progress.record(31).age = 1
	if side, accepted := f.Town.Won(30); side || !accepted {
		t.Fatalf("main completion = %v/%v", side, accepted)
	}
	if f.Town.Chapter() != 40 || f.Town.progress.record(31) != nil || f.Town.Done(31) {
		t.Fatal("expired side mission was retained or marked done")
	}
	s := f.townUI
	before := f.Town.progress.projection()
	if s.openOfferDialogue(TownTavern, TownOffer{Index: -1, Mission: 31, NPC: 22}, 22) || s.room != roomTavern {
		t.Error("expired mission opened its stale request")
	}
	if f.Town.Done(31) || len(s.innQueue) != 0 || !reflect.DeepEqual(f.Town.progress.projection(), before) {
		t.Fatal("suppressing an expired request changed its completion or current campaign state")
	}
}

func TestTavernTalkPreservesLiveSentinelHistoricalAndMercenaryControls(t *testing.T) {
	for _, control := range []string{"live_without_record", "zero_sentinel", "historical_kept", "mercenary"} {
		t.Run(control, func(t *testing.T) {
			f := tavernAvailabilityFrontEnd(t, false)
			s := f.townUI
			cell, wantPath, wantText, wantQueue := 1, "main/text/inn/npc/npc22m31.txt", "Quest request.", 0
			switch control {
			case "live_without_record":
				f.Town.progress.innNPC, f.Town.progress.innMission = []int{22, 90}, []int{31, 0}
				if offer := f.Town.TavernRoster()[0]; offer.Index < 0 || f.Town.progress.record(31) != nil {
					t.Fatalf("live missing-record control = %+v", offer)
				}
				wantQueue = 1
			case "zero_sentinel":
				cell, wantPath, wantText = 2, "main/text/inn/npc/npc90m30.txt", "Town greeting."
			case "historical_kept":
				town := NewTown(f.Campaign.Value())
				town.Arrive()
				if mission, accepted := town.Take(TownTavern, 0); !accepted || mission != 31 {
					t.Fatal("historical control did not accept mission 31")
				}
				town.mercEnabled[3] = true
				f.Town = town
				if f.Town.progress != nil || f.Town.TavernRoster()[0].Index >= 0 {
					t.Fatal("historical control lacks a kept speaker with nil progress")
				}
			case "mercenary":
				cell, wantPath, wantText = 0, "main/text/inn/mercenary/npc03.txt", "Mercenary dialogue unavailable: npc03"
				f.Town.gold = 0
			}
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
			if !s.TownSurface().Buttons[tavernButtonTalk].Enabled {
				t.Fatal("Talk is disabled for a preserved control")
			}
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonTalk}, false)
			if path, ok := s.townTextPath(); s.room != roomTalk || !ok || path != wantPath || len(s.innQueue) != wantQueue {
				t.Fatalf("control room/path/queue = %v/%q/%d, want talk/%q/%d", s.room, path, len(s.innQueue), wantPath, wantQueue)
			}
			text := s.dialogue.text
			if control == "mercenary" {
				text = s.talkDiagnostic
			}
			if !strings.Contains(text, wantText) {
				t.Fatalf("control text = %q, want %q", text, wantText)
			}
			if control == "zero_sentinel" {
				s.AdvanceTownDialogue()
				if !s.openOfferDialogue(TownTavern, TownOffer{Index: -2, NPC: 90}, 90) || s.room != roomTalk ||
					!strings.Contains(s.dialogue.text, wantText) || len(s.innQueue) != 0 {
					t.Fatal("a negative index suppressed the zero-mission greeting")
				}
			}
		})
	}
}
