package ui

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"github.com/hajimehoshi/ebiten/v2"
)

// The owner reference shows one paragraph panel and a centred OK control.
// Geometry uses the native frame and the existing installed popup art.
var questPanelRect = image.Rect(80, 130, 560, 350)
var questButtonRect = image.Rect(270, 304, 370, 328)

func (a *App) SetGameMenuArt(art *MenuPanelArt) {
	a.flow.menuArt = art
	if a.flow.viewer != nil {
		a.flow.viewer.SetDialogFrame(art)
	}
}

func (f *flow) questLines() []string {
	s := strings.TrimSpace(f.menuContext.Objective)
	if s == "" && len(f.menuContext.Objectives) != 0 {
		return nil
	}
	if s == "" {
		s = f.word("quest.empty")
	}
	if f.menuFont == nil {
		return gameMenuTextLines(s)
	}
	paras := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	// Title, blank line, body: the title and the body's first line indent.
	titled := len(paras) > 2 && strings.TrimSpace(paras[0]) != "" &&
		strings.TrimSpace(paras[1]) == "" && strings.TrimSpace(paras[2]) != ""
	var lines []string
	for i, para := range paras {
		if titled && i == 1 {
			continue
		}
		wrapped := shopTipLines(f.menuFont, f.menuDisplayText(para), 400)
		if titled && i <= 2 {
			wrapped[0] = questIndent + wrapped[0]
		}
		lines = append(lines, wrapped...)
	}
	return lines
}

const questIndent = " "

// questLayout is the objectives window as drawn. Below the briefing stands
// one row per shown objective: its 24x24 mark at x 48..72 from y - 3 beside
// its text, wrapped from x 80 to 48 short of the right edge, the first row
// 16 below the briefing and 10 px between rows (R2-ENGINE-888). The window
// grows by its rows about the same centre; without rows it is the base panel.
type questLayout struct {
	Panel, Button, Bar image.Rectangle
	Heading, Body      image.Point
	Rows               []questRowLayout
}

// questRowLayout is one objective row: its mark frame and rectangle, and its
// text lines from Text down at the window's line pitch.
type questRowLayout struct {
	Mark     int
	MarkRect image.Rectangle
	Text     image.Point
	Lines    []string
}

func (f *flow) questPitch() int {
	if f.menuFont == nil {
		return 16
	}
	return max(1, f.menuFont.Height()+2)
}

func (f *flow) questLayout() questLayout {
	pitch := f.questPitch()
	left := questPanelRect.Min.X
	y := questBarRect.Max.Y + 16
	var rows []questRowLayout
	for _, o := range f.menuContext.Objectives {
		lines := f.questRowLines(o.Text, questPanelRect.Dx()-128)
		rows = append(rows, questRowLayout{Mark: o.Mark, MarkRect: image.Rect(left+48, y-3, left+72, y+21),
			Text: image.Pt(left+80, y), Lines: lines})
		y += len(lines)*pitch + 10
	}
	grow := 0
	if len(rows) != 0 {
		grow = y - questBarRect.Max.Y
	}
	d := image.Pt(0, max(0, questPanelRect.Min.Y-grow/2)-questPanelRect.Min.Y)
	l := questLayout{
		Panel:   image.Rect(questPanelRect.Min.X, questPanelRect.Min.Y, questPanelRect.Max.X, questPanelRect.Max.Y+grow).Add(d),
		Button:  questButtonRect.Add(d).Add(image.Pt(0, grow)),
		Bar:     questBarRect.Add(d),
		Heading: image.Pt(120, 174).Add(d),
		Body:    image.Pt(120, 194).Add(d),
	}
	for _, r := range rows {
		r.MarkRect, r.Text = r.MarkRect.Add(d), r.Text.Add(d)
		l.Rows = append(l.Rows, r)
	}
	return l
}

// questRowLines is an objective's text wrapped to width; a text of line
// breaks alone keeps one empty line.
func (f *flow) questRowLines(s string, width int) []string {
	var lines []string
	for _, para := range strings.Split(strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n"), "\n") {
		if f.menuFont == nil {
			lines = append(lines, para)
			continue
		}
		lines = append(lines, shopTipLines(f.menuFont, f.menuDisplayText(para), width)...)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

func (f *flow) questRows() []gameMenuRow {
	rows := []gameMenuRow{{Label: f.words.QuestHeading, Literal: true, Status: true}}
	for _, line := range f.questLines() {
		rows = append(rows, gameMenuRow{Label: line, Literal: true, Status: true})
	}
	for _, r := range f.questLayout().Rows {
		for _, line := range r.Lines {
			rows = append(rows, gameMenuRow{Label: line, Literal: true, Status: true})
		}
	}
	label := f.words.NoticeButton
	if label == "" {
		label = "OK"
	}
	return append(rows, gameMenuRow{Label: label, Literal: true, Action: gameMenuPageReturn, Enabled: true})
}

func (a *App) questPicture() *image.RGBA {
	f := a.flow
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	l := f.questLayout()
	drawFrame(pix, windowFrame(l.Panel, f.menuArt))
	for _, r := range l.Rows {
		if mark := f.menuArt.objectiveMark(r.Mark); mark != nil {
			draw.Draw(pix, image.Rectangle{Min: r.MarkRect.Min, Max: r.MarkRect.Min.Add(mark.Bounds().Size())}, mark, mark.Bounds().Min, draw.Over)
		}
	}
	font := f.menuFont
	if font == nil {
		return pix
	}
	label := func(s string, x, y int) {
		font.Draw(pix, s, x+1, y+1, color.RGBA{A: 255})
		font.Draw(pix, s, x, y, townShellText)
	}
	label(f.words.QuestHeading, l.Heading.X, l.Heading.Y)
	lines := f.questLines()
	pitch := f.questPitch()
	count := max(1, questBarRect.Dy()/pitch)
	top := min(max(0, f.questTop), max(0, len(lines)-count))
	for i := 0; i < count && top+i < len(lines); i++ {
		label(lines[top+i], l.Body.X, l.Body.Y+i*pitch)
	}
	for _, r := range l.Rows {
		for i, line := range r.Lines {
			label(line, r.Text.X, r.Text.Y+i*pitch)
		}
	}
	pointer, pointerOK := a.pointerFrame()
	if len(lines) > count {
		drawVScrollBar(pix, a.media.scroll, questBar(l.Bar, top, len(lines)-count).withPointer(pointer, pointerOK))
	}
	rows := f.questRows()
	inside := pointerOK && pointer.In(l.Button)
	drawPushButton(pix, font, pushButton{Rect: l.Button, Label: rows[len(rows)-1].Label, Literal: true,
		Hover: inside, Inside: inside, Pressed: f.questPress.Pressed(0)})
	return pix
}

// questBar is the briefing text's bar in rect: the shared vertical bar over
// its top-line positions 0..most.
func questBar(rect image.Rectangle, top, most int) vScrollBar {
	return vScrollBar{Rect: rect, Pos: top, Count: most + 1}
}

var questBarRect = image.Rect(530, 194, 554, 294)

// QuestObjectivePanel exposes the same CPU composition uploaded by Draw.
func (a *App) QuestObjectivePanel() *image.RGBA {
	if a == nil || a.flow == nil || a.flow.screen != ScreenGameMenu || a.flow.menuPage != gameMenuQuestObjectivesPage {
		return nil
	}
	return a.questPicture()
}

func (a *App) paintCurrentGameMenu(dst *ebiten.Image) {
	since := markCapture()
	if pic := a.GameMenuPanel(); pic != nil {
		blit := pic
		if a.textSmoothingEnabled && a.gameMenuAcknowledgement() != "" {
			var status []text.DrawCall
			calls := text.Captured()
			for i := since; i < len(calls); i++ {
				if calls[i].Y == pickerMessageY {
					status = append(status, calls[i])
					calls[i].Erased = true
				}
			}
			if len(status) > 0 {
				blit = image.NewRGBA(pic.Bounds())
				copy(blit.Pix, pic.Pix)
				textsmooth.Erase(blit, status)
			}
		}
		dst.DrawImage(ebiten.NewImageFromImage(blit), nil)
		a.menuLog.over(pic, image.Point{})
		return
	}
	paintGameMenu(dst, a.flow.menuFont, a.flow.menuPanelSurface(), a.flow.menuRows(), a.flow.menuList, a.flow.menuGeometryProfile(), &a.menuLog)
}

func (a *App) stepQuestObjectives(in appInput) {
	f := a.flow
	if in.Unfocused {
		f.questPress, f.questBar = buttonLatch{}, scrollBarInput{}
		return
	}
	if in.Enter {
		f.leaveMenuWindow(0)
		f.questPress.Clear()
		return
	}
	if in.Up {
		f.questTop = max(0, f.questTop+questList.keyStep(true, false))
	}
	if in.Down {
		f.questTop += questList.keyStep(false, true)
	}
	if wheeled(in) {
		f.questTop = max(0, f.questTop+questList.wheelStep(in))
	}
	pitch := f.questPitch()
	l := f.questLayout()
	most := max(0, len(f.questLines())-max(1, 100/pitch))
	f.questTop = min(f.questTop, most)
	p, valid := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if most > 0 {
		if req, pos := f.questBar.step(questBar(l.Bar, f.questTop, most), p, valid, in); req != barNone {
			if req == barSetPos {
				f.questTop = pos
			} else if step, ok := questList.barStep(req, max(1, 100/pitch)); ok {
				f.questTop += step
			}
			f.questTop = min(max(f.questTop, 0), most)
		}
		if f.questBar.active() {
			return
		}
	}
	hit := valid && p.In(l.Button)
	if in.PrimaryPressed {
		f.questPress.Press(0, hit)
	}
	if in.PrimaryReleased {
		_, activate := f.questPress.Release(0, hit)
		if activate {
			a.playUISound(UISoundCommonControl)
			f.leaveMenuWindow(0)
		}
	}
}
