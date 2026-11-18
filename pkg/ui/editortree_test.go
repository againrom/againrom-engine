package ui

import (
	"fmt"
	"image"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/render/text"
)

// TestEditorTreeLaysOutAsTheFlatSpec lays out every movable window of the
// views fixture, in both languages, at every right panel width, in every
// view, record and section, at its own size and at a dragged size, once
// through the tree and once through the flat-spec layout kept below, and
// asks for the same geometry.
func TestEditorTreeLaysOutAsTheFlatSpec(t *testing.T) {
	seen := 0
	ids := []editorPopupID{popList, popObject, popPrompt, popTree, popProps, popToolbar}
	for _, lang := range []string{"en", "ru"} {
		for _, width := range wideWidths {
			for _, view := range []string{"plain", "technical"} {
				for _, record := range []int{viewsUnit, viewsStructure, viewsSack} {
					e := wideEditor(t, lang, width, view, nil)
					e.Select(record, false)
					e.objectSpec()
					for _, section := range wideSections(e) {
						openSection(t, e, section)
						for _, id := range ids {
							s, ok := e.movableSpec(id)
							if !ok {
								continue
							}
							name := fmt.Sprintf("%s %d %s %d %q window %d", lang, width, view, record, section, id)
							treeMatchesSpec(t, name, s, e.font, e.bodyFont())
							sized := s
							sized.Size = image.Pt(max(240, s.Width-40), 300)
							treeMatchesSpec(t, name+" sized", sized, e.font, e.bodyFont())
							seen++
						}
					}
				}
			}
		}
	}
	t.Logf("%d windows", seen)
	if seen < 2*len(wideWidths)*6 {
		t.Fatalf("only %d windows seen", seen)
	}
}

// TestEditorTreeLaysOutModWindowsAsTheFlatSpec does the same for the mod
// screens' windows, which hold the grids and the canvas.
func TestEditorTreeLaysOutModWindowsAsTheFlatSpec(t *testing.T) {
	seen := 0
	for _, lang := range []string{"en", "ru"} {
		for _, p := range modesPaths {
			e := modesWordsEditor(t, "workspace", lang, image.Pt(1024, 768))
			for _, step := range p.steps {
				modesAct(e, step)
			}
			for _, id := range []editorPopupID{popList, popObject, popPrompt, popTree, popProps, popToolbar} {
				s, ok := e.movableSpec(id)
				if !ok {
					continue
				}
				treeMatchesSpec(t, fmt.Sprintf("%s %s window %d", lang, p.name, id), s, e.font, e.bodyFont())
				seen++
			}
		}
	}
	t.Logf("%d windows", seen)
	if seen < len(modesPaths) {
		t.Fatalf("only %d windows seen", seen)
	}
}

func treeMatchesSpec(t *testing.T, name string, s popupSpec, title, body *text.Font) {
	t.Helper()
	// The flat reference predates fixed-width columns.
	s.ColumnWidth = 0
	got, want := layoutPopup(s, title, body), legacyLayoutPopup(s, title, body)
	if !s.Bounds.Empty() && want.Rect.Dy() > s.Bounds.Dy() {
		// A window taller than its bounds gives way; the flat spec ran past.
		if got.Rect.Dy() > s.Bounds.Dy() {
			t.Errorf("%s: %v still leaves the bounds %v", name, got.Rect, s.Bounds)
		}
		return
	}
	window := got.Window
	got.Window = nil
	if fmt.Sprintf("%+v", *window) != fmt.Sprintf("%+v", specWindow(s)) {
		t.Errorf("%s: the layout keeps a window other than its spec's", name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: the tree lays out\n%+v\nthe flat spec\n%+v", name, got, want)
	}
}

func legacyCardHeld(s popupSpec) bool {
	return s.Card != nil && slices.ContainsFunc(s.Rows, func(r popupRow) bool { return r.Hold == holdCard })
}

func legacyGridRoom(s popupSpec) image.Rectangle {
	if s.Room.Empty() {
		return s.Bounds
	}
	return s.Room
}

func legacyFixedGridWidth(s popupSpec) int {
	g := s.Grid
	if g == nil || g.Cols <= 0 {
		return 0
	}
	return 2*popupPad + g.HeadW + g.Cols*g.side()
}

func legacyGridWidth(s popupSpec, w int) int {
	g, room := s.Grid, legacyGridRoom(s)
	if g == nil || g.Rows > 0 || g.Cols > 0 || room.Empty() {
		return w
	}
	need := 2*popupPad + g.HeadW + min(len(g.Cells), gridLeastCols)*g.side() + widgetSpriteSize + 2
	return max(w, min(need, room.Dx()))
}

func legacyFitGrid(s popupSpec, w, rest, sizedH int, sized bool) *popupGrid {
	g := s.Grid
	if g == nil || g.Rows > 0 {
		return g
	}
	total := g.totalRows(g.columns(w - 2*popupPad - g.HeadW))
	limit, room := total, legacyGridRoom(s)
	free := func(h int) int { return (h - rest - g.headHeight() - popupPad) / g.rowHigh(0) }
	switch {
	case sized:
		limit = free(sizedH)
	case !room.Empty():
		limit = max(g.least(), free(room.Max.Y-max(s.At.Y, room.Min.Y)))
	}
	fit := *g
	fit.Rows = max(1, min(total, limit))
	return &fit
}

func legacyListHeadLines(s popupSpec, body *text.Font, w int) ([][]string, int) {
	if s.List == nil || len(s.ListColumns) == 0 {
		return nil, 0
	}
	stops := columnStops(listShares(s.ListColumns), w-2*popupPad-widgetSpriteSize-s.ListAside-2*listTextX)
	heads := make([][]string, len(s.ListColumns))
	lines := 1
	for k, c := range s.ListColumns {
		heads[k] = editorTextLines(body, c.Head, max(1, stops[k+1]-stops[k]-listColumnGap))
		lines = max(lines, len(heads[k]))
	}
	return heads, lines*fontHeight(body, 15) + popupPad
}

func legacyCardSize(s popupSpec) image.Point {
	return image.Pt(compactPanelW, compactPanelH).Mul(max(1, s.CardScale))
}

func legacyCardAside(s popupSpec) int {
	if s.Card == nil || legacyCardHeld(s) {
		return 0
	}
	return max(legacyCardSize(s).X, s.CardAside)
}

func legacyRowButtonWidths(s popupSpec, row popupRow, body *text.Font, pitch int) []int {
	return rowButtonWidths(specWindow(s), row, body, pitch)
}

func legacyPopupFlowOf(s popupSpec, body *text.Font, w, pitch int) popupFlow {
	f := popupFlow{tabsPerLine: len(s.Tabs), rowExtra: make([]int, len(s.Rows)), labelOver: make([]bool, len(s.Rows))}
	if len(s.Tabs) > 0 {
		f.tabLines = 1
	}
	ws := legacyPopupButtonWidths(s, body)
	f.buttonLine = make([]int, len(ws))
	if len(ws) > 0 || s.SwitchButton != "" {
		f.buttonLines = 1
	}
	f.cardH, f.cardBelow = legacyCardSize(s).Y, s.Card != nil && s.CardBelow
	f.cardTop, f.bandScrolls = f.cardBelow, legacyBandLines(s) > 0
	f.listHeads, f.listHeadH = legacyListHeadLines(s, body, w)
	inner := w - 2*popupPad
	if sw := legacySwitchButtonWidth(s, body); sw > 0 {
		lines, n := flowLines(append([]int{sw}, ws...), popupPad, inner, 0)
		f.buttonLine, f.buttonLines = lines[1:], n
	} else if len(ws) > 0 {
		f.buttonLine, f.buttonLines = flowLines(ws, popupPad, inner, 0)
	}
	if body != nil {
		for f.tabsPerLine > 1 {
			share, whole := inner/f.tabsPerLine, true
			for _, t := range s.Tabs {
				whole = whole && body.Advance(t)+16 <= share
			}
			if whole {
				break
			}
			f.tabsPerLine--
		}
	}
	if f.tabsPerLine > 0 {
		f.tabLines = (len(s.Tabs) + f.tabsPerLine - 1) / f.tabsPerLine
	}
	columns := max(1, s.Columns)
	if !s.Fit && columns == 1 {
		return f
	}
	aside := legacyCardAside(s)
	if s.Fit {
		f.cardTop = s.Card != nil && (s.CardBelow || cardOnTop(w, aside))
	}
	rowW := inner
	if s.MaxRows > 0 && (columns == 1 || legacyPopupRowCount(s) > s.MaxRows) {
		rowW -= widgetSpriteSize + 2
	}
	if s.Card != nil && !f.cardTop {
		rowW -= aside + popupPad
	}
	if columns > 1 {
		rowW = rowW/columns - popupPad
	}
	base := s.LabelWidth
	if base <= 0 {
		base = rowW * 2 / 5
	}
	need := make([]int, columns)
	if body != nil {
		for i, row := range s.Rows {
			if rowHasControl(row) && !row.placed() {
				c, _ := legacyPopupRowPosition(s, i)
				need[c] = max(need[c], body.Advance(row.Label)+8+helpMarkW(row, pitch))
			}
		}
	}
	f.columnLabelW = make([]int, columns)
	for c := range f.columnLabelW {
		f.columnLabelW[c] = max(min(base, rowW*2/5), min(need[c], rowW/2))
	}
	f.labelW = f.columnLabelW[0]
	for i, row := range s.Rows {
		if row.Kind == popupHeading || row.Kind == popupCard || row.placed() {
			continue
		}
		c, _ := legacyPopupRowPosition(s, i)
		labelW := f.columnLabelW[c]
		if rowHasControl(row) && row.Label != "" && body != nil && body.Advance(row.Label)+8+helpMarkW(row, pitch) > labelW {
			f.labelOver[i], f.rowExtra[i] = true, 1
		}
		bws := legacyRowButtonWidths(s, row, body, pitch)
		if len(bws) == 0 {
			continue
		}
		left := 0
		switch {
		case f.labelOver[i]:
			left = popupFitField
		case rowHasControl(row):
			left = labelW + 4 + popupFitField
		case row.Label != "" && body != nil:
			left = body.Advance(row.Label) + 8
		}
		if !rowHasControl(row) {
			left += helpMarkW(row, pitch)
		}
		beside := 0
		for _, bw := range bws {
			beside += bw + 4
		}
		if left+beside > rowW {
			_, n := flowLines(bws, 4, rowW, 1)
			f.rowExtra[i] += n
		}
	}
	return f
}

func legacyPopupMinWidth(s popupSpec, body *text.Font) int {
	w := max(240, legacyButtonRowWidth(s, body))
	pitch := fontHeight(body, 15) + popupRowExtra
	for _, row := range s.Rows {
		widths := legacyRowButtonWidths(s, row, body, pitch)
		if len(widths) == 0 {
			continue
		}
		need := 0
		for _, bw := range widths {
			need += bw + 4
		}
		if s.LabelWidth > 0 {
			w = max(w, s.LabelWidth+4+popupMinField+need+2*popupPad)
		} else {
			w = max(w, (4+popupMinField+need)*5/3+2*popupPad)
		}
	}
	switch {
	case legacyCardHeld(s):
	case s.Card != nil && s.CardBelow:
		bar := 0
		if legacyBandLines(s) > 0 {
			bar = widgetSpriteSize + 2
		}
		w = max(w, legacyCardSize(s).X+2*popupPad+bar)
		if s.CardAside > 0 {
			w = max(w, legacyCardSize(s).X+s.CardAside+3*popupPad+bar)
		}
	case s.Card != nil:
		w += legacyCardAside(s) + popupPad
	}
	return w
}

func legacyPopupRowsHeight(s popupSpec, flow popupFlow, pitch, shown int) int {
	heights := make([]int, max(1, s.Columns))
	for _, i := range legacyPopupShownRows(s, shown) {
		column, _ := legacyPopupRowPosition(s, i)
		heights[column] += flow.rowHeight(s.Rows[i], i, pitch)
	}
	return slices.Max(heights)
}

func legacyPopupFit(s popupSpec, body *text.Font, flow popupFlow, pitch, titleH, h int) (listRows, rowsFit, extraLines, minH int) {
	fixed := titleH + popupPad + len(s.Notes)*pitch + popupPad + s.Grid.height() + s.Canvas.least() + flow.cardBand()
	fixed += flow.buttonLines * (pitch + 4 + popupPad)
	cardLeast := 0
	if s.Card != nil && !flow.cardTop {
		cardLeast = titleH + popupPad + flow.cardH + popupPad + len(s.Notes)*pitch + flow.buttonLines*(pitch+4+popupPad)
	}
	minRows := popupMinRows
	if slices.ContainsFunc(s.Rows, func(r popupRow) bool { return r.Kind == popupDirection }) {
		minRows = 1
	}
	if s.Columns > 1 && s.MaxRows > 0 {
		if minRows == 1 {
			minH = max(fixed+legacyPopupRowsHeight(s, flow, pitch, 1), cardLeast)
			rowsFit = min(legacyPopupRowCount(s)-s.Top, max(1, (max(h, minH)-fixed)/pitch))
			for rowsFit > 1 && fixed+legacyPopupRowsHeight(s, flow, pitch, rowsFit) > max(h, minH) {
				rowsFit--
			}
			return 0, rowsFit, 0, minH
		}
		minH = max(fixed+popupMinRows*pitch, cardLeast)
		// The one change the copy takes: rows taller than the pitch fit.
		rowsFit = min(legacyPopupRowCount(s)-s.Top, max(popupMinRows, (h-fixed)/pitch))
		for rowsFit > popupMinRows && fixed+legacyPopupRowsHeight(s, flow, pitch, rowsFit) > max(h, minH) {
			rowsFit--
		}
		return 0, rowsFit, 0, minH
	}
	top := max(0, min(s.Top, len(s.Rows)))
	rowH := func(i int) int {
		return flow.rowHeight(s.Rows[i], i, pitch)
	}
	flexRows := s.MaxRows > 0
	if !flexRows {
		for i := top; i < len(s.Rows); i++ {
			fixed += rowH(i)
		}
		fixed += legacyBandLines(s) * pitch
	}
	lp, listBase := 0, 0
	if s.List != nil {
		listBase = newListBox(image.Rect(0, 0, 10, 0), 0, body).Rect.Dy()
		lp = newListBox(image.Rect(0, 0, 10, 0), 1, body).Rect.Dy() - listBase
		fixed += listBase + popupPad + flow.listHeadH
	}
	least, listLeast := fixed, popupMinRows
	if s.ListLeast > 0 {
		listLeast = s.ListLeast
	}
	if s.List != nil {
		least += listLeast * lp
	}
	if flexRows {
		for i := top; i < len(s.Rows) && i < top+minRows; i++ {
			least += rowH(i)
		}
	}
	minH = max(least, cardLeast)
	h = max(h, minH)
	avail := h - fixed
	switch {
	case s.List != nil:
		listRows = max(listLeast, avail/lp)
	case flexRows:
		used, full := 0, false
		for i := top; i < len(s.Rows); i++ {
			if s.Rows[i].placed() {
				continue
			}
			if used+rowH(i) > avail && rowsFit >= minRows {
				full = true
				break
			}
			used += rowH(i)
			rowsFit++
		}
		if !full {
			rowsFit += max(0, min(legacyBandLines(s), (avail-used)/pitch))
		}
		rowsFit = min(rowsFit, legacyPopupRowCount(s)-top)
	default:
		extraLines = max(0, avail/pitch)
	}
	return listRows, rowsFit, extraLines, minH
}

func legacyLayoutPopup(s popupSpec, title, body *text.Font) popupLayout {
	subject := s.Card
	if legacyCardHeld(s) {
		s.Card = nil
	}
	pitch := fontHeight(body, 15) + popupRowExtra
	titleH := fontHeight(title, 15) + popupRowExtra
	w := max(160, s.Width)
	if s.MaxRows > 0 && legacyPopupRowCount(s) > s.MaxRows {
		w += widgetSpriteSize + 2
	}
	sized := popupResizeEnabled && s.Size.X > 0 && s.Size.Y > 0
	if !sized {
		w = max(legacyGridWidth(s, w), legacyButtonRowWidth(s, body))
	}
	if sized {
		least := legacyPopupMinWidth(s, body)
		if s.MinWidth > 0 {
			least = min(least, s.MinWidth)
		}
		w = max(s.Size.X, least, legacyFixedGridWidth(s))
	}
	if !s.Bounds.Empty() {
		w = min(w, s.Bounds.Dx())
	}
	flow := legacyPopupFlowOf(s, body, w, pitch)
	titleH += flow.tabLines * pitch
	sizedH := 0
	if sized {
		lr, rf, extra, minH := legacyPopupFit(s, body, flow, pitch, titleH, s.Size.Y)
		sizedH = max(s.Size.Y, minH)
		if !s.Bounds.Empty() {
			sizedH = min(sizedH, max(minH, s.Bounds.Dy()))
		}
		lr, rf, extra, _ = legacyPopupFit(s, body, flow, pitch, titleH, sizedH)
		if s.List != nil {
			s.ListRows = lr
		}
		if s.MaxRows > 0 {
			s.MaxRows = rf
		}
		if extra > 0 {
			s.Rows = slices.Clone(s.Rows)
			for i := max(0, s.Top); i < len(s.Rows); i++ {
				if s.Rows[i].Kind == popupMultiline {
					s.Rows[i].Lines = multilineRows(s.Rows[i]) + extra
					break
				}
			}
		}
	}
	if !s.Bounds.Empty() {
		w = min(w, s.Bounds.Dx())
	}
	shown := legacyPopupRowCount(s) - s.Top
	if len(s.ColumnTops) > 0 {
		shown = 0
		for i := range s.Rows {
			column, line := legacyPopupRowPosition(s, i)
			shown = max(shown, line-legacyPopupColumnTop(s, column)+1)
		}
	}
	if s.MaxRows > 0 {
		shown = min(shown, s.MaxRows)
	}
	shown = max(0, shown)
	rowsH, barH := shown*pitch, shown*pitch
	lines := make([]int, 0, len(s.Rows))
	for i, row := range s.Rows {
		if !row.placed() {
			lines = append(lines, i)
		}
	}
	for i := 0; i < shown && s.Top+i < len(lines); i++ {
		at := lines[s.Top+i]
		rowsH += flow.rowHeight(s.Rows[at], at, pitch) - pitch
		barH += flow.rowExtra[at] * pitch
	}
	if s.Columns > 1 {
		rowsH = legacyPopupRowsHeight(s, flow, pitch, shown)
		barH = rowsH
	}
	h := titleH + popupPad + flow.cardBand() + rowsH
	var list listBox
	if s.List != nil {
		list = newListBox(image.Rect(0, 0, w-2*popupPad-widgetSpriteSize, 0), max(1, s.ListRows), body)
		h += list.Rect.Dy() + popupPad + flow.listHeadH
	}
	h += len(s.Notes)*pitch + popupPad
	h += flow.buttonLines * (pitch + 4 + popupPad)
	s.Grid = legacyFitGrid(s, w, h, sizedH, sized)
	h += s.Grid.height()
	legend, head := s.Canvas.legendPlaces(body, w-2*popupPad)
	s.Canvas = legacyFitCanvas(s, h, head, sizedH, sized)
	h += s.Canvas.height()
	if sized {
		h = max(h, sizedH)
	}
	r := image.Rectangle{Min: s.At, Max: s.At.Add(image.Pt(w, h))}
	if !s.Bounds.Empty() && s.Slack {
		r = slackInside(r, s.Bounds, titleH)
	} else if !s.Bounds.Empty() {
		if r.Max.X > s.Bounds.Max.X {
			r = r.Add(image.Pt(s.Bounds.Max.X-r.Max.X, 0))
		}
		if r.Max.Y > s.Bounds.Max.Y {
			r = r.Add(image.Pt(0, s.Bounds.Max.Y-r.Max.Y))
		}
		if r.Min.X < s.Bounds.Min.X {
			r = r.Add(image.Pt(s.Bounds.Min.X-r.Min.X, 0))
		}
		if r.Min.Y < s.Bounds.Min.Y {
			r = r.Add(image.Pt(0, s.Bounds.Min.Y-r.Min.Y))
		}
	}
	l := popupLayout{Rect: r, Pitch: pitch, ListRows: s.ListRows, RowsFit: s.MaxRows,
		docked: s.Docked, splitter: s.Splitter, gripHover: s.GripHover}
	l.Title = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+titleH)
	if len(s.Tabs) > 0 {
		l.Title.Max.Y -= flow.tabLines * pitch
		l.Tabs = tabStrip{Captions: s.Tabs, PerLine: flow.tabsPerLine}.rects(image.Pt(r.Min.X+popupPad, l.Title.Max.Y), w-2*popupPad, pitch)
	}
	l.Close = image.Rect(r.Max.X-l.Title.Dy(), r.Min.Y+2, r.Max.X-2, l.Title.Max.Y-2)
	if s.TitleHelp != "" {
		l.TitleHelp = l.Close.Sub(image.Pt(l.Close.Dx()+4, 0))
	}
	contentRight := r.Max.X - popupPad
	cardY := r.Min.Y + titleH + popupPad
	card := legacyCardSize(s)
	cardBand := func(y int) {
		band := image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+card.Y)
		l.Card = image.Rect(band.Max.X-card.X, band.Min.Y, band.Max.X, band.Max.Y)
		l.CardRest = image.Rect(band.Min.X, band.Min.Y, l.Card.Min.X-popupPad, band.Max.Y)
	}
	if s.Card != nil {
		switch {
		case flow.cardBelow:
		case flow.cardTop:
			cardBand(cardY)
		default:
			l.Card = image.Rect(r.Max.X-popupPad-card.X, cardY, r.Max.X-popupPad, cardY+card.Y)
			contentRight -= legacyCardAside(s) + popupPad
		}
	}
	contentY := r.Min.Y + titleH + popupPad
	if !flow.cardBelow {
		contentY += flow.cardBand()
	}
	if s.MaxRows > 0 && legacyPopupRowCount(s) > s.MaxRows {
		l.RowBar = image.Rect(contentRight-widgetSpriteSize, contentY, contentRight, contentY+barH)
		contentRight -= widgetSpriteSize + 2
	}
	l.RowsViewport = image.Rect(r.Min.X+popupPad, contentY, contentRight, contentY+rowsH)
	labelW := s.LabelWidth
	if labelW <= 0 {
		labelW = (contentRight - r.Min.X - popupPad) * 2 / 5
	}
	if flow.labelW > 0 {
		labelW = flow.labelW
	}
	y := contentY
	shownRows := legacyPopupShownRows(s, shown)
	columnExtra := make([]int, max(1, s.Columns))
	for _, index := range shownRows {
		row := s.Rows[index]
		if row.placed() {
			continue
		}
		left, right := r.Min.X+popupPad, contentRight
		if s.Columns > 1 {
			column, line := legacyPopupRowPosition(s, index)
			columnW := (contentRight - r.Min.X - popupPad) / s.Columns
			left, right = left+column*columnW, left+(column+1)*columnW-popupPad
			y = contentY + (line-legacyPopupColumnTop(s, column))*pitch + columnExtra[column]
			columnExtra[column] += flow.rowHeight(row, index, pitch) - pitch
		}
		rowH := flow.rowHeight(row, index, pitch)
		extra, over := flow.rowExtra[index], flow.labelOver[index]
		rr := popupRowRect{Index: index, Rect: image.Rect(left, y, right, y+rowH)}
		rowLabelW := labelW
		if s.Columns > 1 {
			column, _ := legacyPopupRowPosition(s, index)
			rowLabelW = flow.columnLabelW[column]
		}
		top, fieldH := y, pitch
		if row.Kind == popupMultiline || pictureRow(row) {
			fieldH = multilineRows(row) * pitch
		}
		if row.Kind == popupDirection {
			fieldH = 3 * pitch
		}
		rr.Label = image.Rect(rr.Rect.Min.X, y, rr.Rect.Min.X+rowLabelW, y+pitch)
		rr.Control = image.Rect(rr.Label.Max.X+4, y+1, rr.Rect.Max.X, y+fieldH-1)
		if over {
			top, extra = y+pitch, extra-1
			rr.Label.Max.X = rr.Rect.Max.X
			rr.Control = image.Rect(rr.Rect.Min.X, top+1, rr.Rect.Max.X, top+fieldH-1)
		}
		line := image.Rect(rr.Rect.Min.X, top, rr.Rect.Max.X, top+fieldH)
		switch row.Kind {
		case popupDirection:
			rr.Buttons = make([]image.Rectangle, len(row.ChoicePad))
			for k, pos := range row.ChoicePad {
				at := rr.Control.Min.Add(pos.Add(image.Pt(1, 1)).Mul(pitch))
				rr.Buttons[k] = image.Rectangle{Min: at, Max: at.Add(image.Pt(pitch-2, pitch-2))}
			}
		case popupHeading:
			rr.Label, rr.Control = line, image.Rectangle{}
		case popupText, popupLink:
			if row.Value == "" && !pictureRow(row) || row.Hold != holdNone {
				rr.Label, rr.Control = line, image.Rectangle{}
			}
		case popupCard:
			if row.Struck {
				rr.Label, rr.Control = line, image.Rectangle{}
			} else {
				rr.Grip = rr.Label
			}
			if !row.Struck && row.Value == "" {
				rr.Label, rr.Control = line, image.Rectangle{}
			}
		}
		if ws := legacyRowButtonWidths(s, row, body, pitch); len(ws) > 0 {
			rr.Buttons = make([]image.Rectangle, len(ws))
			if extra > 0 {
				lines, _ := flowLines(ws, 4, rr.Rect.Dx(), 0)
				xs := make([]int, extra)
				for k := range xs {
					xs[k] = rr.Rect.Max.X
				}
				for k := len(ws) - 1; k >= 0; k-- {
					n := min(lines[k], extra-1)
					top := line.Max.Y + n*pitch
					rr.Buttons[k] = image.Rect(xs[n]-ws[k], top+1, xs[n], top+pitch-1)
					xs[n] -= ws[k] + 4
				}
			} else {
				x := rr.Rect.Max.X
				for k := len(ws) - 1; k >= 0; k-- {
					rr.Buttons[k] = image.Rect(x-ws[k], top+1, x, top+pitch-1)
					x -= ws[k] + 4
				}
				if rr.Control.Empty() {
					rr.Label.Max.X = min(rr.Label.Max.X, x)
				} else {
					rr.Control.Max.X = x
				}
			}
			rr.Button, rr.Off = rr.Buttons[0], row.Disabled
		}
		if row.Help != "" && !row.placed() {
			side := pitch - 4
			rr.Help = image.Rect(rr.Label.Max.X-side-2, rr.Label.Min.Y+2, rr.Label.Max.X-2, rr.Label.Min.Y+2+side)
			rr.Label.Max.X = rr.Help.Min.X - 2
		}
		l.Rows = append(l.Rows, rr)
		y += rowH
	}
	if s.Columns > 1 {
		y = contentY + rowsH
	}
	switch {
	case s.Card != nil && flow.cardBelow && flow.bandScrolls:
		rowLines := legacyPopupRowCount(s) - legacyBandLines(s)
		placed := 0
		for _, index := range shownRows {
			if !s.Rows[index].placed() {
				placed++
			}
		}
		top := y
		switch {
		case s.Top > rowLines:
			top = contentY - (s.Top-rowLines)*pitch
		case s.Top+placed < rowLines:
			top = contentY + rowsH
		}
		band := image.Rect(r.Min.X+popupPad, top+popupPad, contentRight, top+legacyBandLines(s)*pitch)
		l.Card = image.Rect(band.Max.X-card.X, band.Min.Y, band.Max.X, band.Min.Y+card.Y)
		l.CardRest = image.Rect(band.Min.X, band.Min.Y, l.Card.Min.X-popupPad, max(band.Max.Y, l.Card.Max.Y))
		y = contentY + rowsH
	case s.Card != nil && flow.cardBelow:
		cardBand(y + popupPad)
		y = l.Card.Max.Y
	}
	if subject != nil && s.Card == nil {
		for _, rr := range l.Rows {
			if s.Rows[rr.Index].Hold == holdCard {
				l.Card = image.Rectangle{Min: rr.Rect.Min, Max: rr.Rect.Min.Add(card)}
				l.CardRest = image.Rect(l.Card.Max.X+popupPad, rr.Rect.Min.Y, rr.Rect.Max.X, rr.Rect.Max.Y)
				break
			}
		}
	}
	for _, index := range shownRows {
		row := s.Rows[index]
		if !row.placed() || row.HeadOf > 0 || subject == nil || l.Card.Empty() {
			continue
		}
		box := row.Place.Add(l.Card.Min)
		if row.RestLeft {
			box = row.Place.Add(image.Pt(l.CardRest.Min.X, l.Card.Min.Y))
		}
		rr := popupRowRect{Index: index, Rect: box}
		if row.Kind != popupText && row.Kind != popupLink {
			rr.Control = box
		}
		if row.Kind == popupDirection {
			rr.Label = image.Rect(box.Min.X, box.Min.Y, box.Max.X, box.Min.Y+pitch)
			rr.Control = image.Rect(box.Min.X, box.Min.Y+pitch, box.Max.X, box.Max.Y)
			rr.Buttons = make([]image.Rectangle, len(row.ChoicePad))
			for k, pos := range row.ChoicePad {
				at := rr.Control.Min.Add(pos.Add(image.Pt(1, 1)).Mul(pitch))
				rr.Buttons[k] = image.Rectangle{Min: at, Max: at.Add(image.Pt(pitch-2, pitch-2))}
			}
		}
		l.Rows = append(l.Rows, rr)
	}
	l.Drop = -1
	legacyLayoutCards(s, &l)
	if g := s.Grid; g != nil {
		side, shown, top := g.side(), max(1, g.Rows), y
		l.GridRows = shown
		l.GridCols = g.columns(r.Dx() - 2*popupPad - g.HeadW)
		total := g.totalRows(l.GridCols)
		l.GridTop = max(0, min(g.Top, total-shown))
		x := r.Min.X + popupPad + g.HeadW
		for c := 0; c < len(g.ColHeads) && c < l.GridCols; c++ {
			l.GridColHeads = append(l.GridColHeads, image.Rect(x+c*side, y, x+(c+1)*side, y+g.headHeight()))
		}
		y += g.headHeight()
		l.Grid = image.Rect(x, y, x+l.GridCols*side, y+g.rowsHigh(l.GridTop, shown))
		if total > shown {
			l.GridBar = image.Rect(r.Max.X-popupPad-widgetSpriteSize, y, r.Max.X-popupPad, l.Grid.Max.Y)
		}
		l.GridFirst = l.GridTop * l.GridCols
		rows := g.rows(l.GridCols)
		for k, at := 0, y; k < shown && l.GridTop+k < len(rows); k++ {
			row, high := rows[l.GridTop+k], g.rowHigh(l.GridTop+k)
			if row.Head != "" || len(g.Groups) == 0 && l.GridTop+k < len(g.RowHeads) {
				left, right := r.Min.X+popupPad, x
				if len(g.Groups) > 0 {
					left, right = x, l.Grid.Max.X
				}
				l.GridRowHeads = append(l.GridRowHeads, image.Rect(left, at, right, at+high))
				l.GridHeadText = append(l.GridHeadText, row.Head)
			}
			for index, held := range s.Rows {
				if held.HeadOf != l.GridTop+k+1 || !held.placed() {
					continue
				}
				box := held.Place.Add(image.Pt(r.Min.X+popupPad, at))
				rr := popupRowRect{Index: index, Rect: box, Control: box}
				if len(held.Buttons) > 0 {
					rr.Control, rr.Buttons, rr.Button, rr.Off = image.Rectangle{}, []image.Rectangle{box}, box, held.Disabled
				}
				l.Rows = append(l.Rows, rr)
			}
			for column, n := range row.Cells {
				corner := image.Pt(x+column*side, at)
				l.GridCells = append(l.GridCells, image.Rectangle{Min: corner, Max: corner.Add(image.Pt(side, high))}.Inset(1))
				l.GridInert = append(l.GridInert, g.Cells[n].Inert)
				if len(g.Groups) > 0 {
					l.GridIndices = append(l.GridIndices, n)
				}
			}
			at += high
		}
		y = top + g.height()
	}
	if c := s.Canvas; c != nil {
		y = l.placeCanvas(c, r, y, legend)
	}
	if s.List != nil {
		if len(flow.listHeads) > 0 {
			x := r.Min.X + popupPad + listTextX
			stops := columnStops(listShares(s.ListColumns), r.Dx()-2*popupPad-widgetSpriteSize-s.ListAside-2*listTextX)
			for k := range flow.listHeads {
				l.ListHeads = append(l.ListHeads, image.Rect(x+stops[k], y, x+stops[k+1]-listColumnGap/2, y+flow.listHeadH-popupPad))
			}
			l.ListHeadLines = flow.listHeads
			y += flow.listHeadH
		}
		l.List = newListBox(image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad-widgetSpriteSize-s.ListAside, y), max(1, s.ListRows), body)
		if len(s.ListColumns) > 0 {
			l.List.Columns = listShares(s.ListColumns)
		}
		l.HasList = true
		if s.ListAside > 0 {
			l.Aside = image.Rect(r.Max.X-popupPad-s.ListAside+popupPad, y, r.Max.X-popupPad, l.List.Rect.Max.Y)
		}
		y = l.List.Rect.Max.Y + popupPad
	}
	for range s.Notes {
		l.Notes = append(l.Notes, image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+pitch))
		y += pitch
	}
	if n := len(s.Buttons); n > 0 || s.SwitchButton != "" {
		step := pitch + 4 + popupPad
		y += popupPad
		if sized {
			y = r.Max.Y - popupPad - (pitch + 4) - (flow.buttonLines-1)*step
		}
		if sw := legacySwitchButtonWidth(s, body); sw > 0 {
			l.SwitchButton = image.Rect(r.Min.X+popupPad, y, r.Min.X+popupPad+sw, y+pitch+4)
		}
		ws := legacyPopupButtonWidths(s, body)
		xs := make([]int, flow.buttonLines)
		for k := range xs {
			xs[k] = r.Max.X - popupPad
		}
		l.Buttons = make([]image.Rectangle, n)
		for i := n - 1; i >= 0; i-- {
			k := min(flow.buttonLine[i], flow.buttonLines-1)
			top := y + k*step
			l.Buttons[i] = image.Rect(xs[k]-ws[i], top, xs[k], top+pitch+4)
			xs[k] -= ws[i] + popupPad
		}
	}
	if s.Card != nil && !flow.cardTop {
		bottom := r.Max.Y - popupPad
		if len(l.Buttons) > 0 {
			bottom = l.Buttons[0].Min.Y - popupPad
		} else if !l.SwitchButton.Empty() {
			bottom = l.SwitchButton.Min.Y - popupPad
		}
		top := l.Card.Max.Y + popupPad
		l.CardRest = image.Rect(r.Max.X-popupPad-legacyCardAside(s), top, r.Max.X-popupPad, max(top, bottom))
	}
	visible := l.Rows[:0]
	for _, rr := range l.Rows {
		if flow.bandScrolls && s.Rows[rr.Index].placed() && !rr.Rect.In(l.RowsViewport) {
			continue
		}
		if !s.Rows[rr.Index].placed() {
			if !rr.Rect.In(l.RowsViewport) {
				continue
			}
			rr.Label = rr.Label.Intersect(l.RowsViewport)
			rr.Control = rr.Control.Intersect(l.RowsViewport)
			rr.Button = rr.Button.Intersect(l.RowsViewport)
			rr.Grip = rr.Grip.Intersect(l.RowsViewport)
			rr.Card = rr.Card.Intersect(l.RowsViewport)
			for k := range rr.Buttons {
				rr.Buttons[k] = rr.Buttons[k].Intersect(l.RowsViewport)
			}
		}
		visible = append(visible, rr)
	}
	l.Rows = visible
	return l
}

func legacyLayoutCards(s popupSpec, l *popupLayout) {
	ordinal := map[int]int{}
	held, cards := -1, 0
	for i, row := range s.Rows {
		if row.Kind != popupCard {
			continue
		}
		ordinal[i] = cards
		if s.Drag != nil && s.Drag.Row == i {
			held = cards
		}
		cards++
	}
	var shown []int
	for i, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.Kind != popupCard {
			continue
		}
		card := rr.Rect
		for k := 1; k <= row.Span && i+k < len(l.Rows) && l.Rows[i+k].Index == rr.Index+k; k++ {
			card = card.Union(l.Rows[i+k].Rect)
		}
		l.Rows[i].Card = card
		shown = append(shown, i)
	}
	if held < 0 || len(shown) == 0 {
		return
	}
	at := s.Drag.At.Y
	last := l.Rows[shown[len(shown)-1]]
	y := last.Card.Max.Y
	l.Drop = ordinal[last.Index] + 1
	for _, i := range shown {
		if c := l.Rows[i].Card; at < (c.Min.Y+c.Max.Y)/2 {
			l.Drop, y = ordinal[l.Rows[i].Index], c.Min.Y
			break
		}
	}
	if l.Drop != held && l.Drop != held+1 {
		l.Marker = image.Rect(last.Rect.Min.X, y-2, last.Rect.Max.X, y+1)
	}
}

func legacyPopupButtonWidths(s popupSpec, body *text.Font) []int {
	bw := popupButtonW
	if s.ButtonW > 0 {
		bw = s.ButtonW
	}
	ws := make([]int, len(s.Buttons))
	for i, c := range s.Buttons {
		ws[i] = bw
		if body != nil {
			ws[i] = max(bw, body.Advance(c)+16)
		}
	}
	return ws
}

func legacySwitchButtonWidth(s popupSpec, body *text.Font) int {
	if s.SwitchButton == "" {
		return 0
	}
	return legacyPopupButtonWidths(popupSpec{Buttons: []string{s.SwitchButton}, ButtonW: 48}, body)[0]
}

func legacyButtonRowWidth(s popupSpec, body *text.Font) int {
	row := popupPad
	for _, bw := range legacyPopupButtonWidths(s, body) {
		row += bw + popupPad
	}
	if sw := legacySwitchButtonWidth(s, body); sw > 0 {
		row += sw + popupPad
	}
	return row
}

func legacyFitCanvas(s popupSpec, rest, head, sizedH int, sized bool) *popupCanvas {
	c := s.Canvas
	if c == nil {
		return nil
	}
	fit := *c
	fit.head = head
	free := func(h int) int { return max(canvasLeastHeight, h-rest-head-popupPad) }
	room := legacyGridRoom(s)
	switch {
	case sized:
		fit.Height = free(sizedH)
	case c.Height > 0:
		fit.Height = min(c.Height, c.Sheet.Y)
	case !room.Empty():
		fit.Height = min(c.Sheet.Y, free(room.Max.Y-max(s.At.Y, room.Min.Y)))
	default:
		fit.Height = c.Sheet.Y
	}
	return &fit
}

func legacyPopupRowPosition(s popupSpec, index int) (int, int) {
	if s.Columns <= 1 {
		line := index
		for _, row := range s.Rows[:index] {
			if row.placed() {
				line--
			}
		}
		return 0, line
	}
	column, line := max(0, min(s.Rows[index].Column, s.Columns-1)), 0
	for _, row := range s.Rows[:index] {
		if max(0, min(row.Column, s.Columns-1)) == column && !row.placed() {
			line++
		}
	}
	return column, line
}

func legacyPopupRowCount(s popupSpec) int {
	if s.Columns <= 1 {
		n := len(s.Rows) + legacyBandLines(s)
		for _, row := range s.Rows {
			if row.placed() {
				n--
			}
		}
		return n
	}
	counts := make([]int, s.Columns)
	for _, row := range s.Rows {
		if !row.placed() {
			counts[max(0, min(row.Column, s.Columns-1))]++
		}
	}
	total := 0
	for _, count := range counts {
		total = max(total, count)
	}
	return total
}

func legacyPopupShownRows(s popupSpec, shown int) []int {
	var out []int
	line := 0
	for i, row := range s.Rows {
		if row.placed() {
			out = append(out, i)
			continue
		}
		column := 0
		if s.Columns > 1 {
			column, line = legacyPopupRowPosition(s, i)
		}
		top := legacyPopupColumnTop(s, column)
		if line >= top && line < top+shown {
			out = append(out, i)
		}
		if s.Columns <= 1 {
			line++
		}
	}
	return out
}

func legacyBandLines(s popupSpec) int {
	if s.Card == nil || !s.CardBelow || s.Columns > 1 {
		return 0
	}
	return max(0, s.BandLines)
}

func legacyPopupColumnTop(s popupSpec, column int) int {
	if column < len(s.ColumnTops) {
		return s.ColumnTops[column]
	}
	return s.Top
}
