package ui

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"againrom/pkg/namefold"
)

// The open maps of one mod. The window shows one map and keeps the others
// open: each with its document, its view, its selection, its list and its
// brush. A strip of tabs under the top bar names them while two or more are
// open. With one map open the window is as it was: no strip.

// maxOpenMaps is the most maps open at once: each holds its terrain picture
// and its history.
const maxOpenMaps = 8

// InspectionMapSet is implemented by edits whose host keeps several maps of
// one mod open. The mod's project is one; Shown carries it to the map shown.
type InspectionMapSet interface {
	// InMod reports that a mod is open, so another map opens beside this one.
	InMod() bool
	// MissionNumber is the number of the mod's mission the map is; 0: none.
	MissionNumber() int
	// MapDirty reports the map's own unsaved changes.
	MapDirty() bool
	// MapSource is the path the map is saved at: a save of the mod into
	// another folder moves every open map there.
	MapSource() string
	// Shown tells the host these edits are the ones shown: open is the edits
	// of every open map, left the edits shown before, nil for none.
	Shown(left InspectionEdits, open []InspectionEdits) error
}

// MapOpen is a host's answer to opening a map one of the open maps already
// is: Edits is that map's.
type MapOpen struct{ Edits InspectionEdits }

func (e *MapOpen) Error() string { return "the map is already open" }

// openMap is one open map the window does not show now, as it was left.
type openMap struct {
	doc       *InspectionDocument
	selected  int
	refs      []RecordRef
	filter    int
	listShown bool
	mode      keptMode
}

// mapSet is the shown map's host of the open maps, while a mod is open.
func (e *MapEditor) mapSet() (InspectionMapSet, bool) {
	if !e.editable() {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionMapSet)
	return h, ok && h.InMod()
}

// openDocs is the open maps' documents in tab order.
func (e *MapEditor) openDocs() []*InspectionDocument {
	if len(e.openMaps) == 0 {
		if e.doc == nil {
			return nil
		}
		return []*InspectionDocument{e.doc}
	}
	docs := make([]*InspectionDocument, len(e.openMaps))
	for i, m := range e.openMaps {
		docs[i] = m.doc
	}
	docs[e.shownMap] = e.doc
	return docs
}

// OpenMapCount is the number of maps the window has open.
func (e *MapEditor) OpenMapCount() int { return len(e.openDocs()) }

// ShownMap is the place of the map shown among the open maps.
func (e *MapEditor) ShownMap() int { return e.shownMap }

// OpenMapRow is one open map as a driver reads it: Caption is its tab's, with
// the star of unsaved changes.
type OpenMapRow struct {
	Caption string `json:"caption"`
	Path    string `json:"path"`
	Mission int    `json:"mission"`
	Dirty   bool   `json:"dirty"`
	Shown   bool   `json:"shown"`
}

// followMapSources makes each open map that is not shown name the path its
// host saves it at, so a map the mod's save moved keeps its one tab.
func (e *MapEditor) followMapSources() {
	for _, d := range e.openDocs() {
		if h, ok := d.Edits.(InspectionMapSet); ok && d != e.doc && h.MapSource() != "" {
			d.Source = h.MapSource()
		}
	}
}

// OpenMapRows is the open maps in tab order.
func (e *MapEditor) OpenMapRows() []OpenMapRow {
	e.followMapSources()
	var rows []OpenMapRow
	for _, d := range e.openDocs() {
		r := OpenMapRow{Caption: e.mcpDisplay(e.mapTabCaption(d)), Path: d.Source, Dirty: mapDirty(d), Shown: d == e.doc}
		if h, ok := d.Edits.(InspectionMapSet); ok {
			r.Mission = h.MissionNumber()
		}
		rows = append(rows, r)
	}
	return rows
}

// MapView is the view of the map shown: the world point at the window's
// top left and the zoom.
func (e *MapEditor) MapView() (x, y, zoom float64) {
	if e.doc == nil || e.doc.Viewer == nil {
		return 0, 0, 0
	}
	c := e.doc.Viewer.cam
	return c.X, c.Y, c.DrawnZoom()
}

// sameMapSource reports that two sources name one map.
func sameMapSource(a, b string) bool {
	if a == b {
		return true
	}
	if ParseMapSource(a).Kind != SourceHost || ParseMapSource(b).Kind != SourceHost {
		return false
	}
	return namefold.HostEqual(filepath.Clean(filepath.FromSlash(a)), filepath.Clean(filepath.FromSlash(b)))
}

// openMapOf is the open map of source, -1 when it is not open.
func (e *MapEditor) openMapOf(source string) int {
	if len(e.openMaps) < 2 {
		return -1
	}
	e.followMapSources()
	for i, d := range e.openDocs() {
		if sameMapSource(d.Source, source) {
			return i
		}
	}
	return -1
}

// nameOpenMaps tells the host which map is shown, among which, after left.
func (e *MapEditor) nameOpenMaps(left *InspectionDocument) {
	if !e.editable() {
		return
	}
	h, ok := e.doc.Edits.(InspectionMapSet)
	if !ok {
		return
	}
	var open []InspectionEdits
	for _, d := range e.openDocs() {
		if d.Edits != nil {
			open = append(open, d.Edits)
		}
	}
	var from InspectionEdits
	if left != nil {
		from = left.Edits
	}
	if err := h.Shown(from, open); err != nil {
		e.message = err.Error()
	}
}

// addOpenMap opens next beside the open maps and shows it. At the limit it
// refuses and says so.
func (e *MapEditor) addOpenMap(next *InspectionDocument) error {
	if len(e.openDocs()) >= maxOpenMaps {
		e.nameOpenMaps(nil)
		e.message = fmt.Sprintf(e.word("editor.tabs.limit"), maxOpenMaps)
		return fmt.Errorf("%s", fmt.Sprintf(e.plainWord("editor.tabs.limit"), maxOpenMaps))
	}
	left := e.doc
	if len(e.openMaps) == 0 {
		e.openMaps, e.shownMap = []*openMap{{}}, 0
	}
	e.keepOpenMap()
	e.openMaps = append(e.openMaps, &openMap{})
	e.shownMap = len(e.openMaps) - 1
	e.showDoc(next, nil)
	e.nameOpenMaps(left)
	return nil
}

// keepOpenMap keeps the shown map as the window leaves it.
func (e *MapEditor) keepOpenMap() {
	e.brush.endStroke()
	m := e.openMaps[e.shownMap]
	m.doc, m.selected, m.filter, m.listShown = e.doc, e.selected, e.filter, e.listShown
	m.refs = append([]RecordRef(nil), e.selection.refs...)
	m.mode = e.keepMode()
}

// ShowOpenMap shows open map n as it was left; the map shown stays open.
func (e *MapEditor) ShowOpenMap(n int) bool {
	if n < 0 || n >= len(e.openMaps) || n == e.shownMap {
		return n == e.shownMap && n < len(e.openMaps)
	}
	left := e.doc
	e.keepOpenMap()
	e.shownMap = n
	m := e.openMaps[n]
	// The host carries the mod's project before the window reads the map.
	e.doc = m.doc
	e.nameOpenMaps(left)
	e.showDoc(m.doc, m)
	return true
}

// showOpenEdits shows the open map whose edits are edits.
func (e *MapEditor) showOpenEdits(edits InspectionEdits) bool {
	for i, d := range e.openDocs() {
		if d.Edits == edits {
			return e.ShowOpenMap(i)
		}
	}
	return false
}

// CloseOpenMap closes the map shown and shows its neighbour. With unsaved
// changes of its own it asks first. The last open map does not close.
func (e *MapEditor) CloseOpenMap() {
	if len(e.openMaps) < 2 {
		return
	}
	if h, ok := e.mapSet(); ok && h.MapDirty() {
		e.askLeave(leaveStep{closeMap: true})
		return
	}
	e.dropOpenMap()
}

// dropOpenMap closes the map shown without asking.
func (e *MapEditor) dropOpenMap() {
	if len(e.openMaps) < 2 {
		return
	}
	left := e.doc
	e.brush.endStroke()
	e.openMaps = append(e.openMaps[:e.shownMap], e.openMaps[e.shownMap+1:]...)
	e.shownMap = min(e.shownMap, len(e.openMaps)-1)
	m := e.openMaps[e.shownMap]
	e.doc = m.doc
	e.nameOpenMaps(left)
	e.showDoc(m.doc, m)
	if v := left.Viewer; v != nil && v.canvas != nil {
		disposeFrameCanvas(v.canvas)
		v.canvas = nil
	}
}

// mergeOpenMaps closes every other open map of the shown map's file: a host
// step that put another open map's file into the shown map leaves one tab
// of it, the one shown.
func (e *MapEditor) mergeOpenMaps() {
	if len(e.openMaps) < 2 || e.doc == nil {
		return
	}
	changed := false
	for i := len(e.openMaps) - 1; i >= 0; i-- {
		if i != e.shownMap && sameMapSource(e.openMaps[i].doc.Source, e.doc.Source) {
			e.openMaps = append(e.openMaps[:i], e.openMaps[i+1:]...)
			if i < e.shownMap {
				e.shownMap--
			}
			changed = true
		}
	}
	if changed {
		e.nameOpenMaps(nil)
	}
}

// mapDirty reports the unsaved changes of open map document d: its own, or
// with a host that does not tell them apart every change its edits hold.
func mapDirty(d *InspectionDocument) bool {
	if d == nil || d.Edits == nil {
		return false
	}
	if h, ok := d.Edits.(InspectionMapSet); ok && h.InMod() {
		return h.MapDirty()
	}
	return d.Edits.Dirty()
}

// queueLeave lists the open maps the unsaved-changes question asks about
// before the window or the mod is left: each map with changes of its own,
// the shown one first; with none, the shown map for the mod's changes.
func (e *MapEditor) queueLeave() {
	e.leaveQueue = nil
	docs := e.openDocs()
	if e.doc != nil && mapDirty(e.doc) {
		e.leaveQueue = append(e.leaveQueue, e.doc)
	}
	for _, d := range docs {
		if d != e.doc && mapDirty(d) {
			e.leaveQueue = append(e.leaveQueue, d)
		}
	}
	if len(e.leaveQueue) == 0 && e.doc != nil {
		e.leaveQueue = append(e.leaveQueue, e.doc)
	}
}

// askNextLeave shows the next map the question asks about and asks; it
// reports false when every map was asked about.
func (e *MapEditor) askNextLeave(step leaveStep) bool {
	for len(e.leaveQueue) > 0 {
		d := e.leaveQueue[0]
		e.leaveQueue = e.leaveQueue[1:]
		if d != e.doc && !e.showOpenEdits(d.Edits) {
			continue
		}
		e.leave(leaveDrag | leavePick)
		e.openPrompt(editorPrompt{kind: promptLeave, then: step})
		return true
	}
	return false
}

// mapTabCaption is the tab caption of open map document d: its mission
// number, its name, and a star while it has unsaved changes.
func (e *MapEditor) mapTabCaption(d *InspectionDocument) string {
	name := e.editorHostText(d.Name)
	if strings.TrimSpace(name) == "" {
		name = e.leaveName(d.Source)
	}
	if h, ok := d.Edits.(InspectionMapSet); ok {
		if n := h.MissionNumber(); n > 0 {
			name = fmt.Sprintf("%d. %s", n, name)
		}
	}
	if mapDirty(d) {
		name += " *"
	}
	return name
}

// mapTabs is the strip of the open maps, false with fewer than two.
func (e *MapEditor) mapTabs() (tabStrip, bool) {
	if len(e.openMaps) < 2 {
		return tabStrip{}, false
	}
	t := tabStrip{Active: e.shownMap, Switch: true}
	for _, d := range e.openDocs() {
		t.Captions = append(t.Captions, e.mapTabCaption(d))
	}
	return t, true
}

// The strip's widest tab and its inset.
const (
	mapTabWidth = 240
	mapTabPad   = 3
)

// mapTabsRect is the strip's row under the top bar; no height with fewer
// than two maps open.
func (e *MapEditor) mapTabsRect() image.Rectangle {
	top, high := e.barHeight(), 0
	if len(e.openMaps) >= 2 {
		high = e.barRowHeight() + 2*mapTabPad
	}
	return image.Rect(0, top, e.window.X, top+high)
}

// mapTabsLayout is the strip's tabs with their captions cut to fit, and its
// close button at the right end, which closes the map shown: the parts of a
// popup's layout, so the strip is hit as a popup's tabs and close are.
func (e *MapEditor) mapTabsLayout() (tabStrip, popupLayout) {
	t, ok := e.mapTabs()
	if !ok {
		return t, popupLayout{}
	}
	l := popupLayout{Rect: e.mapTabsRect()}
	r := l.Rect.Inset(mapTabPad)
	l.Close = image.Rect(r.Max.X-r.Dy(), r.Min.Y, r.Max.X, r.Max.Y)
	width := min(l.Close.Min.X-mapTabPad-r.Min.X, len(t.Captions)*mapTabWidth)
	l.Tabs = t.rects(r.Min, width, r.Dy())
	if f := e.bodyFont(); f != nil {
		for i := range t.Captions {
			t.Captions[i] = editorClip(f, t.Captions[i], l.Tabs[i].Dx()-16)
		}
	}
	return t, l
}

// controlMapTabRect is the strip's tab n, or with closes its close button;
// nothing with fewer than two maps open.
func (e *MapEditor) controlMapTabRect(n int, closes bool) (image.Rectangle, bool) {
	_, l := e.mapTabsLayout()
	switch {
	case closes:
		return l.Close, !l.Close.Empty()
	case n >= 0 && n < len(l.Tabs):
		return l.Tabs[n], true
	}
	return image.Rectangle{}, false
}

// drawMapTabs draws the strip.
func (e *MapEditor) drawMapTabs(pic *image.RGBA) {
	t, l := e.mapTabsLayout()
	if len(l.Tabs) == 0 {
		return
	}
	drawFrame(pic, stripFrame(l.Rect, edgeBottomBit))
	over, at := popupMiss, 0
	if e.pointerOK {
		over, at = l.hit(e.pointer)
	}
	t.draw(pic, e.bodyFont(), l.Tabs, l.Close, &e.buttonPress, over, at)
}

// mapTabsPress is a press on the strip: a tab shows its map, the close
// button closes the map shown.
func (e *MapEditor) mapTabsPress(p image.Point) {
	_, l := e.mapTabsLayout()
	switch part, i := l.hit(p); part {
	case popupClose:
		e.CloseOpenMap()
	case popupTab:
		e.ShowOpenMap(i)
	}
}
