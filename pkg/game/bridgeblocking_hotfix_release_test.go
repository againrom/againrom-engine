package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func beforeStructureBlockingForm(t *testing.T, form []byte) []byte {
	t.Helper()
	form = beforeNativeTrainingForm(t, form)
	if len(form) > 0 && form[0] == 101 {
		end := len(form)
		if end < 56 || string(form[end-4:]) != "TAC1" || form[end-5] >= 101 {
			t.Fatal("invalid tactical compatibility suffix")
		}
		span := uint64(binary.LittleEndian.Uint32(form[end-9:]))
		if span < 9 || span > uint64(end-47) {
			t.Fatal("invalid tactical compatibility span", span)
		}
		start := end - 9 - int(span)
		payload := form[start : end-9]
		count := uint64(binary.LittleEndian.Uint32(payload[1:]))
		if payload[0] > 1 || count > 65535 || payload[0] == 0 && count != 0 || 9+4*count != span ||
			binary.LittleEndian.Uint32(payload[5+4*int(count):]) != 0 {
			t.Fatal("tactical compatibility fixture has invalid traversal or retained Retreat")
		}
		base := form[end-5]
		form = bytes.Clone(form[:start])
		form[0] = base
	}
	if len(form) == 0 || form[0] != 100 {
		return form
	}
	end := len(form)
	if end < 22 || string(form[end-4:]) != "SBK1" {
		t.Fatal("invalid blocking compatibility suffix")
	}
	span := int(binary.LittleEndian.Uint32(form[end-9:]))
	if span < 12 || span > end-10 {
		t.Fatal("invalid blocking compatibility span", span)
	}
	out := bytes.Clone(form[:end-9-span])
	out[0] = form[end-5]
	return out
}

func requireBridgeSAVMask(t *testing.T, raw []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if doc.World == nil {
		t.Fatal("mission SAV has no world")
	}
	seen := false
	for _, index := range doc.World.Buildings {
		r := &doc.Objects[index-1]
		var pos []byte
		for _, v := range r.Raw {
			if v.Name == "Block12" {
				pos = v.Bytes
			}
		}
		if len(pos) < 2 || pos[0] != 57 || pos[1] != 75 {
			continue
		}
		seen = true
		blocking, err := savedStructureValue(r, "B64")
		if err != nil || blocking != 0x00fc003f {
			t.Fatalf("bridge B64=%#x err=%v, want 0x00fc003f", blocking, err)
		}
		attach, err := savedStructureValue(r, "B68")
		if err != nil || attach != 0x00ffffff {
			t.Fatalf("bridge B68=%#x err=%v", attach, err)
		}
	}
	if !seen {
		t.Fatal("bridge Building absent from generated SAV")
	}
	deck := 0
	for _, b := range doc.World.Blocks {
		if b.Row() >= 76 && b.Row() <= 77 && b.Col() >= 57 && b.Col() <= 62 {
			deck++
			if b.Static&1 != 0 || b.Dyn&1 != 0 {
				t.Fatalf("saved bridge deck (%d,%d) closed with static/dynamic %#x/%#x", b.Col(), b.Row(), b.Static, b.Dyn)
			}
		}
	}
	if deck != 12 {
		t.Fatalf("saved bridge deck block records=%d, want 12", deck)
	}
}

func requireBridgeWorldMask(t *testing.T, w *sim.World, width int) {
	t.Helper()
	found := false
	for _, st := range w.Structures() {
		if st.Col == 57 && st.Row == 75 && st.Width == 6 && st.Height == 4 {
			found = true
			if st.Blocking != 0x00fc003f || st.Attach != 0x00ffffff {
				t.Fatalf("live bridge masks: blocking=%#x attach=%#x", st.Blocking, st.Attach)
			}
		}
	}
	if !found {
		t.Fatal("mission 130 bridge footprint absent")
	}
	grid := w.CurrentPolicy().Terrain.Block
	for y := 76; y <= 77; y++ {
		for x := 57; x <= 62; x++ {
			cell := grid[y*width+x]
			if cell&1 != 0 {
				t.Fatalf("live bridge deck (%d,%d) closed with %#x", x, y, cell)
			}
		}
	}
}

func TestReleaseMissionBridgeBlockingMaskSurvivesSAV(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("bridge blocking SAV")
	if err := app.OpenMission(f.MissionOpenerWith(130, MissionParty(nil, data.BodyList{}, nil))); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	var bridge sim.Structure
	found := false
	for _, st := range w.Structures() {
		if st.Col == 57 && st.Row == 75 && st.Width == 6 && st.Height == 4 {
			bridge, found = st, true
			break
		}
	}
	if !found {
		t.Fatal("mission 130 bridge footprint absent")
	}
	if bridge.Attach != 0x00ffffff {
		t.Fatalf("bridge attach mask %#x", bridge.Attach)
	}
	params := f.Table.Buildings.EntryParams(int(bridge.Kind))
	if len(params) <= 5 || uint32(params[4]) != 0x00fc003f || uint32(params[5]) != 0x00ffffff {
		t.Fatalf("installed bridge class %d masks: %v", bridge.Kind, params)
	}
	requireBridgeWorldMask(t, w, f.live.mission.state.Map.Width)
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	requireBridgeSAVMask(t, raw)
	if kitPath := os.Getenv("AGAINROM_BRIDGE_SAV_OUTPUT"); kitPath != "" {
		if err := os.WriteFile(kitPath, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "bridge.sav")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold := loadAreaContinuation(t, path)
	requireBridgeWorldMask(t, cold.live.world, cold.live.mission.state.Map.Width)
	for range 32 {
		cold.live.tick()
	}
	requireBridgeWorldMask(t, cold.live.world, cold.live.mission.state.Map.Width)
	s, label, err = cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = cold.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	requireBridgeSAVMask(t, raw)
}
