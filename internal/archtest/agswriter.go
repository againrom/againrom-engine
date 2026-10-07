package archtest

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"
)

// CheckNoAGSWriter rejects production references to the legacy envelope encoder.
// Tests retain the encoder to exercise existing AGS reading and migration.
func CheckNoAGSWriter(m *TypeCheckedModule) []string {
	var paths []string
	for path := range m.Packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var findings []string
	for _, path := range paths {
		p := m.Packages[path]
		for i, file := range p.Files[:p.NumProd] {
			rel := p.RelPaths[i]
			ast.Inspect(file, func(node ast.Node) bool {
				id, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				called, _ := p.Info.Uses[id].(*types.Func)
				if called == nil || called.Name() != "EncodeSave" || called.Pkg() == nil || called.Pkg().Path() != "againrom/pkg/game" {
					return true
				}
				findings = append(findings, fmt.Sprintf("%s calls game.EncodeSave, the retired AGS writer", rel))
				return true
			})
		}
	}
	sort.Strings(findings)
	return findings
}
