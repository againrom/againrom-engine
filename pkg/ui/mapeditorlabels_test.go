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
		"triggers":         "9fbf59d62771127e379f3a34dbf07b175d00bc10ac89923f2027f1974978dcc7",
		"markers":          "d3da96591fe009941cdcb66b14004cbf66275335f264c0b2f66b42f5c539aaf1",
		"collision":        "4769299399946e2a0f460f650eb4c6068e567a73f3566d6c729d04270d2bf7b2",
		"selected":         "efd8911ad5c389362e8477f1392bfcca03d8003892ae570717123b8ba8902f67",
		"popup-left":       "2a8f07b71a5ddb8249d39cec30af9651f1b0c22264f7c604d752c286fb7bc708",
		"popup-covered":    "50423dcf8ff826c275e9c16028c666cdbd9566974c9713d8d23d575f5d337f7e",
		"trigger-left":     "3dde879cae4bc3ee3404fc28f8d9b368c3c829528320e3e0cd82461743646a00",
		"trigger-right":    "3dde879cae4bc3ee3404fc28f8d9b368c3c829528320e3e0cd82461743646a00",
		"trigger-top":      "3dde879cae4bc3ee3404fc28f8d9b368c3c829528320e3e0cd82461743646a00",
		"trigger-bottom":   "3dde879cae4bc3ee3404fc28f8d9b368c3c829528320e3e0cd82461743646a00",
		"marker-left":      "e93fc2bf5a7b01aec7657010583f55ca137762c450fbeb3deedab40236555342",
		"marker-right":     "7918d53093d68b5b7a974910db5adbd3e3dec8215a0dee96a4cf0a4dcac34a8c",
		"marker-top":       "fb738cb9064307440cfe43f99717f8fedd87b0318876c92f62ec2a48fe55a4c7",
		"marker-bottom":    "554bc5c4966c997039162466cf3ca6c74d6844ab0cc971024fecb505c469b775",
		"heights":          "12d50b702d1e1e34eaa9a9aa793f6d8422d7585ab0bd4430112a6a19c204cc57",
		"group":            "540e51d068bc96c3f95c1cc3c28e51eec940c5dbdd3632a148408c19ccb51707",
		"height-occupancy": "79a4afbee4fc4e9bd1d1bce0c27722e74b1dd7ff299dd3518594137fa74fab86",
		"height-gesture":   "c9d4fd3fcb807244b5300cd1c3cfe5ae2a7fcc7afefefd083b0d081302c8eb4b",
		"no-font":          "4a81bd86ec16d800fc796eeff0d8213c47531f280e20b749e38de162a5bec848",
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
