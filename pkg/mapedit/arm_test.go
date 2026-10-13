package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// armTrigger writes a trigger record by the layout of ALM-TRIG-047.
func armTrigger(name string, left, right, act, cmp uint32) []byte {
	rec := make([]byte, testTrigger)
	copy(rec, name)
	binary.LittleEndian.PutUint32(rec[0x80:], left)
	binary.LittleEndian.PutUint32(rec[0x84:], right)
	binary.LittleEndian.PutUint32(rec[0x98:], act)
	binary.LittleEndian.PutUint32(rec[0xa8:], cmp)
	binary.LittleEndian.PutUint32(rec[0xb4:], 1)
	return rec
}

func armNode(label string, opcode, id uint32, params ...[2]uint32) []byte {
	rec := testNodeRecord(opcode, id, params...)
	for k := 0x9c; k < 0x9c+8; k++ {
		rec[k] = 0
	}
	copy(rec, label)
	return rec
}

// armMap is the script fixture's map with the expected arm records added to
// its arrays, written independently of AddArm.
func armMap(action, check, limit, trigger []byte) []byte {
	actions := [][]byte{testNodeRecord(0x10002, 1, [2]uint32{5, 5}, [2]uint32{6, 6}), testNodeRecord(4, 2), action}
	payload := append(testArray(actions...), testArray(
		testNodeRecord(5, 1, [2]uint32{4, 7}),
		testNodeRecord(0x10002, 2, [2]uint32{7, 0}), check, limit)...)
	payload = append(payload, testArray(scriptFixtureTrigger(), trigger)...)
	return synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: payload, Units: []synth.ALMUnit{
		{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40},
	}})
}

func scriptFixtureTrigger() []byte {
	trigger := make([]byte, testTrigger)
	copy(trigger, "Win")
	binary.LittleEndian.PutUint32(trigger[0x80:], 1)
	binary.LittleEndian.PutUint32(trigger[0x84:], 2)
	binary.LittleEndian.PutUint32(trigger[0x98:], 2)
	return trigger
}

func TestAddArmWritesTheShippedShapeAsOneEdit(t *testing.T) {
	const checkID, limitID, actionID = 3, 4, 3
	for _, c := range []struct {
		name   string
		kind   mapedit.ArmKind
		target uint32
		want   []byte
	}{
		{"win unit", mapedit.ArmWinUnit, 7, armMap(
			armNode(`Force "Mission Complete" state`, 4, actionID),
			armNode("Win unit alive?", 5, checkID, [2]uint32{4, 7}),
			armNode("Constant", 0x10002, limitID, [2]uint32{1, 0}),
			armTrigger("Win", checkID, limitID, actionID, 0))},
		{"win group", mapedit.ArmWinGroup, 22, armMap(
			armNode(`Force "Mission Complete" state`, 4, actionID),
			armNode("Win group count", 1, checkID, [2]uint32{2, 22}),
			armNode("Constant", 0x10002, limitID, [2]uint32{1, 0}),
			armTrigger("Win", checkID, limitID, actionID, 0))},
		{"win structure", mapedit.ArmWinStructure, 9, armMap(
			armNode(`Force "Mission Complete" state`, 4, actionID),
			armNode("Win structure health", 21, checkID, [2]uint32{9, 9}),
			armNode("Constant", 0x10002, limitID, [2]uint32{1, 0}),
			armTrigger("Win", checkID, limitID, actionID, 5))},
		{"lose unit", mapedit.ArmLoseUnit, 7, armMap(
			armNode(`Force "Mission Fail" state`, 5, actionID),
			armNode("Lose unit alive?", 5, checkID, [2]uint32{4, 7}),
			armNode("Constant", 0x10002, limitID, [2]uint32{1, 0}),
			armTrigger("Lose", checkID, limitID, actionID, 0))},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := scriptFixture()
			e := newEditor(t, b)
			if err := e.AddArm(c.kind, c.target); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(e.Bytes(), c.want) {
				t.Fatal("the arm's bytes differ from the independently written records")
			}
			m, err := alm.Open(e.Bytes())
			if err != nil {
				t.Fatalf("the model rejects the result: %v", err)
			}
			sc, err := m.Script()
			if err != nil || len(sc.Actions) != 3 || len(sc.Conditions) != 4 || len(sc.Triggers) != 2 {
				t.Fatalf("script counts %d/%d/%d, %v", len(sc.Actions), len(sc.Conditions), len(sc.Triggers), err)
			}
			if !e.Undo() || !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
				t.Fatal("one undo did not restore the exact earlier bytes")
			}
			if !e.Redo() || !bytes.Equal(e.Bytes(), c.want) {
				t.Fatal("redo did not restore the arm")
			}
		})
	}
}

func TestAddArmOnAMapWithAnEmptyScript(t *testing.T) {
	e, err := mapedit.NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	blank := e.Bytes()
	if err := e.AddArm(mapedit.ArmWinUnit, 2); err != nil {
		t.Fatal(err)
	}
	if err := e.AddArm(mapedit.ArmLoseUnit, 1); err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(e.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	sc, _ := m.Script()
	if len(sc.Actions) != 2 || len(sc.Conditions) != 4 || len(sc.Triggers) != 2 {
		t.Fatalf("script counts %d/%d/%d", len(sc.Actions), len(sc.Conditions), len(sc.Triggers))
	}
	if sc.Actions[1].ID != 2 || sc.Conditions[2].ID != 3 || sc.Conditions[3].ID != 4 || sc.Triggers[1].Left[0] != 3 || sc.Triggers[1].Acts[0] != 2 {
		t.Fatalf("second arm ids: %+v %+v", sc.Conditions[2:], sc.Triggers[1])
	}
	if !e.Undo() || !e.Undo() || !bytes.Equal(e.Bytes(), blank) {
		t.Fatal("two undos do not restore the blank map")
	}
}

func TestAddArmRefusesWhatItMustAndWritesNothing(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	for name, err := range map[string]error{
		"unit 0":          e.AddArm(mapedit.ArmWinUnit, 0),
		"unit over 65535": e.AddArm(mapedit.ArmWinUnit, 70000),
		"structure 0":     e.AddArm(mapedit.ArmWinStructure, 0),
		"unknown arm":     e.AddArm(mapedit.ArmKind(9), 1),
	} {
		if err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused arm wrote or recorded something")
	}
}

func TestAddArmRefusesAFullRegisterFile(t *testing.T) {
	var conds [][]byte
	for i := uint32(1); i <= 99; i++ {
		conds = append(conds, testNodeRecord(5, i, [2]uint32{4, 7}))
	}
	payload := append(testArray(testNodeRecord(4, 1)), testArray(conds...)...)
	payload = binary.LittleEndian.AppendUint32(payload, 0)
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: payload, Units: []synth.ALMUnit{{X: 256, Y: 256, ClassID: 40}}})
	e := newEditor(t, b)
	if err := e.AddArm(mapedit.ArmWinUnit, 7); err == nil {
		t.Fatal("an arm took registers past the 100 the binder owns")
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("the refusal wrote something")
	}
}

func TestPlaceAllRefusesWhatIsNotOnePayload(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	unit, err := e.UnitRecord(0)
	if err != nil {
		t.Fatal(err)
	}
	for name, batch := range map[string][]mapedit.Placement{
		"nothing":         nil,
		"a unit":          {{Kind: mapedit.KindUnit, Record: unit}},
		"a short trigger": {{Kind: mapedit.KindTrigger, Record: make([]byte, 100)}},
	} {
		if _, err := e.PlaceAll(batch); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused batch wrote something")
	}
}
