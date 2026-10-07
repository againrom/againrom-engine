package game

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// THE MECHANISM BEHIND closure.md's WRITE/READ TABLE (1005 round 2, seventh
// pass). PartyMember.WeaponMaterialized has been wrong in four separate
// places across four adversarial passes, each time because a NEW site
// touched it and the reasoning written down for the old sites did not reach
// the new one. This test parses package game's AND package mapload's own
// source — widened from package game alone (round-2's seventh pass, C3):
// the fix for that pass added the field's first reader outside pkg/game,
// mapload.PartyLoadout (pkg/mapload/loadout.go), and a scan confined to one
// package cannot see a site in the other. It fails when the latch is
// assigned anywhere but materializeStartingWeapon, or mentioned in any
// function the table below does not name.
//
// TWO BLIND SPOTS FOUND IT STILL PASSING (round-2, ninth pass) and were
// closed here rather than argued away. FIRST: the walk matched only
// *ast.SelectorExpr, so a composite-literal key —
// mapload.PartyMember{WeaponMaterialized: true} — read and wrote the field
// through an *ast.KeyValueExpr the walk never inspected; adding func
// probeEvadeScan() mapload.PartyMember { return
// mapload.PartyMember{WeaponMaterialized: true} } to this very file left the
// test green. The walk below now inspects composite-literal keys too, keyed
// separately as "pkg.Func!lit" in weaponLatchSites because a literal is a
// WRITE (it sets the field's value outright, same footing as an assignment)
// and not a read like every other entry, and the ONE-WRITER rule is enforced
// against it exactly as it is against an *ast.AssignStmt. SECOND: the scan
// covered pkg/game and pkg/mapload alone, so a reader in any of the eleven
// other directories that import pkg/mapload — the cmd/* tools this repo
// ships, every one a package main — was invisible to it. weaponLatchDirs
// now lists every directory under cmd/ that imports pkg/mapload in non-test
// code, found by `grep -rl 'againrom/pkg/mapload"' --include=*.go . | grep
// -v _test.go`, alongside game and mapload themselves; a twelfth cmd/* tool
// starting to import mapload needs a row added here for this test to see it,
// the same obligation the write/read table already carries.
//
// WHEN IT FAILS, the fix is not to widen the table quietly. Add the row to
// docs/1005-interactive-doll/closure.md's write/read enumeration WITH the
// file:line and what the site does, then name the function here as
// "pkg.Func". The table and the document are meant to move together, which is
// the whole reason this is a test rather than a comment.

// weaponLatchWriter is the ONE function permitted to assign the latch, keyed
// "pkg.Func" the same way weaponLatchSites is below.
const weaponLatchWriter = "game.materializeStartingWeapon"

// weaponLatchDirs are the source directories this test parses, as a path
// relative to this package's own directory (pkg/game — go test's working
// directory for this package), each paired with the package name
// parser.ParseDir returns for it. The cmd/* entries are every directory in
// this repo that imports againrom/pkg/mapload in non-test code besides
// pkg/game itself, found with (from the repo root):
//
//	grep -rl 'againrom/pkg/mapload"' --include=*.go . | grep -v _test.go | xargs -n1 dirname | sort -u
//
// A directory that starts importing mapload needs a row added here, on the
// same obligation weaponLatchSites already carries for a function.
var weaponLatchDirs = map[string]string{
	".":                          "game",
	"../mapload":                 "mapload",
	"../../cmd/almtool":          "main",
	"../../cmd/appearcheck":      "main",
	"../../cmd/areaoverlaycheck": "main",
	"../../cmd/classdump":        "main",
	"../../cmd/effectmarkcheck":  "main",
	"../../cmd/mapview":          "main",
	"../../cmd/missionrun":       "main",
	"../../cmd/paneldump":        "main",
	"../../cmd/presenceprobe":    "main",
	"../../cmd/spelleffectcheck": "main",
	"../../cmd/weaponspellcheck": "main",
	"../../cmd/wearcheck":        "main",
}

// weaponLatchSites is every function, across both scanned packages, whose
// body may mention PartyMember.WeaponMaterialized at all, with what it does
// there. Keyed "pkg.Func" so a name shared by two packages can never collide.
// The set is exact: a function that stops touching the latch must be
// removed, exactly as one that starts touching it must be added, so the list
// cannot quietly drift into a superset of the truth.
var weaponLatchSites = map[string]string{
	"game.capturePartyPolicy":        "reads the current latch into the SAV presence policy",
	"game.capturePartyWeapon":        "reads it to retain only an unused starting-weapon fallback",
	"game.restoreFromState":          "reads it when deriving the weapon view from restored current equipment",
	"mapload.MemberWeapon":           "reads it when selecting current equipment or the unused starting fallback",
	"game.materializeStartingWeapon": "raises it; the only assignment across both scanned packages",
	"game.resolveWeaponMaterialized": "reads it, and raises it through the writer when slot 1 is occupied for real",
	"game.shopWeaponFallbackCode":    "reads it: a raised latch means the shop offers no fallback code",
	"game.missionDollEquipment": "reads it: a raised latch means the opening doll composes slot 1 from the array alone " +
		"(split out of buildInventorySubject, round-2 twelfth pass, 2026-08-17, C1b, so world.go's own tracker seed " +
		"can call the same composition instead of a second one)",
	"game.canonicalizePartyAppearance": "reads it: an unraised latch means the member is DRAWN holding his " +
		"starting weapon, currentFigureEquipment's own slot-1 widening restated for an arbitrary party member " +
		"(the map-sprite appearance hotfix, 2026-08-22)",
	"game.spellClientClass": "reads it: include an unmaterialized starting weapon when resolving the live " +
		"client class for Cast capability, without requiring loaded art (story1119)",
	"mapload.PartyLoadout": "reads it as ResolveEquipmentLoadout's own everEquipped: a raised latch means " +
		"mission construction treats an empty slot 1 as taken off, not as not-yet-materialized (C3, round-2 " +
		"seventh pass, 2026-08-17)",
	"game.nativeCityHiredEquipmentMismatch": "reads it: a hired mercenary's own latch must match the fresh " +
		"tavern template restoreHiredMercenaries would rebuild him from, or the native SAV refuses rather than " +
		"silently reverting it on the next load (story 1127 correction, R-2)",
}

func TestWeaponMaterializedHasOneWriter(t *testing.T) {
	fset := token.NewFileSet()
	seen := map[string]bool{}
	noTestFiles := func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }
	for dir, wantPkg := range weaponLatchDirs {
		pkgs, err := parser.ParseDir(fset, dir, noTestFiles, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", dir, err)
		}
		pkg, ok := pkgs[wantPkg]
		if !ok {
			t.Fatalf("package %s did not parse from %s", wantPkg, dir)
		}
		// pkgLabel disambiguates the eleven cmd/* directories this test
		// scans, every one of them package main: "main.main" would collide
		// across all of them, so a package-main directory is keyed by its
		// own base name instead of the package name go/parser reports.
		pkgLabel := wantPkg
		if wantPkg == "main" {
			pkgLabel = filepath.Base(dir)
		}
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				name := pkgLabel + "." + fn.Name.Name
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					switch node := n.(type) {
					case *ast.SelectorExpr:
						if node.Sel.Name == "WeaponMaterialized" {
							seen[name] = true
							if _, allowed := weaponLatchSites[name]; !allowed {
								t.Errorf("%s mentions PartyMember.WeaponMaterialized at %s and is not in weaponLatchSites.\n"+
									"Add the site to docs/1005-interactive-doll/closure.md's write/read table with its file:line, then name it here.",
									name, fset.Position(node.Pos()))
							}
						}
					case *ast.CompositeLit:
						for _, elt := range node.Elts {
							kv, ok := elt.(*ast.KeyValueExpr)
							if !ok {
								continue
							}
							key, ok := kv.Key.(*ast.Ident)
							if !ok || key.Name != "WeaponMaterialized" {
								continue
							}
							if name != weaponLatchWriter {
								t.Errorf("%s SETS PartyMember.WeaponMaterialized in a composite literal at %s.\n"+
									"The latch has exactly one writer, %s; a composite-literal key is a write, "+
									"the same as an assignment.", name, fset.Position(key.Pos()), weaponLatchWriter)
							}
						}
					case *ast.AssignStmt:
						for _, lhs := range node.Lhs {
							sel, ok := lhs.(*ast.SelectorExpr)
							if !ok || sel.Sel.Name != "WeaponMaterialized" {
								continue
							}
							if name != weaponLatchWriter {
								t.Errorf("%s ASSIGNS PartyMember.WeaponMaterialized at %s.\n"+
									"The latch has exactly one writer, %s, because four adversarial passes were spent on sites that each set it their own way.",
									name, fset.Position(sel.Pos()), weaponLatchWriter)
							}
						}
					}
					return true
				})
			}
		}
	}
	var stale []string
	for name := range weaponLatchSites {
		if !seen[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("weaponLatchSites names %v, which no longer touch the latch — delete the rows so the table stays the whole truth", stale)
	}
}
