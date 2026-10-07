package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Independent physical ALM walk: no model locator, decoded Map or serializer.
func spans1118(t *testing.T, b []byte) map[uint32]int {
	t.Helper()
	if len(b) < 20 {
		t.Fatal("short fixture")
	}
	result := map[uint32]int{}
	off := 20
	for n := uint32(0); n < binary.LittleEndian.Uint32(b[12:]); n++ {
		if off+20 > len(b) {
			t.Fatal("short record header")
		}
		size := int(binary.LittleEndian.Uint32(b[off+8:]))
		id := binary.LittleEndian.Uint32(b[off+12:])
		if size > len(b)-off-20 {
			t.Fatal("short record payload")
		}
		result[id] = off + 20
		off += 20 + size
	}
	return result
}

func movedBytes1118(t *testing.T, before, after []byte, index int, cell image.Point) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("length changed %d -> %d", len(before), len(after))
	}
	off := spans1118(t, before)[6] + index*70
	// Byte-by-byte, independent of MoveUnit's DWORD write: low fractional
	// bytes stay; only the next three bytes of each coordinate may change.
	want := append([]byte(nil), before...)
	for i, v := range []int{cell.X, cell.Y} {
		want[off+i*4+1] = byte(v)
		want[off+i*4+2] = byte(v >> 8)
		want[off+i*4+3] = byte(v >> 16)
	}
	for i := range before {
		if after[i] != want[i] {
			t.Fatalf("byte %#x: got %#x want %#x; editable span %#x..%#x", i, after[i], want[i], off, off+8)
		}
	}
}

func fixture1118(t *testing.T) []byte {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: triggerPayload1093(), Units: []synth.ALMUnit{{X: 3*256 + 128, Y: 4*256 + 255, ClassID: 300}, {X: 7*256 + 3, Y: 8*256 + 5, ClassID: 301}}})
	s := spans1118(t, b)
	binary.LittleEndian.PutUint16(b[s[6]+0x40:], 2)
	binary.LittleEndian.PutUint32(b[s[6]+0x42:], 5)
	appendSection := func(id uint32, payload []byte) {
		h := make([]byte, 20)
		binary.LittleEndian.PutUint32(h, 7)
		binary.LittleEndian.PutUint32(h[4:], 20)
		binary.LittleEndian.PutUint32(h[8:], uint32(len(payload)))
		binary.LittleEndian.PutUint32(h[12:], id)
		binary.LittleEndian.PutUint32(h[16:], 0xdeadbeef)
		b = append(append(b, h...), payload...)
		binary.LittleEndian.PutUint32(b[12:], binary.LittleEndian.Uint32(b[12:])+1)
	}
	units := append([]byte(nil), b[s[6]:s[6]+140]...)
	// Different overwritten coordinates ensure only the LAST type 6 changes.
	for i := 0; i < 8; i++ {
		b[s[6]+i] = 0x5a
	}
	appendSection(77, []byte("unknown physical payload"))
	appendSection(6, units)
	return append(b, []byte("uncounted trailer")...)
}

func open1118(t *testing.T, b []byte) (*MapInspector, *ui.MapEditor, string) {
	t.Helper()
	x := &MapInspector{tiles: &terrain.Tileset{}}
	path := filepath.Join(t.TempDir(), "source.alm")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open(path); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.Document().Viewer.SetFlat(true)
	return x, e, path
}

func point1118(e *ui.MapEditor, cell image.Point) image.Point {
	x, y := e.Document().Viewer.Camera().WorldToScreen(float64(cell.X*32+16), float64(cell.Y*32+16))
	return image.Pt(int(x), int(y))
}

func drag1118(e *ui.MapEditor, from, to image.Point) {
	now := time.Unix(1, 0)
	a, b := point1118(e, from), point1118(e, to)
	e.Pointer(ui.Input{CursorX: a.X, CursorY: a.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: b.X, CursorY: b.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: b.X, CursorY: b.Y}, now)
}

func TestMapEditor1118LosslessMoveUndoSaveAsFreshOpen(t *testing.T) {
	b := fixture1118(t)
	x, e, source := open1118(t, b)
	e.Select(0, true)
	cam := *e.Document().Viewer.Camera()
	oldTrigger := inspectRecord1093(t, e.Document(), "Condition", 2)
	if len(oldTrigger.References) != 1 || oldTrigger.References[0].Targets[0].Cell != image.Pt(3, 4) {
		t.Fatal("fixture lacks Unit ID 2 reference")
	}
	drag1118(e, image.Pt(3, 4), image.Pt(5, 6))
	d := e.Document()
	if !e.Dirty() || !d.Edits.CanUndo() || d.Edits.CanRedo() || e.Selected() != 0 || d.Records[0].Cell != image.Pt(5, 6) {
		t.Fatal("pointer move did not retain edited selection/history")
	}
	if !reflect.DeepEqual(cam, *d.Viewer.Camera()) {
		t.Fatal("unit drag panned or replaced camera")
	}
	movedBytes1118(t, b, d.SourceBytes, 0, image.Pt(5, 6))
	if !strings.Contains(inspectText1093(d.Records[0]), "X/Y raw: 1408 / 1791") || d.Records[0].Warning == "" {
		t.Fatal("raw details or unresolved-art warning stale")
	}
	ref := inspectRecord1093(t, d, "Condition", 2).References[0]
	if ref.Targets[0].Cell != image.Pt(5, 6) {
		t.Fatal("trigger UnitID placement was not refreshed")
	}
	changed := append([]byte(nil), d.SourceBytes...)
	if err := e.Key("undo"); err != nil || e.Dirty() || !bytes.Equal(b, e.Document().SourceBytes) || e.Document().Edits.CanUndo() {
		t.Fatalf("one drag was not exactly one reversible edit: %v", err)
	}
	if err := e.Key("redo"); err != nil || !e.Dirty() || !bytes.Equal(changed, e.Document().SourceBytes) {
		t.Fatalf("redo not byte exact: %v", err)
	}
	before := e.Document()
	if err := e.Open(source); err == nil || e.Document() != before || !e.Dirty() {
		t.Fatal("dirty open silently discarded work")
	}
	if err := e.Key("close"); err != nil {
		t.Fatal("native close discarded dirty work")
	}
	_ = e.Key("saveas")
	e.TextInput(filepath.Join(t.TempDir(), "cancelled.alm"))
	_ = e.Key("escape")
	if !e.Dirty() || e.Document() != before {
		t.Fatal("path cancellation changed document")
	}
	_ = e.Key("saveas")
	e.TextInput(source)
	if err := e.Key("enter"); err == nil || !e.Dirty() || !e.Document().Edits.CanUndo() {
		t.Fatal("failed overwrite changed checkpoint/history")
	}
	_ = e.Key("escape")
	target := filepath.Join(t.TempDir(), "new.alm")
	_ = e.Key("saveas")
	e.TextInput(target)
	if err := e.Key("enter"); err != nil || e.Dirty() {
		t.Fatalf("explicit Save As failed: %v", err)
	}
	if !e.Document().Edits.CanUndo() || e.Document().Source != target {
		t.Fatal("Save As discarded history or retained wrong source")
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, changed) {
		t.Fatalf("saved wrong bytes: %v", err)
	}
	original, _ := os.ReadFile(source)
	if !bytes.Equal(original, b) {
		t.Fatal("source file changed")
	}
	fresh := x.Editor()
	if err := fresh.Open(target); err != nil || fresh.Dirty() || fresh.Document().Edits.CanUndo() || !bytes.Equal(fresh.Document().SourceBytes, changed) || fresh.Document().Records[0].Cell != image.Pt(5, 6) {
		t.Fatalf("fresh editor reopen failed: %v", err)
	}
	_ = e.Key("undo")
	if !e.Dirty() {
		t.Fatal("undo after Save As failed to become dirty")
	}
	_ = e.Key("redo")
	if e.Dirty() {
		t.Fatal("redo back to saved bytes stayed dirty")
	}
	_ = e.Key("undo")
	drag1118(e, image.Pt(3, 4), image.Pt(6, 7))
	if e.Document().Edits.CanRedo() {
		t.Fatal("new edit did not truncate redo branch")
	}
	movedBytes1118(t, b, e.Document().SourceBytes, 0, image.Pt(6, 7))
}

func TestMapEditor1118CancelledAndNoopDragsKeepHistory(t *testing.T) {
	for _, mode := range []string{"click", "same-cell", "escape", "rail", "outside"} {
		t.Run(mode, func(t *testing.T) {
			b := fixture1118(t)
			_, e, _ := open1118(t, b)
			e.Select(0, true)
			p := point1118(e, image.Pt(3, 4))
			now := time.Unix(1, 0)
			e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
			switch mode {
			case "same-cell":
				p.X += 6
			case "escape":
				_ = e.Key("escape")
				p = point1118(e, image.Pt(6, 7))
			case "rail":
				p.X = 1000
			case "outside":
				p.X = -10
			}
			e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
			e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, now)
			if e.Dirty() || e.Document().Edits.CanUndo() || !bytes.Equal(b, e.Document().SourceBytes) {
				t.Fatal("cancel/no-op changed bytes or history")
			}
			// Release clears capture; the next valid drag must work normally.
			drag1118(e, image.Pt(3, 4), image.Pt(5, 6))
			movedBytes1118(t, b, e.Document().SourceBytes, 0, image.Pt(5, 6))
		})
	}
}

func TestMapEditor1118MalformedBoundsAndNoopAreAtomic(t *testing.T) {
	b := fixture1118(t)
	_, e, _ := open1118(t, b)
	d := e.Document()
	for _, cell := range []image.Point{image.Pt(-1, 4), image.Pt(32, 4), image.Pt(3, 32)} {
		if next, err := d.Edits.MoveUnit(0, cell); err == nil || next != nil || d.Edits.CanUndo() || !bytes.Equal(b, d.SourceBytes) {
			t.Fatal("out-of-bounds accepted or changed history")
		}
	}
	if _, err := d.Edits.MoveUnit(999, image.Pt(1, 1)); err == nil {
		t.Fatal("invalid file-order index accepted")
	}
	if next, err := d.Edits.MoveUnit(0, image.Pt(3, 4)); err != nil || next != nil || d.Edits.CanUndo() {
		t.Fatal("no-op recorded")
	}
	bad := filepath.Join(t.TempDir(), "bad.alm")
	if err := os.WriteFile(bad, []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(bad); err == nil || e.Document() != d {
		t.Fatal("malformed open was not atomic")
	}
}
