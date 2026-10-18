package game

import (
	"image"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondTownThree opens a new second-game campaign and enters town 3 from
// the destination rows under a synthetic animation clock the caller
// advances. The campaign position is constructed: the stage given, town 3
// and mission 70 the destinations, the availability ordinary Leave50 adds to
// (R2-SESSION-123, R2-SESSION-155), without playing the missions before it.
func secondTownThree(t *testing.T, stage int32) (*FrontEnd, *ui.App, *secondCampaignScreen, *time.Time) {
	t.Helper()
	f := secondGameFront(t)
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	app := f.App("second town three")
	app.Layout(640, 480)
	s := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(s)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	c := f.Town.second
	c.current, c.available, c.room = secondLocation{}, []secondLocation{{2, 3}, {1, 70}}, secondTownSquare
	c.bank[768] = stage
	for _, row := range []string{"town 3"} {
		if err := app.HeadlessActivate(row); err != nil {
			t.Fatal(row, err)
		}
	}
	if c.current != (secondLocation{2, 3}) || !s.AtTownSquare() {
		d, _ := s.composedTown()
		t.Fatalf("town 3 is not composed: current %v, %v", c.current, s.roomArt(d, "square").err)
	}
	s.TownSquareActive(true)
	return f, app, s, &now
}

func druidScene(t *testing.T, s *secondCampaignScreen) (*town.Description, *town.Scene) {
	t.Helper()
	d, ok := s.composedTown()
	if !ok || d.Town != "rom2-town3" {
		t.Fatal("town 3 description not shown")
	}
	return d, s.scene(d, "square")
}

// druidPaint paints the shown room once.
func druidPaint(s *secondCampaignScreen) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	s.TownSquareView().Scene.Paint(dst)
	return dst
}

// druidProbe reports whether dst at p shows the pixel of pic drawn with its
// top-left corner at at.
func druidProbe(dst *image.RGBA, pic image.Image, at, p image.Point) bool {
	q := p.Sub(at).Add(pic.Bounds().Min)
	r1, g1, b1, _ := pic.At(q.X, q.Y).RGBA()
	r2, g2, b2, _ := dst.At(p.X, p.Y).RGBA()
	return [3]uint32{r1, g1, b1} == [3]uint32{r2, g2, b2}
}

// druidDiffPoint finds a page point where the picture drawn at at differs
// from the background, so a probe there tells the two apart.
func druidDiffPoint(t *testing.T, base, pic image.Image, at image.Point) image.Point {
	t.Helper()
	b := pic.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := at.Add(image.Pt(x-b.Min.X, y-b.Min.Y))
			r1, g1, b1, _ := pic.At(x, y).RGBA()
			r2, g2, b2, _ := base.At(p.X, p.Y).RGBA()
			if [3]uint32{r1, g1, b1} != [3]uint32{r2, g2, b2} {
				return p
			}
		}
	}
	t.Fatalf("the picture at %v equals the background", at)
	return at
}

// The installed druid mask holds the counts R2-ASSET-064 measured: five
// nonzero bytes, 262188 zero cells, no byte 176, and each byte inside its
// measured rectangle. Each byte reaches the hotspot, tip and control the
// description gives it: the shop, inn and gate tips 233, 236 and 237, and no
// tip for the two keepers (R2-ENGINE-255). The menu arm has no pixel.
func TestReleaseSecondTownDruidMaskCensus(t *testing.T) {
	_, _, s, _ := secondTownThree(t, 60)
	d, sq := druidScene(t, s)
	m := s.roomArt(d, "square").scene.Mask
	if m.Rect != image.Rect(0, 0, 640, 480) {
		t.Fatalf("mask %v", m.Rect)
	}
	counts := map[byte]int{}
	boxes := map[byte]image.Rectangle{}
	for i, v := range m.Pix {
		counts[v]++
		p := image.Pt(i%m.Stride, i/m.Stride)
		cell := image.Rectangle{Min: p, Max: p.Add(image.Pt(1, 1))}
		if b, ok := boxes[v]; ok {
			boxes[v] = b.Union(cell)
		} else {
			boxes[v] = cell
		}
	}
	want := map[byte]int{0: 262188, 80: 5421, 96: 2144, 128: 3397, 144: 24098, 160: 9952}
	if len(counts) != len(want) || counts[176] != 0 {
		t.Errorf("%d distinct bytes, byte 176 %d cells; want %d and none", len(counts), counts[176], len(want))
	}
	for b, n := range want {
		if counts[b] != n {
			t.Errorf("byte %d: %d cells, want %d", b, counts[b], n)
		}
	}
	// R2-ASSET-064 states each rectangle half-open, its maximum exclusive.
	for b, r := range map[byte]image.Rectangle{80: image.Rect(338, 253, 390, 368), 96: image.Rect(166, 202, 201, 270),
		128: image.Rect(120, 189, 169, 274), 144: image.Rect(375, 223, 570, 395), 160: image.Rect(0, 293, 174, 374)} {
		if boxes[b] != r {
			t.Errorf("byte %d spans %v, want %v", b, boxes[b], r)
		}
	}
	view := s.TownSquareView().Scene
	type answer struct {
		hotspot string
		tip     int
		tipped  bool
		control ui.TownSquareControlKind
		door    int
		acts    bool
	}
	for b, w := range map[byte]answer{
		128: {"inn", 236, true, ui.TownSquareControlDoor, 0, true},
		144: {"shop", 233, true, ui.TownSquareControlDoor, 4, true},
		160: {"gate", 237, true, ui.TownSquareControlDoor, 1, true},
		96:  {"innkeeper", 0, false, ui.TownSquareControlDoor, 2, true},
		80:  {"shopkeeper", 0, false, ui.TownSquareControlDoor, 3, true},
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
		if name != w.hotspot || tipped != w.tipped || tipped && tip != w.tip || acts != w.acts ||
			acts && (c.Kind != w.control || c.Door != w.door) {
			t.Errorf("byte %d at %v: hotspot %q tip %d/%v control %+v/%v, want %+v", b, p, name, tip, tipped, c, acts, w)
		}
	}
	menu := sq.Hotspot("menu")
	if slot, ok := menu.TipSlot(); menu == nil || !ok || slot != 235 || town.Menu(menu.Click) == "" {
		t.Error("the menu arm is not the description's tip 235 game menu")
	}
	for _, mb := range d.Mask.Bytes {
		if mb.Hotspot == "menu" {
			t.Errorf("byte %d reaches the menu", mb.Byte)
		}
	}
	t.Logf("druid mask counts %v", counts)
}

// The square paints in the order and at the destinations R2-ENGINE-253
// measured: background, shop highlight, woman, inn highlight, man, lizard and
// bug, from the town_druid keys in the ranges R2-ASSET-065 measured. Idle,
// the woman shows subset 0 file 1 at (336,244) and the man at (164,200);
// under shop or inn hover each shows subset 2 at (340,224) or (152,184)
// beside its highlight.
func TestReleaseSecondTownDruidPainterOrder(t *testing.T) {
	_, _, s, now := secondTownThree(t, 60)
	d, sq := druidScene(t, s)
	type layer struct {
		name string
		at   image.Point
	}
	want := []layer{{"base", image.Pt(0, 0)}, {"hili-shop", image.Pt(420, 224)}, {"woman", image.Pt(0, 0)},
		{"hili-inn", image.Pt(0, 184)}, {"man", image.Pt(0, 0)}, {"lizard", image.Pt(0, 300)}, {"bug", image.Pt(0, 212)}}
	if len(d.Layers) != len(want) {
		t.Fatalf("%d layers, want %d", len(d.Layers), len(want))
	}
	for i, l := range d.Layers {
		if got := (layer{l.Art + l.Actor, l.At.Pt()}); got != want[i] {
			t.Errorf("layer %d = %+v, want %+v", i, got, want[i])
		}
	}
	art := s.roomArt(d, "square").scene
	for name, n := range map[string]int{"woman/0": 15, "woman/1": 20, "woman/2": 20, "man/0": 19, "man/1": 19, "man/2": 20,
		"lizard": 42, "bug": 183, "hili-shop": 1, "hili-inn": 1} {
		if got := len(art.Pictures(name)); got != n {
			t.Errorf("%s: %d frames, want %d", name, got, n)
		}
	}
	for _, a := range d.Art {
		if !strings.HasPrefix(a.Key, "graphics/interface/town_druid/") {
			t.Errorf("%s reads %q outside interface/town_druid", a.Name, a.Key)
		}
	}
	persons := sq.Actor("persons").(*town.Families)
	shows := func(name, sheet string, at image.Point) {
		t.Helper()
		got, frame, p, ok := persons.Shown(sq, name)
		if !ok || got != sheet || frame != 0 || p != at {
			t.Errorf("%s shows %s frame %d at %v (%v), want %s frame 0 at %v", name, got, frame, p, ok, sheet, at)
		}
	}
	shows("woman", "woman/0", image.Pt(336, 244))
	shows("man", "man/0", image.Pt(164, 200))
	dst := druidPaint(s)
	for _, c := range []struct {
		name string
		pic  image.Image
		at   image.Point
	}{{"base", art.Pictures("base")[0], image.Pt(0, 0)}, {"woman", art.Pictures("woman/0")[0], image.Pt(336, 244)},
		{"man", art.Pictures("man/0")[0], image.Pt(164, 200)}} {
		if !druidProbe(dst, c.pic, c.at, c.at.Add(image.Pt(20, 30))) {
			t.Errorf("%s not painted at %v", c.name, c.at)
		}
	}
	for _, h := range []struct {
		b             byte
		person, sheet string
		at, hiliAt    image.Point
		hili          string
	}{{144, "woman", "woman/2", image.Pt(340, 224), image.Pt(420, 224), "hili-shop"},
		{128, "man", "man/2", image.Pt(152, 184), image.Pt(0, 184), "hili-inn"}} {
		s.TownSquarePointer(secondTownPoint(t, s, h.b))
		shows(h.person, h.sheet, h.at)
		dst := druidPaint(s)
		if !druidProbe(dst, art.Pictures(h.hili)[0], h.hiliAt, druidDiffPoint(t, art.Pictures("base")[0], art.Pictures(h.hili)[0], h.hiliAt)) {
			t.Errorf("%s not painted at %v under its hover", h.hili, h.hiliAt)
		}
		if !druidProbe(dst, art.Pictures(h.sheet)[0], h.at, h.at.Add(image.Pt(20, 30))) {
			t.Errorf("%s subset 2 not painted at %v", h.person, h.at)
		}
		druidStep(s, now, 101*time.Millisecond)
	}
}

// druidStep admits one square step: the clock moves past the 100 ms period
// and the screen paints once (R2-ENGINE-256).
func druidStep(s *secondCampaignScreen, now *time.Time, d time.Duration) {
	*now = now.Add(d)
	s.AdvanceTownSquareAnimation()
}

// A changed shop or inn hover starts that person on subset 2 at once, at
// cursor 0 with its clock reset, and latches the hotspot; the same hover
// again or the keeper selectors start nothing; the highlight stays while the
// subset-2 episode runs after the pointer leaves, and the episode ends after
// its 20 files (R2-ENGINE-253, R2-ENGINE-254).
func TestReleaseSecondTownDruidHoverStartsSubsetTwo(t *testing.T) {
	_, _, s, now := secondTownThree(t, 60)
	d, sq := druidScene(t, s)
	persons := sq.Actor("persons").(*town.Families)
	woman, man := persons.Member("woman"), persons.Member("man")
	druidStep(s, now, 101*time.Millisecond)
	druidStep(s, now, 101*time.Millisecond)
	if woman.Active || man.Active {
		t.Fatal("a person started before its 2000 ms entry wait")
	}
	wait := woman.Wait
	*now = now.Add(7 * time.Millisecond)
	s.TownSquarePointer(secondTownPoint(t, s, 144))
	if !woman.Active || woman.Sheet != 2 || woman.Current != 0 || !woman.Clock.Equal(*now) || woman.Wait != wait || !sq.Latch("shop") {
		t.Fatalf("shop hover: %+v latch %v", *woman, sq.Latch("shop"))
	}
	druidStep(s, now, 101*time.Millisecond)
	p := secondTownPoint(t, s, 144).Add(image.Pt(1, 0))
	if h := sq.HotspotAt(p); h == nil || h.Name != "shop" {
		p = secondTownPoint(t, s, 144)
	}
	s.TownSquarePointer(p)
	if woman.Current != 1 {
		t.Fatalf("a hover that did not change restarted the woman: %+v", *woman)
	}
	for _, b := range []byte{96, 80} {
		s.TownSquarePointer(secondTownPoint(t, s, b))
		if man.Active || woman.Current != 1 {
			t.Fatalf("keeper byte %d started an episode: man %+v woman %+v", b, *man, *woman)
		}
	}
	s.TownSquarePointer(secondTownPoint(t, s, 0))
	if sq.Latch("shop") || sq.Latch("inn") || sq.Latch("gate") {
		t.Fatal("leaving every hotspot kept a latch")
	}
	art := s.roomArt(d, "square").scene
	at := druidDiffPoint(t, art.Pictures("base")[0], art.Pictures("hili-shop")[0], image.Pt(420, 224))
	hili := func() bool {
		return druidProbe(druidPaint(s), art.Pictures("hili-shop")[0], image.Pt(420, 224), at)
	}
	if !hili() {
		t.Fatal("the shop highlight left with the pointer while subset 2 runs")
	}
	s.TownSquarePointer(secondTownPoint(t, s, 128))
	if !man.Active || man.Sheet != 2 || man.Current != 0 || woman.Sheet != 2 {
		t.Fatalf("inn hover: man %+v woman %+v", *man, *woman)
	}
	s.TownSquarePointer(secondTownPoint(t, s, 0))
	woman.Wait = time.Hour
	frames := woman.Current + 1
	for woman.Active {
		druidStep(s, now, 101*time.Millisecond)
		if woman.Active {
			frames++
		}
	}
	if frames != 20 || hili() {
		t.Fatalf("subset 2 ran %d frames, want 20; highlight after its end %v", frames, hili())
	}
	t.Logf("woman subset 2 ran %d frames from hover", frames)
}

// Person waits fall in the ranges R2-ENGINE-256 derives: entry 2000..3999 ms
// and later 3500..8499 ms. The scheduler runs on paints that admit no step,
// picks subset R(2) without hover, and starts a person only while it is
// idle: an elapsed wait of a running person restamps its clock without a
// restart.
func TestReleaseSecondTownDruidPersonWaits(t *testing.T) {
	_, _, s, now := secondTownThree(t, 60)
	d, sq := druidScene(t, s)
	persons := sq.Actor("persons").(*town.Families)
	art := s.roomArt(d, "square").scene
	last := func(name string, m town.FamilyMember) int {
		return len(art.Pictures(name+"/"+strconv.Itoa(m.Sheet))) - 1
	}
	ms := time.Millisecond
	for _, name := range []string{"woman", "man"} {
		if m := persons.Member(name); m.Wait < 2000*ms || m.Wait > 3999*ms || m.Active {
			t.Errorf("%s entry wait %v, want 2000..3999 ms", name, m.Wait)
		}
	}
	starts := map[string]int{}
	sheets := map[int]bool{}
	consumed, idlePaintStarts := 0, 0
	prev := map[string]town.FamilyMember{}
	for i := 0; i < 600; i++ {
		d := 150 * ms
		if i%2 == 1 {
			d = 40 * ms
		}
		druidStep(s, now, d)
		for _, name := range []string{"woman", "man"} {
			m := *persons.Member(name)
			p := prev[name]
			if m.Clock.Equal(*now) {
				if m.Wait < 3500*ms || m.Wait > 8499*ms {
					t.Errorf("%s later wait %v, want 3500..8499 ms", name, m.Wait)
				}
				switch {
				case m.Active && m.Current == 0 && (!p.Active || p.Current == last(name, p)):
					starts[name]++
					sheets[m.Sheet] = true
					if i%2 == 1 {
						idlePaintStarts++
					}
				case p.Active && m.Active && (m.Current == p.Current || m.Current == p.Current+1):
					consumed++
				case p.Active:
					t.Errorf("%s restarted while running: %+v after %+v", name, m, p)
				}
			}
			prev[name] = m
		}
	}
	for _, name := range []string{"woman", "man"} {
		if starts[name] == 0 {
			t.Errorf("%s never started in 57 s", name)
		}
	}
	if !sheets[0] || !sheets[1] || sheets[2] || idlePaintStarts == 0 {
		t.Errorf("subsets %v, want 0 and 1 without hover; starts on paints without a step %d", sheets, idlePaintStarts)
	}
	t.Logf("starts %v, subsets %v, waits consumed while running %d, starts on paints without a step %d", starts, sheets, consumed, idlePaintStarts)
}

// The bug waits 7000..11999 ms after entry, picks route R(3), and draws
// frame 61*route+cursor from cursor -1 for 61 admitted steps; it ends with
// cursor 0 and waits 10000..19999 ms, so a later start shows its second
// frame first, and a start while it runs keeps the cursor (R2-ENGINE-256).
// The lizard draws its first sprite frame (DIV-2984).
func TestReleaseSecondTownDruidBugRoutes(t *testing.T) {
	_, _, s, now := secondTownThree(t, 60)
	d, sq := druidScene(t, s)
	bug := sq.Actor("bug").(*town.Route)
	ms := time.Millisecond
	if bug.Wait < 7000*ms || bug.Wait > 11999*ms || bug.Cursor != -1 || bug.Route != -1 {
		t.Fatalf("bug entry: %+v", *bug)
	}
	start := func() {
		t.Helper()
		for i := 0; i < 210 && !bug.Active; i++ {
			druidStep(s, now, 101*ms)
		}
		if !bug.Active || bug.Route < 0 || bug.Route > 2 {
			t.Fatalf("no bug start: %+v", *bug)
		}
	}
	druidStep(s, now, 101*ms)
	start()
	if _, shown := bug.Shown(); shown {
		t.Fatal("the bug is drawn before its first step")
	}
	route := bug.Route
	var frames []int
	for bug.Active {
		druidStep(s, now, 101*ms)
		if f, ok := bug.Shown(); ok {
			frames = append(frames, f)
		}
	}
	if len(frames) != 61 || frames[0] != 61*route || frames[60] != 61*route+60 {
		t.Fatalf("route %d drew %d frames from %v", route, len(frames), frames[:1])
	}
	if bug.Cursor != 0 || bug.Route != -1 || bug.Wait < 10000*ms || bug.Wait > 19999*ms {
		t.Fatalf("bug end: %+v", *bug)
	}
	start()
	druidStep(s, now, 101*ms)
	if f, ok := bug.Shown(); !ok || f != 61*bug.Route+1 {
		t.Fatalf("a later start drew frame %d, want the kept cursor's next %d", f, 61*bug.Route+1)
	}
	for i := 0; i < 5; i++ {
		druidStep(s, now, 101*ms)
	}
	cursor := bug.Cursor
	bug.Wait = 0
	*now = now.Add(ms)
	s.AdvanceTownSquareAnimation()
	if !bug.Active || bug.Cursor != cursor || !bug.Clock.Equal(*now) {
		t.Fatalf("a start while running rewound the cursor: %+v, was %d", *bug, cursor)
	}
	art := s.roomArt(d, "square").scene
	if len(art.Pictures("lizard")) != 42 {
		t.Fatal("lizard sprite frames")
	}
	t.Logf("first route %d, later route %d", route, bug.Route)
}

// Each keeper opens its section of the town text keyed by Scenario variable
// 0x300, bank slot 768, by click and by row (R2-ENGINE-255); bank slot 770
// does not key it. The shop opens the shop page. The square has no gate
// actor, and GATES leaves town 3 and keeps it available.
func TestReleaseSecondTownDruidKeeperClicks(t *testing.T) {
	f, app, s, _ := secondTownThree(t, 60)
	c := f.Town.second
	_, sq := druidScene(t, s)
	sq.Click(sq.Hotspot("shop"))
	if c.payload != nil || c.room != secondTownShop {
		t.Fatal("the shop click did not open the shop page", c.room)
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape from the shop page", err, c.room)
	}
	view := s.TownSquareView().Scene
	for _, stage := range []struct {
		bank768, bank770 int32
		suffix           string
	}{{60, 0, "60"}, {60, 1, "60"}, {70, 2, "70"}} {
		c.bank[768], c.bank[770] = stage.bank768, stage.bank770
		for _, p := range []struct {
			row, key string
			b        byte
		}{{"INNKEEPER", "druidinnkeeper", 96}, {"SHOPKEEPER", "druidshopkeeper", 80}} {
			key := p.key + stage.suffix
			want, err := readSecondTownTalk(&f.InstallResources, key)
			if err != nil || want == nil {
				t.Fatalf("installed %s section: %v", key, err)
			}
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
					t.Fatalf("%s by %s at bank768 %d bank770 %d: open %v body %q, want %s", p.row, how, stage.bank768, stage.bank770, open, body, key)
				}
				for app.HeadlessActivate("notice") == nil {
				}
			}
			t.Logf("%s opens %s (%d bytes) at bank770 %d", p.row, key, len(want), stage.bank770)
		}
	}
	for _, name := range []string{"gate", "gates", "door"} {
		if sq.Actor(name) != nil {
			t.Errorf("the druid square runs a %s actor", name)
		}
	}
	if err := app.HeadlessActivate("GATES"); err != nil || c.current != (secondLocation{}) || !c.has(secondLocation{2, 3}) {
		t.Fatalf("GATES: %v current %v available %v", err, c.current, c.available)
	}
}

// The inn is town 1's inn page with the druid centre: TavernMain at (160,0),
// water at (328,144) cycling its first nine files, and the taverner whose
// wait rand()/16+3200 chooses a1, a2 or the still a30001 by (wait&3)+1 with 4
// mapped to 3 (R2-ENGINE-258). The lower-right crop at (464,238) is
// TextBackL (R2-ENGINE-368, DIV-2986). Exit and Escape return to the square.
func TestReleaseSecondTownDruidInnPage(t *testing.T) {
	f, app, s, now := secondTownThree(t, 60)
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
	inn := s.scene(d, "inn")
	for name, n := range map[string]int{"water": 10, "a1": 40, "a2": 30, "a3": 1, "lower-seam": 1, "center": 1} {
		if got := len(art.scene.Pictures(name)); got != n {
			t.Errorf("%s: %d pictures, want %d", name, got, n)
		}
	}
	dst := druidPaint(s)
	for _, probe := range []struct {
		name string
		pic  image.Image
		at   image.Point
		p    image.Point
	}{
		{"centre", art.scene.Pictures("center")[0], image.Pt(160, 0), image.Pt(300, 20)},
		{"water", art.scene.Pictures("water")[0], image.Pt(328, 144), image.Pt(380, 190)},
		{"left stats", art.scene.Pictures("left-stats")[0], image.Pt(0, 0), image.Pt(80, 100)},
		{"buttons area", art.scene.Pictures("upper")[0], image.Pt(480, 0), image.Pt(560, 220)},
	} {
		if !druidProbe(dst, probe.pic, probe.at, probe.p) {
			t.Errorf("%s not drawn at %v", probe.name, probe.at)
		}
	}
	seam := art.scene.Pictures("lower-seam")[0]
	drawn := 0
	for y := 0; y < 242; y++ {
		for x := 0; x < 16; x++ {
			if r, g, b, _ := seam.At(x, y).RGBA(); r|g|b != 0 {
				if !druidProbe(dst, seam, image.Pt(464, 238), image.Pt(464+x, 238+y)) {
					t.Fatalf("TextBackL pixel (%d,%d) not drawn at (464,238)", x, y)
				}
				drawn++
			}
		}
	}
	if drawn == 0 || seam.Bounds().Size() != image.Pt(16, 242) {
		t.Fatalf("TextBackL %v with %d visible pixels", seam.Bounds(), drawn)
	}
	water := inn.Actor("water").(*town.Loop)
	seen := map[int]bool{}
	for i := 0; i < 30; i++ {
		druidStep(s, now, 101*time.Millisecond)
		seen[water.Shown] = true
	}
	if len(seen) != 9 || seen[9] {
		t.Errorf("water showed files %v, want the first nine", seen)
	}
	tav := inn.Actor("taverner").(*town.Alternating)
	for _, want := range []struct {
		delay, state int
	}{{3200, 1}, {3201, 2}, {3202, 3}, {3203, 3}} {
		for tav.State != 0 {
			druidStep(s, now, 101*time.Millisecond)
		}
		if tav.Delay < 3200*time.Millisecond || tav.Delay > 5247*time.Millisecond {
			t.Errorf("taverner wait %v, want 3200..5247 ms", tav.Delay)
		}
		tav.Delay = time.Duration(want.delay) * time.Millisecond
		inn.SetClock("taverner", *now)
		druidStep(s, now, time.Duration(want.delay+1)*time.Millisecond)
		if tav.State != want.state {
			t.Fatalf("wait %d ms started state %d, want %d", want.delay, tav.State, want.state)
		}
		if want.state == 3 {
			still := art.scene.Pictures("a3")[0]
			if !druidProbe(druidPaint(s), still, image.Pt(264, 152), image.Pt(270, 160)) {
				t.Error("a30001 not drawn at centre (104,152)")
			}
			druidStep(s, now, 101*time.Millisecond)
			if tav.State != 0 {
				t.Errorf("the still taverner held past its second step: state %d", tav.State)
			}
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
	t.Logf("inn rows %v; water files %v", rows, seen)
}

// After mission 70 the druid inn offers NPC 675 topic 78; that TALK stores 2
// in bank slot 770, so the inn no longer offers it (R2-ENGINE-331,
// R2-ENGINE-367).
func TestReleaseSecondTownDruidInnTalkStoresBank770(t *testing.T) {
	f, app, s, _ := secondTownThree(t, 80)
	c := f.Town.second
	c.bank[770] = 1
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	s.TownSquareActive(true)
	has := func() bool {
		for _, r := range s.Rows() {
			if r.Text == "TALK 675" {
				return true
			}
		}
		return false
	}
	if !has() {
		t.Fatalf("inn rows %v lack TALK 675 at stage 80 with bank770 1", s.Rows())
	}
	if err := app.HeadlessActivate("TALK 675"); err != nil {
		t.Fatal(err)
	}
	for app.HeadlessActivate("notice") == nil {
	}
	if c.bank[770] != 2 || has() {
		t.Fatalf("after TALK 675: bank770 %d, rows %v", c.bank[770], s.Rows())
	}
}

// A SAV written on the druid square or in the druid inn loads cold into the
// same room of town 3 with the same campaign, party and rows; the town SAV
// holds no World. The next actions, leaving by the gate and entering mission
// 70, give equal Worlds, and so do the eight ticks after them.
func TestReleaseSecondTownDruidColdLoadKeepsTheRoom(t *testing.T) {
	for _, room := range []secondTownRoom{secondTownSquare, secondTownInn} {
		name := map[secondTownRoom]string{secondTownSquare: "Druid square", secondTownInn: "Druid inn"}[room]
		t.Run(name, func(t *testing.T) {
			f, app, _, _ := secondTownThree(t, 60)
			if room == secondTownInn {
				if err := app.HeadlessActivate("TAVERN"); err != nil {
					t.Fatal(err)
				}
			}
			before := secondTownSampleNow(t, f, app)
			out := t.TempDir()
			raw := secondTownNamedSave(t, f, app, out, name)
			secondTownShape(t, raw)
			secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
			cold, coldApp := secondTownCold(t, out, secondSaveFile(t, out, name))
			s := cold.TownScreen().(*secondCampaignScreen)
			coldApp.SetTown(s)
			c := cold.Town.second
			if c.current != (secondLocation{2, 3}) || c.room != room || !s.AtTownSquare() {
				t.Fatalf("cold LOAD current %v room %v composed %v", c.current, c.room, s.AtTownSquare())
			}
			if d, _ := s.composedTown(); d.Town != "rom2-town3" {
				t.Fatalf("cold LOAD shows %s", d.Town)
			}
			secondTownAssertSample(t, before, secondTownSampleNow(t, cold, coldApp))
			for _, a := range []*ui.App{app, coldApp} {
				targets := []string{"GATES", "mission 70"}
				if room == secondTownInn {
					targets = append([]string{secondInnExit}, targets...)
				}
				for _, target := range targets {
					if err := a.HeadlessActivate(target); err != nil {
						t.Fatal(target, err)
					}
				}
			}
			if app.Screen() != ui.ScreenMap || coldApp.Screen() != ui.ScreenMap || f.liveMission != 70 || cold.liveMission != 70 {
				t.Fatalf("mission 70 entry: screens %v/%v missions %d/%d", app.Screen(), coldApp.Screen(), f.liveMission, cold.liveMission)
			}
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "druid town cold departure")
			first := f.live.world.Hash()
			if cold.live.world.Hash() != first {
				t.Fatal("druid town cold departure World hash differs")
			}
			for range 8 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				if err := coldApp.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				assertCurrentWorldEqual(t, f.live.world, cold.live.world, "druid town continuation ticks")
				if f.live.world.Hash() != cold.live.world.Hash() {
					t.Fatal("druid town continuation World hash differs")
				}
			}
			t.Logf("%s: %d-byte SAV; mission 70 World hash %x equal after 8 ticks", name, len(raw), first)
		})
	}
}
