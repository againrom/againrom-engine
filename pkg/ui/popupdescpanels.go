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
	Left  string `json:"left"`
	Hover string `json:"hover,omitempty"`
}

func (p *popupDescPanels) check() error {
	if p == nil {
		return nil
	}
	switch p.Left {
	case "docked", "minimized", "floating":
	default:
		return fmt.Errorf("panel description: invalid left panel state %q", p.Left)
	}
	switch p.Hover {
	case "":
		return nil
	case "splitter":
		if p.Left == "docked" {
			return nil
		}
	case "grip":
		if p.Left == "floating" {
			return nil
		}
	}
	return fmt.Errorf("panel description: hover %q does not fit left panel state %q", p.Hover, p.Left)
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
	switch d.Panels.Left {
	case "minimized":
		e.showPanel(panelLeft, false)
	case "floating":
		e.undockPanel(panelLeft)
	}

	scene := &mockupPanels{canvas: e.mapArea(), hover: d.Panels.Hover, pointer: image.Pt(-1, -1)}
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
		if i == panelLeft {
			var target image.Rectangle
			switch scene.hover {
			case "splitter":
				target = l.splitterRect()
			case "grip":
				target = l.gripRect()
			}
			if !target.Empty() {
				scene.pointer = target.Min.Add(target.Size().Div(2))
			}
		}
		scene.panels = append(scene.panels, part)
	}
	left, minimized := e.stripSpec(panelLeft)
	if !minimized {
		left, _ = e.treeSpec()
	}
	report, _ := e.stripSpec(panelBottom)
	add(report, panelBottom)
	add(left, panelLeft)
	return scene, nil
}

func (s *mockupPanels) draw(dst *image.RGBA, title, body *text.Font) {
	ink := editorPopupInk
	drawFrame(dst, panelFrame(s.canvas, ink.Ground, editorGridInk))
	for _, p := range s.panels {
		drawPopup(dst, title, body, nil, p.spec, p.layout, ink, s.pointer, true)
	}
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
			}, specWindow(p.plain), p.layout, image.Rectangle{}, nil, false)
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
	})
}
