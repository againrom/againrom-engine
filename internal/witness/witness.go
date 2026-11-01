// Package witness is the one place a test encodes, writes or decodes a PNG
// image. A test hands it an image and a file name; the file it writes carries
// the test's name and the source line of the call in two text chunks, so a
// reviewer who opens the picture can tell which test drew it and where.
package witness

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Reporter is what the helper needs of a test. *testing.T and *testing.B
// satisfy it; a probe that runs outside any test, in a child process or on a
// thread a test cannot stop, supplies its own.
type Reporter interface {
	Helper()
	Name() string
	Errorf(format string, args ...any)
}

// PNG writes img to the file name, in a folder that exists, and reports an
// error to the test when it cannot. The file holds the pixels png.Encode
// gives, plus the text chunks "test" (the test's name) and "source" (the file
// and line of the caller, from the module root).
func PNG(t Reporter, name string, img image.Image) {
	t.Helper()
	raw, err := encode(img)
	if err != nil {
		t.Errorf("witness %s: encode: %v", name, err)
		return
	}
	raw = withText(raw, "test", t.Name(), "source", callerSource(2))
	if err := os.WriteFile(name, raw, 0o666); err != nil {
		t.Errorf("witness %s: %v", name, err)
	}
}

// Bytes encodes img as PNG without writing it and without the text chunks, for
// a test that hands the bytes to the code under test or compares them. It
// returns nil after reporting an error.
func Bytes(t Reporter, img image.Image) []byte {
	t.Helper()
	raw, err := encode(img)
	if err != nil {
		t.Errorf("witness: encode: %v", err)
	}
	return raw
}

// Decode reads a PNG image.
func Decode(r io.Reader) (image.Image, error) { return png.Decode(r) }

// DecodeConfig reads the colour model and size of a PNG image.
func DecodeConfig(r io.Reader) (image.Config, error) { return png.DecodeConfig(r) }

func encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// pngSignatureAndHeader is the PNG signature (8 bytes) and the IHDR chunk
// (4 length, 4 type, 13 data, 4 checksum).
const pngSignatureAndHeader = 8 + 4 + 4 + 13 + 4

// withText inserts one tEXt chunk per key and value pair right after IHDR.
func withText(raw []byte, pairs ...string) []byte {
	if len(raw) < pngSignatureAndHeader {
		return raw
	}
	out := make([]byte, 0, len(raw)+64)
	out = append(out, raw[:pngSignatureAndHeader]...)
	for i := 0; i+1 < len(pairs); i += 2 {
		data := append(append([]byte(latin1(pairs[i])), 0), latin1(pairs[i+1])...)
		out = appendChunk(out, "tEXt", data)
	}
	return append(out, raw[pngSignatureAndHeader:]...)
}

func appendChunk(out []byte, kind string, data []byte) []byte {
	out = binary.BigEndian.AppendUint32(out, uint32(len(data)))
	start := len(out)
	out = append(out, kind...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(out[start:]))
}

// latin1 keeps the characters a tEXt chunk can hold.
func latin1(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e {
			return '?'
		}
		return r
	}, s)
}

// callerSource names the file and line skip frames above its caller, relative
// to the module root when go.mod stands above the file.
func callerSource(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	dir := filepath.Dir(file)
	for up := dir; ; {
		if _, err := os.Stat(filepath.Join(up, "go.mod")); err == nil {
			if rel, err := filepath.Rel(up, file); err == nil {
				file = rel
			}
			break
		}
		next := filepath.Dir(up)
		if next == up {
			break
		}
		up = next
	}
	return filepath.ToSlash(file) + ":" + strconv.Itoa(line)
}
