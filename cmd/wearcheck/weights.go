package main

import (
	"fmt"
	"io"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// weightSweep is one collection's answer.
type weightSweep struct {
	name string
	// rows is how many written rows the collection holds, and refused how
	// many of the triples ItemCodeWeight answered no weight for.
	rows, triples, refused int
	// min and max are the resolved per-unit weight's range over the triples
	// that did resolve; negative counts those below zero and zero those at it.
	min, max         int32
	negative, zeroes int
	// heaviest names the triple that produced max, so a reader can go back to
	// the install and look at it.
	heaviestRow, heaviestShape, heaviestMaterial int
}

// sweepWeights walks the three item collections against the shape and material
// ladders and answers one sweep each.
//
// THE CLASS FIELD IS THE COLLECTION'S OWN, not a value chosen here: a weapon
// code carries the weapon class, a shield code the shield class, and an armour
// code an equipment slot. The armour arm uses slot 4, which is an arbitrary
// wearable place — the weight arithmetic reads no slot at all, and the row's
// own Slot cell decides where the piece lands, so the field only has to be one
// ItemCodeWeight dispatches to the armour arm on.
func sweepWeights(t *mapload.Table) []weightSweep {
	if t == nil || t.Shapes == nil || t.Materials == nil {
		return nil
	}
	const (
		weaponClass = 1
		shieldClass = 2
		armourClass = 4
	)
	out := make([]weightSweep, 0, 3)
	for _, c := range []struct {
		name  string
		class int
		coll  data.Collection
	}{
		{"Weapons", weaponClass, t.Weapons},
		{"Shields", shieldClass, t.Shields},
		{"Armors", armourClass, t.Armors},
	} {
		s := weightSweep{name: c.name}
		if c.coll == nil {
			out = append(out, s)
			continue
		}
		s.rows = c.coll.Len()
		first := true
		for row := 0; row < c.coll.Len(); row++ {
			for shape := 0; shape < t.Shapes.Len() && shape < 8; shape++ {
				for material := 0; material < t.Materials.Len() && material < 16; material++ {
					s.triples++
					code := data.ComposeItemCode(material, c.class, shape, row)
					w, ok, err := data.ItemCodeWeight(code, t.Shapes, t.Materials,
						t.Armors, t.Shields, t.Weapons)
					if err != nil || !ok {
						s.refused++
						continue
					}
					switch {
					case w < 0:
						s.negative++
					case w == 0:
						s.zeroes++
					}
					if first || w < s.min {
						s.min = w
					}
					if first || w > s.max {
						s.max, s.heaviestRow, s.heaviestShape, s.heaviestMaterial = w, row, shape, material
					}
					first = false
				}
			}
		}
		out = append(out, s)
	}
	return out
}

// printWeightReport writes one sweep per collection.
func printWeightReport(out io.Writer, sweeps []weightSweep) error {
	if len(sweeps) == 0 {
		return fmt.Errorf("this install carries no shape or material ladder; nothing to scale a weight by")
	}
	fmt.Fprintln(out, "wearcheck -weights: the shipped item weight column, resolved over every shape and material")
	fmt.Fprintln(out)
	for _, s := range sweeps {
		fmt.Fprintf(out, "  %-8s %5d row(s), %6d code(s) swept, %6d refused\n",
			s.name, s.rows, s.triples, s.refused)
		if s.triples == s.refused {
			fmt.Fprintf(out, "           no code of this class resolves at all\n")
			continue
		}
		fmt.Fprintf(out, "           weight %d..%d, %d negative, %d zero\n",
			s.min, s.max, s.negative, s.zeroes)
		fmt.Fprintf(out, "           heaviest at row %d, shape %d, material %d\n",
			s.heaviestRow, s.heaviestShape, s.heaviestMaterial)
	}
	return nil
}
