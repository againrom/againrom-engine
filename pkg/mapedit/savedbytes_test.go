package mapedit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// savedBytesWant pins the bytes a new map and a painted map save: the SHA-256
// of the session's bytes after each step, taken from the engine before the
// terrain table, the tile-group mask and the height limits were unified.
var savedBytesWant = map[string]string{
	"height -17":     "dfc2039bbc4a8333af92a1ec668b595d244acaf3f99620f88bbb30d99aa9f254",
	"height -300":    "e017ddb88cd6f158b942dbb880adc7c69890ec3e92b3e4c9a42ea6bcf8cd9a60",
	"height 300":     "3f609d84c1a1598bbae3f13c35981a11f69bd25603086f824cc62726a89e805b",
	"height 40":      "bc6854d7f04bd8e88544ce08d384469e31a3c0a6af05a6d77fde55d4df97a915",
	"new 0x11/0":     "fa859dae8cf9d2fd658b53fcfb30173fea89ea8f245227af32a3a50eeb53ed22",
	"new 0x11/126":   "1a4f6c828de2419250746832ac6f380d44c9aa8a7284696ebacade004ec659d5",
	"new 0x11/64":    "380d1dd53035cd837f9840e3c3077bc07082b049855247acd36037299f09b01a",
	"new 0x111/0":    "10260a1f3172c14cae2b8e194e87f8c2661f76c5592bef31b780e9f82e070422",
	"new 0x111/126":  "a99c63a3c00af984ffa344c0309e612f3d31d66913a82c81af016aba37e2662d",
	"new 0x111/64":   "759df4d813e572718b5f482a2acdbff84d45c1a21729a87234d43d55399624f8",
	"new 0x14/0":     "ea2b896f54ef8a182c076ea9e220b81282fa9def32a3163c5c55899d32dcba3b",
	"new 0x14/126":   "fd4c72734135e260ad78c963cbe9aefa6818edbef446211f39c93026e0abeebb",
	"new 0x14/64":    "f5a24c2c4c8db89c87e392a5e27cefaea89b14607b4f0d61a79338f0a10d35bd",
	"new 0x191/0":    "89e1a766f0254a914b27c38cc996d79df343b3edfbdd4f42b913c8f9c140b8aa",
	"new 0x191/126":  "dbac89c66319f6778dc4e2c456e75fbd4028a58df90a8638dba7fd42fd96eff7",
	"new 0x191/64":   "6f4838226a2e6ed2f2de85d77548e49b5f59cd357da9b238e99fc34e85868813",
	"new 0x1d1/0":    "c0df9364f139b1c2ab747e510190fdbeddaa9da9f8da7752e58007b4c1d520e3",
	"new 0x1d1/126":  "5e7947bae5bcce8097ff60531220af5599e51bdc8c2cf92f9c482081ba045fcb",
	"new 0x1d1/64":   "92bafa423fddeaead76b6a242b01d400b284fd909975183670c4af79aff4e861",
	"new 0x211/0":    "25d7d5c85ece70d516bf197d56588616aebb2f092006577ecdc883ed54700f3e",
	"new 0x211/126":  "cf2b11921e7bc96f9ef158d0020485e9e304ae40298826b8fb96181e773c2fe4",
	"new 0x211/64":   "dd4519d08cbef235e92cbda4f06f2cdbbe60313be99fe42797d9210365b78069",
	"new 0x51/0":     "42588be1eb37c52254255696e93740fa95b03c9eff7febd73188272064cc39f3",
	"new 0x51/126":   "f30209e5d6a254aa142ca1b439229f2c74af4626e3f6cb257f43f67daa926068",
	"new 0x51/64":    "864ddd8fe77c067cd3075fa7375ea65746c750a25984d231276596dfd4ebe2b8",
	"new 0x91/0":     "0c81db882de188b3bb8686ec94140d6126c94a8d4007f26fd2f00dd37f0287de",
	"new 0x91/126":   "636f8746f69c9ac5a80f350eef67bdf23e13b3ea8f07f890cfbd8a38ddf57b37",
	"new 0x91/64":    "5a23db5145740bac33e4a3c2ac400eb6508c8cc015859c8e4ff7657b52a37c63",
	"new 0xd1/0":     "6c91af9d22dc48337e906fda960e356b6936889022537cd592f5dc224091bd4d",
	"new 0xd1/126":   "eb5206c3e5cac155617a54ec647d02a05e62b45f582d75a8d8f41c7b818e8a20",
	"new 0xd1/64":    "ca2e3bcdc77828a435ef88afe3d3e21d9ef3524b7ed7e4b6c6cb589902b19301",
	"paint class 1":  "39f2678fe6c56b25a84d6c640b66c3b429891b18502c49fe4ea58a1cf5d56e7a",
	"paint class 10": "3246fcef58344266006bb62b1216168a3e799f7396d28d2a8ebdcb91afd1c0fe",
	"paint class 2":  "ffe5fb73884acdb6c4de44d68aa3b007376568bf9d655441ad649ab748f7454c",
	"paint class 3":  "221996dbe3b7512c75871e021be6e663b357816dbccdd3e76044f177aaa04244",
	"paint class 4":  "6bd929299e1d4b04868f9f82501016c5930d57370689f1c5769559e8ecb95882",
	"paint class 5":  "4aa131967ca7de40a2d01ca47b726692d3497a384336fe34fd737c55224bee09",
	"paint class 6":  "013c59c28ee9fd5b647aa3489bf509b98fed868aafcfa224ee6c3c6361639005",
	"paint class 7":  "019b1b87bdb36e4067a238b98987ea9939f5c8a109b4035981fda50a02424699",
	"paint class 8":  "678263c66aa9544e6b542a98c9a54c78a169596581125963e46abb843f128147",
	"paint class 9":  "3205bc80289b97d118544d87d01916ebfd073190aba10c83731dfd818f41e06b",
}

func savedBytesHashes(t *testing.T) map[string]string {
	t.Helper()
	got := map[string]string{}
	sum := func(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	for _, word := range []uint16{0x14, 0x11, 0x91, 0xd1, 0x51, 0x191, 0x111, 0x1d1, 0x211} {
		for _, alt := range []uint8{0, 64, 126} {
			e, err := mapedit.NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: word, Altitude: alt})
			if err != nil {
				t.Fatal(err)
			}
			got[fmt.Sprintf("new %#x/%d", word, alt)] = sum(e.Bytes())
		}
	}
	e, err := mapedit.NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	for i, class := range []uint8{mapedit.ClassLand, mapedit.ClassSand, mapedit.ClassCracked, mapedit.ClassFlowers,
		mapedit.ClassStones, mapedit.ClassSavanna, mapedit.ClassMountain, mapedit.ClassWater, mapedit.ClassRoad, mapedit.ClassGrass} {
		at := []mapedit.Cell{{X: 6 + 2*i, Y: 16}, {X: 8 + 2*i, Y: 18}}
		if _, err := e.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(class), Radius: 2}, at, testClass); err != nil {
			t.Fatal(err)
		}
		got[fmt.Sprintf("paint class %d", class)] = sum(e.Bytes())
	}
	for i, v := range []int{300, -300, 40, -17} {
		at := []mapedit.Cell{{X: 8 + 5*i, Y: 8 + 3*i}}
		if _, err := e.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: v, Radius: 4, Falloff: mapedit.FalloffCosine}, at, nil); err != nil {
			t.Fatal(err)
		}
		got[fmt.Sprintf("height %d", v)] = sum(e.Bytes())
	}
	return got
}

func TestSavedBytesAreThoseOfTheEngineBeforeUnification(t *testing.T) {
	got := savedBytesHashes(t)
	if len(savedBytesWant) == 0 {
		for k, v := range got {
			t.Logf("PIN %q: %q,", k, v)
		}
		t.Fatal("no pinned bytes")
	}
	if len(got) != len(savedBytesWant) {
		t.Fatalf("%d steps, %d pinned", len(got), len(savedBytesWant))
	}
	for k, want := range savedBytesWant {
		if got[k] != want {
			t.Errorf("%s: saved bytes changed", k)
		}
	}
}
