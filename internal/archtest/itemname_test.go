package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckItemNameNamesEachShape(t *testing.T) {
	const game = "pkg/game/a.go"
	cases := []struct {
		name, file, src, want string
	}{
		{"a shipped lookup", game,
			"package game\nfunc f(t *T) string { n, _ := mapload.ShippedItemName(t, 1); return n }\n", itemNameShippedLookup},
		{"a NameFor", "pkg/mapload/a.go",
			"package mapload\nfunc f(t *Table) string { n, _ := t.Names.NameFor(1); return n }\n", itemNameShippedLookup},
		{"a row name", "pkg/ui/a.go",
			"package ui\nfunc f(t *T) string { return t.Weapons.EntryName(1) }\n", itemNameRowName},
		{"a row name method value", game,
			"package game\nfunc f(c C) string { return g(c.EntryName) }\n", itemNameRowNameValue},
		{"a code's number", game,
			"package game\nfunc f(code data.ItemCode) string { return code.Name() }\n", itemNameCodeNumber},
		{"a converted code's number", game,
			"package game\nfunc f(n uint16) string { return data.ItemCode(n).Name() }\n", itemNameCodeNumber},
		{"the shipped table read", game,
			"package game\nfunc f(t *T) int { return len(t.Names) }\n", itemNameNamesTable},
		{"a second name table", game,
			"package game\nfunc f() data.ItemNames { return data.ItemNames{1: \"x\"} }\n", itemNameNamesTable},
		{"a weapon's name", game,
			"package game\nfunc f(t *T) string { w, _ := data.WeaponFromCode(1, t.Shapes, t.Materials, t.Weapons); return w.Name }\n", itemNameWeaponName},
		{"a code label", game,
			"package game\nfunc f(c uint16) string { return fmt.Sprintf(\"item %#04x\", c) }\n", itemNameCodeLabel},
		{"an empty-name fallback", game,
			"package game\nfunc f(t *T) string { n := itemName(1, t); if n == \"\" { n = \"?\" }; return n }\n", itemNameEmptyFallback},
		{"a fallback on the lookup's ok", game,
			"package game\nfunc f(t *T) string { n, ok := mapload.ShippedItemName(t, 1); if !ok { n = \"?\" }; return n }\n", itemNameEmptyFallback},
		{"a fallback on a resolved text", game,
			"package game\nfunc f(t *T) string { n := mapload.ResolveItemName(t, s).Text(); if n == \"\" { n = \"?\" }; return n }\n", itemNameEmptyFallback},
		{"a built saved name", game,
			"package game\nfunc f(t *T) string { return mapload.ItemDisplayName(t, mapload.ItemSubject{Saved: fmt.Sprint(1)}) }\n", itemNameSavedBuilt},
		{"another composer row", game,
			"package game\nvar g = gameFiles{itemNamer: myNamer}\n", itemNameComposerRow},
		{"a composer set on a table", game,
			"package game\nfunc f(t *mapload.Table) { t.ItemNamer = nil }\n", itemNameComposerRow},
		{"the resolver itself", itemNameResolverFile,
			"package mapload\nfunc f(t *Table) string { n, _ := t.Names.NameFor(1); return n }\n", ""},
		{"a resolver call", game,
			"package game\nfunc f(t *T) string { return mapload.ItemDisplayName(t, mapload.ItemSubject{Code: 1, Saved: saved}) }\n", ""},
		{"the two rows", game,
			"package game\nvar g = gameFiles{itemNamer: mapload.CodeItemNamer}\nvar h = gameFiles{itemNamer: mapload.RowItemNamer}\n", ""},
		{"a spell's row name", game,
			"package game\nfunc f(t *T) string { return t.Spells.EntryName(1) }\n", ""},
		{"a sack index label", game,
			"package game\nfunc f(n int, s string) string { return fmt.Sprintf(\"Item %d: %s\", n, s) }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc f(code data.ItemCode) string { return code.Name() }\n", ""},
		{"another package", "pkg/data/a.go",
			"package data\nfunc f(code ItemCode) string { return code.Name() }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckItemName(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckItemName = %v, want none", got)
				}
				return
			}
			for _, g := range got {
				if strings.HasPrefix(g, tc.want+" ") {
					return
				}
			}
			t.Fatalf("CheckItemName = %v, want one naming %q", got, tc.want)
		})
	}
}

func TestCheckItemNameAllowsTheExistenceChecks(t *testing.T) {
	src := "package game\nfunc (x *MapInspector) checkItem(code int64) error { if _, ok := mapload.ShippedItemName(x.table, 1); !ok { return nil }; return nil }\n"
	if got := CheckItemName(map[string]string{"pkg/game/mapinspectionedit.go": src}); len(got) != 0 {
		t.Fatalf("the existence check was flagged: %v", got)
	}
	if got := CheckItemName(map[string]string{"pkg/game/other.go": src}); len(got) == 0 {
		t.Fatal("the same check in another file was not flagged")
	}
}

func TestItemNameHasOneResolver(t *testing.T) {
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
	if !strings.Contains(files[itemNameResolverFile], "func ResolveItemName(") {
		t.Fatalf("%s holds no ResolveItemName; the scan guards nothing", itemNameResolverFile)
	}
	vs, met := scanItemName(files)
	for _, v := range vs {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
	for key := range itemNameAllowed {
		if !met[key] {
			t.Errorf("allowed site %q holds no such shape; drop it", key)
		}
	}
}
