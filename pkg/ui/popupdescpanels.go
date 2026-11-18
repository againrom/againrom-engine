package ui

import (
	"fmt"
	"image"
	"image/color"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

type popupDescPanels struct {
	Right string `json:"right"`
	Hover string `json:"hover,omitempty"`
}

func (p *popupDescPanels) check() error {
	if p == nil {
		return nil
	}
	switch p.Right {
	case "docked", "hosted", "minimized", "floating":
	default:
		return fmt.Errorf("panel description: invalid right panel state %q", p.Right)
	}
	switch p.Hover {
	case "":
		return nil
	case "splitter":
		if p.Right == "docked" || p.Right == "hosted" {
			return nil
		}
	case "grip":
		if p.Right == "floating" {
			return nil
		}
	case "help":
		if p.Right == "docked" || p.Right == "floating" {
			return nil
		}
	}
	return fmt.Errorf("panel description: hover %q does not fit right panel state %q", p.Hover, p.Right)
}

type mockupPanel struct {
	kind        string
	spec, plain popupSpec
	layout      popupLayout
}

type mockupPanels struct {
	canvas  image.Rectangle
	panels  []mockupPanel
	pointer image.Point
	hover   string
	tip     popupTip
	decode  func(string) string
}

func buildMockupPanels(d popupDesc, lang string, window image.Point, title, body *text.Font, encode, decode func(string) string) (*mockupPanels, error) {
	if window.X < 1920 || window.Y < 1080 {
		return nil, fmt.Errorf("panel description needs a window at least 1920x1080")
	}
	if decode == nil {
		decode = func(s string) string { return s }
	}
	e := NewMapEditor(title, nil, nil)
	e.catalog = false
	e.SetHostTextEncoder(encode, decode)
	e.SetWords(words.For(lang))
	e.SetBodyFont(body)
	e.Layout(window.X, window.Y)
	e.SetPanelProfile(nil, nil, nil)

	// The current panel builders read one synthetic structure from a map.
	const side = 50
	viewer, err := NewViewer("map", terrain.Grid{Width: side, Height: side, Tiles: make([]uint16, side*side)}, &terrain.Tileset{})
	if err != nil {
		return nil, err
	}
	label := ""
	if d.Sample != nil {
		label = encode(d.Sample.Values["structure"])
	}
	e.doc = &InspectionDocument{Viewer: viewer, Width: side, Height: side, Records: []InspectionRecord{
		{Key: 1, Kind: KindStructure, Label: label, ScriptID: 1, Cell: image.Pt(18, 24), Size: image.Pt(2, 2), Spatial: true},
	}}
	right := d.Panels.Right
	switch right {
	case "hosted":
		e.Select(0, false)
	case "minimized":
		e.showPanel(panelRight, false)
	case "floating":
		s, _ := e.propsSpec()
		s.Width = max(sidePanels[panelRight].least, naturalWidth(&s, title, body, s.LabelWidth))
		s.Size, s.Bounds = image.Point{}, e.popupBounds()
		s.Docked, s.Splitter = false, 0
		size := layoutPopup(s, title, body).Rect.Size()
		size.Y += popupMinRows * (fontHeight(body, 15) + popupRowExtra)
		at := e.centreCorner(size, openPlace{})
		e.floatAt(popProps, image.Rectangle{Min: at, Max: at.Add(size)})
		e.relayout()
	}

	scene := &mockupPanels{canvas: e.mapArea(), hover: d.Panels.Hover, pointer: image.Pt(-1, -1), decode: decode}
	add := func(s popupSpec, i int) {
		l := layoutPopup(s, title, body)
		floating := e.panelFloats(i)
		part := mockupPanel{kind: "strip", spec: s, plain: encodedSpec(s, decode), layout: l}
		if floating {
			part.kind = "window"
		}
		if e.panelOpen(i) {
			part.kind = "dock"
		}
		if i == panelRight {
			var target image.Rectangle
			switch scene.hover {
			case "splitter":
				target = l.splitterRect()
			case "grip":
				target = l.gripRect()
			case "help":
				target = l.TitleHelp
			}
			if !target.Empty() {
				scene.pointer = target.Min.Add(target.Size().Div(2))
			}
			if scene.hover == "help" {
				scene.tip, _ = popupTipAt(image.Rectangle{Max: window}, title, e.bodyFont(), s, l, scene.pointer, true)
			}
		}
		scene.panels = append(scene.panels, part)
	}
	left, _ := e.treeSpec()
	add(left, panelLeft)
	report, _ := e.stripSpec(panelBottom)
	add(report, panelBottom)
	switch right {
	case "hosted":
		s, _, _ := e.objectSpec()
		add(s, panelRight)
	case "minimized":
		s, _ := e.stripSpec(panelRight)
		add(s, panelRight)
	default:
		s, _ := e.propsSpec()
		add(s, panelRight)
	}
	return scene, nil
}

func (s *mockupPanels) draw(dst *image.RGBA, title, body *text.Font) {
	ink := editorPopupInk
	drawFrame(dst, panelFrame(s.canvas, ink.Ground, editorGridInk))
	for _, p := range s.panels {
		drawPopup(dst, title, body, nil, p.spec, p.layout, ink, s.pointer, true)
	}
	drawPopupTip(dst, body, s.tip, ink)
}

func (s *mockupPanels) page(d popupDesc, lang string, window image.Point) []byte {
	return mockupPage(d, lang, window, func(part pagePartWriter) {
		noInk := color.RGBA{}
		part("map", "map", -1, s.canvas, noInk, "", "")
		for _, p := range s.panels {
			writePopupParts(func(class, name string, row int, r image.Rectangle, ink color.RGBA, txt, tip string) {
				if name == "window" {
					name = p.kind
				}
				part(class, name, row, r, ink, txt, tip)
			}, specWindow(p.plain), p.layout, image.Rectangle{}, nil)
			for _, mark := range []struct {
				name string
				rect image.Rectangle
			}{{"splitter", p.layout.splitterRect()}, {"grip", p.layout.gripRect()}} {
				class := mark.name
				if p.layout.resizeHandleHovered(s.pointer) {
					class += " hover"
				}
				part(class, mark.name, -1, mark.rect, noInk, "", "")
			}
		}
		part.tip(s.tip, s.decode)
	})
}
