package game

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMission40PlacedHeroBodyBeforeHandoverAndColdLoad(t *testing.T) {
	verify := func(t *testing.T, f *FrontEnd, id sim.EntityID) {
		t.Helper()
		if f.live == nil || f.live.mission == nil || f.live.mission.state == nil {
			t.Fatal("missing live mission")
		}
		ms := f.live.mission.state
		if ms.Number != 40 {
			t.Fatalf("mission %d, want 40", ms.Number)
		}
		for _, partyID := range ms.Start.IDs {
			if partyID == id {
				t.Fatal("Brian already joined; pre-handover discriminator was lost")
			}
		}
		e, exists := f.live.world.Entity(id)
		member, rostered := ms.Start.Roster[id]
		if !exists || !e.Alive() || !rostered || e.TypeID != sim.HeroTypeID(false, false) || member.Mage {
			t.Fatalf("not a living placed male fighter Hero: entity=%+v roster=%+v present=%v/%v", e, member, exists, rostered)
		}
		slots, equipped := f.live.world.Equipped(id)
		worn := 0
		for _, code := range slots {
			if code != 0 {
				worn++
			}
		}
		if !equipped || worn < 7 {
			t.Fatalf("Brian's dressed discriminator was lost: equipped=%v slots=%x", equipped, slots)
		}
		body, dir, class, matched := data.HeroAppearance(f.Bodies, equipmentFromSlots(slots), member.Mage, false)
		if !matched || f.live.units == nil {
			t.Fatalf("cannot resolve live-equipment body %q/%q class %d matched=%v", dir, body, class, matched)
		}
		before := f.live.world.Hash()
		LoadHeroBody(f.Archives.Containers, f.live.units, dir, body)
		key := data.HeroBodyKey(dir, body)
		pc := f.live.units.Bodies[key]
		npc := f.live.units.Classes[class]
		if pc == nil || npc == nil || len(pc.Frames) == 0 || len(npc.Frames) == 0 {
			t.Fatalf("missing installed PC/NPC resources for %q class %d", key, class)
		}
		if reflect.DeepEqual(pc.Frames, npc.Frames) {
			t.Fatal("installed PC and NPC frames are identical; resource loss control cannot discriminate")
		}
		var actual ui.MapEntity
		count := 0
		for _, draw := range f.live.entityDraws() {
			if draw.ID == uint32(id) {
				actual = draw
				count++
			}
		}
		if count != 1 || actual.Art == nil || actual.Frame == nil {
			t.Fatalf("Brian has %d drawable projections: art=%p frame=%p", count, actual.Art, actual.Frame)
		}
		if f.live.art[id] != pc || actual.Art != pc || !reflect.DeepEqual(actual.Art.Frames, pc.Frames) {
			t.Errorf("actual entityDraws uses class art %p, override %p; want installed PC body %q/%p; NPC=%p", actual.Art, f.live.art[id], key, pc, npc)
		}
		frameIndex := -1
		baseFrame := releaseBaseFrame(pc, actual.Frame)
		for i, frame := range pc.Frames {
			if baseFrame == frame {
				frameIndex = i
				break
			}
		}
		if frameIndex < 0 {
			t.Errorf("actual selected frame %p is outside PC body %q", actual.Frame, key)
		} else {
			npcFrames := npc.TierFrames(f.live.tiers[id])
			if frameIndex >= len(npcFrames) || npcFrames[frameIndex] == nil {
				t.Fatal("matching NPC frame is absent; selected-frame loss control cannot discriminate")
			}
			pcFrame, npcFrame := *pc.Frames[frameIndex], *npcFrames[frameIndex]
			tables := releaseOwnerTables(t, f)
			if pc.OwnerShaded {
				pcFrame.Palette = tables[e.Owner&15]
			}
			if npc.OwnerShaded {
				npcFrame.Palette = tables[e.Owner&15]
			}
			pcPixels, npcPixels := pcFrame.RGBA(), npcFrame.RGBA()
			actualPixels := actual.Frame.RGBA()
			if actualPixels.Bounds() != pcPixels.Bounds() || !bytes.Equal(actualPixels.Pix, pcPixels.Pix) {
				t.Error("selected frame differs from independently owner-shaded PC pixels")
			}
			if pcPixels.Bounds() == npcPixels.Bounds() && bytes.Equal(pcPixels.Pix, npcPixels.Pix) {
				t.Fatal("selected PC frame has the NPC pixels; no visible resource discriminator")
			}
			t.Logf("entity %d mapID %d type=%#x PC body=%q frame=%d differs from NPC class %d", id, e.MapUnitID, e.TypeID, key, frameIndex, class)
		}
		if f.live.world.Hash() != before {
			t.Fatal("appearance observation changed simulation state")
		}
	}

	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("placed Hero body")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(40)); err != nil {
		t.Fatal(err)
	}
	id := releaseRosterNPC(t, f.live.mission.state, 25)
	initial, exists := f.live.world.Entity(id)
	if !exists || initial.MapUnitID == 0 {
		t.Fatal("Brian lacks a persistent map identity")
	}
	t.Run("fresh-before-handover", func(t *testing.T) { verify(t, f, id) })

	path, _ := writeOrdinarySAV(t, f, "placed-hero.sav")
	cold, coldApp := loadSAVWindow(t, SaveStore{Dir: filepath.Dir(path)}, filepath.Base(path))
	t.Cleanup(coldApp.StopAudio)
	if cold.live == nil || cold.live.mission == nil || cold.liveMission != 40 {
		t.Fatal("ordinary SAV cold LOAD did not install mission 40")
	}
	coldID, found := sim.EntityID(0), 0
	for _, e := range cold.live.world.Entities() {
		if e.MapUnitID == initial.MapUnitID {
			coldID, found = e.ID, found+1
		}
	}
	if found != 1 {
		t.Fatalf("cold LOAD has %d actors for Brian mapID %d", found, initial.MapUnitID)
	}
	t.Run("cold-load-before-first-tick", func(t *testing.T) { verify(t, cold, coldID) })
	cold.live.tick()
	t.Run("cold-load-next-tick", func(t *testing.T) { verify(t, cold, coldID) })
}
