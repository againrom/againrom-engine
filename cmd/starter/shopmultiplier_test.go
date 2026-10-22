package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestShopStockMultiplierSettings(t *testing.T) {
	for _, value := range []string{"", "1", "5", "10", "1000", "0", "1001", "bad", "1.5", "999999999999999999999"} {
		t.Run(value, func(t *testing.T) {
			file := mustParse(t, "[options]\nshop-stock-multiplier = "+value+"\nfuture = keep\n")
			s := loadSettings(file)
			want := value
			if want == "" {
				want = "1"
			}
			if s.ShopStockMultiplier != want {
				t.Fatal("loaded", s.ShopStockMultiplier, "want", want)
			}
			s.store(file)
			if after := loadSettings(file); after.ShopStockMultiplier != want || !strings.Contains(string(file.Bytes()), "future = keep") {
				t.Fatal("round trip lost a setting")
			}
			args, err := s.launchArgs("root", "mods")
			valid := want == "1" || want == "5" || want == "10" || want == "1000"
			if !valid {
				if err == nil || !strings.Contains(err.Error(), "1000") {
					t.Fatal("invalid value was not visibly refused", args, err)
				}
				return
			}
			expected := []string{"-assets", "root"}
			if want != "1" {
				expected = append(expected, "-shop-stock-multiplier", want)
			}
			if err != nil || !reflect.DeepEqual(args, expected) {
				t.Fatal("launch", args, err, "want", expected)
			}
		})
	}
}

func TestShopStockMultiplierSettingsFieldAndVisibleRefusal(t *testing.T) {
	for _, language := range []string{"english", "russian"} {
		a := rigWithLanguage(t, language).app()
		openSettings(t, a)
		found := false
		for _, it := range a.layout() {
			if it.kind == iField && it.act == (action{aFocus, int(fShopStockMultiplier)}) {
				found = it.text == "1" && it.rect.Max.Y < winH
			}
		}
		if !found {
			t.Fatal("default multiplier field is missing")
		}
		typeInto(t, a, fShopStockMultiplier, "1001")
		do(t, a, action{kind: aPlay})
		if !a.statusBad || !strings.Contains(a.status, "1000") {
			t.Fatal("starter did not show the invalid multiplier", a.status)
		}
	}
}
