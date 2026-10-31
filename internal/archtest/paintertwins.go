package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The painter twin scan: a picture pkg/ui paints on both the CPU and the GPU
// is planned by one producer, and each backend runs the plan.

// A painter's first image parameter is *ebiten.Image (GPU) or *image.RGBA
// (CPU). Its stem is its name less a leading draw, apply, paint, compose,
// blit, overlay, render, run or present and the words CPU, GPU, Ebiten, RGBA
// and Texture; a bare verb's stem is its receiver type. Two painters of one
// stem and both types are twins, and each must run a plan: take a ...Plan,
// call a producer, or call only functions that do, and call none of the
// producer's steps (the unexported pkg/ui functions it calls). A producer
// returns a plan and takes none; a presenter (...Present returning a placed
// *image.RGBA) and painterPlanProducers produce too.

// Beside the pairs, each plan's steps have one caller: the frame shadow
// tiles, mask, tone, level and counted-mask writes only in their producer;
// the native-to-wide segments only in wideFrameLayout.plan; a picture
// centred on a wider frame only in planDetachedDialogue and the layout.

// painterPlanProducers names producers whose plan type is not named ...Plan,
// with the plan they return.
var painterPlanProducers = map[string]string{
	"pkg/ui/mapeditorpassability.go:MapEditor.overlayQuads": "the passability quads the GPU batches and the CPU rasterises",
}

// painterNotTwins names the stem pairs that paint two different things, as
// "GPU key|CPU key".
var painterNotTwins = map[string]string{
	"pkg/ui/viewer.go:Viewer.drawFrame|pkg/ui/widgetframe.go:drawFrame":      "the mission screen's video frame, not a window frame",
	"pkg/ui/gamemenu.go:drawMenuLabel|pkg/ui/menulabel.go:App.drawMenuLabel": "a game menu row's label, not the main menu's corner label",
}

// painterStepOwners names, for each plan step, the functions that may use
// it.
var painterStepOwners = map[string][]string{
	"frameShadowTiles":      {"planFrameShadows"},
	"frameShadowMask":       {"planFrameShadows"},
	"frameShadowTone":       {"planFrameShadows"},
	"frameShadowLevel":      {"DialogueBackdrop.shadowTable"},
	"frameTiles":            {"drawFrameBody", "frameShadowTiles"},
	"dialogueShadowProgram": {"dialogueBackdropState.drawShadowPlan"},
	"submitDialogueShadow":  {"dialogueBackdropState.drawShadowPlan"},
	"segments":              {"wideFrameLayout.plan"},
	"destination":           {"wideFrameLayout.plan"},
}

var (
	painterVerb  = regexp.MustCompile(`^(draw|apply|paint|compose|blit|overlay|render|run|present)`)
	painterWords = regexp.MustCompile(`CPU|GPU|Ebiten|RGBA|Texture`)
)

type painterFunc struct {
	file, key, name, recv string
	decl                  *ast.FuncDecl
	dest                  string
	at                    string
}

// painterStem is fn's name without its verb and backend words.
func painterStem(name, recv string) string {
	s := painterWords.ReplaceAllString(painterVerb.ReplaceAllString(name, ""), "")
	if s == "" {
		s = recv + "."
	}
	return strings.ToLower(s)
}

func painterTypeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + painterTypeName(t.X)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + painterTypeName(t.Elt)
	case *ast.Ellipsis:
		return "..." + painterTypeName(t.Elt)
	}
	return ""
}

func painterRecv(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	return strings.TrimPrefix(painterTypeName(fn.Recv.List[0].Type), "*")
}

func painterTypes(fl *ast.FieldList) []string {
	var out []string
	if fl == nil {
		return out
	}
	for _, f := range fl.List {
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, painterTypeName(f.Type))
		}
	}
	return out
}

// painterCallees is every function name fn's body calls or names: a bare
// identifier, or a selector's name.
func painterCallees(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if fn.Body == nil {
		return out
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			switch f := c.Fun.(type) {
			case *ast.Ident:
				out[f.Name] = true
			case *ast.SelectorExpr:
				out[f.Sel.Name] = true
			}
		}
		return true
	})
	return out
}

// painterNames is every function fn's body calls and every bare name it
// reads; a field selected or keyed by name is not a use.
func painterNames(fn *ast.FuncDecl) map[string]bool {
	out := painterCallees(fn)
	if fn.Body == nil {
		return out
	}
	skip := map[*ast.Ident]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			skip[n.Sel] = true
		case *ast.KeyValueExpr:
			if k, ok := n.Key.(*ast.Ident); ok {
				skip[k] = true
			}
		case *ast.Ident:
			if !skip[n] {
				out[n.Name] = true
			}
		}
		return true
	})
	return out
}

func painterIsPlan(t string, plans map[string]bool) bool {
	return plans[strings.TrimLeft(t, "*[].")]
}

// CheckPainterTwins reports every twin pair that does not run one plan,
// every plan step used outside its owners, and every list entry nothing
// matches. files is keyed by module-relative slash path.
func CheckPainterTwins(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no sources scanned - the painter twin scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if path.Dir(name) == "pkg/ui" && !strings.HasSuffix(name, "_test.go") && path.Ext(name) == ".go" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	plans := map[string]bool{}
	var funcs []*painterFunc
	fset := token.NewFileSet()
	for _, name := range names {
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && strings.HasSuffix(ts.Name.Name, "Plan") {
						plans[ts.Name.Name] = true
					}
				}
			case *ast.FuncDecl:
				p := &painterFunc{file: name, name: d.Name.Name, recv: painterRecv(d), decl: d,
					at: fmt.Sprintf("%s:%d", name, fset.Position(d.Pos()).Line)}
				p.key = name + ":" + p.name
				if p.recv != "" {
					p.key = name + ":" + p.recv + "." + p.name
				}
				for _, t := range painterTypes(d.Type.Params) {
					if t == "*ebiten.Image" || t == "*image.RGBA" {
						p.dest = t
						break
					}
				}
				funcs = append(funcs, p)
			}
		}
	}
	byName := map[string][]*painterFunc{}
	for _, p := range funcs {
		byName[p.name] = append(byName[p.name], p)
	}
	// Producers and the steps each calls.
	producer := map[string]bool{}
	seenProducer := map[string]bool{}
	for _, p := range funcs {
		ins := append(painterTypes(p.decl.Type.Params), p.recv)
		outs := painterTypes(p.decl.Type.Results)
		takes, gives := false, false
		for _, t := range ins {
			takes = takes || painterIsPlan(t, plans)
		}
		for _, t := range outs {
			gives = gives || painterIsPlan(t, plans)
		}
		presenter := strings.HasSuffix(p.name, "Present") && strings.Contains(strings.Join(outs, ","), "*image.RGBA") && strings.Contains(strings.Join(outs, ","), "image.Point")
		if painterPlanProducers[p.key] != "" {
			seenProducer[p.key] = true
			presenter = true
		}
		if gives && !takes || presenter {
			producer[p.name] = true
		}
	}
	steps := map[string]map[string]bool{}
	for _, p := range funcs {
		if !producer[p.name] {
			continue
		}
		if steps[p.name] == nil {
			steps[p.name] = map[string]bool{}
		}
		for c := range painterCallees(p.decl) {
			if len(byName[c]) > 0 && !producer[c] && !ast.IsExported(c) {
				steps[p.name][c] = true
			}
		}
	}
	// A function runs a plan directly when it takes one or calls a producer.
	direct := func(p *painterFunc) (bool, []string) {
		var used []string
		ok := false
		for _, t := range append(painterTypes(p.decl.Type.Params), p.recv) {
			if painterIsPlan(t, plans) {
				ok = true
			}
		}
		for c := range painterCallees(p.decl) {
			if producer[c] {
				ok = true
				used = append(used, c)
			}
		}
		return ok, used
	}
	runs := map[*painterFunc]bool{}
	for _, p := range funcs {
		runs[p], _ = direct(p)
	}
	for changed := true; changed; {
		changed = false
		for _, p := range funcs {
			if runs[p] {
				continue
			}
			for c := range painterCallees(p.decl) {
				cands := byName[c]
				if len(cands) == 0 || c == p.name {
					continue
				}
				all := true
				for _, q := range cands {
					all = all && runs[q]
				}
				if all {
					runs[p], changed = true, true
					break
				}
			}
		}
	}
	// own says why p builds its own geometry, or "" when it runs a plan.
	own := func(p *painterFunc) string {
		if !runs[p] {
			return "takes no plan and calls no plan producer"
		}
		_, used := direct(p)
		calls := painterCallees(p.decl)
		for _, u := range used {
			var hit []string
			for s := range steps[u] {
				if calls[s] {
					hit = append(hit, s)
				}
			}
			sort.Strings(hit)
			if len(hit) > 0 {
				return "calls " + strings.Join(hit, ", ") + ", a step of " + u
			}
		}
		return ""
	}
	byStem := map[string][]*painterFunc{}
	for _, p := range funcs {
		if p.dest != "" {
			st := painterStem(p.name, p.recv)
			byStem[st] = append(byStem[st], p)
		}
	}
	stems := make([]string, 0, len(byStem))
	for st := range byStem {
		stems = append(stems, st)
	}
	sort.Strings(stems)
	seenNot := map[string]bool{}
	for _, st := range stems {
		for _, g := range byStem[st] {
			if g.dest != "*ebiten.Image" {
				continue
			}
			for _, c := range byStem[st] {
				if c.dest != "*image.RGBA" {
					continue
				}
				pair := g.key + "|" + c.key
				if painterNotTwins[pair] != "" {
					seenNot[pair] = true
					continue
				}
				for _, m := range []*painterFunc{g, c} {
					if why := own(m); why != "" {
						vs = append(vs, Violation{From: m.at, Reason: fmt.Sprintf("%s and %s are painter twins (stem %q, %s and %s) and %s builds its own geometry: it %s; plan the picture once and run the plan in each", g.key, c.key, st, g.dest, c.dest, m.name, why)})
					}
				}
			}
		}
	}
	// Plan steps outside their owners.
	for _, p := range funcs {
		owner := p.name
		if p.recv != "" {
			owner = p.recv + "." + p.name
		}
		// A function step is used by a call; a value step by its name.
		calls, named := painterCallees(p.decl), painterNames(p.decl)
		var keys []string
		for step := range painterStepOwners {
			keys = append(keys, step)
		}
		sort.Strings(keys)
		for _, step := range keys {
			used := named
			if len(byName[step]) > 0 {
				used = calls
			}
			if !used[step] || step == p.name {
				continue
			}
			allowed := false
			for _, o := range painterStepOwners[step] {
				allowed = allowed || o == owner
			}
			if !allowed {
				vs = append(vs, Violation{From: p.at, Reason: fmt.Sprintf("%s uses %s, a plan step only %s may use", owner, step, strings.Join(painterStepOwners[step], " and "))})
			}
		}
		if p.decl.Body == nil {
			continue
		}
		ast.Inspect(p.decl.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.IncDecStmt:
				if ix, ok := n.X.(*ast.IndexExpr); ok {
					if sel, ok := ix.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Pix" && owner != "frameShadowMask" {
						vs = append(vs, Violation{From: p.at, Reason: owner + " counts a mask in a picture's Pix; only frameShadowMask builds a counted shadow mask"})
					}
				}
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "NewLevel" && len(n.Args) == 3 {
					if lit, ok := n.Args[2].(*ast.BasicLit); ok && lit.Value == "6" {
						vs = append(vs, Violation{From: p.at, Reason: owner + " builds the frame shadow's level-6 table itself; ask DialogueBackdrop.shadowTable"})
					}
				}
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Pt" && len(n.Args) == 2 && painterWideCentring(n.Args[0], n.Args[1]) && owner != "planDetachedDialogue" {
					vs = append(vs, Violation{From: p.at, Reason: owner + " centres a picture on a wider frame itself; ask planDetachedDialogue"})
				}
			case *ast.BinaryExpr:
				if painterHalfOf(n, "frame.W", false) && !(p.file == "pkg/ui/wideframe.go" && p.recv == "wideFrameLayout") {
					vs = append(vs, Violation{From: p.at, Reason: owner + " centres the native frame on a wider one itself; ask the wideFrameLayout"})
				}
			}
			return true
		})
	}
	for _, list := range []struct {
		m    map[string]string
		seen map[string]bool
		what string
	}{{painterPlanProducers, seenProducer, "a plan producer"}, {painterNotTwins, seenNot, "not a painter twin"}} {
		keys := make([]string, 0, len(list.m))
		for k := range list.m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if !list.seen[k] {
				vs = append(vs, Violation{From: "internal/archtest/paintertwins.go", Reason: k + " is listed as " + list.what + " but nothing matches it; remove the entry"})
			}
		}
	}
	return vs
}

func painterExpr(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return painterExpr(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		return painterExpr(e.Fun) + "()"
	case *ast.ParenExpr:
		return painterExpr(e.X)
	}
	return ""
}

// painterHalfOf reports whether e is (a - b) / 2 with b, or with first a,
// spelled want.
func painterHalfOf(e ast.Expr, want string, first bool) bool {
	q, ok := e.(*ast.BinaryExpr)
	if !ok || q.Op != token.QUO {
		return false
	}
	if lit, ok := q.Y.(*ast.BasicLit); !ok || lit.Value != "2" {
		return false
	}
	d, ok := ast.Unparen(q.X).(*ast.BinaryExpr)
	if !ok || d.Op != token.SUB {
		return false
	}
	if first {
		return painterExpr(d.X) == want
	}
	return painterExpr(d.Y) == want
}

// painterWideCentring reports whether x, y is the origin of a picture
// centred on a frame wider than the native one: ((w - p)/2, (frame.H - q)/2)
// with w not the native width.
func painterWideCentring(x, y ast.Expr) bool {
	if !painterHalfOf(y, "frame.H", true) {
		return false
	}
	q, ok := x.(*ast.BinaryExpr)
	if !ok || q.Op != token.QUO {
		return false
	}
	d, ok := ast.Unparen(q.X).(*ast.BinaryExpr)
	return ok && d.Op == token.SUB && painterExpr(d.X) != "frame.W"
}
