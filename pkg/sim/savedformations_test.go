package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func formation1159World(t *testing.T) *World {
	t.Helper()
	w := mustWorld(t, 1, fmBounds, []Entity{
		{ID: 10, Owner: 2, X: 4, Y: 8, HP: 100, MaxHP: 100, Speed: 20},
		{ID: 20, Owner: 2, X: 6, Y: 8, HP: 100, MaxHP: 100, Speed: 30},
	})
	g := SavedGroup{ID: 71, Selector: 19, Owner: SavedGroupReference{Class: 1, Owner: 1},
		Members: []SavedGroupMember{{Entity: 10, Bound: true}, {Entity: 20, Bound: true}}}
	if err := w.ImportSavedGroups([]SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{101, 1}, {205, 2}}, []SavedGroupContainer{{71, 205}}); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedPlayerFormations([]SavedPlayerFormation{{101, 1, 2, 0}, {205, 2, 1, 1}}, []SavedGroupOwner{{71, 101}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func formation1159Move(t *testing.T, w *World, x int32, want [2]int32) {
	t.Helper()
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 19, Args: [10]int32{4, x, 30}})
	for i, e := range w.Entities() {
		if !e.HasTarget || e.TargetX != want[i] || e.TargetY != 30 {
			t.Fatalf("next Move entity %d target=(%d,%d) want=(%d,30)", e.ID, e.TargetX, e.TargetY, want[i])
		}
	}
}

func TestSavedFormation1159OwnerCommandAndTrigger(t *testing.T) {
	w := formation1159World(t)
	formation1159Move(t, w, 40, [2]int32{40, 40}) // owner101=Off; container205/member2=On.
	w.applyPlayerParameter(1, PlayerParameterFormation, 2)
	formation1159Move(t, w, 40, [2]int32{39, 41}) // +04=1 resolves101, remap2->1.
	w.runInstant(ScriptInstant{Op: ScriptInstantFormation, HasPlayer: true, Player: 2, Args: [10]int32{0}})
	formation1159Move(t, w, 40, [2]int32{40, 40}) // +08=2 resolves101, raw0.
	players, _ := w.SavedPlayerFormations()
	if players[1].Mode != 1 {
		t.Fatal("command/trigger changed the other Player", players)
	}
	for _, mode := range []int32{2, 255, 257} {
		w.setFormationMode(2, mode)
		formation1159Move(t, w, 40, [2]int32{39, 41})
	}
	// Null is an unsupported native boundary, not original reachability proof.
	w.savedGroups.Groups[0].OwnerID = 0
	before := w.entities[0].TargetX
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 19, Args: [10]int32{4, 45, 30}})
	if w.entities[0].TargetX != before || len(w.SavedGroupIssues()) == 0 {
		t.Fatal("unknown owner silently chose a member/container/default")
	}
}

func TestSavedFormation1159DuplicateSignedCommandAndAmbiguousTrigger(t *testing.T) {
	w := formation1159World(t)
	w.savedGroups.Players[0].Slot, w.savedGroups.Players[1].Slot = 65535, 65535
	w.savedGroups.Formations[0].CommandID, w.savedGroups.Formations[1].CommandID = -1, -1
	w.applyPlayerParameter(^uint32(0), PlayerParameterFormation, 2)
	if w.savedGroups.Formations[0].Mode != 1 || w.savedGroups.Formations[1].Mode != 1 {
		t.Fatal("signed command did not match")
	}
	w.savedGroups.Formations[1].Mode = 255
	w.applyPlayerParameter(65535, PlayerParameterFormation, 0)
	if w.savedGroups.Formations[0].Mode != 0 || w.savedGroups.Formations[1].Mode != 255 {
		t.Fatal("duplicate +04 failed first-list-match")
	}
	w.savedGroups.Formations[0].TriggerID = 1
	before := w.Hash()
	w.setFormationMode(1, 9)
	w.applyPlayerParameter(33, PlayerParameterFormation, 1)
	if w.Hash() != before {
		t.Fatal("ambiguous trigger or missing command fabricated a Player")
	}
}

func TestSavedFormation1159NativeLiteralAndAtomicControls(t *testing.T) {
	w := formation1159World(t)
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{2, 0, 0, 0, 101, 0, 0, 0, 1, 0, 2, 0, 0, 0, 0, 205, 0, 0, 0, 2, 0, 1, 0, 0, 0, 1, 1, 0, 0, 0, 71, 0, 0, 0, 101, 0, 0, 0, 38, 0, 0, 0}
	// entityIDFloor (form94) closes the form outside this section entirely.
	end := len(valid) - entityIDFloorLen - spellDeliverySpanLen
	start := end - 8 - len(want)
	if !bytes.Equal(valid[start:end-8], want) {
		t.Fatalf("literal footer %x want %x", valid[start:end-8], want)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(valid); err != nil || fresh.Hash() != w.Hash() {
		t.Fatal("native roundtrip", err)
	}
	formation1159Move(t, &fresh, 40, [2]int32{40, 40})
	for _, edit := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[end-8:], ^uint32(0)) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], ^uint32(0)) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 205) },
		func(b []byte) { b[start+8] = 3 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+26:], 2) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+30:], 99) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+34:], 999) },
	} {
		bad := bytes.Clone(valid)
		edit(bad)
		before := fresh.Hash()
		if err := fresh.UnmarshalBinary(bad); err == nil || fresh.Hash() != before {
			t.Fatal("corrupt footer accepted/mutated receiver", err)
		}
	}
}

func TestSavedFormation1159EmptyPresenceAndEqualValuedPlayers(t *testing.T) {
	w := mustWorld(t, 1, fmBounds, nil)
	if err := w.ImportSavedGroups(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroupPlayers(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedPlayerFormations(nil, nil); err != nil {
		t.Fatal(err)
	}
	b, err := w.MarshalBinary()
	end := len(b) - entityIDFloorLen - spellDeliverySpanLen
	if err != nil || !bytes.Equal(b[end-20:end-8], []byte{0, 0, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0}) {
		t.Fatal("present empty formation footer", err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(b); err != nil || fresh.Hash() != w.Hash() {
		t.Fatal(err)
	}
	if p, present := fresh.SavedPlayerFormations(); !present || len(p) != 0 {
		t.Fatal("empty exact registry collapsed into legacy absence")
	}
	before := fresh.Hash()
	if err := fresh.ImportSavedPlayerFormations(nil, nil); err == nil || fresh.Hash() != before {
		t.Fatal("native LOAD state could be imported a second time", err)
	}
	w = formation1159World(t)
	w.savedGroups.Players[0].Slot, w.savedGroups.Players[1].Slot = 7, 7
	w.savedGroups.Formations = []SavedPlayerFormation{{101, 7, 9, 255}, {205, 7, 9, 255}}
	b, err = w.MarshalBinary()
	if err != nil || fresh.UnmarshalBinary(b) != nil || fresh.Hash() != w.Hash() {
		t.Fatal("equal values lost distinct Player identities", err)
	}
	fresh.applyPlayerParameter(7, PlayerParameterFormation, 0)
	p, _ := fresh.SavedPlayerFormations()
	if !reflect.DeepEqual(p, []SavedPlayerFormation{{101, 7, 9, 0}, {205, 7, 9, 255}}) {
		t.Fatal("duplicate command did not choose first retained Player", p)
	}
	// Reverse the two semantic trigger identities while retaining native list
	// order. A command still changes the first duplicate +04; a trigger follows
	// its unique +08 to the opposite record after the reversal.
	for _, reversed := range []bool{false, true} {
		fresh.savedGroups.Formations[0].Mode, fresh.savedGroups.Formations[1].Mode = 255, 255
		a, b := uint32(1), uint32(2)
		if reversed {
			a, b = b, a
		}
		fresh.savedGroups.Formations[0].TriggerID, fresh.savedGroups.Formations[1].TriggerID = a, b
		fresh.applyPlayerParameter(7, PlayerParameterFormation, 0)
		fresh.setFormationMode(1, 2)
		want := [2]uint8{2, 255}
		if reversed {
			want = [2]uint8{0, 2}
		}
		p, _ := fresh.SavedPlayerFormations()
		if p[0].Mode != want[0] || p[1].Mode != want[1] {
			t.Fatal("command list order conflated with trigger identity", reversed, p)
		}
	}
}

// Independent append/peel adapters preserve every prior pinned byte.
func widenedSavedFormationPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 90
	return widenedSavedWorldEffectsPin(out)
}

func strippedSavedFormationPin(form []byte) []byte {
	out := strippedSavedWorldEffectsPin(form)
	if len(out) > 0 && out[0] >= 90 {
		n := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-n]
		out[0] = 89
	}
	return out
}
