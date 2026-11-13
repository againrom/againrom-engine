package shopstock

// StackKind is a row of the stack table: a potion's, a scroll row's, or any
// other kept unit's that stacks.
type StackKind uint8

const (
	PotionStack StackKind = iota
	ScrollStack
	UnitStack
)

// StackRow is the quantity range Least..Least+Span-1 a stack is drawn in, on
// one number. A Bounded row's span grows by the category's draw bound; a row
// drawn Once takes nothing from an equal pick of the fill.
type StackRow struct {
	Least, Span   int32
	Bounded, Once bool
}

// stackTable is the one stack table of both games' shops: each potion at
// 51..100 per pick (SHOP-GEN-005, R2-ENGINE-265), a scroll row at 51..100
// drawn once (owner, DIV-4870), any other stacking unit at 1..Bound+1 per
// pick (R2-ENGINE-941). The first game stocks only the potion and scroll
// rows.
var stackTable = [...]StackRow{
	PotionStack: {Least: 51, Span: 50},
	ScrollStack: {Least: 51, Span: 50, Once: true},
	UnitStack:   {Least: 1, Span: 1, Bounded: true},
}

// Row is k's row of the stack table.
func (k StackKind) Row() StackRow { return stackTable[k] }

// Draw is a quantity of k's row on one number of src. A Bounded row adds
// bound, the category's draw bound, to its span in 32 bits, as the record
// holds it.
func (k StackKind) Draw(src Source, bound int32) int32 {
	q := stackTable[k]
	span := q.Span
	if q.Bounded {
		span += bound
	}
	return q.Least + int32(src.Intn(int(span)))
}

// Filling is s as a fill joins a pick to an equal entry: a pick whose stack
// row is drawn once adds nothing; its quantity number was drawn all the
// same. row is a game's stack row of an item, false for one with none.
func Filling[I any, K comparable](s Stack[I, K], row func(I) (StackKind, bool)) Stack[I, K] {
	add := s.Add
	s.Add = func(entry, it I) I {
		if k, ok := row(it); ok && stackTable[k].Once {
			return entry
		}
		return add(entry, it)
	}
	return s
}
