package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The screen-art scan: a screen gets its install pictures from the one screen
// art loader, which reads a manifest of data. No production file under pkg/
// outside the loader names an install picture or opens one: a string literal
// holding a picture extension (.bmp, .256, .16a, .16) or a call to a picture
// decoder or reader is a finding. The decoders themselves live in
// pkg/formats and are not scanned; the loader is pkg/screenart and its
// install source pkg/game/screenart.go.
//
// Files that read non-screen pictures are listed with their kind. Every
// screen reads its pictures through the loader; ScreenArtDebt stays empty.

// screenArtLoader are the files of the one loader.
var screenArtLoader = []string{"pkg/screenart/", "pkg/game/screenart.go", "pkg/formats/"}

// screenArtOtherKinds are the files that read pictures of another kind.
var screenArtOtherKinds = map[string]string{
	"pkg/data/itemcode.go":          "item icon and unit sheet paths",
	"pkg/data/portrait.go":          "unit portrait paths",
	"pkg/data/projectile.go":        "projectile and effect sheet paths",
	"pkg/data/sprite.go":            "unit and object sheet paths",
	"pkg/data/spellicon.go":         "spell icon atlas path",
	"pkg/game/spellicon.go":         "spell icons",
	"pkg/game/cursor.go":            "cursors",
	"pkg/game/cursorregistry.go":    "cursors",
	"pkg/game/font.go":              "fonts",
	"pkg/game/inventory.go":         "item icons",
	"pkg/game/modfiles.go":          "a mod's install picture checked against the install's before it is served",
	"pkg/mod/files.go":              "the kinds of install file a mod may carry",
	"pkg/game/nativecity.go":        "campaign marker names a save carries",
	"pkg/game/portrait.go":          "hero figure composed from unit sheets",
	"pkg/game/projectiles.go":       "projectile and spell effect sheets",
	"pkg/game/sacks.go":             "sack object sheets",
	"pkg/game/sackboundaries.go":    "sack object sheets",
	"pkg/game/statics.go":           "static object sheets",
	"pkg/render/terrain/tileset.go": "terrain tiles",
}

// ScreenArtDebt stays empty: every screen uses the one loader.
var ScreenArtDebt = map[string]bool{}

// screenArtReaders are the calls that read or decode an install picture
// besides the picture decoders' own Decode functions.
var screenArtReaders = map[string]bool{
	"terrain.DecodeBMP8": true, "readInstallBMP": true, "readInstallMask": true,
	"decodeInstallBMP": true, "effectFrames": true, "loadItemIcon": true,
	"decodeCursor16AFrames": true, "decodeCursor256Frames": true, "loadTipGemFrame": true,
}

// pictureDecoders are the packages whose Decode functions decode a picture.
var pictureDecoders = []string{"bmp.Decode", "spr256.Decode", "spr16.Decode"}

func pictureReader(name string) bool {
	if screenArtReaders[name] {
		return true
	}
	for _, p := range pictureDecoders {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// pictureLiteral reports a string that names an install picture: one holding
// a picture extension not preceded by a dot, so a range such as "8..256" is
// not a name.
func pictureLiteral(s string) bool {
	l := strings.ToLower(s)
	for _, ext := range []string{".bmp", ".256", ".16a", ".16"} {
		for i := strings.Index(l, ext); i >= 0; {
			end := i + len(ext)
			if (i == 0 || l[i-1] != '.') && (ext != ".16" || end == len(l)) {
				return true
			}
			next := strings.Index(l[end:], ext)
			if next < 0 {
				break
			}
			i = end + next
		}
	}
	return false
}

// ScreenArtFinding is one site and the shape found there.
type ScreenArtFinding struct {
	File  string
	Line  int
	Shape string
}

func (f ScreenArtFinding) String() string {
	return f.File + ":" + strconv.Itoa(f.Line) + " " + f.Shape
}

// ScreenArtFindings scans files, keyed by module-relative slash path as
// LoadDrawnTextSources returns them.
func ScreenArtFindings(files map[string]string) ([]ScreenArtFinding, error) {
	var out []ScreenArtFinding
	for name, src := range files {
		if screenArtLoaderFile(name) {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		aliases, dots := pictureReaderAliases(f)
		add := func(n ast.Node, shape string) {
			out = append(out, ScreenArtFinding{File: name, Line: fset.Position(n.Pos()).Line, Shape: shape})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ImportSpec:
				return false
			case *ast.BasicLit:
				if x.Kind == token.STRING {
					if s, err := strconv.Unquote(x.Value); err == nil && pictureLiteral(s) {
						add(x, "names an install picture "+strconv.Quote(s))
					}
				}
			case *ast.CallExpr:
				name := callName(x.Fun)
				if qualifier, fn, ok := strings.Cut(name, "."); ok && aliases[qualifier] != "" {
					name = aliases[qualifier] + "." + fn
				} else if !ok {
					for _, pkg := range dots {
						if qualified := pkg + "." + name; pictureReader(qualified) {
							name = qualified
							break
						}
					}
				}
				if pictureReader(name) {
					add(x, "opens a picture through "+name)
				}
			}
			return true
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

func pictureReaderAliases(f *ast.File) (map[string]string, []string) {
	out := map[string]string{}
	var dots []string
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		var name string
		switch path {
		case "againrom/pkg/formats/bmp":
			name = "bmp"
		case "againrom/pkg/formats/spr256":
			name = "spr256"
		case "againrom/pkg/formats/spr16":
			name = "spr16"
		case "againrom/pkg/render/terrain":
			name = "terrain"
		default:
			continue
		}
		alias := name
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		if alias == "." {
			dots = append(dots, name)
		} else {
			out[alias] = name
		}
	}
	return out, dots
}

// callName is a called function's name: f or pkg.f.
func callName(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name
		}
	}
	return ""
}

func screenArtLoaderFile(name string) bool {
	for _, l := range screenArtLoader {
		if name == l || strings.HasSuffix(l, "/") && strings.HasPrefix(name, l) {
			return true
		}
	}
	return false
}

// CheckScreenArt turns the findings into violations: a finding in a file
// neither of another kind nor on the debt list, and a debt entry with no
// finding left.
func CheckScreenArt(found []ScreenArtFinding, debt map[string]bool) []string {
	var out []string
	held := map[string]bool{}
	for _, f := range found {
		if screenArtOtherKinds[f.File] != "" {
			continue
		}
		if debt[f.File] {
			held[f.File] = true
			continue
		}
		out = append(out, f.String()+"; name the picture in a screen's art manifest and read it through the one screen art loader")
	}
	names := make([]string, 0, len(debt))
	for name := range debt {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !held[name] {
			out = append(out, name+" no longer opens a picture itself; remove it from ScreenArtDebt")
		}
	}
	return out
}
