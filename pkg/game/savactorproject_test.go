package game

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func actorProjection1115Fixture(t *testing.T, class string) (sav.DocumentData, SnapshotSAVActor, *sim.World) {
	return actorProjectionFixture(t, class)
}

func actorProjectionFixture(t *testing.T, class string) (sav.DocumentData, SnapshotSAVActor, *sim.World) {
	t.Helper()
	profile := &profileFixture1107{stats: [3]uint16{20, 10, 5}, periods: [2]uint16{100, 50}, fractions: [2]byte{13, 27}}
	profile.attack = [24]byte{9, 0, 1, 0, 2, 0, 3, 0, 4, 0, 5, 0, 6, 0, 10, 2, 1, 3, 4, 5, 6, 2, 0xab, 0xcd}
	profile.defence = [22]byte{8, 0, 7, 0, 0xe7, 3, 10, 0, 20, 0, 30, 0, 40, 0, 50, 0, 0xef, 1, 2, 3, 4, 5}
	profile.modifier[40], profile.modifier[41] = 0x67, 0x89
	spell := &poolFixtureSpell{id: 1, rangeByte: 9, defensive: 1, cost: 4}
	item := &holdingFixtureItem{class: "Item", code: 0xe01, count: 2, kind: 3, weight: 7}
	stock := &holdingFixture{items: []*holdingFixtureItem{item}}
	stock.worn[1], stock.worn[10] = item, item
	a := &poolFixtureActor{class: class, mapID: 0, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 200,
		profile: profile, holdings: stock, book: []*poolFixtureSpell{spell, nil, {id: 3, rangeByte: 11, cost: 8}}}
	b := &poolFixtureActor{cell: 0x0807, hp: 30, maxHP: 70}
	player := &poolFixturePlayer{groups: [][]*poolFixtureActor{{a, nil, a}, {b, a}}}
	body := poolFixtureBody([]*poolFixturePlayer{player, nil, player}, nil)
	// Literal programme offsets, independent of the decoder/projector. The
	// source key and MapUnitID are zero; source actor aliases still identify one
	// object. The position/order/mover sentinels must not become native motion.
	binary.LittleEndian.PutUint32(body[a.off+29:], 0)
	for i := 0; i < 180; i++ {
		body[a.off+179+i] = byte(i + 31)
	}
	body[a.off+179], body[a.off+189] = 64, 19
	body[a.off+91+22], body[a.off+91+23] = 0x12, 0x34
	raw := savedContainer(body)
	// Reuse only the independently constructed application tail; all actor
	// bodies, roots, aliases and current values are this fixture's literals.
	tail := completeDocumentFixture1115(t, &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}})
	raw = append(raw[:int(binary.LittleEndian.Uint32(raw[4:]))+256], tail[int(binary.LittleEndian.Uint32(tail[4:]))+256:]...)
	source, origins, err := sav.DecodeDocumentDataWithOrigins(raw)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := f.ActorGraph()
	if err != nil || len(graph.Actors) != 2 {
		t.Fatal(graph, err)
	}
	var object uint16
	for _, origin := range origins {
		if origin.ArchiveIndex == graph.Actors[0].ArchiveIndex {
			object = origin.ObjectIndex
		}
	}
	if object == 0 || object == graph.Actors[0].ArchiveIndex || graph.Actors[0].Identity != 0 {
		t.Fatal("fixture failed to separate DTO/archive/zero-key identity", object, graph.Actors[0])
	}
	classID := map[string]uint8{"Unit": 1, "Human": 2, "Humanoid": 3}[class]
	// Current source values deliberately differ from the imported document.
	// Retained unnamed tails are transferred unchanged, never initialized.
	s := sim.SourceActor{Class: 2, Stats: [14]uint16{31, 0xfffd, 7, 9, 23, 14, 55, 333, 10, 100, 99, 10, 200, 50},
		Attack: profile.attack, Defence: profile.defence, Modifier: profile.modifier,
		Experience: 0xfffffff1, SkillXP: [6]uint32{11, 22, 33, 44, 55, 0xfffffff9},
		ManaFloor: 77, Sight: 1731, MoverSpeed: 37, Fighter: true, HasSpellbook: true,
		EquipmentRuntimePresent: true, Reach: 9, AttackCharge: 23, AttackRelax: 31}
	if class == "Unit" {
		s.Class = 1
	}
	s.Attack[0], s.Attack[1], s.Attack[4], s.Attack[14], s.Attack[16] = 0xe9, 3, 12, 41, 3
	s.Base = [24]byte{20, 0, 7, 0, 8, 0, 9, 0, 10, 0, 11, 0, 12, 0, 13, 14, 2, 15, 16, 0, 0, 0, 0x12, 0x34}
	s.Defence[4], s.Defence[5], s.Defence[16], s.Defence[6] = 0x56, 0x34, 0x98, 17
	s.Modifier[0], s.Modifier[10], s.Modifier[14], s.Modifier[15] = 3, 51, 0x9b, 0xff
	s.Modifier[46], s.Modifier[47], s.Modifier[58] = 0x78, 0x56, 0x9a
	e := sim.Entity{ID: 41, X: 3, Y: 4, HP: 10, MaxHP: 100, Mana: 10, MaxMana: 200, Speed: 23, Facing: 160, Humanoid: class != "Unit",
		SourceBinding: sim.SourceBinding{Class: classID, ArchiveIndex: 0x7654, Identity: 0},
		ActorLoad:     sim.ActorLoad{Present: true, ContainerPresent: true, Source: s}}
	w, err := sim.NewWorld(123, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{e})
	if err != nil {
		t.Fatal(err)
	}
	load := sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, OwnWeight: -7, Source: s}, Load: 55, Capacity: 333, Speed: 23,
		Movement: sim.HumanMovement{Present: true, RawSpeed: 23, NativeSpeed: 23, Load: 55, Capacity: 333}}
	if err := w.RestoreActorLoad(41, load); err != nil {
		t.Fatal(err)
	}
	// These public writers leave the import basis behind. SourceNow must read
	// the current pools, raw speed/load and direct defence instead of that basis.
	if err := w.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: 41, HP: 17, MaxHP: 123, Mana: 19, MaxMana: 231}}); err != nil {
		t.Fatal(err)
	}
	if !w.SetHumanMovement(41, -5, 63) {
		t.Fatal("current movement fixture refused")
	}
	if err := w.ImportOriginalActorProfiles([]sim.OriginalActorProfile{{ID: 41, Defence: -9, Absorption: 12, XPSlot: 3,
		HealthPeriod: 99, ManaPeriod: 50, HealthHundredths: 255, ManaHundredths: 157}}); err != nil {
		t.Fatal(err)
	}
	return source, SnapshotSAVActor{EntityID: 41, ObjectIndex: object}, w
}

func actorProjectionValue(t *testing.T, record sav.DocumentRecordData, name string) uint32 {
	return savedRecordValueForTest(t, record, name)
}

func savedRecordValueForTest(t *testing.T, record sav.DocumentRecordData, name string) uint32 {
	t.Helper()
	for _, value := range record.Values {
		if value.Name == name {
			return value.Value
		}
	}
	t.Fatalf("missing literal field %s", name)
	return 0
}

func actorProjection1115Raw(t *testing.T, record sav.DocumentRecordData, name string) []byte {
	return savedRecordRawForTest(t, record, name)
}

func savedRecordRawForTest(t *testing.T, record sav.DocumentRecordData, name string) []byte {
	t.Helper()
	for _, raw := range record.Raw {
		if raw.Name == name {
			return raw.Bytes
		}
	}
	t.Fatalf("missing literal block %s", name)
	return nil
}

func actorProjection1115Copy(t *testing.T, source sav.DocumentData) sav.DocumentData {
	t.Helper()
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(source); err != nil {
		t.Fatal(err)
	}
	var out sav.DocumentData
	if err := gob.NewDecoder(&encoded).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSavedActorValues1115CurrentValuesExactBindingsAndUnownedState(t *testing.T) {
	for _, class := range []string{"Unit", "Human", "Humanoid"} {
		t.Run(class, func(t *testing.T) {
			source, binding, world := actorProjection1115Fixture(t, class)
			original, err := sav.CloneDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			target, err := sav.CloneDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			worldBefore, err := world.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			before := source.Objects[binding.ObjectIndex-1]
			if err := projectSavedActorValues(&target, []SnapshotSAVActor{binding}, world); err != nil {
				t.Fatal(err)
			}
			after := target.Objects[binding.ObjectIndex-1]
			for name, want := range map[string]uint32{"T0C": 0, "U4B": 0, "T0E": 0, "U49": 0, "Body": 31, "Reaction": 65533, "Mind": 7, "Spirit": 9, "Speed": 65531,
				"U8E": 65529, "U90": 63, "Capacity": 333, "Health": 17, "HealthMax": 123, "HealthRegen": 99,
				"Mana": 19, "ManaMax": 231, "ManaRegen": 50, "UA0": 77, "UA4": 1731, "U130": 0xfffffff1,
				"UA2": 255, "UA3": 157, "U12C": 9, "U134": 23, "U135": 31} {
				if got := actorProjectionValue(t, after, name); got != want {
					t.Fatalf("%s = %#x, want %#x", name, got, want)
				}
			}
			attack := actorProjection1115Raw(t, after, "UA6")
			if !slices.Equal(attack, []byte{0xe9, 3, 1, 0, 12, 0, 3, 0, 4, 0, 5, 0, 6, 0, 41, 2, 3, 3, 4, 5, 6, 2, 0xab, 0xcd}) {
				t.Fatal("current attack/retained tails", attack)
			}
			defence := actorProjection1115Raw(t, after, "UBE")
			if !slices.Equal(defence, []byte{0xf7, 0xff, 12, 0, 0x56, 0x34, 17, 0, 20, 0, 30, 0, 40, 0, 50, 0, 0x98, 1, 2, 3, 4, 5}) {
				t.Fatal("current defence, including owned slot zero", defence)
			}
			base := actorProjection1115Raw(t, after, "U114")
			if !slices.Equal(base, []byte{20, 0, 7, 0, 8, 0, 9, 0, 10, 0, 11, 0, 12, 0, 13, 14, 2, 15, 16, 0, 0, 0, 0x12, 0x34}) {
				t.Fatal("current base", base)
			}
			modifier := actorProjection1115Raw(t, after, "UD4")
			if modifier[0] != 3 || modifier[10] != 51 || binary.LittleEndian.Uint16(modifier[14:]) != 0xff9b ||
				modifier[40] != 0x67 || modifier[41] != 0x89 || binary.LittleEndian.Uint16(modifier[46:]) != 0x5678 || modifier[58] != 0x9a {
				t.Fatal("current modifier and retained tails", modifier)
			}
			if class != "Unit" {
				xp := actorProjection1115Raw(t, after, "H1CC")
				for i, want := range []uint32{11, 22, 33, 44, 55, 0xfffffff9} {
					if binary.LittleEndian.Uint32(xp[4*i:]) != want {
						t.Fatal("per-skill XP was rebuilt from the aggregate", xp)
					}
				}
			}
			mover := actorProjection1115Raw(t, after, "U154")
			if mover[0] != 160 || mover[10] != 37 {
				t.Fatal("current facing/source mover byte", mover[:11])
			}
			// Independent exclusion comparison. Replace only the enumerated owned
			// fields in the before-record, then compare the entire graph and state.
			expected, err := sav.CloneDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			er := &expected.Objects[binding.ObjectIndex-1]
			ownedValues := strings.Fields("T0C U4B T0E U49 Body Reaction Mind Spirit Speed U8E U90 Capacity Health HealthMax HealthRegen Mana ManaMax ManaRegen UA0 UA4 U130 UA2 UA3 U12C U134 U135")
			for i, value := range er.Values {
				if slices.Contains(ownedValues, value.Name) {
					er.Values[i].Value = actorProjectionValue(t, after, value.Name)
				}
			}
			for i, raw := range er.Raw {
				if slices.Contains([]string{"UA6", "UBE", "U114", "UD4", "H1CC"}, raw.Name) {
					er.Raw[i].Bytes = slices.Clone(actorProjection1115Raw(t, after, raw.Name))
				}
				if raw.Name == "U154" {
					er.Raw[i].Bytes[0], er.Raw[i].Bytes[10] = 160, 37
				}
			}
			if !reflect.DeepEqual(target, expected) || !reflect.DeepEqual(source, original) {
				t.Fatal("projection changed source or unowned graph/scalar/unknown fields")
			}
			if !reflect.DeepEqual(before.RefSlots, after.RefSlots) || !reflect.DeepEqual(before.Counts, after.Counts) {
				t.Fatal("nullable/aliased book, inventory or worn slots changed")
			}
			worldAfter, err := world.MarshalBinary()
			if err != nil || !slices.Equal(worldAfter, worldBefore) {
				t.Fatal("projection mutated canonical world", err)
			}
			encoded, err := sav.EncodeDocumentData(target)
			if err != nil {
				t.Fatal(err)
			}
			read, err := sav.Open(encoded)
			if err != nil {
				t.Fatal(err)
			}
			profiles, err := read.ActorCurrentProfiles()
			if err != nil || len(profiles) != 2 || profiles[0].HP != 17 || profiles[0].MaxMana != 231 || profiles[0].ToHit != 1001 || profiles[0].Defence != -9 || profiles[0].HealthHundredths != 255 || profiles[1].HP != 30 {
				t.Fatal("independent current-profile readback", profiles, err)
			}
			pools, err := read.ActorPools()
			if err != nil || len(pools) != 2 || pools[0].HP != 17 || pools[0].Mana != 19 || pools[0].Cell != 0x0605 {
				t.Fatal("independent pool readback", pools, err)
			}
			graph, err := read.ActorGraph()
			if err != nil || len(graph.Actors) != 2 || graph.Actors[0].Identity != 0 || graph.Actors[0].Facing != 160 || graph.Actors[0].Cell != 0x0605 || len(graph.Groups) != 2 {
				t.Fatal("independent graph readback", graph, err)
			}
			// The moved actor detaches; the original positional null remains.
			if !slices.Equal(graph.Groups[0].Members, []uint16{0}) || len(graph.Groups[1].Members) != 2 || graph.Groups[1].Members[1] != graph.Actors[0].ArchiveIndex {
				t.Fatal("aliased record changed final group attachment", graph.Groups)
			}
			reloaded, err := sav.DecodeDocumentData(encoded)
			if err != nil || !reflect.DeepEqual(target, reloaded) {
				t.Fatal("document readback lost nullable/aliased slots", err)
			}
		})
	}
}

func TestSavedActorValues1115RefusalsAndRetirementAreAtomic(t *testing.T) {
	for _, kind := range []string{"missing entity", "zero object", "outside object", "wrong class", "wrong actor class", "absent source basis", "duplicate object", "duplicate entity", "missing field", "duplicate field", "short block", "duplicate block"} {
		t.Run(kind, func(t *testing.T) {
			doc, binding, world := actorProjection1115Fixture(t, "Human")
			bindings := []SnapshotSAVActor{binding}
			switch kind {
			case "missing entity":
				bindings[0].EntityID = 99
			case "zero object":
				bindings[0].ObjectIndex = 0
			case "outside object":
				bindings[0].ObjectIndex = uint16(len(doc.Objects) + 1)
			case "wrong class":
				bindings[0].ObjectIndex = 1 // Player, not an actor
			case "wrong actor class":
				doc.Objects[binding.ObjectIndex-1].Class = "Unit"
			case "absent source basis":
				e := world.Entities()[0]
				e.ActorLoad.Source = sim.SourceActor{}
				var err error
				world, err = sim.NewWorld(123, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{e})
				if err != nil {
					t.Fatal(err)
				}
			case "duplicate object":
				bindings = append(bindings, SnapshotSAVActor{EntityID: 99, ObjectIndex: binding.ObjectIndex})
			case "duplicate entity":
				bindings = append(bindings, SnapshotSAVActor{EntityID: binding.EntityID, ObjectIndex: 1})
			case "missing field":
				doc.Objects[binding.ObjectIndex-1].Values = nil
			case "duplicate field":
				r := &doc.Objects[binding.ObjectIndex-1]
				r.Values = append(r.Values, sav.DocumentValueData{Name: "Health"})
			case "short block":
				for i := range doc.Objects[binding.ObjectIndex-1].Raw {
					if doc.Objects[binding.ObjectIndex-1].Raw[i].Name == "UD4" {
						doc.Objects[binding.ObjectIndex-1].Raw[i].Bytes = nil
					}
				}
			case "duplicate block":
				r := &doc.Objects[binding.ObjectIndex-1]
				r.Raw = append(r.Raw, sav.DocumentRawData{Name: "UA6", Bytes: make([]byte, 24)})
			}
			// Reflection deepcopy is unnecessary: projection must not write even
			// aliased backing arrays on any refused path. Gob gives an independent
			// comparison for malformed records rejected by CloneDocumentData.
			doc = actorProjection1115Copy(t, doc)
			before := actorProjection1115Copy(t, doc)
			entities := world.Entities()
			if err := projectSavedActorValues(&doc, bindings, world); err == nil || !reflect.DeepEqual(doc, before) || !reflect.DeepEqual(world.Entities(), entities) {
				t.Fatal("refusal committed a partial projection", err)
			}
		})
	}
	doc, binding, world := actorProjection1115Fixture(t, "Human")
	doc = actorProjection1115Copy(t, doc)
	before := actorProjection1115Copy(t, doc)
	binding.Retired, binding.EntityID = true, 999
	if err := projectSavedActorValues(&doc, []SnapshotSAVActor{binding}, world); err != nil || !reflect.DeepEqual(doc, before) {
		t.Fatal("retired actor was authored or resurrected", err)
	}
	if err := projectSavedActorValues(nil, nil, world); err == nil {
		t.Fatal("nil document admitted")
	}
	if err := projectSavedActorValues(&doc, nil, nil); err == nil {
		t.Fatal("nil world admitted")
	}
}

func TestSavedActorValues1115AbsentEquipmentRuntimeKeepsStoredBytes(t *testing.T) {
	doc, binding, world := actorProjection1115Fixture(t, "Human")
	e := world.Entities()[0]
	e.ActorLoad.Source.EquipmentRuntimePresent = false
	e.ActorLoad.Source.Reach, e.ActorLoad.Source.AttackCharge, e.ActorLoad.Source.AttackRelax = 0, 0, 0
	e.Reach, e.AttackCharge, e.AttackRelax = 7, 999, -5 // not an authored source-runtime arm
	r := doc.Objects[binding.ObjectIndex-1]
	for i := range r.Values {
		switch r.Values[i].Name {
		case "U12C":
			r.Values[i].Value = 11
		case "U134":
			r.Values[i].Value = 22
		case "U135":
			r.Values[i].Value = 33
		}
	}
	if err := savedActorValueDomain(e); err != nil {
		t.Fatal("absent source runtime acquired an inferred width gate", err)
	}
	out, err := savedActorValueRecord(r, e, false)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]uint32{"U12C": 11, "U134": 22, "U135": 33} {
		if actorProjectionValue(t, out, name) != want {
			t.Fatal("absent runtime synthesized defaults", name)
		}
	}
}

func TestSavedActorValuesWidthsAndMovementAbsenceRemainWritable(t *testing.T) {
	doc, binding, world := actorProjectionFixture(t, "Human")
	for _, mutate := range []func(*sim.Entity){
		func(e *sim.Entity) { e.HP = -32769 }, func(e *sim.Entity) { e.MaxHP = 65536 },
		func(e *sim.Entity) { e.Mana = -32769 }, func(e *sim.Entity) { e.MaxMana = 65536 },
		func(e *sim.Entity) { e.Load = 32768 }, func(e *sim.Entity) { e.Capacity = -32769 },
		func(e *sim.Entity) { e.Defence = 32768 }, func(e *sim.Entity) { e.Absorption = -32769 },
		func(e *sim.Entity) { e.AttackCharge = 256 }, func(e *sim.Entity) { e.AttackRelax = -1 },
		func(e *sim.Entity) { e.HumanMovement.Present = false },
	} {
		e := world.Entities()[0]
		mutate(&e)
		if err := savedActorValueDomain(e); err != nil {
			t.Fatal("valid current value refused instead of carrying width or absence policy", err)
		}
		if _, err := savedActorValueRecord(doc.Objects[binding.ObjectIndex-1], e, false); err != nil {
			t.Fatal("ordinary actor record cannot carry current low words", err)
		}
		if e.HumanMovement.Present && len(e.Values().Widths) == 0 {
			t.Fatal("wide current field lacks an anchored width operand")
		}
	}
	for _, word := range []int32{-32768, -1, 0, 32767, 32768, 65535} {
		e := world.Entities()[0]
		e.HP, e.MaxHP, e.Mana, e.MaxMana = word, word, word, word
		if err := savedActorValueDomain(e); err != nil {
			t.Fatal("representable source word rejected", word, err)
		}
	}
	e := world.Entities()[0]
	e.HumanMovement = sim.HumanMovement{}
	e.Speed = 123
	r, err := savedActorValueRecord(doc.Objects[binding.ObjectIndex-1], e, false)
	if err != nil {
		t.Fatal(err)
	}
	if speed, err := savedStructureValue(&r, "Speed"); err != nil || speed != 123 || e.Values().MovementPresent {
		t.Fatal("absent movement carrier replaced current speed", speed, err)
	}
}

func TestSavedActorValues1115NormalSourceMutationKeepsUnnamedTails(t *testing.T) {
	doc, binding, world := actorProjection1115Fixture(t, "Human")
	mapload.BindSourceDerive(world)
	before := world.Entities()[0].SourceNow()
	// A source-backed health potion runs the complete pure derive rule and
	// therefore exercises retained raw tails under a real admitted producer.
	if err := world.ImportOriginalActorStock([]sim.OriginalActorStock{{ID: 41, Carried: []sim.ItemStack{sim.StackItem(sim.ItemInstance{Code: 0x0e07, Kind: 3, WeightPresent: true, Effects: []sim.ItemEffect{{Kind: 6, Operand: 1}}}, 1)}}}); err != nil {
		t.Fatal(err)
	}
	if !world.UseCarriedPotion(41, 0) {
		t.Fatal("ordinary source potion refused")
	}
	after := world.Entities()[0].SourceNow()
	if after == before || after.Attack[22] != before.Attack[22] || after.Attack[23] != before.Attack[23] ||
		after.Base[22] != before.Base[22] || after.Base[23] != before.Base[23] || after.Modifier[40] != before.Modifier[40] || after.Modifier[41] != before.Modifier[41] {
		t.Fatal("normal producer changed unnamed imported tails")
	}
	if err := projectSavedActorValues(&doc, []SnapshotSAVActor{binding}, world); err != nil {
		t.Fatal(err)
	}
	r := doc.Objects[binding.ObjectIndex-1]
	if actorProjection1115Raw(t, r, "UA6")[22] != 0xab || actorProjection1115Raw(t, r, "U114")[23] != 0x34 || actorProjection1115Raw(t, r, "UD4")[40] != 0x67 {
		t.Fatal("current producer/projection lost retained unnamed bytes")
	}
}

// ROM1's LOAD re-links every actor except a U4C-bit-3 one to the on-map list,
// so live presence must reach that bit in both directions and keep the class
// bits beside it.
func TestSavedActorValuesOffMapBitFollowsLivePresence(t *testing.T) {
	doc, binding, world := actorProjectionFixture(t, "Human")
	r := doc.Objects[binding.ObjectIndex-1]
	for i := range r.Values {
		if r.Values[i].Name == "U4C" {
			r.Values[i].Value = 0x02
		}
	}
	e := world.Entities()[0]
	e.SourceBinding.ClassFlags = 0x02
	e.OffMap = true
	out, err := savedActorValueRecord(r, e, false)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := savedStructureValue(&out, "U4C"); v != 0x0a {
		t.Fatalf("off-map actor U4C = %#x, want 0xa", v)
	}
	e.OffMap = false
	back, err := savedActorValueRecord(out, e, false)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := savedStructureValue(&back, "U4C"); v != 0x02 {
		t.Fatalf("returned actor U4C = %#x, want 0x2", v)
	}
}
