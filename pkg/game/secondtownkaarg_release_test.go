package game

import (
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondTownTwo opens a new second-game campaign and enters town 2 from the
// destination rows under a synthetic animation clock the caller advances.
// The campaign position is constructed: stage 30 with town 2 the only
// destination, the state the departure after mission 20 leaves
// (R2-ENGINE-146), without playing missions 10 and 20.
func secondTownTwo(t *testing.T) (*FrontEnd, *ui.App, *secondCampaignScreen, *time.Time) {
	t.Helper()
	f := secondGameFront(t)
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	app := f.App("second town two")
	app.Layout(640, 480)
	s := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(s)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	c := f.Town.second
	c.current, c.available, c.room = secondLocation{}, []secondLocation{{2, 2}}, secondTownSquare
	c.bank[768] = 30
	for _, row := range []string{"town 2"} {
		if err := app.HeadlessActivate(row); err != nil {
			t.Fatal(row, err)
		}
	}
	if c.current != (secondLocation{2, 2}) || !s.AtTownSquare() {
		d, _ := s.composedTown()
		t.Fatalf("town 2 is not composed: current %v, %v", c.current, s.roomArt(d, "square").err)
	}
	s.TownSquareActive(true)
	return f, app, s, &now
}

// kaargStep admits one square step: the clock moves past the 100 ms period
// and the screen paints once (R2-ENGINE-235).
func kaargStep(s *secondCampaignScreen, now *time.Time, d time.Duration) {
	*now = now.Add(d)
	s.AdvanceTownSquareAnimation()
}

func kaargScene(t *testing.T, s *secondCampaignScreen) (*town.Description, *town.Scene) {
	t.Helper()
	d, ok := s.composedTown()
	if !ok || d.Town != "rom2-town2" {
		t.Fatal("town 2 description not shown")
	}
	return d, s.scene(d, "square")
}

// The installed Kaarg mask holds the counts R2-ASSET-050 measured: eight
// nonzero bytes, no byte 80, 217863 zero cells and nothing else. Each byte
// reaches the hotspot, tip (R2-ENGINE-234) and control the description gives
// it.
func TestReleaseSecondTownKaargMaskCensus(t *testing.T) {
	_, _, s, _ := secondTownTwo(t)
	d, sq := kaargScene(t, s)
	m := s.roomArt(d, "square").scene.Mask
	if m.Rect != image.Rect(0, 0, 640, 480) {
		t.Fatalf("mask %v", m.Rect)
	}
	counts := map[byte]int{}
	for _, v := range m.Pix {
		counts[v]++
	}
	want := map[byte]int{0: 217863, 32: 10285, 64: 4751, 96: 3180, 128: 18424, 144: 3792, 160: 4415, 176: 40667, 192: 3823}
	if len(counts) != len(want) || counts[80] != 0 {
		t.Errorf("%d distinct bytes, byte 80 %d cells; want %d and none", len(counts), counts[80], len(want))
	}
	for b, n := range want {
		if counts[b] != n {
			t.Errorf("byte %d: %d cells, want %d", b, counts[b], n)
		}
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
		144: {"shop", 233, ui.TownSquareControlDoor, 6, true},
		160: {"gate", 237, ui.TownSquareControlDoor, 1, true},
		176: {"menu", 235, ui.TownSquareControlMenu, 0, true},
		96:  {"girl1", 359, ui.TownSquareControlDoor, 2, true},
		192: {"girl2", 360, ui.TownSquareControlDoor, 3, true},
		64:  {"guard", 361, ui.TownSquareControlDoor, 4, true},
		32:  {"dervish", 362, ui.TownSquareControlDoor, 5, true},
		0:   {},
	} {
		p := secondTownPoint(t, s, b)
		h := sq.HotspotAt(p)
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
	t.Logf("Kaarg mask counts %v", counts)
}

// The square paints in the order and at the destinations R2-ENGINE-232
// measured: background, shop or inn highlight under its hover, girl1, girl2,
// the guard, the dervish and the gates, from the keys R2-ENGINE-231 names in
// the ranges R2-ASSET-051 measured.
func TestReleaseSecondTownKaargPainterOrder(t *testing.T) {
	_, _, s, now := secondTownTwo(t)
	d, sq := kaargScene(t, s)
	type layer struct {
		name string
		at   image.Point
	}
	want := []layer{{"base", image.Pt(0, 0)}, {"hili-shop", image.Pt(328, 256)}, {"hili-inn", image.Pt(480, 196)},
		{"girl1", image.Pt(0, 0)}, {"girl2", image.Pt(0, 0)}, {"guard", image.Pt(0, 0)}, {"dervish", image.Pt(0, 0)},
		{"gate", image.Pt(152, 256)}}
	if len(d.Layers) != len(want) {
		t.Fatalf("%d layers, want %d", len(d.Layers), len(want))
	}
	for i, l := range d.Layers {
		name := l.Art + l.Actor
		if got := (layer{name, l.At.Pt()}); got != want[i] || l.Mode != "copy" {
			t.Errorf("layer %d = %+v %s, want %+v copied", i, got, l.Mode, want[i])
		}
	}
	art := s.roomArt(d, "square").scene
	for name, n := range map[string]int{"girl1/0": 31, "girl1/1": 30, "girl2/0": 31, "girl2/1": 30, "guard": 55, "dervish": 30, "gates": 11} {
		if got := len(art.Pictures(name)); got != n {
			t.Errorf("%s: %d frames, want %d", name, got, n)
		}
	}
	for _, a := range d.Art {
		if !strings.HasPrefix(a.Key, "graphics/interface/town_kaarg/") {
			t.Errorf("%s reads %q outside interface/town_kaarg", a.Name, a.Key)
		}
	}
	persons := sq.Actor("persons").(*town.Families)
	idle := map[string]string{"girl1": "girl1/0", "girl2": "girl2/0", "guard": "guard", "dervish": "dervish"}
	for name, at := range map[string]image.Point{"girl1": {216, 284}, "girl2": {260, 284}, "guard": {184, 156}, "dervish": {416, 328}} {
		sheet, frame, got, ok := persons.Shown(sq, name)
		if !ok || got != at || frame != 0 || sheet != idle[name] {
			t.Errorf("idle %s: %s frame %d at %v (%v), want its first frame at %v", name, sheet, frame, got, ok, at)
		}
	}
	probe := func(dst *image.RGBA, pic image.Image, at, p image.Point) bool {
		q := p.Sub(at).Add(pic.Bounds().Min)
		r1, g1, b1, _ := pic.At(q.X, q.Y).RGBA()
		r2, g2, b2, _ := dst.At(p.X, p.Y).RGBA()
		return [3]uint32{r1, g1, b1} == [3]uint32{r2, g2, b2}
	}
	paint := func() *image.RGBA {
		dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
		s.TownSquareView().Scene.Paint(dst)
		return dst
	}
	dst := paint()
	for _, c := range []struct {
		name string
		pic  image.Image
		at   image.Point
	}{
		{"girl1", art.Pictures("girl1/0")[0], image.Pt(216, 284)}, {"girl2", art.Pictures("girl2/0")[0], image.Pt(260, 284)},
		{"dervish", art.Pictures("dervish")[0], image.Pt(416, 328)}, {"gate", art.Pictures("gates")[0], image.Pt(152, 256)},
	} {
		if !probe(dst, c.pic, c.at, c.at.Add(image.Pt(5, 5))) {
			t.Errorf("%s not painted at %v", c.name, c.at)
		}
	}
	for _, h := range []struct {
		b    byte
		pic  string
		at   image.Point
		near image.Point
	}{{144, "hili-shop", image.Pt(328, 256), image.Pt(20, 30)}, {128, "hili-inn", image.Pt(480, 196), image.Pt(70, 20)}} {
		s.TownSquarePointer(secondTownPoint(t, s, h.b))
		kaargStep(s, now, 101*time.Millisecond)
		if !probe(paint(), art.Pictures(h.pic)[0], h.at, h.at.Add(h.near)) {
			t.Errorf("%s not painted at %v under its hover", h.pic, h.at)
		}
	}
}

// Gates walk toward frame 10 while the pointer is over selector 8 and back
// toward frame 0 otherwise, one frame per admitted step (R2-ENGINE-236).
func TestReleaseSecondTownKaargGates(t *testing.T) {
	_, _, s, now := secondTownTwo(t)
	_, sq := kaargScene(t, s)
	gate := sq.Actor("gate").(*town.Stepper)
	kaargStep(s, now, 101*time.Millisecond)
	kaargStep(s, now, 101*time.Millisecond)
	if gate.Shown(sq) != 0 {
		t.Fatalf("entry gate frame %d, want 0", gate.Shown(sq))
	}
	s.TownSquarePointer(secondTownPoint(t, s, 160))
	var up, down []int
	for i := 0; i < 12; i++ {
		kaargStep(s, now, 101*time.Millisecond)
		up = append(up, gate.Shown(sq))
	}
	s.TownSquarePointer(secondTownPoint(t, s, 0))
	*now = now.Add(100 * time.Millisecond)
	s.AdvanceTownSquareAnimation()
	if gate.Shown(sq) != 10 {
		t.Fatalf("frame %d after a paint at exactly 100 ms", gate.Shown(sq))
	}
	for i := 0; i < 12; i++ {
		kaargStep(s, now, 101*time.Millisecond)
		down = append(down, gate.Shown(sq))
	}
	if !equalInts(up, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 10, 10}) || !equalInts(down, []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 0, 0}) {
		t.Errorf("gate frames up %v down %v", up, down)
	}
	t.Logf("gate up %v down %v", up, down)
}

// The guard runs its 55 frames once per episode, drawn at (140,152) while its
// frame lies in 13..37 and at (184,156) otherwise (R2-ENGINE-232,
// R2-ENGINE-235).
func TestReleaseSecondTownKaargGuardPosition(t *testing.T) {
	_, _, s, now := secondTownTwo(t)
	_, sq := kaargScene(t, s)
	persons := sq.Actor("persons").(*town.Families)
	guard := persons.Member("guard")
	kaargStep(s, now, 101*time.Millisecond)
	for i := 0; i < 60 && !guard.Active; i++ {
		kaargStep(s, now, 101*time.Millisecond)
	}
	if !guard.Active || guard.Current != 0 {
		t.Fatalf("no guard episode within 6 s: %+v", *guard)
	}
	seen := map[int]image.Point{}
	for guard.Active {
		_, frame, at, _ := persons.Shown(sq, "guard")
		seen[frame] = at
		kaargStep(s, now, 101*time.Millisecond)
	}
	if len(seen) != 55 {
		t.Fatalf("guard showed %d frames, want 55", len(seen))
	}
	for frame, at := range seen {
		want := image.Pt(184, 156)
		if frame >= 13 && frame <= 37 {
			want = image.Pt(140, 152)
		}
		if at != want {
			t.Errorf("guard frame %d at %v, want %v", frame, at, want)
		}
	}
	if _, frame, at, _ := persons.Shown(sq, "guard"); frame != 0 || at != image.Pt(184, 156) {
		t.Errorf("idle guard frame %d at %v", frame, at)
	}
}

// Person waits fall in the ranges R2-ENGINE-235 derives: entry 2000..3999 ms
// for girl1, girl2 and the guard and 4000..4499 for the dervish; later
// 3500..8499, 7500..12499 and 3700..4199. The scheduler runs on paints that
// admit no step, and with paints every 150 ms the dervish's 30 frames outlast
// its wait, so a running episode restarts from frame 0.
func TestReleaseSecondTownKaargPersonWaits(t *testing.T) {
	_, _, s, now := secondTownTwo(t)
	_, sq := kaargScene(t, s)
	persons := sq.Actor("persons").(*town.Families)
	type span struct{ lo, hi time.Duration }
	ms := time.Millisecond
	entry := map[string]span{"girl1": {2000 * ms, 3999 * ms}, "girl2": {2000 * ms, 3999 * ms},
		"guard": {2000 * ms, 3999 * ms}, "dervish": {4000 * ms, 4499 * ms}}
	later := map[string]span{"girl1": {3500 * ms, 8499 * ms}, "girl2": {3500 * ms, 8499 * ms},
		"guard": {7500 * ms, 12499 * ms}, "dervish": {3700 * ms, 4199 * ms}}
	for name, w := range entry {
		if m := persons.Member(name); m.Wait < w.lo || m.Wait > w.hi || m.Active {
			t.Errorf("%s entry wait %v, want %v..%v", name, m.Wait, w.lo, w.hi)
		}
	}
	starts := map[string]int{}
	sheets := map[int]bool{}
	restarted, idlePaintStarts := false, 0
	prev := map[string]town.FamilyMember{}
	for i := 0; i < 400; i++ {
		d := 150 * ms
		if i%2 == 1 {
			d = 40 * ms
		}
		kaargStep(s, now, d)
		for name, w := range later {
			m := *persons.Member(name)
			p := prev[name]
			if m.Active && m.Current == 0 && m.Clock.Equal(*now) {
				starts[name]++
				if m.Wait < w.lo || m.Wait > w.hi {
					t.Errorf("%s later wait %v, want %v..%v", name, m.Wait, w.lo, w.hi)
				}
				if name == "girl1" || name == "girl2" {
					sheets[m.Sheet] = true
				}
				if name == "dervish" && p.Active && p.Current > 0 && p.Current < 29 {
					restarted = true
				}
				if i%2 == 1 {
					idlePaintStarts++
				}
			}
			prev[name] = m
		}
	}
	for name := range later {
		if starts[name] == 0 {
			t.Errorf("%s never started in 76 s", name)
		}
	}
	if !restarted || !sheets[0] || !sheets[1] || idlePaintStarts == 0 {
		t.Errorf("dervish restart while running %v, girl subsets %v, starts on paints without a step %d", restarted, sheets, idlePaintStarts)
	}
	t.Logf("starts %v, subsets %v, dervish restart while running %v, starts on paints without a step %d", starts, sheets, restarted, idlePaintStarts)
}

// Each person opens its stage-keyed section from the town text by click and
// by row, and the shop opens the shop page (R2-ENGINE-234).
func TestReleaseSecondTownKaargPersonClicks(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	c := f.Town.second
	_, sq := kaargScene(t, s)
	sq.Click(sq.Hotspot("shop"))
	if c.payload != nil || c.room != secondTownShop {
		t.Fatal("the shop click did not open the shop page", c.room)
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape from the shop page", err, c.room)
	}
	view := s.TownSquareView().Scene
	for _, p := range []struct {
		row, key string
		b        byte
	}{{"GIRL 1", "kaargwoman30", 96}, {"GIRL 2", "kaargwoman30", 192}, {"GUARD", "kaargguard30", 64}, {"DERVISH", "kaargman30", 32}} {
		want, err := readSecondTownTalk(&f.InstallResources, p.key)
		if err != nil || want == nil {
			t.Fatalf("installed %s section: %v", p.key, err)
		}
		first := strings.TrimSpace(strings.SplitN(string(want), "\n", 2)[0])
		for _, how := range []string{"row", "click"} {
			if how == "row" {
				if err := app.HeadlessActivate(p.row); err != nil {
					t.Fatal(p.row, err)
				}
			} else {
				ctl, ok := view.ControlAt(secondTownPoint(t, s, p.b))
				if !ok || ctl.Kind != ui.TownSquareControlDoor {
					t.Fatalf("%s: no control", p.row)
				}
				s.Choose(ctl.Door)
			}
			body, open := s.dialogueBody()
			if !open || c.room != secondTownSquare || !strings.Contains(string(want), strings.TrimSpace(strings.SplitN(body, "\n", 2)[0])) {
				t.Fatalf("%s by %s: open %v body %q, want %s starting %q", p.row, how, open, body, p.key, first)
			}
			for app.HeadlessActivate("notice") == nil {
			}
		}
		t.Logf("%s opens %s (%d bytes)", p.row, p.key, len(want))
	}
}

// The inn is town 1's inn page with the Kaarg centre: TavernMain at (160,0)
// and the taverner's first a1 picture at centre (72,88) (R2-ENGINE-237,
// R2-ENGINE-277, R2-ENGINE-353); Exit and Escape return to the square.
func TestReleaseSecondTownKaargInnPage(t *testing.T) {
	f, app, s, _ := secondTownTwo(t)
	c := f.Town.second
	if err := app.HeadlessActivate("TAVERN"); err != nil || c.room != secondTownInn || !s.AtTownSquare() {
		t.Fatalf("TAVERN: %v room %v composed %v", err, c.room, s.AtTownSquare())
	}
	s.TownSquareActive(true)
	var rows []string
	for _, r := range s.Rows() {
		rows = append(rows, r.Text)
	}
	if len(rows) != len(c.speakers())+1 || rows[len(rows)-1] != secondInnExit {
		t.Fatalf("inn rows %v for %d speakers", rows, len(c.speakers()))
	}
	d, _ := s.composedTown()
	art := s.roomArt(d, "inn")
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	s.TownSquareView().Scene.Paint(dst)
	for _, probe := range []struct {
		name string
		pic  image.Image
		at   image.Point
		p    image.Point
	}{
		{"centre", art.scene.Pictures("center")[0], image.Pt(160, 0), image.Pt(300, 20)},
		{"taverner", art.scene.Pictures("taverner")[0], image.Pt(232, 88), image.Pt(350, 230)},
		{"left stats", art.scene.Pictures("left-stats")[0], image.Pt(0, 0), image.Pt(80, 100)},
		{"buttons area", art.scene.Pictures("upper")[0], image.Pt(480, 0), image.Pt(560, 220)},
	} {
		q := probe.p.Sub(probe.at).Add(probe.pic.Bounds().Min)
		r1, g1, b1, _ := probe.pic.At(q.X, q.Y).RGBA()
		r2, g2, b2, _ := dst.At(probe.p.X, probe.p.Y).RGBA()
		if [3]uint32{r1, g1, b1} != [3]uint32{r2, g2, b2} {
			t.Errorf("%s not drawn at %v", probe.name, probe.at)
		}
	}
	if err := app.HeadlessActivate(secondInnExit); err != nil || c.room != secondTownSquare || !s.AtTownSquare() {
		t.Fatalf("EXIT: %v room %v", err, c.room)
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare || app.Screen() != ui.ScreenTown {
		t.Fatalf("Escape: %v room %v screen %v", err, c.room, app.Screen())
	}
	if err := app.HeadlessActivate("GATES"); err != nil || c.current != (secondLocation{}) || !c.has(secondLocation{2, 2}) {
		t.Fatalf("GATES: %v current %v available %v", err, c.current, c.available)
	}
	t.Logf("inn rows %v", rows)
}

// A SAV written in the Kaarg inn loads cold into the Kaarg inn page.
func TestReleaseSecondTownKaargColdLoadKeepsTheRoom(t *testing.T) {
	f, app, _, _ := secondTownTwo(t)
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	secondTownNamedSave(t, f, app, out, "Kaarg inn")
	cold, coldApp := secondTownCold(t, out, secondSaveFile(t, out, "Kaarg inn"))
	s := cold.TownScreen().(*secondCampaignScreen)
	coldApp.SetTown(s)
	c := cold.Town.second
	if c.current != (secondLocation{2, 2}) || c.room != secondTownInn || !s.AtTownSquare() {
		t.Fatalf("cold LOAD current %v room %v composed %v", c.current, c.room, s.AtTownSquare())
	}
	if d, _ := s.composedTown(); d.Town != "rom2-town2" {
		t.Fatalf("cold LOAD shows %s", d.Town)
	}
	if err := coldApp.HeadlessActivate(secondInnExit); err != nil || c.room != secondTownSquare {
		t.Fatalf("cold EXIT: %v", err)
	}
}
