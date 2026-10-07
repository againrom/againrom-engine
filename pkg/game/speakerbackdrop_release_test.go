package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Mission 70's first message is raised by the mission's own program on its
// first pass and its first part is npc25's, Brian's record (`Hero,Face,!Female,
// !Mage`, man fighter sheet 1). Brian is not a placement in that mission, so a
// party that does not carry him leaves no live actor for the record and the
// speaker is synthesised: twelve empty equipment slots (DLG-SPEAKER-022) on the
// warrior backdrop a drawable with the Hero token's bit and no mage bit gets
// from the figure compositor (TAVERN-TALKSTATS-017, HERO-FIGURE-144). A party
// that carries him answers with the live Brian: his worn set on the same
// backdrop.
//
// Each state is reached with ordinary frames and the Enter key, and the picture
// is the one standing in the message window's portrait pane. The expected
// pictures are composed from the installed man fighter sheet, the installed
// backdrop sheet and the worn set the world holds, through the inventory
// composer and a separate backdrop draw, and each is told apart from the other
// and from the sheet alone.
//
// Mission 40's own two npc25 lines wait on a fight and a walk to the turn point,
// so that map is checked where its speaker is resolved: the placed Brian is a
// live actor there, and the picture is his worn set on the backdrop.
func TestReleaseSynthesisedHeroSpeakerStandsOnTheHeroBackdrop(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	src := f.Archives.Containers
	rec, ok := f.NPCFaces[25]
	if !ok || rec.Kind != data.NPCFigure || !rec.Tokens.Has(data.NPCTokenHero) ||
		rec.Dir != data.FigureDirManFighter || rec.Face != 1 {
		t.Fatalf("npc25's record changed: %+v", rec)
	}

	// Brian as mission 40 hands him over: the placement that states npc25.
	m40 := releaseMissionMap(t, f, 40)
	addr40, _ := MissionMap(40)
	start40, err := StartMissionFrom(m40, addr40, 40, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatal(err)
	}
	brianID := sim.EntityID(^uint32(0))
	for i, u := range m40.Units {
		if u.Flags&1 != 0 && u.ClassSubID == 25 {
			brianID = sim.EntityID(i)
			break
		}
	}
	brian, ok := start40.Start.Roster[brianID]
	if !ok {
		t.Fatal("mission 40 has no npc25 placement")
	}

	castOf := func(live *mapWorld) speakerCast {
		cast := speakerCast{actors: live.speakerActors, alive: live.entityAlive, worn: live.equipmentOf}
		if dir, ok := live.playerFigureDir(); ok {
			cast.playerDir, cast.hasPlayer = dir, true
		}
		return cast
	}
	onBackdrop := func(live *mapWorld, id sim.EntityID) *image.RGBA {
		t.Helper()
		worn, _ := live.world.Equipped(id)
		subject, _ := composeInventorySubject(src, uint32(id), equipmentFromSlots(worn), rec.Dir, rec.Face)
		if subject.Figure == nil {
			t.Fatalf("figure %s %d is unreadable", rec.Dir, rec.Face)
		}
		return heroBackgroundExpected(t, src, rec.Dir, subject.Figure)
	}

	// speak opens mission 70 with the party, steps ordinary frames until its
	// first message opens, checks that the first part is npc25's, reads the
	// portrait pane's picture and pages the message closed with Enter.
	speak := func(party []mapload.PartyMember, when string) (*mapWorld, *image.RGBA) {
		t.Helper()
		a := f.App("hero backdrop speaker")
		t.Cleanup(a.StopAudio)
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpenerWith(70, party)); err != nil {
			t.Fatalf("%s: open mission 70: %v", when, err)
		}
		live := f.live
		start := live.world.Tick()
		for i := 0; i < 400 && !a.HeadlessNoticeOpen(); i++ {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		if !a.HeadlessNoticeOpen() || live.world.Tick() == start {
			t.Fatalf("%s: no message opened while the world ran from tick %d to %d", when, start, live.world.Tick())
		}
		payload, ok := ReadEventText(src, 70, 1)
		m := live.mission
		if !ok || !bytes.Equal(m.payload, payload) || m.part != 1 {
			t.Fatalf("%s: the open message is part %d of another text than mission 70's first", when, m.part)
		}
		if speaker, named := EventPartSpeaker(payload, 1, m.audience); !named || speaker != 25 {
			t.Fatalf("%s: the first part names speaker %d/%t, want npc25", when, speaker, named)
		}
		portrait, face := live.view.NoticeSpeaker()
		if !portrait || face == nil {
			t.Fatalf("%s: the message window shows no portrait pane", when)
		}
		for n := 0; n < 12 && a.HeadlessNoticeOpen(); n++ {
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		if a.HeadlessNoticeOpen() {
			t.Fatalf("%s: Enter left the message open", when)
		}
		return live, face
	}

	alone, syntheticFace := speak(MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table), "party without Brian")
	if _, found := castOf(alone).resolve(rec); found {
		t.Fatal("a live actor answers for npc25 in the party without Brian; the speaker is not synthesised")
	}
	carried, liveFace := speak(append(MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table), brian), "party carrying Brian")
	actor, found := castOf(carried).resolve(rec)
	if !found {
		t.Fatal("no live actor answers for npc25 in the party carrying Brian")
	}
	worn, _ := carried.world.Equipped(actor.id)
	if worn[0] == 0 || worn[5] == 0 || worn[11] == 0 {
		t.Fatalf("the carried Brian's worn set = %v, want his weapon, cuirass and boots", worn)
	}

	bare, _ := composeInventorySubject(src, 0, data.Equipment{}, rec.Dir, rec.Face)
	if bare.Figure == nil {
		t.Fatalf("figure %s %d is unreadable", rec.Dir, rec.Face)
	}
	bareOnBackdrop := heroBackgroundExpected(t, src, rec.Dir, bare.Figure)
	dressedOnBackdrop := onBackdrop(carried, actor.id)
	backdropPixels := differingPixels(bareOnBackdrop, bare.Figure)
	clothingPixels := differingPixels(dressedOnBackdrop, bareOnBackdrop)
	if backdropPixels < 100 || clothingPixels < 100 {
		t.Fatalf("the fixture cannot tell the pictures apart: backdrop %d pixels, clothing %d pixels",
			backdropPixels, clothingPixels)
	}
	if !imagesEqual(syntheticFace, bareOnBackdrop) {
		t.Errorf("synthesised npc25 is not the bare man fighter on the hero backdrop: differs from it in %d pixels, from the sheet alone in %d, from the dressed Brian in %d",
			differingPixels(syntheticFace, bareOnBackdrop), differingPixels(syntheticFace, bare.Figure),
			differingPixels(syntheticFace, dressedOnBackdrop))
	}
	if !imagesEqual(liveFace, dressedOnBackdrop) {
		t.Errorf("live npc25 is not Brian's worn set on the hero backdrop: differs from it in %d pixels, from the bare figure on the backdrop in %d",
			differingPixels(liveFace, dressedOnBackdrop), differingPixels(liveFace, bareOnBackdrop))
	}

	// Mission 40: the placed Brian answers for npc25 from the first frame.
	a := f.App("mission 40 speaker")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(40)); err != nil {
		t.Fatalf("open mission 40: %v", err)
	}
	placed := f.live
	on40, found := castOf(placed).resolve(rec)
	if !found || on40.id != brianID {
		t.Fatalf("mission 40's npc25 resolves to entity %d/%t, want the placed Brian %d", on40.id, found, brianID)
	}
	pic, _, ok := placed.SpeakerFace(25)
	if !ok || !imagesEqual(pic, onBackdrop(placed, brianID)) {
		t.Errorf("mission 40's npc25 is not the placed Brian's worn set on the hero backdrop (answered %t)", ok)
	}
	t.Logf("npc25 without Brian: bare on the backdrop, %d backdrop pixels; with Brian: %d clothing pixels more; mission 40: entity %d",
		backdropPixels, clothingPixels, brianID)
}

// differingPixels counts the pixels where two pictures of one size differ.
func differingPixels(a, b *image.RGBA) int {
	if a.Bounds() != b.Bounds() {
		return a.Bounds().Dx() * a.Bounds().Dy()
	}
	n := 0
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}
