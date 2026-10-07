package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestReleaseTownCompanionAndPlatoonPortraitsSurviveNativeSave(t *testing.T) {
	f := releaseFront(t)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Portrait hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.Town.mercEnabled[14] = true
	f.addChapterCompanions(f.Town.Chapter())
	for _, npc := range []int32{23, 24, 25} {
		member, ok := mapload.CampaignNPCMember(f.Table, npc, 0, f.Carried)
		if !ok {
			t.Fatal("missing installed companion", npc)
		}
		f.Carried = append(f.Carried, member)
	}
	want := append([]mapload.PartyMember(nil), f.Carried...)
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := f.ExportNativeCitySave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	for generation := 1; generation <= 2; generation++ {
		cold := releaseFront(t)
		_, town, err := cold.RestoreOriginal(encoded)
		if err != nil || !town {
			t.Fatalf("generation%d town=%v err=%v", generation, town, err)
		}
		if len(cold.Carried) != len(want) {
			t.Fatal("roster count changed")
		}
		s := cold.TownScreen().(*townScreen)
		s.composeShopFaces()
		for i, original := range want {
			got := cold.Carried[i]
			if got.CompanionNPC != original.CompanionNPC || got.Mage != original.Mage || got.FigureDir != original.FigureDir || got.FigureFace != original.FigureFace || mapload.EquipmentFromParty(got) != mapload.EquipmentFromParty(original) {
				t.Fatalf("generation%d member%d identity or equipment changed: npc %d -> %d", generation, i, original.CompanionNPC, got.CompanionNPC)
			}
			if original.CompanionNPC == 0 {
				continue
			}
			dir, face := memberFigure(original)
			expected, _ := composeInventorySubject(cold.Archives.Containers, uint32(i+1), mapload.EquipmentFromParty(original), dir, face)
			if original.PlayerCharacter && !original.Hired() && !dir.Mage() {
				expected.Figure = heroBackgroundExpected(t, cold.Archives.Containers, dir, expected.Figure)
			}
			pic, _, ok := s.resolver.SpeakerFace(original.CompanionNPC)
			if !ok || !imagesEqual(pic, expected.Figure) {
				t.Fatalf("generation%d npc%d dialogue lost its own equipped figure", generation, original.CompanionNPC)
			}
		}
		// The chapter-110 optional talk uses npc2's 69,21 source window.
		// Build the exact shipped offer independently of the mercenary button.
		mission := 110
		chapter := cold.Campaign.Value().Chapters[110]
		for i, npc := range chapter.InnNPC {
			if npc == 2 && chapter.Inn[i] != 0 {
				mission = chapter.Inn[i]
			}
		}
		if !s.openTownDialogue(TownTavern, TownOffer{Mission: mission}, 2) {
			t.Fatal("missing installed npc2 campaign talk")
		}
		line, ok := expectedInstalledDialoguePart(t, s.townPayload(), 1, HeroAudience(cold.Carried))
		if !ok {
			t.Fatal("missing talk part")
		}
		layout, _, _ := s.townDialogueLayout()
		rec := cold.NPCFaces[2]
		expected := ui.RenderNotice(layout.WithFaceWindow(ui.NoticeFaceWindow(rec.WindowX, rec.WindowY)), cold.Font.Value(), line, expectedInstalledMercenaryTalkPicture(t, cold, 2))
		actual, ok := s.TownDialogue()
		if !ok || !imagesEqual(actual, expected) {
			t.Fatal("platoon talk substituted or miscropped the hero")
		}
		if generation == 1 {
			s.atSquare()
			snap, label, err = cold.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err = cold.ExportOriginalSave(snap, label)
			if err != nil {
				t.Fatal("second native save", err)
			}
		}
	}
}
