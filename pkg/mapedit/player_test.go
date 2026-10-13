package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
)

func playerFixture(t *testing.T, count int) []byte {
	t.Helper()
	action := testNodeRecord(0xfefefefe, 17, [2]uint32{3, 1}, [2]uint32{2, 2})
	binary.LittleEndian.PutUint32(action[0x74+9*4:], 3)
	binary.LittleEndian.PutUint32(action[0x4c+9*4:], 4)
	binary.LittleEndian.PutUint32(action[0x74+8*4:], 3)
	binary.LittleEndian.PutUint32(action[0x4c+8*4:], 0x10003)
	condition := testNodeRecord(0xedededed, 19)
	binary.LittleEndian.PutUint32(condition[0x74+9*4:], 3)
	binary.LittleEndian.PutUint32(condition[0x4c+9*4:], 3)
	binary.LittleEndian.PutUint32(condition[0x74+8*4:], 3)
	binary.LittleEndian.PutUint32(condition[0x4c+8*4:], 0xffffffff)
	script := append(testArray(testNodeRecord(1, 10), action), testArray(testNodeRecord(2, 11), condition)...)
	script = append(script, testArray(make([]byte, testTrigger))...)
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: script,
		Units:   []synth.ALMUnit{{ClassID: 40}, {ClassID: 40}, {ClassID: 40}},
		Objects: []synth.ALMObject{{Kind: 5, Field0E: 1}, {Kind: 0x21, Field0E: 3}, {Kind: 7, Field0E: 0x10004}},
	})
	_, units := recordAt(t, b, 6)
	for i, owner := range []uint32{1, 3, 0x10003} {
		binary.LittleEndian.PutUint32(b[units+i*70+0x14:], owner)
	}
	roster := make([]byte, 76*count)
	for i := 0; i < count; i++ {
		rec := roster[76*i : 76*(i+1)]
		binary.LittleEndian.PutUint32(rec, uint32(0x10203040+i))
		binary.LittleEndian.PutUint32(rec[4:], uint32(0x50607080+i))
		binary.LittleEndian.PutUint32(rec[8:], uint32(0x90a0b0c0+i))
		for j := 12; j < 44; j++ {
			rec[j] = byte(j + i)
		}
		copy(rec[12:], fmt.Sprintf("Player %d\x00", i+1))
		for j := 0; j < 16; j++ {
			binary.LittleEndian.PutUint16(rec[44+2*j:], uint16(0x3100+16*i+j))
		}
	}
	h, p := recordAt(t, b, 5)
	b = splice(b, p, 76, roster)
	binary.LittleEndian.PutUint32(b[h+8:], uint32(len(roster)))
	binary.LittleEndian.PutUint32(b[metaAt(t, b, 0x1c):], uint32(count))
	return b
}

func TestPlayerFieldsAreTheRosterSectionRows(t *testing.T) {
	b := playerFixture(t, 4)
	e := newEditor(t, b)
	fields, err := e.Fields(mapedit.KindPlayer, 2)
	if err != nil || len(fields) != 7 {
		t.Fatalf("fields: %+v, %v", fields, err)
	}
	want := []string{"name", "colour", "control", "relation0", "relation1", "relation2", "relation3"}
	for i, f := range fields {
		if f.Key != want[i] || (f.Text != (i == 0)) {
			t.Fatalf("field %d: %+v", i, f)
		}
	}
	if e.Count(mapedit.KindPlayer) != 4 {
		t.Fatal("roster count")
	}
	if _, _, err := e.Position(mapedit.KindPlayer, 2); err == nil {
		t.Fatal("player acquired a cell")
	}
	if err := e.Move(mapedit.KindPlayer, 2, 1, 2); err == nil {
		t.Fatal("player moved")
	}
	if _, err := e.CloneRecord(mapedit.KindPlayer, 2, 1, 2); err == nil {
		t.Fatal("player cloned through coordinates")
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("reading the roster changed bytes or history")
	}
}

func TestPlayerNumericEditsWriteOnlyTheirOwnBytes(t *testing.T) {
	for _, tc := range []struct {
		key           string
		offset, width int
		value         int64
	}{
		{"colour", 0, 4, 15}, {"control", 4, 4, 0},
		{"relation0", 44, 2, 1}, {"relation2", 48, 2, 0}, {"relation3", 50, 2, 2},
	} {
		t.Run(tc.key, func(t *testing.T) {
			b := playerFixture(t, 4)
			e := newEditor(t, b)
			if err := e.SetField(mapedit.KindPlayer, 2, tc.key, tc.value); err != nil {
				t.Fatal(err)
			}
			_, p := recordAt(t, b, 5)
			want := append([]byte(nil), b...)
			at := p + 2*76 + tc.offset
			if tc.width == 2 {
				binary.LittleEndian.PutUint16(want[at:], uint16(tc.value))
			} else {
				binary.LittleEndian.PutUint32(want[at:], uint32(tc.value))
			}
			if !bytes.Equal(e.Bytes(), want) {
				t.Fatal("field wrote outside its bytes or mirrored the relation")
			}
			if got, err := e.FieldValue(mapedit.KindPlayer, 2, tc.key); err != nil || got != tc.value {
				t.Fatalf("value %d, %v", got, err)
			}
			undoRedoExact(t, e, 1, b, want)
		})
	}
}

func TestPlayerRenameOwnsExactlyTheNameAndSupportsFullWidthLegacyText(t *testing.T) {
	for _, tc := range []struct {
		text    string
		encoded []byte
	}{
		{"Enemy", []byte("Enemy")}, {"Союз", []byte{0xd1, 0xee, 0xfe, 0xe7}},
		{strings.Repeat("Я", 32), bytes.Repeat([]byte{0xdf}, 32)}, {"", nil},
	} {
		t.Run(tc.text, func(t *testing.T) {
			b := playerFixture(t, 4)
			e := newEditor(t, b)
			if err := e.SetTextField(mapedit.KindPlayer, 1, "name", tc.text); err != nil {
				t.Fatal(err)
			}
			_, p := recordAt(t, b, 5)
			want := append([]byte(nil), b...)
			name := want[p+76+12 : p+76+44]
			clear(name)
			copy(name, tc.encoded)
			if !bytes.Equal(e.Bytes(), want) {
				t.Fatal("rename wrote outside the name")
			}
			if got, err := e.TextFieldValue(mapedit.KindPlayer, 1, "name"); err != nil || got != tc.text {
				t.Fatalf("name %q, %v", got, err)
			}
			undoRedoExact(t, e, 1, b, want)
		})
	}
}

func TestPlayerFieldRejectionsLeaveBytesAndHistory(t *testing.T) {
	b := playerFixture(t, 4)
	e := newEditor(t, b)
	for _, tc := range []struct {
		key   string
		value int64
	}{
		{"colour", -1}, {"colour", 16}, {"control", -1}, {"control", 2},
		{"relation0", -1}, {"relation0", 3}, {"relation4", 0}, {"scalar", 1}, {"name", 1},
	} {
		if err := e.SetField(mapedit.KindPlayer, 1, tc.key, tc.value); err == nil {
			t.Fatalf("accepted %s=%d", tc.key, tc.value)
		}
	}
	for _, text := range []string{strings.Repeat("Я", 33), "a\x00b", "漢", "\xff"} {
		if err := e.SetTextField(mapedit.KindPlayer, 1, "name", text); err == nil {
			t.Fatalf("accepted name %q", text)
		}
	}
	if err := e.SetTextField(mapedit.KindPlayer, 1, "colour", "Blue"); err == nil {
		t.Fatal("numeric field accepted text")
	}
	if _, err := e.FieldValue(mapedit.KindPlayer, 1, "name"); err == nil {
		t.Fatal("name read as a number")
	}
	if _, err := e.TextFieldValue(mapedit.KindPlayer, 1, "colour"); err == nil {
		t.Fatal("colour read as text")
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("rejection wrote bytes or history")
	}
}

func TestPlayerAddUsesTheSectionCountAndWritesOnlyItsRecordAndFraming(t *testing.T) {
	b := playerFixture(t, 4)
	e := newEditor(t, b)
	rec, err := e.NewPlayerRecord()
	if err != nil {
		t.Fatal(err)
	}
	wantRecord := make([]byte, 76)
	binary.LittleEndian.PutUint32(wantRecord, 5)
	binary.LittleEndian.PutUint32(wantRecord[4:], 1)
	binary.LittleEndian.PutUint32(wantRecord[8:], 5000)
	copy(wantRecord[12:44], "New Player 5")
	for j := 0; j < 16; j++ {
		v := uint16(1)
		if j == 4 {
			v = 2
		}
		binary.LittleEndian.PutUint16(wantRecord[44+2*j:], v)
	}
	if !bytes.Equal(rec, wantRecord) || !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("new player record or pure constructor")
	}
	if got, err := e.Place(mapedit.KindPlayer, rec); err != nil || got != 4 {
		t.Fatalf("place %d, %v", got, err)
	}
	h, p := recordAt(t, b, 5)
	want := splice(b, p+4*76, 0, wantRecord)
	addWord(want, h+8, 76)
	addWord(want, metaAt(t, b, 0x1c), 1)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("add touched old players or another section")
	}
	undoRedoExact(t, e, 1, b, want)
}

func TestPlayerPlacementChecksEveryAuthoredFieldAndTheRosterLimit(t *testing.T) {
	b := playerFixture(t, 4)
	e := newEditor(t, b)
	valid, err := e.NewPlayerRecord()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func([]byte) []byte
	}{
		{"short", func(r []byte) []byte { return r[:75] }},
		{"long", func(r []byte) []byte { return append(r, 0) }},
		{"colour", func(r []byte) []byte { binary.LittleEndian.PutUint32(r, 16); return r }},
		{"control", func(r []byte) []byte { binary.LittleEndian.PutUint32(r[4:], 2); return r }},
		{"relation", func(r []byte) []byte { binary.LittleEndian.PutUint16(r[74:], 3); return r }},
		{"name", func(r []byte) []byte { r[12] = 0x98; return r }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := e.Place(mapedit.KindPlayer, tc.change(append([]byte(nil), valid...))); err == nil {
				t.Fatal("invalid record accepted")
			}
			if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
				t.Fatal("invalid placement mutated the map")
			}
		})
	}
	e = newEditor(t, playerFixture(t, 15))
	last, err := e.NewPlayerRecord()
	if err != nil || binary.LittleEndian.Uint32(last) != 15 {
		t.Fatalf("final colour, %v", err)
	}
	if _, err := e.Place(mapedit.KindPlayer, last); err != nil {
		t.Fatal(err)
	}
	full := e.Bytes()
	if _, err := e.NewPlayerRecord(); err == nil {
		t.Fatal("built seventeenth player")
	}
	if _, err := e.Place(mapedit.KindPlayer, valid); err == nil {
		t.Fatal("placed seventeenth player")
	}
	if !bytes.Equal(e.Bytes(), full) {
		t.Fatal("limit changed bytes")
	}
	if !e.Undo() || e.CanUndo() || e.Count(mapedit.KindPlayer) != 15 {
		t.Fatal("limit added undo entries")
	}
}

func TestPlayerDeletePreservesOwnerAndRelationIdentitiesInOneEdit(t *testing.T) {
	b := playerFixture(t, 4)
	e := newEditor(t, b)
	if err := e.Delete(mapedit.KindPlayer, 1); err != nil {
		t.Fatal(err)
	}
	want := append([]byte(nil), b...)
	_, units := recordAt(t, want, 6)
	for i, owner := range []uint32{1, 2, 0x10003} {
		binary.LittleEndian.PutUint32(want[units+70*i+0x14:], owner)
	}
	_, objects := recordAt(t, want, 4)
	binary.LittleEndian.PutUint32(want[objects+20+0x0e:], 2)
	binary.LittleEndian.PutUint32(want[objects+48+0x0e:], 0x10003)
	_, script := recordAt(t, want, 7)
	binary.LittleEndian.PutUint32(want[script+4+796+0x4c+9*4:], 3)
	binary.LittleEndian.PutUint32(want[script+4+2*796+4+796+0x4c+9*4:], 2)
	h, roster := recordAt(t, want, 5)
	for i := 0; i < 4; i++ {
		if i == 1 {
			continue
		}
		row := want[roster+76*i+44 : roster+76*(i+1)]
		copy(row[2:], row[4:])
		clear(row[30:])
	}
	want = splice(want, roster+76, 76, nil)
	addWord(want, h+8, -76)
	addWord(want, metaAt(t, want, 0x1c), -1)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("delete did not preserve the surviving identities exactly")
	}
	undoRedoExact(t, e, 1, b, want)
}

func TestPlayerDeleteRefusesEveryReferenceAndTheFirstPlayer(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		offset        func(*testing.T, []byte) int
	}{
		{"unit", "player slot 2: unit 2", func(t *testing.T, b []byte) int { _, p := recordAt(t, b, 6); return p + 2*70 + 0x14 }},
		{"structure", "player slot 2: structure 2", func(t *testing.T, b []byte) int { _, p := recordAt(t, b, 4); return p + 48 + 0x0e }},
		{"extended structure", "player slot 2: structure 1", func(t *testing.T, b []byte) int { _, p := recordAt(t, b, 4); return p + 20 + 0x0e }},
		{"action", "player slot 2: action 1 parameter 9", func(t *testing.T, b []byte) int { _, p := recordAt(t, b, 7); return p + 4 + 796 + 0x4c + 9*4 }},
		{"condition", "player slot 2: condition 1 parameter 9", func(t *testing.T, b []byte) int {
			_, p := recordAt(t, b, 7)
			return p + 4 + 2*796 + 4 + 796 + 0x4c + 9*4
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := playerFixture(t, 4)
			binary.LittleEndian.PutUint32(b[tc.offset(t, b):], 2)
			e := newEditor(t, b)
			if err := e.Delete(mapedit.KindPlayer, 1); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("delete error %v, want %q", err, tc.message)
			}
			if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
				t.Fatal("refused delete mutated bytes or history")
			}
		})
	}
	b := playerFixture(t, 4)
	e := newEditor(t, b)
	for _, index := range []int{0, -1, 4} {
		if err := e.Delete(mapedit.KindPlayer, index); err == nil {
			t.Fatalf("deleted player %d", index)
		}
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("invalid index delete mutated the map")
	}
}

func TestPlayerDeleteFailsClosedOnMalformedScript(t *testing.T) {
	for _, array := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(array), func(t *testing.T) {
			b := playerFixture(t, 4)
			_, p := recordAt(t, b, 7)
			p += []int{0, 4 + 2*796, 8 + 4*796}[array]
			binary.LittleEndian.PutUint32(b[p:], 0xffffffff)
			e := newEditor(t, b)
			if err := e.Delete(mapedit.KindPlayer, 1); err == nil || !strings.Contains(err.Error(), "type-7") {
				t.Fatalf("malformed script delete: %v", err)
			}
			if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
				t.Fatal("malformed script changed bytes or history")
			}
		})
	}
}

func TestPlayerDeleteBeyondTheRelationColumnsPreservesTheRows(t *testing.T) {
	b := playerFixture(t, 17)
	_, units := recordAt(t, b, 6)
	binary.LittleEndian.PutUint32(b[units+2*70+0x14:], 1)
	_, objects := recordAt(t, b, 4)
	binary.LittleEndian.PutUint32(b[objects+48+0x0e:], 1)
	e := newEditor(t, b)
	if err := e.Delete(mapedit.KindPlayer, 16); err != nil {
		t.Fatal(err)
	}
	h, roster := recordAt(t, b, 5)
	want := splice(b, roster+16*76, 76, nil)
	addWord(want, h+8, -76)
	addWord(want, metaAt(t, want, 0x1c), -1)
	if !bytes.Equal(e.Bytes(), want) || e.Count(mapedit.KindPlayer) != 16 {
		t.Fatal("deleting a player without a relation column changed the remaining rows")
	}
	undoRedoExact(t, e, 1, b, want)
}

func TestPlayerDeleteRefusesStructureLowWordAliases(t *testing.T) {
	for _, tc := range []struct {
		index, offset int
	}{{1, 20}, {2, 48}} {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			b := playerFixture(t, 4)
			_, p := recordAt(t, b, 4)
			binary.LittleEndian.PutUint32(b[p+tc.offset+0x0e:], 0xf1230002)
			e := newEditor(t, b)
			want := fmt.Sprintf("player slot 2: structure %d", tc.index)
			if err := e.Delete(mapedit.KindPlayer, 1); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("structure owner alias: %v, want %q", err, want)
			}
			if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
				t.Fatal("refused alias delete changed bytes or history")
			}
		})
	}
}

func TestPlayerDeletePreservesReferencesOutsideTheRoster(t *testing.T) {
	b := playerFixture(t, 4)
	_, units := recordAt(t, b, 6)
	binary.LittleEndian.PutUint32(b[units+2*70+0x14:], 0xf1230002)
	_, objects := recordAt(t, b, 4)
	binary.LittleEndian.PutUint32(b[objects+48+0x0e:], 0xf1230009)
	e := newEditor(t, b)
	if err := e.Delete(mapedit.KindPlayer, 1); err != nil {
		t.Fatal(err)
	}
	after := e.Bytes()
	_, units = recordAt(t, after, 6)
	_, objects = recordAt(t, after, 4)
	_, script := recordAt(t, after, 7)
	for _, tc := range []struct {
		offset int
		value  uint32
	}{
		{units + 2*70 + 0x14, 0xf1230002},
		{objects + 48 + 0x0e, 0xf1230009},
		{script + 4 + 796 + 0x4c + 8*4, 0x10003},
		{script + 4 + 2*796 + 4 + 796 + 0x4c + 8*4, 0xffffffff},
	} {
		if got := binary.LittleEndian.Uint32(after[tc.offset:]); got != tc.value {
			t.Fatalf("missing reference at %d changed from %#x to %#x", tc.offset, tc.value, got)
		}
	}
	undoRedoExact(t, e, 1, b, after)
}

func absentPlayerMap(t *testing.T, trailing bool, stale uint32) ([]byte, int) {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16})
	h, p := recordAt(t, b, 2)
	end := p + int(binary.LittleEndian.Uint32(b[h+8:]))
	if !trailing {
		b = b[:end]
	}
	binary.LittleEndian.PutUint32(b[12:], 3)
	binary.LittleEndian.PutUint32(b[metaAt(t, b, 0x1c):], stale)
	return b, end
}

func assertCreatedPlayerSection(t *testing.T, b []byte) (*mapedit.Editor, []byte) {
	t.Helper()
	e := newEditor(t, b)
	if e.Count(mapedit.KindPlayer) != 0 {
		t.Fatal("absent roster exposed its stale count or uncounted bytes")
	}
	rec, err := e.NewPlayerRecord()
	if err != nil {
		t.Fatal(err)
	}
	wantRecord := make([]byte, 76)
	binary.LittleEndian.PutUint32(wantRecord, 1)
	binary.LittleEndian.PutUint32(wantRecord[4:], 1)
	binary.LittleEndian.PutUint32(wantRecord[8:], 5000)
	copy(wantRecord[12:44], "New Player 1")
	for j := 0; j < 16; j++ {
		value := uint16(1)
		if j == 0 {
			value = 2
		}
		binary.LittleEndian.PutUint16(wantRecord[44+2*j:], value)
	}
	if !bytes.Equal(rec, wantRecord) || !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("constructor used the stale count or mutated the map")
	}
	if index, err := e.Place(mapedit.KindPlayer, rec); err != nil || index != 0 {
		t.Fatalf("first placement: %d, %v", index, err)
	}
	header := []byte{7, 0, 0, 0, 20, 0, 0, 0, 76, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0}
	metaHeader, meta := recordAt(t, b, 0)
	insert := meta + int(binary.LittleEndian.Uint32(b[metaHeader+8:]))
	want := splice(b, insert, 0, append(header, wantRecord...))
	binary.LittleEndian.PutUint32(want[12:], 4)
	binary.LittleEndian.PutUint32(want[meta+0x1c:], 1)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("section creation changed bytes outside its new frame and two counts")
	}
	m, err := e.Map()
	if err != nil || len(m.Groups) != 1 || !m.Present(5) || m.Groups[0].Name != "New Player 1" {
		t.Fatalf("created roster is not readable: %v", err)
	}
	if saved, err := e.SaveBytes(); err != nil || !bytes.Equal(saved, want) {
		t.Fatalf("created roster does not save unchanged: %v", err)
	}
	return e, want
}

func TestPlayerCreationReplacesAbsentRosterStaleCounts(t *testing.T) {
	for _, stale := range []uint32{0, 29, 0xffffffff} {
		t.Run(fmt.Sprint(stale), func(t *testing.T) {
			b, _ := absentPlayerMap(t, false, stale)
			e, want := assertCreatedPlayerSection(t, b)
			undoRedoExact(t, e, 1, b, want)
		})
	}
}

func TestPlayerCreationPreservesUncountedTrailingRecords(t *testing.T) {
	b, end := absentPlayerMap(t, true, 0xf1234567)
	trailer := append([]byte(nil), b[end:]...)
	e, want := assertCreatedPlayerSection(t, b)
	if !bytes.Equal(want[end+96:], trailer) {
		t.Fatal("section creation changed the uncounted records")
	}
	if e.Count(mapedit.KindPlayer) != 1 || e.Count(mapedit.KindUnit) != 0 || e.Count(mapedit.KindAction) != 0 {
		t.Fatal("section creation made trailing records reachable")
	}
	undoRedoExact(t, e, 1, b, want)
}

func TestAbsentRosterRefusesInvalidPlacementWithoutCreatingASection(t *testing.T) {
	b, _ := absentPlayerMap(t, true, 0xffffffff)
	e := newEditor(t, b)
	rec, err := e.NewPlayerRecord()
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(rec, 16)
	if _, err := e.Place(mapedit.KindPlayer, rec); err == nil {
		t.Fatal("invalid player created a section")
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("invalid placement changed bytes or history")
	}
}
