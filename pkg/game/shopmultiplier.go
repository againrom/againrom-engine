package game

import (
	"fmt"
	"strconv"
)

const MaxShopStockMultiplier = 10

func ParseShopStockMultiplier(value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > MaxShopStockMultiplier {
		return 0, fmt.Errorf("-shop-stock-multiplier must be a whole number from 1 to %d", MaxShopStockMultiplier)
	}
	return n, nil
}

func (f *FrontEnd) SetShopStockMultiplier(n int) error {
	if n < 1 || n > MaxShopStockMultiplier {
		return fmt.Errorf("shop stock multiplier must be from 1 to %d", MaxShopStockMultiplier)
	}
	f.runtime.shopStockMultiplier = int32(n)
	return nil
}
