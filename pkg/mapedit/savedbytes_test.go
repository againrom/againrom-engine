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
// terrain table, the tile-group mask and the height limits were unified, then
// repinned for the new map's two players and zeroed text region (ALM-NEWMAP-257,
// ALM-NEWMAP-258), which change every new map's bytes, and for the corner rule
// of terrain strokes, which changes the painted plane the later steps carry.
var savedBytesWant = map[string]string{
	"height -17":     "0213ebfb624b77e877ccc017d859bcca22248c446905d232cd3f440da761e18c",
	"height -300":    "7321f7a6a7196fb7d1a7a3c085eec42171d27d564ca8cbdfa246457ad3116af5",
	"height 300":     "851efcb7ef7104ae3ea857fa7e95412b98b9abfb77245047d20434455eba0510",
	"height 40":      "f3150b94b83ae43d89af7a4d9efafed66c8463753a7aae689d42743cab5549ae",
	"new 0x11/0":     "e5a0279651977f6affd8e972d1b8e5f53468661d87484069a06db3a1bbef5b48",
	"new 0x11/126":   "b21ed29a3b2ae2dd81c6a99ccb0664aa807601165d651e9fd61403609a8634b1",
	"new 0x11/64":    "587a33023c1c6f9f95e44e3f4b4ad00547419fd71759a7e918952b0ba2e94d54",
	"new 0x111/0":    "74ab43b3a68162440e8fb3d95911ca13fb7592fc3e2e6f26f3e5cae5fd6dd314",
	"new 0x111/126":  "d58b5f94b8c6ad91e5e94956368512ccef802ae1c2aef541a928e62315a2a703",
	"new 0x111/64":   "056e3e4e89a540d4890ad05d6b2e50187551a2c63214885db944808417050a12",
	"new 0x14/0":     "a52041f1565fa301ee59b64f15764d289b1550d517646633a47edd61a5a9c82b",
	"new 0x14/126":   "239a35d55c5deb255a330efab816fd2481135505e4d03039ef51aaec7b8d8e67",
	"new 0x14/64":    "94d5d1d4553d3580bc7efaac154233332374a6b546a95c8fd36a990c74ff3092",
	"new 0x191/0":    "e96eff765e1d4767361cba8fade54428d61b6b418440bc4878ab87cf03ca87fc",
	"new 0x191/126":  "c711d547724596c3b9fa16b60ee3b8e81ac47a9ceab3833062000f4af0045462",
	"new 0x191/64":   "3e7de57b4303e00597c750269a0ca4bf54fe8f38925aef786c47403aadc4cd6b",
	"new 0x1d1/0":    "94a6f9a75d54371da4a00bb385ab14cc478fd7268e76de577b6070aef1f0022b",
	"new 0x1d1/126":  "d64973a8a1865a2ede0bf0021952d1dc89519f3c3d319bb425dc4153403b585e",
	"new 0x1d1/64":   "771e4ff7d8defb1a22d89c0e120fc26d9ea218bf9a3f3254d5a5973102ba9c6d",
	"new 0x211/0":    "033a4cd5cd757c8dafd89f6d04b25f1c34d94f1fa29a795be97387b560e8afc3",
	"new 0x211/126":  "9f56c77a967d1406a8a0b7f466cc894f51516db549eaadd1cc2d9e5d720f0c42",
	"new 0x211/64":   "8441d15f2b2b447ff77de0b55995a2bc6b52bc56b9993c0b0082951752c02424",
	"new 0x51/0":     "7c0330b3116185f20a9d6688d073614ded0379c86200be0a0d9764403517f7e6",
	"new 0x51/126":   "8ad71c70de92b56346abca8251d0c15a0eaa11de7974cd702e840243849f95f2",
	"new 0x51/64":    "38834f419779bff0be50e48708a7543d78fa1ba8d3f9351cb3a77b5191c08fab",
	"new 0x91/0":     "0d3357e9d44cd3dd93ac71fe600461f907196753c2d04bf5941e21ff40ec6d94",
	"new 0x91/126":   "2e2f2202a0eb781c24401ee867a284c8d263be495b5150c42c2da6152a4711bc",
	"new 0x91/64":    "de085c700fc9bf912d86ea797a6d52993a964c19186e1ef63ff16052a962a368",
	"new 0xd1/0":     "b4786b85b43d0e20d7ac1c25e711a6e19bbdfd4b1253b7ccd2d9a0ed52670402",
	"new 0xd1/126":   "9f3322eeed3cfd88f335c638f542f1764767709aa2be841dd77f67efddda944f",
	"new 0xd1/64":    "b38b38e053288e18684aabcbc932882728e4573c6a7869ea1fad2c482c4f0de1",
	"paint class 1":  "c9ccfe7f4901f0ba910c86f70bb43856f5b22cc01ee89e428d5a481594bf8443",
	"paint class 10": "e1bf19fc5978277c7b2a2d314f447cc756ed2a9d09f50cf474cd977e910c2a1b",
	"paint class 2":  "1cc38662494ce155c2cee98e94bb570c888f38cf1225059ad92348978ea1fa16",
	"paint class 3":  "581e98e3317116beb3042810e20c7460db9070e3de6632f6ea9ca52d76f17565",
	"paint class 4":  "e813fc0661775839acc5552d484687b152d4a472eb5c997264790bcd55deb5f3",
	"paint class 5":  "c2eb4e3ecbc4bf82d326058bfbc72bfa0036148d9bcd29a86b6b7f58ecd675a3",
	"paint class 6":  "ad44b4094e8b003192ac25c7130e7b6cb83bc44d49a2d5af39f7676281c7af94",
	"paint class 7":  "7fffb88a021618c0beeb3e7d443c90d0cbcb4abcdcbbfd8c33fe777f03146c4d",
	"paint class 8":  "bdb940910db0f749538613fdb86e3e825b2732e1bed199fbd61acb97a04662fd",
	"paint class 9":  "58d5d398195ccbfe99f24902103879d1835edcbcbd5f5cd15fa4d3084b0a8ca0",
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
			t.Errorf("%s: saved bytes changed to %s", k, got[k])
		}
	}
}
