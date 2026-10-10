package game

import (
	"testing"

	"againrom/pkg/ui"
)

// The first game's generator lists its four commands at the shop's four plaque
// rectangles, Accept, Restore, Reset and Back from top to bottom, over the
// shop's 176-wide menu with no seam, each plaque's bitmap pressed-only and
// keyed (owner, DIV-2868 after DIV-483). The second game keeps its three.
func TestFirstGeneratorCommandsAreTheShopPlaques(t *testing.T) {
	d := generatorDescriptions["rom1"].Detail
	if d.Nav.Rect.Rectangle() != ui.TownWideUpperRegion || !d.Nav.Keyed || d.NavSeam != nil {
		t.Fatalf("panel body %v keyed %t seam %v; want the shop menu over %v, keyed, no seam",
			d.Nav.Rect.Rectangle(), d.Nav.Keyed, d.NavSeam, ui.TownWideUpperRegion)
	}
	at := map[string]int{"play": 0, "restore": 1, "reset": 2, "back": 3}
	if len(d.Commands) != len(at) {
		t.Fatalf("%d commands, want 4", len(d.Commands))
	}
	for _, c := range d.Commands {
		i, ok := at[c.Role]
		if !ok {
			t.Fatalf("unexpected command %q", c.Role)
		}
		r := ui.ShopButtonRect(i)
		if c.Rect.Rectangle() != r || c.Off != "" || c.On == "" || !c.Keyed || c.Size[0] != r.Dx() || c.Size[1] != r.Dy() {
			t.Errorf("%s: rect %v off %q on %q keyed %t size %v; want the shop plaque %d %v, pressed-only and keyed",
				c.Role, c.Rect.Rectangle(), c.Off, c.On, c.Keyed, c.Size, i, r)
		}
	}
	if n := len(generatorDescriptions["rom2"].Detail.Commands); n != 3 {
		t.Fatalf("the second game lists %d commands, want 3", n)
	}
}
