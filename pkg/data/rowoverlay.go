package data

// OverlayRow is one row of a RowOverlay: a row the base collection does not
// hold, or a replacement for one it does.
type OverlayRow struct {
	Name   string
	Params []int32
	// Raw is the row's record bytes: the ten-byte block of five material masks
	// for an armour, shield or weapon row. A nil Raw keeps the base row's.
	Raw []byte
}

// RowOverlay is a Collection that answers rows of a base collection except
// where an overlay row stands in. An overlay row past the end of the base
// extends it; the indices between the end and the overlay row answer as
// entries nothing was written to.
//
// It is also a MaskTable, so the shop pool reads the material masks of the
// overlay rows and of the base rows alike. The base is never changed.
type RowOverlay struct {
	base Collection
	rows map[int]OverlayRow
	n    int
}

// NewRowOverlay lays rows over base. Negative indices and the reserved index 0
// are ignored: row 0 of a one-based collection stays empty.
func NewRowOverlay(base Collection, rows map[int]OverlayRow) *RowOverlay {
	o := &RowOverlay{base: base, rows: map[int]OverlayRow{}}
	if base != nil {
		o.n = base.Len()
	}
	for i, r := range rows {
		if i < 1 {
			continue
		}
		o.rows[i] = OverlayRow{Name: r.Name, Params: append([]int32(nil), r.Params...), Raw: append([]byte(nil), r.Raw...)}
		if i+1 > o.n {
			o.n = i + 1
		}
	}
	return o
}

// Len is the base length, or one past the last overlay row.
func (o *RowOverlay) Len() int { return o.n }

func (o *RowOverlay) inBase(i int) bool { return o.base != nil && i >= 0 && i < o.base.Len() }

// EntryName is the overlay row's name, else the base row's, else empty.
func (o *RowOverlay) EntryName(i int) string {
	if r, ok := o.rows[i]; ok {
		return r.Name
	}
	if o.inBase(i) {
		return o.base.EntryName(i)
	}
	return ""
}

// EntryParams is the overlay row's parameters, else the base row's, else nil.
func (o *RowOverlay) EntryParams(i int) []int32 {
	if r, ok := o.rows[i]; ok {
		return r.Params
	}
	if o.inBase(i) {
		return o.base.EntryParams(i)
	}
	return nil
}

// EntryStrings is the base row's trailing strings; an overlay row past the base
// carries none.
func (o *RowOverlay) EntryStrings(i int) []string {
	if o.inBase(i) {
		return o.base.EntryStrings(i)
	}
	return nil
}

// EntryRaw is the overlay row's record bytes when it gives some, else the base
// row's when the base is a MaskTable, else nil.
func (o *RowOverlay) EntryRaw(i int) []byte {
	if r, ok := o.rows[i]; ok && len(r.Raw) != 0 {
		return append([]byte(nil), r.Raw...)
	}
	if m, ok := o.base.(MaskTable); ok && o.inBase(i) {
		return m.EntryRaw(i)
	}
	return nil
}

var _ MaskTable = (*RowOverlay)(nil)
