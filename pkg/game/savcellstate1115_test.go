package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func beforeTurnStateForm(t *testing.T, raw []byte) []byte {
	t.Helper()
	if len(raw) == 0 || raw[0] != 115 {
		return raw
	}
	end := len(raw)
	if end < 56 || string(raw[end-4:]) != "TRN1" || raw[end-5] >= 115 {
		t.Fatal("invalid turn-state fixture footer")
	}
	span := uint64(binary.LittleEndian.Uint32(raw[end-9:]))
	if span < 13 || span > uint64(end-43) {
		t.Fatal("invalid turn-state fixture span")
	}
	start := end - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(raw[start:]))
	if count == 0 || span != 4+9*count {
		t.Fatal("invalid turn-state fixture population")
	}
	var prior uint32
	for n := uint64(0); n < count; n++ {
		at := start + 4 + 9*int(n)
		id := binary.LittleEndian.Uint32(raw[at:])
		if n > 0 && id <= prior || raw[at+4] > 3 || raw[at+7] > 15 || raw[at+8] > 128 {
			t.Fatal("invalid turn-state fixture record")
		}
		prior = id
	}
	out := bytes.Clone(raw[:start])
	out[0] = raw[end-5]
	return out
}

// Both archives contain only synthetic fixture data. Read the existing
// independent actor/map grammar before reopening its VFS with explicit terrain
// parameters; absence of map.reg in older fixtures remains meaningful.
func cellStateFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := newGroupFront(t, -1)
	mapBytes, err := f.Archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	npcBytes, err := f.Archives.Containers.ReadFile("scenario/npc.reg")
	if err != nil {
		t.Fatal(err)
	}
	parameters := synth.Reg(0x11, []synth.RegNode{{Name: "Terrain", Kind: 1, Children: []synth.RegNode{
		{Name: "CostLand", Kind: 2, Int: 8},
	}}})
	dir := t.TempDir()
	worldPath, scenarioPath := filepath.Join(dir, WorldArchive), filepath.Join(dir, ScenarioArchive)
	for _, archive := range []struct {
		path string
		data []byte
	}{
		{worldPath, synth.Archive([]synth.File{{Path: "data/map.reg", Data: parameters}})},
		{scenarioPath, synth.Archive([]synth.File{{Path: "10.alm", Data: mapBytes}, {Path: "npc.reg", Data: npcBytes}})},
	} {
		if err := os.WriteFile(archive.path, archive.data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fsys, err := vfs.Open([]string{worldPath, scenarioPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fsys
	return f
}

func cellStateBlocks1115(bit4 bool) []sav.BlockRecord {
	oldStatic, oldDynamic := byte(0), byte(0x40)
	if bit4 {
		oldStatic, oldDynamic = 0x10, 0x50
	}
	return []sav.BlockRecord{
		// TERR-PASS-053 scans 0807..eded inclusive and emits only Dyn>15.
		// Source rows outside that population must not survive by append.
		{Cell: 0x0806, Dyn: 16},
		{Cell: 0x0807, Dyn: 16},
		{Cell: 0x0909, Dyn: 15},
		{Cell: 0x090a, Dyn: 16},
		{Cell: 0x100f, Static: oldStatic, Dyn: oldDynamic},
		{Cell: 0x1010, Dyn: 0x40},
		{Cell: 0x120f, Dyn: 0x40},
		{Cell: 0x140f, Dyn: 0x40},
		{Cell: 0xeded, Dyn: 16},
		{Cell: 0xedee, Dyn: 16},
	}
}

// The complete archive/actor grammar comes from the independent crossing
// fixture. The future cell is missing, the old cell has no retainer, and its
// saved baseline17 deliberately differs from the ALM current cost8.
func cellStateLiteral(t *testing.T, f *FrontEnd, bit4 bool) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(crossingCellLiteral(t, f, false))
	if err != nil {
		t.Fatal(err)
	}
	var cells []sav.DocumentCellData
	for _, cell := range doc.World.Cells {
		if cell.Cell == 0x1010 {
			continue
		}
		if cell.Cell == 0x100f && cell.GroundActor == 1004 {
			cell.Cost = 17
		}
		cells = append(cells, cell)
	}
	doc.World.Cells, doc.World.Blocks = cells, cellStateBlocks1115(bit4)
	oldCount := 0
	for _, cell := range cells {
		if cell.Cell == 0x100f {
			oldCount++
		}
	}
	if oldCount != 2 || len(cells) != 4 {
		t.Fatal("literal cell lifecycle must contain two old source overlays and no future cell")
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func cellStateOriginalDoor1115(t *testing.T, fromMap, bit4 bool) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	f := cellStateFront(t)
	app := f.App("synthetic cell lifecycle")
	if fromMap {
		if err := app.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
	}
	inputs, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
	path := filepath.Join(inputs, "game1115.sav")
	raw := cellStateLiteral(t, f, bit4)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: inputs}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1115.sav")
	clear(raw)
	// The source is only this test's synthetic file. Cold LOAD uses its new SAV.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	return f, app, store
}

// Independent literal plane expectation for the synthetic40x40 zero ALM.
// The square map has an eight-cell border; the raw planes retain stride256.
// No current importer, cell constructor or projection helper builds this value.
func cellStateWantPlanes1115(bit4 bool, tick int) *sim.SavedCellPlanes {
	p := &sim.SavedCellPlanes{Costs: [11]byte{255, 8, 8, 9, 14, 6, 12, 11, 16, 8, 6}}
	for i := range p.Cost {
		p.Cost[i], p.CostKnown[i] = 1, 1
	}
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			at := y*256 + x
			p.Cost[at] = 8
			if x < 8 || x >= 32 || y < 8 || y >= 32 {
				p.Static[at], p.Dynamic[at] = 0x1f, 0x1f
			}
		}
	}
	for _, block := range cellStateBlocks1115(bit4) {
		p.Static[block.Cell], p.Dynamic[block.Cell] = block.Static, block.Dyn
	}
	if tick > 0 {
		// Empty-node deletion restores its saved baseline, not ALM8. Static
		// bit4 comes from the current plane; the saved payload's Static is0.
		old := byte(0)
		if bit4 {
			old = 0x10
		}
		p.Cost[0x100f], p.Static[0x100f], p.Dynamic[0x100f] = 17, old, old
		if tick < 5 {
			p.Dynamic[0x100f] |= 0x40
		}
		p.Static[0x1010], p.Dynamic[0x1010] = 0x20, 0x60
	}
	return p
}

func cellStateAssertPlanes1115(t *testing.T, w *sim.World, bit4 bool, tick int) *sim.SavedCellPlanes {
	t.Helper()
	got, present := w.SavedCellPlanes()
	if !present || got == nil {
		t.Fatal("explicit synthetic registry did not establish cell-plane authority")
	}
	want := cellStateWantPlanes1115(bit4, tick)
	if got.Costs != want.Costs {
		t.Fatal("cell-lifecycle cost parameters changed", got.Costs, want.Costs)
	}
	for _, plane := range []struct {
		name      string
		got, want *[65536]byte
	}{
		{"cost", &got.Cost, &want.Cost}, {"static", &got.Static, &want.Static},
		{"dynamic", &got.Dynamic, &want.Dynamic}, {"height", &got.Height, &want.Height},
		{"known", &got.CostKnown, &want.CostKnown},
	} {
		for cell, value := range plane.want {
			if plane.got[cell] != value {
				t.Fatalf("tick%d %s plane[%04x]=%02x want%02x", tick, plane.name, cell, plane.got[cell], value)
			}
		}
	}
	return got
}

func cellStateCheck1115(t *testing.T, f *FrontEnd, bit4 bool, tick int) Snapshot {
	t.Helper()
	planes := cellStateAssertPlanes1115(t, f.live.world, bit4, tick)
	e, motion := crossingMotion1115(t, f)
	wantCell, wantFine := uint16(0x100f), byte(224)
	if tick > 0 {
		wantCell, wantFine = 0x1010, byte((tick-1)*32)
	}
	if tick >= 5 {
		wantFine = 128
	}
	if e.X != int32(wantCell&255) || e.Y != 16 || !motion.Current || motion.Active != (tick < 5) ||
		motion.Position.Cell != wantCell || motion.Position.PackedCell != wantCell ||
		motion.Position.FineX != wantFine || motion.Position.FineY != 128 ||
		strings.Contains(motion.Issue, "new cell construction") || strings.Contains(motion.Issue, "empty cell deletion") {
		t.Fatalf("tick%d cell lifecycle did not continue the literal fine crossing: %+v", tick, motion)
	}
	_, liveCells, _, present := f.live.world.SavedActorMotions()
	if !present || len(liveCells) != 3 {
		t.Fatal("current hashed cell population changed", len(liveCells), present)
	}
	var literal [52]byte
	literal[0] = 8
	binary.LittleEndian.PutUint32(literal[4:], 1004)
	if tick == 0 {
		literal[0], literal[3], literal[50], literal[51] = 17, 0x35, 0xfe, 0xca
	}
	found := false
	for _, cell := range liveCells {
		if tick > 0 && cell.Cell == 0x100f {
			t.Fatal("empty old cell still exists in hashed node state")
		}
		if cell.Cell == wantCell {
			found = true
			if cell.Payload != literal || cell.Ground.Key != 1004 || !cell.Ground.Bound || cell.Ground.Entity != e.ID {
				t.Fatalf("tick%d hashed52-byte cell or exact actor binding changed: %+v", tick, cell)
			}
		}
	}
	if !found {
		t.Fatal("current actor has no hashed cell node")
	}
	before := f.live.world.Hash()
	snapshot, _, err := f.Snapshot(true)
	if err != nil || before != f.live.world.Hash() {
		t.Fatal("cell-lifecycle Snapshot failed or mutated live state", err)
	}
	nativeForm := beforeTurnStateForm(t, snapshot.World)
	legacy := beforeStructureUseForm1150(t, nativeForm)
	clockEnd := len(legacy) - 8
	if binary.LittleEndian.Uint32(nativeForm[clockEnd:]) != 0 {
		t.Fatal("unexpected native Roam counter")
	}
	planeEnd := clockEnd - 4 - int(binary.LittleEndian.Uint32(nativeForm[clockEnd-4:]))
	planeEnd -= 4 + int(binary.LittleEndian.Uint32(nativeForm[planeEnd-4:]))
	planeEnd -= 4 + int(binary.LittleEndian.Uint32(nativeForm[planeEnd-4:]))
	if beforeAutoHealing1191(t, nativeForm)[0] != 95 || binary.LittleEndian.Uint32(legacy[len(legacy)-4:]) != 0 || planeEnd < 4 || binary.LittleEndian.Uint32(nativeForm[planeEnd-4:]) != 327691 ||
		snapshot.SavedDocument == nil || snapshot.SavedDocument.Document == nil || snapshot.SavedDocument.Document.World == nil {
		t.Fatal("cell lifecycle lacks complete native plane state before form84 objects and story 1139's own carried-resume span")
	}
	currentPlanes, present := f.live.world.SavedCellPlanes()
	if !present || *currentPlanes != *planes {
		t.Fatal("Snapshot mutated cell-plane authority")
	}
	doc := snapshot.SavedDocument.Document.World
	oldCount, newCount := 0, 0
	for _, cell := range doc.Cells {
		switch cell.Cell {
		case 0x100f:
			oldCount++
			if cell.Payload() != literal {
				t.Fatalf("old current cell lost retained baseline/residue before detach: %x", cell.Payload())
			}
		case 0x1010:
			newCount++
			var expected [52]byte
			expected[0] = 8
			binary.LittleEndian.PutUint32(expected[4:], 1004)
			if cell.Payload() != expected {
				t.Fatalf("new cell is not zero52 plus baseline cost/static and actor: %x", cell.Payload())
			}
		case 0x120f:
			if cell != (sav.DocumentCellData{Cell: 0x120f, Cost: 8, GroundActor: 1006}) {
				t.Fatal("unaffected second actor cell changed", cell)
			}
		case 0x140f:
			if cell != (sav.DocumentCellData{Cell: 0x140f, Cost: 8, GroundActor: 1008}) {
				t.Fatal("unaffected third actor cell changed", cell)
			}
		default:
			t.Fatal("cell lifecycle fabricated an unrelated payload", cell)
		}
	}
	// With plane authority, SAVE emits current nodes, not overwritten archive
	// history. The two source100f overlays become one current node at tick0;
	// after detach neither historical overlay may reappear.
	if tick == 0 && (oldCount != 1 || newCount != 0 || len(doc.Cells) != 3) ||
		tick > 0 && (oldCount != 0 || newCount != 1 || len(doc.Cells) != 3) {
		t.Fatalf("tick%d historical old/new cell counts=%d/%d total%d", tick, oldCount, newCount, len(doc.Cells))
	}
	// This independent literal scan must replace the source delta list. In
	// particular stale/duplicate/out-of-window/Dyn<=15 source rows cannot append.
	wantPlanes := cellStateWantPlanes1115(bit4, tick)
	var wantBlocks []sav.BlockRecord
	for cell := 0x0807; cell <= 0xeded; cell++ {
		if wantPlanes.Dynamic[cell] > 15 {
			wantBlocks = append(wantBlocks, sav.BlockRecord{Cell: uint16(cell), Dyn: wantPlanes.Dynamic[cell], Static: wantPlanes.Static[cell]})
		}
	}
	if !reflect.DeepEqual(doc.Blocks, wantBlocks) {
		t.Fatalf("tick%d raw block scan differs: got%d rows want%d", tick, len(doc.Blocks), len(wantBlocks))
	}
	return snapshot
}

func TestSavedCellState1115OriginalDoorsConstructorDeletionNativeContinuation(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		for _, bit4 := range []bool{false, true} {
			for saveAt := 0; saveAt <= 5; saveAt++ {
				t.Run(fmt.Sprintf("fromMap=%t/bit4=%t/saveAt=%d", fromMap, bit4, saveAt), func(t *testing.T) {
					f, app, store := cellStateOriginalDoor1115(t, fromMap, bit4)
					for tick := 0; tick < saveAt; tick++ {
						cellStateCheck1115(t, f, bit4, tick)
						f.live.tick()
					}
					before := cellStateCheck1115(t, f, bit4, saveAt)
					hash := f.live.world.Hash()
					name := crossingMenuSave(t, app, store)
					if f.live.world.Hash() != hash {
						t.Fatal("ordinary menu SAVE mutated live cell state")
					}
					raw, err := store.Read(name)
					if err != nil {
						t.Fatal(err)
					}
					stored, err := sav.DecodeDocumentData(raw)
					if err != nil || stored.World == nil || !reflect.DeepEqual(stored.World.Cells, before.SavedDocument.Document.World.Cells) || !reflect.DeepEqual(stored.World.Blocks, before.SavedDocument.Document.World.Blocks) {
						t.Fatal("ordinary menu SAVE lost complete current cell state", err)
					}
					fresh := cellStateFront(t)
					freshApp := fresh.App("fresh cell-lifecycle native LOAD")
					save, list, load := agsSaveSeams(fresh, store, OriginalStore{}, nil)
					freshApp.SetSaveSeams(save, list, load)
					groundAppLoad(t, freshApp, list, name)
					after := cellStateCheck1115(t, fresh, bit4, saveAt)
					if !bytes.Equal(before.World, after.World) || !reflect.DeepEqual(before.SavedDocument.Document.World.Cells, after.SavedDocument.Document.World.Cells) || !reflect.DeepEqual(before.SavedDocument.Document.World.Blocks, after.SavedDocument.Document.World.Blocks) {
						t.Fatal("fresh native LOAD reconstructed or lost cell lifecycle state")
					}
					secondStore := SaveStore{Dir: t.TempDir()}
					save, list, load = agsSaveSeams(fresh, secondStore, OriginalStore{}, nil)
					freshApp.SetSaveSeams(save, list, load)
					secondName := crossingMenuSave(t, freshApp, secondStore)
					second := cellStateFront(t)
					secondApp := second.App("second cell-lifecycle cold LOAD")
					save, list, load = agsSaveSeams(second, secondStore, OriginalStore{}, nil)
					secondApp.SetSaveSeams(save, list, load)
					groundAppLoad(t, secondApp, list, secondName)
					secondSnapshot := cellStateCheck1115(t, second, bit4, saveAt)
					if !bytes.Equal(before.World, secondSnapshot.World) || !reflect.DeepEqual(before.SavedDocument.Document.World.Cells, secondSnapshot.SavedDocument.Document.World.Cells) || !reflect.DeepEqual(before.SavedDocument.Document.World.Blocks, secondSnapshot.SavedDocument.Document.World.Blocks) {
						t.Fatal("second SAV/cold LOAD lost exact cell planes or payloads")
					}
					fresh = second
					for tick := saveAt + 1; tick <= 20; tick++ {
						f.live.tick()
						fresh.live.tick()
						if tick <= 5 {
							cellStateCheck1115(t, f, bit4, tick)
							cellStateCheck1115(t, fresh, bit4, tick)
						}
						if f.live.world.Hash() != fresh.live.world.Hash() {
							t.Fatal("native cell-lifecycle continuation diverged", tick)
						}
					}
					cellStateAssertPlanes1115(t, fresh.live.world, bit4, 20)
				})
			}
		}
	}
}

func TestSavedCellState1115MissingRegistryKeepsAuthorityAbsent(t *testing.T) {
	f, _, _ := crossingOriginalDoor1115(t, false)
	beforeCellStateAbsent1115(t, f.live.world)
	for tick := 0; tick <= 5; tick++ {
		// This older fixture deliberately has no world/data/map.reg. Neither
		// its Cell.Cost bytes nor native passability may stand in for that file.
		crossingCheck1115(t, f, tick, false)
		beforeCellStateAbsent1115(t, f.live.world)
		if tick < 5 {
			f.live.tick()
		}
	}
}

// The native World remains valid and byte-identical in every case. Only the
// detached document disagrees; the checksum-valid envelope must reach semantic
// validation and refuse before either direct Restore or menu LOAD adopts it.
func TestSavedCellState1115DocumentDisagreementRefusedBeforeAdoption(t *testing.T) {
	currentCell := func(t *testing.T, doc *sav.DocumentWorldData) *sav.DocumentCellData {
		t.Helper()
		for i := range doc.Cells {
			if doc.Cells[i].Cell == 0x1010 {
				return &doc.Cells[i]
			}
		}
		t.Fatal("corruption fixture lacks the constructed current cell")
		return nil
	}
	currentBlock := func(t *testing.T, doc *sav.DocumentWorldData) *sav.BlockRecord {
		t.Helper()
		for i := range doc.Blocks {
			if doc.Blocks[i].Cell == 0x1010 {
				return &doc.Blocks[i]
			}
		}
		t.Fatal("corruption fixture lacks the current occupancy block")
		return nil
	}
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, *sav.DocumentWorldData)
	}{
		{"cell baseline cost", func(t *testing.T, doc *sav.DocumentWorldData) {
			currentCell(t, doc).Cost = 17
		}},
		{"cell residue", func(t *testing.T, doc *sav.DocumentWorldData) {
			currentCell(t, doc).Residue32 = 0xcafe
		}},
		{"cell layer key", func(t *testing.T, doc *sav.DocumentWorldData) {
			currentCell(t, doc).Layers[5] = 0x7a777001
		}},
		{"deleted old cell resurrection", func(_ *testing.T, doc *sav.DocumentWorldData) {
			doc.Cells = append([]sav.DocumentCellData{{Cell: 0x100f, Cost: 17, Residue03: 0x35, Residue32: 0xcafe}}, doc.Cells...)
		}},
		{"duplicate current cell key", func(t *testing.T, doc *sav.DocumentWorldData) {
			duplicate := *currentCell(t, doc)
			doc.Cells = append([]sav.DocumentCellData{duplicate}, doc.Cells...)
		}},
		{"extra block", func(t *testing.T, doc *sav.DocumentWorldData) {
			old := doc.Blocks
			doc.Blocks = make([]sav.BlockRecord, 0, len(old)+1)
			inserted := false
			for _, block := range old {
				if block.Cell == 0x0909 {
					t.Fatal("extra block must target a key excluded by the real Dyn15 scan")
				}
				if !inserted && block.Cell > 0x0909 {
					doc.Blocks = append(doc.Blocks, sav.BlockRecord{Cell: 0x0909, Dyn: 16})
					inserted = true
				}
				doc.Blocks = append(doc.Blocks, block)
			}
			if !inserted {
				t.Fatal("corruption fixture has no block insertion point")
			}
		}},
		{"missing block", func(t *testing.T, doc *sav.DocumentWorldData) {
			for i, block := range doc.Blocks {
				if block.Cell == 0x1010 {
					doc.Blocks = append(doc.Blocks[:i], doc.Blocks[i+1:]...)
					return
				}
			}
			t.Fatal("corruption fixture has no block to remove")
		}},
		{"block dynamic value", func(t *testing.T, doc *sav.DocumentWorldData) {
			currentBlock(t, doc).Dyn ^= 1
		}},
		{"block static value", func(t *testing.T, doc *sav.DocumentWorldData) {
			currentBlock(t, doc).Static ^= 1
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _, _ := cellStateOriginalDoor1115(t, false, false)
			f.live.tick()
			before := cellStateCheck1115(t, f, false, 1)
			bad := before
			var err error
			bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(t, bad.SavedDocument.Document.World)
			envelope := uncheckedDocumentEnvelope1115(t, bad)
			label := []byte("Disagreeing cell lifecycle")
			binary.LittleEndian.PutUint16(envelope[9:11], uint16(len(label)))
			header := append(bytes.Clone(envelope[:11]), label...)
			envelope = append(header, envelope[11:]...)
			decoded, _, err := DecodeSave(envelope)
			if err != nil {
				t.Fatal("checksum-valid cell DTO must reach semantic World validation", err)
			}
			if !bytes.Equal(decoded.World, before.World) || reflect.DeepEqual(decoded.SavedDocument, before.SavedDocument) {
				t.Fatal("corruption fixture must change only detached document state")
			}
			oldLive, oldTown, oldShop, oldHash := f.live, f.Town, f.Shop, f.live.world.Hash()
			unchanged := func() {
				t.Helper()
				if f.live != oldLive || f.Town != oldTown || f.Shop != oldShop || f.live.world.Hash() != oldHash ||
					!reflect.DeepEqual(before, cellStateCheck1115(t, f, false, 1)) {
					t.Fatal("refused cell DTO changed live hash, session or Snapshot")
				}
			}
			if open, town, err := f.Restore(decoded); err == nil || open != nil || town {
				t.Fatal("Restore adopted disagreeing cell lifecycle", open != nil, town, err)
			}
			unchanged()
			store := SaveStore{Dir: t.TempDir()}
			if err := os.WriteFile(filepath.Join(store.Dir, "disagreeing-cell-lifecycle.ags"), envelope, 0600); err != nil {
				t.Fatal(err)
			}
			app := f.App("refused cell-lifecycle document")
			save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
			app.SetSaveSeams(save, list, load)
			rows := list()
			if len(rows) != 1 {
				t.Fatal("hostile cell DTO has no ordinary LOAD row", rows)
			}
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate(rows[0].Label); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" {
				t.Fatal("ordinary LOAD did not expose cell DTO refusal", app.Screen(), app.HeadlessMessage())
			}
			unchanged()
		})
	}
}
