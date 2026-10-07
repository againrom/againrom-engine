package reg

import (
	"encoding/binary"
	"fmt"
)

// SetInt overwrites a TypeInt leaf's stored value: it returns a new slice
// equal to data except for that one node's own 4-byte data word. data must
// be the exact bytes r was parsed from — r's node positions are byte offsets
// into it — and neither data nor r is mutated.
//
// This is the same kind of record patch setPlayer applies to a Player class
// field (pkg/formats/sav): the value lives directly in the record and never
// touches the heap, so nothing else in the stream can move.
func SetInt(data []byte, r *Reg, section, key string, v int32) ([]byte, error) {
	n, err := lookupValue(r, section, key, TypeInt)
	if err != nil {
		return nil, err
	}
	off := headerSize + nodeSize*n.index
	if off+nodeSize > len(data) {
		return nil, fmt.Errorf("reg: node %d record at %d overruns a %d-byte stream", n.index, off, len(data))
	}
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(out[off+4:off+8], uint32(v))
	return out, nil
}

// SetIntArray replaces a TypeIntArray leaf's whole value. The replacement's
// byte length need not equal the old one's, so — unlike SetInt — this can
// also rewrite the heap-offset word of every OTHER String/IntArray node whose
// own span sits at or after the replaced one's in the heap. Only that word
// moves for such a node: its own Str/Ints value, decoded from the new bytes
// at its new position, is unchanged. Node count, the node table's order, and
// every directory's own child range are untouched, since no node is added or
// removed.
//
// This mirrors Head.MapName in pkg/formats/sav (sav.File.SetMapName): a
// length change shifts what follows it, bounded to position words rather
// than the values they locate.
func SetIntArray(data []byte, r *Reg, section, key string, values []int32) ([]byte, error) {
	n, err := lookupValue(r, section, key, TypeIntArray)
	if err != nil {
		return nil, err
	}
	if len(data) < headerSize {
		return nil, fmt.Errorf("reg: stream too small: %d bytes", len(data))
	}
	nodeCount := int(binary.LittleEndian.Uint32(data[0x10:0x14]))
	heapOrigin := headerSize + nodeSize*nodeCount
	if heapOrigin+4 > len(data) {
		return nil, fmt.Errorf("reg: node table of %d nodes needs %d bytes, stream has %d", nodeCount, heapOrigin+4, len(data))
	}
	oldHeapSize := int(binary.LittleEndian.Uint32(data[heapOrigin : heapOrigin+4]))
	heapStart := heapOrigin + 4
	if heapStart+oldHeapSize > len(data) {
		return nil, fmt.Errorf("reg: heap [%d, %d) past the end of a %d-byte stream", heapStart, heapStart+oldHeapSize, len(data))
	}
	oldOff, oldSize := int(n.rawD), int(n.rawSz)
	if oldOff < 0 || oldSize < 0 || oldOff+oldSize > oldHeapSize {
		return nil, fmt.Errorf("reg: node %d heap span [%d, %d) outside a %d-byte heap", n.index, oldOff, oldOff+oldSize, oldHeapSize)
	}

	newBytes := make([]byte, 4*len(values))
	for i, v := range values {
		binary.LittleEndian.PutUint32(newBytes[4*i:], uint32(v))
	}
	delta := len(newBytes) - oldSize

	// The node table (header included) is copied whole, then exactly two
	// kinds of word are patched in place: the target's own size word, and
	// the offset word of every later heap-resident node.
	table := append([]byte(nil), data[:heapOrigin]...)
	tOff := headerSize + nodeSize*n.index
	binary.LittleEndian.PutUint32(table[tOff+8:tOff+12], uint32(len(newBytes)))
	for i := 0; i < nodeCount; i++ {
		if i == n.index {
			continue
		}
		rec := headerSize + nodeSize*i
		kind := binary.LittleEndian.Uint32(data[rec+0x0c : rec+0x10])
		if kind&flagDir != 0 {
			continue
		}
		typ := ValueType(kind & typeMask)
		if typ != TypeString && typ != TypeIntArray {
			continue
		}
		d := int(binary.LittleEndian.Uint32(data[rec+4 : rec+8]))
		if d >= oldOff+oldSize {
			binary.LittleEndian.PutUint32(table[rec+4:rec+8], uint32(d+delta))
		}
	}

	heap := data[heapStart : heapStart+oldHeapSize]
	newHeap := make([]byte, 0, oldHeapSize+delta)
	newHeap = append(newHeap, heap[:oldOff]...)
	newHeap = append(newHeap, newBytes...)
	newHeap = append(newHeap, heap[oldOff+oldSize:]...)

	out := make([]byte, 0, len(table)+4+len(newHeap))
	out = append(out, table...)
	var lenWord [4]byte
	binary.LittleEndian.PutUint32(lenWord[:], uint32(len(newHeap)))
	out = append(out, lenWord[:]...)
	out = append(out, newHeap...)
	return out, nil
}

// lookupValue resolves (section, key) and reports a bounded error for a
// missing node or one of the wrong type, so SetInt and SetIntArray never
// patch a node their caller did not name.
func lookupValue(r *Reg, section, key string, want ValueType) (*Node, error) {
	if r == nil {
		return nil, fmt.Errorf("reg: nil registry")
	}
	n, ok := r.lookup(section, key)
	if !ok {
		return nil, fmt.Errorf("reg: no %s/%s node", section, key)
	}
	if n.Dir || n.Type != want {
		return nil, fmt.Errorf("reg: %s/%s is not type %d", section, key, want)
	}
	return n, nil
}
