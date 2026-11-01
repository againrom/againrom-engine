package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/textfit"
)

// dockSizes are the window sizes the docked report is proved at.
var dockSizes = []image.Point{{640, 480}, {1024, 768}, {1280, 800}, {1920, 1080}}

// textOfLines is a finding text that wraps to exactly n lines at width.
func textOfLines(t *testing.T, e *MapEditor, n, width int) string {
	t.Helper()
	text := ""
	for i := 0; ; i++ {
		next := strings.TrimSpace(fmt.Sprintf("%s word%d", text, i))
		if len(editorTextLines(e.bodyFont(), next, width)) > n {
			if len(editorTextLines(e.bodyFont(), text, width)) != n {
				t.Fatalf("no text wraps to %d lines at %d", n, width)
			}
			return text
		}
		text = next
	}
}

// statusPixels is the status line of the composed chrome.
func statusPixels(e *MapEditor) []byte {
	img, r := e.ChromeImage(), e.statusRect()
	var out []byte
	for y := r.Min.Y; y < r.Max.Y; y++ {
		out = append(out, img.Pix[img.PixOffset(r.Min.X, y):img.PixOffset(r.Max.X, y)]...)
	}
	return out
}

// The docked report stays inside the bottom panel: for a finding of one,
// two and five lines at the panel's least and default heights the frame
// lies inside the panel, the status line draws as with no report, the
// notes wrap at the panel's width, and every note line gets a screen row,
// scrolled to by the wheel over the notes where they do not all show.
func TestMapEditorDockedReportStaysInsideTheBottomPanel(t *testing.T) {
	for _, size := range dockSizes {
		for _, lang := range []string{"en", "ru"} {
			for _, lines := range []int{1, 2, 5} {
				for _, height := range []int{sidePanels[panelBottom].least, sidePanels[panelBottom].width} {
					t.Run(fmt.Sprintf("%dx%d/%s/%dlines/%d", size.X, size.Y, lang, lines, height), func(t *testing.T) {
						e, _, _ := panelsEditor(t, size.X, size.Y, lang)
						e.panels.state[panelBottom] = panelState{Restored: true, Width: height}
						e.relayout()
						panel := e.sidePanelRect(panelBottom)
						if panel.Dy() != height {
							t.Fatalf("the panel is %v, want %d high", panel, height)
						}
						quiet := statusPixels(e)
						mark := e.bodyFont().Advance(textfit.Mark)
						width := panel.Dx() - 2*popupPad - 8 - mark
						text := textOfLines(t, e, lines, width)
						e.openReport([]InspectionFinding{
							{Error: true, Rule: "script-reference", Kind: "Unit", Index: 0, Cell: image.Pt(10, 10), HasCell: true, Text: text},
							{Rule: "unit-terrain", Kind: "Unit", Index: 1, Cell: image.Pt(12, 10), HasCell: true, Text: "Unit 1 stands on water."},
						}, "Report: 1 errors, 1 warnings", nil)
						if !e.reportDocked() {
							t.Fatal("the report does not stand in the bottom panel")
						}
						want := editorTextLines(e.bodyFont(), text, width)
						seen := make([]bool, len(want))
						for step := 0; step <= len(want)+2; step++ {
							s, ok := e.listSpec()
							if !ok {
								t.Fatal("no report popup")
							}
							l := layoutPopup(s, e.font, e.bodyFont())
							if !l.Rect.In(panel) {
								t.Fatalf("step %d: the report's frame %v leaves the panel %v", step, l.Rect, panel)
							}
							if got := statusPixels(e); string(got) != string(quiet) {
								t.Fatalf("step %d: the report changes the status line", step)
							}
							for _, note := range s.Notes {
								for i, w := range want {
									if note == w || note == w+textfit.Mark {
										seen[i] = true
									}
								}
							}
							if len(l.Notes) == 0 {
								t.Fatal("the report shows no note line")
							}
							p := center(l.Notes[0])
							e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -1}, time.Unix(1, 0))
						}
						for i, ok := range seen {
							if !ok {
								t.Errorf("note line %d %q never shows; lines %q", i, want[i], want)
							}
						}
					})
				}
			}
		}
	}
}

// checkReportShown reports that the report shows the host's check findings.
func checkReportShown(e *MapEditor) bool {
	c, ok := e.checker()
	if !ok || !e.report.open {
		return false
	}
	errs, warnings := CountFindings(c.Report())
	return e.report.note == fmt.Sprintf("Report: %d errors, %d warnings", errs, warnings) && len(e.report.rows) == len(c.Report())
}

// The Report button shows on only while the check report shows; pressed,
// or R, it opens the check report in place of the F1 key list or the MCP
// connection list and closes the check report itself, with the bottom
// panel docked and minimized.
func TestMapEditorReportButtonTellsTheCheckReportFromOtherLists(t *testing.T) {
	lists := []struct {
		name  string
		open  func(e *MapEditor)
		check bool
	}{
		{"none", func(*MapEditor) {}, false},
		{"check", func(e *MapEditor) {
			c, _ := e.checker()
			errs, warnings := CountFindings(c.Report())
			e.openReport(c.Report(), fmt.Sprintf("Report: %d errors, %d warnings", errs, warnings), nil)
		}, true},
		{"keys", func(e *MapEditor) { _ = e.Key("help") }, false},
		{"mcp", func(e *MapEditor) { e.SetMCPConnection("MCP on", MCPDetail{Label: "Port: ", Value: "4711"}) }, false},
	}
	for _, size := range dockSizes {
		for _, lang := range []string{"en", "ru"} {
			for _, docked := range []bool{true, false} {
				for _, list := range lists {
					for _, how := range []string{"button", "R"} {
						t.Run(fmt.Sprintf("%dx%d/%s/docked=%v/%s/%s", size.X, size.Y, lang, docked, list.name, how), func(t *testing.T) {
							e, _, _ := panelsEditor(t, size.X, size.Y, lang)
							e.panels.state[panelBottom].Restored = docked
							e.relayout()
							list.open(e)
							if list.name != "none" && !e.report.open {
								t.Fatal("the list did not open")
							}
							if checkReportShown(e) != list.check {
								t.Fatalf("the list shows the check report %v, want %v", !list.check, list.check)
							}
							if on, off, _ := e.ToolbarState("report"); on != list.check || off {
								t.Fatalf("the Report button on=%v disabled=%v, want on=%v", on, off, list.check)
							}
							if how == "R" {
								_ = e.Key("report")
							} else {
								modesClick(e, toolPoint(t, e, "report"))
							}
							if list.check {
								if e.report.open {
									t.Fatal("the press left the check report open")
								}
								if e.panelOpen(panelBottom) {
									t.Fatal("the press left the bottom panel restored")
								}
							} else {
								if !checkReportShown(e) || !e.reportDocked() {
									t.Fatalf("the press did not show the check report in the bottom panel: open=%v note=%q docked=%v", e.report.open, e.report.note, e.reportDocked())
								}
							}
							if on, _, _ := e.ToolbarState("report"); on != checkReportShown(e) {
								t.Fatalf("after the press the Report button on=%v, the check report shows %v", on, checkReportShown(e))
							}
						})
					}
				}
			}
		}
	}
}
