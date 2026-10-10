package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
)

// CheckSaveProducer admits one SAV serializer and two load re-encoders. Every
// listed SAV document or part producer is a serializer; a listed reader is not;
// an unlisted SAV byte producer is a finding.
// Resolved function references include callbacks. Compatibility names only
// delegate; construction cannot reach AGS encoding or migration.
func CheckSaveProducer(p *CheckedPackage) []string {
	aliases := map[string]bool{"ExportCurrentWorldSave": true, "ExportOriginalSave": true, "ExportNativeCitySave": true}
	forbidden := map[string]bool{"EncodeSave": true, "CheckSaveForm": true}
	decls := map[*types.Func]*ast.FuncDecl{}
	var root *types.Func
	var findings []string
	for _, file := range p.Files[:p.NumProd] {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			obj, _ := p.Info.Defs[fn.Name].(*types.Func)
			if obj == nil {
				continue
			}
			decls[obj] = fn
			if fn.Name.Name == "ExportCurrentSave" && saveReceiverName(obj) == "FrontEnd" {
				root = obj
			}
		}
	}
	if root == nil {
		return append(findings, "current SAV producer is absent")
	}
	for _, file := range p.Files[:p.NumProd] {
		for _, decl := range file.Decls {
			var owner *types.Func
			path := "package scope"
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner, _ = p.Info.Defs[fn.Name].(*types.Func)
				path = fn.Name.Name
			}
			loadReencoder := owner != nil && saveReceiverName(owner) == "" &&
				(owner.Name() == "repairLoadedEquipmentRows" || owner.Name() == "applyModMark")
			ast.Inspect(decl, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				called, _ := p.Info.Uses[id].(*types.Func)
				if savByteProducerSignature(called) {
					entry, listed := savByteProducers[savFuncKey(called)]
					if !listed {
						findings = append(findings, path+" calls unlisted SAV byte producer "+savFuncKey(called))
						return true
					}
					if entry.kind == savReader {
						return true
					}
				} else if !saveSerializer(called, p.ImportPath) {
					return true
				}
				if called.Name() == "EncodeDocumentData" && (owner == root || loadReencoder) {
					return true
				}
				findings = append(findings, path+" selects second SAV producer via "+called.Name())
				return true
			})
		}
	}
	for obj, fn := range decls {
		if aliases[obj.Name()] && !directCurrentSaveDelegate(p, fn, root) {
			findings = append(findings, obj.Name()+" must delegate directly to ExportCurrentSave")
		}
	}
	// A caller choosing an old exporter by provenance would reintroduce runtime
	// writer selection even while both old exporters happen to be delegates.
	for obj, fn := range decls {
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			called, _ := p.Info.Uses[id].(*types.Func)
			if called != nil && called.Pkg() != nil && called.Pkg().Path() == p.ImportPath && aliases[called.Name()] {
				findings = append(findings, obj.Name()+" selects compatibility writer "+called.Name())
			}
			return true
		})
	}
	functionValues := map[types.Object][]ast.Expr{}
	for _, file := range p.Files[:p.NumProd] {
		ast.Inspect(file, func(node ast.Node) bool {
			var names []ast.Expr
			var values []ast.Expr
			switch n := node.(type) {
			case *ast.AssignStmt:
				names, values = n.Lhs, n.Rhs
			case *ast.ValueSpec:
				values = n.Values
				for _, name := range n.Names {
					names = append(names, name)
				}
			}
			if len(names) == len(values) {
				for i, name := range names {
					id, ok := name.(*ast.Ident)
					if !ok {
						continue
					}
					obj := p.Info.Defs[id]
					if obj == nil {
						obj = p.Info.Uses[id]
					}
					if obj != nil {
						if _, ok := obj.Type().Underlying().(*types.Signature); ok {
							functionValues[obj] = append(functionValues[obj], values[i])
						}
					}
				}
			}
			return true
		})
	}
	check := func(body ast.Node, path string, requireProducer bool) {
		seen := map[*types.Func]bool{}
		seenValues := map[types.Object]bool{}
		var visit func(ast.Node, string)
		visit = func(body ast.Node, path string) {
			ast.Inspect(body, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				obj := p.Info.Uses[id]
				if !seenValues[obj] && len(functionValues[obj]) != 0 {
					seenValues[obj] = true
					for _, value := range functionValues[obj] {
						visit(value, path+" -> "+id.Name)
					}
				}
				called, _ := obj.(*types.Func)
				if called == nil || called.Pkg() == nil {
					return true
				}
				pkg := called.Pkg().Path()
				privateCityWriter := called.Name() == "marshal" && saveReceiverName(called) == "originalCitySaveState"
				if (forbidden[called.Name()] || privateCityWriter) && (pkg == p.ImportPath || pkg == "againrom/pkg/sim") {
					findings = append(findings, fmt.Sprintf("%s -> %s reaches legacy save behavior", path, called.Name()))
				} else if pkg == p.ImportPath && !seen[called] && decls[called] != nil {
					seen[called] = true
					visit(decls[called].Body, path+" -> "+called.Name())
				}
				return true
			})
		}
		visit(body, path)
		if requireProducer && !seen[root] {
			findings = append(findings, path+" does not reach ExportCurrentSave")
		}
	}
	check(decls[root].Body, root.Name(), false)
	// The dialog's reader may decode legacy input. Only preparation and its
	// returned commit, including the installed confirmation wrapper, write saves.
	for obj, fn := range decls {
		if saveReceiverName(obj) != "FrontEnd" {
			continue
		}
		switch obj.Name() {
		case "SaveDialogSeams":
			found := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				kv, ok := node.(*ast.KeyValueExpr)
				if !ok {
					return true
				}
				key, ok := kv.Key.(*ast.Ident)
				if ok && key.Name == "Prepare" {
					found = true
					check(kv.Value, "SaveDialogSeams.Prepare", true)
				}
				return true
			})
			if !found {
				findings = append(findings, "SaveDialogSeams.Prepare is absent")
			}
		case "ConfigureSaveSeams":
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				assign, ok := node.(*ast.AssignStmt)
				if !ok || len(assign.Lhs) != len(assign.Rhs) {
					return true
				}
				for i, lhs := range assign.Lhs {
					field, ok := lhs.(*ast.SelectorExpr)
					if ok && (field.Sel.Name == "Prepare" || field.Sel.Name == "Commit") {
						check(assign.Rhs[i], "ConfigureSaveSeams."+field.Sel.Name, false)
					}
				}
				return true
			})
		}
	}
	sort.Strings(findings)
	return findings
}

func saveSerializer(fn *types.Func, gamePath string) bool {
	if fn == nil || fn.Pkg() == nil {
		return false
	}
	if fn.Pkg().Path() == "againrom/pkg/formats/sav" {
		return fn.Name() == "EncodeDocumentData" || fn.Name() == "Marshal"
	}
	return fn.Pkg().Path() == gamePath && fn.Name() == "Marshal" && saveReceiverName(fn) == "originalCityDocument"
}

func saveReceiverName(fn *types.Func) string {
	signature, ok := fn.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return ""
	}
	t := signature.Recv().Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

func directCurrentSaveDelegate(p *CheckedPackage, fn *ast.FuncDecl, root *types.Func) bool {
	if len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || p.Info.Uses[selector.Sel] != root || fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 {
		return false
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok || p.Info.Uses[receiver] != p.Info.Defs[fn.Recv.List[0].Names[0]] {
		return false
	}
	var params []types.Object
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			params = append(params, p.Info.Defs[name])
		}
	}
	if len(params) != len(call.Args) {
		return false
	}
	for i, arg := range call.Args {
		id, ok := arg.(*ast.Ident)
		if !ok || p.Info.Uses[id] != params[i] {
			return false
		}
	}
	return true
}
