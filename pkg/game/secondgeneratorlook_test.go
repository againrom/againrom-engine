package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The second game's generator description states the original's look: the
// pre-create hover help lines (R2-ENGINE-384), the name field's inks
// (R2-ENGINE-386), the detail tip where the owner directs it (DIV-3015), the
// attribute values' face, ink, shadow and anchors (R2-ENGINE-387), four
// commands on the shop's command composition (R2-ENGINE-249, DIV-3014), the
// hover box style (R2-ENGINE-385) and the statistics card canvas (DIV-3017).
func TestSecondGeneratorDescribesTheOriginalLook(t *testing.T) {
	d := generatorDescriptions["rom2"]
	pre, det := &d.PreCreate, &d.Detail
	slot := func(n *int) int {
		if n == nil {
			return -1
		}
		return *n
	}
	t.Run("difficulty and hero hover help", func(t *testing.T) {
		for i, want := range []int{247, 248, 249} {
			if got := slot(pre.Levels[i].Tooltip); got != want {
				t.Errorf("level %d hover line %d, want %d", i, got, want)
			}
		}
		lines := map[int]int{80: 250, 100: 252, 120: 253, 140: 251}
		for i, h := range pre.Heroes {
			if got := slot(h.Tooltip); got != lines[h.Mask] {
				t.Errorf("hero %d (mask %d) hover line %d, want %d", i, h.Mask, got, lines[h.Mask])
			}
		}
		if slot(pre.Back.Tooltip) != 255 || slot(pre.Forward.Tooltip) != 254 || slot(pre.Name.Tooltip) != 256 {
			t.Errorf("Cancel, OK and name hover lines %d, %d, %d; want 255, 254, 256",
				slot(pre.Back.Tooltip), slot(pre.Forward.Tooltip), slot(pre.Name.Tooltip))
		}
	})
	t.Run("name inks", func(t *testing.T) {
		if pre.Name.Prompt.Ink != (ui.GeneratorInk{185, 159, 73}) || pre.Name.Ink != (ui.GeneratorInk{255, 255, 116}) {
			t.Errorf("prompt ink %v, name ink %v; want gold (185,159,73) and yellow (255,255,116)", pre.Name.Prompt.Ink, pre.Name.Ink)
		}
	})
	t.Run("detail tip between the card and the portrait", func(t *testing.T) {
		if got := TipTable(base.GameROM2.Edition()).Row("detail-enter", "", "fighter").Rectangle(); got != image.Rect(160, 280, 472, 480) {
			t.Errorf("detail tip %v, want (160,280)-(472,480)", got)
		}
	})
	t.Run("attribute values", func(t *testing.T) {
		st := &det.Stats
		gold := ui.GeneratorInk{185, 159, 73}
		switch {
		case st.ValueFont == nil || st.ValueFont.Name != "font4" || st.ValueFont.Atlas != "16a":
			t.Errorf("value font %+v, want font4 16a", st.ValueFont)
		case st.ValueInk != gold || st.PoolInk != gold:
			t.Errorf("value ink %v, pool ink %v; want %v", st.ValueInk, st.PoolInk, gold)
		case st.ValueShadow == nil || *st.ValueShadow != (ui.GeneratorInk{}):
			t.Errorf("value shadow %v, want black", st.ValueShadow)
		case st.ValueAt == nil || st.Value[0].Rectangle().Min.Add(st.ValueAt.Pt()) != image.Pt(84, 58):
			t.Errorf("value offset %v, want the first value at (84,58)", st.ValueAt)
		case st.PoolAt == nil || st.PoolAt.Pt().X != 84:
			t.Errorf("pool anchor %v, want centred on x 84", st.PoolAt)
		}
	})
	t.Run("four commands on the shop composition", func(t *testing.T) {
		if det.Nav.Key != "graphics/interface/shopmenu.bmp" || det.Nav.Rect.Rectangle() != image.Rect(464, 0, 640, 238) {
			t.Errorf("command panel %s at %v, want shopmenu.bmp at the shop's buttons child (464,0)-(640,238)", det.Nav.Key, det.Nav.Rect)
		}
		want := map[string]image.Rectangle{"play": image.Rect(494, 15, 614, 67), "restore": image.Rect(483, 67, 623, 113),
			"reset": image.Rect(483, 114, 623, 160), "back": image.Rect(494, 160, 614, 212)}
		if len(det.Commands) != 4 {
			t.Fatalf("%d commands, want 4", len(det.Commands))
		}
		for _, c := range det.Commands {
			if c.Rect.Rectangle() != want[c.Role] {
				t.Errorf("%s at %v, want %v", c.Role, c.Rect, want[c.Role])
			}
		}
		if r := det.RestoreFor("russian"); r.Values != "template" || r.Skill != "preset" {
			t.Errorf("Restore %+v, want the template and the starting skill", r)
		}
	})
	t.Run("hover box style", func(t *testing.T) {
		h := d.Hover
		if h == nil || h.Font != "text" || h.Ink != (ui.GeneratorInk{160, 120, 50}) || h.Shadow == nil ||
			*h.Shadow != (ui.GeneratorInk{8, 8, 8}) || h.Lines != "authored" {
			t.Errorf("hover style %+v, want font2 (160,120,50) over (8,8,8), authored lines", h)
		}
	})
	t.Run("statistics card canvas", func(t *testing.T) {
		if det.CardOffset != (ui.GeneratorPoint{12, 0}) || det.CardSeam == nil || det.CardSeam.Key != "graphics/interface/chrgen/fullstatsr.bmp" {
			t.Errorf("card offset %v, seam %+v; want (12,0) and fullstatsr.bmp", det.CardOffset, det.CardSeam)
		}
		if !slices.Contains(d.Detail.Card.Cite, "R2-ENGINE-388") {
			t.Error("the card cites no sheet claim")
		}
	})
}
