// Package savefixture holds asset-free, frozen saves from published producers.
package savefixture

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"io"
	"strings"
)

//go:embed testdata/form75.gz.base64
var genuine75Packed string

// Genuine75 is the exact World.MarshalBinary output of implementation
// 1fdf689586edcfcb8b015fdec5f7c55566f831ad. Two synthetic Unit actors, signed
// load state and a quantity-two source weapon exercise the variable sections.
// SHA256: c6a07b23647192fd225d9d30b3956987dbed56ff6be0ebd37919cb554578e967.
// Producer and command provenance: docs/1111/story.md. No game bytes are used.
func Genuine75() ([]byte, error) {
	packed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(genuine75Packed))
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
