package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func structureFront(t *testing.T, generated ...bool) *FrontEnd {
	f := structureFixtureFront(t, 51, 52, 53)
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	f.Font = resolved(missionFont(), nil)
	buildingTable := f.Table.Buildings
	f.Table = actorRegistryTable()
	f.Table.Buildings = buildingTable
	objects := []synth.ALMObject{{X: 10 << 8, Y: 10 << 8, Kind: 1, Field12: 51}, {X: 18 << 8, Y: 10 << 8, Kind: 1, Field12: 52}, {X: 26 << 8, Y: 10 << 8, Kind: 1, Field12: 53}}
	if len(generated) > 2 && generated[2] {
		objects[1].Field12 = 51
	}
	path := filepath.Join(t.TempDir(), ScenarioArchive)
	paths := []string{path}
	options := synth.ALMOptions{Width: 40, Height: 40, Objects: objects}
	if len(generated) > 1 && generated[1] {
		units := f.Table.Units.(dbCollection)
		units[1].params[29], units[1].params[30], units[1].params[32] = 35, 0, 2
		options.Units = []synth.ALMUnit{{X: 8 << 8, Y: 8 << 8, ClassID: 35}}
	}
	if len(generated) != 0 && generated[0] {
		drop := make([]byte, 4+796+8)
		binary.LittleEndian.PutUint32(drop, 1)
		binary.LittleEndian.PutUint32(drop[4+0x40:], 0x10002)
		binary.LittleEndian.PutUint32(drop[4+0x44:], 2)
		binary.LittleEndian.PutUint32(drop[4+0x4c:], 16)
		binary.LittleEndian.PutUint32(drop[4+0x50:], 16)
		binary.LittleEndian.PutUint32(drop[4+0x74:], 5)
		binary.LittleEndian.PutUint32(drop[4+0x78:], 6)
		options.Type7Payload = drop
		worldPath := filepath.Join(t.TempDir(), WorldArchive)
		if err := os.WriteFile(worldPath, synth.Archive([]synth.File{{Path: "data/map.reg", Data: synth.Reg(17, nil)}}), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, worldPath)
		f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
		f.Town = NewTown(f.Campaign.Value())
	}
	if err := os.WriteFile(path, synth.Archive([]synth.File{{Path: "10.alm", Data: synth.ALM(options)}, {Path: "npc.reg", Data: synth.NPCReg(nil)}}), 0600); err != nil {
		t.Fatal(err)
	}
	fs, err := vfs.Open(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Archives.Containers = fs
	f.Units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{35: worldFixtureArt(16, 16, 8, 16, 3, 3, 1)}}
	f.Structures = &terrain.StructureSet{}
	for k := 1; k <= 2; k++ {
		frame := &terrain.StaticFrame{Width: 32, Height: 32, Pixels: make([]terrain.StaticPixel, 1024)}
		frame.Palette[1] = color.RGBA{R: byte(100 * k), G: 30, A: 255}
		for i := range frame.Pixels {
			frame.Pixels[i] = terrain.StaticPixel{Index: 1, Opaque: true}
		}
		portrait := image.NewRGBA(image.Rect(0, 0, 160, 240))
		for i := range portrait.Pix {
			portrait.Pix[i] = byte(80 * k)
		}
		f.Structures.Classes[k] = &terrain.StructureClass{ID: int32(k), Name: []string{"Moved building", "Source-only building"}[k-1],
			TileWidth: 1, TileHeight: 1, FullHeight: 1, Frames: []*terrain.StaticFrame{frame}, Portrait: portrait}
	}
	return f
}

// Literal writer: two distinct Building objects plus one repeated MFC ref.
// The Player/Unit fixture establishes archive indices1..4; Building is5 and
// its objects6/7. No production Building/slot projection supplies expected data.
func structureSave1114(t *testing.T, remint uint32, badKey bool) []byte {
	profile := &profileFixture1107{}
	profile.attack[20], profile.attack[21] = 20, 1
	a := &poolFixtureActor{cell: 0x0c0b, hp: 100, maxHP: 100, name: "Builder witness", profile: profile,
		loadWords: &[4]int16{100, 0, 0, 300}, equipmentRuntime: &[3]byte{1, 3, 2}}
	body := poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{a}}}}, nil)
	body[a.off+16] = 1
	binary.LittleEndian.PutUint16(body[a.off+17:], 35)
	body[a.off+509], body[a.off+510], body[a.off+511] = 1, 1, 3
	body[a.off+179], body[a.off+189] = 64, 100
	// Locate only the splice boundary, not an expected field value. Building
	// and cell payloads below are independently written at literal offsets.
	sf, err := sav.Open(savedContainer(body))
	if err != nil {
		t.Fatal(err)
	}
	b := append([]byte(nil), body[:sf.World.BlocksOff-8]...)
	b = binary.LittleEndian.AppendUint32(b, 3)
	for i := 0; i < 2; i++ {
		if i == 0 {
			b = binary.LittleEndian.AppendUint16(b, 0xffff)
			b = binary.LittleEndian.AppendUint16(b, 1)
			b = binary.LittleEndian.AppendUint16(b, 8)
			b = append(b, "Building"...)
		} else {
			b = binary.LittleEndian.AppendUint16(b, 0x8005)
		}
		r := [77]byte{}
		r[0], r[1] = []byte{12, 20}[i], []byte{12, 14}[i]
		r[2], r[3] = r[0], r[1]
		r[4], r[5] = 91, 73
		binary.LittleEndian.PutUint32(r[12:], uint32(70-i))
		if i == 0 {
			binary.LittleEndian.PutUint32(r[19:], 51)
		}
		binary.LittleEndian.PutUint32(r[29:], remint+uint32(700+i))
		for j := 37; j < 59; j++ {
			r[j] = byte(j)
		}
		r[59] = byte(i + 1)
		hp, maxhp := uint16(90), uint16(130)
		if i == 1 {
			hp, maxhp = 0xffff, 0
		}
		binary.LittleEndian.PutUint16(r[60:], hp)
		binary.LittleEndian.PutUint16(r[62:], maxhp)
		r[67], r[68] = 3, 2
		binary.LittleEndian.PutUint32(r[69:], 5)
		binary.LittleEndian.PutUint32(r[73:], 63)
		b = append(b, r[:]...)
	}
	b = binary.LittleEndian.AppendUint16(b, 6)
	b = binary.LittleEndian.AppendUint32(b, 0) // SpellEffects
	b = binary.LittleEndian.AppendUint16(b, 3)
	for _, v := range []uint32{0x0c0c2525, 0x0c0de020, 0x0c0e2525} {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	b = binary.LittleEndian.AppendUint16(b, 6)
	for i, c := range []struct {
		cell uint16
		key  uint32
	}{{0x0c0c, 700}, {0x0c0d, 700}, {0x0c0e, 700}, {0x0c0d, 701}, {0xffff, 701}, {0x0c0c, 0}} {
		b = binary.LittleEndian.AppendUint16(b, c.cell)
		r := [52]byte{byte(6 + i), byte(i & 1)}
		if c.key != 0 {
			c.key += remint
		}
		if badKey && i == 5 {
			c.key = 0xdeadbeef
		}
		binary.LittleEndian.PutUint32(r[12:], c.key)
		b = append(b, r[:]...)
	}
	b = append(b, make([]byte, 4+4374+4)...)
	return savedContainer(savedTrailer(b))
}

func assertStructures1114(t *testing.T, w *sim.World) {
	t.Helper()
	got := w.Structures()
	want := []sim.Structure{{Kind: 1, ID: 0, Field42: 90, MaxHealth: 130, Col: 12, Row: 12, Width: 3, Height: 2, Attach: 63, Blocking: 5},
		{Kind: 2, ID: 4, Field42: 0xffff, MaxHealth: 0, Col: 20, Row: 14, Width: 3, Height: 2, Attach: 63, Blocking: 5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roster %+v want %+v", got, want)
	}
	meta, cells, present := w.SavedStructures()
	if !present || len(meta) != 2 || len(cells) != 4 || meta[0].ArchiveIndex != 6 || meta[1].ArchiveIndex != 7 || !meta[0].HasAuthored || meta[1].HasAuthored || meta[1].AuthoredID != 0 {
		t.Fatalf("namespaces %+v cells%+v", meta, cells)
	}
	wantCells := []sim.SavedStructureCell{{Cell: 0x0c0c, BaselineCost: 11, BaselineStatic: 1},
		{Cell: 0x0c0d, BaselineCost: 9, BaselineStatic: 1, ID: 4, HasStructure: true},
		{Cell: 0x0c0e, BaselineCost: 8, ID: 0, HasStructure: true},
		{Cell: 0xffff, BaselineCost: 10, ID: 4, HasStructure: true}}
	if !reflect.DeepEqual(cells, wantCells) {
		t.Fatalf("last-write explicit cells %+v", cells)
	}
}

func structureWitnessActor1114(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.SourceBinding.TypeID == 35 {
			return e
		}
	}
	t.Fatal("source actor absent")
	return sim.Entity{}
}

func TestSavedStructures1114BothDoorsAndFreshNativeAction(t *testing.T) {
	f := structureFront(t)
	app := f.App("1114 saved structures")
	app.Layout(1024, 768)
	if path := os.Getenv("AGAINROM_STRUCTURES_1114_NATIVE"); path != "" {
		store := SaveStore{Dir: filepath.Dir(path)}
		save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, filepath.Base(path))
		assertStructures1114(t, f.live.world)
		e := structureWitnessActor1114(t, f.live.world)
		if e.X != 11 || e.Y != 13 {
			t.Fatalf("saved actor %+v", e)
		}
		f.live.enqueue(uint32(e.ID), 13, 13)
		registryReach1111(t, f.live, e.ID, 13, 13)
		t.Log("1114 fresh native LOAD and next movement PASS")
		return
	}
	payload := structureSave1114(t, 0, false)
	ms, report, err := loadOriginalMission(f, payload)
	if err != nil || report.Structures.Restored != 2 || report.Structures.Absent != 2 {
		t.Fatalf("diagnostic %+v %v", report.Structures, err)
	}
	assertStructures1114(t, ms.World)
	ms2, _, err := loadOriginalMission(f, structureSave1114(t, 0x100000, false))
	if err != nil || !reflect.DeepEqual(ms2.World.Structures(), ms.World.Structures()) {
		t.Fatalf("reminted identities changed bindings: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	assertStructures1114(t, f.live.world)
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	for _, id := range []uint32{0, 4} {
		ref := ui.InspectionSubject{Kind: ui.InspectionStructure, ID: id}
		inspectionCentre(f.live, 12, 12)
		x, y, err := f.live.view.InspectionPoint(ref)
		if id == 4 {
			// The source-only zero-maximum record still persists and draws,
			// but owner-directed decoration policy excludes it from hover.
			if err == nil {
				t.Fatal("saved decoration without a health pool became inspectable")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		panel, ok := f.live.view.InspectionPanel()
		if !ok || panel.ID != id {
			t.Fatalf("saved hover %+v", panel)
		}
		if card, err := app.HeadlessMissionCard(); err != nil || len(card.Pix) == 0 {
			t.Fatalf("saved card %v", err)
		}
	}
	if entries, _ := f.live.view.Structures(); entries != 2 {
		t.Fatalf("first-frame cached roster entries%d", entries)
	}
	e := structureWitnessActor1114(t, f.live.world)
	f.live.enqueue(uint32(e.ID), 11, 13)
	registryReach1111(t, f.live, e.ID, 11, 13)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("SAVE %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestSavedStructures1114BothDoorsAndFreshNativeAction$", "-test.v")
	cmd.Env = append(os.Environ(), "AGAINROM_STRUCTURES_1114_NATIVE="+filepath.Join(store.Dir, entries[0].Name))
	output, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("next movement PASS")) {
		t.Fatalf("fresh process %v\n%s", err, output)
	}
	t.Log(string(output))
}

func TestSavedStructures1114LateReferenceRefusesAtomically(t *testing.T) {
	f := structureFront(t, true)
	native, nativeErr := StartMission(f.Archives.Containers, 10, f.Table, mapload.DifficultyNormal, f.NextParty())
	if nativeErr != nil {
		t.Fatal(nativeErr)
	}
	if native.Start.Fallback || native.Start.Drop != (mapload.Cell{X: 16, Y: 16}) {
		t.Fatal("fixture has no authored in-bounds native start")
	}
	app := f.App("1114 atomic")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	old, hash := f.live, f.live.world.Hash()
	before, _, err := f.Snapshot(true)
	if err != nil || len(before.World) == 0 || before.SavedDocument != nil {
		t.Fatal("native entry lacked current World or manufactured a source document", err)
	}
	for _, e := range native.World.Entities() {
		want, _ := native.World.Carried(e.ID)
		got, _ := f.live.world.Carried(e.ID)
		if !reflect.DeepEqual(want, got) {
			t.Fatal("current code-only inventory changed during construction")
		}
		wantItems, _ := native.World.CarriedItems(e.ID)
		items, _ := f.live.world.CarriedItems(e.ID)
		if len(items) != len(wantItems) {
			t.Fatal("native entry changed the item population", e.ID)
		}
		for i := range items {
			items[i].ObjectID, wantItems[i].ObjectID = 0, 0
			if !reflect.DeepEqual(items[i], wantItems[i]) {
				currentItemFieldDiagnostics(t, "entry Item", reflect.ValueOf(wantItems[i]), reflect.ValueOf(items[i]))
				t.Fatal("native entry changed current item operands before any save", e.ID)
			}
		}
	}
	if _, _, err := f.RestoreOriginal(structureSave1114(t, 0, true)); err == nil {
		t.Fatal("unresolved late Building ref accepted")
	}
	if f.live != old || f.live.world.Hash() != hash {
		t.Fatal("late reference replaced active mission")
	}
	after, _, err := f.Snapshot(true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("late reference changed current persistence state", err)
	}
}

func TestSavedStructures1114AppAttackHitsMovedBuilding(t *testing.T) {
	f := structureFront(t)
	app := f.App("1114 physical attack")
	app.Layout(1024, 768)
	opener, _, err := f.RestoreOriginal(structureSave1114(t, 0, false))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	actor := structureWitnessActor1114(t, live.world)
	inspectionCentre(live, int(actor.X), int(actor.Y))
	if err := app.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, 12, 12)
	if err := app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	x, y, err := live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: 0})
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttackStructure || live.pending[0].X != 0 {
		t.Fatalf("saved sprite attack queued%+v", live.pending)
	}
	for i := 0; i < 256 && live.world.Structures()[0].Field42 == 90; i++ {
		live.tick()
	}
	if st := live.world.Structures(); st[0].Field42 == 90 || st[1].Field42 != 0xffff {
		t.Fatalf("physical strike target %+v", st)
	}
}
