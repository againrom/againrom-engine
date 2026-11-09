package game

import (
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/words"
)

func decodeFixture() *mapload.Table {
	return &mapload.Table{
		Weapons:    dbCollection{{}, {name: "Two Handed Sword"}},
		Shields:    dbCollection{{}, {name: "Buckler"}},
		Armors:     dbCollection{{}, {name: "Chain Helm"}, {name: "Soft gauntlets"}},
		Materials:  fixtureScale{names: []string{"None", "Iron", "Meteoric", ""}},
		MagicItems: dbCollection{{}, {}, {}, {}, {}, {}, {}, {}, {name: "Potion of Health"}, {name: "Scroll of Fire"}, {name: "Book of Fire"}, {name: "Ring"}},
	}
}

func TestDecodeItemReadsKindAndMaterialRows(t *testing.T) {
	table := decodeFixture()
	for _, tc := range []struct {
		name  string
		code  data.ItemCode
		table *mapload.Table
		want  mapload.ItemDecoding
	}{
		{"weapon", 0x2101, table, mapload.ItemDecoding{Kind: "Two Handed Sword", KindKey: "twohandedsword", Material: "Meteoric", MaterialKey: "meteoric", Equipment: true}},
		{"shield", 0x1201, table, mapload.ItemDecoding{Kind: "Buckler", KindKey: "buckler", Material: "Iron", MaterialKey: "iron", Equipment: true}},
		{"armour", 0x0401, table, mapload.ItemDecoding{Kind: "Chain Helm", KindKey: "chainhelm", Material: "None", MaterialKey: "none", Equipment: true}},
		{"row spelt without a key", 0x0402, table, mapload.ItemDecoding{Kind: "Soft gauntlets", Material: "None", MaterialKey: "none", Equipment: true}},
		{"past the end of the class", 0x0109, table, mapload.ItemDecoding{Material: "None", MaterialKey: "none", Equipment: true}},
		{"material without a name", 0x3101, table, mapload.ItemDecoding{Kind: "Two Handed Sword", KindKey: "twohandedsword", Material: "#3", Equipment: true}},
		{"material past the end", 0x9101, table, mapload.ItemDecoding{Kind: "Two Handed Sword", KindKey: "twohandedsword", Material: "#9", Equipment: true}},
		{"potion", 0x0e08, table, mapload.ItemDecoding{Kind: "Potions", KindKey: "potions", Material: "None", MaterialKey: "none", Carried: 3}},
		{"scroll", 0x0e09, table, mapload.ItemDecoding{Kind: "Scrolls", KindKey: "scrolls", Material: "None", MaterialKey: "none", Carried: 4}},
		{"book", 0x0e0a, table, mapload.ItemDecoding{Kind: "Books", KindKey: "books", Material: "None", MaterialKey: "none", Carried: 5}},
		{"other carried", 0x0e0b, table, mapload.ItemDecoding{Kind: "Other items", KindKey: "otheritems", Material: "None", MaterialKey: "none"}},
		{"not an item class", 0x0f0f, table, mapload.ItemDecoding{Material: "None", MaterialKey: "none"}},
		{"no table", 0x0101, nil, mapload.ItemDecoding{Material: "None", MaterialKey: "none"}},
	} {
		if got := mapload.DecodeItem(tc.table, tc.code); got != tc.want {
			t.Errorf("%s: %#04x decoded %+v, want %+v", tc.name, uint16(tc.code), got, tc.want)
		}
	}
}

// itemGroupKeysToday are the kind and material row names that had a group
// word, with the key each had, before the decoder derived keys.
var itemGroupKeysToday = map[string]string{
	"kind:BareHands":          "barehands",
	"kind:Dagger":             "dagger",
	"kind:Short Sword":        "shortsword",
	"kind:Long Sword":         "longsword",
	"kind:Bastard Sword":      "bastardsword",
	"kind:Two Handed Sword":   "twohandedsword",
	"kind:Club":               "club",
	"kind:Spiked Club":        "spikedclub",
	"kind:Mace":               "mace",
	"kind:Morning Star":       "morningstar",
	"kind:Pick Hammer":        "pickhammer",
	"kind:War Hammer":         "warhammer",
	"kind:Staff":              "staff",
	"kind:Shaman Staff":       "shamanstaff",
	"kind:Pike":               "pike",
	"kind:Halberd":            "halberd",
	"kind:Lance":              "lance",
	"kind:Axe":                "axe",
	"kind:Two Handed Axe":     "twohandedaxe",
	"kind:Short Bow":          "shortbow",
	"kind:Long Bow":           "longbow",
	"kind:Crossbow":           "crossbow",
	"kind:Sonic Beam":         "sonicbeam",
	"kind:Flame Thrower":      "flamethrower",
	"kind:Boulder Thrower":    "boulderthrower",
	"kind:Plasma Sword":       "plasmasword",
	"kind:Ring":               "ring",
	"kind:Amulet":             "amulet",
	"kind:Hat":                "hat",
	"kind:Cap":                "cap",
	"kind:Low Hat":            "lowhat",
	"kind:Helm":               "helm",
	"kind:Soft Helm":          "softhelm",
	"kind:Chain Helm":         "chainhelm",
	"kind:Full Helm":          "fullhelm",
	"kind:Plate Helm":         "platehelm",
	"kind:Cloak":              "cloak",
	"kind:Cape":               "cape",
	"kind:Robe":               "robe",
	"kind:Dress":              "dress",
	"kind:Soft Mail":          "softmail",
	"kind:Chain Mail":         "chainmail",
	"kind:Scale Mail":         "scalemail",
	"kind:Cuirass":            "cuirass",
	"kind:Plate Cuirass":      "platecuirass",
	"kind:Bracers":            "bracers",
	"kind:Soft Bracers":       "softbracers",
	"kind:Plate Bracers":      "platebracers",
	"kind:Gloves":             "gloves",
	"kind:Soft Gauntlets":     "softgauntlets",
	"kind:Chain Gauntlets":    "chaingauntlets",
	"kind:Scale Gauntlets":    "scalegauntlets",
	"kind:Shoes":              "shoes",
	"kind:Soft Boots":         "softboots",
	"kind:Chain Boots":        "chainboots",
	"kind:Plate Boots":        "plateboots",
	"kind:Buckler":            "buckler",
	"kind:Small":              "small",
	"kind:Soft Small":         "softsmall",
	"kind:Wooden Small":       "woodensmall",
	"kind:Large":              "large",
	"kind:Soft Large":         "softlarge",
	"kind:Wooden Large":       "woodenlarge",
	"kind:Tower":              "tower",
	"kind:Wooden Tower":       "woodentower",
	"kind:Potions":            "potions",
	"kind:Scrolls":            "scrolls",
	"kind:Books":              "books",
	"kind:Other items":        "otheritems",
	"material:Iron":           "iron",
	"material:Bronze":         "bronze",
	"material:Steel":          "steel",
	"material:Silver":         "silver",
	"material:Gold":           "gold",
	"material:Mithrill":       "mithrill",
	"material:Adamantium":     "adamantium",
	"material:Meteoric":       "meteoric",
	"material:Wood":           "wood",
	"material:Magic Wood":     "magicwood",
	"material:Leather":        "leather",
	"material:Hard Leather":   "hardleather",
	"material:Dragon Leather": "dragonleather",
	"material:Bone":           "bone",
	"material:Crystal":        "crystal",
	"material:None":           "none",
}

func TestItemRowKeysResolveAsTheyDidForTheInstalls(t *testing.T) {
	found := 0
	for _, name := range []string{"en", "ru", "rom2-en", "rom2-ru"} {
		root := seatPath("gameversions/" + name)
		if root == "" {
			continue
		}
		found++
		a, err := OpenArchives(root)
		if err != nil {
			t.Fatal(err)
		}
		defs, err := LoadDefinitionsFor(a.Containers, a.Game())
		if err != nil {
			t.Fatal(err)
		}
		book := words.For(a.Base.Profile.Language)
		caption := func(axis, rowName, key string) string {
			if key != "" {
				if s := book.Text("editor.item." + axis + "." + key); s != "editor.item."+axis+"."+key {
					return s
				}
			}
			return rowName
		}
		today := func(axis, rowName string) string {
			if key, ok := itemGroupKeysToday[axis+":"+rowName]; ok {
				return book.Text("editor.item." + axis + "." + key)
			}
			return rowName
		}
		for c := 1; c <= 65535; c++ {
			d := mapload.DecodeItem(defs.Table, data.ItemCode(c))
			if d.Kind == "" {
				continue
			}
			if got, want := caption("kind", d.Kind, d.KindKey), today("kind", d.Kind); got != want {
				t.Fatalf("%s: kind row %q resolves to %q, today %q", name, d.Kind, got, want)
			}
			if got, want := caption("material", d.Material, d.MaterialKey), today("material", d.Material); got != want {
				t.Fatalf("%s: material row %q resolves to %q, today %q", name, d.Material, got, want)
			}
		}
	}
	if found == 0 {
		t.Skip("no install root beside the checkout")
	}
}

// TestEditorItemWordsAreExactlyTheKeysTheDecoderProduces holds the English ids
// under editor.item.kind. and editor.item.material., which itemGroupWord forms
// at run time, to the keys of itemGroupKeysToday: every key has an id and every
// id has a key. The carried and no-material keys are in the map too.
func TestEditorItemWordsAreExactlyTheKeysTheDecoderProduces(t *testing.T) {
	ids, err := words.IDs(words.Engine())
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, id := range ids {
		if strings.HasPrefix(id, "editor.item.kind.") || strings.HasPrefix(id, "editor.item.material.") {
			have[id] = true
		}
	}
	want := map[string]bool{}
	for k, key := range itemGroupKeysToday {
		axis, _, _ := strings.Cut(k, ":")
		want["editor.item."+axis+"."+key] = true
	}
	for id := range have {
		if !want[id] {
			t.Errorf("English id %q is formed from no decoder key", id)
		}
	}
	for id := range want {
		if !have[id] {
			t.Errorf("decoder key forms id %q, which the English table lacks", id)
		}
	}
}
