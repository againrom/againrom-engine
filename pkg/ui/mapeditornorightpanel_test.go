package ui

import (
	"encoding/json"
	"image"
	"testing"
)

func TestMapEditorRecordWindowFloatsWithoutRightPanel(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := panelsEditor(t, 1920, 1080, lang)
			e.Select(0, false)
			s, _, ok := e.objectSpec()
			if !ok {
				t.Fatal("selected record has no window")
			}
			r := layoutPopup(s, e.font, e.bodyFont()).Rect
			if s.Fit || r.Min != e.centreCorner(r.Size(), openPlace{}) {
				t.Errorf("record window %v, want a floating window centred at %v", r, e.centreCorner(r.Size(), openPlace{}))
			}
			for _, r := range e.panelRects() {
				if r.Min.X > 0 && r.Max.X == 1920 && r.Min.Y < e.mapArea().Max.Y {
					t.Errorf("right panel rectangle %v", r)
				}
			}
			if got := e.mapArea().Max.X; got != 1920 {
				t.Errorf("map right edge = %d, want 1920", got)
			}
			d := image.Pt(-50, -30)
			dragTitle(t, e, d)
			at := r.Min.Add(d)
			s, _, _ = e.objectSpec()
			if got := layoutPopup(s, e.font, e.bodyFont()).Rect.Min; got != at {
				t.Errorf("record window at %v, want stored corner %v", got, at)
			}
		})
	}
}

func TestMapEditorLegacyRightPanelProfile(t *testing.T) {
	e, _, _ := panelsEditor(t, 1920, 1080, "")
	const legacy = `{"left":{"minimized":true,"width":350},"right":{"minimized":false,"width":660,"float":{"x":140,"y":160,"width":420,"height":300},"view":"technical"},"bottom":{"open":true,"width":220}}`
	var saved []byte
	e.SetPanelProfile([]byte(legacy), nil, func(data []byte) error {
		saved = data
		return nil
	})
	if got := e.panels.state[panelLeft]; got != (panelState{Width: 350}) {
		t.Errorf("left panel state = %+v", got)
	}
	if got := e.panels.state[panelBottom]; got != (panelState{Restored: true, Width: 220}) {
		t.Errorf("bottom panel state = %+v", got)
	}
	if e.props.plain {
		t.Error("legacy technical view was lost")
	}
	e.savePanels()
	var profile map[string]json.RawMessage
	if err := json.Unmarshal(saved, &profile); err != nil {
		t.Fatal(err)
	}
	if _, ok := profile["right"]; ok {
		t.Errorf("saved profile still contains right panel: %s", saved)
	}
	e.SetPanelProfile(saved, nil, nil)
	if e.props.plain {
		t.Error("technical view was lost after saving")
	}
}
