package data

import (
	"fmt"
	"strconv"

	"againrom/pkg/formats/reg"
)

// Loading one registry into its classes, in three passes over the class
// sections.
//
// Stage 1 records, per section and per key-table row, own[i][k]: the *reg.Node
// THAT SECTION'S OWN body carries for row k, or nil. Nothing is inherited here,
// and the ID -> section index map is built alongside, so a duplicate ID is
// caught before any inheritance is resolved.
//
// Stage 2 fills eff[i][k], the node the resolved value is taken from, under the
// two guards this contract keeps apart. The difference between them is one
// word:
//
//	scalar  own[k] != nil -> the class's own node, a written 0 or "" included;
//	        else the parent's eff[k]. Reading the parent's RESOLVED row is what
//	        makes a scalar chain through ancestors, and the child's PRESENCE,
//	        never any value, is what decides.
//	array   own[k] of length > 0 stands; else the parent's OWN[k]. Reading the
//	        parent's own row is what holds an array to exactly one hop, so an
//	        array the parent itself inherited leaves the child nil. Length, not
//	        presence, is the whole test — which is why the empty-string sentinel
//	        falls through it and nothing in this format clears an array.
//
// Stage 3, and only then, fills the public structs from eff, and resolves each
// class's sprite base beside it — the one derived quantity a class carries, and
// the stage that has the RESOLVED File in hand (sprite.go). That order is the
// design and not an implementation detail: while inheritance is being decided no
// loaded field exists to test, so the wrong loader — one that inherits when a
// field is at its Go zero value, and so hands back 4 for a class that writes
// AttackDelay = 0 over a parent's 4 — is not merely wrong here, it is
// unwritable.
//
// A row whose eff is nil in every ancestor — the key is set NOWHERE on the chain
// — takes that key's absent-everywhere default, which is the type's zero only
// where the registry's table says so (keys.go). A key marked noInherit there
// never consults the parent at all, so a class whose own section omits it takes
// the default whatever an ancestor holds: objects.reg's File, and nothing else.
//
// One forward pass is sufficient because a Parent must name a class at a LOWER
// section index (spec.md, Inheritance), so by the time class i is resolved every
// class it can inherit from already is.
//
// Validation's three length rules — ShootOffset's 16, a structure's
// AnimMask against its geometry, and an animation pair that must resolve to
// one length — are evaluated at the end of stage 2, where a class's
// RESOLVED values are complete and its public struct does not yet exist
// (keys.go).

// --- the collections -------------------------------------------------------

// classSet is what a Load* entry point returns, one instantiation per registry:
// a list in build order — which is numeric section order, since that is the
// order the sections are looked up in — and a map into that same list.
//
// "A class is reachable by ByID if and only if it appears in All()" therefore
// holds because there is one list and one set of pointers, not because anything
// checks it.
type classSet[T any] struct {
	list  []*T
	index map[int32]*T
}

// All returns the registry's classes in numeric section order. The slice is a
// copy, so a caller reordering or truncating it cannot reach the collection; the
// classes themselves are shared.
func (s *classSet[T]) All() []*T {
	out := make([]*T, len(s.list))
	copy(out, s.list)
	return out
}

// ByID returns the class whose ID key holds id. The ID domain is sparse and is
// not the section index in any registry that can tell the two apart, so this is
// the only lookup a placed record or a Parent may use.
//
// id is int32, the type the .reg parser stores an integer at, so a caller
// converts where the sign stays visible.
func (s *classSet[T]) ByID(id int32) (*T, bool) {
	c, ok := s.index[id]
	return c, ok
}

// UnitClasses is the loaded units/units.reg.
type UnitClasses struct{ classSet[UnitClass] }

// ObjectClasses is the loaded objects/objects.reg.
type ObjectClasses struct{ classSet[ObjectClass] }

// StructureClasses is the loaded structures/structures.reg.
type StructureClasses struct{ classSet[StructureClass] }

// ByCode returns the class a map's object layer names by the byte b. 0 is NO
// OBJECT — not the class whose ID is 0 — and any other byte names the class
// whose ID is b-1. A byte naming no loaded class answers a miss, exactly as
// ByID does for an ID no class holds.
//
// This is the one place the offset between a placement byte and an identity is
// written, and it sits on ObjectClasses rather than on the embedded classSet
// because the byte is the object layer's own encoding: neither of the other two
// registries is reached by one, and a twin on either would be a second
// convention with nothing to name it.
//
// The widening happens before the subtraction, at the type ByID takes, so the
// one byte the offset cannot be applied to is the one the guard has already
// answered — and no case of this lookup is left to unsigned wraparound.
func (c *ObjectClasses) ByCode(b byte) (*ObjectClass, bool) {
	if b == 0 {
		return nil, false
	}
	return c.ByID(int32(b) - 1)
}

// --- the entry points ------------------------------------------------------
//
// Each takes a registry already parsed by pkg/formats/reg and performs no IO of
// any kind: a load is a pure function of the tree. A malformed registry yields a
// NIL collection and an error naming the offending section and its key — never a
// partial collection, never a panic.

// LoadUnitClasses resolves a parsed units/units.reg into its classes.
func LoadUnitClasses(r *reg.Reg) (*UnitClasses, error) {
	set, err := loadSet(r, unitKeys, unitDesc, unitLengths, unitDefaults,
		func(c *UnitClass, base string) { c.base = base })
	if err != nil {
		return nil, err
	}
	return &UnitClasses{set}, nil
}

// LoadObjectClasses resolves a parsed objects/objects.reg into its classes.
func LoadObjectClasses(r *reg.Reg) (*ObjectClasses, error) {
	set, err := loadSet(r, objectKeys, objectDesc, objectLengths, objectDefaults,
		func(c *ObjectClass, base string) { c.base = base })
	if err != nil {
		return nil, err
	}
	return &ObjectClasses{set}, nil
}

// LoadStructureClasses resolves a parsed structures/structures.reg into its
// classes. This registry does not inherit: a Parent on a structure is malformed,
// not ignored.
func LoadStructureClasses(r *reg.Reg) (*StructureClasses, error) {
	set, err := loadSet(r, structureKeys, structureDesc, structureLengths, structureDefaults,
		func(c *StructureClass, base string) { c.base = base })
	if err != nil {
		return nil, err
	}
	return &StructureClasses{set}, nil
}

// --- the loader ------------------------------------------------------------

// loadSet is the whole of the three passes, over one registry's key table and
// descriptor. Every failure path is `return none, err`, and the collection is
// built by the final statement, so atomicity needs no mechanism of its own.
//
// setBase stores a class's resolved sprite base. It is a parameter rather than a
// key table row because base is not a key: the row types are what the reflect
// bijection walks, and a field that is not an inventory key must not appear
// there — nor be exported for this generic function to reach it.
func loadSet[T any](r *reg.Reg, rows []keyRow[T], d descriptor, lens lengths, defs []scalarDefault,
	setBase func(*T, string),
) (classSet[T], error) {
	var none classSet[T]

	count, err := globalCount(r, d)
	if err != nil {
		return none, err
	}

	sprites, err := newSpriteEnv(r, d)
	if err != nil {
		return none, err
	}

	checks := lengthChecks(rows, lens)
	defaults, noInherit := defaultsFor(rows, defs)

	idRow := rowIndex(rows, "ID")
	parentRow := rowIndex(rows, "Parent")
	fileRow := rowIndex(rows, "File")

	// --- Stage 1: each section's own nodes, and the ID -> index map ---------
	//
	// Sections are addressed by CONSTRUCTED name — prefix + the index — and the
	// node table is never walked: build order is therefore numeric order with no
	// sort, and a missing dense section is a lookup miss reported at the index
	// where it happens rather than a class quietly absent from the list.
	//
	// Nothing is sized from count. It is a value read from the stream, and make
	// with a negative capacity panics.
	var names []string
	var owns [][]*reg.Node
	byID := make(map[int32]int)
	for i := 0; i < count; i++ {
		name := d.sectionPrefix + strconv.Itoa(i)
		sec := findSection(r, name)
		if sec == nil {
			return none, fmt.Errorf("%s: section missing", name)
		}

		// Parent is a key table row only where it is legal, so in a registry
		// that does not inherit it would otherwise be ignored as an unknown key.
		// It is malformed there, and only the section can say so.
		if !d.parentLegal && findChild(sec, "Parent") != nil {
			return none, fmt.Errorf("%s: Parent: this registry does not inherit", name)
		}

		own := make([]*reg.Node, len(rows))
		for k, row := range rows {
			node, state := readKey(sec, row.name, row.kind)
			switch state {
			case keyWrongKind:
				return none, fmt.Errorf("%s: %s: wrong kind, want %s", name, row.name, row.kind)
			case keyPresent:
				own[k] = node
			}
		}

		// The identity is the class's own ID node: it is what a Parent and a
		// placed record name, and reading it here is what lets stage 2 tell an
		// unresolvable Parent from a forward one. Every class of every shipped
		// registry carries the key; a section without one is not a case
		// Validation names, and is left unreachable by ByID rather than rejected
		// on a rule the contract does not state.
		if idRow >= 0 && own[idRow] != nil {
			id := own[idRow].Int
			if prev, dup := byID[id]; dup {
				return none, fmt.Errorf("%s: ID: %d is already the ID of %s", name, id, names[prev])
			}
			byID[id] = i
		}

		names = append(names, name)
		owns = append(owns, own)
	}

	// --- Stage 2: the resolved node per key, under the two guards -----------
	effs := make([][]*reg.Node, len(owns))
	for i, own := range owns {
		// A present Parent is resolved through byID. No entry is unresolvable;
		// an entry at index >= i is forward, and since every cycle needs at
		// least one edge to an index not below the child's — a self-reference
		// included — every cycle IS a forward edge and is rejected as one. No
		// cycle detector exists, and nothing here compares a Parent to zero:
		// Parent = 0 is an ordinary hit on the class whose ID is 0.
		parent := -1
		if parentRow >= 0 && own[parentRow] != nil {
			pid := own[parentRow].Int
			p, ok := byID[pid]
			if !ok {
				return none, fmt.Errorf("%s: Parent: no class has ID %d", names[i], pid)
			}
			if p >= i {
				return none, fmt.Errorf("%s: Parent: ID %d is %s, which is not below this section",
					names[i], pid, names[p])
			}
			parent = p
		}

		eff := make([]*reg.Node, len(rows))
		for k, row := range rows {
			if row.kind == kindArray {
				switch {
				case arrayLen(own[k]) > 0:
					eff[k] = own[k]
				case parent >= 0 && arrayLen(owns[parent][k]) > 0:
					eff[k] = owns[parent][k] // the parent's OWN row: one hop
				}
				continue
			}
			switch {
			case own[k] != nil:
				eff[k] = own[k]
			case parent >= 0 && !noInherit[k]:
				eff[k] = effs[parent][k] // the parent's RESOLVED row: chaining
			}
			// A noInherit row never reads the parent, so eff[k] stays nil
			// however deep the chain runs and stage 3 gives the class the key's
			// default instead of an ancestor's value. objects.reg's File is the
			// only such key: its default is a literal, not the parent's field.
		}
		effs[i] = eff

		// The length rules, on this class's resolved values. They run here rather
		// than beside the struct that is filled from them because a length is a
		// property of the resolution: an animation pair written on one key and
		// inherited on the other mismatches even though each node it came from is
		// well-formed.
		for _, check := range checks {
			if err := check(names[i], eff); err != nil {
				return none, err
			}
		}
	}

	// --- Stage 3: the public structs, once every class is resolved ----------
	list := make([]*T, len(owns))
	for i, eff := range effs {
		c := new(T)
		for k, row := range rows {
			n := eff[k]
			if n == nil {
				// The key is set by NO section on this class's chain, so the
				// field takes that key's absent-everywhere default — the type's
				// zero only where the registry's table says so (keys.go). A
				// registry with no table defaults every row to 0, which is what
				// the fill did unconditionally before.
				if row.kind == kindInt {
					*(row.ptr(c).(*int32)) = defaults[k]
				}
				continue
			}
			// The assertions cannot fail: keys_test.go pins every row's pointer
			// to a field of this struct at this kind's Go type.
			switch row.kind {
			case kindInt:
				*(row.ptr(c).(*int32)) = n.Int
			case kindStr:
				*(row.ptr(c).(*string)) = n.Str
			case kindArray:
				// Copied, never aliased: Node.Ints is the parser's own memory,
				// and one parent's array reaches every child that inherits it.
				*(row.ptr(c).(*[]int32)) = append([]int32(nil), n.Ints...)
			}
		}

		// The sprite base, once per class and here because this is where the
		// class's RESOLVED File sits: a class that inherits File resolves the
		// ancestor's index, and File's bound and the [Files] entry behind it are
		// checked nowhere else.
		var file *reg.Node
		if fileRow >= 0 {
			file = eff[fileRow]
		}
		base, err := sprites.base(names[i], file)
		if err != nil {
			return none, err
		}
		setBase(c, base)

		list[i] = c
	}

	index := make(map[int32]*T, len(byID))
	for id, i := range byID {
		index[id] = list[i]
	}
	return classSet[T]{list: list, index: index}, nil
}

// globalCount reads the class count from [Global]. A missing [Global] and a
// missing count key are the same answer — the count is not there — and a count
// at another value type is the other.
func globalCount(r *reg.Reg, d descriptor) (int, error) {
	n, state := readKey(findSection(r, "Global"), d.countKey, kindInt)
	switch state {
	case keyPresent:
		return int(n.Int), nil
	case keyWrongKind:
		return 0, fmt.Errorf("Global: %s: wrong kind, want int", d.countKey)
	}
	return 0, fmt.Errorf("Global: %s: missing", d.countKey)
}

// rowIndex returns the index of the row named name, or -1. The two rows the
// loader must reach by name are ID, the identity, and Parent, the edge — and
// Parent is absent from the registry that does not inherit.
func rowIndex[T any](rows []keyRow[T], name string) int {
	for i, row := range rows {
		if row.name == name {
			return i
		}
	}
	return -1
}

// arrayLen is the array guard's whole test. A nil node is length 0, and so is
// the empty-string sentinel, whose Ints is nil — which is what makes an
// explicitly emptied array inherit rather than clear.
func arrayLen(n *reg.Node) int {
	if n == nil {
		return 0
	}
	return len(n.Ints)
}

// strOf and intOf read a resolved node's value, an unset key answering the Go
// zero the class's field would carry.
func strOf(n *reg.Node) string {
	if n == nil {
		return ""
	}
	return n.Str
}

func intOf(n *reg.Node) int32 {
	if n == nil {
		return 0
	}
	return n.Int
}

// defaultsFor resolves one registry's default rules (keys.go) against its key
// table, once per load, into two values per row: what that row's field takes when
// the key resolves to no node at all, and whether the row consults the parent at
// all. A row with no rule answers 0 and false, so a registry with no table
// behaves exactly as the unconditional zero and unconditional inheritance did.
//
// A rule naming a key this table does not carry, or carrying it at another kind,
// yields nothing — the same silence the length rules keep, and pinned the same
// way, by a test that holds every row of a registry's own table to a kindInt row
// of it. Only int keys have defaults: a str resolving nowhere is "" and an array
// nil, and neither is a value the registry states.
func defaultsFor[T any](rows []keyRow[T], rules []scalarDefault) (values []int32, noInherit []bool) {
	values = make([]int32, len(rows))
	noInherit = make([]bool, len(rows))
	for _, rule := range rules {
		if k := rowIndex(rows, rule.key); k >= 0 && rows[k].kind == kindInt {
			values[k] = rule.value
			noInherit[k] = rule.noInherit
		}
	}
	return values, noInherit
}

// lengthChecks resolves one registry's length rules (keys.go) against its key
// table, once per load, into one closure per rule over a class's resolved nodes.
// Resolving the names here is what keeps the rules readable as key names while
// the loader works in row indices.
//
// A rule naming a key this table does not carry yields no closure. That is how a
// rule stays silent over a registry whose inventory does not have the key at
// all; that no rule of a registry's OWN set misses its table is pinned by
// validate_test.go, so a misspelled rule cannot pass as an inapplicable one.
func lengthChecks[T any](rows []keyRow[T], lens lengths) []func(section string, eff []*reg.Node) error {
	var out []func(string, []*reg.Node) error

	for _, rule := range lens.fixed {
		k := rowIndex(rows, rule.key)
		if k < 0 {
			continue
		}
		out = append(out, func(section string, eff []*reg.Node) error {
			// Length 0 is the key resolving to nothing, which is legal: some
			// classes omit it with no ancestor to take it from, and the contract
			// names only a value of the wrong length.
			if n := arrayLen(eff[k]); n != 0 && n != rule.n {
				return fmt.Errorf("%s: %s: length %d, want %d", section, rule.key, n, rule.n)
			}
			return nil
		})
	}

	for _, rule := range lens.pairs {
		a, b := rowIndex(rows, rule.time), rowIndex(rows, rule.frame)
		if a < 0 || b < 0 {
			continue
		}
		out = append(out, func(section string, eff []*reg.Node) error {
			// The pair is named from its first key so one section reports one
			// key for the rule whichever half is short — neither is the offender
			// on its own, and both lengths are what the reader needs.
			if la, lb := arrayLen(eff[a]), arrayLen(eff[b]); la != lb {
				return fmt.Errorf("%s: %s: length %d, but %s is length %d",
					section, rule.time, la, rule.frame, lb)
			}
			return nil
		})
	}

	if m := lens.mask; m.key != "" {
		k, w, h := rowIndex(rows, m.key), rowIndex(rows, m.width), rowIndex(rows, m.height)
		if k >= 0 && w >= 0 && h >= 0 {
			out = append(out, func(section string, eff []*reg.Node) error {
				mask := strOf(eff[k])
				if mask == "" {
					return nil // an empty mask is 52 of the 66 shipped structures
				}
				want := int64(intOf(eff[w])) * int64(intOf(eff[h]))
				if int64(len(mask)) != want {
					return fmt.Errorf("%s: %s: length %d, want %s * %s = %d",
						section, m.key, len(mask), m.width, m.height, want)
				}
				return nil
			})
		}
	}

	return out
}
