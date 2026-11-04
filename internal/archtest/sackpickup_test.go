package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckSackPickupNamesEachShape drives the scan with synthetic sources,
// one planted instance per shape, beside the clean forms and named entries.
func TestCheckSackPickupNamesEachShape(t *testing.T) {
	const take = "package sim\nfunc (w *World) pickUpSack() { w.TakeSack(); w.MoveCarried(); w.CompleteSackPickup() }\n" +
		"func (w *World) SettleSackPickup() { w.sackPickupArrived(0); w.pickUpSack(); w.CompleteSackPickup(); w.cancelSackPickup() }\n" +
		"func (w *World) CheatPickupAll() { w.cheatPickupEach(); w.pickUpSack() }\n"
	cases := []struct {
		name, path, src, want string
	}{
		{"a take outside sim", "pkg/game/a.go", "package game\nfunc f(w *sim.World) { w.TakeSack(1, 2, 3) }\n", "f calls TakeSack"},
		{"a documents move outside sim", "pkg/game/a.go", "package game\nfunc f(w *sim.World) { w.MoveCarried(1, 2, 3, 4) }\n", "f calls MoveCarried"},
		{"a completion outside sim", "cmd/tool/main.go", "package main\nfunc f(w *sim.World) { w.CompleteSackPickup(1) }\n", "f calls CompleteSackPickup"},
		{"an arrival call outside sim", "pkg/game/a.go", "package game\nfunc f(w *sim.World) bool { return w.SackPickupArrived(1) }\n", "f calls SackPickupArrived"},
		{"a second take", "pkg/game/a.go", "package game\nfunc (mw *mapWorld) takeSackFor() {}\n", "takeSackFor is a second sack take"},
		{"a rebuilt pickup order", "pkg/game/a.go", "package game\nfunc f() { _ = sim.PickUp(1, sim.CellPoint{}) }\n", "f builds the pickup command"},
		{"a sack list write", "pkg/game/a.go", "package game\nfunc f(w *sim.World) { w.ReplaceGroundSacks(nil) }\n", "f writes the World's sack list"},
		{"the ground import", "pkg/game/originalground.go", "package game\nfunc applyOriginalGround(w *sim.World) { w.ReplaceGroundSacks(nil) }\n", ""},
		{"an arrival by hand", "pkg/game/a.go", "package game\nfunc f(w *sim.World, e sim.Entity) bool { x, _, _ := w.ActorFinePosition(1); return e.Transit == 0 && x == 128 }\n", "f tests arrival by hand"},
		{"the tool's walk stop", "cmd/missionrun/main.go", "package main\nfunc pickUpAt(w *sim.World, e sim.Entity) bool { x, _, _ := w.ActorFinePosition(1); return e.Transit == 0 && x == 128 }\n", ""},
		{"a fine position drawn", "pkg/game/a.go", "package game\nfunc f(w *sim.World, e sim.Entity) { x, _, _ := w.ActorFinePosition(1); x = 128; _ = e.Transit }\n", ""},
		{"a starting hero loop", "pkg/game/a.go", "package game\nfunc f(party []mapload.PartyMember) { for _, p := range party { if primaryPlayerHero(p) { return } } }\n", "f looks the starting hero up in a loop"},
		{"the starting hero lookup", "pkg/game/a.go", "package game\nfunc startingHeroIndex(party []mapload.PartyMember) { for _, p := range party { if primaryPlayerHero(p) { return } } }\nfunc g(p mapload.PartyMember) bool { return primaryPlayerHero(p) }\n", ""},
		{"a documents holder named", "pkg/game/a.go", "package game\nfunc f() { _ = sim.SackDocuments{Holder: 1, Held: true} }\n", "f names the quest documents' holder"},
		{"the documents lookup", "pkg/game/a.go", "package game\nfunc questDocuments() sim.SackDocuments { return sim.SackDocuments{} }\n", ""},
		{"a take elsewhere in sim", "pkg/sim/b.go", "package sim\nfunc (w *World) g() { w.TakeSack(1, 2, 3) }\n", "g calls TakeSack; only the sack pickup transaction"},
		{"a settle in another function", "pkg/sim/sackpickup.go", take + "func (w *World) g() { w.cancelSackPickup(1) }\n", "g calls cancelSackPickup"},
		{"a second take in sim", "pkg/sim/sackpickup.go", take + "func (w *World) h() { w.pickUpSack() }\n", "h calls pickUpSack"},
		{"no take", "pkg/sim/sackpickup.go", "package sim\n", "found 0 pickUpSack functions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"pkg/sim/sackpickup.go": take}
			files[tc.path] = tc.src
			var got []string
			for _, v := range CheckSackPickup(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckSackPickup = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckSackPickup = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckSackPickup(nil); len(vs) != 1 {
		t.Fatalf("CheckSackPickup(nil) = %v, want one violation", vs)
	}
}

// TestSackPickupHasOneBuilder is the live-tree half.
func TestSackPickupHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadSackPickupSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[sackPickupFile]; !ok {
		t.Fatalf("LoadSackPickupSources found no %s among %d files", sackPickupFile, len(files))
	}
	for _, v := range CheckSackPickup(files) {
		t.Error(v)
	}
}
