package main

import (
	"reflect"
	"testing"

	"againrom/pkg/game"
)

func TestShopStockMultiplierFlag(t *testing.T) {
	control, err := parse(nil)
	if err != nil || control.shopStockMultiplier != 1 {
		t.Fatal("default", control.shopStockMultiplier, err)
	}
	for _, value := range []string{"1", "5", "10", "0", "11", "bad", "1.5", "999999999999999999999"} {
		got, err := parse([]string{"-shop-stock-multiplier", value})
		valid := value == "1" || value == "5" || value == "10"
		if (err == nil) != valid {
			t.Fatal("flag", value, err)
		}
		if valid {
			n, _ := game.ParseShopStockMultiplier(value)
			if got.shopStockMultiplier != n {
				t.Fatal("wrong multiplier", got.shopStockMultiplier)
			}
			got.shopStockMultiplier = 1
			if !reflect.DeepEqual(got, control) {
				t.Fatal("multiplier changed another option")
			}
		}
	}
}
