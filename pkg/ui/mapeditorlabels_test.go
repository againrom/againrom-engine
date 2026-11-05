package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"testing"
)

func mapLabelEditor(t *testing.T) *MapEditor {
	t.Helper()
	e := newMapEditorFixture(t)
	e.Layout(1024, 768)
	e.selected, e.objectHidden = -1, true
	e.doc.Records = []InspectionRecord{
		{Kind: KindEnchantment, Spatial: true, Cell: image.Pt(20, 20), Marker: "Lightning marker"},
		{Kind: KindTrigger, Index: 0},
		{Kind: KindTrigger, Index: 1},
	}
	e.doc.Script = &InspectionScriptModel{
		Triggers: []InspectionTrigger{{Record: 1, Name: "First"}, {Record: 2, Name: "Second"}},
		Areas: []InspectionArea{
			{Trigger: 0, Rect: image.Rect(20, 20, 21, 21)},
			{Trigger: 1, Rect: image.Rect(20, 20, 22, 22)},
		},
	}
	e.doc.Viewer.grid.Altitudes = make([]uint8, 2500)
	e.doc.Viewer.cam.SetZoom(1)
	return e
}

func mapLabelAnchor(e *MapEditor, marker bool, at image.Point) {
	ox, oy := e.mapOrigin()
	var x, y float64
	if marker {
		wx, wy := e.cellCenter(e.doc.Records[0].Cell)
		x, y = e.doc.Viewer.cam.WorldToScreen(wx, wy)
		x += 7
	} else {
		sx, sy := e.corner(e.doc.Script.Areas[0].Rect.Min)
		x, y = float64(sx), float64(sy)
	}
	e.doc.Viewer.cam.X += x + ox - float64(at.X)
	e.doc.Viewer.cam.Y += y + oy - float64(at.Y)
}

func TestEditorMapLabelPixels(t *testing.T) {
	cases := []string{
		"triggers", "markers", "collision", "selected", "popup-left", "popup-covered",
		"trigger-left", "trigger-right", "trigger-top", "trigger-bottom",
		"marker-left", "marker-right", "marker-top", "marker-bottom",
		"heights", "group", "height-occupancy", "height-gesture", "no-font",
	}
	want := map[string]string{
		"triggers":         "0891f432624d4de4524357b5d596a124a5367490dce9d5547664c63283a139c5",
		"markers":          "bb86641e6364a03f5154984d51e4da50a6b29eafef89be52b2407874f457bd8b",
		"collision":        "6ff7d2c3e33d80b20eac22c90cac02744eae19a4b39e4b4b9050e14b2902633b",
		"selected":         "714db741788f269090bb96a384f9d14c15baa7275875b9afbe4fed2dd9d99ab0",
		"popup-left":       "236b7a7f00b1fc3e68cae678d185724f15e4768eb3cc67301dc7a640a29391db",
		"popup-covered":    "ad74fd0bbe48ab359d6c896f66ef6284e995abac982094e29486b04780fb44ca",
		"trigger-left":     "df0c89b02544a0377baf82136c1537b0f6a753bfbe112fa3a25100d5ad4031ba",
		"trigger-right":    "df0c89b02544a0377baf82136c1537b0f6a753bfbe112fa3a25100d5ad4031ba",
		"trigger-top":      "df0c89b02544a0377baf82136c1537b0f6a753bfbe112fa3a25100d5ad4031ba",
		"trigger-bottom":   "df0c89b02544a0377baf82136c1537b0f6a753bfbe112fa3a25100d5ad4031ba",
		"marker-left":      "51e9039c8704a8e93f7578c0fbe62cbbc93871916b0a0e56e0ea95cde2b63977",
		"marker-right":     "79773f3ec0e8e5c44a0f5f7009185a998fb55e8f7c895a7dc7b06e88814fa3f0",
		"marker-top":       "bdaf0e987d8ab8ab744716107618d2466cf384d615c7314f3251d8eab22ed332",
		"marker-bottom":    "b0f42b5dda55cac287a0c12967840d8feef320b2336e74c2a3beb9448dfbd3cf",
		"heights":          "1cd0f08a9f497f2575c6e5fa587f81a59a2a399a5b8f9b200a97fa05c0f096aa",
		"group":            "98ff084ab26773531165261a9e8b9eb2320dbea779b3ae7d9aa1382645f9bc20",
		"height-occupancy": "6e6cd21acda8a4957d7beff6286e17a7de3217f098e5f5c16da8b94deee59a46",
		"height-gesture":   "ce95d15f30a086eb9941b68b4041c141a4701a5627912d65c9dc53eec4a7f82b",
		"no-font":          "32c363f1554f9868cb220ab5b18e076cd74f6998e71b83f04a0e72756239ad80",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			e := mapLabelEditor(t)
			area := e.mapArea()
			at := area.Min.Add(image.Pt(180, 160))
			marker := false
			switch name {
			case "triggers":
				e.doc.Records[0].Marker = ""
			case "markers":
				e.view.hideAreas = true
			case "collision", "height-occupancy":
				e.doc.Viewer.cam.SetZoom(0.25)
			case "selected":
				e.selected = 1
			case "popup-left", "popup-covered":
				e.message = "Popup over a trigger label"
				p := e.popupRects()
				if len(p) == 0 {
					t.Fatal("missing popup")
				}
				at = p[0].Min.Add(image.Pt(-12, 18))
				if name == "popup-covered" {
					at.X = p[0].Min.X + 5
				}
			case "trigger-left", "marker-left":
				at.X = area.Min.X - 12
			case "trigger-right", "marker-right":
				at.X = area.Max.X - 12
				if name == "trigger-right" {
					at.X = area.Max.X + 3
				}
			case "trigger-top", "marker-top":
				at.Y = area.Min.Y + 3
			case "trigger-bottom", "marker-bottom":
				at.Y = area.Max.Y + 3
			case "height-gesture":
				e.brush.down = true
				ox, oy := e.mapOrigin()
				e.brush.set = &heightSet{target: 93, pointer: at.Sub(image.Pt(int(ox), int(oy)))}
			case "no-font":
				e.font, e.body = nil, nil
			}
			if len(name) >= 7 && name[:7] == "marker-" {
				marker, e.view.hideAreas = true, true
			}
			if len(name) >= 8 && name[:8] == "trigger-" {
				e.doc.Records[0].Marker = ""
			}
			if name == "heights" || name == "height-occupancy" {
				e.view.heights = true
			}
			if name == "group" || name == "height-occupancy" {
				e.groupSel = editorGroup{set: true, group: 17, members: []int{0}}
			}
			mapLabelAnchor(e, marker, at)
			if name == "popup-left" {
				labels := e.mapLabelsOf(labelTrigger)
				if len(labels) != 2 || labels[0].rect.Min.X >= at.X {
					t.Fatalf("popup did not move trigger labels left: %+v", labels)
				}
			}
			if name == "popup-covered" && len(e.mapLabelsOf(labelTrigger)) != 0 {
				t.Fatal("covered trigger was not omitted")
			}
			if name == "collision" {
				triggers, markers := e.mapLabelsOf(labelTrigger), e.mapLabelsOf(labelRecord)
				if len(triggers) != 2 || len(markers) != 1 || markers[0].rect.Min.Y < triggers[1].rect.Max.Y {
					t.Fatalf("collision not exercised: triggers=%+v markers=%+v", triggers, markers)
				}
			}
			if len(name) >= 8 && name[:8] == "trigger-" && len(e.mapLabelsOf(labelTrigger)) != 0 {
				t.Fatal("edge trigger was not omitted")
			}
			if marker {
				labels := e.mapLabelsOf(labelRecord)
				if len(labels) != 1 || labels[0].rect.In(area) {
					t.Fatalf("marker was not clipped: %+v", labels)
				}
			}
			pic := e.ChromeImage()
			hash := sha256.New()
			for y := area.Min.Y; y < area.Max.Y; y++ {
				i := pic.PixOffset(area.Min.X, y)
				hash.Write(pic.Pix[i : i+area.Dx()*4])
			}
			got := fmt.Sprintf("%x", hash.Sum(nil))
			t.Logf("%s %s", name, got)
			if expected := want[name]; got != expected {
				t.Fatalf("map chrome SHA-256 %s, want %s", got, expected)
			}
		})
	}
}
