package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"sort"
	"testing"
)

// layoutProbe is the answer of one screen's controls: the rectangles its
// draw hands the shared builders and what its hit test answers on a grid of
// points over the 640x480 frame. Every window size maps onto that frame
// through windowToNativeFrame, which the layout does not touch.
func layoutProbe(t *testing.T, draw func(), hit func(p image.Point) int) string {
	t.Helper()
	calls := recordWidgets(t, draw)
	var drawn []string
	for _, c := range calls {
		switch c.kind {
		case widgetPushButton, widgetEdit, widgetListBox:
			drawn = append(drawn, fmt.Sprintf("%d%v", c.kind, c.rect))
		}
	}
	sort.Strings(drawn)
	h := fnv.New64a()
	fmt.Fprint(h, drawn)
	for y := 0; y < 480; y += 2 {
		for x := 0; x < 640; x += 2 {
			fmt.Fprint(h, hit(image.Pt(x, y)), ",")
		}
	}
	return fmt.Sprintf("%d drawn %x", len(drawn), h.Sum64())
}

// TestControlLayoutKeepsEveryDrawAndHitAnswer pins the rectangle each control
// draws at and answers hits at, recorded before the controls moved to the
// layout descriptor.
func TestControlLayoutKeepsEveryDrawAndHitAnswer(t *testing.T) {
	got := map[string]string{}

	for _, mode := range []string{"form", "overwrite", "delete"} {
		a := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
		a.SetWords(AuthoredWords(), chargenTestFont(), nil)
		d := a.flow.saveDialog
		switch mode {
		case "overwrite":
			d.prepared = &PreparedSave{Paths: []string{"x"}}
		case "delete":
			d.remove = func() error { return nil }
			d.removePath = "x"
		}
		got["dialog "+mode] = layoutProbe(t, func() {
			if _, err := a.composeSaveDialogScreen(); err != nil {
				t.Fatal(err)
			}
		}, func(p image.Point) int {
			c, ok := d.controlAt(p)
			if !ok {
				return -1
			}
			return int(c)
		})
	}

	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0001.sav", Label: "Alpha"}}}
	a := lookApp(t, fake)
	save := a.flow.pages.save
	for i, caption := range []string{"Ok", "Delete", "Cancel"} {
		save.page.Buttons[i] = PageButton{Rect: image.Rect(154+i*122, 318, 250+i*122, 342), Caption: caption}
	}
	v := save.view()
	got["save page"] = layoutProbe(t, func() {
		if _, err := a.composePage(v); err != nil {
			t.Fatal(err)
		}
	}, func(p image.Point) int {
		i, ok := v.buttonAt(p)
		if !ok {
			return -1
		}
		return i
	})
	// The name box answers a press by taking the focus from the list.
	name := ""
	r := save.page.NameBox
	for _, p := range []image.Point{{r.Min.X - 1, r.Min.Y}, r.Min, {r.Max.X - 1, r.Max.Y - 1}, r.Max,
		{r.Min.X, r.Min.Y - 1}, {r.Max.X - 1, r.Max.Y}} {
		save.listFocus = true
		if err := a.HeadlessPointer("press", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		name += fmt.Sprint(save.listFocus)
		if err := a.HeadlessPointer("release", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	got["save page name"] = name

	var loaded []string
	l := loadScrollTestApp(t, loadScrollTestFrames(), &loaded)
	lv := l.flow.loadView()
	got["load page"] = layoutProbe(t, func() { loadScrollTestFrame(t, l) }, func(p image.Point) int {
		i, ok := lv.buttonAt(p)
		if !ok {
			return -1
		}
		return i
	})

	want := map[string]string{
		"dialog delete":    "2 drawn 4081d3ff7704f82e",
		"dialog form":      "9 drawn 976e0cd87ebb4476",
		"dialog overwrite": "2 drawn 192c3c41e996af83",
		"load page":        "14 drawn bcdd79a631ba8f24",
		"save page":        "6 drawn 77adfa20fda497b2",
		"save page name":   "truefalsefalsetruetruetrue",
	}
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%q: %q,", k, got[k])
		if w, ok := want[k]; !ok || w != got[k] {
			t.Errorf("%s: got %q, want %q", k, got[k], w)
		}
	}
}
