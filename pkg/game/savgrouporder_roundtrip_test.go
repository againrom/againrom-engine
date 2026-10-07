//go:build sessioncorpusaudit

package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCorpusResavePreservesCurrentGroupOrders(t *testing.T) {
	assets, corpus := os.Getenv("AGAINROM_ASSETS"), os.Getenv("AGAINROM_SAVE_CORPUS")
	if assets == "" || corpus == "" {
		t.Fatal("AGAINROM_ASSETS and AGAINROM_SAVE_CORPUS are required")
	}
	for _, name := range []string{"game0002-bigsack.sav", "game0017-victory.sav"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(corpus, "2026-09-24", name))
			if err != nil {
				t.Fatal(err)
			}
			c := savGateCase{name: name + " resave", raw: raw}
			f, _, refusal, town := savGateOpen(t, assets, name, c, raw)
			if refusal != "" || town {
				t.Fatal(refusal, town)
			}
			_, before, present := f.live.world.SavedGroups()
			if !present {
				t.Fatal("source has no Group state")
			}
			bindings := map[sim.EntityID]uint32{}
			state := f.live.mission.state.savedDocument
			bind := func(id sim.EntityID, index uint16) {
				t.Helper()
				key, err := savedStructureValue(&state.Document.Objects[index-1], "Identity")
				if err != nil {
					t.Fatal(err)
				}
				bindings[id] = key
			}
			for _, actor := range state.Actors {
				if actor.Retired {
					continue
				}
				bind(actor.EntityID, actor.ObjectIndex)
			}
			if state.GroupBindings != nil {
				for _, member := range state.GroupBindings.Members {
					if member.Bound && member.ObjectIndex != 0 && bindings[member.EntityID] == 0 {
						bind(member.EntityID, member.ObjectIndex)
					}
				}
			}
			g, _, written, refusal := savGateSaveLoad(t, assets, c, f)
			if refusal != "" {
				t.Fatal(refusal)
			}
			file, err := sav.Open(written)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := file.ActorGraph()
			if err != nil {
				t.Fatal(err)
			}
			writtenByKey := map[uint32]sav.ActorRecord{}
			for _, actor := range graph.Actors {
				writtenByKey[actor.Identity] = actor
			}
			writtenDocument, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			writtenAction := map[uint32]uint32{}
			for i := range writtenDocument.Objects {
				record := &writtenDocument.Objects[i]
				key, err := savedStructureValue(record, "Identity")
				if err != nil || key == 0 {
					continue
				}
				rawAction, err := savedMotionRaw(record, "U54", 4)
				if err == nil {
					writtenAction[key] = binary.LittleEndian.Uint32(rawAction)
				}
			}
			_, loaded, present := g.live.world.SavedGroups()
			if !present {
				t.Fatal("cold LOAD has no Group state")
			}
			loadedByID := map[sim.EntityID]sim.SavedActorOrder{}
			for _, order := range loaded {
				loadedByID[order.Entity] = order
			}
			beforeMotion, _, _, _ := f.live.world.SavedActorMotions()
			loadedMotion, _, _, _ := g.live.world.SavedActorMotions()
			beforeAction, loadedAction := map[sim.EntityID]uint32{}, map[sim.EntityID]uint32{}
			for _, motion := range beforeMotion {
				beforeAction[motion.Entity] = motion.ActorAction
			}
			for _, motion := range loadedMotion {
				loadedAction[motion.Entity] = motion.ActorAction
			}
			for _, order := range before {
				key := bindings[order.Entity]
				actor, writtenOK := writtenByKey[key]
				after, loadedOK := loadedByID[order.Entity]
				if key == 0 || !writtenOK || !loadedOK {
					t.Errorf("actor %d key %d: written=%t loaded=%t", order.Entity, key, writtenOK, loadedOK)
					continue
				}
				for i := range order.Raw {
					if order.Raw[i] != actor.Order[i] && order.Raw[i] == after.Raw[i] {
						t.Logf("actor %d key %d order+%02x: wire-only %02x -> %02x -> %02x", order.Entity, key, i, order.Raw[i], actor.Order[i], after.Raw[i])
					}
					if order.Raw[i] != after.Raw[i] {
						t.Errorf("actor %d key %d order+%02x: World before=%02x SAV=%02x World loaded=%02x; state %d inner %d progress %d; ActorAction %d -> %d",
							order.Entity, key, i, order.Raw[i], actor.Order[i], after.Raw[i], order.State, order.Raw[8], order.Raw[9], beforeAction[order.Entity], loadedAction[order.Entity])
						if i == 0x0c {
							e, _ := f.live.world.Entity(order.Entity)
							t.Logf("actor %d order target word %08x -> %08x -> %08x; typed attack target=%d kind=%d bound=%t phase=%d",
								order.Entity, binary.LittleEndian.Uint32(order.Raw[0x0c:]), binary.LittleEndian.Uint32(actor.Order[0x0c:]), binary.LittleEndian.Uint32(after.Raw[0x0c:]),
								e.AttackTarget, e.AttackTargetKind, e.HasAttackTarget, e.AttackPhase)
						}
					}
				}
				if beforeAction[order.Entity] != loadedAction[order.Entity] {
					t.Logf("actor %d key %d ActorAction World before=%d SAV U54=%d World loaded=%d", order.Entity, key, beforeAction[order.Entity], writtenAction[key], loadedAction[order.Entity])
				}
			}
			if len(before) != len(loaded) {
				t.Errorf("Group order count %d -> %d", len(before), len(loaded))
			}
		})
	}
}
