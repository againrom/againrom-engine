package modrt

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"

	"againrom/pkg/mod"
)

// dataValue is game.data: the data files a mod may load.
type dataValue struct {
	sh       *shared
	id       string
	dir      string
	loader   *loader
	loaded   map[string]bool
	frozen   bool
	settings []mod.SettingValue
	// state is set when the script adds a data file.
	state *bool
}

func (d *dataValue) String() string        { return "<game.data>" }
func (d *dataValue) Type() string          { return "game.data" }
func (d *dataValue) Freeze()               { d.frozen = true }
func (d *dataValue) Truth() starlark.Bool  { return true }
func (d *dataValue) Hash() (uint32, error) { return 0, errors.New("unhashable type: game.data") }
func (d *dataValue) AttrNames() []string   { return []string{"add"} }
func (d *dataValue) Attr(name string) (starlark.Value, error) {
	if name == "add" {
		return starlark.NewBuiltin("game.data.add", d.add), nil
	}
	return nil, nil
}

// add is game.data.add(path): it reads a data file of the mod folder and adds
// what it describes to the game. A file is loaded once, during init.
func (d *dataValue) add(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	if d.frozen {
		return nil, errors.New("game.data.add: data is loaded during init only")
	}
	src, rel, err := d.loader.read(path)
	if err != nil {
		return nil, fmt.Errorf("game.data.add(%q): %w", path, err)
	}
	kind, ok := mod.ScriptDataKind(rel)
	if !ok {
		return nil, fmt.Errorf("game.data.add(%q): no data file of that name is loaded (loadable: %s)", path, mod.ScriptDataFiles())
	}
	if d.loaded[rel] {
		return nil, fmt.Errorf("game.data.add(%q): the file is already loaded", path)
	}
	d.loaded[rel] = true
	if d.state != nil {
		*d.state = true
	}
	r := mod.DataReader{Mod: d.id, Dir: d.dir, Lang: d.sh.lang, Settings: d.settings}
	if err := kind.Read(r, &d.sh.data, rel, src); err != nil {
		return nil, err
	}
	return starlark.None, nil
}
