package ui

import (
	"fmt"
	"image"
	"os"
	"strings"
	"testing"
)

// scrollAnswerInputs is every wheel, key and page input a list is asked over.
func scrollAnswerInputs() []appInput {
	var out []appInput
	keys := []appInput{{}, {Up: true}, {Down: true}, {PageUp: true}, {PageDown: true},
		{Home: true}, {End: true}, {Up: true, Down: true}}
	for _, k := range keys {
		for _, w := range []float64{0, 1, -1, 0.5, -0.5, 2.5, -2.5} {
			k.WheelY = w
			out = append(out, k)
		}
	}
	return out
}

func describeInput(in appInput) string {
	var b strings.Builder
	for _, f := range []struct {
		on   bool
		name string
	}{{in.Up, "U"}, {in.Down, "D"}, {in.PageUp, "PU"}, {in.PageDown, "PD"}, {in.Home, "H"}, {in.End, "E"}} {
		if f.on {
			b.WriteString(f.name)
		}
	}
	return fmt.Sprintf("%s/%g", b.String(), in.WheelY)
}

// scrollScenario is one list over fixture data. build makes the fixture once;
// reset puts the list back to a start position, apply gives it one tick of
// input and state names what the list shows.
type scrollScenario struct {
	name   string
	starts []int
	build  func(t *testing.T) (reset func(start int), apply func(appInput), state func() string)
}

func pickerState(p *Picker) func() string {
	return func() string {
		top, n := p.Visible()
		return fmt.Sprintf("sel=%d top=%d n=%d", p.Selection(), top, n)
	}
}

func scrollScenarios() []scrollScenario {
	return []scrollScenario{
		{"picker screen", []int{0, 1, 20, 39}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a := newTestApp(t, appRows(40), okLoader(t))
			a.flow.screen = ScreenPicker
			return func(start int) {
					a.flow.picker = NewPicker(appRows(40))
					a.flow.picker.Select(start)
				}, func(in appInput) { a.step(in, atAt) },
				func() string { return pickerState(a.flow.picker)() }
		}},
		{"list screen", []int{0, 1, 20, 39}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a := newTestApp(t, appRows(40), okLoader(t))
			var list *Picker
			return func(start int) {
					list = NewPicker(appRows(40))
					list.Select(start)
				}, func(in appInput) { a.stepList(in, list, func() {}) },
				func() string { return pickerState(list)() }
		}},
		{"list keys", []int{0, 1, 20, 39}, func(t *testing.T) (func(int), func(appInput), func() string) {
			var list *Picker
			return func(start int) {
					list = NewPicker(appRows(40))
					list.Select(start)
				}, func(in appInput) { listKey(list, in.Up, in.Down, in.PageUp, in.PageDown) },
				func() string { return pickerState(list)() }
		}},
		{"list bar", []int{0, 1, 20, 39}, func(t *testing.T) (func(int), func(appInput), func() string) {
			var list *Picker
			from := 0
			return func(start int) {
					list = NewPicker(appRows(40))
					list.Select(start)
					from = start
				}, func(in appInput) {
					for _, req := range []barRequest{barLineUp, barLineDown, barPageUp, barPageDown, barSetPos} {
						if (req == barLineUp && in.Up) || (req == barLineDown && in.Down) || (req == barPageUp && in.PageUp) ||
							(req == barPageDown && in.PageDown) || (req == barSetPos && in.Home) {
							listBarRequest(list, req, from/2)
						}
					}
				}, func() string { return pickerState(list)() }
		}},
		{"mod screen", []int{0, 1, 5}, func(t *testing.T) (func(int), func(appInput), func() string) {
			long := strings.Repeat("word ", 400)
			a := modTestApp(t, []ModScreen{{Mod: "m", Key: "k", Kind: ModScreenInfo, Title: "T", MenuLabel: "L", Main: true, Paragraphs: []string{long}}})
			click(a, midOf(modMenuEntryRects(1)[0]))
			return func(start int) { a.flow.modUI.top = min(start, a.modMaxTop()) },
				func(in appInput) { a.step(in, atAt) },
				func() string { return fmt.Sprintf("top=%d max=%d", a.flow.modUI.top, a.modMaxTop()) }
		}},
		{"help text", []int{0, 1, 5}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a, v, now := helpScaledApp(t, 0, 0)
			return func(start int) {
					v.helpSetPos(0)
					v.help.okFocus = false
					v.helpSetPos(start)
				}, func(in appInput) { a.step(in, now) }, func() string {
					first, last := v.HelpScroll()
					return fmt.Sprintf("first=%d last=%d cur=%d ok=%v", first, last, v.help.cur, v.help.okFocus)
				}
		}},
		{"notice window", []int{0, 1, 7, 30}, func(t *testing.T) (func(int), func(appInput), func() string) {
			w := &noticeWindow{}
			l := newNoticeWindowLayout(gameMenuTestFont(), loadPanel, 1)
			const lines = 60
			return func(start int) { w.top = start },
				func(in appInput) { w.scrollStep(in, image.Pt(-1, -1), false, l, lines) },
				func() string { return fmt.Sprintf("top=%d rows=%d", w.top, l.List.Rows) }
		}},
		{"quest objectives", []int{0, 1, 4}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a := newTestApp(t, appRows(2), okLoader(t))
			f := a.flow
			f.menuFont = gameMenuTestFont()
			paras := make([]string, 40)
			for i := range paras {
				paras[i] = fmt.Sprintf("Paragraph %d of the briefing.", i)
			}
			f.menuContext.Objective = strings.Join(paras, "\r\n\r\n")
			return func(start int) { f.questTop = start }, func(in appInput) { a.stepQuestObjectives(in) },
				func() string { return fmt.Sprintf("top=%d lines=%d", f.questTop, len(f.questLines())) }
		}},
		{"ending credits", []int{0, 1, 4}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a, _ := rollTestApp(t, 60)
			a.flow.endingPage = 1
			a.flow.ending.Credits = make([]string, 60)
			return func(start int) { a.flow.endingTop, a.flow.endingFocus = start, 0 },
				func(in appInput) { a.stepEnding(in) },
				func() string { return fmt.Sprintf("top=%d focus=%d", a.flow.endingTop, a.flow.endingFocus) }
		}},
		{"save list focused", []int{0, 1, 20, 39}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a, _, _ := saveChooserStyleApp(t, 40)
			d := a.flow.saveDialog
			rows := d.list.Rows()
			return func(start int) {
					d.list = NewPicker(rows)
					d.setFocus(saveListControl)
					d.list.Select(start)
				}, func(in appInput) { a.step(in, atAt) }, func() string {
					top, n := d.list.Visible()
					return fmt.Sprintf("sel=%d top=%d n=%d focus=%d", d.list.Selection(), top, n, d.focus)
				}
		}},
		{"save list hovered", []int{0, 1, 20, 39}, func(t *testing.T) (func(int), func(appInput), func() string) {
			a, _, _ := saveChooserStyleApp(t, 40)
			d := a.flow.saveDialog
			rows := d.list.Rows()
			at := saveListRect.Min.Add(saveListRect.Max).Div(2)
			return func(start int) {
					d.list = NewPicker(rows)
					d.setFocus(saveNameControl)
					d.list.Select(start)
				}, func(in appInput) {
					in.CursorX, in.CursorY = at.X, at.Y
					a.step(in, atAt)
				}, func() string {
					top, n := d.list.Visible()
					return fmt.Sprintf("sel=%d top=%d n=%d focus=%d", d.list.Selection(), top, n, d.focus)
				}
		}},
		{"editor wheel", []int{0}, func(t *testing.T) (func(int), func(appInput), func() string) {
			var rem float64
			var rows []int
			return func(int) { rem, rows = 0, nil },
				func(in appInput) { rows = append(rows, editorRowsOf(&rem, in.WheelY)) },
				func() string { return fmt.Sprintf("rows=%v rem=%g", rows, rem) }
		}},
	}
}

func scrollAnswers(t *testing.T) string {
	var b strings.Builder
	for _, sc := range scrollScenarios() {
		reset, apply, state := sc.build(t)
		for _, start := range sc.starts {
			for _, in := range scrollAnswerInputs() {
				reset(start)
				apply(in)
				first := state()
				apply(in)
				fmt.Fprintf(&b, "%s | start %d | %s | %s | %s\n", sc.name, start, describeInput(in), first, state())
			}
		}
	}
	return b.String()
}

// TestScrollListAnswers pins the answer of every scrolling list over fixture
// inputs. The golden file was recorded before the lists shared one model; set
// RECORD_SCROLL_ANSWERS to the golden path to record it again.
func TestScrollListAnswers(t *testing.T) {
	got := scrollAnswers(t)
	const golden = "testdata/scrolllist_answers.golden"
	if path := os.Getenv("RECORD_SCROLL_ANSWERS"); path != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range gl {
			if i >= len(wl) || gl[i] != wl[i] {
				w := ""
				if i < len(wl) {
					w = wl[i]
				}
				t.Fatalf("scroll answers differ at line %d:\n got  %s\n want %s", i+1, gl[i], w)
			}
		}
		t.Fatalf("scroll answers differ in length: got %d lines, want %d", len(gl), len(wl))
	}
}
