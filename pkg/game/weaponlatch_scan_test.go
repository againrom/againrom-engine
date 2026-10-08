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

// Keep the documented latch readers exact across every package that imports mapload.
// Only materializeStartingWeapon may assign it, including composite literals.

// weaponLatchWriter is the ONE function permitted to assign the latch, keyed
// "pkg.Func" the same way weaponLatchSites is below.
const weaponLatchWriter = "game.materializeStartingWeapon"

// Parse every production package that imports mapload.
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

// The latch reader set is exact; remove a site when its read disappears.
var weaponLatchSites = map[string]string{
	"game.capturePartyPolicy":             "reads the current latch into the SAV presence policy",
	"game.capturePartyWeapon":             "reads it to retain only an unused starting-weapon fallback",
	"game.restoreFromState":               "reads it when deriving the weapon view from restored current equipment",
	"game.projectCurrentPartyActorFields": "reads the captured latch when deriving native Body from current worn items",
	"mapload.MemberWeapon":                "reads it when selecting current equipment or the unused starting fallback",
	"game.materializeStartingWeapon":      "raises it; the only assignment across both scanned packages",
	"game.resolveWeaponMaterialized":      "reads it, and raises it through the writer when slot 1 is occupied for real",
	"game.shopWeaponFallbackCode":         "reads it: a raised latch means the shop offers no fallback code",
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
