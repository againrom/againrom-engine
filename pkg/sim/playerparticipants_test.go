package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func participantWorld(t *testing.T) *World {
	t.Helper()
	w := savedGroupWorld(t)
	if err := w.ImportSavedGroupPlayers([]SavedGroupPlayer{{101, 7}, {205, 7}}, []SavedGroupContainer{{71, 101}, {72, 205}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestPlayerParticipantExactIdentityPresenceAndHash(t *testing.T) {
	w := participantWorld(t)
	legacy, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	rows := []PlayerParticipant{{101, 0}, {205, 0xf1234567}}
	if err := w.RestorePlayerParticipants(rows, true); err != nil {
		t.Fatal(err)
	}
	rows[0].Value = 99
	got, present := w.PlayerParticipants()
	if !present || !reflect.DeepEqual(got, []PlayerParticipant{{101, 0}, {205, 0xf1234567}}) {
		t.Fatal("current zero or distinct shared-slot Player was lost", got, present)
	}
	got[1].Value = 23
	before := w.Hash()
	if w.SetPlayerParticipant(7, 1) == nil || w.Hash() != before {
		t.Fatal("semantic slot was accepted as an exact Player ID")
	}
	if err := w.SetPlayerParticipant(205, 0); err != nil || w.Hash() == before {
		t.Fatal("current zero did not reach hashed state", err)
	}
	form, err := w.MarshalBinary()
	if err != nil || CheckSaveForm(form) != nil || form[0] != playerParticipantFormVersion {
		t.Fatal("Participant form was not admitted", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("Participant was lost by canonical transport", err)
	}
	if err := cold.UnmarshalBinary(legacy); err != nil {
		t.Fatal(err)
	}
	if got, present := cold.PlayerParticipants(); present || len(got) != 0 {
		t.Fatal("old absence was synthesized as current zero", got)
	}
	if err := w.RestorePlayerParticipants(nil, false); err != nil {
		t.Fatal(err)
	}
	absent, _ := w.MarshalBinary()
	if !bytes.Equal(absent, legacy) {
		t.Fatal("restored absence changed unrelated canonical state")
	}
}

func TestPlayerParticipantRejectsIncompleteAndCorruptStateAtomically(t *testing.T) {
	w := participantWorld(t)
	rows := []PlayerParticipant{{101, 0}, {205, 0x87654321}}
	if err := w.RestorePlayerParticipants(rows, true); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	for _, bad := range [][]PlayerParticipant{nil, rows[:1], {{101, 1}, {101, 2}}, {{205, 2}, {101, 1}}, {{101, 1}, {206, 2}}} {
		if w.RestorePlayerParticipants(bad, true) == nil || w.Hash() != before {
			t.Fatal("incomplete, unordered or unknown identity changed current state", bad)
		}
	}
	if w.RestorePlayerParticipants(rows, false) == nil || w.Hash() != before {
		t.Fatal("absent values changed current state")
	}
	raw, _ := w.MarshalBinary()
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	for _, edit := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], ^uint32(0)) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 0) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+12:], 101) },
		func(b []byte) { b[len(b)-5] = playerParticipantFormVersion },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-9:], ^uint32(0)) },
	} {
		bad := bytes.Clone(raw)
		edit(bad)
		if w.UnmarshalBinary(bad) == nil || CheckSaveForm(bad) == nil || w.Hash() != before {
			t.Fatal("corrupt Participant form was accepted or changed current state")
		}
	}
}

func TestPlayerParticipantTracksGroupAndPlayerLifetime(t *testing.T) {
	w := participantWorld(t)
	if err := w.RestorePlayerParticipants([]PlayerParticipant{{101, 17}, {205, 29}}, true); err != nil {
		t.Fatal(err)
	}
	groups, orders, _ := w.SavedGroups()
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreGroupContinuations([]GroupContinuation{{Group: 71, ID: 71}, {Group: 72, RootOnly: true}}, 72, 205); err != nil {
		t.Fatal(err)
	}
	if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, []PlayerParticipant{{101, 17}}) {
		t.Fatal("Player removal lost surviving value or retained removed identity", got)
	}
	if err := w.RestoreGroupCarrierPresence(false, false); err != nil {
		t.Fatal(err)
	}
	if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, []PlayerParticipant{{101, 17}}) {
		t.Fatal("removing command provenance lost surviving current Player", got)
	}
	empty := savedGroupWorld(t)
	if err := empty.ImportSavedGroupPlayers(nil, nil); err == nil {
		t.Fatal("fixture requires complete Group containers")
	}
	if err := empty.ImportSavedGroups(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := empty.ImportSavedGroupPlayers(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := empty.RestorePlayerParticipants(nil, true); err != nil {
		t.Fatal(err)
	}
	raw, err := empty.MarshalBinary()
	var cold World
	if err != nil || cold.UnmarshalBinary(raw) != nil {
		t.Fatal("present empty Participant registry failed transport", err)
	}
	if got, present := cold.PlayerParticipants(); !present || len(got) != 0 {
		t.Fatal("present empty Participant registry became absent")
	}
}

func TestPlayerParticipantWrapsIndependentBasisAndBookHistory(t *testing.T) {
	w := participantWorld(t)
	e := &w.entities[0]
	e.AdmittedBookSpell = 28
	e.NativeBasis = NativeActorBasis{BodyPresent: true, BodyKnown: true, Body: 61}
	base, err := w.MarshalBinary()
	if err != nil || base[0] != nativeBasisFormVersion {
		t.Fatal("basis did not wrap book history", err)
	}
	if err := w.RestorePlayerParticipants([]PlayerParticipant{{101, 0}, {205, 0xf1234567}}, true); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	var cold World
	if err != nil || cold.UnmarshalBinary(raw) != nil || CheckSaveForm(raw) != nil || cold.Hash() != w.Hash() || HasStructureBlockingForm(raw) != HasStructureBlockingForm(base) {
		t.Fatal("Participant wrapper lost basis, book or predecessor policy", err)
	}
	if got := cold.entities[0]; got.AdmittedBookSpell != 28 || got.NativeBasis != e.NativeBasis {
		t.Fatal("wrapped independent actor histories changed")
	}
}

func TestPlayerParticipantSurvivesLegacyGroupCarrierAbsence(t *testing.T) {
	w := participantWorld(t)
	want := []PlayerParticipant{{101, 0}, {205, 0xf1234567}}
	if err := w.RestorePlayerParticipants(want, true); err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreGroupCarrierPresence(false, false); err != nil {
		t.Fatal(err)
	}
	if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
		t.Fatal("legacy native Group absence erased independent Participant words", got, present)
	}
	if _, _, present := w.SavedGroups(); !present {
		t.Fatal("Participant repair removed unrelated native Group state")
	}
	if _, present := w.SavedGroupPlayers(); present {
		t.Fatal("Participant repair changed legacy command/container dispatch")
	}
	form, err := w.MarshalBinary()
	var cold World
	if err != nil || cold.UnmarshalBinary(form) != nil || cold.Hash() != w.Hash() {
		t.Fatal("independent Participant did not survive canonical transport", err)
	}
	if got, present := cold.PlayerParticipants(); !present || !reflect.DeepEqual(got, want) {
		t.Fatal("independent Participant words changed on cold LOAD", got, present)
	}
}

func TestCurrentPlayerRegistryIndependentScopeAndTransport(t *testing.T) {
	w := spWorld(t, 917, nil, spEnt(1, 1, 1))
	legacy, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	players := []SavedGroupPlayer{{101, 0}, {205, 0}}
	rows := []PlayerParticipant{{101, 0}, {205, 0xf1234567}}
	if err := w.RestoreCurrentPlayers(players, rows, true); err != nil {
		t.Fatal(err)
	}
	players[0].ID, rows[0].Value = 1, 91
	got, present := w.CurrentPlayers()
	if !present || !reflect.DeepEqual(got, []SavedGroupPlayer{{101, 0}, {205, 0}}) {
		t.Fatal("exact current IDs or valid zero Slot were lost", got)
	}
	got[1].Slot = 7
	if _, _, present := w.SavedGroups(); present {
		t.Fatal("current Player registry installed command/Group state")
	}
	form, err := w.MarshalBinary()
	if err != nil || form[0] != currentPlayerFormVersion || CheckSaveForm(form) != nil {
		t.Fatal("independent Player form was not admitted", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("independent Player ID/Slot/value transport changed hashed state", err)
	}
	if err := cold.RestorePlayerParticipants(nil, false); err != nil {
		t.Fatal(err)
	}
	absent, err := cold.MarshalBinary()
	if err != nil || cold.UnmarshalBinary(absent) != nil || cold.Hash() == w.Hash() {
		t.Fatal("Participant absence collapsed to present zero", err)
	}
	if got, present := cold.CurrentPlayers(); !present || len(got) != 2 {
		t.Fatal("Participant absence erased exact Player identities")
	}
	if _, present := cold.PlayerParticipants(); present {
		t.Fatal("absent Participant carrier became present")
	}
	before := w.Hash()
	start := len(form) - 9 - int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	for _, edit := range []func([]byte){
		func(b []byte) { b[start] = 2 },
		func(b []byte) { b[start] = 0 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+1:], 65536) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+5:], 0) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+17:], 101) },
		func(b []byte) { b[len(b)-5] = currentPlayerFormVersion },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-9:], ^uint32(0)) },
	} {
		bad := bytes.Clone(form)
		edit(bad)
		if w.UnmarshalBinary(bad) == nil || w.Hash() != before || CheckSaveForm(bad) == nil {
			t.Fatal("malformed independent Player form changed current state")
		}
	}
	if err := cold.UnmarshalBinary(legacy); err != nil {
		t.Fatal(err)
	}
	if _, present := cold.CurrentPlayers(); present {
		t.Fatal("old absent Player registry acquired constructor IDs")
	}
}

func TestCurrentPlayerIdentityRenamePreservesExactValuesAndDispatch(t *testing.T) {
	w := participantWorld(t)
	if err := w.RestorePlayerParticipants([]PlayerParticipant{{101, 0}, {205, 0xf1234567}}, true); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	for _, ids := range []map[uint32]uint32{{101: 501}, {101: 501, 205: 501}, {101: 605, 205: 501}, {101: 501, 206: 605}} {
		if w.RestoreCurrentPlayerIdentities(ids) == nil || w.Hash() != before {
			t.Fatal("invalid current identity join changed state", ids)
		}
	}
	entities := w.Entities()
	groups, orders, _ := w.SavedGroups()
	if err := w.RestoreCurrentPlayerIdentities(map[uint32]uint32{101: 501, 205: 605}); err != nil {
		t.Fatal(err)
	}
	if got, present := w.CurrentPlayers(); !present || !reflect.DeepEqual(got, []SavedGroupPlayer{{501, 7}, {605, 7}}) {
		t.Fatal("identity join lost shared Slots", got)
	}
	if got, present := w.PlayerParticipants(); !present || !reflect.DeepEqual(got, []PlayerParticipant{{501, 0}, {605, 0xf1234567}}) {
		t.Fatal("identity join changed current Participant", got)
	}
	for i := range groups {
		if groups[i].ContainerID == 101 {
			groups[i].ContainerID = 501
		} else if groups[i].ContainerID == 205 {
			groups[i].ContainerID = 605
		}
	}
	got, nextOrders, _ := w.SavedGroups()
	if !reflect.DeepEqual(got, groups) || !reflect.DeepEqual(nextOrders, orders) || !reflect.DeepEqual(w.Entities(), entities) {
		t.Fatal("identity transport changed dispatch or semantic actor ownership")
	}
	raw, err := w.MarshalBinary()
	var cold World
	if err != nil || cold.UnmarshalBinary(raw) != nil || cold.Hash() != w.Hash() {
		t.Fatal("renamed exact identity did not survive canonical transport", err)
	}
}
