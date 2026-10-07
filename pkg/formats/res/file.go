package res

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// FileArchive is a read-only archive index backed by a host file. It retains
// only the registry and path index; each ReadFile opens and reads one payload.
// It is suitable for large optional archives whose complete bytes must not
// remain resident.
type FileArchive struct {
	path    string
	entries []Entry
	index   map[string]int
}

// OpenFileIndex reads only the fixed header and registry from path.
func OpenFileIndex(path string) (*FileArchive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	total := info.Size()
	if total < headerSize {
		return nil, fmt.Errorf("res: archive too small: %d bytes", total)
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	if sig := binary.LittleEndian.Uint32(header[0x00:0x04]); sig != signature {
		return nil, fmt.Errorf("res: bad signature %#08x", sig)
	}
	regOffset := int64(binary.LittleEndian.Uint32(header[0x10:0x14]))
	headerNodeCount := int64(binary.LittleEndian.Uint32(header[0x14:0x18]))
	if regOffset < headerSize || regOffset > total {
		return nil, fmt.Errorf("res: regOffset %d outside [%d, %d]", regOffset, headerSize, total)
	}
	nodeBytes := headerNodeCount * nodeSize
	if regOffset+nodeBytes > total {
		return nil, fmt.Errorf("res: registry of %d nodes at %d ends at %d, past the archive's %d bytes",
			headerNodeCount, regOffset, regOffset+nodeBytes, total)
	}
	records := make([]byte, int(nodeBytes))
	if n, err := f.ReadAt(records, regOffset); err != nil || int64(n) != nodeBytes {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	nodes, err := parseNodeRecords(records, regOffset, total, int(headerNodeCount))
	if err != nil {
		return nil, err
	}
	entries, index, err := indexNodes(nodes)
	if err != nil {
		return nil, err
	}
	return &FileArchive{path: path, entries: entries, index: index}, nil
}

func (a *FileArchive) Entries() []Entry {
	if a == nil {
		return nil
	}
	out := make([]Entry, len(a.entries))
	copy(out, a.entries)
	return out
}

func (a *FileArchive) ReadFile(name string) ([]byte, error) {
	if a == nil {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	i, ok := a.index[normalize(name)]
	if !ok {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	e := a.entries[i]
	f, err := os.Open(a.path)
	if err != nil {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: err}
	}
	defer f.Close()
	out := make([]byte, int(e.Size))
	n, err := f.ReadAt(out, e.Offset)
	if err != nil && err != io.EOF {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: err}
	}
	if int64(n) != e.Size {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: io.ErrUnexpectedEOF}
	}
	return out, nil
}
