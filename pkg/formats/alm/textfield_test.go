package alm

import (
	"encoding/binary"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// Each authored text field reads and writes Cyrillic through its own code
// page and refuses what that code page or its width cannot hold.
func TestTextFieldsStateTheirCodec(t *testing.T) {
	for _, f := range []TextField{MapNameText, DescriptionText, PlayerNameText, ScriptLabelText} {
		img, err := f.Image("Стража")
		if err != nil || len(img) != f.Width || f.Decode(img) != "Стража" {
			t.Errorf("%s: % x %v", f.Name, img, err)
		}
		if _, err := f.Image("中"); !errors.Is(err, ErrFieldUnencodable) {
			t.Errorf("%s accepts a rune Windows-1251 lacks: %v", f.Name, err)
		}
		if _, err := f.Image("a\x00b"); !errors.Is(err, ErrFieldRoundTrip) {
			t.Errorf("%s accepts an embedded NUL: %v", f.Name, err)
		}
		full := strings.Repeat("Я", f.Width)
		_, err = f.Image(full)
		if f.Terminated != errors.Is(err, ErrFieldTooLong) {
			t.Errorf("%s: %d bytes with Terminated %v: %v", f.Name, f.Width, f.Terminated, err)
		}
		if _, err := f.Image(full + "Я"); !errors.Is(err, ErrFieldTooLong) {
			t.Errorf("%s accepts %d bytes: %v", f.Name, f.Width+1, err)
		}
	}
	if _, err := EncodeName("Стража"); !errors.Is(err, ErrFieldUnencodable) {
		t.Errorf("EncodeName left the format's ASCII rule: %v", err)
	}
}

// The script reader returns a node label and a trigger name as the text
// Windows-1251 bytes spell.
func TestScriptNamesReadThroughTheirField(t *testing.T) {
	label, _ := charmap.Windows1251.NewEncoder().Bytes([]byte("Смерть"))
	name, _ := charmap.Windows1251.NewEncoder().Bytes([]byte("Победа"))
	node := make([]byte, scriptNodeSize)
	copy(node, label)
	nodes, _, err := readScriptNodes(node, 0, 1, "action")
	if err != nil || nodes[0].Label != "Смерть" {
		t.Fatalf("node label %q %v", nodes[0].Label, err)
	}
	trig := make([]byte, scriptTriggerSize)
	copy(trig, name)
	binary.LittleEndian.PutUint32(trig[0xb4:], 1)
	triggers, _, err := readScriptTriggers(trig, 0, 1)
	if err != nil || triggers[0].Name != "Победа" {
		t.Fatalf("trigger name %q %v", triggers[0].Name, err)
	}
}

// Only TextField's own methods call the code-page halves, so no field is read
// or written around its codec.
func TestOnlyTextFieldCallsTheCodePages(t *testing.T) {
	halves := map[string]bool{"decodeASCII": true, "decodeCP1251": true, "encodeASCII": true, "encodeCP1251": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range parsed.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			owner := fn.Recv != nil && strings.Contains(string(src[fn.Recv.Pos()-1:fn.Recv.End()-1]), "TextField")
			ast.Inspect(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok && halves[id.Name] && !owner {
					t.Errorf("%s: %s calls %s around TextField", file, fn.Name.Name, id.Name)
				}
				return true
			})
		}
	}
}
