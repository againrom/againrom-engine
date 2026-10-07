package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/data"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The window every message line witness plays in. The mission frame is one
// fixed size at every window size (DIV-210), so the message line holds the
// 1024x768 figures: 22 lines, wrapped at 864 pixels.
const (
	messageWitnessW = 640
	messageWitnessH = 480
)

// Every line the map's message line shows, reached through ordinary play in a
// 640x480 window on one installed root, from the owner's save of mission 40:
// an order onto a Sack of two of one Item and gold at (77,110), a single Item
// dropped from the pack bar and picked up again, one unit of a stack dropped
// and picked up again, and a skill raise by attack orders in mission 20. An
// Item line is main.txt global strings 85, 86 and 87, read here from the
// container, around the Item's name and its count after the take; the gold
// line is strings 88 and 89 around the amount (ITEM-PICKTEXT-145). Every
// drawn line stands at the message line's origin, one pitch below the line
// before it, in white, and its picture holds the font's own ink with a flat
// shadow one pixel down and right (MISSION-MSGLINE-056). The lines leave
// oldest first, one after another, each 3000 ms once it is the oldest
// (MISSION-MSGLINE-057, MISSION-MSGPOST-058).
func TestReleaseMessageLineDrawsInstalledPickupText(t *testing.T) {
	root := releaseFront(t)
	words := messageInstalledWords(t, root)

	openSave := func(t *testing.T) (*FrontEnd, *ui.App, *mapWorld, sim.EntityID) {
		t.Helper()
		_, raw := groundCorpusFile(t, "2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b")
		f := releaseFront(t)
		app, _ := openOriginalSAVAppAt(t, f, raw, "game0018.sav", messageWitnessW, messageWitnessH)
		live := f.live
		if size := live.view.FrameSize(); size != image.Pt(1024, 768) {
			t.Fatalf("the mission frame is %v at a %dx%d window, want the fixed 1024x768", size, messageWitnessW, messageWitnessH)
		}
		return f, app, live, live.mission.ids[0]
	}

	t.Run("two of one Item and gold", func(t *testing.T) {
		f, app, live, hero := openSave(t)
		sack, ok := live.sackAt(77, 110)
		if !ok || len(sack.Items) != 2 || sack.Items[0] != sack.Items[1] || sack.Gold == 0 {
			t.Fatalf("mission %d's sack at (77,110) = %+v, %v; want two of one Item and gold", live.mission.number, sack, ok)
		}
		code := sack.Items[0]
		var held uint32
		for _, s := range messageStacks(live, hero, code) {
			held += s.Count
		}
		messagePickUp(t, app, live, hero, 77, 110)
		got := messageStacks(live, hero, code)
		if len(got) != 1 || got[0].Count != held+2 {
			t.Fatalf("the take left %+v of the Item in the pack, want one Item of %d", got, held+2)
		}
		name := itemName(data.ItemCode(code), live.invParty.table)
		requireMessageLines(t, f, []string{
			words[3] + " " + strconv.FormatUint(uint64(sack.Gold), 10) + " " + words[4],
			messageItemLine(words, name, got[0].Count),
		})
		requireMessageExpiry(t, f, app, live, 2)
	})

	t.Run("a single Item", func(t *testing.T) {
		f, app, live, hero := openSave(t)
		if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		stacks, _ := live.world.CarriedStacks(hero)
		cell := slices.IndexFunc(stacks, func(s sim.ItemStack) bool {
			return s.Count == 1 && len(messageStacks(live, hero, s.Code)) == 1
		})
		if cell < 0 {
			t.Fatalf("the hero carries no Item of one that stands alone: %+v", stacks)
		}
		code := stacks[cell].Code
		e, _ := live.entity(hero)
		sx, sy := messageDropOne(t, app, live, cell, code, e.X-2, e.Y)
		if got := messageStacks(live, hero, code); len(got) != 0 {
			t.Fatalf("the drop left %+v in the pack, want none", got)
		}
		messagePickUp(t, app, live, hero, sx, sy)
		if got := messageStacks(live, hero, code); len(got) != 1 || got[0].Count != 1 {
			t.Fatalf("the take left %+v in the pack, want one Item of one", got)
		}
		name := itemName(data.ItemCode(code), live.invParty.table)
		requireMessageLines(t, f, []string{messageItemLine(words, name, 1)})
	})

	t.Run("a unit merged into a carried Item", func(t *testing.T) {
		f, app, live, hero := openSave(t)
		if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		stacks, _ := live.world.CarriedStacks(hero)
		cell := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Count > 1 })
		if cell < 0 {
			t.Fatalf("the hero carries no Item of two or more: %+v", stacks)
		}
		code, count := stacks[cell].Code, stacks[cell].Count
		e, _ := live.entity(hero)
		sx, sy := messageDropOne(t, app, live, cell, code, e.X-2, e.Y)
		if got := messageStacks(live, hero, code); len(got) != 1 || got[0].Count != count-1 {
			t.Fatalf("the drop left %+v in the pack, want one Item of %d", got, count-1)
		}
		t.Logf("one of %d dropped from pack cell %d lies at (%d,%d)", count, cell, sx, sy)
		messagePickUp(t, app, live, hero, sx, sy)
		if got := messageStacks(live, hero, code); len(got) != 1 || got[0].Count != count {
			t.Fatalf("the take left %+v in the pack, want the carried Item back at %d", got, count)
		}
		name := itemName(data.ItemCode(code), live.invParty.table)
		requireMessageLines(t, f, []string{messageItemLine(words, name, count)})
	})

	t.Run("gold and four Items", func(t *testing.T) {
		f, _, live, _ := openSave(t)
		codes := []data.ItemCode{data.QuestDocumentCode, 0xf70d, 0x810e, 0xfc1b}
		var reached []sim.ItemStack
		want := []string{words[3] + " 272 " + words[4]}
		for _, code := range codes {
			reached = append(reached, sim.PlainStack(uint16(code), 1))
			want = append(want, messageItemLine(words, itemName(code, live.invParty.table), 1))
		}
		if words[0] == "Picked up" {
			enNames := []string{"Valuable Documents", "Mage's Robe", "Wood Shaman Staff", "Mage's Shoes"}
			for i, n := range enNames {
				if want[i+1] != "Picked up "+n {
					t.Fatalf("row %d reads %q, want %q", i+1, want[i+1], "Picked up "+n)
				}
			}
		}
		view := f.Words
		announce(live.view, pickupLinesForTake(reached, 272, live.invParty.table, &view))
		requireMessageLines(t, f, want)
	})

	t.Run("skill raise", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		raw, err := f.Archives.Containers.ReadFile(MainTextPath)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(raw), "\r\n")
		before, after, _ := raiseSkillByOrders(t, f, false)
		slot := -1
		for s := range after.Skill {
			if after.Skill[s] != before.Skill[s] {
				slot = s
			}
		}
		if slot < 1 || slot > 5 || len(lines) <= 130+slot-1 {
			t.Fatalf("skills %v -> %v: want one weapon skill raised", before.Skill, after.Skill)
		}
		requireMessageLines(t, f, []string{lines[130+slot-1] + ": " + strconv.Itoa(int(after.Skill[slot]))})
	})
}

// messageInstalledWords is main.txt global strings 85 to 89 as the container
// holds them. It fails unless the front end's words are those lines and each
// draws inked glyphs.
func messageInstalledWords(t *testing.T, f *FrontEnd) [5]string {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(MainTextPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\r\n")
	if len(lines) <= 89 {
		t.Fatalf("main.txt holds %d lines, want the pickup words 85..89", len(lines))
	}
	want := [5]string{lines[85], lines[86], lines[87], lines[88], lines[89]}
	got := [5]string{f.Words.PickedUp, f.Words.PickedUpNow, f.Words.PickedUpPieces, f.Words.PickedUpGold, f.Words.PickedUpGoldUnit}
	font := f.Font.Value()
	shown := make([]string, len(want))
	for i := range want {
		if want[i] == "" || got[i] != want[i] {
			t.Fatalf("pickup word %d = %q, want installed main.txt[%d] %q", i, got[i], 85+i, want[i])
		}
		requireInkedNoticeGlyphs(t, font, want[i])
		shown[i] = messageShown(t, font, want[i])
	}
	t.Logf("selector %d: main.txt[85..89] %q", font.Selector, shown)
	return want
}

// messageItemLine is the documented Item line: string 85 and the name, and for
// a count above one ` (`, string 86, the count, string 87 and `)`, each part
// after one space.
func messageItemLine(words [5]string, name string, count uint32) string {
	if count > 1 {
		return fmt.Sprintf("%s %s (%s %d %s)", words[0], name, words[1], count, words[2])
	}
	return words[0] + " " + name
}

// messageStacks is the carried stacks of code in hero's pack.
func messageStacks(live *mapWorld, hero sim.EntityID, code uint16) []sim.ItemStack {
	stacks, _ := live.world.CarriedStacks(hero)
	var out []sim.ItemStack
	for _, s := range stacks {
		if s.Code == code {
			out = append(out, s)
		}
	}
	return out
}

// messageDropOne drags pack cell from the pack bar and releases it over the
// ground cell (x, y), steps frames until a sack holding code stands within four
// cells of there, and answers that sack's cell.
func messageDropOne(t *testing.T, app *ui.App, live *mapWorld, cell int, code uint16, x, y int32) (int32, int32) {
	t.Helper()
	px, py, err := app.HeadlessPackCellPoint(cell)
	if err != nil {
		t.Fatal(err)
	}
	gx, gy := messageGroundPixel(t, app, x, y)
	for _, ev := range []struct {
		edge string
		x, y int
	}{{"press", px, py}, {"move", gx, gy}, {"release", gx, gy}} {
		if err := app.HeadlessPointer(ev.edge, ev.x, ev.y); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n < 600; n++ {
		for _, s := range live.world.Sacks() {
			if dx, dy := s.X-x, s.Y-y; dx*dx+dy*dy <= 16 && slices.Contains(s.Items, code) {
				return s.X, s.Y
			}
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("no sack holding the dropped Item stands near (%d,%d)", x, y)
	return 0, 0
}

// messagePickUp selects hero, presses and releases the primary button on a
// window pixel over the ground cell (x, y), and steps frames until the sack
// standing there is gone. An open notice is closed with Enter first.
func messagePickUp(t *testing.T, app *ui.App, live *mapWorld, hero sim.EntityID, x, y int32) {
	t.Helper()
	closeNotices := func() {
		for i := 0; i < 16 && app.HeadlessNoticeOpen(); i++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
	}
	closeNotices()
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	px, py := messageGroundPixel(t, app, x, y)
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, px, py); err != nil {
			t.Fatal(err)
		}
	}
	if !live.pickup.set || live.pickup.id != hero || live.pickup.x != x || live.pickup.y != y {
		t.Fatalf("a click on (%d,%d) armed %+v, want a pickup order for %d", x, y, live.pickup, hero)
	}
	for n := 0; n < 3000; n++ {
		if _, standing := live.sackAt(x, y); !standing {
			return
		}
		closeNotices()
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("the sack at (%d,%d) still stands after 3000 frames", x, y)
}

// messageGroundPixel is a window pixel away from the view's edges whose ground
// cell is (x, y).
func messageGroundPixel(t *testing.T, app *ui.App, x, y int32) (int, int) {
	t.Helper()
	for py := messageWitnessH * 13 / 100; py < messageWitnessH*73/100; py += 2 {
		for px := messageWitnessW * 16 / 100; px < messageWitnessW*73/100; px += 2 {
			if cx, cy, err := app.HeadlessDropCell(px, py); err == nil && cx == int(x) && cy == int(y) {
				return px, py
			}
		}
	}
	t.Fatalf("no window pixel answers ground cell (%d,%d)", x, y)
	return 0, 0
}

// requireMessageLines fails unless the message line ends with want and every
// line it holds stands where the drawing puts it: the list's picture at frame
// pixel (8, 8), each line's pen one pitch of 17 below the one before, in white
// for 3000 ms, and the picture holds exactly the ink the map font paints for
// those lines, each with the flat shadow behind it.
func requireMessageLines(t *testing.T, f *FrontEnd, want []string) {
	t.Helper()
	font := f.Font.Value()
	if font.Height() != 15 {
		t.Fatalf("the map font's cells are %d high, want font1's 15", font.Height())
	}
	pic, at, draws, ok := f.live.view.MessageLog()
	lines := f.live.view.MessageLines()
	if !ok || len(draws) < len(want) || len(draws) != len(lines) {
		t.Fatalf("the message line holds %+v, want its last lines %q", lines, want)
	}
	if at != image.Pt(8, 8) {
		t.Fatalf("the message line's picture is at %v, want (8,8)", at)
	}
	for i, w := range want {
		if got := draws[len(draws)-len(want)+i].Text; got != w {
			t.Fatalf("drawn line %d reads %q, want %q", len(draws)-len(want)+i, messageShown(t, font, got), messageShown(t, font, w))
		}
	}
	for i, d := range draws {
		if d.Pen != image.Pt(8, 8+17*i) || d.Ink != ui.MessageWhite || lines[i].Life != 3*time.Second {
			t.Fatalf("line %d %q: pen %v, ink %d, life %v; want pen %v, white, 3s", i, messageShown(t, font, d.Text), d.Pen, d.Ink, lines[i].Life, image.Pt(8, 8+17*i))
		}
		requireInkedNoticeGlyphs(t, font, d.Text)
		t.Logf("line %d at %v: %q", i, d.Pen, messageShown(t, font, d.Text))
	}
	requireMessagePixels(t, font, pic, at, draws)
	writeMessageWitness(t, pic, at, len(draws))
}

func writeMessageWitness(t *testing.T, pic *image.RGBA, at image.Point, lines int) {
	t.Helper()
	dir := os.Getenv("AGAINROM_PICKUP_ROWS_WITNESS_DIR")
	if dir == "" {
		return
	}
	out := image.NewRGBA(image.Rect(0, 0, at.X+pic.Bounds().Dx()+8, at.Y+pic.Bounds().Dy()+8))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	draw.Draw(out, pic.Bounds().Add(at), pic, pic.Bounds().Min, draw.Over)
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()) + "-" + strconv.Itoa(lines) + "rows.png"
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, out); err != nil {
		t.Fatal(err)
	}
}

// requireMessagePixels compares the picture with what the map font paints for
// the drawn lines. Where a line's text paints, the pixel is opaque grey on the
// white ramp, 17 per level; where only its shadow, one pixel down and right,
// paints, the pixel is 8 in every channel; everywhere else in the line's
// pitch it is transparent. A line has both kinds of pixel, and a full-level
// one.
func requireMessagePixels(t *testing.T, font *text.Font, pic *image.RGBA, at image.Point, draws []ui.MessageDraw) {
	t.Helper()
	shadow := color.RGBA{R: 8, G: 8, B: 8, A: 255}
	for i, d := range draws {
		pen := d.Pen.Sub(at)
		inked, shade, bright := 0, 0, false
		for y := pen.Y; y < min(pen.Y+17, pic.Bounds().Max.Y); y++ {
			for x := pic.Bounds().Min.X; x < pic.Bounds().Max.X; x++ {
				got := pic.RGBAAt(x, y)
				switch {
				case font.PaintedAt(d.Text, pen.X, pen.Y, image.Pt(x, y)):
					if got.A != 255 || got.R != got.G || got.G != got.B || got.R%17 != 0 {
						t.Fatalf("line %d: text pixel (%d,%d) is %v, want opaque grey on the 17-per-level ramp", i, x, y, got)
					}
					inked++
					bright = bright || got.R == 255
				case font.PaintedAt(d.Text, pen.X+1, pen.Y+1, image.Pt(x, y)):
					if got != shadow {
						t.Fatalf("line %d: shadow pixel (%d,%d) is %v, want %v", i, x, y, got, shadow)
					}
					shade++
				default:
					if got.A != 0 {
						t.Fatalf("line %d: pixel (%d,%d) outside the text and its shadow is %v, want transparent", i, x, y, got)
					}
				}
			}
		}
		if inked == 0 || shade == 0 || !bright {
			t.Fatalf("line %d: %d text pixels, %d shadow pixels, full level %v; want ink, shadow and a full-level pixel", i, inked, shade, bright)
		}
		t.Logf("line %d: %d text pixels on the white ramp, %d shadow pixels of %v", i, inked, shade, shadow)
	}
}

// requireMessageExpiry steps ordinary frames until the last n lines of the
// message line, all posted by the take just made, have left it, and answers
// how they left. Each frame is one 20 ms tick, and play goes on: another line
// may join at the end meanwhile. The take's own frame counted 20 ms, so a line
// posted into an empty list leaves at the 150th frame after it, when 3020 ms
// is the first count above 3000 ms. The next line is counted from zero when
// that one leaves and so leaves 151 frames later, and every line leaves after
// the line posted before it (MISSION-MSGLINE-057).
func requireMessageExpiry(t *testing.T, f *FrontEnd, app *ui.App, live *mapWorld, n int) {
	t.Helper()
	held := live.view.MessageLines()
	watch := held[len(held)-n:]
	alone := len(held) == n
	texts := func(lines []ui.MessageLine) []string {
		out := make([]string, len(lines))
		for i, ln := range lines {
			out[i] = ln.Text
		}
		return out
	}
	names := texts(watch)
	for i, name := range names {
		if slices.Contains(names[:i], name) {
			t.Fatalf("the watched lines repeat %q", name)
		}
	}
	gone := make([]int, n)
	for frame := 1; slices.Contains(gone, 0); frame++ {
		if frame > (len(held)+1)*151 {
			t.Fatalf("lines %q are still on the message line after %d frames, gone at %v", names, frame-1, gone)
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		present := texts(live.view.MessageLines())
		for i, name := range names {
			if gone[i] == 0 && !slices.Contains(present, name) {
				gone[i] = frame
			}
		}
	}
	if alone && gone[0] != 150 {
		t.Fatalf("the first line, posted into an empty list, left at frame %d, want 150 (%v)", gone[0], gone)
	}
	for i := 1; i < n; i++ {
		if gap := gone[i] - gone[i-1]; gap != 151 {
			t.Fatalf("line %d left %d frames after line %d (%v), want 151", i+1, gap, i, gone)
		}
	}
	shown := make([]string, n)
	for i, name := range names {
		shown[i] = messageShown(t, f.Font.Value(), name)
	}
	t.Logf("lines %q left oldest first at frames %v after the take (posted into an empty list: %v)", shown, gone, alone)
}

// messageShown is s as a reader sees it: CP866 decoded under the converting
// selector, the bytes themselves otherwise.
func messageShown(t *testing.T, font *text.Font, s string) string {
	t.Helper()
	if font.Selector != text.SelectorConverting {
		return s
	}
	shown, err := charmap.CodePage866.NewDecoder().String(s)
	if err != nil {
		t.Fatal(err)
	}
	return shown
}
