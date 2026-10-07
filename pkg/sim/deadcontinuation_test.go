package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestGeneratedCorpseRetirementKeepsIdentityThroughColdSave(t *testing.T) {
	w := deadWorld(t)
	for i, class := range []uint8{GeneratedUnitBinding, GeneratedHumanBinding} {
		e := &w.entities[i]
		e.SourceBinding = SourceBinding{Class: class, Identity: uint32(100 + i), RuntimeID: uint32(200 + i)}
		e.Humanoid = class == GeneratedHumanBinding
		e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: e.SourceBinding.ActorClass()}}
		e.HP, e.Decay = decayGoneHP-1, DecayBones
		if err := e.SourceBinding.Validate(*e); err != nil {
			t.Fatal(err)
		}
	}
	w.remove([]EntityID{1, 2})
	before := w.OriginalDeadActors()
	if len(before) != 2 || before[0].Source.Class != GeneratedUnitBinding || before[1].Source.Class != GeneratedHumanBinding {
		t.Fatalf("generated retirement provenance: %+v", before)
	}
	for cut := range 2 {
		data, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(data); err != nil || cold.Hash() != w.Hash() {
			t.Fatalf("cut %d: %v", cut, err)
		}
		for range 64 {
			Step(w, nil)
			Step(&cold, nil)
			if w.Hash() != cold.Hash() {
				t.Fatal("retired actor continuation changed")
			}
		}
		if !reflect.DeepEqual(cold.OriginalDeadActors(), before) {
			t.Fatal("retired provenance changed")
		}
		w = &cold
	}
	section := make([]byte, w.originalDeadSectionLen())
	w.encodeOriginalDead(section, 0)
	section = section[:len(section)-4]
	if _, err := decodeOriginalDead(section, w.bounds, w.entities, w.carried, w.equipment); err != nil {
		t.Fatal("valid generated retirement section rejected", err)
	}
	for name, change := range map[string]func([]byte){
		"missing key":              func(b []byte) { binary.LittleEndian.PutUint32(b[4:], 0) },
		"repeated key":             func(b []byte) { copy(b[originalDeadRecordLen+4:], b[4:8]) },
		"invented archive":         func(b []byte) { binary.LittleEndian.PutUint16(b[36:], 9) },
		"imported missing archive": func(b []byte) { b[40] = 1 },
		"not terminal":             func(b []byte) { b[58] = 4 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := append([]byte(nil), section...)
			change(bad)
			if _, err := decodeOriginalDead(bad, w.bounds, w.entities, w.carried, w.equipment); err == nil {
				t.Fatal("invalid retirement binding accepted")
			}
		})
	}
}

func TestVirtualCorpseSessionPhaseAndTerminalBoundary(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sub, full uint32
		stage     uint8
		hp        int16
		wantStage uint8
		wantHP    int16
	}{
		{"before phase", 11, 0, 2, -19, 2, -19},
		{"even full tick", 12, 0, 2, -19, 3, -20},
		{"odd full tick", 12, 1, 2, -19, 2, -19},
		{"independent stage", 12, 1, 2, -40, 4, -40},
		{"negative signed subtick", 0xfffffffc, 0, 2, -19, 2, -19},
		{"terminal transition", 12, 0, 4, -600, 5, -10001},
		{"imported terminal", 12, 0, 5, -10007, 5, -10007},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := deadWorld(t)
			input := deadInput(10, tc.stage, tc.hp)
			input.Source.MapUnitID = 0
			if err := w.ImportOriginalDeadActors([]OriginalDeadActor{input}); err != nil {
				t.Fatal(err)
			}
			w.hasSessionClock, w.tick, w.fullTick = true, uint64(tc.sub), tc.full
			Step(w, nil)
			got := w.OriginalDeadActors()[0]
			want := input.Source.State
			want.Stage, want.HP = tc.wantStage, tc.wantHP
			if want.Stage == 5 {
				want.RuntimeID = 0
			}
			if got.Source != input.Source || got.Current != want {
				t.Fatalf("source/current = %+v, want source unchanged and current %+v", got, want)
			}
			data, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var back World
			if err := back.UnmarshalBinary(data); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 64; i++ {
				Step(w, nil)
				Step(&back, nil)
				if w.Hash() != back.Hash() {
					t.Fatalf("cold continuation differs at step %d", i)
				}
			}
			if got.Current.Stage == 5 && !reflect.DeepEqual(w.OriginalDeadActors(), []OriginalDeadRecord{got}) {
				t.Fatal("terminal tuple changed or resurrected")
			}
		})
	}
}

func TestVirtualCorpseBinaryRejectsRewrittenHistory(t *testing.T) {
	w := deadWorld(t)
	input := deadInput(10, 3, -24)
	input.Source.MapUnitID = 0
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{input}); err != nil {
		t.Fatal(err)
	}
	section := make([]byte, w.originalDeadSectionLen())
	w.encodeOriginalDead(section, 0)
	valid := input.Source.State
	valid.HP = -25
	for name, change := range map[string]func(*DeadActorState){
		"increased health":  func(s *DeadActorState) { s.HP = -23 },
		"earlier stage":     func(s *DeadActorState) { s.Stage = 2 },
		"position":          func(s *DeadActorState) { s.Cell++ },
		"runtime identity":  func(s *DeadActorState) { s.RuntimeID++ },
		"invented terminal": func(s *DeadActorState) { s.Stage, s.HP, s.RuntimeID = 5, -10007, 0 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := append([]byte(nil), section[:originalDeadRecordLen]...)
			current := valid
			change(&current)
			putDeadState(bad[62:], current)
			if _, err := decodeOriginalDead(bad, w.bounds, w.entities, w.carried, w.equipment); err == nil {
				t.Fatal("accepted altered dead history")
			}
		})
	}
	putDeadState(section[62:], valid)
	if _, err := decodeOriginalDead(section[:originalDeadRecordLen], w.bounds, w.entities, w.carried, w.equipment); err != nil {
		t.Fatal("current dead state cannot continue through the existing binary form", err)
	}
}
