package data

// RowEdit changes cells of one row of a collection: parameter slots, and
// trailing strings by position. An absent entry leaves the cell as it is.
type RowEdit struct {
	Params map[int]int32
	Cells  map[int]string
}

// EditedRows is a Collection that answers the rows of a base collection with
// the edits of some rows applied. It never changes the base, keeps the length,
// the names and every row without an edit as they are, and so serves any reader
// of the base without a change.
type EditedRows struct {
	base    Collection
	params  map[int][]int32
	strings map[int][]string
}

// NewEditedRows applies edits, keyed by row index, over base. An index outside
// the base, a negative slot and a parameter slot past the row's width are
// ignored; a string position past the row's strings extends them with empty
// strings.
func NewEditedRows(base Collection, edits map[int]RowEdit) *EditedRows {
	e := &EditedRows{base: base, params: map[int][]int32{}, strings: map[int][]string{}}
	if base == nil {
		return e
	}
	for i, edit := range edits {
		if i < 1 || i >= base.Len() {
			continue
		}
		if len(edit.Params) != 0 {
			p := append([]int32(nil), base.EntryParams(i)...)
			for slot, v := range edit.Params {
				if slot >= 0 && slot < len(p) {
					p[slot] = v
				}
			}
			e.params[i] = p
		}
		if len(edit.Cells) != 0 {
			s := append([]string(nil), base.EntryStrings(i)...)
			for pos, v := range edit.Cells {
				if pos < 0 {
					continue
				}
				for len(s) <= pos {
					s = append(s, "")
				}
				s[pos] = v
			}
			e.strings[i] = s
		}
	}
	return e
}

// Len is the base length.
func (e *EditedRows) Len() int {
	if e.base == nil {
		return 0
	}
	return e.base.Len()
}

// EntryName is the base row's name.
func (e *EditedRows) EntryName(i int) string {
	if e.base == nil {
		return ""
	}
	return e.base.EntryName(i)
}

// EntryParams is the edited parameters of row i, else the base row's.
func (e *EditedRows) EntryParams(i int) []int32 {
	if p, ok := e.params[i]; ok {
		return p
	}
	if e.base == nil {
		return nil
	}
	return e.base.EntryParams(i)
}

// EntryStrings is the edited strings of row i, else the base row's.
func (e *EditedRows) EntryStrings(i int) []string {
	if s, ok := e.strings[i]; ok {
		return s
	}
	if e.base == nil {
		return nil
	}
	return e.base.EntryStrings(i)
}

var _ Collection = (*EditedRows)(nil)

// The Humans parameter slots that name a person's class and figure, and the
// positions of the ten equipment cells a Humans row carries after its
// parameters: the weapon, the shield, then eight armour and jewellery cells.
const (
	HumanSlotTypeID = 16
	HumanSlotFace   = 17
	HumanSlotGender = 18

	HumanCellWeapon    = 0
	HumanCellShield    = 1
	HumanCellArmour    = 2
	HumanCellCount     = 10
	HumanSlotsRequired = HumanSlotGender + 1
)
