package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestMissionPackKeepsBaseIconsAndCarriesExactStarSelector(t *testing.T) {
	code := uint16(0x4321)
	src := missionSource{graphicsPrefix + data.ItemIconPath(data.ItemCode(code)): packIconStream(0x19, 0x2b, 0x3d)}
	cache := map[uint16]*image.RGBA{}
	plain := sim.PlainStack(code, 1)
	enchanted := sim.StackItem(sim.ItemInstance{Code: code, Kind: 2, Effects: []sim.ItemEffect{{Kind: 41, Operand: 9}}}, 1)
	potion := sim.StackItem(sim.ItemInstance{Code: code, Kind: 3, Effects: []sim.ItemEffect{{Kind: 17, Operand: 5}}}, 1)

	icons, _, unread := buildInventoryPack(src, []sim.ItemStack{plain, enchanted, potion}, cache)
	if len(unread) != 0 || len(icons) != 3 {
		t.Fatalf("pack icons = %v, unread %v", icons, unread)
	}
	base := cache[code]
	for i, icon := range icons {
		if icon != base {
			t.Fatalf("icon %d = %p, want shared base %p; a trail must not be baked into the icon", i, icon, base)
		}
	}
	if px := base.RGBAAt(0, 0); px != (color.RGBA{R: 0x19, G: 0x2b, B: 0x3d, A: 0xff}) {
		t.Fatalf("shared code cache was changed to %+v", px)
	}
	want := []bool{false, true, false}
	got := itemStackStarFlags([]sim.ItemStack{plain, enchanted, potion})
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("star flags = %v, want %v (Potion explicitly clears the bit)", got, want)
		}
	}
}

func TestShopThreeGridsCarryStarsButWornAndHeldPicturesStayBase(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 80, 80))
	plain := sim.PlainItem(0x1111)
	enchanted := sim.ItemInstance{Code: plain.Code, Kind: 2, Effects: []sim.ItemEffect{{Kind: 17, Operand: 5}}}
	potion := enchanted.Clone()
	potion.Kind = 3

	_, s := shopRoom(t, nil)
	s.shopIconCache = map[uint16]*image.RGBA{plain.Code: base}
	shelf := s.shopShelfCell(shopItemFromInstance(enchanted, 1), 0)
	place := s.shopPlaceCell(ShopPlace{ShopItem: shopItemFromInstance(enchanted, 1), Mine: true})
	pack := s.shopPackCell(sim.StackItem(enchanted, 1))
	for name, cell := range map[string]struct {
		Icon *image.RGBA
		Star bool
	}{
		"shelf": {shelf.Icon, shelf.Star},
		"table": {place.Icon, place.Star},
		"pack":  {pack.Icon, pack.Star},
	} {
		if cell.Icon != base || !cell.Star {
			t.Fatalf("%s cell = icon %p star %v, want base %p and animated trail", name, cell.Icon, cell.Star, base)
		}
	}
	if got := s.shopShelfCell(shopItemFromInstance(potion, 1), 0); got.Icon != base || got.Star {
		t.Fatalf("Potion cell = icon %p star %v, want base %p and no trail", got.Icon, got.Star, base)
	}

	member := &s.sess.Carried[0]
	member.Carry.Equipped[0] = plain.Code
	member.Carry.EquippedItems[0] = enchanted.Clone()
	if got := s.ShopScreen().SlotIcon[0]; got != base {
		t.Fatalf("worn/held icon = %p, want unmarked base %p", got, base)
	}
}
