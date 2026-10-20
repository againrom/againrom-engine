package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCurrentActionWriterCarrierBitsAreStable(t *testing.T) {
	for _, tc := range []struct {
		name string
		bit  uint32
		want uint32
	}{
		{"Buildings", currentCarrierBuildings, 0x001},
		{"CellRecords", currentCarrierCellRecords, 0x002},
		{"DeadRoots", currentCarrierDeadRoots, 0x004},
		{"DyingOwnerActors", currentCarrierDyingOwnerActors, 0x008},
		{"Groups", currentCarrierGroups, 0x010},
		{"MoverRoutes", currentCarrierMoverRoutes, 0x020},
		{"Players", currentCarrierPlayers, 0x040},
		{"Sacks", currentCarrierSacks, 0x080},
		{"SessionBlock", currentCarrierSessionBlock, 0x100},
		{"SpellEffects", currentCarrierSpellEffects, 0x200},
		{"TerrainBlockPlane", currentCarrierTerrainBlockPlane, 0x400},
		{"all", currentCarrierAll, 0x7ff},
	} {
		if tc.bit != tc.want {
			t.Errorf("%s writer carrier bit = %#x, want %#x", tc.name, tc.bit, tc.want)
		}
	}
}

func TestCurrentActionWriterCarriersRequireCheckedSupplement(t *testing.T) {
	if a, err := readCurrentActions(&sav.DocumentData{}); err != nil || a != nil {
		t.Fatal("an original document was admitted as Againrom-written", a, err)
	}
	for _, tc := range []struct {
		name string
		raw  string
		want uint32
		bad  bool
	}{
		{"historical absence", `{"Version":1}`, 0, false},
		{"explicit none", `{"Version":1,"WriterCarriers":0}`, 0, false},
		{"Groups only", `{"Version":1,"WriterCarriers":16}`, 0x010, false},
		{"one folded member", `{"Version":1,"writercarriers":16}`, 0x010, false},
		{"one escaped member", `{"Version":1,"Writer\u0043arriers":16}`, 0x010, false},
		{"one Unicode-folded member", `{"Version":1,"WriterCarrier\u017f":16}`, 0x010, false},
		{"all known", `{"Version":1,"WriterCarriers":2047}`, 0x7ff, false},
		{"unknown bit", `{"Version":1,"WriterCarriers":2048}`, 0, true},
		{"known and unknown", `{"Version":1,"WriterCarriers":2049}`, 0, true},
		{"negative", `{"Version":1,"WriterCarriers":-1}`, 0, true},
		{"fractional", `{"Version":1,"WriterCarriers":0.5}`, 0, true},
		{"overflow", `{"Version":1,"WriterCarriers":4294967296}`, 0, true},
		{"unsupported version", `{"Version":2,"WriterCarriers":0}`, 0, true},
		{"trailing value", `{"Version":1,"WriterCarriers":0} {}`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var doc sav.DocumentData
			if err := sav.SetNativeActions(&doc.State, []byte(tc.raw)); err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if tc.bad {
				if err == nil || a != nil {
					t.Fatal("invalid supplement admitted a writer carrier rule", a, err)
				}
				return
			}
			if err != nil || a == nil || a.WriterCarriers != tc.want {
				t.Fatal("checked writer carrier record differs", a, err)
			}
		})
	}
	var framed sav.DocumentData
	if err := sav.SetNativeActions(&framed.State, []byte(`{"Version":1,"WriterCarriers":0}`)); err != nil {
		t.Fatal(err)
	}
	framed.State.ValueRecords[0].Value.Bytes[0] = 2
	if a, err := readCurrentActions(&framed); err == nil || a != nil {
		t.Fatal("invalid native framing admitted a writer carrier rule", a, err)
	}
}

type writerCarrierRecordCase struct {
	name, members, control string
}

func ambiguousWriterCarrierRecords() []writerCarrierRecordCase {
	return []writerCarrierRecordCase{
		{"null", `"WriterCarriers":null`, `"WriterCarriers":0`},
		{"duplicate known", `"WriterCarriers":16,"WriterCarriers":0`, `"WriterCarriers":16`},
		{"duplicate unknown", `"WriterCarriers":2048,"WriterCarriers":0`, `"WriterCarriers":0`},
		{"lowercase collision", `"WriterCarriers":16,"writercarriers":0`, `"WriterCarriers":16`},
		{"uppercase collision", `"WriterCarriers":16,"WRITERCARRIERS":0`, `"WriterCarriers":16`},
		{"equal duplicates", `"WriterCarriers":0,"WriterCarriers":0`, `"WriterCarriers":0`},
		{"reversed collision", `"writercarriers":0,"WriterCarriers":16`, `"WriterCarriers":16`},
		{"escaped collision", `"WriterCarriers":16,"Writer\u0043arriers":0`, `"WriterCarriers":16`},
		{"unicode collision", `"WriterCarriers":16,"WriterCarrier\u017f":0`, `"WriterCarriers":16`},
		{"string", `"WriterCarriers":"16"`, `"WriterCarriers":16`},
		{"fraction", `"WriterCarriers":0.5`, `"WriterCarriers":0`},
		{"negative", `"WriterCarriers":-1`, `"WriterCarriers":0`},
		{"negative zero", `"WriterCarriers":-0`, `"WriterCarriers":0`},
		{"exponent", `"WriterCarriers":0e0`, `"WriterCarriers":0`},
		{"overflow", `"WriterCarriers":4294967296`, `"WriterCarriers":0`},
		{"unknown bit", `"WriterCarriers":2048`, `"WriterCarriers":0`},
		{"boolean", `"WriterCarriers":false`, `"WriterCarriers":0`},
		{"array", `"WriterCarriers":[]`, `"WriterCarriers":0`},
		{"object", `"WriterCarriers":{}`, `"WriterCarriers":0`},
	}
}

func TestCurrentActionWriterCarriersRejectAmbiguousRecords(t *testing.T) {
	f := currentArchiveFixture(t)
	hash := f.live.world.Hash()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "writer record controls")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	want := uint32(0)
	if _, held := f.live.world.SavedCellPlanes(); held {
		want = currentWriterCarriers
	}
	canonical := []byte(fmt.Sprintf(`"WriterCarriers":%d`, want))
	if err != nil || !present || bytes.Count(leaf, canonical) != 1 {
		t.Fatal("canonical writer record", present, err)
	}
	for _, tc := range ambiguousWriterCarrierRecords() {
		t.Run(tc.name, func(t *testing.T) {
			for _, variant := range []struct {
				name, members string
				bad           bool
			}{{"canonical control", tc.control, false}, {"ambiguous", tc.members, true}} {
				t.Run(variant.name, func(t *testing.T) {
					modified := bytes.Replace(leaf, canonical, []byte(variant.members), 1)
					if err := sav.SetNativeActions(&doc.State, modified); err != nil {
						t.Fatal(err)
					}
					a, readErr := readCurrentActions(&doc)
					written, err := sav.EncodeDocumentData(doc)
					if err != nil {
						t.Fatal(err)
					}
					cold := itemMutationFront(t, 1)
					opener, town, loadErr := cold.RestoreOriginal(written)
					if variant.bad {
						for route, err := range map[string]error{"reader": readErr, "production LOAD": loadErr} {
							if err == nil || !strings.Contains(err.Error(), "current actions: invalid WriterCarriers record") {
								t.Errorf("%s did not refuse the record with its named error: %v", route, err)
							}
						}
						if a != nil {
							t.Error("ambiguous reader returned a writer capability")
						}
						return
					}
					if readErr != nil || a == nil || loadErr != nil || town {
						t.Fatal("canonical control refused", readErr, loadErr, town)
					}
					if err := cold.App("canonical writer record").OpenMission(opener); err != nil {
						t.Fatal(err)
					}
					if cold.live.world.Hash() != hash {
						t.Fatal("canonical record changed the cold World")
					}
				})
			}
		})
	}
}

func TestCurrentActionWriterRecordsHeldCarriers(t *testing.T) {
	f := currentArchiveFixture(t)
	hash := f.live.world.Hash()
	want := "0"
	if _, held := f.live.world.SavedCellPlanes(); held {
		want = fmt.Sprint(currentWriterCarriers)
	}
	t.Logf("fixture World holds cell planes: %t", want != "0")
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "writer carrier record")
	if err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != hash {
		t.Fatal("writer carrier metadata changed the live World")
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("current writer omitted the continuation supplement", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(leaf, &fields); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fields["WriterCarriers"], []byte(want)) {
		t.Fatalf("current writer carrier record = %s, want explicit %s", fields["WriterCarriers"], want)
	}
	cold := openCurrentArchive(t, raw)
	if cold.live.world.Hash() != hash {
		t.Fatal("writer carrier metadata changed cold LOAD's World")
	}
	delete(fields, "WriterCarriers")
	historical, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, historical); err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || a.WriterCarriers != 0 {
		t.Fatal("historical absent writer record did not default to none", a, err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold = openCurrentArchive(t, raw)
	if cold.live.world.Hash() != hash {
		t.Fatal("historical absent writer record changed cold LOAD's World")
	}
}

func TestCurrentActionWriterCarrierRecordErrorIsBounded(t *testing.T) {
	raw := []byte(`{"Version":1,"WriterCarriers":` + strings.Repeat("9", 1<<20) + `}`)
	if _, err := decodeCurrentActions(raw); err == nil || !strings.Contains(err.Error(), "current actions: invalid WriterCarriers record") || len(err.Error()) > 160 {
		t.Fatal("oversized numeric token did not produce a bounded writer record error")
	}
	if _, err := decodeCurrentActions([]byte(strings.Repeat("9", 1<<20))); err == nil || len(err.Error()) > 160 {
		t.Fatal("oversized numeric root did not produce a bounded type error")
	}
	for _, raw := range []string{
		`{"Version":1,"Options":{"Text":"WriterCarriers"},"WriterCarriers":0}`,
		`{"Version":1,"Options":{"WriterCarriers":16},"WriterCarriers":0}`,
	} {
		if err := checkCurrentWriterCarriers([]byte(raw)); err != nil {
			t.Fatal("nested content was counted as a root writer record", err)
		}
	}
}
