package archtest

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

const savPackagePath = "againrom/pkg/formats/sav"

type savProducerKind int

const (
	// savDocument serializes a whole save file.
	savDocument savProducerKind = iota
	// savPart builds one part of the document the document producer encodes.
	savPart
	// savReader matches the byte-producer signature but returns bytes read
	// from its input or numbers that are not serialized output.
	savReader
)

type savByteProducer struct {
	kind   savProducerKind
	reason string
}

// savByteProducers lists every exported function or method of the SAV
// package that returns serialized bytes or writes them: a []byte or []uint32
// result, or an io.Writer parameter. An unlisted one is a finding.
var savByteProducers = map[string]savByteProducer{
	"EncodeDocumentData":           {savDocument, "the one document encoder; the current producer and the two load re-encoders call it"},
	"File.Marshal":                 {savDocument, "re-emits a container opened from bytes, for round-trip and edit tools"},
	"CityProvenance.Marshal":       {savDocument, "authors a town save from provenance, for synthetic fixtures"},
	"Compress":                     {savPart, "compresses the document body inside the container encoder"},
	"EncodeDocPayload":             {savPart, "packs the native payload envelope into document chunks"},
	"Decompress":                   {savReader, "decodes a stored body"},
	"File.CellRecord":              {savReader, "returns a view of one record in an opened body"},
	"NativeActions":                {savReader, "reads the native action supplement from a loaded document"},
	"NativeMods":                   {savReader, "reads the mod mark payload from a loaded document"},
	"ReserveDocumentKeys":          {savReader, "allocates document key numbers, not bytes"},
	"DocumentFragment.MarshalJSON": {savReader, "a JSON view of a fragment for tools, not SAV bytes"},
}

// savByteCallers lists, by package and function, every production caller of
// a document or part producer outside the SAV package and pkg/game. A new
// caller is a finding; pkg/game is held by CheckSaveProducer.
var savByteCallers = map[string]string{
	"againrom/cmd/savtool verify":            "re-emits each opened save in memory to report byte identity; writes no file",
	"againrom/cmd/savtool set":               "developer edit of an opened save, written only to an explicit -out path",
	"againrom/internal/cityfixture Original": "synthetic town bytes; only tests import the package",
}

// CheckSaveByteProducers compares the SAV package's exported byte producers
// with savByteProducers in both directions.
func CheckSaveByteProducers(p *CheckedPackage) []string {
	var findings []string
	found := map[string]bool{}
	for _, file := range p.Files[:p.NumProd] {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			obj, _ := p.Info.Defs[fn.Name].(*types.Func)
			if !savByteProducerSignature(obj) {
				continue
			}
			key := savFuncKey(obj)
			found[key] = true
			if _, listed := savByteProducers[key]; !listed {
				findings = append(findings, key+" is an unlisted SAV byte producer")
			}
		}
	}
	for key := range savByteProducers {
		if !found[key] {
			findings = append(findings, "listed SAV byte producer "+key+" is absent")
		}
	}
	sort.Strings(findings)
	return findings
}

// CheckSaveByteCallers finds production references to SAV byte producers in
// every package except the SAV package and pkg/game. A listed reader is free;
// a document or part producer needs a savByteCallers entry; an unlisted
// producer is always a finding. A caller entry nothing uses is stale.
func CheckSaveByteCallers(packages []*CheckedPackage) []string {
	var findings []string
	used := map[string]bool{}
	for _, p := range packages {
		if p.ImportPath == savPackagePath || p.ImportPath == "againrom/pkg/game" {
			continue
		}
		for _, file := range p.Files[:p.NumProd] {
			for _, decl := range file.Decls {
				owner := "package scope"
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if obj, _ := p.Info.Defs[fn.Name].(*types.Func); obj != nil {
						owner = savFuncKey(obj)
					}
				}
				caller := p.ImportPath + " " + owner
				ast.Inspect(decl, func(node ast.Node) bool {
					id, ok := node.(*ast.Ident)
					if !ok {
						return true
					}
					called, _ := p.Info.Uses[id].(*types.Func)
					if !savByteProducerSignature(called) {
						return true
					}
					key := savFuncKey(called)
					entry, listed := savByteProducers[key]
					switch {
					case !listed:
						findings = append(findings, caller+" calls unlisted SAV byte producer "+key)
					case entry.kind == savReader:
					case savByteCallers[caller] == "":
						findings = append(findings, caller+" is an unlisted caller of SAV byte producer "+key)
					default:
						used[caller] = true
					}
					return true
				})
			}
		}
	}
	for caller := range savByteCallers {
		if !used[caller] {
			findings = append(findings, "listed SAV byte caller "+caller+" calls no producer")
		}
	}
	sort.Strings(findings)
	return findings
}

// savByteProducerSignature reports an exported SAV function or method with a
// []byte or []uint32 result or an io.Writer parameter.
func savByteProducerSignature(fn *types.Func) bool {
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != savPackagePath || !fn.Exported() {
		return false
	}
	signature, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	for i := 0; i < signature.Results().Len(); i++ {
		slice, ok := signature.Results().At(i).Type().Underlying().(*types.Slice)
		if !ok {
			continue
		}
		if elem, ok := slice.Elem().Underlying().(*types.Basic); ok && (elem.Kind() == types.Uint8 || elem.Kind() == types.Uint32) {
			return true
		}
	}
	for i := 0; i < signature.Params().Len(); i++ {
		if types.Implements(signature.Params().At(i).Type(), savWriterInterface) {
			return true
		}
	}
	return false
}

var savWriterInterface = func() *types.Interface {
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.NewSlice(types.Typ[types.Byte])))
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]), types.NewVar(token.NoPos, nil, "err", types.Universe.Lookup("error").Type()))
	write := types.NewFunc(token.NoPos, nil, "Write", types.NewSignatureType(nil, nil, nil, params, results, false))
	return types.NewInterfaceType([]*types.Func{write}, nil).Complete()
}()

func savFuncKey(fn *types.Func) string {
	if receiver := saveReceiverName(fn); receiver != "" {
		return receiver + "." + fn.Name()
	}
	return fn.Name()
}
