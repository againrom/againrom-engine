package archtest

import (
	"os"
	"strings"
	"testing"
)

const shopStateFixture = `package game
type Shop struct { records []secondshop.Record; shopDeal[ShopItem, ShopShelf] }
func (s *Shop) rules() shopRules[ShopItem, ShopShelf] { return shopRules[ShopItem, ShopShelf]{} }
`

func TestShopStatePlantedShapes(t *testing.T) {
	for _, row := range []struct{ shape, source string }{
		{"shop-state", `type extra struct { records []secondshop.Record }`},
		{"shop-state", `type extra struct { records shopRecords }`},
		{"shop-state", `type extra struct { stock secondshop.Stock }`},
		{"shop-state", `type extra struct { pending []ShopPlace }`},
		{"shop-state", `type extra struct { Shop }`},
		{"shop-rules", `func (s *extra) rules() shopRules[ShopItem, ShopShelf] { return shopRules[ShopItem, ShopShelf]{} }`},
		{"shop-rules", `func extraRules() shopRules[ShopItem, ShopShelf] { return shopRules[ShopItem, ShopShelf]{} }`},
		{"shop-game-choice", `func caller() { if p.Game == base.GameROM2 { shop.rules() } }`},
		{"shop-game-choice", `func caller() { switch p.Game { case base.GameROM1: shop.wholeBasket=true } }`},
		{"shop-game-choice", `func caller() { if s.second != nil { shop.RestockWith(nil,0,nil) } }`},
		{"shop-game-choice", `func caller() { if t.hostRules().Shop == town.ShopBank { return t.bankShopShown() } }`},
	} {
		t.Run(row.source, func(t *testing.T) {
			vs := ShopStateSites(map[string]string{"pkg/game/shop.go": shopStateFixture, "pkg/game/planted.go": "package game\n" + row.source})
			for _, v := range vs {
				if strings.Contains(v.From, "pkg/game/planted.go:") && strings.Contains(v.From, row.shape) {
					return
				}
			}
			t.Fatalf("plant %s was not named: %v", row.shape, vs)
		})
	}
	if vs := ShopStateSites(map[string]string{"pkg/game/shop.go": shopStateFixture}); len(vs) != 0 {
		t.Fatal(vs)
	}
}

func TestShopStateOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range ShopStateSites(files) {
		t.Error(v)
	}
}
