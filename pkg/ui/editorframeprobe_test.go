package ui_test

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/game"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The editor frame-time probe. It runs the shipped editor, built through
// the same opener cmd/mapedit uses, inside an offscreen unfocused window at
// 1920x1080 and times Update and Draw per frame, feeding input only through
// the editor's input seam. Gate: AGAINROM_FRAME_PROBE=1.
//
//	AGAINROM_ASSETS     install root
//	AGAINROM_FRAME_OUT  directory for the raw files
//	AGAINROM_FRAME_TAG  file name prefix
//	AGAINROM_FRAME_VSYNC 0 turns vsync off and syncs ticks with frames
//	AGAINROM_FRAME_PLAN comma list of: cases, actions, real
//	AGAINROM_FRAME_PROFILE phase names to CPU-profile
//	AGAINROM_FRAME_ONLY  run only this action, looped for AGAINROM_FRAME_LOOP seconds

const (
	probeW = 1920
	probeH = 1080
)

type probeRec struct {
	upd, draw, period time.Duration // upd: every Update since the previous Draw; period: Draw end to Draw end
	ticks             int
	mallocs, bytes    uint64
	pause             time.Duration
}

type probePhase struct {
	name   string
	setup  func()
	hook   func() // runs inside the timed Update of the first recorded tick
	input  func(i int) ui.ProbeInput
	done   func() bool // the response
	done2  func() bool // a second response
	settle int         // ticks run before the script starts and recording begins
	frames int         // recorded frames
	tail   int         // with done: recorded frames after the response, then stop
	loop   time.Duration
	extra  string

	begun, over       bool
	tick, frame, n    int
	resp, resp2       int
	respT, resp2T     int
	setupDur          time.Duration
	recs              []probeRec
	t0, t1            time.Time
	tps1, fps1        float64
	profile           *os.File
	cycles            int
	respAll, resp2All []string
}

type probeGame struct {
	e        *ui.MapEditor
	phases   []*probePhase
	cur      int
	in       ui.ProbeInput
	lastDraw time.Time
	updAcc   time.Duration
	ticks    int
	msPrev   runtime.MemStats
	profiles map[string]bool
	outDir   string
	tag      string
	err      error
	gfx      string
}

func (g *probeGame) Layout(_, _ int) (int, int) {
	g.e.Layout(probeW, probeH)
	return probeW, probeH
}

func (g *probeGame) begin(p *probePhase) {
	t := time.Now()
	if p.setup != nil {
		p.setup()
	}
	p.setupDur += time.Since(t)
	p.begun, p.tick, p.frame, p.n, p.resp, p.resp2, p.respT, p.resp2T = true, 0, 0, 0, 0, 0, 0, 0
	if p.t0.IsZero() {
		p.t0 = time.Now()
		if g.profiles[p.name] {
			f, err := os.Create(filepath.Join(g.outDir, g.tag+"-prof-"+p.name+".pprof"))
			if err == nil {
				if pprof.StartCPUProfile(f) == nil {
					p.profile = f
				} else {
					f.Close()
				}
			}
		}
	}
	runtime.ReadMemStats(&g.msPrev)
	g.updAcc, g.ticks = 0, 0
	g.lastDraw = time.Now()
}

func (g *probeGame) Update() error {
	for g.cur < len(g.phases) && g.phases[g.cur].over {
		p := g.phases[g.cur]
		p.t1 = time.Now()
		p.tps1, p.fps1 = ebiten.ActualTPS(), ebiten.ActualFPS()
		if p.profile != nil {
			pprof.StopCPUProfile()
			p.profile.Close()
		}
		g.cur++
	}
	if g.cur >= len(g.phases) {
		return ebiten.Termination
	}
	p := g.phases[g.cur]
	if !p.begun {
		g.begin(p)
	}
	if g.gfx == "" {
		var dbg ebiten.DebugInfo
		ebiten.ReadDebugInfo(&dbg)
		g.gfx = fmt.Sprintf("%v", dbg.GraphicsLibrary)
	}
	g.in = ui.ProbeInput{}
	if p.input != nil {
		g.in = p.input(p.tick)
	}
	t := time.Now()
	if p.hook != nil && p.tick == p.settle {
		p.hook()
	}
	err := g.e.Update()
	g.updAcc += time.Since(t)
	g.ticks++
	p.tick++
	return err
}

func (g *probeGame) Draw(screen *ebiten.Image) {
	if g.cur >= len(g.phases) || g.phases[g.cur].over || !g.phases[g.cur].begun {
		g.e.Draw(screen)
		return
	}
	p := g.phases[g.cur]
	t := time.Now()
	g.e.Draw(screen)
	end := time.Now()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	p.frame++
	rec := probeRec{upd: g.updAcc, draw: end.Sub(t), period: end.Sub(g.lastDraw), ticks: g.ticks,
		mallocs: m1.Mallocs - g.msPrev.Mallocs, bytes: m1.TotalAlloc - g.msPrev.TotalAlloc,
		pause: time.Duration(m1.PauseTotalNs - g.msPrev.PauseTotalNs)}
	g.msPrev, g.updAcc, g.ticks, g.lastDraw = m1, 0, 0, time.Now()
	if p.tick <= p.settle {
		return
	}
	p.recs = append(p.recs, rec)
	p.n++
	if p.done != nil && p.resp == 0 && p.done() {
		p.resp, p.respT = p.n, p.tick-p.settle
	}
	if p.done2 != nil && p.resp2 == 0 && p.done2() {
		p.resp2, p.resp2T = p.n, p.tick-p.settle
	}
	stop := p.n >= p.frames
	if p.done != nil && p.tail > 0 && p.resp > 0 && p.n-p.resp >= p.tail {
		stop = true
	}
	if !stop {
		return
	}
	p.respAll = append(p.respAll, fmt.Sprintf("%df/%dt", p.resp, p.respT))
	p.resp2All = append(p.resp2All, fmt.Sprintf("%df/%dt", p.resp2, p.resp2T))
	p.cycles++
	if p.loop > 0 && time.Since(p.t0) < p.loop {
		p.begun = false
		g.begin(p)
	} else {
		p.over = true
	}
}

func us(d time.Duration) float64 { return float64(d) / 1e3 }

func pct(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := slices.Clone(v)
	sort.Float64s(s)
	i := int(math.Ceil(q*float64(len(s)))) - 1
	return s[max(0, min(len(s)-1, i))]
}

func (p *probePhase) summary() string {
	var tot, upd, drw, per, tick []float64
	var mal, byt, pause float64
	for _, r := range p.recs {
		tot = append(tot, us(r.upd+r.draw)/1e3)
		tick = append(tick, float64(r.ticks))
		upd = append(upd, us(r.upd)/1e3)
		drw = append(drw, us(r.draw)/1e3)
		per = append(per, us(r.period)/1e3)
		mal += float64(r.mallocs)
		byt += float64(r.bytes)
		pause += us(r.pause)
	}
	n := float64(max(1, len(p.recs)))
	wall := p.t1.Sub(p.t0).Seconds()
	return fmt.Sprintf("%-34s n=%-5d U+D ms med=%.3f p95=%.3f max=%.3f | upd med=%.3f draw med=%.3f | period ms med=%.2f p95=%.2f max=%.2f | ticks/frame mean=%.2f max=%.0f | allocs/frame=%.0f B/frame=%.0f gcpause us/frame=%.1f | setup ms=%.1f wall s=%.2f ActualTPS=%.1f ActualFPS=%.1f resp=%v resp2=%v %s",
		p.name, len(p.recs), pct(tot, .5), pct(tot, .95), pct(tot, 1), pct(upd, .5), pct(drw, .5), pct(per, .5), pct(per, .95), pct(per, 1),
		mean(tick), pct(tick, 1), mal/n, byt/n, pause/n, us(p.setupDur)/1e3, wall, p.tps1, p.fps1, p.respAll, p.resp2All, p.extra)
}

type probeMap struct {
	key           string
	cells, record int
	label         string
}

func probeEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func TestEditorFrameProbe(t *testing.T) {
	if os.Getenv("AGAINROM_FRAME_PROBE") != "1" {
		t.Skip("AGAINROM_FRAME_PROBE=1 required")
	}
	root := os.Getenv("AGAINROM_ASSETS")
	out := probeEnv("AGAINROM_FRAME_OUT", t.TempDir())
	tag := probeEnv("AGAINROM_FRAME_TAG", "probe")
	plan := strings.Split(probeEnv("AGAINROM_FRAME_PLAN", "cases,actions,real"), ",")
	only := os.Getenv("AGAINROM_FRAME_ONLY")
	vsync := probeEnv("AGAINROM_FRAME_VSYNC", "1") != "0"
	if root == "" {
		t.Skip("no AGAINROM_ASSETS")
	}
	x, err := game.NewMapInspector(root)
	if err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	e.SetPanelProfile(nil, os.ErrNotExist, nil) // a fresh profile, as cmd/mapedit passes one without a panel file
	e.Layout(probeW, probeH)

	// The campaign maps: cells and records of each.
	var maps []probeMap
	for _, m := range x.Maps {
		if !m.FromArchive {
			continue
		}
		key := ui.MapSourceKey(ui.SourceScenario, m.Source)
		if err := e.Open(key); err != nil {
			continue
		}
		d := e.Document()
		maps = append(maps, probeMap{key: key, cells: d.Width * d.Height, record: len(d.Records), label: fmt.Sprintf("%s %dx%d", m.Source, d.Width, d.Height)})
	}
	if len(maps) == 0 {
		t.Fatal("no campaign maps")
	}
	var listing strings.Builder
	for _, m := range maps {
		fmt.Fprintf(&listing, "%s\tcells=%d\trecords=%d\n", m.label, m.cells, m.record)
	}
	_ = os.WriteFile(filepath.Join(out, tag+"-maps.txt"), []byte(listing.String()), 0o644)
	byCells := slices.Clone(maps)
	sort.SliceStable(byCells, func(i, j int) bool { return byCells[i].cells > byCells[j].cells })
	byRec := slices.Clone(maps)
	sort.SliceStable(byRec, func(i, j int) bool { return byRec[i].record > byRec[j].record })
	biggest, mostRec, mid := byCells[0], byRec[0], byCells[len(byCells)/2]
	// A mid-size map: median by cells among those with a median-ish record count.
	newMap := probeMap{key: game.NewMapSource("256x256"), cells: 256 * 256, label: "new 256x256"}

	g := &probeGame{e: e, outDir: out, tag: tag, profiles: map[string]bool{}}
	for _, n := range strings.Split(os.Getenv("AGAINROM_FRAME_PROFILE"), ",") {
		if n != "" {
			g.profiles[n] = true
		}
	}
	idle := func() ui.ProbeInput {
		r := e.MapRect()
		return ui.ProbeInput{Canvas: ui.Input{CursorX: (r.Min.X + r.Max.X) / 2, CursorY: (r.Min.Y + r.Max.Y) / 2}}
	}
	ctr := func() image.Point { r := e.MapRect(); return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2) }
	ui.ProbeSetInput(e, func() ui.ProbeInput { return g.in })
	has := func(s string) bool { return slices.Contains(plan, s) }

	openMap := func(m probeMap) func() {
		return func() {
			for range 3 {
				_ = e.Key("escape")
			}
			if err := e.Open(m.key); err != nil {
				g.err = err
			}
			e.Layout(probeW, probeH)
		}
	}
	openFull := func(m probeMap) func() {
		return func() {
			openMap(m)()
			d := e.Document()
			e.ShowCell(image.Pt(d.Width/2, d.Height/2))
		}
	}
	zoomNote := func() string { _, _, z := ui.ProbeCamera(e); return fmt.Sprintf("zoom=%.3f", z) }
	var phases []*probePhase
	add := func(p *probePhase) *probePhase {
		if p.input == nil {
			p.input = func(int) ui.ProbeInput { return idle() }
		}
		old, base := p.setup, p.extra
		p.setup = func() {
			if old != nil {
				old()
			}
			if e.Document() != nil {
				x, y, z := ui.ProbeCamera(e)
				p.extra = base + fmt.Sprintf(" [zoom=%.3f cam=%.0f,%.0f map=%v]", z, x, y, e.MapRect())
			}
		}
		phases = append(phases, p)
		return p
	}

	if has("cases") && only == "" {
		cases := []probeMap{biggest, mostRec, mid, newMap}
		seen := map[string]bool{}
		for _, m := range cases {
			if seen[m.key] {
				continue
			}
			seen[m.key] = true
			label := m.label
			add(&probePhase{name: "open " + label, setup: openMap(m), frames: 1, settle: 0, extra: fmt.Sprintf("cells=%d records=%d", m.cells, m.record)})
			p := add(&probePhase{name: "idle-default " + label, settle: 30, frames: 150})
			pp := p
			pp.setup = func() { pp.extra = zoomNote() }
			f := add(&probePhase{name: "idle-full " + label, settle: 20, frames: 150})
			ff := f
			ff.setup = func() { e.ShowCell(image.Pt(e.Document().Width/2, e.Document().Height/2)); ff.extra = zoomNote() }
			q := add(&probePhase{name: "idle-widest " + label, settle: 20, frames: 150})
			qq := q
			qq.setup = func() { e.Fit(); qq.extra = zoomNote() }
		}
	}

	// Actions on the biggest map (map-oriented) and a dialogue map.
	var dlgPage uint32
	if has("actions") || only != "" {
		units := []int{}
		openFull(biggest)()
		for i, r := range e.Document().Records {
			if strings.EqualFold(r.Kind.String(), "unit") && r.Spatial {
				units = append(units, i)
			}
		}
		recCell := func(i int) image.Point { return e.Document().Records[i].Cell }
		if len(units) >= 2 {
			u0, u1 := units[0], units[len(units)/2]
			var startX, startY, startZ float64
			snap := func() { startX, startY, startZ = ui.ProbeCamera(e) }
			moved := func() bool { a, b, _ := ui.ProbeCamera(e); return a != startX || b != startY }
			act := func(a *probePhase) {
				a.settle, a.frames = 5, 60
				if only != "" {
					if a.name != only {
						return
					}
					a.loop = time.Duration(atoi(probeEnv("AGAINROM_FRAME_LOOP", "10"))) * time.Second
				}
				add(a)
			}
			reset := func() {
				for range 3 {
					_ = e.Key("escape")
				}
				e.Layout(probeW, probeH)
			}
			act(&probePhase{name: "act-pan-drag", setup: func() { openFull(biggest)(); snap() }, tail: 3,
				input: func(i int) ui.ProbeInput {
					c := ctr()
					in := ui.Input{CursorX: c.X + i*6, CursorY: c.Y + i*3, MiddleDown: i >= 5}
					if i < 5 {
						in.CursorX, in.CursorY = c.X, c.Y
					} else {
						in.CursorX, in.CursorY = c.X+(i-5)*6, c.Y+(i-5)*3
					}
					return ui.ProbeInput{Canvas: in}
				}, done: moved})
			act(&probePhase{name: "act-pan-keys", setup: func() { openFull(biggest)(); snap() }, tail: 3,
				input: func(i int) ui.ProbeInput {
					in := idle()
					in.Canvas.PanRight = i >= 5
					in.Held = nil
					if i >= 5 {
						in.Held = []ebiten.Key{ebiten.KeyArrowRight}
					}
					return in
				}, done: moved})
			act(&probePhase{name: "act-wheel-zoom", setup: func() { openFull(biggest)(); snap() }, tail: 3,
				input: func(i int) ui.ProbeInput {
					in := idle()
					if i >= 5 && i < 12 {
						in.Canvas.WheelY = 1
					}
					return in
				}, done: func() bool { _, _, z := ui.ProbeCamera(e); return z != startZ }})
			act(&probePhase{name: "act-hover-unit", setup: func() { openFull(biggest)(); e.ShowCell(recCell(u0)) }, tail: 2,
				input: func(i int) ui.ProbeInput {
					p := e.CellPoint(recCell(u0))
					if i < 5 {
						p = ctr().Add(image.Pt(-300, -200))
					}
					return ui.ProbeInput{Canvas: ui.Input{CursorX: p.X, CursorY: p.Y}}
				}})
			act(&probePhase{name: "act-hover-toolbar-tip", setup: func() { openFull(biggest)() }, tail: 2,
				input: func(i int) ui.ProbeInput {
					in := idle()
					for _, k := range []string{"open", "new", "saveas", "catalog", "testplay", "terrain", "unit", "help"} {
						if r, ok := e.ToolbarRect(k); ok {
							in.Canvas.CursorX, in.Canvas.CursorY = (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
							break
						}
					}
					return in
				}, done: func() bool { return e.TipShown() != "" }})
			act(&probePhase{name: "act-hover-sweep", setup: func() { openFull(biggest)() },
				input: func(i int) ui.ProbeInput {
					c := ctr()
					a := 2 * math.Pi * float64(i) / 60
					return ui.ProbeInput{Canvas: ui.Input{CursorX: c.X + int(500*math.Cos(a)), CursorY: c.Y + int(300*math.Sin(a))}}
				}})
			prev := -1
			act(&probePhase{name: "act-click-select", setup: func() { openFull(biggest)(); e.ShowCell(recCell(u1)); prev = e.Selected() }, tail: 3,
				input: func(i int) ui.ProbeInput {
					p := e.CellPoint(recCell(u1))
					return ui.ProbeInput{Canvas: ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: i >= 5 && i < 6}}
				}, done: func() bool { return e.Selected() >= 0 && e.Selected() != prev },
				done2: func() bool { return ui.ProbeWindowOpen(e) }})
			act(&probePhase{name: "act-palette-terrain-key", setup: func() { openFull(biggest)(); reset() }, tail: 3,
				input: func(i int) ui.ProbeInput {
					in := idle()
					if i == 5 {
						in.Key, in.HasKey = ebiten.KeyT, true
					}
					return in
				}, done: func() bool { return ui.ProbeListOpen(e) }, done2: func() bool { return ui.ProbeWindowOpen(e) }})
			var top0 int
			act(&probePhase{name: "act-scripts-tree-scroll", setup: func() {
				openFull(biggest)()
				reset()
				top0, _ = ui.ProbeTreeTop(e)
			}, tail: 3,
				input: func(i int) ui.ProbeInput {
					in := idle()
					if r, ok := e.TreeRowRect(0); ok {
						in.Canvas.CursorX, in.Canvas.CursorY = (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
					}
					if i >= 5 && i < 8 {
						in.Canvas.WheelY = -3
					}
					return in
				}, done: func() bool { t, _ := ui.ProbeTreeTop(e); return t != top0 }})
			act(&probePhase{name: "act-scripts-key-open", setup: func() { openFull(biggest)(); reset() }, tail: 3,
				input: func(i int) ui.ProbeInput {
					in := idle()
					if i == 5 {
						in.Key, in.HasKey = ebiten.KeyF2, true
					}
					return in
				}, done: func() bool { return ui.ProbeListOpen(e) }})
		}
		// Dialogue map for the message popup and the speaker pick.
		dkey := "scenario/130.alm"
		for _, m := range maps {
			if strings.HasSuffix(m.key, "130.alm") {
				dkey = m.key
			}
		}
		dm := probeMap{key: dkey, label: dkey}
		openMap(dm)()
		if host, ok := e.Document().Edits.(ui.InspectionMessages); ok {
			for p := uint32(1); p < 400 && dlgPage == 0; p++ {
				if len(host.MessageFields(p)) >= 2 {
					dlgPage = p
				}
			}
		}
		if dlgPage != 0 {
			act2 := func(a *probePhase) {
				a.settle, a.frames, a.tail = 5, 60, 3
				if only != "" {
					if a.name != only {
						return
					}
					a.loop = time.Duration(atoi(probeEnv("AGAINROM_FRAME_LOOP", "10"))) * time.Second
				}
				add(a)
			}
			act2(&probePhase{name: "act-message-popup-open", setup: func() { openMap(dm)() },
				hook: func() { ui.ProbeOpenMessage(e, dlgPage) }, input: func(int) ui.ProbeInput { return idle() },
				done: func() bool { return ui.ProbeMessagePage(e) == dlgPage }, extra: fmt.Sprintf("map=%s page=%d", dkey, dlgPage)})
			act2(&probePhase{name: "act-speaker-pick-arm", setup: func() {
				openMap(dm)()
				ui.ProbeOpenMessage(e, dlgPage)
			},
				input: func(i int) ui.ProbeInput {
					in := idle()
					if x, y, ok := ui.ProbeSpeakerPickButton(e, 0); ok && i >= 5 && i < 6 {
						in.Canvas.CursorX, in.Canvas.CursorY, in.Canvas.PrimaryDown = x, y, true
					} else if ok && i >= 5 {
						in.Canvas.CursorX, in.Canvas.CursorY = x, y
					}
					return in
				}, done: func() bool { return ui.ProbeSpeakerPickArmed(e) }, extra: fmt.Sprintf("map=%s page=%d", dkey, dlgPage)})
		}
	}
	phases = reorderForPlan(phases)

	if (has("real") && only == "") || os.Getenv("AGAINROM_FRAME_REAL") == "1" {
		add(&probePhase{name: "real-open", setup: openFull(biggest), frames: 1})
		add(&probePhase{name: "real-idle", settle: 60, frames: 600, extra: "map=" + biggest.label})
		pa := &probePhase{name: "real-pan", settle: 20, frames: 600, extra: "map=" + biggest.label}
		pa.input = func(i int) ui.ProbeInput {
			c := ctr()
			a := 2 * math.Pi * float64(i) / 150
			return ui.ProbeInput{Canvas: ui.Input{CursorX: c.X + int(250*math.Cos(a)), CursorY: c.Y + int(150*math.Sin(a)), MiddleDown: true}}
		}
		add(pa)
	}
	g.phases = phases

	state := fmt.Sprintf("NumCPU=%d GOMAXPROCS=%d GOMAXPROCS-env=%q vsync=%v biggest=%s mostRecords=%s mid=%s plan=%v only=%q\n",
		runtime.NumCPU(), runtime.GOMAXPROCS(0), os.Getenv("GOMAXPROCS"), vsync, biggest.label, mostRec.label+fmt.Sprintf(" rec=%d", mostRec.record), mid.label, plan, only)

	ebiten.SetWindowSize(probeW, probeH)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetWindowTitle("editor frame probe")
	ebiten.SetRunnableOnUnfocused(true)
	if !vsync {
		ebiten.SetVsyncEnabled(false)
		ebiten.SetTPS(ebiten.SyncWithFPS)
	}
	start := time.Now()
	err = ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true})
	if err != nil && err != ebiten.Termination {
		t.Fatal(err)
	}
	var sum strings.Builder
	sum.WriteString(state)
	fmt.Fprintf(&sum, "graphics library=%s run wall=%v\n", g.gfx, time.Since(start).Round(time.Second))
	var csv strings.Builder
	csv.WriteString("phase,frame,upd_us,draw_us,period_us,mallocs,bytes,gcpause_us\n")
	for _, p := range g.phases {
		sum.WriteString(p.summary() + "\n")
		for i, r := range p.recs {
			fmt.Fprintf(&csv, "%s,%d,%.1f,%.1f,%.1f,%d,%d,%.1f\n", p.name, i, us(r.upd), us(r.draw), us(r.period), r.mallocs, r.bytes, us(r.pause))
		}
	}
	if g.err != nil {
		fmt.Fprintf(&sum, "ERROR %v\n", g.err)
	}
	_ = os.WriteFile(filepath.Join(out, tag+"-summary.txt"), []byte(sum.String()), 0o644)
	_ = os.WriteFile(filepath.Join(out, tag+"-frames.csv"), []byte(csv.String()), 0o644)
	t.Log("\n" + sum.String())
}

func reorderForPlan(p []*probePhase) []*probePhase { return p }

func atoi(s string) int {
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n
}

func mean(v []float64) float64 {
	t := 0.0
	for _, x := range v {
		t += x
	}
	return t / float64(max(1, len(v)))
}
