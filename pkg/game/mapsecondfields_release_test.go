//go:build release

package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

func TestReleaseSecondMapEditsUnitStructureAndSackFields(t *testing.T) {
	x := installedInspector(t)
	sources, raws := installedSecondMaps(t, x)

	pick, bridgeMap, bridgeIdx := -1, -1, -1
	for i, raw := range raws {
		f := readSecondFacts(t, raw)
		if pick < 0 && len(f.units) > 0 && len(f.structs) > 0 && len(f.sackOwner) > 0 {
			m, err := alm.OpenROM2(raw)
			if err == nil && len(m.Groups) >= 2 {
				pick = i
			}
		}
		if bridgeMap < 0 {
			m, err := alm.OpenROM2(raw)
			if err == nil {
				for j, o := range m.Objects {
					if len(o.Ext) == alm.ObjectExtSize {
						bridgeMap, bridgeIdx = i, j
						break
					}
				}
			}
		}
		if pick >= 0 && bridgeMap >= 0 {
			break
		}
	}
	if pick < 0 {
		t.Fatal("no installed map holds a unit, a structure, a sack and two players")
	}

	d, err := x.Open(sources[pick])
	if err != nil {
		t.Fatal(sources[pick], err)
	}
	host := d.Edits.(*mapInspectionEdits)
	raw := raws[pick]

	forbidden := []string{"X/Y raw", "Flags +10", "Definition unresolved", "Raw 0C/0E/12", "Extension:"}
	for _, r := range d.Records {
		if r.Kind != ui.KindUnit && r.Kind != ui.KindStructure && r.Kind != ui.KindSack && r.Kind != ui.KindStock {
			continue
		}
		joined := strings.Join(r.Details, "\n")
		for _, bad := range forbidden {
			if strings.Contains(joined, bad) || strings.Contains(r.Warning, bad) {
				t.Fatalf("%s %s %d carries technical text %q:\n%s", sources[pick], r.Kind, r.Index, bad, joined)
			}
		}
	}

	uFields := host.Fields(ui.KindUnit, 0)
	for _, key := range []string{"owner", "definition", "hp", "hpmax", "flags", "group", "field08", "field0a", "field10", "field20"} {
		if _, ok := fieldOf(uFields, key); !ok {
			t.Fatalf("%s unit 0 lacks field %q", sources[pick], key)
		}
	}
	sFields := host.Fields(ui.KindStructure, 0)
	for _, key := range []string{"kind", "owner", "field0c"} {
		if _, ok := fieldOf(sFields, key); !ok {
			t.Fatalf("%s structure 0 lacks field %q", sources[pick], key)
		}
	}
	sackKind := ui.KindSack
	for _, r := range d.Records {
		if (r.Kind == ui.KindSack || r.Kind == ui.KindStock) && r.Index == 0 {
			sackKind = r.Kind
			break
		}
	}
	kFields := host.Fields(sackKind, 0)
	for _, key := range []string{"field10", "additem"} {
		if _, ok := fieldOf(kFields, key); !ok {
			t.Fatalf("%s sack 0 lacks field %q", sources[pick], key)
		}
	}

	// Edit one field of each record kind, verify save, reload and undo.
	curOwner, _ := fieldOf(uFields, "owner")
	newOwner := int64(1)
	if curOwner.Value == 1 {
		newOwner = 2
	}
	curHP, _ := fieldOf(uFields, "hp")
	newHP := int64(321)
	if curHP.Value == newHP {
		newHP = 322
	}
	cur0C, _ := fieldOf(sFields, "field0c")
	new0C := int64(77)
	if cur0C.Value == new0C {
		new0C = 78
	}
	cur10, _ := fieldOf(kFields, "field10")
	new10 := int64(555)
	if cur10.Value == new10 {
		new10 = 556
	}

	if _, err := host.SetField(ui.KindUnit, 0, "owner", newOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindUnit, 0, "hp", newHP); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindStructure, 0, "field0c", new0C); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(sackKind, 0, "field10", new10); err != nil {
		t.Fatal(err)
	}
	target, err := host.SaveAs(filepath.Join(t.TempDir(), "fields.alm"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	got := readSecondFacts(t, saved)
	checkSecondFacts(t, "saved", got)
	le := binary.LittleEndian
	if le.Uint32(got.units[0][alm.SecondUnitOwner:]) != uint32(newOwner) || int16(le.Uint16(got.units[0][alm.SecondUnitHP:])) != int16(newHP) {
		t.Fatalf("saved unit 0 owner/hp mismatch")
	}
	reloaded, err := x.Open(target)
	if err != nil || !bytes.Equal(reloaded.SourceBytes, saved) {
		t.Fatalf("reload: %v", err)
	}
	rHost := reloaded.Edits.(*mapInspectionEdits)
	if f, _ := fieldOf(rHost.Fields(ui.KindStructure, 0), "field0c"); f.Value != new0C {
		t.Fatalf("reloaded structure field0c = %d, want %d", f.Value, new0C)
	}
	if f, _ := fieldOf(rHost.Fields(sackKind, 0), "field10"); f.Value != new10 {
		t.Fatalf("reloaded sack field10 = %d, want %d", f.Value, new10)
	}
	for range 4 {
		if _, err := host.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(host.model.Bytes(), raw) {
		t.Fatal("undo did not restore original bytes")
	}

	if bridgeMap >= 0 {
		bd, err := x.Open(sources[bridgeMap])
		if err != nil {
			t.Fatal(err)
		}
		bHost := bd.Edits.(*mapInspectionEdits)
		bf := bHost.Fields(ui.KindStructure, bridgeIdx)
		if _, ok := fieldOf(bf, "field14"); !ok {
			t.Fatalf("%s bridge %d lacks field14", sources[bridgeMap], bridgeIdx)
		}
		if _, ok := fieldOf(bf, "field18"); !ok {
			t.Fatalf("%s bridge %d lacks field18", sources[bridgeMap], bridgeIdx)
		}
	}
	t.Logf("PASS: %s edited unit, structure and sack fields, saved, reloaded and undid (%s); bridge checked=%v",
		sources[pick], installLanguage(x), bridgeMap >= 0)
}

func TestReleaseFirstMapUnitFieldsUnchanged(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	d, err := x.Open("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)
	fields := host.Fields(ui.KindUnit, 0)
	var keys []string
	for _, f := range fields {
		keys = append(keys, f.Key)
	}
	for _, want := range []string{"identity", "owner", "group", "hp", "hpmax", "mana", "body", "facing", "name", "speaker", "keys"} {
		if !slices.Contains(keys, want) {
			t.Fatalf("first-game unit field %q missing from %v", want, keys)
		}
	}
	t.Logf("PASS: first-game unit field list unchanged (%d fields, %s)", len(fields), installLanguage(x))
}
