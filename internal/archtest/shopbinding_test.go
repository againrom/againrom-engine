package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckShopBindingNamesEachViolation drives the scan with synthetic
// sources, one case per arm.
func TestCheckShopBindingNamesEachViolation(t *testing.T) {
	const binder = "func (t *townScreen) bindShop(row shopBinding) {}\n"
	cases := []struct {
		name, src, want string
	}{
		{"a second binder", "package game\n" + binder + "func (t *townScreen) bindBankShop() {}\n", "bindBankShop is a second shop page binder"},
		{"a second binder for the other game", "package game\n" + binder + "func (t *townScreen) bindChapterShop() {}\n", "bindChapterShop is a second shop page binder"},
		{"an item hook on the controller", "package game\n" + binder + "type shopController struct{ described func(sim.ItemInstance) sim.ItemInstance }\n", "described is a func(sim.ItemInstance) sim.ItemInstance field"},
		{"a settle test", "package game\n" + binder + "func f(t *shopController) { if t.settle != nil { t.settle() } }\n", "settle is tested against nil"},
		{"a described test", "package game\n" + binder + "func f(t *shopController) { if t.described != nil { _ = 1 } }\n", "described is tested against nil"},
		{"no binder", "package game\n", "found 0 townScreen.bindShop functions"},
		{"the one binder", "package game\n" + binder + "type shopController struct {\n\trow   shopBinding\n\tlabel func(sim.ItemInstance) []string\n}\nfunc f(t *shopController) { if t.bag != nil { _ = 1 } }\n", ""},
		{"a binder of another screen", "package game\n" + binder + "func (t *otherScreen) bindMiscShop() {}\nfunc (t *townScreen) bindMisc() {}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckShopBinding(map[string]string{"pkg/game/a.go": tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckShopBinding = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckShopBinding = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckShopBinding(nil); len(vs) != 1 {
		t.Fatalf("CheckShopBinding(nil) = %v, want one violation", vs)
	}
}

// TestShopPageHasOneBinder is the live-tree half.
func TestShopPageHasOneBinder(t *testing.T) {
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
	for _, v := range CheckShopBinding(files) {
		t.Errorf("%s", v)
	}
}
