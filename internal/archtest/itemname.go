package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The item display name has one builder: mapload.ResolveItemName in
// itemNameResolverFile, whose fallback is the game's composer row. Outside it,
// production pkg/game, pkg/ui and pkg/mapload build no item name.
const itemNameResolverFile = "pkg/mapload/itemname.go"

// The shapes of an item name built outside the resolver.
const (
	itemNameShippedLookup = "shipped-lookup" // ShippedItemName or NameFor
	itemNameRowName       = "row-name"       // EntryName on an item table
	itemNameRowNameValue  = "row-name-value" // EntryName as a method value
	itemNameCodeNumber    = "code-number"    // Name() of an item code
	itemNameNamesTable    = "names-table"    // the shipped table read or a second one made
	itemNameWeaponName    = "weapon-name"    // Name of a code's weapon, armour or shield
	itemNameCodeLabel     = "code-label"     // an item code formatted in hex
	itemNameEmptyFallback = "empty-fallback" // a resolved name replaced by a test on it
	itemNameSavedBuilt    = "saved-built"    // a built value passed as the saved name
	itemNameComposerRow   = "composer-row"   // a composer other than the two rows
)

// itemNameAllowed are the sites that read item rows or the shipped table for
// something other than a shown name, keyed by file, function and shape.
var itemNameAllowed = map[string]string{
	"pkg/game/mapinspectionedit.go (*MapInspector).checkItem " + itemNameShippedLookup:            "existence check: refuses a code the shipped table does not name (DIV-3134)",
	"pkg/game/mapinspectionequipment.go (*mapInspectionEdits).setEquipment " + itemNameNamesTable: "existence check: refuses a worn variant the shipped table does not name",
	"pkg/game/table.go LoadDefinitionsFor " + itemNameNamesTable:                                  "the loader stores the shipped table",
	"pkg/game/itemnames.go readItemNamesAt " + itemNameNamesTable:                                 "the loader builds the shipped table",
	"pkg/game/moditemlayer.go (*FrontEnd).setModItems " + itemNameNamesTable:                      "the mod layer adds its items' names to the table the resolver reads",
	"pkg/game/mapeditormodcontrols.go (*mapInspectionEdits).partyItemName " + itemNameWeaponName:  "a character cell's data name, matched against the cell text",
	"pkg/mapload/cheatfactory.go CheatItem " + itemNameRowName:                                    "name-to-code lookup of a cheat",
	"pkg/mapload/itemeffects.go ItemInstanceFromCode " + itemNameRowName:                          "the row name gives the item's kind, not its shown name",
	"pkg/mapload/structureuse.go StructureUseMetadata " + itemNameRowName:                         "name-to-row lookup",
	"pkg/game/shop.go shopSchoolTemplateCode " + itemNameRowName:                                  "name-to-code lookup of a school template",
	"pkg/game/shopconsumables.go shopPotionStock " + itemNameRowName:                              "name-to-row lookup",
	"pkg/game/moditems.go resolveOriginalItem " + itemNameRowName:                                 "parses a mod cell's class and material words",
	"pkg/game/modbodies.go modWeaponRow " + itemNameRowName:                                       "name-to-row lookup of a mod body's weapon",
	"pkg/game/moditemlayer.go itemMaterialPatch " + itemNameRowName:                               "a mod patch field's key: the class and material words",
	"pkg/game/moditemlayer.go checkWeaponDamage " + itemNameRowName:                               "a mod diagnostic names the class and material rows, not an item",
	"pkg/game/moditemlayer.go pairPicturesOf " + itemNameRowName:                                  "a mod diagnostic names the class and material rows, not an item",
	"pkg/game/table.go readUnder " + itemNameComposerRow:                                          "installs the game's composer row",
}

var itemTableNames = map[string]bool{
	"MagicItems": true, "Weapons": true, "Armors": true, "Shields": true, "Materials": true, "Shapes": true,
	"magicItems": true, "weapons": true, "armors": true, "shields": true, "materials": true, "shapes": true,
}

var itemCodeLabel = regexp.MustCompile(`(?i)item ?%[-+# 0-9]*[xX]`)

// CheckItemName reports each item name built outside the resolver.
func CheckItemName(files map[string]string) []Violation {
	vs, _ := scanItemName(files)
	return vs
}

// scanItemName is CheckItemName's scan, with the allowed sites it met.
func scanItemName(files map[string]string) ([]Violation, map[string]bool) {
	met := map[string]bool{}
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the item-name scan found nothing to read"}}, met
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if (strings.HasPrefix(name, "pkg/game/") || strings.HasPrefix(name, "pkg/ui/") || strings.HasPrefix(name, "pkg/mapload/")) &&
			!strings.HasSuffix(name, "_test.go") && name != itemNameResolverFile {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, decl := range f.Decls {
			fn := ""
			if d, ok := decl.(*ast.FuncDecl); ok {
				fn = itemNameFuncName(d)
			}
			for _, h := range itemNameHits(decl) {
				if key := name + " " + fn + " " + h.shape; itemNameAllowed[key] != "" {
					met[key] = true
					continue
				}
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(h.pos).Line),
					Reason: h.shape + " in " + fn + ": " + h.what + "; name the item through mapload.ResolveItemName",
				})
			}
		}
	}
	return vs, met
}

type itemNameHit struct {
	pos         token.Pos
	shape, what string
}

func itemNameFuncName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	t := d.Recv.List[0].Type
	star := ""
	if s, ok := t.(*ast.StarExpr); ok {
		t, star = s.X, "*"
	}
	if g, ok := t.(*ast.IndexExpr); ok {
		t = g.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return "(" + star + id.Name + ")." + d.Name.Name
	}
	return d.Name.Name
}

func itemNameLast(x ast.Expr) string {
	switch e := x.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.ParenExpr:
		return itemNameLast(e.X)
	case *ast.StarExpr:
		return itemNameLast(e.X)
	}
	return ""
}

func itemNameCallName(c *ast.CallExpr) string { return itemNameLast(c.Fun) }

// itemNameHits are the shapes one declaration holds.
func itemNameHits(decl ast.Decl) []itemNameHit {
	var hits []itemNameHit
	hit := func(n ast.Node, shape, what string) { hits = append(hits, itemNameHit{n.Pos(), shape, what}) }
	called := map[ast.Expr]bool{}
	ast.Inspect(decl, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			called[c.Fun] = true
		}
		return true
	})
	// fromCode holds the variables a weapon, armour or shield is resolved
	// into; resolved those an item name is; flags the ok of a shipped lookup.
	fromCode, resolved := map[string]bool{}, map[string]bool{}
	ast.Inspect(decl, func(n ast.Node) bool {
		a, ok := n.(*ast.AssignStmt)
		if !ok || len(a.Rhs) != 1 {
			return true
		}
		into := func(set map[string]bool) {
			for _, l := range a.Lhs {
				if id, ok := l.(*ast.Ident); ok && id.Name != "_" {
					set[id.Name] = true
				}
			}
		}
		for x := a.Rhs[0]; x != nil; {
			switch e := x.(type) {
			case *ast.CallExpr:
				switch itemNameCallName(e) {
				case "WeaponFromCode", "ArmorFromCode", "ShieldFromCode":
					into(fromCode)
				case "ItemDisplayName", "ResolveItemName", "ShippedItemName", "itemName", "shopItemName":
					into(resolved)
				}
				x = e.Fun
			case *ast.SelectorExpr:
				x = e.X
			default:
				x = nil
			}
		}
		return true
	})
	ast.Inspect(decl, func(n ast.Node) bool {
		switch e := n.(type) {
		case *ast.CallExpr:
			name := itemNameCallName(e)
			switch {
			case name == "ShippedItemName" || name == "NameFor":
				hit(e, itemNameShippedLookup, name+" reads the shipped table")
			case name == "EntryName":
				if sel, ok := e.Fun.(*ast.SelectorExpr); ok && itemTableNames[itemNameLast(sel.X)] {
					hit(e, itemNameRowName, itemNameLast(sel.X)+".EntryName names an item row")
				}
			case name == "Name" && len(e.Args) == 0:
				if sel, ok := e.Fun.(*ast.SelectorExpr); ok && itemNameIsCode(sel.X) {
					hit(e, itemNameCodeNumber, "an item code's Name() is its number")
				}
			case name == "Sprintf" || name == "Sprint":
				if len(e.Args) > 0 {
					if lit, ok := e.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING && itemCodeLabel.MatchString(lit.Value) {
						hit(e, itemNameCodeLabel, "formats an item code into a label")
					}
				}
			case name == "make" && len(e.Args) > 0 && itemNameLast(e.Args[0]) == "ItemNames":
				hit(e, itemNameNamesTable, "makes an item-name table")
			}
		case *ast.SelectorExpr:
			switch {
			case e.Sel.Name == "EntryName" && !called[e]:
				hit(e, itemNameRowNameValue, "EntryName passed as a value names rows beside the resolver")
			case e.Sel.Name == "Names" && !called[e] && itemNameLast(e.X) != "Words":
				hit(e, itemNameNamesTable, "reads the shipped item-name table")
			case e.Sel.Name == "Name":
				if id, ok := e.X.(*ast.Ident); ok && fromCode[id.Name] {
					hit(e, itemNameWeaponName, id.Name+".Name is a resolved row's name")
				}
			}
		case *ast.CompositeLit:
			switch itemNameLast(e.Type) {
			case "ItemNames":
				hit(e, itemNameNamesTable, "builds an item-name table")
			case "ItemSubject":
				for _, el := range e.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok && itemNameLast(kv.Key) == "Saved" {
						switch kv.Value.(type) {
						case *ast.Ident, *ast.SelectorExpr:
						default:
							hit(kv, itemNameSavedBuilt, "the saved name is a built value")
						}
					}
				}
			}
		case *ast.KeyValueExpr:
			if k := itemNameLast(e.Key); k == "itemNamer" || k == "ItemNamer" {
				if v := itemNameLast(e.Value); v != "CodeItemNamer" && v != "RowItemNamer" {
					hit(e, itemNameComposerRow, "a composer row other than CodeItemNamer or RowItemNamer")
				}
			}
		case *ast.AssignStmt:
			for _, l := range e.Lhs {
				if itemNameLast(l) == "ItemNamer" {
					hit(e, itemNameComposerRow, "sets a table's composer row")
				}
			}
		case *ast.IfStmt:
			if itemNameMentions(e.Cond, resolved) && (itemNameAssigns(e.Body, resolved) || e.Else != nil && itemNameAssigns(e.Else, resolved)) {
				hit(e, itemNameEmptyFallback, "a resolved item name is tested and replaced")
			}
		}
		return true
	})
	return hits
}

// itemNameIsCode says x is an item code: a conversion to ItemCode, or a name
// that is code or ends in Code.
func itemNameIsCode(x ast.Expr) bool {
	if c, ok := x.(*ast.CallExpr); ok {
		return itemNameLast(c.Fun) == "ItemCode"
	}
	n := itemNameLast(x)
	return n == "code" || strings.HasSuffix(n, "Code")
}

func itemNameMentions(x ast.Node, set map[string]bool) bool {
	found := false
	ast.Inspect(x, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && set[id.Name] {
			found = true
		}
		return !found
	})
	return found
}

func itemNameAssigns(x ast.Node, set map[string]bool) bool {
	found := false
	ast.Inspect(x, func(n ast.Node) bool {
		if a, ok := n.(*ast.AssignStmt); ok {
			for _, l := range a.Lhs {
				if id, ok := l.(*ast.Ident); ok && set[id.Name] {
					found = true
				}
			}
		}
		return !found
	})
	return found
}
