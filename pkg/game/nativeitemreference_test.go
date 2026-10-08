package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestNativeItemReferenceAbsenceUsesBoundObjectIdentity(t *testing.T) {
	const firstKey, secondKey, renamedKey, unresolved = uint32(0x100001), uint32(0x100002), uint32(0x110001), uint32(0x7f123456)
	for _, name := range []string{"same Player renamed", "another object", "unresolved reference"} {
		t.Run(name, func(t *testing.T) {
			first, second, item := mustNewRecord("Player"), mustNewRecord("Player"), mustNewRecord("Item")
			mustSetValue(&first, "This", firstKey)
			mustSetValue(&second, "This", secondKey)
			mustSetValue(&item, "Identity", 0x200001)
			mustSetValue(&item, "RuntimeID", 3)
			mustSetValue(&item, "Reference", firstKey)
			mustSetValue(&item, "F40", 0x0e01)
			mustSetValue(&item, "F42", 1)
			mustSetValue(&item, "T1C", 19)
			doc := sav.DocumentData{Version: sav.DocumentDataVersion, Players: []uint16{1, 2}, Objects: []sav.DocumentRecordData{first, second, item}}
			policy := currentOwnedObject{Object: 3, Kind: 1}
			if err := captureCurrentItemAbsence(&doc, &policy); err != nil {
				t.Fatal(err)
			}
			if policy.ID != 0 || policy.NativeRecordKnown || policy.NativeRecordAnchor == nil {
				t.Fatal("fixture lacks an unregistered item with observed native record absence")
			}
			baseline, err := readCurrentItem(&doc, policy, nil)
			if err != nil || baseline.Value.NativeRecord != nil {
				t.Fatal("unchanged ordinary item acquired a native record", baseline.Value, err)
			}
			wantReference := firstKey
			switch name {
			case "same Player renamed":
				mustSetValue(&doc.Objects[0], "This", renamedKey)
				wantReference = renamedKey
			case "another object":
				wantReference = secondKey
			case "unresolved reference":
				wantReference = unresolved
			}
			mustSetValue(&doc.Objects[2], "Reference", wantReference)
			nextPolicy := policy
			if err := captureNativeItemRecordAnchor(&doc, &nextPolicy); err != nil {
				t.Fatal(err)
			}
			renamed := name == "same Player renamed"
			if nextPolicy.NativeRecordAnchor == nil || (*policy.NativeRecordAnchor == *nextPolicy.NativeRecordAnchor) != renamed {
				t.Fatalf("reference absence anchor does not distinguish key renaming from an ordinary reference edit: before %v after %v", policy.NativeRecordAnchor, nextPolicy.NativeRecordAnchor)
			}
			got, err := readCurrentItem(&doc, policy, nil)
			if err != nil {
				t.Fatal(err)
			}
			if renamed {
				if got.Value.NativeRecord != nil {
					t.Fatal("renaming the same Player changed observed native record absence", got.Value.NativeRecord)
				}
			} else if got.Value.NativeRecord == nil || got.Value.NativeRecord.Token.Reference != wantReference {
				t.Fatal("ordinary reference edit lost to native record absence", got.Value.NativeRecord, wantReference)
			}
		})
	}
}

func TestNativeItemReferenceAnchorHasIndependentVersionedBytes(t *testing.T) {
	const playerKey, otherKey, itemKey = uint32(0x100001), uint32(0x100002), uint32(0x200001)
	for _, name := range []string{"Player root", "Player renamed", "other Player", "other object", "unresolved", "ambiguous"} {
		t.Run(name, func(t *testing.T) {
			first, second, item := mustNewRecord("Player"), mustNewRecord("Player"), mustNewRecord("Item")
			mustSetValue(&first, "This", playerKey)
			mustSetValue(&second, "This", otherKey)
			mustSetValue(&item, "Identity", itemKey)
			mustSetValue(&item, "RuntimeID", 3)
			mustSetValue(&item, "F40", 0x0e01)
			mustSetValue(&item, "F42", 1)
			doc := sav.DocumentData{Version: sav.DocumentDataVersion, Players: []uint16{1, 2}, Objects: []sav.DocumentRecordData{first, second, item}}
			modes := nativeActorItemModes{playerRoots: []uint16{2, 5}, objectKeys: map[uint16]uint32{2: playerKey, 5: otherKey, 9: itemKey}}
			reference, ordinal := playerKey, uint32(1)
			switch name {
			case "Player renamed":
				reference = 0x110001
				mustSetValue(&doc.Objects[0], "This", reference)
				modes.objectKeys[2] = reference
			case "other Player":
				reference, ordinal = otherKey, 2
			case "other object":
				reference, ordinal = itemKey, 0
			case "unresolved":
				reference, ordinal = 0x7f123456, 0
			case "ambiguous":
				ordinal = 0
				mustSetValue(&doc.Objects[1], "This", playerKey)
				modes.objectKeys[5] = playerKey
			}
			mustSetValue(&doc.Objects[2], "Reference", reference)
			history := sim.NativeItemRecord{Token: sim.SavedObjectToken{RuntimeID: 3, Identity: itemKey, Reference: reference}}
			anchored := history
			prefix := []byte{1, 0}
			if ordinal != 0 {
				prefix = []byte{1, 1, byte(ordinal), 0, 0, 0}
				anchored.Token.Reference = 0
			}
			wire, err := binary.Append(prefix, binary.LittleEndian, anchored)
			if err != nil {
				t.Fatal(err)
			}
			want := sha256.Sum256(wire)
			policy := currentOwnedObject{Object: 3, Kind: 1}
			if err := captureNativeItemRecordAnchor(&doc, &policy); err != nil {
				t.Fatal(err)
			}
			if policy.NativeRecordAnchorVersion != 1 || policy.NativeRecordAnchor == nil || *policy.NativeRecordAnchor != want {
				t.Fatal("producer record anchor differs from independent version/mode/root bytes", policy, want)
			}
			if got := nativeActorItemRecordAnchor(history, 1, modes); got != want {
				t.Fatal("independent raw reader record anchor differs", got, want)
			}
			legacyWire, err := binary.Append(nil, binary.LittleEndian, history)
			if err != nil {
				t.Fatal(err)
			}
			legacy := sha256.Sum256(legacyWire)
			policy.NativeRecordAnchorVersion, policy.NativeRecordAnchor = 0, &legacy
			got, err := readCurrentItem(&doc, policy, nil)
			if err != nil || got.Value.NativeRecord != nil || nativeActorItemRecordAnchor(history, 0, modes) != legacy {
				t.Fatal("legacy raw anchor behavior changed", got.Value.NativeRecord, err)
			}
			for _, version := range []uint8{1, 2} {
				bad := policy
				bad.NativeRecordAnchorVersion = version
				if version == 1 {
					bad.NativeRecordAnchor = nil
				}
				if _, err := readCurrentItem(&doc, bad, nil); err == nil {
					t.Fatal("unbound or unsupported record anchor version accepted", version)
				}
			}
		})
	}
}

func TestNativeItemReferenceEditsSurviveColdLOADAndNextSAVE(t *testing.T) {
	for _, name := range []string{"another object", "unresolved reference"} {
		t.Run(name, func(t *testing.T) {
			raw, source, _, id := nativeActorNativeItemFixture(t)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil || len(doc.Players) == 0 {
				t.Fatal("synthetic current item fixture lacks Player or item bindings", err)
			}
			r := nativeActorNativeItemRecord(t, &doc, id, 0)
			var policy *currentOwnedObject
			for i := range a.Ownership {
				row := &a.Ownership[i]
				if row.Kind == 1 && row.Object != 0 && &doc.Objects[row.Object-1] == r {
					policy = row
				}
			}
			if policy == nil || policy.ID != 0 || policy.NativeRecordKnown {
				t.Fatal("synthetic item is not unregistered with absent native record")
			}
			playerKey, err := savedStructureValue(&doc.Objects[doc.Players[0]-1], "This")
			if err != nil || playerKey == 0 {
				t.Fatal("synthetic Player has no ordinary identity", err)
			}
			mustSetValue(r, "Reference", playerKey)
			if err := captureCurrentItemAbsence(&doc, policy); err != nil {
				t.Fatal(err)
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			baseline, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			unchanged := openCurrentEffectSave(t, source, baseline)
			pack, ok := unchanged.live.world.CarriedStacks(id)
			if !ok || len(pack) == 0 || pack[0].ObjectID != 0 || pack[0].NativeRecord != nil {
				t.Fatal("cold unchanged ordinary Reference acquired a native record", pack)
			}
			wantReference := uint32(0x7f123456)
			if name == "another object" {
				other := nativeActorNativeItemRecord(t, &doc, id, 1)
				wantReference, err = savedStructureValue(other, "Identity")
				if err != nil || wantReference == 0 || wantReference == playerKey {
					t.Fatal("synthetic edit lacks a different ordinary object", err)
				}
			}
			mustSetValue(r, "Reference", wantReference)
			nextPolicy := *policy
			if err := captureNativeItemRecordAnchor(&doc, &nextPolicy); err != nil {
				t.Fatal(err)
			}
			if policy.NativeRecordAnchor == nil || nextPolicy.NativeRecordAnchor == nil || *policy.NativeRecordAnchor == *nextPolicy.NativeRecordAnchor {
				t.Fatal("ordinary Reference edit did not change the absence anchor")
			}
			edited, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := sav.DecodeDocumentData(edited)
			if err != nil {
				t.Fatal(err)
			}
			retained, _, err := sav.NativeActions(decoded.State)
			if err != nil || !bytes.Equal(leaf, retained) {
				t.Fatal("ordinary Reference edit changed the supplemental policy", err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := openCurrentEffectSave(t, source, edited)
				pack, ok := cold.live.world.CarriedStacks(id)
				if !ok || len(pack) == 0 || pack[0].ObjectID != 0 || pack[0].NativeRecord == nil || pack[0].NativeRecord.Token.Reference != wantReference {
					t.Fatal("cold LOAD lost the ordinary Reference edit", cycle, pack, wantReference)
				}
				var next sav.DocumentData
				edited, next, _ = saveCurrentEffect(t, cold)
				got, err := savedStructureValue(nativeActorNativeItemRecord(t, &next, id, 0), "Reference")
				if err != nil || got != wantReference {
					t.Fatal("next SAVE lost the ordinary Reference edit", cycle, got, wantReference, err)
				}
			}
		})
	}
}
