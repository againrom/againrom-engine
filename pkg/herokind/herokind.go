// Package herokind is the one chooser of a record by the starting hero's
// class and sex: the four kinds a hero is, the sets of kinds an alternative
// is chosen for, and the choice among alternatives.
package herokind

import (
	"fmt"
	"strings"
)

// Kind is a hero's class and sex in the selector order of the composed
// archetype rows: male fighter, female fighter, male mage, female mage, so
// female adds 1 and mage adds 2 (HERO-JOIN-120).
type Kind uint8

const (
	MaleFighter Kind = iota
	FemaleFighter
	MaleMage
	FemaleMage
	kinds
)

// Of is the kind of a hero of the class and sex given.
func Of(mage, female bool) Kind {
	k := MaleFighter
	if female {
		k++
	}
	if mage {
		k += 2
	}
	return k
}

// Index is the kind's place in the selector order, 0..3: the order the
// composed archetype rows, the base names and the persisted type ids share.
func (k Kind) Index() int { return int(k) }

// Order is a table's own order of the four kinds: entry i is the kind the
// table keeps at place i. A table that keeps the kinds in an order other than
// the selector order declares it as data and reads through In.
type Order [kinds]Kind

// PictureOrder is the order of the four hero pictures: male fighter, male
// mage, female fighter, female mage (the face and installed-name tables).
var PictureOrder = Order{MaleFighter, MaleMage, FemaleFighter, FemaleMage}

// In is the place of k in the order o, 0..3; a kind the order omits answers 0.
func (k Kind) In(o Order) int {
	for i, c := range o {
		if c == k {
			return i
		}
	}
	return 0
}

func (k Kind) Mage() bool   { return k&2 != 0 }
func (k Kind) Female() bool { return k&1 != 0 }

// String is the kind in words, "female mage".
func (k Kind) String() string {
	sex, class := Male, Fighter
	if k.Female() {
		sex = Female
	}
	if k.Mage() {
		class = Mage
	}
	return sex + " " + class
}

// The words a declaration names a class and a sex with.
const (
	Fighter = "fighter"
	Mage    = "mage"
	Male    = "male"
	Female  = "female"
)

// Set is a set of kinds: bit k holds kind k.
type Set uint8

// Every holds all four kinds; it is the set of an alternative that names
// neither class nor sex.
const Every Set = 1<<kinds - 1

// SetOf is the set of the kinds given.
func SetOf(ks ...Kind) Set {
	var s Set
	for _, k := range ks {
		s |= 1 << k
	}
	return s
}

// Parse is the set a class word and a sex word name: "fighter" or "mage",
// "male" or "female", an empty word leaving that axis open.
func Parse(class, sex string) (Set, error) {
	s := Every
	switch class {
	case "":
	case Fighter:
		s &= SetOf(MaleFighter, FemaleFighter)
	case Mage:
		s &= SetOf(MaleMage, FemaleMage)
	default:
		return 0, fmt.Errorf("class %q is neither %q nor %q", class, Fighter, Mage)
	}
	switch sex {
	case "":
	case Male:
		s &= SetOf(MaleFighter, MaleMage)
	case Female:
		s &= SetOf(FemaleFighter, FemaleMage)
	default:
		return 0, fmt.Errorf("sex %q is neither %q nor %q", sex, Male, Female)
	}
	return s, nil
}

// Named is the set Parse reads, or no hero for a word Parse refuses: a
// record whose words were checked when it was read names its set here.
func Named(class, sex string) Set {
	s, err := Parse(class, sex)
	if err != nil {
		return 0
	}
	return s
}

func (s Set) Holds(k Kind) bool   { return k < kinds && s&(1<<k) != 0 }
func (s Set) Overlaps(o Set) bool { return s&o != 0 }

// String names the set's heroes, "every hero", "a mage", "a female fighter"
// or "a male fighter or a female mage".
func (s Set) String() string {
	s &= Every
	switch s {
	case Every:
		return "every hero"
	case 0:
		return "no hero"
	}
	for _, axis := range []struct{ class, sex string }{
		{Fighter, ""}, {Mage, ""}, {"", Male}, {"", Female},
	} {
		if p, _ := Parse(axis.class, axis.sex); p == s {
			if axis.class != "" {
				return "a " + axis.class
			}
			return "a " + axis.sex + " hero"
		}
	}
	var parts []string
	for k := MaleFighter; k < kinds; k++ {
		if s.Holds(k) {
			parts = append(parts, "a "+k.String())
		}
	}
	return strings.Join(parts, " or ")
}

// Choose is the one choice by the starting hero: the index of the first of
// alternatives whose set holds k, else def. Alternatives that do not overlap
// (Overlap) answer the same whatever their order.
func Choose(k Kind, alternatives []Set, def int) int {
	for i, s := range alternatives {
		if s.Holds(k) {
			return i
		}
	}
	return def
}

// Overlap answers the first pair i < j of alternatives that one kind is in.
func Overlap(alternatives []Set) (i, j int, ok bool) {
	for j := range alternatives {
		for i := range alternatives[:j] {
			if alternatives[i].Overlaps(alternatives[j]) {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}
