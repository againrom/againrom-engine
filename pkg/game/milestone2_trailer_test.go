package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// SAV-790 transfers all 400 bytes even when the preceding marker differs.
// SAV-DECPAD-238 permits one arbitrary alignment byte after an odd end.
// Only the structure location comes from sav.Open; expected values never
// pass through File.Trailer or DecodeDocumentData.
func trailerAcceptanceRaw(body []byte, off int) ([100]uint32, error) {
	var out [100]uint32
	if off < 0 || off > len(body)-400 {
		return out, fmt.Errorf("trailer span at %d exceeds %d bytes", off, len(body))
	}
	end := off + 400
	if len(body) != end+(end&1) {
		return out, fmt.Errorf("trailer ends at %d, decoded size %d", end, len(body))
	}
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(body[off+4*i:])
	}
	return out, nil
}

// This compares retained Document state. The sim.World has no trailer
// consumer; its hash alone cannot prove that these dwords survived SAVE.
func trailerAcceptanceDifferences(want [100]uint32, state *SnapshotSAVDocument) []string {
	if state == nil {
		return []string{"trailer: complete saved Document unavailable"}
	}
	if state.Document == nil || state.Unavailable != "" {
		return []string{fmt.Sprintf("trailer: complete saved Document unavailable: %s", state.Unavailable)}
	}
	var diff []string
	for i, v := range want {
		if got := state.Document.Trailer[i]; got != v {
			diff = append(diff, fmt.Sprintf("trailer dword %d: retained=%#08x file=%#08x", i, got, v))
		}
	}
	return diff
}

func trailerAcceptancePattern() (out [100]uint32) {
	for i := range out {
		out[i] = 0xa1000203 + uint32(i)*0x010307
	}
	return out
}

func TestTrailerAcceptanceControls(t *testing.T) {
	want := trailerAcceptancePattern()
	for _, off := range []int{6, 7} {
		body := make([]byte, off)
		for _, v := range want {
			body = binary.LittleEndian.AppendUint32(body, v)
		}
		if len(body)&1 != 0 {
			body = append(body, 0xa7)
		}
		got, err := trailerAcceptanceRaw(body, off)
		if err != nil || got != want {
			t.Fatal("raw nonzero dwords or arbitrary padding changed", off, err)
		}
		for _, bad := range [][]byte{body[:off+399], append(append([]byte(nil), body...), 0, 0)} {
			if _, err := trailerAcceptanceRaw(bad, off); err == nil {
				t.Fatal("accepted truncated trailer or extra extent")
			}
		}
		for _, badOff := range []int{-1, off - 2, off + 2, len(body) + 1} {
			if _, err := trailerAcceptanceRaw(body, badOff); err == nil {
				t.Fatal("accepted wrong trailer locator", badOff)
			}
		}
	}
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{Trailer: want}}
	if diff := trailerAcceptanceDifferences(want, state); len(diff) != 0 {
		t.Fatal(diff)
	}
	for i := range want {
		state.Document.Trailer = want
		state.Document.Trailer[i] = 0
		diff := trailerAcceptanceDifferences(want, state)
		if len(diff) != 1 || !strings.Contains(diff[0], fmt.Sprintf("dword %d:", i)) {
			t.Fatalf("omission of dword %d escaped: %v", i, diff)
		}
	}
	state.Document.Trailer = want
	state.Document.Trailer[0], state.Document.Trailer[1] = want[1], want[0]
	if diff := trailerAcceptanceDifferences(want, state); len(diff) != 2 {
		t.Fatal("swapped named dwords escaped", diff)
	}
	for _, missing := range []*SnapshotSAVDocument{nil, {}, {Document: state.Document, Unavailable: "missing"}} {
		if len(trailerAcceptanceDifferences(want, missing)) == 0 {
			t.Fatal("missing retained Document escaped")
		}
	}
}

// Both UI LOAD doors, menu SAVE using the explicit AGS codec, a fresh FrontEnd LOAD and driver
// continuation. The factory permits asset-free synthetic and EN/RU runs.
func trailerAcceptanceApp(t *testing.T, raw []byte, want [100]uint32, front func() *FrontEnd) {
	t.Helper()
	f := front()
	f.SetDeterministicFrames(true)
	check := func(state *SnapshotSAVDocument) {
		t.Helper()
		if diff := trailerAcceptanceDifferences(want, state); len(diff) != 0 {
			t.Fatal(diff)
		}
	}
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatal(err)
	}
	check(ms.savedDocument)
	checkFront := func(fe *FrontEnd) {
		t.Helper()
		snapshot, _, err := fe.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		check(snapshot.SavedDocument)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "trailer.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("trailer acceptance")
	app.Layout(1024, 768)
	save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	if app.Screen() != ui.ScreenMenu {
		t.Fatal("original title LOAD witness did not start at title")
	}
	groundAppLoad(t, app, list, "trailer.sav")
	checkFront(f)
	first := f.live
	groundAppLoad(t, app, list, "trailer.sav") // ScreenMap opens the mission menu.
	if f.live == first {
		t.Fatal("mission-menu original LOAD did not replace the driver")
	}
	checkFront(f)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := listAGS(store)
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".ags" {
		t.Fatalf("menu SAVE using the explicit AGS codec: %+v %v", entries, err)
	}
	// Remove only the copy this test created. Fresh LOAD must use the AGS
	// document and cannot recover omitted trailer values from the source.
	if err := os.Remove(filepath.Join(dir, "trailer.sav")); err != nil {
		t.Fatal(err)
	}
	fresh := front()
	fresh.SetDeterministicFrames(true)
	app2 := fresh.App("fresh trailer LOAD")
	s, l, ld := nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
	app2.SetSaveSeams(s, l, ld)
	groundAppLoad(t, app2, l, entries[0].Name)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("fresh LOAD changed World hash")
	}
	t.Log("fresh native LOAD has the same World hash; checking retained trailer separately")
	checkFront(fresh)
	initial := f.live.world.Tick()
	for i := range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatalf("native continuation differs at step %d", i)
		}
	}
	if f.live.world.Tick() == initial {
		t.Fatal("driver continuation did not advance")
	}
	checkFront(f)
	checkFront(fresh)
	t.Log("100 raw trailer dwords agree after title and mission-menu original LOAD, ordinary SAVE/fresh LOAD and 20 advancing continuation hashes")
}

func TestTrailerAcceptanceNonzeroAppContinuation(t *testing.T) {
	want := trailerAcceptancePattern()
	parities := map[int]bool{}
	for _, name := range []string{"A", "AB"} {
		for _, marker := range []uint32{0xbadface1, 0x12345678} {
			t.Run(fmt.Sprintf("name=%s/marker=%x", name, marker), func(t *testing.T) {
				a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 40, maxHP: 60, human: true, name: name}
				body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil)
				// The synthetic writer's literal marker, not the decoder's locator,
				// defines the new suffix and its expected offset.
				at := bytes.LastIndex(body, []byte{0xe1, 0xac, 0xdf, 0xba})
				if at < 0 {
					t.Fatal("synthetic writer lost its marker")
				}
				body = binary.LittleEndian.AppendUint32(body[:at], marker)
				if marker == 0xbadface1 {
					body = binary.LittleEndian.AppendUint32(body, 0x87654321)
				}
				off := len(body)
				parities[off&1] = true
				for _, v := range want {
					body = binary.LittleEndian.AppendUint32(body, v)
				}
				if len(body)&1 != 0 {
					body = append(body, 0xa7)
				}
				front := func() *FrontEnd {
					f := poolFixtureFront(t, 91)
					f.Campaign = resolved(saveCampaign(), nil)
					return f
				}
				raw := completeDocumentTail1115(t, front(), savedContainer(body))
				file, err := sav.Open(raw)
				if err != nil || file.TrailerOff != off {
					t.Fatalf("synthetic trailer locator: %v (want %d)", err, off)
				}
				got, err := trailerAcceptanceRaw(file.Body, file.TrailerOff)
				if err != nil || got != want {
					t.Fatal("synthetic trailer changed before import", err)
				}
				trailerAcceptanceApp(t, raw, want, front)
			})
		}
	}
	if len(parities) != 2 {
		t.Fatal("synthetic App fixtures did not cover both alignment parities")
	}
}
