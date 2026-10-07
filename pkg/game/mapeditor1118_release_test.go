package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Raw node parameters identify placed-unit references independently of the
// inspector's caption and target builder. Bad/absent scripts are not selected.
func rawReferencedUnits1118(b []byte) map[uint32]bool {
	refs := map[uint32]bool{}
	off := 0
	for family := 0; family < 2; family++ {
		if off+4 > len(b) {
			return refs
		}
		n := int(binary.LittleEndian.Uint32(b[off:]))
		off += 4
		if n > (len(b)-off)/796 {
			return refs
		}
		for i := 0; i < n; i++ {
			for j := 0; j < 10; j++ {
				if binary.LittleEndian.Uint32(b[off+0x74+4*j:]) == 4 {
					refs[binary.LittleEndian.Uint32(b[off+0x4c+4*j:])] = true
				}
			}
			off += 796
		}
	}
	return refs
}

func assertUnitRefs1118(t *testing.T, d *ui.InspectionDocument, id uint32, cell image.Point) {
	t.Helper()
	want := fmt.Sprintf("Unit ID %d", id)
	count := 0
	for _, r := range d.Records {
		for _, ref := range r.References {
			if strings.HasPrefix(ref.Label, want+" /") {
				if len(ref.Targets) != 1 || ref.Targets[0].Cell != cell {
					t.Fatalf("stale %s target in %s:%d: %+v", want, r.Kind, r.Index, ref.Targets)
				}
				count++
			}
		}
	}
	if count == 0 {
		t.Fatalf("raw referenced unit %d has no displayed references", id)
	}
}

func TestReleaseMapEditor1118MoveUndoRedoSaveAsReopen(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	x, err := NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	if width := x.font.Advance("Ctrl+Z / Ctrl+Y / Ctrl+S"); width > 300 {
		t.Fatalf("installed shortcut hint exceeds rail: %d", width)
	}
	seen := map[bool]bool{}
	for _, entry := range x.Maps {
		if seen[entry.FromArchive] {
			continue
		}
		key := "loose/" + entry.Source
		if entry.FromArchive {
			key = "scenario/" + entry.Source
		}
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		b := append([]byte(nil), e.Document().SourceBytes...)
		sections := rawSections1092(t, b)
		units := sections[6]
		refs := rawReferencedUnits1118(sections[7])
		idCount := map[uint32]int{}
		for off := 0; off+70 <= len(units); off += 70 {
			idCount[uint32(binary.LittleEndian.Uint16(units[off+0x40:]))]++
		}
		chosen := -1
		var from image.Point
		var id uint32
		for i := 0; i*70+70 <= len(units); i++ {
			raw := units[i*70:]
			id = uint32(binary.LittleEndian.Uint16(raw[0x40:]))
			from = image.Pt(int(binary.LittleEndian.Uint32(raw)>>8), int(binary.LittleEndian.Uint32(raw[4:])>>8))
			r := inspectRecord1093(t, e.Document(), "Unit", i)
			if (!entry.FromArchive || refs[id] && idCount[id] == 1) && r.Preview != nil && from.X > 3 && from.Y > 3 && from.X+4 < e.Document().Width && from.Y+4 < e.Document().Height {
				chosen = i
				break
			}
		}
		if chosen < 0 {
			continue
		}
		seen[entry.FromArchive] = true
		hasReference := refs[id] && idCount[id] == 1
		t.Run(key, func(t *testing.T) {
			e.Layout(1280, 800)
			recordIndex := -1
			for i, r := range e.Document().Records {
				if r.Kind == "Unit" && r.Index == chosen {
					recordIndex = i
				}
			}
			if hasReference {
				assertUnitRefs1118(t, e.Document(), id, from)
			}
			// Actual displaced editor view; raw height-plane projection, not
			// MapInspector's decoded positions, supplies pointer coordinates.
			proj := terrain.Project(sections[2], e.Document().Width, e.Document().Height)
			point := func(cell image.Point) image.Point {
				a, b := proj.WorldCorner(cell.X, cell.Y)
				c, d := proj.WorldCorner(cell.X+1, cell.Y+1)
				sx, sy := e.Document().Viewer.Camera().WorldToScreen(float64((a+c)/2), float64((b+d)/2))
				return image.Pt(int(sx), int(sy))
			}
			left, top := proj.WorldCorner(from.X, from.Y)
			right, bottom := proj.WorldCorner(from.X+1, from.Y+1)
			cam := e.Document().Viewer.Camera()
			cam.SetZoom(1)
			cam.CenterOn(float64((left+right)/2), float64((top+bottom)/2))
			// Centre from raw geometry, then select through the actual pointer
			// door. No index-selection API preselects the unit for this drive.
			now := time.Unix(1, 0)
			pick := point(from)
			e.Pointer(ui.Input{CursorX: pick.X, CursorY: pick.Y, PrimaryDown: true}, now)
			e.Pointer(ui.Input{CursorX: pick.X, CursorY: pick.Y}, now)
			if e.Selected() != recordIndex {
				t.Fatalf("pointer selected record %d, want unit record %d", e.Selected(), recordIndex)
			}
			to := from.Add(image.Pt(3, 2))
			before, note, err := e.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			a, target := point(from), point(to)
			e.Pointer(ui.Input{CursorX: a.X, CursorY: a.Y, PrimaryDown: true}, now)
			e.Pointer(ui.Input{CursorX: target.X, CursorY: target.Y, PrimaryDown: true}, now)
			if e.Dirty() || !bytes.Equal(b, e.Document().SourceBytes) {
				t.Fatal("preview changed installed bytes before release")
			}
			e.Pointer(ui.Input{CursorX: target.X, CursorY: target.Y}, now)
			changed := append([]byte(nil), e.Document().SourceBytes...)
			movedBytes1118(t, b, changed, chosen, to)
			if hasReference {
				assertUnitRefs1118(t, e.Document(), id, to)
			}
			if e.Selected() != recordIndex || e.Document().Records[recordIndex].Cell != to || !e.Dirty() {
				t.Fatal("live editor selection/placement failed")
			}
			raw := changed[spans1118(t, changed)[6]+chosen*70:]
			text := fmt.Sprintf("X/Y raw: %d / %d", binary.LittleEndian.Uint32(raw), binary.LittleEndian.Uint32(raw[4:]))
			if !strings.Contains(inspectText1093(e.Document().Records[recordIndex]), text) {
				t.Fatal("installed coordinate detail is stale")
			}
			after, _, err := e.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			gold := color.RGBA{0xb8, 0x95, 0x4f, 255}
			if after.RGBAAt(target.X, target.Y) != gold || bytes.Equal(before.Pix, after.Pix) {
				t.Fatal("actual composed frame lacks moved selection marker")
			}
			_ = e.Key("undo")
			if !bytes.Equal(b, e.Document().SourceBytes) || e.Dirty() {
				t.Fatal("installed Undo was not exact")
			}
			undo, _, err := e.HeadlessFrame()
			if err != nil {
				t.Fatalf("Undo did not restore actual composed frame: %v", err)
			}
			// The Redo button is now enabled; the canvas itself must return
			// byte-identically, including the unit and selected marker.
			w := e.Document().Viewer.ViewportSize().X
			for y := 0; y < 800; y++ {
				if !bytes.Equal(before.Pix[y*before.Stride:y*before.Stride+w*4], undo.Pix[y*undo.Stride:y*undo.Stride+w*4]) {
					t.Fatalf("Undo did not restore canvas row %d", y)
				}
			}
			_ = e.Key("redo")
			if !bytes.Equal(changed, e.Document().SourceBytes) {
				t.Fatal("installed Redo was not exact")
			}
			out := filepath.Join(t.TempDir(), "edited.alm")
			_ = e.Key("saveas")
			e.TextInput(filepath.Join(t.TempDir(), "long explicit output folder that does not exist", "edited-map-with-a-long-name.alm"))
			prompt, _, err := e.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			if err := e.Key("enter"); err == nil {
				t.Fatal("missing output directory unexpectedly accepted")
			}
			refused, _, err := e.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			_ = e.Key("escape")
			_ = e.Key("saveas")
			e.TextInput(out)
			if err := e.Key("enter"); err != nil || e.Dirty() {
				t.Fatalf("installed explicit Save As: %v", err)
			}
			written, err := os.ReadFile(out)
			if err != nil || !bytes.Equal(changed, written) {
				t.Fatalf("written bytes differ: %v", err)
			}
			fresh := x.Editor()
			if err := fresh.Open(out); err != nil || fresh.Dirty() || fresh.Document().Edits.CanUndo() || !bytes.Equal(changed, fresh.Document().SourceBytes) {
				t.Fatalf("fresh reopened document differs: %v", err)
			}
			if hasReference {
				assertUnitRefs1118(t, fresh.Document(), id, to)
			}
			fresh.Layout(1280, 800)
			fresh.Select(recordIndex, false)
			*fresh.Document().Viewer.Camera() = *e.Document().Viewer.Camera()
			reopened, _, err := fresh.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			for y := 0; y < 800; y++ {
				if !bytes.Equal(after.Pix[y*after.Stride:y*after.Stride+w*4], reopened.Pix[y*reopened.Stride:y*reopened.Stride+w*4]) {
					t.Fatalf("fresh reopen canvas differs at row %d", y)
				}
			}
			again, err := x.Open(key)
			if err != nil || !bytes.Equal(b, again.SourceBytes) {
				t.Fatal("source installed map changed")
			}
			if dir := os.Getenv("AGAINROM_EDITOR_CAPTURE"); dir != "" && entry.FromArchive {
				if err := refuseOriginalWriteTarget(dir, root); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				for name, pic := range map[string]*image.RGBA{"before": before, "moved": after, "prompt": prompt, "refused": refused, "reopened": reopened} {
					f, err := os.Create(filepath.Join(dir, name+".png"))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, pic)
					closeErr := f.Close()
					if err != nil || closeErr != nil {
						t.Fatalf("capture: %v %v", err, closeErr)
					}
				}
			}
			t.Logf("unit file index=%d id=%d %v->%v; all %d bytes checked; %s", chosen, id, from, to, len(b), note)
		})
	}
	if !seen[true] || !seen[false] {
		t.Fatalf("missing installed source routes: %v", seen)
	}
}
