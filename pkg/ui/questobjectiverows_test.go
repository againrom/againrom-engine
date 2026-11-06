package ui

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

// solidMark is a 24x24 mark frame of one colour.
func solidMark(c color.RGBA) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for i := 0; i < len(m.Pix); i += 4 {
		m.Pix[i], m.Pix[i+1], m.Pix[i+2], m.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return m
}

var questMarkColours = [3]color.RGBA{{R: 250, A: 255}, {G: 250, A: 255}, {B: 250, A: 255}}

// The objectives window's layout record against R2-ENGINE-888: below the
// briefing, one row per shown objective with its mark at x 48..72 from
// y - 3 and its text from x 80, the first row 16 below the briefing and 10 px
// between rows; the window grows by the rows about the same centre and its
// button moves below them. Without rows the window is the base panel.
func TestQuestObjectiveRowsStandBelowTheBriefing(t *testing.T) {
	f := openMissionMenu(t)
	f.menuFont = gameMenuTestFont()
	f.menuContext.Objective = "Find the gate."
	base := f.questLayout()
	if base.Panel != questPanelRect || base.Button != questButtonRect || base.Bar != questBarRect || len(base.Rows) != 0 {
		t.Fatalf("the window without rows is %+v", base)
	}
	long := strings.Repeat("Reach the elder before the bridge falls. ", 4)
	f.menuContext.Objectives = []QuestObjective{{Text: "", Mark: 0}, {Text: "Find the sack.", Mark: 1},
		{Text: long, Mark: 2}, {Text: "Plain", Mark: NoObjectiveMark}}
	l := f.questLayout()
	pitch := f.questPitch()
	wantLines := []int{1, 1, len(shopTipLines(f.menuFont, long, questPanelRect.Dx()-128)), 1}
	if wantLines[2] < 2 {
		t.Fatalf("the long text wraps to %d lines", wantLines[2])
	}
	y := questBarRect.Max.Y + 16
	var ys []int
	for _, n := range wantLines {
		ys = append(ys, y)
		y += n*pitch + 10
	}
	grow := y - questBarRect.Max.Y
	d := image.Pt(0, -(grow / 2))
	if l.Panel != image.Rect(80, 130, 560, 350+grow).Add(d) || l.Button != questButtonRect.Add(d).Add(image.Pt(0, grow)) ||
		l.Bar != questBarRect.Add(d) || l.Heading != image.Pt(120, 174).Add(d) || l.Body != image.Pt(120, 194).Add(d) {
		t.Fatalf("the grown window is %+v, grow %d", l, grow)
	}
	if len(l.Rows) != 4 {
		t.Fatalf("%d rows, want 4", len(l.Rows))
	}
	for i, r := range l.Rows {
		if r.Mark != f.menuContext.Objectives[i].Mark || len(r.Lines) != wantLines[i] ||
			r.MarkRect != image.Rect(128, ys[i]-3, 152, ys[i]+21).Add(d) || r.Text != image.Pt(160, ys[i]).Add(d) {
			t.Fatalf("row %d is %+v, want mark at y %d", i, r, ys[i]+d.Y)
		}
	}
	if l.Rows[0].Lines[0] != "" || l.Rows[1].Lines[0] != "Find the sack." {
		t.Fatalf("row texts %q %q", l.Rows[0].Lines, l.Rows[1].Lines)
	}
	var texts []string
	for _, row := range f.questRows() {
		texts = append(texts, row.Label)
	}
	want := append([]string{f.words.QuestHeading, "Find the gate.", "", "Find the sack."}, l.Rows[2].Lines...)
	want = append(want, "Plain")
	if len(texts) == 0 || !slices.Equal(texts[:len(texts)-1], want) {
		t.Fatalf("the window's rows read %q, want %q", texts, want)
	}
}

// Each shown row draws its mark frame of subobj.256 at its mark rectangle
// and a row without a mark draws none; the button stands where the layout
// puts it.
func TestQuestObjectiveRowsDrawTheirMarks(t *testing.T) {
	a := NewApp("quest", appAssets(t), appRows(1), nil)
	a.Layout(640, 480)
	f := a.flow
	f.menuFont = gameMenuTestFont()
	f.menuArt = &MenuPanelArt{ObjectiveMarks: [3]*image.RGBA{
		solidMark(questMarkColours[0]), solidMark(questMarkColours[1]), solidMark(questMarkColours[2])}}
	f.menuContext.Objective = "Find the gate."
	f.menuContext.Objectives = []QuestObjective{{Text: "open", Mark: 0}, {Text: "done", Mark: 1},
		{Text: "failed", Mark: 2}, {Text: "none", Mark: NoObjectiveMark}}
	pix := a.questPicture()
	l := f.questLayout()
	for i, r := range l.Rows {
		got := pix.RGBAAt(r.MarkRect.Min.X+12, r.MarkRect.Min.Y+12)
		if r.Mark == NoObjectiveMark {
			if slices.Contains(questMarkColours[:], got) {
				t.Fatalf("row %d without a mark drew %v", i, got)
			}
			continue
		}
		if got != questMarkColours[r.Mark] {
			t.Fatalf("row %d mark %d drew %v", i, r.Mark, got)
		}
	}
	if l.Button.Min.Y <= l.Rows[3].Text.Y {
		t.Fatalf("the button %v stands above the last row at %d", l.Button, l.Rows[3].Text.Y)
	}
}

// A Russian row reaches the font's bytes as the briefing does, one byte per
// letter, with no stand-in.
func TestQuestObjectiveRowsKeepRussianText(t *testing.T) {
	f := openMissionMenu(t)
	f.menuFont = &text.Font{Selector: text.SelectorConverting, Spacing: 1, Glyphs: gameMenuTestFont().Glyphs}
	f.encodeMenuKey = func(r rune) (byte, bool) { return textinput.EncodeRune(r, text.SelectorConverting) }
	row := "Найти мешок."
	f.menuContext.Objectives = []QuestObjective{{Text: row, Mark: 1}}
	l := f.questLayout()
	if len(l.Rows) != 1 || len(l.Rows[0].Lines) != 1 {
		t.Fatalf("the Russian row is %+v", l.Rows)
	}
	got := l.Rows[0].Lines[0]
	if got != f.menuDisplayText(row) || len(got) != len([]rune(row)) || strings.Contains(got, "?") {
		t.Fatalf("the Russian row reads % x", []byte(got))
	}
	if lines := f.questLines(); len(lines) != 0 {
		t.Fatalf("an empty briefing beside rows reads %q", lines)
	}
}
