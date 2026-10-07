package sav

import "sort"

// DocumentObjectLocation locates a tagged object's body for an independent
// byte reader. It deliberately exposes no values, counts, references or end
// offsets. ArchiveIndex is local to this original stream, not a native ID.
type DocumentObjectLocation struct {
	ArchiveIndex uint16
	Class        string
	Off          int
}

// DocumentObjectLocations returns the exact envelope walk's object starts in
// archive order. Inline records have no archive index and are not included.
func (f *File) DocumentObjectLocations() ([]DocumentObjectLocation, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return nil, err
	}
	out := make([]DocumentObjectLocation, 0, len(doc.objects))
	for index, record := range doc.objects {
		out = append(out, DocumentObjectLocation{index, record.Class, record.Off})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ArchiveIndex < out[j].ArchiveIndex })
	return out, nil
}
