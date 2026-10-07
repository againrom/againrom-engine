package sim

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestCurrentROM2PolicyPreservesBankAndActivity(t *testing.T) {
	source := secondGameActivityWorld(t, 1, 1)
	for i := range source.rom2.Scenario {
		source.rom2.Scenario[i] = int32(i) * -97
	}
	source.runInstant(ScriptInstant{Op: ScriptInstantClearGroupActivity, Group: 9, HasGroup: true})
	captured := source.CurrentPolicy()
	encoded, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	var policy CurrentWorldPolicy
	if err := json.Unmarshal(encoded, &policy); err != nil {
		t.Fatal(err)
	}
	cold := secondGameActivityWorld(t, 1, 1)
	if reflect.DeepEqual(source.rom2, cold.rom2) {
		t.Fatal("constructor is not an independent loss control")
	}
	if err := cold.RestoreCurrentContinuation(&policy, nil, source.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source.rom2, cold.rom2) {
		t.Fatal("complete bank or Forced rows lost")
	}
	for range 40 {
		Step(source, nil)
		Step(cold, nil)
	}
	if source.Hash() != cold.Hash() {
		t.Fatal("ordinary post-load combat changed")
	}
	victim, _ := cold.Entity(3)
	if victim.HP != 20 {
		t.Fatal("restored inactive group attacked")
	}
	source.rom2.Scenario[1]++
	if policy.ROM2.Scenario[1] == source.rom2.Scenario[1] {
		t.Fatal("captured bank aliases live state")
	}
}

func TestCurrentROM2PolicyLossAndAtomicRefusal(t *testing.T) {
	source := secondGameActivityWorld(t, 1, 1)
	source.rom2.Scenario[1000] = -123
	source.runInstant(ScriptInstant{Op: ScriptInstantClearGroupActivity, Group: 9, HasGroup: true})
	for _, loss := range []string{"bank", "Forced"} {
		t.Run(loss, func(t *testing.T) {
			policy := source.CurrentPolicy()
			if loss == "bank" {
				policy.ROM2.Scenario[1000] = 0
			} else {
				for i := range policy.ROM2.Groups {
					if policy.ROM2.Groups[i].Owner == 2 && policy.ROM2.Groups[i].Group == 9 {
						policy.ROM2.Groups[i].Forced = true
					}
				}
			}
			cold := secondGameActivityWorld(t, 1, 1)
			if err := cold.RestoreCurrentContinuation(&policy, nil, source.Actions(), nil); err != nil {
				t.Fatal(err)
			}
			if loss == "bank" {
				if cold.Hash() == source.Hash() {
					t.Fatal("bank loss invisible")
				}
				return
			}
			scriptTicks(cold, 40, nil)
			victim, _ := cold.Entity(3)
			if victim.HP == 20 {
				t.Fatal("Forced loss had no ordinary combat effect")
			}
		})
	}
	for _, invalid := range []string{"absent", "duplicate", "unsorted", "overflow", "dialect"} {
		t.Run(invalid, func(t *testing.T) {
			cold := secondGameActivityWorld(t, 1, 1)
			before, _ := cold.MarshalBinary()
			policy := source.CurrentPolicy()
			switch invalid {
			case "absent":
				policy.ROM2 = nil
			case "duplicate":
				policy.ROM2.Groups = append(policy.ROM2.Groups, policy.ROM2.Groups[len(policy.ROM2.Groups)-1])
			case "unsorted":
				policy.ROM2.Groups[0], policy.ROM2.Groups[1] = policy.ROM2.Groups[1], policy.ROM2.Groups[0]
			case "overflow":
				policy.ROM2.Groups = make([]CurrentROM2GroupActivity, maxROM2Groups+1)
			case "dialect":
				cold.script, _ = NewScript(nil, nil, nil)
				cold.rom2 = nil
				before, _ = cold.MarshalBinary()
			}
			if err := cold.RestoreCurrentContinuation(&policy, nil, source.Actions(), nil); err == nil {
				t.Fatal("invalid policy admitted")
			}
			after, _ := cold.MarshalBinary()
			if !bytes.Equal(before, after) {
				t.Fatal("refusal changed live World")
			}
		})
	}
}

func TestCurrentROM2PolicyBoundedJSON(t *testing.T) {
	for _, count := range []int{0, 1023, 1025} {
		raw, _ := json.Marshal(struct {
			Scenario []int32
			Groups   []CurrentROM2GroupActivity
		}{Scenario: make([]int32, count)})
		var p CurrentROM2Policy
		if json.Unmarshal(raw, &p) == nil {
			t.Fatalf("bank length%d admitted", count)
		}
	}
	raw, _ := json.Marshal(struct {
		Scenario [1024]int32
		Groups   []CurrentROM2GroupActivity
	}{Groups: make([]CurrentROM2GroupActivity, maxROM2Groups+1)})
	var p CurrentROM2Policy
	if json.Unmarshal(raw, &p) == nil {
		t.Fatal("overflow activity admitted")
	}
	if json.Unmarshal(raw[:len(raw)-1], &p) == nil {
		t.Fatal("truncated policy admitted")
	}
}
