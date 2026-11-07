package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"image"
	"os"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// triggerWriteFixture is a 16x16 map whose script holds a drop action and a
// win action, a unit-alive check and a constant, and one trigger "Win" whose
// 64 unnamed bytes at +0x40 are set, so a write that clears them shows.
func triggerWriteFixture() []byte {
	node := func(opcode, id uint32, params ...[2]uint32) []byte {
		rec := make([]byte, 796)
		binary.LittleEndian.PutUint32(rec[0x40:], opcode)
		binary.LittleEndian.PutUint32(rec[0x44:], id)
		for i, p := range params {
			binary.LittleEndian.PutUint32(rec[0x4c+4*i:], p[1])
			binary.LittleEndian.PutUint32(rec[0x74+4*i:], p[0])
		}
		return rec
	}
	array := func(records ...[]byte) []byte {
		out := binary.LittleEndian.AppendUint32(nil, uint32(len(records)))
		for _, r := range records {
			out = append(out, r...)
		}
		return out
	}
	trigger := make([]byte, 184)
	copy(trigger, "Win")
	for k := 0x40; k < 0x80; k++ {
		trigger[k] = byte(k)
	}
	binary.LittleEndian.PutUint32(trigger[0x80:], 1)
	binary.LittleEndian.PutUint32(trigger[0x84:], 2)
	binary.LittleEndian.PutUint32(trigger[0x98:], 2)
	binary.LittleEndian.PutUint32(trigger[0xa8:], outsideCode)
	binary.LittleEndian.PutUint32(trigger[0xb4:], 1)
	payload := array(node(0x10002, 1, [2]uint32{5, 5}, [2]uint32{6, 6}), node(4, 2))
	payload = append(payload, array(node(5, 1, [2]uint32{4, 7}), node(0x10002, 2, [2]uint32{7, 0}))...)
	payload = append(payload, array(trigger)...)
	return synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: payload, Units: []synth.ALMUnit{
		{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40},
	}})
}

// triggerWriteCases are the editor operations that write a trigger record or
// a trigger field, each on a fresh fixture.
var triggerWriteCases = []struct {
	name string
	op   func(m *mapedit.Editor) error
}{
	{"arm win unit", func(m *mapedit.Editor) error { return m.AddArm(mapedit.ArmWinUnit, 7) }},
	{"arm win group", func(m *mapedit.Editor) error { return m.AddArm(mapedit.ArmWinGroup, 3) }},
	{"arm win structure", func(m *mapedit.Editor) error { return m.AddArm(mapedit.ArmWinStructure, 9) }},
	{"arm lose unit", func(m *mapedit.Editor) error { return m.AddArm(mapedit.ArmLoseUnit, 7) }},
	{"drop zone", func(m *mapedit.Editor) error { _, err := m.AddDropZone(9, 10); return err }},
	{"new trigger", func(m *mapedit.Editor) error { _, err := addScriptRecord(m, mapedit.KindTrigger); return err }},
	{"box area on trigger", func(m *mapedit.Editor) error { return addBoxArea(m, image.Rect(2, 3, 6, 8), 0) }},
	{"box area new trigger", func(m *mapedit.Editor) error { return addBoxArea(m, image.Rect(1, 1, 4, 4), -1) }},
	{"wizard new trigger", func(m *mapedit.Editor) error {
		return writeWizardTrigger(m, ui.WizardTrigger{Index: -1, Name: "Wizard", Once: 1,
			Pairs:   []ui.WizardPair{{Left: ui.WizardNode{ID: 1, Keep: true}, Cmp: 0, Right: ui.WizardNode{ID: 2, Keep: true}}},
			Actions: []ui.WizardNode{{ID: 2, Keep: true}}})
	}},
	{"wizard rewrite trigger", func(m *mapedit.Editor) error {
		return writeWizardTrigger(m, ui.WizardTrigger{Index: 0, Name: "Again", Once: 0,
			Pairs: []ui.WizardPair{{Left: ui.WizardNode{ID: 2, Keep: true}, Cmp: 3, Right: ui.WizardNode{ID: 1, Keep: true}},
				{Left: ui.WizardNode{ID: 1, Keep: true}, Cmp: 5, Right: ui.WizardNode{ID: 2, Keep: true}}},
			Actions: []ui.WizardNode{{ID: 1, Keep: true}, {ID: 2, Keep: true}}})
	}},
	{"wizard keeps a code outside the alphabet", func(m *mapedit.Editor) error {
		return writeWizardTrigger(m, ui.WizardTrigger{Index: 0, Once: 1,
			Pairs:   []ui.WizardPair{{Left: ui.WizardNode{ID: 1, Keep: true}, Cmp: outsideCode, Right: ui.WizardNode{ID: 2, Keep: true}}},
			Actions: []ui.WizardNode{{ID: 1, Keep: true}}})
	}},
	{"set trigger fields", func(m *mapedit.Editor) error {
		return m.SetFields(mapedit.KindTrigger, 0, mapedit.FieldSet{Key: "left1", Value: 2}, mapedit.FieldSet{Key: "right1", Value: 1},
			mapedit.FieldSet{Key: "cmp1", Value: 2}, mapedit.FieldSet{Key: "action3", Value: 1}, mapedit.FieldSet{Key: "once", Value: 0})
	}},
}

// outsideCode is the comparison code the fixture's trigger carries, outside
// the alphabet 0..5, as one shipped pair does.
const outsideCode = 0xffffffff

// triggerWriteGolden is each case's map digest and last trigger record digest
// as the engine's trigger writers produced them before they became one.
var triggerWriteGolden = map[string][2]string{
	"arm win unit":                             {"0e9d07dbc6605b6e7b8f17ce", "ad26e809c19c3930e31e7d1d"},
	"arm win group":                            {"0d2ed6cc831b649c276fcd65", "ad26e809c19c3930e31e7d1d"},
	"arm win structure":                        {"edc6da97a6182a292d056f94", "0e09eac9366f705be99c0301"},
	"arm lose unit":                            {"986556b07b8b75035df24977", "604df6f8f31d956f15845abb"},
	"drop zone":                                {"7ad6c997f0901f563bbb8e20", "d171bd56047b7c29683ea0d6"},
	"new trigger":                              {"bf6cb5122adfc85bd472631e", "53c85ee5597fdb373e3ccd71"},
	"box area on trigger":                      {"25400a6a3df8d06056d318b6", "e8cb45d6973c3687be8e18e2"},
	"box area new trigger":                     {"0a284e1c2c1c8ec15a069fa2", "ab8a267af44da1e0ff8aac3d"},
	"wizard new trigger":                       {"5f4e46ae12345afe5cc731f9", "384b1aa4d309cf5281ea1701"},
	"wizard rewrite trigger":                   {"16bbe45fa27539f1cb178645", "bbf07b2a36a8ddce3fad793f"},
	"wizard keeps a code outside the alphabet": {"3b0bd31fe427192ff020b03c", "2bb824371b7e7068c381d86b"},
	"set trigger fields":                       {"100bef8f3af24bd976192b97", "282b1d30cf27c4dc05ee843a"},
}

func TestTriggerWritersKeepTheirBytes(t *testing.T) {
	for _, c := range triggerWriteCases {
		m, err := mapedit.New(triggerWriteFixture())
		if err != nil {
			t.Fatal(err)
		}
		if err := c.op(m); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		n := m.Count(mapedit.KindTrigger)
		rec, err := m.Record(mapedit.KindTrigger, n-1)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		whole, last := sha256.Sum256(m.Bytes()), sha256.Sum256(rec)
		got := [2]string{hex.EncodeToString(whole[:12]), hex.EncodeToString(last[:12])}
		if os.Getenv("SCRIPT_WRITERS_PRINT") != "" {
			t.Logf("%q: {%q, %q}, // %d triggers", c.name, got[0], got[1], n)
			continue
		}
		if want, ok := triggerWriteGolden[c.name]; !ok || got != want {
			t.Errorf("%s: map and trigger digests %v, want %v", c.name, got, want)
		}
	}
}
