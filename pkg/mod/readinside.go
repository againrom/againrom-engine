package mod

// MaxFileBytes bounds a file a mod's script, data or picture is read from.
const MaxFileBytes = 1 << 20

// ReadInside reads a regular file within the mod at dir, its folder or its
// RES file, returning its relative slash path; see Tree.ReadInside.
func ReadInside(dir, name string) ([]byte, string, error) {
	t, err := OpenTree(dir)
	if err != nil {
		return nil, "", err
	}
	return t.ReadInside(name)
}
