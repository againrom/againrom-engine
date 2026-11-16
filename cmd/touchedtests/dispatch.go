package main

import (
	"go/types"

	"againrom/internal/archtest"
)

// dispatch holds the calls no source line shows: those made through an
// interface.
type dispatch struct {
	// implements maps the key of a method of a concrete type to the keys of the
	// methods of the module interfaces it implements: a call through the
	// interface reaches it.
	implements map[string][]string
}

// analyse finds the interface dispatch of the whole module.
func analyse(m *archtest.TypeCheckedModule) *dispatch {
	d := &dispatch{implements: map[string][]string{}}
	var ifaces, concretes []*types.Named
	seen := map[*types.TypeName]bool{}
	for _, cp := range m.Packages {
		for _, info := range []*types.Info{cp.Info, cp.XTestInfo} {
			if info == nil {
				continue
			}
			for _, o := range info.Defs {
				if o == nil || o.Pkg() == nil || ownPath(o.Pkg().Path()) == "" {
					continue
				}
				tn, ok := o.(*types.TypeName)
				if !ok || tn.IsAlias() || seen[tn] {
					continue
				}
				seen[tn] = true
				if named, ok := tn.Type().(*types.Named); ok {
					if types.IsInterface(named) {
						ifaces = append(ifaces, named)
					} else {
						concretes = append(concretes, named)
					}
				}
			}
		}
	}

	// The methods of every concrete type, promoted ones too, by name.
	type method struct {
		typ *types.Named
		set *types.MethodSet
	}
	byName := map[string][]method{}
	for _, c := range concretes {
		set := types.NewMethodSet(types.NewPointer(c))
		for i := 0; i < set.Len(); i++ {
			byName[set.At(i).Obj().Name()] = append(byName[set.At(i).Obj().Name()], method{c, set})
		}
	}
	for _, in := range ifaces {
		u, _ := in.Underlying().(*types.Interface)
		if u == nil || u.NumMethods() == 0 {
			continue
		}
		rare := u.Method(0).Name()
		for i := 1; i < u.NumMethods(); i++ {
			if n := u.Method(i).Name(); len(byName[n]) < len(byName[rare]) {
				rare = n
			}
		}
		for _, c := range byName[rare] {
			if !types.Implements(types.NewPointer(c.typ), u) {
				continue
			}
			for i := 0; i < u.NumMethods(); i++ {
				im := u.Method(i)
				sel := c.set.Lookup(im.Pkg(), im.Name())
				if sel == nil {
					continue
				}
				ck := useKey(sel.Obj())
				if ck == "" || !concreteMethod(ck) {
					continue
				}
				d.implements[ck] = append(d.implements[ck], useKey(im))
			}
		}
	}

	return d
}

// defKeys are the keys a declared object answers to: a method of a concrete
// type also answers to the interface methods it implements and to the calls
// through an interface the source does not name.
func (d *dispatch) defKeys(o types.Object) []string {
	k := useKey(o)
	if k == "" {
		return nil
	}
	if f, ok := o.(*types.Func); ok && concreteMethod(k) {
		if sig, _ := f.Type().(*types.Signature); sig != nil && sig.Recv() != nil {
			return append([]string{k, "m\x00" + f.Name()}, d.implements[k]...)
		}
	}
	return []string{k}
}
