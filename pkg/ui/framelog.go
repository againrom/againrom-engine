package ui

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const FrameLogEnv = "AGAINROM_FRAMELOG"

type frameLog struct {
	app  *App
	out  *os.File
	last time.Time

	updates, draws []float64
	gaps           []float64
	lastDraw       time.Time
	windowStart    time.Time
	fallbackStart  int
	gcStart        runtime.MemStats
}

func newFrameLog(a *App, path string) *frameLog {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil
	}
	l := &frameLog{app: a, out: f, windowStart: time.Now()}
	runtime.ReadMemStats(&l.gcStart)
	fmt.Fprintf(f, "# frame log start screen=%s\n", a.Screen())
	return l
}

func (l *frameLog) Layout(w, h int) (int, int) { return l.app.Layout(w, h) }

func (l *frameLog) Update() error {
	t := time.Now()
	err := l.app.Update()
	l.updates = append(l.updates, float64(time.Since(t))/1e6)
	return err
}

func (l *frameLog) Draw(screen *ebiten.Image) {
	t := time.Now()
	if !l.lastDraw.IsZero() {
		l.gaps = append(l.gaps, float64(t.Sub(l.lastDraw))/1e6)
	}
	l.lastDraw = t
	l.app.Draw(screen)
	l.draws = append(l.draws, float64(time.Since(t))/1e6)
	if t.Sub(l.windowStart) >= time.Second {
		l.flush(t)
	}
}

func frameLogStats(v []float64) (mean, p95, max float64, over16, over33 int) {
	if len(v) == 0 {
		return
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	var sum float64
	for _, x := range s {
		sum += x
		if x > 16.7 {
			over16++
		}
		if x > 33.3 {
			over33++
		}
	}
	return sum / float64(len(s)), s[(len(s)*95)/100%len(s)], s[len(s)-1], over16, over33
}

func (l *frameLog) flush(now time.Time) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	_, _, fallbacks := l.app.TextSettle()
	um, up, ux, _, _ := frameLogStats(l.updates)
	dm, dp, dx, _, _ := frameLogStats(l.draws)
	gm, gp, gx, g16, g33 := frameLogStats(l.gaps)
	fmt.Fprintf(l.out, "screen=%s updates=%d update_mean=%.2f p95=%.2f max=%.2f | draws=%d draw_mean=%.2f p95=%.2f max=%.2f | frame_gap_mean=%.2f p95=%.2f max=%.2f over16.7=%d over33=%d | readback_frames=%d gc=%d gc_pause_ms=%.2f\n",
		l.app.Screen(), len(l.updates), um, up, ux, len(l.draws), dm, dp, dx, gm, gp, gx, g16, g33,
		fallbacks-l.fallbackStart, ms.NumGC-l.gcStart.NumGC, float64(ms.PauseTotalNs-l.gcStart.PauseTotalNs)/1e6)
	l.updates, l.draws, l.gaps = l.updates[:0], l.draws[:0], l.gaps[:0]
	l.windowStart, l.fallbackStart, l.gcStart = now, fallbacks, ms
}

func gameFor(a *App, path string) (ebiten.Game, func()) {
	if path != "" {
		if l := newFrameLog(a, path); l != nil {
			return l, func() { l.out.Close() }
		}
	}
	return a, func() {}
}

func runGame(a *App) error {
	g, closeLog := gameFor(a, os.Getenv(FrameLogEnv))
	defer closeLog()
	return ebiten.RunGame(g)
}
