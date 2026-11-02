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

// The light plane scan: a frame's map light reaches the viewer as one source
// list through one setter, merged into one plane by one builder and read by
// one merge. In pkg it refuses, outside the site each is allowed:
//   - a Viewer method named Set...Light... other than SetLightSources and the
//     day clock's SetLightClock, which carries no light source;
//   - a call of SetLightSources outside mapWorld.push, a second call there,
//     and any call of a former setter;
//   - in pkg/ui, a read of the plane outside its builder and its merge, a
//     former plane field anywhere, the plane type outside its declaration,
//     builder and Viewer field, and a call of the builder or the merge
//     outside the setter and the two readers.

const (
	lightPlaneFile = "pkg/ui/objectlight.go"
	lightPushFile  = "pkg/game/world.go"
)

var lightSetterName = regexp.MustCompile(`^Set\w*Light`)

type lightPlaneShape struct {
	reason  string
	uiOnly  bool
	allowed func(file, context string) bool
}

func lightPlaneSites(contexts ...string) func(string, string) bool {
	return func(_, c string) bool {
		for _, want := range contexts {
			if c == want {
				return true
			}
		}
		return false
	}
}

func lightPlaneNowhere(string, string) bool { return false }

var lightPlaneFieldShape = lightPlaneShape{
	"the light plane is read here; read it through Viewer.mergeLight, which cornerScales and spellSpriteFactor ask",
	true, lightPlaneSites("buildLightPlane", "Viewer.mergeLight")}

var formerLightField = lightPlaneShape{
	"a former light plane field is used here; the frame's light is the one plane SetLightSources builds",
	true, lightPlaneNowhere}

var formerLightSetter = lightPlaneShape{
	"a former light setter is called here; hand the frame's light source list to SetLightSources",
	false, lightPlaneNowhere}

var lightPlaneShapes = map[string]lightPlaneShape{
	"frameLight": {
		"the light plane is read here; read it through Viewer.mergeLight, which cornerScales and spellSpriteFactor ask",
		true, lightPlaneSites("Viewer.SetLightSources", "Viewer.mergeLight")},
	"cellLevels":           lightPlaneFieldShape,
	"cellTerrain":          lightPlaneFieldShape,
	"vertexStamps":         lightPlaneFieldShape,
	"pathStamps":           lightPlaneFieldShape,
	"spellLighting":        formerLightField,
	"spellTerrainLighting": formerLightField,
	"lightStamps":          formerLightField,
	"boltLightStamps":      formerLightField,
	"SetSpellLighting":     formerLightSetter,
	"SetLightStamps":       formerLightSetter,
	"SetLightSources": {
		"the light setter is called here; mapWorld.push hands the viewer each frame's one source list",
		false, func(f, c string) bool { return f == lightPushFile && c == "mapWorld.push" }},
	"mergeLight": {
		"the light merge is asked here; cornerScales and spellSpriteFactor are its readers",
		true, lightPlaneSites("Viewer.cornerScales", "Viewer.spellSpriteFactor")},
	"buildLightPlane": {
		"a light plane is built here; SetLightSources builds the one plane",
		true, lightPlaneSites("Viewer.SetLightSources")},
	"lightPlane": {
		"a second light plane is held here; the viewer's light field is the one plane",
		true, lightPlaneSites("lightPlane", "buildLightPlane", "Viewer")},
}

// CheckLightPlane reports every light setter, plane and merge outside the
// one source list's sites. files is keyed by module-relative slash path.
func CheckLightPlane(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the light plane scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	setters, pushCalls := 0, 0
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		ui := strings.HasPrefix(name, "pkg/ui/")
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		for _, unit := range declUnits(f) {
			context := unit.context
			own := declaredName(unit.node)
			if ui && strings.HasPrefix(context, "Viewer.") && lightSetterName.MatchString(strings.TrimPrefix(context, "Viewer.")) {
				switch context {
				case "Viewer.SetLightSources":
					setters++
				case "Viewer.SetLightClock":
				default:
					vs = append(vs, Violation{From: at(unit.node), Reason: "a second light setter is declared here; the frame's light is one source list handed to SetLightSources"})
				}
			}
			ast.Inspect(unit.node, func(n ast.Node) bool {
				var key string
				switch n := n.(type) {
				case *ast.SelectorExpr:
					key = n.Sel.Name
				case *ast.Ident:
					if n == own || n.Name != "buildLightPlane" && n.Name != "lightPlane" {
						return true
					}
					key = n.Name
				default:
					return true
				}
				shape, ok := lightPlaneShapes[key]
				if !ok || shape.uiOnly && !ui {
					return true
				}
				if !shape.allowed(name, context) {
					vs = append(vs, Violation{From: at(n), Reason: shape.reason})
					return false
				}
				if key == "SetLightSources" {
					if pushCalls++; pushCalls > 1 {
						vs = append(vs, Violation{From: at(n), Reason: "the light setter is called twice in a push; a frame hands one source list"})
					}
				}
				return true
			})
		}
	}
	if _, ok := files[lightPlaneFile]; ok && setters == 0 {
		vs = append(vs, Violation{From: lightPlaneFile, Reason: "the light setter SetLightSources is gone - the light plane scan would hold nothing"})
	}
	if _, ok := files[lightPushFile]; ok && pushCalls == 0 {
		vs = append(vs, Violation{From: lightPushFile, Reason: "mapWorld.push no longer calls SetLightSources - the light plane scan would hold nothing"})
	}
	return vs
}

// declaredName is the identifier a declaration names, nil for none.
func declaredName(n ast.Node) *ast.Ident {
	switch d := n.(type) {
	case *ast.FuncDecl:
		return d.Name
	case *ast.TypeSpec:
		return d.Name
	}
	return nil
}
