package game

import (
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func nativeTerminalSubjectFixture(t *testing.T) (*FrontEnd, []byte) {
	t.Helper()
	source, first := nativeSubjectFixture(t)
	f := nativeSubjectCold(t, source, first)
	entities := f.live.world.Entities()
	entities[0].HP, entities[0].Decay, entities[0].Dwell = -601, sim.DecayStage(4), 0
	w, err := sim.NewWorld(9, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	players, playerIDs := f.live.world.CurrentPlayers()
	participants, participantValues := f.live.world.PlayerParticipants()
	if playerIDs {
		if err := w.RestoreCurrentPlayers(players, participants, participantValues); err != nil {
			t.Fatal(err)
		}
	}
	if registry := f.live.world.SavedObjects(); registry != nil {
		if err := w.ImportSavedObjects(registry, nil); err != nil {
			t.Fatal(err)
		}
	}
	f.live.world, f.live.mission.state.World = w, w
	sim.Step(w, nil)
	terminal := w.CurrentTerminalActors()
	if len(terminal) != 1 || terminal[0].ID != 0 || terminal[0].HP != -601 || terminal[0].Cell != 0x100f || len(w.Entities()) != 1 {
		t.Fatal("actual compaction did not retain the independent terminal tuple", terminal)
	}
	raw, _, _ := saveCurrentEffect(t, f)
	return f, raw
}

func TestNativeTerminalActorSubjectColdCorrespondence(t *testing.T) {
	source, raw := nativeTerminalSubjectFixture(t)
	cold := nativeSubjectCold(t, source, raw)
	ms := cold.live.mission.state
	a, c, actors, combat := nativeSubjectCompare(t, raw, ms, ms.World.Entities())
	if actors != 1 || combat != 1 || strings.Contains(strings.Join(append(a, c...), ";"), "terminal actor subject") || strings.Contains(strings.Join(append(a, c...), ";"), "eligible raw actor missing") || strings.Contains(strings.Join(c, ";"), "expected live source basis unavailable") {
		t.Fatal("exact terminal subject was treated as a missing live Entity", actors, combat, a, c)
	}
	ms.actorRegistry.sources = nil
	a, c, actors, combat = nativeSubjectCompare(t, raw, ms, ms.World.Entities())
	if actors != 1 || combat != 1 || strings.Contains(strings.Join(append(a, c...), ";"), "terminal actor subject") {
		t.Fatal("late DeadActors-only terminal incorrectly required a graph Entity source", a, c)
	}
}

func TestNativeTerminalActorSubjectLossControls(t *testing.T) {
	source, raw := nativeTerminalSubjectFixture(t)
	for _, name := range []string{"no-manager", "source-class", "source-identity", "dead-root", "ordinary-identity", "live-ID", "raw-health", "raw-cell", "current-tuple"} {
		t.Run(name, func(t *testing.T) {
			cold := nativeSubjectCold(t, source, raw)
			ms := cold.live.mission.state
			roots, reader, origins, want := nativeSubjectRead(t, raw)
			observed := ms.World.Entities()
			for i := range roots {
				if roots[i].current == nil || roots[i].current.Terminal == nil {
					continue
				}
				object := origins[roots[i].loc.ArchiveIndex]
				if name == "source-class" || name == "source-identity" {
					var source sav.ActorRecord
					source.Off, source.ArchiveIndex, source.Class = roots[i].loc.Off, roots[i].loc.ArchiveIndex, roots[i].loc.Class
					source.Identity, source.RuntimeID = binary.LittleEndian.Uint32(reader.body[source.Off+29:]), binary.LittleEndian.Uint32(reader.body[source.Off+12:])
					source.TerminalActor = true
					ms.actorRegistry.sources = []sav.ActorRecord{source}
				}
				switch name {
				case "no-manager":
					w, err := sim.NewWorld(9, ms.World.Bounds(), sim.ModeCanonical, nil, observed)
					if err != nil {
						t.Fatal(err)
					}
					ms.World = w
				case "source-class":
					ms.actorRegistry.sources[0].Class = "Human"
				case "source-identity":
					ms.actorRegistry.sources[0].Identity ^= 1
				case "dead-root":
					ms.savedDocument.Document.DeadActors = nil
				case "ordinary-identity":
					savedObjectSetValue(&ms.savedDocument.Document.Objects[object-1], "Identity", 1)
				case "live-ID":
					observed = append(observed, sim.Entity{ID: roots[i].current.ID})
				case "raw-health":
					roots[i].hp++
					for j := range want.records {
						if want.records[j].current != nil && want.records[j].current.Terminal != nil {
							want.records[j].hp++
						}
					}
				case "raw-cell":
					reader.body[roots[i].loc.Off] ^= 1
					for j := range want.records {
						if want.records[j].current != nil && want.records[j].current.Terminal != nil {
							want.records[j].position[0] ^= 1
						}
					}
				case "current-tuple":
					roots[i].current.Terminal.HP++
					for j := range want.records {
						if want.records[j].current != nil && want.records[j].current.Terminal != nil {
							want.records[j].current.Terminal.HP++
						}
					}
				}
			}
			a, _ := actorRootEntityDifferences(roots, reader, origins, ms.savedDocument, ms.World, observed, ms)
			c, _, _ := want.entityDifferences(observed, ms.savedDocument, ms)
			if !strings.Contains(strings.Join(a, ";"), "terminal actor subject") || !strings.Contains(strings.Join(c, ";"), "terminal actor subject") {
				t.Fatal("terminal subject loss was accepted", name, a, c)
			}
			if name == "raw-health" || name == "raw-cell" || name == "current-tuple" {
				if strings.Contains(strings.Join(a, ";"), "eligible raw actor missing") || strings.Contains(strings.Join(c, ";"), "expected live source basis unavailable") {
					t.Fatal("tuple value mismatch changed exact terminal ownership", name, a, c)
				}
			}
		})
	}
}
