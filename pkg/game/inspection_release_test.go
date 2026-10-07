package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The pointer passes through App.step. Only the fixture's presentation fog is
// opened so distant installed objects can be sampled without playing a mission.
// No input is sent to a desktop window and no install is written.
func TestReleaseInspectionLivePanels(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("1083-hover")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	if err := a.HeadlessSelectEntity(uint32(live.mission.ids[0])); err != nil {
		t.Fatal(err)
	}
	selected, _ := live.view.SelectedUnit()
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	invBefore := live.invSubject
	before := live.world.Hash()
	pending := len(live.pending)
	regBytes, err := f.Archives.Containers.ReadFile("graphics/structures/structures.reg")
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(regBytes)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := data.LoadStructureClasses(r)
	if err != nil {
		t.Fatal(err)
	}

	// Mission-10 structure 0 and unit 0 are distinct even though their handle
	// numbers overlap. Decorations without a health pool are covered separately.
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1024, 768}, {1280, 720}, {1920, 1080}} {
		a.Layout(size.X, size.Y)
		t.Logf("window=%v frame=%v viewport=%v scale=%g", size, live.view.FrameSize(), live.view.ViewportSize(), live.view.Placement().Scale())
		for _, sid := range []sim.StructureID{0, 7} {
			var st sim.Structure
			for _, s := range live.world.Structures() {
				if s.ID == sid {
					st = s
				}
			}
			kind := live.mission.state.Map.Objects[sid].Kind
			c, ok := classes.ByID(int32(kind))
			if !ok {
				t.Fatalf("missing registry kind %d", kind)
			}
			inspectionCentre(live, int(st.Col), int(st.Row))
			ref := ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(sid)}
			releaseHoverInspection(t, a, live, ref)
			s, ok := live.view.InspectionPanel()
			wantName := f.Words.BuildingNames[kind]
			if wantName == "" {
				t.Fatalf("the install states no building name for class %d", kind)
			}
			if !ok || s.Kind != ui.InspectionStructure || s.ClassID != int32(kind) || s.Name != wantName || s.HP != int(int16(st.Field42)) || s.MaxHP != int(st.MaxHealth) {
				t.Fatalf("structure %d panel=%+v want building %d %q HP%d/%d", sid, s, kind, wantName, st.Field42, st.MaxHealth)
			}
			addr := "graphics/infowindow/" + c.Picture + ".bmp"
			b, readErr := f.Archives.Containers.ReadFile(addr)
			pic, _, err := a.HeadlessCharacterPane()
			if err != nil {
				t.Fatal(err)
			}
			if readErr != nil {
				t.Fatalf("missing %s: %v", addr, readErr)
			}
			checkInspectionFigure(t, pic, inspectionPortraitBMP(t, b), addr)
			t.Logf("%v structure %d class=%d HP=%d/%d picture=%s", size, sid, kind, s.HP, s.MaxHP, addr)
			card, err := a.HeadlessMissionCard()
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(card.Pix, pic.Pix) {
				t.Fatal("stats card is a duplicate figure")
			}
			statement, ok := live.view.PanelStatement()
			if !ok || !strings.Contains(strings.Join(statement, " "), fmt.Sprintf("%d/%d", st.Field42, st.MaxHealth)) {
				t.Fatalf("structure live health absent: %v", statement)
			}
		}
		// Fixed shipped Human discriminator: definition face 17, a single
		// 0x8107 weapon. Expected art is built from two explicit archive leaves,
		// not unitFigure, FigureFor, ItemFigurePath, or the production compositor.
		const human sim.EntityID = 22
		e, ok := live.entity(human)
		if !ok || e.Owner == sim.SelfSlot {
			t.Fatal("non-owned human22 missing")
		}
		eq, _ := live.world.Equipped(human)
		if eq != ([sim.EquipSlots]uint16{0x8107}) {
			t.Fatalf("human22 discriminator changed: %x", eq)
		}
		inspectionCentre(live, int(e.X), int(e.Y))
		releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: uint32(human)})
		want := inspectionSprite(t, f, "graphics/equipment/mfighter/17.256")
		weapon := inspectionSprite(t, f, "graphics/equipment/mfighter/primary/0801007.256")
		changed := 0
		for y := 0; y < want.Bounds().Dy(); y++ {
			for x := 0; x < want.Bounds().Dx(); x++ {
				c := weapon.RGBAAt(x, y)
				if c.A != 0 {
					if want.RGBAAt(x, y) != c {
						changed++
					}
					want.SetRGBA(x, y, c)
				}
			}
		}
		if changed < 20 {
			t.Fatal("weapon source did not discriminate equipped figure")
		}
		pic, _, err := a.HeadlessCharacterPane()
		if err != nil {
			t.Fatal(err)
		}
		checkInspectionFigure(t, pic, want, "human22 explicit base+weapon")
		s, _ := live.view.InspectionPanel()
		if s.ID != uint32(human) || s.Kind != ui.InspectionUnit || s.HP != int(e.HP) || s.MaxHP != int(e.MaxHP) || s.Combat.Defence != int(e.Defence) {
			t.Fatalf("human live card=%+v entity=%+v", s, e)
		}
		if got, _ := live.view.SelectedUnit(); got != selected {
			t.Fatal("hover changed selection")
		}
		if !reflect.DeepEqual(live.invSubject, invBefore) {
			t.Fatal("hover replaced inventory subject")
		}
	}
	if live.world.Hash() != before || len(live.pending) != pending {
		t.Fatal("paused hover altered world hash/commands")
	}

	// A real sim equipment write changes the non-owned figure without moving
	// selection or making that unit an inventory interaction subject.
	headlessDamage(t, live.world, 22, 1)
	sim.Step(live.world, []sim.Command{{Kind: sim.KindUnequip, Entity: 22, X: 1}})
	live.push()
	e, _ := live.entity(22)
	inspectionCentre(live, int(e.X), int(e.Y))
	releaseHoverInspection(t, a, live, ui.InspectionSubject{Kind: ui.InspectionUnit, ID: 22})
	eq, _ := live.world.Equipped(22)
	if eq[0] != 0 {
		t.Fatal("fixture unequip did not run")
	}
	bare := inspectionSprite(t, f, "graphics/equipment/mfighter/17.256")
	pic, _, err := a.HeadlessCharacterPane()
	if err != nil {
		t.Fatal(err)
	}
	checkInspectionFigure(t, pic, bare, "live unequipped human22")
	s, _ := live.view.InspectionPanel()
	if s.HP != int(e.HP) {
		t.Fatal("live damage not projected")
	}
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	if ref, ok := live.view.Inspection(); !ok || ref != (ui.InspectionSubject{Kind: ui.InspectionUnit, ID: selected}) {
		t.Fatal("pointer leave failed selected fallback")
	}
	if !reflect.DeepEqual(live.invSubject, invBefore) {
		t.Fatal("enemy equip/damage changed selected inventory")
	}
	// Moving from a foreign hover into the doll returns to selection before
	// dispatching its slot click. Both foreign kinds must queue the hero's item,
	// never a numerically colliding unit or the inspected NPC's equipment.
	for _, ref := range []ui.InspectionSubject{{Kind: ui.InspectionUnit, ID: 22}, {Kind: ui.InspectionStructure, ID: 0}} {
		if ref.Kind == ui.InspectionUnit {
			inspectionCentre(live, int(e.X), int(e.Y))
		} else {
			inspectionCentre(live, 23, 65)
		}
		releaseHoverInspection(t, a, live, ref)
		x, y, err := a.HeadlessDollSlotPoint(1)
		if err != nil {
			t.Fatal(err)
		}
		n := len(live.pending)
		if err := a.HeadlessPointer("press", x, y); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessPointer("release", x, y); err != nil {
			t.Fatal(err)
		}
		// Requests are drained by the next production frame's world advance.
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		if len(live.pending) != n+1 || live.pending[n].Entity != sim.EntityID(selected) || live.pending[n].Kind != sim.KindUnequip {
			t.Fatalf("after hover %+v doll click queued %+v, want selected%d unequip", ref, live.pending[n:], selected)
		}
	}
	t.Logf("paused pointer matrix kept hash=%016x; non-owned live equipment/damage refreshed; selected fallback=%d", before, selected)
}

func inspectionCentre(live *mapWorld, col, row int) {
	cam := live.view.Camera()
	cam.X = float64(col*32 - cam.ViewW/2)
	cam.Y = float64(row*32 - cam.ViewH/2)
	cam.Clamp()
}

func releaseHoverInspection(t *testing.T, a *ui.App, live *mapWorld, ref ui.InspectionSubject) {
	t.Helper()
	x, y, err := live.view.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	if got, ok := live.view.Inspection(); !ok || got != ref {
		t.Fatalf("pointer got %+v/%v want %+v", got, ok, ref)
	}
}

// Independent 24-bit BMP reading: no production portrait decoder/converter.
func inspectionBMP(t *testing.T, b []byte) *image.RGBA {
	t.Helper()
	if len(b) < 54 || string(b[:2]) != "BM" || binary.LittleEndian.Uint16(b[28:]) != 24 || binary.LittleEndian.Uint32(b[30:]) != 0 {
		t.Fatal("unexpected witness BMP shape")
	}
	w, h := int(binary.LittleEndian.Uint32(b[18:])), int(int32(binary.LittleEndian.Uint32(b[22:])))
	if w <= 0 || w > 1024 || h <= 0 || h > 1024 {
		t.Fatal("unexpected BMP dimensions")
	}
	off := int(binary.LittleEndian.Uint32(b[10:]))
	stride := (w*3 + 3) &^ 3
	if off < 54 || off+stride*h > len(b) {
		t.Fatal("truncated witness BMP")
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := off + (h-1-y)*stride + 3*x
			out.SetRGBA(x, y, color.RGBA{R: b[i+2], G: b[i+1], B: b[i], A: 255})
		}
	}
	return out
}

// The .256 parser is shared; the figure address, layer choice, pixel copy and
// layer order are independent of the production figure/render route.
func inspectionSprite(t *testing.T, f *FrontEnd, addr string) *image.RGBA {
	t.Helper()
	b, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatal(err)
	}
	s, err := spr256.Decode(b)
	if err != nil || len(s.Frames) != 1 || len(s.Palette) != 256 {
		t.Fatalf("%s: invalid figure source: %v", addr, err)
	}
	fm := s.Frames[0]
	out := image.NewRGBA(image.Rect(0, 0, fm.Width, fm.Height))
	for i, p := range fm.Pixels {
		if p.Opaque {
			c := s.Palette[p.Index]
			out.SetRGBA(i%fm.Width, i/fm.Width, color.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}
	return out
}

func checkInspectionFigure(t *testing.T, got, want *image.RGBA, what string) {
	t.Helper()
	n := 0
	// Source origin is pane-local (16,2). The centre columns exclude corner
	// controls; all bottom rows are checked without using composer geometry.
	for y := 40; y < 240; y++ {
		for x := 32; x < 120; x++ {
			c := want.RGBAAt(x, y)
			if c.A != 0 {
				n++
				if got.RGBAAt(x+16, y+2) != c {
					t.Fatalf("%s: wrong composed pixel at source %d,%d", what, x, y)
				}
			}
		}
	}
	if n < 1000 {
		t.Fatalf("%s: only %d independent opaque pixels", what, n)
	}
}
