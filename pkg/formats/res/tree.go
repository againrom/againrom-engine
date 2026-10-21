package res

import (
	"fmt"
	"io/fs"
)

// Node is one registry node as the archive stores it. Name is the node's
// own name, decoded as the index decodes it, with its case kept. A file's
// Offset and Size delimit its payload; a directory's Children are the
// indices of its child nodes, in registry order.
type Node struct {
	Name     string
	Dir      bool
	Offset   int64
	Size     int64
	Children []int
}

// treeOf turns the validated registry into Nodes.
func treeOf(nodes []node) []Node {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = Node{Name: n.name, Dir: n.typ == typeDir}
		if n.typ == typeDir {
			for c := int(n.off); c < int(n.off)+int(n.size); c++ {
				out[i].Children = append(out[i].Children, c)
			}
			continue
		}
		out[i].Offset, out[i].Size = int64(n.off), int64(n.size)
	}
	return out
}

// Tree returns every node in registry order and the indices of the
// top-level nodes. Unlike Entries it keeps directories, empty ones too, and
// each name as stored.
func (a *Archive) Tree() ([]Node, []int) {
	return treeOf(a.nodes), append([]int(nil), a.roots...)
}

// ReadNode returns a copy of the payload of file node i.
func (a *Archive) ReadNode(i int) ([]byte, error) {
	if i < 0 || i >= len(a.nodes) || a.nodes[i].typ != typeFile {
		return nil, &fs.PathError{Op: "readnode", Path: fmt.Sprint(i), Err: fs.ErrInvalid}
	}
	n := a.nodes[i]
	out := make([]byte, n.size)
	copy(out, a.data[int64(n.off):int64(n.off)+int64(n.size)])
	return out, nil
}

// Tree returns every node in registry order and the indices of the
// top-level nodes; see Archive.Tree.
func (a *FileArchive) Tree() ([]Node, []int) {
	return treeOf(a.nodes), append([]int(nil), a.roots...)
}

// ReadNode returns the payload of file node i, read from the file.
func (a *FileArchive) ReadNode(i int) ([]byte, error) {
	if i < 0 || i >= len(a.nodes) || a.nodes[i].typ != typeFile {
		return nil, &fs.PathError{Op: "readnode", Path: fmt.Sprint(i), Err: fs.ErrInvalid}
	}
	return a.readRange(a.nodes[i].name, int64(a.nodes[i].off), int64(a.nodes[i].size))
}
