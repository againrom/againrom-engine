package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/text"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// townRoomRecordRooms are the town rooms the record visits per game.
var townRoomRecordRooms = map[bool][]string{false: {"square", "tavern", "school"}, true: {"square", "inn"}}

// townRoomRecordFixture is the inn record's fixture of one game, its clocks
// fixed and its pages' art loaded, shown in room.
func townRoomRecordFixture(t *testing.T, second, installed bool, room string) (*FrontEnd, *townScreen) {
	t.Helper()
	f, s := innRecordFixture(t, second)
	if installed {
		// An install resolves each named caption and its main slot from the
		// same line.
		w := &f.Words
		for slot, dst := range map[int][]*string{231: {&w.SchoolTrain}, 232: {&w.TavernExit, &w.SchoolExit}, 242: {&w.TavernTalk},
			257: {&w.TavernHired}, 258: {&w.TavernHire}, 259: {&w.TavernFire}} {
			w.Hover[slot] = fmt.Sprintf("slot-%d", slot)
			for _, d := range dst {
				*d = w.Hover[slot]
			}
		}
	}
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	if !second {
		a, err := LoadTownTavernArt(ROM1TownDescription(), townTavernSource())
		if err != nil {
			t.Fatal(err)
		}
		f.TownTavernArt = resolved(a, nil)
		s.room = map[string]townRoom{"square": roomSquare, "tavern": roomTavern, "school": roomSchool}[room]
		return f, s
	}
	s.draws = recordPageDraws{townDraws: s.draws, now: &now}
	d, _ := s.composedTown()
	for _, name := range append([]string{d.Square.Name}, recordPageRooms(d)...) {
		s.composed.art[d.Town+"/"+name] = &townRoomArt{scene: &town.Art{Frames: map[string][]image.Image{}}, page: recordPageArt(d.Room(name))}
	}
	s.state().room = secondTownRooms[room]
	if room == "inn" {
		s.enterSeats()
	}
	return f, s
}

// townRoomRecordControl is one named control of a shown room: its
// rectangle and the hit-test point that presses it.
type townRoomRecordControl struct {
	name string
	rect image.Rectangle
}

// townRoomRecordControls are the shown room's buttons and seats.
func townRoomRecordControls(s *townScreen, second bool) []townRoomRecordControl {
	var out []townRoomRecordControl
	if !second {
		if s.room != roomTavern && s.room != roomSchool {
			return nil
		}
		v := s.TownSurface()
		for i := range v.Buttons {
			out = append(out, townRoomRecordControl{fmt.Sprintf("button %d", i), TownRoomButtonRect(v.Kind, i)})
		}
		for i, seat := range v.Page.Seats {
			out = append(out, townRoomRecordControl{fmt.Sprintf("seat %d", i), seat.Rect})
		}
		return out
	}
	d, _ := s.composedTown()
	r := d.Room(s.roomName())
	if r == nil {
		return nil
	}
	for i, b := range r.Buttons {
		out = append(out, townRoomRecordControl{fmt.Sprintf("button %d %s", i, b.Name), b.Rect.Rectangle()})
	}
	view := s.roomView(s.roomName())
	p := view.page(r, s.roomArt(d, s.roomName()).page, s.in.Words, s.in.Font.Value())
	for i, seat := range p.Seats {
		out = append(out, townRoomRecordControl{fmt.Sprintf("seat %d", i), seat.Rect})
	}
	return out
}

// townRoomRecordHit is the hit test the live App runs at p, written as text.
func townRoomRecordHit(s *townScreen, second bool, p image.Point) string {
	if !second {
		if s.room != roomTavern && s.room != roomSchool {
			return "none"
		}
		c, ok := ui.TownSurfaceControlAt(s.TownSurface(), p)
		return fmt.Sprintf("%v/%v", c, ok)
	}
	v := s.TownSquareView()
	if v.Scene == nil {
		return "none"
	}
	c, ok := v.Scene.ControlAt(p)
	return fmt.Sprintf("%v/%v", c, ok)
}

// townRoomRecordPress runs the action the live App runs for a release at p.
func townRoomRecordPress(s *townScreen, second bool, p image.Point) ui.TownAction {
	if !second {
		c, ok := ui.TownSurfaceControlAt(s.TownSurface(), p)
		if !ok {
			return ui.TownAction{}
		}
		s.TownSurfacePress(c)
		return s.TownSurfaceClick(c, false)
	}
	v := s.TownSquareView()
	if v.Scene == nil {
		return ui.TownAction{}
	}
	c, ok := v.Scene.ControlAt(p)
	if !ok {
		return ui.TownAction{}
	}
	switch c.Kind {
	case ui.TownSquareControlDoor:
		return s.Choose(c.Door)
	case ui.TownSquareControlSelect:
		s.SelectTownSquare(c.Door)
	case ui.TownSquareControlButton:
		return s.TownSquareButton(c.Door)
	}
	return ui.TownAction{}
}

// townRoomRecordSelect selects seat i of the shown room the way a seat press
// does; a negative i leaves the entry selection.
func townRoomRecordSelect(s *townScreen, second bool, i int) {
	if i < 0 {
		return
	}
	if second {
		s.SelectTownSquare(i + 1)
		return
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}, false)
}

func townRoomRecord(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&out, format+"\n", args...) }
	state := func(key string, f *FrontEnd, s *townScreen) {
		t.Helper()
		party := make([]string, len(f.Carried))
		for i, p := range f.Carried {
			party[i] = p.ID
		}
		sav := "none"
		if raw, err := exportTownRecord(f); err != nil {
			sav = fmt.Sprintf("error %q", err)
		} else {
			sav = fmt.Sprintf("%d %x", len(raw), sha256.Sum256(raw))
		}
		cursor := s.talkPage()
		selected := fmt.Sprint(s.tavernSelection)
		if s.state() != nil && s.composed != nil {
			selected = fmt.Sprint(s.composed.seat)
		}
		line("%s room %q selected %s npc %d candidate %d dialogue %q text %q party %v gold %d queue %v SAV %s",
			key, s.roomName(), selected, s.npc, s.mercenaryTalk, cursor.key, s.dialogue.text, party, f.Town.Gold(), s.innQueue, sav)
	}
	frame := func(key string, s *townScreen) {
		t.Helper()
		var err error
		var m *image.RGBA
		calls := text.Record(func() { m, err = ui.ComposeTownScreenFrame(s, "", 0) })
		if err != nil {
			line("%s frame error %q", key, err)
			return
		}
		var glyphs strings.Builder
		for _, c := range calls {
			fmt.Fprintf(&glyphs, "%d,%d,%v,%v", c.X, c.Y, c.Color, c.Clip)
			if c.Glyph != nil {
				fmt.Fprintf(&glyphs, ",%d,%d,%v", c.Glyph.Width, c.Glyph.Height, c.Glyph.Pixels)
			}
			glyphs.WriteByte(';')
		}
		line("%s frame %x text %d %x", key, sha256.Sum256(m.Pix), len(calls), sha256.Sum256([]byte(glyphs.String())))
	}
	grid := func(key string, s *townScreen, second bool) {
		var hits strings.Builder
		for y := 0; y < 480; y += 4 {
			for x := 0; x < 640; x += 4 {
				fmt.Fprintf(&hits, "%s;", townRoomRecordHit(s, second, image.Pt(x, y)))
			}
		}
		line("%s hit grid %x", key, sha256.Sum256([]byte(hits.String())))
	}
	for _, run := range []struct{ second, installed bool }{{false, false}, {false, true}, {true, false}, {true, true}} {
		second, installed := run.second, run.installed
		game := map[bool]string{false: "rom1", true: "rom2"}[second] + map[bool]string{false: "", true: " installed"}[installed]
		for _, room := range townRoomRecordRooms[second] {
			_, probe := townRoomRecordFixture(t, second, installed, room)
			seats := 0
			for _, c := range townRoomRecordControls(probe, second) {
				if strings.HasPrefix(c.name, "seat ") {
					seats++
				}
			}
			for sel := -1; sel < seats; sel++ {
				key := fmt.Sprintf("%s %s select %d", game, room, sel)
				f, s := townRoomRecordFixture(t, second, installed, room)
				townRoomRecordSelect(s, second, sel)
				for i, r := range s.Rows() {
					line("%s row %d %q %v", key, i, r.Text, r.Choosable)
				}
				if !second && (s.room == roomTavern || s.room == roomSchool) {
					for i, b := range s.TownSurface().Buttons {
						line("%s caption %d %q %q %v", key, i, b.Label, b.Value, b.Enabled)
					}
				}
				state(key, f, s)
				frame(key, s)
				grid(key, s, second)
				controls := townRoomRecordControls(s, second)
				for _, c := range controls {
					centre := c.rect.Min.Add(c.rect.Max).Div(2)
					line("%s %s %v centre %s outside %s", key, c.name, c.rect, townRoomRecordHit(s, second, centre), townRoomRecordHit(s, second, c.rect.Max))
				}
				for _, c := range controls {
					g, gs := townRoomRecordFixture(t, second, installed, room)
					townRoomRecordSelect(gs, second, sel)
					a := townRoomRecordPress(gs, second, c.rect.Min.Add(c.rect.Max).Div(2))
					probe := key + " press " + c.name
					line("%s result %q info %v open %v", probe, a.Msg, a.Info, a.Open != nil)
					state(probe, g, gs)
				}
			}
			for i := -1; i <= len(probe.Rows()); i++ {
				f, s := townRoomRecordFixture(t, second, installed, room)
				a := s.Choose(i)
				key := fmt.Sprintf("%s %s choose %d", game, room, i)
				line("%s result %q info %v open %v", key, a.Msg, a.Info, a.Open != nil)
				state(key, f, s)
			}
		}
	}
	return out.String()
}

// TestTownRoomRecord records, for each game's town rooms over the inn
// record's fixtures, the rows, the captions, each control's rectangle and
// its hit at its centre and just outside, the whole hit map, the composed
// frame, and the action and SAV of each press and row choice.
func TestTownRoomRecord(t *testing.T) {
	got := townRoomRecord(t)
	path := filepath.Join("testdata", "town_room_record.txt")
	if os.Getenv("AGAINROM_WRITE_TOWN_ROOM_RECORD") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := 0; i < max(len(wl), len(gl)); i++ {
		w, g := "", ""
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			t.Fatalf("town room record line %d changed:\n got %s\nwant %s", i+1, g, w)
		}
	}
}
