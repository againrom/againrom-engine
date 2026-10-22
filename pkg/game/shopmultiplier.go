package game

import (
	"fmt"

	"againrom/pkg/launch"
)

const MaxShopStockMultiplier = launch.MaxShopStockMultiplier

func ParseShopStockMultiplier(value string) (int, error) {
	return launch.ShopStockMultiplier.Whole(value)
}

func (f *FrontEnd) SetShopStockMultiplier(n int) error {
	if st := launch.ShopStockMultiplier; !st.InBounds(n) {
		return fmt.Errorf("shop stock multiplier must be from %d to %d", st.Min, st.Max)
	}
	f.runtime.shopStockMultiplier = int32(n)
	return nil
}
