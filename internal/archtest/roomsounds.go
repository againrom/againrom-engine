package archtest

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// roomSoundHome is the one file that turns a room event into a sound request.
const roomSoundHome = "pkg/game/roompage.go"

// roomEventDoors are the functions that raise an event on a town room. Their
// callers name an event by a constant of roomSoundHome, never by a string.
var roomEventDoors = map[string]bool{
	"roomSound": true, "shownRoomSound": true, "shopRoomSound": true,
	"raiseRoomEvent": true, "raiseShownRoomEvent": true,
}

// soundSinks are the functions that hand a sound name to a player or to a
// scene's sound slots.
var soundSinks = map[string]bool{
	"namedSample": true, "FixedRequest": true, "Dispatch": true, "RequestFor": true,
	"Request": true, "Restart": true, "RestartFor": true, "RequestSample": true,
	"PlaySound": true, "Trigger": true,
}

// RoomSoundKeys answers the sample names the town descriptions' sound tables
// request: every key of a slot or of a trigger under a room's sounds.
func RoomSoundKeys(root string) (map[string]bool, error) {
	dir := filepath.Join(root, "pkg", "game", "towns")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var tree any
		if err := json.Unmarshal(raw, &tree); err != nil {
			return nil, err
		}
		collectSoundKeys(tree, false, keys)
	}
	return keys, nil
}

func collectSoundKeys(v any, inSounds bool, keys map[string]bool) {
	switch v := v.(type) {
	case []any:
		for _, e := range v {
			collectSoundKeys(e, inSounds, keys)
		}
	case map[string]any:
		for k, e := range v {
			if s, ok := e.(string); ok && inSounds && k == "key" && s != "" {
				keys[s] = true
			}
			collectSoundKeys(e, inSounds || k == "sounds", keys)
		}
	}
}

// RoomSoundSites lists the places in production code of pkg/game and pkg/ui,
// outside the room page file, that choose a town room's sound beside the room
// description: a call of a room event door with a string, a sound name passed
// to a player or the scene's slots, a map from an action to a sound name, and
// a string equal to a key the descriptions' sound tables request. Tests are
// not scanned.
func RoomSoundSites(root string) ([]string, error) {
	keys, err := RoomSoundKeys(root)
	if err != nil {
		return nil, err
	}
	var sites []string
	for _, top := range []string{"pkg/game", "pkg/ui"} {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(top)), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || rel == roomSoundHome {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, roomSoundsInSource(rel, src, keys)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

func roomSoundsInSource(name string, src []byte, keys map[string]bool) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var hits []string
	add := func(pos token.Pos, what string) {
		site := name + ":" + strconv.Itoa(fset.Position(pos).Line) + " " + what
		if !seen[site] {
			seen[site] = true
			hits = append(hits, site)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			callee := calleeName(n)
			for _, arg := range n.Args {
				switch {
				case roomEventDoors[callee] && hasStringLiteral(arg, func(string) bool { return true }):
					add(arg.Pos(), callee+" is called with a string")
				case soundSinks[callee] && hasStringLiteral(arg, func(s string) bool { return soundName(s, keys) }):
					add(arg.Pos(), callee+" is passed a sound name")
				}
			}
		case *ast.CompositeLit:
			if _, ok := n.Type.(*ast.MapType); !ok {
				return true
			}
			for _, el := range n.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok && hasStringLiteral(kv.Value, func(s string) bool { return soundName(s, keys) }) {
					add(kv.Pos(), "a map names a sound")
				}
			}
		case *ast.BasicLit:
			if n.Kind == token.STRING {
				if s, err := strconv.Unquote(n.Value); err == nil && keys[s] {
					add(n.Pos(), "a string is the key "+s+" of a room sound table")
				}
			}
		}
		return true
	})
	return hits
}

func soundName(s string, keys map[string]bool) bool {
	return keys[s] || strings.HasSuffix(strings.ToLower(s), ".wav")
}

// hasStringLiteral reports whether e holds a string literal that accept takes.
func hasStringLiteral(e ast.Expr, accept func(string) bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil && accept(s) {
				found = true
			}
		}
		return !found
	})
	return found
}
