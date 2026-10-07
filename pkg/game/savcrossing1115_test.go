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

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const crossingTerrain1115 = uint32(0x7a1e0c01)
const crossingSack1115 = uint32(0x7bac0031)

func crossingRawField(t *testing.T, record *sav.DocumentRecordData, name string) []byte {
	t.Helper()
	for i := range record.Raw {
		if record.Raw[i].Name == name {
			return record.Raw[i].Bytes
		}
	}
	t.Fatalf("literal crossing record lacks %s", name)
	return nil
}

func crossingWordList(t *testing.T, record *sav.DocumentRecordData, name string, words []uint16) {
	t.Helper()
	foundRaw, foundCount := false, false
	for i := range record.Raw {
		if record.Raw[i].Name == name {
			raw := make([]byte, 2*len(words))
			for j, word := range words {
				binary.LittleEndian.PutUint16(raw[2*j:], word)
			}
			record.Raw[i].Bytes, foundRaw = raw, true
		}
	}
	for i := range record.Counts {
		if record.Counts[i].Name == name {
			record.Counts[i].Count, foundCount = uint32(len(words)), true
		}
	}
	if !foundRaw || !foundCount {
		t.Fatal("literal crossing route fields missing", name)
	}
}

func crossingCells1115() []sav.DocumentCellData {
	return []sav.DocumentCellData{
		// Archive history is deliberately different from the final payload.
		{Cell: 0x100f, Cost: 6, GroundActor: newGroupB1115, Residue03: 0x11, Residue32: 0x1234},
		{Cell: 0x100f, Cost: 8, GroundActor: newGroupA, Sack: crossingSack1115, Residue03: 0x35, Residue32: 0xcafe},
		{Cell: 0x1010, Cost: 8, Residue03: 0x57, Residue32: 0xbeef},
		{Cell: 0x120f, Cost: 8, GroundActor: newGroupB1115},
		{Cell: 0x140f, Cost: 8, GroundActor: newGroupC1115},
	}
}

func crossingBlocks1115() []sav.BlockRecord {
	return []sav.BlockRecord{{Cell: 0x100f, Dyn: 0x40}, {Cell: 0x1010, Dyn: 0x40},
		{Cell: 0x120f, Dyn: 0x40}, {Cell: 0x140f, Dyn: 0x40}}
}

// Existing independent builders supply only the complete container and actor
// grammar. Every spatial input below is a literal, not the current projection.
// Both boundary cells already exist: this does not invent a new cell payload.
// An exact Sack object retains the old record after the actor leaves; residue
// bytes alone would not retain it (SAV-CELLLEAVE-584).
func crossingLiteral1115(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	return crossingCellLiteral(t, f, true)
}

func crossingCellLiteral(t *testing.T, f *FrontEnd, keepSack bool) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(newGroupLiteral(t, f, false))
	if err != nil {
		t.Fatal(err)
	}
	doc.World.TerrainIdentity = crossingTerrain1115
	doc.World.Cells, doc.World.Blocks = crossingCells1115(), crossingBlocks1115()
	if keepSack {
		sacks, err := sav.DecodeDocumentData(completeDocumentTail1115(t, f, groundSave(t, []sav.GroundSack{{
			Identity: crossingSack1115, Cell: 0x100f, FineX: 128, FineY: 128, Gold: 23,
		}})))
		if err != nil || sacks.World == nil || len(sacks.World.Sacks) != 1 {
			t.Fatal("literal retaining Sack grammar", err)
		}
		sack := sacks.Objects[sacks.World.Sacks[0]-1]
		if sack.Class != "Sack" || actorProjectionValue(t, sack, "Identity") != crossingSack1115 {
			t.Fatal("retainer is not the exact literal Sack")
		}
		for _, refs := range sack.RefSlots {
			if len(refs.Objects) != 0 {
				t.Fatal("literal Sack unexpectedly needs archive reference rebinding", refs)
			}
		}
		binary.LittleEndian.PutUint32(crossingRawField(t, &sack, "Block12")[8:], crossingTerrain1115)
		doc.Objects = append(doc.Objects, sack)
		doc.World.Sacks = []uint16{uint16(len(doc.Objects))}
	} else {
		doc.World.Cells[1].Sack = 0
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Human" {
			continue
		}
		p := crossingRawField(t, r, "Block12")
		binary.LittleEndian.PutUint32(p[8:], crossingTerrain1115)
		if actorProjectionValue(t, *r, "Identity") != newGroupA {
			continue
		}
		copy(p, []byte{15, 16, 15, 16, 224, 128, 0x7c, 0xa1, 1, 12, 30, 122})
		m := crossingRawField(t, r, "U154")
		clear(m)
		m[0], m[1], m[10], m[0xb0], m[0xb1] = 64, 64, 100, 32, 0
		m[0xb2], m[0xb3] = 0xd1, 0xe2 // explicit retained source suffix
		copy(m[0x82:0x8a], []byte{0x91, 0x92, 0x93, 0x94, 0xa1, 0xa2, 0xa3, 0xa4})
		for _, field := range []struct {
			offset int
			value  uint16
		}{
			{0x06, 0x1010}, {0x70, 0x100f}, {0x74, 0x1010}, {0x76, 0x1010},
			{0x80, 0x1010}, {0xa6, 0x1010}, {0xa8, 32}, {0xaa, 8}, {0xac, 3}, {0xae, 2},
		} {
			binary.LittleEndian.PutUint16(m[field.offset:], field.value)
		}
		o := crossingRawField(t, r, "U158")
		clear(o)
		o[0], o[1], o[8], o[9], o[10], o[11] = 15, 16, 1, 3, 16, 16
		binary.LittleEndian.PutUint32(crossingRawField(t, r, "U50"), 0xb)
		binary.LittleEndian.PutUint32(crossingRawField(t, r, "U54"), 0)
		crossingWordList(t, r, "U15C", []uint16{0x1010})
		crossingWordList(t, r, "U178", []uint16{0x1010})
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func crossingOriginalDoor1115(t *testing.T, fromMap bool) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	return crossingCellOriginalDoor1115(t, fromMap, true)
}

func crossingCellOriginalDoor1115(t *testing.T, fromMap, keepSack bool) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	f := newGroupFront(t, -1)
	app := f.App("literal active original crossing")
	if fromMap {
		if err := app.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
	}
	inputs, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
	path := filepath.Join(inputs, "game1115.sav")
	raw := crossingCellLiteral(t, f, keepSack)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := agsSaveSeams(f, store, OriginalStore{Dir: inputs}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game1115.sav")
	clear(raw)
	// This is exclusively the synthetic fixture in t.TempDir.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	return f, app, store
}

func crossingMotion1115(t *testing.T, f *FrontEnd) (sim.Entity, sim.SavedActorMotion) {
	t.Helper()
	e := newGroupActors1115(t, f.live.world)[newGroupA]
	motions, _, _, present := f.live.world.SavedActorMotions()
	if !present {
		t.Fatal("original LOAD lost motion authority")
	}
	for _, motion := range motions {
		if motion.Entity == e.ID {
			return e, motion
		}
	}
	t.Fatal("exact actor has no motion record")
	return sim.Entity{}, sim.SavedActorMotion{}
}

// MOVE-STEP-010 adds signed32 to the combined 16-bit x once per sub-tick
// (MOVE-CLOCK-032). Starting15:224, elapsed3/8 gives16:0,32,64,96,128.
// MOVE-STEP-040 clears progress3 only at center; CLAIM-007/REFRESH-012 clear
// the named timing/reservation words and dynamic list on that arrival.
// The same progress3 arm writes actor+54=1, even when the incoming literal is0;
// SAV-CROSSNEXT-585 then clears it back to0 on centered completion.
// formats/move's coordination rule records the old boundary cell in +a6 and
// retains its occupancy bit until arrival, independently of its emptied slot.
// The source's +70/+a6 are literal inputs, not inferred constructor values.
// This checks the retained-record crossing subset, not the unexpanded +72
// speed callback or complete original post-load scheduling.
func crossingCheck1115(t *testing.T, f *FrontEnd, tick int, replaced bool) Snapshot {
	t.Helper()
	wantCell, wantFine, wantElapsed := uint16(0x100f), byte(224), uint16(3)
	if tick > 0 {
		wantCell, wantFine, wantElapsed = 0x1010, byte((tick-1)*32), uint16(3+tick)
	}
	if tick == 5 {
		wantFine, wantElapsed = 128, 0
	}
	e, motion := crossingMotion1115(t, f)
	fx, fy, present := f.live.world.ActorFinePosition(e.ID)
	if !present || fx != wantFine || fy != 128 || e.X != int32(wantCell&255) || e.Y != 16 ||
		!motion.Current || motion.Active != (tick < 5) || motion.Position.Cell != wantCell || motion.Position.PackedCell != wantCell ||
		motion.Position.FineX != wantFine || motion.Position.FineY != 128 || motion.Position.Residue != 0xa17c || motion.Position.TerrainKey != crossingTerrain1115 {
		t.Fatalf("tick%d literal fine continuation: entity=%+v motion=%+v fine=%d,%d present=%t", tick, e, motion, fx, fy, present)
	}
	if e.Stride.Present || e.Transit != 0 || e.TransitTotal != 0 {
		t.Fatal("original fine crossing fabricated a native coarse stride", e)
	}
	if tick > 0 && !replaced && !strings.Contains(motion.Issue, "speed callback") {
		t.Fatal("crossing hides the unexpanded boundary-speed callback", motion.Issue)
	}
	drawn := false
	for _, draw := range f.live.entityDraws() {
		if draw.ID != uint32(e.ID) {
			continue
		}
		drawn = true
		if !draw.FinePosition || draw.FineX != wantFine || draw.FineY != 128 || draw.Cell.X != int(wantCell&255) || draw.Cell.Y != 16 || draw.Transit != 0 || draw.TransitSpan != 0 {
			t.Fatalf("tick%d first/current draw does not carry literal fine position: %+v", tick, draw)
		}
	}
	if !drawn {
		t.Fatal("loaded crossing actor absent from draw seam")
	}
	hash := f.live.world.Hash()
	snapshot := groupDocumentSnapshot1115(t, f)
	if hash != f.live.world.Hash() {
		t.Fatal("SAVE mutated live spatial state")
	}
	var record *sav.DocumentRecordData
	for _, b := range snapshot.SavedDocument.Actors {
		if b.EntityID == e.ID {
			record = &snapshot.SavedDocument.Document.Objects[b.ObjectIndex-1]
		}
	}
	if record == nil {
		t.Fatal("saved document lost exact actor binding")
	}
	wantAction := uint32(0)
	if tick > 0 && tick < 5 {
		wantAction = 1
	}
	if got := binary.LittleEndian.Uint32(crossingRawField(t, record, "U54")); got != wantAction || motion.ActorAction != wantAction {
		t.Fatalf("tick%d actor+54 document=%d native=%d want%d", tick, got, motion.ActorAction, wantAction)
	}
	wantPosition := []byte{byte(wantCell), byte(wantCell >> 8), byte(wantCell), byte(wantCell >> 8), wantFine, 128, 0x7c, 0xa1, 1, 12, 30, 122}
	if got := crossingRawField(t, record, "Block12"); !bytes.Equal(got, wantPosition) {
		t.Fatalf("tick%d document Position=%x want%x", tick, got, wantPosition)
	}
	m := crossingRawField(t, record, "U154")
	wantRate, wantTotal, wantClaim, wantOccupied := uint16(32), uint16(8), uint16(0x1010), uint16(0x1010)
	if tick > 0 {
		wantOccupied = 0x100f
	}
	if tick == 5 {
		wantRate, wantTotal, wantClaim, wantOccupied = 0, 0, 0, 0
	}
	for _, field := range []struct {
		offset int
		value  uint16
	}{
		{0xa8, wantRate}, {0xaa, wantTotal}, {0xac, wantElapsed}, {0x80, wantClaim}, {0xa6, wantOccupied},
		{0xae, 2}, {0x70, 0x100f},
	} {
		if got := binary.LittleEndian.Uint16(m[field.offset:]); got != field.value {
			t.Fatalf("tick%d document mover+%02x=%04x want%04x", tick, field.offset, got, field.value)
		}
	}
	if m[0xb0] != 32 || m[0xb1] != 0 || m[0xb2] != 0xd1 || m[0xb3] != 0xe2 {
		t.Fatal("SAVE rewrote literal accepted steps or source suffix", m[0xb0:])
	}
	// SAV-CELLFAIL-583/CELLLEAVE-584's canonical table gives byte order,
	// independently of this build's capture code. The single boundary enters
	// 16:0,16:128 and detaches15:224,16:128. No constructor/default or +72
	// callback value is inferred from the deliberately different incoming bytes.
	wantCaches := []byte{0x91, 0x92, 0x93, 0x94, 0xa1, 0xa2, 0xa3, 0xa4}
	if tick > 0 {
		wantCaches = []byte{16, 16, 0, 128, 15, 16, 224, 128}
	}
	if !bytes.Equal(m[0x82:0x8a], wantCaches) || !bytes.Equal(motion.Mover[0x82:0x8a], wantCaches) {
		t.Fatalf("tick%d entry/detach caches: document=%x native=%x want=%x", tick, m[0x82:0x8a], motion.Mover[0x82:0x8a], wantCaches)
	}
	if !bytes.Equal(crossingRawField(t, record, "U15C"), []byte{0x10, 0x10}) || newGroupCount1115(t, *record, "U15C") != 1 {
		t.Fatal("completed dynamic crossing rewrote static source route")
	}
	wantDynamic := []byte{0x10, 0x10}
	if tick == 5 {
		wantDynamic = nil
	}
	if got := crossingRawField(t, record, "U178"); !bytes.Equal(got, wantDynamic) || newGroupCount1115(t, *record, "U178") != uint32(len(wantDynamic)/2) {
		t.Fatal("dynamic route/count did not follow center arrival", tick, got)
	}
	if !replaced {
		progress := byte(3)
		if tick == 5 {
			progress = 0
		}
		if got := crossingRawField(t, record, "U158")[9]; got != progress {
			t.Fatalf("tick%d order progress=%d want%d", tick, got, progress)
		}
	}
	wantCells, wantBlocks := crossingCells1115(), crossingBlocks1115()
	if tick > 0 {
		wantCells[1].GroundActor, wantCells[2].GroundActor = 0, newGroupA
	}
	if tick == 5 {
		wantBlocks[0].Dyn = 0
	}
	if got := snapshot.SavedDocument.Document.World.Cells; !reflect.DeepEqual(got, wantCells) {
		t.Fatalf("tick%d cell links/history/unowned payload changed:\n got%+v\nwant%+v", tick, got, wantCells)
	}
	_, _, blocks, present := f.live.world.SavedActorMotions()
	if !present || len(blocks) != len(wantBlocks) {
		t.Fatalf("tick%d native occupancy population=%d want%d", tick, len(blocks), len(wantBlocks))
	}
	ordinary := map[uint16]sav.BlockRecord{}
	for _, block := range snapshot.SavedDocument.Document.World.Blocks {
		ordinary[block.Cell] = block
	}
	for i, want := range wantBlocks {
		got := blocks[i]
		if got.Cell != want.Cell || got.Dyn != want.Dyn || got.Static != want.Static || ordinary[want.Cell] != want {
			t.Fatalf("tick%d occupancy at%04x: native%+v ordinary%+v want%+v", tick, want.Cell, got, ordinary[want.Cell], want)
		}
	}
	return snapshot
}

func crossingMenuSave(t *testing.T, app *ui.App, store SaveStore) string {
	t.Helper()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || !strings.HasSuffix(entries[0].Name, ".sav") {
		t.Fatal("ordinary mid-crossing menu SAVE", entries, err)
	}
	return entries[0].Name
}

func crossingNativeDoor(t *testing.T, store SaveStore, name string) *FrontEnd {
	t.Helper()
	f := newGroupFront(t, -1)
	app := f.App("fresh native crossing")
	save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, localOriginalSaveToken(name))
	return f
}

func TestSavedCrossing1115OriginalDoorsMenuSaveFineContinuation(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("from-map-%t/replacement-%t", fromMap, replace), func(t *testing.T) {
				f, app, store := crossingOriginalDoor1115(t, fromMap)
				initial := crossingCheck1115(t, f, 0, false)
				immutable, err := cloneSavedDocument(initial.SavedDocument)
				if err != nil {
					t.Fatal(err)
				}
				f.live.tick()
				tick := 1
				if replace {
					e, _ := crossingMotion1115(t, f)
					f.live.enqueue(uint32(e.ID), 22, 12)
					f.live.tick()
					tick++
				}
				before := crossingCheck1115(t, f, tick, replace)
				name := crossingMenuSave(t, app, store)
				fresh := crossingNativeDoor(t, store, name)
				after := crossingCheck1115(t, fresh, tick, replace)
				if !bytes.Equal(before.World, after.World) || f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("mid-crossing SAV LOAD changed the complete current World")
				}
				for tick++; tick <= 5; tick++ {
					f.live.tick()
					fresh.live.tick()
					crossingCheck1115(t, f, tick, replace)
					crossingCheck1115(t, fresh, tick, replace)
					if f.live.world.Hash() != fresh.live.world.Hash() {
						t.Fatal("fresh native fine continuation differs", tick)
					}
				}
				if !reflect.DeepEqual(initial.SavedDocument, immutable) {
					t.Fatal("later movement/SAVE mutated a detached old Snapshot")
				}
				if replace {
					e, motion := crossingMotion1115(t, fresh)
					if e.TargetX != 22 || e.TargetY != 12 || motion.Issue == "" {
						t.Fatal("replacement command lost target or hid continuation boundary", e, motion)
					}
				}
			})
		}
	}
}

func TestSavedCrossing1115RelocationDoesNotResumeRetainedSource(t *testing.T) {
	f, app, store := crossingOriginalDoor1115(t, false)
	crossingCheck1115(t, f, 0, false)
	e, _ := crossingMotion1115(t, f)
	// Production deterministic relocation, not an assertion about an original
	// teleport callback's unknown treatment of its mover. The native writer must
	// not leave the imported source crossing executable at the new destination.
	if err := f.live.world.HeadlessPlace(e.ID, 22, 16); err != nil {
		t.Fatal(err)
	}
	e, motion := crossingMotion1115(t, f)
	px, py := e.X, e.Y
	_, _, hasFine := f.live.world.ActorFinePosition(e.ID)
	if px < 21 || px > 24 || py < 15 || py > 18 || motion.Current || motion.Active || motion.Issue == "" || hasFine {
		t.Fatal("relocation retained executable source authority", e, motion, hasFine)
	}
	name := crossingMenuSave(t, app, store)
	fresh := crossingNativeDoor(t, store, name)
	for tick := 0; tick < 4; tick++ {
		e, motion := crossingMotion1115(t, fresh)
		if e.X != px || e.Y != py || motion.Current || motion.Active || f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("native LOAD resumed an invalidated original crossing", tick, e, motion)
		}
		f.live.tick()
		fresh.live.tick()
	}
}

// SAV-CELLLEAVE-584 makes this old payload deletion-eligible: every occupant
// besides the departing actor, layer count and operation byte is zero. Its
// nonzero residue is not a retainer. Native continuation must state the missing
// lifecycle operation, not claim that retaining this record matches ROM1.
func TestSavedCrossing1115EmptyOldCellKeepsLifecycleIssueAcrossNativeLoad(t *testing.T) {
	f, app, store := crossingCellOriginalDoor1115(t, false, false)
	e, motion := crossingMotion1115(t, f)
	if !motion.Active || motion.Issue != "" || motion.Position.Cell != 0x100f || motion.Position.FineX != 224 || len(f.live.world.Sacks()) != 0 {
		t.Fatal("empty-old-cell literal did not enter the active subset", e, motion)
	}
	f.live.tick()
	before := groupDocumentSnapshot1115(t, f)
	name := crossingMenuSave(t, app, store)
	fresh := crossingNativeDoor(t, store, name)
	after := groupDocumentSnapshot1115(t, fresh)
	if !bytes.Equal(before.World, after.World) || f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("lifecycle-issue native checkpoint changed current state")
	}
	for tick := 1; tick <= 5; tick++ {
		for _, current := range []*FrontEnd{f, fresh} {
			e, motion := crossingMotion1115(t, current)
			wantAction := uint32(1)
			if tick == 5 {
				wantAction = 0
			}
			if e.X != 16 || e.Y != 16 || motion.Position.FineX != byte((tick-1)*32) || motion.Position.FineY != 128 || motion.Active != (tick < 5) || motion.ActorAction != wantAction || !strings.Contains(motion.Issue, "empty cell deletion") {
				t.Fatalf("tick%d lost explicit lifecycle boundary or fine continuation: entity=%+v motion=%+v", tick, e, motion)
			}
		}
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("native continuation with lifecycle issue diverged", tick)
		}
		if tick < 5 {
			f.live.tick()
			fresh.live.tick()
		}
	}
}

// These remain grammar-valid documents and checksum-valid native envelopes.
// Only the detached document changes: native LOAD must reject disagreement,
// not reconstruct or repair the authoritative World from the retained DTO.
func TestSavedCrossing1115DocumentDisagreementRefusedBeforeAdoption(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, *sav.DocumentRecordData, *sav.DocumentData)
	}{
		{"position fraction", func(t *testing.T, actor *sav.DocumentRecordData, _ *sav.DocumentData) {
			crossingRawField(t, actor, "Block12")[4] = 1
		}},
		{"position terrain identity", func(t *testing.T, actor *sav.DocumentRecordData, _ *sav.DocumentData) {
			crossingRawField(t, actor, "Block12")[8] ^= 1
		}},
		{"accepted axis step", func(t *testing.T, actor *sav.DocumentRecordData, _ *sav.DocumentData) {
			crossingRawField(t, actor, "U154")[0xb0] = 31
		}},
		{"actor action", func(t *testing.T, actor *sav.DocumentRecordData, _ *sav.DocumentData) {
			binary.LittleEndian.PutUint32(crossingRawField(t, actor, "U54"), 0)
		}},
		{"static route", func(t *testing.T, actor *sav.DocumentRecordData, _ *sav.DocumentData) {
			crossingRawField(t, actor, "U15C")[0] = 17
		}},
		{"dynamic route", func(t *testing.T, actor *sav.DocumentRecordData, _ *sav.DocumentData) {
			crossingRawField(t, actor, "U178")[0] = 17
		}},
		{"ground slot", func(_ *testing.T, _ *sav.DocumentRecordData, doc *sav.DocumentData) {
			doc.World.Cells[2].GroundActor = newGroupB1115
		}},
		{"air slot", func(_ *testing.T, _ *sav.DocumentRecordData, doc *sav.DocumentData) {
			doc.World.Cells[2].AirActor = newGroupB1115
		}},
		{"reservation plane", func(_ *testing.T, _ *sav.DocumentRecordData, doc *sav.DocumentData) {
			doc.World.Blocks[0].Dyn ^= 0x40
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _, _ := crossingOriginalDoor1115(t, false)
			f.live.tick()
			before := crossingCheck1115(t, f, 1, false)
			bad := before
			var err error
			bad.SavedDocument, err = cloneSavedDocument(before.SavedDocument)
			if err != nil {
				t.Fatal(err)
			}
			e, _ := crossingMotion1115(t, f)
			var actor *sav.DocumentRecordData
			for _, binding := range bad.SavedDocument.Actors {
				if binding.EntityID == e.ID {
					actor = &bad.SavedDocument.Document.Objects[binding.ObjectIndex-1]
				}
			}
			if actor == nil {
				t.Fatal("corruption fixture lacks exact actor binding")
			}
			tc.mutate(t, actor, bad.SavedDocument.Document)
			envelope := uncheckedDocumentEnvelope1115(t, bad)
			label := []byte("Disagreeing saved crossing")
			binary.LittleEndian.PutUint16(envelope[9:11], uint16(len(label)))
			header := append(bytes.Clone(envelope[:11]), label...)
			envelope = append(header, envelope[11:]...)
			decoded, _, err := DecodeSave(envelope)
			if err != nil {
				t.Fatal("DTO fixture must reach semantic World validation", err)
			}
			if !bytes.Equal(decoded.World, before.World) {
				t.Fatal("DTO-only corruption changed native World")
			}
			oldLive, oldTown, oldShop := f.live, f.Town, f.Shop
			if open, town, err := f.Restore(decoded); err == nil || open != nil || town {
				t.Fatal("Restore adopted disagreeing saved crossing", open != nil, town, err)
			}
			if !reflect.DeepEqual(before, groupDocumentSnapshot1115(t, f)) || f.live != oldLive || f.Town != oldTown || f.Shop != oldShop {
				t.Fatal("refused DTO changed FrontEnd state")
			}
			store := SaveStore{Dir: t.TempDir()}
			if err := os.WriteFile(filepath.Join(store.Dir, "disagreeing-crossing.ags"), envelope, 0600); err != nil {
				t.Fatal(err)
			}
			app := f.App("refused crossing document")
			save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
			app.SetSaveSeams(save, list, load)
			rows := list()
			if len(rows) != 1 {
				t.Fatal("hostile DTO has no ordinary LOAD row", rows)
			}
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate(rows[0].Label); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenLoad || app.HeadlessMessage() == "" {
				t.Fatal("ordinary LOAD did not expose DTO refusal", app.Screen(), app.HeadlessMessage())
			}
			if !reflect.DeepEqual(before, groupDocumentSnapshot1115(t, f)) || f.live != oldLive || f.Town != oldTown || f.Shop != oldShop {
				t.Fatal("ordinary refused DTO LOAD adopted partial state")
			}
		})
	}
}
