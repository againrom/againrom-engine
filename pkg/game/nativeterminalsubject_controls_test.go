package game

import (
	"encoding/binary"
	"encoding/json"
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
	// The dead-list ladder removes the body on the first phase-12 sub-tick
	// (SESS-TICK-006); that full tick is even, so it walks one rung first
	// (HERO-DEATH-026).
	for n := 0; n < 16 && len(w.Entities()) != 1; n++ {
		sim.Step(w, nil)
	}
	terminal := w.CurrentTerminalActors()
	if len(terminal) != 1 || terminal[0].ID != 0 || terminal[0].HP != -602 || terminal[0].Cell != 0x100f || len(w.Entities()) != 1 {
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
	for _, name := range []string{"no-manager", "source-class", "source-identity", "dead-root", "ordinary-identity", "live-ID", "raw-health", "raw-maximum", "raw-cell", "current-tuple"} {
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
				case "raw-maximum":
					roots[i].maxHP++
					for j := range want.records {
						if want.records[j].current != nil && want.records[j].current.Terminal != nil {
							want.records[j].maxHP++
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
			if name == "raw-health" || name == "raw-maximum" || name == "raw-cell" || name == "current-tuple" {
				if strings.Contains(strings.Join(a, ";"), "eligible raw actor missing") || strings.Contains(strings.Join(c, ";"), "expected live source basis unavailable") {
					t.Fatal("tuple value mismatch changed exact terminal ownership", name, a, c)
				}
			}
		})
	}
}

func TestNativeTerminalActorSubjectOrdinaryMaximum(t *testing.T) {
	for _, tc := range []struct {
		name    string
		marker  int32
		present bool
		wire    uint16
		want    int32
	}{
		{"legacy marker", 0, false, 509, 509},
		{"known zero", 0, true, 0, 0},
		{"ordinary edit", 60, true, 509, 509},
		{"wide matching word", 65596, true, 60, 65596},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, raw := nativeTerminalSubjectFixture(t)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			actions, err := readCurrentActions(&doc)
			if err != nil || actions == nil || actions.Values[0].CurrentTerminal == nil {
				t.Fatal("native terminal marker unavailable", err)
			}
			marker := actions.Values[0].CurrentTerminal
			marker.MaxHP, marker.MaxHPPresent = tc.marker, tc.present
			encoded, err := json.Marshal(actions)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
				t.Fatal(err)
			}
			if len(doc.DeadActors) != 1 {
				t.Fatal("native terminal dead root population", len(doc.DeadActors))
			}
			mustSetValue(&doc.Objects[doc.DeadActors[0]-1], "HealthMax", uint32(tc.wire))
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			cold := nativeSubjectCold(t, f, raw)
			world := cold.live.world
			terminal := world.CurrentTerminalActors()
			if len(terminal) != 1 || terminal[0].MaxHP != tc.want || !terminal[0].MaxHPPresent {
				t.Fatal("terminal maximum does not follow the ordinary word and matching width", terminal)
			}
			ms := cold.live.mission.state
			roots, combat, actors, compared := nativeSubjectCompare(t, raw, ms, world.Entities())
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			scalars, err := unitScalarExpected(file, raw)
			if err != nil {
				t.Fatal(err)
			}
			scalarDifferences, _, scalarCount := scalars.entityDifferences(world.Entities(), nil, ms)
			differences := strings.Join(append(append(roots, combat...), scalarDifferences...), ";")
			if actors != 1 || compared != 1 || scalarCount != 1 || strings.Contains(differences, "terminal actor subject") || strings.Contains(differences, "eligible raw actor missing") || strings.Contains(differences, "expected live source basis unavailable") {
				t.Fatal("independent terminal subject comparison rejected the current maximum", actors, compared, scalarCount, differences)
			}
		})
	}
}
