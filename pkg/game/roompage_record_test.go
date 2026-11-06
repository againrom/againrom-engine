package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/text"
	"againrom/pkg/screenart"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

func TestRoomPageRecord(t *testing.T) {
	var out strings.Builder
	line := func(f string, a ...any) { fmt.Fprintf(&out, f+"\n", a...) }
	f := shellFrontEnd()
	f.Font = resolved(missionFont(), nil)
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	s := f.townUI
	a, err := LoadTownTavernArt(ROM1TownDescription(), townTavernSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownTavernArt = resolved(a, nil)
	for _, room := range []townRoom{roomTavern, roomSchool} {
		s.room = room
		for _, phase := range []int{0, 1, 2, 5} {
			v := s.TownSurface()
			v.AnimationFrame = phase
			key := fmt.Sprintf("rom1/%s/phase-%d", townRoomName(room), phase)
			for i, c := range v.Cells {
				line("%s cell %d %s %q %q %q %v %v %v frames %d", key, i, c.Key, c.Label, c.Detail, c.Price, c.Enabled, c.Selected, c.Hired, len(c.Frames))
			}
			line("%s candidate %v %v", key, v.Candidate.HasSubject, ui.PanelStatement(ui.AuthoredPanelLayout(), v.Candidate.Subject))
			recordPageLayout(line, key, v.Page)
			for i, b := range v.Buttons {
				r := TownRoomButtonRect(v.Kind, i)
				line("%s button %d %q %q %v %v", key, i, b.Label, b.Value, b.Enabled, r)
				for _, p := range recordPageEdges(r) {
					v.Hover, v.HasHover = p, true
					c, ok := ui.TownSurfaceControlAt(v, p)
					line("%s at %v %v %v pixels %s", key, p, c, ok, recordPagePixels(func() *image.RGBA { return ui.ComposeTownSurface(v) }))
				}
			}
		}
	}
	for _, name := range matrixDescriptions {
		s, d, _ := matrixScreen(t, name)
		now = time.Unix(100, 0)
		s.draws = recordPageDraws{townDraws: s.draws, now: &now}
		for _, room := range append([]string{d.Square.Name}, recordPageRooms(d)...) {
			spec := d.Scene(room)
			if spec == nil {
				continue
			}
			scene := s.scene(d, room)
			key := name + "/" + room
			dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
			scene.Paint(dst, "")
			line("%s scene %v %x", key, scene.Size(), sha256.Sum256(dst.Pix))
			if name == "rom1-square" {
				continue
			}
			r := d.Room(room)
			page := recordPageArt(r)
			s.composed.art[d.Town+"/"+room] = &townRoomArt{scene: &town.Art{Frames: map[string][]image.Image{}}, page: page}
			view := roomPageView{t: s, d: d, room: room}
			line("%s paint %s", key, recordPagePixels(func() *image.RGBA { m := image.NewRGBA(image.Rect(0, 0, 640, 480)); view.Paint(m); return m }))
			if r == nil {
				continue
			}
			if r.Seats != nil {
				s.enterSeats()
				line("%s enter seat %d frames %v", key, s.composed.seat, s.composed.seatFrames)
				for _, delta := range []time.Duration{0, 125 * time.Millisecond, 126 * time.Millisecond, 126 * time.Millisecond} {
					now = now.Add(delta)
					s.composed.seatStamp = now.Add(-delta)
					// A fixed host keeps the exact boundary independent of wall time.
					s.advanceSeats(scene, r.Seats, page.Pictures(r.Seats.Actor))
					line("%s step %s seat %d frames %v", key, delta, s.composed.seat, s.composed.seatFrames)
				}
				p := view.page(r, page, s.in.Words, s.in.Font.Value())
				recordPageLayout(line, key, p)
				for i := range p.Seats {
					s.composed.seat = i
					patron := view.selectedPatron(r, page)
					line("%s person %d known %v lines %v", key, i, patron.known, ui.PanelStatement(ui.AuthoredPanelLayout(), patron.subject))
				}
			}
			for i, b := range r.Buttons {
				line("%s button %d %s %v", key, i, b.Name, b.Rect.Rectangle())
				for _, p := range recordPageEdges(b.Rect.Rectangle()) {
					s.composed.pointer = p
					c, ok := view.ControlAt(p)
					tip, tipOK := view.TipAt(p)
					line("%s at %v %v %v tip %d %v pixels %s", key, p, c, ok, tip, tipOK, recordPagePixels(func() *image.RGBA { m := image.NewRGBA(image.Rect(0, 0, 640, 480)); view.Paint(m); return m }))
				}
			}
		}
	}
	recordFirstSleeps(t, line, 0, true)
	for _, id := range []int{1, 2, 3} {
		recordSecondSleeps(t, line, id, 0, true)
	}
	path := filepath.Join("testdata", "room_page_record.txt")
	if os.Getenv("AGAINROM_WRITE_ROOM_RECORD") == "1" {
		if err := os.WriteFile(path, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w, g := strings.Split(string(want), "\n"), strings.Split(out.String(), "\n")
	for i := 0; i < max(len(w), len(g)); i++ {
		if i >= len(w) || i >= len(g) {
			t.Fatalf("room record lengths %d / %d", len(g), len(w))
		}
		if w[i] != g[i] {
			t.Fatalf("room record line %d:\n got %s\nwant %s", i+1, g[i], w[i])
		}
	}
}

func recordPageRooms(d *town.Description) []string {
	var names []string
	for _, r := range d.Rooms {
		names = append(names, r.Name)
	}
	return names
}

func recordPageEdges(r image.Rectangle) []image.Point {
	return []image.Point{r.Min, r.Max.Sub(image.Pt(1, 1)), r.Min.Sub(image.Pt(1, 0)), r.Max, r.Min.Add(r.Max).Div(2)}
}

func recordPagePixels(paint func() *image.RGBA) string {
	var m *image.RGBA
	calls := text.Record(func() { m = paint() })
	var glyphs strings.Builder
	for _, c := range calls {
		fmt.Fprintf(&glyphs, "%d,%d,%v,%v;", c.X, c.Y, c.Color, c.Clip)
	}
	return fmt.Sprintf("%x text %x", sha256.Sum256(m.Pix), sha256.Sum256([]byte(glyphs.String())))
}

func recordPageLayout(line func(string, ...any), key string, p ui.RoomPage) {
	line("%s page phase %d seats %d figure %v %v %v", key, p.Phase, len(p.Seats), p.Person.Figure.Rect, p.Person.Figure.Offset, p.Person.Figure.Centre)
	for i, seat := range p.Seats {
		line("%s seat %d %v", key, i, seat.Rect)
	}
	if p.Person.Card != nil {
		line("%s card %v %v %v", key, p.Person.Card.PaneRect, p.Person.Card.CardOffset, p.Person.Card.Statistics)
	}
}

func recordPageArt(r *town.RoomSpec) *screenart.Set {
	a := &screenart.Set{Frames: map[string][]image.Image{}}
	if r == nil {
		return a
	}
	for i, row := range r.Art {
		w, h := 48, 64
		if row.Size != nil {
			w, h = row.Size[0], row.Size[1]
		}
		if w <= 0 || h <= 0 {
			w, h = 48, 64
		}
		a.Frames[row.Name] = []image.Image{roomPageTestPicture(w, h, color.RGBA{R: uint8(i*23 + 1), G: uint8(i*11 + 2), B: uint8(i*7 + 3), A: 255})}
	}
	if r.Seats != nil {
		a.Frames[r.Seats.Actor] = []image.Image{roomPageTestPicture(48, 64, color.RGBA{R: 10, A: 255}), roomPageTestPicture(48, 64, color.RGBA{R: 20, A: 255}), roomPageTestPicture(48, 64, color.RGBA{R: 30, A: 255})}
	}
	return a
}

type recordPageDraws struct {
	townDraws
	now *time.Time
}

func (d recordPageDraws) animationClock() func() time.Time { return func() time.Time { return *d.now } }
