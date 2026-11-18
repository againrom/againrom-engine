package ui

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"strings"
	"testing"

	"againrom/pkg/render/textfit"
)

func TestDescriptionPanelPreviews(t *testing.T) {
	window := image.Pt(1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		font := wordsFont(lang)
		long := strings.Repeat("Northern bridge and guard ", 12)
		if lang == "ru" {
			long = strings.Repeat("Северный мост и стража ", 12)
		}
		long = strings.TrimSpace(long)
		check := func(t *testing.T, d popupDesc) *EditorMockup {
			t.Helper()
			m, err := NewEditorMockup(d.encode(), lang, window, font, font, mockupHostEncoder(font))
			if err != nil {
				t.Fatal(err)
			}
			page := m.Page()
			round, err := PageDescription(page)
			if err != nil || !bytes.Equal(round, d.encode()) {
				t.Fatalf("round trip: %v", err)
			}
			for _, r := range m.layout.Rows {
				if !r.Rect.In(m.layout.Rect) || font.Advance(m.paint.Rows[r.Index].Label) > r.Label.Dx()-2 {
					t.Fatalf("row %d: label or rectangle does not fit", r.Index)
				}
			}
			tip := m.previewTip()
			if d.Preview.Hover != nil {
				if tip.Rect.Empty() || !tip.Rect.In(image.Rectangle{Max: window}) {
					t.Fatalf("tip lies outside editor: %v", tip.Rect)
				}
				if d.Preview.Hover.Part != "titleHelp" && tip.Rect.Dx() != popupTipW {
					t.Fatalf("settled tip width = %d", tip.Rect.Dx())
				}
				box := fmt.Sprintf(`data-part="tip" data-box="%d %d %d %d"`, tip.Rect.Min.X, tip.Rect.Min.Y, tip.Rect.Dx(), tip.Rect.Dy())
				if !bytes.Contains(page, []byte(box)) || !bytes.Contains(page, []byte(html.EscapeString(tip.Full))) {
					t.Fatal("page lost the tip's rectangle or text")
				}
			}
			if !m.layout.Rect.In(image.Rectangle{Max: window}) {
				t.Fatalf("panel %v escapes editor", m.layout.Rect)
			}
			return m
		}
		t.Run(lang+"/map", func(t *testing.T) {
			d := popupDesc{Window: "map", Title: "editor.panel.title", TitleHelp: "editor.panel.hint",
				Preview: &popupPreview{MaxWidth: 460, MaxHeight: 192},
				Sample:  &popupScope{Values: map[string]string{"name": long}}}
			for range 11 {
				d.Rows = append(d.Rows, popupDescRow{Kind: "link", Label: "editor.panel.name", Value: "name"})
			}
			first := check(t, d)
			if first.layout.RowBar.Empty() || len(first.layout.Rows) >= len(d.Rows) || first.layout.Rows[0].Index != 0 {
				t.Fatal("body does not page from its first row")
			}
			d.Preview.Top = len(d.Rows) - 1
			d.Preview.Hover = &popupPreviewHover{Part: "row", Row: len(d.Rows) - 1}
			last := check(t, d)
			if last.layout.Title != first.layout.Title || last.layout.Rows[0].Index == 0 || !strings.Contains(last.previewTip().Full, long) {
				t.Fatal("last page does not retain its title or the full link value")
			}
		})
		findings := []popupScope{{Values: map[string]string{"caption": long}, Flags: []string{"current"}}}
		for i := range 65 {
			findings = append(findings, popupScope{Values: map[string]string{"caption": fmt.Sprintf("W message-overlap Trigger #%d", i+1)}})
		}
		t.Run(lang+"/objects", func(t *testing.T) {
			d := popupDesc{Window: "objects", Title: "editor.side.objects", Preview: &popupPreview{Dock: "left", Hover: &popupPreviewHover{Part: "list", Row: 0}},
				List:   &popupDescList{Each: "rows", Bind: "caption", Chosen: "current", Open: "expanded", Heading: "expanded"},
				Rows:   []popupDescRow{{Kind: "edit", Label: "editor.side.search", Hint: "editor.side.searchhint"}},
				Sample: &popupScope{Lists: map[string][]popupScope{"rows": findings}}}
			m := check(t, d)
			if m.layout.Rect != m.paint.Bounds || m.layout.Rect.Min.X != 0 || m.layout.ListRows >= len(findings) {
				t.Fatalf("left dock or paging differs: %v, %d", m.layout.Rect, m.layout.ListRows)
			}
			if part, i := m.layout.hit(center(m.layout.List.Row(0))); part != popupListRow || i != 0 {
				t.Fatalf("first list row hit %v %d", part, i)
			}
			if m.previewTip().Full != long || !bytes.Contains(m.Page(), []byte(textfit.Mark+"</div>")) {
				t.Fatal("list clipping lost the fitted text or its full tip")
			}
		})
		t.Run(lang+"/report", func(t *testing.T) {
			notes := make([]popupScope, 5)
			for i := range notes {
				notes[i].Words = map[string]string{"note": "editor.panel.defeat.none"}
			}
			d := popupDesc{Window: "report", Title: "editor.side.report", Preview: &popupPreview{Dock: "bottom", DockSize: 320},
				List: &popupDescList{Each: "findings", Bind: "caption"}, Notes: []popupDescNote{{Each: "details", Bind: "note"}},
				Sample: &popupScope{Lists: map[string][]popupScope{"findings": findings, "details": notes}}}
			m := check(t, d)
			if m.layout.Rect != m.paint.Bounds || len(m.layout.Notes) != len(notes) || m.layout.ListRows < popupMinRows || m.layout.ListRows >= len(findings) {
				t.Fatalf("report should page findings and retain notes: %d rows, %d notes", m.layout.ListRows, len(m.layout.Notes))
			}
			d.Preview.DockSize = 160
			d.Preview.Hover = &popupPreviewHover{Part: "titleHelp"}
			m = check(t, d)
			if len(m.layout.Notes) != 0 || m.layout.ListRows < popupMinRows || m.previewTip().Full != strings.Join(m.plain.Notes, " ") {
				t.Fatalf("short dock should fold notes and retain at least three findings: %d rows, %d notes", m.layout.ListRows, len(m.layout.Notes))
			}
		})
	}
}
