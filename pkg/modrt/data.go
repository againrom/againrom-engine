package modrt

import (
	"errors"
	"fmt"
	"io/fs"

	"go.starlark.net/starlark"

	"againrom/pkg/mod"
)

type dataValue struct {
	sh               *shared
	id, dir          string
	loader           *loader
	loaded, explicit map[string]bool
	frozen           bool
	settings         []mod.SettingValue
	state            bool
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

func readBesideManifest(m mod.Entry, values []mod.SettingValue, sh *shared) (*dataValue, error) {
	d := &dataValue{sh: sh, id: m.Manifest.ID, dir: m.Dir, loaded: map[string]bool{}, explicit: map[string]bool{}, settings: values}
	t, err := mod.OpenTree(m.Dir)
	if err != nil {
		return nil, err
	}
	r := mod.DataReader{Mod: d.id, Dir: d.dir, Lang: sh.lang, Settings: values}
	for _, k := range mod.DataKinds() {
		for _, rel := range k.Files {
			st, err := t.Stat(rel)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("mod %q: %s: %w", d.id, rel, err)
			}
			if st.IsDir() {
				continue
			}
			if err := d.read(k, rel); err != nil {
				return nil, fmt.Errorf("mod %q: %w", d.id, err)
			}
		}
		held, err := k.ReadNamed(r, &sh.data, t)
		if err != nil {
			return nil, fmt.Errorf("mod %q: %w", d.id, err)
		}
		d.state = d.state || held
	}
	return d, nil
}

func (d *dataValue) read(k mod.DataKind, rel string) error {
	key := k.Name
	if d.loaded[key] {
		return nil
	}
	src, _, err := mod.ReadInside(d.dir, rel)
	if err != nil {
		return err
	}
	r := mod.DataReader{Mod: d.id, Dir: d.dir, Lang: d.sh.lang, Settings: d.settings}
	if err := k.Read(r, &d.sh.data, rel, src); err != nil {
		return err
	}
	d.loaded[key] = true
	d.state = d.state || k.State
	return nil
}

func (d *dataValue) add(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var file string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &file); err != nil {
		return nil, err
	}
	if d.frozen {
		return nil, errors.New("game.data.add: data is loaded during init only")
	}
	_, rel, err := d.loader.read(file)
	if err != nil {
		return nil, fmt.Errorf("game.data.add(%q): %w", file, err)
	}
	kind, ok := mod.ScriptDataKind(rel)
	if !ok {
		return nil, fmt.Errorf("game.data.add(%q): no data file of that name is loaded (loadable: %s)", file, mod.ScriptDataFiles())
	}
	key := kind.Name
	if d.explicit[key] {
		return nil, fmt.Errorf("game.data.add(%q): the file is already loaded", file)
	}
	d.explicit[key] = true
	if err := d.read(kind, rel); err != nil {
		return nil, err
	}
	return starlark.None, nil
}
