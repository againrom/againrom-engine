package mod

import (
	"strings"
	"testing"
)

func TestItemsWriterKeepsTheBytesOutsideTheFieldsItChanges(t *testing.T) {
	src := "# the mod's items\n" +
		"[[item]]\n" +
		"key   = \"bone_mail\"   # a body armour\n" +
		"name  = \"item.bone_mail\"\n" +
		"slot  = \"body\"\n" +
		"defence = 5\n" +
		"price = 40\n" +
		"stand-in = \"Soft Mail\"\n" +
		"\n" +
		"# a shipped row\n" +
		"[[change]]\n" +
		"item  = \"Cap\"\n" +
		"price = 7\n" +
		"\n" +
		"[[material]]\n" +
		"item = \"Soft Mail\"\n" +
		"tier = \"Common\"\n" +
		"material = \"Bone\"\n" +
		"allow = true\n"
	text := func(string) (string, bool) { return "Bone", true }
	d, err := ParseItems("m", ItemsFile, []byte(src), text, "en")
	if err != nil {
		t.Fatal(err)
	}
	kind, ok := WrittenKind(ItemsFile)
	if !ok {
		t.Fatal("the items kind has no writer")
	}
	same, err := kind.Write(ItemsFile, []byte(src), Data{Items: d})
	if err != nil || string(same) != src {
		t.Fatalf("an unchanged set rewrote the file (%v):\n%s", err, same)
	}
	price, off := int32(55), true
	d.Rows[0].Price = price
	d.Changes[0].Off = &off
	d.Materials = nil
	d.Weapons = append(d.Weapons, WeaponRow{Key: "star", NameKey: "item.star", Base: "Dagger", Params: []WeaponParamValue{{WeaponParam: WeaponParams[0], Value: 9}}})
	out, err := kind.Write(ItemsFile, []byte(src), Data{Items: d})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(src, "price = 40\n", "price = 55\n", 1)
	// A pair a kept record gains follows the blank lines after it, as the
	// one row writer appends it.
	want = strings.Replace(want, "price = 7\n\n", "price = 7\n\noff = true\n", 1)
	want = strings.Replace(want, "[[material]]\nitem = \"Soft Mail\"\ntier = \"Common\"\nmaterial = \"Bone\"\nallow = true\n",
		"[[weapon]]\nkey = \"star\"\nname = \"item.star\"\nbase = \"Dagger\"\nprice = 9\n", 1)
	if string(out) != want {
		t.Fatalf("written:\n%s\nwant:\n%s", out, want)
	}
	back, err := ParseItems("m", ItemsFile, out, text, "en")
	if err != nil || back.Changes[0].Off == nil || !*back.Changes[0].Off || len(back.Weapons) != 1 || back.Rows[0].Price != 55 {
		t.Fatalf("read back %+v, %v", back, err)
	}
}

func TestItemChangeOffIsReadByTheItemsRule(t *testing.T) {
	text := func(string) (string, bool) { return "x", true }
	d, err := ParseItems("m", ItemsFile, []byte("[[change]]\nitem = \"Cap\"\noff = true\n"), text, "en")
	if err != nil || d.Changes[0].Off == nil || !*d.Changes[0].Off {
		t.Fatalf("off = true: %+v, %v", d, err)
	}
	_, err = ParseItems("m", ItemsFile, []byte("[[change]]\nitem = \"Cap\"\noff = 1\n"), text, "en")
	if err == nil || !strings.Contains(err.Error(), "off must be true or false") {
		t.Fatalf("off = 1: %v", err)
	}
}

func TestItemsWriterDeleteOfTheFirstRepeatKeepsTheOthersLines(t *testing.T) {
	source := "# head\n[[change]]\nitem = \"Cap\" # A first\nprice = 7 # A price\n\n[[change]]\nitem = \"Cap\" # B second\nprice = 9 # B price\n"
	d, err := ParseItems("m", ItemsFile, []byte(source), func(string) (string, bool) { return "-", true }, "en")
	if err != nil || len(d.Changes) != 2 {
		t.Fatalf("parse: %+v, %v", d, err)
	}
	d.Changes = d.Changes[1:]
	out, err := writeItems("", []byte(source), Data{Items: d})
	want := "# head\n\n[[change]]\nitem = \"Cap\" # B second\nprice = 9 # B price\n"
	if err != nil || string(out) != want {
		t.Fatalf("delete first: %q, %v; want %q", out, err, want)
	}
}
