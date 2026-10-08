package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The literal writer owns all offsets and input values. Two equal-valued
// Units are distinct archive objects; aliases/nulls in the lists are not new
// objects. Zero/duplicate Token identity values cannot select a DTO join.
func unit1158Literal(t *testing.T, identity uint32) ([]byte, unit1158Set, *SnapshotSAVDocument) {
	t.Helper()
	a, b := &poolFixtureActor{name: "A"}, &poolFixtureActor{name: "A"}
	c := &poolFixtureActor{name: "Long Human CString", human: true}
	d := &poolFixtureActor{name: "Humanoid", humanoid: true}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a, b, a, nil, c, d, b}}}}, nil)
	for _, actor := range []*poolFixtureActor{a, b, c, d} {
		binary.LittleEndian.PutUint32(body[actor.off+29:], identity)
		for i := range 134 {
			body[actor.off+45+i] = byte(0x31 + i)
		}
		if actor.human || actor.humanoid {
			for i, value := range []uint32{0x80000001, 0x12345678, 0xffffffff, 0x7fffffff, 0x00010000, 0x89abcdef} {
				binary.LittleEndian.PutUint32(body[actor.off+609+len(actor.name)+4*i:], value)
			}
		}
	}
	f := unit1158FixtureFront(t)
	raw := completeDocumentTail1115(t, f, savedContainer(body))
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := readUnitCombatExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := decodeSavedDocument(raw)
	if differences := want.documentDifferences(state); len(differences) != 0 {
		t.Fatal("positive literal baseline", differences)
	}
	if len(want.records) != 4 || want.records[0].raw["UA6"][23] != 0x48 || want.records[0].raw["UBE"][21] != 0x5e || want.records[0].raw["U114"][23] != 0x76 || want.records[0].raw["UD4"][63] != 0xb6 || binary.LittleEndian.Uint32(want.records[2].raw["H1CC"][20:]) != 0x89abcdef {
		t.Fatal("independent literal combat anchors changed", want.records)
	}
	locations, _ := source.DocumentActorLocations()
	for i, actor := range []*poolFixtureActor{a, b, c, d} {
		if locations[i].RawBlocksOff != actor.off+45 || i >= 2 && locations[i].HumanoidXPOff != actor.off+609+len(actor.name) {
			t.Fatal("structural start disagrees with literal writer", i, locations[i])
		}
	}
	return raw, want, state
}

func TestUnit1158DocumentControls(t *testing.T) {
	for _, identity := range []uint32{0, 0x12345678} {
		t.Run(fmt.Sprintf("TokenIdentity-%x", identity), func(t *testing.T) {
			_, want, state := unit1158Literal(t, identity)
			for _, r := range want.records {
				index := want.origins[r.archive] - 1
				for _, block := range unit1158Blocks {
					if _, present := r.raw[block.name]; !present {
						continue
					}
					for _, control := range []string{"omit", "duplicate", "short", "long"} {
						t.Run(fmt.Sprintf("archive%d-%s-%s", r.archive, block.name, control), func(t *testing.T) {
							bad := cloneSavedDocumentFixture(t, state)
							fields := bad.Document.Objects[index].Raw
							for i, field := range fields {
								if field.Name != block.name {
									continue
								}
								switch control {
								case "omit":
									fields = append(fields[:i], fields[i+1:]...)
								case "duplicate":
									fields = append(fields, field)
								case "short":
									fields[i].Bytes = fields[i].Bytes[:len(field.Bytes)-1]
								case "long":
									fields[i].Bytes = append(fields[i].Bytes, 0)
								}
								break
							}
							bad.Document.Objects[index].Raw = fields
							if len(want.documentDifferences(bad)) == 0 {
								t.Fatal("accepted block shape loss")
							}
						})
					}
					for i := range block.width {
						t.Run(fmt.Sprintf("archive%d-%s-byte%d", r.archive, block.name, i), func(t *testing.T) {
							bad := cloneSavedDocumentFixture(t, state)
							for _, field := range bad.Document.Objects[index].Raw {
								if field.Name == block.name {
									field.Bytes[i] ^= 0x80
								}
							}
							if len(want.documentDifferences(bad)) == 0 {
								t.Fatal("accepted changed block byte")
							}
						})
					}
				}
			}
			for _, control := range []string{"missing-document", "unavailable", "class", "extra-Unit-XP", "drop-equal-Unit", "collapse-equal-DTO", "zero-DTO", "out-of-range-DTO"} {
				t.Run(control, func(t *testing.T) {
					bad := cloneSavedDocumentFixture(t, state)
					changed := want
					changed.origins = map[uint16]uint16{}
					for archive, index := range want.origins {
						changed.origins[archive] = index
					}
					index := want.origins[want.records[0].archive] - 1
					switch control {
					case "missing-document":
						bad.Document = nil
					case "unavailable":
						bad.Unavailable = "control"
					case "class":
						bad.Document.Objects[index].Class = "Humanoid"
					case "extra-Unit-XP":
						bad.Document.Objects[index].Raw = append(bad.Document.Objects[index].Raw, sav.DocumentRawData{Name: "H1CC", Bytes: make([]byte, 24)})
					case "drop-equal-Unit":
						at := want.origins[want.records[1].archive] - 1
						bad.Document.Objects = append(bad.Document.Objects[:at], bad.Document.Objects[at+1:]...)
					case "collapse-equal-DTO":
						changed.origins[want.records[1].archive] = uint16(index + 1)
					case "zero-DTO":
						changed.origins[want.records[1].archive] = 0
					case "out-of-range-DTO":
						changed.origins[want.records[1].archive] = 0xffff
					}
					if len(changed.documentDifferences(bad)) == 0 {
						t.Fatal("accepted identity/population control")
					}
				})
			}
		})
	}
}

func TestUnit1158ReaderControls(t *testing.T) {
	raw, _, _ := unit1158Literal(t, 0)
	f, _ := sav.Open(raw)
	locations, _ := f.DocumentActorLocations()
	objects, _ := f.DocumentObjectLocations()
	_, origins, _ := sav.DecodeDocumentDataWithOrigins(raw)
	for _, control := range []string{"missing-locator", "duplicate-locator", "class", "object-start", "negative-block", "overlap-block", "truncated-block", "overlap-control", "truncated-state", "Unit-XP", "missing-XP", "overlap-XP", "truncated-XP", "missing-origin", "duplicate-origin", "collapse-origin", "zero-DTO", "alien-origin", "duplicate-object", "zero-object"} {
		t.Run(control, func(t *testing.T) {
			locs, objs, ids := slices.Clone(locations), slices.Clone(objects), slices.Clone(origins)
			switch control {
			case "missing-locator":
				locs = locs[1:]
			case "duplicate-locator":
				locs = append(locs, locs[0])
			case "class":
				locs[0].Class = "Player"
			case "object-start":
				locs[0].Off++
			case "negative-block":
				locs[0].RawBlocksOff = -1
			case "overlap-block":
				locs[0].RawBlocksOff = locs[0].Off
			case "truncated-block":
				locs[0].RawBlocksOff = len(f.Body) - 133
			case "overlap-control":
				locs[0].ControlOff = locs[0].RawBlocksOff
			case "truncated-state":
				locs[0].StateOff = len(f.Body) - 1
			case "Unit-XP":
				locs[0].HumanoidXPOff = locs[2].HumanoidXPOff
			case "missing-XP":
				locs[2].HumanoidXPOff = 0
			case "overlap-XP":
				locs[2].HumanoidXPOff = locs[2].StateOff
			case "truncated-XP":
				locs[2].HumanoidXPOff = len(f.Body) - 23
			case "missing-origin":
				ids = ids[1:]
			case "duplicate-origin":
				ids[0].ArchiveIndex = ids[1].ArchiveIndex
			case "collapse-origin":
				ids[0].ObjectIndex = ids[1].ObjectIndex
			case "zero-DTO":
				ids[0].ObjectIndex = 0
			case "alien-origin":
				ids[0].ArchiveIndex = 0xffff
			case "duplicate-object":
				objs = append(objs, objs[0])
			case "zero-object":
				objs[0].ArchiveIndex = 0
			}
			if _, err := unit1158Read(f.Body, locs, objs, ids); err == nil {
				t.Fatal("accepted invalid structure/origin")
			}
		})
	}
}

func TestUnit1158LiveControls(t *testing.T) {
	raw := unit1158AppFixture(t)
	f := unit1158FixtureFront(t)
	source, _ := sav.Open(raw)
	want, err := readUnitCombatExpected(source, raw)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	unit1158InitialCheck(t, want, ms)
	entities := ms.World.Entities()
	index := -1
	for i, e := range entities {
		if e.SourceBinding.ArchiveIndex == want.records[1].archive {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("missing fixture Human")
	}
	check := func(t *testing.T, edit func(*sim.Entity)) {
		t.Helper()
		bad := slices.Clone(entities)
		edit(&bad[index])
		if differences, _, _ := want.entityDifferences(bad, ms.savedDocument); len(differences) == 0 {
			t.Fatal("accepted live combat loss")
		}
	}
	for _, block := range unit1158Blocks[:4] {
		for i := range block.width {
			t.Run(fmt.Sprintf("basis-%s-byte%d", block.name, i), func(t *testing.T) {
				check(t, func(e *sim.Entity) {
					s := &e.ActorLoad.Source
					switch block.name {
					case "UA6":
						s.Attack[i] ^= 0x80
					case "UBE":
						s.Defence[i] ^= 0x80
					case "U114":
						s.Base[i] ^= 0x80
					case "UD4":
						s.Modifier[i] ^= 0x80
					}
				})
			})
		}
	}
	for i := range 6 {
		for byteIndex := range 4 {
			t.Run(fmt.Sprintf("basis-XP%d-byte%d", i, byteIndex), func(t *testing.T) {
				check(t, func(e *sim.Entity) { e.ActorLoad.Source.SkillXP[i] ^= 0x80 << (8 * byteIndex) })
			})
			t.Run(fmt.Sprintf("entity-XP%d-byte%d", i, byteIndex), func(t *testing.T) {
				check(t, func(e *sim.Entity) { e.SkillXP[i] ^= int32(uint32(0x80) << (8 * byteIndex)) })
			})
		}
		t.Run(fmt.Sprintf("skill%d", i), func(t *testing.T) {
			check(t, func(e *sim.Entity) { e.Skill[i] ^= 0x100 })
		})
	}
	for i := range 5 {
		t.Run(fmt.Sprintf("protection%d", i), func(t *testing.T) {
			check(t, func(e *sim.Entity) { e.Protection[i] ^= 0x100 })
		})
		t.Run(fmt.Sprintf("resistance%d", i), func(t *testing.T) {
			check(t, func(e *sim.Entity) { e.Resistance[i] ^= 0x80 })
		})
	}
	for _, control := range []struct {
		name string
		edit func(*sim.Entity)
	}{
		{"ToHit", func(e *sim.Entity) { e.ToHit++ }}, {"Defence", func(e *sim.Entity) { e.Defence++ }}, {"Absorption", func(e *sim.Entity) { e.Absorption++ }},
		{"DamageBase", func(e *sim.Entity) { e.DamageBase++ }}, {"DamageSpread", func(e *sim.Entity) { e.DamageSpread++ }}, {"XPSlot", func(e *sim.Entity) { e.XPSlot++ }},
		{"SecondBase", func(e *sim.Entity) { e.SecondBase++ }}, {"SecondSpread", func(e *sim.Entity) { e.SecondSpread++ }},
		{"ElementalBase", func(e *sim.Entity) { e.SecondaryDamage.Base++ }}, {"ElementalSpread", func(e *sim.Entity) { e.SecondaryDamage.Spread++ }}, {"ElementalSelector", func(e *sim.Entity) { e.SecondaryDamage.Selector++ }},
		{"HealthRegeneration", func(e *sim.Entity) { e.HealthRegeneration++ }}, {"ManaRegeneration", func(e *sim.Entity) { e.ManaRegeneration++ }},
		{"basis-presence", func(e *sim.Entity) { e.ActorLoad.Present = false }}, {"basis-class", func(e *sim.Entity) { e.ActorLoad.Source.Class = 0 }},
		{"missing-source", func(e *sim.Entity) { e.SourceBinding.Class = 0 }}, {"exact-class", func(e *sim.Entity) { e.SourceBinding.Class = 3 }},
		{"alien-archive", func(e *sim.Entity) { e.SourceBinding.ArchiveIndex = 0xffff }},
	} {
		t.Run(control.name, func(t *testing.T) { check(t, control.edit) })
	}
	for _, control := range []string{"missing", "duplicate", "retired", "DTO-is-archive", "Entity-is-DTO"} {
		t.Run("projection-"+control, func(t *testing.T) {
			bad := cloneSavedDocumentFixture(t, ms.savedDocument)
			switch control {
			case "missing":
				bad.Actors = nil
			case "duplicate":
				bad.Actors = append(bad.Actors, bad.Actors[0])
			case "retired":
				bad.Actors[0].Retired = true
			case "DTO-is-archive":
				bad.Actors[0].ObjectIndex = want.records[0].archive
			case "Entity-is-DTO":
				bad.Actors[0].EntityID = sim.EntityID(bad.Actors[0].ObjectIndex)
			}
			if differences, _, _ := want.entityDifferences(entities, bad); len(differences) == 0 {
				t.Fatal("accepted incorrect identity namespace")
			}
		})
	}
}

func TestNativePartyCombatRawSkillRepairBoundaries(t *testing.T) {
	for _, test := range []struct {
		name                 string
		level, xp, cap, want int32
		admit                bool
		class                string
	}{
		{"owner tuple", 22, 171859, 100, 55, true, "Human"},
		{"settled purchase", 22, 7141, 100, 22, true, "Human"},
		{"threshold equality", 21, 7140, 100, 22, true, "Human"},
		{"zero XP", 22, 0, 100, 22, true, "Human"},
		{"negative XP", 22, -1, 100, 22, true, "Human"},
		{"higher base", 70, 171859, 100, 70, true, "Human"},
		{"installed lower cap", 22, 171859, 50, 50, true, "Human"},
		{"non-party", 22, 171859, 100, 22, false, "Human"},
		{"Unit arithmetic", 22, 171859, 100, 22, true, "Unit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rules, err := sim.NewRules(sim.RulesParams{SkillCap: test.cap})
			if err != nil {
				t.Fatal(err)
			}
			raw := map[string][]byte{"U114": make([]byte, 24), "H1CC": make([]byte, 24), "UA6": {17, 23}}
			binary.LittleEndian.PutUint16(raw["U114"][2:], 9)
			binary.LittleEndian.PutUint32(raw["H1CC"], 171859)
			binary.LittleEndian.PutUint16(raw["U114"][4:], uint16(test.level))
			binary.LittleEndian.PutUint32(raw["H1CC"][4:], uint32(test.xp))
			before := bytes.Clone(raw["U114"])
			want := bytes.Clone(before)
			binary.LittleEndian.PutUint16(want[4:], uint16(test.want))
			got := unitNativePartyCombatRaw(raw, &milestoneActorCurrent{NativeParty: test.admit}, test.class, rules)
			if !bytes.Equal(got["U114"], want) || !bytes.Equal(raw["U114"], before) || !bytes.Equal(got["UA6"], raw["UA6"]) || !bytes.Equal(got["H1CC"], raw["H1CC"]) {
				t.Fatal("independent native base repair changed unrelated or source bytes", got, raw)
			}
		})
	}
}

func TestNativePartyCombatLOADRepairKeepsIndependentLossControls(t *testing.T) {
	settled := [6]int32{1, 55, 1, 1, 1, 1}
	levels := settled
	levels[1] = 22
	xp := [6]int32{0, 171859}
	front := skillRankLoadFront(t, settled)
	raw := skillRankLegacySave(t, front, levels, settled, xp, false)
	raw = nativeActorItemLeafEdit(t, raw, func(input map[string]any) {
		input["NativeHistoryVersion"] = uint8(0)
		delete(input, "NativeBasisWires")
		delete(input, "RemovedNativeBases")
		delete(input, "HeldNativeBasisWires")
		actors := input["Actions"].(map[string]any)["Actors"].([]any)
		for _, value := range actors {
			row := value.(map[string]any)
			if current, present := row["Current"].(map[string]any); present {
				delete(current, "NativeBasis")
			}
		}
		if held, present := input["Held"].([]any); present {
			for _, value := range held {
				row := value.(map[string]any)
				if current, present := row["Current"].(map[string]any); present {
					delete(current, "NativeBasis")
				}
			}
		}
	})
	cold := openCurrentEffectSave(t, front, raw)
	ms := cold.live.mission.state
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := readUnitCombatExpected(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	var targets []unitCombatRecord
	for _, record := range want.records {
		if record.class == "Human" && record.current != nil && record.current.NativeParty {
			targets = append(targets, record)
		}
	}
	if len(targets) != 1 {
		t.Fatal("exact raw native party Human population", len(targets))
	}
	target := targets[0]
	want.records = targets
	var observed []sim.Entity
	for _, entity := range ms.World.Entities() {
		if entity.ID == target.current.ID {
			observed = append(observed, entity)
		}
	}
	if len(observed) != 1 {
		t.Fatal("exact native party subject population", len(observed))
	}
	if differences, _, n := want.entityDifferences(observed, ms.savedDocument, ms); len(differences) != 0 || n != 1 {
		t.Fatal("native party LOAD repair raw baseline", n, differences)
	}
	for _, name := range []string{"repaired base", "unrelated base", "base mask", "XP"} {
		t.Run(name, func(t *testing.T) {
			bad := slices.Clone(observed)
			switch name {
			case "repaired base":
				bad[0].NativeBasis.Base[4] = 22
			case "unrelated base":
				bad[0].NativeBasis.Base[0]++
			case "base mask":
				bad[0].NativeBasis.BaseKnown ^= 1 << 4
			case "XP":
				bad[0].SkillXP[1]++
			}
			if differences, _, _ := want.entityDifferences(bad, ms.savedDocument, ms); len(differences) == 0 {
				t.Fatal("accepted native party combat loss", name)
			}
		})
	}
	unadmitted := nativeActorItemLeafEdit(t, raw, func(input map[string]any) { delete(input, "Party"); delete(input, "Roster") })
	unadmittedFile, err := sav.Open(unadmitted)
	if err != nil {
		t.Fatal(err)
	}
	unadmittedWant, err := readUnitCombatExpected(unadmittedFile, unadmitted)
	if err != nil {
		t.Fatal(err)
	}
	var unadmittedTargets []unitCombatRecord
	for _, record := range unadmittedWant.records {
		if record.archive == target.archive && record.off == target.off && record.class == target.class {
			unadmittedTargets = append(unadmittedTargets, record)
		}
	}
	if len(unadmittedTargets) != 1 {
		t.Fatal("exact unadmitted raw Human population", len(unadmittedTargets))
	}
	unadmittedWant.records = unadmittedTargets
	if differences, _, _ := unadmittedWant.entityDifferences(observed, ms.savedDocument, ms); !strings.Contains(strings.Join(differences, ";"), "native U114 byte4 differs") {
		t.Fatal("non-party native base was normalized", differences)
	}
}
