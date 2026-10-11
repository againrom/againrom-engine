package game

import (
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondTownOne opens a new second-game campaign at town 1 on the installed
// root, under a synthetic animation clock the caller advances.
func secondTownOne(t *testing.T) (*FrontEnd, *ui.App, *secondCampaignScreen, *time.Time) {
	t.Helper()
	f := secondGameFront(t)
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	app := f.App("second town one")
	app.Layout(640, 480)
	s := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(s)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if !s.AtTownSquare() {
		d, _ := s.composedTown()
		t.Fatalf("town 1 is not composed: %v", s.roomArt(d, "square").err)
	}
	s.TownSquareActive(true)
	return f, app, s, &now
}

// secondTownStep admits one square step: the clock moves past the 67 ms
// period and the screen paints once (R2-ENGINE-242).
func secondTownStep(s *secondCampaignScreen, now *time.Time) {
	*now = now.Add(68 * time.Millisecond)
	s.AdvanceTownSquareAnimation()
}

// secondTownPoint is the first mask cell, in row order, holding byte b.
func secondTownPoint(t *testing.T, s *secondCampaignScreen, b byte) image.Point {
	t.Helper()
	d, _ := s.composedTown()
	m := s.roomArt(d, "square").scene.Mask
	for i, v := range m.Pix {
		if v == b {
			return image.Pt(i%m.Stride, i/m.Stride)
		}
	}
	t.Fatalf("no mask cell holds byte %d", b)
	return image.Point{}
}

// The installed mask holds the nine positive selector bytes in exactly the
// counts R2-ENGINE-241 measured, 150 distinct indices, and 267384 cells of
// every other index; each mapped byte reaches the hotspot, tip and control
// the description gives it, and the four small figure bytes reach none.
func TestReleaseSecondTownOneMaskCensus(t *testing.T) {
	_, _, s, _ := secondTownOne(t)
	d, _ := s.composedTown()
	m := s.roomArt(d, "square").scene.Mask
	if m.Rect != image.Rect(0, 0, 640, 480) {
		t.Fatalf("mask %v", m.Rect)
	}
	counts := map[byte]int{}
	for _, v := range m.Pix {
		counts[v]++
	}
	want := map[byte]int{32: 17, 64: 18, 80: 4, 96: 6, 128: 5764, 144: 5771, 160: 7911, 176: 5592, 192: 14733}
	others := 0
	for b, n := range counts {
		if _, ok := want[b]; !ok {
			others += n
		}
	}
	for b, n := range want {
		if counts[b] != n {
			t.Errorf("byte %d: %d cells, want %d", b, counts[b], n)
		}
	}
	if len(counts) != 150 || others != 267384 {
		t.Errorf("%d distinct indices and %d other cells, want 150 and 267384", len(counts), others)
	}
	view := s.TownSquareView().Scene
	type answer struct {
		hotspot string
		tip     int
		control ui.TownSquareControlKind
		door    int
		acts    bool
	}
	for b, w := range map[byte]answer{
		128: {"inn", 236, ui.TownSquareControlDoor, 0, true},
		144: {"shop", 233, 0, 0, false},
		160: {"gate", 237, ui.TownSquareControlDoor, 1, true},
		176: {"menu", 235, ui.TownSquareControlMenu, 0, true},
		192: {"school", 234, 0, 0, false},
		32:  {}, 64: {}, 80: {}, 96: {},
	} {
		p := secondTownPoint(t, s, b)
		h := s.scene(d, "square").HotspotAt(p)
		name := ""
		if h != nil {
			name = h.Name
		}
		tip, tipped := view.TipAt(p)
		c, acts := view.ControlAt(p)
		if name != w.hotspot || tipped != (w.hotspot != "") || tipped && tip != w.tip || acts != w.acts ||
			acts && (c.Kind != w.control || c.Door != w.door) {
			t.Errorf("byte %d at %v: hotspot %q tip %d/%v control %+v/%v, want %+v", b, p, name, tip, tipped, c, acts, w)
		}
	}
	t.Logf("mask: %d distinct, others %d; counts %v", len(counts), others, want)
}

// The square draws its layers in the order and at the destinations
// R2-ENGINE-240 measured, from the keys its loader names.
func TestReleaseSecondTownOneLayers(t *testing.T) {
	_, _, s, _ := secondTownOne(t)
	d, _ := s.composedTown()
	type layer struct {
		name string
		at   image.Point
	}
	want := []layer{
		{"base", image.Pt(0, 0)}, {"birds", image.Pt(0, 0)}, {"overlay", image.Pt(0, 0)},
		{"label-shop", image.Pt(264, 264)}, {"label-inn", image.Pt(144, 332)}, {"tavern", image.Pt(124, 312)},
		{"sign", image.Pt(360, 232)}, {"door", image.Pt(180, 148)}, {"star", image.Pt(340, 288)},
		{"shopie", image.Pt(276, 296)}, {"fluger", image.Pt(308, 64)}, {"guard", image.Pt(184, 158)},
		{"horse", image.Pt(0, 0)}, {"baba", image.Pt(0, 0)}, {"dervish", image.Pt(0, 0)},
	}
	if len(d.Layers) != len(want) {
		t.Fatalf("%d layers, want %d", len(d.Layers), len(want))
	}
	for i, l := range d.Layers {
		name := l.Art
		if name == "" {
			name = l.Actor
		}
		if got := (layer{name, l.At.Pt()}); got != want[i] {
			t.Errorf("layer %d = %+v, want %+v", i, got, want[i])
		}
	}
	keys := map[string]string{}
	for _, a := range d.Art {
		keys[a.Name] = strings.ToLower(a.Key)
	}
	for name, key := range map[string]string{
		"base": "townmain.bmp", "overlay": "town_add.bmp", "label-shop": "shop_l.bmp", "label-inn": "tavern_l.bmp",
	} {
		if !strings.HasSuffix(keys[name], "interface/town/"+key) {
			t.Errorf("%s reads %q, want interface/town/%s", name, keys[name], key)
		}
	}
	art := s.roomArt(d, "square").scene
	for name, n := range map[string]int{"sign": 10, "door": 9, "stars": 9, "fluger": 8, "guard": 8} {
		if got := len(art.Pictures(name)); got != n {
			t.Errorf("%s: %d frames, want %d", name, got, n)
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	s.TownSquareView().Scene.Paint(dst)
	base := art.Pictures("base")[0]
	// (600,450) is outside every patch and every wildlife position.
	if r, g, b, _ := base.At(600, 450).RGBA(); [3]uint32{r, g, b} != func() [3]uint32 {
		r, g, b, _ := dst.At(600, 450).RGBA()
		return [3]uint32{r, g, b}
	}() {
		t.Error("the painted square does not show its base picture outside the patches")
	}
}

// Hover and click per selector: the school shows tip 234 and does nothing;
// the shop opens no room; the closed gate shows the plagatguard section and
// keeps the party in town (R2-ENGINE-241, R2-SESSION-128).
func TestReleaseSecondTownOneSelectors(t *testing.T) {
	f, app, s, now := secondTownOne(t)
	d, _ := s.composedTown()
	sq := s.scene(d, "square")
	c := f.Town.second
	for _, name := range []string{"school", "shop"} {
		b := map[string]byte{"school": 192, "shop": 144}[name]
		s.TownSquarePointer(secondTownPoint(t, s, b))
		secondTownStep(s, now)
		if sq.Hover() == nil || sq.Hover().Name != name {
			t.Fatalf("%s hover not delivered", name)
		}
		sq.Click(sq.Hover())
		if c.room != secondTownSquare || c.current != (secondLocation{2, 1}) || c.payload != nil {
			t.Errorf("%s click acted: room %v current %v", name, c.room, c.current)
		}
	}
	if c.bank[769] != 0 {
		t.Fatal("a new campaign starts with the gate flag set")
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	body, open := s.dialogueBody()
	want, err := readSecondTownTalk(&f.InstallResources, secondGateText)
	if err != nil || want == nil {
		t.Fatalf("installed %s section: %v", secondGateText, err)
	}
	if !open || c.current != (secondLocation{2, 1}) || !strings.Contains(string(want), strings.TrimSpace(strings.SplitN(body, "\n", 2)[0])) {
		t.Fatalf("closed gate: open %v current %v body %q", open, c.current, body)
	}
	t.Logf("closed gate shows %s (%d bytes) and keeps town 1", secondGateText, len(want))
}

// The gate and guard follow R2-ENGINE-243 for both flag values: closed, the
// gate holds T08 and gate hover walks the guard to frame 0 and back; open,
// gate hover walks the gate from T08 to T00 and the guard stays at 7.
func TestReleaseSecondTownOneGateAndGuard(t *testing.T) {
	for _, open := range []bool{false, true} {
		f, _, s, now := secondTownOne(t)
		d, _ := s.composedTown()
		sq := s.scene(d, "square")
		if open {
			f.Town.second.bank[769] = 1
		}
		door, guard := sq.Actor("door").(*town.Stepper), sq.Actor("guard").(*town.Driven)
		secondTownStep(s, now)
		if door.Shown(sq) != 8 || guard.Shown(sq) != 7 {
			t.Fatalf("open=%v entry: gate %d guard %d, want 8 and 7", open, door.Shown(sq), guard.Shown(sq))
		}
		s.TownSquarePointer(secondTownPoint(t, s, 160))
		var gates, guards []int
		for i := 0; i < 10; i++ {
			secondTownStep(s, now)
			gates, guards = append(gates, door.Shown(sq)), append(guards, guard.Shown(sq))
		}
		wantGate, wantGuard := []int{8, 8, 8, 8, 8, 8, 8, 8, 8, 8}, []int{6, 5, 4, 3, 2, 1, 0, 0, 0, 0}
		if open {
			wantGate, wantGuard = []int{7, 6, 5, 4, 3, 2, 1, 0, 0, 0}, []int{7, 7, 7, 7, 7, 7, 7, 7, 7, 7}
		}
		if !equalInts(gates, wantGate) || !equalInts(guards, wantGuard) {
			t.Errorf("open=%v gate hover: gate %v guard %v, want %v and %v", open, gates, guards, wantGate, wantGuard)
		}
		s.TownSquarePointer(secondTownPoint(t, s, 0))
		for i := 0; i < 10; i++ {
			secondTownStep(s, now)
		}
		if door.Shown(sq) != 8 || guard.Shown(sq) != 7 {
			t.Errorf("open=%v away: gate %d guard %d, want 8 and 7", open, door.Shown(sq), guard.Shown(sq))
		}
		t.Logf("open=%v: gate %v guard %v", open, gates, guards)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Under the synthetic clock the tavern runs ten frames from inn hover and
// stops at 0; a paint at 67 ms admits no step (R2-ENGINE-242).
func TestReleaseSecondTownOneEpisodes(t *testing.T) {
	_, _, s, now := secondTownOne(t)
	d, _ := s.composedTown()
	sq := s.scene(d, "square")
	tavern := sq.Actor("tavern").(*town.Episode)
	secondTownStep(s, now)
	s.TownSquarePointer(secondTownPoint(t, s, 128))
	*now = now.Add(67 * time.Millisecond)
	s.AdvanceTownSquareAnimation()
	if tavern.Enabled && tavern.Frame != 0 {
		t.Fatal("a paint at exactly 67 ms advanced the tavern")
	}
	var frames []int
	for i := 0; i < 11; i++ {
		secondTownStep(s, now)
		frames = append(frames, tavern.Frame)
	}
	s.TownSquarePointer(secondTownPoint(t, s, 0))
	if frames[0] != 1 || frames[8] != 9 || frames[9] != 0 {
		t.Errorf("tavern frames %v, want 1..9 then 0", frames)
	}
	t.Logf("tavern frames under inn hover: %v", frames)
}

// The inn is a page: three panels, the seat grid and three buttons at the
// rectangles R2-ENGINE-246 and R2-ENGINE-247 measured; a seat click selects,
// Talk runs TalkTo for the selected seat, and Exit and Escape return to the
// square (R2-ENGINE-250).
func TestReleaseSecondTownOneInnPage(t *testing.T) {
	f, app, s, now := secondTownOne(t)
	c := f.Town.second
	if err := app.HeadlessActivate("TAVERN"); err != nil || c.room != secondTownInn || !s.AtTownSquare() {
		t.Fatalf("TAVERN: %v room %v composed %v", err, c.room, s.AtTownSquare())
	}
	s.TownSquareActive(true)
	speakers := c.speakers()
	var rows []string
	for _, r := range s.Rows() {
		rows = append(rows, r.Text)
	}
	if len(speakers) != 3 || strings.Join(rows, ",") != "TALK 207,TALK 2108,TALK 517,EXIT" {
		t.Fatalf("inn rows %v", rows)
	}
	view := s.TownSquareView().Scene
	for i, at := range []image.Point{{200, 448}, {248, 448}, {296, 448}} {
		if ctl, ok := view.ControlAt(at); !ok || ctl.Kind != ui.TownSquareControlSelect || ctl.Door != i {
			t.Errorf("seat %d at %v: %+v %v", i, at, ctl, ok)
		}
	}
	if _, ok := view.ControlAt(image.Pt(344, 448)); ok {
		t.Error("an empty fourth seat answers a click")
	}
	if _, ok := view.ControlAt(image.Pt(200, 384)); ok {
		t.Error("an empty second-row seat answers a click")
	}
	if _, ok := view.ControlAt(image.Pt(554, 67)); ok {
		t.Error("the hire button acts on a talk roster")
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 114)); !ok || ctl.Door != 0 {
		t.Errorf("talk button with seat 0 selected: %+v %v", ctl, ok)
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 161)); !ok || ctl.Door != 3 {
		t.Errorf("exit button: %+v %v", ctl, ok)
	}
	for _, p := range []image.Point{{483, 67}, {624, 67}, {554, 43}, {554, 185}} {
		if _, ok := view.ControlAt(p); ok {
			t.Errorf("%v outside every button answers", p)
		}
	}
	s.SelectTownSquare(2)
	s.TownSquareActive(true)
	for i := 0; i < 3; i++ {
		*now = now.Add(126 * time.Millisecond)
		s.AdvanceTownSquareAnimation()
	}
	if s.composed.seat != 2 || s.seatFrame(2) != 3 || s.seatFrame(0) != 0 {
		t.Errorf("seat 2 selected: seat %d frames %d/%d", s.composed.seat, s.seatFrame(2), s.seatFrame(0))
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 114)); !ok || ctl.Door != 2 {
		t.Errorf("talk button with seat 2 selected: %+v %v", ctl, ok)
	}

	d, _ := s.composedTown()
	art := s.roomArt(d, "inn")
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	view.Paint(dst)
	for _, probe := range []struct {
		name string
		pic  image.Image
		at   image.Point
		p    image.Point
	}{
		{"left stats", art.scene.Pictures("left-stats")[0], image.Pt(0, 0), image.Pt(80, 100)},
		{"left picture", art.scene.Pictures("left-picture")[0], image.Pt(0, 238), image.Pt(80, 300)},
		{"buttons area", art.scene.Pictures("upper")[0], image.Pt(480, 0), image.Pt(560, 220)},
		{"hire button", art.page.Picture("button/0/0"), image.Pt(484, 44), image.Pt(490, 50)},
	} {
		if probe.pic == nil {
			t.Errorf("%s did not load", probe.name)
			continue
		}
		q := probe.p.Sub(probe.at).Add(probe.pic.Bounds().Min)
		r1, g1, b1, _ := probe.pic.At(q.X, q.Y).RGBA()
		r2, g2, b2, _ := dst.At(probe.p.X, probe.p.Y).RGBA()
		if [3]uint32{r1, g1, b1} != [3]uint32{r2, g2, b2} {
			t.Errorf("%s not drawn at %v", probe.name, probe.at)
		}
	}

	if err := app.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	if !c.has(secondLocation{1, 10}) || c.bank[769] != 1 || c.payload == nil {
		t.Fatal("TALK 517 did not run TalkTo or show its section")
	}
	for app.HeadlessActivate("notice") == nil {
	}
	if err := app.HeadlessActivate("EXIT"); err != nil || c.room != secondTownSquare || !s.AtTownSquare() {
		t.Fatalf("EXIT: %v room %v", err, c.room)
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare || app.Screen() != ui.ScreenTown {
		t.Fatalf("Escape: %v room %v screen %v", err, c.room, app.Screen())
	}
	t.Logf("inn rows %v; seats 3 at x=176,224,272 y=416; buttons hire/talk/exit; TALK 517 admits mission 10 and bank769", rows)
}

// A SAV written in the inn loads cold into the inn page.
func TestReleaseSecondTownOneColdLoadKeepsTheRoom(t *testing.T) {
	f, app, _, _ := secondTownOne(t)
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	secondTownNamedSave(t, f, app, out, "Inn room")
	cold, coldApp := secondTownCold(t, out, "Inn room.sav")
	s := cold.TownScreen().(*secondCampaignScreen)
	coldApp.SetTown(s)
	if cold.Town.second.room != secondTownInn || !s.AtTownSquare() {
		t.Fatalf("cold LOAD room %v composed %v", cold.Town.second.room, s.AtTownSquare())
	}
	if err := coldApp.HeadlessActivate("EXIT"); err != nil || cold.Town.second.room != secondTownSquare {
		t.Fatalf("cold EXIT: %v", err)
	}
}
