package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseDyingPositionFollowsCurrentActorThroughSAV(t *testing.T) {
	_, source := groundCorpusFile(t, "2026-08-02/game0000.sav", "3407c9fce2ac9ce091871a17b0c08013cf2d1e1849396c6142b1a2257aec707d")
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(source)
	if err != nil || town {
		t.Fatalf("source LOAD town=%t error=%v", town, err)
	}
	if err := f.App("dying position").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(30)
	hero := f.live.mission.ids[0]
	e, ok := f.live.entity(hero)
	if !ok {
		t.Fatal("no lead hero")
	}
	at := sim.CellPoint{X: e.X, Y: e.Y}
	tries := []sim.Command{sim.DropCarried(hero, 0, at)}
	for slot := sim.EquipSlot(1); slot <= 12; slot++ {
		tries = append(tries, sim.DropWorn(hero, slot, at))
	}
	beforeSacks := len(f.live.world.Sacks())
	for _, cmd := range tries {
		f.live.pending = append(f.live.pending, cmd)
		f.LiveAdvance(1)
		if len(f.live.world.Sacks()) != beforeSacks {
			break
		}
	}
	party := map[sim.EntityID]bool{}
	for _, id := range f.live.mission.ids {
		party[id] = true
	}
	motionFor := func(id sim.EntityID) sim.SavedActorMotion {
		t.Helper()
		motions, _, _, present := f.live.world.SavedActorMotions()
		if !present {
			t.Fatal("source World has no saved motion")
		}
		for _, motion := range motions {
			if motion.Entity == id {
				return motion
			}
		}
		t.Fatalf("actor %d has no saved motion", id)
		return sim.SavedActorMotion{}
	}
	const subject sim.EntityID = 0
	prior, held := f.live.world.Entity(subject)
	priorMotion := motionFor(subject)
	if !held || !prior.Alive() || prior.X != 50 || prior.Y != 46 || priorMotion.Current || priorMotion.Position.Cell != 0x2d31 || priorMotion.Position.PackedCell != 0x2d31 {
		t.Fatalf("living superseded actor %d: held=%t cell=(%d,%d), saved motion=%+v", subject, held, prior.X, prior.Y, priorMotion)
	}
	killed := 0
	for _, e := range f.live.world.Entities() {
		if killed == 5 {
			break
		}
		if party[e.ID] || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone {
			continue
		}
		f.LiveKill(uint32(e.ID))
		killed++
	}
	if killed != 5 {
		t.Fatalf("killed %d actors, want 5", killed)
	}
	fallen, held := f.live.world.Entity(subject)
	fallenMotion := motionFor(subject)
	if !held || fallen.Alive() || fallen.Decay == sim.DecayNone || fallenMotion.Position.Cell != 0x2e32 || fallenMotion.Position.PackedCell != 0x2e32 {
		t.Fatalf("dying actor %d transition: held=%t cell=(%d,%d) stage=%d motion=%+v", subject, held, fallen.X, fallen.Y, fallen.Decay, fallenMotion)
	}
	wantPosition := priorMotion.Position
	wantPosition.Cell, wantPosition.PackedCell = 0x2e32, 0x2e32
	if fallenMotion.Position != wantPosition {
		t.Fatalf("dying actor %d changed frozen Position outside cell: before=%+v after=%+v", subject, priorMotion.Position, fallenMotion.Position)
	}
	f.LiveAdvance(2400)
	motions, _, _, _ := f.live.world.SavedActorMotions()
	for _, motion := range motions {
		actor, found := f.live.world.Entity(motion.Entity)
		if !found || actor.Alive() || actor.Decay == sim.DecayNone {
			continue
		}
		cell := uint16(actor.Y)<<8 | uint16(actor.X)
		if motion.Position.Cell != cell || motion.Position.PackedCell != cell {
			t.Fatalf("dying actor %d at cell %#04x has saved motion Cell=%#04x PackedCell=%#04x current=%t issue=%q", actor.ID, cell, motion.Position.Cell, motion.Position.PackedCell, motion.Current, motion.Issue)
		}
	}
	written := saveSourceFlagsMission(t, f)
	doc, err := sav.DecodeDocumentData(written)
	if err != nil {
		t.Fatal(err)
	}
	writtenSubject := false
	for i := range doc.Objects {
		r := &doc.Objects[i]
		key, err := savedStructureValue(r, "Identity")
		if err != nil || key != prior.SourceBinding.Identity {
			continue
		}
		block, err := savedMotionRaw(r, "Block12", 12)
		if err != nil {
			t.Fatal(err)
		}
		if binary.LittleEndian.Uint16(block) != 0x2e32 || binary.LittleEndian.Uint16(block[2:]) != 0x2e32 || block[4] != priorMotion.Position.FineX || block[5] != priorMotion.Position.FineY {
			t.Fatalf("dying actor %d SAV Block12=%x, want current cell 0x2e32 and frozen fine (%d,%d)", subject, block, priorMotion.Position.FineX, priorMotion.Position.FineY)
		}
		writtenSubject = true
	}
	if !writtenSubject {
		t.Fatalf("SAV omitted dying actor %d source identity %#x", subject, prior.SourceBinding.Identity)
	}
	g := loadSourceFlagsMission(t, written)
	if f.live.world.Hash() != g.live.world.Hash() {
		t.Fatalf("GAME/SAV/LOAD World hash %016x -> %016x", f.live.world.Hash(), g.live.world.Hash())
	}
	f.LiveAdvance(1)
	g.LiveAdvance(1)
	if f.live.world.Hash() != g.live.world.Hash() {
		t.Fatalf("post LOAD tick World hash %016x -> %016x", f.live.world.Hash(), g.live.world.Hash())
	}
}
