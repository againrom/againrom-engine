package sav

import "slices"

// DocumentDeadRootLocation supplies structural starts only. An independent
// reader owns the raw u32 count and each archive tag; repeated roots retain
// separate tag positions. No deduplicated DeadActors projection supplies them.
type DocumentDeadRootLocation struct {
	CountOff int
	RefOffs  []int
}

func (f *File) DocumentDeadRootLocation() (DocumentDeadRootLocation, error) {
	doc, _, err := f.exactDocument()
	if err != nil {
		return DocumentDeadRootLocation{}, err
	}
	return DocumentDeadRootLocation{doc.deadRoots.CountOff, slices.Clone(doc.deadRoots.RefOffs)}, nil
}
