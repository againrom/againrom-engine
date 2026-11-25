package game

import (
	"os"
	"path/filepath"
	"slices"

	"againrom/pkg/namefold"
)

// profileListing is one listing of a folder of files: the folder, the file
// pattern, and how one file of the pattern reads into a row of type T. The
// saves (saveScan) and the characters (characterStoreRow) are its rows.
type profileListing[T any] struct {
	// dir is the folder; empty lists nothing.
	dir string
	// exts are the extensions listed, in any letter case.
	exts []string
	// regular lists regular files only; otherwise every non-directory.
	regular bool
	// read answers a file's row. False with no error passes over a file the
	// listing does not keep; an error counts a file that will not read.
	read func(dir string, e os.DirEntry) (T, bool, error)
	// order sorts the rows. first stops at the first row and sorts nothing.
	order func(a, b T) int
	first bool
}

// matches reports whether name has one of the listing's extensions.
func (l profileListing[T]) matches(name string) bool {
	ext := filepath.Ext(name)
	return slices.ContainsFunc(l.exts, func(e string) bool { return namefold.HostEqual(ext, e) })
}

// listProfileFiles is the one lister of a folder of profile or install
// files: the rows of l's files in l's order and the count of files that
// would not read. An absent folder lists nothing; a folder that will not
// read is the error. It writes nothing.
func listProfileFiles[T any](l profileListing[T]) ([]T, int, error) {
	if l.dir == "" {
		return nil, 0, nil
	}
	entries, err := os.ReadDir(l.dir)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var out []T
	left := 0
	for _, e := range entries {
		if e.IsDir() || l.regular && !e.Type().IsRegular() || !l.matches(e.Name()) {
			continue
		}
		row, ok, err := l.read(l.dir, e)
		if err != nil {
			left++
			continue
		}
		if !ok {
			continue
		}
		out = append(out, row)
		if l.first {
			return out, left, nil
		}
	}
	slices.SortStableFunc(out, l.order)
	return out, left, nil
}
