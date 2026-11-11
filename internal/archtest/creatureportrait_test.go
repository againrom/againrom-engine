package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestCreaturePortraitRoutesShareOneResolver(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		file, function, builder string
	}{
		{"pkg/game/tiers.go", "placementTier", "resolveCreaturePortrait"},
		{"pkg/game/tiers.go", "sourceActorTier", "resolveCreaturePortrait"},
		{"pkg/game/tiers.go", "savedActorTiers", "resolveCreaturePortrait"},
		{"pkg/game/tiers.go", "partyTiers", "resolveCreaturePortrait"},
		{"pkg/game/actormanifest.go", "installActorManifest", "sourceActorTier"},
		{"pkg/mapload/actorconstructor.go", "ConstructActorBasis", "placedCreatureFace"},
	} {
		src, err := os.ReadFile(filepath.Join(root, row.file))
		if err != nil {
			t.Fatal(err)
		}
		calls := editorIdentityCalls(string(src))[row.function]
		if calls[row.builder] != 1 {
			t.Errorf("%s must call %s once", row.function, row.builder)
		}
		if row.file == "pkg/game/tiers.go" && (calls["ActorClass"] != 0 || calls["ComposesFigure"] != 0) {
			t.Errorf("%s must leave actor and party eligibility to the resolver", row.function)
		}
	}
	src, err := os.ReadFile(filepath.Join(root, "pkg/mapload/placedfigure.go"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	decoders := 0
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "placedCreatureFace" {
			continue
		}
		decoders++
		if fn.Recv == nil || len(fn.Recv.List) != 1 {
			t.Fatal("placedCreatureFace must decode one format field")
		}
		receiver, ok := fn.Recv.List[0].Type.(*ast.Ident)
		if !ok || receiver.Name != "creatureFaceField" {
			t.Fatal("placedCreatureFace must use format field data")
		}
		if len(fn.Body.List) != 1 {
			t.Fatal("placedCreatureFace must contain only field extraction")
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			t.Fatal("placedCreatureFace must return one field")
		}
		if _, ok := ret.Results[0].(*ast.IndexExpr); !ok {
			t.Fatal("placedCreatureFace must index format field data")
		}
	}
	if decoders != 1 {
		t.Fatalf("placedCreatureFace decoders = %d, want 1", decoders)
	}
}
