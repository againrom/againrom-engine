package game

import (
	"encoding/binary"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseDyingPositionFollowsCurrentActorThroughSAV(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
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
	// The cell each actor's source file names, before the world moves them.
	sourceCell := map[sim.EntityID]uint16{}
	if motions, _, _, present := f.live.world.SavedActorMotions(); present {
		for _, m := range motions {
			sourceCell[m.Entity] = m.Position.Cell
		}
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
	// The subject is a living actor the world has moved off the cell its
	// source file names, so the file's cell is superseded.
	packed := func(e sim.Entity) uint16 { return uint16(e.Y)<<8 | uint16(e.X) }
	var subject sim.EntityID
	subjectFound := false
	saved, _, _, _ := f.live.world.SavedActorMotions()
	for _, m := range saved {
		e, ok := f.live.world.Entity(m.Entity)
		if !ok || party[e.ID] || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone || !e.Alive() || m.Current {
			continue
		}
		if m.Position.Cell == m.Position.PackedCell && sourceCell[e.ID] != 0 && packed(e) != sourceCell[e.ID] {
			subject, subjectFound = e.ID, true
			break
		}
	}
	if !subjectFound {
		t.Fatal("no living actor stands off the cell its source file names")
	}
	prior, held := f.live.world.Entity(subject)
	priorMotion := motionFor(subject)
	if !held || !prior.Alive() || priorMotion.Current || sourceCell[subject] == packed(prior) {
		t.Fatalf("living superseded actor %d: held=%t cell=(%d,%d), saved motion=%+v", subject, held, prior.X, prior.Y, priorMotion)
	}
	f.LiveKill(uint32(subject))
	killed := 1
	for _, e := range f.live.world.Entities() {
		if killed == 5 {
			break
		}
		if e.ID == subject || party[e.ID] || e.Owner == sim.SelfSlot || e.HP <= 0 || e.Decay != sim.DecayNone {
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
	if !held || fallen.Alive() || fallen.Decay == sim.DecayNone || fallenMotion.Position.Cell != packed(fallen) || fallenMotion.Position.PackedCell != packed(fallen) {
		t.Fatalf("dying actor %d transition: held=%t cell=(%d,%d) stage=%d motion=%+v", subject, held, fallen.X, fallen.Y, fallen.Decay, fallenMotion)
	}
	wantPosition := priorMotion.Position
	wantPosition.Cell, wantPosition.PackedCell = packed(fallen), packed(fallen)
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
		body, _ := f.live.world.Entity(subject)
		if binary.LittleEndian.Uint16(block) != packed(body) || binary.LittleEndian.Uint16(block[2:]) != packed(body) || block[4] != priorMotion.Position.FineX || block[5] != priorMotion.Position.FineY {
			t.Fatalf("dying actor %d SAV Block12=%x, want current cell %#04x and frozen fine (%d,%d)", subject, block, packed(body), priorMotion.Position.FineX, priorMotion.Position.FineY)
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
